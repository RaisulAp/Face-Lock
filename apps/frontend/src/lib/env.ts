// Single place that reads import.meta.env, so the rest of the app never
// touches Vite's env typing directly.
// When VITE_API_BASE_URL is empty or omitted, requests use relative paths
// so they go through Vite's same-origin reverse proxy (/api -> faceclock-api:8080).
export const env = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? "",
} as const;
