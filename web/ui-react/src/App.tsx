import { Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'sonner'
import Layout from './components/Layout'
import Overview from './pages/Overview'
import Tokens from './pages/Tokens'
import PVP from './pages/PVP'
import Registry from './pages/Registry'
import PIP from './pages/PIP'
import PoA from './pages/PoA'
import E2ETesting from './pages/E2ETesting'
import Metrics from './pages/Metrics'
import Login from './pages/Login'

function App() {
  return (
    <>
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/overview" replace />} />
          <Route path="/overview" element={<Overview />} />
          <Route path="/tokens" element={<Tokens />} />
          <Route path="/pvp" element={<PVP />} />
          <Route path="/registry" element={<Registry />} />
          <Route path="/pip" element={<PIP />} />
          <Route path="/poa" element={<PoA />} />
          <Route path="/e2e" element={<E2ETesting />} />
          <Route path="/metrics" element={<Metrics />} />
          <Route path="/login" element={<Login />} />
        </Routes>
      </Layout>
      <Toaster richColors position="top-right" />
    </>
  )
}

export default App
