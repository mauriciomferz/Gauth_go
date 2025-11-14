import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(date: string | Date): string {
  return new Date(date).toLocaleString()
}

export function formatDuration(ns: number): string {
  if (ns < 1000) return `${ns.toFixed(2)} ns`
  if (ns < 1000000) return `${(ns / 1000).toFixed(2)} µs`
  return `${(ns / 1000000).toFixed(2)} ms`
}

export function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}
