import { Link, useLocation } from 'react-router-dom'
import { 
  LayoutDashboard, GitBranch, Settings, MessageSquare, 
  ChevronLeft, ChevronRight
} from 'lucide-react'
import { cn } from '../../lib/utils'
import { useUIStore } from '../../store/ui'

const navItems = [
  { icon: LayoutDashboard, label: 'Dashboard', path: '/' },
  { icon: GitBranch, label: 'Pipelines', path: '/pipelines' },
  { icon: MessageSquare, label: 'LLM Chat', path: '/chat' },
  { icon: Settings, label: 'Settings', path: '/settings' },
]

export default function Sidebar() {
  const location = useLocation()
  const { sidebarCollapsed, toggleSidebar } = useUIStore()

  return (
    <div className={cn(
      'flex flex-col bg-bg-elevated border-r border-border transition-all duration-300',
      sidebarCollapsed ? 'w-16' : 'w-64'
    )}>
      <div className="flex items-center h-14 px-4 border-b border-border">
        {!sidebarCollapsed && (
          <span className="text-lg font-bold text-accent">Automator</span>
        )}
        {sidebarCollapsed && (
          <span className="text-lg font-bold text-accent mx-auto">A</span>
        )}
      </div>

      <nav className="flex-1 py-4 px-2 space-y-1">
        {navItems.map(({ icon: Icon, label, path }) => {
          const isActive = location.pathname === path || (path !== '/' && location.pathname.startsWith(path))
          return (
            <Link
              key={path}
              to={path}
              className={cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                isActive
                  ? 'bg-accent/10 text-accent'
                  : 'text-text-muted hover:bg-bg-overlay hover:text-text'
              )}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!sidebarCollapsed && <span>{label}</span>}
            </Link>
          )
        })}
      </nav>

      <div className="p-2 border-t border-border">
        <button
          onClick={toggleSidebar}
          className="flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm text-text-muted hover:text-text hover:bg-bg-overlay transition-colors"
        >
          {sidebarCollapsed 
            ? <ChevronRight className="w-5 h-5 mx-auto" />
            : <><ChevronLeft className="w-5 h-5" /><span>Collapse</span></>
          }
        </button>
      </div>
    </div>
  )
}
