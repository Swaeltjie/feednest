# Changelog

All notable changes to FeedNest will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

#### Backend
- Fix race condition on thumbnail backfill counter using `sync/atomic`
- Fix articles with empty URLs being silently excluded from listings
- Fix regex hide rules matching against truncated snippet instead of full article content
- Fix fragile slice append pattern in catch-up query that could corrupt arguments
- Surface category creation errors during feed creation instead of silently swallowing them
- Remove duplicate `MaxBytesReader` wrapping in OPML import handler (middleware already applies it)

#### Frontend
- Fix hardcoded `localhost:8082` in CommandPalette breaking OPML import/export in production
- Fix `value={null}` and `value={undefined}` in select elements producing string values instead of proper nulls
- Fix global keyboard handler in ArticleReader capturing keystrokes while typing in text inputs
- Fix silent auth failure on expired session — now clears state and redirects to login
- Save refresh token during `checkAuth` to prevent stale token after page reload
- Fix calm mode defaulting to on for new users (now defaults to off)
- Use `navigator.sendBeacon` for read time tracking on page unload for reliable delivery
- Add `preventDefault` to swipe `touchmove` handler to prevent scroll conflict
- Prefer feed-provided favicon icons over third-party favicon service
- Block `<style>` tags in DOMPurify sanitizer to prevent CSS injection from malicious feeds
- Remove undocumented `o` shortcut from keyboard hints (only `Enter` opens articles)
- Fix `Content-Type: application/json` being sent on bodyless GET/DELETE requests

#### Configuration
- Bind frontend Docker port to `127.0.0.1` to match backend's localhost-only binding
- Add missing `ALLOWED_ORIGINS` environment variable to `docker-compose.dev.yml`

### Security

#### Backend
- Pin Swagger UI CDN resources to exact version (5.18.2) with Subresource Integrity hashes
- Remove wildcard `Access-Control-Allow-Origin: *` from OpenAPI YAML endpoint
- Add JWT `Issuer` claim for token provenance verification
- Sanitize feed error messages to prevent internal network information leakage
- Limit regex filter patterns to 200 characters to mitigate CPU-based DoS
- Add semaphore to `FetchFeedNow` to limit concurrent immediate fetches (max 5)

#### Frontend
- Pin Swagger UI CDN resources to exact version with SRI hashes (matching backend)
- Use `fetch` with `keepalive: true` instead of `sendBeacon` for read tracking to preserve JWT auth header

### Performance

#### Backend
- Reuse HTTP connections via shared transport with connection pooling (was creating new client per request)
- Skip readability extraction for articles that already exist (check GUID before expensive HTTP fetch)
- Optimize `ListFeeds` unread count query from N correlated subqueries to single `LEFT JOIN` + `GROUP BY`
- Wrap `UpdateSettings` in a single transaction instead of N separate write transactions
- Add lightweight `ArticleBelongsToUser` ownership check for tags/events handlers (avoids fetching full article content)
- Run thumbnail backfill concurrently with first feed fetch on startup

#### Frontend
- Throttle article reader scroll handler via `requestAnimationFrame` to prevent excessive re-renders
- Parallelize initial API loads (feeds, categories, articles) instead of sequential waterfall
- Replace render-blocking CSS `@import` for Google Font with non-blocking `<link preload>`

## [1.0.0] - 2026-03-08

### Features

- **Feed Management** — Subscribe to RSS/Atom feeds with auto-discovery, organize into categories, OPML import/export
- **Article Reader** — Full article extraction via readability, lazy content fetching for paywalled/bot-protected sites
- **Smart Ranking** — Intelligent article scoring combining recency (60%) and engagement (40%) with 24-hour half-life decay
- **Three View Modes** — Card (grid), list, and hybrid (magazine) layouts
- **Search & Filtering** — Full-text search with 300ms debounce, filter by feed, category, tag, read status, and reading time
- **Bulk Operations** — Mark read/unread, star/unstar multiple articles, mark all read per feed/category, catch-up strategies
- **Tags** — Tag articles for custom organization
- **Keyboard Shortcuts** — Vim-inspired navigation with chord support (gg, G, j/k, o, s, etc.)
- **Command Palette** — Cmd+K quick access to feeds, actions, and navigation
- **Reading Stats** — Personalized words-per-minute tracking, reading time estimates
- **Filter Rules** — Auto-apply actions (star, mark read) to articles matching patterns
- **Feed Error Recovery** — Retry failed feeds, clear error states, per-feed fetch intervals
- **Dark/Light Theme** — System-aware theme with manual toggle
- **Docker Deployment** — Multi-stage Docker builds with health checks, non-root users, persistent SQLite volume

### Security

- JWT authentication with auto-generated secrets and token refresh
- SSRF protection blocking private/internal network URLs
- Rate limiting on auth endpoints (10 req/min per IP)
- Content sanitization via DOMPurify
- Security headers (X-Content-Type-Options, X-Frame-Options, Referrer-Policy)
- Request body size limits (1MB general, 5MB OPML imports)

### Developer Experience

- Comprehensive test suite — Go handler/store/middleware tests + Vitest frontend tests
- OpenAPI 3.0 spec with embedded Swagger UI at /api/docs
- Hot-reload development with Vite + SvelteKit
