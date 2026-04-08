import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { Outlet, useLocation } from 'react-router'
import { motion } from 'framer-motion'
import { Sidebar } from './Sidebar'
import { BottomNav } from './BottomNav'
import { QuickAdd } from '../tasks/QuickAdd'
import { useLists } from '../../hooks/useLists'
import { useLabels } from '../../hooks/useLabels'
import { useTasks } from '../../hooks/useTasks'
import { initialSync } from '../../db/initial-sync'
import { SyncEngine } from '../../db/sync-engine'
import { InstallBanner } from '../common/InstallBanner'
import { SearchOverlay } from '../common/SearchOverlay'
import { TaskDetail } from '../tasks/TaskDetail'
import type { List, Label, Task } from '../../api/types'

interface TaskCounts {
  inbox: number
  today: number
  upcoming: number
  byList: Record<string, number>
}

interface LayoutContext {
  lists: List[]
  labels: Label[]
  quickAddRef: React.RefObject<{ focus: () => void } | null>
  taskCounts: TaskCounts
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

// ---------------------------------------------------------------------------
// Extracted hooks (co-located — tightly coupled to AppLayout)
// ---------------------------------------------------------------------------

function useTaskCounts(tasks: Task[]): TaskCounts {
  const today = new Date().toISOString().split('T')[0]
  const nextWeek = new Date()
  nextWeek.setDate(nextWeek.getDate() + 7)
  const nextWeekStr = nextWeek.toISOString().split('T')[0]

  return useMemo(() => ({
    inbox: tasks.filter((t) => !t.list_id).length,
    today: tasks.filter((t) => t.due_date === today).length,
    upcoming: tasks.filter((t) => t.due_date && t.due_date > today && t.due_date <= nextWeekStr).length,
    byList: tasks.reduce<Record<string, number>>((acc, t) => {
      if (t.list_id) acc[t.list_id] = (acc[t.list_id] || 0) + 1
      return acc
    }, {}),
  }), [tasks, today, nextWeekStr])
}

function useKeyboardShortcuts({
  setQuickAddOpen,
  setSearchOpen,
}: {
  setQuickAddOpen: React.Dispatch<React.SetStateAction<boolean>>
  setSearchOpen: React.Dispatch<React.SetStateAction<boolean>>
}) {
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      const isInInput = e.target instanceof HTMLElement && (
        ['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName) || e.target.isContentEditable
      )

      // Cmd+K — open search
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setSearchOpen(true)
        return
      }
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
  }, [setQuickAddOpen, setSearchOpen])
}

function useMobileDrawer(pathname: string) {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const toggleDrawer = useCallback(() => setDrawerOpen((v) => !v), [])
  const closeDrawer = useCallback(() => setDrawerOpen(false), [])

  // Close drawer on route change — setState in effect is intentional here
  const prevPath = useRef(pathname)
  useEffect(() => {
    if (prevPath.current !== pathname) {
      prevPath.current = pathname
      setDrawerOpen(false) // eslint-disable-line react-hooks/set-state-in-effect
    }
  }, [pathname])

  return { drawerOpen, toggleDrawer, closeDrawer }
}

// ---------------------------------------------------------------------------
// QuickAddModal (extracted component)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// AppLayout
// ---------------------------------------------------------------------------

export function AppLayout() {
  const { lists } = useLists()
  const { labels } = useLabels()
  const { tasks } = useTasks({ is_completed: 'false' })
  const quickAddRef = useRef<{ focus: () => void } | null>(null)
  const location = useLocation()

  // Populate IndexedDB from server, then start sync engine.
  useEffect(() => {
    const engine = new SyncEngine()
    initialSync()
      .then(() => engine.start())
      .catch(() => engine.start()) // Start sync even if initial load fails (may be offline)

    // Expose for testing — allows Playwright to trigger sync on demand
    window.__syncEngine = engine

    return () => {
      engine.stop()
      delete window.__syncEngine
    }
  }, [])

  const taskCounts = useTaskCounts(tasks)
  const { drawerOpen, toggleDrawer, closeDrawer } = useMobileDrawer(location.pathname)

  const [quickAddOpen, setQuickAddOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchSelectedId, setSearchSelectedId] = useState<string | null>(null)

  useKeyboardShortcuts({ setQuickAddOpen, setSearchOpen })

  return (
    <LayoutCtx.Provider value={useMemo(() => ({ lists, labels, quickAddRef, taskCounts }), [lists, labels, quickAddRef, taskCounts])}>
      <div className="flex h-screen">
        {/* Desktop sidebar */}
        <div className="hidden md:block">
          <Sidebar lists={lists} labels={labels} taskCounts={taskCounts} onSearchOpen={() => setSearchOpen(true)} />
        </div>

        {/* Mobile drawer — iOS-style spring easing.
            `inert` when closed: removes drawer children from tab order and
            the a11y tree without tripping axe's aria-hidden-focus rule. */}
        <div
          className={`fixed inset-0 z-50 md:hidden ${
            drawerOpen ? 'pointer-events-auto' : 'pointer-events-none'
          }`}
          inert={!drawerOpen}
        >
          {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
          <div
            className={`absolute inset-0 bg-[rgba(0,0,0,0.35)] transition-opacity duration-[350ms] ease-out ${
              drawerOpen ? 'opacity-100' : 'opacity-0'
            }`}
            onClick={closeDrawer}
          />
          {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
          <div
            className={`absolute left-0 top-0 h-full will-change-transform ${
              drawerOpen ? 'translate-x-0' : '-translate-x-full'
            }`}
            style={{ transition: 'transform 0.35s var(--ease-ios-spring)' }}
            onClick={(e) => e.stopPropagation()}
          >
            <Sidebar lists={lists} labels={labels} taskCounts={taskCounts} onSearchOpen={() => setSearchOpen(true)} />
          </div>
        </div>

        <main className="flex-1 overflow-y-auto pb-[60px] md:pb-0">
          <InstallBanner />
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.15 }}
          >
            <Outlet />
          </motion.div>
        </main>

        <BottomNav taskCounts={taskCounts} onMenuToggle={toggleDrawer} />

        {/* Global quick-add FAB — accent-tinted shadow */}
        <button
          type="button"
          onClick={() => setQuickAddOpen(true)}
          aria-label="New task"
          className="fixed right-5 bottom-[calc(var(--bottom-nav-height)+env(safe-area-inset-bottom,0px)+16px)] md:bottom-6 w-[56px] h-[56px] rounded-full bg-accent text-white shadow-fab flex items-center justify-center z-50 hover:bg-accent-hover hover:scale-105 active:scale-95 transition-all"
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

        {/* Search overlay */}
        {searchOpen && (
          <SearchOverlay
            onClose={() => setSearchOpen(false)}
            onSelectTask={(id) => { setSearchOpen(false); setSearchSelectedId(id) }}
          />
        )}

        {/* Task detail from search */}
        {searchSelectedId && (
          <TaskDetail taskId={searchSelectedId} lists={lists} onClose={() => setSearchSelectedId(null)} />
        )}
      </div>
    </LayoutCtx.Provider>
  )
}
