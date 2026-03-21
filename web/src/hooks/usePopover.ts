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
    if (open && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect()
      const left = Math.min(rect.left, window.innerWidth - contentWidth)
      const top = rect.bottom + 4
      setPos({
        top: Math.min(top, window.innerHeight - contentHeight),
        left: Math.max(8, left),
      })
    }
  }, [open, contentWidth, contentHeight])

  const toggle = useCallback(() => setOpen(prev => !prev), [])
  const close = useCallback(() => setOpen(false), [])

  return { open, pos, triggerRef, toggle, close }
}
