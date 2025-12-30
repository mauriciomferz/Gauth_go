# Multi-stage Docker build for AgentAuth
# Build stage - REQUIRES Go 1.25.5 for security patches (CVE fixes)
FROM golang:1.25.5-alpine AS builder

# Install build dependencies
RUN apk update && apk add --no-cache git ca-certificates tzdata build-base gcc abuild binutils binutils-doc gcc-doc linux-headers pkgconf

# Set working directory
WORKDIR /app

# Copy go modules files first for better caching
COPY go.mod go.sum ./

# Download dependencies with verbose output
RUN echo "=== Downloading Go modules ===" && \
    go mod download && \
    echo "=== Go modules downloaded successfully ==="

# Copy source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY internal/ ./internal/
COPY web/ ./web/

# Verify dependencies
RUN echo "=== Verifying Go modules ===" && \
    go mod verify && \
    echo "=== Go modules verified successfully ==="

# Verify source files are present
RUN echo "=== Checking source files ===" && \
    ls -la ./cmd/web-server/ && \
    echo "=== Source files verified ==="

# Build the applications with verbose output
RUN echo "=== Starting Go build ===" && \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags='-w -s' \
    -o gauth-server ./cmd/web-server && \
    echo "=== Build completed successfully ==="

# Verify binary
RUN echo "=== Verifying binary ===" && \
    ls -la gauth-server && \
    file gauth-server && \
    echo "=== Binary verified ==="

# Production stage
FROM alpine:3.18.4

# Install runtime dependencies
RUN apk update && apk add --no-cache ca-certificates tzdata wget curl libstdc++ libgcc

# Create non-root user
RUN adduser -D -s /bin/sh gauth

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gauth-server .

# Create necessary directories
RUN mkdir -p ./configs ./logs && \
    chown -R gauth:gauth /app

# Switch to non-root user
USER gauth

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command (can be overridden)
CMD ["./gauth-server"]
