import ReactJsonView from '@microlink/react-json-view'
import { X } from 'lucide-react'
import type { NodeExecutionLogData, NodeType } from '../../types'

interface NodeExecutionModalProps {
  nodeId: string
  nodeLabel: string
  nodeType: NodeType
  log: NodeExecutionLogData
  onClose: () => void
}

function toViewerSource(value: unknown): object {
  if (value === undefined) {
    return {}
  }

  if (value !== null && typeof value === 'object') {
    return value as object
  }

  return { value }
}

function buildResultSource(log: NodeExecutionLogData): object {
  if (log.error) {
    return {
      status: log.status,
      error: log.error,
      output: log.output ?? null,
    }
  }

  if (log.output !== undefined) {
    return toViewerSource(log.output)
  }

  return { status: log.status }
}

const viewerTheme = {
  base00: '#101722',
  base01: '#172130',
  base02: '#243246',
  base03: '#6b7280',
  base04: '#98a2b3',
  base05: '#d3dae6',
  base06: '#f1f5f9',
  base07: '#ffffff',
  base08: '#f87171',
  base09: '#fb923c',
  base0A: '#fbbf24',
  base0B: '#4ade80',
  base0C: '#22d3ee',
  base0D: '#60a5fa',
  base0E: '#c084fc',
  base0F: '#f472b6',
}

export default function NodeExecutionModal({ nodeId, nodeLabel, nodeType, log, onClose }: NodeExecutionModalProps) {
  return (
    <div
      className="fixed inset-0 z-[60] bg-slate-950/70 backdrop-blur-sm flex items-center justify-center p-6"
      onClick={onClose}
    >
      <div
        className="w-full max-w-6xl max-h-[85vh] bg-bg-elevated border border-border rounded-2xl shadow-2xl overflow-hidden"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4 px-5 py-4 border-b border-border bg-bg-overlay">
          <div className="min-w-0">
            <p className="text-lg font-semibold text-text truncate">{nodeLabel}</p>
            <p className="text-xs text-text-dimmed mt-1">
              {nodeId} · {nodeType} · {log.status}
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-text-dimmed hover:text-text transition-colors"
            aria-label="Close execution log"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 divide-y lg:divide-y-0 lg:divide-x divide-border max-h-[calc(85vh-81px)] overflow-y-auto">
          <section className="min-h-0">
            <div className="px-5 py-3 border-b border-border bg-bg-overlay/80">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-text-muted">Input Data</p>
            </div>
            <div className="p-4 overflow-auto">
              <ReactJsonView
                src={toViewerSource(log.input)}
                name={false}
                collapsed={1}
                collapseStringsAfterLength={120}
                displayDataTypes={false}
                displayObjectSize={true}
                enableClipboard
                iconStyle="triangle"
                theme={viewerTheme}
                style={{
                  backgroundColor: '#101722',
                  padding: '16px',
                  borderRadius: '12px',
                  fontSize: '12px',
                }}
              />
            </div>
          </section>

          <section className="min-h-0">
            <div className="px-5 py-3 border-b border-border bg-bg-overlay/80">
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-text-muted">Execution Result</p>
            </div>
            <div className="p-4 overflow-auto">
              <ReactJsonView
                src={buildResultSource(log)}
                name={false}
                collapsed={1}
                collapseStringsAfterLength={120}
                displayDataTypes={false}
                displayObjectSize={true}
                enableClipboard
                iconStyle="triangle"
                theme={viewerTheme}
                style={{
                  backgroundColor: '#101722',
                  padding: '16px',
                  borderRadius: '12px',
                  fontSize: '12px',
                }}
              />
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}
