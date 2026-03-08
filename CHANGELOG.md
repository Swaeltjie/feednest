# Changelog

All notable changes to FeedNest will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
