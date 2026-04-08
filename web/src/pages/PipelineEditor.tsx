import { useCallback, useRef, useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  ReactFlow,
  ReactFlowProvider,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  applyNodeChanges,
  addEdge,
  type OnConnect,
  type Node,
  type Edge,
  type OnNodesChange,
  type OnEdgesChange,
  MarkerType,
  Panel,
  useReactFlow,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Save, Play, ArrowLeft, Loader2, Trash2, Copy, ListChecks, Scissors, Edit2, GitBranch, Zap, Clock, Webhook, Square, Globe, Code, Split, Brain, Link, MessageSquare, Send, Power, Bot, Workflow, List, Wrench, CornerDownLeft, RefreshCw, Server, Shield } from 'lucide-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

import NodePalette from '../components/flow/NodePalette'
import NodeConfigPanel from '../components/flow/NodeConfigPanel'
import ExecutionLog from '../components/flow/ExecutionLog'
import NodeExecutionModal from '../components/flow/NodeExecutionModal'
import AutomatorNode from '../components/flow/nodes/AutomatorNode'
import AutomatorEdge from '../components/flow/edges/AutomatorEdge'
import { NODE_CATEGORIES } from '../components/flow/nodeTypes'
import { DEFAULT_NODE_BORDER_COLOR, getNodeBorderTint } from '../components/flow/nodeAppearance'
import { api } from '../api/client'
import { useUIStore } from '../store/ui'
import Button from '../components/ui/Button'
import Badge from '../components/ui/Badge'
import ContextMenu, { type ContextMenuItem } from '../components/ui/ContextMenu'
import Input from '../components/ui/Input'
import { Textarea } from '../components/ui/Form'
import type { NodeExecutionLogData, Pipeline, PipelineRunResponse, NodeType } from '../types'

const nodeTypes = {
  automator: AutomatorNode,
}

const edgeTypes = {
  automator: AutomatorEdge,
}

const defaultEdgeOptions = {
  type: 'automator',
  markerEnd: {
    type: MarkerType.ArrowClosed,
    color: '#1e2d3d',
  },
  style: {
    stroke: '#1e2d3d',
    strokeWidth: 2,
  },
}

const toolEdgeOptions = {
  type: 'automator',
  markerEnd: {
    type: MarkerType.ArrowClosed,
    color: '#38bdf8',
  },
  style: {
    stroke: '#38bdf8',
    strokeWidth: 2,
    strokeDasharray: '8 4',
  },
}

const nodeMenuIconMap: Record<string, React.ElementType> = {
  zap: Zap,
  clock: Clock,
  webhook: Webhook,
  'message-square': MessageSquare,
  play: Play,
  square: Square,
  copy: Copy,
  globe: Globe,
  link: Link,
  code: Code,
  send: Send,
  'git-branch': GitBranch,
  split: Split,
  brain: Brain,
  bot: Bot,
  workflow: Workflow,
  list: List,
  wrench: Wrench,
  'refresh-cw': RefreshCw,
  'trash-2': Trash2,
  'corner-down-left': CornerDownLeft,
}

const categoryMenuIconMap: Record<string, React.ElementType> = {
  trigger: Zap,
  action: Play,
  tool: Wrench,
  logic: GitBranch,
  llm: Brain,
}

function isProxmoxNodeType(type: NodeType): boolean {
  return [
    'action:proxmox_list_nodes',
    'action:proxmox_list_workloads',
    'action:vm_start',
    'action:vm_stop',
    'action:vm_clone',
    'tool:proxmox_list_nodes',
    'tool:proxmox_list_workloads',
    'tool:vm_start',
    'tool:vm_stop',
    'tool:vm_clone',
  ].includes(type)
}

function isKubernetesNodeType(type: NodeType): boolean {
  return type.includes(':kubernetes_')
}

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false
  }

  if (target.isContentEditable) {
    return true
  }

  return Boolean(target.closest('input, textarea, select, [contenteditable="true"]'))
}

type EditorContextMenuState = {
  x: number
  y: number
  items: ContextMenuItem[]
  searchable?: boolean
  searchPlaceholder?: string
  emptyMessage?: string
}

const VISUAL_GROUP_TYPE: NodeType = 'visual:group'
const DEFAULT_GROUP_COLOR = '#64748b'
const GROUP_PADDING_X = 32
const GROUP_PADDING_TOP = 56
const GROUP_PADDING_BOTTOM = 28
const MIN_GROUP_WIDTH = 280
const MIN_GROUP_HEIGHT = 180
const DEFAULT_NODE_WIDTH = 220
const DEFAULT_NODE_HEIGHT = 120

type NodeBounds = {
  x: number
  y: number
  width: number
  height: number
}

function isVisualGroupNodeType(type: unknown): type is NodeType {
  return type === VISUAL_GROUP_TYPE
}

function isGroupNode(node: Node | null | undefined): boolean {
  return isVisualGroupNodeType(node?.data?.type)
}

function getNodeWidth(node: Node): number {
  if (typeof node.measured?.width === 'number') return node.measured.width
  if (typeof node.width === 'number') return node.width
  if (typeof node.initialWidth === 'number') return node.initialWidth

  const styleWidth = node.style?.width
  if (typeof styleWidth === 'number') return styleWidth

  return DEFAULT_NODE_WIDTH
}

function getNodeHeight(node: Node): number {
  if (typeof node.measured?.height === 'number') return node.measured.height
  if (typeof node.height === 'number') return node.height
  if (typeof node.initialHeight === 'number') return node.initialHeight

  const styleHeight = node.style?.height
  if (typeof styleHeight === 'number') return styleHeight

  return DEFAULT_NODE_HEIGHT
}

function getAbsoluteNodePosition(node: Node, nodesById: Map<string, Node>): { x: number; y: number } {
  let x = node.position.x
  let y = node.position.y
  let parentId = node.parentId
  const visited = new Set<string>()

  while (parentId) {
    if (visited.has(parentId)) {
      break
    }

    visited.add(parentId)

    const parent = nodesById.get(parentId)
    if (!parent) {
      break
    }

    x += parent.position.x
    y += parent.position.y
    parentId = parent.parentId
  }

  return { x, y }
}

function getNodeBounds(node: Node, nodesById: Map<string, Node>): NodeBounds {
  const absolutePosition = getAbsoluteNodePosition(node, nodesById)

  return {
    ...absolutePosition,
    width: getNodeWidth(node),
    height: getNodeHeight(node),
  }
}

function isPointInsideBounds(
  point: { x: number; y: number },
  bounds: NodeBounds,
  padding = 0,
): boolean {
  return (
    point.x >= bounds.x + padding
    && point.x <= bounds.x + bounds.width - padding
    && point.y >= bounds.y + padding
    && point.y <= bounds.y + bounds.height - padding
  )
}

function findContainingGroup(node: Node, nodes: Node[]): Node | null {
  const nodesById = new Map(nodes.map((candidate) => [candidate.id, candidate]))
  const nodeBounds = getNodeBounds(node, nodesById)
  const center = {
    x: nodeBounds.x + (nodeBounds.width / 2),
    y: nodeBounds.y + (nodeBounds.height / 2),
  }

  const containingGroups = nodes
    .filter((candidate) => candidate.id !== node.id && isGroupNode(candidate))
    .map((group) => ({
      group,
      bounds: getNodeBounds(group, nodesById),
    }))
    .filter(({ bounds }) => isPointInsideBounds(center, bounds, 8))
    .sort((left, right) => (
      (left.bounds.width * left.bounds.height) - (right.bounds.width * right.bounds.height)
    ))

  return containingGroups[0]?.group ?? null
}

function getMinimumGroupDimensions(group: Node, nodes: Node[]): { width: number; height: number } {
  const children = nodes.filter((node) => node.parentId === group.id)

  if (children.length === 0) {
    return {
      width: MIN_GROUP_WIDTH,
      height: MIN_GROUP_HEIGHT,
    }
  }

  const maxChildRight = Math.max(...children.map((node) => node.position.x + getNodeWidth(node)))
  const maxChildBottom = Math.max(...children.map((node) => node.position.y + getNodeHeight(node)))

  return {
    width: Math.max(maxChildRight + GROUP_PADDING_X, MIN_GROUP_WIDTH),
    height: Math.max(maxChildBottom + GROUP_PADDING_BOTTOM, MIN_GROUP_HEIGHT),
  }
}

function getNextGroupLabel(nodes: Node[]): string {
  const count = nodes.filter((node) => isGroupNode(node)).length
  return `Group ${count + 1}`
}

function canGroupNodes(nodes: Node[]): boolean {
  return nodes.length > 1 && nodes.every((node) => !isGroupNode(node) && !node.parentId)
}

function normalizeNodesForSubflows(nodes: Node[]): Node[] {
  const nodesById = new Map(nodes.map((node) => [node.id, node]))
  const childrenByParentId = new Map<string, Node[]>()

  nodes.forEach((node) => {
    if (!node.parentId || !nodesById.has(node.parentId)) {
      return
    }

    const siblings = childrenByParentId.get(node.parentId) ?? []
    siblings.push(node)
    childrenByParentId.set(node.parentId, siblings)
  })

  const ordered: Node[] = []
  const visited = new Set<string>()

  const visit = (node: Node) => {
    if (visited.has(node.id)) {
      return
    }

    visited.add(node.id)
    ordered.push(node)

    const children = childrenByParentId.get(node.id) ?? []
    children.forEach(visit)
  }

  nodes.forEach((node) => {
    if (node.parentId && nodesById.has(node.parentId)) {
      return
    }

    visit(node)
  })

  nodes.forEach(visit)

  return ordered
}

function getVisualExecutionStatus(status?: string): 'pending' | 'running' | 'success' | 'error' | undefined {
  if (!status) return undefined

  switch (status) {
    case 'completed':
      return 'success'
    case 'failed':
      return 'error'
    case 'running':
      return 'running'
    case 'pending':
      return 'pending'
    default:
      return undefined
  }
}

function isToolNodeType(type: unknown): type is NodeType {
  return typeof type === 'string' && type.startsWith('tool:')
}

function hasReturnNode(nodes: Node[]): boolean {
  return nodes.some((node) => node.data?.type === 'logic:return')
}

function getRenderedNodeBorderColor(node: Node | undefined, selectedNodeIds: ReadonlySet<string>): string {
  if (!node || typeof node.data?.type !== 'string') {
    return DEFAULT_NODE_BORDER_COLOR
  }

  const nodeType = node.data.type

  const config = typeof node.data?.config === 'object' && node.data?.config !== null
    ? node.data.config as Record<string, unknown>
    : undefined

  return getNodeBorderTint({
    nodeType: nodeType as NodeType,
    selected: selectedNodeIds.has(node.id),
    isHighlight: node.data?.isHighlight === true,
    status: node.data?.status as 'pending' | 'running' | 'success' | 'error' | undefined,
    config,
  })
}

function PipelineEditor() {
  const { id } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const reactFlowWrapper = useRef<HTMLDivElement>(null)
  const { screenToFlowPosition, getViewport } = useReactFlow()
  const { selectedNodeId, setSelectedNodeId, addToast } = useUIStore()

  const [nodes, setNodes] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])
  const [isSaving, setIsSaving] = useState(false)
  const [isRunning, setIsRunning] = useState(false)
  const [pipelineName, setPipelineName] = useState('')
  const [pipelineDescription, setPipelineDescription] = useState('')
  const [pipelineStatus, setPipelineStatus] = useState<Pipeline['status']>('draft')
  const [editingDetails, setEditingDetails] = useState(false)
  const [showExecutionLog, setShowExecutionLog] = useState(false)
  const [highlightedNodes, setHighlightedNodes] = useState<Set<string>>(new Set())
  const [nodeStatuses, setNodeStatuses] = useState<Record<string, string>>({})
  const [nodeLogs, setNodeLogs] = useState<Record<string, NodeExecutionLogData>>({})
  const [activeNodeLogId, setActiveNodeLogId] = useState<string | null>(null)
  const [contextMenu, setContextMenu] = useState<EditorContextMenuState | null>(null)
  const [isBlockingOverlayOpen, setIsBlockingOverlayOpen] = useState(false)
  const nameInputRef = useRef<HTMLInputElement>(null)
  const selectedNodes = nodes.filter((node) => node.selected)
  const selectedNodeIds = selectedNodes.map((node) => node.id)

  const { data: pipeline, isLoading } = useQuery<Pipeline>({
    queryKey: ['pipeline', id],
    queryFn: () => api.pipelines.get(id!),
    enabled: !!id,
  })

  useEffect(() => {
    if (pipeline) {
      try {
        setPipelineName(pipeline.name)
        setPipelineDescription(pipeline.description || '')
        setPipelineStatus((pipeline.status as Pipeline['status']) || 'draft')
        const parsedNodes = JSON.parse(pipeline.nodes || '[]')
        const parsedEdges = JSON.parse(pipeline.edges || '[]')
        
        const flowNodes: Node[] = parsedNodes.map((n: any) => ({
          ...n,
          type: 'automator',
          data: {
            ...n.data,
            type: n.data?.type || 'trigger:manual',
            config: n.data?.config || {},
            label: n.data?.label || 'Node',
          },
        }))
        
        const flowEdges: Edge[] = parsedEdges.map((e: any) => ({
          ...e,
          ...(e.sourceHandle === 'tool' ? toolEdgeOptions : defaultEdgeOptions),
        }))

        setNodes(normalizeNodesForSubflows(flowNodes))
        setEdges(flowEdges)
      } catch (err) {
        console.error('Failed to parse pipeline data:', err)
      }
    }
  }, [pipeline])

  useEffect(() => {
    if (editingDetails && nameInputRef.current) {
      nameInputRef.current.focus()
      nameInputRef.current.select()
    }
  }, [editingDetails])

  const saveMutation = useMutation({
    mutationFn: async (nextStatus?: Pipeline['status']) => {
      if (!id) return
      const flowNodes = normalizeNodesForSubflows(nodes).map(({ type: _t, ...rest }) => rest)
      const flowEdges = edges.map(({ ...rest }) => rest)
      
      return api.pipelines.update(id, {
        name: pipelineName,
        description: pipelineDescription,
        nodes: JSON.stringify(flowNodes),
        edges: JSON.stringify(flowEdges),
        viewport: JSON.stringify(getViewport()),
        status: nextStatus || pipelineStatus,
      })
    },
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['pipeline', id] })
      queryClient.invalidateQueries({ queryKey: ['pipelines'] })
      if (result?.status) {
        setPipelineStatus(result.status as Pipeline['status'])
      }
      setEditingDetails(false)
      addToast({ type: 'success', title: 'Pipeline saved' })
      setIsSaving(false)
    },
    onError: (err) => {
      addToast({ type: 'error', title: 'Failed to save', message: err.message })
      setIsSaving(false)
    },
  })

  const runMutation = useMutation({
    mutationFn: () => api.pipelines.run(id!),
    onSuccess: (result: PipelineRunResponse) => {
      if (result.status === 'completed') {
        addToast({ type: 'success', title: 'Pipeline completed' })
      } else if (result.status === 'cancelled') {
        addToast({
          type: 'warning',
          title: 'Pipeline stopped',
          message: result.error || 'The execution was cancelled.',
        })
      } else {
        addToast({
          type: 'error',
          title: 'Pipeline failed',
          message: result.error || 'The execution finished with an error.',
        })
      }
      setIsRunning(false)
    },
    onError: (err) => {
      addToast({ type: 'error', title: 'Pipeline failed', message: err.message })
      setIsRunning(false)
    },
  })

  const handleSave = useCallback(() => {
    setIsSaving(true)
    saveMutation.mutate(undefined)
  }, [saveMutation])

  const handleToggleStatus = useCallback((nextStatus: Pipeline['status']) => {
    setIsSaving(true)
    saveMutation.mutate(nextStatus)
  }, [saveMutation])

  const handleRun = useCallback(() => {
    setShowExecutionLog(true)
    setIsRunning(true)
    runMutation.mutate()
  }, [runMutation])

  const handleCancelDetailsEdit = useCallback(() => {
    setPipelineName(pipeline?.name || '')
    setPipelineDescription(pipeline?.description || '')
    setEditingDetails(false)
  }, [pipeline])

  const handleExecutionHighlight = useCallback((data: {
    nodeIds: string[]
    nodeStatuses: Record<string, string>
    nodeErrors: Record<string, string | undefined>
    nodeLogs: Record<string, NodeExecutionLogData>
  } | null) => {
    if (!data) {
      setHighlightedNodes(new Set())
      setNodeStatuses({})
      setNodeLogs({})
      setActiveNodeLogId(null)
      return
    }
    setHighlightedNodes(new Set(data.nodeIds))
    setNodeStatuses(data.nodeStatuses)
    setNodeLogs(data.nodeLogs)
  }, [])

  const handleCloseExecutionLog = useCallback(() => {
    setShowExecutionLog(false)
    setHighlightedNodes(new Set())
    setNodeStatuses({})
    setNodeLogs({})
    setActiveNodeLogId(null)
  }, [])

  useEffect(() => {
    if (activeNodeLogId && !nodeLogs[activeNodeLogId]) {
      setActiveNodeLogId(null)
    }
  }, [activeNodeLogId, nodeLogs])

  const onConnect: OnConnect = useCallback(
    (params) => {
      const sourceNode = nodes.find((node) => node.id === params.source)
      const targetNode = nodes.find((node) => node.id === params.target)
      const sourceType = sourceNode?.data?.type
      const targetType = targetNode?.data?.type
      const isToolConnection = params.sourceHandle === 'tool'
      const sourceIsTool = isToolNodeType(sourceType)
      const targetIsTool = isToolNodeType(targetType)
      const sourceIsReturn = sourceType === 'logic:return'
      const targetIsTrigger = typeof targetType === 'string' && targetType.startsWith('trigger:')
      const sourceIsGroup = isVisualGroupNodeType(sourceType)
      const targetIsGroup = isVisualGroupNodeType(targetType)

      if (sourceIsGroup || targetIsGroup) {
        addToast({
          type: 'warning',
          title: 'Groups are visual only',
          message: 'Visual groups cannot have incoming or outgoing connections.',
        })
        return
      }

      if (isToolConnection) {
        if (sourceType !== 'llm:agent' || !targetIsTool) {
          addToast({
            type: 'warning',
            title: 'Invalid tool connection',
            message: 'Only an LLM Agent tool pin can connect to tool nodes.',
          })
          return
        }
      } else if (sourceIsReturn) {
        addToast({
          type: 'warning',
          title: 'Return nodes end the pipeline',
          message: 'Return nodes cannot connect to downstream nodes.',
        })
        return
      } else if (targetIsTrigger) {
        addToast({
          type: 'warning',
          title: 'Triggers cannot accept input',
          message: 'Trigger nodes can only start a pipeline and cannot have incoming connections.',
        })
        return
      } else if (sourceIsTool || targetIsTool) {
        addToast({
          type: 'warning',
          title: 'Tool nodes only connect to agents',
          message: 'Use the blue tool pin on an LLM Agent to connect tool nodes.',
        })
        return
      }

      setEdges((eds) => addEdge({
        ...params,
        ...(isToolConnection ? toolEdgeOptions : defaultEdgeOptions),
      }, eds))
    },
    [addToast, nodes, setEdges],
  )

  const createNodeAtPosition = useCallback((
    type: NodeType,
    label: string,
    config: Record<string, unknown>,
    clientPosition: { x: number; y: number },
  ) => {
    if (type === 'logic:return' && hasReturnNode(nodes)) {
      addToast({
        type: 'warning',
        title: 'Return node already exists',
        message: 'Each pipeline can only contain one Return node.',
      })
      return
    }

    const position = reactFlowWrapper.current
      ? screenToFlowPosition(clientPosition)
      : clientPosition

    const newNode: Node = {
      id: `${type}-${Date.now()}`,
      type: 'automator',
      position,
      data: {
        label,
        type,
        config,
        enabled: true,
      },
    }

    setNodes((nds) => nds.concat(newNode))
    setSelectedNodeId(newNode.id)
  }, [addToast, nodes, screenToFlowPosition, setNodes, setSelectedNodeId])

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault()

      const type = event.dataTransfer.getData('application/reactflow/type')
      const label = event.dataTransfer.getData('application/reactflow/label')
      const config = event.dataTransfer.getData('application/reactflow/config')

      if (!type || !reactFlowWrapper.current) return

      createNodeAtPosition(
        type as NodeType,
        label || 'Node',
        JSON.parse(config || '{}'),
        {
          x: event.clientX,
          y: event.clientY,
        },
      )
    },
    [createNodeAtPosition],
  )

  const onNodesChangeHandler: OnNodesChange = useCallback(
    (changes) => {
      setNodes((currentNodes) => {
        const nextNodes = applyNodeChanges(changes, currentNodes).map((node) => {
          if (!isGroupNode(node)) {
            return node
          }

          const dimensionChange = changes.find((change) => change.type === 'dimensions' && change.id === node.id)
          if (!dimensionChange || !dimensionChange.dimensions) {
            return node
          }

          const minDimensions = getMinimumGroupDimensions(node, currentNodes)
          const width = Math.max(dimensionChange.dimensions.width, minDimensions.width)
          const height = Math.max(dimensionChange.dimensions.height, minDimensions.height)

          return {
            ...node,
            width,
            height,
            style: {
              ...node.style,
              width,
              height,
            },
          }
        })
        const nextSelectedNodes = nextNodes.filter((node) => node.selected)

        if (nextSelectedNodes.length === 1) {
          setSelectedNodeId(nextSelectedNodes[0].id)
        } else {
          setSelectedNodeId(null)
        }

        return nextNodes
      })
      setContextMenu(null)
    },
    [setNodes, setSelectedNodeId],
  )

  const handleNodeDragStop = useCallback((_: React.MouseEvent, _draggedNode: Node, draggedNodes: Node[]) => {
    setNodes((currentNodes) => {
      const draggedNodesById = new Map(draggedNodes.map((node) => [node.id, node]))
      let nextNodes = currentNodes.map((node) => {
        const draggedSnapshot = draggedNodesById.get(node.id)
        if (!draggedSnapshot) {
          return node
        }

        return {
          ...node,
          position: { ...draggedSnapshot.position },
        }
      })

      draggedNodes.forEach((draggedSnapshot) => {
        const candidateNode = nextNodes.find((node) => node.id === draggedSnapshot.id)
        if (!candidateNode || isGroupNode(candidateNode)) {
          return
        }

        const nodesById = new Map(nextNodes.map((node) => [node.id, node]))
        const absolutePosition = getAbsoluteNodePosition(candidateNode, nodesById)
        const containingGroup = findContainingGroup(candidateNode, nextNodes)

        if (containingGroup) {
          if (candidateNode.parentId === containingGroup.id) {
            return
          }

          const groupAbsolutePosition = getAbsoluteNodePosition(containingGroup, nodesById)

          nextNodes = nextNodes.map((node) => {
            if (node.id !== candidateNode.id) {
              return node
            }

            return {
              ...node,
              position: {
                x: absolutePosition.x - groupAbsolutePosition.x,
                y: absolutePosition.y - groupAbsolutePosition.y,
              },
              parentId: containingGroup.id,
              extent: undefined,
            }
          })

          return
        }

        if (!candidateNode.parentId) {
          return
        }

        nextNodes = nextNodes.map((node) => {
          if (node.id !== candidateNode.id) {
            return node
          }

          return {
            ...node,
            position: absolutePosition,
            parentId: undefined,
            extent: undefined,
          }
        })
      })

      return normalizeNodesForSubflows(nextNodes)
    })
  }, [setNodes])

  const onDragStart = useCallback((event: React.DragEvent, nodeType: string, label: string, config: Record<string, unknown>) => {
    event.dataTransfer.setData('application/reactflow/type', nodeType)
    event.dataTransfer.setData('application/reactflow/label', label)
    event.dataTransfer.setData('application/reactflow/config', JSON.stringify(config))
    event.dataTransfer.effectAllowed = 'move'
  }, [])

  const buildPaneContextMenuItems = useCallback((clientX: number, clientY: number): ContextMenuItem[] => {
    const buildNodeItem = (
      nodeType: typeof NODE_CATEGORIES[number]['types'][number],
      contextLabel: string,
    ): ContextMenuItem => {
      const Icon = nodeMenuIconMap[nodeType.icon] || Zap

      return {
        label: nodeType.label,
        icon: <Icon className="w-3.5 h-3.5" style={{ color: nodeType.color }} />,
        searchText: `${nodeType.label} ${nodeType.description} ${contextLabel}`,
        onClick: () => createNodeAtPosition(
          nodeType.type,
          nodeType.label,
          { ...nodeType.defaultConfig },
          { x: clientX, y: clientY },
        ),
      }
    }

    const buildProviderGroup = (
      label: string,
      Icon: React.ElementType,
      color: string,
      nodeTypes: typeof NODE_CATEGORIES[number]['types'],
      categoryLabel: string,
    ): ContextMenuItem | null => {
      if (nodeTypes.length === 0) {
        return null
      }

      return {
        label,
        icon: <Icon className="w-3.5 h-3.5" style={{ color }} />,
        searchText: `${label} ${categoryLabel} add node`,
        children: nodeTypes.map((nodeType) => buildNodeItem(nodeType, `${categoryLabel} ${label}`)),
      }
    }

    const addNodeItems = NODE_CATEGORIES.map((category) => {
      const CategoryIcon = categoryMenuIconMap[category.id] || Zap

      if (category.id === 'action' || category.id === 'tool') {
        const generalTypes = category.types.filter((nodeType) => (
          !isProxmoxNodeType(nodeType.type) && !isKubernetesNodeType(nodeType.type)
        ))
        const proxmoxTypes = category.types.filter((nodeType) => isProxmoxNodeType(nodeType.type))
        const kubernetesTypes = category.types.filter((nodeType) => isKubernetesNodeType(nodeType.type))
        const providerGroups = [
          buildProviderGroup('General', Workflow, category.color, generalTypes, category.label),
          buildProviderGroup('Proxmox', Server, category.color, proxmoxTypes, category.label),
          buildProviderGroup('Kubernetes', Shield, category.color, kubernetesTypes, category.label),
        ].filter((item): item is ContextMenuItem => item !== null)

        return {
          label: category.label,
          icon: <CategoryIcon className="w-3.5 h-3.5" style={{ color: category.color }} />,
          searchText: `${category.label} add node`,
          children: providerGroups,
        }
      }

      return {
        label: category.label,
        icon: <CategoryIcon className="w-3.5 h-3.5" style={{ color: category.color }} />,
        searchText: `${category.label} add node`,
        children: category.types.map((nodeType) => buildNodeItem(nodeType, category.label)),
      }
    })

    return [
      ...addNodeItems,
      { divider: true, label: '' },
      {
        label: 'Save pipeline',
        icon: <Save className="w-3.5 h-3.5" />,
        shortcut: 'Ctrl+S',
        onClick: handleSave,
      },
      {
        label: 'Run pipeline',
        icon: <Play className="w-3.5 h-3.5" />,
        onClick: handleRun,
      },
    ]
  }, [createNodeAtPosition, handleRun, handleSave])

  const selectedNode = nodes.find((n) => n.id === selectedNodeId)
  const activeNodeLog = activeNodeLogId ? nodeLogs[activeNodeLogId] : null
  const activeNodeLogNode = activeNodeLogId ? nodes.find((node) => node.id === activeNodeLogId) : undefined

  const updateNodeConfig = useCallback((config: Record<string, unknown>) => {
    if (!selectedNodeId) return
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === selectedNodeId) {
          return {
            ...node,
            data: {
              ...node.data,
              config,
            },
          }
        }
        return node
      })
    )
  }, [selectedNodeId, setNodes])

  const updateNodeLabel = useCallback((label: string) => {
    if (!selectedNodeId) return
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === selectedNodeId) {
          return {
            ...node,
            data: {
              ...node.data,
              label,
            },
          }
        }
        return node
      })
    )
  }, [selectedNodeId, setNodes])

  const removeSelectedNodeSourceHandles = useCallback((handleIds: string[]) => {
    if (!selectedNodeId || handleIds.length === 0) return

    const removedHandles = new Set(handleIds)
    setEdges((eds) => eds.filter((edge) => (
      edge.source !== selectedNodeId
        || !edge.sourceHandle
        || !removedHandles.has(edge.sourceHandle)
    )))
  }, [selectedNodeId, setEdges])

  const removeNodesByIds = useCallback((nodeIds: string[]) => {
    if (nodeIds.length === 0) return

    const idsToRemove = new Set(nodeIds)

    setNodes((currentNodes) => {
      const removedGroups = new Map(
        currentNodes
          .filter((node) => idsToRemove.has(node.id) && isGroupNode(node))
          .map((node) => [node.id, { x: node.position.x, y: node.position.y }]),
      )

      return normalizeNodesForSubflows(currentNodes.flatMap((node) => {
        if (idsToRemove.has(node.id)) {
          return []
        }

        if (node.parentId && removedGroups.has(node.parentId)) {
          const parentPosition = removedGroups.get(node.parentId)!

          return [{
            ...node,
            position: {
              x: parentPosition.x + node.position.x,
              y: parentPosition.y + node.position.y,
            },
            parentId: undefined,
            extent: undefined,
            selected: false,
          }]
        }

        return [node]
      }))
    })
    setEdges((currentEdges) => currentEdges.filter((edge) => !idsToRemove.has(edge.source) && !idsToRemove.has(edge.target)))
    setSelectedNodeId(null)
    setContextMenu(null)
  }, [setEdges, setNodes, setSelectedNodeId])

  const ungroupNode = useCallback((groupId: string) => {
    setNodes((currentNodes) => {
      const groupNode = currentNodes.find((node) => node.id === groupId)
      if (!groupNode || !isGroupNode(groupNode)) {
        return currentNodes
      }

      return normalizeNodesForSubflows(currentNodes.flatMap((node) => {
        if (node.id === groupId) {
          return []
        }

        if (node.parentId === groupId) {
          return [{
            ...node,
            position: {
              x: groupNode.position.x + node.position.x,
              y: groupNode.position.y + node.position.y,
            },
            parentId: undefined,
            extent: undefined,
            selected: false,
          }]
        }

        return [node]
      }))
    })
    setSelectedNodeId(null)
    setContextMenu(null)
  }, [setNodes, setSelectedNodeId])

  const createGroupFromSelection = useCallback(() => {
    if (!canGroupNodes(selectedNodes)) {
      addToast({
        type: 'warning',
        title: 'Cannot group this selection',
        message: 'Groups can only be created from two or more top-level non-group nodes.',
      })
      return
    }

    const minX = Math.min(...selectedNodes.map((node) => node.position.x))
    const minY = Math.min(...selectedNodes.map((node) => node.position.y))
    const maxX = Math.max(...selectedNodes.map((node) => node.position.x + getNodeWidth(node)))
    const maxY = Math.max(...selectedNodes.map((node) => node.position.y + getNodeHeight(node)))

    const groupPosition = {
      x: minX - GROUP_PADDING_X,
      y: minY - GROUP_PADDING_TOP,
    }
    const groupWidth = Math.max((maxX - minX) + GROUP_PADDING_X * 2, MIN_GROUP_WIDTH)
    const groupHeight = Math.max((maxY - minY) + GROUP_PADDING_TOP + GROUP_PADDING_BOTTOM, MIN_GROUP_HEIGHT)
    const groupId = `visual:group-${Date.now()}`
    const selectedIds = new Set(selectedNodes.map((node) => node.id))
    const groupLabel = getNextGroupLabel(nodes)

    setNodes((currentNodes) => normalizeNodesForSubflows([
      {
        id: groupId,
        type: 'automator',
        position: groupPosition,
        style: {
          width: groupWidth,
          height: groupHeight,
        },
        width: groupWidth,
        height: groupHeight,
        selected: true,
        data: {
          label: groupLabel,
          type: VISUAL_GROUP_TYPE,
          config: { color: DEFAULT_GROUP_COLOR },
          enabled: true,
        },
      },
      ...currentNodes.map((node) => {
        if (!selectedIds.has(node.id)) {
          return { ...node, selected: false }
        }

        return {
          ...node,
          position: {
            x: node.position.x - groupPosition.x,
            y: node.position.y - groupPosition.y,
          },
          parentId: groupId,
          extent: undefined,
          selected: false,
        }
      }),
    ]))

    setSelectedNodeId(groupId)
    setContextMenu(null)
  }, [addToast, nodes, selectedNodes, setNodes, setSelectedNodeId])

  const buildSelectionContextMenuItems = useCallback((selection: Node[]): ContextMenuItem[] => {
    const groupable = canGroupNodes(selection)

    return [
      {
        label: 'Make group',
        icon: <Workflow className="w-3.5 h-3.5" />,
        disabled: !groupable,
        onClick: createGroupFromSelection,
      },
      {
        divider: true,
        label: '',
      },
      {
        label: 'Delete selected nodes',
        icon: <Trash2 className="w-3.5 h-3.5" />,
        shortcut: 'Del',
        danger: true,
        onClick: () => removeNodesByIds(selection.map((node) => node.id)),
      },
    ]
  }, [createGroupFromSelection, removeNodesByIds])

  const duplicateNode = useCallback(() => {
    if (!selectedNodeId) return
    const node = nodes.find((n) => n.id === selectedNodeId)
    if (!node) return
    if (isGroupNode(node)) {
      addToast({
        type: 'warning',
        title: 'Duplicate is not supported for groups',
        message: 'Create a new group from a node selection instead.',
      })
      return
    }
    if (node.data?.type === 'logic:return') {
      addToast({
        type: 'warning',
        title: 'Return node already exists',
        message: 'Each pipeline can only contain one Return node.',
      })
      return
    }

    const newNode: Node = {
      id: `${node.data.type}-${Date.now()}`,
      type: 'automator',
      position: { x: node.position.x + 40, y: node.position.y + 40 },
      data: { ...node.data },
    }

    setNodes((nds) => nds.concat(newNode))
    setSelectedNodeId(newNode.id)
  }, [addToast, selectedNodeId, nodes, setNodes, setSelectedNodeId])

  const deleteNode = useCallback(() => {
    if (!selectedNodeId) return
    removeNodesByIds([selectedNodeId])
  }, [removeNodesByIds, selectedNodeId])

  const disconnectNode = useCallback(() => {
    if (!selectedNodeId) return
    setEdges((eds) => eds.filter((e) => e.source !== selectedNodeId && e.target !== selectedNodeId))
  }, [selectedNodeId, setEdges])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (isBlockingOverlayOpen) {
        return
      }

      const isEditingField = isEditableTarget(e.target)
      if (isEditingField) {
        if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
          e.preventDefault()
          handleSave()
        }
        return
      }

      const activeSelectionIds = selectedNodeIds.length > 0
        ? selectedNodeIds
        : selectedNodeId
        ? [selectedNodeId]
        : []

      if ((e.key === 'Delete' || e.key === 'Backspace') && activeSelectionIds.length > 0) {
        e.preventDefault()
        removeNodesByIds(activeSelectionIds)
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'd' && activeSelectionIds.length === 1 && selectedNodeId) {
        e.preventDefault()
        duplicateNode()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'k' && activeSelectionIds.length === 1 && selectedNodeId) {
        e.preventDefault()
        disconnectNode()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        handleSave()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [selectedNodeId, selectedNodeIds, duplicateNode, disconnectNode, handleSave, isBlockingOverlayOpen, removeNodesByIds])

  const handleNodeContextMenu = useCallback((event: React.MouseEvent, node: Node) => {
    event.preventDefault()
    if (node.selected && selectedNodes.length > 1) {
      setSelectedNodeId(null)
      setContextMenu({
        x: event.clientX,
        y: event.clientY,
        items: buildSelectionContextMenuItems(selectedNodes),
      })
      return
    }

    if (isGroupNode(node)) {
      setSelectedNodeId(node.id)
      setContextMenu({
        x: event.clientX,
        y: event.clientY,
        items: [
          {
            label: 'Edit title',
            icon: <Edit2 className="w-3.5 h-3.5" />,
            onClick: () => setSelectedNodeId(node.id),
          },
          {
            label: 'Ungroup',
            icon: <Workflow className="w-3.5 h-3.5" />,
            onClick: () => ungroupNode(node.id),
          },
        ],
      })
      return
    }

    setSelectedNodeId(node.id)
    const items: ContextMenuItem[] = [
      {
        label: 'Edit',
        icon: <Edit2 className="w-3.5 h-3.5" />,
        onClick: () => setSelectedNodeId(node.id),
      },
      {
        label: 'Duplicate',
        icon: <Copy className="w-3.5 h-3.5" />,
        shortcut: 'Ctrl+D',
        onClick: () => {
          if (node.data?.type === 'logic:return') {
            addToast({
              type: 'warning',
              title: 'Return node already exists',
              message: 'Each pipeline can only contain one Return node.',
            })
            return
          }
          const newNode: Node = {
            id: `${node.data.type}-${Date.now()}`,
            type: 'automator',
            position: { x: node.position.x + 40, y: node.position.y + 40 },
            data: { ...node.data },
          }
          setNodes((nds) => nds.concat(newNode))
          setSelectedNodeId(newNode.id)
        },
      },
      {
        label: 'Disconnect',
        icon: <Scissors className="w-3.5 h-3.5" />,
        shortcut: 'Ctrl+K',
        onClick: () => {
          setEdges((eds) => eds.filter((e) => e.source !== node.id && e.target !== node.id))
        },
      },
      {
        divider: true,
        label: '',
        onClick: () => {},
      },
      {
        label: 'Delete',
        icon: <Trash2 className="w-3.5 h-3.5" />,
        shortcut: 'Del',
        danger: true,
        onClick: () => removeNodesByIds([node.id]),
      },
    ]
    setContextMenu({ x: event.clientX, y: event.clientY, items })
  }, [addToast, buildSelectionContextMenuItems, removeNodesByIds, selectedNodes, setSelectedNodeId, setNodes, setEdges, ungroupNode])

  const handleEdgeContextMenu = useCallback((event: React.MouseEvent, edge: Edge) => {
    event.preventDefault()
    const items: ContextMenuItem[] = [
      {
        label: 'Delete connection',
        icon: <Scissors className="w-3.5 h-3.5" />,
        danger: true,
        onClick: () => {
          setEdges((eds) => eds.filter((e) => e.id !== edge.id))
        },
      },
    ]
    setContextMenu({ x: event.clientX, y: event.clientY, items })
  }, [setEdges])

  const handleSelectionContextMenu = useCallback((event: MouseEvent, selection: Node[]) => {
    event.preventDefault()
    setSelectedNodeId(null)
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      items: buildSelectionContextMenuItems(selection),
    })
  }, [buildSelectionContextMenuItems, setSelectedNodeId])

  const handlePaneContextMenu = useCallback((event: React.MouseEvent) => {
    event.preventDefault()

    setSelectedNodeId(null)
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      items: buildPaneContextMenuItems(event.clientX, event.clientY),
      searchable: true,
      searchPlaceholder: 'Search nodes by name...',
      emptyMessage: 'No nodes match your search.',
    })
  }, [buildPaneContextMenuItems, setSelectedNodeId])

  const renderedNodes = nodes.map((node) => {
    const isHighlightMode = highlightedNodes.size > 0
    const isHighlighted = highlightedNodes.has(node.id)
    const execStatus = nodeStatuses[node.id]
    const visualStatus = getVisualExecutionStatus(execStatus)

    if (!isHighlightMode) {
      return node
    }

    return {
      ...node,
      data: {
        ...node.data,
        status: visualStatus,
        enabled: isHighlighted,
        isHighlight: isHighlighted,
        executionLog: nodeLogs[node.id],
        canViewLog: isHighlighted && !!nodeLogs[node.id],
        onViewLog: () => setActiveNodeLogId(node.id),
      },
    }
  })

  const renderedNodeById = new Map(renderedNodes.map((node) => [node.id, node]))
  const activeSelectionIds = new Set(
    selectedNodeIds.length > 0
      ? selectedNodeIds
      : selectedNodeId
      ? [selectedNodeId]
      : [],
  )

  const renderedEdges = edges.map((edge) => {
    const isHighlightMode = highlightedNodes.size > 0
    const isHighlighted = highlightedNodes.has(edge.source) && highlightedNodes.has(edge.target)
    const baseEdgeOptions = edge.sourceHandle === 'tool' ? toolEdgeOptions : defaultEdgeOptions

    if (isHighlightMode) {
      return {
        ...edge,
        style: {
          ...baseEdgeOptions.style,
          strokeDasharray: isHighlighted ? 'none' : '6 4',
          stroke: isHighlighted ? '#f59e0b' : baseEdgeOptions.style.stroke,
          strokeWidth: isHighlighted ? 2.5 : 1.5,
          opacity: isHighlighted ? 1 : 0.3,
        },
        animated: isHighlighted,
        markerEnd: {
          ...baseEdgeOptions.markerEnd,
          color: isHighlighted ? '#f59e0b' : baseEdgeOptions.markerEnd.color,
        },
        data: {
          ...(typeof edge.data === 'object' && edge.data !== null ? edge.data : {}),
          useGradient: false,
        },
      }
    }

    const isConnectedToSelection = activeSelectionIds.has(edge.source) || activeSelectionIds.has(edge.target)
    const connectedSelectionCount = Number(activeSelectionIds.has(edge.source)) + Number(activeSelectionIds.has(edge.target))
    const edgeStrokeWidth = isConnectedToSelection
      ? connectedSelectionCount === 2 ? 2.9 : 2.6
      : (edge.style?.strokeWidth ?? baseEdgeOptions.style.strokeWidth)
    const sourceNode = renderedNodeById.get(edge.source)
    const targetNode = renderedNodeById.get(edge.target)
    const sourceBorderColor = getRenderedNodeBorderColor(sourceNode, activeSelectionIds)
    const targetBorderColor = getRenderedNodeBorderColor(targetNode, activeSelectionIds)

    return {
      ...edge,
      style: {
        ...baseEdgeOptions.style,
        ...edge.style,
        strokeWidth: edgeStrokeWidth,
      },
      markerEnd: {
        ...baseEdgeOptions.markerEnd,
        color: isConnectedToSelection ? targetBorderColor : baseEdgeOptions.markerEnd.color,
      },
      data: {
        ...(typeof edge.data === 'object' && edge.data !== null ? edge.data : {}),
        useGradient: isConnectedToSelection,
        gradientStartColor: sourceBorderColor,
        gradientEndColor: targetBorderColor,
      },
    }
  })

  if (isLoading) {
    return (
      <div className="h-screen flex items-center justify-center bg-bg">
        <Loader2 className="w-8 h-8 text-accent animate-spin" />
      </div>
    )
  }

  return (
    <div className="flex h-screen bg-bg">
      {/* Left: Node Palette */}
      <NodePalette onDragStart={onDragStart} />

      {/* Center: Canvas */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 bg-bg-elevated border-b border-border">
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" onClick={() => navigate('/pipelines')}>
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <div className="min-w-0">
              {editingDetails ? (
                <div className="min-w-[320px] max-w-[520px] space-y-2">
                  <Input
                    ref={nameInputRef}
                    value={pipelineName}
                    onChange={(e) => setPipelineName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Escape') {
                        handleCancelDetailsEdit()
                      }
                    }}
                    placeholder="Pipeline name"
                    className="text-base font-semibold"
                  />
                  <Textarea
                    value={pipelineDescription}
                    onChange={(e) => setPipelineDescription(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Escape') {
                        handleCancelDetailsEdit()
                      }
                    }}
                    placeholder="Add a short description"
                    rows={2}
                    className="text-sm"
                  />
                  <div className="flex items-center gap-2">
                    <Button variant="secondary" size="sm" onClick={() => setEditingDetails(false)}>
                      Done
                    </Button>
                    <Button variant="ghost" size="sm" onClick={handleCancelDetailsEdit}>
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <h1
                      className="truncate text-lg font-semibold text-text cursor-pointer hover:text-accent transition-colors px-2 py-0.5 rounded hover:bg-bg-overlay"
                      onClick={() => setEditingDetails(true)}
                      title="Click to edit details"
                    >
                      {pipelineName || 'Untitled'}
                    </h1>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setEditingDetails(true)}
                      title="Edit pipeline details"
                    >
                      <Edit2 className="w-4 h-4" />
                    </Button>
                  </div>
                  <p
                    className="truncate px-2 text-xs text-text-muted cursor-pointer hover:text-text transition-colors"
                    onClick={() => setEditingDetails(true)}
                    title="Click to edit description"
                  >
                    {pipelineDescription.trim() || 'Add a pipeline description'}
                  </p>
                </div>
              )}
            </div>
            <Badge variant={pipelineStatus === 'active' ? 'success' : pipelineStatus === 'draft' ? 'warning' : 'default'}>
              {pipelineStatus}
            </Badge>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              loading={isSaving}
              onClick={() => handleToggleStatus(pipelineStatus === 'active' ? 'draft' : 'active')}
            >
              <Power className="w-4 h-4" />
              {pipelineStatus === 'active' ? 'Deactivate' : 'Activate'}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                if (showExecutionLog) {
                  handleCloseExecutionLog()
                  return
                }
                setShowExecutionLog(true)
              }}
              className={showExecutionLog ? 'text-accent border-accent/50' : ''}
            >
              <ListChecks className="w-4 h-4" />
              Log
            </Button>
            <Button
              variant="secondary"
              size="sm"
              loading={isSaving}
              onClick={handleSave}
            >
              <Save className="w-4 h-4" />
              Save
            </Button>
            <Button
              size="sm"
              loading={isRunning}
              onClick={handleRun}
            >
              <Play className="w-4 h-4" />
              Run
            </Button>
          </div>
        </div>

        {/* Canvas */}
        <div ref={reactFlowWrapper} className="flex-1">
          <ReactFlow
            nodes={renderedNodes}
            edges={renderedEdges}
            onNodesChange={onNodesChangeHandler}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onDrop={onDrop}
            onDragOver={onDragOver}
            onNodeDragStop={handleNodeDragStop}
            onNodeClick={(_, node) => {
              setSelectedNodeId(node.id)
              setContextMenu(null)
            }}
            onNodeContextMenu={handleNodeContextMenu}
            onSelectionContextMenu={handleSelectionContextMenu}
            onEdgeContextMenu={handleEdgeContextMenu}
            onPaneContextMenu={handlePaneContextMenu}
            onPaneClick={() => {
              setSelectedNodeId(null)
              setContextMenu(null)
            }}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            defaultEdgeOptions={defaultEdgeOptions}
            panActivationKeyCode={isBlockingOverlayOpen ? null : 'Space'}
            deleteKeyCode={isBlockingOverlayOpen ? null : 'Backspace'}
            selectionKeyCode={isBlockingOverlayOpen ? null : 'Shift'}
            multiSelectionKeyCode={isBlockingOverlayOpen ? null : undefined}
            zoomActivationKeyCode={isBlockingOverlayOpen ? null : undefined}
            disableKeyboardA11y={isBlockingOverlayOpen}
            fitView
            fitViewOptions={{ padding: 0.2 }}
            minZoom={0.1}
            maxZoom={4}
            snapToGrid
            snapGrid={[15, 15]}
          >
            <Controls position="bottom-left" />
            <Background color="#1e2d3d" gap={20} size={1} />
            
            <Panel position="bottom-right" className="!m-4">
              <div className="bg-bg-elevated border border-border rounded-lg shadow-lg p-2 flex gap-1">
                {selectedNodeId && (
                  <>
                    {selectedNode?.data?.type !== VISUAL_GROUP_TYPE && (
                      <Button variant="ghost" size="sm" onClick={duplicateNode} title="Duplicate">
                        <Copy className="w-4 h-4" />
                      </Button>
                    )}
                    <Button variant="ghost" size="sm" onClick={deleteNode} title="Delete">
                      <Trash2 className="w-4 h-4 text-red-400" />
                    </Button>
                  </>
                )}
              </div>
            </Panel>
          </ReactFlow>
        </div>
      </div>

      {/* Right: Config Panel */}
      {selectedNode && !showExecutionLog && (
        <NodeConfigPanel
          pipelineId={id!}
          nodes={nodes}
          edges={edges}
          nodeId={selectedNode.id}
          nodeType={selectedNode.data.type as NodeType}
          nodeLabel={selectedNode.data.label || 'Node'}
          config={selectedNode.data.config || {}}
          onUpdate={updateNodeConfig}
          onLabelChange={updateNodeLabel}
          onRemoveSourceHandles={removeSelectedNodeSourceHandles}
          onOverlayOpenChange={setIsBlockingOverlayOpen}
          onClose={() => setSelectedNodeId(null)}
        />
      )}

      {/* Right: Execution Log */}
      {showExecutionLog && (
        <ExecutionLog
          pipelineId={id!}
          isOpen={showExecutionLog}
          onClose={handleCloseExecutionLog}
          onExecutionSelect={handleExecutionHighlight}
        />
      )}

      {activeNodeLog && activeNodeLogNode && (
        <NodeExecutionModal
          nodeId={activeNodeLogId!}
          nodeLabel={activeNodeLogNode.data.label || activeNodeLogNode.id}
          nodeType={activeNodeLogNode.data.type as NodeType}
          log={activeNodeLog}
          onClose={() => setActiveNodeLogId(null)}
        />
      )}

      {/* Context Menu */}
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={contextMenu.items}
          searchable={contextMenu.searchable}
          searchPlaceholder={contextMenu.searchPlaceholder}
          emptyMessage={contextMenu.emptyMessage}
          onClose={() => setContextMenu(null)}
        />
      )}
    </div>
  )
}

function PipelineEditorWithProvider() {
  return (
    <ReactFlowProvider>
      <PipelineEditor />
    </ReactFlowProvider>
  )
}

export default PipelineEditorWithProvider
