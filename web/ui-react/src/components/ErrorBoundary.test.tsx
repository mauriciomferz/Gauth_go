import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '../test/utils'
import { ErrorBoundary } from './ErrorBoundary'
import userEvent from '@testing-library/user-event'

// Component that throws an error when shouldThrow is true
function ProblematicComponent({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error')
  }
  return <div>Normal content</div>
}

describe('ErrorBoundary Component', () => {
  // Suppress console.error for these tests since we're intentionally throwing errors
  const originalError = console.error
  beforeAll(() => {
    console.error = vi.fn()
  })
  afterAll(() => {
    console.error = originalError
  })

  it('renders children when there is no error', () => {
    render(
      <ErrorBoundary>
        <ProblematicComponent shouldThrow={false} />
      </ErrorBoundary>
    )
    
    expect(screen.getByText('Normal content')).toBeInTheDocument()
  })

  it('renders error UI when child component throws', () => {
    render(
      <ErrorBoundary>
        <ProblematicComponent shouldThrow={true} />
      </ErrorBoundary>
    )
    
    expect(screen.getByText(/something went wrong/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /reload page/i })).toBeInTheDocument()
  })

  it('renders custom fallback when provided', () => {
    const customFallback = <div>Custom error message</div>
    
    render(
      <ErrorBoundary fallback={customFallback}>
        <ProblematicComponent shouldThrow={true} />
      </ErrorBoundary>
    )
    
    expect(screen.getByText('Custom error message')).toBeInTheDocument()
  })

  it('shows error details in development mode', () => {
    const originalEnv = process.env.NODE_ENV
    process.env.NODE_ENV = 'development'
    
    render(
      <ErrorBoundary>
        <ProblematicComponent shouldThrow={true} />
      </ErrorBoundary>
    )
    
    expect(screen.getByText(/error details/i)).toBeInTheDocument()
    expect(screen.getByText(/test error/i)).toBeInTheDocument()
    
    process.env.NODE_ENV = originalEnv
  })

  it('handles try again button click', async () => {
    const user = userEvent.setup()
    
    const { rerender } = render(
      <ErrorBoundary>
        <ProblematicComponent shouldThrow={true} />
      </ErrorBoundary>
    )
    
    expect(screen.getByText(/something went wrong/i)).toBeInTheDocument()
    
    await user.click(screen.getByRole('button', { name: /try again/i }))
    
    // After reset, re-render with non-throwing component
    rerender(
      <ErrorBoundary>
        <ProblematicComponent shouldThrow={false} />
      </ErrorBoundary>
    )
    
    // Should show normal content after successful retry
    expect(screen.getByText('Normal content')).toBeInTheDocument()
  })

  it('has proper ARIA attributes for accessibility', () => {
    render(
      <ErrorBoundary>
        <ProblematicComponent shouldThrow={true} />
      </ErrorBoundary>
    )
    
    const tryAgainButton = screen.getByRole('button', { name: /try to recover/i })
    const reloadButton = screen.getByRole('button', { name: /reload the page/i })
    
    expect(tryAgainButton).toHaveAttribute('aria-label')
    expect(reloadButton).toHaveAttribute('aria-label')
  })
})
