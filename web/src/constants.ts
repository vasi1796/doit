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

/** Priority → color mapping used in TaskItem, search results, calendar */
export const PRIORITY_COLORS: Partial<Record<0 | 1 | 2 | 3, string>> = {
  1: COLORS.green,
  2: COLORS.orange,
  3: COLORS.red,
}

/** Semantic UI colors used across the app (Apple HIG inspired) */
export const UI = {
  accent: '#007aff',
  danger: '#ff3b30',
  textPrimary: '#1d1d1f',
  textSecondary: '#86868b',
  textTertiary: '#c7c7cc',
  textNote: '#3c3c43',
  codeBg: '#f5f5f7',
} as const
