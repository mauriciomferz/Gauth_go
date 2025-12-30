import { useEffect, useState } from 'react'

/**
 * SkipLink provides a keyboard-accessible way to skip navigation and jump directly
 * to the main content. This is a critical accessibility feature for screen reader
 * and keyboard users who would otherwise have to tab through all navigation items.
 *
 * WCAG 2.1 Level A requirement: 2.4.1 Bypass Blocks
 */
export function SkipLink() {
  const [isVisible, setIsVisible] = useState(false)

  // Handle keyboard focus
  const handleFocus = () => setIsVisible(true)
  const handleBlur = () => setIsVisible(false)

  // Skip to main content
  const skipToContent = (e: React.MouseEvent | React.KeyboardEvent) => {
    e.preventDefault()
    const mainContent = document.getElementById('main-content')
    if (mainContent) {
      mainContent.focus()
      mainContent.scrollIntoView({ behavior: 'smooth' })
    }
  }

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      skipToContent(e)
    }
  }

  useEffect(() => {
    // Ensure main content has tabindex for focusing
    const mainContent = document.getElementById('main-content')
    if (mainContent && !mainContent.hasAttribute('tabindex') {
      mainContent.setAttribute('tabindex', '-1')
    }
  }, [])

  return (
    <a
      href="#main-content"
      className={`
        fixed top-4 left-4 z-[9999]
        px-4 py-2 bg-primary-600 text-white rounded-lg
        font-medium shadow-lg
        transition-all duration-200
        ${isVisible ? 'translate-y-0 opacity-100' : '-translate-y-20 opacity-0'}
        focus:translate-y-0 focus:opacity-100
      `}
      onFocus={handleFocus}
      onBlur={handleBlur}
      onClick={skipToContent}
      onKeyDown={handleKeyDown}
      aria-label="Skip to main content"
    >
      Skip to main content
    </a>
  )
}
