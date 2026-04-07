import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

const Select = forwardRef<HTMLSelectElement, React.SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, children, ...props }, ref) => {
    return (
      <select
        ref={ref}
        className={cn(
          'w-full px-3 py-2 bg-bg-input border border-border rounded-lg text-text text-sm',
          'focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent',
          'disabled:opacity-50 disabled:pointer-events-none',
          'transition-colors appearance-none',
          className
        )}
        {...props}
      >
        {children}
      </select>
    )
  }
)

export default Select
