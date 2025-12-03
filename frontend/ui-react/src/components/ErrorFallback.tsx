import { AlertCircle, RefreshCw, Home } from 'lucide-react'
import { Button } from './Button'
import { useNavigate } from 'react-router-dom'

interface ErrorFallbackProps {
  error?: Error
  resetError?: () => void
  title?: string
  message?: string
}

/**
 * ErrorFallback component displays user-friendly error messages
 * and provides recovery actions like retry or navigation.
 * 
 * Can be used with ErrorBoundary or standalone for API errors.
 */
export function ErrorFallback({
  error,
  resetError,
  title = 'Unable to Load Content',
  message = 'We encountered an error while loading this content. Please try again.',
}: ErrorFallbackProps) {
  const navigate = useNavigate()

  const handleGoHome = () => {
    navigate('/')
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-[400px] px-4">
      <div className="text-center max-w-md">
        <AlertCircle 
          className="h-16 w-16 text-yellow-500 mx-auto mb-4" 
          aria-hidden="true"
        />
        
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
          {title}
        </h2>
        
        <p className="text-gray-600 dark:text-gray-300 mb-6">
          {message}
        </p>

        {process.env.NODE_ENV === 'development' && error && (
          <details className="mb-6 p-4 bg-gray-100 dark:bg-gray-700 rounded-lg text-left">
            <summary className="cursor-pointer font-semibold text-sm text-gray-700 dark:text-gray-200 mb-2">
              Error Details
            </summary>
            <pre className="text-xs text-red-600 dark:text-red-400 overflow-auto whitespace-pre-wrap">
              {error.message}
              {error.stack && `\n\n${error.stack}`}
            </pre>
          </details>
        )}

        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          {resetError && (
            <Button
              onClick={resetError}
              variant="primary"
              icon={<RefreshCw className="h-4 w-4" />}
              aria-label="Retry loading content"
            >
              Try Again
            </Button>
          )}
          <Button
            onClick={handleGoHome}
            variant="secondary"
            icon={<Home className="h-4 w-4" />}
            aria-label="Go to home page"
          >
            Go Home
          </Button>
        </div>
      </div>
    </div>
  )
}
