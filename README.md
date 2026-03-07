<div align="center">

<img src="assets/banner.svg" alt="FeedNest" width="100%"/>

<br/>
<br/>

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?style=flat-square&logo=svelte&logoColor=white)](https://svelte.dev)
[![SQLite](https://img.shields.io/badge/SQLite-WAL-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white)](https://docker.com)
[![License](https://img.shields.io/badge/License-GPL--3.0-green?style=flat-square)](LICENSE)

**A blazing-fast, self-hosted RSS reader with a stunning glassmorphic UI,<br/>smart article ranking, and a reading experience that puts content first.**

No tracking. No ads. No algorithms deciding what you see.

[Quick Start](#quick-start) ·
[Features](#features) ·
[Command Palette](#command-palette) ·
[Keyboard Shortcuts](#keyboard-shortcuts) ·
[Tech Stack](#tech-stack) ·
[Development](#development)

---

<br/>

<img src="assets/screenshot-hybrid-light.png" alt="Hybrid view — light theme" width="100%"/>

<br/>

<p align="center">
<img src="assets/screenshot-reader-light.png" alt="Reading pane — light theme" width="49%"/>
<img src="assets/screenshot-reader-dark.png" alt="Reading pane — dark theme" width="49%"/>
</p>

</div>

## Why FeedNest?

Most RSS readers feel like they stopped evolving in 2010. FeedNest brings a modern, Feedly-inspired experience to self-hosting — built on **glassmorphism**, **gradient accents**, and **buttery-smooth spring animations**. It's the reading experience you deserve, on infrastructure you control.

<br/>

## Quick Start

```bash
git clone https://github.com/Swaeltjie/feednest.git
cd feednest
docker compose up -d
```

Open **http://localhost:3000**, create your account, and add your first feed. That's it.

> **Tip:** Set `JWT_SECRET` to something secure in your environment before deploying.

<br/>

## Features

### Reading Experience

- **Inline reading pane** — split-pane layout keeps article list visible while you read
- **Focus mode** — press `f` to hide the list and go full-width for distraction-free reading
- **Three view modes** — Hybrid (hero cards + dense list), Card grid, or compact List
- **Smart prioritization** — articles scored by recency (60%) and feed engagement (40%) with exponential decay
- **Beautiful typography** — tuned prose with proper headings, blockquote accents, and code formatting
- **Content extraction** — pulls full articles even from summary-only feeds using readability
- **Reading progress bar** — gradient bar tracks your scroll position through the article
- **Article navigation** — `j`/`k` to move between articles without closing the reader

### Organization

- **Categories & tags** — drag-and-drop categories, tag individual articles
- **Inline category creation** — new categories right from the "Add Feed" modal
- **Cross-feed deduplication** — same article in multiple feeds? You'll only see it once
- **Ad filtering** — automatically hides sponsored posts and bot-protection pages
- **OPML import/export** — migrate from any reader in seconds (also available via command palette)

### Interface

- **Glassmorphic design** — frosted glass toolbars, gradient accents, adaptive dark/light themes
- **Full-text search** — instant debounced search across all article titles and content
- **Command palette** — `Ctrl+K` to access everything: navigation, views, sorting, feeds, actions, and article search
- **Live refresh timer** — animated countdown ring shows next auto-refresh (click to refresh now)
- **Keyboard-first** — vim-style navigation, chord sequences (`gg`, `G`), and single-key actions
- **Spring animations** — physics-based motion system with staggered entrances and parallax effects
- **Dynamic feed colors** — accent colors extracted from feed favicons
- **Animated unread badges** — odometer-style count transitions in sidebar
- **Mobile gestures** — swipe right to mark read, swipe left to star
- **Responsive** — works beautifully from phones to ultra-wides

### Self-Hosting Done Right

- **Single binary + SQLite** — no Postgres, no Redis, no external dependencies
- **Multi-user** — JWT auth with automatic token refresh
- **SSRF protection** — blocks requests to private/internal networks
- **XSS protection** — article content sanitized with DOMPurify
- **Security headers** — X-Frame-Options, nosniff, referrer policy out of the box
- **Full REST API** — Swagger UI included at `/api/docs`

<br/>

## Command Palette

Press `Ctrl+K` (or `Cmd+K` on Mac) to open the command palette — your fastest way to do anything in FeedNest.

| Category | Commands |
|----------|----------|
| **Navigation** | Jump to All Articles, Unread, Starred, any category, or any feed |
| **Views** | Switch between Hybrid, Card Grid, and List views |
| **Sorting** | Change sort order: Smart, Newest First, Oldest First |
| **Actions** | Refresh feeds, Mark all as read, Add feed, Toggle dark mode |
| **OPML** | Import or export your feed subscriptions |
| **Articles** | Search articles by title (type 2+ characters to search) |
| **Utilities** | Open article in browser, View keyboard shortcuts |

Results are fuzzy-matched as you type. Arrow keys to navigate, Enter to execute, Esc to close.

<br/>

## Keyboard Shortcuts

| Key | Action |
|:---:|--------|
| `j` / `k` | Navigate down / up |
| `Enter` | Open article |
| `Escape` | Close reader |
| `s` | Toggle star |
| `m` | Toggle read/unread |
| `d` | Dismiss |
| `f` | Toggle focus mode |
| `v` | Cycle view mode |
| `1` / `2` / `3` | Hybrid / Cards / List view |
| `g g` | Jump to first article |
| `G` | Jump to last article |
| `/` | Focus search |
| `r` | Refresh feeds |
| `Ctrl+K` | Command palette |
| `?` | Keyboard shortcuts help |

See [docs/keyboard-shortcuts.md](docs/keyboard-shortcuts.md) for the full reference.

<br/>

## Tech Stack

| Technology | Purpose |
|-----------|---------|
| **SvelteKit 5** + Svelte 5 Runes | Reactive frontend with TypeScript |
| **Tailwind CSS 4** | Utility-first styling with glassmorphism |
| **Go 1.26** + Chi router | Fast, lightweight API server |
| **SQLite** (WAL mode) | Zero-config embedded database |
| **gofeed** + go-readability | RSS/Atom parsing + content extraction |
| **JWT** (HS256) | Stateless auth with refresh tokens |
| **Docker Compose** | One-command deployment |
| **DOMPurify** | XSS-safe article rendering |

<br/>

## Development

### Prerequisites

- **Go 1.26+** and **Node 24+** for local dev
- **Docker** for containerized deployment

### Local Development

```bash
# Backend — runs on :8082
cd backend && go run ./cmd/feednest/

# Frontend — runs on :5173 with HMR
cd frontend && npm install && npm run dev
```

### Docker

```bash
docker compose up --build -d     # Build and run
docker compose logs -f           # Watch logs
docker compose down              # Stop
```

### Project Structure

```
feednest/
├── backend/
│   ├── cmd/feednest/             # Entry point
│   └── internal/
│       ├── api/                  # Routes, auth, middleware, Swagger
│       │   └── handlers/         # Request handlers
│       ├── fetcher/              # RSS/Atom feed fetcher
│       ├── readability/          # Full content extraction
│       ├── scheduler/            # Background refresh
│       ├── scorer/               # Smart article ranking (recency + engagement)
│       ├── urlutil/              # SSRF protection
│       └── store/                # SQLite data layer with indexes
├── frontend/
│   └── src/
│       ├── lib/
│       │   ├── api/              # HTTP client with token refresh
│       │   ├── components/       # Svelte 5 components
│       │   ├── stores/           # Reactive state management
│       │   └── utils/            # Helpers (keyboard, color, particles, parallax, swipe)
│       └── routes/               # SvelteKit pages
├── docs/                         # Design docs, keyboard shortcuts reference
├── assets/                       # Logo and banner SVGs
└── docker-compose.yml
```

<br/>

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | `change-me-in-production` | **Set this.** JWT signing key. |
| `PORT` | `8080` | Backend listen port |
| `DB_PATH` | `./feednest.db` | SQLite database path |
| `ORIGIN` | `http://localhost:3000` | SvelteKit origin (CSRF) |

<br/>

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and PR guidelines.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting and security measures.

---

<div align="center">

**Built with obsessive attention to detail.**

<sub>GPL-3.0 License</sub>

</div>
