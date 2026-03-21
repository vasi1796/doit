interface EmptyStateProps {
  message?: string
  hint?: string
  action?: { label: string; onClick: () => void }
}

export function EmptyState({ message = 'No tasks', hint, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-[#86868b]">
      <svg
        width="48"
        height="48"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="mb-3 opacity-40"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="m9 12 2 2 4-4" />
      </svg>
      <p className="text-sm font-medium">{message}</p>
      {hint && <p className="text-xs mt-1 opacity-70">{hint}</p>}
      {action && (
        <button
          onClick={action.onClick}
          className="mt-3 text-sm text-[#007aff] font-medium min-h-[44px] px-4"
        >
          {action.label}
        </button>
      )}
    </div>
  )
}
