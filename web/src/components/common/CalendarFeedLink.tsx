import { useState, useEffect } from 'react'
import { useToast } from './Toast'

export function CalendarFeedLink() {
  const { toast } = useToast()
  const [feedUrl, setFeedUrl] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/v1/ical/token', { credentials: 'include' })
      .then(async (res) => {
        if (res.ok) {
          const data = await res.json() as { enabled: boolean; url?: string }
          if (data.enabled && data.url) setFeedUrl(data.url)
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const handleClick = async () => {
    setLoading(true)
    try {
      let url = feedUrl
      if (!url) {
        const res = await fetch('/api/v1/ical/token', { method: 'POST', credentials: 'include' })
        if (!res.ok) throw new Error('Failed to enable calendar feed')
        const data = await res.json() as { url: string }
        url = data.url
        setFeedUrl(url)
      }
      try {
        await navigator.clipboard.writeText(url)
      } catch {
        // Fallback for non-HTTPS or no focus — select text in a temp input
        const input = document.createElement('input')
        input.value = url
        document.body.appendChild(input)
        input.select()
        document.execCommand('copy')
        document.body.removeChild(input)
      }
      toast('Calendar feed URL copied!', 'success')
    } catch {
      toast('Failed to copy calendar feed URL', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="px-2">
      <button
        type="button"
        onClick={handleClick}
        disabled={loading}
        title={feedUrl ? 'Copy calendar feed URL' : 'Enable and copy calendar feed URL'}
        className="flex items-center gap-3 px-3 min-h-[44px] rounded-[10px] text-[13px] text-text-secondary hover:bg-black/[0.04] w-full transition-colors group"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
        <span className="flex-1 text-left">Calendar Feed</span>
        {feedUrl && (
          <span className="text-[10px] text-accent font-medium">Active</span>
        )}
      </button>
    </div>
  )
}
