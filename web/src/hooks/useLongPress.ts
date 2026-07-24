import { useCallback, useRef } from 'react'

const LONG_PRESS_MS = 500
const MOVE_TOLERANCE_PX = 10

/**
 * Long-press detection for touch pointers only — mouse users get
 * right-click and hover affordances instead. The press is cancelled when
 * the pointer moves beyond a small tolerance so scrolls and horizontal
 * swipes (e.g. swipe-to-delete) never trigger it.
 */
export function useLongPress(onLongPress: (point: { x: number; y: number }) => void) {
  const timer = useRef<number | null>(null)
  const origin = useRef<{ x: number; y: number } | null>(null)
  const fired = useRef(false)

  const clear = useCallback(() => {
    if (timer.current !== null) {
      window.clearTimeout(timer.current)
      timer.current = null
    }
    origin.current = null
  }, [])

  const onPointerDown = useCallback((e: React.PointerEvent) => {
    if (e.pointerType !== 'touch') return
    const point = { x: e.clientX, y: e.clientY }
    fired.current = false
    origin.current = point
    timer.current = window.setTimeout(() => {
      timer.current = null
      fired.current = true
      onLongPress(point)
    }, LONG_PRESS_MS)
  }, [onLongPress])

  const onPointerMove = useCallback((e: React.PointerEvent) => {
    if (!origin.current || timer.current === null) return
    const dx = e.clientX - origin.current.x
    const dy = e.clientY - origin.current.y
    if (Math.hypot(dx, dy) > MOVE_TOLERANCE_PX) {
      clear()
    }
  }, [clear])

  const onClickCapture = useCallback((e: React.MouseEvent) => {
    // Swallow the synthetic click that follows a completed long-press so
    // the row underneath doesn't also navigate.
    if (fired.current) {
      e.preventDefault()
      e.stopPropagation()
      fired.current = false
    }
  }, [])

  return {
    onPointerDown,
    onPointerMove,
    onPointerUp: clear,
    onPointerCancel: clear,
    onClickCapture,
  }
}
