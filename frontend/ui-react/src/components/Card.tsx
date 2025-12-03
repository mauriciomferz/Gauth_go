import { ReactNode, isValidElement, createElement } from 'react'
import { cn } from '../lib/utils'

interface CardProps {
  children: ReactNode
  className?: string
  title?: string
  // Allow either a ready element or a component type
  icon?: ReactNode | (() => ReactNode) | any
}

export function Card({ children, className, title, icon }: CardProps) {
  // Defensive rendering: if icon is a component type (function) create an element
  let renderedIcon: ReactNode | null = null
  if (icon) {
    if (isValidElement(icon)) {
      renderedIcon = icon
    } else if (typeof icon === 'function') {
      try {
        renderedIcon = icon()
      } catch (e) {
        console.warn('Icon component threw during render:', e)
      }
    } else if (icon && typeof icon === 'object' && 'default' in icon && typeof icon.default === 'function') {
      // Handle potential dynamic import object
      try {
        renderedIcon = createElement(icon.default)
      } catch (e) {
        console.warn('Dynamic icon default export failed:', e)
      }
    } else {
      // Last resort: try createElement if it looks like a component (has render or type)
      if (icon && (icon.render || icon.type)) {
        try {
          renderedIcon = createElement(icon.type || icon)
        } catch {
          renderedIcon = null
        }
      } else {
        renderedIcon = icon as ReactNode
      }
    }
  }
  return (
    <div
      className={cn(
        'bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6 transition-all hover:shadow-lg',
        className
      )}
    >
      {(title || icon) && (
        <div className="flex items-center gap-3 mb-4">
          {renderedIcon && <div className="text-primary-500">{renderedIcon}</div>}
          {title && <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{title}</h3>}
        </div>
      )}
      {children}
    </div>
  )
}

interface StatCardProps {
  title: string
  value: string | number
  icon: ReactNode | (() => ReactNode) | any
  trend?: string
  gradient: string
}

export function StatCard({ title, value, icon, trend, gradient }: StatCardProps) {
  // Normalize icon similar to Card
  let renderedIcon: ReactNode = null
  if (icon) {
    if (isValidElement(icon)) {
      renderedIcon = icon
    } else if (typeof icon === 'function') {
      try { renderedIcon = icon() } catch { /* ignore */ }
    } else if (icon && icon.type) {
      try { renderedIcon = createElement(icon.type) } catch { /* ignore */ }
    } else {
      renderedIcon = icon as ReactNode
    }
  }
  return (
    <Card className="flex items-center gap-4">
      <div
        className={cn('p-4 rounded-xl text-white flex items-center justify-center')}
        style={{ background: gradient }}
      >
        {renderedIcon}
      </div>
      <div className="flex-1">
        <h3 className="text-3xl font-bold text-gray-900 dark:text-white">{value}</h3>
        <p className="text-sm text-gray-600 dark:text-gray-400">{title}</p>
        {trend && (
          <p className="text-xs text-success-600 dark:text-success-400 mt-1 font-semibold">
            {trend}
          </p>
        )}
      </div>
    </Card>
  )
}
