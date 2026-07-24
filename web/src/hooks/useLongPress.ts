import { useCallback, useRef } from 'react'

const LONG_PRESS_MS = 500
const MOVE_TOLERANCE_PX = 10
const CLICK_SUPPRESS_WINDOW_MS = 3000

/**
 * The synthetic click Safari fires on finger lift after a long-press lands
 * wherever the finger is released — usually the menu backdrop that has just
 * rendered under it, not the pressed row. Swallow exactly one click,
 * document-wide in the capture phase, so neither the row navigates nor the
 * freshly opened menu dismisses. A new pointerdown disarms the suppressor:
 * the next gesture's click must go through even if the synthetic click
 * never arrived.
 */
function suppressNextClick() {
  const suppress = (e: MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    cleanup()
  }
  const disarm = () => cleanup()
  const cleanup = () => {
    document.removeEventListener('click', suppress, true)
    document.removeEventListener('pointerdown', disarm, true)
    window.clearTimeout(timeout)
  }
  document.addEventListener('click', suppress, true)
  document.addEventListener('pointerdown', disarm, true)
  const timeout = window.setTimeout(cleanup, CLICK_SUPPRESS_WINDOW_MS)
}

/**
 * Long-press detection for touch pointers only — mouse users get
 * right-click and hover affordances instead. The press is cancelled when
 * the pointer moves beyond a small tolerance so scrolls and horizontal
 * swipes (e.g. swipe-to-delete) never trigger it.
 */
export function useLongPress(onLongPress: (point: { x: number; y: number }) => void) {
  const timer = useRef<number | null>(null)
  const origin = useRef<{ x: number; y: number } | null>(null)

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
    origin.current = point
    timer.current = window.setTimeout(() => {
      timer.current = null
      suppressNextClick()
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

  return {
    onPointerDown,
    onPointerMove,
    onPointerUp: clear,
    onPointerCancel: clear,
  }
}
