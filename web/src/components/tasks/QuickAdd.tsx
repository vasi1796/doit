import { useState, useRef, forwardRef, useImperativeHandle } from 'react'
import { api } from '../../api/client'
import { useToast } from '../common/Toast'
import { PriorityPicker } from '../common/PriorityPicker'
import { DatePicker } from '../common/DatePicker'
import { TimePicker } from '../common/TimePicker'
import { RecurrencePicker } from '../common/RecurrencePicker'
import { ListSelect } from '../common/ListSelect'
import type { List, Label, Priority } from '../../api/types'
import { PRESET_COLORS } from '../../constants'

interface QuickAddProps {
  listId?: string
  lists?: List[]
  labels?: Label[]
  onCreated: () => void
  onListsChanged?: () => void
  onLabelsChanged?: () => void
}

export const QuickAdd = forwardRef<{ focus: () => void }, QuickAddProps>(function QuickAdd({ listId, lists, labels, onCreated, onListsChanged, onLabelsChanged }, ref) {
  const { toast } = useToast()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<Priority>(0)
  const [dueDate, setDueDate] = useState('')
  const [dueTime, setDueTime] = useState('')
  const [recurrence, setRecurrence] = useState('')
  const [selectedListId, setSelectedListId] = useState(listId || '')
  const [selectedLabelIds, setSelectedLabelIds] = useState<string[]>([])
  const [expanded, setExpanded] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [creatingLabel, setCreatingLabel] = useState(false)
  const [newLabelName, setNewLabelName] = useState('')
  const [newLabelColour, setNewLabelColour] = useState(PRESET_COLORS[0])
  const inputRef = useRef<HTMLInputElement>(null)
  const formRef = useRef<HTMLDivElement>(null)

  useImperativeHandle(ref, () => ({
    focus: () => {
      if (!expanded) {
        setExpanded(true)
        setTimeout(() => inputRef.current?.focus(), 80)
      } else {
        inputRef.current?.focus()
      }
    },
  }), [expanded])

  const resetForm = () => {
    setTitle('')
    setDescription('')
    setPriority(0)
    setDueDate('')
    setDueTime('')
    setRecurrence('')
    if (!listId) setSelectedListId('')
    setSelectedLabelIds([])
  }

  const handleSubmit = async () => {
    const trimmed = title.trim()
    if (!trimmed) {
      toast('Task title is required', 'error')
      inputRef.current?.focus()
      return
    }
    if (submitting) return

    setSubmitting(true)
    try {
      const result = await api.createTask({
        title: trimmed,
        description: description.trim() || undefined,
        priority,
        due_date: dueDate || undefined,
        due_time: dueTime || undefined,
        list_id: selectedListId || undefined,
        position: Date.now().toString(),
      })

      if (recurrence) await api.updateTask(result.id, { recurrence_rule: recurrence })
      for (const labelId of selectedLabelIds) await api.addLabel(result.id, labelId)

      resetForm()
      toast('Task created', 'success')
      onCreated()
      setTimeout(() => inputRef.current?.focus(), 50)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to create task', 'error')
    } finally {
      setSubmitting(false)
    }
  }

  const toggleLabel = (id: string) => {
    setSelectedLabelIds((prev) =>
      prev.includes(id) ? prev.filter((l) => l !== id) : [...prev, id]
    )
  }

  if (!expanded) {
    return (
      <button
        type="button"
        onClick={() => { setExpanded(true); setTimeout(() => inputRef.current?.focus(), 50) }}
        className="mx-4 my-3 flex items-center gap-3 px-4 py-3 rounded-xl bg-[#f8f8fa] hover:bg-[#f0f0f2] transition-colors text-[15px] text-text-secondary w-[calc(100%-32px)]"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#007aff" strokeWidth="2" strokeLinecap="round">
          <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        New task...
      </button>
    )
  }

  return (
    <div
      ref={formRef}
      className="mx-4 my-3 bg-white rounded-xl shadow-[0_2px_12px_rgba(0,0,0,0.1)] border border-gray-200"
    >
      <div className="flex items-center justify-between px-4 py-2 border-b border-gray-100">
        <button
          type="button"
          onClick={() => { resetForm(); setExpanded(false) }}
          className="text-sm text-text-secondary hover:text-text-primary min-h-[36px] px-1"
        >
          Cancel
        </button>
        <span className="text-sm font-semibold text-text-primary">New Task</span>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={!title.trim() || submitting}
          className="text-sm font-semibold text-accent min-h-[36px] px-1 disabled:opacity-30"
        >
          {submitting ? 'Adding...' : 'Add'}
        </button>
      </div>

      <div className="px-4 pt-3 pb-1">
        <input
          ref={inputRef}
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSubmit() } }}
          placeholder="Task name"
          aria-label="Task name"
          className="w-full text-[17px] font-medium outline-none placeholder:text-text-tertiary"
        />
      </div>

      <div className="px-4 pb-3">
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Notes"
          aria-label="Task notes"
          rows={2}
          className="w-full text-[16px] outline-none placeholder:text-text-tertiary resize-none text-text-note"
        />
      </div>

      <div className="border-t border-gray-100">
        <div className="flex items-center px-1 border-b border-gray-50">
          <DatePicker value={dueDate} onChange={setDueDate} onClear={() => setDueDate('')} />
          <TimePicker value={dueTime} onChange={setDueTime} onClear={() => setDueTime('')} />
        </div>

        <div className="flex items-center px-1 border-b border-gray-50">
          <RecurrencePicker value={recurrence} onChange={setRecurrence} />
          {lists && !listId && (
            <ListSelect
              value={selectedListId}
              lists={lists}
              onChange={setSelectedListId}
              onListCreated={onListsChanged || (() => {})}
            />
          )}
        </div>

        <div className="px-4 py-2.5 border-b border-gray-50">
          <p className="text-[11px] text-text-secondary font-medium uppercase tracking-wider mb-1.5">Priority</p>
          <PriorityPicker value={priority} onChange={setPriority} compact />
        </div>

        {labels && (
          <div className="px-4 py-2.5 flex flex-wrap gap-1.5 items-center">
            {labels.map((label) => {
              const selected = selectedLabelIds.includes(label.id)
              return (
                <button
                  key={label.id}
                  type="button"
                  onClick={() => toggleLabel(label.id)}
                  className={`text-[12px] px-2.5 py-1 rounded-full font-medium transition-all ${
                    selected ? 'ring-1 ring-offset-1' : 'opacity-40 hover:opacity-70'
                  }`}
                  style={{
                    backgroundColor: (label.colour || '#86868b') + (selected ? '25' : '12'),
                    color: label.colour || '#86868b',
                  }}
                >
                  {selected && '✓ '}{label.name}
                </button>
              )
            })}
            {creatingLabel ? (
              <form
                onSubmit={async (e) => {
                  e.preventDefault()
                  if (!newLabelName.trim()) return
                  try {
                    const result = await api.createLabel({ name: newLabelName.trim(), colour: newLabelColour })
                    setSelectedLabelIds(prev => [...prev, result.id])
                    setNewLabelName('')
                    setCreatingLabel(false)
                    onLabelsChanged?.()
                    toast('Label created', 'success')
                  } catch (err) {
                    toast(err instanceof Error ? err.message : 'Failed', 'error')
                  }
                }}
                className="flex items-center gap-1.5"
              >
                <div className="flex gap-0.5">
                  {PRESET_COLORS.slice(0, 4).map(c => (
                    <button
                      key={c}
                      type="button"
                      onClick={() => setNewLabelColour(c)}
                      className={`w-4 h-4 rounded-full ${newLabelColour === c ? 'ring-2 ring-offset-1 ring-accent/40' : ''}`}
                      style={{ backgroundColor: c }}
                    />
                  ))}
                </div>
                <input
                  type="text"
                  value={newLabelName}
                  onChange={e => setNewLabelName(e.target.value)}
                  placeholder="Name"
                  className="text-[12px] outline-none border-b border-gray-200 py-0.5 w-20"
                  autoFocus
                />
                <button type="submit" className="text-[11px] text-accent font-medium">Add</button>
                <button type="button" onClick={() => setCreatingLabel(false)} className="text-[11px] text-text-secondary">✕</button>
              </form>
            ) : (
              <button
                type="button"
                onClick={() => setCreatingLabel(true)}
                className="text-[12px] px-2.5 py-1 rounded-full font-medium text-accent bg-accent/8 hover:bg-accent/15 transition-all"
              >
                + New
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
})
