import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
const Api = 'http://localhost:8080'
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
        target: Api,
        changeOrigin: true,
      },
      '/api': {
        target: Api,
        changeOrigin: true,
      },
      '/server-api': {
        target: Api,
        changeOrigin: true,
      },
      '/user': {
        target: Api,
        changeOrigin: true,
      },
      '/logout': {
        target: Api,
        changeOrigin: true,
      },
      '/LOL-AmongUs': {
        target: Api,
        changeOrigin: true,
      }
    }
  }
})
