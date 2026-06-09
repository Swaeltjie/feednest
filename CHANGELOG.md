# Changelog

All notable changes to FeedNest will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.5] - 2026-06-09

A correctness, security, and CI-hardening release from a full-codebase audit (multi-agent bug hunt with adversarial verification). No new features and no breaking changes.

### Fixed
- **`published_after` filtering was silently broken** — date-scoped views (Best Of's "last 7 days", any `?published_after=` query, relative `24h`/`7d`/`1w`) returned near-empty results and a wrong `total`. Stored timestamps are space-separated (`2026-06-09 12:00:00+00:00`) while the threshold was formatted RFC3339 (`T`-separated), and the two were compared lexicographically, so every same-day article sorted below the threshold and vanished. Both sides are now normalized with SQLite `datetime()` (offset-aware, so non-UTC feed timestamps compare correctly too). Added a regression test.
- **"Ask Your Feeds" returned nothing without FTS5** — when the SQLite build lacked FTS5, the LIKE fallback matched the *entire raw question* as one substring (`%what have my feeds said about Rust lately?%`), so natural-language queries never matched. The fallback now tokenizes and drops stopwords exactly like the FTS path (shared `topicalTerms` helper), surfacing the topical terms.
- **Transient backend blips logged users out** — a network error or 5xx coinciding with a routine access-token refresh destroyed the still-valid refresh token and forced re-login. Token refresh is now tri-state (`ok` / `invalid` / `transient`): only a definitive `401/403` clears credentials; transient failures surface a retryable error and keep the session.
- **Article `PUT` returned 204 for non-existent / other users' IDs and empty bodies** — the update handler now verifies ownership (404 on miss, mirroring Dismiss) and rejects a body with no `is_read`/`is_starred` (400), instead of silently reporting success.
- **Filter-rule regex validation gap** — sending `{"operator":"regex"}` with no value skipped the compile check, letting an invalid stored pattern become a regex that silently never matched. Validation now checks the *effective* operator+value (including operator-only updates).
- **Adding a feed with an existing category name left it uncategorized** — naming an already-existing category in the "new category" field hit a UNIQUE collision and silently dropped the assignment; it now resolves to and uses the existing category.
- **Article creation + auto-rules now atomic** — `auto_read`/`auto_star` rule application is committed in the same transaction as the article insert, so a crash mid-create can no longer leave a persisted article with its rules permanently un-applied.
- **OAuth-mode AI refresh storm** — when the mounted credentials file omitted an expiry and a refresh returned no `expires_in`, every Claude request re-refreshed under the lock, serializing AI traffic; a conservative expiry is now assumed so the fast path engages.
- **Scheduler "N new" over-count** — failed/duplicate inserts are no longer counted as new items in the per-feed fetch log.

### Security
- **Immediate-fetch goroutine flood** — `FetchFeedNow` (fired by add-feed and feed retry, both unrated) acquired its concurrency slot *inside* a freshly spawned goroutine, so spamming those endpoints piled up unbounded blocked goroutines. It now acquires the slot non-blocking *before* spawning and drops overflow (the periodic scheduler reconciles), and `POST /api/feeds/discover` (which fans out to ~26 outbound requests) is now per-user rate limited.
- **Outbound port allowlist on all SSRF paths** — previously only the image proxy restricted destination ports; feed fetch, discovery, OPML import, and readability could be coerced into connecting to arbitrary TCP ports on public hosts (e.g. `:6379`). The port allowlist (80/443) is now enforced inside `IsSafeURL` and at connection time (covering redirect hops).
- **Bounded outbound DNS** — SSRF host resolution now uses a 5 s context-bounded resolver instead of a deadline-less `net.LookupHost`, so a hostile/slow resolver (e.g. a large OPML of bad hosts) can't pin a request goroutine.
- **OPML import bounded** — imports are capped at 1000 entries (a 5 MB OPML could otherwise create tens of thousands of feeds the scheduler fetches forever); the response now reports `total`/`processed`/`truncated`.
- **HTTP server timeouts (Slowloris) + graceful shutdown** — the server now sets `ReadHeaderTimeout`/`ReadTimeout`/`IdleTimeout` (no `WriteTimeout`, so `/api/ask`'s streamed answers aren't truncated) and handles `SIGINT`/`SIGTERM` so `db.Close()`/`scheduler.Stop()` actually run on shutdown instead of being skipped by `log.Fatal`'s exit.
- **Cached-summary write scoped to owner** — `UpdateArticleSummary` now scopes its write by feed ownership, closing a latent cross-tenant write should the user-scoped precheck ever be bypassed.
- **SSR proxy header hygiene** — the `/api/*` reverse proxy strips upstream `content-encoding`/`content-length` and hop-by-hop headers (undici has already decoded the body), preventing `ERR_CONTENT_DECODING_FAILED`/truncation once backend compression or streaming is enabled.

### Changed
- **CI tests both search backends** — the backend test job now runs as a matrix over `["", "sqlite_fts5"]`, so both the LIKE fallback and the production FTS5 path are exercised (the gap that let the search regression reach `main`).
- **FTS5 index no longer re-tokenizes on every row update** — the `AFTER UPDATE` trigger is now scoped to the indexed text columns with a NULL-safe `WHEN` guard (and the old unguarded trigger is dropped on upgrade), so bulk mark-read and the periodic rescore no longer needlessly re-index article bodies and hold the single SQLite write connection.
- **`loadMore` pagination race** — infinite-scroll/"load more" now bails when a filter `load()` is in flight and guards the page base against concurrent resets, avoiding a skipped/duplicated page on fast filter switches.

## [1.0.4] - 2026-06-08

### Added
- **Best Of — smart ranking activated** — the article scorer is now live. A background pass (on the existing feed-refresh schedule) computes a per-feed engagement score from your own `reading_events` (read-through rate, read depth vs. article length, click signal, dismiss penalty over a 90-day window) and writes a recency-plus-engagement `score` to recent articles. A new **Best Of** sidebar view surfaces the top-ranked articles from the last 7 days (`GET /api/articles?sort=smart&published_after=...`), and the existing **Smart** sort now reflects real signal everywhere instead of always ordering by date. All ranking is computed and stored locally — no behavior leaves your instance. (Previously `scorer.CalculateScore`, `articles.score`, and `feeds.engagement_score` were dead code.)
- **Ask Your Feeds — AI answers grounded in your archive** — a new **Ask** button (and command-palette action) opens a prompt that answers natural-language questions about your subscriptions. Relevant article passages are retrieved from the existing FTS5 index (no embeddings, no vector DB) and sent to Claude, which answers using only those passages and returns inline `[n]` citations linking to the source articles. New endpoint `POST /api/ask`; shares the AI gate with summaries (`GET /api/summary/config`), so it appears only when `ANTHROPIC_API_KEY` (or OAuth) is configured.

### Security
- **SSRF blocklist expanded** — the URL guard (image proxy, feed fetch, discovery) now also blocks CGNAT/shared-address space (`100.64.0.0/10`), NAT64 (`64:ff9b::/96`, including the `169.254.169.254` metadata-bypass vector), and other reserved ranges (`198.18.0.0/15`, `240.0.0.0/4`, documentation ranges).
- **Image proxy hardened** — outbound destination ports restricted to 80/443 on the initial URL *and* every redirect hop, a global outbound-concurrency cap, response size lowered to 5 MB, and per-IP rate limiting (120/min) so the public proxy can't be abused as an open relay.
- **AI endpoints rate-limited** — `POST /api/ask` and `POST /api/articles/{id}/summary` are now per-user rate limited (10/min) to bound LLM cost and abuse.
- **Auth rate-limiter fixes** — keys on the client IP with the ephemeral source port stripped (repeated fresh connections now share a bucket), and behind a trusted proxy honors only the right-most `X-Forwarded-For` hop (anti-spoof).
- **Filter-rule feed ownership** — `feed_id` on filter rules is validated against the caller's own feeds, closing an IDOR/cross-tenant gap (invalid ids now return 400 instead of 500).
- **JWT secret strength** — an operator-supplied `JWT_SECRET` shorter than 32 characters now aborts startup instead of running with a weak key.
- **CORS** — disabled unnecessary credentialed CORS (authentication is Bearer-token only).
- **Feed discovery bounded** — discovery's outbound fetches now inherit the request context and a 30 s aggregate deadline (cancel on client disconnect, bounded work).
- **Scheduler amplification cap** — per-feed processing is capped at the 200 newest items to prevent readability-fetch amplification.
- **Prompt-injection hardening** — RSS-derived title/feed/excerpt sent to Claude (Ask and summaries) are wrapped in delimited untrusted-data blocks with escaping.
- **Cache purge on logout** — the service worker evicts user-specific `/api/articles` and proxied-image responses on sign-out.

### Fixed
- **Regex hide-rules + pagination** — regex hide rules are now applied before paging, so the total count and every page are correct (previously articles were silently dropped and counts were wrong).
- **"Mark all as read" scope** — now respects the active view/filter and the full result set instead of only the currently-loaded page.
- **`published_after` validation** — an invalid value returns 400 instead of being silently forwarded and matching the wrong articles.
- **Infinite scroll after dismiss** — `loadMore` tracks a page cursor instead of deriving the page from the mutated list, avoiding a wasted refetch and a possible skipped article.
- **AI request cancellation** — a local timeout now maps to 504 and a client disconnect to a no-op (instead of a generic 502), and the OAuth token refresh honors the request deadline.
- **Feed `/retry` dedup** — re-running a feed fetch no longer re-extracts already-stored articles.

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
- **AI summaries OAuth mode** — the mounted Claude Code token is now sent via `Authorization: Bearer` (sending it as `x-api-key` returns a 401), so OAuth-mode summarization works end-to-end.
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
