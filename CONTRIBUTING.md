# Contributing to FeedNest

Thanks for your interest in contributing! This guide will help you get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/feednest.git`
3. Create a feature branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Push and open a pull request

## Development Setup

### Prerequisites

- **Go 1.26+**
- **Node 24+**
- **Docker** (optional, for containerized development)

### Running Locally

```bash
# Backend (runs on :8082)
cd backend && go run ./cmd/feednest/

# Frontend (runs on :5173 with HMR)
cd frontend && npm install && npm run dev
```

### Running with Docker

```bash
docker compose up --build -d
```

## Code Style

- **Frontend**: Tab indentation, Svelte 5 runes syntax (`$state`, `$derived`, `$effect`, `$props`)
- **Backend**: Standard Go formatting (`gofmt`)
- **Commits**: Use [conventional commits](https://www.conventionalcommits.org/) — `feat:`, `fix:`, `docs:`, `chore:`, etc.

## Project Structure

- `backend/` — Go API server with Chi router and SQLite
- `frontend/` — SvelteKit 5 app with TailwindCSS 4
- `docs/` — Design documents and references

## Testing

```bash
# Backend tests
cd backend && go test ./...

# Frontend type checking
cd frontend && npm run check
```

## Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR
- Include a clear description of what changed and why
- Ensure backend tests pass and frontend type-checks cleanly
- Update documentation if your change affects user-facing behavior

## Reporting Issues

- Use [GitHub Issues](https://github.com/Swaeltjie/feednest/issues) to report bugs or request features
- Include steps to reproduce for bug reports
- Check existing issues before creating a new one

## License

By contributing, you agree that your contributions will be licensed under the [GPL-3.0 License](LICENSE).
