import { useAuthStore } from '@/stores/auth'

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:3002'

function shouldClearSession(path: string, status: number): boolean {
  if (status !== 401 && status !== 403) return false
  if (path.startsWith('/api/auth/login')) return false
  if (path.startsWith('/api/auth/oidc')) return false
  // permissions/refresh recover stale JWT via usePermissionsSync; do not clear here
  if (path === '/api/auth/permissions' || path === '/api/auth/refresh') return false
  return true
}

export type ApiBody<T = unknown> = {
  code: string
  message?: string
  data?: T
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message?: string,
  ) {
    super(message ?? code)
  }
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit & { token?: string | null } = {},
): Promise<T> {
  const { token, headers, ...rest } = options
  const res = await fetch(`${API_BASE}${path}`, {
    ...rest,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
  })
  const body = (await res.json().catch(() => ({}))) as ApiBody<T>
  if (!res.ok) {
    if (shouldClearSession(path, res.status)) {
      useAuthStore.getState().clearSession()
    }
    throw new ApiError(res.status, body.code ?? 'error', body.message)
  }
  return body.data as T
}

export type LoginPayload = {
  username: string
  password: string
  tenant_slug?: string
}

export type LoginResult = {
  token: string
  user_id: string
  tenant_id: string
  username: string
  display_name: string
  permissions: string[]
}

export type PermissionsResult = {
  user_id: string
  tenant_id: string
  permissions: string[]
}

export function login(payload: LoginPayload) {
  return apiRequest<LoginResult>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export type OidcConfig = {
  enabled: boolean
}

export function fetchOidcConfig() {
  return apiRequest<OidcConfig>('/api/auth/oidc/config')
}

export function startOidcLogin() {
  const base = API_BASE.replace(/\/$/, '')
  window.location.href = `${base}/api/auth/oidc/login`
}

export function fetchPermissions(token: string) {
  return apiRequest<PermissionsResult>('/api/auth/permissions', { token })
}

export function refreshSession(token: string) {
  return apiRequest<LoginResult>('/api/auth/refresh', {
    method: 'POST',
    token,
    body: JSON.stringify({}),
  })
}

export type User = {
  id: string
  tenant_id: string
  username: string
  display_name: string
  status: string
  created_at: string
}

export type CreateUserPayload = {
  username: string
  password: string
  display_name?: string
}

export type UpdateUserPayload = {
  display_name?: string
  status?: 'active' | 'disabled'
  password?: string
}

export function listUsers(token: string) {
  return apiRequest<User[]>('/api/users', { token })
}

export function createUser(token: string, payload: CreateUserPayload) {
  return apiRequest<User>('/api/users', {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  })
}

export function updateUser(token: string, id: string, payload: UpdateUserPayload) {
  return apiRequest<User>(`/api/users/${id}`, {
    method: 'PUT',
    token,
    body: JSON.stringify(payload),
  })
}

export function deleteUser(token: string, id: string) {
  return apiRequest<{ deleted: boolean }>(`/api/users/${id}`, {
    method: 'DELETE',
    token,
  })
}

export type Role = {
  id: string
  tenant_id: string
  name: string
  description: string
}

export type Permission = {
  id: string
  code: string
  description: string
}

export type RolePermissionBinding = {
  role_id: string
  permissions: string[]
}

export function listRoles(token: string) {
  return apiRequest<Role[]>('/api/rbac/roles', { token })
}

export function listPermissions(token: string) {
  return apiRequest<Permission[]>('/api/rbac/permissions', { token })
}

export function listRolePermissions(token: string) {
  return apiRequest<RolePermissionBinding[]>('/api/rbac/role-permissions', { token })
}

export function getRolePermissions(token: string, roleId: string) {
  return apiRequest<Permission[]>(`/api/rbac/roles/${roleId}/permissions`, { token })
}

export function updateRolePermissions(
  token: string,
  roleId: string,
  permissions: string[],
) {
  return apiRequest<Permission[]>(`/api/rbac/roles/${roleId}/permissions`, {
    method: 'PUT',
    token,
    body: JSON.stringify({ permissions }),
  })
}

export type AuditEvent = {
  id: string
  tenant_id: string
  user_id?: string | null
  action: string
  path: string
  status_code: number
  created_at: string
}

export type AuditListParams = {
  limit?: number
  offset?: number
  action?: string
  path?: string
  from?: string
  to?: string
}

export type AuditListResult = {
  items: AuditEvent[]
  total: number
  limit: number
  offset: number
}

export function listAuditEvents(token: string, params: AuditListParams = {}) {
  const qs = new URLSearchParams()
  if (params.limit != null) qs.set('limit', String(params.limit))
  if (params.offset != null) qs.set('offset', String(params.offset))
  if (params.action) qs.set('action', params.action)
  if (params.path) qs.set('path', params.path)
  if (params.from) qs.set('from', params.from)
  if (params.to) qs.set('to', params.to)
  const query = qs.toString()
  const path = query ? `/api/audit?${query}` : '/api/audit'
  return apiRequest<AuditListResult>(path, { token })
}

export type Workflow = {
  id: string
  tenant_id: string
  name: string
  description: string
  status: string
  created_at: string
  updated_at: string
}

export function listWorkflows(token: string) {
  return apiRequest<Workflow[]>('/api/workflows', { token })
}

export type CreateWorkflowPayload = {
  name: string
  description?: string
  status?: string
}

export type UpdateWorkflowPayload = {
  name?: string
  description?: string
  status?: string
}

export function createWorkflow(token: string, payload: CreateWorkflowPayload) {
  return apiRequest<Workflow>('/api/workflows', {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  })
}

export function updateWorkflow(token: string, id: string, payload: UpdateWorkflowPayload) {
  return apiRequest<Workflow>(`/api/workflows/${id}`, {
    method: 'PUT',
    token,
    body: JSON.stringify(payload),
  })
}

export function deleteWorkflow(token: string, id: string) {
  return apiRequest<{ deleted: boolean; id: string }>(`/api/workflows/${id}`, {
    method: 'DELETE',
    token,
  })
}

export type SearchDocument = {
  id: string
  tenant_id: string
  title: string
  content: string
  created_at: string
}

export type SearchResult = {
  query: string
  items: SearchDocument[]
  total: number
}

export function searchDocuments(token: string, query: string) {
  const qs = query.trim() ? `?q=${encodeURIComponent(query.trim())}` : ''
  return apiRequest<SearchResult>(`/api/search${qs}`, { token })
}

export type Notification = {
  id: string
  tenant_id: string
  user_id?: string | null
  title: string
  body: string
  read_at?: string | null
  created_at: string
}

export function listNotifications(token: string) {
  return apiRequest<Notification[]>('/api/notifications', { token })
}

export type CreateNotificationPayload = {
  title: string
  body?: string
  user_id?: string
}

export function createNotification(token: string, payload: CreateNotificationPayload) {
  return apiRequest<Notification>('/api/notifications', {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  })
}

export function markNotificationRead(token: string, id: string) {
  return apiRequest<Notification>(`/api/notifications/${id}/read`, {
    method: 'PATCH',
    token,
    body: JSON.stringify({}),
  })
}

export type CreateDocumentPayload = {
  title: string
  content?: string
}

export function indexSearchDocument(token: string, payload: CreateDocumentPayload) {
  return apiRequest<SearchDocument>('/api/search/documents', {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  })
}

export function deleteSearchDocument(token: string, id: string) {
  return apiRequest<{ deleted: boolean; id: string }>(`/api/search/documents/${id}`, {
    method: 'DELETE',
    token,
  })
}

export type Tenant = {
  id: string
  name: string
  slug: string
  status: string
  created_at: string
}

export type CreateTenantPayload = {
  name: string
  slug: string
}

export function listTenants(token: string) {
  return apiRequest<Tenant[]>('/api/tenants', { token })
}

export function createTenant(token: string, payload: CreateTenantPayload) {
  return apiRequest<Tenant>('/api/tenants', {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  })
}

export type UpdateTenantPayload = {
  name?: string
  status?: 'active' | 'disabled'
}

export function updateTenant(token: string, id: string, payload: UpdateTenantPayload) {
  return apiRequest<Tenant>(`/api/tenants/${id}`, {
    method: 'PUT',
    token,
    body: JSON.stringify(payload),
  })
}

export type DashboardStats = {
  users?: number
  roles?: number
  workflows?: number
  search_documents?: number
  notifications?: number
  files?: number
  audit_events?: number
  tenants?: number
}

export function fetchDashboardStats(token: string) {
  return apiRequest<DashboardStats>('/api/dashboard/stats', { token })
}

export type FileObject = {
  id: string
  tenant_id: string
  object_key: string
  filename: string
  content_type: string
  size_bytes: number
  created_at: string
}

export type UploadFilePayload = {
  filename: string
  content_type?: string
  content_base64: string
}

export function listFiles(token: string) {
  return apiRequest<FileObject[]>('/api/files', { token })
}

export function uploadFile(token: string, payload: UploadFilePayload) {
  return apiRequest<FileObject>('/api/files', {
    method: 'POST',
    token,
    body: JSON.stringify(payload),
  })
}

export function uploadFileWithProgress(
  token: string,
  payload: UploadFilePayload,
  onProgress?: (percent: number) => void,
): Promise<FileObject> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    const body = JSON.stringify(payload)

    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable && onProgress) {
        onProgress(50 + Math.round((event.loaded / event.total) * 50))
      }
    }

    xhr.onload = () => {
      let parsed = { code: 'error' } as ApiBody<FileObject>
      try {
        parsed = JSON.parse(xhr.responseText) as ApiBody<FileObject>
      } catch {
        reject(new ApiError(xhr.status, 'error', 'invalid response'))
        return
      }

      if (xhr.status >= 200 && xhr.status < 300) {
        onProgress?.(100)
        resolve(parsed.data as FileObject)
        return
      }

      reject(new ApiError(xhr.status, parsed.code ?? 'error', parsed.message))
    }

    xhr.onerror = () => reject(new ApiError(0, 'network_error', 'network error'))
    xhr.open('POST', `${API_BASE}/api/files`)
    xhr.setRequestHeader('Content-Type', 'application/json')
    xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.send(body)
  })
}

export function deleteFile(token: string, id: string) {
  return apiRequest<{ deleted: boolean; id: string }>(`/api/files/${id}`, {
    method: 'DELETE',
    token,
  })
}

export async function downloadFileContent(token: string, id: string): Promise<Blob> {
  const res = await fetch(`${API_BASE}/api/files/${id}/content`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as ApiBody
    throw new ApiError(res.status, body.code ?? 'error', body.message)
  }
  return res.blob()
}
