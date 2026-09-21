import { Center, Loader } from '@mantine/core'
import { Navigate, Outlet, useLocation } from 'react-router'
import { redirectState } from '@/features/auth/redirect'
import { useAuthStore } from '@/store/useAuthStore'

/**
 * 需要登录的路由守卫。
 *
 * 只作体验优化：真正的权限校验在服务端（frontend/conventions
 * 「路由与访问守卫」）。这里的价值是让未登录用户直接看到登录页，
 * 而不是先渲染一个注定请求失败的页面。
 *
 * 恢复登录态期间保持加载态，避免已登录用户刷新资料页时被弹到登录页。
 */
export function RequireAuth() {
  const status = useAuthStore((state) => state.status)
  const location = useLocation()

  if (status === 'idle' || status === 'loading') {
    return (
      <Center py="xl">
        <Loader />
      </Center>
    )
  }

  if (status !== 'authenticated') {
    // 保留原目标（含查询串与 hash），登录成功后可原样回去。
    const target = `${location.pathname}${location.search}${location.hash}`
    return <Navigate to="/login" replace state={redirectState(target)} />
  }

  return <Outlet />
}
