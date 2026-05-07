import { useState, useEffect, useCallback } from 'react'
import { X, CheckCircle, AlertCircle, AlertTriangle, Info } from 'lucide-react'

const ICONS = {
  success: CheckCircle,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
}

const STYLES = {
  success: 'bg-emerald-50 border-emerald-200 text-emerald-800',
  error: 'bg-red-50 border-red-200 text-red-800',
  warning: 'bg-amber-50 border-amber-200 text-amber-800',
  info: 'bg-blue-50 border-blue-200 text-blue-800',
}

const ICON_STYLES = {
  success: 'text-emerald-500',
  error: 'text-red-500',
  warning: 'text-amber-500',
  info: 'text-blue-500',
}

export default function ToastContainer() {
  const [toasts, setToasts] = useState([])

  const addToast = useCallback((message, type = 'error') => {
    const id = Date.now() + Math.random()
    setToasts(prev => {
      if (prev.some(t => t.message === message)) return prev
      return [...prev, { id, message, type }]
    })

    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id))
    }, 4000)
  }, [])

  const removeToast = useCallback((id) => {
    setToasts(prev => prev.filter(t => t.id !== id))
  }, [])

  useEffect(() => {
    const handler = (e) => {
      addToast(e.detail.message, e.detail.type)
    }
    window.addEventListener('loghub-toast', handler)
    return () => window.removeEventListener('loghub-toast', handler)
  }, [addToast])

  if (toasts.length === 0) return null

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {toasts.map(toast => {
        const Icon = ICONS[toast.type] || AlertCircle
        return (
          <div
            key={toast.id}
            className={`flex items-start gap-3 px-4 py-3 rounded-xl border shadow-lg toast-enter ${STYLES[toast.type] || STYLES.error}`}
          >
            <Icon className={`w-5 h-5 mt-0.5 shrink-0 ${ICON_STYLES[toast.type] || ICON_STYLES.error}`} />
            <p className="text-sm font-medium flex-1">{toast.message}</p>
            <button
              onClick={() => removeToast(toast.id)}
              className="shrink-0 p-0.5 rounded-lg hover:bg-black/5 transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        )
      })}
    </div>
  )
}

export function showToast(message, type = 'info') {
  window.dispatchEvent(new CustomEvent('loghub-toast', {
    detail: { message, type }
  }))
}
