import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { ColorSwatchRow } from './ColorSwatchRow'

interface EditNameColourDialogProps {
  open: boolean
  title: string
  initialName: string
  initialColour: string
  onSave: (changes: { name?: string; colour?: string }) => void
  onCancel: () => void
}

/**
 * Small modal for renaming/recolouring a list or label, mirroring the
 * sidebar create-list form. Only changed fields are passed to onSave so
 * untouched fields keep their per-field HLC timestamps.
 *
 * Mount with a key derived from the entity being edited — state is
 * initialised from props once per mount rather than synced in effects.
 */
export function EditNameColourDialog({
  open,
  title,
  initialName,
  initialColour,
  onSave,
  onCancel,
}: EditNameColourDialogProps) {
  const [name, setName] = useState(initialName)
  const [colour, setColour] = useState(initialColour)

  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open, onCancel])

  if (!open) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    const changes: { name?: string; colour?: string } = {}
    if (trimmed !== initialName) changes.name = trimmed
    if (colour !== initialColour) changes.colour = colour
    onSave(changes)
  }

  return createPortal(
    <div role="dialog" aria-modal="true" aria-label={title} className="fixed inset-0 z-[200] flex items-center justify-center px-6">
      <div className="absolute inset-0 bg-[rgba(0,0,0,0.35)]" onClick={onCancel} role="presentation" />
      <form
        onSubmit={handleSubmit}
        className="relative bg-bg-elevated rounded-[20px] shadow-modal border border-separator w-full max-w-[320px] overflow-hidden"
      >
        <div className="px-4 pt-4 pb-1">
          <p className="text-[13px] font-semibold text-text-secondary mb-2">{title}</p>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            aria-label="Name"
            className="w-full text-[16px] font-medium outline-none bg-transparent text-text-primary placeholder:text-text-tertiary min-h-[44px]"
            // eslint-disable-next-line jsx-a11y/no-autofocus
            autoFocus
          />
        </div>
        <div className="px-4 pb-3">
          <ColorSwatchRow value={colour} onChange={setColour} />
        </div>
        <div className="flex border-t border-separator">
          <button
            type="button"
            onClick={onCancel}
            className="flex-1 text-sm text-text-secondary font-medium py-2.5 hover:bg-bg-secondary transition-colors min-h-[44px]"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={!name.trim()}
            className="flex-1 text-sm text-accent font-semibold py-2.5 hover:bg-accent-light transition-colors border-l border-separator min-h-[44px] disabled:opacity-30"
          >
            Save
          </button>
        </div>
      </form>
    </div>,
    document.body
  )
}
