import { createBrowserRouter } from 'react-router'
import { RouterProvider } from 'react-router/dom'
import { AppLayout } from '@/components/AppLayout'
import { HomePage } from '@/pages/HomePage'
import { LoginPage } from '@/pages/LoginPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { RegisterPage } from '@/pages/RegisterPage'
import { RequireAuth } from './RequireAuth'

// 路由集中声明（frontend/conventions「路由与访问守卫」）。
// 需要登录态的页面嵌在 RequireAuth 之下，未登录时统一跳转登录页。
export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: 'login', element: <LoginPage /> },
      { path: 'register', element: <RegisterPage /> },
      {
        element: <RequireAuth />,
        children: [{ path: 'profile', element: <ProfilePage /> }],
      },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
