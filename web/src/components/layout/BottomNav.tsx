import { NavLink } from 'react-router'

const TABS = [
  { to: '/inbox', label: 'Inbox', countKey: 'inbox', icon: 'M22 12h-6l-2 3h-4l-2-3H2M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z' },
  { to: '/today', label: 'Today', countKey: 'today', icon: 'M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z' },
  { to: '/upcoming', label: 'Upcoming', countKey: 'upcoming', icon: 'M13 17l5-5-5-5M6 17l5-5-5-5' },
  { to: '/completed', label: 'Done', countKey: null, icon: 'M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z' },
  { to: '/trash', label: 'Trash', countKey: null, icon: 'M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16' },
]

interface BottomNavProps {
  taskCounts: { inbox: number; today: number; upcoming: number }
}

export function BottomNav({ taskCounts }: BottomNavProps) {
  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-white/95 backdrop-blur-sm border-t border-gray-200 flex md:hidden" style={{ paddingBottom: 'env(safe-area-inset-bottom, 0px)' }}>
      {TABS.map((tab) => {
        const count = tab.countKey ? taskCounts[tab.countKey as keyof typeof taskCounts] : 0
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
                <span className="absolute -top-1.5 -right-2 bg-accent text-white text-[9px] font-bold min-w-[16px] h-4 rounded-full flex items-center justify-center px-1">
                  {count}
                </span>
              )}
            </div>
            <span className="text-[10px] mt-0.5">{tab.label}</span>
          </NavLink>
        )
      })}
    </nav>
  )
}
