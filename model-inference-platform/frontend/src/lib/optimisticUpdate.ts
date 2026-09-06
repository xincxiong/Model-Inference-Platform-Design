'use client'

import { useState, useCallback } from 'react'

interface OptimisticUpdateOptions<T> {
  onSuccess?: (result: T) => void
  onError?: (error: Error, previousState: T) => void
  errorMessage?: string
}

export function useOptimisticUpdate<T>(
  initialState: T,
  updateFn: (id: string, patch: Partial<T>) => Promise<T>,
) {
  const [state, setState] = useState<T>(initialState)
  const [isUpdating, setIsUpdating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const optimisticUpdate = useCallback(async (
    id: string,
    patch: Partial<T>,
    options: OptimisticUpdateOptions<T> = {}
  ) => {
    const previousState = state
    const { onSuccess, onError, errorMessage = '更新失败' } = options

    setIsUpdating(true)
    setError(null)

    setState((prev) => ({ ...prev, ...patch }))

    try {
      const result = await updateFn(id, patch)
      setState(result)
      onSuccess?.(result)
      return result
    } catch (err) {
      setState(previousState)
      const error = err instanceof Error ? err : new Error(errorMessage)
      setError(error.message)
      onError?.(error, previousState)
      throw error
    } finally {
      setIsUpdating(false)
    }
  }, [state, updateFn])

  const batchOptimisticUpdate = useCallback(async (
    updates: Array<{ id: string; patch: Partial<T> }>,
    options: { onSuccess?: (results: T[]) => void; onError?: (error: Error, previousState: T) => void; errorMessage?: string } = {}
  ) => {
    const previousState = state
    const { onSuccess, onError, errorMessage = '批量更新失败' } = options

    setIsUpdating(true)
    setError(null)

    const patches = updates.reduce((acc, { id, patch }) => {
      acc[id] = patch
      return acc
    }, {} as Record<string, Partial<T>>)

    setState((prev) => ({ ...prev, ...patches }))

    try {
      const results = await Promise.all(
        updates.map(({ id, patch }) => updateFn(id, patch))
      )
      onSuccess?.(results)
      return results
    } catch (err) {
      setState(previousState)
      const error = err instanceof Error ? err : new Error(errorMessage)
      setError(error.message)
      onError?.(error, previousState)
      throw error
    } finally {
      setIsUpdating(false)
    }
  }, [state, updateFn])

  const resetError = useCallback(() => {
    setError(null)
  }, [])

  const resetState = useCallback(() => {
    setState(initialState)
    setError(null)
    setIsUpdating(false)
  }, [initialState])

  return {
    state,
    isUpdating,
    error,
    optimisticUpdate,
    batchOptimisticUpdate,
    resetError,
    resetState,
    setState,
  }
}

interface OptimisticListUpdateOptions<T> {
  onSuccess?: (result: T) => void
  onError?: (error: Error, previousList: T[]) => void
  errorMessage?: string
  findById?: (item: T, id: string) => boolean
}

export function useOptimisticListUpdate<T extends { id: string }>(
  initialList: T[] = [],
  updateFn: (id: string, patch: Partial<T>) => Promise<T>,
) {
  const [list, setList] = useState<T[]>(initialList)
  const [isUpdating, setIsUpdating] = useState<Record<string, boolean>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})

  const optimisticUpdate = useCallback(async (
    id: string,
    patch: Partial<T>,
    options: OptimisticListUpdateOptions<T> = {}
  ) => {
    const previousList = [...list]
    const {
      onSuccess,
      onError,
      errorMessage = '更新失败',
      findById = (item) => item.id === id,
    } = options

    setIsUpdating((prev) => ({ ...prev, [id]: true }))
    setErrors((prev) => {
      const next = { ...prev }
      delete next[id]
      return next
    })

    setList((prev) =>
      prev.map((item) =>
        findById(item, id) ? { ...item, ...patch } : item
      )
    )

    try {
      const result = await updateFn(id, patch)
      setList((prev) =>
        prev.map((item) =>
          findById(item, id) ? result : item
        )
      )
      onSuccess?.(result)
      return result
    } catch (err) {
      setList(previousList)
      const error = err instanceof Error ? err : new Error(errorMessage)
      setErrors((prev) => ({ ...prev, [id]: error.message }))
      onError?.(error, previousList)
      throw error
    } finally {
      setIsUpdating((prev) => {
        const next = { ...prev }
        delete next[id]
        return next
      })
    }
  }, [list, updateFn])

  const clearError = useCallback((id: string) => {
    setErrors((prev) => {
      const next = { ...prev }
      delete next[id]
      return next
    })
  }, [])

  const resetList = useCallback(() => {
    setList(initialList)
    setErrors({})
    setIsUpdating({})
  }, [initialList])

  return {
    list,
    isUpdating,
    errors,
    optimisticUpdate,
    clearError,
    resetList,
    setList,
  }
}
