import { useState, useEffect, useCallback } from 'react'
import { api } from '../api/client'
import type { Task } from '../api/types'

export function useTasks(params?: Record<string, string>) {
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const key = params ? JSON.stringify(params) : ''

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.listTasks(params)
      setTasks(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tasks')
    } finally {
      setLoading(false)
    }
  }, [key])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { tasks, loading, error, refresh }
}
