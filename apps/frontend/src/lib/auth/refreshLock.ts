import { env } from "../env";
import { tokenStore } from "./tokenStore";
import { authBroadcast } from "./broadcast";
import type { RefreshResponse } from "../../types/auth";
import type { SuccessEnvelope } from "../../types/api";

let inTabRefreshPromise: Promise<string | null> | null = null;
let lastRefreshedAt = 0;

async function executeRefreshHttp(): Promise<string | null> {
  const currentRt = tokenStore.getRefreshToken();
  if (!currentRt) {
    tokenStore.clear();
    return null;
  }

  const response = await fetch(`${env.apiBaseUrl}/api/v1/auth/refresh`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest",
    },
    credentials: "include",
    body: JSON.stringify({ refresh_token: currentRt }),
  });

  if (!response.ok) {
    tokenStore.clear();
    authBroadcast.send({ type: "auth:logout" });
    return null;
  }

  const json = (await response.json()) as SuccessEnvelope<RefreshResponse>;
  const { access_token, refresh_token } = json.data;

  tokenStore.setAccessToken(access_token);
  tokenStore.setRefreshToken(refresh_token);
  lastRefreshedAt = Date.now();

  authBroadcast.send({ type: "auth:refreshed", timestamp: lastRefreshedAt });
  return access_token;
}

export async function requestSingleFlightRefresh(): Promise<string | null> {
  // If an in-tab refresh is already in flight, reuse its promise
  if (inTabRefreshPromise) {
    return inTabRefreshPromise;
  }

  // Web Locks API coordination across browser tabs
  if (typeof navigator !== "undefined" && "locks" in navigator && navigator.locks) {
    inTabRefreshPromise = (async () => {
      try {
        return await navigator.locks.request("faceclock-refresh", async () => {
          // Double-check: if a refresh occurred within the last 2.5 seconds, the existing token is fresh
          if (Date.now() - lastRefreshedAt < 2500 && tokenStore.getAccessToken()) {
            return tokenStore.getAccessToken();
          }
          return await executeRefreshHttp();
        });
      } finally {
        inTabRefreshPromise = null;
      }
    })();
    return inTabRefreshPromise;
  }

  // Fallback for browsers without Web Locks API
  inTabRefreshPromise = (async () => {
    try {
      return await executeRefreshHttp();
    } finally {
      inTabRefreshPromise = null;
    }
  })();
  return inTabRefreshPromise;
}
