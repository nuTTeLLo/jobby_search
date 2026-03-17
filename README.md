# Job Tracker Application

A full-stack job tracking application that helps you manage your job search journey. Search for jobs across multiple platforms, track applications, store attachments, and monitor your progress — all in one place.

## Features

- **Job Search**: Search for jobs across Indeed, LinkedIn, Glassdoor, and other platforms via the MCP server
- **Application Tracking**: Track job applications with status (applied, interviewing, offer, rejected, etc.)
- **Attachment Management**: Upload and store resumes, cover letters, and other documents
- **RESTful API**: Full backend API for managing jobs and attachments
- **Modern Frontend**: React-based user interface with React Router for navigation

## Tech Stack

### Backend (Go)
- **Framework**: chi/v5 router
- **ORM**: GORM
- **Database**: PostgreSQL
- **Utilities**: google/uuid

### Frontend (React)
- **Framework**: React 19
- **Routing**: React Router 7
- **Build Tool**: Vite 7
- **HTTP Client**: Axios

### MCP Server (Node.js)
- **Protocol**: @modelcontextprotocol/sdk
- **Server**: Express
- **Logging**: Winston
- **Validation**: Zod

## Project Structure

```
jobby_search/
├── backend/              # Go REST API server
├── job-tracker-frontend/ # React frontend application
├── jobspy-mcp-server/   # MCP server for job searching
├── scripts/              # Database utility scripts
└── docker-compose.yml   # PostgreSQL container setup
```

## Prerequisites

- Go 1.21+
- Node.js 18+
- Bun (for Node.js package management)
- PostgreSQL (via Docker)
- Docker & Docker Compose

## Installation & Setup

### 1. Clone the repository

```bash
git clone <repository-url>
cd jobby_search
```

### 2. Start PostgreSQL

```bash
cd backend
docker-compose up -d
```

### 3. Setup MCP Server

```bash
cd jobspy-mcp-server
bun install
bun start              # Production
# or
bun dev                # Development
```

### 4. Setup Backend

```bash
cd backend
go build -o job-tracker-backend ./cmd/server/main.go
go run cmd/server/main.go
```

To use a custom port:
```bash
SERVER_PORT=8081 go run cmd/server/main.go
```

### 5. Setup Frontend

```bash
cd job-tracker-frontend
bun install
bun run dev
```

## API Endpoints

### Backend REST API (Port 8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/jobs` | List jobs (supports `?status=applied`) |
| POST | `/api/jobs` | Add job manually |
| GET | `/api/jobs/:id` | Get job by ID |
| PUT | `/api/jobs/:id` | Update job |
| DELETE | `/api/jobs/:id` | Delete job |
| PATCH | `/api/jobs/:id/status` | Update job status |
| POST | `/api/jobs/search` | Search via MCP and save results |
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

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | kickapoo.tailee323f.ts.net | PostgreSQL host |
| `DB_PORT` | 30432 | PostgreSQL port |
| `DB_USER` | jobuser | Database user |
| `DB_PASSWORD` | jobpass | Database password |
| `DB_NAME` | jobtracker | Database name |
| `SERVER_PORT` | 8080 | Backend server port |
| `MCP_SERVER_URL` | http://localhost:9423 | MCP server URL |

## Database Management

### Backup

```bash
pg_dump -h localhost -U jobuser -d jobtracker -Fc -f jobtracker_backup.dump

# Or use the provided script
./scripts/backup.sh
```

### Restore

```bash
pg_restore -h localhost -U jobuser -d jobtracker -c jobtracker_backup.dump

# Or use the provided script
./scripts/restore.sh backups/jobtracker_20240220.dump
```

### Via Docker

```bash
docker exec -it job-tracker-db pg_dump -U jobuser -d jobtracker -Fc -f /tmp/backup.dump
docker cp job-tracker-db:/tmp/backup.dump ./
```

## Development Commands

### MCP Server
```bash
cd jobspy-mcp-server
bun install
bun start              # Production
bun dev                # Development
bun lint / bun lint:fix
```

### Go Backend
```bash
cd backend
go build -o job-tracker-backend ./cmd/server/main.go
go run cmd/server/main.go
go test ./...
go fmt ./... && go vet ./... && go mod tidy
```

### React Frontend
```bash
cd job-tracker-frontend
bun install
bun run dev
bun run build
bun run lint
```

## Credits

- [JobSpy](https://github.com/speedyapply/JobSpy) - Python library for searching jobs across Indeed, LinkedIn, Glassdoor, Google, ZipRecruiter & more
- [JobSpy MCP Server](https://github.com/borgius/jobspy-mcp-server) - Model Context Protocol server this project was forked from

## License

MIT
