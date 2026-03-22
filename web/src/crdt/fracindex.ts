// Printable ASCII range for position strings.
// Supports any mix of digits, letters, and symbols.
const MIN_CHAR = '!'.charCodeAt(0) // 33
const MAX_CHAR = '~'.charCodeAt(0) // 126
const MID_CHAR = 'O'.charCodeAt(0) // 79 — middle of printable range

export function first(): string {
  return String.fromCharCode(MID_CHAR)
}

export function last(): string {
  return String.fromCharCode(MAX_CHAR)
}

/**
 * Generate a position string that sorts lexicographically between `before` and `after`.
 * If before is empty, generates a position before after.
 * If after is empty, generates a position after before.
 * Works with any printable ASCII strings (digits, letters, mixed).
 */
export function between(before: string, after: string): string {
  if (!before) before = String.fromCharCode(MIN_CHAR)
  if (!after) after = String.fromCharCode(MAX_CHAR)

  // Ensure before < after lexicographically
  if (before >= after) {
    return before + String.fromCharCode(MID_CHAR)
  }

  const maxLen = Math.max(before.length, after.length)
  const bPadded = padRight(before, maxLen)
  const aPadded = padRight(after, maxLen)

  for (let i = 0; i < maxLen; i++) {
    const bChar = bPadded.charCodeAt(i)
    const aChar = aPadded.charCodeAt(i)

    if (bChar < aChar) {
      const mid = bChar + Math.floor((aChar - bChar) / 2)
      if (mid > bChar) {
        return before.slice(0, Math.min(i, before.length)) + String.fromCharCode(mid)
      }
    }
  }

  // No room between — append a middle character after `before`
  return before + String.fromCharCode(MID_CHAR)
}

function padRight(s: string, length: number): string {
  while (s.length < length) s += String.fromCharCode(MIN_CHAR)
  return s
}
