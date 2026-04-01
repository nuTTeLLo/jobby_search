# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Jobby Search is a full-stack job tracking application with three services:
- **Backend**: Go REST API (port 8080)
- **Frontend**: React SPA (port 5173)
- **MCP Server**: Node.js job search service via JobSpy (port 9423)

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
npm install                 # install dependencies
npm run dev                 # start dev server
npm run build               # production build
npm run lint                # lint
npm run preview             # preview production build
```

### MCP Server (Node.js)
```bash
npm install                 # install dependencies
npm run dev                 # dev with nodemon
npm start                   # production start
npm run lint                # lint
npm run lint:fix            # auto-fix lint issues
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

## Environment Variables

**Backend** (defaults shown):
```
SERVER_PORT=8080
DB_HOST=localhost
DB_PORT=5432
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

## Code Style

- **Go**: tabs for indentation, `gofmt` enforced, camelCase for vars/funcs, PascalCase for exported
- **JavaScript/React**: 2-space indentation, Airbnb ESLint config
- **Job status values**: `new`, `viewed`, `applied`, `rejected`, `shortlisted`
