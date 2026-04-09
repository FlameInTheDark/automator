import type { Edge, Node, Viewport } from '@xyflow/react'

import type { LivePipelineOperation } from '../types'

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function mergeObjects<T extends Record<string, unknown>>(base: T, patch: Record<string, unknown>): T {
  const merged: Record<string, unknown> = { ...base }

  for (const [key, patchValue] of Object.entries(patch)) {
    const baseValue = merged[key]
    if (isRecord(baseValue) && isRecord(patchValue)) {
      merged[key] = mergeObjects(baseValue, patchValue)
      continue
    }

    merged[key] = patchValue
  }

  return merged as T
}

function assertValidNode(node: Node, operationType: LivePipelineOperation['type']) {
  if (!node?.id) {
    throw new Error(`${operationType} produced a node without an id.`)
  }

  if (
    !node.position
    || typeof node.position.x !== 'number'
    || typeof node.position.y !== 'number'
  ) {
    throw new Error(`${operationType} produced node "${node.id}" without a valid position.`)
  }
}

function assertValidEdge(edge: Edge, operationType: LivePipelineOperation['type']) {
  if (!edge?.id) {
    throw new Error(`${operationType} produced an edge without an id.`)
  }
  if (!edge.source || !edge.target) {
    throw new Error(`${operationType} produced edge "${edge.id}" without source or target.`)
  }
}

export function applyLivePipelineOperations(input: {
  nodes: Node[]
  edges: Edge[]
  viewport: Viewport
  operations: LivePipelineOperation[]
}): {
  nodes: Node[]
  edges: Edge[]
  viewport: Viewport
} {
  let nodes = [...input.nodes]
  let edges = [...input.edges]
  let viewport = input.viewport

  for (const operation of input.operations) {
    switch (operation.type) {
      case 'add_nodes': {
        const nextNodes = (operation.nodes ?? []).map((node) => {
          assertValidNode(node, operation.type)
          return node
        })
        nodes = nodes.concat(nextNodes)
        break
      }
      case 'update_nodes': {
        const updates = new Map((operation.nodes ?? []).map((node) => [node.id, node]))
        nodes = nodes.map((node) => {
          const update = updates.get(node.id)
          if (!update) {
            return node
          }

          const mergedNode = mergeObjects(node as Record<string, unknown>, update as Record<string, unknown>) as Node
          assertValidNode(mergedNode, operation.type)
          return mergedNode
        })
        break
      }
      case 'delete_nodes': {
        const toDelete = new Set(operation.node_ids ?? [])
        nodes = nodes.filter((node) => !toDelete.has(node.id))
        edges = edges.filter((edge) => !toDelete.has(edge.source) && !toDelete.has(edge.target))
        break
      }
      case 'add_edges': {
        const nextEdges = (operation.edges ?? []).map((edge) => {
          assertValidEdge(edge, operation.type)
          return edge
        })
        edges = edges.concat(nextEdges)
        break
      }
      case 'update_edges': {
        const updates = new Map((operation.edges ?? []).map((edge) => [edge.id, edge]))
        edges = edges.map((edge) => {
          const update = updates.get(edge.id)
          if (!update) {
            return edge
          }

          const mergedEdge = mergeObjects(edge as Record<string, unknown>, update as Record<string, unknown>) as Edge
          assertValidEdge(mergedEdge, operation.type)
          return mergedEdge
        })
        break
      }
      case 'delete_edges': {
        const toDelete = new Set(operation.edge_ids ?? [])
        edges = edges.filter((edge) => !toDelete.has(edge.id))
        break
      }
      case 'set_viewport':
        if (operation.viewport) {
          viewport = operation.viewport
        }
        break
    }
  }

  return { nodes, edges, viewport }
}
