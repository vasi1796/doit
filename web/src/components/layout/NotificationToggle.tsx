import { useState, useEffect } from 'react'
import { useToast } from '../common/Toast'
import { isPushSupported, isPushSubscribed, subscribeToPush, unsubscribeFromPush } from '../../push'

export function NotificationToggle() {
  const { toast } = useToast()
  const [supported] = useState(isPushSupported)
  const [subscribed, setSubscribed] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (supported) {
      isPushSubscribed().then(setSubscribed)
    }
  }, [supported])

  if (!supported) return null

  const handleToggle = async () => {
    setLoading(true)
    try {
      if (subscribed) {
        await unsubscribeFromPush()
        setSubscribed(false)
        toast('Reminders disabled', 'success')
      } else {
        const ok = await subscribeToPush()
        if (ok) {
          setSubscribed(true)
          toast('Due date reminders enabled', 'success')
        } else {
          toast('Notification permission denied', 'error')
        }
      }
    } catch {
      toast('Failed to update notifications', 'error')
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
        className="flex items-center gap-3 px-3 min-h-[44px] rounded-[10px] text-[13px] text-text-secondary hover:bg-black/[0.04] w-full transition-colors"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" />
        </svg>
        <span className="flex-1 text-left">Reminders</span>
        <span
          className={`inline-block shrink-0 relative w-[44px] h-[26px] rounded-full transition-colors ${
            subscribed ? 'bg-accent' : 'bg-bg-tertiary'
          }`}
        >
          <span
            className={`absolute top-[3px] w-5 h-5 rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,0.2)] transition-[left] duration-200 ${
              subscribed ? 'left-[21px]' : 'left-[3px]'
            }`}
          />
        </span>
      </button>
    </div>
  )
}
