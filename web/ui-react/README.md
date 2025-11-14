# GAuth 1.0 React Dashboard

Modern React-based Single Page Application (SPA) for the GAuth 1.0 RFC-0111/0115 compliant authorization framework.

## 🚀 Features

- **Modern Tech Stack**: React 18, TypeScript, Vite, Tailwind CSS
- **Fast Development**: Hot Module Replacement (HMR) with Vite
- **Type Safety**: Full TypeScript support
- **Responsive Design**: Mobile-first Tailwind CSS styling
- **Dark Mode**: Built-in dark/light theme toggle with persistence
- **State Management**: Zustand for lightweight global state
- **API Integration**: Axios client with full backend integration
- **Real-time Updates**: Toast notifications with Sonner
- **Modern Icons**: Lucide React icon library
- **Charts & Graphs**: Recharts for data visualization

## 📦 Tech Stack

- **Framework**: React 18.3
- **Build Tool**: Vite 5.4
- **Language**: TypeScript 5.6
- **Styling**: Tailwind CSS 3.4
- **Routing**: React Router DOM 6.26
- **State**: Zustand 4.5
- **HTTP Client**: Axios 1.7
- **Icons**: Lucide React
- **Charts**: Recharts 2.12
- **Notifications**: Sonner 1.5

## 🛠️ Setup Instructions

### Prerequisites

- Node.js 18+ and npm (or yarn/pnpm)
- GAuth Go backend running on `localhost:8080`

### Installation

1. **Navigate to the UI directory:**
   ```bash
   cd web/ui-react
   ```

2. **Install dependencies:**
   ```bash
   npm install
   ```

3. **Start development server:**
   ```bash
   npm run dev
   ```

   The app will be available at `http://localhost:3000`

### Available Scripts

- `npm run dev` - Start development server with HMR
- `npm run build` - Build for production
- `npm run preview` - Preview production build locally
- `npm run lint` - Run ESLint checks
- `npm run format` - Format code with Prettier

## 📁 Project Structure

```
web/ui-react/
├── src/
│   ├── components/        # Reusable UI components
│   │   ├── Layout.tsx    # Main layout with header/footer
│   │   ├── Card.tsx      # Card and StatCard components
│   │   ├── Button.tsx    # Button component
│   │   └── Form.tsx      # Form input components
│   ├── pages/             # Page components (routes)
│   │   ├── Overview.tsx  # Dashboard overview
│   │   ├── Tokens.tsx    # Extended token management
│   │   ├── PVP.tsx       # PVP identity verification
│   │   ├── Registry.tsx  # Commercial registry
│   │   ├── PIP.tsx       # Policy information point
│   │   ├── PoA.tsx       # Power of Attorney
│   │   ├── E2ETesting.tsx # E2E testing
│   │   └── Metrics.tsx   # System metrics
│   ├── lib/               # Utilities and API client
│   │   ├── api.ts        # API client with types
│   │   └── utils.ts      # Utility functions
│   ├── store/             # State management
│   │   └── theme.ts      # Theme store
│   ├── App.tsx            # Main app component
│   ├── main.tsx           # Entry point
│   └── index.css          # Global styles
├── public/                # Static assets
├── index.html             # HTML template
├── vite.config.ts         # Vite configuration
├── tailwind.config.js     # Tailwind configuration
├── tsconfig.json          # TypeScript configuration
└── package.json           # Dependencies and scripts
```

## 🎨 Features Breakdown

### Pages

1. **Overview** - Dashboard with system stats and quick links
2. **Extended Tokens** - Create, validate, and manage RFC-0111 tokens
3. **PVP** - eIDAS identity verification with trust levels
4. **Registry** - Commercial register entity verification (HRB/Companies House)
5. **PIP** - Authorization validation with policy checks
6. **PoA** - Power of Attorney management (RFC-0115)
7. **E2E Testing** - End-to-end integration testing
8. **Metrics** - System performance and health metrics

### Components

- **Layout** - Responsive header, navigation, and footer
- **Card** - Reusable card container with optional title/icon
- **StatCard** - Statistics display with gradients and trends
- **Button** - Customizable button with variants and loading states
- **Form** - Input, Select, and Textarea components with validation

### API Integration

The `lib/api.ts` file provides a complete API client with TypeScript types for all GAuth backend endpoints:

- Token operations (create, validate)
- PVP identity verification
- Commercial registry queries
- PIP authorization validation
- PoA management
- Metrics and health checks

### State Management

- **Theme Store** (Zustand) - Persists dark/light mode preference to localStorage
- **Future**: Add stores for tokens, PoAs, and application state

## 🔌 API Proxy Configuration

Vite is configured to proxy API requests to the Go backend:

```typescript
// vite.config.ts
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

## 🏗️ Production Build

### Build the app:

```bash
npm run build
```

Output will be in the `dist/` directory.

### Preview production build:

```bash
npm run preview
```

### Serve from Go backend:

Update your Go server to serve the React build:

```go
// In cmd/web-server/main.go
http.Handle("/", http.FileServer(http.Dir("./web/ui-react/dist")))
```

## 🎨 Customization

### Tailwind Theme

Edit `tailwind.config.js` to customize colors, fonts, and other design tokens:

```javascript
theme: {
  extend: {
    colors: {
      primary: { /* your colors */ },
    },
  },
}
```

### Dark Mode

The app uses Tailwind's `class` strategy for dark mode. Toggle is handled by the theme store.

## 📝 Development Guidelines

### Adding a New Page

1. Create component in `src/pages/NewPage.tsx`
2. Add route in `src/App.tsx`
3. Add navigation link in `src/components/Layout.tsx`

### Adding API Endpoints

1. Add types in `lib/api.ts` (e.g., `NewRequest`, `NewResponse`)
2. Add method to `ApiClient` class
3. Use in components with try/catch and loading states

### Styling Guidelines

- Use Tailwind utility classes
- Follow mobile-first responsive design
- Use dark mode variants: `dark:bg-gray-800`
- Leverage custom Tailwind classes in `index.css`

## 🐛 Troubleshooting

### TypeScript Errors Before Install

The TypeScript errors shown before running `npm install` are expected. They'll disappear once dependencies are installed.

### Port Already in Use

If port 3000 is already in use, change it in `vite.config.ts`:

```typescript
server: {
  port: 3001,  // or any available port
}
```

### API Connection Issues

Ensure the Go backend is running on `localhost:8080`. Check proxy configuration in `vite.config.ts`.

## 🚢 Deployment

### Docker

A Dockerfile can be created for containerized deployment:

```dockerfile
FROM node:18-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
```

### Static Hosting

The `dist/` folder can be deployed to:
- Vercel
- Netlify
- AWS S3 + CloudFront
- GitHub Pages

Configure your Go backend to handle API requests separately.

## 📄 License

MIT License - Same as parent GAuth project

## 🤝 Contributing

1. Follow the existing code style
2. Add TypeScript types for all new code
3. Test in both light and dark modes
4. Ensure responsive design works on mobile
5. Run linter before committing: `npm run lint`

## 📚 Resources

- [React Documentation](https://react.dev/)
- [Vite Documentation](https://vitejs.dev/)
- [Tailwind CSS](https://tailwindcss.com/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Zustand](https://github.com/pmndrs/zustand)
- [React Router](https://reactrouter.com/)

---

**Built with ❤️ for GAuth 1.0 - Production-Ready Authorization Framework**
