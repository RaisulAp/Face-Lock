import { env } from "./env";
import type { ApiErrorCode, ApiFieldError, ErrorEnvelope, SuccessEnvelope } from "../types/api";

// ApiError is what every failed call throws — never a bare Error, so
// callers can always branch on `.code` instead of parsing `.message`
// (Plan/01-Fase0.md § 2.6: "message boleh dibaca manusia, code yang dipakai
// program").
export class ApiError extends Error {
  status: number;
  code: ApiErrorCode | (string & {});
  details?: ApiFieldError[];
  requestId?: string;
  extra?: Record<string, unknown>;

  constructor(
    status: number,
    code: ApiErrorCode | (string & {}),
    message: string,
    details?: ApiFieldError[],
    requestId?: string,
    extra?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
    this.extra = extra;
  }
}

// getAuthToken is a seam for Fase 1: today it always returns null, so no
// Authorization header is sent. Fase 1 replaces the body of this one
// function with a read from the in-memory token store — nothing else in
// this file changes.
function getAuthToken(): string | null {
  return null;
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, headers, ...rest } = options;
  const token = getAuthToken();

  const response = await fetch(`${env.apiBaseUrl}${path}`, {
    ...rest,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const requestId = response.headers.get("X-Request-Id") ?? undefined;

  let json: unknown;
  try {
    json = await response.json();
  } catch {
    // A response with no JSON body at all (e.g. a proxy error page) still
    // has to surface as an ApiError, not an uncaught SyntaxError.
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
      err,
    );
  }

  return (json as SuccessEnvelope<T>).data;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: "POST", body }),
  patch: <T>(path: string, body?: unknown) => request<T>(path, { method: "PATCH", body }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: "PUT", body }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
