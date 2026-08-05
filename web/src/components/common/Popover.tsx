import { useEffect } from 'react'

interface PopoverProps {
  pos: { top: number; left: number }
  label: string
  onClose: () => void
  className?: string
  children: React.ReactNode
}

/** Shared popover shell for pickers: dismiss backdrop, positioned container,
 * Escape-to-close. Pair with usePopover for the trigger and positioning. */
export function Popover({ pos, label, onClose, className = '', children }: PopoverProps) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [onClose])

  return (
    <>
      <div className="fixed inset-0 z-[60]" onClick={onClose} aria-hidden="true" />
      <div
        role="dialog"
        aria-label={label}
        className={`fixed bg-bg-elevated rounded-[14px] shadow-popover border border-separator z-[61] ${className}`}
        style={{ top: pos.top, left: pos.left }}
      >
        {children}
      </div>
    </>
  )
}

interface PickerTriggerProps {
  triggerRef: React.RefObject<HTMLButtonElement | null>
  onClick: () => void
  active: boolean
  icon: React.ReactNode
  label: string
  onClear?: () => void
  clearLabel?: string
}

/** Trigger button with an optional clear control rendered as a sibling —
 * a button nested inside a button is invalid HTML with unreliable focus
 * behaviour in WebKit/VoiceOver. */
export function PickerTrigger({ triggerRef, onClick, active, icon, label, onClear, clearLabel }: PickerTriggerProps) {
  const showClear = active && onClear
  return (
    <div className={`flex items-center rounded-[10px] hover:bg-bg-secondary transition-colors text-sm ${active ? 'text-accent' : 'text-text-secondary'}`}>
      <button
        ref={triggerRef}
        type="button"
        onClick={onClick}
        className={`flex items-center gap-2 min-h-[40px] pl-3 ${showClear ? 'pr-1' : 'pr-3'}`}
      >
        {icon}
        <span className={active ? 'text-text-primary' : 'text-text-secondary'}>{label}</span>
      </button>
      {showClear && (
        <button
          type="button"
          onClick={onClear}
          className="min-h-[40px] min-w-[28px] pr-2 text-text-secondary hover:text-danger"
          aria-label={clearLabel}
        >
          ×
        </button>
      )}
    </div>
  )
}
