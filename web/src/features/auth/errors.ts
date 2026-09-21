// 鉴权表单的错误文案映射。
//
// `api/contract` 要求前端能仅凭 code 区分错误类型，注册页据此把
// 「用户名已被占用」与「字段格式错」区分开——前者给出具体原因，
// 后者把后端 message 透出来（后端已指明字段名）。
// 因此这里严禁把 40901 归并进通用错误文案。

import { ApiCode } from '@/types/api'
import { ApiError } from '@/api/http'

/** 把接口异常翻译成可直接展示的中文文案。 */
export function describeAuthError(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.code) {
      case ApiCode.Conflict:
        return '该用户名已被占用，换一个试试'
      case ApiCode.Unauthorized:
        return '用户名或密码不正确'
      case ApiCode.InvalidParam:
        // 后端 message 里已含字段名与原因，直接透出最准确。
        return error.message
      case ApiCode.ServerError:
        return '服务暂时不可用，请稍后重试'
      default:
        return error.message
    }
  }
  if (error instanceof TypeError) {
    // fetch 在网络层失败时抛 TypeError。
    return '无法连接服务器，请检查网络或后端是否已启动'
  }
  return '发生未知错误，请稍后重试'
}
