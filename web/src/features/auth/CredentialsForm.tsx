// 用户名 + 密码表单，注册页与登录页共用。
//
// frontend/conventions「组件实现约定」：业务逻辑放在 features 内。
// 本组件只做受控表单与提交触发，实际的接口调用由页面通过 `onSubmit`
// 注入——这使注册与登录复用同一份校验与样式，而各自负责跳转语义。

import { Alert, Button, PasswordInput, Stack, TextInput } from '@mantine/core'
import { useState } from 'react'
import type { FormEvent } from 'react'

export interface CredentialsFormProps {
  /** 提交按钮文案。 */
  submitLabel: string
  /** 提交中时按钮的文案。 */
  pendingLabel: string
  /** 执行提交，抛出的异常由调用方负责翻译成文案后回传 `errorMessage`。 */
  onSubmit: (username: string, password: string) => Promise<void>
  /** 当前要展示的错误文案，`null` 表示无错误。 */
  errorMessage: string | null
  /** 用户名输入框下方的辅助说明，注册页用来说明命名规则。 */
  usernameHint?: string
}

/** 用户名长度下限，与后端 `validateUsername` 的 3–32 位保持一致。 */
const MIN_USERNAME_LEN = 3
/** 密码长度下限，与后端 `minPasswordLen` 的 8 位保持一致。 */
const MIN_PASSWORD_LEN = 8

export function CredentialsForm({
  submitLabel,
  pendingLabel,
  onSubmit,
  errorMessage,
  usernameHint,
}: CredentialsFormProps) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)

  // 只做「够不够提交」的前置判断，格式细节交给后端——
  // 两边都实现一遍就会有两套规则，迟早不一致。
  const canSubmit =
    username.trim().length >= MIN_USERNAME_LEN && password.length >= MIN_PASSWORD_LEN

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!canSubmit || pending) {
      return
    }
    setPending(true)
    try {
      await onSubmit(username.trim(), password)
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <Stack gap="sm">
        <TextInput
          label="用户名"
          description={usernameHint}
          value={username}
          onChange={(event) => setUsername(event.currentTarget.value)}
          autoComplete="username"
          required
        />
        <PasswordInput
          label="密码"
          description={`至少 ${MIN_PASSWORD_LEN} 位`}
          value={password}
          onChange={(event) => setPassword(event.currentTarget.value)}
          autoComplete="current-password"
          required
        />
        {errorMessage !== null ? <Alert color="red">{errorMessage}</Alert> : null}
        <Button type="submit" loading={pending} disabled={!canSubmit}>
          {pending ? pendingLabel : submitLabel}
        </Button>
      </Stack>
    </form>
  )
}
