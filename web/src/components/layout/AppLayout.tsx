import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { Outlet, useLocation } from 'react-router'
import { Sidebar } from './Sidebar'
import { BottomNav } from './BottomNav'
import { useLists } from '../../hooks/useLists'
import { useLabels } from '../../hooks/useLabels'
import { useTasks } from '../../hooks/useTasks'
import { initialSync } from '../../db/initial-sync'
import { SyncEngine } from '../../db/sync-engine'
import type { List, Label } from '../../api/types'

interface LayoutContext {
  lists: List[]
  labels: Label[]
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
  quickAddRef: { current: null },
  taskCounts: { inbox: 0, today: 0, upcoming: 0, byList: {} },
})

export function useLayoutContext() {
  return useContext(LayoutCtx)
}

export function AppLayout() {
  const { lists } = useLists()
  const { labels } = useLabels()
  const { tasks } = useTasks({ is_completed: 'false' })
  const quickAddRef = useRef<{ focus: () => void } | null>(null)

  // Populate IndexedDB from server, then start sync engine.
  useEffect(() => {
    const engine = new SyncEngine()
    initialSync()
      .then(() => engine.start())
      .catch(() => engine.start()) // Start sync even if initial load fails (may be offline)

    // Expose for testing — allows Playwright to trigger sync on demand
    ;(window as unknown as { __syncEngine?: SyncEngine }).__syncEngine = engine

    return () => {
      engine.stop()
      delete (window as unknown as { __syncEngine?: SyncEngine }).__syncEngine
    }
  }, [])

  // Compute task counts
  const today = new Date().toISOString().split('T')[0]
  const nextWeek = new Date()
  nextWeek.setDate(nextWeek.getDate() + 7)
  const nextWeekStr = nextWeek.toISOString().split('T')[0]

  const taskCounts = useMemo(() => ({
    inbox: tasks.filter((t) => !t.list_id).length,
    today: tasks.filter((t) => t.due_date === today).length,
    upcoming: tasks.filter((t) => t.due_date && t.due_date > today && t.due_date <= nextWeekStr).length,
    byList: tasks.reduce<Record<string, number>>((acc, t) => {
      if (t.list_id) acc[t.list_id] = (acc[t.list_id] || 0) + 1
      return acc
    }, {}),
  }), [tasks, today, nextWeekStr])

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

  const [drawerOpen, setDrawerOpen] = useState(false)
  const toggleDrawer = useCallback(() => setDrawerOpen((v) => !v), [])
  const closeDrawer = useCallback(() => setDrawerOpen(false), [])
  const location = useLocation()

  // Close drawer on navigation (user tapped a list/label link)
  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

  return (
    <LayoutCtx.Provider value={useMemo(() => ({ lists, labels, quickAddRef, taskCounts }), [lists, labels, quickAddRef, taskCounts])}>
      <div className="flex h-screen">
        {/* Desktop sidebar */}
        <div className="hidden md:block">
          <Sidebar lists={lists} labels={labels} taskCounts={taskCounts} />
        </div>

        {/* Mobile drawer */}
        <div
          className={`fixed inset-0 z-50 md:hidden transition-opacity duration-200 ${
            drawerOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'
          }`}
        >
          <div className="absolute inset-0 bg-black/30" onClick={closeDrawer} />
          <div
            className={`absolute left-0 top-0 h-full transition-transform duration-200 ${
              drawerOpen ? 'translate-x-0' : '-translate-x-full'
            }`}
            onClick={(e) => e.stopPropagation()}
          >
            <Sidebar lists={lists} labels={labels} taskCounts={taskCounts} />
          </div>
        </div>

        <main className="flex-1 overflow-y-auto pb-[60px] md:pb-0">
          <Outlet />
        </main>

        <BottomNav taskCounts={taskCounts} onMenuToggle={toggleDrawer} />
      </div>
    </LayoutCtx.Provider>
  )
}
