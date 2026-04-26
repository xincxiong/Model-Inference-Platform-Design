# 全局状态管理使用指南

## 概述

我们使用 Zustand 实现了全局状态管理，解决了以下问题：

1. **API Key 同步** - 跨页面保持一致
2. **全局通知** - 统一的消息提示系统
3. **数据缓存** - 模型/部署/端点列表缓存
4. **跨页面数据同步** - 创建/更新/删除后自动刷新其他页面

## 安装

```bash
npm install zustand
```

## Store 结构

```typescript
// stores/appStore.ts

interface AppState {
  // API Key 管理
  apiKey: string
  setApiKey: (key: string) => void
  clearApiKey: () => void

  // 全局通知
  notifications: Notification[]
  addNotification: (notification) => void
  removeNotification: (id: string) => void

  // 模型缓存
  modelsCache: Model[] | null
  modelsLoading: boolean
  refreshModels: () => Promise<void>

  // 部署缓存
  deploymentsCache: Deployment[] | null
  refreshDeployments: () => Promise<void>
  updateDeployment: (id: string, deployment: Deployment) => void
  removeDeployment: (id: string) => void

  // 端点缓存
  endpointsCache: Endpoint[] | null
  refreshEndpoints: () => Promise<void>
  updateEndpoint: (id: string, endpoint: Endpoint) => void
  removeEndpoint: (id: string) => void

  // 全局加载
  globalLoading: boolean
  setGlobalLoading: (loading: boolean) => void

  // 数据同步
  lastUpdateTimestamp: number
  triggerUpdate: () => void
}
```

## 使用示例

### 1. API Key 管理

```typescript
// 在 api-keys/page.tsx 中使用
import { useApiKey } from '@/stores/appStore'

function APIKeysPage() {
  const { apiKey, setApiKey, clearApiKey } = useApiKey()

  const handleSave = (key: string) => {
    setApiKey(key)
    // 自动同步到 localStorage 和所有页面
  }

  return (
    <div>
      <input value={apiKey} onChange={(e) => handleSave(e.target.value)} />
    </div>
  )
}
```

### 2. 全局通知

```typescript
// 在任何页面中使用
import { useNotifications } from '@/stores/appStore'

function AnyPage() {
  const { addNotification } = useNotifications()

  const handleAction = async () => {
    try {
      await apiCall()
      addNotification({
        type: 'success',
        message: '操作成功！',
      })
    } catch (error) {
      addNotification({
        type: 'error',
        message: error.message,
      })
    }
  }
}
```

### 3. 数据缓存与同步

```typescript
// 在 deployments/page.tsx 中使用
import { useDeploymentsCache } from '@/stores/appStore'

function DeploymentsPage() {
  const {
    deploymentsCache,
    deploymentsLoading,
    refreshDeployments,
    updateDeployment,
    removeDeployment,
  } = useDeploymentsCache()

  // 首次加载
  useEffect(() => {
    if (!deploymentsCache) {
      refreshDeployments()
    }
  }, [])

  // 创建后更新
  const handleCreate = async (data) => {
    const newDeployment = await createDeployment(data)
    // 自动更新缓存
    await refreshDeployments()
    // 触发全局更新
    triggerUpdate()
  }

  // 乐观更新
  const handleUpdate = async (id, patch) => {
    // 立即更新缓存（乐观更新）
    const updated = { ...deploymentsCache.find(d => d.id === id), ...patch }
    updateDeployment(id, updated)

    try {
      const result = await patchDeployment(id, patch)
      updateDeployment(id, result)
    } catch (error) {
      // 失败时自动回滚（通过 refreshDeployments）
      await refreshDeployments()
      throw error
    }
  }
}
```

### 4. 跨页面数据监听

```typescript
// 在 models/page.tsx 中监听部署变化
import { useDataSync, useModelsCache } from '@/stores/appStore'

function ModelsPage() {
  const { lastUpdateTimestamp } = useDataSync()
  const { refreshModels } = useModelsCache()

  // 当其他页面触发更新时，刷新当前页面数据
  useEffect(() => {
    refreshModels()
  }, [lastUpdateTimestamp])
}
```

### 5. 全局加载状态

```typescript
import { useGlobalLoading } from '@/stores/appStore'

function SomeComponent() {
  const { globalLoading, setGlobalLoading } = useGlobalLoading()

  const handleSubmit = async () => {
    setGlobalLoading(true)
    try {
      await submitData()
    } finally {
      setGlobalLoading(false)
    }
  }
}
```

## 最佳实践

### 1. 使用便捷钩子

```typescript
// ✅ 推荐
const { apiKey, setApiKey } = useApiKey()

// ❌ 不推荐（会订阅整个 store）
const { apiKey, setApiKey } = useAppStore()
```

### 2. 乐观更新模式

```typescript
const handleDelete = async (id) => {
  // 1. 立即从缓存中移除（乐观删除）
  removeDeployment(id)

  try {
    // 2. 调用 API
    await deleteDeployment(id)
    addNotification({ type: 'success', message: '删除成功' })
  } catch (error) {
    // 3. 失败时恢复（自动触发 refresh）
    await refreshDeployments()
    addNotification({ type: 'error', message: '删除失败' })
  }
}
```

### 3. 错误处理

```typescript
const handleRefresh = async () => {
  try {
    await refreshDeployments()
  } catch (error) {
    addNotification({
      type: 'error',
      message: '刷新失败: ' + error.message,
    })
  }
}
```

### 4. 防止重复请求

```typescript
const handleRefresh = async () => {
  if (deploymentsLoading) return // 防止重复请求
  await refreshDeployments()
}
```

## 通知系统

通知组件已集成到 `layout.tsx`，会自动显示所有通知。

```typescript
const { addNotification } = useNotifications()

// 成功通知
addNotification({ type: 'success', message: '操作成功' })

// 错误通知
addNotification({ type: 'error', message: '操作失败' })

// 警告通知
addNotification({ type: 'warning', message: '请注意' })

// 信息通知
addNotification({ type: 'info', message: '提示信息' })
```

通知会自动在 5 秒后消失，或点击关闭按钮手动关闭。

## 数据持久化

只有 `apiKey` 会被持久化到 localStorage，其他状态（通知、缓存）在页面刷新后会重置。

如需持久化其他数据：

```typescript
partialize: (state) => ({
  apiKey: state.apiKey,
  // 添加需要持久化的字段
  someSetting: state.someSetting,
}),
```

## 迁移指南

### 从 Props Drilling 迁移

**Before:**
```typescript
// Parent
function Parent() {
  const [data, setData] = useState()
  return <Child data={data} setData={setData} />
}

// Child
function Child({ data, setData }) {
  return <GrandChild data={data} setData={setData} />
}
```

**After:**
```typescript
// Parent
function Parent() {
  const { data } = useSomeCache()
  return <Child />
}

// Child
function Child() {
  const { data, updateData } = useSomeCache()
  // 直接使用，无需 props
}
```

### 从 Context 迁移

**Before:**
```typescript
// Provider setup
const Context = createContext()

function Provider({ children }) {
  const [state, setState] = useState()
  return (
    <Context.Provider value={{ state, setState }}>
      {children}
    </Context.Provider>
  )
}

// Usage
function Component() {
  const { state, setState } = useContext(Context)
}
```

**After:**
```typescript
// No provider needed!

function Component() {
  const { state, setState } = useAppStore()
}
```

## 注意事项

1. **不要在 SSR 组件中使用** - Store 是 client-only，标记为 `'use client'`
2. **避免订阅整个 store** - 使用拆分后的便捷钩子
3. **正确处理错误** - 乐观更新失败时要恢复状态
4. **不要过度缓存** - 敏感数据不要缓存，实时性要求高的数据及时刷新
