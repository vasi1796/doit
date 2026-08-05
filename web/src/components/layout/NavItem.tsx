import { NavLink } from 'react-router'

export function NavItem({ to, label, icon, count, badgeTone = 'default' }: {
  to: string
  label: string
  icon: string
  count?: number
  badgeTone?: 'default' | 'danger'
}) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `relative flex items-center gap-3 px-3 min-h-[44px] rounded-[10px] text-[15px] transition-colors ${
          isActive
            ? 'text-accent font-medium'
            : 'text-text-primary hover:bg-black/[0.04]'
        }`
      }
    >
      {({ isActive }) => (
        <>
          {isActive && (
            <span
              aria-hidden="true"
              className="absolute inset-0 rounded-[10px] bg-accent-light animate-nav-active-in"
            />
          )}
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            className={`shrink-0 relative z-[1] ${isActive ? '' : 'text-text-secondary'}`}
          >
            <path d={icon} />
          </svg>
          <span className="flex-1 relative z-[1]">{label}</span>
          {count !== undefined && count > 0 && (
            badgeTone === 'danger' ? (
              <span className="relative z-[1] inline-flex items-center justify-center min-w-[20px] h-[20px] px-1.5 rounded-full text-[11px] font-semibold bg-danger text-white">
                {count}
              </span>
            ) : (
              <span className={`text-[12px] font-medium relative z-[1] ${isActive ? 'text-accent/70' : 'text-text-tertiary'}`}>{count}</span>
            )
          )}
        </>
      )}
    </NavLink>
  )
}
