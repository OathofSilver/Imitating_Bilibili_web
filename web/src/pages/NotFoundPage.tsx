import { Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router'

export function NotFoundPage() {
  return (
    <Stack gap="xs" py="xl">
      <Title order={2}>页面不存在</Title>
      <Text c="dimmed">该路径没有匹配的路由。</Text>
      <Link to="/">返回首页</Link>
    </Stack>
  )
}
