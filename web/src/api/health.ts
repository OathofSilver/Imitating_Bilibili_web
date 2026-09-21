// 各域健康检查接口的封装。
//
// 骨架阶段 HomePage 直接调用 http.ts 的 fetchHealth；请求层重构后
// 统一收到这里，页面不再关心路径拼装（frontend/conventions「类型与接口封装」）。

import type { HealthData } from '@/types/api'
import { get } from './http'

/** 查询某个业务域的健康状态，不需要登录态。 */
export function fetchHealth(domain: string): Promise<HealthData> {
  return get<HealthData>(`/${domain}/health`)
}
