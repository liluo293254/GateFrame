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
import { ApiError, deleteSearchDocument, indexSearchDocument, searchDocuments } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

export function SearchPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const queryClient = useQueryClient()
  const [query, setQuery] = useState('')
  const [submittedQuery, setSubmittedQuery] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  const searchQuery = useQuery({
    queryKey: ['search', submittedQuery],
    queryFn: () => searchDocuments(token, submittedQuery),
  })

  const indexMutation = useMutation({
    mutationFn: () => indexSearchDocument(token, { title, content }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['search'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      setDialogOpen(false)
      setTitle('')
      setContent('')
      setFormError(null)
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setFormError(t('search.forbidden'))
      } else {
        setFormError(t('search.indexFailed'))
      }
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteSearchDocument(token, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['search'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
    },
  })

  const items = searchQuery.data?.items ?? []

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t('search.title')}</h1>
            <p className="text-muted-foreground">{t('search.subtitle')}</p>
          </div>
          <PermissionGate permission={PERMISSIONS.SEARCH_MANAGE}>
            <Button
              onClick={() => {
                setTitle('')
                setContent('')
                setFormError(null)
                setDialogOpen(true)
              }}
            >
              {t('search.indexDocument')}
            </Button>
          </PermissionGate>
        </div>

        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            setSubmittedQuery(query)
          }}
        >
          <div className="min-w-[240px] flex-1 space-y-2">
            <Label htmlFor="search-q">{t('search.query')}</Label>
            <Input
              id="search-q"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t('search.queryPlaceholder')}
            />
          </div>
          <Button type="submit">{t('search.submit')}</Button>
        </form>

        {searchQuery.isLoading ? (
          <p className="text-muted-foreground">{t('common.loading')}</p>
        ) : searchQuery.isError ? (
          <p className="text-destructive">{t('search.loadFailed')}</p>
        ) : items.length === 0 ? (
          <p className="text-muted-foreground">{t('search.empty')}</p>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">
              {t('search.total', { count: searchQuery.data?.total ?? 0 })}
            </p>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('search.documentTitle')}</TableHead>
                  <TableHead>{t('search.content')}</TableHead>
                  <TableHead>{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((doc) => (
                  <TableRow key={doc.id}>
                    <TableCell className="font-medium">{doc.title}</TableCell>
                    <TableCell>{doc.content}</TableCell>
                    <TableCell>
                      <PermissionGate permission={PERMISSIONS.SEARCH_MANAGE}>
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={deleteMutation.isPending}
                          onClick={() => {
                            if (window.confirm(t('search.deleteConfirm', { title: doc.title }))) {
                              deleteMutation.mutate(doc.id)
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
          </>
        )}

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTitle>{t('search.indexTitle')}</DialogTitle>
          <DialogDescription>{t('search.indexHint')}</DialogDescription>
          <form
            className="mt-4 space-y-4"
            onSubmit={(e) => {
              e.preventDefault()
              indexMutation.mutate()
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="doc-title">{t('search.documentTitle')}</Label>
              <Input
                id="doc-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="doc-content">{t('search.content')}</Label>
              <Input
                id="doc-content"
                value={content}
                onChange={(e) => setContent(e.target.value)}
              />
            </div>
            {formError ? <p className="text-sm text-destructive">{formError}</p> : null}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={indexMutation.isPending}>
                {indexMutation.isPending ? t('common.saving') : t('common.save')}
              </Button>
            </div>
          </form>
        </Dialog>
      </div>
    </AppLayout>
  )
}
