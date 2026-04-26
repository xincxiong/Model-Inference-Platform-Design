'use client'

import { useEffect } from 'react'

export default function ErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error('Application error:', error)
  }, [error])

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--bg-secondary)] p-8">
      <div className="max-w-md w-full bg-white rounded-2xl shadow-xl p-8 text-center">
        <div className="w-16 h-16 bg-red-50 rounded-full flex items-center justify-center mx-auto mb-6">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#ef4444" strokeWidth="2" strokeLinecap="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>

        <h2 className="text-xl font-semibold text-[var(--text)] mb-2">
          出错了
        </h2>

        <p className="text-sm text-[var(--text-muted)] mb-6">
          {error.message || '应用遇到了意外错误，请重试'}
        </p>

        {error.digest && (
          <p className="text-xs text-[var(--text-muted)] mb-6 font-mono bg-gray-50 px-3 py-2 rounded">
            Error ID: {error.digest}
          </p>
        )}

        <div className="flex gap-3 justify-center">
          <button
            onClick={reset}
            className="btn-primary px-6 py-2.5"
          >
            重试
          </button>
          <button
            onClick={() => window.location.href = '/models'}
            className="btn-outline px-6 py-2.5"
          >
            返回首页
          </button>
        </div>
      </div>
    </div>
  )
}
