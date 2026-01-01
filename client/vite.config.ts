import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import commonjs from 'vite-plugin-commonjs'

export default defineConfig({
  plugins: [react(), commonjs()],
  server: {
    proxy: {
      '/players': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/players/, ''),
      },
    },
  },
})

