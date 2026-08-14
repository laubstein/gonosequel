import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The Go API server's port, so `make dev` can point this at whatever
// HTTP_PORT it started the backend on. Defaults to 8081 to match the
// Go server's own default (see pkg/command/options.go).
const apiPort = process.env.VITE_API_PORT ?? '8081'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': `http://localhost:${apiPort}`,
    },
  },
  build: {
    outDir: 'dist',
  },
})
