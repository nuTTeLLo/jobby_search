# Job Tracker Backend

Go backend API for managing job applications with file attachments (resumes and cover letters).

## Quick Start

```bash
# Start PostgreSQL database
docker-compose up -d

# Run the server
go run cmd/server/main.go
```

The server will start on http://localhost:8080

## Configuration

Environment variables (can also be set in `.env` file):

| Variable       | Default               | Description       |
| -------------- | --------------------- | ----------------- |
| `DB_HOST`        | localhost             | PostgreSQL host   |
| `DB_PORT`        | 5432                  | PostgreSQL port   |
| `DB_USER`        | jobuser               | Database user     |
| `DB_PASSWORD`    | jobpass               | Database password |
| `DB_NAME`        | jobtracker            | Database name     |
| `SERVER_PORT`    | 8080                  | Backend port      |
| `MCP_SERVER_URL` | http://localhost:9423 | MCP server URL    |

## Database

PostgreSQL 16 with BYTEA storage for file attachments.

### Start Database

```bash
docker-compose up -d
```

### Stop Database

```bash
docker-compose down
```

## Database Backup & Restore

Since files (resumes/cover letters) are stored in the database (BYTEA), a single `pg_dump` captures everything.

### Manual Backup

```bash
# Full database dump (includes all attachments)
pg_dump -h localhost -U jobuser -d jobtracker -Fc -f jobtracker_backup.dump
```

### Manual Restore

```bash
pg_restore -h localhost -U jobuser -d jobtracker -c jobtracker_backup.dump
```

### Automated Backups

```bash
# Run backup script (keeps last 7 backups)
./scripts/backup.sh

# Restore from backup
./scripts/restore.sh backups/jobtracker_20240220_120000.dump
```

### Docker Backup (if pg_dump not installed locally)

```bash
# Backup
docker exec -it job-tracker-db pg_dump -U jobuser -d jobtracker -Fc -f /tmp/backup.dump
docker cp job-tracker-db:/tmp/backup.dump ./

# Restore
docker cp backup.dump job-tracker-db:/tmp/
docker exec -it job-tracker-db pg_restore -U jobuser -d jobtracker -c /tmp/backup.dump
```

## API Endpoints

### Jobs

| Method | Endpoint              | Description             |
| ------ | --------------------- | ----------------------- |
| GET    | `/api/jobs`           | List jobs               |
| POST   | `/api/jobs`           | Create job              |
| GET    | `/api/jobs/:id`       | Get job by ID           |
| PUT    | `/api/jobs/:id`       | Update job              |
| DELETE | `/api/jobs/:id`       | Delete job              |
| PATCH  | `/api/jobs/:id/status`| Update job status       |
| POST   | `/api/jobs/search`    | Search jobs via MCP    |

### Attachments

| Method | Endpoint                      | Description                |
| ------ | ----------------------------- | -------------------------- |
| POST   | `/api/jobs/:id/attachments`   | Upload resume/cover letter|
| GET    | `/api/jobs/:id/attachments`   | List job's attachments     |
| GET    | `/api/jobs/:id/attachments/:id` | Get attachment metadata |
| GET    | `/api/jobs/:id/attachments/:id/download` | Download file    |
| DELETE | `/api/jobs/:id/attachments/:id` | Delete attachment     |

### Upload Example

```bash
curl -X POST "http://localhost:8080/api/jobs/{job_id}/attachments" \
  -F "file=@/path/to/resume.pdf" \
  -F "file_type=resume"
```

## Development

```bash
# Build
go build -o job-tracker-backend ./cmd/server/main.go

# Run tests
go test ./...

# Format and vet
go fmt ./... && go vet ./... && go mod tidy
```
