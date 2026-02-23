# Build stage
FROM golang:1.25 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO
RUN CGO_ENABLED=1 GOOS=linux go build -o job-tracker-backend ./cmd/server/main.go

# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies (PostgreSQL client instead of SQLite)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libpq-dev \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/job-tracker-backend .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./job-tracker-backend"]
