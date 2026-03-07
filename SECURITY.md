# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in FeedNest, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email the maintainer directly or use [GitHub's private vulnerability reporting](https://github.com/Swaeltjie/feednest/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We aim to acknowledge reports within 48 hours and provide a fix within 7 days for critical issues.

## Security Measures

FeedNest implements the following security protections:

### Authentication
- JWT-based authentication with HS256 signing
- Automatic token refresh
- Password hashing with bcrypt

### Input Validation
- **XSS protection** — Article HTML sanitized with DOMPurify before rendering
- **SQL injection protection** — Parameterized queries throughout the data layer
- **SSRF protection** — Feed URLs validated against private/internal network ranges
- **CSRF protection** — SvelteKit origin checking enabled

### HTTP Security Headers
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: strict-origin-when-cross-origin`
- CORS restricted to configured origins

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |

## Best Practices for Self-Hosting

- **Set `JWT_SECRET`** to a strong, random value (at least 32 characters)
- Run behind a reverse proxy (nginx, Caddy, Traefik) with TLS
- Keep Docker images updated
- Restrict network access to the backend port (8082) — only the frontend needs to reach it
