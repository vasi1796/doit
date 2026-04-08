import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'

interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ open, title, message, confirmLabel = 'Delete', onConfirm, onCancel }: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null)
  const confirmRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (open) cancelRef.current?.focus()
  }, [open])

  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onCancel()
        return
      }
      if (e.key === 'Tab') {
        const buttons = [cancelRef.current, confirmRef.current].filter(Boolean) as HTMLElement[]
        const active = document.activeElement
        const idx = buttons.indexOf(active as HTMLElement)
        if (e.shiftKey) {
          e.preventDefault()
          buttons[idx <= 0 ? buttons.length - 1 : idx - 1].focus()
        } else {
          e.preventDefault()
          buttons[idx >= buttons.length - 1 ? 0 : idx + 1].focus()
        }
      }
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open, onCancel])

  if (!open) return null

  return createPortal(
    <div role="dialog" aria-modal="true" aria-labelledby="confirm-dialog-title" className="fixed inset-0 z-[200] flex items-center justify-center">
      <div className="absolute inset-0 bg-[rgba(0,0,0,0.35)]" onClick={onCancel} role="presentation" />
      <div className="relative bg-bg-elevated rounded-[20px] shadow-modal border border-separator w-[280px] overflow-hidden text-center">
        <div className="px-5 pt-5 pb-4">
          <h3 id="confirm-dialog-title" className="text-[17px] font-semibold text-text-primary">{title}</h3>
          <p className="text-[13px] text-text-secondary mt-1 leading-snug">{message}</p>
        </div>
        <div className="flex border-t border-separator">
          <button
            ref={cancelRef}
            type="button"
            onClick={onCancel}
            className="flex-1 py-3 text-[17px] text-accent font-normal border-r border-separator active:bg-bg-secondary min-h-[44px] transition-colors"
          >
            Cancel
          </button>
          <button
            ref={confirmRef}
            type="button"
            onClick={onConfirm}
            className="flex-1 py-3 text-[17px] text-danger font-semibold active:bg-bg-secondary min-h-[44px] transition-colors"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body
  )
}
