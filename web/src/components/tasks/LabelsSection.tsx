import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { LabelPicker } from '../common/LabelPicker'
import { COLORS } from '../../constants'
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
        <h3 className="text-[11px] font-semibold text-text-tertiary uppercase tracking-wider">Labels</h3>
        <span className="text-[12px] text-accent font-semibold">
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
          className="flex flex-wrap gap-1.5 cursor-pointer"
          onClick={() => setShowPicker(true)}
          onKeyDown={(e) => { if (e.key === 'Enter') setShowPicker(true) }}
          role="button"
          tabIndex={0}
          aria-label="Edit labels"
        >
          {taskLabels.length === 0 && (
            <span className="text-[12px] text-text-tertiary">Tap to add labels</span>
          )}
          <AnimatePresence initial={false}>
            {taskLabels.map((label) => (
              <motion.span
                key={label.id}
                className="px-2.5 py-1 text-[11px] rounded-full font-medium"
                style={{
                  backgroundColor: (label.colour || COLORS.gray) + '24',
                  color: label.colour || COLORS.gray,
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
