import { useState, useEffect, useCallback } from 'react'
import { api } from '../api/client'
import type { Label } from '../api/types'

export function useLabels() {
  const [labels, setLabels] = useState<Label[]>([])
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const data = await api.listLabels()
      setLabels(data)
    } catch {
      // Labels failing shouldn't break the app
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { labels, loading, refresh }
}
