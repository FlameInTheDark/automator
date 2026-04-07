import type { Edge, Node } from '@xyflow/react'
import type { NodeExecution, NodeType, TemplateSuggestion } from '../types'

type NodeOutputHint = {
  expression: string
  label: string
  description?: string
}

const NODE_OUTPUT_HINTS: Partial<Record<NodeType, NodeOutputHint[]>> = {
  'action:proxmox_list_nodes': [
    { expression: 'input.clusterId', label: 'Cluster ID' },
    { expression: 'input.clusterName', label: 'Cluster name' },
    { expression: 'input.count', label: 'Node count' },
    { expression: 'input.nodes', label: 'Nodes list (JSON)' },
    { expression: 'input.nodes[0].node', label: 'First node name' },
    { expression: 'input.nodes[0].status', label: 'First node status' },
  ],
  'action:proxmox_list_workloads': [
    { expression: 'input.clusterName', label: 'Cluster name' },
    { expression: 'input.workloads', label: 'Workloads list (JSON)' },
    { expression: 'input.vms', label: 'VM list (JSON)' },
    { expression: 'input.containers', label: 'Container list (JSON)' },
    { expression: 'input.workloads[0].vmid', label: 'First workload VMID' },
    { expression: 'input.workloads[0].status', label: 'First workload status' },
    { expression: 'input.workloads[0].node', label: 'First workload node' },
  ],
  'action:vm_start': [
    { expression: 'input.clusterName', label: 'Cluster name' },
    { expression: 'input.node', label: 'Node name' },
    { expression: 'input.vmid', label: 'VM ID' },
    { expression: 'input.status', label: 'Action status' },
  ],
  'action:vm_stop': [
    { expression: 'input.clusterName', label: 'Cluster name' },
    { expression: 'input.node', label: 'Node name' },
    { expression: 'input.vmid', label: 'VM ID' },
    { expression: 'input.status', label: 'Action status' },
  ],
  'action:vm_clone': [
    { expression: 'input.clusterName', label: 'Cluster name' },
    { expression: 'input.node', label: 'Node name' },
    { expression: 'input.vmid', label: 'Source VM ID' },
    { expression: 'input.newId', label: 'New VM ID' },
    { expression: 'input.newName', label: 'New VM name' },
    { expression: 'input.status', label: 'Action status' },
  ],
  'action:http': [
    { expression: 'input.status_code', label: 'HTTP status code' },
    { expression: 'input.response', label: 'Response body (JSON)' },
  ],
  'logic:condition': [
    { expression: 'input.result', label: 'Condition result' },
    { expression: 'input.condition', label: 'Condition expression' },
    { expression: 'input.error', label: 'Evaluation error' },
  ],
  'logic:switch': [
    { expression: 'input.conditions', label: 'Condition evaluations (JSON)' },
    { expression: 'input.matches', label: 'Branch match map (JSON)' },
    { expression: 'input.matched', label: 'Matched conditions (JSON)' },
    { expression: 'input.hasMatch', label: 'Any condition matched' },
    { expression: 'input.defaultMatched', label: 'Else branch matched' },
  ],
  'llm:prompt': [
    { expression: 'input.prompt', label: 'Rendered prompt' },
    { expression: 'input.content', label: 'LLM response text' },
    { expression: 'input.usage', label: 'Token usage (JSON)' },
    { expression: 'input.usage.total_tokens', label: 'Total tokens' },
  ],
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export function parseExecutionJSON(value?: string): unknown {
  if (!value) {
    return undefined
  }

  try {
    return JSON.parse(value)
  } catch {
    return value
  }
}

function collectPaths(
  value: unknown,
  path: string,
  results: TemplateSuggestion[],
  seen: Set<string>,
  depth: number,
): void {
  if (depth > 3 || !path) {
    return
  }

  addSuggestion(results, seen, path, path, describeValue(value))

  if (Array.isArray(value)) {
    if (value.length === 0) {
      return
    }
    collectPaths(value[0], `${path}[0]`, results, seen, depth + 1)
    return
  }

  if (!isObject(value)) {
    return
  }

  Object.entries(value).forEach(([key, child]) => {
    collectPaths(child, `${path}.${key}`, results, seen, depth + 1)
  })
}

function describeValue(value: unknown): string | undefined {
  if (Array.isArray(value)) {
    return 'Array value, inserted as JSON.'
  }
  if (isObject(value)) {
    return 'Object value, inserted as JSON.'
  }
  if (typeof value === 'boolean' || typeof value === 'number') {
    return 'Scalar value.'
  }
  if (typeof value === 'string') {
    return 'String value.'
  }
  return undefined
}

function addSuggestion(
  results: TemplateSuggestion[],
  seen: Set<string>,
  expression: string,
  label: string,
  description?: string,
): void {
  if (!expression || seen.has(expression)) {
    return
  }

  seen.add(expression)
  results.push({
    expression,
    template: `{{${expression}}}`,
    label,
    description,
  })
}

function buildSchemaSuggestions(nodeType: NodeType | undefined, sourceLabel?: string): TemplateSuggestion[] {
  const hints = nodeType ? NODE_OUTPUT_HINTS[nodeType] : undefined
  if (!hints) {
    return []
  }

  return hints.map((hint) => ({
    expression: hint.expression,
    template: `{{${hint.expression}}}`,
    label: hint.label,
    description: sourceLabel ? `${hint.description ?? 'Suggested field.'} Source: ${sourceLabel}.` : hint.description,
  }))
}

function mergeSourceOutputs(outputs: unknown[]): Record<string, unknown> {
  return outputs.reduce<Record<string, unknown>>((acc, current) => {
    if (isObject(current)) {
      Object.assign(acc, current)
    }
    return acc
  }, {})
}

export function buildTemplateSuggestions(
  selectedNodeId: string,
  nodes: Node[],
  edges: Edge[],
  latestNodeExecutions: NodeExecution[] = [],
): TemplateSuggestion[] {
  const results: TemplateSuggestion[] = []
  const seen = new Set<string>()

  const incomingEdges = edges.filter((edge) => edge.target === selectedNodeId)
  if (incomingEdges.length === 0) {
    return results
  }

  addSuggestion(results, seen, 'input', 'Entire merged input', 'All incoming data, inserted as JSON.')

  const sourceNodes = incomingEdges
    .map((edge) => nodes.find((node) => node.id === edge.source))
    .filter((node): node is Node => !!node)

  const latestOutputs = sourceNodes
    .map((node) => latestNodeExecutions.find((execution) => execution.node_id === node.id))
    .map((execution) => parseExecutionJSON(execution?.output))
    .filter((value): value is Record<string, unknown> => isObject(value))

  const mergedOutput = mergeSourceOutputs(latestOutputs)
  if (Object.keys(mergedOutput).length > 0) {
    collectPaths(mergedOutput, 'input', results, seen, 0)
  }

  sourceNodes.forEach((node) => {
    const sourceType = node.data?.type as NodeType | undefined
    const sourceLabel = (node.data?.label as string | undefined) || node.id
    buildSchemaSuggestions(sourceType, sourceLabel).forEach((suggestion) => {
      addSuggestion(results, seen, suggestion.expression, suggestion.label, suggestion.description)
    })
  })

  return results
}
