# FeedNest Design Document

A modern, self-hosted RSS feed reader with a beautiful UI, smart prioritization, and multi-user support.

## Problem

Existing RSS readers (FreshRSS, Miniflux, etc.) suffer from:
- Cluttered, dense UIs with information overload
- Poor article reading experience (bad typography, raw HTML feel)
- Weak navigation and content discovery
- Outdated aesthetics

Feedly gets closest to a good experience but is paid and cloud-hosted.

## Goals

- Beautiful, Feedly-inspired card-based UI for fast headline scanning
- Clean, distraction-free article reader with great typography
- Lightweight smart prioritization based on reading behavior
- Self-hosted via Docker with multi-user support
- Standalone app (no FreshRSS dependency)
- Support for 30-100+ feeds per user

## Architecture

```
Docker Compose
  +------------------+       +----------------------+
  |  Go Backend      |       |  SvelteKit Frontend  |
  |                  |       |                      |
  |  - Feed fetch    |<----->|  - Card/List views   |
  |  - RSS parse     | REST  |  - Article reader    |
  |  - Scoring       | API   |  - Feed management   |
  |  - Scheduler     |       |  - Settings          |
  |  - Auth (JWT)    |       |  - Auth pages        |
  +--------+---------+       +----------------------+
           |
           v
  +------------------+
  |   SQLite DB      |
  |   (volume)       |
  +------------------+
```

### Tech Stack

- **Frontend**: SvelteKit + TypeScript + Tailwind CSS
- **Backend**: Go (REST API, feed fetching, content extraction, auth)
- **Database**: SQLite (Docker volume mount)
- **Deployment**: Docker Compose (two containers + shared volume)

### Why These Choices

- **SvelteKit**: Compiled away framework overhead = snappy UI. Svelte's reactivity is ideal for the card-based scanning interface. Smaller bundles than React/Vue.
- **Go**: Excellent for concurrent feed fetching (goroutines), low memory footprint, compiles to a single binary. Perfect for a background scheduler service.
- **SQLite**: Zero-config database. Single file on a volume. No extra container needed. Sufficient for the expected scale.

## UI Design

### Three Main Views

#### 1. Feed Dashboard (Home)

Left sidebar + main content area:

```
+-------------------------------------------------------------+
|  FeedNest                         Search       Settings      |
+-------------+-----------------------------------------------+
|             |  Filter: All | Unread | Starred                |
|  FEEDS      |  Sort: Smart | Newest | Oldest                |
|             |  View: Cards | List                            |
|  Starred    |                                                |
|  All        |  +----------+ +----------+ +----------+       |
|             |  | thumb    | | thumb    | | thumb    |       |
|  Tech       |  | Title    | | Title    | | Title    |       |
|    Ars      |  | Src . 2h | | Src . 4h | | Src . 6h |       |
|    HN       |  | snippet  | | snippet  | | snippet  |       |
|    Verge    |  +----------+ +----------+ +----------+       |
|             |  +----------+ +----------+ +----------+       |
|  Design     |  | thumb    | | thumb    | | thumb    |       |
|    Dribbble |  | Title    | | Title    | | Title    |       |
|             |  | Src . 8h | | Src . 1d | | Src . 1d |       |
|  News       |  | snippet  | | snippet  | | snippet  |       |
|    BBC      |  +----------+ +----------+ +----------+       |
|             |                                                |
|  + Add Feed |              Load More                         |
+-------------+-----------------------------------------------+
```

- **Card view** (default): Responsive grid (3 cols desktop, 2 tablet, 1 mobile). Each card: thumbnail, title, source + relative time, 2-line snippet.
- **List view** (toggle): Compact rows: `[icon] Title -- Source . 2h ago . star`
- **Left sidebar**: Collapsible. Feeds grouped by user-defined categories. Unread counts per feed/category.
- **Smart sort** (default): Articles ranked by scoring algorithm.
- **Filters**: All / Unread / Starred quick toggles.

#### 2. Article Reader

```
+-------------------------------------------------------------+
|  <- Back                      Star    Share    Open Original |
+-------------------------------------------------------------+
|                                                              |
|              The Article Title Here                          |
|              Source Name . March 7, 2026 . 5 min read        |
|                                                              |
|         +-------------------------------------+              |
|         |          Hero Image                 |              |
|         +-------------------------------------+              |
|                                                              |
|    Clean article content with beautiful typography.          |
|    Max-width ~680px, line-height 1.7, proper font.           |
|    Images inline, code blocks styled.                        |
|                                                              |
|  +---------------------------+---------------------------+   |
|  |  <- Previous Article      |    Next Article ->        |   |
|  +---------------------------+---------------------------+   |
+-------------------------------------------------------------+
```

- **Readability-extracted content**: Go backend uses go-readability to strip ads, nav, footers.
- **Typography-first**: ~680px max-width, 1.7 line-height, Inter or system font stack.
- **Reading time estimate** from word count.
- **Previous/Next navigation** within the current feed/filter context.
- **Open original** button for the full source page.

#### 3. Feed Management

- Add feed by URL (auto-detect RSS/Atom from any page URL)
- Organize into categories (drag-and-drop reordering)
- Per-feed settings: custom fetch interval, auto-mark-read rules
- OPML import/export for migration

## Data Model

```sql
users
  id              INTEGER PRIMARY KEY
  username        TEXT UNIQUE NOT NULL
  email           TEXT UNIQUE NOT NULL
  password_hash   TEXT NOT NULL
  created_at      DATETIME
  updated_at      DATETIME

feeds
  id              INTEGER PRIMARY KEY
  user_id         INTEGER REFERENCES users(id)
  url             TEXT NOT NULL
  title           TEXT
  site_url        TEXT
  icon_url        TEXT
  category_id     INTEGER REFERENCES categories(id)
  fetch_interval  INTEGER DEFAULT 900  -- seconds
  last_fetched    DATETIME
  engagement_score REAL DEFAULT 0.0
  created_at      DATETIME

categories
  id              INTEGER PRIMARY KEY
  user_id         INTEGER REFERENCES users(id)
  name            TEXT NOT NULL
  position        INTEGER DEFAULT 0

articles
  id              INTEGER PRIMARY KEY
  feed_id         INTEGER REFERENCES feeds(id)
  guid            TEXT NOT NULL       -- RSS guid for dedup
  title           TEXT
  url             TEXT
  author          TEXT
  content_raw     TEXT               -- original HTML
  content_clean   TEXT               -- readability-extracted
  thumbnail_url   TEXT
  published_at    DATETIME
  fetched_at      DATETIME
  word_count      INTEGER DEFAULT 0
  reading_time    INTEGER DEFAULT 0  -- minutes
  is_read         BOOLEAN DEFAULT 0
  is_starred      BOOLEAN DEFAULT 0
  read_at         DATETIME
  score           REAL DEFAULT 0.0
  UNIQUE(feed_id, guid)

tags
  id              INTEGER PRIMARY KEY
  user_id         INTEGER REFERENCES users(id)
  name            TEXT NOT NULL
  UNIQUE(user_id, name)

article_tags
  article_id      INTEGER REFERENCES articles(id)
  tag_id          INTEGER REFERENCES tags(id)
  PRIMARY KEY (article_id, tag_id)

reading_events
  id              INTEGER PRIMARY KEY
  article_id      INTEGER REFERENCES articles(id)
  event_type      TEXT NOT NULL       -- click, read, star, dismiss
  duration_seconds INTEGER DEFAULT 0
  created_at      DATETIME

settings
  id              INTEGER PRIMARY KEY
  user_id         INTEGER REFERENCES users(id)
  key             TEXT NOT NULL
  value           TEXT
  UNIQUE(user_id, key)
```

## Smart Prioritization

Lightweight scoring -- no ML required:

```
score = (recency_weight * recency_score)
      + (engagement_weight * source_engagement_score)
      + (freshness_penalty if article > 24h old)
```

### Signals

| Signal | Weight | Description |
|--------|--------|-------------|
| Recency | High | Newer articles score higher (exponential decay) |
| Source engagement | Medium | Sources you click/read more rank higher |
| Read duration | Low | Longer reads = more interesting source |
| Stars | Strong positive | Starred articles boost that source significantly |
| Dismissals | Weak negative | Dismissed without reading = slight downrank |

All data per-user, stored locally. No external tracking.

## Authentication

- **Registration**: Username + email + password (bcrypt hashed)
- **Login**: JWT tokens (access token + refresh token)
- **Sessions**: Access token in memory, refresh token in httpOnly cookie
- **First-run setup**: If no users exist, show registration page
- **Optional**: Can be configured as single-user mode (skip login)

## REST API

```
# Auth
POST   /api/auth/register            # Create account (first user = admin)
POST   /api/auth/login                # Get JWT tokens
POST   /api/auth/refresh              # Refresh access token
POST   /api/auth/logout               # Invalidate refresh token

# Feeds
GET    /api/feeds                     # List all feeds with unread counts
POST   /api/feeds                     # Add new feed (auto-discover RSS)
PUT    /api/feeds/:id                 # Update feed settings
DELETE /api/feeds/:id                 # Remove feed and its articles

# Categories
GET    /api/categories                # List categories
POST   /api/categories                # Create category
PUT    /api/categories/:id            # Update (rename, reorder)
DELETE /api/categories/:id            # Delete category

# Articles
GET    /api/articles                  # Paginated, filterable list
       ?category=X&feed=X&status=unread|starred|read
       &sort=smart|newest|oldest
       &tag=X&page=1&limit=30
GET    /api/articles/:id              # Full article with clean content
PUT    /api/articles/:id              # Mark read, star, etc.
POST   /api/articles/:id/dismiss      # Mark read + negative scoring signal

# Tags
GET    /api/tags                      # List all user tags
POST   /api/articles/:id/tags         # Add tag to article
DELETE /api/articles/:id/tags/:tag    # Remove tag from article

# Bulk Operations
POST   /api/articles/bulk             # Bulk mark read, bulk tag, etc.

# OPML
POST   /api/opml/import               # Import OPML file
GET    /api/opml/export                # Export feeds as OPML

# Reading Events
POST   /api/events                    # Log reading event for scoring

# Settings
GET    /api/settings                  # Get user settings
PUT    /api/settings                  # Update user settings
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| j / k | Next / previous article in list |
| Enter | Open selected article |
| Escape | Back to feed list |
| s | Star / unstar article |
| m | Toggle read / unread |
| d | Dismiss article |
| v | Toggle card / list view |
| / | Focus search |
| ? | Show keyboard shortcuts |

## Features Summary

### In v1

- Card + list view toggle
- Smart sort (behavioral scoring)
- Clean readability-extracted article reader
- Categories for feed organization
- Tags for article labeling
- Multi-user auth (JWT)
- OPML import/export
- Feed auto-discovery from any URL
- Reading time estimates
- Keyboard shortcuts
- Dark / light theme (system-aware + manual)
- Docker Compose deployment
- Responsive design (desktop, tablet, mobile)

### Not in v1 (future consideration)

- Push notifications / web push
- Full-text search (FTS5)
- Mobile native app
- Social features (sharing between users)
- AI summarization
- Podcast / video feed support
- Browser extension
- Webhook integrations

## Project Structure

```
feednest/
  backend/
    cmd/
      feednest/
        main.go              # Entry point
    internal/
      api/
        router.go            # HTTP router setup
        middleware.go         # Auth, CORS, logging
        handlers/
          auth.go
          feeds.go
          categories.go
          articles.go
          tags.go
          opml.go
          events.go
          settings.go
      models/                # Data structures
      store/                 # SQLite data access layer
      fetcher/               # RSS/Atom feed fetcher
      parser/                # Feed parsing (RSS, Atom)
      readability/           # Content extraction
      scorer/                # Article scoring engine
      scheduler/             # Background fetch scheduler
    go.mod
    go.sum
    Dockerfile
  frontend/
    src/
      routes/
        +layout.svelte       # App shell, sidebar
        +page.svelte          # Dashboard (card/list view)
        article/[id]/
          +page.svelte        # Article reader
        feeds/
          +page.svelte        # Feed management
        settings/
          +page.svelte        # User settings
        auth/
          login/+page.svelte
          register/+page.svelte
      lib/
        components/
          ArticleCard.svelte
          ArticleList.svelte
          Sidebar.svelte
          ArticleReader.svelte
          FeedForm.svelte
          TagPicker.svelte
        stores/
          articles.ts
          feeds.ts
          auth.ts
          settings.ts
        api/
          client.ts           # HTTP client for backend API
        utils/
          keyboard.ts         # Keyboard shortcut handler
          time.ts             # Relative time formatting
    tailwind.config.ts
    svelte.config.js
    package.json
    Dockerfile
  docker-compose.yml
  README.md
```
