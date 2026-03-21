import { useState } from 'react'
import * as operations from '../../db/operations'
import { useToast } from './Toast'
import type { Label } from '../../api/types'
import { PRESET_COLORS } from '../../constants'

interface LabelPickerProps {
  allLabels: Label[]
  attachedIds: Set<string>
  taskId: string
}

export function LabelPicker({ allLabels, attachedIds, taskId }: LabelPickerProps) {
  const { toast } = useToast()
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColour, setNewColour] = useState(PRESET_COLORS[0])

  const handleToggle = async (label: Label) => {
    try {
      if (attachedIds.has(label.id)) {
        await operations.removeLabel(taskId, label.id)
      } else {
        await operations.addLabel(taskId, label.id)
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newName.trim()) return
    try {
      await operations.createLabel({ name: newName.trim(), colour: newColour })
      setNewName('')
      setCreating(false)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  return (
    <div className="space-y-1">
      {allLabels.map((label) => (
        <button
          key={label.id}
          type="button"
          onClick={() => handleToggle(label)}
          className="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-gray-50 transition-colors min-h-[36px]"
        >
          <span
            className="w-3 h-3 rounded-full shrink-0"
            style={{ backgroundColor: label.colour || '#86868b' }}
          />
          <span className="text-sm flex-1 text-left">{label.name}</span>
          {attachedIds.has(label.id) && (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#007aff" strokeWidth="2.5" strokeLinecap="round">
              <path d="m5 12 5 5L20 7" />
            </svg>
          )}
        </button>
      ))}

      {creating ? (
        <form onSubmit={handleCreate} className="flex items-center gap-2 px-2 py-1.5">
          <div className="flex gap-1">
            {PRESET_COLORS.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setNewColour(c)}
                className={`w-4 h-4 rounded-full ${newColour === c ? 'ring-2 ring-offset-1 ring-accent/30' : ''}`}
                style={{ backgroundColor: c }}
              />
            ))}
          </div>
          <input
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Label name"
            className="flex-1 text-sm outline-none border-b border-gray-200 py-1"
            // eslint-disable-next-line jsx-a11y/no-autofocus
            autoFocus
          />
          <button type="submit" className="text-accent text-sm font-medium">Add</button>
        </form>
      ) : (
        <button
          type="button"
          onClick={() => setCreating(true)}
          className="w-full text-left px-2 py-1.5 text-sm text-accent hover:bg-gray-50 rounded-lg min-h-[36px]"
        >
          + Create label
        </button>
      )}
    </div>
  )
}
