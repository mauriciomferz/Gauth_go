# React UI + Go Backend Integration Guide

This guide explains how to integrate the new React dashboard with the existing AgentAuth Go backend.

## 🔌 Integration Options

### Option 1: Development Proxy (Recommended for Dev)

**Current Setup**: Vite proxies `/api` requests to `localhost:8080`

```typescript
// vite.config.ts (already configured)
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
}
```

**Usage**:
1. Start Go backend: `./bin/web-server` (port 8080)
2. Start React dev server: `cd web/ui-react && npm run dev` (port 3000)
3. Access React UI at `http://localhost:3000`
4. API calls automatically proxy to Go backend

---

### Option 2: Serve React Build from Go (Production)

**Step 1: Build React App**

```bash
cd web/ui-react
npm run build
# Output: dist/ directory
```

**Step 2: Update Go Server**

Add the following to your Go web server (`cmd/web-server/main.go` or `web/server.go`):

```go
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed ui-react/dist/*
var reactUI embed.FS

func main() {
	// ... existing setup ...

	// Serve React UI
	reactDist, err := fs.Sub(reactUI, "ui-react/dist")
	if err != nil {
		log.Fatal("Failed to load React UI:", err)
	}
	
	// API routes (register BEFORE static file handler)
	http.HandleFunc("/api/", handleAPI)
	
	// Serve React SPA (fallback to index.html for client-side routing)
	http.Handle("/", spaHandler(http.FileServer(http.FS(reactDist))))
	
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// SPA handler ensures React Router works by serving index.html for non-API routes
func spaHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If path is /api, let it pass to API handlers
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			h.ServeHTTP(w, r)
			return
		}
		
		// For all other paths, try to serve the file
		// If file doesn't exist, serve index.html (SPA fallback)
		h.ServeHTTP(w, r)
	})
}
```

**Step 3: Rebuild Go Binary**

```bash
go build -o bin/web-server ./cmd/web-server
```

**Step 4: Run**

```bash
./bin/web-server
# Access at http://localhost:8080
```

---

### Option 3: Separate React Server with CORS (Alternative)

**Go Backend** (`cmd/web-server/main.go`):

```go
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	// ... register your API handlers ...
	
	handler := enableCORS(mux)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
```

**React Config** (`vite.config.ts`):

```typescript
export default defineConfig({
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

---

## 📡 API Endpoints Required

The React app expects these endpoints (already defined in `lib/api.ts`):

### Token Management
- `POST /api/v1/gauth/token` - Create extended token
- `POST /api/v1/gauth/token/validate` - Validate token
- `GET /api/v1/token/revocation/head` - Get revocation head

### Rotation
- `GET /api/v1/rotation/summary` - Get rotation summary

### Capability
- `GET /api/v1/capability/anchor/latest` - Get latest anchor

### Errors
- `GET /api/v1/errors/catalog` - Get error catalog

### Algorithms
- `GET /api/v1/beta/discovery` - Get supported algorithms

### PVP (Identity)
- `POST /api/v1/gauth/pvp/verify` - Verify identity chain

### Commercial Registry
- `POST /api/v1/gauth/registry/verify` - Verify entity
- `POST /api/v1/gauth/registry/signatory` - Verify signatory

### PIP (Authorization)
- `POST /api/v1/gauth/pip/authorize` - Validate authorization
- `GET /api/v1/gauth/pip/cache/stats` - Get cache statistics

### PoA (Power of Attorney)
- `POST /api/v1/gauth/poa` - Create PoA
- `POST /api/v1/gauth/poa/validate` - Validate PoA
- `GET /api/v1/gauth/poa/list` - List all PoAs

### Metrics
- `GET /api/v1/gauth/metrics` - Get system metrics
- `GET /api/v1/health` - Health check

---

## 🔐 Environment Variables

Create `.env` files for different environments:

**Development** (`.env.development`):
```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_APP_TITLE=AgentAuth 1.0 Dashboard (Dev)
```

**Production** (`.env.production`):
```env
VITE_API_BASE_URL=/api/v1
VITE_APP_TITLE=AgentAuth 1.0 Dashboard
```

**Usage in React**:
```typescript
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'
```

---

## 🐳 Docker Integration

**Dockerfile** (`web/ui-react/Dockerfile`):

```dockerfile
# Build stage
FROM node:18-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Production stage (nginx)
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

**Nginx Config** (`web/ui-react/nginx.conf`):

```nginx
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # Serve static files
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API requests to Go backend
    location /api/ {
        proxy_pass http://gauth-backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

**Docker Compose** (`docker-compose.yml`):

```yaml
version: '3.8'

services:
  gauth-backend:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
    networks:
      - gauth-network

  gauth-ui:
    build:
      context: ./web/ui-react
      dockerfile: Dockerfile
    ports:
      - "80:80"
    depends_on:
      - gauth-backend
    networks:
      - gauth-network

networks:
  gauth-network:
    driver: bridge
```

---

## 🔄 Development Workflow

### Daily Development

```bash
# Terminal 1: Go Backend
cd /path/to/Gauth_go
go run ./cmd/web-server

# Terminal 2: React UI
cd /path/to/Gauth_go/web/ui-react
npm run dev
```

Access at: `http://localhost:3000`

### Production Build

```bash
# Build React
cd web/ui-react
npm run build

# Build Go with embedded React
go build -o bin/web-server ./cmd/web-server

# Run
./bin/web-server
```

Access at: `http://localhost:8080`

---

## 🧪 Testing the Integration

### 1. Health Check

```bash
# Go backend
curl http://localhost:8080/api/v1/health

# React proxy
curl http://localhost:3000/api/v1/health
```

### 2. API Test

```bash
# Test token creation
curl -X POST http://localhost:3000/api/v1/gauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "test-client",
    "ownersAuthorizer": "HRB12345-DE",
    "clientOwner": "12345678-GB",
    "scope": ["read", "write"],
    "expirationHours": 24
  }'
```

### 3. UI Test

1. Open browser: `http://localhost:3000`
2. Navigate to "Extended Tokens"
3. Fill out form and submit
4. Check browser Network tab for API calls

---

## 🚨 Common Issues & Solutions

### Issue 1: API 404 Errors

**Symptom**: React app can't reach backend APIs

**Solution**: Verify proxy configuration in `vite.config.ts` and ensure Go backend is running on correct port

```bash
# Check Go server is running
curl http://localhost:8080/api/v1/health
```

### Issue 2: CORS Errors

**Symptom**: "Access-Control-Allow-Origin" errors in browser console

**Solution**: Add CORS middleware to Go backend (see Option 3 above)

### Issue 3: React Router 404s in Production

**Symptom**: Refreshing React routes gives 404

**Solution**: Ensure Go backend has SPA fallback handler (see Option 2 above)

### Issue 4: Slow Development Server

**Symptom**: Vite HMR is slow

**Solution**: Exclude `node_modules` in file watcher and reduce bundle size

```typescript
// vite.config.ts
export default defineConfig({
  server: {
    watch: {
      ignored: ['**/node_modules/**', '**/dist/**'],
    },
  },
})
```

---

## 📊 Performance Optimization

### Go Backend

```go
// Enable compression
import "github.com/gorilla/handlers"

handler := handlers.CompressHandler(mux)
```

### React Build

```bash
# Analyze bundle size
npm run build -- --mode production
npx vite-bundle-visualizer
```

---

## 🔒 Security Checklist

- [ ] HTTPS in production
- [ ] CORS properly configured
- [ ] API authentication headers
- [ ] Input validation on backend
- [ ] Rate limiting
- [ ] CSP headers
- [ ] No sensitive data in frontend
- [ ] Environment variables for secrets
- [ ] Secure cookies (HttpOnly, Secure, SameSite)

---

## 📝 Deployment Checklist

### Pre-Deployment

- [ ] Run `npm run build` successfully
- [ ] Test production build with `npm run preview`
- [ ] Run `go build` successfully
- [ ] Test all API endpoints
- [ ] Verify environment variables
- [ ] Check bundle size (< 200KB gzipped)
- [ ] Run linter: `npm run lint`
- [ ] Run formatter: `npm run format`

### Deployment

- [ ] Build React app
- [ ] Embed in Go binary or deploy separately
- [ ] Configure reverse proxy (nginx/traefik)
- [ ] Setup SSL certificates
- [ ] Configure monitoring
- [ ] Setup logging
- [ ] Test in production environment

### Post-Deployment

- [ ] Verify all routes work
- [ ] Check API integration
- [ ] Test dark mode toggle
- [ ] Verify responsive design
- [ ] Check browser console for errors
- [ ] Monitor performance metrics

---

## 🎯 Next Steps

1. **Implement Remaining Features**
   - Complete form implementations in each page
   - Add real API integration
   - Implement data fetching hooks

2. **Add Real-Time Updates**
   - WebSocket connection
   - Live metrics updates
   - Toast notifications

3. **Testing**
   - Unit tests with Jest
   - E2E tests with Playwright
   - API integration tests

4. **Documentation**
   - Component Storybook
   - API documentation site
   - User guide

---

## 📚 Resources

- [Vite Proxy Docs](https://vitejs.dev/config/server-options.html#server-proxy)
- [Go embed Package](https://pkg.go.dev/embed)
- [React Router Docs](https://reactrouter.com/)
- [Tailwind CSS Docs](https://tailwindcss.com/)

---

**Questions?** Check the main `README.md` or create an issue in the repository.

---

*Last Updated: November 12, 2025*  
*AgentAuth 1.0 React Dashboard Integration Guide*
