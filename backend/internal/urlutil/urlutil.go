package urlutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AllowPrivate can be set to true in tests to skip the private IP check.
var AllowPrivate bool

// IsSafeURL validates that a URL points to a public internet host,
// blocking SSRF attempts against internal/private networks.
func IsSafeURL(rawURL string) error {
	if AllowPrivate {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}

	// Block localhost variants
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("localhost URLs are not allowed")
	}

	// Resolve hostname and check IP
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("failed to resolve host %s: %w", host, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if isPrivateIP(ip) {
			return fmt.Errorf("private/internal IP addresses are not allowed")
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// SafeHTTPClient returns an http.Client that validates resolved IPs
// at connection time, preventing DNS rebinding attacks.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	if AllowPrivate {
		return &http.Client{Timeout: timeout}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.LookupHost(host)
			if err != nil {
				return nil, err
			}
			for _, ipStr := range ips {
				ip := net.ParseIP(ipStr)
				if ip != nil && isPrivateIP(ip) {
					return nil, fmt.Errorf("connection to private IP %s is not allowed", ipStr)
				}
			}
			// Connect to the first resolved IP to pin it
			if len(ips) > 0 {
				addr = net.JoinHostPort(ips[0], port)
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}
