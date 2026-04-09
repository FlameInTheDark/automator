import { describe, expect, it } from 'vitest'

import { getNodeColor, getNodeIcon, getNodeLabel } from './nodeTypes'

describe('nodeTypes helpers', () => {
  it('returns safe fallbacks for missing node types', () => {
    expect(getNodeColor(undefined)).toBe('#6b7280')
    expect(getNodeLabel(undefined)).toBe('Unknown node type')
    expect(getNodeIcon(undefined)).toBe('circle')
  })
})
