import { ReactNode, useEffect } from 'react'
import { NavLink } from 'react-router-dom'
import {
  Home,
  Key,
  UserCheck,
  Building2,
  Gavel,
  FileText,
  Plug,
  Activity,
  BarChart3,
  Shield,
  Moon,
  Sun,
  FileCheck,
  ShieldCheck,
} from 'lucide-react'
import { useThemeStore } from '../store/theme'
import { cn } from '../lib/utils'

interface LayoutProps {
  children: ReactNode
}

const navigation = [
  { name: 'Overview', href: '/overview', icon: Home },
  { name: 'Extended Tokens', href: '/tokens', icon: Key },
  { name: 'PVP', href: '/pvp', icon: UserCheck },
  { name: 'Registry', href: '/registry', icon: Building2 },
  { name: 'PIP', href: '/pip', icon: Gavel },
  { name: 'PAP', href: '/pap', icon: FileText },
  { name: 'PDP', href: '/pdp', icon: FileCheck },
  { name: 'PEP', href: '/pep', icon: ShieldCheck },
  { name: 'PoA', href: '/poa', icon: FileText },
  { name: 'MCP', href: '/mcp', icon: Plug },
  { name: 'E2E Testing', href: '/e2e', icon: Activity },
  { name: 'Metrics', href: '/metrics', icon: BarChart3 },
  { name: 'Login', href: '/login', icon: Shield },
]

export default function Layout({ children }: LayoutProps) {
  const { theme, toggleTheme } = useThemeStore()

  useEffect(() => {
    // Initialize theme on mount
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header
        className="sticky top-0 z-50 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 shadow-sm"
        role="banner"
      >
        <div className="container mx-auto px-4 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <div className="flex items-center gap-3">
              <Shield className="h-8 w-8 text-primary-500" />
              <div>
                <h1 className="text-xl font-bold text-gray-900 dark:text-white">GAuth 1.0</h1>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  RFC-0111 & RFC-0115 Compliant
                </p>
              </div>
              <span
                className="ml-2 px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 text-xs font-semibold rounded-full hover:ring-2 hover:ring-blue-300 dark:hover:ring-blue-700 transition"
                title="Beta – not for production use."
                aria-label="Beta – not for production use."
                role="note"
              >
                Beta Version
              </span>
            </div>

            {/* Theme Toggle */}
            <button
              onClick={toggleTheme}
              className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              aria-label="Toggle theme"
            >
              {theme === 'light' ? (
                <Moon className="h-5 w-5 text-gray-700 dark:text-gray-300" />
              ) : (
                <Sun className="h-5 w-5 text-gray-700 dark:text-gray-300" />
              )}
            </button>
          </div>

          {/* Navigation */}
          <nav
            className="flex gap-1 overflow-x-auto pb-2 -mb-px scrollbar-hide"
            role="navigation"
            aria-label="Main navigation"
          >
            {navigation.map((item) => (
              <NavLink
                key={item.name}
                to={item.href}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg whitespace-nowrap transition-colors',
                    isActive
                      ? 'bg-primary-500 text-white'
                      : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                  )
                }
              >
                <item.icon className="h-4 w-4" />
                <span>{item.name}</span>
              </NavLink>
            ))}
          </nav>
        </div>
      </header>

      {/* Main Content */}
      <main
        id="main-content"
        className="container mx-auto px-4 lg:px-8 py-8"
        role="main"
        aria-label="Main content"
      >
        {children}
      </main>

      {/* Footer */}
      <footer
        className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 mt-12"
        role="contentinfo"
      >
        <div className="container mx-auto px-4 lg:px-8 py-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div>
              <div className="flex items-center gap-2 mb-2">
                <Shield className="h-5 w-5 text-primary-500" />
                <h4 className="font-semibold text-gray-900 dark:text-white">GAuth 1.0</h4>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                RFC-0111 & RFC-0115 Compliant Authorization Framework
              </p>
            </div>
            <div>
              <h4 className="font-semibold text-gray-900 dark:text-white mb-2">Quick Links</h4>
              <ul className="space-y-1 text-sm text-gray-600 dark:text-gray-400">
                <li>
                  <a href="/api/v1/docs" target="_blank" rel="noopener noreferrer" className="hover:text-primary-500">
                    Documentation
                  </a>
                </li>
                <li>
                  <a href="/swagger-ui/" target="_blank" rel="noopener noreferrer" className="hover:text-primary-500">
                    API Reference
                  </a>
                </li>
                <li>
                  <a
                    href="https://github.com/mauriciomferz/Gauth_go"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-primary-500"
                  >
                    View Source
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="font-semibold text-gray-900 dark:text-white mb-2">System Status</h4>
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 bg-success-500 rounded-full animate-pulse"></div>
                <span className="text-sm text-gray-600 dark:text-gray-400">All Systems Operational</span>
              </div>
            </div>
          </div>
          <div className="mt-6 pt-6 border-t border-gray-200 dark:border-gray-700 text-center text-sm text-gray-500 dark:text-gray-400">
            <p>© 2025 Gimel Foundation. Licensed under MIT.</p>
          </div>
        </div>
      </footer>
    </div>
  )
}
