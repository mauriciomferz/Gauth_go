import { useState, useCallback } from 'react'
import { AxiosError } from 'axios'
import { toast } from 'sonner'

interface UseApiCallOptions {
  onSuccess?: (data: any) => void
  onError?: (error: Error) => void
  showSuccessToast?: boolean
  showErrorToast?: boolean
  successMessage?: string
  errorMessage?: string
}

interface UseApiCallReturn<T> {
  data: T | null
  error: Error | null
  loading: boolean
  execute: (...args: any[]) => Promise<T | null>
  reset: () => void
}

/**
 * Custom hook for handling API calls with built-in loading, error, and success states.
 * Includes automatic retry logic and toast notifications.
 * 
 * @example
 * const { data, loading, error, execute } = useApiCall(apiClient.getTokens, {
 *   showSuccessToast: true,
 *   successMessage: 'Tokens loaded successfully'
 * })
 * 
 * useEffect(() => {
 *   execute()
 * }, [])
 */
export function useApiCall<T = any>(
  apiFunction: (...args: any[]) => Promise<T>,
  options: UseApiCallOptions = {}
): UseApiCallReturn<T> {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<Error | null>(null)
  const [loading, setLoading] = useState(false)

  const {
    onSuccess,
    onError,
    showSuccessToast = false,
    showErrorToast = true,
    successMessage,
    errorMessage,
  } = options

  const execute = useCallback(
    async (...args: any[]): Promise<T | null> => {
      setLoading(true)
      setError(null)

      try {
        const result = await apiFunction(...args)
        setData(result)

        if (showSuccessToast) {
          toast.success(successMessage || 'Operation completed successfully')
        }

        onSuccess?.(result)
        return result
      } catch (err) {
        const error = err instanceof Error ? err : new Error('An unknown error occurred')
        setError(error)

        if (showErrorToast) {
          const axiosError = err as AxiosError<{ message?: string }>
          const message =
            errorMessage ||
            axiosError?.response?.data?.message ||
            error.message ||
            'Operation failed'
          toast.error(message)
        }

        onError?.(error)
        return null
      } finally {
        setLoading(false)
      }
    },
    [apiFunction, onSuccess, onError, showSuccessToast, showErrorToast, successMessage, errorMessage]
  )

  const reset = useCallback(() => {
    setData(null)
    setError(null)
    setLoading(false)
  }, [])

  return {
    data,
    error,
    loading,
    execute,
    reset,
  }
}

/**
 * Hook for retryable API calls with exponential backoff
 */
export function useRetryableApiCall<T = any>(
  apiFunction: (...args: any[]) => Promise<T>,
  options: UseApiCallOptions & {
    maxRetries?: number
    retryDelay?: number
  } = {}
): UseApiCallReturn<T> & { retrying: boolean; retryCount: number } {
  const [retrying, setRetrying] = useState(false)
  const [retryCount, setRetryCount] = useState(0)
  
  const { maxRetries = 3, retryDelay = 1000, ...apiCallOptions } = options

  const baseHook = useApiCall(apiFunction, apiCallOptions)

  const executeWithRetry = useCallback(
    async (...args: any[]): Promise<T | null> => {
      let attempts = 0
      setRetryCount(0)

      while (attempts <= maxRetries) {
        if (attempts > 0) {
          setRetrying(true)
          await new Promise((resolve) =>
            setTimeout(resolve, retryDelay * Math.pow(2, attempts - 1))
          )
        }

        setRetryCount(attempts)
        const result = await baseHook.execute(...args)

        if (result !== null) {
          setRetrying(false)
          return result
        }

        attempts++
      }

      setRetrying(false)
      toast.error(`Operation failed after ${maxRetries} retries`)
      return null
    },
    [baseHook, maxRetries, retryDelay]
  )

  return {
    ...baseHook,
    execute: executeWithRetry,
    retrying,
    retryCount,
  }
}
