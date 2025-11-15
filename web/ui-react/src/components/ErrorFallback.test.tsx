import { describe, it, expect, vi } from 'vitest'
import { renderWithRouter, screen, userEvent } from '../test/utils'
import { ErrorFallback } from './ErrorFallback'

describe('ErrorFallback Component', () => {
  it('renders default error message', () => {
    renderWithRouter(<ErrorFallback />)
    
    expect(screen.getByText(/unable to load content/i)).toBeInTheDocument()
    expect(screen.getByText(/we encountered an error/i)).toBeInTheDocument()
  })

  it('renders custom title and message', () => {
    renderWithRouter(
      <ErrorFallback
        title="Custom Error Title"
        message="Custom error message"
      />
    )
    
    expect(screen.getByText('Custom Error Title')).toBeInTheDocument()
    expect(screen.getByText('Custom error message')).toBeInTheDocument()
  })

  it('shows error details in development mode', () => {
    const originalEnv = process.env.NODE_ENV
    process.env.NODE_ENV = 'development'
    
    const testError = new Error('Test error message')
    testError.stack = 'Error stack trace'
    
    renderWithRouter(<ErrorFallback error={testError} />)
    
    expect(screen.getByText(/error details/i)).toBeInTheDocument()
    expect(screen.getByText(/test error message/i)).toBeInTheDocument()
    
    process.env.NODE_ENV = originalEnv
  })

  it('calls resetError callback when Try Again is clicked', async () => {
    const user = userEvent.setup()
    const resetError = vi.fn()
    
    renderWithRouter(<ErrorFallback resetError={resetError} />)
    
    await user.click(screen.getByRole('button', { name: /retry loading content/i }))
    expect(resetError).toHaveBeenCalledTimes(1)
  })

  it('shows Go Home button', () => {
    renderWithRouter(<ErrorFallback />)
    
    expect(screen.getByRole('button', { name: /go to home page/i })).toBeInTheDocument()
  })

  it('has proper accessibility attributes', () => {
    renderWithRouter(<ErrorFallback resetError={() => {}} />)
    
    const tryAgainButton = screen.getByRole('button', { name: /retry loading content/i })
    const homeButton = screen.getByRole('button', { name: /go to home page/i })
    
    expect(tryAgainButton).toHaveAttribute('aria-label')
    expect(homeButton).toHaveAttribute('aria-label')
  })
})
