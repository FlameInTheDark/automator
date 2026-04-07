import { useCallback, useRef, useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  ReactFlow,
  ReactFlowProvider,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
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
import { Save, Play, ArrowLeft, Loader2, Trash2, Copy, ListChecks, Scissors, Edit2, GitBranch, Zap, Clock, Webhook, Square, Globe, Code, Split, Brain, Link, MessageSquare, Send, Power, Bot, Workflow, List, Wrench, CornerDownLeft } from 'lucide-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

import NodePalette from '../components/flow/NodePalette'
import NodeConfigPanel from '../components/flow/NodeConfigPanel'
import ExecutionLog from '../components/flow/ExecutionLog'
import NodeExecutionModal from '../components/flow/NodeExecutionModal'
import AutomatorNode from '../components/flow/nodes/AutomatorNode'
import { NODE_CATEGORIES } from '../components/flow/nodeTypes'
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

const defaultEdgeOptions = {
  type: 'default',
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
  type: 'default',
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

type EditorContextMenuState = {
  x: number
  y: number
  items: ContextMenuItem[]
  searchable?: boolean
  searchPlaceholder?: string
  emptyMessage?: string
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

function PipelineEditor() {
  const { id } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const reactFlowWrapper = useRef<HTMLDivElement>(null)
  const { screenToFlowPosition, getViewport } = useReactFlow()
  const { selectedNodeId, setSelectedNodeId, addToast } = useUIStore()

  const [nodes, setNodes, onNodesChange] = useNodesState([])
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

        setNodes(flowNodes)
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
      const flowNodes = nodes.map(({ type: _t, ...rest }) => rest)
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
      onNodesChange(changes)
      changes.forEach((change) => {
        if (change.type === 'remove') {
          const node = nodes.find((n) => n.id === change.id)
          if (node?.id === selectedNodeId) {
            setSelectedNodeId(null)
          }
        }
      })
    },
    [onNodesChange, nodes, selectedNodeId, setSelectedNodeId],
  )

  const onDragStart = useCallback((event: React.DragEvent, nodeType: string, label: string, config: Record<string, unknown>) => {
    event.dataTransfer.setData('application/reactflow/type', nodeType)
    event.dataTransfer.setData('application/reactflow/label', label)
    event.dataTransfer.setData('application/reactflow/config', JSON.stringify(config))
    event.dataTransfer.effectAllowed = 'move'
  }, [])

  const buildPaneContextMenuItems = useCallback((clientX: number, clientY: number): ContextMenuItem[] => {
    const addNodeItems = NODE_CATEGORIES.map((category) => {
      const CategoryIcon = categoryMenuIconMap[category.id] || Zap

      return {
        label: category.label,
        icon: <CategoryIcon className="w-3.5 h-3.5" style={{ color: category.color }} />,
        searchText: `${category.label} add node`,
        children: category.types.map((nodeType) => {
          const Icon = nodeMenuIconMap[nodeType.icon] || Zap

          return {
            label: nodeType.label,
            icon: <Icon className="w-3.5 h-3.5" style={{ color: nodeType.color }} />,
            searchText: `${nodeType.label} ${nodeType.description} ${category.label}`,
            onClick: () => createNodeAtPosition(
              nodeType.type,
              nodeType.label,
              { ...nodeType.defaultConfig },
              { x: clientX, y: clientY },
            ),
          }
        }),
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

  const duplicateNode = useCallback(() => {
    if (!selectedNodeId) return
    const node = nodes.find((n) => n.id === selectedNodeId)
    if (!node) return
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
    setNodes((nds) => nds.filter((n) => n.id !== selectedNodeId))
    setEdges((eds) => eds.filter((e) => e.source !== selectedNodeId && e.target !== selectedNodeId))
    setSelectedNodeId(null)
  }, [selectedNodeId, setNodes, setEdges, setSelectedNodeId])

  const disconnectNode = useCallback(() => {
    if (!selectedNodeId) return
    setEdges((eds) => eds.filter((e) => e.source !== selectedNodeId && e.target !== selectedNodeId))
  }, [selectedNodeId, setEdges])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (isBlockingOverlayOpen) {
        return
      }

      if (e.key === 'Delete' && selectedNodeId) {
        setNodes((nds) => nds.filter((n) => n.id !== selectedNodeId))
        setEdges((eds) => eds.filter((edge) => edge.source !== selectedNodeId && edge.target !== selectedNodeId))
        setSelectedNodeId(null)
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'd' && selectedNodeId) {
        e.preventDefault()
        duplicateNode()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'k' && selectedNodeId) {
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
  }, [selectedNodeId, duplicateNode, disconnectNode, handleSave, isBlockingOverlayOpen, setNodes, setEdges, setSelectedNodeId])

  const handleNodeContextMenu = useCallback((event: React.MouseEvent, node: Node) => {
    event.preventDefault()
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
        onClick: () => {
          setNodes((nds) => nds.filter((n) => n.id !== node.id))
          setEdges((eds) => eds.filter((e) => e.source !== node.id && e.target !== node.id))
          setSelectedNodeId(null)
        },
      },
    ]
    setContextMenu({ x: event.clientX, y: event.clientY, items })
  }, [addToast, setSelectedNodeId, setNodes, setEdges])

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
            nodes={nodes.map((n) => {
              const isHighlightMode = highlightedNodes.size > 0
              const isHighlighted = highlightedNodes.has(n.id)
              const execStatus = nodeStatuses[n.id]
              const visualStatus = getVisualExecutionStatus(execStatus)

              if (!isHighlightMode) return n

              return {
                ...n,
                data: {
                  ...n.data,
                  status: visualStatus,
                  enabled: isHighlighted,
                  isHighlight: isHighlighted,
                  executionLog: nodeLogs[n.id],
                  canViewLog: isHighlighted && !!nodeLogs[n.id],
                  onViewLog: () => setActiveNodeLogId(n.id),
                },
              }
            })}
            edges={edges.map((e) => {
              const isHighlightMode = highlightedNodes.size > 0
              const isHighlighted = highlightedNodes.has(e.source) && highlightedNodes.has(e.target)
              const baseEdgeOptions = e.sourceHandle === 'tool' ? toolEdgeOptions : defaultEdgeOptions
              if (!isHighlightMode) return e
              return {
                ...e,
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
              }
            })}
            onNodesChange={onNodesChangeHandler}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onDrop={onDrop}
            onDragOver={onDragOver}
            onNodeClick={(_, node) => setSelectedNodeId(node.id)}
            onNodeContextMenu={handleNodeContextMenu}
            onEdgeContextMenu={handleEdgeContextMenu}
            onPaneContextMenu={handlePaneContextMenu}
            onPaneClick={() => {
              setSelectedNodeId(null)
              setContextMenu(null)
            }}
            nodeTypes={nodeTypes}
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
                    <Button variant="ghost" size="sm" onClick={duplicateNode} title="Duplicate">
                      <Copy className="w-4 h-4" />
                    </Button>
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
