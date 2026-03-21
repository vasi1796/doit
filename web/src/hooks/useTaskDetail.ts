import { useState, useEffect, useCallback } from 'react'
import { api } from '../api/client'
import type { Task } from '../api/types'

export function useTaskDetail(id: string | null) {
  const [task, setTask] = useState<Task | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (!id) {
      setTask(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.getTask(id)
      setTask(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load task')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { task, loading, error, refresh }
}
