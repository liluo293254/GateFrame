import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PermissionGate } from '@/components/auth/PermissionGate'
import { AppLayout } from '@/components/layout/AppLayout'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { PERMISSIONS } from '@/config/permissions'
import {
  ApiError,
  listPermissions,
  listRolePermissions,
  listRoles,
  updateRolePermissions,
} from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

export function RolesPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRoleId, setEditingRoleId] = useState<string | null>(null)
  const [editingRoleName, setEditingRoleName] = useState('')
  const [selectedCodes, setSelectedCodes] = useState<string[]>([])
  const [formError, setFormError] = useState<string | null>(null)

  const rolesQuery = useQuery({
    queryKey: ['roles'],
    queryFn: () => listRoles(token),
  })

  const bindingsQuery = useQuery({
    queryKey: ['role-permissions'],
    queryFn: () => listRolePermissions(token),
  })

  const permissionsQuery = useQuery({
    queryKey: ['permissions-catalog'],
    queryFn: () => listPermissions(token),
  })

  const permissionDescriptions = useMemo(() => {
    const map = new Map<string, string>()
    for (const p of permissionsQuery.data ?? []) {
      map.set(p.code, p.description)
    }
    return map
  }, [permissionsQuery.data])

  const permissionsByRole = useMemo(() => {
    const map = new Map<string, string[]>()
    for (const binding of bindingsQuery.data ?? []) {
      map.set(binding.role_id, binding.permissions)
    }
    return map
  }, [bindingsQuery.data])

  const saveMutation = useMutation({
    mutationFn: () => {
      if (!editingRoleId) throw new Error('missing role')
      return updateRolePermissions(token, editingRoleId, selectedCodes)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['role-permissions'] })
      setDialogOpen(false)
      setFormError(null)
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setFormError(t('roles.forbidden'))
      } else {
        setFormError(t('roles.saveFailed'))
      }
    },
  })

  const isLoading =
    rolesQuery.isLoading || bindingsQuery.isLoading || permissionsQuery.isLoading
  const error = rolesQuery.error ?? bindingsQuery.error ?? permissionsQuery.error

  function openEdit(roleId: string, roleName: string) {
    setEditingRoleId(roleId)
    setEditingRoleName(roleName)
    setSelectedCodes([...(permissionsByRole.get(roleId) ?? [])])
    setFormError(null)
    setDialogOpen(true)
  }

  function toggleCode(code: string) {
    setSelectedCodes((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code].sort(),
    )
  }

  return (
    <AppLayout>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">{t('roles.title')}</h1>
        <p className="text-sm text-muted-foreground">{t('roles.subtitle')}</p>
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      ) : null}
      {error ? (
        <p className="text-sm text-destructive">{t('roles.loadFailed')}</p>
      ) : null}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('roles.name')}</TableHead>
            <TableHead>{t('roles.description')}</TableHead>
            <TableHead>{t('roles.permissions')}</TableHead>
            <TableHead className="text-right">{t('common.actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {(rolesQuery.data ?? []).map((role) => {
            const codes = permissionsByRole.get(role.id) ?? []
            return (
              <TableRow key={role.id}>
                <TableCell className="font-medium">{role.name}</TableCell>
                <TableCell>{role.description}</TableCell>
                <TableCell>
                  {codes.length === 0 ? (
                    <span className="text-sm text-muted-foreground">
                      {t('roles.noPermissions')}
                    </span>
                  ) : (
                    <ul className="flex flex-wrap gap-1">
                      {codes.map((code) => (
                        <li
                          key={code}
                          className="rounded-md bg-muted px-2 py-0.5 font-mono text-xs"
                          title={permissionDescriptions.get(code) ?? code}
                        >
                          {code}
                        </li>
                      ))}
                    </ul>
                  )}
                </TableCell>
                <TableCell className="text-right">
                  <PermissionGate permission={PERMISSIONS.RBAC_MANAGE}>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openEdit(role.id, role.name)}
                    >
                      {t('roles.editPermissions')}
                    </Button>
                  </PermissionGate>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogTitle>{t('roles.editTitle')}</DialogTitle>
        <DialogDescription>
          {t('roles.editHint')} ({editingRoleName})
        </DialogDescription>
        <form
          className="mt-4 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            saveMutation.mutate()
          }}
        >
          <ul className="max-h-64 space-y-2 overflow-y-auto rounded-md border p-3">
            {(permissionsQuery.data ?? []).map((perm) => (
              <li key={perm.id} className="flex items-start gap-2 text-sm">
                <input
                  id={`perm-${perm.code}`}
                  type="checkbox"
                  className="mt-1"
                  checked={selectedCodes.includes(perm.code)}
                  onChange={() => toggleCode(perm.code)}
                />
                <label htmlFor={`perm-${perm.code}`} className="cursor-pointer">
                  <span className="font-mono">{perm.code}</span>
                  <span className="block text-muted-foreground">{perm.description}</span>
                </label>
              </li>
            ))}
          </ul>
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
    </AppLayout>
  )
}
