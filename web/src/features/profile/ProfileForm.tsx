// 资料编辑表单。
//
// 只提交被改动的字段：后端把「未提交」与「提交为空值」区分开
// （UpdateProfileReq 全指针），所以少传一个字段不等于把它清空。
// 反过来说，若把所有字段都提交一遍，就等于把空字符串写进未填字段。
//
// 白名单外的字段在前端根本不出现——后端对越界字段是整体拒绝，
// 这里也不给用户制造触发它的机会（identity/profile「本人资料修改」）。

import { Alert, Button, Group, Select, Stack, TextInput, Textarea } from '@mantine/core'
import { useState } from 'react'
import type { FormEvent } from 'react'
import { GENDER_FEMALE, GENDER_MALE, GENDER_UNKNOWN } from '@/types/auth'
import type { UpdateProfilePayload, UserProfile } from '@/types/auth'

export interface ProfileFormProps {
  profile: UserProfile
  onSubmit: (payload: UpdateProfilePayload) => Promise<void>
  errorMessage: string | null
  /** 保存成功后的提示钩子，由页面决定展示方式。 */
  onSaved: () => void
}

const GENDER_OPTIONS = [
  { value: String(GENDER_UNKNOWN), label: '保密' },
  { value: String(GENDER_MALE), label: '男' },
  { value: String(GENDER_FEMALE), label: '女' },
]

/** 生日输入格式，与后端 `parseBirthday` 一致。 */
const BIRTHDAY_PLACEHOLDER = '1990-01-01'

export function ProfileForm({ profile, onSubmit, errorMessage, onSaved }: ProfileFormProps) {
  const [nickname, setNickname] = useState(profile.nickname)
  const [avatarUrl, setAvatarUrl] = useState(profile.avatar_url)
  const [signature, setSignature] = useState(profile.signature)
  const [gender, setGender] = useState(String(profile.gender))
  const [birthday, setBirthday] = useState(profile.birthday)
  const [pending, setPending] = useState(false)

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (pending) {
      return
    }
    setPending(true)
    try {
      // 逐字段比对，只把真正变了的放进请求体。
      const payload: UpdateProfilePayload = {}
      if (nickname !== profile.nickname) {
        payload.nickname = nickname
      }
      if (avatarUrl !== profile.avatar_url) {
        payload.avatar_url = avatarUrl
      }
      if (signature !== profile.signature) {
        payload.signature = signature
      }
      if (gender !== String(profile.gender)) {
        payload.gender = Number(gender)
      }
      if (birthday !== profile.birthday) {
        payload.birthday = birthday
      }
      if (Object.keys(payload).length === 0) {
        onSaved()
        return
      }
      await onSubmit(payload)
      onSaved()
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <Stack gap="sm">
        <TextInput
          label="昵称"
          value={nickname}
          onChange={(event) => setNickname(event.currentTarget.value)}
          required
        />
        <TextInput
          label="头像地址"
          description="填写图片 URL，本版本不支持上传文件"
          value={avatarUrl}
          onChange={(event) => setAvatarUrl(event.currentTarget.value)}
        />
        <Textarea
          label="个性签名"
          value={signature}
          onChange={(event) => setSignature(event.currentTarget.value)}
          autosize
          minRows={2}
        />
        <Select
          label="性别"
          description="与后端 gender 字段的取值一一对应"
          value={gender}
          onChange={(value) => setGender(value ?? String(GENDER_UNKNOWN))}
          data={GENDER_OPTIONS}
          allowDeselect={false}
          checkIconPosition="right"
        />
        <TextInput
          label="生日"
          description={`格式 ${BIRTHDAY_PLACEHOLDER}，留空表示不公开`}
          placeholder={BIRTHDAY_PLACEHOLDER}
          value={birthday}
          onChange={(event) => setBirthday(event.currentTarget.value)}
        />
        {errorMessage !== null ? <Alert color="red">{errorMessage}</Alert> : null}
        <Group justify="flex-end">
          <Button type="submit" loading={pending}>
            保存
          </Button>
        </Group>
      </Stack>
    </form>
  )
}
