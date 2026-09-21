// 用户域的接口封装。
//
// 页面与 features 一律通过本模块访问用户域，不得直接 fetch
// （frontend/conventions「类型与接口封装」）。
// 路径前缀 `/api/v1/user` 由 Vite 代理转发到用户域服务。

import type {
  AuthSession,
  LoginPayload,
  PublicUserProfile,
  RegisterPayload,
  UpdateProfilePayload,
  UserProfile,
} from '@/types/auth'
import { get, post, put } from './http'

const PREFIX = '/user'

/**
 * 注册。
 *
 * 后端注册成功即直接签发凭证（identity/auth「用户注册」），
 * 因此返回值里带 token，调用方拿到即可进入登录态。
 * 开放接口，不注入 Authorization。
 */
export function register(payload: RegisterPayload): Promise<AuthSession> {
  return post<AuthSession>(`${PREFIX}/register`, payload, { skipAuth: true })
}

/** 登录。开放接口，不注入 Authorization。 */
export function login(payload: LoginPayload): Promise<AuthSession> {
  return post<AuthSession>(`${PREFIX}/login`, payload, { skipAuth: true })
}

/**
 * 退出登录。
 *
 * 服务端会递增会话版本号使凭证立即失效（identity/auth「登录态撤销」），
 * 因此必须在清除本地凭证之前调用，否则请求带不上凭证、撤销无从生效。
 */
export function logout(): Promise<null> {
  return post<null>(`${PREFIX}/logout`, {})
}

/** 查询本人资料，需要登录态。 */
export function fetchMe(): Promise<UserProfile> {
  return get<UserProfile>(`${PREFIX}/me`)
}

/** 修改本人资料，只提交需要变更的字段。 */
export function updateMe(payload: UpdateProfilePayload): Promise<UserProfile> {
  return put<UserProfile>(`${PREFIX}/me`, payload)
}

/** 查询他人公开资料，开放接口。 */
export function fetchPublicProfile(userId: number): Promise<PublicUserProfile> {
  return get<PublicUserProfile>(`${PREFIX}/users/${userId}`)
}
