import { useState, useRef, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Send, Bot, User, Loader2, Brain, Server } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { api } from '../api/client'
import { Card } from '../components/ui/Card'
import Input from '../components/ui/Input'
import Button from '../components/ui/Button'
import Select from '../components/ui/Select'
import Badge from '../components/ui/Badge'
import { cn } from '../lib/utils'
import type { Cluster, LLMChatResponse, LLMProvider, LLMToolCall, LLMToolResult } from '../types'

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  tool_calls?: LLMToolCall[]
  tool_results?: LLMToolResult[]
}

export default function LLMChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [selectedProvider, setSelectedProvider] = useState('')
  const [selectedCluster, setSelectedCluster] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const { data: providers } = useQuery<LLMProvider[]>({
    queryKey: ['llm-providers'],
    queryFn: () => api.llmProviders.list(),
  })

  const { data: clusters } = useQuery<Cluster[]>({
    queryKey: ['clusters'],
    queryFn: () => api.clusters.list(),
  })

  useEffect(() => {
    if (providers && !selectedProvider) {
      const defaultProvider = providers.find((p) => p.is_default)
      if (defaultProvider) {
        setSelectedProvider(defaultProvider.id)
        return
      }
      if (providers.length > 0) {
        setSelectedProvider(providers[0].id)
      }
    }
  }, [providers, selectedProvider])

  useEffect(() => {
    if (clusters && !selectedCluster && clusters.length > 0) {
      setSelectedCluster(clusters[0].id)
    }
  }, [clusters, selectedCluster])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = async () => {
    if (!input.trim() || isLoading) return

    const userMessage: ChatMessage = { role: 'user', content: input.trim() }
    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setIsLoading(true)

    try {
      const response: LLMChatResponse = await api.llm.chat({
        message: userMessage.content,
        provider_id: selectedProvider || undefined,
        cluster_id: selectedCluster || undefined,
      })

      const assistantMessage: ChatMessage = {
        role: 'assistant',
        content: response.content || '',
        tool_calls: response.tool_calls,
        tool_results: response.tool_results,
      }
      setMessages((prev) => [...prev, assistantMessage])
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: `Error: ${err.message}` },
      ])
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void handleSend()
    }
  }

  return (
    <div className="flex flex-col h-screen">
      <div className="flex items-center justify-between px-6 py-4 bg-bg-elevated border-b border-border gap-4">
        <div className="flex items-center gap-3 min-w-0">
          <Brain className="w-5 h-5 text-accent flex-shrink-0" />
          <div className="min-w-0">
            <h1 className="text-lg font-semibold text-text">LLM Chat</h1>
            <p className="text-xs text-text-muted">Interact with your Proxmox cluster using natural language</p>
          </div>
        </div>
        <div className="flex items-center gap-3 flex-wrap justify-end">
          {clusters && clusters.length > 0 ? (
            <div className="flex items-center gap-2">
              <Server className="w-4 h-4 text-text-dimmed" />
              <Select
                value={selectedCluster}
                onChange={(e) => setSelectedCluster(e.target.value)}
                className="w-52"
              >
                {clusters.map((cluster) => (
                  <option key={cluster.id} value={cluster.id}>
                    {cluster.name}
                  </option>
                ))}
              </Select>
            </div>
          ) : (
            <Badge variant="warning">No clusters</Badge>
          )}
          {providers && providers.length > 0 && (
            <div className="flex items-center gap-2">
              <Brain className="w-4 h-4 text-text-dimmed" />
              <Select
                value={selectedProvider}
                onChange={(e) => setSelectedProvider(e.target.value)}
                className="w-56"
              >
                {providers.map((provider) => (
                  <option key={provider.id} value={provider.id}>
                    {provider.name} ({provider.model})
                  </option>
                ))}
              </Select>
            </div>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-4">
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-center">
            <div className="w-16 h-16 rounded-2xl bg-accent/10 flex items-center justify-center mb-4">
              <Bot className="w-8 h-8 text-accent" />
            </div>
            <h2 className="text-xl font-semibold text-text mb-2">Proxmox AI Assistant</h2>
            <p className="text-text-muted max-w-md">
              Ask me to manage your Proxmox cluster. I can list VMs, start or stop machines, check node status, and inspect resources.
            </p>
            <div className="flex flex-wrap gap-2 mt-6 justify-center">
              {['List all VMs', 'Show cluster resources', 'List nodes', 'Check storage'].map((suggestion) => (
                <button
                  key={suggestion}
                  onClick={() => setInput(suggestion)}
                  className="px-3 py-1.5 text-xs bg-bg-elevated border border-border rounded-full text-text-muted hover:text-text hover:border-accent/50 transition-colors"
                >
                  {suggestion}
                </button>
              ))}
            </div>
          </div>
        )}

        {messages.map((msg, i) => (
          <div
            key={i}
            className={cn(
              'flex gap-3 max-w-3xl',
              msg.role === 'user' ? 'ml-auto flex-row-reverse' : '',
            )}
          >
            <div className={cn(
              'w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0',
              msg.role === 'user' ? 'bg-accent/10' : 'bg-purple-500/10',
            )}>
              {msg.role === 'user' ? (
                <User className="w-4 h-4 text-accent" />
              ) : (
                <Bot className="w-4 h-4 text-purple-400" />
              )}
            </div>
            <div className={cn(
              'rounded-xl px-4 py-3 max-w-[80%]',
              msg.role === 'user'
                ? 'bg-accent/10 border border-accent/20'
                : 'bg-bg-elevated border border-border',
            )}>
              <ReactMarkdown remarkPlugins={[remarkGfm]} className="chat-markdown text-sm text-text">
                {msg.content}
              </ReactMarkdown>

              {msg.tool_results && msg.tool_results.length > 0 && (
                <div className="mt-3 space-y-2">
                  {msg.tool_results.map((result, idx) => (
                    <Card key={idx} className="bg-bg-input border-border">
                      <details className="group">
                        <summary className="flex cursor-pointer list-none items-center justify-between gap-3 p-3">
                          <span className="text-xs font-mono text-text">{result.tool}</span>
                          <Badge variant={result.error ? 'error' : 'success'}>
                            {result.error ? 'Failed' : 'Success'}
                          </Badge>
                        </summary>
                        <div className="border-t border-border px-3 py-3 space-y-3">
                          {result.arguments !== undefined && (
                            <div>
                              <p className="text-[11px] uppercase tracking-wide text-text-dimmed mb-1">Arguments</p>
                              <pre className="text-xs text-text-muted overflow-x-auto">
                                {JSON.stringify(result.arguments, null, 2)}
                              </pre>
                            </div>
                          )}
                          {result.error ? (
                            <div>
                              <p className="text-[11px] uppercase tracking-wide text-text-dimmed mb-1">Error</p>
                              <pre className="text-xs text-red-400 overflow-x-auto whitespace-pre-wrap">
                                {result.error}
                              </pre>
                            </div>
                          ) : (
                            <div>
                              <p className="text-[11px] uppercase tracking-wide text-text-dimmed mb-1">Result</p>
                              <pre className="text-xs text-text-muted overflow-x-auto">
                                {JSON.stringify(result.result, null, 2)}
                              </pre>
                            </div>
                          )}
                        </div>
                      </details>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}

        {isLoading && (
          <div className="flex gap-3">
            <div className="w-8 h-8 rounded-lg bg-purple-500/10 flex items-center justify-center">
              <Bot className="w-4 h-4 text-purple-400" />
            </div>
            <div className="bg-bg-elevated border border-border rounded-xl px-4 py-3">
              <Loader2 className="w-4 h-4 text-text-muted animate-spin" />
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="p-4 bg-bg-elevated border-t border-border">
        <div className="flex gap-3 max-w-3xl mx-auto">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask me to manage your selected cluster..."
            className="flex-1"
          />
          <Button onClick={() => void handleSend()} disabled={!input.trim() || isLoading}>
            <Send className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
