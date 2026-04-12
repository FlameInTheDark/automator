import { describe, expect, it } from 'vitest'

import { buildNodeCatalog, buildNodeMenuCategories, getNodeColor, getNodeIcon, getNodeLabel } from './nodeTypes'

describe('nodeTypes helpers', () => {
  it('returns safe fallbacks for missing node types', () => {
    expect(getNodeColor(undefined)).toBe('#6b7280')
    expect(getNodeLabel(undefined)).toBe('Unknown node type')
    expect(getNodeIcon(undefined)).toBe('circle')
  })

  it('merges runtime plugin definitions into the node catalog', () => {
    const catalog = buildNodeCatalog([
      {
        type: 'action:plugin/acme/request',
        category: 'action',
        source: 'plugin',
        plugin_id: 'acme',
        plugin_name: 'Acme Toolkit',
        label: 'Acme Request',
        description: 'Call the Acme API',
        icon: 'globe',
        color: '#f97316',
        menu_path: ['Acme', 'Requests'],
        default_config: { endpoint: '/status' },
        fields: [],
        outputs: [{ id: 'success', label: 'Success' }],
        output_hints: [{ expression: 'input.result', label: 'Result' }],
      },
    ])

    const definition = catalog.map['action:plugin/acme/request']
    expect(definition).toMatchObject({
      type: 'action:plugin/acme/request',
      source: 'plugin',
      pluginId: 'acme',
      pluginName: 'Acme Toolkit',
      label: 'Acme Request',
      color: '#f97316',
      menuPath: ['Acme', 'Requests'],
      defaultConfig: { endpoint: '/status' },
    })

    const actionCategory = catalog.categories.find((category) => category.id === 'action')
    expect(actionCategory?.types.some((type) => type.type === 'action:plugin/acme/request')).toBe(true)
  })

  it('adds trigger plugin definitions to the trigger catalog', () => {
    const catalog = buildNodeCatalog([
      {
        type: 'trigger:plugin/acme/inbox',
        category: 'trigger',
        source: 'plugin',
        plugin_id: 'acme',
        plugin_name: 'Acme Toolkit',
        label: 'Inbox Event',
        description: 'Subscribe to Acme inbox events',
        icon: 'bell',
        color: '#f59e0b',
        menu_path: ['Acme'],
        default_config: { mailbox: 'support' },
        fields: [],
        outputs: [],
        output_hints: [],
      },
    ])

    const definition = catalog.map['trigger:plugin/acme/inbox']
    expect(definition).toMatchObject({
      type: 'trigger:plugin/acme/inbox',
      category: 'trigger',
      pluginId: 'acme',
      pluginName: 'Acme Toolkit',
      label: 'Inbox Event',
      menuPath: ['Acme'],
    })

    const triggerCategory = catalog.categories.find((category) => category.id === 'trigger')
    expect(triggerCategory?.types.some((type) => type.type === 'trigger:plugin/acme/inbox')).toBe(true)
  })

  it('builds nested menu categories from menu paths', () => {
    const catalog = buildNodeCatalog([
      {
        type: 'action:plugin/acme/request',
        category: 'action',
        source: 'plugin',
        plugin_id: 'acme',
        plugin_name: 'Acme Toolkit',
        label: 'Acme Request',
        description: 'Call the Acme API',
        icon: 'globe',
        color: '#f97316',
        menu_path: ['Acme', 'Requests'],
        default_config: {},
      },
      {
        type: 'action:plugin/acme/status',
        category: 'action',
        source: 'plugin',
        plugin_id: 'acme',
        plugin_name: 'Acme Toolkit',
        label: 'Acme Status',
        description: 'Check status',
        icon: 'server',
        color: '#f97316',
        menu_path: ['Acme'],
        default_config: {},
      },
    ])

    const actionCategory = buildNodeMenuCategories(catalog.categories).find((category) => category.id === 'action')
    const acmeGroup = actionCategory?.groups.find((group) => group.label === 'Acme')
    const requestsGroup = acmeGroup?.groups.find((group) => group.label === 'Requests')

    expect(actionCategory).toBeDefined()
    expect(acmeGroup?.types.map((type) => type.label)).toContain('Acme Status')
    expect(requestsGroup?.types.map((type) => type.label)).toContain('Acme Request')
  })

  it('includes built-in data transformation groups for logic nodes', () => {
    const catalog = buildNodeCatalog()
    const logicCategory = buildNodeMenuCategories(catalog.categories).find((category) => category.id === 'logic')
    const actionCategory = buildNodeMenuCategories(catalog.categories).find((category) => category.id === 'action')
    const transformationGroup = logicCategory?.groups.find((group) => group.label === 'Data Transformation')
    const generalActionGroup = actionCategory?.groups.find((group) => group.label === 'General')
    const listOperationsGroup = transformationGroup?.groups.find((group) => group.label === 'List Operations')
    const combineGroup = transformationGroup?.groups.find((group) => group.label === 'Combine')
    const analyticsGroup = transformationGroup?.groups.find((group) => group.label === 'Analytics')

    expect(listOperationsGroup?.types.map((type) => type.type)).toEqual(
      expect.arrayContaining(['logic:sort', 'logic:limit', 'logic:remove_duplicates']),
    )
    expect(combineGroup?.types.map((type) => type.type)).toEqual(
      expect.arrayContaining(['logic:merge', 'logic:aggregate']),
    )
    expect(analyticsGroup?.types.map((type) => type.type)).toContain('logic:summarize')
    expect(generalActionGroup?.types.map((type) => type.type)).toContain('action:lua')
  })

  it('exposes a visible default token field for webhook triggers', () => {
    const catalog = buildNodeCatalog()
    expect(catalog.map['trigger:webhook']?.defaultConfig).toEqual({
      path: '',
      method: 'POST',
      token: '',
    })
  })
})
