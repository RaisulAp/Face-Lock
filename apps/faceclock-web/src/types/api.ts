// Wire-level envelope types, locked in Plan/01-Fase0.md § 2.6. Every fetch
// through lib/api.ts unwraps one of these two shapes — no endpoint is
// allowed to return anything else.

export interface SuccessEnvelope<T> {
  data: T;
  meta?: PaginationMeta;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface ApiFieldError {
  field: string;
  message: string;
}

export interface ErrorEnvelope {
  error: {
    code: string;
    message: string;
    details?: ApiFieldError[];
    request_id: string;
    [extra: string]: unknown;
  };
}

// The 13 codes locked in Fase 0 § 2.6. Later fases only ADD to this union
// via a reported contract revision (Plan/09-Revisions-Log.md) — see
// Fase 5 § 2.3, which requires a Record<ApiErrorCode, ...> mapping so an
// unmapped code fails the TypeScript build instead of rendering "undefined".
export type ApiErrorCode =
  | "BAD_REQUEST"
  | "UNAUTHENTICATED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "PAYLOAD_TOO_LARGE"
  | "UNSUPPORTED_MEDIA_TYPE"
  | "VALIDATION_ERROR"
  | "RATE_LIMITED"
  | "INTERNAL_ERROR"
  | "UPSTREAM_ERROR"
  | "SERVICE_UNAVAILABLE"
  | "UPSTREAM_TIMEOUT";

export interface ReadyzCheck {
  status: "ok" | "fail";
  latency_ms?: number;
  error?: string;
}

export interface ReadyzData {
  status: "ok" | "degraded";
  checks: {
    database: ReadyzCheck;
    inference: ReadyzCheck;
  };
}

export interface VersionData {
  version: string;
  commit: string;
  build_time: string;
  min_supported_client: Record<string, string>;
}
