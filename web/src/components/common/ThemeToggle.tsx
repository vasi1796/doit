import { useTheme, type Theme } from '../../hooks/useTheme'

const OPTIONS: { value: Theme; label: string; icon: string }[] = [
  {
    value: 'light',
    label: 'Light',
    icon: 'M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42',
  },
  {
    value: 'system',
    label: 'System',
    icon: 'M20 3H4a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2zM8 21h8M12 17v4',
  },
  {
    value: 'dark',
    label: 'Dark',
    icon: 'M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z',
  },
]

/**
 * Segmented control for switching between light, system, and dark themes.
 * Renders as a horizontal pill row; labels are hidden on small screens.
 * Designed to live in the sidebar footer.
 */
export function ThemeToggle() {
  const { theme, setTheme } = useTheme()

  return (
    <div className="px-3">
      <div className="flex items-center bg-bg-secondary rounded-[10px] p-0.5 gap-0.5" role="radiogroup" aria-label="Theme">
        {OPTIONS.map((opt) => {
          const isActive = theme === opt.value
          return (
            <button
              key={opt.value}
              type="button"
              onClick={() => setTheme(opt.value)}
              role="radio"
              aria-checked={isActive}
              aria-label={`${opt.label} theme`}
              className={`flex-1 flex items-center justify-center gap-1.5 min-h-[32px] rounded-[8px] text-[12px] font-medium transition-all ${
                isActive
                  ? 'bg-bg-elevated text-text-primary shadow-card'
                  : 'text-text-tertiary hover:text-text-secondary'
              }`}
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                {opt.value === 'light' ? (
                  <>
                    <circle cx="12" cy="12" r="4" />
                    <path d={opt.icon} />
                  </>
                ) : (
                  <path d={opt.icon} />
                )}
              </svg>
              <span className="hidden sm:inline">{opt.label}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
