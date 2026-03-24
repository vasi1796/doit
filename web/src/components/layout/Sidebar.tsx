import { useState, useEffect } from 'react'
import { NavLink, useNavigate } from 'react-router'
import { motion, LayoutGroup } from 'framer-motion'
import * as operations from '../../db/operations'
import { useToast } from '../common/Toast'
import { SyncStatus } from '../common/SyncStatus'
import { isPushSupported, isPushSubscribed, subscribeToPush, unsubscribeFromPush } from '../../push'
import { CalendarFeedLink } from '../common/CalendarFeedLink'
import type { List, Label } from '../../api/types'
import { PRESET_COLORS } from '../../constants'

const NAV_ITEMS = [
  { to: '/inbox', label: 'Inbox', icon: 'M22 12h-6l-2 3h-4l-2-3H2M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z' },
  { to: '/today', label: 'Today', icon: 'M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z' },
  { to: '/upcoming', label: 'Upcoming', icon: 'M13 17l5-5-5-5M6 17l5-5-5-5' },
  { to: '/matrix', label: 'Matrix', icon: 'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z' },
  { to: '/calendar', label: 'Calendar', icon: 'M8 2v4M16 2v4M3 10h18M21 6v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z' },
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
  onSearchOpen?: () => void
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
        <span style={{
          display: 'inline-block',
          flexShrink: 0,
          width: 44,
          height: 26,
          borderRadius: 13,
          backgroundColor: subscribed ? '#007aff' : '#d1d5db',
          position: 'relative',
          transition: 'background-color 0.2s',
        }}>
          <span style={{
            position: 'absolute',
            top: 3,
            left: subscribed ? 21 : 3,
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

function NavItem({ to, label, icon, count, layoutId = 'sidebar-active' }: { to: string; label: string; icon: string; count?: number; layoutId?: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `relative flex items-center gap-3 px-3 min-h-[44px] rounded-xl text-[14px] transition-colors ${
          isActive
            ? 'text-accent font-medium'
            : 'text-text-primary hover:bg-black/[0.03]'
        }`
      }
    >
      {({ isActive }) => (
        <>
          {isActive && (
            <motion.div
              layoutId={layoutId}
              className="absolute inset-0 bg-accent/10 rounded-xl"
              transition={{ type: 'spring', stiffness: 500, damping: 35 }}
            />
          )}
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="shrink-0 relative z-[1]">
            <path d={icon} />
          </svg>
          <span className="flex-1 relative z-[1]">{label}</span>
          {count !== undefined && count > 0 && (
            <span className="text-[11px] text-text-secondary font-medium relative z-[1]">{count}</span>
          )}
        </>
      )}
    </NavLink>
  )
}

export function Sidebar({ lists, labels, taskCounts, onSearchOpen }: SidebarProps) {
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

      {/* Search */}
      {onSearchOpen && (
        <div className="px-2 mb-1">
          <button
            type="button"
            onClick={onSearchOpen}
            className="flex items-center gap-3 px-3 min-h-[44px] rounded-xl text-[14px] text-text-secondary hover:bg-black/[0.03] w-full transition-colors"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="shrink-0">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
            <span className="flex-1 text-left">Search</span>
            <kbd className="hidden md:inline-block text-[11px] text-text-tertiary border border-gray-200 rounded px-1.5 py-0.5">⌘K</kbd>
          </button>
        </div>
      )}

      {/* Smart lists */}
      <LayoutGroup>
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
          <form onSubmit={handleCreateList} className="mx-1 mb-2 rounded-xl bg-white border border-gray-200 shadow-sm overflow-hidden">
            <div className="px-3 pt-3 pb-2">
              <input
                type="text"
                value={newListName}
                onChange={(e) => setNewListName(e.target.value)}
                placeholder="List name"
                className="w-full text-[16px] font-medium outline-none bg-transparent text-text-primary placeholder:text-text-tertiary"
                // eslint-disable-next-line jsx-a11y/no-autofocus
                autoFocus
              />
            </div>
            <div className="px-3 pb-2 flex items-center gap-1">
              {PRESET_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setNewListColour(c)}
                  className={`w-5 h-5 rounded-full ${newListColour === c ? 'ring-2 ring-offset-1 ring-accent/40' : ''}`}
                  style={{ backgroundColor: c }}
                  aria-label={`Color ${c}`}
                />
              ))}
            </div>
            <div className="flex border-t border-gray-100">
              <button
                type="button"
                onClick={() => { setNewListName(''); setAddingList(false) }}
                className="flex-1 text-sm text-text-secondary font-medium py-2.5 hover:bg-gray-50 transition-colors min-h-[44px]"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!newListName.trim()}
                className="flex-1 text-sm text-accent font-semibold py-2.5 hover:bg-accent/5 transition-colors border-l border-gray-100 min-h-[44px] disabled:opacity-30"
              >
                Create
              </button>
            </div>
          </form>
        )}
        <div className="space-y-0.5">
          {lists.map((list) => (
            <div key={list.id} className="group flex items-center">
              <NavLink
                to={`/lists/${list.id}`}
                className={({ isActive }) =>
                  `relative flex-1 flex items-center gap-3 px-3 min-h-[36px] rounded-lg text-[14px] transition-colors ${
                    isActive
                      ? 'text-accent font-medium'
                      : 'text-text-primary hover:bg-black/5'
                  }`
                }
              >
                {({ isActive }) => (
                  <>
                    {isActive && (
                      <motion.div
                        layoutId="sidebar-active"
                        className="absolute inset-0 bg-accent/10 rounded-lg"
                        transition={{ type: 'spring', stiffness: 500, damping: 35 }}
                      />
                    )}
                    <span
                      className="w-3 h-3 rounded-full shrink-0 relative z-[1]"
                      style={{ backgroundColor: list.colour || '#86868b' }}
                    />
                    <span className="flex-1 relative z-[1]">{list.name}</span>
                    {(taskCounts.byList[list.id] || 0) > 0 && (
                      <span className="text-[11px] text-text-secondary font-medium relative z-[1]">{taskCounts.byList[list.id]}</span>
                    )}
                  </>
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
                    `relative flex-1 flex items-center gap-3 px-3 min-h-[36px] rounded-lg text-[14px] transition-colors ${
                      isActive
                        ? 'text-accent font-medium'
                        : 'text-text-primary hover:bg-black/5'
                    }`
                  }
                >
                  {({ isActive }) => (
                    <>
                      {isActive && (
                        <motion.div
                          layoutId="sidebar-active"
                          className="absolute inset-0 bg-accent/10 rounded-lg"
                          transition={{ type: 'spring', stiffness: 500, damping: 35 }}
                        />
                      )}
                      <span
                        className="w-3 h-3 rounded-sm shrink-0 relative z-[1]"
                        style={{ backgroundColor: label.colour || '#86868b' }}
                      />
                      <span className="relative z-[1]">{label.name}</span>
                    </>
                  )}
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
      </LayoutGroup>

      {/* Sync status */}
      <SyncStatus />

      {/* Notification toggle */}
      <NotificationToggle />

      {/* Calendar feed */}
      <CalendarFeedLink />

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
