# AGENTS.md - Job Tracker Application

Three components: MCP Server (`jobspy-mcp-server/`), Go Backend (`backend/`), React Frontend (`job-tracker-frontend/`).

> **Important**: Use **pnpm** instead of npm for Node.js package management.

---

## Commands

### MCP Server (Node.js)
```bash
cd jobspy-mcp-server
pnpm install
pnpm start              # Production
pnpm dev                # Development (nodemon)
pnpm lint / pnpm lint:fix
```

### Go Backend (PostgreSQL)
```bash
cd backend
docker-compose up -d  # Start PostgreSQL (port 5432)
go build -o job-tracker-backend ./cmd/server/main.go
go run cmd/server/main.go
SERVER_PORT=8081 go run cmd/server/main.go  # Custom port
go test ./...              # All tests
go test -v ./internal/repository/...  # Single test
go fmt ./... && go vet ./... && go mod tidy
```

### React Frontend
```bash
cd job-tracker-frontend
npm install
npm run dev     # Development
npm run build   # Production
npm run lint
```

---

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

---

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

### Environment Variables

| Variable       | Default               | Description       |
| -------------- | --------------------- | ----------------- |
| `DB_HOST`        | localhost             | PostgreSQL host   |
| `DB_PORT`        | 5432                  | PostgreSQL port   |
| `DB_USER`        | jobuser               | Database user     |
| `DB_PASSWORD`    | jobpass               | Database password |
| `DB_NAME`        | jobtracker            | Database name     |
| `SERVER_PORT`    | 8080                  | Backend port      |
| `MCP_SERVER_URL` | http://localhost:9423 | MCP server URL    |

---

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

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api` | POST | Search jobs via JobSpy |
| `/health` | GET | Health check |
| `/sse` | GET | SSE transport |
| `/messages` | POST | SSE message handling |
