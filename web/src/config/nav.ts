import { PERMISSIONS } from '@/config/permissions'

export type NavItem = {
  path: string
  labelKey: string
  permission: string
}

export const navItems: NavItem[] = [
  { path: '/dashboard', labelKey: 'nav.dashboard', permission: PERMISSIONS.USER_READ },
  { path: '/users', labelKey: 'nav.users', permission: PERMISSIONS.USER_READ },
  { path: '/roles', labelKey: 'nav.roles', permission: PERMISSIONS.RBAC_READ },
  { path: '/audit', labelKey: 'nav.audit', permission: PERMISSIONS.AUDIT_READ },
  { path: '/workflows', labelKey: 'nav.workflows', permission: PERMISSIONS.WORKFLOW_READ },
  { path: '/search', labelKey: 'nav.search', permission: PERMISSIONS.SEARCH_READ },
  { path: '/notifications', labelKey: 'nav.notifications', permission: PERMISSIONS.NOTIFICATION_READ },
  { path: '/files', labelKey: 'nav.files', permission: PERMISSIONS.FILE_READ },
  { path: '/tenants', labelKey: 'nav.tenants', permission: PERMISSIONS.TENANT_READ },
]

export function filterNavByPermissions(
  items: NavItem[],
  permissions: string[],
): NavItem[] {
  const has = (code: string) =>
    permissions.includes(code) || permissions.includes('*')
  return items.filter((item) => has(item.permission))
}
