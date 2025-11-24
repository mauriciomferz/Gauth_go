import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Spinner } from '@fluentui/react-components'
import AdminLogin from './pages/AdminLogin'
import AdminLayout from './components/AdminLayout'

// Lazy load admin pages for better performance and code splitting
const Dashboard = lazy(() => import('./pages/admin/Dashboard'))
const SystemMetrics = lazy(() => import('./pages/admin/SystemMetrics'))
const Subscribers = lazy(() => import('./pages/admin/Subscribers'))
const SubscribersList = lazy(() => import('./pages/admin/SubscribersList'))
const TokenManagement = lazy(() => import('./pages/admin/TokenManagement'))
const AuthorizationEngine = lazy(() => import('./pages/admin/AuthorizationEngine'))
const PowerOfAttorney = lazy(() => import('./pages/admin/PowerOfAttorney'))
const EventSystem = lazy(() => import('./pages/admin/EventSystem'))
const ResiliencePatterns = lazy(() => import('./pages/admin/ResiliencePatterns'))
const AuditTrail = lazy(() => import('./pages/admin/AuditTrail'))
const ConfigurationManager = lazy(() => import('./pages/admin/ConfigurationManager'))
const RevocationTransparency = lazy(() => import('./pages/admin/RevocationTransparency'))
const OIDCProviders = lazy(() => import('./pages/admin/OIDCProviders'))
const AuthCallback = lazy(() => import('./pages/AuthCallback'))
const OIDCLogin = lazy(() => import('./pages/OIDCLogin'))
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
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
      <Spinner size="large" label="Loading..." />
    </div>
  )
}

function App() {
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        {/* OIDC Authentication Routes */}
        <Route path="/auth/callback" element={<AuthCallback />} />
        <Route path="/oidc-login" element={<OIDCLogin />} />

        {/* Admin Portal Routes */}
        <Route path="/admin/login" element={<AdminLogin />} />
        <Route path="/admin/*" element={<AdminLayout />}>
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="metrics" element={<SystemMetrics />} />
          <Route path="subscribers" element={<Subscribers />} />
          <Route path="subscribers/list" element={<SubscribersList />} />
          <Route path="tokens" element={<TokenManagement />} />
          <Route path="authorization" element={<AuthorizationEngine />} />
          <Route path="poa" element={<PowerOfAttorney />} />
          <Route path="events" element={<EventSystem />} />
          <Route path="resilience" element={<ResiliencePatterns />} />
          <Route path="audit" element={<AuditTrail />} />
          <Route path="revocation" element={<RevocationTransparency />} />
          <Route path="configuration" element={<ConfigurationManager />} />
          <Route path="oidc-providers" element={<OIDCProviders />} />
          <Route path="performance" element={<div>Performance Page (Coming Soon)</div>} />
          <Route index element={<Navigate to="/admin/dashboard" replace />} />
        </Route>

        {/* Legacy/User Routes - Keep existing functionality */}
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
  )
}

export default App
