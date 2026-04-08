import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { ChevronRight, Search } from 'lucide-react'
import { cn } from '../../lib/utils'
import Input from './Input'

export interface ContextMenuItem {
  label: string
  icon?: React.ReactNode
  shortcut?: string
  onClick?: () => void
  danger?: boolean
  disabled?: boolean
  divider?: boolean
  children?: ContextMenuItem[]
  searchText?: string
}

interface ContextMenuProps {
  x: number
  y: number
  items: ContextMenuItem[]
  onClose: () => void
  searchable?: boolean
  searchPlaceholder?: string
  emptyMessage?: string
}

const MENU_WIDTH = 240
const MENU_MARGIN = 12

function normalizeItems(items: ContextMenuItem[]): ContextMenuItem[] {
  const normalized: ContextMenuItem[] = []

  items.forEach((item) => {
    if (item.divider) {
      if (normalized.length === 0 || normalized[normalized.length - 1].divider) {
        return
      }
      normalized.push(item)
      return
    }

    normalized.push(item)
  })

  while (normalized[normalized.length - 1]?.divider) {
    normalized.pop()
  }

  return normalized
}

function flattenLeafItems(items: ContextMenuItem[]): ContextMenuItem[] {
  return items.flatMap((item) => {
    if (item.divider) {
      return []
    }

    if (item.children?.length) {
      return flattenLeafItems(item.children)
    }

    return [{ ...item, children: undefined }]
  })
}

function filterItems(items: ContextMenuItem[], query: string): ContextMenuItem[] {
  const trimmedQuery = query.trim().toLowerCase()
  if (!trimmedQuery) {
    return normalizeItems(items)
  }

  const filtered = items.flatMap((item) => {
    if (item.divider) {
      return []
    }

    const haystack = `${item.label} ${item.searchText ?? ''}`.toLowerCase()
    const matches = haystack.includes(trimmedQuery)

    if (item.children?.length) {
      if (matches) {
        return flattenLeafItems(item.children)
      }

      return filterItems(item.children, query)
    }

    if (!matches) {
      return []
    }

    return [{ ...item, children: undefined }]
  })

  const seen = new Set<string>()

  return filtered.filter((item) => {
    const key = `${item.label}::${item.searchText ?? ''}::${item.shortcut ?? ''}`
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

interface MenuListProps {
  items: ContextMenuItem[]
  onClose: () => void
}

function clamp(value: number, min: number, max: number): number {
  if (max < min) {
    return min
  }

  return Math.max(min, Math.min(value, max))
}

function getClampedMenuPosition(
  x: number,
  y: number,
  menuWidth: number,
  menuHeight: number,
  viewportWidth: number,
  viewportHeight: number,
): { left: number; top: number } {
  return {
    left: clamp(x, MENU_MARGIN, viewportWidth - menuWidth - MENU_MARGIN),
    top: clamp(y, MENU_MARGIN, viewportHeight - menuHeight - MENU_MARGIN),
  }
}

function MenuList({ items, onClose }: MenuListProps) {
  const [openIndex, setOpenIndex] = useState<number | null>(null)
  const [submenuPosition, setSubmenuPosition] = useState<{ left: number; top: number } | null>(null)
  const buttonRefs = useRef<Record<number, HTMLButtonElement | null>>({})
  const submenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setOpenIndex(null)
  }, [items])

  const updateSubmenuPosition = useCallback(() => {
    if (openIndex === null || typeof window === 'undefined') {
      setSubmenuPosition(null)
      return
    }

    const trigger = buttonRefs.current[openIndex]
    const submenu = submenuRef.current

    if (!trigger || !submenu) {
      return
    }

    const triggerRect = trigger.getBoundingClientRect()
    const submenuRect = submenu.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight
    const submenuWidth = submenuRect.width || MENU_WIDTH
    const submenuHeight = submenuRect.height || 240

    const openToRightX = triggerRect.right + 4
    const openToLeftX = triggerRect.left - submenuWidth - 4
    const fitsRight = openToRightX + submenuWidth + MENU_MARGIN <= viewportWidth
    const fitsLeft = openToLeftX >= MENU_MARGIN

    const left = fitsRight || (!fitsLeft && viewportWidth - triggerRect.right >= triggerRect.left)
      ? clamp(openToRightX, MENU_MARGIN, viewportWidth - submenuWidth - MENU_MARGIN)
      : clamp(openToLeftX, MENU_MARGIN, viewportWidth - submenuWidth - MENU_MARGIN)

    const top = clamp(
      triggerRect.top,
      MENU_MARGIN,
      viewportHeight - submenuHeight - MENU_MARGIN,
    )

    setSubmenuPosition((current) => (
      current?.left === left && current?.top === top
        ? current
        : { left, top }
    ))
  }, [openIndex])

  useLayoutEffect(() => {
    updateSubmenuPosition()
  }, [updateSubmenuPosition, items])

  useEffect(() => {
    if (openIndex === null || typeof window === 'undefined') {
      return
    }

    const handleViewportChange = () => updateSubmenuPosition()

    window.addEventListener('resize', handleViewportChange)
    window.addEventListener('scroll', handleViewportChange, true)

    return () => {
      window.removeEventListener('resize', handleViewportChange)
      window.removeEventListener('scroll', handleViewportChange, true)
    }
  }, [openIndex, updateSubmenuPosition])

  return (
    <div className="py-1">
      {items.map((item, i) => {
        if (item.divider) {
          return <div key={`divider-${i}`} className="my-1 border-t border-border" />
        }

        const hasChildren = !!item.children?.length
        const isOpen = openIndex === i

        return (
          <div
            key={`${item.label}-${i}`}
            className="relative"
            onMouseEnter={() => setOpenIndex(hasChildren ? i : null)}
          >
            <button
              ref={(element) => {
                buttonRefs.current[i] = element
              }}
              onClick={() => {
                if (hasChildren) {
                  setOpenIndex((current) => (current === i ? null : i))
                  return
                }

                if (item.disabled) {
                  return
                }

                item.onClick?.()
                onClose()
              }}
              disabled={item.disabled}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-sm transition-colors',
                item.disabled
                  ? 'text-text-dimmed cursor-not-allowed'
                  : item.danger
                  ? 'text-red-400 hover:bg-red-600/10'
                  : 'text-text hover:bg-bg-overlay'
              )}
            >
              {item.icon && <span className="w-4 h-4 flex-shrink-0">{item.icon}</span>}
              <span className="flex-1 text-left">{item.label}</span>
              {item.shortcut && (
                <span className="text-xs text-text-dimmed">{item.shortcut}</span>
              )}
              {hasChildren && (
                <ChevronRight className="w-3.5 h-3.5 text-text-dimmed flex-shrink-0" />
              )}
            </button>

            {hasChildren && isOpen && (
              <div
                ref={submenuRef}
                className="fixed z-[60] min-w-[240px] max-h-[calc(100vh-24px)] bg-bg-elevated border border-border rounded-lg shadow-2xl overflow-hidden"
                style={submenuPosition ?? { left: MENU_MARGIN, top: MENU_MARGIN }}
              >
                <div className="max-h-[calc(100vh-24px)] overflow-y-auto">
                  <MenuList
                    items={item.children ?? []}
                    onClose={onClose}
                  />
                </div>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

export default function ContextMenu({
  x,
  y,
  items,
  onClose,
  searchable = false,
  searchPlaceholder = 'Search...',
  emptyMessage = 'No results found.',
}: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)
  const searchInputRef = useRef<HTMLInputElement>(null)
  const [query, setQuery] = useState('')
  const [menuPosition, setMenuPosition] = useState({ left: x, top: y })

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [onClose])

  useEffect(() => {
    setQuery('')
  }, [items, searchable])

  useEffect(() => {
    if (!searchable || !searchInputRef.current) return
    searchInputRef.current.focus()
    searchInputRef.current.select()
  }, [searchable, items])

  const filteredItems = useMemo(() => filterItems(items, query), [items, query])
  const updateMenuPosition = useCallback(() => {
    if (!menuRef.current || typeof window === 'undefined') {
      return
    }

    const menuRect = menuRef.current.getBoundingClientRect()
    const nextPosition = getClampedMenuPosition(
      x,
      y,
      menuRect.width || MENU_WIDTH,
      menuRect.height || 320,
      window.innerWidth,
      window.innerHeight,
    )

    setMenuPosition((current) => (
      current.left === nextPosition.left && current.top === nextPosition.top
        ? current
        : nextPosition
    ))
  }, [x, y])

  useLayoutEffect(() => {
    updateMenuPosition()
  }, [updateMenuPosition, filteredItems, searchable])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }

    const handleViewportChange = () => updateMenuPosition()

    window.addEventListener('resize', handleViewportChange)
    window.addEventListener('scroll', handleViewportChange, true)

    return () => {
      window.removeEventListener('resize', handleViewportChange)
      window.removeEventListener('scroll', handleViewportChange, true)
    }
  }, [updateMenuPosition])

  return (
    <div
      ref={menuRef}
      className="fixed z-50 min-w-[240px] max-h-[calc(100vh-24px)] bg-bg-elevated border border-border rounded-lg shadow-2xl animate-fade-in overflow-hidden flex flex-col"
      style={{ left: menuPosition.left, top: menuPosition.top }}
    >
      {searchable && (
        <div className="px-2 pt-2 pb-1 border-b border-border">
          <div className="relative">
            <Search className="w-4 h-4 text-text-dimmed absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
            <Input
              ref={searchInputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.stopPropagation()}
              placeholder={searchPlaceholder}
              className="pl-9"
            />
          </div>
        </div>
      )}

      {filteredItems.length > 0 ? (
        <div className="max-h-[calc(100vh-24px)] overflow-y-auto">
          <MenuList
            items={filteredItems}
            onClose={onClose}
          />
        </div>
      ) : (
        <div className="px-3 py-4 text-sm text-text-dimmed">{emptyMessage}</div>
      )}
    </div>
  )
}
