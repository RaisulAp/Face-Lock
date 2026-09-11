import { env } from "./env";
import { tokenStore } from "./auth/tokenStore";
import { requestSingleFlightRefresh } from "./auth/refreshLock";
import type { ApiErrorCode, ApiFieldError, ErrorEnvelope, SuccessEnvelope } from "../types/api";

// ApiError is what every failed call throws — never a bare Error, so
// callers can always branch on `.code` instead of parsing `.message`.
export class ApiError extends Error {
  status: number;
  code: ApiErrorCode | (string & {});
  details?: ApiFieldError[];
  requestId?: string;
  hints?: string[];
  extra?: Record<string, unknown>;

  constructor(
    status: number,
    code: ApiErrorCode | (string & {}),
    message: string,
    details?: ApiFieldError[],
    requestId?: string,
    hints?: string[],
    extra?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
    this.hints = hints;
    this.extra = extra;
  }
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  retryOn401?: boolean;
}

async function executeFetch(path: string, options: RequestOptions = {}): Promise<Response> {
  const { body, headers, method = "GET", ...rest } = options;
  const token = tokenStore.getAccessToken();

  const isMutating = ["POST", "PUT", "PATCH", "DELETE"].includes(method.toUpperCase());

  const fullUrl = path.startsWith("http") ? path : `${env.apiBaseUrl}${path}`;

  return fetch(fullUrl, {
    ...rest,
    method,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(isMutating ? { "X-Requested-With": "XMLHttpRequest" } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
}

async function requestRaw<T>(
  path: string,
  options: RequestOptions = {},
): Promise<SuccessEnvelope<T>> {
  let response = await executeFetch(path, options);

  // If 401 and refresh is possible (and not an auth endpoint), retry once with fresh token
  const isAuthEndpoint = path.includes("/auth/login") || path.includes("/auth/refresh");
  if (response.status === 401 && !isAuthEndpoint && options.retryOn401 !== false) {
    const newToken = await requestSingleFlightRefresh();
    if (newToken) {
      response = await executeFetch(path, { ...options, retryOn401: false });
    }
  }

  const requestId = response.headers.get("X-Request-Id") ?? undefined;

  let json: unknown;
  try {
    json = await response.json();
  } catch {
    if (response.status === 204) {
      return { data: undefined as unknown as T };
    }
    throw new ApiError(
      response.status,
      "INTERNAL_ERROR",
      "Response tidak valid dari server",
      undefined,
      requestId,
    );
  }

  if (!response.ok) {
    const envelope = json as ErrorEnvelope;
    const err = envelope.error;
    throw new ApiError(
      response.status,
      err?.code ?? "INTERNAL_ERROR",
      err?.message ?? "Terjadi kesalahan",
      err?.details,
      err?.request_id ?? requestId,
      err?.hints,
      err,
    );
  }

  return json as SuccessEnvelope<T>;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const envelope = await requestRaw<T>(path, options);
  return envelope.data;
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) => request<T>(path, { ...options, method: "GET" }),
  getWithMeta: <T>(path: string, options?: RequestOptions) =>
    requestRaw<T>(path, { ...options, method: "GET" }),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "POST", body }),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "PATCH", body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "PUT", body }),
  delete: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "DELETE" }),

  // For binary downloads like CSV export
  getBlob: async (path: string): Promise<{ blob: Blob; filename?: string }> => {
    let token = tokenStore.getAccessToken();
    let response = await fetch(`${env.apiBaseUrl}${path}`, {
      method: "GET",
      credentials: "include",
      headers: {
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });

    if (response.status === 401) {
      token = await requestSingleFlightRefresh();
      if (token) {
        response = await fetch(`${env.apiBaseUrl}${path}`, {
          method: "GET",
          credentials: "include",
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
      }
    }

    if (!response.ok) {
      let errCode = "INTERNAL_ERROR";
      let errMsg = "Gagal mengunduh berkas";
      try {
        const json = await response.json();
        if (json?.error) {
          errCode = json.error.code ?? errCode;
          errMsg = json.error.message ?? errMsg;
        }
      } catch {
        // Not JSON
      }
      throw new ApiError(response.status, errCode, errMsg);
    }

    const disposition = response.headers.get("Content-Disposition");
    let filename: string | undefined;
    if (disposition && disposition.includes("filename=")) {
      const match = disposition.match(/filename="?([^";]+)"?/);
      if (match) filename = match[1];
    }

    const blob = await response.blob();
    return { blob, filename };
  },
};

