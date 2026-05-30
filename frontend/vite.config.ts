import path from 'node:path'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, '.'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/sse': 'http://127.0.0.1:8080',
      '/message': 'http://127.0.0.1:8080',
      '/stream': 'http://127.0.0.1:8080',
    },
  },
})
