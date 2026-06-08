package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Public OAuth constants for the Claude Code client. These are not secrets —
// they are the well-known client identifier and token endpoint used by the
// official CLI's PKCE flow.
//
// NOTE ON TERMS OF SERVICE: using a Claude.ai Pro/Max OAuth subscription token
// to power a third-party application (rather than Claude Code / claude.ai) may
// violate Anthropic's terms. The supported path for FeedNest is an
// ANTHROPIC_API_KEY (see client.go). OAuth mode exists because the user
// explicitly requested mounting their subscription token; treat it as opt-in.
const (
	defaultOAuthClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	defaultOAuthTokenURL = "https://console.anthropic.com/v1/oauth/token"
)

// oauthProvider loads a mounted Claude Code credentials file and keeps the
// access token fresh, refreshing it via the OAuth refresh-token grant when it
// is close to expiry. It persists refreshed tokens back to the credentials
// file (0600) so the rotation survives restarts.
type oauthProvider struct {
	mu        sync.Mutex
	path      string
	clientID  string
	tokenURL  string
	http      *http.Client
	access    string
	refresh   string
	expiresAt time.Time
}

// credentialsFile mirrors ~/.claude/.credentials.json. Claude Code has used
// both a flat shape and one nested under "claudeAiOauth"; we accept either.
type credentialsFile struct {
	AccessToken  string          `json:"accessToken"`
	RefreshToken string          `json:"refreshToken"`
	ExpiresAt    json.RawMessage `json:"expiresAt"`
	ClaudeAiOAuth *struct {
		AccessToken  string          `json:"accessToken"`
		RefreshToken string          `json:"refreshToken"`
		ExpiresAt    json.RawMessage `json:"expiresAt"`
	} `json:"claudeAiOauth,omitempty"`
}

// defaultCredentialsPath returns the mounted credentials path, honoring an
// override and falling back to ~/.claude/.credentials.json.
func defaultCredentialsPath() string {
	if p := os.Getenv("CLAUDE_CREDENTIALS_PATH"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".claude", ".credentials.json")
}

func newOAuthProvider(path string) (*oauthProvider, error) {
	if path == "" {
		path = defaultCredentialsPath()
	}
	if path == "" {
		return nil, fmt.Errorf("no Claude credentials path resolved")
	}
	p := &oauthProvider{
		path:     path,
		clientID: envOr("CLAUDE_OAUTH_CLIENT_ID", defaultOAuthClientID),
		tokenURL: envOr("CLAUDE_OAUTH_TOKEN_URL", defaultOAuthTokenURL),
		http:     &http.Client{Timeout: 30 * time.Second},
	}
	if err := p.load(); err != nil {
		return nil, err
	}
	if p.access == "" {
		return nil, fmt.Errorf("no access token in %s", path)
	}
	return p, nil
}

func (p *oauthProvider) load() error {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}
	var f credentialsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}
	access, refresh, expRaw := f.AccessToken, f.RefreshToken, f.ExpiresAt
	if f.ClaudeAiOAuth != nil && f.ClaudeAiOAuth.AccessToken != "" {
		access = f.ClaudeAiOAuth.AccessToken
		refresh = f.ClaudeAiOAuth.RefreshToken
		expRaw = f.ClaudeAiOAuth.ExpiresAt
	}
	p.access = access
	p.refresh = refresh
	p.expiresAt = parseExpiresAt(expRaw)
	return nil
}

// parseExpiresAt accepts epoch milliseconds (number) or an RFC3339 string.
func parseExpiresAt(raw json.RawMessage) time.Time {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return time.Time{}
	}
	// Numeric epoch milliseconds
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.UnixMilli(ms)
	}
	// Quoted string: could be RFC3339 or a numeric string
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		if ms, err := strconv.ParseInt(str, 10, 64); err == nil {
			return time.UnixMilli(ms)
		}
		if t, err := time.Parse(time.RFC3339, str); err == nil {
			return t
		}
	}
	return time.Time{}
}

// Token returns a valid access token, refreshing when within 60s of expiry.
func (p *oauthProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// expiresAt zero means unknown — attempt use, refresh on demand if it fails.
	if !p.expiresAt.IsZero() && time.Until(p.expiresAt) > 60*time.Second {
		return p.access, nil
	}
	if p.refresh == "" {
		// No refresh token: use whatever we have until it stops working.
		return p.access, nil
	}
	if err := p.doRefresh(ctx); err != nil {
		// If the (possibly stale) access token is still non-empty, return it so
		// the request can surface a precise upstream auth error.
		if p.access != "" {
			return p.access, nil
		}
		return "", err
	}
	return p.access, nil
}

func (p *oauthProvider) doRefresh(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": p.refresh,
		"client_id":     p.clientID,
	})
	// Bound the refresh by 30s while still honoring the caller's deadline /
	// cancellation, whichever fires sooner.
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, p.tokenURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return fmt.Errorf("oauth refresh request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oauth refresh failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return fmt.Errorf("oauth refresh parse: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("oauth refresh returned no access_token")
	}
	p.access = tr.AccessToken
	if tr.RefreshToken != "" {
		p.refresh = tr.RefreshToken
	}
	if tr.ExpiresIn > 0 {
		p.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	p.persist()
	return nil
}

// persist writes the rotated tokens back to the credentials file (best effort).
func (p *oauthProvider) persist() {
	out := credentialsFile{
		AccessToken:  p.access,
		RefreshToken: p.refresh,
	}
	if !p.expiresAt.IsZero() {
		out.ExpiresAt = json.RawMessage(strconv.FormatInt(p.expiresAt.UnixMilli(), 10))
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(p.path, data, 0600)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
