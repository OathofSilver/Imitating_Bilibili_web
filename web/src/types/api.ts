export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

/**
 * 统一响应里的业务错误码，与 server/common/errcode 保持一致。
 *
 * 前端只挑需要按码分支处理的几个登记在此——`api/contract` 要求
 * 前端 MUST 能仅凭 code 区分错误类型，因此严禁在页面里写裸数字。
 */
export const ApiCode = {
  /** 成功。 */
  Ok: 0,
  /** 参数非法，message 中会指明字段。 */
  InvalidParam: 40001,
  /** 未认证：凭证缺失、过期、被撤销或凭据错误。 */
  Unauthorized: 40100,
  /** 资源不存在。 */
  NotFound: 40400,
  /** 资源冲突（用户名已被占用）。 */
  Conflict: 40901,
  /** 服务端内部错误或依赖不可用。 */
  ServerError: 50001,
} as const

export type ApiCodeValue = (typeof ApiCode)[keyof typeof ApiCode]

export interface PageResult<T> {
  list: T[]
  page: number
  page_size: number
  has_more: boolean
  total?: number
}

export interface HealthData {
  service: string
  dependencies: Record<string, boolean>
}
