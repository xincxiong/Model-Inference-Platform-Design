'use client'

import { useState, useCallback } from 'react'
import { useNotifications } from '@/stores/appStore'

interface ApiError {
  message: string
  code?: string
  status?: number
}

export function useErrorHandler() {
  const { addNotification } = useNotifications()

  const handleError = useCallback((error: unknown, context?: string): ApiError => {
    let errorMessage: string
    let errorCode: string | undefined
    let statusCode: number | undefined

    if (error instanceof Error) {
      errorMessage = error.message
    } else if (typeof error === 'string') {
      errorMessage = error
    } else if (error && typeof error === 'object') {
      const err = error as Record<string, unknown>
      errorMessage = (err.message as string) || (err.error as string) || '未知错误'
      errorCode = err.code as string | undefined
      statusCode = err.status as number | undefined
    } else {
      errorMessage = '未知错误'
    }

    const prefix = context ? `[${context}] ` : ''
    const fullMessage = `${prefix}${errorMessage}`

    console.error('API Error:', {
      message: errorMessage,
      code: errorCode,
      status: statusCode,
      context,
      originalError: error,
    })

    addNotification({
      type: 'error',
      message: fullMessage,
    })

    return {
      message: fullMessage,
      code: errorCode,
      status: statusCode,
    }
  }, [addNotification])

  const handleApiCall = useCallback(async <T>(
    apiCall: () => Promise<T>,
    options?: {
      context?: string
      showError?: boolean
      onError?: (error: ApiError) => void
      onSuccess?: (result: T) => void
    }
  ): Promise<T | null> => {
    const { context, showError = true, onError, onSuccess } = options || {}

    try {
      const result = await apiCall()
      onSuccess?.(result)
      return result
    } catch (error) {
      const apiError = handleError(error, context)
      if (showError) {
        onError?.(apiError)
      }
      return null
    }
  }, [handleError])

  return {
    handleError,
    handleApiCall,
  }
}

export function useAsyncOperation<T, P extends unknown[]>(
  operation: (...args: P) => Promise<T>,
  options?: {
    context?: string
    showLoading?: boolean
    showError?: boolean
    onSuccess?: (result: T) => void
    onError?: (error: ApiError) => void
  }
) {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<ApiError | null>(null)
  const { addNotification } = useNotifications()
  const { handleError } = useErrorHandler()

  const execute = useCallback(async (...args: P): Promise<T | null> => {
    setIsLoading(true)
    setError(null)

    try {
      const result = await operation(...args)
      setIsLoading(false)
      options?.onSuccess?.(result)
      return result
    } catch (err) {
      setIsLoading(false)
      const apiError = handleError(err, options?.context)
      setError(apiError)

      if (options?.showError !== false) {
        options?.onError?.(apiError)
      }

      return null
    }
  }, [operation, options, handleError])

  const reset = useCallback(() => {
    setIsLoading(false)
    setError(null)
  }, [])

  return {
    execute,
    isLoading,
    error,
    reset,
  }
}

export function createApiError(message: string, code?: string, status?: number): ApiError {
  return { message, code, status }
}

export function isApiError(error: unknown): error is ApiError {
  return (
    typeof error === 'object' &&
    error !== null &&
    'message' in error &&
    typeof (error as ApiError).message === 'string'
  )
}
