/** Palette colors for lists, labels, and user-facing swatches (Apple HIG evolved) */
export const COLORS = {
  blue: '#3478F6',
  red: '#FF453A',
  orange: '#FF9F0A',
  yellow: '#FFD60A',
  green: '#30D158',
  teal: '#64D2FF',
  purple: '#BF5AF2',
  pink: '#FF375F',
  indigo: '#5E5CE6',
  brown: '#AC8E68',
  gray: '#8E8E93',
} as const

export const PRESET_COLORS = [
  COLORS.blue, COLORS.red, COLORS.orange, COLORS.green,
  COLORS.purple, COLORS.pink, COLORS.gray,
]

/** Priority → color mapping used in TaskItem, search results, calendar */
export const PRIORITY_COLORS: Partial<Record<0 | 1 | 2 | 3, string>> = {
  1: COLORS.green,   // low
  2: COLORS.orange,  // medium
  3: COLORS.red,     // high
}

/** Semantic UI colors used across the app (Apple HIG inspired) */
export const UI = {
  accent: '#3478F6',
  accentHover: '#2563EB',
  danger: '#FF453A',
  success: '#30D158',
  warning: '#FF9F0A',
  textPrimary: '#1C1C1E',
  textSecondary: '#636366',
  textTertiary: '#AEAEB2',
  textQuaternary: '#C7C7CC',
  textNote: '#3C3C43', // legacy — kept until MarkdownEditor is migrated
  bgPrimary: '#FFFFFF',
  bgSecondary: '#F2F2F7',
  bgTertiary: '#E5E5EA',
  codeBg: '#F2F2F7',
} as const
