import { useState } from 'react'
import { LabelPicker } from '../common/LabelPicker'
import type { Label } from '../../api/types'

interface LabelsSectionProps {
  taskId: string
  taskLabels: Label[]
  allLabels: Label[]
  onChanged: () => void
  onLabelsChanged: () => void
}

export function LabelsSection({ taskId, taskLabels, allLabels, onChanged, onLabelsChanged }: LabelsSectionProps) {
  const [showPicker, setShowPicker] = useState(false)
  const attachedIds = new Set(taskLabels.map((l) => l.id))

  return (
    <div className="mb-4">
      <button
        type="button"
        onClick={() => setShowPicker(!showPicker)}
        className="flex items-center justify-between mb-2 w-full text-left"
      >
        <h3 className="text-xs font-medium text-[#86868b] uppercase tracking-wide">Labels</h3>
        <span className="text-xs text-[#007aff]">
          {showPicker ? 'Done' : 'Edit'}
        </span>
      </button>

      {showPicker ? (
        <LabelPicker
          allLabels={allLabels}
          attachedIds={attachedIds}
          taskId={taskId}
          onChanged={onChanged}
          onLabelsChanged={onLabelsChanged}
        />
      ) : (
        <div
          className="flex flex-wrap gap-1 cursor-pointer"
          onClick={() => setShowPicker(true)}
        >
          {taskLabels.length === 0 && (
            <span className="text-xs text-[#86868b]">Tap to add labels</span>
          )}
          {taskLabels.map((label) => (
            <span
              key={label.id}
              className="px-2.5 py-1 text-xs rounded-full font-medium"
              style={{
                backgroundColor: (label.colour || '#86868b') + '20',
                color: label.colour || '#86868b',
              }}
            >
              {label.name}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
