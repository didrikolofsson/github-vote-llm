import path from 'path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import type { Plugin } from 'vite';

// In dev, rewrite /portal.html/<anything> → /portal.html so BrowserRouter
// with basename="/portal.html" works without a real web server.
function portalRewritePlugin(): Plugin {
  return {
    name: 'portal-rewrite',
    configureServer(server) {
      server.middlewares.use((req, _res, next) => {
        if (req.url?.startsWith('/portal.html/')) {
          req.url = '/portal.html';
        }
        next();
      });
    },
  };
}

export default defineConfig({
  base: '/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  plugins: [react(), tailwindcss(), portalRewritePlugin()],
  server: {
    proxy: {
      '/v1': 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: 'index.html',
        portal: 'portal.html',
      },
    },
  },
});
