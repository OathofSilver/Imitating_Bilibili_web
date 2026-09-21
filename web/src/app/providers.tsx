import { MantineProvider } from '@mantine/core'
import type { ReactNode } from 'react'
import { theme } from '@/theme/mantineTheme'
import { AuthBootstrap } from './AuthBootstrap'

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <MantineProvider theme={theme} defaultColorScheme="light">
      {/* AuthBootstrap 放在路由之外：它注册的 40100 回调与首次登录态恢复
          都不依赖当前路由，且必须早于任何受保护页面开始渲染。 */}
      <AuthBootstrap>{children}</AuthBootstrap>
    </MantineProvider>
  )
}
