import { useState, useEffect, useCallback } from 'react'
import { api } from '../api/client'
import type { List } from '../api/types'

export function useLists() {
  const [lists, setLists] = useState<List[]>([])
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const data = await api.listLists()
      setLists(data)
    } catch {
      // Lists failing shouldn't break the app
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { lists, loading, refresh }
}
