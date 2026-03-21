import { createContext, useContext, useEffect, useMemo, useRef } from 'react'
import { Outlet } from 'react-router'
import { Sidebar } from './Sidebar'
import { BottomNav } from './BottomNav'
import { useLists } from '../../hooks/useLists'
import { useLabels } from '../../hooks/useLabels'
import { useTasks } from '../../hooks/useTasks'
import type { List, Label } from '../../api/types'

interface LayoutContext {
  lists: List[]
  labels: Label[]
  refreshLists: () => void
  refreshLabels: () => void
  refreshCounts: () => void
  quickAddRef: React.RefObject<{ focus: () => void } | null>
  taskCounts: {
    inbox: number
    today: number
    upcoming: number
    byList: Record<string, number>
  }
}

const LayoutCtx = createContext<LayoutContext>({
  lists: [],
  labels: [],
  refreshLists: () => {},
  refreshLabels: () => {},
  refreshCounts: () => {},
  quickAddRef: { current: null },
  taskCounts: { inbox: 0, today: 0, upcoming: 0, byList: {} },
})

export function useLayoutContext() {
  return useContext(LayoutCtx)
}

export function AppLayout() {
  const { lists, refresh: refreshLists } = useLists()
  const { labels, refresh: refreshLabels } = useLabels()
  const { tasks, refresh: refreshTasks } = useTasks({ is_completed: 'false' })
  const quickAddRef = useRef<{ focus: () => void } | null>(null)

  // Compute task counts
  const today = new Date().toISOString().split('T')[0]
  const nextWeek = new Date()
  nextWeek.setDate(nextWeek.getDate() + 7)
  const nextWeekStr = nextWeek.toISOString().split('T')[0]

  const taskCounts = {
    inbox: tasks.filter((t) => !t.list_id).length,
    today: tasks.filter((t) => t.due_date === today).length,
    upcoming: tasks.filter((t) => t.due_date && t.due_date > today && t.due_date <= nextWeekStr).length,
    byList: tasks.reduce<Record<string, number>>((acc, t) => {
      if (t.list_id) acc[t.list_id] = (acc[t.list_id] || 0) + 1
      return acc
    }, {}),
  }

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      const isInInput = e.target instanceof HTMLElement && ['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName)

      // Cmd+N — always try (works in PWA, browser may also capture)
      if ((e.metaKey || e.ctrlKey) && e.key === 'n') {
        e.preventDefault()
        quickAddRef.current?.focus()
      }
      // "/" to focus quick add (only if not already in an input)
      if (e.key === '/' && !isInInput) {
        e.preventDefault()
        quickAddRef.current?.focus()
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [])

  return (
    <LayoutCtx.Provider value={useMemo(() => ({ lists, labels, refreshLists, refreshLabels, refreshCounts: refreshTasks, quickAddRef, taskCounts }), [lists, labels, refreshLists, refreshLabels, refreshTasks, quickAddRef, taskCounts])}>
      <div className="flex h-screen">
        <div className="hidden md:block">
          <Sidebar lists={lists} labels={labels} onListsChanged={refreshLists} onLabelsChanged={refreshLabels} taskCounts={taskCounts} />
        </div>

        <main className="flex-1 overflow-y-auto pb-[60px] md:pb-0">
          <Outlet />
        </main>

        <BottomNav taskCounts={taskCounts} />
      </div>
    </LayoutCtx.Provider>
  )
}
