import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, CheckCircle2, Download, FolderOpen, Loader2, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PermissionGate } from '@/components/auth/PermissionGate'
import { AppLayout } from '@/components/layout/AppLayout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog'
import { FileUploadZone, FileZoneShell } from '@/components/ui/file-upload'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  allowedTypesLabel,
  FILE_INPUT_ACCEPT,
  isAllowedFile,
  isWithinSizeLimit,
  maxFileSizeLabel,
} from '@/config/file-upload'
import { PERMISSIONS } from '@/config/permissions'
import {
  fileIconTone,
  formatAbsoluteTime,
  formatContentTypeLabel,
  formatRelativeTime,
  getFileIcon,
} from '@/lib/file-display'
import { readFileAsBase64WithProgress } from '@/lib/file-upload-client'
import {
  ApiError,
  deleteFile,
  downloadFileContent,
  listFiles,
  uploadFileWithProgress,
  type FileObject,
} from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

export function FilesPage() {
  const { t, i18n } = useTranslation()
  const token = useAuthStore((s) => s.token)!
  const hasPermission = useAuthStore((s) => s.hasPermission)
  const canManageFiles = hasPermission(PERMISSIONS.FILE_MANAGE)
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [uploadSuccessMessage, setUploadSuccessMessage] = useState<string | null>(null)
  const [uploadingFileName, setUploadingFileName] = useState<string | null>(null)
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const [uploadQueue, setUploadQueue] = useState<{ current: number; total: number } | null>(null)
  const [isUploading, setIsUploading] = useState(false)
  const [downloadingId, setDownloadingId] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<FileObject | null>(null)

  const filesQuery = useQuery({
    queryKey: ['files'],
    queryFn: () => listFiles(token),
  })

  useEffect(() => {
    if (!uploadSuccessMessage) return
    const timer = window.setTimeout(() => setUploadSuccessMessage(null), 4000)
    return () => window.clearTimeout(timer)
  }, [uploadSuccessMessage])

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteFile(token, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['files'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      setDeleteTarget(null)
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        setUploadError(t('files.forbidden'))
      } else {
        setUploadError(t('files.deleteFailed'))
      }
    },
  })

  async function uploadSingleFile(file: File) {
    const contentBase64 = await readFileAsBase64WithProgress(file, (percent) => {
      setUploadProgress(percent)
    })
    await uploadFileWithProgress(
      token,
      {
        filename: file.name,
        content_type: file.type || undefined,
        content_base64: contentBase64,
      },
      (percent) => {
        setUploadProgress(percent)
      },
    )
  }

  async function handleFilesSelect(files: File[]) {
    const typed = files.filter(isAllowedFile)
    const sized = typed.filter(isWithinSizeLimit)
    const rejectedTypeCount = files.length - typed.length
    const rejectedSizeCount = typed.length - sized.length
    const sizeLabel = maxFileSizeLabel(i18n.language)

    if (sized.length === 0) {
      if (rejectedSizeCount > 0) {
        setUploadError(t('files.tooLarge', { size: sizeLabel }))
      } else {
        setUploadError(t('files.invalidType'))
      }
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
      return
    }

    setUploadError(null)
    setUploadSuccessMessage(null)
    setIsUploading(true)

    if (rejectedTypeCount > 0) {
      setUploadError(t('files.invalidTypeSkipped', { count: rejectedTypeCount }))
    } else if (rejectedSizeCount > 0) {
      setUploadError(t('files.tooLargeSkipped', { count: rejectedSizeCount, size: sizeLabel }))
    }

    const uploadedNames: string[] = []

    try {
      for (let index = 0; index < sized.length; index += 1) {
        const file = sized[index]
        setUploadQueue({ current: index + 1, total: sized.length })
        setUploadingFileName(file.name)
        setUploadProgress(0)
        await uploadSingleFile(file)
        uploadedNames.push(file.name)
      }

      queryClient.invalidateQueries({ queryKey: ['files'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })

      if (uploadedNames.length === 1) {
        setUploadSuccessMessage(t('files.uploadSuccess', { filename: uploadedNames[0] }))
      } else {
        setUploadSuccessMessage(t('files.uploadSuccessMultiple', { count: uploadedNames.length }))
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setUploadError(t('files.forbidden'))
      } else if (err instanceof ApiError && err.status === 400) {
        setUploadError(err.message || t('files.uploadFailed'))
      } else {
        setUploadError(t('files.uploadFailed'))
      }
    } finally {
      setIsUploading(false)
      setUploadQueue(null)
      setUploadingFileName(null)
      setUploadProgress(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  async function handleDownload(file: FileObject) {
    setDownloadingId(file.id)
    try {
      const blob = await downloadFileContent(token, file.id)
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = file.filename
      anchor.click()
      URL.revokeObjectURL(url)
    } catch {
      setUploadError(t('files.downloadFailed'))
    } finally {
      setDownloadingId(null)
    }
  }

  const files = filesQuery.data ?? []
  const locale = i18n.language
  const uploadingDetail =
    uploadingFileName && uploadQueue
      ? t('files.uploadingQueue', {
          current: uploadQueue.current,
          total: uploadQueue.total,
          filename: uploadingFileName,
        })
      : uploadingFileName
        ? t('files.uploadingFile', { filename: uploadingFileName })
        : null

  return (
    <AppLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t('files.title')}</h1>
          <p className="text-muted-foreground">{t('files.subtitle')}</p>
        </div>

        <PermissionGate permission={PERMISSIONS.FILE_MANAGE}>
          <Card>
            <CardContent className="pt-6">
              <FileUploadZone
                id="file-upload"
                ref={fileInputRef}
                title={t('files.upload')}
                hint={t('files.uploadHint')}
                allowedTypesHint={t('files.uploadLimits', {
                  types: allowedTypesLabel(),
                  size: maxFileSizeLabel(i18n.language),
                })}
                browseLabel={t('files.browse')}
                uploadingLabel={t('files.uploading')}
                uploadingDetail={uploadingDetail}
                progress={uploadProgress}
                accept={FILE_INPUT_ACCEPT}
                multiple
                disabled={isUploading}
                uploading={isUploading}
                onFilesSelect={(selected) => {
                  void handleFilesSelect(selected)
                }}
              />
            </CardContent>
          </Card>
        </PermissionGate>

        {uploadSuccessMessage ? (
          <div
            className="flex items-start gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/40 dark:text-emerald-300"
            role="status"
          >
            <CheckCircle2 className="mt-0.5 size-4 shrink-0" aria-hidden />
            <span>{uploadSuccessMessage}</span>
          </div>
        ) : null}

        {uploadError ? (
          <div
            className="flex items-start gap-3 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
            role="alert"
          >
            <AlertCircle className="mt-0.5 size-4 shrink-0" aria-hidden />
            <span>{uploadError}</span>
          </div>
        ) : null}

        <Card>
          <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 space-y-0">
            <CardTitle className="text-lg">{t('files.listTitle')}</CardTitle>
            {!filesQuery.isLoading && !filesQuery.isError ? (
              <span className="rounded-full bg-muted px-3 py-1 text-xs font-medium text-muted-foreground">
                {t('files.fileCount', { count: files.length })}
              </span>
            ) : null}
          </CardHeader>
          <CardContent>
            {filesQuery.isLoading ? (
              <div className="flex items-center gap-2 py-10 text-muted-foreground">
                <Loader2 className="size-4 animate-spin" aria-hidden />
                <span>{t('common.loading')}</span>
              </div>
            ) : filesQuery.isError ? (
              <div className="flex items-center gap-2 py-10 text-destructive">
                <AlertCircle className="size-4 shrink-0" aria-hidden />
                <span>{t('files.loadFailed')}</span>
              </div>
            ) : files.length === 0 ? (
              <FileZoneShell>
                <span className="flex size-14 items-center justify-center rounded-full bg-muted">
                  <FolderOpen className="size-7 text-muted-foreground" aria-hidden />
                </span>
                <div className="space-y-1">
                  <p className="font-medium text-foreground">{t('files.emptyTitle')}</p>
                  <p className="max-w-md text-sm text-muted-foreground">
                    {canManageFiles ? t('files.emptyHint') : t('files.emptyReadOnly')}
                  </p>
                </div>
              </FileZoneShell>
            ) : (
              <div className="overflow-x-auto rounded-lg border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('files.filename')}</TableHead>
                      <TableHead>{t('files.contentType')}</TableHead>
                      <TableHead>{t('files.size')}</TableHead>
                      <TableHead>{t('files.createdAt')}</TableHead>
                      <TableHead className="text-right">{t('common.actions')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {files.map((file) => {
                      const Icon = getFileIcon(file.filename, file.content_type)
                      const iconTone = fileIconTone(file.filename, file.content_type)
                      const absoluteTime = formatAbsoluteTime(file.created_at, locale)
                      const relativeTime = formatRelativeTime(file.created_at, locale)
                      return (
                        <TableRow key={file.id}>
                          <TableCell>
                            <div className="flex min-w-[12rem] items-center gap-3">
                              <span
                                className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${iconTone}`}
                              >
                                <Icon className="size-4" aria-hidden />
                              </span>
                              <span className="truncate font-medium" title={file.filename}>
                                {file.filename}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            {formatContentTypeLabel(file.content_type)}
                          </TableCell>
                          <TableCell className="tabular-nums text-muted-foreground">
                            {formatBytes(file.size_bytes)}
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            <time dateTime={file.created_at} title={absoluteTime}>
                              {relativeTime}
                            </time>
                          </TableCell>
                          <TableCell className="text-right">
                            <div className="flex flex-wrap justify-end gap-2">
                              <Button
                                variant="outline"
                                size="sm"
                                disabled={downloadingId === file.id}
                                onClick={() => void handleDownload(file)}
                              >
                                {downloadingId === file.id ? (
                                  <>
                                    <Loader2 className="size-4 animate-spin" aria-hidden />
                                    {t('files.downloading')}
                                  </>
                                ) : (
                                  <>
                                    <Download className="size-4" aria-hidden />
                                    {t('files.download')}
                                  </>
                                )}
                              </Button>
                              <PermissionGate permission={PERMISSIONS.FILE_MANAGE}>
                                <Button
                                  variant="destructive"
                                  size="sm"
                                  disabled={deleteMutation.isPending}
                                  onClick={() => setDeleteTarget(file)}
                                >
                                  <Trash2 className="size-4" aria-hidden />
                                  {t('common.delete')}
                                </Button>
                              </PermissionGate>
                            </div>
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        <Dialog open={deleteTarget != null} onOpenChange={(open) => !open && setDeleteTarget(null)}>
          <DialogTitle>{t('files.deleteTitle')}</DialogTitle>
          <DialogDescription>
            {deleteTarget
              ? t('files.deleteConfirm', { filename: deleteTarget.filename })
              : null}
          </DialogDescription>
          <div className="mt-6 flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setDeleteTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (deleteTarget) {
                  deleteMutation.mutate(deleteTarget.id)
                }
              }}
            >
              {deleteMutation.isPending ? t('common.loading') : t('common.delete')}
            </Button>
          </div>
        </Dialog>
      </div>
    </AppLayout>
  )
}
