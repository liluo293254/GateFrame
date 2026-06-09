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
import {
  ApiError,
  createUser,
  deleteUser,
  listUsers,
  updateUser,
  type CreateUserPayload,
  type User,
} from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

type FormMode = 'create' | 'edit'

export function UsersPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [formMode, setFormMode] = useState<FormMode>('create')
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [username, setUsername] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [status, setStatus] = useState<'active' | 'disabled'>('active')
  const [formError, setFormError] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const { data: users = [], isLoading, error } = useQuery({
    queryKey: ['users'],
    queryFn: () => listUsers(token),
  })

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (formMode === 'create') {
        const payload: CreateUserPayload = {
          username,
          password,
          display_name: displayName || username,
        }
        return createUser(token, payload)
      }
      if (!editingUser) throw new Error('missing user')
      return updateUser(token, editingUser.id, {
        display_name: displayName,
        status,
        ...(password ? { password } : {}),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      closeDialog()
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setFormError(t('users.forbidden'))
      } else {
        setFormError(t('users.saveFailed'))
      }
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteUser(token, id),
    onSuccess: () => {
      setDeleteError(null)
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setDeleteError(t('users.forbidden'))
      } else {
        setDeleteError(t('users.saveFailed'))
      }
    },
  })

  function openCreate() {
    setFormMode('create')
    setEditingUser(null)
    setUsername('')
    setDisplayName('')
    setPassword('')
    setStatus('active')
    setFormError(null)
    setDialogOpen(true)
  }

  function openEdit(user: User) {
    setFormMode('edit')
    setEditingUser(user)
    setUsername(user.username)
    setDisplayName(user.display_name)
    setPassword('')
    setStatus(user.status === 'disabled' ? 'disabled' : 'active')
    setFormError(null)
    setDialogOpen(true)
  }

  function closeDialog() {
    setDialogOpen(false)
    setFormError(null)
  }

  function statusLabel(value: string) {
    return value === 'disabled' ? t('users.statusDisabled') : t('users.statusActive')
  }

  return (
    <AppLayout>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{t('users.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('users.subtitle')}</p>
        </div>
        <PermissionGate permission={PERMISSIONS.USER_CREATE}>
          <Button onClick={openCreate}>{t('users.addUser')}</Button>
        </PermissionGate>
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      ) : null}
      {error ? (
        <p className="text-sm text-destructive">{t('users.loadFailed')}</p>
      ) : null}
      {deleteError ? (
        <p className="text-sm text-destructive">{deleteError}</p>
      ) : null}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('users.username')}</TableHead>
            <TableHead>{t('users.displayName')}</TableHead>
            <TableHead>{t('users.status')}</TableHead>
            <TableHead className="text-right">{t('common.actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((user) => (
            <TableRow key={user.id}>
              <TableCell>{user.username}</TableCell>
              <TableCell>{user.display_name}</TableCell>
              <TableCell>{statusLabel(user.status)}</TableCell>
              <TableCell className="space-x-2 text-right">
                <PermissionGate permission={PERMISSIONS.USER_UPDATE}>
                  <Button variant="outline" size="sm" onClick={() => openEdit(user)}>
                    {t('common.edit')}
                  </Button>
                </PermissionGate>
                <PermissionGate permission={PERMISSIONS.USER_DELETE}>
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={deleteMutation.isPending}
                    onClick={() => {
                      if (window.confirm(t('users.deleteConfirm', { username: user.username }))) {
                        setDeleteError(null)
                        deleteMutation.mutate(user.id)
                      }
                    }}
                  >
                    {t('common.delete')}
                  </Button>
                </PermissionGate>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogTitle>
          {formMode === 'create' ? t('users.createTitle') : t('users.editTitle')}
        </DialogTitle>
        <DialogDescription>
          {formMode === 'create' ? t('users.createHint') : t('users.editHint')}
        </DialogDescription>
        <form
          className="mt-4 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            saveMutation.mutate()
          }}
        >
          {formMode === 'create' ? (
            <div className="space-y-2">
              <Label htmlFor="username">{t('users.username')}</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                minLength={2}
              />
            </div>
          ) : null}
          <div className="space-y-2">
            <Label htmlFor="displayName">{t('users.displayName')}</Label>
            <Input
              id="displayName"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">
              {formMode === 'create' ? t('users.password') : t('users.newPasswordOptional')}
            </Label>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required={formMode === 'create'}
              minLength={formMode === 'create' ? 8 : undefined}
            />
          </div>
          {formMode === 'edit' ? (
            <div className="space-y-2">
              <Label htmlFor="status">{t('users.status')}</Label>
              <select
                id="status"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={status}
                onChange={(e) =>
                  setStatus(e.target.value as 'active' | 'disabled')
                }
              >
                <option value="active">{t('users.statusActive')}</option>
                <option value="disabled">{t('users.statusDisabled')}</option>
              </select>
            </div>
          ) : null}
          {formError ? <p className="text-sm text-destructive">{formError}</p> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={closeDialog}>
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
