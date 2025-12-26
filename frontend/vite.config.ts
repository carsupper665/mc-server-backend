import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/mc-api': 'http://localhost:8080',
      '/server-api': 'http://localhost:8080',
      '/test-api': 'http://localhost:8080',
      '/Authentication': 'http://localhost:8080',
      '/logout': 'http://localhost:8080',
      '/user': 'http://localhost:8080',
      '/op': 'http://localhost:8080',
    }
  }
})
