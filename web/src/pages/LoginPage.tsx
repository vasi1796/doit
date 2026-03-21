import { useState } from 'react'
import { useNavigate } from 'react-router'

export function LoginPage() {
  const navigate = useNavigate()
  const [devEmail, setDevEmail] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const isDev = import.meta.env.DEV

  const handleGoogleLogin = () => {
    setLoading(true)
    window.location.href = '/auth/google/login'
  }

  const handleDevLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await fetch('/auth/dev', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: devEmail || 'dev@test.com' }),
      })
      if (!res.ok) {
        const body = await res.text()
        setError(body || 'Login failed')
        setLoading(false)
        return
      }
      navigate('/inbox')
    } catch {
      setError('Connection failed')
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gradient-to-b from-[#f5f5f7] to-[#e8e8ed] px-4">
      {/* Hero */}
      <div className="text-center mb-10">
        {/* App icon */}
        <div className="w-20 h-20 bg-accent rounded-[22px] mx-auto mb-6 flex items-center justify-center shadow-lg shadow-accent/20">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="m9 12 2 2 4-4" />
            <circle cx="12" cy="12" r="10" />
          </svg>
        </div>
        <h1 className="text-4xl font-bold text-text-primary tracking-tight">DoIt</h1>
        <p className="text-text-secondary text-lg mt-2 max-w-xs mx-auto leading-relaxed">
          Your tasks, organized.<br />Beautifully simple.
        </p>
      </div>

      {/* Card */}
      <div className="bg-white/80 backdrop-blur-xl rounded-2xl shadow-xl shadow-black/5 p-8 w-full max-w-sm">
        <button
          onClick={handleGoogleLogin}
          disabled={loading}
          className="w-full flex items-center justify-center gap-3 min-h-[50px] px-4 bg-text-primary text-white rounded-2xl font-semibold text-[15px] hover:bg-[#333] transition-all active:scale-[0.98] disabled:opacity-70"
        >
          <svg width="18" height="18" viewBox="0 0 24 24">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
          </svg>
          {loading ? 'Signing in...' : 'Continue with Google'}
        </button>

        {isDev && (
          <>
            <div className="my-6 flex items-center gap-3">
              <div className="flex-1 border-t border-gray-200/60" />
              <span className="text-[11px] text-text-secondary font-medium uppercase tracking-wider">Development</span>
              <div className="flex-1 border-t border-gray-200/60" />
            </div>
            <form onSubmit={handleDevLogin} className="space-y-3">
              <input
                type="email"
                value={devEmail}
                onChange={(e) => setDevEmail(e.target.value)}
                placeholder="dev@test.com"
                className="w-full min-h-[50px] px-4 bg-[#f5f5f7] border border-transparent rounded-2xl text-sm outline-none focus:border-accent focus:bg-white transition-all"
              />
              <button
                type="submit"
                disabled={loading}
                className="w-full min-h-[50px] px-4 bg-accent text-white rounded-2xl font-semibold text-[15px] hover:bg-accent/85 transition-all active:scale-[0.98] disabled:opacity-70"
              >
                {loading ? 'Signing in...' : 'Dev Login'}
              </button>
            </form>
          </>
        )}

        {error && (
          <div className="mt-4 text-center text-sm text-danger bg-danger/8 rounded-xl py-2 px-3">
            {error}
          </div>
        )}
      </div>

      {/* Footer */}
      <p className="mt-8 text-[12px] text-text-secondary">
        Personal · Self-hosted · Offline-first
      </p>
    </div>
  )
}
