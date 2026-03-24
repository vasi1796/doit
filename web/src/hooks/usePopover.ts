import { useState, useRef, useEffect, useCallback } from 'react'

interface PopoverPosition {
  top: number
  left: number
}

interface UsePopoverOptions {
  /** Minimum width needed for the popover content */
  contentWidth?: number
  /** Minimum height needed for the popover content */
  contentHeight?: number
}

export function usePopover(options: UsePopoverOptions = {}) {
  const { contentWidth = 200, contentHeight = 300 } = options
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<PopoverPosition>({ top: 0, left: 0 })
  const triggerRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open || !triggerRef.current) return

    const reposition = () => {
      if (!triggerRef.current) return
      const rect = triggerRef.current.getBoundingClientRect()
      // Use visualViewport for accurate dimensions when iOS keyboard is open
      const vp = window.visualViewport
      const vpWidth = vp?.width ?? window.innerWidth
      const vpHeight = vp?.height ?? window.innerHeight
      const vpOffsetTop = vp?.offsetTop ?? 0

      const left = Math.min(rect.left, vpWidth - contentWidth)
      const top = rect.bottom + 4

      // If the popover would be below the visible viewport, flip it above the trigger
      const maxTop = vpOffsetTop + vpHeight - contentHeight
      setPos({
        top: top > maxTop ? Math.max(vpOffsetTop + 8, rect.top - contentHeight - 4) : top,
        left: Math.max(8, left),
      })
    }

    reposition()

    // Reposition when iOS keyboard resizes the visual viewport
    const vp = window.visualViewport
    vp?.addEventListener('resize', reposition)
    vp?.addEventListener('scroll', reposition)
    return () => {
      vp?.removeEventListener('resize', reposition)
      vp?.removeEventListener('scroll', reposition)
    }
  }, [open, contentWidth, contentHeight])

  const toggle = useCallback(() => setOpen(prev => !prev), [])
  const close = useCallback(() => setOpen(false), [])

  return { open, pos, triggerRef, toggle, close }
}
