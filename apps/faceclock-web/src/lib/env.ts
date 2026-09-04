// Single place that reads import.meta.env, so the rest of the app never
// touches Vite's env typing directly.
export const env = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080",
} as const;
