import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

function parseDateValue(value?: string): Date | null {
  const trimmed = value?.trim()
  if (!trimmed) {
    return null
  }

  const candidates = [
    trimmed,
    trimmed.replace(' UTC', 'Z'),
    trimmed.replace(' +0000 UTC', 'Z'),
  ]

  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(trimmed)) {
    candidates.push(trimmed.replace(' ', 'T') + 'Z')
  }

  for (const candidate of candidates) {
    const parsed = new Date(candidate)
    if (!Number.isNaN(parsed.getTime())) {
      return parsed
    }
  }

  return null
}

export function formatDate(date: string, fallbackDate?: string): string {
  const parsed = parseDateValue(date)
  if (parsed && parsed.getUTCFullYear() > 1) {
    return parsed.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const fallback = parseDateValue(fallbackDate)
  if (fallback && fallback.getUTCFullYear() > 1) {
    return fallback.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return 'Unknown'
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}
