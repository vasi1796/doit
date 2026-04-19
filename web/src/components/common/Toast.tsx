import { createContext, useCallback, useContext, useState } from 'react'

type ToastType = 'success' | 'error' | 'info'

interface ToastAction {
  label: string
  onClick: () => void
}

interface Toast {
  id: number
  message: string
  type: ToastType
  action?: ToastAction
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType, action?: ToastAction) => void
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} })

export function useToast() {
  return useContext(ToastContext)
}

let nextId = 0

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const toast = useCallback((message: string, type: ToastType = 'info', action?: ToastAction) => {
    const id = nextId++
    setToasts((prev) => [...prev, { id, message, type, action }])
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, action ? 5000 : 3000)
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-20 md:bottom-6 left-1/2 -translate-x-1/2 z-[100] flex flex-col gap-2 pointer-events-none">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`px-4 py-2.5 rounded-[14px] text-sm font-medium shadow-modal animate-[toast-in_0.2s_ease-out] pointer-events-auto flex items-center gap-3 ${
              t.type === 'error'
                ? 'bg-danger text-white'
                : t.type === 'info'
                  ? 'bg-bg-elevated text-text-primary border border-separator'
                  : 'bg-text-primary text-bg'
            }`}
          >
            {t.message}
            {t.action && (
              <button
                onClick={() => {
                  t.action!.onClick()
                  setToasts((prev) => prev.filter((toast) => toast.id !== t.id))
                }}
                className="font-semibold underline underline-offset-2 whitespace-nowrap min-h-[44px] min-w-[44px] flex items-center justify-center"
              >
                {t.action.label}
              </button>
            )}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
