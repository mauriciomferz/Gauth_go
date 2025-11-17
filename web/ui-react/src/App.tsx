import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'sonner'
import { ErrorBoundary } from './components/ErrorBoundary'
import Layout from './components/Layout'
import { Loader2 } from 'lucide-react'

// Lazy load pages for better performance and code splitting
const Overview = lazy(() => import('./pages/Overview'))
const Tokens = lazy(() => import('./pages/Tokens'))
const PVP = lazy(() => import('./pages/PVP'))
const Registry = lazy(() => import('./pages/Registry'))
const PIP = lazy(() => import('./pages/PIP'))
const PAP = lazy(() => import('./pages/PAP'))
const PDP = lazy(() => import('./pages/PDP'))
const PEP = lazy(() => import('./pages/PEP'))
const PoA = lazy(() => import('./pages/PoA'))
const MCP = lazy(() => import('./pages/MCP'))
const E2ETesting = lazy(() => import('./pages/E2ETesting'))
const Metrics = lazy(() => import('./pages/Metrics'))
const Login = lazy(() => import('./pages/Login'))

// Loading fallback component
function PageLoader() {
  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="text-center">
        <Loader2 className="h-12 w-12 animate-spin text-primary-500 mx-auto mb-4" aria-hidden="true" />
        <p className="text-gray-600 dark:text-gray-400">Loading page...</p>
      </div>
    </div>
  )
}

function App() {
  return (
    <ErrorBoundary>
      <Layout>
        <Suspense fallback={<PageLoader />}>
          <Routes>
            <Route path="/" element={<Navigate to="/overview" replace />} />
            <Route path="/overview" element={<Overview />} />
            <Route path="/tokens" element={<Tokens />} />
            <Route path="/pvp" element={<PVP />} />
            <Route path="/registry" element={<Registry />} />
            <Route path="/pip" element={<PIP />} />
            <Route path="/pap" element={<PAP />} />
            <Route path="/pdp" element={<PDP />} />
            <Route path="/pep" element={<PEP />} />
            <Route path="/poa" element={<PoA />} />
            <Route path="/mcp" element={<MCP />} />
            <Route path="/e2e" element={<E2ETesting />} />
            <Route path="/metrics" element={<Metrics />} />
            <Route path="/login" element={<Login />} />
          </Routes>
        </Suspense>
      </Layout>
      <Toaster richColors position="top-right" />
    </ErrorBoundary>
  )
}

export default App
