import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    // Windows 바인드 마운트는 파일 변경 이벤트가 컨테이너로 전달되지 않아 polling 으로 감시한다.
    watch: {
      usePolling: true,
      interval: 300,
    },
    // /api/* 요청을 백엔드로 넘긴다. 예: /api/healthz → {BACKEND_URL}/healthz
    proxy: {
      '/api': {
        target: process.env.BACKEND_URL || 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
