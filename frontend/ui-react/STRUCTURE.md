---
title: Structure
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# React UI Project Structure

```
web/ui-react/                           # React SPA Root Directory
│
├── 📋 Documentation (5 files)
│   ├── README.md                       # Complete setup guide (263 lines)
│   ├── QUICK_START.md                  # 5-minute quick start
│   ├── INTEGRATION_GUIDE.md            # Backend integration options
│   ├── STATUS_REPORT.md                # Detailed progress report
│   ├── IMPLEMENTATION_SUMMARY.md       # Feature comparison
│   └── SUMMARY.md                      # Executive summary
│
├── ⚙️ Configuration (10 files)
│   ├── package.json                    # Dependencies & scripts
│   ├── vite.config.ts                  # Vite build config + API proxy
│   ├── tsconfig.json                   # TypeScript strict mode
│   ├── tsconfig.node.json              # Node TypeScript config
│   ├── tailwind.config.js              # Custom theme (primary, success)
│   ├── postcss.config.js               # PostCSS plugins
│   ├── .prettierrc                     # Code formatting rules
│   ├── .gitignore                      # Git exclusions
│   ├── index.html                      # HTML template
│   └── setup.sh                        # Automated setup script ✨
│
├── 📦 src/                             # Source Code (1,607 lines)
│   │
│   ├── 🧩 components/                  # Reusable UI Components (4 files, 354 lines)
│   │   ├── Layout.tsx                  # App layout (header, nav, footer) - 136 lines
│   │   ├── Card.tsx                    # Card & StatCard components - 57 lines
│   │   ├── Button.tsx                  # Multi-variant button - 65 lines
│   │   └── Form.tsx                    # Input, Select, Textarea - 96 lines
│   │
│   ├── 📄 pages/                       # Route Pages (8 files, 809 lines)
│   │   ├── Overview.tsx                # ✅ Dashboard with stats - 127 lines
│   │   ├── Tokens.tsx                  # ✅ Token management - 250 lines
│   │   ├── PVP.tsx                     # ✅ Identity verification - 240 lines
│   │   ├── Registry.tsx                # 🔄 Entity verification - 23 lines
│   │   ├── PIP.tsx                     # 🔄 Authorization - 23 lines
│   │   ├── PoA.tsx                     # 🔄 Power of Attorney - 23 lines
│   │   ├── E2ETesting.tsx              # 🔄 Test execution - 23 lines
│   │   └── Metrics.tsx                 # 🔄 System metrics - 23 lines
│   │
│   ├── 📚 lib/                         # Libraries & Utilities (2 files, 326 lines)
│   │   ├── api.ts                      # ✅ Complete API client - 308 lines
│   │   │                               #    • 20+ typed methods
│   │   │                               #    • Full TypeScript interfaces
│   │   │                               #    • Error interceptors
│   │   └── utils.ts                    # ✅ Helper functions - 18 lines
│   │                                   #    • cn() - class merger
│   │                                   #    • formatDate(), formatDuration()
│   │                                   #    • generateId()
│   │
│   ├── 🗂️ store/                       # State Management (1 file, 39 lines)
│   │   └── theme.ts                    # ✅ Zustand theme store
│   │                                   #    • Dark/light toggle
│   │                                   #    • localStorage persistence
│   │
│   ├── App.tsx                         # ✅ Route configuration - 28 lines
│   │                                   #    • 9 routes defined
│   │                                   #    • Toast notifications setup
│   │
│   ├── main.tsx                        # ✅ React entry point - 11 lines
│   │                                   #    • ReactDOM render
│   │                                   #    • BrowserRouter setup
│   │
│   └── index.css                       # ✅ Tailwind base + custom utilities
│
└── 📦 node_modules/                    # Dependencies (created after npm install)
    └── (React, Vite, Tailwind, etc.)

```

## 📊 Statistics

| Category | Count | Lines of Code |
|----------|-------|---------------|
| **Total Files** | 34 | 1,607 (TypeScript/React) |
| **Documentation** | 6 | ~1,000+ (Markdown) |
| **Configuration** | 10 | ~200 |
| **Components** | 4 | 354 |
| **Pages** | 8 | 809 |
| **Libraries** | 2 | 326 |
| **State Management** | 1 | 39 |
| **App Setup** | 2 | 39 |
| **Styles** | 1 | ~40 |

## 🎯 Component Breakdown

### Layout.tsx (136 lines)
```
Header (sticky)
├── Logo (Shield icon + "GAuth 1.0")
├── Badge ("Production Ready")
├── Navigation (8 links with icons)
│   ├── Overview (Home icon)
│   ├── Tokens (Key icon)
│   ├── PVP (UserCheck icon)
│   ├── Registry (Building2 icon)
│   ├── PIP (Gavel icon)
│   ├── PoA (FileText icon)
│   ├── E2E Testing (Activity icon)
│   └── Metrics (BarChart3 icon)
├── Theme Toggle (Moon/Sun icon)
└── Footer (3 columns)
    ├── About
    ├── Quick Links
    └── System Status
```

### Card.tsx (57 lines)
```
Card Component
├── Optional title
├── Optional icon
├── Children content
└── Hover shadow effect

StatCard Component
├── Gradient icon circle
├── Large value display
├── Title
└── Optional trend (with arrow)
```

### Button.tsx (65 lines)
```
Button Component
├── Variants: primary, secondary, success, danger
├── Sizes: sm, md, lg
├── States: default, loading, disabled
├── Optional icon
└── Hover/focus effects
```

### Form.tsx (96 lines)
```
Input Component
├── Label
├── Text input
├── Error message
└── Focus ring

Select Component
├── Label
├── Dropdown
├── Options (children)
└── Error message

Textarea Component
├── Label
├── Multi-line input
├── Error message
└── Configurable rows
```

## 🚀 Page Breakdown

### Overview.tsx (127 lines) ✅ COMPLETE
```
Hero Section
├── Shield icon
└── Gradient heading

Stats Grid (4 cards)
├── 91 tests (100%)
├── 19 benchmarks
├── 72.6% coverage
└── 1.3µs E2E average

Info Grid (3 cards)
├── RFC Compliance (3 items)
├── System Components (5 items)
└── Quick Start (5 steps)
```

### Tokens.tsx (250 lines) ✅ COMPLETE
```
Create Token Form
├── Client ID
├── Owner's Authorizer Entity
├── Client Owner Entity
├── Scope (comma-separated)
├── Expiration (hours)
└── Submit button

Success Display
├── Client ID
├── Expires date
├── Scope
└── Token (with copy button)

Validate Token Form
├── Token textarea
└── Submit button

Validation Result
├── Valid/Invalid status
├── Checks (color-coded)
└── Decoded payload (JSON)

Recent Tokens List (last 5)
├── Client ID
├── Expiration
├── Scope
└── Status badge
```

### PVP.tsx (240 lines) ✅ COMPLETE
```
Stats Grid (4 cards)
├── Total Verifications
├── Success Rate
├── Active TSPs
└── Avg Response Time

Verify Identity Form
├── Identity Type (individual/legal_entity)
├── Trust Level (substantial/high)
├── Entity ID
└── TSP selector

Verification Result
├── Verified/Failed status
├── Entity ID
├── Trust Level
├── TSP
└── Attributes (JSON)

Available TSPs (4 providers)
├── eIDAS TSP (EU)
├── National ID Service (DE)
├── UK Gov Verify (GB)
└── US Fed Identity Service (US)

eIDAS Trust Levels Info
├── Substantial (Level 2)
└── High (Level 3)

Verification History Table
├── Entity ID
├── Status (Verified/Failed)
└── Timestamp
```

## 🔧 API Client (api.ts - 308 lines)

```
ApiClient Class
├── Axios instance (baseURL: /api)
├── Error interceptor
└── 20+ Methods:

Token APIs
├── createToken()
├── validateToken()
└── getRevocationHead()

Rotation APIs
└── getRotationSummary()

Capability APIs
└── getCapabilityAnchor()

Error APIs
└── getErrorCatalog()

Algorithm APIs
└── getAlgorithms()

PVP APIs
└── verifyIdentity()

Registry APIs
├── verifyEntity()
└── verifySignatory()

PIP APIs
├── validateAuthorization()
└── getCacheStats()

PoA APIs
├── createPoA()
├── validatePoA()
└── listPoAs()

System APIs
├── getMetrics()
└── healthCheck()

TypeScript Interfaces (15+)
├── CreateTokenRequest
├── TokenResponse
├── TokenValidationResponse
├── VerifyIdentityRequest
├── IdentityVerificationResponse
├── EntityVerificationResponse
├── SignatoryVerificationResponse
├── AuthorizationValidationResponse
├── CacheStatsResponse
├── CreatePoARequest
├── ValidatePoARequest
├── PoAResponse
├── MetricsResponse
└── HealthCheckResponse
```

## 📦 Dependencies (package.json)

```
Production Dependencies (11)
├── react: 18.3.1
├── react-dom: 18.3.1
├── react-router-dom: 6.26.2
├── zustand: 4.5.5
├── axios: 1.7.7
├── lucide-react: 0.451.0
├── sonner: 1.5.0
├── recharts: 2.12.7
├── tailwind-merge: 2.5.4
├── clsx: 2.1.1
└── (automatic: react/jsx-runtime)

Dev Dependencies (10)
├── @types/react: 18.3.11
├── @types/react-dom: 18.3.1
├── @vitejs/plugin-react: 4.3.3
├── typescript: 5.6.2
├── vite: 5.4.9
├── tailwindcss: 3.4.14
├── postcss: 8.4.47
├── autoprefixer: 10.4.20
├── prettier: 3.3.3
└── prettier-plugin-tailwindcss: 0.6.8
```

## 🎨 Theme Configuration

```
tailwind.config.js
├── Dark mode: class strategy
├── Content paths: ["./index.html", "./src/**/*.{ts,tsx}"]
├── Custom colors:
│   ├── primary: { 50-900 } (purple #667eea)
│   └── success: { 50-900 } (green #22c55e)
└── Font family: system fonts
```

## 🔌 Vite Configuration

```
vite.config.ts
├── React plugin
├── Dev server:
│   └── Port: 3000
├── API Proxy:
│   └── /api/* → http://localhost:8080
├── Path aliases:
│   └── @/ → /src/
└── Build output: dist/
```

## 📝 TypeScript Configuration

```
tsconfig.json
├── Strict mode: true
├── Target: ESNext
├── Module: ESNext
├── JSX: react-jsx
├── Path mapping: @/* → ./src/*
└── Lib: [ES2020, DOM, DOM.Iterable]
```

## 🚀 Available Scripts

```bash
npm run dev          # Vite dev server (port 3000, HMR enabled)
npm run build        # TypeScript compile + Vite build → dist/
npm run preview      # Preview production build
npm run format       # Prettier formatting
npm run type-check   # TypeScript type checking (no build)
./setup.sh           # Automated setup (npm install + optional dev start)
```

## ✅ Completion Status

```
Foundation:           100% ✅
Configuration:        100% ✅
Components:           100% ✅
Pages (Complete):      37% ✅ (3/8)
Pages (Placeholder):   63% 🔄 (5/8)
API Client:           100% ✅
State Management:     100% ✅
Documentation:        100% ✅
Setup Scripts:        100% ✅

Overall Progress:      ~70% ✅
```

## 🎯 Next Steps

1. **Immediate**: Run `npm install` to install dependencies
2. **Short Term**: Complete remaining 5 pages (Registry, PIP, PoA, E2E, Metrics)
3. **Medium Term**: Implement backend API endpoints
4. **Long Term**: Real-time features, testing, deployment

---

**Created**: November 2025  
**Lines of Code**: 1,607 (TypeScript/React)  
**Total Files**: 34  
**Dependencies**: 21  
**Status**: Foundation Complete ✅
