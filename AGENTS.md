# AGENTS.md - Job Tracker Application

Three components: MCP Server (`jobspy-mcp-server/`), Go Backend (`backend/`), React Frontend (`job-tracker-frontend/`).

> **Important**: Use **pnpm** instead of npm for all Node.js package management (symlink preservation for nested node_modules).

---

## Commands

### MCP Server (Node.js)
```bash
cd jobspy-mcp-server
pnpm install
pnpm start              # Production
pnpm dev                # Development (nodemon)
pnpm lint / pnpm lint:fix
# Test directly
curl -X POST "http://localhost:9423/api" -H "Content-Type: application/json" \
  -d '{"method":"search_jobs","params":{"search_term":"software engineer","location":"remote","site_names":"indeed"}}'
```

### Go Backend
```bash
cd backend
go build -o job-tracker-backend ./cmd/server/main.go
go run cmd/server/main.go
SERVER_PORT=8081 go run cmd/server/main.go  # Custom port
go test ./...              # All tests
go test -v ./...           # Verbose
go test -v ./internal/repository/...  # Specific test
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

```go
import (
    "fmt"
    "net/http"

    "github.com/go-chi/chi/v5"
    "gorm.io/gorm"

    "job-tracker-backend/internal/domain"
)
```

### React Frontend (JavaScript/JSX)

| Aspect | Rule |
|--------|------|
| **Indentation** | 2 spaces |
| **Quotes** | Single quotes |
| **Semicolons** | Required |
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
| **Errors** | Return error objects, never throw (prevents crashes) |
| **Logging** | Use winston logger |

```javascript
// Order: built-ins → external → local
import fs from 'fs';
import { z } from 'zod';
import { searchParams } from '../schemas/searchParamsSchema.js';
```

---

## Database

**SQLite**: `backend/data/jobs.db` (auto-created via GORM)

```sql
CREATE TABLE jobs (
    id VARCHAR(36) PRIMARY KEY,
    job_title VARCHAR(500) NOT NULL,
    company_name VARCHAR(500),
    location VARCHAR(500),
    job_url VARCHAR(2000) UNIQUE,
    description TEXT,
    salary VARCHAR(200),
    job_type VARCHAR(100),
    is_remote BOOLEAN DEFAULT 0,
    source VARCHAR(100),
    status VARCHAR(50) DEFAULT 'new',
    notes TEXT,
    created_at DATETIME,
    updated_at DATETIME
);
```

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
| GET | `/health` | Health check |

### MCP Server API (Port 9423)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api` | POST | Search jobs via JobSpy |
| `/health` | GET | Health check |
| `/sse` | GET | SSE transport |
| `/messages` | POST | SSE message handling |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | Backend port |
| `DATABASE_PATH` | `./data/jobs.db` | SQLite file path |
| `MCP_SERVER_URL` | `http://localhost:9423` | MCP server URL |
| `JOBSPY_PORT` | 9423 | MCP server port |
| `JOBSPY_HOST` | 0.0.0.0 | MCP server host |
| `ENABLE_SSE` | 0 | Enable SSE transport |

---

## Testing

```bash
# Start all services
cd jobspy-mcp-server && pnpm start
cd backend && go run cmd/server/main.go
cd job-tracker-frontend && npm run dev

# Test job search
curl -X POST "http://localhost:8080/api/jobs/search" \
  -H "Content-Type: application/json" \
  -d '{"site_names":"indeed","search_term":"software engineer","location":"remote","results_wanted":5}'
```
