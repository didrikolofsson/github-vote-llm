import path from 'path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  base: '/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/v1': 'http://localhost:8080',
      '/board': {
        target: 'http://localhost:8080',
        bypass: (req) => {
          const path = req.url?.split('?')[0] ?? '';
          return path.match(/^\/board\/[^/]+\/[^/]+\/?$/) ? '/board.html' : undefined;
        },
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: 'index.html',
        board: 'board.html',
      },
    },
  },
});
