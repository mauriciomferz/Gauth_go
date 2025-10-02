# 🎯 GAuth Web Demo Application

**Modern Interactive Demo for RFC 111 & 115 Implementation**

A comprehensive web application demonstrating the complete GAuth authorization system with RFC compliance, legal framework integration, and real-time interaction capabilities.

## 🚀 **Quick Start**

### **Option 1: Single Command Startup**
```bash
./start.sh
```
Then open: http://localhost:3000

### **Option 2: Manual Startup**
```bash
# Terminal 1: Start Backend Server
cd backend
go mod tidy
go run main.go

# Terminal 2: Start Frontend Server
cd ..
python3 -m http.server 3000

# Open: http://localhost:3000
```

### **Option 3: Backend Only (API Testing)**
```bash
cd backend
go run main.go
# API available at: http://localhost:8080/api/v1
```

## 🎯 **Features Demonstrated**

### **🔐 RFC 111 Authorization**
- **Basic Authorization**: Simple power-of-attorney delegation
- **Advanced Authorization**: Multi-jurisdiction compliance with industry sectors
- **Real-time Validation**: Complete PoA Definition validation
- **Token Management**: JWT generation and validation

### **📋 RFC 115 PoA Definition**
- **Interactive Builder**: Create complete PoA definitions
- **Legal Framework Integration**: Multi-jurisdiction compliance
- **Organization Management**: Principal and authorizer configuration
- **AI Client Types**: Support for various AI system types

### **⚖️ Legal Compliance**
- **Multi-Jurisdiction Support**: US, EU, CA, UK, AU
- **Regulation Frameworks**: GDPR, SOX, CCPA, PIPEDA, MiFID
- **Entity Validation**: Corporation, LLC, Partnership, Individual
- **Capacity Verification**: Legal authority confirmation

### **📊 System Monitoring**
- **Real-time Status**: System health and performance metrics
- **Audit Trail**: Comprehensive event logging
- **Compliance Tracking**: RFC compliance status monitoring
- **Export Capabilities**: CSV audit log export

## 🏗️ **Architecture**

```
gauth-demo-app/web/
├── index.html              # Main demo interface
├── backend/                # Go API server
│   ├── main.go            # Server implementation
│   ├── go.mod             # Go dependencies
│   └── go.sum             # Dependency checksums
├── start.sh               # Startup script
└── README.md              # This file
```

### **Frontend Stack**
- **Vanilla JavaScript**: No framework dependencies
- **Modern CSS**: Responsive design with animations
- **Font Awesome**: Professional icons
- **Tabbed Interface**: Organized feature demonstration

### **Backend Stack**
- **Go 1.21+**: Modern Go implementation
- **Gorilla Mux**: HTTP routing
- **CORS Support**: Cross-origin resource sharing
- **GAuth Integration**: Full RFC implementation

## 🔌 **API Endpoints**

### **Authorization**
- `POST /api/v1/authorize` - Basic authorization
- `POST /api/v1/authorize/advanced` - Advanced authorization

### **PoA Definition**
- `POST /api/v1/poa/create` - Create PoA definition
- `POST /api/v1/poa/validate` - Validate PoA definition

### **Token Management**
- `POST /api/v1/token/validate` - Validate JWT token
- `POST /api/v1/token/generate` - Generate sample token

### **Compliance**
- `POST /api/v1/compliance/check` - Legal compliance check

### **System**
- `GET /api/v1/system/status` - System status
- `GET /api/v1/system/health` - Health check

### **Audit**
- `GET /api/v1/audit/logs` - Fetch audit logs
- `GET /api/v1/audit/export` - Export audit CSV

## 🧪 **Testing Scenarios**

### **1. Basic Authorization Flow**
1. Enter Principal ID: `company-corp-2025`
2. Enter AI Agent ID: `ai_assistant_v1`
3. Select Power Type: `Research Assistant`
4. Click "Authorize"
5. Observe successful RFC 111 compliance

### **2. Advanced Authorization**
1. Switch to "Advanced" tab
2. Select Jurisdiction: `US` or `EU`
3. Choose Industry Sector: `Financial Services`
4. Set Validity Period: `30` days
5. Click "Advanced Authorize"
6. Review extended compliance details

### **3. PoA Definition Creation**
1. Enter Organization: `GlobalTech Corporation`
2. Set Managing Director: `Dr. Sarah Johnson`
3. Choose AI Client Type: `Digital Agent`
4. Select Authorized Actions: Multiple options
5. Click "Create PoA Definition"
6. Examine RFC 115 compliance structure

### **4. Legal Compliance Testing**
1. Select Entity Type: `Corporation`
2. Choose Regulation: `GDPR` or `SOX`
3. Enable Capacity Verification
4. Click "Check Compliance"
5. Review multi-jurisdiction validation

### **5. Token Validation**
1. Click "Generate Sample" for test token
2. Paste JWT token in textarea
3. Click "Validate Token"
4. Examine JWT claims and metadata

### **6. System Monitoring**
1. Click "Check Status" for system overview
2. Click "Health Check" for detailed diagnostics
3. Review RFC compliance status
4. Monitor performance metrics

## 📱 **User Interface**

### **Modern Design Features**
- **Responsive Layout**: Works on desktop, tablet, mobile
- **Interactive Cards**: Hover effects and animations
- **Real-time Feedback**: Loading indicators and status updates
- **Demo Styling**: Professional appearance for demonstration purposes
- **Tabbed Navigation**: Organized feature access

### **Visual Indicators**
- ✅ **Success States**: Green indicators for successful operations
- ❌ **Error States**: Red indicators for validation failures
- 🔄 **Loading States**: Spinners during processing
- 📊 **Data Display**: Formatted JSON output

## 🔧 **Development**

### **Prerequisites**
- Go 1.21 or higher
- Python 3.7 or higher
- Modern web browser
- Terminal/command line access

### **Installation**
```bash
# Clone or navigate to the project
cd gauth-demo-app/web

# Install Go dependencies
cd backend
go mod tidy

# Return to web directory
cd ..

# Make startup script executable
chmod +x start.sh
```

### **Running in Development**
```bash
# Start both servers
./start.sh

# Or run individually:
# Backend: cd backend && go run main.go
# Frontend: python3 -m http.server 3000
```

### **Customization**
- **API Integration**: Modify `backend/main.go` for real API integration
- **UI Styling**: Update CSS in `index.html` for branding
- **Features**: Add new demo cards for additional functionality
- **Configuration**: Adjust ports and settings in startup scripts

## 🚀 **Production Deployment**

### **Build for Production**
```bash
# Build backend binary
cd backend
go build -o gauth-demo-server main.go

# Deploy static files
# Copy index.html and assets to web server
```

### **Docker Deployment**
```dockerfile
FROM golang:1.21-alpine AS backend
WORKDIR /app
COPY backend/ .
RUN go build -o server main.go

FROM nginx:alpine
COPY index.html /usr/share/nginx/html/
COPY --from=backend /app/server /usr/local/bin/
EXPOSE 80
CMD ["server"]
```

## 📚 **Documentation Links**

- [Main GAuth Documentation](../../../README.md)
- [RFC 111 Specification](../../../docs/RFC_ARCHITECTURE.md)
- [API Reference](../../../docs/API_REFERENCE.md)
- [Security Guide](../../../SECURITY.md)
- [Getting Started](../../../docs/GETTING_STARTED.md)

## 🤝 **Contributing**

1. Fork the repository
2. Create feature branch
3. Make changes to web application
4. Test thoroughly
5. Submit pull request

## 📄 **License**

This demo application is part of the GAuth project and follows the same licensing terms.

---

**🎉 Ready to explore the future of AI authorization with GAuth!** 

Start the demo and experience RFC 111 & 115 compliance in action.