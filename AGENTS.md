# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Jobby Search is a full-stack job tracking application with three services:
- **Backend**: Go REST API (port 8080)
- **Frontend**: React SPA (port 5173)
- **MCP Server**: Node.js job search service via JobSpy (port 9423)

> **Important**: Use **bun** for Node.js package management.

## Commands

### Backend (Go)
```bash
cd backend
go run ./cmd/server/        # start dev server
go build ./...              # build
go test ./...               # run all tests
go test ./internal/...      # run tests in a package
go fmt ./...                # format code
go vet ./...                # static analysis
go mod tidy                 # clean up dependencies
```

### Frontend (React + Vite)
```bash
cd job-tracker-frontend
bun install
bun run dev                 # start dev server
bun run build               # production build
bun run lint
```

### MCP Server (Node.js)
```bash
cd jobspy-mcp-server
bun install
bun start                   # production
bun dev                     # development (nodemon)
bun lint / bun lint:fix
```

### Docker
```bash
docker compose up -d        # start all services (uses cloud-hosted Postgres)
docker compose down         # stop services
```

## Architecture

```
React Frontend → Go Backend API → MCP Server → JobSpy (multi-site job search)
                      ↓
                 PostgreSQL
```

**Data flow for job search**: Frontend form → `POST /api/jobs/search` on backend → backend calls MCP server at `MCP_SERVER_URL/api` → MCP executes JobSpy → results returned to frontend.

### Backend Structure (Go)

Layered architecture: **Handler → Service → Repository → Database**

```
backend/
├── cmd/server/         # main entry point
├── internal/
│   ├── domain/         # GORM models (Job, Attachment)
│   ├── handler/        # HTTP handlers (chi router)
│   ├── service/        # business logic
│   └── repository/     # GORM data access
└── pkg/
    ├── errors/         # custom error types
    └── response/       # standardized JSON responses
```

- GORM with PostgreSQL; auto-migration on startup
- UUID primary keys via `BeforeCreate` hooks
- Chi router with CORS middleware (all origins allowed)
- File attachments stored as BYTEA directly in PostgreSQL

### Frontend Structure (React)

Single-page app with React hooks for state management. No Redux — state lives in `App.jsx`. Axios service layer in `src/services/api.js`. Uses inline CSS-in-JS styling throughout (no CSS files/modules).

```
job-tracker-frontend/src/
├── App.jsx             # root state, tab/filter logic
├── services/api.js     # all Axios API calls
└── components/         # JobSearch, JobList, JobModal, StatusBadge
```

### MCP Server Structure (Node.js)

```
src/
├── index.js            # entry point, Express + SSE setup
├── tools/              # MCP tool implementations (job search)
├── prompts/            # AI prompt templates
└── schemas/            # Zod validation schemas
```

Supports SSE and Stdio transports. SSE enabled by default (`ENABLE_SSE=1`).

## API Endpoints

### Backend REST API (Port 8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/jobs` | List jobs (`?status=applied`) |
| POST | `/api/jobs` | Add job manually |
| GET | `/api/jobs/:id` | Get job by ID |
| PUT | `/api/jobs/:id` | Update job |
| DELETE | `/api/jobs/:id` | Delete job |
| PATCH | `/api/jobs/:id/status` | Update job status |
| POST | `/api/jobs/search` | Search via MCP and save |
| POST | `/api/jobs/:id/attachments` | Upload attachment |
| GET | `/api/jobs/:id/attachments` | List attachments |
| GET | `/api/jobs/:id/attachments/:id/download` | Download attachment |
| DELETE | `/api/jobs/:id/attachments/:id` | Delete attachment |
| GET | `/health` | Health check |

### MCP Server API (Port 9423)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api` | Search jobs via JobSpy |
| GET | `/health` | Health check |
| GET | `/sse` | SSE transport |
| POST | `/messages` | SSE message handling |

## Environment Variables

**Backend** (defaults shown):
```
SERVER_PORT=8080
DB_HOST=<your-db-host>
DB_PORT=<your-db-port>
DB_USER=jobuser
DB_PASSWORD=jobpass
DB_NAME=jobtracker
MCP_SERVER_URL=http://localhost:9423
```

**Frontend**:
- `.env.development`: `VITE_API_BASE=http://localhost:8081`
- Dev proxy in `vite.config.js` routes to `http://localhost:8080`

**MCP Server**:
```
JOBSPY_PORT=9423
JOBSPY_HOST=0.0.0.0
ENABLE_SSE=1
```

## Database

### Backup & Restore
```bash
# Backup (BYTEA columns included)
pg_dump -h localhost -U jobuser -d jobtracker -Fc -f jobtracker_backup.dump

# Restore
pg_restore -h localhost -U jobuser -d jobtracker -c jobtracker_backup.dump

# Or use scripts
./scripts/backup.sh
./scripts/restore.sh backups/jobtracker_20240220.dump

# Docker (if pg_dump not installed)
docker exec -it job-tracker-db pg_dump -U jobuser -d jobtracker -Fc -f /tmp/backup.dump
docker cp job-tracker-db:/tmp/backup.dump ./
```

## Code Style

### Go Backend

| Aspect | Rule |
|--------|------|
| **Indentation** | Tabs |
| **Imports** | stdlib → third-party → project |
| **Packages** | lowercase (`service`, `repository`) |
| **Types/Functions** | PascalCase (`JobService`), camelCase (`createJob`) |
| **DB Columns** | snake_case in tags, PascalCase in structs |
| **Error Handling** | Return errors last; use `fmt.Errorf("context: %w", err)` |
| **GORM** | Use `*gorm.DB` in `BeforeCreate` hook for UUIDs |

### React Frontend (JavaScript/JSX)

| Aspect | Rule |
|--------|------|
| **Indentation** | 2 spaces |
| **Quotes/Semicolons** | Single quotes, required |
| **Components** | PascalCase files (`.jsx`), camelCase functions |
| **Constants** | UPPER_SNAKE_CASE |
| **API** | Use axios, service modules in `src/services/` |

### MCP Server (Node.js)

| Aspect | Rule |
|--------|------|
| **Indentation** | 2 spaces |
| **Quotes/Semicolons** | Single quotes, required |
| **Modules** | ES6 `import`/`export` (no CommonJS) |
| **Validation** | Zod schemas in `src/schemas/` |
| **Errors** | Return error objects, never throw |
| **Logging** | Use winston logger |

**Job status values**: `new`, `viewed`, `applied`, `rejected`, `shortlisted`
