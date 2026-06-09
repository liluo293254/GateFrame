import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { AppLayout } from '@/components/layout/AppLayout'
import { Button } from '@/components/ui/button'
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
import { listAuditEvents, type AuditEvent } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

const PAGE_SIZE = 20
const EXPORT_LIMIT = 5000

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function csvEscape(value: string) {
  if (value.includes('"') || value.includes(',') || value.includes('\n')) {
    return `"${value.replace(/"/g, '""')}"`
  }
  return value
}

function buildAuditCsv(events: AuditEvent[], t: TFunction) {
  const na = t('common.notAvailable')
  const header = [
    t('audit.time'),
    t('audit.action'),
    t('audit.path'),
    t('audit.status'),
    t('audit.user'),
  ]
  const rows = events.map((ev) => [
    formatTime(ev.created_at),
    ev.action,
    ev.path,
    String(ev.status_code),
    ev.user_id ?? na,
  ])
  return [header, ...rows]
    .map((row) => row.map((cell) => csvEscape(String(cell))).join(','))
    .join('\n')
}

function downloadCsv(filename: string, content: string) {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

export function AuditPage() {
  const { t } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const [page, setPage] = useState(0)
  const [actionFilter, setActionFilter] = useState('')
  const [pathFilter, setPathFilter] = useState('')
  const [fromFilter, setFromFilter] = useState('')
  const [toFilter, setToFilter] = useState('')
  const [appliedAction, setAppliedAction] = useState('')
  const [appliedPath, setAppliedPath] = useState('')
  const [appliedFrom, setAppliedFrom] = useState('')
  const [appliedTo, setAppliedTo] = useState('')
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)

  const queryParams = {
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    action: appliedAction || undefined,
    path: appliedPath || undefined,
    from: appliedFrom || undefined,
    to: appliedTo || undefined,
  }

  const { data, isLoading, error } = useQuery({
    queryKey: ['audit', page, appliedAction, appliedPath, appliedFrom, appliedTo],
    queryFn: () => listAuditEvents(token, queryParams),
  })

  const events = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function applyFilters() {
    setPage(0)
    setAppliedAction(actionFilter.trim())
    setAppliedPath(pathFilter.trim())
    setAppliedFrom(fromFilter.trim())
    setAppliedTo(toFilter.trim())
  }

  function clearFilters() {
    setActionFilter('')
    setPathFilter('')
    setFromFilter('')
    setToFilter('')
    setAppliedAction('')
    setAppliedPath('')
    setAppliedFrom('')
    setAppliedTo('')
    setPage(0)
  }

  async function exportCsv() {
    setExportError(null)
    setExporting(true)
    try {
      const result = await listAuditEvents(token, {
        ...queryParams,
        limit: EXPORT_LIMIT,
        offset: 0,
      })
      const csv = buildAuditCsv(result.items, t)
      downloadCsv('audit-log.csv', csv)
    } catch {
      setExportError(t('audit.exportFailed'))
    } finally {
      setExporting(false)
    }
  }

  return (
    <AppLayout>
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">{t('audit.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('audit.subtitle')}</p>
        </div>
        <Button
          type="button"
          variant="outline"
          disabled={exporting || isLoading}
          onClick={() => void exportCsv()}
        >
          {exporting ? t('audit.exporting') : t('audit.exportCsv')}
        </Button>
      </div>

      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="space-y-1">
          <Label htmlFor="audit-action">{t('audit.filterAction')}</Label>
          <Input
            id="audit-action"
            value={actionFilter}
            onChange={(e) => setActionFilter(e.target.value)}
            placeholder={t('audit.filterActionPlaceholder')}
            className="w-40"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="audit-path">{t('audit.filterPath')}</Label>
          <Input
            id="audit-path"
            value={pathFilter}
            onChange={(e) => setPathFilter(e.target.value)}
            placeholder={t('audit.filterPathPlaceholder')}
            className="w-56"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="audit-from">{t('audit.filterFrom')}</Label>
          <Input
            id="audit-from"
            type="date"
            value={fromFilter}
            onChange={(e) => setFromFilter(e.target.value)}
            className="w-40"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="audit-to">{t('audit.filterTo')}</Label>
          <Input
            id="audit-to"
            type="date"
            value={toFilter}
            onChange={(e) => setToFilter(e.target.value)}
            className="w-40"
          />
        </div>
        <Button type="button" onClick={applyFilters}>
          {t('audit.applyFilters')}
        </Button>
        <Button type="button" variant="outline" onClick={clearFilters}>
          {t('audit.clearFilters')}
        </Button>
      </div>

      {exportError ? (
        <p className="mb-2 text-sm text-destructive">{exportError}</p>
      ) : null}
      {isLoading ? (
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      ) : null}
      {error ? (
        <p className="text-sm text-destructive">{t('audit.loadFailed')}</p>
      ) : null}
      {!isLoading && !error && events.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t('audit.empty')}</p>
      ) : null}

      {events.length > 0 ? (
        <>
          <p className="mb-2 text-sm text-muted-foreground">
            {t('audit.total', { count: total })}
          </p>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('audit.time')}</TableHead>
                <TableHead>{t('audit.action')}</TableHead>
                <TableHead>{t('audit.path')}</TableHead>
                <TableHead>{t('audit.status')}</TableHead>
                <TableHead>{t('audit.user')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {events.map((ev) => (
                <TableRow key={ev.id}>
                  <TableCell className="whitespace-nowrap">
                    {formatTime(ev.created_at)}
                  </TableCell>
                  <TableCell>{ev.action}</TableCell>
                  <TableCell className="font-mono text-sm">{ev.path}</TableCell>
                  <TableCell>{ev.status_code}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {ev.user_id ?? t('common.notAvailable')}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="mt-4 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">
              {t('audit.page', { page: page + 1 })}
            </span>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
              >
                {t('audit.prev')}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page + 1 >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                {t('audit.next')}
              </Button>
            </div>
          </div>
        </>
      ) : null}
    </AppLayout>
  )
}
