import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig, loadEnv } from "vite";

// Fase 0 § 2.7: proxy /api to the backend so relative fetches work the same
// way inside compose (service name `faceclock-api`) and on a bare host
// (`localhost`). src/lib/api.ts currently talks to VITE_API_BASE_URL
// directly over CORS; this proxy is kept available for the same-origin
// path later fases may prefer.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  return {
    plugins: [react(), tailwindcss()],
    server: {
      host: true,
      port: 5173,
      proxy: {
        "/api": {
          target: env.BACKEND_URL || process.env.BACKEND_URL || "http://localhost:8080",
          changeOrigin: true,
        },
      },
    },
  };
});
