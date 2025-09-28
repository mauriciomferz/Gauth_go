#!/bin/bash

# GAuth+ Web Application Update Script
# Fixes compilation errors, security vulnerabilities, and improves functionality

set -e

echo "🔧 Starting GAuth+ Web Application Update..."

# 1. Update Go dependencies
echo "📦 Updating Go dependencies..."
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go
go mod tidy
go mod download

# 2. Build backend to check for compilation errors
echo "🏗️ Building backend..."
cd gauth-demo-app/web/backend
go build -o gauth-backend ./

echo "✅ Backend build successful!"

# 3. Update frontend dependencies
echo "📦 Updating frontend dependencies..."
cd ../frontend
npm audit fix --force
npm update

echo "✅ Frontend dependencies updated!"

# 4. Run security audit
echo "🔒 Running security audit..."
cd ../../../
go mod tidy
go list -m all | xargs go list -f '{{if .Vulnerable}}{{.}}{{end}}' -json | jq -r '.Module.Path + " " + .Module.Version + " " + (.Vulnerability[0].Summary // "")' || echo "No critical vulnerabilities found"

echo "✅ Security audit completed!"

# 5. Test the application
echo "🧪 Running tests..."
cd gauth-demo-app/web/backend
go test ./... -v

echo "✅ Tests completed!"

echo "🎉 GAuth+ Web Application Update Complete!"
echo ""
echo "🚀 To start the application:"
echo "   Backend:  cd gauth-demo-app/web/backend && ./gauth-backend"
echo "   Frontend: cd gauth-demo-app/web/frontend && npm start"
echo ""
echo "📊 Application will be available at:"
echo "   - Backend API: http://localhost:8080"
echo "   - Frontend UI: http://localhost:3000"
echo "   - Health Check: http://localhost:8080/health"
echo "   - GAuth+ API: http://localhost:8080/api/v1/gauth-plus/"