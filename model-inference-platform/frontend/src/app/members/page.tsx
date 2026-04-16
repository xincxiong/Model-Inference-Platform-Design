'use client'

import { useEffect, useState } from 'react'
import { listMembers, inviteMember, patchMemberRole, removeMember } from '@/lib/api'

interface OrgMember {
  id: string
  user_id: string
  email: string
  name: string
  role: 'owner' | 'admin' | 'member' | 'viewer'
  status: 'active' | 'pending'
  invited_by: string
  joined_at?: string
  created_at: string
}

const roleConfig: Record<string, { label: string; cls: string }> = {
  owner:  { label: '所有者', cls: 'tag-purple' },
  admin:  { label: '管理员', cls: 'tag-blue'   },
  member: { label: '成员',   cls: 'tag-green'  },
  viewer: { label: '只读',   cls: 'tag-amber'  },
}

const statusConfig: Record<string, { label: string; cls: string }> = {
  active:  { label: '已加入', cls: 'tag-green' },
  pending: { label: '待接受', cls: 'tag-amber' },
}

const roles = ['owner', 'admin', 'member', 'viewer'] as const

// ── 邀请弹窗 ──────────────────────────────────────────────────────────────
function InviteModal({ onClose, onInvited }: {
  onClose: () => void
  onInvited: (m: OrgMember) => void
}) {
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<typeof roles[number]>('member')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit() {
    if (!email.trim()) { setError('请输入邮箱地址'); return }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) { setError('邮箱格式不正确'); return }
    setSubmitting(true); setError('')
    try {
      const m = await inviteMember(email.trim(), role)
      onInvited(m)
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '邀请失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-white rounded-2xl shadow-2xl w-[440px] overflow-hidden">
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-light)]">
          <div>
            <h2 className="text-base font-semibold text-[var(--text)]">邀请成员</h2>
            <p className="text-xs text-[var(--text-muted)] mt-0.5">通过邮箱邀请，成员接受后可使用平台资源</p>
          </div>
          <button onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-muted)]">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
        <div className="px-6 py-5 space-y-4">
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">邮箱地址 *</label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleSubmit()}
              placeholder="colleague@example.com"
              autoFocus
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">角色</label>
            <div className="grid grid-cols-4 gap-2">
              {roles.map(r => (
                <button
                  key={r}
                  onClick={() => setRole(r)}
                  className={`py-2 rounded-lg text-xs font-medium border transition-all ${
                    role === r
                      ? 'border-[var(--accent)] bg-[var(--accent-light)] text-[var(--accent)]'
                      : 'border-[var(--border-light)] text-[var(--text-secondary)] hover:border-[var(--border)]'
                  }`}
                >
                  {roleConfig[r].label}
                </button>
              ))}
            </div>
            <p className="text-[11px] text-[var(--text-muted)] mt-2">
              {role === 'owner' && '可管理所有资源和成员'}
              {role === 'admin' && '可管理资源，不可删除所有者'}
              {role === 'member' && '可使用 API 和工作空间资源'}
              {role === 'viewer' && '只读访问，不可调用 API'}
            </p>
          </div>
          {error && <p className="text-xs text-[var(--danger)]">{error}</p>}
        </div>
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-light)]">
          <button onClick={onClose} className="btn-outline">取消</button>
          <button onClick={handleSubmit} disabled={submitting} className="btn-primary flex items-center gap-2">
            {submitting ? (
              <><svg className="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>发送中...</>
            ) : '发送邀请'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── 主页面 ────────────────────────────────────────────────────────────────
export default function MembersPage() {
  const [members, setMembers] = useState<OrgMember[]>([])
  const [loading, setLoading] = useState(true)
  const [showInvite, setShowInvite] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editRole, setEditRole] = useState<string>('')
  const [savingId, setSavingId] = useState<string | null>(null)
  const [removingId, setRemovingId] = useState<string | null>(null)

  useEffect(() => {
    listMembers().then(r => setMembers(r.data ?? [])).catch(() => {}).finally(() => setLoading(false))
  }, [])

  async function handleRoleSave(id: string) {
    setSavingId(id)
    try {
      await patchMemberRole(id, editRole)
      setMembers(prev => prev.map(m => m.id === id ? { ...m, role: editRole as OrgMember['role'] } : m))
      setEditingId(null)
    } catch {}
    finally { setSavingId(null) }
  }

  async function handleRemove(id: string, email: string) {
    if (!confirm(`确认移除成员 ${email}？`)) return
    setRemovingId(id)
    try {
      await removeMember(id)
      setMembers(prev => prev.filter(m => m.id !== id))
    } finally {
      setRemovingId(null)
    }
  }

  const activeCount  = members.filter(m => m.status === 'active').length
  const pendingCount = members.filter(m => m.status === 'pending').length

  return (
    <div className="min-h-screen flex flex-col">
      {/* Top bar */}
      <div className="flex items-center justify-between px-8 py-5">
        <div>
          <h1 className="text-[15px] font-semibold text-[var(--text)]">成员管理</h1>
          {!loading && (
            <p className="text-xs text-[var(--text-muted)] mt-0.5">
              {activeCount} 位已加入{pendingCount > 0 ? `，${pendingCount} 位待接受` : ''}
            </p>
          )}
        </div>
        <button onClick={() => setShowInvite(true)} className="btn-primary flex items-center gap-1.5 text-sm py-2 px-4">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          邀请成员
        </button>
      </div>

      <div className="px-8 pb-8 flex-1">
        {/* Loading */}
        {loading && (
          <div className="flex items-center gap-2 text-[var(--text-muted)] text-sm py-12 justify-center">
            <svg className="animate-spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
            加载中...
          </div>
        )}

        {/* Empty */}
        {!loading && members.length === 0 && (
          <div className="flex flex-col items-center justify-center py-24 gap-4">
            <div className="w-16 h-16 rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-light)] flex items-center justify-center">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>
                <path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>
              </svg>
            </div>
            <div className="text-center">
              <p className="text-[15px] font-semibold text-[var(--text)]">还没有团队成员</p>
              <p className="text-sm text-[var(--text-muted)] mt-1">邀请同事加入，共同使用平台资源</p>
            </div>
            <button onClick={() => setShowInvite(true)} className="btn-primary mt-2 text-sm py-2 px-5">
              邀请第一位成员
            </button>
          </div>
        )}

        {/* Member list */}
        {!loading && members.length > 0 && (
          <div className="bg-white rounded-2xl border border-[var(--border-light)] overflow-hidden" style={{ boxShadow: 'var(--shadow-sm)' }}>
            <table>
              <thead>
                <tr className="border-b border-[var(--border-light)]">
                  <th className="px-5 py-3.5 text-left">成员</th>
                  <th className="px-5 py-3.5 text-left">角色</th>
                  <th className="px-5 py-3.5 text-left">状态</th>
                  <th className="px-5 py-3.5 text-left">加入时间</th>
                  <th className="px-5 py-3.5 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {members.map(m => {
                  const rc = roleConfig[m.role] ?? { label: m.role, cls: 'tag-blue' }
                  const sc = statusConfig[m.status] ?? { label: m.status, cls: 'tag-blue' }
                  const isEditing = editingId === m.id
                  return (
                    <tr key={m.id} className="hover:bg-[var(--bg-secondary)] transition-colors">
                      {/* 成员信息 */}
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-[var(--accent-light)] flex items-center justify-center text-sm font-semibold text-[var(--accent)]">
                            {(m.name || m.email).charAt(0).toUpperCase()}
                          </div>
                          <div>
                            {m.name && <p className="text-sm font-medium text-[var(--text)]">{m.name}</p>}
                            <p className={`text-xs ${m.name ? 'text-[var(--text-muted)]' : 'text-sm font-medium text-[var(--text)]'}`}>{m.email}</p>
                          </div>
                        </div>
                      </td>
                      {/* 角色 */}
                      <td className="px-5 py-4">
                        {isEditing ? (
                          <select
                            value={editRole}
                            onChange={e => setEditRole(e.target.value)}
                            className="w-28 text-xs py-1.5 px-2"
                          >
                            {roles.map(r => <option key={r} value={r}>{roleConfig[r].label}</option>)}
                          </select>
                        ) : (
                          <span className={`tag ${rc.cls}`}>{rc.label}</span>
                        )}
                      </td>
                      {/* 状态 */}
                      <td className="px-5 py-4">
                        <span className={`tag ${sc.cls}`}>{sc.label}</span>
                      </td>
                      {/* 时间 */}
                      <td className="px-5 py-4">
                        <span className="text-xs text-[var(--text-muted)]">
                          {m.joined_at
                            ? new Date(m.joined_at).toLocaleDateString('zh-CN')
                            : new Date(m.created_at).toLocaleDateString('zh-CN')}
                        </span>
                      </td>
                      {/* 操作 */}
                      <td className="px-5 py-4">
                        <div className="flex items-center justify-end gap-2">
                          {isEditing ? (
                            <>
                              <button
                                onClick={() => handleRoleSave(m.id)}
                                disabled={savingId === m.id}
                                className="text-xs text-[var(--accent)] hover:underline disabled:opacity-50"
                              >
                                {savingId === m.id ? '保存中...' : '保存'}
                              </button>
                              <button
                                onClick={() => setEditingId(null)}
                                className="text-xs text-[var(--text-muted)] hover:underline"
                              >
                                取消
                              </button>
                            </>
                          ) : (
                            <>
                              <button
                                onClick={() => { setEditingId(m.id); setEditRole(m.role) }}
                                className="text-xs text-[var(--accent)] hover:underline"
                              >
                                修改角色
                              </button>
                              <button
                                onClick={() => handleRemove(m.id, m.email)}
                                disabled={removingId === m.id}
                                className="text-xs text-[var(--danger)] hover:underline disabled:opacity-50"
                              >
                                移除
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {showInvite && (
        <InviteModal
          onClose={() => setShowInvite(false)}
          onInvited={m => setMembers(prev => [...prev, m as OrgMember])}
        />
      )}
    </div>
  )
}
