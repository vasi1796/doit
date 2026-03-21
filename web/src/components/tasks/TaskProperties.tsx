import { PriorityPicker } from '../common/PriorityPicker'
import { DatePicker } from '../common/DatePicker'
import { TimePicker } from '../common/TimePicker'
import { RecurrencePicker } from '../common/RecurrencePicker'
import { ListSelect } from '../common/ListSelect'
import type { Task, List } from '../../api/types'

interface TaskPropertiesProps {
  task: Task
  lists?: List[]
  onSave: (field: string, value: unknown) => void
  onListsChanged?: () => void
}

export function TaskProperties({ task, lists, onSave, onListsChanged }: TaskPropertiesProps) {
  return (
    <div className="space-y-2 mb-4 bg-gray-50 rounded-lg p-3">
      <div className="flex items-center gap-3">
        <span className="text-xs text-text-secondary w-16 shrink-0">Priority</span>
        <PriorityPicker value={task.priority} onChange={(p) => onSave('priority', p)} />
      </div>

      <div className="flex items-center gap-3">
        <span className="text-xs text-text-secondary w-16 shrink-0">Due date</span>
        <DatePicker
          value={task.due_date || ''}
          onChange={(d) => onSave('due_date', d || '')}
          onClear={() => onSave('due_date', '')}
        />
      </div>

      <div className="flex items-center gap-3">
        <span className="text-xs text-text-secondary w-16 shrink-0">Time</span>
        <TimePicker
          value={task.due_time || ''}
          onChange={(t) => onSave('due_time', t || '')}
          onClear={() => onSave('due_time', '')}
        />
      </div>

      <div className="flex items-center gap-3">
        <span className="text-xs text-text-secondary w-16 shrink-0">Repeat</span>
        <RecurrencePicker
          value={task.recurrence_rule || ''}
          onChange={(r) => onSave('recurrence_rule', r)}
        />
      </div>

      {lists && (
        <div className="flex items-center gap-3">
          <span className="text-xs text-text-secondary w-16 shrink-0">List</span>
          <ListSelect
            value={task.list_id || ''}
            lists={lists}
            onChange={(id) => onSave('list_id', id)}
            onListCreated={onListsChanged || (() => {})}
          />
        </div>
      )}
    </div>
  )
}
