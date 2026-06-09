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
  createWorkflow,
  deleteWorkflow,
  listWorkflows,
  updateWorkflow,
  type Workflow,
} from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

type FormMode = 'create' | 'edit'

const STATUS_OPTIONS = ['draft', 'active', 'archived'] as const

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

export function WorkflowsPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [formMode, setFormMode] = useState<FormMode>('create')
  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState<(typeof STATUS_OPTIONS)[number]>('draft')
  const [formError, setFormError] = useState<string | null>(null)

  const workflowsQuery = useQuery({
    queryKey: ['workflows'],
    queryFn: () => listWorkflows(token),
  })

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (formMode === 'create') {
        return createWorkflow(token, { name, description, status })
      }
      if (!editingWorkflow) throw new Error('missing workflow')
      return updateWorkflow(token, editingWorkflow.id, { name, description, status })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflows'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      closeDialog()
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setFormError(t('workflows.forbidden'))
      } else {
        setFormError(t('workflows.saveFailed'))
      }
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteWorkflow(token, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflows'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
    },
  })

  function openCreate() {
    setFormMode('create')
    setEditingWorkflow(null)
    setName('')
    setDescription('')
    setStatus('draft')
    setFormError(null)
    setDialogOpen(true)
  }

  function openEdit(workflow: Workflow) {
    setFormMode('edit')
    setEditingWorkflow(workflow)
    setName(workflow.name)
    setDescription(workflow.description)
    setStatus(
      STATUS_OPTIONS.includes(workflow.status as (typeof STATUS_OPTIONS)[number])
        ? (workflow.status as (typeof STATUS_OPTIONS)[number])
        : 'draft',
    )
    setFormError(null)
    setDialogOpen(true)
  }

  function closeDialog() {
    setDialogOpen(false)
    setFormError(null)
  }

  const workflows = workflowsQuery.data ?? []

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t('workflows.title')}</h1>
            <p className="text-muted-foreground">{t('workflows.subtitle')}</p>
          </div>
          <PermissionGate permission={PERMISSIONS.WORKFLOW_MANAGE}>
            <Button onClick={openCreate}>{t('workflows.addWorkflow')}</Button>
          </PermissionGate>
        </div>

        {workflowsQuery.isLoading ? (
          <p className="text-muted-foreground">{t('common.loading')}</p>
        ) : workflowsQuery.isError ? (
          <p className="text-destructive">{t('workflows.loadFailed')}</p>
        ) : workflows.length === 0 ? (
          <p className="text-muted-foreground">{t('workflows.empty')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('workflows.name')}</TableHead>
                <TableHead>{t('workflows.description')}</TableHead>
                <TableHead>{t('workflows.status')}</TableHead>
                <TableHead>{t('workflows.createdAt')}</TableHead>
                <TableHead>{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {workflows.map((wf) => (
                <TableRow key={wf.id}>
                  <TableCell className="font-medium">{wf.name}</TableCell>
                  <TableCell>{wf.description}</TableCell>
                  <TableCell>{wf.status}</TableCell>
                  <TableCell>{formatTime(wf.created_at)}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-2">
                      <PermissionGate permission={PERMISSIONS.WORKFLOW_MANAGE}>
                        <Button variant="outline" size="sm" onClick={() => openEdit(wf)}>
                          {t('common.edit')}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={deleteMutation.isPending}
                          onClick={() => {
                            if (window.confirm(t('workflows.deleteConfirm', { name: wf.name }))) {
                              deleteMutation.mutate(wf.id)
                            }
                          }}
                        >
                          {t('common.delete')}
                        </Button>
                      </PermissionGate>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTitle>
            {formMode === 'create' ? t('workflows.createTitle') : t('workflows.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {formMode === 'create' ? t('workflows.createHint') : t('workflows.editHint')}
          </DialogDescription>
          <form
            className="mt-4 space-y-4"
            onSubmit={(e) => {
              e.preventDefault()
              saveMutation.mutate()
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="wf-name">{t('workflows.name')}</Label>
              <Input
                id="wf-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="wf-description">{t('workflows.description')}</Label>
              <Input
                id="wf-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="wf-status">{t('workflows.status')}</Label>
              <select
                id="wf-status"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                value={status}
                onChange={(e) =>
                  setStatus(e.target.value as (typeof STATUS_OPTIONS)[number])
                }
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {t(`workflows.status${opt.charAt(0).toUpperCase()}${opt.slice(1)}`)}
                  </option>
                ))}
              </select>
            </div>
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
      </div>
    </AppLayout>
  )
}
