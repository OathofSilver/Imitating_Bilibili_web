import { AppShell, Avatar, Button, Container, Group, Menu, Text } from '@mantine/core'
import { Link, Outlet, useNavigate } from 'react-router'
import { useAuthStore } from '@/store/useAuthStore'

export function AppLayout() {
  const status = useAuthStore((state) => state.status)
  const currentUser = useAuthStore((state) => state.currentUser)
  const logout = useAuthStore((state) => state.logout)
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    // 登出后回首页：留在资料页会立刻被守卫弹到登录页，观感突兀。
    navigate('/', { replace: true })
  }

  return (
    <AppShell header={{ height: 56 }} padding="md">
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between" wrap="nowrap">
          <Group gap="lg" wrap="nowrap">
            <Text component={Link} to="/" fw={600} c="pink" td="none">
              bilibili
            </Text>
            {/* 资料页只在登录后可达，未登录时不展示入口，避免点了被弹走。 */}
            {status === 'authenticated' ? (
              <Text component={Link} to="/profile" size="sm" c="dimmed">
                个人资料
              </Text>
            ) : null}
          </Group>

          {status === 'authenticated' && currentUser !== null ? (
            <Menu position="bottom-end" withinPortal>
              <Menu.Target>
                <Group gap="xs" style={{ cursor: 'pointer' }} wrap="nowrap">
                  <Avatar src={currentUser.avatar_url || null} size="sm" radius="xl">
                    {currentUser.nickname.slice(0, 1)}
                  </Avatar>
                  <Text size="sm" fw={500}>
                    {currentUser.nickname}
                  </Text>
                </Group>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item component={Link} to="/profile">
                  个人资料
                </Menu.Item>
                <Menu.Item onClick={() => void handleLogout()}>退出登录</Menu.Item>
              </Menu.Dropdown>
            </Menu>
          ) : (
            <Group gap="xs" wrap="nowrap">
              <Button component={Link} to="/login" variant="subtle" size="xs">
                登录
              </Button>
              <Button component={Link} to="/register" size="xs">
                注册
              </Button>
            </Group>
          )}
        </Group>
      </AppShell.Header>
      <AppShell.Main>
        <Container size="lg">
          <Outlet />
        </Container>
      </AppShell.Main>
    </AppShell>
  )
}
