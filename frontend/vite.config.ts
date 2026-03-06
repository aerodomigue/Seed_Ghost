import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { execSync } from 'child_process'

let version = process.env.APP_VERSION || 'dev'
try {
  if (version === 'dev') {
    version = execSync('git rev-parse --short HEAD').toString().trim()
  }
} catch {
  // No git available (e.g. Docker build without .git)
}

export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify(version),
  },
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8333',
    },
  },
})
