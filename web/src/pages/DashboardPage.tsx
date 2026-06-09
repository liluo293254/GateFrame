import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePermissionsSync } from '@/hooks/usePermissionsSync'
import { AppLayout } from '@/components/layout/AppLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PERMISSIONS } from '@/config/permissions'
import { navItems } from '@/config/nav'
import { fetchDashboardStats, type DashboardStats } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

type StatCard = {
  key: keyof DashboardStats
  labelKey: string
  path?: string
  permission?: string
}

const statCards: StatCard[] = [
  { key: 'users', labelKey: 'dashboard.statsUsers', path: '/users', permission: PERMISSIONS.USER_READ },
  { key: 'roles', labelKey: 'dashboard.statsRoles', path: '/roles', permission: PERMISSIONS.RBAC_READ },
  { key: 'workflows', labelKey: 'dashboard.statsWorkflows', path: '/workflows', permission: PERMISSIONS.WORKFLOW_READ },
  { key: 'search_documents', labelKey: 'dashboard.statsSearch', path: '/search', permission: PERMISSIONS.SEARCH_READ },
  { key: 'notifications', labelKey: 'dashboard.statsNotifications', path: '/notifications', permission: PERMISSIONS.NOTIFICATION_READ },
  { key: 'files', labelKey: 'dashboard.statsFiles', path: '/files', permission: PERMISSIONS.FILE_READ },
  { key: 'audit_events', labelKey: 'dashboard.statsAudit', path: '/audit', permission: PERMISSIONS.AUDIT_READ },
  { key: 'tenants', labelKey: 'dashboard.statsTenants', path: '/tenants', permission: PERMISSIONS.TENANT_READ },
]

export function DashboardPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const displayName = useAuthStore((s) => s.displayName)
  const username = useAuthStore((s) => s.username)
  const permissions = useAuthStore((s) => s.permissions)
  const hasPermission = useAuthStore((s) => s.hasPermission)
  const { data: permData } = usePermissionsSync()

  const perms = permData?.permissions ?? permissions

  const statsQuery = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: () => fetchDashboardStats(token),
  })

  const visibleCards = statCards.filter((card) => {
    if (!card.permission) return true
    return hasPermission(card.permission) || perms.includes(card.permission) || perms.includes('*')
  })

  const allowedPaths = new Set(
    navItems
      .filter((item) => hasPermission(item.permission) || perms.includes(item.permission) || perms.includes('*'))
      .map((item) => item.path),
  )

  return (
    <AppLayout>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">{t('dashboard.title')}</h1>
        <p className="text-muted-foreground">
          {t('dashboard.welcome', { name: displayName ?? username ?? '' })}
        </p>
      </div>

      {statsQuery.isLoading ? (
        <p className="mb-6 text-muted-foreground">{t('common.loading')}</p>
      ) : statsQuery.isError ? (
        <p className="mb-6 text-destructive">{t('dashboard.statsLoadFailed')}</p>
      ) : (
        <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {visibleCards.map((card) => {
            const value = statsQuery.data?.[card.key]
            if (value == null) return null
            const content = (
              <Card className="h-full">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {t(card.labelKey)}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-3xl font-semibold tabular-nums">{value}</p>
                </CardContent>
              </Card>
            )
            if (card.path && allowedPaths.has(card.path)) {
              return (
                <Link key={card.key} to={card.path} className="block transition-opacity hover:opacity-90">
                  {content}
                </Link>
              )
            }
            return <div key={card.key}>{content}</div>
          })}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>{t('dashboard.permissionsTitle')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="list-inside list-disc text-sm">
            {perms.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </AppLayout>
  )
}
