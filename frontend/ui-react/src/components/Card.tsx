import { ReactNode, isValidElement, createElement, ElementType } from 'react'
import { cn } from '../lib/utils'

type IconProp = ReactNode | ElementType | { default: ElementType }

interface CardProps {
  children: ReactNode
  className?: string
  title?: string
  // Allow either a ready element or a component type
  icon?: IconProp
}

export function Card({ children, className, title, icon }: CardProps) {
  let renderedIcon: ReactNode | null = null
  if (icon) {
    if (isValidElement(icon)) {
      renderedIcon = icon
    } else if (typeof icon === 'function' || typeof icon === 'string') {
      renderedIcon = createElement(icon as ElementType)
    } else if (typeof icon === 'object' && icon && 'default' in icon) {
      const def = (icon as { default: ElementType }).default
      if (typeof def === 'function' || typeof def === 'string') {
        renderedIcon = createElement(def)
      }
    } else {
      renderedIcon = icon as ReactNode
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
  icon: IconProp
  trend?: string
  gradient: string
}

export function StatCard({ title, value, icon, trend, gradient }: StatCardProps) {
  let renderedIcon: ReactNode | null = null
  if (icon) {
    if (isValidElement(icon)) {
      renderedIcon = icon
    } else if (typeof icon === 'function' || typeof icon === 'string') {
      renderedIcon = createElement(icon as ElementType)
    } else if (typeof icon === 'object' && icon && 'default' in icon) {
      const def = (icon as { default: ElementType }).default
      if (typeof def === 'function' || typeof def === 'string') {
        renderedIcon = createElement(def)
      }
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
