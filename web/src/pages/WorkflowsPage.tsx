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
  approveWorkflow,
  createWorkflow,
  deleteWorkflow,
  listWorkflowEvents,
  listWorkflows,
  rejectWorkflow,
  requestWorkflowChanges,
  submitWorkflow,
  updateWorkflow,
  type Workflow,
  type WorkflowEvent,
} from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

type FormMode = 'create' | 'edit'
type TabKey = 'all' | 'mine' | 'inbox'

const CATEGORY_OPTIONS = ['general', 'expense', 'leave', 'procurement'] as const
const PRIORITY_OPTIONS = ['low', 'normal', 'high'] as const

const STATUS_STYLES: Record<string, string> = {
  draft: 'bg-muted text-muted-foreground',
  pending: 'bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-100',
  approved: 'bg-emerald-100 text-emerald-900 dark:bg-emerald-950 dark:text-emerald-100',
  rejected: 'bg-red-100 text-red-900 dark:bg-red-950 dark:text-red-100',
  changes_requested: 'bg-sky-100 text-sky-900 dark:bg-sky-950 dark:text-sky-100',
}

function formatTime(iso: string | undefined) {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function editableStatus(status: string) {
  return status === 'draft' || status === 'changes_requested'
}

export function WorkflowsPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const userId = useAuthStore((s) => s.userId)
  const displayName = useAuthStore((s) => s.displayName)
  const username = useAuthStore((s) => s.username)
  const canManage = useAuthStore((s) => s.hasPermission(PERMISSIONS.WORKFLOW_MANAGE))
  const canRequest =
    canManage || useAuthStore((s) => s.hasPermission(PERMISSIONS.WORKFLOW_READ))

  const actorLabel = displayName || username || userId || ''
  const queryClient = useQueryClient()

  const [tab, setTab] = useState<TabKey>('all')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [formMode, setFormMode] = useState<FormMode>('create')
  const [selected, setSelected] = useState<Workflow | null>(null)
  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState<(typeof CATEGORY_OPTIONS)[number]>('general')
  const [priority, setPriority] = useState<(typeof PRIORITY_OPTIONS)[number]>('normal')
  const [reviewComment, setReviewComment] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  const workflowsQuery = useQuery({
    queryKey: ['workflows'],
    queryFn: () => listWorkflows(token),
  })

  const eventsQuery = useQuery({
    queryKey: ['workflow-events', selected?.id],
    queryFn: () => listWorkflowEvents(token, selected!.id),
    enabled: detailOpen && !!selected?.id,
  })

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['workflows'] })
    queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
    if (selected?.id) {
      queryClient.invalidateQueries({ queryKey: ['workflow-events', selected.id] })
    }
  }

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        name,
        description,
        category,
        priority,
        requester_label: actorLabel,
      }
      if (formMode === 'create') {
        return createWorkflow(token, payload)
      }
      if (!editingWorkflow) throw new Error('missing workflow')
      return updateWorkflow(token, editingWorkflow.id, payload)
    },
    onSuccess: () => {
      invalidate()
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

  const submitMutation = useMutation({
    mutationFn: (wf: Workflow) =>
      submitWorkflow(token, wf.id, { actor_label: actorLabel }),
    onSuccess: () => invalidate(),
  })

  const approveMutation = useMutation({
    mutationFn: (wf: Workflow) =>
      approveWorkflow(token, wf.id, { comment: reviewComment, actor_label: actorLabel }),
    onSuccess: () => {
      setReviewComment('')
      invalidate()
    },
  })

  const rejectMutation = useMutation({
    mutationFn: (wf: Workflow) =>
      rejectWorkflow(token, wf.id, { comment: reviewComment, actor_label: actorLabel }),
    onSuccess: () => {
      setReviewComment('')
      invalidate()
    },
  })

  const changesMutation = useMutation({
    mutationFn: (wf: Workflow) =>
      requestWorkflowChanges(token, wf.id, {
        comment: reviewComment,
        actor_label: actorLabel,
      }),
    onSuccess: () => {
      setReviewComment('')
      invalidate()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteWorkflow(token, id),
    onSuccess: () => {
      setDetailOpen(false)
      setSelected(null)
      invalidate()
    },
  })

  const workflows = workflowsQuery.data ?? []

  const filtered = useMemo(() => {
    if (tab === 'mine') {
      return workflows.filter((wf) => wf.requester_id === userId)
    }
    if (tab === 'inbox') {
      return workflows.filter((wf) => wf.status === 'pending')
    }
    return workflows
  }, [tab, workflows, userId])

  function openCreate() {
    setFormMode('create')
    setEditingWorkflow(null)
    setName('')
    setDescription('')
    setCategory('general')
    setPriority('normal')
    setFormError(null)
    setDialogOpen(true)
  }

  function openEdit(workflow: Workflow) {
    setFormMode('edit')
    setEditingWorkflow(workflow)
    setName(workflow.name)
    setDescription(workflow.description)
    setCategory(
      CATEGORY_OPTIONS.includes(workflow.category as (typeof CATEGORY_OPTIONS)[number])
        ? (workflow.category as (typeof CATEGORY_OPTIONS)[number])
        : 'general',
    )
    setPriority(
      PRIORITY_OPTIONS.includes(workflow.priority as (typeof PRIORITY_OPTIONS)[number])
        ? (workflow.priority as (typeof PRIORITY_OPTIONS)[number])
        : 'normal',
    )
    setFormError(null)
    setDialogOpen(true)
  }

  function openDetail(workflow: Workflow) {
    setSelected(workflow)
    setReviewComment('')
    setDetailOpen(true)
  }

  function closeDialog() {
    setDialogOpen(false)
    setFormError(null)
  }

  function statusLabel(status: string) {
    const key = `workflows.status${status
      .split('_')
      .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
      .join('')}` as 'workflows.statusDraft'
    return t(key, { defaultValue: status })
  }

  function eventLabel(event: WorkflowEvent) {
    return t(`workflows.event.${event.event_type}`, { defaultValue: event.event_type })
  }

  const pendingCount = workflows.filter((w) => w.status === 'pending').length

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t('workflows.title')}</h1>
            <p className="text-muted-foreground">{t('workflows.subtitle')}</p>
          </div>
          {canRequest ? (
            <Button onClick={openCreate}>{t('workflows.newRequest')}</Button>
          ) : null}
        </div>

        <div className="flex flex-wrap gap-2">
          {(
            [
              ['all', t('workflows.tabAll')],
              ['mine', t('workflows.tabMine')],
              ['inbox', t('workflows.tabInbox', { count: pendingCount })],
            ] as const
          ).map(([key, label]) => (
            <Button
              key={key}
              variant={tab === key ? 'default' : 'outline'}
              size="sm"
              onClick={() => setTab(key)}
            >
              {label}
            </Button>
          ))}
        </div>

        {workflowsQuery.isLoading ? (
          <p className="text-muted-foreground">{t('common.loading')}</p>
        ) : workflowsQuery.isError ? (
          <p className="text-destructive">{t('workflows.loadFailed')}</p>
        ) : filtered.length === 0 ? (
          <p className="text-muted-foreground">{t('workflows.empty')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('workflows.name')}</TableHead>
                <TableHead>{t('workflows.category')}</TableHead>
                <TableHead>{t('workflows.requester')}</TableHead>
                <TableHead>{t('workflows.status')}</TableHead>
                <TableHead>{t('workflows.submittedAt')}</TableHead>
                <TableHead>{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((wf) => (
                <TableRow key={wf.id}>
                  <TableCell className="font-medium">{wf.name}</TableCell>
                  <TableCell>{t(`workflows.category${wf.category.charAt(0).toUpperCase()}${wf.category.slice(1)}`)}</TableCell>
                  <TableCell>
                    {wf.requester_id === userId
                      ? t('workflows.requesterMe')
                      : wf.requester_label || wf.requester_id?.slice(0, 8) || '—'}
                  </TableCell>
                  <TableCell>
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLES[wf.status] ?? STATUS_STYLES.draft}`}
                    >
                      {statusLabel(wf.status)}
                    </span>
                  </TableCell>
                  <TableCell>{formatTime(wf.submitted_at)}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-2">
                      <Button variant="outline" size="sm" onClick={() => openDetail(wf)}>
                        {t('workflows.view')}
                      </Button>
                      {canRequest && editableStatus(wf.status) ? (
                        <Button variant="outline" size="sm" onClick={() => openEdit(wf)}>
                          {t('common.edit')}
                        </Button>
                      ) : null}
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
                placeholder={t('workflows.namePlaceholder')}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="wf-description">{t('workflows.description')}</Label>
              <Input
                id="wf-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t('workflows.descriptionPlaceholder')}
              />
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="wf-category">{t('workflows.category')}</Label>
                <select
                  id="wf-category"
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                  value={category}
                  onChange={(e) =>
                    setCategory(e.target.value as (typeof CATEGORY_OPTIONS)[number])
                  }
                >
                  {CATEGORY_OPTIONS.map((opt) => (
                    <option key={opt} value={opt}>
                      {t(`workflows.category${opt.charAt(0).toUpperCase()}${opt.slice(1)}`)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="wf-priority">{t('workflows.priority')}</Label>
                <select
                  id="wf-priority"
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                  value={priority}
                  onChange={(e) =>
                    setPriority(e.target.value as (typeof PRIORITY_OPTIONS)[number])
                  }
                >
                  {PRIORITY_OPTIONS.map((opt) => (
                    <option key={opt} value={opt}>
                      {t(`workflows.priority${opt.charAt(0).toUpperCase()}${opt.slice(1)}`)}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            {formMode === 'create' ? (
              <p className="text-sm text-muted-foreground">{t('workflows.createNextSteps')}</p>
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

        <Dialog open={detailOpen} onOpenChange={setDetailOpen}>
          {selected ? (
            <>
              <DialogTitle>{selected.name}</DialogTitle>
              <DialogDescription>{selected.description || t('workflows.noDescription')}</DialogDescription>
              <div className="mt-4 space-y-4">
                <div className="grid gap-2 text-sm sm:grid-cols-2">
                  <div>
                    <span className="text-muted-foreground">{t('workflows.status')}: </span>
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLES[selected.status] ?? ''}`}>
                      {statusLabel(selected.status)}
                    </span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">{t('workflows.requester')}: </span>
                    {selected.requester_label || selected.requester_id?.slice(0, 8)}
                  </div>
                  <div>
                    <span className="text-muted-foreground">{t('workflows.submittedAt')}: </span>
                    {formatTime(selected.submitted_at)}
                  </div>
                  <div>
                    <span className="text-muted-foreground">{t('workflows.reviewedAt')}: </span>
                    {formatTime(selected.reviewed_at)}
                  </div>
                </div>

                {selected.review_comment ? (
                  <div className="rounded-md border bg-muted/40 p-3 text-sm">
                    <p className="font-medium">{t('workflows.lastReviewComment')}</p>
                    <p className="mt-1 whitespace-pre-wrap">{selected.review_comment}</p>
                  </div>
                ) : null}

                <div>
                  <p className="mb-2 text-sm font-medium">{t('workflows.timeline')}</p>
                  {eventsQuery.isLoading ? (
                    <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
                  ) : (
                    <ul className="space-y-2 text-sm">
                      {(eventsQuery.data ?? []).map((ev) => (
                        <li key={ev.id} className="rounded-md border px-3 py-2">
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <span className="font-medium">{eventLabel(ev)}</span>
                            <span className="text-muted-foreground">{formatTime(ev.created_at)}</span>
                          </div>
                          {ev.comment ? (
                            <p className="mt-1 text-muted-foreground whitespace-pre-wrap">{ev.comment}</p>
                          ) : null}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>

                {canRequest && editableStatus(selected.status) ? (
                  <Button
                    disabled={submitMutation.isPending}
                    onClick={() => submitMutation.mutate(selected)}
                  >
                    {t('workflows.submitForReview')}
                  </Button>
                ) : null}

                <PermissionGate permission={PERMISSIONS.WORKFLOW_MANAGE}>
                  {selected.status === 'pending' ? (
                    <div className="space-y-3 rounded-md border p-3">
                      <p className="text-sm font-medium">{t('workflows.reviewActions')}</p>
                      <div className="space-y-2">
                        <Label htmlFor="review-comment">{t('workflows.reviewComment')}</Label>
                        <Input
                          id="review-comment"
                          value={reviewComment}
                          onChange={(e) => setReviewComment(e.target.value)}
                          placeholder={t('workflows.reviewCommentPlaceholder')}
                        />
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <Button
                          disabled={approveMutation.isPending}
                          onClick={() => approveMutation.mutate(selected)}
                        >
                          {t('workflows.approve')}
                        </Button>
                        <Button
                          variant="outline"
                          disabled={changesMutation.isPending || !reviewComment.trim()}
                          onClick={() => changesMutation.mutate(selected)}
                        >
                          {t('workflows.requestChanges')}
                        </Button>
                        <Button
                          variant="destructive"
                          disabled={rejectMutation.isPending || !reviewComment.trim()}
                          onClick={() => rejectMutation.mutate(selected)}
                        >
                          {t('workflows.reject')}
                        </Button>
                      </div>
                      <p className="text-xs text-muted-foreground">{t('workflows.rejectHint')}</p>
                    </div>
                  ) : null}
                  {selected.status !== 'pending' ? (
                    <Button
                      variant="outline"
                      disabled={deleteMutation.isPending}
                      onClick={() => {
                        if (window.confirm(t('workflows.deleteConfirm', { name: selected.name }))) {
                          deleteMutation.mutate(selected.id)
                        }
                      }}
                    >
                      {t('common.delete')}
                    </Button>
                  ) : null}
                </PermissionGate>
              </div>
            </>
          ) : null}
        </Dialog>
      </div>
    </AppLayout>
  )
}
