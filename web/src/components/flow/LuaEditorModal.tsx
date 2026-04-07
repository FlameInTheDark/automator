import { useEffect, useMemo, useRef, useState } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import type { editor as MonacoEditorNS } from 'monaco-editor'
import { Code, X } from 'lucide-react'
import Button from '../ui/Button'
import { TemplateInsertButton } from '../ui/TemplateFields'
import type { TemplateSuggestion } from '../../types'

interface LuaEditorModalProps {
  value: string
  suggestions: TemplateSuggestion[]
  onSave: (value: string) => void
  onClose: () => void
}

export default function LuaEditorModal({ value, suggestions, onSave, onClose }: LuaEditorModalProps) {
  const [draft, setDraft] = useState(value)
  const editorRef = useRef<MonacoEditorNS.IStandaloneCodeEditor | null>(null)
  const modalRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setDraft(value)
  }, [value])

  useEffect(() => {
    const modal = modalRef.current
    if (!modal) {
      return undefined
    }

    const handleKeyboardEvent = (event: KeyboardEvent) => {
      event.stopPropagation()

      if (event.type === 'keydown' && event.key === 'Escape') {
        event.preventDefault()
        onClose()
      }
    }

    modal.addEventListener('keydown', handleKeyboardEvent)
    modal.addEventListener('keyup', handleKeyboardEvent)

    return () => {
      modal.removeEventListener('keydown', handleKeyboardEvent)
      modal.removeEventListener('keyup', handleKeyboardEvent)
    }
  }, [onClose])

  const lineCount = useMemo(() => {
    if (!draft) {
      return 0
    }
    return draft.split(/\r?\n/).length
  }, [draft])

  const handleMount: OnMount = (editor) => {
    editorRef.current = editor
    editor.focus()
  }

  const handleInsertTemplate = (template: string) => {
    const editor = editorRef.current
    if (!editor) {
      setDraft((current) => `${current}${template}`)
      return
    }

    const selection = editor.getSelection()
    if (!selection) {
      setDraft((current) => `${current}${template}`)
      return
    }

    editor.executeEdits('template-insert', [
      {
        range: selection,
        text: template,
        forceMoveMarkers: true,
      },
    ])
    editor.focus()
  }

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/70 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        ref={modalRef}
        className="grid h-[88vh] min-h-[560px] w-full max-w-6xl grid-rows-[auto_auto_minmax(0,1fr)_auto] overflow-hidden rounded-2xl border border-border bg-bg-elevated shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex shrink-0 items-start justify-between gap-4 border-b border-border bg-bg-overlay px-5 py-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <Code className="h-5 w-5 text-accent" />
              <p className="truncate text-lg font-semibold text-text">Lua Script Editor</p>
            </div>
            <p className="mt-1 text-xs text-text-dimmed">
              {lineCount} lines · {draft.length} characters
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-text-dimmed transition-colors hover:text-text"
            aria-label="Close Lua editor"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex shrink-0 items-center justify-between gap-3 border-b border-border px-5 py-3">
          <p className="text-sm text-text-muted">
            Edit the script in a larger workspace and insert pipeline templates at the cursor position.
          </p>
          <TemplateInsertButton suggestions={suggestions} onInsert={handleInsertTemplate} />
        </div>

        <div className="min-h-0 overflow-hidden">
          <Editor
            height="100%"
            defaultLanguage="lua"
            value={draft}
            onMount={handleMount}
            onChange={(nextValue) => setDraft(nextValue || '')}
            theme="vs-dark"
            options={{
              minimap: { enabled: false },
              fontSize: 14,
              lineNumbers: 'on',
              scrollBeyondLastLine: false,
              automaticLayout: true,
              padding: { top: 12, bottom: 16 },
              wordWrap: 'on',
              tabSize: 2,
            }}
          />
        </div>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t border-border bg-bg-overlay px-5 py-4">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={() => {
              onSave(draft)
              onClose()
            }}
          >
            Save code
          </Button>
        </div>
      </div>
    </div>
  )
}
