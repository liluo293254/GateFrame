import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type AuthState = {
  token: string | null
  userId: string | null
  tenantId: string | null
  username: string | null
  displayName: string | null
  permissions: string[]
  setSession: (data: {
    token: string
    userId: string
    tenantId: string
    username: string
    displayName: string
    permissions: string[]
  }) => void
  setPermissions: (permissions: string[]) => void
  updateSessionToken: (token: string, permissions: string[]) => void
  clearSession: () => void
  hasPermission: (code: string) => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      userId: null,
      tenantId: null,
      username: null,
      displayName: null,
      permissions: [],
      setSession: (data) =>
        set({
          token: data.token,
          userId: data.userId,
          tenantId: data.tenantId,
          username: data.username,
          displayName: data.displayName,
          permissions: data.permissions,
        }),
      setPermissions: (permissions) => set({ permissions }),
      updateSessionToken: (token, permissions) => set({ token, permissions }),
      clearSession: () =>
        set({
          token: null,
          userId: null,
          tenantId: null,
          username: null,
          displayName: null,
          permissions: [],
        }),
      hasPermission: (code) => {
        const perms = get().permissions
        return perms.includes(code) || perms.includes('*')
      },
    }),
    { name: 'gateframe_auth' },
  ),
)
