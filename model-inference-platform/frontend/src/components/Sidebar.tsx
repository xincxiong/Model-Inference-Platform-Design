'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

type NavItem = {
  href: string
  label: string
  icon: React.ReactNode
}

type NavGroup = {
  title: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    title: '工作空间',
    items: [
      { href: '/models', label: '模型列表', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
      )},
      { href: '/playground', label: 'Playground', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
      )},
      { href: '/deployments', label: '模型部署', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
      )},
      { href: '/endpoints', label: '专属端点', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
      )},
      { href: '/usage', label: '用量统计', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
      )},
    ],
  },
  {
    title: '模型微调',
    items: [
      { href: '/finetuning', label: '微调任务', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M12 3v18"/><path d="M5 9h14"/><path d="M7 15h10"/><path d="M9 21h6"/></svg>
      )},
    ],
  },
  {
    title: '批量推理',
    items: [
      { href: '/batches', label: '批量任务', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="3" width="20" height="4" rx="1"/><rect x="2" y="10" width="20" height="4" rx="1"/><rect x="2" y="17" width="20" height="4" rx="1"/></svg>
      )},
    ],
  },
  {
    title: '数据实验室',
    items: [
      { href: '/datasets', label: '数据集', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v6c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/><path d="M3 11v6c0 1.66 4.03 3 9 3s9-1.34 9-3v-6"/></svg>
      )},
    ],
  },
  {
    title: '权限管理',
    items: [
      { href: '/api-keys', label: 'API Keys', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
      )},
      { href: '/members', label: '成员管理', icon: (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      )},
    ],
  },
]

export function Sidebar() {
  const pathname = usePathname()

  return (
    <aside className="fixed left-0 top-0 h-full w-60 bg-white border-r border-[var(--border-light)] flex flex-col z-10">
      <div className="px-6 py-5 flex items-center gap-3">
        <div className="w-8 h-8 rounded-lg bg-[var(--accent)] flex items-center justify-center">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
        </div>
        <div>
          <h1 className="text-sm font-semibold text-[var(--text)]">Inference Platform</h1>
          <p className="text-[11px] text-[var(--text-muted)]">Cloud Console</p>
        </div>
      </div>

      <div className="h-px bg-[var(--border-light)] mx-4" />

      <nav className="flex-1 px-3 py-3 overflow-y-auto">
        {navGroups.map((group, gi) => (
          <div key={group.title} className={gi > 0 ? 'mt-4' : ''}>
            <p className="px-3 mb-1 text-[10.5px] font-semibold uppercase tracking-wider text-[var(--text-muted)]">
              {group.title}
            </p>
            <div className="space-y-0.5">
              {group.items.map((item) => {
                const active = pathname === item.href
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`
                      flex items-center gap-2.5 px-3 py-2 rounded-lg text-[0.8125rem] font-medium transition-all
                      ${active
                        ? 'bg-[var(--accent-light)] text-[var(--accent)]'
                        : 'text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]'
                      }
                    `}
                  >
                    <span className={active ? 'text-[var(--accent)]' : 'text-[var(--text-muted)]'}>{item.icon}</span>
                    <span>{item.label}</span>
                  </Link>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="px-6 py-4 border-t border-[var(--border-light)]">
        <p className="text-[11px] text-[var(--text-muted)]">Phase 3 进行中 · v0.3.0 · GPU 虚拟化 & 跨域算力</p>
      </div>
    </aside>
  )
}
