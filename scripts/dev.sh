#!/bin/bash

set -o pipefail

CONTAINER_NAME="jobspy-mcp-dev"
IMAGE_NAME="jobspy-mcp-server"
MCP_PORT=9423
HEALTH_CHECK_URL="http://localhost:${MCP_PORT}/health"
MAX_WAIT_SECONDS=30

# PostgreSQL config
POSTGRES_CONTAINER="job-tracker-db-dev"
POSTGRES_PORT=5432

# Check if required ports are available
check_port() {
    local port=$1
    local name=$2
    if lsof -i :$port >/dev/null 2>&1; then
        echo "Warning: Port $port ($name) is already in use"
        return 1
    else
        echo "Port $port ($name) is available"
        return 0
    fi
}

echo "Checking port availability..."
check_port 8080 "backend" || true
check_port 5173 "frontend" || true
check_port $MCP_PORT "MCP" || true
check_port 5432 "PostgreSQL" || true
echo ""

# Start PostgreSQL container for dev
start_postgres() {
    echo "Starting PostgreSQL container..."
    if podman ps --format "{{.Names}}" | grep -q "^${POSTGRES_CONTAINER}$"; then
        echo "PostgreSQL container already running"
    elif podman ps -a --format "{{.Names}}" | grep -q "^${POSTGRES_CONTAINER}$"; then
        echo "Starting existing PostgreSQL container..."
        podman start "$POSTGRES_CONTAINER"
    else
        echo "Creating new PostgreSQL container..."
        podman run -d --name "$POSTGRES_CONTAINER" \
            -e POSTGRES_USER=jobuser \
            -e POSTGRES_PASSWORD=jobpass \
            -e POSTGRES_DB=jobtracker \
            -p ${POSTGRES_PORT}:5432 \
            -v ./data/postgres_data:/var/lib/postgresql/data \
            postgres:16-alpine
    fi
    
    # Wait for PostgreSQL to be ready
    echo "Waiting for PostgreSQL..."
    while ! podman exec "$POSTGRES_CONTAINER" pg_isready -U jobuser -d jobtracker 2>/dev/null; do
        sleep 1
    done
    echo "PostgreSQL is ready!"
}

stop_postgres() {
    echo "Stopping PostgreSQL container..."
    podman stop "$POSTGRES_CONTAINER" 2>/dev/null || true
}

# Cleanup function - called on EXIT, SIGINT, and SIGTERM
cleanup() {
    echo "Cleaning up..."
    
    # Kill background processes first
    if [ -n "$BACKEND_PID" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
        echo "Stopping backend (PID: $BACKEND_PID)..."
        kill -TERM "$BACKEND_PID" 2>/dev/null || true
    fi
    
    if [ -n "$FRONTEND_PID" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
        echo "Stopping frontend (PID: $FRONTEND_PID)..."
        kill -TERM "$FRONTEND_PID" 2>/dev/null || true
    fi
    
    # Wait for processes to terminate (max 5 seconds)
    wait "$BACKEND_PID" 2>/dev/null || true
    wait "$FRONTEND_PID" 2>/dev/null || true
    
    # Stop and remove MCP container
    echo "Stopping MCP container..."
    podman stop "$CONTAINER_NAME" 2>/dev/null || podman rm -f "$CONTAINER_NAME" 2>/dev/null || true
    
    # Stop PostgreSQL
    stop_postgres
    
    echo "Cleanup complete"
}

# Set up signal handlers - cleanup runs on EXIT, SIGINT (Ctrl+C), and SIGTERM
trap cleanup EXIT INT TERM

# Check if container already exists and handle it
echo "Checking for existing MCP container..."
if podman ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
    echo "Container '$CONTAINER_NAME' already exists. Removing stale container..."
    podman rm -f "$CONTAINER_NAME" 2>/dev/null || true
fi

echo "Starting MCP container..."
podman run -d --name "$CONTAINER_NAME" -p ${MCP_PORT}:9423 "$IMAGE_NAME"

echo "Waiting for MCP container to be healthy..."
elapsed=0
while [ $elapsed -lt $MAX_WAIT_SECONDS ]; do
    if curl -sf "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
        echo "MCP container is healthy!"
        break
    fi
    sleep 1
    elapsed=$((elapsed + 1))
    echo "Waiting for health check... ($elapsed/${MAX_WAIT_SECONDS}s)"
done

if [ $elapsed -ge $MAX_WAIT_SECONDS ]; then
    echo "Error: MCP container failed to become healthy within ${MAX_WAIT_SECONDS} seconds"
    exit 1
fi

# Start PostgreSQL
start_postgres

echo "Starting backend and frontend..."
cd backend
go run cmd/server/main.go &
BACKEND_PID=$!
cd ..

cd job-tracker-frontend
pnpm run dev &
FRONTEND_PID=$!
cd ..

echo "Backend PID: $BACKEND_PID"
echo "Frontend PID: $FRONTEND_PID"
echo "All services started. Press Ctrl+C to stop."

# Wait for any background job to exit
while kill -0 $BACKEND_PID 2>/dev/null && kill -0 $FRONTEND_PID 2>/dev/null; do
  sleep 1
done
