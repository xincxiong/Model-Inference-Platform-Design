'use client'

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { fetchModels } from '@/lib/api'

interface Notification {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
  timestamp: number
}

interface Model {
  id: string
  name: string
  provider: string
  type: string
}

interface Deployment {
  id: string
  name: string
  status: string
  model_name: string
}

interface Endpoint {
  id: string
  name: string
  status: string
  model_name: string
  routing_key: string
}

interface AppState {
  apiKey: string
  setApiKey: (key: string) => void
  clearApiKey: () => void

  notifications: Notification[]
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void
  removeNotification: (id: string) => void
  clearNotifications: () => void

  modelsCache: Model[] | null
  modelsLoading: boolean
  modelsError: string | null
  refreshModels: () => Promise<void>
  clearModelsCache: () => void

  deploymentsCache: Deployment[] | null
  deploymentsLoading: boolean
  deploymentsError: string | null
  refreshDeployments: () => Promise<void>
  updateDeployment: (id: string, deployment: Deployment) => void
  removeDeployment: (id: string) => void
  clearDeploymentsCache: () => void

  endpointsCache: Endpoint[] | null
  endpointsLoading: boolean
  endpointsError: string | null
  refreshEndpoints: () => Promise<void>
  updateEndpoint: (id: string, endpoint: Endpoint) => void
  removeEndpoint: (id: string) => void
  clearEndpointsCache: () => void

  globalLoading: boolean
  setGlobalLoading: (loading: boolean) => void

  lastUpdateTimestamp: number
  triggerUpdate: () => void
}

const MAX_NOTIFICATIONS = 10

export const useAppStore = create<AppState>()(
  persist(
    (set, get) => ({
      apiKey: '',
      setApiKey: (key) => {
        localStorage.setItem('api_key', key)
        set({ apiKey: key })
      },
      clearApiKey: () => {
        localStorage.removeItem('api_key')
        set({ apiKey: '' })
      },

      notifications: [],
      addNotification: (notification) => {
        const id = Math.random().toString(36).substring(2, 9)
        set((state) => ({
          notifications: [
            ...state.notifications,
            { ...notification, id, timestamp: Date.now() },
          ].slice(-MAX_NOTIFICATIONS),
        }))
      },
      removeNotification: (id) => {
        set((state) => ({
          notifications: state.notifications.filter((n) => n.id !== id),
        }))
      },
      clearNotifications: () => set({ notifications: [] }),

      modelsCache: null,
      modelsLoading: false,
      modelsError: null,
      refreshModels: async () => {
        set({ modelsLoading: true, modelsError: null })
        try {
          const data = await fetchModels()
          set({ modelsCache: data.models || [], modelsLoading: false })
        } catch (error) {
          set({
            modelsError: error instanceof Error ? error.message : '获取模型列表失败',
            modelsLoading: false,
          })
        }
      },
      clearModelsCache: () => set({ modelsCache: null, modelsError: null }),

      deploymentsCache: null,
      deploymentsLoading: false,
      deploymentsError: null,
      refreshDeployments: async () => {
        set({ deploymentsLoading: true, deploymentsError: null })
        try {
          const response = await fetch('/v1/deployments')
          const data = await response.json()
          set({ deploymentsCache: data.data || [], deploymentsLoading: false })
        } catch (error) {
          set({
            deploymentsError: error instanceof Error ? error.message : '获取部署列表失败',
            deploymentsLoading: false,
          })
        }
      },
      updateDeployment: (id, deployment) => {
        set((state) => ({
          deploymentsCache: state.deploymentsCache?.map((d) =>
            d.id === id ? deployment : d
          ) || null,
        }))
      },
      removeDeployment: (id) => {
        set((state) => ({
          deploymentsCache: state.deploymentsCache?.filter((d) =>
            d.id !== id
          ) || null,
        }))
      },
      clearDeploymentsCache: () => set({ deploymentsCache: null, deploymentsError: null }),

      endpointsCache: null,
      endpointsLoading: false,
      endpointsError: null,
      refreshEndpoints: async () => {
        set({ endpointsLoading: true, endpointsError: null })
        try {
          const response = await fetch('/v0/dedicated_endpoints')
          const data = await response.json()
          set({ endpointsCache: data.data || [], endpointsLoading: false })
        } catch (error) {
          set({
            endpointsError: error instanceof Error ? error.message : '获取端点列表失败',
            endpointsLoading: false,
          })
        }
      },
      updateEndpoint: (id, endpoint) => {
        set((state) => ({
          endpointsCache: state.endpointsCache?.map((e) =>
            e.id === id ? endpoint : e
          ) || null,
        }))
      },
      removeEndpoint: (id) => {
        set((state) => ({
          endpointsCache: state.endpointsCache?.filter((e) =>
            e.id !== id
          ) || null,
        }))
      },
      clearEndpointsCache: () => set({ endpointsCache: null, endpointsError: null }),

      globalLoading: false,
      setGlobalLoading: (loading) => set({ globalLoading: loading }),

      lastUpdateTimestamp: 0,
      triggerUpdate: () => set({ lastUpdateTimestamp: Date.now() }),
    }),
    {
      name: 'inference-app-storage',
      partialize: (state) => ({
        apiKey: state.apiKey,
      }),
    }
  )
)

export const useApiKey = () => {
  const { apiKey, setApiKey, clearApiKey } = useAppStore()
  return { apiKey, setApiKey, clearApiKey }
}

export const useNotifications = () => {
  const { notifications, addNotification, removeNotification, clearNotifications } = useAppStore()
  return { notifications, addNotification, removeNotification, clearNotifications }
}

export const useModelsCache = () => {
  const { modelsCache, modelsLoading, modelsError, refreshModels, clearModelsCache } = useAppStore()
  return { modelsCache, modelsLoading, modelsError, refreshModels, clearModelsCache }
}

export const useDeploymentsCache = () => {
  const {
    deploymentsCache,
    deploymentsLoading,
    deploymentsError,
    refreshDeployments,
    updateDeployment,
    removeDeployment,
    clearDeploymentsCache,
  } = useAppStore()
  return {
    deploymentsCache,
    deploymentsLoading,
    deploymentsError,
    refreshDeployments,
    updateDeployment,
    removeDeployment,
    clearDeploymentsCache,
  }
}

export const useEndpointsCache = () => {
  const {
    endpointsCache,
    endpointsLoading,
    endpointsError,
    refreshEndpoints,
    updateEndpoint,
    removeEndpoint,
    clearEndpointsCache,
  } = useAppStore()
  return {
    endpointsCache,
    endpointsLoading,
    endpointsError,
    refreshEndpoints,
    updateEndpoint,
    removeEndpoint,
    clearEndpointsCache,
  }
}

export const useGlobalLoading = () => {
  const { globalLoading, setGlobalLoading } = useAppStore()
  return { globalLoading, setGlobalLoading }
}

export const useDataSync = () => {
  const { lastUpdateTimestamp, triggerUpdate } = useAppStore()
  return { lastUpdateTimestamp, triggerUpdate }
}
