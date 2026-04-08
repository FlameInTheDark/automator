import { FormEvent, useState } from 'react'
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { LockKeyhole, Shield } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import Button from '../components/ui/Button'
import Input from '../components/ui/Input'
import { Label } from '../components/ui/Form'
import { APIError, api } from '../api/client'
import { AUTH_SESSION_QUERY_KEY, normalizeNextPath, useAuthSession } from '../lib/auth'
import type { AuthSession } from '../types'

export default function Login() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const sessionQuery = useAuthSession()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('admin')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const nextPath = normalizeNextPath(searchParams.get('next'))

  if (sessionQuery.data) {
    return <Navigate to={nextPath} replace />
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setError(null)

    try {
      const session = await api.auth.login({ username, password })
      queryClient.setQueryData<AuthSession | null>(AUTH_SESSION_QUERY_KEY, session)
      navigate(nextPath, { replace: true })
    } catch (err) {
      if (err instanceof APIError && err.status === 401) {
        setError('Invalid username or password.')
      } else if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Failed to sign in.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-bg px-4 py-10">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(45,212,191,0.12),_transparent_38%),radial-gradient(circle_at_bottom_right,_rgba(16,185,129,0.10),_transparent_34%)]" />
      <Card className="relative w-full max-w-md overflow-hidden border-border/90 bg-bg-elevated/95 shadow-2xl">
        <CardHeader className="border-border/80 bg-bg-overlay/60">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-accent/15 text-accent">
              <Shield className="h-5 w-5" />
            </div>
            <div>
              <CardTitle>Sign In</CardTitle>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-5 pt-6">
          <form className="space-y-4" onSubmit={handleSubmit}>
            <div>
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                autoComplete="username"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="admin"
              />
            </div>
            <div>
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="admin"
              />
            </div>
            {error && (
              <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                {error}
              </div>
            )}
            {sessionQuery.error && (
              <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-200">
                {sessionQuery.error.message}
              </div>
            )}
            <Button type="submit" className="w-full" loading={isSubmitting}>
              <LockKeyhole className="h-4 w-4" />
              Sign In
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
