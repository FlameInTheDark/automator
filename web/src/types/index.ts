export interface Cluster {
  id: string
  name: string
  host: string
  port: number
  api_token_id: string
  api_token_secret?: string
  skip_tls_verify: boolean
  created_at: string
  updated_at: string
}

export interface Channel {
  id: string
  name: string
  type: string
  config?: string
  welcome_message: string
  connect_url?: string
  enabled: boolean
  state?: string
  created_at: string
  updated_at: string
}

export interface ChannelContact {
  id: string
  channel_id: string
  external_user_id: string
  external_chat_id: string
  username?: string
  display_name?: string
  connection_code?: string
  code_expires_at?: string
  connected_at?: string
  last_message_at?: string
  created_at: string
  updated_at: string
}

export interface LLMProvider {
  id: string
  name: string
  provider_type: string
  api_key?: string
  base_url?: string
  model: string
  config?: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface Pipeline {
  id: string
  name: string
  description: string | null
  nodes: string
  edges: string
  viewport?: string
  status: 'draft' | 'active' | 'archived'
  created_at: string
  updated_at: string
}

export interface DashboardStats {
  clusters: number
  pipelines: number
  active_pipelines: number
  active_jobs: number
  executions_24h: number
  channels: number
}

export interface Execution {
  id: string
  pipeline_id: string
  trigger_type: string
  status: 'running' | 'completed' | 'failed' | 'cancelled'
  started_at: string
  completed_at?: string
  error?: string
  context?: string
}

export interface ActiveExecution {
  execution_id: string
  pipeline_id: string
  trigger_type: string
  status: 'running' | 'cancelling'
  started_at: string
  current_node_id?: string
  current_node_type?: string
  current_node_started_at?: string
}

export interface NodeExecution {
  id: string
  execution_id: string
  node_id: string
  node_type: string
  status: string
  input?: string
  output?: string
  error?: string
  started_at?: string
  completed_at?: string
}

export interface ExecutionDetail {
  execution: Execution
  node_executions: NodeExecution[]
}

export interface PipelineRunResponse {
  execution_id: string
  status: 'completed' | 'failed' | 'cancelled'
  duration: string
  nodes_run: number
  error?: string
  returned?: boolean
  return_value?: unknown
}

export interface NodeExecutionLogData {
  status: string
  input?: unknown
  output?: unknown
  error?: string
  node_type?: string
}

export interface LLMModelInfo {
  id: string
  name?: string
  description?: string
  context_length?: number
}

export interface LLMToolFunction {
  name: string
  arguments: string
}

export interface LLMToolCall {
  id: string
  type: string
  function: LLMToolFunction
}

export interface LLMToolResult {
  tool: string
  arguments?: unknown
  result?: unknown
  error?: string
}

export interface LLMUsage {
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

export interface LLMChatResponse {
  content: string
  tool_calls?: LLMToolCall[]
  tool_results?: LLMToolResult[]
  usage?: LLMUsage
}

export interface TemplateSuggestion {
  template: string
  expression: string
  label: string
  description?: string
}

export type NodeType =
  | 'trigger:manual'
  | 'trigger:cron'
  | 'trigger:webhook'
  | 'trigger:channel_message'
  | 'action:proxmox_list_nodes'
  | 'action:proxmox_list_workloads'
  | 'action:vm_start'
  | 'action:vm_stop'
  | 'action:vm_clone'
  | 'action:http'
  | 'action:shell_command'
  | 'action:lua'
  | 'action:channel_send_message'
  | 'action:channel_send_and_wait'
  | 'action:pipeline_run'
  | 'tool:proxmox_list_nodes'
  | 'tool:proxmox_list_workloads'
  | 'tool:vm_start'
  | 'tool:vm_stop'
  | 'tool:vm_clone'
  | 'tool:http'
  | 'tool:shell_command'
  | 'tool:pipeline_list'
  | 'tool:pipeline_create'
  | 'tool:pipeline_update'
  | 'tool:pipeline_delete'
  | 'tool:pipeline_run'
  | 'tool:channel_send_and_wait'
  | 'logic:condition'
  | 'logic:switch'
  | 'logic:merge'
  | 'logic:aggregate'
  | 'logic:return'
  | 'llm:prompt'
  | 'llm:agent'

export interface NodeTypeDefinition {
  type: NodeType
  label: string
  description: string
  icon: string
  category: string
  color: string
  defaultConfig: Record<string, unknown>
}

export interface NodeCategory {
  id: string
  label: string
  color: string
  types: NodeTypeDefinition[]
}

export interface Toast {
  id: string
  type: 'success' | 'error' | 'info' | 'warning'
  title: string
  message?: string
  duration?: number
}
