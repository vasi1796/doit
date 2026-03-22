import { useEffect, useRef } from 'react'
import {
  EditorView,
  keymap,
  placeholder as cmPlaceholder,
  Decoration,
  type DecorationSet,
  ViewPlugin,
  type ViewUpdate,
} from '@codemirror/view'
import { EditorState, type Range } from '@codemirror/state'
import { syntaxTree } from '@codemirror/language'
import { markdown } from '@codemirror/lang-markdown'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'

interface MarkdownEditorProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  minHeight?: string
}

// Map syntax tree node names to CSS classes for styled content
const CONTENT_STYLES: Record<string, Decoration> = {
  StrongEmphasis: Decoration.mark({ class: 'cm-lp-strong' }),
  Emphasis: Decoration.mark({ class: 'cm-lp-emphasis' }),
  Strikethrough: Decoration.mark({ class: 'cm-lp-strikethrough' }),
  InlineCode: Decoration.mark({ class: 'cm-lp-code' }),
}

// Marker node types to hide
const MARKER_TYPES = new Set([
  'EmphasisMark',
  'StrikethroughMark',
  'CodeMark',
  'HeaderMark',
])

const hideDecoration = Decoration.replace({})

// ViewPlugin that provides Obsidian-style live preview:
// - Hides markdown markers on lines without the cursor
// - Applies formatting styles to the content text
const livePreviewPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet

    constructor(view: EditorView) {
      this.decorations = this.buildDecorations(view)
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.selectionSet) {
        this.decorations = this.buildDecorations(update.view)
      }
    }

    buildDecorations(view: EditorView): DecorationSet {
      const { state } = view
      const cursorLine = state.doc.lineAt(state.selection.main.head).number
      const decorations: Range<Decoration>[] = []

      syntaxTree(state).iterate({
        enter: (node) => {
          const nodeLine = state.doc.lineAt(node.from).number
          const isActiveLine = nodeLine === cursorLine

          // Hide markers on non-active lines
          if (MARKER_TYPES.has(node.name) && !isActiveLine) {
            decorations.push(hideDecoration.range(node.from, node.to))
            return
          }

          // Apply content styles on non-active lines
          const styleDeco = CONTENT_STYLES[node.name]
          if (styleDeco && !isActiveLine) {
            decorations.push(styleDeco.range(node.from, node.to))
          }
        },
      })

      return Decoration.set(decorations.sort((a, b) => a.from - b.from || a.value.startSide - b.value.startSide))
    }
  },
  { decorations: (v) => v.decorations },
)

export function MarkdownEditor({ value, onChange, placeholder = 'Notes', minHeight = '80px' }: MarkdownEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const onChangeRef = useRef(onChange)
  useEffect(() => { onChangeRef.current = onChange }, [onChange])

  useEffect(() => {
    if (!containerRef.current) return

    const view = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [
          markdown(),
          history(),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          cmPlaceholder(placeholder),
          livePreviewPlugin,
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              onChangeRef.current(update.state.doc.toString())
            }
          }),
          EditorView.theme({
            '&': {
              fontSize: '16px',
              minHeight,
            },
            '&.cm-focused': {
              outline: 'none',
            },
            '.cm-content': {
              fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", system-ui, sans-serif',
              padding: '0',
              caretColor: '#007aff',
            },
            '.cm-line': {
              padding: '2px 0',
            },
            '.cm-placeholder': {
              color: '#c7c7cc',
            },
            // Live preview styles (applied by our plugin on non-active lines)
            '.cm-lp-strong': { fontWeight: '700' },
            '.cm-lp-emphasis': { fontStyle: 'italic' },
            '.cm-lp-strikethrough': { textDecoration: 'line-through' },
            '.cm-lp-code': {
              fontFamily: 'ui-monospace, "SF Mono", Menlo, monospace',
              fontSize: '0.9em',
              backgroundColor: '#f5f5f7',
              borderRadius: '3px',
              padding: '1px 4px',
            },
            // Syntax highlighting on active line (default CodeMirror classes)
            '.cm-header-1': { fontSize: '1.4em', fontWeight: '700' },
            '.cm-header-2': { fontSize: '1.2em', fontWeight: '600' },
            '.cm-header-3': { fontSize: '1.1em', fontWeight: '600' },
            '.cm-strong': { fontWeight: '700' },
            '.cm-emphasis': { fontStyle: 'italic' },
            '.cm-strikethrough': { textDecoration: 'line-through' },
            '.cm-monospace': {
              fontFamily: 'ui-monospace, "SF Mono", Menlo, monospace',
              fontSize: '0.9em',
              backgroundColor: '#f5f5f7',
              borderRadius: '3px',
              padding: '1px 4px',
            },
            '.cm-url': { color: '#007aff' },
            '.cm-link': { color: '#007aff', textDecoration: 'underline' },
          }),
          EditorView.lineWrapping,
        ],
      }),
      parent: containerRef.current,
    })

    viewRef.current = view

    return () => {
      view.destroy()
      viewRef.current = null
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Sync external value changes (e.g., from initial load)
  useEffect(() => {
    const view = viewRef.current
    if (!view) return
    const current = view.state.doc.toString()
    if (current !== value) {
      view.dispatch({
        changes: { from: 0, to: current.length, insert: value },
      })
    }
  }, [value])

  return <div ref={containerRef} className="text-text-note" />
}
