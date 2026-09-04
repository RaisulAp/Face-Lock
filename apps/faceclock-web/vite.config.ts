import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// Fase 0 § 2.7: proxy /api to the backend so relative fetches work the same
// way inside compose (service name `faceclock-api`) and on a bare host
// (`localhost`). src/lib/api.ts currently talks to VITE_API_BASE_URL
// directly over CORS; this proxy is kept available for the same-origin
// path later fases may prefer.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: true,
    port: 5173,
    proxy: {
      "/api": {
        target: process.env.VITE_API_BASE_URL ?? "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
