import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 18535,
    proxy: {
      '/api/v1': 'http://127.0.0.1:19535',
      '/api/healthz': {
        target: 'http://127.0.0.1:19535',
        rewrite: () => '/healthz',
      },
    },
  },
})
