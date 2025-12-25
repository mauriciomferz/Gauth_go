---
title: Quick Start
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Quick Start Guide - React UI

This guide will get the React UI running in under 5 minutes.

## Prerequisites

- **Node.js** 18+ and **npm** 9+ (or **yarn**/**pnpm**)
- **Go** 1.22+ (for backend)
- Git

## Installation Steps

### 1. Install Dependencies

```bash
cd web/ui-react
npm install
```

This installs all required packages:
- React 18.3.1
- TypeScript 5.6.2
- Vite 5.4.9
- Tailwind CSS 3.4.14
- React Router, Zustand, Axios, and more

### 2. Start the Development Server

```bash
npm run dev
```

The UI will be available at **http://localhost:3000**

### 3. Start the Go Backend (in a separate terminal)

```bash
cd /path/to/Gauth_go
go run ./cmd/web-server
```

The backend API will be available at **http://localhost:8080**

### 4. Access the Application

Open your browser to **http://localhost:3000** and you'll see:

- **Overview Dashboard** with stats and RFC compliance info
- **Token Management** - Create and validate extended tokens
- **PVP Identity Verification** - Verify identities with eIDAS trust levels
- **Commercial Registry** - Verify entities and signatories
- **PIP Authorization** - Validate authorization chains
- **Power of Attorney** - Create and validate PoAs
- **E2E Testing** - Run end-to-end tests
- **Metrics** - View system performance metrics

## Features Available Immediately

### ✅ Working Now (No Backend Required)
- Responsive UI with dark/light theme
- Navigation between all pages
- Form inputs and validations
- Mock data visualization
- Component showcase

### 🔄 Requires Backend Integration
- Token creation/validation
- Identity verification
- Entity verification
- Authorization validation
- PoA management
- Real-time metrics

## Development Workflow

### Hot Module Replacement (HMR)
The dev server supports instant updates - edit any file and see changes immediately without page reload.

### API Integration
The Vite config includes a proxy to forward `/api/*` requests to `localhost:8080`. Update the target in `vite.config.ts` if your backend runs on a different port.

### Type Safety
TypeScript strict mode is enabled. The `src/lib/api.ts` file contains complete type definitions for all GAuth endpoints.

### Styling
Tailwind CSS utility classes are used throughout. Custom colors:
- **Primary**: #667eea (purple)
- **Success**: #22c55e (green)

## Available Scripts

```bash
# Development server with HMR (port 3000)
npm run dev

# Type checking (without build)
npm run type-check

# Production build (outputs to dist/)
npm run build

# Preview production build
npm run preview

# Format code with Prettier
npm run format

# Lint code (when configured)
npm run lint
```

## Project Structure

```
web/ui-react/
├── src/
│   ├── components/         # Reusable UI components
│   │   ├── Layout.tsx      # App layout with header/nav/footer
│   │   ├── Card.tsx        # Card and StatCard components
│   │   ├── Button.tsx      # Multi-variant button
│   │   └── Form.tsx        # Input, Select, Textarea
│   ├── pages/              # Route pages
│   │   ├── Overview.tsx    # Dashboard (complete)
│   │   ├── Tokens.tsx      # Token management (complete)
│   │   ├── PVP.tsx         # Identity verification (complete)
│   │   ├── Registry.tsx    # Coming soon
│   │   ├── PIP.tsx         # Coming soon
│   │   ├── PoA.tsx         # Coming soon
│   │   ├── E2ETesting.tsx  # Coming soon
│   │   └── Metrics.tsx     # Coming soon
│   ├── lib/                # Utilities and API client
│   │   ├── api.ts          # Complete API client (308 lines)
│   │   └── utils.ts        # Helper functions
│   ├── store/              # State management
│   │   └── theme.ts        # Theme store (dark/light)
│   ├── App.tsx             # Route configuration
│   ├── main.tsx            # React entry point
│   └── index.css           # Tailwind base styles
├── public/                 # Static assets
├── index.html              # HTML template
├── package.json            # Dependencies and scripts
├── vite.config.ts          # Vite configuration
├── tsconfig.json           # TypeScript config
├── tailwind.config.js      # Tailwind config
└── README.md               # Full documentation
```

## Customization

### Change Theme Colors
Edit `tailwind.config.js`:
```js
colors: {
  primary: {
    50: '#f5f7ff',
    // ... other shades
    600: '#your-color',
  },
}
```

### Add New Pages
1. Create `src/pages/YourPage.tsx`
2. Add route in `src/App.tsx`:
   ```tsx
   <Route path="/your-page" element={<YourPage />} />
   ```
3. Add nav link in `src/components/Layout.tsx`

### Connect to Different Backend
Update `vite.config.ts`:
```ts
proxy: {
  '/api': {
    target: 'http://your-backend:port',
    changeOrigin: true,
  },
},
```

## Troubleshooting

### Port 3000 Already in Use
```bash
# Use a different port
npm run dev -- --port 3001
```

### TypeScript Errors
All current errors are expected before `npm install`. After installation, run:
```bash
npm run type-check
```

### API Requests Failing
1. Ensure Go backend is running on port 8080
2. Check browser console for CORS errors
3. Verify API proxy in `vite.config.ts`

### Build Errors
Clear cache and reinstall:
```bash
rm -rf node_modules dist
npm install
npm run build
```

## Next Steps

1. **Complete Backend Integration**: Implement remaining page features
2. **Add Real-Time Updates**: WebSocket connections for live data
3. **Production Deployment**: Build Docker image and deploy
4. **Testing**: Add unit tests with Vitest and E2E tests with Playwright

## Resources

- **React Docs**: https://react.dev
- **TypeScript**: https://www.typescriptlang.org
- **Vite**: https://vitejs.dev
- **Tailwind CSS**: https://tailwindcss.com
- **React Router**: https://reactrouter.com

## Support

For issues or questions:
1. Check `README.md` for detailed documentation
2. Review `INTEGRATION_GUIDE.md` for backend integration
3. See `IMPLEMENTATION_SUMMARY.md` for feature status

---

**Status**: Foundation Complete ✅ | Backend Integration In Progress 🔄

**Last Updated**: November 2025
