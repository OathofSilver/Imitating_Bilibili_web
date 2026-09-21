// 登录态的运行时接线。
//
// 这里做三件事，都属于「组合根」职责，不宜散落到页面里：
// 1. 把请求层的 40100 回调接到 store 的 clear，使凭证失效能反映到 UI；
// 2. 首次挂载时尝试用本地凭证恢复登录态；
// 3. 渲染前先等首次恢复结束，避免页头在「已登录」与「未登录」之间闪烁。

import { useEffect } from 'react'
import { setUnauthorizedHandler } from '@/api/http'
import { useAuthStore } from '@/store/useAuthStore'
import type { ReactNode } from 'react'

export function AuthBootstrap({ children }: { children: ReactNode }) {
  const status = useAuthStore((state) => state.status)
  const restore = useAuthStore((state) => state.restore)
  const clear = useAuthStore((state) => state.clear)

  useEffect(() => {
    setUnauthorizedHandler(() => {
      clear()
    })
    return () => {
      setUnauthorizedHandler(null)
    }
  }, [clear])

  useEffect(() => {
    if (status === 'idle') {
      void restore()
    }
  }, [status, restore])

  return <>{children}</>
}
