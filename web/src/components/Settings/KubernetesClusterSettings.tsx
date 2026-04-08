import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Edit2, Plus, RefreshCw, Shield, Trash2 } from 'lucide-react'
import { api } from '../../api/client'
import { Card, CardContent } from '../ui/Card'
import Button from '../ui/Button'
import Input from '../ui/Input'
import Select from '../ui/Select'
import { Checkbox, Label, Textarea } from '../ui/Form'
import Badge from '../ui/Badge'
import Skeleton from '../ui/Skeleton'
import { useUIStore } from '../../store/ui'
import type { KubernetesCluster, KubernetesManualAuthConfig } from '../../types'

type KubernetesClusterFormState = {
  name: string
  source_type: 'kubeconfig' | 'manual'
  kubeconfig: string
  context_name: string
  default_namespace: string
  manual: KubernetesManualAuthConfig
}

function getDefaultManualConfig(): KubernetesManualAuthConfig {
  return {
    server: '',
    token: '',
    username: '',
    password: '',
    ca_data: '',
    client_certificate_data: '',
    client_key_data: '',
    insecure_skip_tls_verify: false,
  }
}

function getDefaultKubernetesClusterForm(): KubernetesClusterFormState {
  return {
    name: '',
    source_type: 'kubeconfig',
    kubeconfig: '',
    context_name: '',
    default_namespace: 'default',
    manual: getDefaultManualConfig(),
  }
}

function kubernetesClusterToForm(cluster: KubernetesCluster): KubernetesClusterFormState {
  return {
    name: cluster.name,
    source_type: cluster.source_type === 'manual' ? 'manual' : 'kubeconfig',
    kubeconfig: cluster.kubeconfig || '',
    context_name: cluster.context_name || '',
    default_namespace: cluster.default_namespace || 'default',
    manual: {
      ...getDefaultManualConfig(),
      ...(cluster.manual || {}),
      server: cluster.manual?.server || cluster.server || '',
    },
  }
}

function buildPayload(form: KubernetesClusterFormState) {
  return {
    name: form.name,
    source_type: form.source_type,
    kubeconfig: form.source_type === 'kubeconfig' ? form.kubeconfig : '',
    context_name: form.context_name,
    default_namespace: form.default_namespace,
    manual: form.source_type === 'manual' ? form.manual : undefined,
  }
}

export default function KubernetesClusterSettings() {
  const queryClient = useQueryClient()
  const { addToast } = useUIStore()
  const [showForm, setShowForm] = useState(false)
  const [editingClusterId, setEditingClusterId] = useState<string | null>(null)
  const [loadingClusterId, setLoadingClusterId] = useState<string | null>(null)
  const [form, setForm] = useState<KubernetesClusterFormState>(getDefaultKubernetesClusterForm())

  const { data: clusters, isLoading } = useQuery<KubernetesCluster[]>({
    queryKey: ['kubernetes-clusters'],
    queryFn: () => api.kubernetesClusters.list(),
  })

  const createMutation = useMutation({
    mutationFn: () => api.kubernetesClusters.create(buildPayload(form)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kubernetes-clusters'] })
      resetForm()
      addToast({ type: 'success', title: 'Kubernetes cluster added' })
    },
    onError: (err: Error) => {
      addToast({ type: 'error', title: 'Failed to add Kubernetes cluster', message: err.message })
    },
  })

  const updateMutation = useMutation({
    mutationFn: () => api.kubernetesClusters.update(editingClusterId!, buildPayload(form)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kubernetes-clusters'] })
      resetForm()
      addToast({ type: 'success', title: 'Kubernetes cluster updated' })
    },
    onError: (err: Error) => {
      addToast({ type: 'error', title: 'Failed to update Kubernetes cluster', message: err.message })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.kubernetesClusters.delete(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['kubernetes-clusters'] })
      addToast({ type: 'success', title: 'Kubernetes cluster removed' })
      if (editingClusterId === id) {
        resetForm()
      }
    },
    onError: (err: Error) => {
      addToast({ type: 'error', title: 'Failed to remove Kubernetes cluster', message: err.message })
    },
  })

  const testMutation = useMutation({
    mutationFn: () => api.kubernetesClusters.test(buildPayload(form)),
    onSuccess: (result) => {
      addToast({
        type: 'success',
        title: 'Connection succeeded',
        message: `${result.server_version} on ${result.server} (${result.effective_context}, ${result.default_namespace})`,
      })
    },
    onError: (err: Error) => {
      addToast({ type: 'error', title: 'Connection test failed', message: err.message })
    },
  })

  function resetForm() {
    setShowForm(false)
    setEditingClusterId(null)
    setLoadingClusterId(null)
    setForm(getDefaultKubernetesClusterForm())
  }

  function startCreate() {
    setEditingClusterId(null)
    setForm(getDefaultKubernetesClusterForm())
    setShowForm(true)
  }

  async function startEdit(id: string) {
    setLoadingClusterId(id)
    try {
      const cluster = await api.kubernetesClusters.get(id)
      setEditingClusterId(id)
      setForm(kubernetesClusterToForm(cluster))
      setShowForm(true)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      addToast({ type: 'error', title: 'Failed to load Kubernetes cluster', message })
    } finally {
      setLoadingClusterId(null)
    }
  }

  function updateManualField<K extends keyof KubernetesManualAuthConfig>(key: K, value: KubernetesManualAuthConfig[K]) {
    setForm((current) => ({
      ...current,
      manual: {
        ...current.manual,
        [key]: value,
      },
    }))
  }

  function submitForm() {
    if (editingClusterId) {
      updateMutation.mutate()
      return
    }
    createMutation.mutate()
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Kubernetes Clusters</h2>
          <p className="text-sm text-text-muted">Store kubeconfigs or manual credentials for Kubernetes-aware nodes and chat tools.</p>
        </div>
        <Button onClick={showForm ? resetForm : startCreate}>
          <Plus className="w-4 h-4" />
          {showForm ? 'Close' : 'Add Cluster'}
        </Button>
      </div>

      {showForm && (
        <Card>
          <CardContent className="space-y-4">
            <div>
              <h3 className="text-base font-semibold text-text">
                {editingClusterId ? 'Edit Kubernetes Cluster' : 'Add Kubernetes Cluster'}
              </h3>
              <p className="mt-1 text-sm text-text-muted">
                {editingClusterId
                  ? 'Update the stored cluster connection and default context details.'
                  : 'Choose whether to save a kubeconfig directly or build one from manual connection details.'}
              </p>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <Label>Display Name</Label>
                <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Production cluster" />
              </div>
              <div>
                <Label>Connection Mode</Label>
                <Select value={form.source_type} onChange={(e) => setForm({ ...form, source_type: e.target.value as 'kubeconfig' | 'manual' })}>
                  <option value="kubeconfig">Kubeconfig</option>
                  <option value="manual">Manual</option>
                </Select>
              </div>
              <div>
                <Label>Context Name</Label>
                <Input value={form.context_name} onChange={(e) => setForm({ ...form, context_name: e.target.value })} placeholder="Optional override" />
              </div>
              <div>
                <Label>Default Namespace</Label>
                <Input value={form.default_namespace} onChange={(e) => setForm({ ...form, default_namespace: e.target.value })} placeholder="default" />
              </div>
            </div>

            {form.source_type === 'kubeconfig' ? (
              <div>
                <Label>Kubeconfig</Label>
                <Textarea
                  value={form.kubeconfig}
                  onChange={(e) => setForm({ ...form, kubeconfig: e.target.value })}
                  rows={14}
                  className="font-mono text-xs"
                  placeholder="Paste the kubeconfig YAML or JSON here"
                />
              </div>
            ) : (
              <div className="space-y-4 rounded-xl border border-border bg-bg-input/40 p-4">
                <div className="flex items-center gap-2 text-sm text-text">
                  <Shield className="h-4 w-4 text-accent" />
                  Manual mode is normalized into a synthetic kubeconfig before it is stored.
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="md:col-span-2">
                    <Label>Server URL</Label>
                    <Input value={form.manual.server} onChange={(e) => updateManualField('server', e.target.value)} placeholder="https://cluster.example:6443" />
                  </div>
                  <div>
                    <Label>Bearer Token</Label>
                    <Textarea value={form.manual.token} onChange={(e) => updateManualField('token', e.target.value)} rows={4} className="font-mono text-xs" placeholder="Optional if using another auth method" />
                  </div>
                  <div className="space-y-4">
                    <div>
                      <Label>Username</Label>
                      <Input value={form.manual.username} onChange={(e) => updateManualField('username', e.target.value)} placeholder="Optional" />
                    </div>
                    <div>
                      <Label>Password</Label>
                      <Input type="password" value={form.manual.password} onChange={(e) => updateManualField('password', e.target.value)} placeholder="Optional" />
                    </div>
                  </div>
                  <div>
                    <Label>CA Data</Label>
                    <Textarea value={form.manual.ca_data} onChange={(e) => updateManualField('ca_data', e.target.value)} rows={4} className="font-mono text-xs" placeholder="PEM or base64" />
                  </div>
                  <div>
                    <Label>Client Certificate Data</Label>
                    <Textarea value={form.manual.client_certificate_data} onChange={(e) => updateManualField('client_certificate_data', e.target.value)} rows={4} className="font-mono text-xs" placeholder="Optional PEM or base64" />
                  </div>
                  <div className="md:col-span-2">
                    <Label>Client Key Data</Label>
                    <Textarea value={form.manual.client_key_data} onChange={(e) => updateManualField('client_key_data', e.target.value)} rows={4} className="font-mono text-xs" placeholder="Optional PEM or base64" />
                  </div>
                </div>

                <label className="flex items-start gap-3 rounded-lg border border-border bg-bg-overlay/60 px-3 py-2">
                  <Checkbox checked={form.manual.insecure_skip_tls_verify} onChange={(e) => updateManualField('insecure_skip_tls_verify', e.target.checked)} className="mt-0.5" />
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-text">Skip TLS verification</div>
                    <div className="mt-1 text-xs text-text-muted">Useful for local labs, but avoid enabling this for production clusters.</div>
                  </div>
                </label>
              </div>
            )}

            <div className="flex flex-wrap items-center gap-3">
              <Button
                variant="secondary"
                onClick={() => testMutation.mutate()}
                disabled={testMutation.isPending || createMutation.isPending || updateMutation.isPending}
              >
                <RefreshCw className="w-4 h-4" />
                Test Connection
              </Button>
              <Button
                onClick={submitForm}
                disabled={createMutation.isPending || updateMutation.isPending || testMutation.isPending}
              >
                {editingClusterId ? 'Save Changes' : 'Save Cluster'}
              </Button>
              <Button variant="ghost" onClick={resetForm}>
                Cancel
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2].map((index) => (
            <Skeleton key={index} className="h-32 w-full rounded-xl" />
          ))}
        </div>
      ) : clusters?.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <h3 className="text-lg font-medium text-text mb-2">No Kubernetes clusters configured</h3>
            <p className="text-text-muted">Add your first Kubernetes cluster to enable Kubernetes nodes and chat tools.</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {clusters?.map((cluster) => (
            <Card key={cluster.id}>
              <CardContent className="flex items-start justify-between gap-4">
                <div className="space-y-2">
                  <div className="flex items-center gap-3">
                    <h3 className="text-base font-semibold text-text">{cluster.name}</h3>
                    <Badge variant={cluster.source_type === 'manual' ? 'warning' : 'info'}>
                      {cluster.source_type}
                    </Badge>
                  </div>
                  <div className="grid gap-1 text-sm text-text-muted">
                    <p>Server: {cluster.server || 'Unavailable'}</p>
                    <p>Context: {cluster.context_name || 'Default context'}</p>
                    <p>Namespace: {cluster.default_namespace || 'default'}</p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => void startEdit(cluster.id)}
                    disabled={loadingClusterId === cluster.id || deleteMutation.isPending}
                    title="Edit cluster"
                  >
                    <Edit2 className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => deleteMutation.mutate(cluster.id)}
                    disabled={loadingClusterId === cluster.id || deleteMutation.isPending}
                    title="Delete cluster"
                  >
                    <Trash2 className="w-4 h-4 text-red-400" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
