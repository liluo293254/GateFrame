import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PermissionGate } from '@/components/auth/PermissionGate'
import { AppLayout } from '@/components/layout/AppLayout'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { PERMISSIONS } from '@/config/permissions'
import { ApiError, createTenant, listTenants, updateTenant, type Tenant } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

type FormMode = 'create' | 'edit'

const STATUS_OPTIONS = ['active', 'disabled'] as const

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

export function TenantsPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [formMode, setFormMode] = useState<FormMode>('create')
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null)
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [status, setStatus] = useState<(typeof STATUS_OPTIONS)[number]>('active')
  const [formError, setFormError] = useState<string | null>(null)

  const tenantsQuery = useQuery({
    queryKey: ['tenants'],
    queryFn: () => listTenants(token),
  })

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (formMode === 'create') {
        return createTenant(token, { name, slug })
      }
      if (!editingTenant) throw new Error('missing tenant')
      return updateTenant(token, editingTenant.id, { name, status })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenants'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      setDialogOpen(false)
      setFormError(null)
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setFormError(t('tenants.forbidden'))
      } else {
        setFormError(t('tenants.saveFailed'))
      }
    },
  })

  function openCreate() {
    setFormMode('create')
    setEditingTenant(null)
    setName('')
    setSlug('')
    setStatus('active')
    setFormError(null)
    setDialogOpen(true)
  }

  function openEdit(tenant: Tenant) {
    setFormMode('edit')
    setEditingTenant(tenant)
    setName(tenant.name)
    setSlug(tenant.slug)
    setStatus(
      STATUS_OPTIONS.includes(tenant.status as (typeof STATUS_OPTIONS)[number])
        ? (tenant.status as (typeof STATUS_OPTIONS)[number])
        : 'active',
    )
    setFormError(null)
    setDialogOpen(true)
  }

  const tenants = tenantsQuery.data ?? []

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t('tenants.title')}</h1>
            <p className="text-muted-foreground">{t('tenants.subtitle')}</p>
          </div>
          <PermissionGate permission={PERMISSIONS.TENANT_MANAGE}>
            <Button onClick={openCreate}>{t('tenants.addTenant')}</Button>
          </PermissionGate>
        </div>

        {tenantsQuery.isLoading ? (
          <p className="text-muted-foreground">{t('common.loading')}</p>
        ) : tenantsQuery.isError ? (
          <p className="text-destructive">{t('tenants.loadFailed')}</p>
        ) : tenants.length === 0 ? (
          <p className="text-muted-foreground">{t('tenants.empty')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('tenants.name')}</TableHead>
                <TableHead>{t('tenants.slug')}</TableHead>
                <TableHead>{t('tenants.status')}</TableHead>
                <TableHead>{t('tenants.createdAt')}</TableHead>
                <TableHead>{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tenants.map((tenant) => (
                <TableRow key={tenant.id}>
                  <TableCell className="font-medium">{tenant.name}</TableCell>
                  <TableCell>{tenant.slug}</TableCell>
                  <TableCell>{t(`tenants.status${tenant.status === 'active' ? 'Active' : 'Disabled'}`)}</TableCell>
                  <TableCell>{formatTime(tenant.created_at)}</TableCell>
                  <TableCell>
                    <PermissionGate permission={PERMISSIONS.TENANT_MANAGE}>
                      <Button variant="outline" size="sm" onClick={() => openEdit(tenant)}>
                        {t('common.edit')}
                      </Button>
                    </PermissionGate>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTitle>
            {formMode === 'create' ? t('tenants.createTitle') : t('tenants.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {formMode === 'create' ? t('tenants.createHint') : t('tenants.editHint')}
          </DialogDescription>
          <form
            className="mt-4 space-y-4"
            onSubmit={(e) => {
              e.preventDefault()
              saveMutation.mutate()
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="tenant-name">{t('tenants.name')}</Label>
              <Input
                id="tenant-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            {formMode === 'create' ? (
              <div className="space-y-2">
                <Label htmlFor="tenant-slug">{t('tenants.slug')}</Label>
                <Input
                  id="tenant-slug"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  required
                  pattern="[a-z0-9-]+"
                />
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="tenant-status">{t('tenants.status')}</Label>
                <select
                  id="tenant-status"
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                  value={status}
                  onChange={(e) =>
                    setStatus(e.target.value as (typeof STATUS_OPTIONS)[number])
                  }
                >
                  {STATUS_OPTIONS.map((opt) => (
                    <option key={opt} value={opt}>
                      {t(`tenants.status${opt === 'active' ? 'Active' : 'Disabled'}`)}
                    </option>
                  ))}
                </select>
              </div>
            )}
            {formError ? <p className="text-sm text-destructive">{formError}</p> : null}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={saveMutation.isPending}>
                {saveMutation.isPending ? t('common.saving') : t('common.save')}
              </Button>
            </div>
          </form>
        </Dialog>
      </div>
    </AppLayout>
  )
}
