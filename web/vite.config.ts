import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    modulePreload: {
      polyfill: false,
      resolveDependencies: (_filename, deps) => {
        // Only preload the tiny essentials (react, main entry, client).
        // Skip charts (112KB gzip, only needed after login) and page chunks.
        return deps.filter(d =>
          !d.includes('/charts-') &&
          !d.includes('/Dashboard-') &&
          !d.includes('/Users-') &&
          !d.includes('/Nodes-') &&
          !d.includes('/UserPortal-') &&
          !d.includes('/Layout-') &&
          !d.includes('/Plans-') &&
          !d.includes('/PlanEditModal-') &&
          !d.includes('/StaticIPs-') &&
          !d.includes('/icons-')
        )
      },
    },
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes('node_modules/recharts') || id.includes('node_modules/d3-')) return 'charts'
          if (id.includes('node_modules/react') || id.includes('node_modules/react-dom') || id.includes('node_modules/react-router-dom') || id.includes('node_modules/scheduler')) return 'react'
          if (id.includes('node_modules/lucide-react')) return 'icons'
        },
      },
    },
  },
})
