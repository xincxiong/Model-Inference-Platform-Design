'use client'

import { useEffect } from 'react'
import { useNotifications } from '@/stores/appStore'

export function NotificationContainer() {
  const { notifications, removeNotification } = useNotifications()

  useEffect(() => {
    const timers = notifications.map((notification) =>
      setTimeout(() => {
        removeNotification(notification.id)
      }, 5000)
    )

    return () => {
      timers.forEach(clearTimeout)
    }
  }, [notifications, removeNotification])

  if (notifications.length === 0) return null

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {notifications.map((notification) => (
        <div
          key={notification.id}
          className={`
            flex items-start gap-3 px-4 py-3 rounded-lg shadow-lg border
            animate-in slide-in-from-right fade-in
            ${
              notification.type === 'success'
                ? 'bg-green-50 border-green-200 text-green-800'
                : notification.type === 'error'
                ? 'bg-red-50 border-red-200 text-red-800'
                : notification.type === 'warning'
                ? 'bg-amber-50 border-amber-200 text-amber-800'
                : 'bg-blue-50 border-blue-200 text-blue-800'
            }
          `}
        >
          <div className="flex-1">
            <p className="text-sm font-medium">{notification.message}</p>
          </div>
          <button
            onClick={() => removeNotification(notification.id)}
            className="text-current opacity-60 hover:opacity-100 transition-opacity"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      ))}
    </div>
  )
}
