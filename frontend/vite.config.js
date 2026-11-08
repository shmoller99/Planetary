
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/posts': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/comments': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/users': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },

      '/chat': {
  target: 'http://localhost:8080',
  changeOrigin: true,
},
    },
  },
})
