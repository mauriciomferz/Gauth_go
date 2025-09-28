# 🚀 Gimel-App-0001 Installation Guide

## Quick Install & Run (30 seconds)

```bash
# 1. Clone the repository
git clone https://github.com/Gimel-Foundation/Gimel-App-0001.git
cd Gimel-App-0001

# 2. Run the application
./deploy-web-app.sh standalone

# 3. Open in browser
open http://localhost:3000/standalone-demo.html
```

## Requirements
- Go 1.23+ (for backend)
- Python 3.8+ (for demo server)
- Modern web browser

## Deployment Options

### 🎯 Standalone Demo
Perfect for presentations and testing:
```bash
./deploy-web-app.sh standalone
```

### 🛠️ Development Environment  
Full development setup with auto-reload:
```bash
./deploy-web-app.sh development
```

### 🏭 Production Deployment
Optimized for production use:
```bash
./deploy-web-app.sh production
```

### 🐳 Docker Deployment
Container-based deployment:
```bash
docker-compose up -d
```

## Verification
After deployment, test the application:
```bash
# Check API health
curl http://localhost:8080/health

# Access interactive demo
open http://localhost:3000/standalone-demo.html
```

**Success**: You should see 100% test success rate in the demo interface.

## Support
- 📚 Complete documentation in README.md
- 🔧 Development guide in DEVELOPMENT.md  
- 📊 Project status in PROJECT_STATUS.md
- 🌐 API reference in API_REFERENCE.md
