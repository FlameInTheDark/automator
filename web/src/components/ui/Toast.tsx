import { useEffect } from 'react'
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useUIStore } from '../../store/ui'

const icons = {
  success: CheckCircle,
  error: AlertCircle,
  info: Info,
  warning: AlertTriangle,
}

export default function ToastContainer() {
  const { toasts, removeToast } = useUIStore()

  useEffect(() => {
    toasts.forEach((toast) => {
      if (toast.duration && toast.duration > 0) {
        const timer = setTimeout(() => removeToast(toast.id), toast.duration)
        return () => clearTimeout(timer)
      }
    })
  }, [toasts, removeToast])

  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2">
      {toasts.map((toast) => {
        const Icon = icons[toast.type]
        return (
          <div
            key={toast.id}
            className={cn(
              'flex items-center gap-3 px-4 py-3 rounded-lg border shadow-lg min-w-[320px] max-w-md animate-slide-in',
              'bg-bg-elevated border-border'
            )}
          >
            <Icon className={cn('w-5 h-5 flex-shrink-0', {
              'text-green-400': toast.type === 'success',
              'text-red-400': toast.type === 'error',
              'text-blue-400': toast.type === 'info',
              'text-amber-400': toast.type === 'warning',
            })} />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-text">{toast.title}</p>
              {toast.message && <p className="text-xs text-text-muted mt-0.5">{toast.message}</p>}
            </div>
            <button onClick={() => removeToast(toast.id)} className="text-text-dimmed hover:text-text flex-shrink-0">
              <X className="w-4 h-4" />
            </button>
          </div>
        )
      })}
    </div>
  )
}
