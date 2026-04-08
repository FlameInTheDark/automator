import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { CircleHelp } from 'lucide-react'
import { cn } from '../../lib/utils'

type TooltipSide = 'top' | 'bottom'

interface HelpTooltipProps {
  content: React.ReactNode
  label?: string
  side?: TooltipSide
  className?: string
  panelClassName?: string
}

type TooltipPosition = {
  top: number
  left: number
  side: TooltipSide
}

export default function HelpTooltip({
  content,
  label = 'Show help',
  side = 'top',
  className,
  panelClassName,
}: HelpTooltipProps) {
  const tooltipId = useId()
  const triggerRef = useRef<HTMLButtonElement | null>(null)
  const panelRef = useRef<HTMLDivElement | null>(null)
  const [hovered, setHovered] = useState(false)
  const [pinned, setPinned] = useState(false)
  const [position, setPosition] = useState<TooltipPosition | null>(null)

  const open = hovered || pinned

  useLayoutEffect(() => {
    if (!open) {
      return
    }

    const updatePosition = () => {
      if (!triggerRef.current || !panelRef.current) {
        return
      }

      const triggerRect = triggerRef.current.getBoundingClientRect()
      const panelRect = panelRef.current.getBoundingClientRect()
      const gap = 10
      const viewportPadding = 8

      const canShowTop = triggerRect.top >= panelRect.height + gap + viewportPadding
      const canShowBottom = window.innerHeight - triggerRect.bottom >= panelRect.height + gap + viewportPadding

      let resolvedSide = side
      if (resolvedSide === 'top' && !canShowTop && canShowBottom) {
        resolvedSide = 'bottom'
      } else if (resolvedSide === 'bottom' && !canShowBottom && canShowTop) {
        resolvedSide = 'top'
      }

      const unclampedTop = resolvedSide === 'top'
        ? triggerRect.top - panelRect.height - gap
        : triggerRect.bottom + gap
      const unclampedLeft = triggerRect.left + (triggerRect.width / 2) - (panelRect.width / 2)

      const top = Math.max(
        viewportPadding,
        Math.min(unclampedTop, window.innerHeight - panelRect.height - viewportPadding),
      )
      const left = Math.max(
        viewportPadding,
        Math.min(unclampedLeft, window.innerWidth - panelRect.width - viewportPadding),
      )

      setPosition({ top, left, side: resolvedSide })
    }

    const frame = window.requestAnimationFrame(updatePosition)
    const handleViewportChange = () => updatePosition()
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node | null
      if (
        triggerRef.current?.contains(target)
        || panelRef.current?.contains(target)
      ) {
        return
      }

      setPinned(false)
      setHovered(false)
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') {
        return
      }

      setPinned(false)
      setHovered(false)
      triggerRef.current?.blur()
    }

    window.addEventListener('resize', handleViewportChange)
    window.addEventListener('scroll', handleViewportChange, true)
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleKeyDown)

    return () => {
      window.cancelAnimationFrame(frame)
      window.removeEventListener('resize', handleViewportChange)
      window.removeEventListener('scroll', handleViewportChange, true)
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [content, open, side])

  useEffect(() => {
    if (open) {
      return
    }

    setPosition(null)
  }, [open])

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        aria-label={label}
        aria-describedby={open ? tooltipId : undefined}
        aria-expanded={open}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        onFocus={() => setHovered(true)}
        onBlur={() => {
          setHovered(false)
          setPinned(false)
        }}
        onClick={() => setPinned((current) => !current)}
        className={cn(
          'inline-flex h-5 w-5 items-center justify-center rounded-full border border-border bg-bg-overlay text-text-muted transition-colors hover:border-accent/50 hover:text-accent focus:outline-none focus:ring-2 focus:ring-accent/40 focus:ring-offset-0',
          className,
        )}
      >
        <CircleHelp className="h-3.5 w-3.5" />
      </button>
      {open && typeof document !== 'undefined' && createPortal(
        <div
          ref={panelRef}
          id={tooltipId}
          role="tooltip"
          className={cn(
            'pointer-events-auto fixed z-[260] max-w-sm rounded-xl border border-border/90 bg-[#0f1720]/98 px-3 py-2.5 text-xs leading-5 text-text shadow-[0_18px_60px_rgba(0,0,0,0.45)] backdrop-blur-sm',
            panelClassName,
          )}
          style={{
            top: position?.top ?? -9999,
            left: position?.left ?? -9999,
          }}
          onMouseEnter={() => setHovered(true)}
          onMouseLeave={() => setHovered(false)}
        >
          <div
            className="pointer-events-none absolute left-1/2 h-2.5 w-2.5 -translate-x-1/2 rotate-45 border border-border/90 bg-[#0f1720]/98"
            style={{
              top: position?.side === 'bottom' ? -6 : undefined,
              bottom: position?.side === 'top' ? -6 : undefined,
            }}
          />
          <div className="relative space-y-1">
            {content}
          </div>
        </div>,
        document.body,
      )}
    </>
  )
}
