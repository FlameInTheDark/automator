import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

export function Label({ className, ...props }: React.LabelHTMLAttributes<HTMLLabelElement>) {
  return <label className={cn('block text-sm font-medium text-text-muted mb-1.5', className)} {...props} />
}

export const Textarea = forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        ref={ref}
        className={cn(
          'w-full px-3 py-2 bg-bg-input border border-border rounded-lg text-text text-sm',
          'placeholder:text-text-dimmed',
          'focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent',
          'disabled:opacity-50 disabled:pointer-events-none',
          'transition-colors resize-y',
          className
        )}
        {...props}
      />
    )
  }
)

Textarea.displayName = 'Textarea'

export function Checkbox({ className, ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      type="checkbox"
      className={cn(
        'w-4 h-4 rounded border-border bg-bg-input text-accent',
        'focus:ring-2 focus:ring-accent/50 focus:ring-offset-0',
        'cursor-pointer',
        className
      )}
      {...props}
    />
  )
}
