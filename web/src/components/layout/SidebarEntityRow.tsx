import { useState } from 'react'
import { NavLink } from 'react-router'
import { motion, useMotionValue, useTransform } from 'framer-motion'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useLongPress } from '../../hooks/useLongPress'

const SWIPE_THRESHOLD = 80

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

export function SortableEntityRow({ id, ...rowProps }: { id: string } & Parameters<typeof SidebarEntityRow>[0]) {
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
