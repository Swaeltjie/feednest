// Package claude provides article summarization via the Anthropic Messages API.
//
// Two auth modes are supported:
//   - API key (default, supported): set ANTHROPIC_API_KEY.
//   - OAuth subscription token (opt-in): mount a Claude Code credentials file
//     (~/.claude/.credentials.json). The access token is sent as an
//     Authorization: Bearer token with the oauth beta headers and auto-refreshed.
//     See oauth.go for the ToS note.
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
			// Claude Code OAuth subscription tokens authenticate via
			// Authorization: Bearer plus the oauth beta headers. Sending them as
			// x-api-key returns 401 "invalid x-api-key". Drop the SDK's
			// placeholder x-api-key and set the bearer token instead.
			req.Header.Del("X-Api-Key")
			req.Header.Set("Authorization", "Bearer "+tok)
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

// Passage is one retrieved article excerpt passed to Answer for grounding. It
// mirrors store.Passage but lives here so the claude package stays decoupled
// from the store package (the API handler maps between the two).
type Passage struct {
	ID        int64
	Title     string
	URL       string
	FeedTitle string
	Excerpt   string
}

// Answer produces a grounded, cited answer to a question using only the
// supplied passages (the user's own RSS subscriptions). It instructs the model
// to cite inline as [n] and to admit when the feeds don't cover the question,
// avoiding any outside knowledge. Returns ErrDisabled when no auth is
// configured.
func (c *Client) Answer(ctx context.Context, question string, passages []Passage) (string, error) {
	if !c.enabled {
		return "", ErrDisabled
	}

	system := []anthropic.TextBlockParam{}
	if c.mode == "oauth" {
		system = append(system, anthropic.TextBlockParam{Text: claudeCodeIdentity})
	}
	system = append(system, anthropic.TextBlockParam{Text: "You answer questions using ONLY the numbered excerpts provided below, which are drawn from the user's own RSS feed subscriptions. " +
		"Cite every claim inline using the bracketed number of its source, like [1] or [2][3]. " +
		"If the excerpts do not contain enough information to answer, say plainly that the user's feeds haven't covered it — do not guess or use any outside knowledge. " +
		"Be concise and factual."})

	// Build the numbered passage block, bounding total excerpt characters so we
	// never blow the context budget. Mirrors the maxContentChars idea.
	var pb strings.Builder
	used := 0
	for i, p := range passages {
		excerpt := strings.TrimSpace(p.Excerpt)
		remaining := maxContentChars - used
		if remaining <= 0 {
			break
		}
		if len([]rune(excerpt)) > remaining {
			excerpt = string([]rune(excerpt)[:remaining])
		}
		used += len([]rune(excerpt))
		if i > 0 {
			pb.WriteString("\n\n")
		}
		fmt.Fprintf(&pb, "[%d] %s — %s\n%s", i+1, p.Title, p.FeedTitle, excerpt)
	}

	userPrompt := fmt.Sprintf("Question: %s\n\nExcerpts from your feeds:\n%s", question, pb.String())

	msg, err := c.sdk.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 600,
		System:    system,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude answer: %w", err)
	}

	var sb strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	out := strings.TrimSpace(sb.String())
	if out == "" {
		return "", fmt.Errorf("claude returned an empty answer")
	}
	return out, nil
}
