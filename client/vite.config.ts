import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/ui/',
  build: {
    outDir: '../server/web/dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: 'index.html',
        board: 'board.html',
      },
    },
  },
});
