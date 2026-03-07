# FeedNest - Claude Code Instructions

## Project Overview

FeedNest is a self-hosted RSS/Atom feed reader with a Go backend and SvelteKit frontend, deployed via Docker Compose.

- **Backend**: Go 1.26, Chi router, SQLite, JWT auth
- **Frontend**: SvelteKit 5 with Svelte 5 runes, TailwindCSS 4, TypeScript
- **Ports**: Backend 8082, Frontend 3000

## Quick Start

```bash
docker compose up --build -d          # Build and run
docker compose logs -f backend        # Watch backend logs
docker compose logs -f frontend       # Watch frontend logs
docker compose down                   # Stop everything
```

## Project Structure

```
backend/
  cmd/feednest/main.go               # Entry point (PORT, DB_PATH, JWT_SECRET env vars)
  internal/
    api/
      router.go                      # All route definitions (Chi)
      auth.go                        # Register, Login, Refresh, JWT middleware
      middleware.go                  # Auth middleware
      swagger.go                    # Swagger UI + embedded OpenAPI spec
      openapi.yaml                  # Full API spec (OpenAPI 3.0)
      handlers/                     # Route handlers (articles, feeds, categories, tags, settings, opml, events)
    store/                          # SQLite data layer (article_store, feed_store, category_store, etc.)
    models/                         # Shared types (Article, Feed, Category, User, etc.)
    fetcher/fetcher.go              # RSS/Atom fetch + readability extraction + ad filtering
    scheduler/scheduler.go          # Background feed refresh (1 min interval)
    scorer/scorer.go                # Smart article ranking
    readability/readability.go      # Content extraction

frontend/
  src/
    routes/
      +page.svelte                  # Main app page (article list, toolbar, modals)
      +layout.svelte                # Root layout (auth check, theme)
      article/[id]/+page.svelte     # Article detail page
      api/docs/+server.ts           # Swagger UI proxy route
    lib/
      api/client.ts                 # HTTP client with JWT token management
      stores/                       # Svelte stores (articles, feeds, auth, settings)
      components/                   # UI components (Sidebar, ArticleCard, ArticleList, ArticleReader, etc.)
      utils/                        # Helpers (favicon, keyboard, time)
    app.css                         # Global styles, animations, CSS variables
```

## Svelte 5 Rules (CRITICAL)

This project uses **Svelte 5 runes syntax**. Follow these strictly:

- Use `$state()`, `$derived()`, `$effect()`, `$props()` - NOT Svelte 4 reactive declarations
- `{@const}` MUST be an immediate child of `{#if}`, `{#each}`, `{:else}`, or `{:then}` blocks. It CANNOT be inside HTML elements like `<div>` or `<span>`
- Props use `let { prop } = $props()` with type annotations
- Stores use `$storeName` syntax for auto-subscription

### Common Svelte 5 Patterns

```svelte
<!-- Props -->
let { value = defaultValue }: { value?: Type } = $props();

<!-- Reactive state -->
let count = $state(0);
let doubled = $derived(count * 2);

<!-- Effects with cleanup -->
$effect(() => {
  const timer = setTimeout(() => { ... }, 300);
  return () => clearTimeout(timer);
});
```

## Backend Conventions

- All routes prefixed with `/api/`
- Protected routes use `AuthMiddleware(jwtSecret)` in a Chi group
- User ID extracted via `apiutil.UserID(r.Context())`
- Database is SQLite with WAL mode, auto-migrations in `store/migrations.go`
- Feed fetching uses `gofeed` parser + `go-readability` for content extraction
- Sponsored content filtered at both ingestion (`fetcher.go`) and query time (`article_store.go`)
- Cross-feed deduplication via `MIN(id)` subquery per unique article URL

## Frontend Conventions

- API client at `$lib/api/client.ts` handles auth headers and token refresh
- Favicon helper: `getFaviconUrl(iconUrl, siteUrl, feedUrl)` - falls back to Google Favicon API
- All `<img>` tags for feed icons should include `onerror` handlers for broken image fallback
- Search uses 300ms debounce via `$effect` cleanup pattern
- Three view modes: card (grid), list, hybrid (magazine)
- Keyboard shortcuts defined in `$lib/utils/keyboard.ts`

## API Documentation

- Swagger UI: http://localhost:3000/api/docs (or http://localhost:8082/api/docs)
- OpenAPI spec: `backend/internal/api/openapi.yaml`

## Key API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/auth/register | Create account |
| POST | /api/auth/login | Login (returns JWT) |
| POST | /api/auth/refresh | Refresh token |
| GET | /api/feeds | List feeds with unread counts |
| POST | /api/feeds | Add feed (supports `new_category` field) |
| PUT | /api/feeds/{id} | Update feed (title, category_id, fetch_interval) |
| GET | /api/articles | List articles (params: status, sort, feed, category, tag, search, page, limit) |
| POST | /api/articles/bulk | Bulk mark_read/mark_unread/star/unstar |
| GET | /api/categories | List categories |
| POST | /api/opml/import | Import OPML file |
| GET | /api/opml/export | Export as OPML |

## Testing

```bash
cd backend && go test ./...           # Run all backend tests
cd frontend && npm run check          # TypeScript + Svelte check
```

## Common Gotchas

1. **Tab indentation**: All frontend files use tabs. The Edit tool often fails with tab matching - use Python scripts as a workaround when Edit tool string matching fails.

2. **`//go:embed` directive**: `openapi.yaml` is embedded into the Go binary. The embed directive must reference a file in the same package directory.

3. **CORS**: Backend allows origins `localhost:5173` and `localhost:3000`. Add new origins in `router.go` if needed.

4. **Docker volumes**: SQLite database persists in `feednest-data` named volume at `/data/feednest.db`.

5. **Feed titles**: Some feeds store their description as the title (e.g., Engadget). The title comes from the RSS feed's `<title>` element and can be updated via PUT `/api/feeds/{id}`.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Backend listen port |
| DB_PATH | ./feednest.db | SQLite database path |
| JWT_SECRET | change-me-in-production | JWT signing secret |
| ORIGIN | http://localhost:3000 | SvelteKit origin (CSRF) |
