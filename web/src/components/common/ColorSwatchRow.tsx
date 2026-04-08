import { PRESET_COLORS } from '../../constants'

interface ColorSwatchRowProps {
  value: string
  onChange: (color: string) => void
  /** Subset of PRESET_COLORS to show. Defaults to the full preset list. */
  colors?: readonly string[]
  /** Swatch diameter in px. Defaults to 20. */
  size?: number
  /** Gap between swatches. Defaults to 4px. */
  gap?: 'tight' | 'normal'
}

/**
 * Horizontal row of preset color swatches with a ring on the selected one.
 * Used by ListSelect, LabelPicker, InlineLabelCreator, and the sidebar
 * create-list form.
 */
export function ColorSwatchRow({
  value,
  onChange,
  colors = PRESET_COLORS,
  size = 20,
  gap = 'normal',
}: ColorSwatchRowProps) {
  return (
    <div className={`flex items-center ${gap === 'tight' ? 'gap-0.5' : 'gap-1'}`}>
      {colors.map((c) => (
        <button
          key={c}
          type="button"
          onClick={() => onChange(c)}
          className={`rounded-full ${value === c ? 'ring-2 ring-offset-1 ring-accent/40' : ''}`}
          style={{ backgroundColor: c, width: size, height: size }}
          aria-label={`Color ${c}`}
        />
      ))}
    </div>
  )
}
