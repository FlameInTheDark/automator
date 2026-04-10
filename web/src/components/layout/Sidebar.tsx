import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { 
  LayoutDashboard, GitBranch, Settings, MessageSquare, Copy,
  ChevronLeft, ChevronRight, LogOut, Gem
} from 'lucide-react'
import { cn } from '../../lib/utils'
import { useUIStore } from '../../store/ui'
import { api } from '../../api/client'
import { AUTH_SESSION_QUERY_KEY, useAuthSession } from '../../lib/auth'
import type { AuthSession } from '../../types'

const navItems = [
  { icon: LayoutDashboard, label: 'Dashboard', path: '/' },
  { icon: GitBranch, label: 'Pipelines', path: '/pipelines' },
  { icon: Copy, label: 'Templates', path: '/templates' },
  { icon: MessageSquare, label: 'LLM Chat', path: '/chat' },
  { icon: Settings, label: 'Settings', path: '/settings' },
]

export default function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const sessionQuery = useAuthSession()
  const { sidebarCollapsed, toggleSidebar, addToast, requestActiveLeaveConfirmation } = useUIStore()
  const username = sessionQuery.data?.username?.trim() ?? ''
  const displayName = username || 'Unknown user'
  const userInitial = displayName.charAt(0).toUpperCase() || '?'

  async function handleLogout() {
    try {
      const canLeave = await requestActiveLeaveConfirmation()
      if (!canLeave) {
        return
      }

      await api.auth.logout()
      queryClient.setQueryData<AuthSession | null>(AUTH_SESSION_QUERY_KEY, null)
      navigate('/login', { replace: true })
    } catch (err) {
      addToast({
        type: 'error',
        title: 'Failed to sign out',
        message: err instanceof Error ? err.message : 'Unknown error',
      })
    }
  }

  return (
    <div className={cn(
      'flex flex-col overflow-hidden bg-bg-elevated border-r border-border transition-all duration-300',
      sidebarCollapsed ? 'w-16' : 'w-64'
    )}>
      <div className={cn(
        'flex h-14 items-center border-b border-border',
        sidebarCollapsed ? 'justify-center px-2' : 'px-4',
      )}>
        <div className="flex min-w-0 items-center gap-3 overflow-hidden">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl border border-accent/20 bg-accent/12 text-accent shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]">
            <Gem className="h-4 w-4" />
          </div>
          {!sidebarCollapsed && (
            <span className="truncate whitespace-nowrap text-lg font-semibold tracking-[0.02em] text-text">Emerald</span>
          )}
        </div>
      </div>

      <nav className="flex-1 py-4 px-2 space-y-1">
        {navItems.map(({ icon: Icon, label, path }) => {
          const isActive = location.pathname === path || (path !== '/' && location.pathname.startsWith(path))
          return (
            <Link
              key={path}
              to={path}
              className={cn(
                'flex items-center gap-3 overflow-hidden px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                isActive
                  ? 'bg-accent/10 text-accent'
                  : 'text-text-muted hover:bg-bg-overlay hover:text-text'
              )}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!sidebarCollapsed && <span className="truncate whitespace-nowrap">{label}</span>}
            </Link>
          )
        })}
      </nav>

      <div className="space-y-2 border-t border-border p-2">
        {sessionQuery.data && (
          sidebarCollapsed ? (
            <div className="flex justify-center">
              <div
                className="flex h-12 w-12 items-center justify-center rounded-2xl border border-border bg-bg-input/70 shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]"
                title={displayName}
                aria-label={`Signed in as ${displayName}`}
              >
                <div className="flex h-9 w-9 items-center justify-center rounded-full border border-accent/30 bg-accent/12 text-sm font-semibold text-accent">
                  {userInitial}
                </div>
              </div>
            </div>
          ) : (
            <div className="overflow-hidden rounded-xl border border-border bg-bg-input/70 p-3 shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]">
              <div className="flex items-center gap-3 min-w-0">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-accent/30 bg-accent/12 text-sm font-semibold text-accent">
                  {userInitial}
                </div>
                <div className="min-w-0">
                  <div className="truncate whitespace-nowrap text-[11px] uppercase tracking-[0.18em] text-text-dimmed">Signed in</div>
                  <div className="mt-1 truncate text-sm font-semibold text-text">
                    {displayName}
                  </div>
                </div>
              </div>
            </div>
          )
        )}
        <button
          onClick={() => void handleLogout()}
          title={sidebarCollapsed ? 'Sign out' : undefined}
          className="flex w-full items-center gap-3 overflow-hidden rounded-lg px-3 py-2.5 text-sm text-text-muted transition-colors hover:bg-bg-overlay hover:text-text"
        >
          {sidebarCollapsed
            ? <LogOut className="w-5 h-5 mx-auto" />
            : <><LogOut className="w-5 h-5 shrink-0" /><span className="truncate whitespace-nowrap">Sign out</span></>
          }
        </button>
        <button
          onClick={toggleSidebar}
          title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="flex w-full items-center gap-3 overflow-hidden rounded-lg px-3 py-2.5 text-sm text-text-muted transition-colors hover:bg-bg-overlay hover:text-text"
        >
          {sidebarCollapsed 
            ? <ChevronRight className="w-5 h-5 mx-auto" />
            : <><ChevronLeft className="w-5 h-5 shrink-0" /><span className="truncate whitespace-nowrap">Collapse</span></>
          }
        </button>
      </div>
    </div>
  )
}
