import { useState } from 'react'
import { usePopover } from '../../hooks/usePopover'
import { api } from '../../api/client'
import { useToast } from './Toast'
import type { List } from '../../api/types'
import { PRESET_COLORS } from '../../constants'

interface ListSelectProps {
  value: string
  lists: List[]
  onChange: (listId: string) => void
  onListCreated: () => void
}

export function ListSelect({ value, lists, onChange, onListCreated }: ListSelectProps) {
  const { toast } = useToast()
  const { open, pos, triggerRef, toggle, close } = usePopover({ contentWidth: 200 })
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(PRESET_COLORS[0])

  const currentList = lists.find((l) => l.id === value)
  const displayName = currentList?.name || 'Inbox'
  const displayColor = currentList?.colour || '#86868b'

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newName.trim()) return
    try {
      const result = await api.createList({
        name: newName.trim(),
        colour: newColor,
        position: Date.now().toString(),
      })
      setNewName('')
      setCreating(false)
      close()
      toast('List created', 'success')
      onListCreated()
      onChange(result.id)
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
        className="flex items-center gap-2 min-h-[40px] px-3 rounded-lg hover:bg-gray-50 transition-colors text-sm"
      >
        <span className="w-3 h-3 rounded-full shrink-0" style={{ backgroundColor: displayColor }} />
        <span className="text-[#1d1d1f]">{displayName}</span>
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-[60]" onClick={() => { close(); setCreating(false) }} />
          <div
            className="fixed bg-white rounded-xl shadow-xl border border-gray-200 py-1 z-[61] min-w-[200px] max-h-[50vh] overflow-y-auto"
            style={{ top: pos.top, left: pos.left }}
          >
            <button
              type="button"
              onClick={() => { onChange(''); close() }}
              className={`w-full flex items-center gap-2.5 text-left px-4 py-2.5 text-[15px] hover:bg-gray-50 ${
                !value ? 'text-[#007aff] font-medium' : 'text-[#1d1d1f]'
              }`}
            >
              <span className="w-3 h-3 rounded-full bg-[#86868b]" />
              Inbox
            </button>
            {lists.map((l) => (
              <button
                key={l.id}
                type="button"
                onClick={() => { onChange(l.id); close() }}
                className={`w-full flex items-center gap-2.5 text-left px-4 py-2.5 text-[15px] hover:bg-gray-50 ${
                  value === l.id ? 'text-[#007aff] font-medium' : 'text-[#1d1d1f]'
                }`}
              >
                <span className="w-3 h-3 rounded-full" style={{ backgroundColor: l.colour || '#86868b' }} />
                {l.name}
              </button>
            ))}
            <div className="border-t border-gray-100 mt-1 pt-1">
              {creating ? (
                <form onSubmit={handleCreate} className="px-4 py-2 space-y-2">
                  <input
                    type="text"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder="List name"
                    className="w-full text-[15px] outline-none border-b border-gray-200 pb-1"
                    autoFocus
                  />
                  <div className="flex items-center gap-1">
                    {PRESET_COLORS.slice(0, 5).map((c) => (
                      <button
                        key={c}
                        type="button"
                        onClick={() => setNewColor(c)}
                        className={`w-5 h-5 rounded-full ${newColor === c ? 'ring-2 ring-offset-1 ring-[#007aff]/40' : ''}`}
                        style={{ backgroundColor: c }}
                      />
                    ))}
                    <button type="submit" className="ml-auto text-[13px] text-[#007aff] font-medium">Create</button>
                  </div>
                </form>
              ) : (
                <button
                  type="button"
                  onClick={() => setCreating(true)}
                  className="w-full text-left px-4 py-2.5 text-[15px] text-[#007aff] hover:bg-gray-50"
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
