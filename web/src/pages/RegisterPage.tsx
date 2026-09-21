import { Anchor, Card, Stack, Text, Title } from '@mantine/core'
import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router'
import { CredentialsForm } from '@/features/auth/CredentialsForm'
import { describeAuthError } from '@/features/auth/errors'
import { readRedirectTarget } from '@/features/auth/redirect'
import { useAuthStore } from '@/store/useAuthStore'

export function RegisterPage() {
  const register = useAuthStore((state) => state.register)
  const navigate = useNavigate()
  const location = useLocation()
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (username: string, password: string) => {
    setError(null)
    try {
      // 后端注册成功直接签发凭证（identity/auth「用户注册」），
      // 因此这里无需再走一次登录。
      await register(username, password)
      navigate(readRedirectTarget(location.state) ?? '/', { replace: true })
    } catch (err) {
      setError(describeAuthError(err))
    }
  }

  return (
    <Card withBorder radius="md" padding="lg" maw={420} mx="auto" mt="xl">
      <Stack gap="md">
        <Title order={3}>注册</Title>
        <Text size="sm" c="dimmed">
          注册成功即自动登录。
        </Text>
        <CredentialsForm
          submitLabel="注册"
          pendingLabel="注册中"
          onSubmit={handleSubmit}
          errorMessage={error}
          usernameHint="3-32 位，仅限字母、数字与下划线"
        />
        <Text size="sm" c="dimmed">
          已有账号？
          <Anchor component={Link} to="/login" ml={4}>
            去登录
          </Anchor>
        </Text>
      </Stack>
    </Card>
  )
}
