import { Navigate, Outlet, useLocation } from 'react-router-dom'
import Button from '../ui/Button'
import { useAuthSession } from '../../lib/auth'

export default function RequireAuth() {
  const location = useLocation()
  const sessionQuery = useAuthSession()

  if (sessionQuery.isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-bg text-text">
        <div className="rounded-xl border border-border bg-bg-elevated px-5 py-4 text-sm text-text-muted">
          Checking session...
        </div>
      </div>
    )
  }

  if (sessionQuery.error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-bg px-4 text-text">
        <div className="w-full max-w-md rounded-2xl border border-border bg-bg-elevated p-6">
          <h1 className="text-lg font-semibold text-text">Unable to verify your session</h1>
          <p className="mt-2 text-sm text-text-muted">{sessionQuery.error.message}</p>
          <Button className="mt-4" onClick={() => sessionQuery.refetch()}>
            Try Again
          </Button>
        </div>
      </div>
    )
  }

  if (!sessionQuery.data) {
    const next = `${location.pathname}${location.search}${location.hash}`
    return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />
  }

  return <Outlet />
}
