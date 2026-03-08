# Development Guide

## Prerequisites

- **Go 1.26+** and **Node 24+** for local development
- **Docker** for containerized deployment

## Local Development

```bash
# Backend — runs on :8082
cd backend && go run ./cmd/feednest/

# Frontend — runs on :5173 with HMR
cd frontend && npm install && npm run dev
```

## Docker

```bash
docker compose up --build -d     # Build and run
docker compose logs -f           # Watch logs
docker compose down              # Stop
```

## Project Structure

```
feednest/
├── backend/
│   ├── cmd/feednest/             # Entry point
│   └── internal/
│       ├── api/                  # Routes, auth, middleware, Swagger
│       │   ├── handlers/         # Request handlers
│       │   └── apiutil/          # Auth helpers
│       ├── fetcher/              # RSS/Atom feed fetcher
│       ├── readability/          # Full content extraction
│       ├── scheduler/            # Background refresh (5 min interval)
│       ├── scorer/               # Smart article ranking
│       ├── urlutil/              # SSRF protection
│       ├── models/               # Shared types
│       └── store/                # SQLite data layer
├── frontend/
│   └── src/
│       ├── lib/
│       │   ├── api/              # HTTP client with JWT token refresh
│       │   ├── components/       # Svelte 5 components
│       │   ├── stores/           # Reactive state (articles, feeds, auth, settings)
│       │   └── utils/            # Helpers (keyboard, color, favicon, particles, swipe)
│       └── routes/               # SvelteKit pages
├── docs/                         # Documentation and design plans
├── assets/                       # Logo and banner SVGs
└── docker-compose.yml
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `PORT` | `8080` | Backend listen port |
| `DB_PATH` | `./feednest.db` | SQLite database path |
| `ORIGIN` | `http://localhost:3000` | SvelteKit origin (CSRF) |

## API Reference

Swagger UI is available at `/api/docs` (both `http://localhost:3000/api/docs` and `http://localhost:8082/api/docs`).

### Key Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/auth/register | Create account |
| POST | /api/auth/login | Login (returns JWT) |
| POST | /api/auth/refresh | Refresh token |
| GET | /api/feeds | List feeds with unread counts |
| POST | /api/feeds | Add feed (supports `new_category` field) |
| PUT | /api/feeds/{id} | Update feed (title, category_id, fetch_interval) |
| DELETE | /api/feeds/{id} | Remove feed |
| GET | /api/articles | List articles (params: status, sort, feed, category, tag, search, page, limit, published_after, min_reading_time, max_reading_time) |
| GET | /api/articles/{id} | Get single article with full content |
| PUT | /api/articles/{id} | Update article (is_read, is_starred) |
| POST | /api/articles/bulk | Bulk mark_read/mark_unread/star/unstar |
| POST | /api/articles/mark-all-read | Mark all read (optional feed_id, category_id scope) |
| POST | /api/articles/catch-up | Catch-up: keep_newest or older_than strategy |
| GET | /api/categories | List categories |
| POST | /api/categories | Create category |
| GET | /api/rules | List filter rules |
| POST | /api/rules | Create filter rule |
| PUT | /api/rules/{id} | Update filter rule |
| DELETE | /api/rules/{id} | Delete filter rule |
| POST | /api/opml/import | Import OPML file |
| GET | /api/opml/export | Export as OPML |
| GET | /api/settings | Get user settings |
| PUT | /api/settings | Update settings |
| GET | /api/settings/wpm | Get personalized WPM |
| GET | /api/settings/reading-stats | Weekly reading statistics |
| POST | /api/events | Track reading events |

## Svelte 5 Conventions

This project uses **Svelte 5 runes syntax**:

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

- `{@const}` must be an immediate child of `{#if}`, `{#each}`, `{:else}`, or `{:then}` blocks
- All frontend files use **tab indentation**
- Stores use `$storeName` syntax for auto-subscription

## Testing

```bash
cd backend && go test ./...           # Run all backend tests
cd frontend && npm run check          # TypeScript + Svelte check
```

## Command Palette

Press `Ctrl+K` (or `Cmd+K` on Mac) to open the command palette.

| Category | Commands |
|----------|----------|
| **Navigation** | Jump to All Articles, Unread, Starred, any category, or any feed |
| **Views** | Switch between Hybrid, Card Grid, and List views |
| **Sorting** | Change sort order: Smart, Newest First, Oldest First |
| **Actions** | Refresh feeds, Mark all as read, Add feed, Toggle dark mode |
| **OPML** | Import or export feed subscriptions |
| **Articles** | Search articles by title (type 2+ characters) |
| **Utilities** | Open article in browser, View keyboard shortcuts, Filter rules |

Results are fuzzy-matched as you type. Arrow keys to navigate, Enter to execute, Esc to close.
