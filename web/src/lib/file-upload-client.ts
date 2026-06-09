const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:3002'

export type ApiBody<T = unknown> = {
  code: string
  message?: string
  data?: T
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message?: string,
  ) {
    super(message ?? code)
  }
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit & { token?: string | null } = {},
): Promise<T> {
  const { token, headers, ...rest } = options
  const res = await fetch(`${API_BASE}${path}`, {
    ...rest,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
  })
  const body = (await res.json().catch(() => ({}))) as ApiBody<T>
  if (!res.ok) {
    throw new ApiError(res.status, body.code ?? 'error', body.message)
  }
  return body.data as T
}

export function readFileAsBase64WithProgress(
  file: File,
  onProgress?: (percent: number) => void,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()

    reader.onprogress = (event) => {
      if (event.lengthComputable && onProgress) {
        onProgress(Math.round((event.loaded / event.total) * 50))
      }
    }

    reader.onload = () => {
      onProgress?.(50)
      const result = reader.result as string
      const base64 = result.includes(',') ? result.split(',')[1] : result
      resolve(base64)
    }

    reader.onerror = () => reject(reader.error ?? new Error('file read failed'))
    reader.readAsDataURL(file)
  })
}
