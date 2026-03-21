import { useState, useCallback } from 'react'
import { useTasks } from './useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'

/**
 * Wraps useTasks with layout context integration.
 * Provides a unified refreshAll that refreshes both the page's tasks and sidebar counts.
 */
export function usePageTasks(params?: Record<string, string>) {
  const { tasks, loading, error, refresh } = useTasks(params)
  const { refreshCounts } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const refreshAll = useCallback(() => {
    refresh()
    refreshCounts()
  }, [refresh, refreshCounts])

  return { tasks, loading, error, refresh, refreshAll, selectedId, setSelectedId }
}
