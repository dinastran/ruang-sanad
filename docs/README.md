# Laju Go Documentation

Welcome to the Laju Go documentation. This folder contains guides for building applications with Laju Go — a high-performance SaaS boilerplate with Go Fiber, Svelte 5, Inertia.js, and SQLite.

## 📚 Documentation Structure

### Guide

| Document | Description |
|----------|-------------|
| [Architecture](guide/architecture.md) | Layered architecture, design patterns, and best practices |
| [Routing](guide/routing.md) | Route definitions, middleware, and request handling |
| [Handlers](guide/handlers.md) | Building HTTP handlers, request/response handling |
| [Database](guide/database.md) | SQLite setup, migrations, and type-safe queries with sqlc |
| [Templ](guide/templ.md) | Type-safe HTML components via templ |
| [Frontend](guide/frontend.md) | Svelte 5 components and Inertia.js integration |
| [File Upload](guide/file-upload.md) | Avatar upload, validation, storage |
| [Email](guide/email.md) | SMTP password reset email |
| [Validation](guide/validation.md) | Input validation techniques |
| [Forms](guide/forms.md) | Form handling with Inertia |
| [Styling](guide/styling.md) | Tailwind CSS styling |
| [Storage](guide/storage.md) | File storage management |
| [Data Protection](guide/data-protection.md) | SQLite data protection and recovery |
| [Testing](guide/testing.md) | Testing strategies |

### Deployment

| Document | Description |
|----------|-------------|
| [Development Workflow](deployment/development.md) | Hot reload, scripts, and development best practices |
| [Production Deployment](deployment/production.md) | Ubuntu/Debian deployment with systemd and Nginx |
| [SQLite Configuration](deployment/sqlite-configuration.md) | SQLite tuning by server resources |

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/maulanashalihin/laju-go.git
cd laju-go

# Install dependencies
go mod download && npm install

# Configure environment
cp .env.example .env

# Start development
npm run dev:all
```

Visit `http://localhost:8080` to see your application.

## 🎯 Common Tasks

### Development

| Task | Guide |
|------|-------|
| Set up development environment | [Development Workflow](deployment/development.md) |
| Configure environment variables | `.env.example` → `.env` |
| Run development servers | `npm run dev:all` |
| Create new route & handler | [Routing](guide/routing.md) + [Handlers](guide/handlers.md) |
| Add database model | [Database](guide/database.md) |

### Deployment

| Task | Guide |
|------|-------|
| Build for production | [Production Deployment](deployment/production.md) |
| Optimize SQLite | [SQLite Configuration](deployment/sqlite-configuration.md) |

## 🔧 Resources

### External Links

- [Go Fiber Documentation](https://docs.gofiber.io/)
- [Svelte Documentation](https://svelte.dev/docs)
- [Inertia.js Documentation](https://inertiajs.com/)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [sqlc — Type-safe SQL](https://sqlc.dev/)
- [Goose Migrations](https://github.com/pressly/goose)

---

**Last Updated**: July 2026
