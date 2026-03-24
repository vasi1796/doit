import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { LabelPicker } from '../common/LabelPicker'
import type { Label } from '../../api/types'

interface LabelsSectionProps {
  taskId: string
  taskLabels: Label[]
  allLabels: Label[]
}

export function LabelsSection({ taskId, taskLabels, allLabels }: LabelsSectionProps) {
  const [showPicker, setShowPicker] = useState(false)
  const attachedIds = new Set(taskLabels.map((l) => l.id))

  return (
    <div className="mb-4">
      <button
        type="button"
        onClick={() => setShowPicker(!showPicker)}
        className="flex items-center justify-between mb-2 w-full text-left"
      >
        <h3 className="text-xs font-medium text-text-secondary uppercase tracking-wide">Labels</h3>
        <span className="text-xs text-accent">
          {showPicker ? 'Done' : 'Edit'}
        </span>
      </button>

      {showPicker ? (
        <LabelPicker
          allLabels={allLabels}
          attachedIds={attachedIds}
          taskId={taskId}
        />
      ) : (
        <div
          className="flex flex-wrap gap-1 cursor-pointer"
          onClick={() => setShowPicker(true)}
          onKeyDown={(e) => { if (e.key === 'Enter') setShowPicker(true) }}
          role="button"
          tabIndex={0}
        >
          {taskLabels.length === 0 && (
            <span className="text-xs text-text-secondary">Tap to add labels</span>
          )}
          <AnimatePresence initial={false}>
            {taskLabels.map((label) => (
              <motion.span
                key={label.id}
                className="px-2.5 py-1 text-xs rounded-full font-medium"
                style={{
                  backgroundColor: (label.colour || '#86868b') + '20',
                  color: label.colour || '#86868b',
                }}
                initial={{ opacity: 0, scale: 0 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0 }}
                transition={{ duration: 0.2, ease: 'easeOut' }}
              >
                {label.name}
              </motion.span>
            ))}
          </AnimatePresence>
        </div>
      )}
    </div>
  )
}
