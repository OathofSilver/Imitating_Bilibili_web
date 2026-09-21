import { ApiCode } from '@/types/api'
import type { ApiResponse } from '@/types/api'
import { readToken, writeToken } from './credential'

const BASE_URL = '/api/v1'

/** 业务错误：`code` 非 0 时抛出，页面按 `code` 决定展示与分支。 */
export class ApiError extends Error {
  readonly code: number

  constructor(code: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

/**
 * 凭证失效时的回调，由应用层注册。
 *
 * 请求层刻意不 import router 或 store：否则 `api` 会反向依赖 `app`，
 * 且组件之外调用请求时也会牵出渲染层。注册回调把「谁负责跳转」
 * 留给组合根（`app/`），请求层只负责「发现失效」。
 */
type UnauthorizedHandler = () => void

let onUnauthorized: UnauthorizedHandler | null = null

export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  onUnauthorized = handler
}

/**
 * 处理一次 40100：清除本地凭证并通知应用层。
 *
 * 先清凭证再回调，保证回调里的跳转发生在「已登出」的确定状态下，
 * 避免登录页看到残留凭证而误判为已登录。
 */
function handleUnauthorized(): void {
  writeToken(null)
  onUnauthorized?.()
}

/** 拼接请求头：始终注入凭证（若有），并保留调用方给定的头。 */
function buildHeaders(init?: RequestInit): Headers {
  const headers = new Headers(init?.headers)
  if (!headers.has('Content-Type') && init?.body != null) {
    headers.set('Content-Type', 'application/json')
  }
  const token = readToken()
  if (token !== null && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  return headers
}

/** 把非 2xx 响应收敛成 ApiError，尽量取出后端给的业务码与文案。 */
async function toApiError(res: Response): Promise<ApiError> {
  let code: number = ApiCode.ServerError
  let message = `请求失败（HTTP ${res.status}）`
  try {
    const body = (await res.json()) as Partial<ApiResponse<unknown>>
    if (typeof body.code === 'number') {
      code = body.code
    }
    if (typeof body.message === 'string' && body.message !== '') {
      message = body.message
    }
  } catch {
    // 响应体不是 JSON（例如网关返回的 HTML 错误页），保留默认文案。
  }
  return new ApiError(code, message)
}

export interface RequestOptions extends RequestInit {
  /** 不注入 Authorization 头，用于注册、登录、公开资料等开放接口。 */
  skipAuth?: boolean
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { skipAuth = false, ...init } = options
  const headers = buildHeaders(init)
  if (skipAuth) {
    headers.delete('Authorization')
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers })

  if (!res.ok) {
    const error = await toApiError(res)
    if (error.code === ApiCode.Unauthorized) {
      handleUnauthorized()
    }
    throw error
  }

  const body = (await res.json()) as ApiResponse<T>
  if (body.code !== ApiCode.Ok) {
    const error = new ApiError(body.code, body.message)
    if (error.code === ApiCode.Unauthorized) {
      handleUnauthorized()
    }
    throw error
  }
  return body.data
}

/** GET，并对查询参数做编码；值为 `undefined` 的项会被跳过。 */
export function get<T>(path: string, query: Record<string, string | number | undefined> = {}): Promise<T> {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined) {
      search.set(key, String(value))
    }
  }
  const suffix = search.toString() === '' ? '' : `?${search.toString()}`
  return request<T>(`${path}${suffix}`)
}

/** POST JSON。 */
export function post<T>(path: string, body: unknown, options: RequestOptions = {}): Promise<T> {
  return request<T>(path, { ...options, method: 'POST', body: JSON.stringify(body) })
}

/** PUT JSON。 */
export function put<T>(path: string, body: unknown, options: RequestOptions = {}): Promise<T> {
  return request<T>(path, { ...options, method: 'PUT', body: JSON.stringify(body) })
}
