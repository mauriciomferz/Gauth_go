# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# Install build dependencies
RUN apk add --no-cache git make

# Build the application
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gauth ./cmd/gauth

# Final stage
FROM alpine:latest

# Install CA certificates and other runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/gauth .

# Copy configs
COPY ./config /app/config

# Create non-root user
RUN adduser -D -g 'gauth' gauth \
    && chown -R gauth:gauth /app

USER gauth

# Expose ports
EXPOSE 8080 9090

# Set environment variables
ENV GAUTH_ENV=production \
    GAUTH_CONFIG_PATH=/app/config

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/gauth"]