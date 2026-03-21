/** Palette colors for lists, labels, and user-facing swatches */
export const COLORS = {
  blue: '#007aff',
  red: '#ff3b30',
  orange: '#ff9500',
  green: '#4cd964',
  purple: '#5856d6',
  pink: '#af52de',
  gray: '#86868b',
} as const

export const PRESET_COLORS = [
  COLORS.blue, COLORS.red, COLORS.orange, COLORS.green,
  COLORS.purple, COLORS.pink, COLORS.gray,
]

/** Semantic UI colors used across the app (Apple HIG inspired) */
export const UI = {
  accent: '#007aff',
  danger: '#ff3b30',
  textPrimary: '#1d1d1f',
  textSecondary: '#86868b',
  textTertiary: '#c7c7cc',
  textNote: '#3c3c43',
} as const
