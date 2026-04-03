'use client'

import { useState, useEffect } from 'react'
import { listApiKeys, createApiKey, deleteApiKey, setApiKey, getStoredApiKey } from '@/lib/api'

interface ApiKeyItem {
  id: string
  name: string
  key_prefix: string
  service_tier: string
  created_at: string
}

export default function ApiKeysPage() {
  const [keys, setKeys] = useState<ApiKeyItem[]>([])
  const [newName, setNewName] = useState('')
  const [newTier, setNewTier] = useState('default')
  const [createdKey, setCreatedKey] = useState('')
  const [currentKey, setCurrentKey] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    setCurrentKey(getStoredApiKey())
    loadKeys()
  }, [])

  const loadKeys = () => {
    listApiKeys()
      .then((data) => setKeys(data.api_keys || []))
      .catch((e) => setError(e.message))
  }

  const handleCreate = async () => {
    if (!newName) return
    try {
      const res = await createApiKey(newName, newTier)
      setCreatedKey(res.key)
      setNewName('')
      loadKeys()
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this API key?')) return
    try {
      await deleteApiKey(id)
      loadKeys()
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleSetActive = (key: string) => {
    setApiKey(key)
    setCurrentKey(key)
  }

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">API Keys</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">管理 API 密钥，用于鉴权和访问控制</p>
      </div>

      {/* Set active key */}
      <div className="card mb-4">
        <h3 className="font-medium text-sm text-[var(--text)] mb-1">Active API Key</h3>
        <p className="text-xs text-[var(--text-muted)] mb-3">
          设置当前控制台使用的 API Key
        </p>
        <div className="flex gap-3">
          <input
            value={currentKey}
            onChange={(e) => setCurrentKey(e.target.value)}
            placeholder="sk-..."
            type="password"
          />
          <button onClick={() => handleSetActive(currentKey)} className="btn-primary whitespace-nowrap">
            Save
          </button>
        </div>
      </div>

      {/* Create new key */}
      <div className="card mb-4">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">Create New Key</h3>
        <div className="flex gap-3 items-end">
          <div className="flex-1">
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Name</label>
            <input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="My API Key" />
          </div>
          <div className="w-36">
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Tier</label>
            <select value={newTier} onChange={(e) => setNewTier(e.target.value)}>
              <option value="default">default</option>
              <option value="flex">flex</option>
              <option value="auto">auto</option>
            </select>
          </div>
          <button onClick={handleCreate} className="btn-primary whitespace-nowrap">Create</button>
        </div>

        {createdKey && (
          <div className="mt-4 p-4 bg-[var(--success-bg)] rounded-lg border border-[var(--success)]">
            <p className="text-xs text-[var(--success)] font-medium mb-2">Key 已创建！请立即复制，之后将无法再次查看。</p>
            <div className="flex gap-2 items-center">
              <code className="text-xs flex-1 break-all bg-white/60 px-3 py-2 rounded-md text-[var(--text)]">{createdKey}</code>
              <button
                onClick={() => { navigator.clipboard.writeText(createdKey); handleSetActive(createdKey) }}
                className="btn-primary text-xs whitespace-nowrap"
              >
                Copy & Use
              </button>
            </div>
          </div>
        )}
      </div>

      {error && (
        <div className="text-[var(--danger)] bg-[var(--danger-bg)] text-sm px-4 py-3 rounded-lg mb-4">{error}</div>
      )}

      {/* Key list */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">Existing Keys</h3>
        {keys.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-4 text-center">尚未创建 API Key</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b border-[var(--border-light)]">
                <th className="pb-3 pr-4">Name</th>
                <th className="pb-3 pr-4">Prefix</th>
                <th className="pb-3 pr-4">Tier</th>
                <th className="pb-3 pr-4">Created</th>
                <th className="pb-3 w-20"></th>
              </tr>
            </thead>
            <tbody>
              {keys.map((k) => (
                <tr key={k.id}>
                  <td className="py-3 pr-4 font-medium text-[var(--text)]">{k.name}</td>
                  <td className="py-3 pr-4 font-mono text-xs text-[var(--text-muted)]">{k.key_prefix}...</td>
                  <td className="py-3 pr-4"><span className="tag tag-blue">{k.service_tier}</span></td>
                  <td className="py-3 pr-4 text-[var(--text-muted)]">
                    {new Date(k.created_at).toLocaleDateString()}
                  </td>
                  <td className="py-3">
                    <button onClick={() => handleDelete(k.id)} className="btn-danger">Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
