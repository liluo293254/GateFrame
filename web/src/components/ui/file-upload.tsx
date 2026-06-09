import { Loader2, Upload } from 'lucide-react'
import * as React from 'react'
import { useState } from 'react'
import { cn } from '@/lib/utils'

export const fileZoneSurfaceClassName =
  'rounded-xl border-2 border-dashed border-muted-foreground/25 bg-muted/20'

type FileZoneShellProps = {
  className?: string
  children: React.ReactNode
}

export function FileZoneShell({ className, children }: FileZoneShellProps) {
  return (
    <div className={cn('flex w-full flex-col items-center gap-3 px-6 py-10 text-center', fileZoneSurfaceClassName, className)}>
      {children}
    </div>
  )
}

export type FileUploadZoneProps = {
  id: string
  title: string
  hint: string
  allowedTypesHint?: string
  browseLabel: string
  uploadingLabel: string
  uploadingDetail?: string | null
  progress?: number | null
  accept?: string
  multiple?: boolean
  disabled?: boolean
  uploading?: boolean
  onFilesSelect: (files: File[]) => void
  className?: string
}

export const FileUploadZone = React.forwardRef<HTMLInputElement, FileUploadZoneProps>(
  (
    {
      id,
      title,
      hint,
      allowedTypesHint,
      browseLabel,
      uploadingLabel,
      uploadingDetail,
      progress = null,
      accept,
      multiple = false,
      disabled = false,
      uploading = false,
      onFilesSelect,
      className,
    },
    ref,
  ) => {
    const [isDragging, setIsDragging] = useState(false)
    const inactive = disabled || uploading

    function handleFiles(fileList: FileList | null | undefined) {
      if (!fileList || inactive) return
      const files = Array.from(fileList)
      if (files.length > 0) {
        onFilesSelect(files)
      }
    }

    function handleDragOver(event: React.DragEvent<HTMLLabelElement>) {
      event.preventDefault()
      if (!inactive) {
        setIsDragging(true)
      }
    }

    function handleDragLeave(event: React.DragEvent<HTMLLabelElement>) {
      event.preventDefault()
      setIsDragging(false)
    }

    function handleDrop(event: React.DragEvent<HTMLLabelElement>) {
      event.preventDefault()
      setIsDragging(false)
      if (inactive) return
      handleFiles(event.dataTransfer.files)
    }

    return (
      <label
        htmlFor={id}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={cn(
          'group relative flex w-full cursor-pointer flex-col items-center gap-4 px-6 py-8 text-center transition-all sm:flex-row sm:items-center sm:text-left',
          fileZoneSurfaceClassName,
          isDragging
            ? 'scale-[1.01] border-primary bg-primary/5 shadow-sm'
            : 'hover:border-primary/50 hover:bg-muted/40',
          inactive && 'cursor-not-allowed opacity-70',
          className,
        )}
      >
        <input
          ref={ref}
          id={id}
          type="file"
          accept={accept}
          multiple={multiple}
          disabled={inactive}
          className="sr-only"
          onChange={(event) => {
            handleFiles(event.target.files)
          }}
        />

        <span
          className={cn(
            'flex size-14 shrink-0 items-center justify-center rounded-full transition-colors',
            isDragging ? 'bg-primary/15 text-primary' : 'bg-primary/10 text-primary group-hover:bg-primary/15',
          )}
        >
          {uploading ? (
            <Loader2 className="size-7 animate-spin" aria-hidden />
          ) : (
            <Upload className="size-7" aria-hidden />
          )}
        </span>

        <span className="min-w-0 flex-1 space-y-1">
          <span className="block text-base font-medium text-foreground">{title}</span>
          {uploading && uploadingDetail ? (
            <span className="block truncate text-sm text-muted-foreground">{uploadingDetail}</span>
          ) : (
            <span className="block text-sm text-muted-foreground">{uploading ? uploadingLabel : hint}</span>
          )}
          {!uploading && allowedTypesHint ? (
            <span className="block text-xs text-muted-foreground/80">{allowedTypesHint}</span>
          ) : null}
          {uploading && progress != null ? (
            <span className="mt-2 block w-full max-w-md space-y-1">
              <span className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{uploadingLabel}</span>
                <span>{progress}%</span>
              </span>
              <span className="block h-2 overflow-hidden rounded-full bg-muted">
                <span
                  className="block h-full rounded-full bg-primary transition-[width] duration-200"
                  style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
                />
              </span>
            </span>
          ) : null}
        </span>

        {!uploading ? (
          <span className="inline-flex h-10 shrink-0 items-center justify-center rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground shadow-sm transition-opacity group-hover:opacity-90">
            {browseLabel}
          </span>
        ) : null}
      </label>
    )
  },
)
FileUploadZone.displayName = 'FileUploadZone'
