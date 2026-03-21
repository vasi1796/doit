const MIN_CHAR = 'a'.charCodeAt(0) // 97
const MAX_CHAR = 'z'.charCodeAt(0) // 122

export function first(): string {
  return 'a'
}

export function last(): string {
  return 'z'
}

/**
 * Generate a position string that sorts between `before` and `after`.
 * If before is empty, generates a position before after.
 * If after is empty, generates a position after before.
 */
export function between(before: string, after: string): string {
  if (!before) before = String.fromCharCode(MIN_CHAR)
  if (!after) after = String.fromCharCode(MAX_CHAR)

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
  return before + String.fromCharCode(MIN_CHAR + Math.floor((MAX_CHAR - MIN_CHAR) / 2))
}

function padRight(s: string, length: number): string {
  while (s.length < length) s += String.fromCharCode(MIN_CHAR)
  return s
}
