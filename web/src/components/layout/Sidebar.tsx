import { useState, useEffect } from 'react'
import { NavLink, useNavigate } from 'react-router'
import * as operations from '../../db/operations'
import { useToast } from '../common/Toast'
import { SyncStatus } from '../common/SyncStatus'
import { isPushSupported, isPushSubscribed, subscribeToPush, unsubscribeFromPush } from '../../push'
import type { List, Label } from '../../api/types'
import { PRESET_COLORS } from '../../constants'

const NAV_ITEMS = [
  { to: '/inbox', label: 'Inbox', icon: 'M22 12h-6l-2 3h-4l-2-3H2M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z' },
  { to: '/today', label: 'Today', icon: 'M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z' },
  { to: '/upcoming', label: 'Upcoming', icon: 'M13 17l5-5-5-5M6 17l5-5-5-5' },
]

const BOTTOM_ITEMS = [
  { to: '/completed', label: 'Completed', icon: 'M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z' },
  { to: '/trash', label: 'Trash', icon: 'M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16' },
]

interface SidebarProps {
  lists: List[]
  labels: Label[]
  taskCounts: {
    inbox: number
    today: number
    upcoming: number
    byList: Record<string, number>
  }
}

function NotificationToggle() {
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
        className="flex items-center gap-3 px-3 min-h-[44px] rounded-xl text-[13px] text-text-secondary hover:bg-black/[0.03] w-full transition-colors"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" />
        </svg>
        <span className="flex-1 text-left">Reminders</span>
        <span className={`w-8 h-5 rounded-full transition-colors relative ${subscribed ? 'bg-accent' : 'bg-gray-300'}`}>
          <span className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${subscribed ? 'translate-x-3.5' : 'translate-x-0.5'}`} />
        </span>
      </button>
    </div>
  )
}

function NavItem({ to, label, icon, count }: { to: string; label: string; icon: string; count?: number }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-3 px-3 min-h-[44px] rounded-xl text-[14px] transition-colors ${
          isActive
            ? 'bg-accent/10 text-accent font-medium'
            : 'text-text-primary hover:bg-black/[0.03]'
        }`
      }
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="shrink-0">
        <path d={icon} />
      </svg>
      <span className="flex-1">{label}</span>
      {count !== undefined && count > 0 && (
        <span className="text-[11px] text-text-secondary font-medium">{count}</span>
      )}
    </NavLink>
  )
}

export function Sidebar({ lists, labels, taskCounts }: SidebarProps) {
  const { toast } = useToast()
  const navigate = useNavigate()
  const [addingList, setAddingList] = useState(false)
  const [newListName, setNewListName] = useState('')
  const [newListColour, setNewListColour] = useState(PRESET_COLORS[0])

  const handleCreateList = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newListName.trim()) return
    try {
      await operations.createList({
        name: newListName.trim(),
        colour: newListColour,
        position: Date.now().toString(),
      })
      setNewListName('')
      setAddingList(false)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  return (
    <aside className="w-[280px] h-screen bg-[#f5f5f7] border-r border-gray-200 flex flex-col shrink-0 overflow-y-auto pb-[60px] md:pb-0">
      {/* App title */}
      <div className="px-4 pt-4 pb-2">
        <h1 className="text-lg font-semibold text-text-primary">DoIt</h1>
      </div>

      {/* Smart lists */}
      <nav className="px-2 space-y-0.5">
        {NAV_ITEMS.map((item) => {
          const countMap: Record<string, number> = {
            '/inbox': taskCounts.inbox,
            '/today': taskCounts.today,
            '/upcoming': taskCounts.upcoming,
          }
          return <NavItem key={item.to} {...item} count={countMap[item.to]} />
        })}
      </nav>

      {/* Divider */}
      <div className="mx-4 my-3 border-t border-gray-300" />

      {/* User lists */}
      <div className="px-2 flex-1">
        <div className="flex items-center justify-between px-3 mb-1">
          <p className="text-[11px] font-semibold text-text-secondary uppercase tracking-wider">Lists</p>
          <button
            type="button"
            onClick={() => setAddingList(!addingList)}
            className="text-text-secondary hover:text-accent transition-colors"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
              <path d="M12 5v14M5 12h14" />
            </svg>
          </button>
        </div>
        {addingList && (
          <form onSubmit={handleCreateList} className="px-3 mb-2 space-y-2">
            <input
              type="text"
              value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              placeholder="List name"
              className="w-full text-sm outline-none border-b border-gray-300 py-1 bg-transparent"
              // eslint-disable-next-line jsx-a11y/no-autofocus
              autoFocus
            />
            <div className="flex items-center gap-1">
              {PRESET_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setNewListColour(c)}
                  className={`w-5 h-5 rounded-full ${newListColour === c ? 'ring-2 ring-offset-1 ring-accent/30' : ''}`}
                  style={{ backgroundColor: c }}
                />
              ))}
              <button type="submit" className="ml-auto text-xs text-accent font-medium">Create</button>
            </div>
          </form>
        )}
        <div className="space-y-0.5">
          {lists.map((list) => (
            <div key={list.id} className="group flex items-center">
              <NavLink
                to={`/lists/${list.id}`}
                className={({ isActive }) =>
                  `flex-1 flex items-center gap-3 px-3 min-h-[36px] rounded-lg text-[14px] transition-colors ${
                    isActive
                      ? 'bg-accent/10 text-accent font-medium'
                      : 'text-text-primary hover:bg-black/5'
                  }`
                }
              >
                <span
                  className="w-3 h-3 rounded-full shrink-0"
                  style={{ backgroundColor: list.colour || '#86868b' }}
                />
                <span className="flex-1">{list.name}</span>
                {(taskCounts.byList[list.id] || 0) > 0 && (
                  <span className="text-[11px] text-text-secondary font-medium">{taskCounts.byList[list.id]}</span>
                )}
              </NavLink>
              <button
                type="button"
                onClick={async (e) => {
                  e.stopPropagation()
                  try {
                    await operations.deleteList(list.id)
                    toast('List deleted', 'success')
                    navigate('/inbox')
                  } catch (err) {
                    toast(err instanceof Error ? err.message : 'Failed to delete', 'error')
                  }
                }}
                className="opacity-0 group-hover:opacity-100 text-text-secondary hover:text-danger p-1 mr-1 transition-opacity"
                aria-label="Delete list"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Labels */}
      {labels.length > 0 && (
        <div className="px-2 mt-2">
          <div className="mx-2 mb-2 border-t border-gray-300" />
          <p className="px-3 text-[11px] font-semibold text-text-secondary uppercase tracking-wider mb-1">Labels</p>
          <div className="space-y-0.5">
            {labels.map((label) => (
              <div key={label.id} className="group flex items-center">
                <NavLink
                  to={`/labels/${label.id}`}
                  className={({ isActive }) =>
                    `flex-1 flex items-center gap-3 px-3 min-h-[36px] rounded-lg text-[14px] transition-colors ${
                      isActive
                        ? 'bg-accent/10 text-accent font-medium'
                        : 'text-text-primary hover:bg-black/5'
                    }`
                  }
                >
                  <span
                    className="w-3 h-3 rounded-sm shrink-0"
                    style={{ backgroundColor: label.colour || '#86868b' }}
                  />
                  <span>{label.name}</span>
                </NavLink>
                <button
                  type="button"
                  onClick={async (e) => {
                    e.stopPropagation()
                    try {
                      await operations.deleteLabel(label.id)
                      toast('Label deleted', 'success')
                      navigate('/inbox')
                    } catch (err) {
                      toast(err instanceof Error ? err.message : 'Failed to delete', 'error')
                    }
                  }}
                  className="opacity-0 group-hover:opacity-100 text-text-secondary hover:text-danger p-1 mr-1 transition-opacity"
                  aria-label="Delete label"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                    <path d="M18 6 6 18M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Bottom section */}
      <nav className="px-2 pb-2 space-y-0.5">
        <div className="mx-2 mb-2 border-t border-gray-300" />
        {BOTTOM_ITEMS.map((item) => (
          <NavItem key={item.to} {...item} />
        ))}
      </nav>

      {/* Sync status */}
      <SyncStatus />

      {/* Notification toggle */}
      <NotificationToggle />

      {/* Logout */}
      <div className="px-2 pb-4">
        <button
          type="button"
          onClick={async () => {
            try {
              await fetch('/auth/logout', { method: 'POST', credentials: 'include' })
              window.location.href = '/login'
            } catch (err) {
              toast(err instanceof Error ? err.message : 'Failed to sign out', 'error')
            }
          }}
          className="flex items-center gap-3 px-3 min-h-[44px] rounded-xl text-[13px] text-text-secondary hover:bg-black/[0.03] w-full transition-colors"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
          Sign out
        </button>
      </div>
    </aside>
  )
}
