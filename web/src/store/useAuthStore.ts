// 登录态切片。
//
// frontend/conventions「状态管理边界」：Zustand 只承载跨页面共享的
// 客户端状态。登录态天然属于这一类——页头、路由守卫、资料页都要读它。
//
// 凭证本身由 `api/credential.ts` 管（localStorage），这里只把它镜像成
// 内存状态，让 React 能订阅「是否已登录」的变化。

import { create } from 'zustand'
import { readToken, writeToken } from '@/api/credential'
import * as userApi from '@/api/user'
import type { UserProfile } from '@/types/auth'

export type AuthStatus = 'idle' | 'loading' | 'authenticated' | 'anonymous'

interface AuthState {
  /** 由凭证派生的登录态；`idle` 表示尚未做首次恢复判断。 */
  status: AuthStatus
  /** 当前登录用户资料；未登录时为 `null`。 */
  currentUser: UserProfile | null
  /** 首次挂载时恢复登录态：有凭证则拉取本人资料，失败即视为未登录。 */
  restore: () => Promise<void>
  /** 注册，成功后直接进入登录态。 */
  register: (username: string, password: string) => Promise<void>
  /** 登录。 */
  login: (username: string, password: string) => Promise<void>
  /** 退出登录：先撤销服务端凭证，再清理本地。 */
  logout: () => Promise<void>
  /** 更新当前用户资料，并回写切片。 */
  updateProfile: (payload: Parameters<typeof userApi.updateMe>[0]) => Promise<void>
  /** 就地清空登录态，供凭证失效回调使用。 */
  clear: () => void
}

/**
 * 把一次成功的鉴权结果落到本地与内存。
 *
 * 顺序很关键：先写 localStorage 再 set 状态，确保任何在状态变更后
 * 立刻发出的请求都能带上凭证。
 */
function persist(token: string): void {
  writeToken(token)
}

export const useAuthStore = create<AuthState>((set, get) => ({
  status: 'idle',
  currentUser: null,

  async restore() {
    if (readToken() === null) {
      set({ status: 'anonymous', currentUser: null })
      return
    }
    set({ status: 'loading' })
    try {
      const user = await userApi.fetchMe()
      set({ status: 'authenticated', currentUser: user })
    } catch {
      // 凭证过期或被撤销：请求层已在 40100 时清掉了 localStorage，
      // 这里只需把内存状态归位。
      set({ status: 'anonymous', currentUser: null })
    }
  },

  async register(username, password) {
    const session = await userApi.register({ username, password })
    persist(session.token)
    set({ status: 'authenticated', currentUser: session.user })
  },

  async login(username, password) {
    const session = await userApi.login({ username, password })
    persist(session.token)
    set({ status: 'authenticated', currentUser: session.user })
  },

  async logout() {
    try {
      // 先通知服务端撤销；失败也要继续清理本地，
      // 否则用户会遇到「点了登出却还登着」。
      await userApi.logout()
    } finally {
      writeToken(null)
      set({ status: 'anonymous', currentUser: null })
    }
  },

  async updateProfile(payload) {
    const user = await userApi.updateMe(payload)
    set({ currentUser: user })
  },

  clear() {
    if (get().status === 'anonymous' && get().currentUser === null) {
      return
    }
    set({ status: 'anonymous', currentUser: null })
  },
}))
