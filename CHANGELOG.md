# Changelog

All notable changes to FeedNest will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Update Go module version from 1.25.3 to 1.26 to match Dockerfile
- Update TypeScript from 5.9 to 6.0
- Update jsdom from 28.x to 29.x
- Update isomorphic-dompurify from 3.0.0 to 3.7.1
- Update @sveltejs/kit, svelte, vitest, tailwindcss, svelte-check to latest minor/patch versions
- Remove unused `@sveltejs/adapter-auto` dependency (project uses adapter-node)

### Fixed
- Fix race condition on thumbnail backfill counter using `sync/atomic`
- Fix articles with empty URLs being silently excluded from listings
- Fix regex hide rules matching against truncated snippet instead of full article content
- Fix fragile slice append pattern in catch-up query that could corrupt arguments
- Surface category creation errors during feed creation instead of silently swallowing them
- Remove duplicate `MaxBytesReader` wrapping in OPML import handler
- Fix hardcoded `localhost:8082` in CommandPalette breaking OPML import/export in production
- Fix select elements producing string `"null"`/`"undefined"` instead of proper null values
- Fix keyboard handler in ArticleReader capturing keystrokes while typing in text inputs
- Fix silent auth failure on expired session — now clears state and redirects to login
- Save refresh token during `checkAuth` to prevent stale token after page reload
- Fix calm mode defaulting to on for new users (now defaults to off)
- Fix swipe `touchmove` handler conflicting with page scroll
- Prefer feed-provided favicon icons over third-party service
- Fix `Content-Type` header being sent on bodyless GET/DELETE requests
- Remove undocumented `o` shortcut from keyboard hints
- Bind frontend Docker port to `127.0.0.1` to match backend security posture
- Add missing `ALLOWED_ORIGINS` env var to `docker-compose.dev.yml`

### Security
- Pin Swagger UI CDN to exact version (5.18.2) with Subresource Integrity hashes
- Remove wildcard `Access-Control-Allow-Origin: *` from OpenAPI YAML endpoint
- Add JWT `Issuer` claim for token provenance verification
- Sanitize feed error messages to prevent internal network information leakage
- Block `<style>` tags in DOMPurify sanitizer to prevent CSS injection from feeds
- Limit regex filter patterns to 200 characters to mitigate CPU-based DoS
- Add semaphore to `FetchFeedNow` to cap concurrent immediate fetches at 5
- Use `fetch` with `keepalive` instead of `sendBeacon` to preserve auth headers on page unload

### Performance
- Reuse HTTP connections via shared transport with connection pooling
- Skip readability extraction for articles that already exist (GUID check before fetch)
- Optimize `ListFeeds` unread count from N correlated subqueries to single `LEFT JOIN`
- Wrap `UpdateSettings` in a single transaction instead of N separate writes
- Add lightweight ownership check for tags/events (avoids fetching full article content)
- Run thumbnail backfill concurrently with first feed fetch on startup
- Throttle article reader scroll handler via `requestAnimationFrame`
- Parallelize initial API loads (feeds, categories, articles) instead of waterfall
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
