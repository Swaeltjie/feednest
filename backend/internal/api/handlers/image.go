package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/feednest/backend/internal/urlutil"
)

const maxImageBytes = 5 * 1024 * 1024 // 5MB

// maxConcurrentFetches caps the number of in-flight outbound image fetches
// across all callers. Because each fetch can pull up to maxImageBytes, an
// unbounded number of concurrent requests would let a single client amplify
// bandwidth/connection usage; this semaphore bounds that.
const maxConcurrentFetches = 16

// ImageHandler proxies remote article thumbnails through the backend. Browsers
// block many otherwise-valid third-party images (Opaque Response Blocking,
// referer-based hotlink protection, mixed content). Fetching server-side with a
// browser User-Agent and no Referer, then serving same-origin, defeats those.
//
// This route is intentionally public (an <img> tag cannot send a JWT header).
// It is constrained to image responses, size-limited, and SSRF-protected via
// urlutil.IsSafeURL on the initial URL and every redirect hop.
type ImageHandler struct {
	client *http.Client
	// sem bounds the number of concurrent outbound fetches (see
	// maxConcurrentFetches). Acquire by sending, release by receiving.
	sem chan struct{}
}

// isAllowedPort restricts the destination port to the standard web ports so the
// proxy cannot be used as a general TCP port prober. An empty port means the
// scheme default (80/443), which is fine.
func isAllowedPort(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	port := u.Port()
	if port != "" && port != "80" && port != "443" {
		return fmt.Errorf("port not allowed: %s", port)
	}
	return nil
}

// browserUA mimics a normal browser so hotlink protections that key on
// User-Agent don't reject the fetch.
const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

func NewImageHandler() *ImageHandler {
	client := urlutil.SafeHTTPClient(15 * time.Second)
	// Re-validate every redirect target to prevent SSRF via redirect, and cap
	// the redirect chain.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if err := urlutil.IsSafeURL(req.URL.String()); err != nil {
			return fmt.Errorf("unsafe redirect: %w", err)
		}
		// Enforce the port allowlist on every redirect hop too, so a redirect
		// can't escape the initial-URL port check.
		if err := isAllowedPort(req.URL.String()); err != nil {
			return fmt.Errorf("unsafe redirect: %w", err)
		}
		return nil
	}
	return &ImageHandler{
		client: client,
		sem:    make(chan struct{}, maxConcurrentFetches),
	}
}

func (h *ImageHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	if raw == "" {
		http.Error(w, `{"error":"missing url"}`, http.StatusBadRequest)
		return
	}
	if err := urlutil.IsSafeURL(raw); err != nil {
		http.Error(w, `{"error":"url not allowed"}`, http.StatusForbidden)
		return
	}
	if err := isAllowedPort(raw); err != nil {
		http.Error(w, `{"error":"url not allowed"}`, http.StatusForbidden)
		return
	}

	// Bound concurrent outbound fetches. Fail fast when saturated rather than
	// blocking, so a flood can't pile up unbounded goroutines/connections.
	// Acquired after the cheap validation above so bad requests don't burn a slot.
	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		http.Error(w, `{"error":"busy"}`, http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, raw, nil)
	if err != nil {
		http.Error(w, `{"error":"invalid url"}`, http.StatusBadRequest)
		return
	}
	// Browser-like UA, no Referer — this is what defeats hotlink/ORB blocks.
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, `{"error":"fetch failed"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, `{"error":"upstream error"}`, http.StatusBadGateway)
		return
	}

	ct := resp.Header.Get("Content-Type")
	ctLower := strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
	// Refuse non-images, and SVG specifically: SVG can carry inline <script>,
	// and because this proxy serves same-origin, a malicious feed supplying an
	// SVG "thumbnail" would otherwise be a stored-XSS vector (JWT theft).
	if !strings.HasPrefix(ctLower, "image/") || ctLower == "image/svg+xml" || ctLower == "image/svg" {
		http.Error(w, `{"error":"unsupported image type"}`, http.StatusUnsupportedMediaType)
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Defense in depth: even if some non-image bytes slip through, this response
	// must never be able to execute script or be framed.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Content-Disposition", `inline; filename="image"`)
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, maxImageBytes))
}
