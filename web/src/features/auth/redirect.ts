// 路由守卫与登录页之间传递「原目标地址」的约定。
//
// 单独成模块是为了避免循环依赖：守卫在 `app/router.tsx` 里写这个值，
// 登录页在 `pages/` 里读它，若常量与读取函数定义在任一侧，
// 都会让 `app` 与 `pages` 互指。

/** 存放原目标地址的 location.state 键名。 */
export const REDIRECT_STATE_KEY = 'from'

/** 构造守卫跳转时携带的 state。 */
export function redirectState(from: string): Record<string, string> {
  return { [REDIRECT_STATE_KEY]: from }
}

/**
 * 读出守卫记录的原目标地址。
 *
 * 只接受以 `/` 开头的站内路径：state 可能来自被篡改的 history，
 * 不做这层校验就等于开放了任意地址跳转。
 */
export function readRedirectTarget(state: unknown): string | null {
  if (typeof state !== 'object' || state === null) {
    return null
  }
  const value = (state as Record<string, unknown>)[REDIRECT_STATE_KEY]
  if (typeof value !== 'string' || !value.startsWith('/')) {
    return null
  }
  // `//evil.com` 会被浏览器当作协议相对 URL，必须排除。
  if (value.startsWith('//')) {
    return null
  }
  return value
}
