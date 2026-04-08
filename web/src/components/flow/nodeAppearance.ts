import { getNodeColor } from './nodeTypes'
import type { NodeType } from '../../types'

export const DEFAULT_NODE_BORDER_COLOR = '#1e2d3d'
export const DEFAULT_GROUP_COLOR = '#64748b'

export type NodeVisualStatus = 'pending' | 'running' | 'success' | 'error' | undefined

export function isHexColor(value: unknown): value is string {
  return typeof value === 'string' && /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(value)
}

export function withAlpha(color: string, alphaHex: string): string {
  if (!isHexColor(color)) {
    return color
  }

  if (color.length === 4) {
    const [, r, g, b] = color
    return `#${r}${r}${g}${g}${b}${b}${alphaHex}`
  }

  return `${color}${alphaHex}`
}

export function resolveGroupColor(config?: Record<string, unknown>): string {
  return isHexColor(config?.color) ? config.color : DEFAULT_GROUP_COLOR
}

export function getNodeStatusColor(status?: NodeVisualStatus): string | null {
  return status === 'success' ? '#22c55e'
    : status === 'error' ? '#ef4444'
    : status === 'running' ? '#f59e0b'
    : status === 'pending' ? '#6b7280'
    : null
}

export function getNodeBorderTint({
  nodeType,
  selected = false,
  isHighlight = false,
  status,
  config,
}: {
  nodeType: NodeType
  selected?: boolean
  isHighlight?: boolean
  status?: NodeVisualStatus
  config?: Record<string, unknown>
}): string {
  if (nodeType === 'visual:group') {
    return resolveGroupColor(config)
  }

  const color = getNodeColor(nodeType)
  const statusColor = getNodeStatusColor(status)
  const highlightColor = statusColor || '#f59e0b'

  if (selected) {
    return color
  }

  if (isHighlight) {
    return highlightColor
  }

  if (statusColor) {
    return statusColor
  }

  return DEFAULT_NODE_BORDER_COLOR
}
