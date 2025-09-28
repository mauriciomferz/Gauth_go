# Gimel-App-0001 Web Application Container
FROM golang:1.23-alpine AS backend-builder

# Install dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy backend source
COPY web/backend/ ./

# Build backend
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o gimel-app-server main.go

# Multi-stage build for frontend (if needed)
FROM node:18-alpine AS frontend-builder

WORKDIR /app
COPY web/frontend/package*.json ./
RUN npm ci --production
COPY web/frontend/ ./
RUN npm run build || echo "No frontend build step"

# Final runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/gimel-app-server ./

# Copy web assets
COPY web/ ./web/

# Copy frontend build if exists
COPY --from=frontend-builder /app/build ./web/build/ 2>/dev/null || true

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Start application
CMD ["./gimel-app-server"]
