import { useState } from 'react'
import { usePopover } from '../../hooks/usePopover'
import * as operations from '../../db/operations'
import { useToast } from './Toast'
import { ColorSwatchRow } from './ColorSwatchRow'
import type { List } from '../../api/types'
import { PRESET_COLORS } from '../../constants'

interface ListSelectProps {
  value: string
  lists: List[]
  onChange: (listId: string) => void
}

export function ListSelect({ value, lists, onChange }: ListSelectProps) {
  const { toast } = useToast()
  const { open, pos, triggerRef, toggle, close } = usePopover({ contentWidth: 200 })
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColour, setNewColour] = useState<string>(PRESET_COLORS[0])

  const currentList = lists.find((l) => l.id === value)
  const displayName = currentList?.name || 'Inbox'
  const displayColor = currentList?.colour || 'var(--color-gray)'

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newName.trim()) return
    try {
      const id = await operations.createList({
        name: newName.trim(),
        colour: newColour,
        position: Date.now().toString(),
      })
      setNewName('')
      setCreating(false)
      close()
      toast('List created', 'success')
      onChange(id)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={toggle}
        className="flex items-center gap-2 min-h-[40px] px-3 rounded-[10px] hover:bg-bg-secondary transition-colors text-sm text-text-primary"
      >
        <span className="w-3 h-3 rounded-full shrink-0" style={{ backgroundColor: displayColor }} />
        <span>{displayName}</span>
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-[60]" onClick={() => { close(); setCreating(false) }} aria-hidden="true" />
          <div
            className="fixed bg-bg-elevated rounded-[14px] shadow-popover border border-separator py-1 z-[61] min-w-[220px] max-h-[50vh] overflow-y-auto"
            style={{ top: pos.top, left: pos.left }}
          >
            <button
              type="button"
              onClick={() => { onChange(''); close() }}
              className={`w-full flex items-center gap-2.5 text-left px-4 py-2.5 text-[15px] hover:bg-bg-secondary transition-colors ${
                !value ? 'text-accent font-medium' : 'text-text-primary'
              }`}
            >
              <span className="w-3 h-3 rounded-full" style={{ backgroundColor: 'var(--color-gray)' }} />
              Inbox
            </button>
            {lists.map((l) => (
              <button
                key={l.id}
                type="button"
                onClick={() => { onChange(l.id); close() }}
                className={`w-full flex items-center gap-2.5 text-left px-4 py-2.5 text-[15px] hover:bg-bg-secondary transition-colors ${
                  value === l.id ? 'text-accent font-medium' : 'text-text-primary'
                }`}
              >
                <span className="w-3 h-3 rounded-full" style={{ backgroundColor: l.colour || 'var(--color-gray)' }} />
                {l.name}
              </button>
            ))}
            <div className="border-t border-separator mt-1 pt-1">
              {creating ? (
                <form onSubmit={handleCreate} className="px-4 py-2 space-y-2">
                  <input
                    type="text"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder="List name"
                    className="w-full text-[16px] outline-none border-b border-separator pb-1 bg-transparent text-text-primary placeholder:text-text-tertiary"
                    // eslint-disable-next-line jsx-a11y/no-autofocus
                    autoFocus
                  />
                  <div className="flex items-center gap-1">
                    <ColorSwatchRow
                      value={newColour}
                      onChange={setNewColour}
                      colors={PRESET_COLORS.slice(0, 5)}
                    />
                    <button type="submit" className="ml-auto text-[13px] text-accent font-semibold">Create</button>
                  </div>
                </form>
              ) : (
                <button
                  type="button"
                  onClick={() => setCreating(true)}
                  className="w-full text-left px-4 py-2.5 text-[15px] text-accent hover:bg-accent-light transition-colors"
                >
                  + New list
                </button>
              )}
            </div>
          </div>
        </>
      )}
    </>
  )
}
