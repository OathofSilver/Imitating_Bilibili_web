import { Badge, Card, Divider, Group, Loader, Stack, Text, Title } from '@mantine/core'
import { useState } from 'react'
import { ApiError } from '@/api/http'
import { describeAuthError } from '@/features/auth/errors'
import { ProfileForm } from '@/features/profile/ProfileForm'
import { useAuthStore } from '@/store/useAuthStore'
import { ApiCode } from '@/types/api'
import type { UpdateProfilePayload, UserProfile } from '@/types/auth'

const GENDER_LABELS: Record<number, string> = {
  0: '保密',
  1: '男',
  2: '女',
}

/** 把 RFC3339 时间戳裁成 `YYYY-MM-DD`；空串原样返回。 */
function toDateOnly(value: string): string {
  return value.length >= 10 ? value.slice(0, 10) : value
}

/**
 * 资料表单的 key：任一可编辑字段变化就换 key。
 *
 * 让表单以服务端返回值为准重新初始化——否则提交成功后本地态
 * 仍是旧值，用户会以为没生效（3.6 验收要求提交后展示新值）。
 */
function profileFormKey(profile: UserProfile): string {
  return [
    profile.nickname,
    profile.avatar_url,
    profile.signature,
    profile.gender,
    profile.birthday,
  ].join('|')
}

export function ProfilePage() {
  const profile = useAuthStore((state) => state.currentUser)
  const status = useAuthStore((state) => state.status)
  const updateProfile = useAuthStore((state) => state.updateProfile)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  // 路由守卫已保证进到这里时是登录态，此分支只覆盖
  // 「凭证恰好在渲染间隙失效」的边角情况。
  if (status === 'loading' || status === 'idle') {
    return (
      <Group justify="center" py="xl">
        <Loader />
      </Group>
    )
  }
  if (profile === null) {
    return (
      <Card withBorder radius="md" padding="lg" maw={640} mx="auto" mt="xl">
        <Text>登录态已失效，请重新登录。</Text>
      </Card>
    )
  }

  const handleSubmit = async (payload: UpdateProfilePayload) => {
    setError(null)
    setSaved(false)
    try {
      await updateProfile(payload)
    } catch (err) {
      if (err instanceof ApiError && err.code === ApiCode.InvalidParam) {
        setError(err.message)
        return
      }
      setError(describeAuthError(err))
    }
  }

  return (
    <Stack gap="md" py="md" maw={640} mx="auto">
      <Card withBorder radius="md" padding="lg">
        <Stack gap="xs">
          <Title order={3}>个人资料</Title>
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              用户名
            </Text>
            <Text size="sm" fw={500}>
              {profile.username}
            </Text>
            <Badge variant="light">Lv{profile.level}</Badge>
          </Group>
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              注册时间
            </Text>
            <Text size="sm">{toDateOnly(profile.created_at) || '—'}</Text>
          </Group>
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              性别
            </Text>
            <Text size="sm">{GENDER_LABELS[profile.gender] ?? '保密'}</Text>
          </Group>
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              生日
            </Text>
            <Text size="sm">{profile.birthday || '—'}</Text>
          </Group>
          <Group gap="xs" align="flex-start">
            <Text size="sm" c="dimmed">
              签名
            </Text>
            <Text size="sm">{profile.signature || '—'}</Text>
          </Group>
        </Stack>
      </Card>

      <Card withBorder radius="md" padding="lg">
        <Stack gap="md">
          <Text fw={500}>编辑资料</Text>
          <Divider />
          <ProfileForm
            key={profileFormKey(profile)}
            profile={profile}
            onSubmit={handleSubmit}
            errorMessage={error}
            onSaved={() => setSaved(true)}
          />
          {saved ? (
            <Text size="sm" c="teal">
              已保存
            </Text>
          ) : null}
        </Stack>
      </Card>
    </Stack>
  )
}
