// 登录凭证在浏览器侧的读写位置。
//
// 单独成模块（而非直接放在 store 里）是为了打破循环依赖：
// `api/http.ts` 需要读凭证，而 `store` 需要调 `api` 登录接口——
// 若凭证读写写在 store 中，二者就互指。凭证只是一个字符串，
// 放在这里可让请求层与状态层都只依赖它。
//
// 决策 7：凭证存 localStorage 并走 `Authorization` 请求头。
// 代价是对 XSS 无抵抗力，缓解手段见变更 design.md 的 Risks。

const TOKEN_KEY = 'bw.auth.token'

/**
 * 读取当前凭证。
 *
 * localStorage 在隐私模式或被禁用时会抛异常，因此整体包在 try 中：
 * 读不到凭证的后果是「未登录」，这比整个应用崩溃要好。
 */
export function readToken(): string | null {
  try {
    return window.localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

/** 写入凭证；传 `null` 表示清除。 */
export function writeToken(token: string | null): void {
  try {
    if (token === null) {
      window.localStorage.removeItem(TOKEN_KEY)
      return
    }
    window.localStorage.setItem(TOKEN_KEY, token)
  } catch {
    // 写失败不致命：本次会话内的内存状态仍然可用，
    // 只是刷新后需要重新登录。
  }
}
