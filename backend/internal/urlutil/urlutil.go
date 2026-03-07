package urlutil

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// IsSafeURL validates that a URL points to a public internet host,
// blocking SSRF attempts against internal/private networks.
func IsSafeURL(rawURL string) error {
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
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("private/internal IP addresses are not allowed")
		}
	}

	return nil
}
