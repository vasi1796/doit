import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Prevent dev server crash on abrupt client disconnects (mobile devices)
process.on('uncaughtException', (err) => {
  if ('code' in err && err.code === 'ECONNRESET') return
  console.error(err)
  process.exit(1)
})

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8080', ws: true },
      '/auth': 'http://localhost:8080',
    },
  },
})
