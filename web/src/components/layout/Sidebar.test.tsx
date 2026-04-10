import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import Sidebar from './Sidebar'
import { useUIStore } from '../../store/ui'

const { mockApi, mockUseAuthSession } = vi.hoisted(() => ({
  mockApi: {
    auth: {
      logout: vi.fn(),
    },
  },
  mockUseAuthSession: vi.fn(),
}))

vi.mock('../../api/client', () => ({
  api: mockApi,
}))

vi.mock('../../lib/auth', () => ({
  AUTH_SESSION_QUERY_KEY: ['auth-session'],
  useAuthSession: mockUseAuthSession,
}))

describe('Sidebar sign out', () => {
  afterEach(() => {
    cleanup()
  })

  beforeEach(() => {
    vi.clearAllMocks()

    mockApi.auth.logout.mockResolvedValue(undefined)
    mockUseAuthSession.mockReturnValue({
      data: {
        authenticated: true,
        username: 'vikto',
        expires_at: '2026-04-10T15:00:00Z',
      },
    })

    useUIStore.setState({
      sidebarCollapsed: false,
      selectedNodeId: null,
      toasts: [],
      showExecutionLog: false,
      activeLeaveConfirmation: null,
    })
  })

  it('cancels logout when the active leave guard declines it', async () => {
    const user = userEvent.setup()
    const confirmation = vi.fn().mockResolvedValue(false)
    useUIStore.setState({ activeLeaveConfirmation: confirmation })

    renderSidebar()

    await user.click(screen.getByRole('button', { name: /Sign out/i }))

    await waitFor(() => expect(confirmation).toHaveBeenCalledTimes(1))
    expect(mockApi.auth.logout).not.toHaveBeenCalled()
    expect(screen.getByTestId('location')).toHaveTextContent('/')
  })

  it('continues logout after the active leave guard allows it', async () => {
    const user = userEvent.setup()
    const confirmation = vi.fn().mockResolvedValue(true)
    useUIStore.setState({ activeLeaveConfirmation: confirmation })

    renderSidebar()

    await user.click(screen.getByRole('button', { name: /Sign out/i }))

    await waitFor(() => expect(confirmation).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(mockApi.auth.logout).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/login'))
  })
})

function renderSidebar() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: Infinity,
      },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="*" element={<Sidebar />} />
        </Routes>
        <LocationDisplay />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function LocationDisplay() {
  const location = useLocation()
  return <div data-testid="location">{location.pathname}</div>
}
