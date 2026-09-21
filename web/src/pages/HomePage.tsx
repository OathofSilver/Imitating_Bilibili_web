import { Alert, Badge, Button, Card, Code, Group, Stack, Text, Title } from '@mantine/core'
import { useState } from 'react'
import { fetchHealth } from '@/api/health'
import type { HealthData } from '@/types/api'

const DOMAINS = ['user', 'video', 'interaction', 'comment', 'search']

export function HomePage() {
  const [health, setHealth] = useState<HealthData | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const check = async () => {
    setLoading(true)
    setError(null)
    try {
      setHealth(await fetchHealth('video'))
    } catch (err) {
      setError(err instanceof Error ? err.message : '未知错误')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Stack gap="md" py="md">
      <Title order={2}>项目骨架已就绪</Title>
      <Text c="dimmed">
        前端 Vite + React 18 + Mantine 已启动。后端按业务域划分为 {DOMAINS.length} 个服务，
        健康检查路径为 <Code>/api/v1/&lt;domain&gt;/health</Code>。
      </Text>

      <Card withBorder radius="md" padding="md">
        <Group justify="space-between">
          <Text fw={500}>后端联通自检</Text>
          <Button size="xs" loading={loading} onClick={check}>
            检查 video 服务
          </Button>
        </Group>
        {error ? (
          <Alert color="red" mt="sm">
            {error}
          </Alert>
        ) : null}
        {health ? (
          <Stack gap="xs" mt="sm">
            <Text size="sm">
              服务：<Code>{health.service}</Code>
            </Text>
            <Group gap="xs">
              {Object.entries(health.dependencies).map(([name, ok]) => (
                <Badge key={name} color={ok ? 'green' : 'gray'} variant="light">
                  {name}: {ok ? '可用' : '不可用'}
                </Badge>
              ))}
            </Group>
          </Stack>
        ) : null}
      </Card>

      <Group gap="xs">
        {DOMAINS.map((domain) => (
          <Badge key={domain} variant="default">
            {domain}
          </Badge>
        ))}
      </Group>
    </Stack>
  )
}
