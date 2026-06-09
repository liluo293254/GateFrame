import {
  File,
  FileArchive,
  FileCode,
  FileImage,
  FileSpreadsheet,
  FileText,
  Film,
  Music,
  type LucideIcon,
} from 'lucide-react'

const rtfUnits: Array<[Intl.RelativeTimeFormatUnit, number]> = [
  ['year', 60 * 60 * 24 * 365],
  ['month', 60 * 60 * 24 * 30],
  ['week', 60 * 60 * 24 * 7],
  ['day', 60 * 60 * 24],
  ['hour', 60 * 60],
  ['minute', 60],
  ['second', 1],
]

const imageExt = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico'])
const videoExt = new Set(['mp4', 'mov', 'avi', 'webm', 'mkv'])
const audioExt = new Set(['mp3', 'wav', 'ogg', 'flac', 'aac'])
const archiveExt = new Set(['zip', 'rar', '7z', 'tar', 'gz', 'bz2'])
const spreadsheetExt = new Set(['xls', 'xlsx', 'csv'])
const textExt = new Set(['md', 'markdown', 'txt', 'pdf', 'doc', 'docx', 'rtf'])
const codeExt = new Set(['js', 'ts', 'tsx', 'jsx', 'go', 'rs', 'py', 'json', 'yaml', 'yml', 'html', 'css', 'sql', 'sh'])

function fileExtension(filename: string): string {
  const dot = filename.lastIndexOf('.')
  if (dot <= 0 || dot === filename.length - 1) return ''
  return filename.slice(dot + 1).toLowerCase()
}

export function formatAbsoluteTime(iso: string, locale: string): string {
  try {
    return new Date(iso).toLocaleString(locale === 'zh-CN' ? 'zh-CN' : undefined)
  } catch {
    return iso
  }
}

export function formatRelativeTime(iso: string, locale: string): string {
  try {
    const date = new Date(iso)
    const diffSeconds = Math.round((date.getTime() - Date.now()) / 1000)
    const rtf = new Intl.RelativeTimeFormat(locale === 'zh-CN' ? 'zh-CN' : 'en', {
      numeric: 'auto',
    })

    for (const [unit, secondsInUnit] of rtfUnits) {
      if (Math.abs(diffSeconds) >= secondsInUnit || unit === 'second') {
        return rtf.format(Math.round(diffSeconds / secondsInUnit), unit)
      }
    }
    return formatAbsoluteTime(iso, locale)
  } catch {
    return iso
  }
}

export function getFileIcon(filename: string, contentType?: string): LucideIcon {
  const ext = fileExtension(filename)
  const type = contentType?.toLowerCase() ?? ''

  if (type.startsWith('image/') || imageExt.has(ext)) return FileImage
  if (type.startsWith('video/') || videoExt.has(ext)) return Film
  if (type.startsWith('audio/') || audioExt.has(ext)) return Music
  if (type.includes('zip') || type.includes('compressed') || archiveExt.has(ext)) return FileArchive
  if (type.includes('spreadsheet') || type.includes('csv') || spreadsheetExt.has(ext)) return FileSpreadsheet
  if (type.startsWith('text/') || type.includes('pdf') || type.includes('document') || textExt.has(ext)) {
    return FileText
  }
  if (type.includes('json') || type.includes('javascript') || codeExt.has(ext)) return FileCode
  return File
}

export function formatContentTypeLabel(contentType: string): string {
  if (!contentType) return '—'
  const subtype = contentType.split('/').pop()?.toLowerCase() ?? contentType
  const labels: Record<string, string> = {
    plain: 'Plain text',
    markdown: 'Markdown',
    csv: 'CSV',
    pdf: 'PDF',
    json: 'JSON',
    javascript: 'JavaScript',
    'svg+xml': 'SVG',
  }
  if (labels[subtype]) return labels[subtype]
  return subtype
    .split(/[+.-]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

export function fileIconTone(filename: string, contentType?: string): string {
  const ext = fileExtension(filename)
  const type = contentType?.toLowerCase() ?? ''

  if (type.startsWith('image/') || imageExt.has(ext)) {
    return 'bg-sky-500/10 text-sky-600 dark:text-sky-400'
  }
  if (type.startsWith('video/') || videoExt.has(ext)) {
    return 'bg-violet-500/10 text-violet-600 dark:text-violet-400'
  }
  if (type.startsWith('audio/') || audioExt.has(ext)) {
    return 'bg-pink-500/10 text-pink-600 dark:text-pink-400'
  }
  if (type.includes('zip') || archiveExt.has(ext)) {
    return 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
  }
  if (type.includes('spreadsheet') || spreadsheetExt.has(ext)) {
    return 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
  }
  if (codeExt.has(ext) || type.includes('json') || type.includes('javascript')) {
    return 'bg-orange-500/10 text-orange-600 dark:text-orange-400'
  }
  return 'bg-muted text-muted-foreground'
}
