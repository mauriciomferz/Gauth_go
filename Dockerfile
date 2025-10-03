# Multi-stage Docker build for GAuth
# Build stage  
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk update && apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go modules files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY internal/ ./internal/

# Verify dependencies
RUN go mod verify

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gauth-server ./cmd/demo

# Verify binary
RUN ls -la gauth-server

# Production stage
FROM alpine:3.18.4

# Install runtime dependencies
RUN apk update && apk add --no-cache ca-certificates tzdata wget curl

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