import { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'
import { cn } from '../lib/utils'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
}

export function Input({ label, error, className, ...props }: InputProps) {
  return (
    <div className="w-full">
      {label && (
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {label}
        </label>
      )}
      <input
        className={cn(
          'w-full px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-600',
          'bg-white dark:bg-gray-700 text-gray-900 dark:text-white',
          'focus:ring-2 focus:ring-primary-500 focus:border-primary-500',
          'disabled:opacity-50 disabled:cursor-not-allowed',
          'transition-colors',
          error && 'border-red-500 focus:ring-red-500',
          className
        )}
        {...props}
      />
      {error && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>}
    </div>
  )
}

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
  error?: string
}

export function Select({ label, error, className, children, ...props }: SelectProps) {
  return (
    <div className="w-full">
      {label && (
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {label}
        </label>
      )}
      <select
        className={cn(
          'w-full px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-600',
          'bg-white dark:bg-gray-700 text-gray-900 dark:text-white',
          'focus:ring-2 focus:ring-primary-500 focus:border-primary-500',
          'disabled:opacity-50 disabled:cursor-not-allowed',
          'transition-colors',
          error && 'border-red-500 focus:ring-red-500',
          className
        )}
        {...props}
      >
        {children}
      </select>
      {error && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>}
    </div>
  )
}

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string
  error?: string
}

export function Textarea({ label, error, className, ...props }: TextareaProps) {
  return (
    <div className="w-full">
      {label && (
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {label}
        </label>
      )}
      <textarea
        className={cn(
          'w-full px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-600',
          'bg-white dark:bg-gray-700 text-gray-900 dark:text-white',
          'focus:ring-2 focus:ring-primary-500 focus:border-primary-500',
          'disabled:opacity-50 disabled:cursor-not-allowed',
          'transition-colors resize-vertical',
          'font-mono text-sm',
          error && 'border-red-500 focus:ring-red-500',
          className
        )}
        {...props}
      />
      {error && <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>}
    </div>
  )
}
