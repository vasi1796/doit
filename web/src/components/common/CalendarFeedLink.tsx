import { useState, useEffect } from 'react'
import { useToast } from './Toast'

export function CalendarFeedLink() {
  const { toast } = useToast()
  const [enabled, setEnabled] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/v1/ical/token', { credentials: 'include' })
      .then((res) => {
        setEnabled(res.ok)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const handleToggle = async () => {
    setLoading(true)
    try {
      if (enabled) {
        await fetch('/api/v1/ical/token', { method: 'DELETE', credentials: 'include' })
        setEnabled(false)
        toast('Calendar feed disabled', 'success')
      } else {
        const res = await fetch('/api/v1/ical/token', { method: 'POST', credentials: 'include' })
        if (!res.ok) throw new Error('Failed to enable calendar feed')
        const data = await res.json() as { url: string }
        await navigator.clipboard.writeText(data.url)
        setEnabled(true)
        toast('Calendar feed URL copied!', 'success')
      }
    } catch {
      toast('Failed to update calendar feed', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="px-2">
      <button
        type="button"
        onClick={handleToggle}
        disabled={loading}
        className="flex items-center gap-3 px-3 min-h-[44px] rounded-xl text-[13px] text-text-secondary hover:bg-black/[0.03] w-full transition-colors"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
          <line x1="16" y1="2" x2="16" y2="6" />
          <line x1="8" y1="2" x2="8" y2="6" />
          <line x1="3" y1="10" x2="21" y2="10" />
          <path d="M8 14h2v2H8z" />
        </svg>
        <span className="flex-1 text-left">Calendar Feed</span>
        <span style={{
          display: 'inline-block',
          flexShrink: 0,
          width: 44,
          height: 26,
          borderRadius: 13,
          backgroundColor: enabled ? '#007aff' : '#d1d5db',
          position: 'relative',
          transition: 'background-color 0.2s',
        }}>
          <span style={{
            position: 'absolute',
            top: 3,
            left: enabled ? 21 : 3,
            width: 20,
            height: 20,
            borderRadius: 10,
            backgroundColor: '#fff',
            boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
            transition: 'left 0.2s',
          }} />
        </span>
      </button>
    </div>
  )
}
