import { NavLink } from 'react-router'
import { useSyncStatus } from '../../hooks/useSyncStatus'

const TABS = [
  { to: '/inbox', label: 'Inbox', countKey: 'inbox', icon: 'M22 12h-6l-2 3h-4l-2-3H2M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z' },
  { to: '/today', label: 'Today', countKey: 'today', icon: 'M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z' },
  { to: '/upcoming', label: 'Upcoming', countKey: 'upcoming', icon: 'M13 17l5-5-5-5M6 17l5-5-5-5' },
  { to: '/completed', label: 'Done', countKey: null, icon: 'M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z' },
]

interface BottomNavProps {
  taskCounts: { inbox: number; today: number; upcoming: number }
  onMenuToggle: () => void
}

export function BottomNav({ taskCounts, onMenuToggle }: BottomNavProps) {
  const { state } = useSyncStatus()

  return (
    <nav
      className="frosted-bottom-nav fixed bottom-0 left-0 right-0 border-t border-separator flex md:hidden"
      style={{ paddingBottom: 'env(safe-area-inset-bottom, 0px)' }}
    >
      {state !== 'synced' && (
        <div className="absolute top-0 left-0 right-0 flex justify-center -translate-y-full pb-1">
          <span className={`text-[11px] font-medium px-2.5 py-0.5 rounded-full ${
            state === 'offline' ? 'bg-bg-tertiary text-text-secondary' :
            state === 'syncing' ? 'bg-accent-light text-accent' :
            'bg-warning/10 text-warning'
          }`}>
            {state === 'offline' ? 'Offline' : state === 'syncing' ? 'Syncing…' : 'Pending changes'}
          </span>
        </div>
      )}
      {TABS.map((tab) => {
        const count = tab.countKey ? taskCounts[tab.countKey as keyof typeof taskCounts] : 0
        const isTodayTab = tab.to === '/today'
        return (
          <NavLink
            key={tab.to}
            to={tab.to}
            className={({ isActive }) =>
              `flex-1 flex flex-col items-center justify-center min-h-[50px] pt-1.5 pb-1 transition-colors relative ${
                isActive ? 'text-accent' : 'text-text-secondary'
              }`
            }
          >
            <div className="relative">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <path d={tab.icon} />
              </svg>
              {count > 0 && (
                <span
                  className={`absolute -top-1.5 -right-2 text-white text-[9px] font-bold min-w-[16px] h-4 rounded-full flex items-center justify-center px-1 ${
                    isTodayTab ? 'bg-danger' : 'bg-accent'
                  }`}
                >
                  {count}
                </span>
              )}
            </div>
            <span className="text-[10px] mt-0.5 leading-tight">{tab.label}</span>
          </NavLink>
        )
      })}

      {/* More button — opens sidebar drawer with lists, labels, trash */}
      <button
        type="button"
        onClick={onMenuToggle}
        className="flex-1 flex flex-col items-center justify-center min-h-[50px] pt-1.5 pb-1 text-text-secondary"
        aria-label="More"
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <line x1="3" y1="12" x2="21" y2="12" />
          <line x1="3" y1="6" x2="21" y2="6" />
          <line x1="3" y1="18" x2="21" y2="18" />
        </svg>
        <span className="text-[10px] mt-0.5">More</span>
      </button>
    </nav>
  )
}
