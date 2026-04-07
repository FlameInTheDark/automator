import { cn } from '../../lib/utils'

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'success' | 'warning' | 'error' | 'info'
}

export default function Badge({ className, variant = 'default', ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
        {
          'bg-bg-overlay text-text-muted border border-border': variant === 'default',
          'bg-green-600/20 text-green-400 border border-green-600/30': variant === 'success',
          'bg-amber-600/20 text-amber-400 border border-amber-600/30': variant === 'warning',
          'bg-red-600/20 text-red-400 border border-red-600/30': variant === 'error',
          'bg-blue-600/20 text-blue-400 border border-blue-600/30': variant === 'info',
        },
        className
      )}
      {...props}
    />
  )
}
