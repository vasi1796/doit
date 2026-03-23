import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { Outlet, useLocation } from 'react-router'
import { Sidebar } from './Sidebar'
import { BottomNav } from './BottomNav'
import { QuickAdd } from '../tasks/QuickAdd'
import { useLists } from '../../hooks/useLists'
import { useLabels } from '../../hooks/useLabels'
import { useTasks } from '../../hooks/useTasks'
import { initialSync } from '../../db/initial-sync'
import { SyncEngine } from '../../db/sync-engine'
import { InstallBanner } from '../common/InstallBanner'
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

function QuickAddModal({ lists, labels, pathname, onClose }: { lists: List[]; labels: Label[]; pathname: string; onClose: () => void }) {
  const quickAddRef = useRef<{ focus: () => void } | null>(null)

  // Derive context from current route
  const listMatch = pathname.match(/^\/lists\/(.+)/)
  const labelMatch = pathname.match(/^\/labels\/(.+)/)
  const isToday = pathname === '/today'

  const prefilledListId = listMatch ? listMatch[1] : undefined
  const prefilledDueDate = isToday ? new Date().toISOString().split('T')[0] : undefined
  const prefilledLabelId = labelMatch ? labelMatch[1] : undefined

  // Auto-focus on mount + Escape to close
  useEffect(() => {
    setTimeout(() => quickAddRef.current?.focus(), 50)
    const handleEscape = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [onClose])

  return (
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions, jsx-a11y/no-noninteractive-element-interactions
    <div
      className="fixed inset-0 bg-black/20 z-[60] flex items-start justify-center pt-[15vh] animate-[fade-in_0.15s_ease-out]"
      role="dialog"
      aria-modal="true"
      aria-label="New task"
      onClick={onClose}
    >
      {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
      <div className="w-full max-w-[480px] mx-4" onClick={(e) => e.stopPropagation()}>
        <QuickAdd
          ref={quickAddRef}
          lists={lists}
          labels={labels}
          listId={prefilledListId}
          dueDate={prefilledDueDate}
          labelId={prefilledLabelId}
          onCreated={onClose}
          initialExpanded
        />
      </div>
    </div>
  )
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

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [quickAddOpen, setQuickAddOpen] = useState(false)

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      const isInInput = e.target instanceof HTMLElement && (
        ['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName) || e.target.isContentEditable
      )

      // Cmd+N — open global quick add
      if ((e.metaKey || e.ctrlKey) && e.key === 'n') {
        e.preventDefault()
        setQuickAddOpen(true)
      }
      // "/" to open quick add (only if not already in an input)
      if (e.key === '/' && !isInInput) {
        e.preventDefault()
        setQuickAddOpen(true)
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [])
  const toggleDrawer = useCallback(() => setDrawerOpen((v) => !v), [])
  const closeDrawer = useCallback(() => setDrawerOpen(false), [])
  const location = useLocation()

  // Close drawer on route change — setState in effect is intentional here
  const prevPath = useRef(location.pathname)
  useEffect(() => {
    if (prevPath.current !== location.pathname) {
      prevPath.current = location.pathname
      setDrawerOpen(false) // eslint-disable-line react-hooks/set-state-in-effect
    }
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
          {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
          <div className="absolute inset-0 bg-black/30" onClick={closeDrawer} />
          {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
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
          <InstallBanner />
          <Outlet />
        </main>

        <BottomNav taskCounts={taskCounts} onMenuToggle={toggleDrawer} />

        {/* Global quick-add FAB */}
        <button
          type="button"
          onClick={() => setQuickAddOpen(true)}
          aria-label="New task"
          className="fixed right-5 bottom-[80px] md:bottom-6 w-[56px] h-[56px] rounded-full bg-accent text-white shadow-lg flex items-center justify-center z-40 hover:bg-accent/90 active:scale-95 transition-transform"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
            <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
          </svg>
        </button>

        {/* Quick-add modal overlay */}
        {quickAddOpen && (
          <QuickAddModal
            lists={lists}
            labels={labels}
            pathname={location.pathname}
            onClose={() => setQuickAddOpen(false)}
          />
        )}
      </div>
    </LayoutCtx.Provider>
  )
}
