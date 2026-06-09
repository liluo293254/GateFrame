export const PERMISSIONS = {
  USER_READ: 'user.read',
  USER_CREATE: 'user.create',
  USER_UPDATE: 'user.update',
  USER_DELETE: 'user.delete',
  RBAC_READ: 'rbac.read',
  RBAC_MANAGE: 'rbac.manage',
  AUDIT_READ: 'audit.read',
  WORKFLOW_READ: 'workflow.read',
  WORKFLOW_MANAGE: 'workflow.manage',
  SEARCH_READ: 'search.read',
  SEARCH_MANAGE: 'search.manage',
  NOTIFICATION_READ: 'notification.read',
  NOTIFICATION_MANAGE: 'notification.manage',
  TENANT_READ: 'tenant.read',
  TENANT_MANAGE: 'tenant.manage',
  FILE_READ: 'file.read',
  FILE_MANAGE: 'file.manage',
} as const

export type PermissionCode = (typeof PERMISSIONS)[keyof typeof PERMISSIONS]
