import { useState } from 'react'

const DISMISS_KEY = 'doit_install_banner_dismissed'

function shouldShow(): boolean {
  if (typeof window === 'undefined') return false
  const isStandalone = window.matchMedia('(display-mode: standalone)').matches
    || ('standalone' in navigator && (navigator as { standalone?: boolean }).standalone === true)
  if (isStandalone) return false
  if (localStorage.getItem(DISMISS_KEY)) return false
  return true
}

function isIOS(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent) && !('MSStream' in window)
}

export function InstallBanner() {
  const [visible, setVisible] = useState(shouldShow)

  if (!visible) return null

  const dismiss = () => {
    localStorage.setItem(DISMISS_KEY, '1')
    setVisible(false)
  }

  const instructions = isIOS()
    ? 'Tap the Share button, then "Add to Home Screen"'
    : 'In Safari, go to File \u2192 Add to Dock'

  return (
    <div className="bg-accent/10 border-b border-accent/20 px-4 py-3 flex items-center gap-3 text-[14px]">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="shrink-0 text-accent">
        <path d="M12 5v14M5 12l7-7 7 7" />
      </svg>
      <span className="flex-1 text-text-primary">
        <span className="font-medium">Install DoIt</span>
        <span className="text-text-secondary"> &mdash; {instructions}</span>
      </span>
      <button
        type="button"
        onClick={dismiss}
        aria-label="Dismiss install banner"
        className="w-[44px] h-[44px] -mr-2 flex items-center justify-center shrink-0 text-text-secondary hover:text-text-primary"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      </button>
    </div>
  )
}
