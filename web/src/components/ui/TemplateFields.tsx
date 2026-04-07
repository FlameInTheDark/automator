import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Braces, Search } from 'lucide-react'
import Input from './Input'
import { Textarea } from './Form'
import { cn } from '../../lib/utils'
import type { TemplateSuggestion } from '../../types'

interface TemplateInsertButtonProps {
  suggestions: TemplateSuggestion[]
  onInsert: (template: string) => void
}

const TEMPLATE_MENU_WIDTH = 320
const TEMPLATE_MENU_MARGIN = 12
const TEMPLATE_MENU_OFFSET = 8
const TEMPLATE_MENU_FALLBACK_HEIGHT = 360

export function TemplateInsertButton({ suggestions, onInsert }: TemplateInsertButtonProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [menuPosition, setMenuPosition] = useState({ left: 0, top: 0 })
  const containerRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const searchInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!isOpen) {
      return undefined
    }

    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node
      if (!containerRef.current?.contains(target) && !menuRef.current?.contains(target)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handlePointerDown)
    return () => document.removeEventListener('mousedown', handlePointerDown)
  }, [isOpen])

  useEffect(() => {
    if (!isOpen || !searchInputRef.current) {
      return
    }

    searchInputRef.current.focus()
    searchInputRef.current.select()
  }, [isOpen])

  useEffect(() => {
    if (!isOpen) {
      setQuery('')
      return undefined
    }

    const updatePosition = () => {
      const button = buttonRef.current
      if (!button) {
        return
      }

      const rect = button.getBoundingClientRect()
      const menuHeight = menuRef.current?.offsetHeight ?? TEMPLATE_MENU_FALLBACK_HEIGHT
      const viewportWidth = window.innerWidth
      const viewportHeight = window.innerHeight

      const preferredLeft = rect.right - TEMPLATE_MENU_WIDTH
      const safeLeft = Math.max(
        TEMPLATE_MENU_MARGIN,
        Math.min(preferredLeft, viewportWidth - TEMPLATE_MENU_WIDTH - TEMPLATE_MENU_MARGIN)
      )

      const preferredTop = rect.bottom + TEMPLATE_MENU_OFFSET
      const shouldOpenUpward = preferredTop + menuHeight > viewportHeight - TEMPLATE_MENU_MARGIN &&
        rect.top - TEMPLATE_MENU_OFFSET - menuHeight >= TEMPLATE_MENU_MARGIN

      const safeTop = shouldOpenUpward
        ? Math.max(TEMPLATE_MENU_MARGIN, rect.top - menuHeight - TEMPLATE_MENU_OFFSET)
        : Math.max(
            TEMPLATE_MENU_MARGIN,
            Math.min(preferredTop, viewportHeight - menuHeight - TEMPLATE_MENU_MARGIN)
          )

      setMenuPosition({ left: safeLeft, top: safeTop })
    }

    updatePosition()

    window.addEventListener('resize', updatePosition)
    window.addEventListener('scroll', updatePosition, true)
    return () => {
      window.removeEventListener('resize', updatePosition)
      window.removeEventListener('scroll', updatePosition, true)
    }
  }, [isOpen])

  const filteredSuggestions = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase()
    if (!normalizedQuery) {
      return suggestions
    }

    return suggestions.filter((suggestion) =>
      [
        suggestion.label,
        suggestion.expression,
        suggestion.description ?? '',
      ].join(' ').toLowerCase().includes(normalizedQuery)
    )
  }, [query, suggestions])

  if (suggestions.length === 0) {
    return null
  }

  const menu = isOpen ? createPortal(
    <div
      ref={menuRef}
      className="fixed z-[70] w-80 overflow-hidden rounded-xl border border-border bg-bg-elevated shadow-2xl"
      style={{ left: menuPosition.left, top: menuPosition.top }}
    >
      <div className="border-b border-border p-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-text-dimmed" />
          <Input
            ref={searchInputRef}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search templates..."
            className="py-2 pl-9 pr-3 text-xs"
          />
        </div>
      </div>
      <div className="max-h-72 overflow-y-auto p-1.5">
        {filteredSuggestions.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-text-dimmed">
            No template fields match your search.
          </div>
        ) : (
          filteredSuggestions.map((suggestion) => (
            <button
              key={suggestion.expression}
              type="button"
              onClick={() => {
                onInsert(suggestion.template)
                setIsOpen(false)
                setQuery('')
              }}
              className="w-full rounded-lg px-3 py-2 text-left transition-colors hover:bg-bg-overlay"
            >
              <div className="text-xs font-medium text-text">{suggestion.label}</div>
              <div className="mt-1 font-mono text-[11px] text-accent">{suggestion.template}</div>
              {suggestion.description && (
                <div className="mt-1 text-[11px] text-text-dimmed">{suggestion.description}</div>
              )}
            </button>
          ))
        )}
      </div>
    </div>,
    document.body
  ) : null

  return (
    <div ref={containerRef} className="relative">
      <button
        ref={buttonRef}
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className="inline-flex items-center gap-1 rounded-md border border-border bg-bg-overlay px-2 py-1 text-[11px] font-medium text-text-muted transition-colors hover:text-accent hover:border-accent/50"
      >
        <Braces className="w-3 h-3" />
        Templates
      </button>

      {menu}
    </div>
  )
}

function dispatchNativeInput(element: HTMLInputElement | HTMLTextAreaElement, nextValue: string) {
  if (element instanceof HTMLInputElement) {
    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')?.set
    setter?.call(element, nextValue)
  } else {
    const setter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value')?.set
    setter?.call(element, nextValue)
  }

  element.dispatchEvent(new Event('input', { bubbles: true }))
}

function insertTemplateValue(
  element: HTMLInputElement | HTMLTextAreaElement | null,
  currentValue: string,
  template: string,
) {
  if (!element) {
    return
  }

  const start = element.selectionStart ?? currentValue.length
  const end = element.selectionEnd ?? currentValue.length
  const nextValue = `${currentValue.slice(0, start)}${template}${currentValue.slice(end)}`

  dispatchNativeInput(element, nextValue)

  requestAnimationFrame(() => {
    const cursor = start + template.length
    element.focus()
    element.setSelectionRange(cursor, cursor)
  })
}

const templateTokenPattern = /\{\{[\s\S]*?\}\}/g

function renderHighlightedTemplateText(value: string) {
  if (!value) {
    return null
  }

  const parts: React.ReactNode[] = []
  let lastIndex = 0

  value.replace(templateTokenPattern, (match, offset) => {
    if (offset > lastIndex) {
      parts.push(
        <Fragment key={`text-${lastIndex}`}>
          {value.slice(lastIndex, offset)}
        </Fragment>
      )
    }

    parts.push(
      <span
        key={`token-${offset}`}
        className="rounded bg-amber-400/15 px-0.5 text-amber-300 ring-1 ring-amber-400/25"
      >
        {match}
      </span>
    )

    lastIndex = offset + match.length
    return match
  })

  if (lastIndex < value.length) {
    parts.push(
      <Fragment key={`text-${lastIndex}`}>
        {value.slice(lastIndex)}
      </Fragment>
    )
  }

  return parts
}

function syncOverlayScroll(
  source: HTMLInputElement | HTMLTextAreaElement | null,
  overlay: HTMLDivElement | null,
) {
  if (!source || !overlay) {
    return
  }

  overlay.scrollTop = source.scrollTop
  overlay.scrollLeft = source.scrollLeft
}

function buildEditorMirrorStyle(style?: React.CSSProperties): React.CSSProperties | undefined {
  if (!style) {
    return {
      color: 'transparent',
      caretColor: '#e6edf3',
      WebkitTextFillColor: 'transparent',
    }
  }

  return {
    ...style,
    color: 'transparent',
    caretColor: '#e6edf3',
    WebkitTextFillColor: 'transparent',
  }
}

interface TemplateInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  suggestions?: TemplateSuggestion[]
}

export function TemplateInput({ className, suggestions = [], value, ...props }: TemplateInputProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const overlayRef = useRef<HTMLDivElement>(null)
  const stringValue = typeof value === 'string' ? value : value?.toString() ?? ''

  useEffect(() => {
    syncOverlayScroll(inputRef.current, overlayRef.current)
  }, [stringValue])

  return (
    <div className="space-y-1.5">
      <div
        className={cn(
          'relative overflow-hidden rounded-lg border border-border bg-bg-input transition-colors focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/50',
          props.disabled && 'pointer-events-none opacity-50',
        )}
      >
        {stringValue && (
          <div
            ref={overlayRef}
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 overflow-hidden px-3 py-2 text-text"
          >
            <div className={cn('whitespace-pre text-sm', className)}>
              {renderHighlightedTemplateText(stringValue)}
            </div>
          </div>
        )}
        <input
          ref={inputRef}
          {...props}
          value={value}
          className={cn(
            'relative z-10 block w-full bg-transparent px-3 py-2 text-sm text-text outline-none placeholder:text-text-dimmed',
            stringValue && 'selection:bg-accent/30',
            className,
          )}
          style={stringValue ? buildEditorMirrorStyle(props.style) : props.style}
          onScroll={(event) => {
            syncOverlayScroll(event.currentTarget, overlayRef.current)
            props.onScroll?.(event)
          }}
        />
      </div>
      <div className={cn('flex justify-end', suggestions.length === 0 && 'hidden')}>
        <TemplateInsertButton
          suggestions={suggestions}
          onInsert={(template) => insertTemplateValue(inputRef.current, stringValue, template)}
        />
      </div>
    </div>
  )
}

interface TemplateTextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  suggestions?: TemplateSuggestion[]
}

export function TemplateTextarea({ className, suggestions = [], value, ...props }: TemplateTextareaProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const overlayRef = useRef<HTMLDivElement>(null)
  const stringValue = typeof value === 'string' ? value : value?.toString() ?? ''

  useEffect(() => {
    syncOverlayScroll(textareaRef.current, overlayRef.current)
  }, [stringValue])

  return (
    <div className="space-y-1.5">
      <div
        className={cn(
          'relative overflow-hidden rounded-lg border border-border bg-bg-input transition-colors focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/50',
          props.disabled && 'pointer-events-none opacity-50',
        )}
      >
        {stringValue && (
          <div
            ref={overlayRef}
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 overflow-hidden px-3 py-2 text-text"
          >
            <div className={cn('min-h-full whitespace-pre-wrap break-words text-sm', className)}>
              {renderHighlightedTemplateText(stringValue)}
            </div>
          </div>
        )}
        <Textarea
          ref={textareaRef}
          {...props}
          value={value}
          className={cn(
            'relative z-10 border-0 bg-transparent leading-5 shadow-none focus:border-0 focus:ring-0',
            stringValue && 'selection:bg-accent/30',
            className,
          )}
          style={stringValue ? buildEditorMirrorStyle(props.style) : props.style}
          onScroll={(event) => {
            syncOverlayScroll(event.currentTarget, overlayRef.current)
            props.onScroll?.(event)
          }}
        />
      </div>
      <div className={cn('flex justify-end', suggestions.length === 0 && 'hidden')}>
        <TemplateInsertButton
          suggestions={suggestions}
          onInsert={(template) => insertTemplateValue(textareaRef.current, stringValue, template)}
        />
      </div>
    </div>
  )
}
