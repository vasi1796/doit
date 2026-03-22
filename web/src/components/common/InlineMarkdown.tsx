/**
 * Lightweight inline markdown renderer for task titles.
 * Supports: **bold**, *italic*, ~~strikethrough~~, `inline code`.
 * No block-level elements — titles are single-line.
 */

interface InlineMarkdownProps {
  text: string
  className?: string
}

interface Segment {
  text: string
  bold?: boolean
  italic?: boolean
  strike?: boolean
  code?: boolean
}

export function InlineMarkdown({ text, className }: InlineMarkdownProps) {
  const segments = parseInlineMarkdown(text)

  if (segments.length === 1 && !segments[0].bold && !segments[0].italic && !segments[0].strike && !segments[0].code) {
    return <span className={className}>{text}</span>
  }

  return (
    <span className={className}>
      {segments.map((seg, i) => {
        let el: React.ReactNode = seg.text
        if (seg.code) el = <code key={i} className="text-[0.9em] bg-[#f5f5f7] rounded px-1 font-mono">{seg.text}</code>
        else {
          if (seg.bold) el = <strong key={`b${i}`}>{el}</strong>
          if (seg.italic) el = <em key={`i${i}`}>{el}</em>
          if (seg.strike) el = <s key={`s${i}`}>{el}</s>
        }
        return <span key={i}>{el}</span>
      })}
    </span>
  )
}

function parseInlineMarkdown(text: string): Segment[] {
  const segments: Segment[] = []
  // Match: `code`, **bold**, *italic*, ~~strike~~
  const regex = /(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*|~~[^~]+~~)/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ text: text.slice(lastIndex, match.index) })
    }

    const raw = match[0]
    if (raw.startsWith('`')) {
      segments.push({ text: raw.slice(1, -1), code: true })
    } else if (raw.startsWith('**')) {
      segments.push({ text: raw.slice(2, -2), bold: true })
    } else if (raw.startsWith('~~')) {
      segments.push({ text: raw.slice(2, -2), strike: true })
    } else if (raw.startsWith('*')) {
      segments.push({ text: raw.slice(1, -1), italic: true })
    }

    lastIndex = match.index + raw.length
  }

  if (lastIndex < text.length) {
    segments.push({ text: text.slice(lastIndex) })
  }

  if (segments.length === 0) {
    segments.push({ text })
  }

  return segments
}
