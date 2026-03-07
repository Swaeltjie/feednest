<div align="center">

# FeedNest

### Your feeds. Your rules. Beautifully organized.

A blazing-fast, self-hosted RSS reader with a stunning glassmorphic UI, smart article prioritization, and a reading experience that puts content first.

**[Quick Start](#-quick-start)** · **[Features](#-features)** · **[Screenshots](#-screenshots)** · **[API](#-api)** · **[Development](#-development)**

---

</div>

## Why FeedNest?

Most RSS readers feel like they stopped evolving in 2010. FeedNest brings a modern, Feedly-inspired experience to self-hosted readers — with a design system built on glassmorphism, gradient accents, and buttery-smooth animations. No tracking. No ads. No algorithms deciding what you see.

## Features

### Reading Experience
- **Feedly-style slide-in reader** — read articles in a gorgeous side panel without losing your place
- **Three view modes** — Hybrid (hero cards + dense list), Card grid, and compact List view
- **Smart prioritization** — articles scored by your reading patterns, not engagement metrics
- **Clean typography** — carefully tuned prose styling with proper heading hierarchy, blockquote accents, and code block formatting

### Organization
- **Categories & tags** — organize feeds into categories, tag individual articles
- **Inline category creation** — create new categories right from the "Add Feed" modal
- **Smart deduplication** — same article across multiple feeds? You'll only see it once
- **Sponsored content filtering** — automatically hides sponsored posts and advertisements

### Interface
- **Glassmorphic design** — frosted glass toolbars, gradient accents, and adaptive dark/light themes
- **Full-text search** — instantly search across all article titles and content with debounced input
- **Live refresh countdown** — animated circular timer shows when feeds will refresh next (click to refresh now)
- **Keyboard shortcuts** — `j`/`k` navigate, `Enter` opens, `s` stars, `m` marks read, `v` cycles views, `/` focuses search
- **Staggered animations** — articles fade in with carefully timed delays for a premium feel

### Technical
- **Auto-refresh** — feeds checked every 60 seconds, background sync keeps everything fresh
- **OPML import/export** — migrate from any reader in seconds
- **Multi-user** — JWT auth with token refresh, each user gets their own feed universe
- **Feed icons** — automatic favicon resolution via Google's favicon API
- **Readability extraction** — fetches full article content even from summary-only feeds
- **Zero dependencies at runtime** — single Go binary + SQLite, no external databases needed

## Quick Start

```bash
git clone https://github.com/yourusername/feednest.git
cd feednest
cp .env.example .env    # Edit and set JWT_SECRET
docker compose up -d
```

Open **http://localhost:3000**, create your account, and add your first feed.

### Docker Compose

```yaml
services:
  backend:
    build: ./backend
    ports: ["8082:8082"]
    volumes: ["./data:/data"]
    environment:
      - JWT_SECRET=change-me-in-production
      - DB_PATH=/data/feednest.db

  frontend:
    build: ./frontend
    ports: ["3000:3000"]
    environment:
      - VITE_API_URL=http://backend:8082
```

## API

FeedNest has a full REST API. All endpoints (except auth) require a Bearer token.

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth/register` | Create account (`username`, `email`, `password`) |
| `POST` | `/api/auth/login` | Login, returns `access_token` + `refresh_token` |
| `POST` | `/api/auth/refresh` | Refresh access token |

### Feeds
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/feeds` | List all feeds with unread counts |
| `POST` | `/api/feeds` | Add feed (`url`, optional `category_id`, `new_category`) |
| `PUT` | `/api/feeds/:id` | Update feed title/category |
| `DELETE` | `/api/feeds/:id` | Remove feed and all its articles |

### Articles
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/articles` | List articles (supports `search`, `status`, `sort`, `feed`, `category`, `tag`, `page`, `limit`) |
| `GET` | `/api/articles/:id` | Get full article with content |
| `PUT` | `/api/articles/:id` | Update read/starred status |
| `POST` | `/api/articles/:id/dismiss` | Mark as read + log event |
| `POST` | `/api/articles/bulk` | Bulk mark read/unread/star/unstar |

### Categories
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/categories` | List categories |
| `POST` | `/api/categories` | Create category |
| `PUT` | `/api/categories/:id` | Update category |
| `DELETE` | `/api/categories/:id` | Delete category |

### Other
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET/PUT` | `/api/settings` | User preferences |
| `POST` | `/api/opml/import` | Import OPML file |
| `GET` | `/api/opml/export` | Export feeds as OPML |
| `GET/POST/DELETE` | `/api/tags/*` | Tag management |

### Search Example

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8082/api/articles?search=kubernetes&status=unread&sort=newest"
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate down / up |
| `Enter` | Open article in reader panel |
| `Escape` | Close reader panel |
| `s` | Toggle star |
| `m` | Toggle read/unread |
| `d` | Dismiss article |
| `v` | Cycle view mode |
| `/` | Focus search bar |

## Development

### Backend (Go 1.22+)

```bash
cd backend
go run ./cmd/feednest/
# Runs on :8082
```

### Frontend (Node 20+, SvelteKit 5)

```bash
cd frontend
npm install
npm run dev
# Runs on :5173 with HMR
```

### Project Structure

```
feednest/
├── backend/
│   ├── cmd/feednest/          # Entry point
│   └── internal/
│       ├── api/               # HTTP handlers + middleware
│       │   └── handlers/      # Articles, feeds, categories, tags, events, settings, OPML
│       ├── fetcher/           # RSS/Atom feed parser
│       ├── readability/       # Article content extraction
│       ├── scheduler/         # Background feed refresh
│       └── store/             # SQLite queries
├── frontend/
│   └── src/
│       ├── lib/
│       │   ├── api/           # HTTP client with token refresh
│       │   ├── components/    # Svelte 5 components (runes)
│       │   ├── stores/        # Reactive stores (articles, feeds, auth)
│       │   └── utils/         # Time formatting, favicons, keyboard
│       └── routes/            # SvelteKit pages
└── docker-compose.yml
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | SvelteKit 5, TypeScript, Tailwind CSS 4, Svelte 5 Runes |
| Backend | Go, Chi router, SQLite (WAL mode) |
| Content | gofeed parser, go-readability extractor |
| Auth | JWT (HS256) with access + refresh tokens |
| Deploy | Docker Compose, multi-stage Alpine builds |

## License

MIT
