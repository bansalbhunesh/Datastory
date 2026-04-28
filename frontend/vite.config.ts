/// <reference types="vitest" />
import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

// Vite config:
// - Dev proxy target is configurable via VITE_API_PROXY_TARGET; defaults to
//   the local Go backend (`make backend`).
// - Production: served from the Go binary via FRONTEND_DIST, so /api is
//   same-origin — no proxy needed.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const target = env.VITE_API_PROXY_TARGET || "http://127.0.0.1:8080";
  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        "/api": { target, changeOrigin: true },
        "/healthz": { target, changeOrigin: true },
      },
    },
    build: {
      target: "es2020",
      sourcemap: false,
      chunkSizeWarningLimit: 900,
    },
    test: {
      environment: "jsdom",
      globals: true,
      setupFiles: ["./src/test/setup.ts"],
      css: false,
    },
  };
});
