import path from 'node:path'
import { fileURLToPath } from 'node:url'

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// API 代理配置 - 指向爬虫系统服务（使用 8081 避免与主项目 8080 冲突）
const apiTarget = 'http://localhost:8081'

const apiProxy = {
  '/api': {
    target: apiTarget,
    changeOrigin: true,
  },
  '/healthz': {
    target: apiTarget,
    changeOrigin: true,
  },
} as const

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  optimizeDeps: {
    include: [
      'react-admin',
      'ra-core',
      'ra-ui-materialui',
      'ra-i18n-polyglot',
      'ra-language-english',
      '@mui/material',
      '@mui/material/styles',
      '@mui/icons-material',
      '@emotion/react',
      '@emotion/styled',
    ],
  },
  server: {
    host: '0.0.0.0',
    port: 5174,
    strictPort: true,
    proxy: apiProxy,
  },
  preview: {
    host: '0.0.0.0',
    port: 5174,
    proxy: apiProxy,
  },
})
