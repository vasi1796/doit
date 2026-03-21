import { useState } from 'react'
import * as operations from '../../db/operations'
import { useToast } from './Toast'
import { PRESET_COLORS } from '../../constants'

interface InlineLabelCreatorProps {
  onCreated: (labelId: string) => void
  onCancel: () => void
}

export function InlineLabelCreator({ onCreated, onCancel }: InlineLabelCreatorProps) {
  const { toast } = useToast()
  const [name, setName] = useState('')
  const [colour, setColour] = useState(PRESET_COLORS[0])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    try {
      const id = await operations.createLabel({ name: name.trim(), colour })
      toast('Label created', 'success')
      onCreated(id)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-center gap-1.5">
      <div className="flex gap-0.5">
        {PRESET_COLORS.slice(0, 4).map(c => (
          <button
            key={c}
            type="button"
            onClick={() => setColour(c)}
            className={`w-4 h-4 rounded-full ${colour === c ? 'ring-2 ring-offset-1 ring-accent/40' : ''}`}
            style={{ backgroundColor: c }}
          />
        ))}
      </div>
      <input
        type="text"
        value={name}
        onChange={e => setName(e.target.value)}
        placeholder="Name"
        className="text-[12px] outline-none border-b border-gray-200 py-0.5 w-20"
        // eslint-disable-next-line jsx-a11y/no-autofocus
        autoFocus
      />
      <button type="submit" className="text-[11px] text-accent font-medium">Add</button>
      <button type="button" onClick={onCancel} className="text-[11px] text-text-secondary">&#x2715;</button>
    </form>
  )
}
