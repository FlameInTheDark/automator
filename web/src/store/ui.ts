import { create } from 'zustand'
import type { Toast } from '../types'

interface UIState {
  sidebarCollapsed: boolean
  toggleSidebar: () => void

  selectedNodeId: string | null
  setSelectedNodeId: (id: string | null) => void

  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => void
  removeToast: (id: string) => void

  showExecutionLog: boolean
  toggleExecutionLog: () => void
}

export const useUIStore = create<UIState>((set) => ({
  sidebarCollapsed: true,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

  selectedNodeId: null,
  setSelectedNodeId: (id) => set({ selectedNodeId: id }),

  toasts: [],
  addToast: (toast) => set((s) => ({
    toasts: [...s.toasts, { ...toast, id: crypto.randomUUID(), duration: toast.duration ?? 4000 }]
  })),
  removeToast: (id) => set((s) => ({
    toasts: s.toasts.filter((t) => t.id !== id)
  })),

  showExecutionLog: false,
  toggleExecutionLog: () => set((s) => ({ showExecutionLog: !s.showExecutionLog })),
}))
