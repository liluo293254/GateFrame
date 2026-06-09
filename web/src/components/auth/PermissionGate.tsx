import type { ReactNode } from 'react'
import { useAuthStore } from '@/stores/auth'

type Props = {
  permission: string
  children: ReactNode
  fallback?: ReactNode
}

export function PermissionGate({ permission, children, fallback = null }: Props) {
  const hasPermission = useAuthStore((s) => s.hasPermission)
  if (!hasPermission(permission)) {
    return fallback
  }
  return children
}
