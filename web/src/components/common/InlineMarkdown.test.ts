import { describe, it, expect } from 'vitest'
import { parseInlineMarkdown } from './InlineMarkdown'

describe('parseInlineMarkdown', () => {
  it('returns plain text for no markdown', () => {
    expect(parseInlineMarkdown('hello world')).toEqual([
      { text: 'hello world' },
    ])
  })

  it('parses **bold**', () => {
    expect(parseInlineMarkdown('a **bold** word')).toEqual([
      { text: 'a ' },
      { text: 'bold', bold: true },
      { text: ' word' },
    ])
  })

  it('parses *italic*', () => {
    expect(parseInlineMarkdown('an *italic* word')).toEqual([
      { text: 'an ' },
      { text: 'italic', italic: true },
      { text: ' word' },
    ])
  })

  it('parses ~~strikethrough~~', () => {
    expect(parseInlineMarkdown('a ~~deleted~~ word')).toEqual([
      { text: 'a ' },
      { text: 'deleted', strike: true },
      { text: ' word' },
    ])
  })

  it('parses `inline code`', () => {
    expect(parseInlineMarkdown('use `npm install`')).toEqual([
      { text: 'use ' },
      { text: 'npm install', code: true },
    ])
  })

  it('parses multiple markers in one string', () => {
    const result = parseInlineMarkdown('**bold** and *italic*')
    expect(result).toEqual([
      { text: 'bold', bold: true },
      { text: ' and ' },
      { text: 'italic', italic: true },
    ])
  })

  it('handles markers at start and end', () => {
    expect(parseInlineMarkdown('**bold**')).toEqual([
      { text: 'bold', bold: true },
    ])
  })

  it('returns plain text for empty string', () => {
    expect(parseInlineMarkdown('')).toEqual([{ text: '' }])
  })

  it('does not match unbalanced markers', () => {
    expect(parseInlineMarkdown('a * b')).toEqual([{ text: 'a * b' }])
  })

  it('handles adjacent markers', () => {
    expect(parseInlineMarkdown('**bold***italic*')).toEqual([
      { text: 'bold', bold: true },
      { text: 'italic', italic: true },
    ])
  })
})
