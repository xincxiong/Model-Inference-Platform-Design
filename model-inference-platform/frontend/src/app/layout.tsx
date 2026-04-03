import type { Metadata } from 'next'
import './globals.css'
import { Sidebar } from '@/components/Sidebar'

export const metadata: Metadata = {
  title: 'Inference Platform Console',
  description: 'Model Inference Cloud Platform',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh">
      <body className="flex min-h-screen bg-[var(--bg-secondary)]">
        <Sidebar />
        <main className="flex-1 ml-60 p-8 max-w-[1400px]">{children}</main>
      </body>
    </html>
  )
}
