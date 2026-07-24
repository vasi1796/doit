import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'

export interface ContextMenuItem {
  label: string
  danger?: boolean
  onSelect: () => void
}

interface ContextMenuProps {
  open: boolean
  position: { x: number; y: number }
  items: ContextMenuItem[]
  onClose: () => void
}

/**
 * Fixed-position popover menu opened by long-press (touch) or
 * right-click / the row's … button (desktop). Clamped to the viewport,
 * dismissed by outside click or Escape.
 */
export function ContextMenu({ open, position, items, onClose }: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)

  // Clamp to the viewport using estimated dimensions so the menu never
  // renders off-screen — computed at render time to avoid a measure pass.
  const estimatedWidth = 200
  const estimatedHeight = items.length * 44 + 8
  const clamped = {
    x: Math.max(8, Math.min(position.x, window.innerWidth - estimatedWidth - 8)),
    y: Math.max(8, Math.min(position.y, window.innerHeight - estimatedHeight - 8)),
  }

  // Focus depends only on `open` — keeping it out of the listener effect so
  // a parent re-render (new onClose identity) can't yank focus back to the
  // first item mid-keyboard-navigation.
  useEffect(() => {
    if (!open) return
    menuRef.current?.querySelector<HTMLElement>('[role="menuitem"]')?.focus()
  }, [open])

  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
        return
      }
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault()
        const buttons = Array.from(
          menuRef.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? []
        )
        const idx = buttons.indexOf(document.activeElement as HTMLElement)
        const next = e.key === 'ArrowDown'
          ? buttons[(idx + 1) % buttons.length]
          : buttons[(idx - 1 + buttons.length) % buttons.length]
        next?.focus()
      }
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open, onClose])

  if (!open) return null

  return createPortal(
    <div className="fixed inset-0 z-[150]">
      {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
      <div className="absolute inset-0" onClick={onClose} onContextMenu={(e) => { e.preventDefault(); onClose() }} />
      <div
        ref={menuRef}
        role="menu"
        className="absolute min-w-[180px] bg-bg-elevated rounded-[14px] shadow-modal border border-separator overflow-hidden py-1 animate-[fade-in_0.12s_ease-out]"
        style={{ left: clamped.x, top: clamped.y }}
      >
        {items.map((item) => (
          <button
            key={item.label}
            type="button"
            role="menuitem"
            onClick={() => { onClose(); item.onSelect() }}
            className={`w-full text-left px-4 min-h-[44px] text-[15px] flex items-center transition-colors ${
              item.danger
                ? 'text-danger hover:bg-danger/10'
                : 'text-text-primary hover:bg-bg-secondary'
            }`}
          >
            {item.label}
          </button>
        ))}
      </div>
    </div>,
    document.body
  )
}
