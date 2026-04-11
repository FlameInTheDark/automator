import type { LucideIcon } from 'lucide-react'

import Select from '../ui/Select'
import { cn } from '../../lib/utils'

export type SettingsNavigationItem = {
  id: string
  label: string
  icon: LucideIcon
}

export type SettingsNavigationGroup = {
  id: string
  label: string
  items: SettingsNavigationItem[]
}

interface SettingsNavigationProps {
  groups: SettingsNavigationGroup[]
  activeSection: string
  onSelect: (sectionId: string) => void
}

export default function SettingsNavigation({
  groups,
  activeSection,
  onSelect,
}: SettingsNavigationProps) {
  const options = groups.flatMap((group) =>
    group.items.map((item) => ({
      ...item,
      groupLabel: group.label,
    })),
  )

  return (
    <>
      <div className="md:hidden">
        <label htmlFor="settings-section-select" className="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-text-dimmed">
          Settings Section
        </label>
        <Select
          id="settings-section-select"
          aria-label="Settings section"
          value={activeSection}
          onChange={(event) => onSelect(event.target.value)}
        >
          {options.map((item) => (
            <option key={item.id} value={item.id}>
              {item.groupLabel} / {item.label}
            </option>
          ))}
        </Select>
      </div>

      <aside className="hidden w-full shrink-0 md:block md:w-72">
        <nav aria-label="Settings sections" className="rounded-2xl border border-border bg-bg-input/60 p-2">
          {groups.map((group, groupIndex) => (
            <div
              key={group.id}
              className={cn(groupIndex > 0 && 'mt-4 border-t border-border/70 pt-4')}
            >
              <p className="px-3 pb-2 text-[10px] font-semibold uppercase tracking-[0.18em] text-text-dimmed">
                {group.label}
              </p>
              <div className="space-y-1">
                {group.items.map((item) => {
                  const isActive = item.id === activeSection
                  const Icon = item.icon

                  return (
                    <button
                      key={item.id}
                      type="button"
                      aria-current={isActive ? 'page' : undefined}
                      onClick={() => onSelect(item.id)}
                      className={cn(
                        'flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm transition-colors',
                        isActive
                          ? 'bg-bg-elevated text-text shadow-sm ring-1 ring-border'
                          : 'text-text-muted hover:bg-bg-overlay hover:text-text',
                      )}
                    >
                      <Icon className={cn('h-4 w-4 shrink-0', isActive ? 'text-accent' : 'text-text-dimmed')} />
                      <span className="min-w-0 font-medium">{item.label}</span>
                    </button>
                  )
                })}
              </div>
            </div>
          ))}
        </nav>
      </aside>
    </>
  )
}
