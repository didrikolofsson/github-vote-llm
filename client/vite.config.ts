import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "path";
import type { Plugin } from "vite";
import { defineConfig } from "vite";

function portalRewritePlugin(): Plugin {
  return {
    name: "portal-rewrite",
    configureServer(server) {
      server.middlewares.use((req, _res, next) => {
        if (!req.url) return next();
        const path = req.url.split(/[?#]/)[0] ?? "";
        if (
          (path === "/portal" || path.startsWith("/portal/")) &&
          path !== "/portal/index.html" &&
          !path.startsWith("/portal/src/") &&
          !path.startsWith("/portal/@")
        ) {
          const qs = req.url.includes("?")
            ? req.url.slice(req.url.indexOf("?"))
            : "";
          req.url = "/portal/index.html" + qs;
        }
        next();
      });
    },
  };
}

export default defineConfig({
  base: "/",
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  plugins: [react(), tailwindcss(), portalRewritePlugin()],
  server: {
    proxy: {
      "/v1": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: "index.html",
        portal: "portal/index.html",
      },
    },
  },
});
