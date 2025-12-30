#!/bin/bash

# AgentAuth React UI - Quick Setup Script
# This script installs dependencies and starts the development server

set -e

echo "🚀 AgentAuth React UI - Quick Setup"
echo "================================"
echo ""

# Check Node.js version
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ first."
    echo "   Visit: https://nodejs.org/"
    exit 1
fi

NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo "⚠️  Node.js version $NODE_VERSION detected. Version 18+ is recommended."
fi

echo "✅ Node.js $(node -v) detected"
echo "✅ npm $(npm -v) detected"
echo ""

# Install dependencies
echo "📦 Installing dependencies..."
echo "   This may take a few minutes..."
echo ""

npm install

if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Dependency installation failed."
    echo "   Try: rm -rf node_modules package-lock.json && npm install"
    exit 1
fi

echo ""
echo "✅ Dependencies installed successfully!"
echo ""

# Type check
echo "🔍 Running type check..."
npm run type-check

if [ $? -eq 0 ]; then
    echo "✅ TypeScript type check passed!"
else
    echo "⚠️  TypeScript warnings detected (this is normal for new projects)"
fi

echo ""
echo "================================"
echo "✅ Setup Complete!"
echo "================================"
echo ""
echo "Available commands:"
echo "  npm run dev       - Start development server (port 3000)"
echo "  npm run build     - Build for production"
echo "  npm run preview   - Preview production build"
echo "  npm run format    - Format code with Prettier"
echo ""
echo "Next steps:"
echo "  1. Start the dev server:    npm run dev"
echo "  2. Start the Go backend:    go run ./cmd/web-server"
echo "  3. Open browser:            http://localhost:3000"
echo ""
echo "📚 Documentation:"
echo "  - README.md              - Full documentation"
echo "  - QUICK_START.md         - Quick start guide"
echo "  - INTEGRATION_GUIDE.md   - Backend integration"
echo ""

# Ask if user wants to start dev server
read -p "Start development server now? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo "🚀 Starting development server..."
    echo "   Press Ctrl+C to stop"
    echo ""
    npm run dev
fi
