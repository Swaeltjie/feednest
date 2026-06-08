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

// reservedCIDRs holds special-use ranges that the net.IP classifier methods
// (IsLoopback/IsPrivate/IsLinkLocal*/IsUnspecified) do not cover but that
// frequently route to internal infrastructure or can be used to bypass intent.
// net.IPNet.Contains normalizes IPv4-in-IPv6 forms, so a range matches both
// 4-byte and 16-byte parses. Built once at package init.
var reservedCIDRs = func() []*net.IPNet {
	nets := []*net.IPNet{}
	for _, c := range []string{
		"100.64.0.0/10",  // RFC 6598 CGNAT/shared address space (e.g. Tailscale default)
		"192.0.0.0/24",   // RFC 6890 IETF protocol assignments
		"198.18.0.0/15",  // RFC 2544 benchmarking
		"240.0.0.0/4",    // reserved (class E), incl. 255.255.255.255
		"64:ff9b::/96",   // RFC 6052 NAT64 well-known prefix (maps e.g. 169.254.169.254)
		"64:ff9b:1::/48", // RFC 8215 NAT64 local-use prefix
		// documentation ranges (defense-in-depth, non-routable):
		"192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24", "2001:db8::/32",
	} {
		if _, n, err := net.ParseCIDR(c); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}()

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
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range reservedCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// sharedTransport is a long-lived transport with SSRF-safe dialing and connection pooling.
var sharedTransport = &http.Transport{
	MaxIdleConns:        50,
	MaxIdleConnsPerHost: 5,
	IdleConnTimeout:     90 * time.Second,
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
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		if len(ips) > 0 {
			addr = net.JoinHostPort(ips[0], port)
		}
		return dialer.DialContext(ctx, network, addr)
	},
}

// SafeHTTPClient returns an http.Client that validates resolved IPs
// at connection time, preventing DNS rebinding attacks. Uses a shared
// transport for connection pooling and keep-alive reuse.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	if AllowPrivate {
		return &http.Client{Timeout: timeout}
	}
	return &http.Client{Timeout: timeout, Transport: sharedTransport}
}
