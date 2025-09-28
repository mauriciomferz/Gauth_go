# Build stage
FROM golang:1.23.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata sed

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./

# Remove the problematic local module dependency that's not needed for cmd/demo
# The gauth-demo-app directory is excluded via .dockerignore to prevent cache key issues
RUN sed -i '/github.com\/Gimel-Foundation\/gauth\/gauth-demo-app\/web\/backend/d' go.mod && \
    sed -i '/replace.*gauth-demo-app.*web.*backend/d' go.mod

# Download dependencies (without the local backend module)
RUN go mod download

# Copy the source code (gauth-demo-app directory excluded via .dockerignore)
COPY . ./

# Verify dependencies
RUN go mod verify

# Build the demo application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gauth-server ./cmd/demo

# Verify the binary was created successfully
RUN ls -la gauth-server

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