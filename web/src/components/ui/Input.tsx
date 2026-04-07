import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

const Input = forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, type = 'text', ...props }, ref) => {
    return (
      <input
        ref={ref}
        type={type}
        className={cn(
          'w-full px-3 py-2 bg-bg-input border border-border rounded-lg text-text text-sm',
          'placeholder:text-text-dimmed',
          'focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent',
          'disabled:opacity-50 disabled:pointer-events-none',
          'transition-colors',
          className
        )}
        {...props}
      />
    )
  }
)

export default Input
