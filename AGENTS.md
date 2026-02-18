# AGENTS.md - Job Tracker Application

This file contains instructions for agentic coding agents working on the Job Tracker application codebase.

## Project Overview

This is a job tracking application with three main components:
1. **MCP Server** (`src/`) - Node.js Model Context Protocol server for JobSpy integration
2. **Go Backend** (`backend/`) - REST API server with SQLite database
3. **React Frontend** (`job-tracker-frontend/`) - Single-page application

---

## Important: Use pnpm Instead of npm

This project uses **pnpm** instead of npm due to symlink preservation for the node_modules. Using npm may break the symlink setup.

```bash
# Install pnpm if needed
npm install -g pnpm

# Use pnpm for all package management
pnpm install
pnpm start
pnpm run dev
```

---

## Build, Lint, and Test Commands

### MCP Server (Node.js)

```bash
# Install dependencies
npm install

# Start production server
npm start

# Start development server with auto-reload
npm run dev

# Run ESLint
npm run lint

# Auto-fix ESLint issues
npm run lint:fix

# Test MCP server directly
curl -X POST "http://localhost:9423/api" \
  -H "Content-Type: application/json" \
  -d '{"method":"search_jobs","params":{"search_term":"software engineer","location":"remote","site_names":"indeed"}}'
```

### Go Backend

```bash
# Navigate to backend
cd backend

# Build the application
go build -o job-tracker-backend ./cmd/server/main.go

# Run the server
go run cmd/server/main.go

# Run with custom port
SERVER_PORT=8081 go run cmd/server/main.go

# Run tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -v ./internal/repository/...

# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Tidy dependencies
go mod tidy
```

### React Frontend

```bash
# Navigate to frontend
cd job-tracker-frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Run linter
npm run lint
```

---

## Code Style Guidelines

### Go Backend

#### Formatting
- **Indentation**: Use tabs (Go default)
- **Line Length**: No strict limit, but keep lines reasonable
- **Imports**: Group standard library first, then third-party, then project imports
  ```go
  import (
      "fmt"
      "io"
      "net/http"

      "github.com/go-chi/chi/v5"
      "gorm.io/gorm"

      "job-tracker-backend/internal/domain"
  )
  ```

#### Naming Conventions
- **Packages**: Lowercase, short names (`service`, `repository`, `handler`)
- **Types/Interfaces**: PascalCase (`JobService`, `JobRepository`)
- **Functions/Variables**: camelCase (`createJob`, `jobService`)
- **Constants**: PascalCase for exported, camelCase for unexported
- **File Names**: Lowercase with underscores (`job_service.go`, `job_handler.go`)
- **DB Columns**: Snake_case in tags, PascalCase in structs
  ```go
  type Job struct {
      ID          string    `json:"id" gorm:"primaryKey"`
      JobTitle    string    `json:"job_title" gorm:"not null"`
      CreatedAt   time.Time `json:"created_at"`
  }
  ```

#### Error Handling
- Return errors as last return value
- Use wrapped errors with `fmt.Errorf("context: %w", err)`
- Define custom errors in `pkg/errors/`
- Check errors immediately with early return

#### Project Structure
```
backend/
├── cmd/server/          # Entry points
│   └── main.go
├── internal/
│   ├── config/          # Configuration
│   ├── domain/          # Business entities
│   ├── handler/         # HTTP handlers
│   ├── repository/      # Database operations
│   └── service/         # Business logic
└── pkg/
    ├── errors/          # Custom error types
    └── response/        # Response helpers
```

#### GORM Conventions
- Use `*gorm.DB` in BeforeCreate hook:
  ```go
  func (j *Job) BeforeCreate(tx *gorm.DB) error { ... }
  ```
- Use GORM tags for column definitions
- Implement `BeforeCreate` hook for auto-generating UUIDs

---

### React Frontend (JavaScript/JSX)

#### Formatting
- **Indentation**: 2 spaces
- **Quotes**: Single quotes for strings
- **Semicolons**: Required at end of statements
- **File Extensions**: `.jsx` for components, `.js` for utilities

#### Naming Conventions
- **Components**: PascalCase (`JobList`, `JobModal`)
- **Functions/Variables**: camelCase (`handleSubmit`, `formData`)
- **Constants**: UPPER_SNAKE_CASE for true constants
- **File Names**: PascalCase for components (`JobList.jsx`), camelCase for utilities (`api.js`)

#### Component Structure
```jsx
import { useState, useEffect } from 'react';
import api from '../services/api';

export default function ComponentName({ prop1, onAction }) {
  const [state, setState] = useState(null);

  useEffect(() => {
    // side effects
  }, []);

  const handler = () => {
    // event handlers
  };

  return (
    <div>
      {/* JSX */}
    </div>
  );
}

const styles = {
  // inline styles object
};
```

#### API Patterns
- Use axios for HTTP requests
- Create service modules in `src/services/`
- Handle errors with try/catch and display user-friendly messages

---

### MCP Server (Node.js)

#### Formatting
- **Indentation**: 2 spaces
- **Quotes**: Single quotes for strings
- **Semicolons**: Required

#### Naming Conventions
- **Functions/Variables**: camelCase
- **Constants**: UPPER_SNAKE_CASE
- **Files**: kebab-case or camelCase

#### ES6 Modules
- Use `import`/`export` syntax (no CommonJS)
- `"type": "module"` in package.json

---

## Database

### SQLite
- Database file: `backend/data/jobs.db`
- Auto-created on first run via GORM auto-migration

### Schema
```sql
-- Jobs table (auto-created by GORM)
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
| GET | `/api/jobs` | List jobs (query: `?status=applied`) |
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

### Backend
- `SERVER_PORT`: Server port (default: 8080)
- `DATABASE_PATH`: SQLite file path (default: `./data/jobs.db`)
- `MCP_SERVER_URL`: MCP server URL (default: `http://localhost:9423`)

### MCP Server
- `JOBSPY_PORT`: Server port (default: 9423)
- `JOBSPY_HOST`: Host (default: 0.0.0.0)
- `ENABLE_SSE`: Enable SSE transport (default: 0)

---

## Testing the Application

1. Start MCP Server: `cd jobspy-mcp-server && npm start`
2. Start Backend: `cd backend && go run cmd/server/main.go`
3. Start Frontend: `cd job-tracker-frontend && npm run dev`

Test job search:
```bash
curl -X POST "http://localhost:8080/api/jobs/search" \
  -H "Content-Type: application/json" \
  -d '{"site_names":"indeed","search_term":"software engineer","location":"remote","results_wanted":5}'
```
