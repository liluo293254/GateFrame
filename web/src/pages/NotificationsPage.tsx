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
  createNotification,
  listNotifications,
  markNotificationRead,
} from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

export function NotificationsPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  const notificationsQuery = useQuery({
    queryKey: ['notifications'],
    queryFn: () => listNotifications(token),
  })

  const createMutation = useMutation({
    mutationFn: () => createNotification(token, { title, body }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      setDialogOpen(false)
      setTitle('')
      setBody('')
      setFormError(null)
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setFormError(t('notifications.forbidden'))
      } else {
        setFormError(t('notifications.saveFailed'))
      }
    },
  })

  const markReadMutation = useMutation({
    mutationFn: (id: string) => markNotificationRead(token, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })

  const items = notificationsQuery.data ?? []

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t('notifications.title')}</h1>
            <p className="text-muted-foreground">{t('notifications.subtitle')}</p>
          </div>
          <PermissionGate permission={PERMISSIONS.NOTIFICATION_MANAGE}>
            <Button
              onClick={() => {
                setTitle('')
                setBody('')
                setFormError(null)
                setDialogOpen(true)
              }}
            >
              {t('notifications.addNotification')}
            </Button>
          </PermissionGate>
        </div>

        {notificationsQuery.isLoading ? (
          <p className="text-muted-foreground">{t('common.loading')}</p>
        ) : notificationsQuery.isError ? (
          <p className="text-destructive">{t('notifications.loadFailed')}</p>
        ) : items.length === 0 ? (
          <p className="text-muted-foreground">{t('notifications.empty')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('notifications.titleColumn')}</TableHead>
                <TableHead>{t('notifications.body')}</TableHead>
                <TableHead>{t('notifications.status')}</TableHead>
                <TableHead>{t('notifications.createdAt')}</TableHead>
                <TableHead>{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((n) => (
                <TableRow key={n.id}>
                  <TableCell className="font-medium">{n.title}</TableCell>
                  <TableCell>{n.body}</TableCell>
                  <TableCell>
                    {n.read_at ? t('notifications.read') : t('notifications.unread')}
                  </TableCell>
                  <TableCell>{formatTime(n.created_at)}</TableCell>
                  <TableCell>
                    {!n.read_at ? (
                      <PermissionGate permission={PERMISSIONS.NOTIFICATION_MANAGE}>
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={markReadMutation.isPending}
                          onClick={() => markReadMutation.mutate(n.id)}
                        >
                          {t('notifications.markRead')}
                        </Button>
                      </PermissionGate>
                    ) : (
                      t('common.notAvailable')
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTitle>{t('notifications.createTitle')}</DialogTitle>
          <DialogDescription>{t('notifications.createHint')}</DialogDescription>
          <form
            className="mt-4 space-y-4"
            onSubmit={(e) => {
              e.preventDefault()
              createMutation.mutate()
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="notif-title">{t('notifications.titleColumn')}</Label>
              <Input
                id="notif-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="notif-body">{t('notifications.body')}</Label>
              <Input
                id="notif-body"
                value={body}
                onChange={(e) => setBody(e.target.value)}
              />
            </div>
            {formError ? <p className="text-sm text-destructive">{formError}</p> : null}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? t('common.saving') : t('common.save')}
              </Button>
            </div>
          </form>
        </Dialog>
      </div>
    </AppLayout>
  )
}
