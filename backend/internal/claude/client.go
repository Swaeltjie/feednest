// Package claude provides article summarization via the Anthropic Messages API.
//
// Two auth modes are supported:
//   - API key (default, supported): set ANTHROPIC_API_KEY.
//   - OAuth subscription token (opt-in): mount a Claude Code credentials file
//     (~/.claude/.credentials.json). The access token is sent as x-api-key with
//     the oauth beta headers and auto-refreshed. See oauth.go for the ToS note.
//
// Model defaults to claude-haiku-4-5 (cheap, fast — appropriate for the
// high-volume summarization use case), overridable via CLAUDE_SUMMARY_MODEL.
package claude

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	defaultModel = "claude-haiku-4-5"
	// oauthBetaHeaders is required when authenticating with a Claude Code OAuth
	// access token instead of a console API key.
	oauthBetaHeaders = "oauth-2025-04-20,claude-code-20250219"
	// claudeCodeIdentity is prepended as the first system block in OAuth mode;
	// OAuth subscription tokens are scoped to the Claude Code client.
	claudeCodeIdentity = "You are Claude Code, Anthropic's official CLI for Claude."
	// maxContentChars bounds how much article text we send for summarization.
	maxContentChars = 12000
)

// Client summarizes articles via Claude. It is safe for concurrent use.
type Client struct {
	sdk     anthropic.Client
	model   string
	mode    string // "apikey" | "oauth"
	enabled bool
}

// New constructs a Client from the environment. It never fails: if no auth is
// configured it returns a disabled client whose Summarize returns ErrDisabled.
func New() *Client {
	model := defaultModel
	if m := os.Getenv("CLAUDE_SUMMARY_MODEL"); m != "" {
		model = m
	}

	authMode := strings.ToLower(os.Getenv("CLAUDE_AUTH_MODE"))
	apiKey := os.Getenv("ANTHROPIC_API_KEY")

	// Decide mode: explicit override wins; otherwise prefer API key, then
	// fall back to a mounted OAuth credentials file if present.
	if authMode == "" {
		if apiKey != "" {
			authMode = "apikey"
		} else if p := defaultCredentialsPath(); p != "" {
			if _, err := os.Stat(p); err == nil {
				authMode = "oauth"
			}
		}
	}

	switch authMode {
	case "apikey":
		if apiKey == "" {
			return &Client{model: model}
		}
		return &Client{
			sdk:     anthropic.NewClient(option.WithAPIKey(apiKey)),
			model:   model,
			mode:    "apikey",
			enabled: true,
		}
	case "oauth":
		prov, err := newOAuthProvider(os.Getenv("CLAUDE_CREDENTIALS_PATH"))
		if err != nil {
			fmt.Printf("claude: OAuth mode requested but credentials unavailable: %v\n", err)
			return &Client{model: model}
		}
		mw := func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
			tok, err := prov.Token()
			if err != nil {
				return nil, fmt.Errorf("claude oauth token: %w", err)
			}
			// OAuth tokens authenticate via x-api-key (Bearer is rejected) plus
			// the oauth beta headers.
			req.Header.Set("x-api-key", tok)
			req.Header.Del("authorization")
			req.Header.Set("anthropic-beta", oauthBetaHeaders)
			return next(req)
		}
		return &Client{
			sdk: anthropic.NewClient(
				option.WithAPIKey("oauth"), // placeholder; overridden per-request
				option.WithMiddleware(mw),
			),
			model:   model,
			mode:    "oauth",
			enabled: true,
		}
	default:
		return &Client{model: model}
	}
}

// Enabled reports whether summarization is configured.
func (c *Client) Enabled() bool { return c.enabled }

// Mode returns the active auth mode ("apikey", "oauth", or "" when disabled).
func (c *Client) Mode() string { return c.mode }

// ErrDisabled is returned by Summarize when no auth is configured.
var ErrDisabled = fmt.Errorf("claude summarization is not configured")

// Summarize returns a short (2-3 sentence) TL;DR of the article.
func (c *Client) Summarize(ctx context.Context, title, content string) (string, error) {
	if !c.enabled {
		return "", ErrDisabled
	}

	content = strings.TrimSpace(content)
	if len([]rune(content)) > maxContentChars {
		content = string([]rune(content)[:maxContentChars])
	}
	if content == "" {
		content = title
	}

	system := []anthropic.TextBlockParam{}
	if c.mode == "oauth" {
		system = append(system, anthropic.TextBlockParam{Text: claudeCodeIdentity})
	}
	system = append(system, anthropic.TextBlockParam{Text: "You write concise TL;DR summaries of news and blog articles. Given an article, reply with 2-3 plain-sentence summary capturing the key points. No preamble, no markdown, no bullet points — just the summary."})

	userPrompt := fmt.Sprintf("Summarize this article.\n\nTitle: %s\n\nContent:\n%s", title, content)

	msg, err := c.sdk.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 350,
		System:    system,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude summarize: %w", err)
	}

	var sb strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	out := strings.TrimSpace(sb.String())
	if out == "" {
		return "", fmt.Errorf("claude returned an empty summary")
	}
	return out, nil
}
