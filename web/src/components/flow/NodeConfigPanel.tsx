import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { Edge, Node } from '@xyflow/react'
import {
  X, Settings, Play, Square, Copy, Globe, Code, Zap, Clock, Webhook,
  GitBranch, Split, Brain, Link, Plus, Trash2, MessageSquare, Send,
  Bot, Workflow, List, Wrench, CornerDownLeft, CircleHelp,
} from 'lucide-react'
import { cn } from '../../lib/utils'
import { NODE_TYPE_MAP, getNodeColor, getNodeLabel } from './nodeTypes'
import type { ExecutionDetail, LLMModelInfo, NodeType, Pipeline, TemplateSuggestion } from '../../types'
import { api } from '../../api/client'
import Input from '../ui/Input'
import { Checkbox, Label } from '../ui/Form'
import Select from '../ui/Select'
import Button from '../ui/Button'
import { TemplateInput, TemplateTextarea } from '../ui/TemplateFields'
import { buildTemplateSuggestions } from '../../lib/templates'
import LuaEditorModal from './LuaEditorModal'

const iconMap: Record<string, React.ElementType> = {
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

const proxmoxNodeTypes = new Set<NodeType>([
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
])

const channelNodeTypes = new Set<NodeType>([
  'trigger:channel_message',
  'action:channel_send_message',
  'action:channel_send_and_wait',
  'tool:channel_send_and_wait',
])

const pipelineMutationToolNodeTypes = new Set<NodeType>([
  'tool:pipeline_create',
  'tool:pipeline_update',
  'tool:pipeline_delete',
])

const EXPR_LANGUAGE_DOCS_URL = 'https://expr-lang.org/docs/language-definition'

interface NodeConfigPanelProps {
  pipelineId: string
  nodes: Node[]
  edges: Edge[]
  nodeId: string
  nodeType: NodeType
  nodeLabel: string
  config: Record<string, unknown>
  onUpdate: (config: Record<string, unknown>) => void
  onLabelChange: (label: string) => void
  onRemoveSourceHandles?: (handleIds: string[]) => void
  onOverlayOpenChange?: (open: boolean) => void
  onClose: () => void
}

type SwitchConditionConfig = {
  id: string
  label: string
  expression: string
}

function normalizeSwitchConditions(value: unknown): SwitchConditionConfig[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value.map((condition, index) => {
    const record = typeof condition === 'object' && condition !== null
      ? condition as Record<string, unknown>
      : {}

    return {
      id: typeof record.id === 'string' && record.id.trim()
        ? record.id.trim()
        : `condition-${index + 1}`,
      label: typeof record.label === 'string' && record.label.trim()
        ? record.label.trim()
        : `Condition ${index + 1}`,
      expression: typeof record.expression === 'string' ? record.expression : '',
    }
  })
}

function createSwitchCondition(existing: SwitchConditionConfig[]): SwitchConditionConfig {
  const takenIds = new Set(existing.map((condition) => condition.id))
  let index = existing.length + 1
  let id = `condition-${index}`

  while (takenIds.has(id)) {
    index += 1
    id = `condition-${index}`
  }

  return {
    id,
    label: `Condition ${index}`,
    expression: '',
  }
}

function ExpressionLabel() {
  return (
    <div className="mb-1.5 flex items-center gap-2">
      <Label className="mb-0">Expression</Label>
      <a
        href={EXPR_LANGUAGE_DOCS_URL}
        target="_blank"
        rel="noreferrer"
        className="inline-flex h-5 w-5 items-center justify-center rounded-full border border-border bg-bg-overlay text-text-muted transition-colors hover:border-accent/50 hover:text-accent"
        title="Open expression language documentation"
        aria-label="Open expression language documentation"
      >
        <CircleHelp className="h-3.5 w-3.5" />
      </a>
    </div>
  )
}

export default function NodeConfigPanel({
  pipelineId,
  nodes,
  edges,
  nodeId,
  nodeType,
  nodeLabel,
  config,
  onUpdate,
  onLabelChange,
  onRemoveSourceHandles,
  onOverlayOpenChange,
  onClose,
}: NodeConfigPanelProps) {
  const [activeTab, setActiveTab] = useState<'general' | 'config'>('general')
  const [label, setLabel] = useState(nodeLabel)
  const [localConfig, setLocalConfig] = useState(config)
  const [isLuaEditorOpen, setIsLuaEditorOpen] = useState(false)
  const { data: clusters } = useQuery({
    queryKey: ['clusters'],
    queryFn: () => api.clusters.list(),
  })
  const { data: llmProviders } = useQuery({
    queryKey: ['llm-providers'],
    queryFn: () => api.llmProviders.list(),
  })
  const { data: channels } = useQuery({
    queryKey: ['channels'],
    queryFn: () => api.channels.list(),
  })
  const { data: pipelines } = useQuery<Pipeline[]>({
    queryKey: ['pipelines'],
    queryFn: () => api.pipelines.list(),
  })
  const { data: executions } = useQuery({
    queryKey: ['executions', pipelineId],
    queryFn: () => api.executions.listByPipeline(pipelineId),
    enabled: !!pipelineId,
  })

  const latestExecutionId = executions?.[0]?.id
  const { data: latestExecutionDetail } = useQuery<ExecutionDetail>({
    queryKey: ['execution', latestExecutionId],
    queryFn: () => api.executions.get(latestExecutionId!),
    enabled: !!latestExecutionId,
  })

  const defaultProvider = useMemo(
    () => llmProviders?.find((provider) => provider.is_default),
    [llmProviders],
  )
  const selectedProviderId = ((localConfig.providerId as string) || defaultProvider?.id || '').trim()
  const { data: providerModels } = useQuery<LLMModelInfo[]>({
    queryKey: ['llm-provider-models', selectedProviderId],
    queryFn: () => api.llmProviders.models(selectedProviderId),
    enabled: (nodeType === 'llm:prompt' || nodeType === 'llm:agent') && !!selectedProviderId,
  })

  useEffect(() => {
    setLabel(nodeLabel)
    setLocalConfig(config)
    setIsLuaEditorOpen(false)
  }, [nodeId, nodeLabel, config])

  useEffect(() => {
    onOverlayOpenChange?.(isLuaEditorOpen)

    return () => {
      onOverlayOpenChange?.(false)
    }
  }, [isLuaEditorOpen, onOverlayOpenChange])

  const handleLabelBlur = () => {
    onLabelChange(label)
  }

  const handleConfigChange = (key: string, value: unknown) => {
    const newConfig = { ...localConfig, [key]: value }
    setLocalConfig(newConfig)
    onUpdate(newConfig)
  }

  const nodeDef = NODE_TYPE_MAP[nodeType]
  const Icon = iconMap[nodeDef?.icon || 'zap']
  const color = getNodeColor(nodeType)
  const showClusterSelect = proxmoxNodeTypes.has(nodeType)
  const showChannelSelect = channelNodeTypes.has(nodeType)
  const templateSuggestions = useMemo<TemplateSuggestion[]>(() => (
    buildTemplateSuggestions(nodeId, nodes, edges, latestExecutionDetail?.node_executions ?? [])
  ), [nodeId, nodes, edges, latestExecutionDetail])
  const agentTemplateSuggestions = useMemo<TemplateSuggestion[]>(() => {
    if (nodeType !== 'llm:agent' || !Boolean(localConfig.enableSkills)) {
      return templateSuggestions
    }

    return [
      {
        expression: 'skills',
        template: '{{skills}}',
        label: 'Available skills',
        description: 'Current local skills list with names and descriptions.',
      },
      ...templateSuggestions,
    ]
  }, [localConfig.enableSkills, nodeType, templateSuggestions])
  const switchConditions = useMemo(
    () => normalizeSwitchConditions(localConfig.conditions),
    [localConfig.conditions],
  )
  const modelOptions = useMemo(() => {
    const currentModel = (localConfig.model as string) || ''
    const options = [...(providerModels || [])]
    if (currentModel && !options.some((model) => model.id === currentModel)) {
      options.unshift({ id: currentModel, name: `${currentModel} (current)` })
    }
    return options
  }, [localConfig.model, providerModels])
  const availablePipelines = useMemo(
    () => (pipelines || []).filter((pipeline) => pipeline.id !== pipelineId),
    [pipelineId, pipelines],
  )
  const isToolNode = nodeType.startsWith('tool:')
  const connectedToolCount = useMemo(
    () => edges.filter((edge) => edge.source === nodeId && edge.sourceHandle === 'tool').length,
    [edges, nodeId],
  )

  const handleSwitchConditionChange = (conditionId: string, key: keyof SwitchConditionConfig, value: string) => {
    handleConfigChange(
      'conditions',
      switchConditions.map((condition) => (
        condition.id === conditionId
          ? { ...condition, [key]: value }
          : condition
      )),
    )
  }

  const handleSwitchConditionAdd = () => {
    handleConfigChange('conditions', [...switchConditions, createSwitchCondition(switchConditions)])
  }

  const handleSwitchConditionRemove = (conditionId: string) => {
    handleConfigChange(
      'conditions',
      switchConditions.filter((condition) => condition.id !== conditionId),
    )
    onRemoveSourceHandles?.([conditionId])
  }

  return (
    <div className="w-80 bg-bg-elevated border-l border-border flex flex-col h-full">
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ backgroundColor: `${color}20` }}>
            <Icon className="w-4 h-4" style={{ color }} />
          </div>
          <div>
            <p className="text-sm font-medium text-text">{nodeLabel}</p>
            <p className="text-xs text-text-dimmed">{nodeType}</p>
          </div>
        </div>
        <button onClick={onClose} className="text-text-dimmed hover:text-text transition-colors">
          <X className="w-4 h-4" />
        </button>
      </div>

      <div className="flex border-b border-border">
        <button
          onClick={() => setActiveTab('general')}
          className={cn(
            'flex-1 px-4 py-2.5 text-sm font-medium transition-colors border-b-2',
            activeTab === 'general'
              ? 'text-accent border-accent'
              : 'text-text-muted border-transparent hover:text-text',
          )}
        >
          General
        </button>
        <button
          onClick={() => setActiveTab('config')}
          className={cn(
            'flex-1 px-4 py-2.5 text-sm font-medium transition-colors border-b-2',
            activeTab === 'config'
              ? 'text-accent border-accent'
              : 'text-text-muted border-transparent hover:text-text',
          )}
        >
          Configuration
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        {activeTab === 'general' && (
          <div className="space-y-4">
            <div>
              <Label>Node Label</Label>
              <Input
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                onBlur={handleLabelBlur}
                placeholder="Enter node label"
              />
            </div>
            <div>
              <Label>Node ID</Label>
              <div className="px-3 py-2 bg-bg-input border border-border rounded-lg text-xs text-text-dimmed font-mono">
                {nodeId}
              </div>
            </div>
            <div>
              <Label>Type</Label>
              <div className="px-3 py-2 bg-bg-input border border-border rounded-lg text-sm text-text">
                {getNodeLabel(nodeType)}
              </div>
            </div>
            <div>
              <Label>Description</Label>
              <p className="text-sm text-text-muted">{nodeDef?.description}</p>
            </div>
          </div>
        )}

        {activeTab === 'config' && (
          <div className="space-y-4">
            {showClusterSelect && (
              <div>
                <Label>Cluster</Label>
                <Select
                  value={(localConfig.clusterId as string) || ''}
                  onChange={(e) => handleConfigChange('clusterId', e.target.value)}
                >
                  <option value="">Select cluster</option>
                  {clusters?.map((cluster) => (
                    <option key={cluster.id} value={cluster.id}>
                      {cluster.name}
                    </option>
                  ))}
                </Select>
              </div>
            )}

            {showChannelSelect && (
              <div>
                <Label>Channel</Label>
                <Select
                  value={(localConfig.channelId as string) || ''}
                  onChange={(e) => handleConfigChange('channelId', e.target.value)}
                >
                  <option value="">Select channel</option>
                  {channels?.map((channel) => (
                    <option key={channel.id} value={channel.id}>
                      {channel.name} ({channel.type})
                    </option>
                  ))}
                </Select>
              </div>
            )}

            {nodeType === 'action:proxmox_list_nodes' || nodeType === 'tool:proxmox_list_nodes' ? (
              <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                {nodeType === 'tool:proxmox_list_nodes'
                  ? 'This tool lists all nodes in the selected cluster when the agent calls it.'
                  : 'This node lists all nodes in the selected cluster.'}
              </div>
            ) : nodeType === 'trigger:channel_message' ? (
              <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                Connected messages from the selected channel will trigger this pipeline when the pipeline is active.
              </div>
            ) : nodeType === 'action:proxmox_list_workloads' || nodeType === 'tool:proxmox_list_workloads' ? (
              <>
                <div>
                  <Label>{nodeType === 'tool:proxmox_list_workloads' ? 'Default Node Filter' : 'Node Filter'}</Label>
                  <TemplateInput
                    value={(localConfig.node as string) || ''}
                    onChange={(e) => handleConfigChange('node', e.target.value)}
                    placeholder="Optional node name, e.g., pve1"
                    suggestions={templateSuggestions}
                  />
                </div>
                {nodeType === 'tool:proxmox_list_workloads' && (
                  <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                    The agent can optionally pass a <span className="font-mono text-text">node</span> argument to override this default filter.
                  </div>
                )}
              </>
            ) : nodeType === 'action:vm_start' || nodeType === 'action:vm_stop' || nodeType === 'tool:vm_start' || nodeType === 'tool:vm_stop' ? (
              <>
                <div>
                  <Label>{isToolNode ? 'Default Proxmox Node' : 'Proxmox Node'}</Label>
                  <TemplateInput
                    value={(localConfig.node as string) || ''}
                    onChange={(e) => handleConfigChange('node', e.target.value)}
                    placeholder="e.g., pve1"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>{isToolNode ? 'Default VM ID' : 'VM ID'}</Label>
                  <Input
                    type="number"
                    value={(localConfig.vmid as number) || ''}
                    onChange={(e) => handleConfigChange('vmid', parseInt(e.target.value) || 0)}
                    placeholder="e.g., 100"
                  />
                </div>
                {isToolNode && (
                  <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                    The agent can provide <span className="font-mono text-text">node</span> and <span className="font-mono text-text">vmid</span> when calling this tool. Any values here act as defaults.
                  </div>
                )}
              </>
            ) : nodeType === 'action:vm_clone' || nodeType === 'tool:vm_clone' ? (
              <>
                <div>
                  <Label>{isToolNode ? 'Default Proxmox Node' : 'Proxmox Node'}</Label>
                  <TemplateInput
                    value={(localConfig.node as string) || ''}
                    onChange={(e) => handleConfigChange('node', e.target.value)}
                    placeholder="e.g., pve1"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>{isToolNode ? 'Default Source VM ID' : 'Source VM ID'}</Label>
                  <Input
                    type="number"
                    value={(localConfig.vmid as number) || ''}
                    onChange={(e) => handleConfigChange('vmid', parseInt(e.target.value) || 0)}
                    placeholder="e.g., 100"
                  />
                </div>
                <div>
                  <Label>{isToolNode ? 'Default New VM Name' : 'New VM Name'}</Label>
                  <TemplateInput
                    value={(localConfig.newName as string) || ''}
                    onChange={(e) => handleConfigChange('newName', e.target.value)}
                    placeholder="e.g., cloned-vm"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>{isToolNode ? 'Default New VM ID' : 'New VM ID'}</Label>
                  <Input
                    type="number"
                    value={(localConfig.newId as number) || ''}
                    onChange={(e) => handleConfigChange('newId', parseInt(e.target.value) || 0)}
                    placeholder="e.g., 200"
                  />
                </div>
                {isToolNode && (
                  <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                    The agent can override <span className="font-mono text-text">node</span>, <span className="font-mono text-text">vmid</span>, <span className="font-mono text-text">newName</span>, and <span className="font-mono text-text">newId</span> when it calls this tool.
                  </div>
                )}
              </>
            ) : nodeType === 'action:http' || nodeType === 'tool:http' ? (
              <>
                <div>
                  <Label>{nodeType === 'tool:http' ? 'Default URL' : 'URL'}</Label>
                  <TemplateInput
                    value={(localConfig.url as string) || ''}
                    onChange={(e) => handleConfigChange('url', e.target.value)}
                    placeholder="https://api.example.com/endpoint"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>{nodeType === 'tool:http' ? 'Default Method' : 'Method'}</Label>
                  <select
                    value={(localConfig.method as string) || 'GET'}
                    onChange={(e) => handleConfigChange('method', e.target.value)}
                    className="w-full px-3 py-2 bg-bg-input border border-border rounded-lg text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent"
                  >
                    <option value="GET">GET</option>
                    <option value="POST">POST</option>
                    <option value="PUT">PUT</option>
                    <option value="DELETE">DELETE</option>
                    <option value="PATCH">PATCH</option>
                  </select>
                </div>
                <div>
                  <Label>{nodeType === 'tool:http' ? 'Default Body' : 'Body'}</Label>
                  <TemplateTextarea
                    value={(localConfig.body as string) || ''}
                    onChange={(e) => handleConfigChange('body', e.target.value)}
                    placeholder='{"key": "value"}'
                    rows={4}
                    className="font-mono text-xs"
                    suggestions={templateSuggestions}
                  />
                </div>
                {nodeType === 'tool:http' && (
                  <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                    The agent can pass <span className="font-mono text-text">url</span>, <span className="font-mono text-text">method</span>, <span className="font-mono text-text">headers</span>, and <span className="font-mono text-text">body</span>. Values here act as defaults.
                  </div>
                )}
              </>
            ) : nodeType === 'action:shell_command' || nodeType === 'tool:shell_command' ? (
              <>
                <div>
                  <Label>{nodeType === 'tool:shell_command' ? 'Default Command' : 'Command'}</Label>
                  <TemplateTextarea
                    value={(localConfig.command as string) || ''}
                    onChange={(e) => handleConfigChange('command', e.target.value)}
                    placeholder="e.g., Get-ChildItem .agents\\skills"
                    rows={4}
                    className="font-mono text-xs"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>Working Directory</Label>
                  <TemplateInput
                    value={(localConfig.workingDirectory as string) || ''}
                    onChange={(e) => handleConfigChange('workingDirectory', e.target.value)}
                    placeholder="Optional relative or absolute path"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>Timeout Seconds</Label>
                  <Input
                    type="number"
                    min="1"
                    value={(localConfig.timeoutSeconds as number) ?? 60}
                    onChange={(e) => handleConfigChange('timeoutSeconds', parseInt(e.target.value, 10) || 60)}
                  />
                </div>
                {nodeType === 'tool:shell_command' ? (
                  <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                    The agent can pass <span className="font-mono text-text">command</span>, <span className="font-mono text-text">workingDirectory</span>, and <span className="font-mono text-text">timeoutSeconds</span>. Values here act as defaults.
                  </div>
                ) : (
                  <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                    Runs a local shell command during pipeline execution and returns stdout, stderr, exit code, and timing information.
                  </div>
                )}
              </>
            ) : nodeType === 'action:channel_send_message' ? (
              <>
                <div>
                  <Label>Recipient</Label>
                  <TemplateInput
                    value={(localConfig.recipient as string) || ''}
                    onChange={(e) => handleConfigChange('recipient', e.target.value)}
                    placeholder="Optional contact ID or chat ID. Leave empty to reply to the triggering user."
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>Message</Label>
                  <TemplateTextarea
                    value={(localConfig.message as string) || ''}
                    onChange={(e) => handleConfigChange('message', e.target.value)}
                    placeholder="Write the message to send"
                    rows={6}
                    suggestions={templateSuggestions}
                  />
                </div>
              </>
            ) : nodeType === 'action:channel_send_and_wait' || nodeType === 'tool:channel_send_and_wait' ? (
              <>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  {nodeType === 'tool:channel_send_and_wait'
                    ? 'The agent can message a connected user and pause until that user replies. Matching replies are routed back into this tool instead of triggering other channel pipelines.'
                    : 'This node sends a message to a connected user, then waits until that same user replies or the timeout is reached.'}
                </div>
                <div>
                  <Label>{nodeType === 'tool:channel_send_and_wait' ? 'Default Recipient' : 'Recipient'}</Label>
                  <TemplateInput
                    value={(localConfig.recipient as string) || ''}
                    onChange={(e) => handleConfigChange('recipient', e.target.value)}
                    placeholder="Optional contact ID or chat ID. Leave empty to reply to the triggering user."
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>{nodeType === 'tool:channel_send_and_wait' ? 'Default Message' : 'Message'}</Label>
                  <TemplateTextarea
                    value={(localConfig.message as string) || ''}
                    onChange={(e) => handleConfigChange('message', e.target.value)}
                    placeholder="Write the message to send before waiting"
                    rows={6}
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>{nodeType === 'tool:channel_send_and_wait' ? 'Default Timeout (seconds)' : 'Timeout (seconds)'}</Label>
                  <Input
                    type="number"
                    min="1"
                    value={(localConfig.timeoutSeconds as number) || 300}
                    onChange={(e) => handleConfigChange('timeoutSeconds', parseInt(e.target.value, 10) || 0)}
                    placeholder="300"
                  />
                </div>
              </>
            ) : nodeType === 'action:pipeline_run' ? (
              <>
                <div>
                  <Label>Pipeline</Label>
                  <Select
                    value={(localConfig.pipelineId as string) || ''}
                    onChange={(e) => handleConfigChange('pipelineId', e.target.value)}
                  >
                    <option value="">Select pipeline</option>
                    {availablePipelines.map((pipeline) => (
                      <option key={pipeline.id} value={pipeline.id}>
                        {pipeline.name}
                      </option>
                    ))}
                  </Select>
                </div>
                <div>
                  <Label>Parameters JSON</Label>
                  <TemplateTextarea
                    value={(localConfig.params as string) || ''}
                    onChange={(e) => handleConfigChange('params', e.target.value)}
                    placeholder='{"message":"hello","target":"ops"}'
                    rows={6}
                    className="font-mono text-xs"
                    suggestions={templateSuggestions}
                  />
                </div>
              </>
            ) : nodeType === 'action:lua' ? (
              <div className="space-y-3">
                <Label>Lua Script</Label>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-3">
                  <p className="text-sm text-text">
                    {((localConfig.script as string) || '').trim()
                      ? 'Open the full-screen editor to update the Lua script.'
                      : 'No Lua script yet. Open the editor to add one.'}
                  </p>
                  <p className="mt-1 text-xs text-text-dimmed">
                    {(((localConfig.script as string) || '').split(/\r?\n/).filter(Boolean).length || 0)} lines saved
                  </p>
                </div>
                <div className="flex justify-end">
                  <Button variant="secondary" onClick={() => setIsLuaEditorOpen(true)}>
                    <Code className="w-4 h-4" />
                    Edit code
                  </Button>
                </div>
              </div>
            ) : nodeType === 'tool:pipeline_list' ? (
              <div className="space-y-3">
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  This tool exposes the list of available pipelines to a connected agent node.
                </div>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-xs text-text-muted">
                  The agent can optionally pass <span className="font-mono text-text">pipelineId</span>, <span className="font-mono text-text">pipelineName</span>, and <span className="font-mono text-text">includeDefinition</span> when it needs full nodes, edges, and viewport for editing.
                </div>
              </div>
            ) : pipelineMutationToolNodeTypes.has(nodeType) ? (
              <>
                <div>
                  <Label>Tool Name</Label>
                  <Input
                    value={(localConfig.toolName as string) || ''}
                    onChange={(e) => handleConfigChange('toolName', e.target.value)}
                    placeholder="Optional custom function name"
                  />
                </div>
                <div>
                  <Label>Tool Description</Label>
                  <TemplateTextarea
                    value={(localConfig.toolDescription as string) || ''}
                    onChange={(e) => handleConfigChange('toolDescription', e.target.value)}
                    placeholder="Explain when the model should use this tool."
                    rows={4}
                    suggestions={templateSuggestions}
                  />
                </div>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  {nodeType === 'tool:pipeline_create'
                    ? 'The connected agent can create new pipelines with a name, description, status, nodes, edges, and an optional viewport.'
                    : nodeType === 'tool:pipeline_update'
                    ? 'The connected agent can update an existing pipeline by pipelineId or exact pipelineName. Only the fields it sends will change.'
                    : 'The connected agent can delete a pipeline by pipelineId or exact pipelineName.'}
                </div>
              </>
            ) : nodeType === 'tool:pipeline_run' ? (
              <>
                <div>
                  <Label>Tool Name</Label>
                  <Input
                    value={(localConfig.toolName as string) || ''}
                    onChange={(e) => handleConfigChange('toolName', e.target.value)}
                    placeholder="Optional custom function name"
                  />
                </div>
                <div>
                  <Label>Tool Description</Label>
                  <TemplateTextarea
                    value={(localConfig.toolDescription as string) || ''}
                    onChange={(e) => handleConfigChange('toolDescription', e.target.value)}
                    placeholder="Explain when the model should use this tool."
                    rows={4}
                    suggestions={templateSuggestions}
                  />
                </div>
                <label className="flex items-start gap-3 rounded-lg border border-border bg-bg-input px-3 py-2">
                  <Checkbox
                    checked={Boolean(localConfig.allowModelPipelineId)}
                    onChange={(e) => handleConfigChange('allowModelPipelineId', e.target.checked)}
                    className="mt-0.5"
                  />
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-text">Let model choose pipeline ID</div>
                    <div className="mt-1 text-xs text-text-muted">
                      Expose a <span className="font-mono text-text">pipelineId</span> argument in the tool schema so the model can select the target pipeline dynamically.
                    </div>
                  </div>
                </label>
                <div>
                  <Label>{Boolean(localConfig.allowModelPipelineId) ? 'Default Pipeline' : 'Pipeline'}</Label>
                  <Select
                    value={(localConfig.pipelineId as string) || ''}
                    onChange={(e) => handleConfigChange('pipelineId', e.target.value)}
                  >
                    <option value="">{Boolean(localConfig.allowModelPipelineId) ? 'Optional default pipeline' : 'Select pipeline'}</option>
                    {availablePipelines.map((pipeline) => (
                      <option key={pipeline.id} value={pipeline.id}>
                        {pipeline.name}
                      </option>
                    ))}
                  </Select>
                </div>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  The connected agent can call this tool with a <span className="font-mono text-text">params</span> object. The target pipeline will receive those values as manual-run input, and a <span className="font-medium text-text">Return</span> node will send data back to the agent.
                  {Boolean(localConfig.allowModelPipelineId) && (
                    <span> When enabled, the model can also send a <span className="font-mono text-text">pipelineId</span> argument.</span>
                  )}
                </div>
              </>
            ) : nodeType === 'logic:condition' ? (
              <div>
                <ExpressionLabel />
                <TemplateTextarea
                  value={(localConfig.expression as string) || ''}
                  onChange={(e) => handleConfigChange('expression', e.target.value)}
                  placeholder="e.g., input.status == 'running'"
                  rows={3}
                  className="font-mono text-xs"
                  suggestions={templateSuggestions}
                />
              </div>
            ) : nodeType === 'logic:switch' ? (
              <div className="space-y-3">
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  Every condition gets its own output pin. All branches that evaluate to <span className="font-medium text-text">true</span> will fire, and the <span className="font-medium text-text">Else</span> pin runs when none match.
                </div>
                {switchConditions.map((condition) => (
                  <div key={condition.id} className="space-y-3 rounded-xl border border-border bg-bg-input/80 p-3">
                    <div className="flex items-start gap-2">
                      <div className="flex-1">
                        <Label>Branch Label</Label>
                        <Input
                          value={condition.label}
                          onChange={(e) => handleSwitchConditionChange(condition.id, 'label', e.target.value)}
                          placeholder="e.g., Healthy"
                        />
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="mt-6"
                        onClick={() => handleSwitchConditionRemove(condition.id)}
                        title="Remove condition"
                      >
                        <Trash2 className="w-4 h-4 text-red-400" />
                      </Button>
                    </div>
                    <div>
                      <ExpressionLabel />
                      <TemplateTextarea
                        value={condition.expression}
                        onChange={(e) => handleSwitchConditionChange(condition.id, 'expression', e.target.value)}
                        placeholder="e.g., input.status_code == 200"
                        rows={3}
                        className="font-mono text-xs"
                        suggestions={templateSuggestions}
                      />
                    </div>
                    <div className="rounded-md border border-border/70 bg-bg-overlay/70 px-2.5 py-2 text-xs text-text-dimmed">
                      Handle ID: <span className="font-mono text-text">{condition.id}</span>
                    </div>
                  </div>
                ))}
                <Button variant="secondary" onClick={handleSwitchConditionAdd}>
                  <Plus className="w-4 h-4" />
                  Add condition
                </Button>
              </div>
            ) : nodeType === 'logic:merge' ? (
              <>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  Merge waits for every connected upstream branch, then combines object outputs into one payload. Downstream nodes get the merged fields directly, plus a <span className="font-mono text-text">merged</span> object and <span className="font-mono text-text">entries</span> metadata.
                </div>
                <div>
                  <Label>Mode</Label>
                  <Select
                    value={(localConfig.mode as string) || 'shallow'}
                    onChange={(e) => handleConfigChange('mode', e.target.value)}
                  >
                    <option value="shallow">Shallow merge</option>
                    <option value="deep">Deep merge</option>
                  </Select>
                </div>
              </>
            ) : nodeType === 'logic:aggregate' ? (
              <div className="space-y-3">
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  Aggregate waits for every connected upstream branch and outputs:
                </div>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-xs text-text-muted">
                  <div><span className="font-mono text-text">items</span>: ordered list of upstream outputs</div>
                  <div className="mt-1"><span className="font-mono text-text">entries</span>: upstream outputs with node ids, labels, and types</div>
                  <div className="mt-1"><span className="font-mono text-text">byNodeId</span>: upstream outputs keyed by source node id</div>
                </div>
              </div>
            ) : nodeType === 'logic:return' ? (
              <>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  This node stops the pipeline and returns data to the caller. Leave the value empty to return the full current input object.
                </div>
                <div>
                  <Label>Return Value</Label>
                  <TemplateTextarea
                    value={(localConfig.value as string) || ''}
                    onChange={(e) => handleConfigChange('value', e.target.value)}
                    placeholder='{{input}} or {"status":"ok","message":{{input.message}}}'
                    rows={5}
                    className="font-mono text-xs"
                    suggestions={templateSuggestions}
                  />
                </div>
              </>
            ) : nodeType === 'llm:prompt' ? (
              <>
                <div>
                  <Label>Provider</Label>
                  <Select
                    value={(localConfig.providerId as string) || ''}
                    onChange={(e) => handleConfigChange('providerId', e.target.value)}
                  >
                    <option value="">Default provider{defaultProvider ? ` (${defaultProvider.name})` : ''}</option>
                    {llmProviders?.map((provider) => (
                      <option key={provider.id} value={provider.id}>
                        {provider.name} ({provider.provider_type})
                      </option>
                    ))}
                  </Select>
                </div>
                <div>
                  <Label>Prompt</Label>
                  <TemplateTextarea
                    value={(localConfig.prompt as string) || ''}
                    onChange={(e) => handleConfigChange('prompt', e.target.value)}
                    placeholder="Enter your prompt template..."
                    rows={6}
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>Model</Label>
                  {modelOptions.length > 0 ? (
                    <Select
                      value={(localConfig.model as string) || ''}
                      onChange={(e) => handleConfigChange('model', e.target.value)}
                    >
                      <option value="">Use provider default model</option>
                      {modelOptions.map((model) => (
                        <option key={model.id} value={model.id}>
                          {model.name || model.id}
                        </option>
                      ))}
                    </Select>
                  ) : (
                    <TemplateInput
                      value={(localConfig.model as string) || ''}
                      onChange={(e) => handleConfigChange('model', e.target.value)}
                      placeholder="e.g., openai/gpt-4o-mini"
                      suggestions={templateSuggestions}
                    />
                  )}
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label>Temperature</Label>
                    <Input
                      type="number"
                      step="0.1"
                      min="0"
                      max="2"
                      value={(localConfig.temperature as number) ?? 0.7}
                      onChange={(e) => handleConfigChange('temperature', parseFloat(e.target.value))}
                    />
                  </div>
                  <div>
                    <Label>Max Tokens</Label>
                    <Input
                      type="number"
                      value={(localConfig.max_tokens as number) ?? 1024}
                      onChange={(e) => handleConfigChange('max_tokens', parseInt(e.target.value))}
                    />
                  </div>
                </div>
              </>
            ) : nodeType === 'llm:agent' ? (
              <>
                <div>
                  <Label>Provider</Label>
                  <Select
                    value={(localConfig.providerId as string) || ''}
                    onChange={(e) => handleConfigChange('providerId', e.target.value)}
                  >
                    <option value="">Default provider{defaultProvider ? ` (${defaultProvider.name})` : ''}</option>
                    {llmProviders?.map((provider) => (
                      <option key={provider.id} value={provider.id}>
                        {provider.name} ({provider.provider_type})
                      </option>
                    ))}
                  </Select>
                </div>
                <div className="rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-muted">
                  Connected tools: <span className="font-medium text-text">{connectedToolCount}</span>. Connect tool nodes from the blue pin on the bottom of the agent node to the top pin on each tool node to make them available during multi-turn execution.
                </div>
                <label className="flex items-start gap-3 rounded-lg border border-border bg-bg-input px-3 py-2">
                  <Checkbox
                    checked={Boolean(localConfig.enableSkills)}
                    onChange={(e) => handleConfigChange('enableSkills', e.target.checked)}
                    className="mt-0.5"
                  />
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-text">Enable local skills</div>
                    <div className="mt-1 text-xs text-text-muted">
                      Adds the <span className="font-mono text-text">get_skill</span> tool to this agent and lets you insert <span className="font-mono text-text">{'{{skills}}'}</span> into the prompt.
                    </div>
                  </div>
                </label>
                <div>
                  <Label>Instructions</Label>
                  <TemplateTextarea
                    value={(localConfig.prompt as string) || ''}
                    onChange={(e) => handleConfigChange('prompt', e.target.value)}
                    placeholder="Describe the agent's role, goals, and how it should use connected tools."
                    rows={7}
                    suggestions={agentTemplateSuggestions}
                  />
                </div>
                <div>
                  <Label>Model</Label>
                  {modelOptions.length > 0 ? (
                    <Select
                      value={(localConfig.model as string) || ''}
                      onChange={(e) => handleConfigChange('model', e.target.value)}
                    >
                      <option value="">Use provider default model</option>
                      {modelOptions.map((model) => (
                        <option key={model.id} value={model.id}>
                          {model.name || model.id}
                        </option>
                      ))}
                    </Select>
                  ) : (
                    <TemplateInput
                      value={(localConfig.model as string) || ''}
                      onChange={(e) => handleConfigChange('model', e.target.value)}
                      placeholder="e.g., openai/gpt-4o-mini"
                      suggestions={agentTemplateSuggestions}
                    />
                  )}
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label>Temperature</Label>
                    <Input
                      type="number"
                      step="0.1"
                      min="0"
                      max="2"
                      value={(localConfig.temperature as number) ?? 0.7}
                      onChange={(e) => handleConfigChange('temperature', parseFloat(e.target.value))}
                    />
                  </div>
                  <div>
                    <Label>Max Tokens</Label>
                    <Input
                      type="number"
                      value={(localConfig.max_tokens as number) ?? 1024}
                      onChange={(e) => handleConfigChange('max_tokens', parseInt(e.target.value))}
                    />
                  </div>
                </div>
              </>
            ) : nodeType === 'trigger:cron' ? (
              <>
                <div>
                  <Label>Cron Expression</Label>
                  <TemplateInput
                    value={(localConfig.schedule as string) || ''}
                    onChange={(e) => handleConfigChange('schedule', e.target.value)}
                    placeholder="e.g., 0 * * * *"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>Timezone</Label>
                  <TemplateInput
                    value={(localConfig.timezone as string) || 'UTC'}
                    onChange={(e) => handleConfigChange('timezone', e.target.value)}
                    placeholder="e.g., UTC, America/New_York"
                    suggestions={templateSuggestions}
                  />
                </div>
              </>
            ) : nodeType === 'trigger:webhook' ? (
              <>
                <div>
                  <Label>Path</Label>
                  <TemplateInput
                    value={(localConfig.path as string) || ''}
                    onChange={(e) => handleConfigChange('path', e.target.value)}
                    placeholder="/webhook/my-endpoint"
                    suggestions={templateSuggestions}
                  />
                </div>
                <div>
                  <Label>Method</Label>
                  <select
                    value={(localConfig.method as string) || 'POST'}
                    onChange={(e) => handleConfigChange('method', e.target.value)}
                    className="w-full px-3 py-2 bg-bg-input border border-border rounded-lg text-text text-sm focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent"
                  >
                    <option value="POST">POST</option>
                    <option value="GET">GET</option>
                  </select>
                </div>
              </>
            ) : (
              <div className="text-center py-8">
                <Settings className="w-8 h-8 text-text-dimmed mx-auto mb-2" />
                <p className="text-sm text-text-muted">No configuration options</p>
              </div>
            )}
          </div>
        )}
      </div>

      {nodeType === 'action:lua' && isLuaEditorOpen && (
        <LuaEditorModal
          value={(localConfig.script as string) || ''}
          suggestions={templateSuggestions}
          onSave={(value) => handleConfigChange('script', value)}
          onClose={() => setIsLuaEditorOpen(false)}
        />
      )}
    </div>
  )
}
