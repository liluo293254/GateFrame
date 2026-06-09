import { Navigate, Route, Routes } from 'react-router-dom'
import { PERMISSIONS } from '@/config/permissions'
import { ApiError } from '@/lib/api'
import { usePermissionsSync } from '@/hooks/usePermissionsSync'
import { useAuthStore } from '@/stores/auth'
import { AuditPage } from '@/pages/AuditPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { LoginPage } from '@/pages/LoginPage'
import { OidcCallbackPage } from '@/pages/OidcCallbackPage'
import { RolesPage } from '@/pages/RolesPage'
import { UsersPage } from '@/pages/UsersPage'
import { SearchPage } from '@/pages/SearchPage'
import { NotificationsPage } from '@/pages/NotificationsPage'
import { FilesPage } from '@/pages/FilesPage'
import { TenantsPage } from '@/pages/TenantsPage'
import { WorkflowsPage } from '@/pages/WorkflowsPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  const { isLoading, isError, error } = usePermissionsSync()
  if (!token) {
    return <Navigate to="/login" replace state={{ reason: 'session_expired' }} />
  }
  if (isError && error instanceof ApiError && error.status === 401) {
    return <Navigate to="/login" replace state={{ reason: 'session_expired' }} />
  }
  if (isLoading) {
    return null
  }
  return children
}

function PermissionRoute({
  permission,
  children,
}: {
  permission: string
  children: React.ReactNode
}) {
  const hasPermission = useAuthStore((s) => s.hasPermission)
  const { data, isLoading } = usePermissionsSync()
  if (isLoading) {
    return null
  }
  const allowed =
    (data?.permissions?.includes(permission) ?? false) ||
    (data?.permissions?.includes('*') ?? false) ||
    hasPermission(permission)
  if (!allowed) {
    return <Navigate to="/dashboard" replace />
  }
  return children
}

export function AppRoutes() {
  const token = useAuthStore((s) => s.token)

  return (
    <Routes>
      <Route
        path="/login"
        element={token ? <Navigate to="/dashboard" replace /> : <LoginPage />}
      />
      <Route path="/auth/oidc/callback" element={<OidcCallbackPage />} />
      <Route
        path="/dashboard"
        element={
          <ProtectedRoute>
            <DashboardPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/users"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.USER_READ}>
              <UsersPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route
        path="/roles"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.RBAC_READ}>
              <RolesPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route
        path="/audit"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.AUDIT_READ}>
              <AuditPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route
        path="/workflows"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.WORKFLOW_READ}>
              <WorkflowsPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route
        path="/search"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.SEARCH_READ}>
              <SearchPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route
        path="/notifications"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.NOTIFICATION_READ}>
              <NotificationsPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route
        path="/files"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.FILE_READ}>
              <FilesPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route
        path="/tenants"
        element={
          <ProtectedRoute>
            <PermissionRoute permission={PERMISSIONS.TENANT_READ}>
              <TenantsPage />
            </PermissionRoute>
          </ProtectedRoute>
        }
      />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
