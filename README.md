# FeedNest

A modern, self-hosted RSS feed reader. Beautiful card-based UI, smart prioritization, and clean article reading experience.

## Features

- Card and list views for fast headline scanning
- Clean, distraction-free article reader with great typography
- Smart article prioritization based on your reading behavior
- Categories and tags for organization
- Multi-user support with JWT auth
- OPML import/export for easy migration
- Keyboard shortcuts for power users
- Dark/light theme with system preference detection
- Fully self-hosted via Docker

## Quick Start

```bash
git clone https://github.com/yourusername/feednest.git
cd feednest
cp .env.example .env
# Edit .env and set JWT_SECRET
docker compose up -d
```

Open http://localhost:3000 and create your account.

## Development

### Backend (Go)

```bash
cd backend
go run ./cmd/feednest/
```

### Frontend (SvelteKit)

```bash
cd frontend
npm install
npm run dev
```

## Tech Stack

- **Frontend**: SvelteKit, TypeScript, Tailwind CSS
- **Backend**: Go, Chi router, SQLite
- **Deployment**: Docker Compose

## License

MIT
