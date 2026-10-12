import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      // TODO: 代理后端 API 请求到 Go 服务
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
      },
      // 英雄 / 战队图标静态资源，对应后端 r.Static("/images", "./static/images")
      // 不加这段，/images/teams/*.png 在开发环境会 404，队标无法显示
      '/images': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
