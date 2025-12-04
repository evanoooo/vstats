# Multi-stage build for vstats-server

# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci

# Copy source files
COPY web/ ./

# Build frontend
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.22-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY server-go/go.mod server-go/go.sum ./

# Download dependencies
RUN go mod download

# Copy source files
COPY server-go/ ./

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w" \
    -trimpath \
    -a -installsuffix cgo \
    -o vstats-server \
    .

# Stage 3: Final runtime image
FROM alpine:latest

WORKDIR /app

# Install ca-certificates and wget for health checks
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 vstats && \
    adduser -D -u 1000 -G vstats vstats

# Copy built binary from backend-builder
COPY --from=backend-builder /app/vstats-server /app/vstats-server

# Copy frontend dist from frontend-builder
COPY --from=frontend-builder /app/web/dist /app/web/dist

# Set environment variable for web directory
ENV VSTATS_WEB_DIR=/app/web/dist

# Create directories for config and database
RUN mkdir -p /app/data && \
    chown -R vstats:vstats /app

# Set default environment variables for data paths
ENV VSTATS_CONFIG_PATH=/app/data/vstats-config.json
ENV VSTATS_DB_PATH=/app/data/vstats.db

# Switch to non-root user
USER vstats

# Expose default port (can be overridden via config or env)
EXPOSE 3001

# Health check (uses default port 3001, actual port may differ based on config)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

# Run the server
CMD ["/app/vstats-server"]

