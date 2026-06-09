export const ALLOWED_FILE_EXTENSIONS = [
  'pdf',
  'png',
  'jpg',
  'jpeg',
  'gif',
  'webp',
  'md',
  'txt',
  'csv',
  'json',
  'zip',
] as const

export type AllowedFileExtension = (typeof ALLOWED_FILE_EXTENSIONS)[number]

/** Max decoded file size (must match file-service FILE_MAX_BYTES). */
export const MAX_FILE_BYTES = 50 * 1024 * 1024

export const FILE_INPUT_ACCEPT = ALLOWED_FILE_EXTENSIONS.map((ext) => `.${ext}`).join(',')

export function fileExtension(filename: string): string {
  const dot = filename.lastIndexOf('.')
  if (dot <= 0 || dot === filename.length - 1) return ''
  return filename.slice(dot + 1).toLowerCase()
}

export function isAllowedFile(file: File): boolean {
  const ext = fileExtension(file.name)
  return (ALLOWED_FILE_EXTENSIONS as readonly string[]).includes(ext)
}

export function isWithinSizeLimit(file: File): boolean {
  return file.size <= MAX_FILE_BYTES
}

export function allowedTypesLabel(): string {
  return ALLOWED_FILE_EXTENSIONS.map((ext) => `.${ext}`).join(', ')
}

export function maxFileSizeLabel(locale: string): string {
  const mb = MAX_FILE_BYTES / (1024 * 1024)
  return locale.startsWith('zh') ? `${mb} MB` : `${mb} MB`
}
