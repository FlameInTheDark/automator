import { Fragment, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
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
        suggestion.preview ?? '',
      ].join(' ').toLowerCase().includes(normalizedQuery)
    )
  }, [query, suggestions])

  const hasSampleSuggestions = suggestions.some((suggestion) => suggestion.kind === 'sample')
  const templateSuggestions = filteredSuggestions.filter((suggestion) => suggestion.kind !== 'sample')
  const sampleSuggestions = filteredSuggestions.filter((suggestion) => suggestion.kind === 'sample')
  const templateSuggestionGroups = useMemo(() => {
    const groups = new Map<string, TemplateSuggestion[]>()

    templateSuggestions.forEach((suggestion) => {
      const groupLabel = suggestion.group ?? 'Templates'
      const existing = groups.get(groupLabel)
      if (existing) {
        existing.push(suggestion)
        return
      }
      groups.set(groupLabel, [suggestion])
    })

    return Array.from(groups.entries()).map(([label, items]) => ({ label, items }))
  }, [templateSuggestions])

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
            placeholder={hasSampleSuggestions ? 'Search templates and samples...' : 'Search templates...'}
            className="py-2 pl-9 pr-3 text-xs"
          />
        </div>
      </div>
      <div className="max-h-72 overflow-y-auto p-1.5">
        {filteredSuggestions.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-text-dimmed">
            {hasSampleSuggestions ? 'No templates or samples match your search.' : 'No template fields match your search.'}
          </div>
        ) : (
          <>
            {templateSuggestions.length > 0 && (
              <div className="space-y-1">
                {hasSampleSuggestions && (
                  <div className="px-3 pb-1 pt-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-text-dimmed">
                    Templates
                  </div>
                )}
                {templateSuggestionGroups.map((group, groupIndex) => (
                  <div key={group.label} className="space-y-1">
                    {(templateSuggestionGroups.length > 1 || hasSampleSuggestions) && (
                      <div className={cn(
                        'px-3 text-[10px] font-semibold uppercase tracking-[0.18em] text-text-dimmed',
                        groupIndex === 0 && hasSampleSuggestions ? 'pb-1 pt-1' : groupIndex === 0 ? 'pb-1' : 'pb-1 pt-3',
                      )}>
                        {group.label}
                      </div>
                    )}
                    {group.items.map((suggestion) => (
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
                        <div className="flex items-center gap-2">
                          <div className="text-xs font-medium text-text">{suggestion.label}</div>
                          {suggestion.badge && (
                            <span className="rounded-full border border-border bg-bg-overlay px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-[0.12em] text-text-dimmed">
                              {suggestion.badge}
                            </span>
                          )}
                        </div>
                        <div className="mt-1 font-mono text-[11px] text-accent">{suggestion.template}</div>
                        {suggestion.description && (
                          <div className="mt-1 text-[11px] text-text-dimmed">{suggestion.description}</div>
                        )}
                      </button>
                    ))}
                  </div>
                ))}
              </div>
            )}

            {sampleSuggestions.length > 0 && (
              <div className="space-y-1">
                <div className={cn(
                  'px-3 text-[10px] font-semibold uppercase tracking-[0.18em] text-text-dimmed',
                  templateSuggestions.length > 0 ? 'pb-1 pt-3' : 'pb-1 pt-1',
                )}>
                  Sample Data
                </div>
                {sampleSuggestions.map((suggestion) => (
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
                    <div className="flex items-center gap-2">
                      <div className="text-xs font-medium text-text">{suggestion.label}</div>
                      <span className="rounded-full border border-accent/30 bg-accent/10 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-[0.12em] text-accent">
                        Latest
                      </span>
                    </div>
                    <div className="mt-1 overflow-hidden rounded-md border border-border/70 bg-slate-950/50 px-2 py-1.5 font-mono text-[11px] text-slate-200 whitespace-pre-wrap break-all">
                      {suggestion.preview ?? suggestion.template}
                    </div>
                    {suggestion.description && (
                      <div className="mt-1 text-[11px] text-text-dimmed">{suggestion.description}</div>
                    )}
                  </button>
                ))}
              </div>
            )}
          </>
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
        {hasSampleSuggestions ? 'Insert' : 'Templates'}
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

interface TemplateTextSegment {
  type: 'text' | 'token'
  value: string
  start: number
}

const templateHighlightStyle: React.CSSProperties = {
  padding: 0,
  margin: 0,
  border: 0,
  font: 'inherit',
  letterSpacing: 'inherit',
  verticalAlign: 'baseline',
  boxShadow: 'inset 0 0 0 1px rgba(251, 191, 36, 0.35)',
}

const mirroredOverlayContentCssProperties = [
  'direction',
  'font',
  'font-family',
  'font-feature-settings',
  'font-kerning',
  'font-size',
  'font-style',
  'font-variant-ligatures',
  'font-variation-settings',
  'font-weight',
  'letter-spacing',
  'line-height',
  'padding-top',
  'padding-right',
  'padding-bottom',
  'padding-left',
  'tab-size',
  'text-align',
  'text-indent',
  'text-transform',
  'white-space',
  'word-break',
  'overflow-wrap',
  'word-spacing',
] as const

function tokenizeTemplateText(value: string): TemplateTextSegment[] {
  if (!value) {
    return []
  }

  const segments: TemplateTextSegment[] = []
  let lastIndex = 0

  value.replace(templateTokenPattern, (match, offset) => {
    if (offset > lastIndex) {
      segments.push({
        type: 'text',
        value: value.slice(lastIndex, offset),
        start: lastIndex,
      })
    }

    segments.push({
      type: 'token',
      value: match,
      start: offset,
    })

    lastIndex = offset + match.length
    return match
  })

  if (lastIndex < value.length) {
    segments.push({
      type: 'text',
      value: value.slice(lastIndex),
      start: lastIndex,
    })
  }

  return segments
}

function renderHighlightedTemplateText(value: string) {
  if (!value) {
    return null
  }

  const parts = tokenizeTemplateText(value).map((segment) => {
    if (segment.type === 'text') {
      return (
        <Fragment key={`text-${segment.start}`}>
          {segment.value}
        </Fragment>
      )
    }

    return (
      <mark
        key={`token-${segment.start}`}
        className="rounded-[4px] bg-amber-400/15 text-amber-200"
        style={templateHighlightStyle}
      >
        {segment.value}
      </mark>
    )
  })

  parts.push(<wbr key="sentinel" />)

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

function syncOverlayMetrics(
  source: HTMLInputElement | HTMLTextAreaElement | null,
  overlay: HTMLDivElement | null,
  overlayContent: HTMLDivElement | null,
) {
  if (!source || !overlay || !overlayContent || typeof window === 'undefined') {
    return
  }

  const computed = window.getComputedStyle(source)

  for (const property of mirroredOverlayContentCssProperties) {
    overlayContent.style.setProperty(property, computed.getPropertyValue(property))
  }

  overlay.style.boxSizing = 'border-box'
  overlay.style.width = `${source.clientWidth}px`
  overlay.style.height = `${source.clientHeight}px`
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
  const overlayContentRef = useRef<HTMLDivElement>(null)
  const stringValue = typeof value === 'string' ? value : value?.toString() ?? ''
  const syncOverlay = useCallback(() => {
    syncOverlayMetrics(inputRef.current, overlayRef.current, overlayContentRef.current)
    syncOverlayScroll(inputRef.current, overlayRef.current)
  }, [])

  useLayoutEffect(() => {
    syncOverlay()
  }, [syncOverlay, stringValue, className, props.style])

  useEffect(() => {
    const source = inputRef.current
    if (!source || typeof ResizeObserver === 'undefined') {
      return undefined
    }

    const observer = new ResizeObserver(() => {
      syncOverlay()
    })
    observer.observe(source)
    return () => observer.disconnect()
  }, [syncOverlay])

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
            className="pointer-events-none absolute left-0 top-0 overflow-hidden text-text"
          >
            <div ref={overlayContentRef} className={cn('whitespace-pre', className)}>
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
  const overlayContentRef = useRef<HTMLDivElement>(null)
  const stringValue = typeof value === 'string' ? value : value?.toString() ?? ''
  const syncOverlay = useCallback(() => {
    syncOverlayMetrics(textareaRef.current, overlayRef.current, overlayContentRef.current)
    syncOverlayScroll(textareaRef.current, overlayRef.current)
  }, [])

  useLayoutEffect(() => {
    syncOverlay()
  }, [syncOverlay, stringValue, className, props.style, props.rows])

  useEffect(() => {
    const source = textareaRef.current
    if (!source || typeof ResizeObserver === 'undefined') {
      return undefined
    }

    const observer = new ResizeObserver(() => {
      syncOverlay()
    })
    observer.observe(source)
    return () => observer.disconnect()
  }, [syncOverlay])

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
            className="pointer-events-none absolute left-0 top-0 overflow-hidden text-text"
          >
            <div
              ref={overlayContentRef}
              className={cn('min-h-full whitespace-pre-wrap break-words', className)}
            >
              {renderHighlightedTemplateText(stringValue)}
            </div>
          </div>
        )}
        <Textarea
          ref={textareaRef}
          {...props}
          value={value}
          className={cn(
            'relative z-10 border-0 bg-transparent whitespace-pre-wrap break-words leading-5 shadow-none focus:border-0 focus:ring-0',
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
