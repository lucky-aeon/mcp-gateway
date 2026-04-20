import { create } from 'zustand'

interface AppState {
  sidebarOpen: boolean
  setSidebarOpen: (open: boolean) => void
  currentUser: {
    id: string
    email: string
    display_name: string
    role: string
    status: string
    builtin: boolean
    created_at: string
  } | null
  setCurrentUser: (user: AppState['currentUser']) => void
}

export const useAppStore = create<AppState>((set) => ({
  sidebarOpen: true,
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  currentUser: null,
  setCurrentUser: (user) => set({ currentUser: user }),
}))
