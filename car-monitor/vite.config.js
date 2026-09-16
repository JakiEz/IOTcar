import path from "path"
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    // App.jsx calls /api and /ws with relative paths (so the production
    // nginx proxy works with no rebuild). Mirror that proxy here so `npm run
    // dev` still talks to the local Go backend on :8080 without editing code.
    proxy: {
      "/api": { target: "http://localhost:8080" },
      "/ws": { target: "ws://localhost:8080", ws: true },
    },
  },
})
