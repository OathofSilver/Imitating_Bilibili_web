import { Anchor, Card, Stack, Text, Title } from '@mantine/core'
import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router'
import { CredentialsForm } from '@/features/auth/CredentialsForm'
import { describeAuthError } from '@/features/auth/errors'
import { readRedirectTarget } from '@/features/auth/redirect'
import { useAuthStore } from '@/store/useAuthStore'

export function LoginPage() {
  const login = useAuthStore((state) => state.login)
  const navigate = useNavigate()
  const location = useLocation()
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (username: string, password: string) => {
    setError(null)
    try {
      await login(username, password)
      // 登录成功后回到守卫记录的原目标；没有则回首页。
      navigate(readRedirectTarget(location.state) ?? '/', { replace: true })
    } catch (err) {
      setError(describeAuthError(err))
    }
  }

  return (
    <Card withBorder radius="md" padding="lg" maw={420} mx="auto" mt="xl">
      <Stack gap="md">
        <Title order={3}>登录</Title>
        <Text size="sm" c="dimmed">
          登录后即可查看与修改个人资料。
        </Text>
        <CredentialsForm
          submitLabel="登录"
          pendingLabel="登录中"
          onSubmit={handleSubmit}
          errorMessage={error}
        />
        <Text size="sm" c="dimmed">
          还没有账号？
          <Anchor component={Link} to="/register" ml={4}>
            去注册
          </Anchor>
        </Text>
      </Stack>
    </Card>
  )
}
