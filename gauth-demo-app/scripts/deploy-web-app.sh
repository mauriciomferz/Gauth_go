#!/bin/bash

# 🌐 Gimel-App-0001 Deployment Script
# Optimized for web application deployment

set -e

echo "🌐 Deploying Gimel-App-0001 Web Application..."

# Check for required dependencies
command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed. Aborting." >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "❌ Python 3 is required but not installed. Aborting." >&2; exit 1; }

# Determine deployment mode
MODE=${1:-standalone}

case $MODE in
    "standalone")
        echo "🎯 Standalone Demo Mode"
        
        # Build backend
        echo "🔧 Building Go backend..."
        cd web/backend
        go mod tidy
        go build -o ../../web-server main.go
        cd ../..
        
        # Start services
        echo "🚀 Starting services..."
        ./web-server &
        BACKEND_PID=$!
        
        cd web
        python3 -m http.server 3000 &
        FRONTEND_PID=$!
        cd ..
        
        echo "✅ Services started!"
        echo "🌐 Backend API: http://localhost:8080"
        echo "🎨 Frontend: http://localhost:3000"
        echo "🧪 Interactive Demo: http://localhost:3000/standalone-demo.html"
        echo ""
        echo "Press Ctrl+C to stop all services"
        
        trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit" INT TERM EXIT
        wait
        ;;
        
    "development")
        echo "🛠️ Development Mode"
        
        # Start backend with auto-reload
        echo "🔧 Starting backend (auto-reload)..."
        cd web/backend
        go mod tidy
        go run main.go &
        BACKEND_PID=$!
        cd ../..
        
        # Start frontend development server
        if [ -d "web/frontend" ]; then
            echo "🎨 Starting React development server..."
            cd web/frontend
            if [ ! -d "node_modules" ]; then
                npm install
            fi
            npm start &
            REACT_PID=$!
            cd ../..
        fi
        
        # Start static file server
        echo "📁 Starting static file server..."
        cd web
        python3 -m http.server 3000 &
        STATIC_PID=$!
        cd ..
        
        echo "✅ Development environment ready!"
        echo "🌐 Backend API: http://localhost:8080"
        echo "🎨 Frontend: http://localhost:3000"
        echo "🧪 Demo: http://localhost:3000/standalone-demo.html"
        
        if [ -n "$REACT_PID" ]; then
            trap "kill $BACKEND_PID $REACT_PID $STATIC_PID 2>/dev/null; exit" INT TERM EXIT
        else
            trap "kill $BACKEND_PID $STATIC_PID 2>/dev/null; exit" INT TERM EXIT
        fi
        wait
        ;;
        
    "production")
        echo "🏭 Production Mode"
        
        # Build optimized backend
        echo "🔧 Building production backend..."
        cd web/backend
        go mod tidy
        CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ../../gimel-app-server main.go
        cd ../..
        
        # Build frontend if exists
        if [ -f "web/frontend/package.json" ]; then
            echo "🎨 Building production frontend..."
            cd web/frontend
            npm ci --production
            npm run build
            cd ../..
        fi
        
        # Create production directory
        echo "📦 Creating production package..."
        mkdir -p dist
        cp gimel-app-server dist/
        cp -r web/* dist/
        
        # Create production startup script
        cat > dist/start.sh << 'PRODEOF'
#!/bin/bash
export GIN_MODE=release
export PORT=8080
./gimel-app-server
PRODEOF
        chmod +x dist/start.sh
        
        echo "✅ Production build complete!"
        echo "📦 Production files in: ./dist/"
        echo "🚀 Start with: cd dist && ./start.sh"
        ;;
        
    *)
        echo "❌ Unknown mode: $MODE"
        echo "Usage: $0 [standalone|development|production]"
        exit 1
        ;;
esac
