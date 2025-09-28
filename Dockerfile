# Build stage
FROM golang:1.23.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set the working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Copy the local module dependencies (backend module)
COPY gauth-demo-app/web/backend/go.mod ./gauth-demo-app/web/backend/
COPY gauth-demo-app/web/backend/go.sum ./gauth-demo-app/web/backend/

# Copy the backend source code needed for local dependencies
COPY gauth-demo-app/web/backend/ ./gauth-demo-app/web/backend/

# Download dependencies (with local modules available)
RUN go mod download

# Copy the rest of the source code
COPY . .

# Verify the build can complete
RUN go mod verify

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gauth-server ./cmd/demo

# Final stage
FROM alpine:3.18.4

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user
RUN adduser -D -s /bin/sh gauth

# Set the working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/gauth-server .

# Create configs directory
RUN mkdir -p ./configs

# Change ownership to non-root user
RUN chown -R gauth:gauth /app
USER gauth

# Expose the port the app runs on
EXPOSE 8080

# Install wget for health checks
RUN apk --no-cache add wget

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Command to run the application
CMD ["./gauth-server"]