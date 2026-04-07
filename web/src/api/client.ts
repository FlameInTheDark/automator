import type {
  ActiveExecution,
  Channel,
  ChannelContact,
  Cluster,
  DashboardStats,
  Execution,
  ExecutionDetail,
  LLMChatResponse,
  LLMModelInfo,
  LLMProvider,
  Pipeline,
  PipelineRunResponse,
} from '../types'

const API_BASE = '/api/v1'

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(error.error || `Request failed: ${res.status}`)
  }

  if (res.status === 204) return undefined as unknown as T
  return res.json() as Promise<T>
}

export const api = {
  clusters: {
    list: () => request<Cluster[]>('/clusters'),
    create: (data: unknown) => request<Cluster>('/clusters', { method: 'POST', body: JSON.stringify(data) }),
    get: (id: string) => request<Cluster>(`/clusters/${id}`),
    update: (id: string, data: unknown) => request<Cluster>(`/clusters/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request<void>(`/clusters/${id}`, { method: 'DELETE' }),
  },
  dashboard: {
    stats: () => request<DashboardStats>('/dashboard/stats'),
  },
  channels: {
    list: () => request<Channel[]>('/channels'),
    create: (data: unknown) => request<Channel>('/channels', { method: 'POST', body: JSON.stringify(data) }),
    get: (id: string) => request<Channel>(`/channels/${id}`),
    update: (id: string, data: unknown) => request<Channel>(`/channels/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request<void>(`/channels/${id}`, { method: 'DELETE' }),
    contacts: (id: string) => request<ChannelContact[]>(`/channels/${id}/contacts`),
    connect: (data: unknown) => request<ChannelContact>('/channels/connect', { method: 'POST', body: JSON.stringify(data) }),
  },
  pipelines: {
    list: () => request<Pipeline[]>('/pipelines'),
    create: (data: unknown) => request<Pipeline>('/pipelines', { method: 'POST', body: JSON.stringify(data) }),
    get: (id: string) => request<Pipeline>(`/pipelines/${id}`),
    update: (id: string, data: unknown) => request<Pipeline>(`/pipelines/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request<void>(`/pipelines/${id}`, { method: 'DELETE' }),
    run: (id: string) => request<PipelineRunResponse>(`/pipelines/${id}/run`, { method: 'POST' }),
  },
  llmProviders: {
    list: () => request<LLMProvider[]>('/llm-providers'),
    create: (data: unknown) => request<LLMProvider>('/llm-providers', { method: 'POST', body: JSON.stringify(data) }),
    get: (id: string) => request<LLMProvider>(`/llm-providers/${id}`),
    update: (id: string, data: unknown) => request<LLMProvider>(`/llm-providers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request<void>(`/llm-providers/${id}`, { method: 'DELETE' }),
    models: (id: string) => request<LLMModelInfo[]>(`/llm-providers/${id}/models`),
  },
  executions: {
    listByPipeline: (pipelineId: string) => request<Execution[]>(`/executions/pipelines/${pipelineId}`),
    activeByPipeline: (pipelineId: string) => request<ActiveExecution[]>(`/executions/pipelines/${pipelineId}/active`),
    get: (executionId: string) => request<ExecutionDetail>(`/executions/${executionId}`),
    cancel: (executionId: string) => request<ActiveExecution>(`/executions/${executionId}/cancel`, { method: 'POST' }),
  },
  llm: {
    chat: (data: unknown) => request<LLMChatResponse>('/llm/chat', { method: 'POST', body: JSON.stringify(data) }),
  },
}
