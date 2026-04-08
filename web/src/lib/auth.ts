import { useQuery } from '@tanstack/react-query'
import type { AuthSession } from '../types'
import { APIError, api } from '../api/client'

export const AUTH_SESSION_QUERY_KEY = ['auth-session'] as const

export function useAuthSession() {
  return useQuery<AuthSession | null, Error>({
    queryKey: AUTH_SESSION_QUERY_KEY,
    queryFn: async () => {
      try {
        return await api.auth.session()
      } catch (error) {
        if (error instanceof APIError && error.status === 401) {
          return null
        }
        throw error
      }
    },
    staleTime: 60_000,
    retry: false,
  })
}

export function normalizeNextPath(value: string | null | undefined): string {
  if (!value) {
    return '/'
  }

  return value.startsWith('/') && !value.startsWith('//')
    ? value
    : '/'
}
