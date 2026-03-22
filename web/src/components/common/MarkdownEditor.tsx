import { useEffect, useRef } from 'react'
import {
  EditorView,
  keymap,
  placeholder as cmPlaceholder,
  Decoration,
  type DecorationSet,
  ViewPlugin,
  type ViewUpdate,
  WidgetType,
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

// Marker node types whose delimiters we want to hide in live preview
const MARKER_TYPES = new Set([
  'EmphasisMark',    // * or _
  'StrikethroughMark', // ~~  (if available in lezer-markdown)
  'CodeMark',        // `
  'HeaderMark',      // # ## ###
])

// Widget that renders nothing — used to replace hidden markers
class HiddenMarker extends WidgetType {
  toDOM() {
    const span = document.createElement('span')
    span.style.display = 'none'
    return span
  }
}

const hiddenMarkerWidget = Decoration.replace({ widget: new HiddenMarker() })

// ViewPlugin that hides markdown syntax markers on lines without the cursor
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
          if (!MARKER_TYPES.has(node.name)) return
          const nodeLine = state.doc.lineAt(node.from).number
          // Don't hide markers on the line the cursor is on
          if (nodeLine === cursorLine) return
          decorations.push(hiddenMarkerWidget.range(node.from, node.to))
        },
      })

      return Decoration.set(decorations, true)
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
            // Inline markdown rendering
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
