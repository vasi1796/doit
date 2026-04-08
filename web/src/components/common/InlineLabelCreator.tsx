import { useState } from 'react'
import * as operations from '../../db/operations'
import { useToast } from './Toast'
import { ColorSwatchRow } from './ColorSwatchRow'
import { PRESET_COLORS } from '../../constants'

interface InlineLabelCreatorProps {
  onCreated: (labelId: string) => void
  onCancel: () => void
}

export function InlineLabelCreator({ onCreated, onCancel }: InlineLabelCreatorProps) {
  const { toast } = useToast()
  const [name, setName] = useState('')
  const [colour, setColour] = useState<string>(PRESET_COLORS[0])

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
      <ColorSwatchRow
        value={colour}
        onChange={setColour}
        colors={PRESET_COLORS.slice(0, 4)}
        size={16}
        gap="tight"
      />
      <input
        type="text"
        value={name}
        onChange={e => setName(e.target.value)}
        placeholder="Name"
        aria-label="Label name"
        className="text-[16px] outline-none border-b border-separator py-0.5 w-20 bg-transparent text-text-primary placeholder:text-text-tertiary"
        // eslint-disable-next-line jsx-a11y/no-autofocus
        autoFocus
      />
      <button type="submit" className="text-[11px] text-accent font-semibold">Add</button>
      <button type="button" onClick={onCancel} className="text-[11px] text-text-tertiary hover:text-text-secondary transition-colors">&#x2715;</button>
    </form>
  )
}
