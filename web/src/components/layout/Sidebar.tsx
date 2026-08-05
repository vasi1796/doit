import { useState, useEffect, useCallback } from 'react'
import { NavLink, useNavigate } from 'react-router'
import { motion, AnimatePresence, useMotionValue, useTransform } from 'framer-motion'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  TouchSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy, sortableKeyboardCoordinates, useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { computeDropPosition, healMissingPositions } from '../../utils/reorder'
import * as operations from '../../db/operations'
import { signOut } from '../../session'
import { useToast } from '../common/Toast'
import { ConfirmDialog } from '../common/ConfirmDialog'
import { ContextMenu } from '../common/ContextMenu'
import { EditNameColourDialog } from '../common/EditNameColourDialog'
import { useLongPress, LONG_PRESS_MS } from '../../hooks/useLongPress'
import { SyncStatus } from '../common/SyncStatus'
import { isPushSupported, isPushSubscribed, subscribeToPush, unsubscribeFromPush } from '../../push'
import { CalendarFeedLink } from '../common/CalendarFeedLink'
import { ColorSwatchRow } from '../common/ColorSwatchRow'
import { ThemeToggle } from '../common/ThemeToggle'
import type { List, Label } from '../../api/types'
import { PRESET_COLORS } from '../../constants'

const SWIPE_THRESHOLD = 80

const NAV_ITEMS = [
  { to: '/inbox', label: 'Inbox', icon: 'M22 12h-6l-2 3h-4l-2-3H2M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z' },
  { to: '/today', label: 'Today', icon: 'M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z' },
  { to: '/upcoming', label: 'Upcoming', icon: 'M13 17l5-5-5-5M6 17l5-5-5-5' },
  { to: '/matrix', label: 'Matrix', icon: 'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z' },
  { to: '/calendar', label: 'Calendar', icon: 'M8 2v4M16 2v4M3 10h18M21 6v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z' },
]

const BOTTOM_ITEMS = [
  { to: '/completed', label: 'Completed', icon: 'M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z' },
  { to: '/trash', label: 'Trash', icon: 'M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16' },
]

interface SidebarProps {
  lists: List[]
  labels: Label[]
  taskCounts: {
    inbox: number
    today: number
    upcoming: number
    byList: Record<string, number>
  }
  onSearchOpen?: () => void
}

function NotificationToggle() {
  const { toast } = useToast()
  const [supported] = useState(isPushSupported)
  const [subscribed, setSubscribed] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (supported) {
      isPushSubscribed().then(setSubscribed)
    }
  }, [supported])

  if (!supported) return null

  const handleToggle = async () => {
    setLoading(true)
    try {
      if (subscribed) {
        await unsubscribeFromPush()
        setSubscribed(false)
        toast('Reminders disabled', 'success')
      } else {
        const ok = await subscribeToPush()
        if (ok) {
          setSubscribed(true)
          toast('Due date reminders enabled', 'success')
        } else {
          toast('Notification permission denied', 'error')
        }
      }
    } catch {
      toast('Failed to update notifications', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="px-2">
      <button
        type="button"
        onClick={handleToggle}
        disabled={loading}
        className="flex items-center gap-3 px-3 min-h-[44px] rounded-[10px] text-[13px] text-text-secondary hover:bg-black/[0.04] w-full transition-colors"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" />
        </svg>
        <span className="flex-1 text-left">Reminders</span>
        <span style={{
          display: 'inline-block',
          flexShrink: 0,
          width: 44,
          height: 26,
          borderRadius: 13,
          backgroundColor: subscribed ? 'var(--color-accent)' : 'var(--color-bg-tertiary)',
          position: 'relative',
          transition: 'background-color 0.2s',
        }}>
          <span style={{
            position: 'absolute',
            top: 3,
            left: subscribed ? 21 : 3,
            width: 20,
            height: 20,
            borderRadius: 10,
            backgroundColor: '#fff',
            boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
            transition: 'left 0.2s',
          }} />
        </span>
      </button>
    </div>
  )
}

function NavItem({ to, label, icon, count, badgeTone = 'default' }: {
  to: string
  label: string
  icon: string
  count?: number
  badgeTone?: 'default' | 'danger'
}) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `relative flex items-center gap-3 px-3 min-h-[44px] rounded-[10px] text-[15px] transition-colors ${
          isActive
            ? 'text-accent font-medium'
            : 'text-text-primary hover:bg-black/[0.04]'
        }`
      }
    >
      {({ isActive }) => (
        <>
          {isActive && (
            <span
              aria-hidden="true"
              className="absolute inset-0 rounded-[10px] bg-accent-light animate-nav-active-in"
            />
          )}
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            className={`shrink-0 relative z-[1] ${isActive ? '' : 'text-text-secondary'}`}
          >
            <path d={icon} />
          </svg>
          <span className="flex-1 relative z-[1]">{label}</span>
          {count !== undefined && count > 0 && (
            badgeTone === 'danger' ? (
              <span className="relative z-[1] inline-flex items-center justify-center min-w-[20px] h-[20px] px-1.5 rounded-full text-[11px] font-semibold bg-danger text-white">
                {count}
              </span>
            ) : (
              <span className={`text-[12px] font-medium relative z-[1] ${isActive ? 'text-accent/70' : 'text-text-tertiary'}`}>{count}</span>
            )
          )}
        </>
      )}
    </NavLink>
  )
}

function useIsTouchDevice() {
  const [isTouch] = useState(() => typeof window !== 'undefined' && ('ontouchstart' in window || navigator.maxTouchPoints > 0))
  return isTouch
}

function SwipeableRow({ onDelete, desktopActionButton, children }: {
  onDelete: () => void
  desktopActionButton: React.ReactNode
  children: React.ReactNode
}) {
  const isTouch = useIsTouchDevice()
  const swipeX = useMotionValue(0)
  const bgOpacity = useTransform(swipeX, [-SWIPE_THRESHOLD, -20, 0], [1, 0.5, 0])

  const handleSwipeEnd = (_: unknown, info: { offset: { x: number } }) => {
    if (info.offset.x <= -SWIPE_THRESHOLD) {
      onDelete()
    }
  }

  return (
    <div className="group relative overflow-hidden rounded-lg flex items-center">
      {/* Red delete background — revealed on left swipe */}
      {isTouch && (
        <motion.div
          className="absolute inset-0 flex items-center justify-end px-4 bg-danger rounded-lg"
          style={{ opacity: bgOpacity }}
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16" />
          </svg>
        </motion.div>
      )}
      <motion.div
        drag={isTouch ? 'x' : false}
        dragConstraints={{ left: -120, right: 0 }}
        dragElastic={{ left: 0.2, right: 0 }}
        dragDirectionLock
        dragSnapToOrigin
        style={{ x: swipeX }}
        onDragEnd={handleSwipeEnd}
        className="relative flex-1"
      >
        {children}
      </motion.div>
      {desktopActionButton}
    </div>
  )
}

/**
 * A list/label row in the sidebar: swipe-to-delete on touch, plus a
 * context menu (Edit / Delete) opened by long-press on touch or by
 * right-click / the hover … button on desktop.
 */
function SidebarEntityRow({ to, name, colour, shape, count, entityLabel, onDelete, onOpenMenu, dragHandleProps }: {
  to: string
  name: string
  colour?: string
  shape: 'circle' | 'square'
  count?: number
  entityLabel: 'list' | 'label'
  onDelete: () => void
  onOpenMenu: (point: { x: number; y: number }) => void
  dragHandleProps?: Record<string, unknown>
}) {
  const longPress = useLongPress(onOpenMenu)

  return (
    <SwipeableRow
      onDelete={onDelete}
      desktopActionButton={
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            const rect = e.currentTarget.getBoundingClientRect()
            onOpenMenu({ x: rect.left, y: rect.bottom + 4 })
          }}
          className="hidden md:flex opacity-0 group-hover:opacity-100 focus:opacity-100 text-text-secondary hover:text-text-primary items-center justify-center w-[44px] h-[44px] mr-1 transition-opacity"
          aria-label={`${name} ${entityLabel} options`}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <circle cx="5" cy="12" r="1.8" />
            <circle cx="12" cy="12" r="1.8" />
            <circle cx="19" cy="12" r="1.8" />
          </svg>
        </button>
      }
    >
      <div
        {...longPress}
        onContextMenu={(e) => {
          // Shift+right-click falls through to the browser's native link
          // menu (Open in New Tab, Copy Link)
          if (e.shiftKey) return
          e.preventDefault()
          onOpenMenu({ x: e.clientX, y: e.clientY })
        }}
        className="flex-1 flex items-center select-none [-webkit-touch-callout:none]"
      >
        {dragHandleProps && (
          <button
            type="button"
            aria-label={`Drag to reorder ${name}`}
            className="w-[44px] min-h-[44px] -mr-2 self-stretch flex items-center justify-center shrink-0 touch-none cursor-grab active:cursor-grabbing text-text-quaternary hover:text-text-secondary"
            {...dragHandleProps}
            // Capture phase: framer's native row listener fires before React
            // bubble handlers, and a capture stop skips this element's own
            // bubble onPointerDown — hence the manual activator call
            onPointerDownCapture={(e) => {
              ;(dragHandleProps.onPointerDown as ((e: React.PointerEvent) => void) | undefined)?.(e)
              e.stopPropagation()
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <circle cx="9" cy="6" r="1.5" />
              <circle cx="15" cy="6" r="1.5" />
              <circle cx="9" cy="12" r="1.5" />
              <circle cx="15" cy="12" r="1.5" />
              <circle cx="9" cy="18" r="1.5" />
              <circle cx="15" cy="18" r="1.5" />
            </svg>
          </button>
        )}
        <NavLink
          to={to}
          // Anchors are natively draggable — WebKit's native drag would steal
          // the pointer stream before the sort sensor can activate
          draggable={false}
          className={({ isActive }) =>
            `relative flex-1 flex items-center gap-3 px-3 min-h-[44px] rounded-[10px] text-[15px] transition-colors ${
              isActive
                ? 'text-accent font-medium'
                : 'text-text-primary hover:bg-black/[0.04]'
            }`
          }
        >
          {({ isActive }) => (
            <>
              {isActive && (
                <span
                  aria-hidden="true"
                  className="absolute inset-0 rounded-[10px] bg-accent-light animate-nav-active-in"
                />
              )}
              <span
                className={`w-[10px] h-[10px] shrink-0 relative z-[1] ${shape === 'circle' ? 'rounded-full' : 'rounded-[3px]'}`}
                style={{ backgroundColor: colour || 'var(--color-gray)' }}
              />
              <span className="flex-1 relative z-[1] truncate">{name}</span>
              {(count ?? 0) > 0 && (
                <span className={`text-[12px] font-medium relative z-[1] ${isActive ? 'text-accent/70' : 'text-text-tertiary'}`}>{count}</span>
              )}
            </>
          )}
        </NavLink>
      </div>
    </SwipeableRow>
  )
}

function SortableEntityRow({ id, ...rowProps }: { id: string } & Parameters<typeof SidebarEntityRow>[0]) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 50 : undefined,
    position: isDragging ? ('relative' as const) : undefined,
  }

  return (
    <div ref={setNodeRef} style={style}>
      <SidebarEntityRow {...rowProps} dragHandleProps={{ ...attributes, ...listeners }} />
    </div>
  )
}

export function Sidebar({ lists, labels, taskCounts, onSearchOpen }: SidebarProps) {
  const { toast } = useToast()
  const navigate = useNavigate()
  const [addingList, setAddingList] = useState(false)
  const [newListName, setNewListName] = useState('')
  const [newListColour, setNewListColour] = useState<string>(PRESET_COLORS[0])
  const [pendingDelete, setPendingDelete] = useState<{ type: 'list' | 'label'; id: string; name: string } | null>(null)
  const [labelsCollapsed, setLabelsCollapsed] = useState(false)
  const [entityMenu, setEntityMenu] = useState<{ type: 'list' | 'label'; id: string; name: string; colour: string; x: number; y: number } | null>(null)
  const [editTarget, setEditTarget] = useState<{ type: 'list' | 'label'; id: string; name: string; colour: string } | null>(null)

  const pointerSensor = useSensor(PointerSensor, {
    activationConstraint: { distance: 8 },
  })
  const touchSensor = useSensor(TouchSensor, {
    activationConstraint: { delay: LONG_PRESS_MS, tolerance: 5 },
  })
  const keyboardSensor = useSensor(KeyboardSensor, {
    coordinateGetter: sortableKeyboardCoordinates,
  })
  const sensors = useSensors(pointerSensor, touchSensor, keyboardSensor)

  const handleListDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event
      if (!over || active.id === over.id) return
      const position = computeDropPosition(lists, String(active.id), String(over.id))
      if (!position) return
      operations.updateList(String(active.id), { position }).catch((err: unknown) => {
        toast(err instanceof Error ? err.message : 'Failed to reorder', 'error')
      })
    },
    [lists, toast],
  )

  const handleLabelDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event
      if (!over || active.id === over.id) return
      const reportError = (err: unknown) => {
        toast(err instanceof Error ? err.message : 'Failed to reorder', 'error')
      }
      // Legacy rows (e.g. from a projection rebuild) may lack positions;
      // give them keys matching their display order so the drop lands
      // exactly where the user put it.
      const healed = healMissingPositions(
        labels,
        (id, position) => {
          operations.updateLabel(id, { position }).catch(reportError)
        },
        String(active.id),
      )
      const position = computeDropPosition(healed, String(active.id), String(over.id))
      if (!position) return
      operations.updateLabel(String(active.id), { position }).catch(reportError)
    },
    [labels, toast],
  )

  const handleEditSave = useCallback(async (changes: { name?: string; colour?: string }) => {
    if (!editTarget) return
    setEditTarget(null)
    if (Object.keys(changes).length === 0) return
    try {
      if (editTarget.type === 'list') {
        await operations.updateList(editTarget.id, changes)
      } else {
        await operations.updateLabel(editTarget.id, changes)
      }
      toast(editTarget.type === 'list' ? 'List updated' : 'Label updated', 'success')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to update', 'error')
    }
  }, [editTarget, toast])

  const confirmDelete = useCallback(async () => {
    if (!pendingDelete) return
    const { type, id } = pendingDelete
    setPendingDelete(null)
    try {
      if (type === 'list') {
        await operations.deleteList(id)
        toast('List deleted', 'success')
      } else {
        await operations.deleteLabel(id)
        toast('Label deleted', 'success')
      }
      navigate('/inbox')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to delete', 'error')
    }
  }, [pendingDelete, toast, navigate])

  const handleCreateList = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newListName.trim()) return
    try {
      await operations.createList({
        name: newListName.trim(),
        colour: newListColour,
        position: Date.now().toString(),
      })
      setNewListName('')
      setAddingList(false)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  return (
    <aside
      className="frosted-sidebar h-screen border-r border-separator flex flex-col shrink-0 overflow-y-auto pb-[60px] md:pb-0"
      style={{ width: 'var(--sidebar-width)' }}
    >
      {/* App title */}
      <div className="px-5 pt-5 pb-2 flex items-center gap-3">
        <div
          className="w-7 h-7 rounded-[6px] bg-accent flex items-center justify-center text-white font-bold text-[14px]"
          aria-hidden="true"
        >
          D
        </div>
        <h1 className="text-[20px] font-semibold text-text-primary tracking-tight">DoIt</h1>
      </div>

      {/* Search */}
      {onSearchOpen && (
        <div className="px-3 mt-3 mb-2">
          <button
            type="button"
            onClick={onSearchOpen}
            className="flex items-center gap-2 px-3 min-h-[36px] w-full rounded-[6px] bg-bg-tertiary hover:bg-separator-opaque text-[13px] text-text-tertiary transition-colors"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="shrink-0">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
            <span className="flex-1 text-left">Search</span>
            <kbd className="hidden md:inline-block font-mono text-[11px] text-text-quaternary bg-bg border border-separator rounded px-1.5 py-px">⌘K</kbd>
          </button>
        </div>
      )}

      {/* Smart lists */}
      <nav aria-label="Views" className="px-2 space-y-0.5">
        {NAV_ITEMS.map((item) => {
          const countMap: Record<string, number> = {
            '/inbox': taskCounts.inbox,
            '/today': taskCounts.today,
            '/upcoming': taskCounts.upcoming,
          }
          return (
            <NavItem
              key={item.to}
              {...item}
              count={countMap[item.to]}
              badgeTone={item.to === '/today' && taskCounts.today > 0 ? 'danger' : 'default'}
            />
          )
        })}
      </nav>

      {/* Divider */}
      <div className="mx-5 my-3 border-t border-separator" />

      {/* User lists */}
      <div className="px-2 flex-1">
        <div className="flex items-center justify-between px-3 mb-1">
          <p className="text-[11px] font-semibold text-text-tertiary uppercase tracking-wider">Lists</p>
          <button
            type="button"
            onClick={() => setAddingList(!addingList)}
            className="w-6 h-6 flex items-center justify-center rounded-[6px] text-text-tertiary hover:text-accent hover:bg-accent-light transition-colors"
            aria-label={addingList ? 'Cancel new list' : 'New list'}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
              <path d="M12 5v14M5 12h14" />
            </svg>
          </button>
        </div>
        <AnimatePresence initial={false}>
        {addingList && (
          <motion.form
            onSubmit={handleCreateList}
            className="mx-1 mb-2 rounded-[14px] bg-bg-elevated border border-separator shadow-card overflow-hidden"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
          >
            <div className="px-3 pt-3 pb-2">
              <input
                type="text"
                value={newListName}
                onChange={(e) => setNewListName(e.target.value)}
                placeholder="List name"
                className="w-full text-[16px] font-medium outline-none bg-transparent text-text-primary placeholder:text-text-tertiary"
                // eslint-disable-next-line jsx-a11y/no-autofocus
                autoFocus
              />
            </div>
            <div className="px-3 pb-2">
              <ColorSwatchRow value={newListColour} onChange={setNewListColour} />
            </div>
            <div className="flex border-t border-separator">
              <button
                type="button"
                onClick={() => { setNewListName(''); setAddingList(false) }}
                className="flex-1 text-sm text-text-secondary font-medium py-2.5 hover:bg-bg-secondary transition-colors min-h-[44px]"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!newListName.trim()}
                className="flex-1 text-sm text-accent font-semibold py-2.5 hover:bg-accent-light transition-colors border-l border-separator min-h-[44px] disabled:opacity-30"
              >
                Create
              </button>
            </div>
          </motion.form>
        )}
        </AnimatePresence>
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleListDragEnd}
        >
          <SortableContext items={lists.map((l) => l.id)} strategy={verticalListSortingStrategy}>
            <div className="space-y-0.5">
              {lists.map((list) => (
                <SortableEntityRow
                  key={list.id}
                  id={list.id}
                  to={`/lists/${list.id}`}
                  name={list.name}
                  colour={list.colour}
                  shape="circle"
                  count={taskCounts.byList[list.id] || 0}
                  entityLabel="list"
                  onDelete={() => setPendingDelete({ type: 'list', id: list.id, name: list.name })}
                  onOpenMenu={(point) => setEntityMenu({
                    type: 'list', id: list.id, name: list.name,
                    colour: list.colour || '', x: point.x, y: point.y,
                  })}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      </div>

      {/* Labels */}
      {labels.length > 0 && (
        <div className="px-2 mt-2">
          <div className="mx-3 mb-2 border-t border-separator" />
          <button
            type="button"
            onClick={() => setLabelsCollapsed((v) => !v)}
            className="flex items-center gap-1.5 w-full px-3 py-1 text-[11px] font-semibold text-text-tertiary uppercase tracking-wider mb-1 hover:text-text-secondary transition-colors"
            aria-expanded={!labelsCollapsed}
            aria-controls="sidebar-labels-list"
          >
            <svg
              width="10"
              height="10"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              className={`shrink-0 transition-transform ${labelsCollapsed ? '-rotate-90' : ''}`}
              aria-hidden="true"
            >
              <polyline points="6 9 12 15 18 9" />
            </svg>
            <span>Labels</span>
          </button>
          <AnimatePresence initial={false}>
            {!labelsCollapsed && (
              <motion.div
                id="sidebar-labels-list"
                className="space-y-0.5 overflow-hidden"
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{ duration: 0.2, ease: 'easeOut' }}
              >
                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  onDragEnd={handleLabelDragEnd}
                >
                  <SortableContext items={labels.map((l) => l.id)} strategy={verticalListSortingStrategy}>
                    {labels.map((label) => (
                      <SortableEntityRow
                        key={label.id}
                        id={label.id}
                        to={`/labels/${label.id}`}
                        name={label.name}
                        colour={label.colour}
                        shape="square"
                        entityLabel="label"
                        onDelete={() => setPendingDelete({ type: 'label', id: label.id, name: label.name })}
                        onOpenMenu={(point) => setEntityMenu({
                          type: 'label', id: label.id, name: label.name,
                          colour: label.colour || '', x: point.x, y: point.y,
                        })}
                      />
                    ))}
                  </SortableContext>
                </DndContext>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      )}

      {/* Bottom section */}
      <nav aria-label="Archive" className="px-2 pb-2 space-y-0.5">
        <div className="mx-3 mb-2 border-t border-separator" />
        {BOTTOM_ITEMS.map((item) => (
          <NavItem key={item.to} {...item} />
        ))}
      </nav>

      {/* Sync status */}
      <SyncStatus />

      {/* Notification toggle */}
      <NotificationToggle />

      {/* Calendar feed */}
      <CalendarFeedLink />

      {/* Theme toggle */}
      <ThemeToggle />

      {/* Logout */}
      <div className="px-2 pb-4">
        <button
          type="button"
          onClick={async () => {
            try {
              await signOut()
            } catch (err) {
              toast(err instanceof Error ? err.message : 'Failed to sign out', 'error')
            }
          }}
          className="flex items-center gap-3 px-3 min-h-[44px] rounded-[10px] text-[13px] text-text-secondary hover:bg-black/[0.04] w-full transition-colors"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
          Sign out
        </button>
      </div>
      <ConfirmDialog
        open={!!pendingDelete}
        title={`Delete ${pendingDelete?.type === 'list' ? 'List' : 'Label'}`}
        message={
          pendingDelete?.type === 'list'
            ? `"${pendingDelete.name}" will be deleted. Tasks in this list will be moved to Inbox.`
            : `"${pendingDelete?.name}" will be removed from all tasks.`
        }
        onConfirm={confirmDelete}
        onCancel={() => setPendingDelete(null)}
      />
      <ContextMenu
        open={!!entityMenu}
        position={{ x: entityMenu?.x ?? 0, y: entityMenu?.y ?? 0 }}
        items={[
          {
            label: 'Edit',
            onSelect: () => {
              if (entityMenu) {
                setEditTarget({ type: entityMenu.type, id: entityMenu.id, name: entityMenu.name, colour: entityMenu.colour })
              }
            },
          },
          {
            label: 'Delete',
            danger: true,
            onSelect: () => {
              if (entityMenu) {
                setPendingDelete({ type: entityMenu.type, id: entityMenu.id, name: entityMenu.name })
              }
            },
          },
        ]}
        onClose={() => setEntityMenu(null)}
      />
      <EditNameColourDialog
        key={editTarget ? `${editTarget.type}-${editTarget.id}` : 'closed'}
        open={!!editTarget}
        title={editTarget?.type === 'list' ? 'Edit List' : 'Edit Label'}
        initialName={editTarget?.name ?? ''}
        // Empty string for an unset colour: picking any swatch — including the
        // first preset — then registers as a real change
        initialColour={editTarget?.colour ?? ''}
        onSave={handleEditSave}
        onCancel={() => setEditTarget(null)}
      />
    </aside>
  )
}
