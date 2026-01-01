import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    proxy: {
      '/Authentication': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/mc-api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/server-api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/user': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/logout': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      }
    }
  }
})