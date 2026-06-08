# Changelog

All notable changes to FeedNest will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.3] - 2026-06-08

### Added
- **AI article summaries (Claude)** — generate a 2–3 sentence TL;DR for any article from the reader. Uses the `claude-haiku-4-5` model via the official Anthropic Go SDK. Summaries are cached per-article (`articles.summary`). Authentication is pluggable: set `ANTHROPIC_API_KEY` (recommended), or opt into mounting a Claude Code OAuth credentials file with automatic token refresh (`CLAUDE_AUTH_MODE=oauth`). The Summarize button only appears when summarization is configured (`GET /api/summary/config`).
- **Full-text search (SQLite FTS5)** — article search now uses an FTS5 index over title + content (porter stemming, phrase/token matching) instead of `LIKE` substring matching, with safe client-side highlighting of matched terms in results. Falls back to `LIKE` automatically if the SQLite build lacks FTS5.
- **Feed health dashboard** — a per-feed health view (open via the command palette → "Feed Health") showing healthy/warning/dead counts, last-success time, last error, consecutive-failure counts, and a per-feed retry button. New feed columns: `last_success`, `consecutive_failures`, `last_fetch_status`.
- **Progressive Web App** — FeedNest is now installable: a web app manifest, a service worker that precaches the app shell and caches read endpoints (`/api/articles`, article detail, proxied images) for offline reading, and an online/offline store. Auth endpoints are never cached.
- **Backend image proxy** (`GET /api/image?url=...`) — routes article thumbnails server-side with a browser User-Agent and no Referer, defeating the browser-side blocks (Opaque Response Blocking, referer/hotlink protection, mixed content) that left many feed images broken. SSRF-protected and image-only.

### Fixed
- **Broken article thumbnails** — thumbnails now route through the image proxy and fall back to the gradient placeholder via an `onerror` handler when an image genuinely cannot be loaded (previously a failed image rendered as a broken/empty box; many otherwise-valid images were silently blocked by the browser).
- **Missing article thumbnails (extraction)** — many articles (e.g. BBC, Hacker News) had no thumbnail at all because (1) `readability.Extract` discarded a valid `og:image` whenever the article body couldn't be parsed, (2) the scheduled fetch path never looked at the article page for an image (only the RSS `<description>`), and (3) thumbnail backfill ran only once at startup. Extraction now pulls `og:image`/`twitter:image` independent of body parsing, the scheduler fetches page-level images at ingest, and backfill runs periodically. (BBC coverage went from missing-5 to complete; the only remaining gaps are pages that genuinely have no image.)
- **Filter rules could not be saved** — the rule editor sent `mark_read`/`star` action values that the backend rejected; aligned them to `auto_read`/`auto_star` so auto-mark-read and auto-star rules now save and apply.
- **Frontend container crash on restrictive static-file permissions** — adapter-node crashed with `EACCES` when serving a static file the runtime user could not read; the Docker image now normalizes asset permissions.

### Changed
- Backend Docker image is now built with the `sqlite_fts5` build tag to enable full-text search.

## [1.0.2] - 2026-06-07

### Fixed
- Backend: equalize bcrypt timing on login to prevent username enumeration via response timing
- Backend: return 401 instead of 500 when refreshing a token for a deleted user
- Backend: trim `ALLOWED_ORIGINS` entries so whitespace around comma-separated origins no longer breaks CORS
- Backend: parse the proxy remote address with `net.SplitHostPort` so trusted IPv6 proxies are matched correctly
- Backend: enforce per-user ownership in the article dismiss endpoint (prevents writing reading events for another user's articles)
- Backend: accept an empty request body in "mark all read"
- Backend: recover from panics in the feed/article worker goroutines so a malformed or hostile feed cannot crash the server
- Frontend: route a persistent 401 after token refresh to session-expiry/login instead of leaving the app in a stuck state
- Frontend: guard `loadMore` against a concurrent `load()` so stale pages are no longer appended after a filter/sort change
- Frontend: validate reader settings restored from `localStorage` against allowed values
- Frontend: cancel stale async feed-color fetches in the article card/list (fixes wrong accent colors on recycled cards)
- Frontend: fix hybrid skeleton-loader positioning
- Frontend: re-attach the `IntersectionObserver` to infinite-scroll-loaded articles so auto-mark-read works beyond the first page
- Frontend: fix the `Shift+G` shortcut being swallowed by the `gg` chord prefix
- Frontend: clear the pending keyboard-chord timer on teardown to avoid work against a destroyed component
- Frontend: stop rendering "Invalid Date" for unparseable timestamps

### Security
- Backend: upgrade dependencies to latest — chi 5.2.5 → 5.3.0, go-sqlite3 1.14.34 → 1.14.45, golang.org/x/crypto 0.48 → 0.52, x/net 0.51 → 0.55, x/text 0.34 → 0.37, goquery/cascadia bumped; `govulncheck` reports no known vulnerabilities
- Backend: migrate the deprecated `github.com/go-shiori/go-readability` to its maintained successor `codeberg.org/readeck/go-readability/v2` (v2.1.1)
- Frontend: upgrade dependencies to latest — vite 7 → 8, @sveltejs/vite-plugin-svelte 6 → 7, @sveltejs/kit → 2.63, svelte → 5.56, vitest → 4.1, svelte-check → 4.6, tailwindcss → 4.3, isomorphic-dompurify 3.0.0 → 3.16, jsdom/typescript bumped; `npm audit` reports 0 vulnerabilities (resolves the prior high-severity vite dev-server and moderate postcss advisories)

### Changed
- Frontend: add an explicit `@types/node` dev dependency (previously only present transitively)
- Frontend: set `legacy-peer-deps=true` in `.npmrc` so `npm ci` (Docker build, CI) resolves vite 8 / plugin 7 under the current `@sveltejs/kit` the same way as local installs

## [1.0.1] - 2026-03-27

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
