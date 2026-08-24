import type { ApiEnvelope, QueryParams } from '../types/common'

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
    public readonly requestId?: string,
    public readonly details?: unknown,
  ) {
    super(message)
  }
}

function queryString(params?: QueryParams): string {
  if (!params) return ''
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const encoded = query.toString()
  return encoded ? `?${encoded}` : ''
}

async function request<T>(path: string, options: RequestInit = {}, params?: QueryParams): Promise<T> {
  const token = localStorage.getItem('curing-access-token')
  const response = await fetch(`/api/v1${path}${queryString(params)}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })
  const payload = (await response.json().catch(() => ({}))) as Partial<ApiEnvelope<T>> & {
    details?: unknown
  }
  if (!response.ok) {
    if (response.status === 401 && path !== '/auth/login') window.dispatchEvent(new Event('auth-expired'))
    throw new ApiError(payload.message ?? '请求未完成', response.status, payload.code ?? 'HTTP_ERROR', payload.request_id, payload.details)
  }
  return payload.data as T
}

export const apiClient = {
  get: <T>(path: string, params?: QueryParams) => request<T>(path, {}, params),
  post: <T>(path: string, body?: unknown, headers?: HeadersInit) =>
    request<T>(path, { method: 'POST', body: body === undefined ? undefined : JSON.stringify(body), headers }),
  put: <T>(path: string, body: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
}

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.requestId ? `${error.message}（请求 ${error.requestId.slice(0, 8)}）` : error.message
  }
  return error instanceof Error ? error.message : '发生未知错误'
}
