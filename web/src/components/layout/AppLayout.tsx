import { Link, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { filterNavByPermissions, navItems } from '@/config/nav'
import { usePermissionsSync } from '@/hooks/usePermissionsSync'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { LanguageSwitcher } from '@/components/layout/LanguageSwitcher'
import { cn } from '@/lib/utils'

export function AppLayout({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation()
  const location = useLocation()
  const permissionsQuery = usePermissionsSync()
  const permissions = useAuthStore((s) => s.permissions)
  const displayName = useAuthStore((s) => s.displayName)
  const username = useAuthStore((s) => s.username)
  const clearSession = useAuthStore((s) => s.clearSession)

  const navPermissions = permissionsQuery.data?.permissions ?? permissions
  const items = filterNavByPermissions(navItems, navPermissions)

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-6">
            <span className="font-semibold">GateFrame</span>
            <nav className="flex flex-wrap gap-1">
              {items.map((item) => (
                <Link
                  key={item.path}
                  to={item.path}
                  className={cn(
                    'rounded-md px-3 py-2 text-sm transition-colors hover:bg-muted',
                    location.pathname === item.path && 'bg-muted font-medium',
                  )}
                >
                  {t(item.labelKey)}
                </Link>
              ))}
            </nav>
          </div>
          <div className="flex items-center gap-3">
            <LanguageSwitcher />
            <span className="text-sm text-muted-foreground">
              {displayName ?? username}
            </span>
            <Button variant="outline" size="sm" onClick={() => clearSession()}>
              {t('common.signOut')}
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">{children}</main>
    </div>
  )
}
