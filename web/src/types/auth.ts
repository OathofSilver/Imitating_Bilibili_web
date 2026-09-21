// 用户与鉴权相关的类型定义。
//
// 与 server/app/user/api/internal/types/types.go 的 json 标签严格对应：
// 类型在这里集中声明，由 api 层复用（frontend/conventions「类型与接口封装」）。

/** 性别取值，与后端 users.gender 的约定一致。 */
export const GENDER_UNKNOWN = 0
export const GENDER_MALE = 1
export const GENDER_FEMALE = 2

/** 资料修改接口可提交的字段，与后端白名单一一对应。 */
export interface UpdateProfilePayload {
  nickname?: string
  avatar_url?: string
  signature?: string
  gender?: number
  birthday?: string
}

/** 本人资料，对应后端的 UserDTO。 */
export interface UserProfile {
  id: number
  username: string
  nickname: string
  avatar_url: string
  signature: string
  gender: number
  /** 生日，`YYYY-MM-DD`；未填写为空串。 */
  birthday: string
  level: number
  role: number
  /** 注册时间，RFC3339；未填充时为空串。 */
  created_at: string
}

/**
 * 他人公开资料，对应后端的 PublicUserDTO。
 *
 * 刻意不含 username、birthday、created_at——后端不会返回，
 * 前端类型也不该假装它们存在（identity/profile「他人公开资料查询」）。
 */
export interface PublicUserProfile {
  id: number
  nickname: string
  avatar_url: string
  signature: string
  gender: number
  level: number
}

/** 注册请求体。 */
export interface RegisterPayload {
  username: string
  password: string
}

/** 登录请求体。 */
export interface LoginPayload {
  username: string
  password: string
}

/**
 * 注册与登录的共同响应体：直接携带凭证，使成功即进入登录态。
 *
 * 与后端 RegisterData / LoginData 结构一致，二者刻意共用同一类型，
 * 因为前端对它们的处理完全相同。
 */
export interface AuthSession {
  token: string
  user: UserProfile
}
