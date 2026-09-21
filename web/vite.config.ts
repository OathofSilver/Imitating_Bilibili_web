import { fileURLToPath, URL } from 'node:url'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// 后端各域监听端口，须与 server/app/<domain>/api/etc/*.yaml 的 Port 一致。
// 序号避开 8001-8005：该区间落在 Windows 保留端口段，服务无法绑定。
const backendPorts: Record<string, number> = {
  user: 18001,
  video: 18002,
  interaction: 18003,
  comment: 18004,
  search: 18005,
}

const proxy = Object.entries(backendPorts).reduce<Record<string, { target: string; changeOrigin: boolean }>>(
  (acc, [domain, port]) => {
    acc[`/api/v1/${domain}`] = { target: `http://127.0.0.1:${port}`, changeOrigin: true }
    return acc
  },
  {},
)

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy,
  },
})
