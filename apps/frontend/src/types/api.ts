import type { ApiErrorCode } from "../lib/errors/codes";
export type { ApiErrorCode };

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

export type PaginatedEnvelope<T> = SuccessEnvelope<T[]>;
export type PaginatedResponse<T> = SuccessEnvelope<T[]>;

export interface ApiFieldError {
  field: string;
  message: string;
}

export interface ErrorEnvelope {
  error: {
    code: ApiErrorCode | (string & {});
    message: string;
    details?: ApiFieldError[];
    request_id: string;
    hints?: string[];
    [extra: string]: unknown;
  };
}

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

// App Settings
export interface AppSettingItem {
  key: string;
  value: string;
  description?: string;
  is_secret?: boolean;
  updated_at?: string;
  updated_by?: string;
}

// Face Quality Status (#73)
export interface QualityThresholdStatus {
  key: string;
  settings_value?: number;
  active_value?: number;
  db_value?: number;
  inference_value?: number;
  app_settings_value?: number;
  inference_active_value?: number | null;
  in_sync?: boolean;
}

export interface FaceQualityStatusResponse {
  thresholds?: QualityThresholdStatus[];
  in_sync?: boolean | null;
  checked_at?: string | null;
  inference_available?: boolean;
  inference_ready?: boolean;
  model_name?: string;
  items?: QualityThresholdStatus[];
}

// Biometric Consent
export interface BiometricConsentItem {
  id: string;
  employee_id: string;
  employee_name?: string;
  consent_document_id?: string;
  consent_version?: string;
  status: "granted" | "revoked" | "pending";
  document_reference?: string | null;
  notes?: string | null;
  granted_at?: string | null;
  revoked_at?: string | null;
  recorded_by?: string | null;
  created_at: string;
  updated_at?: string;
}

// Employee
export interface EmployeeItem {
  id: string;
  employee_number: string;
  full_name: string;
  name?: string;
  nip?: string;
  email: string;
  phone?: string | null;
  department?: string | null;
  position?: string | null;
  join_date: string;
  is_active: boolean;
  attendance_mode: "face" | "manual";
  consent_status: "none" | "active" | "outdated" | "withdrawn";
  enrollment_status: "none" | "incomplete" | "complete";
  office_location_id?: string | null;
  face_enrolled?: boolean;
  created_at: string;
  updated_at: string;
}

// Face Reference
export interface FaceReferenceItem {
  id: string;
  employee_id: string;
  photo_path: string;
  quality_score: number;
  capture_source: string;
  model_version: string;
  is_active: boolean;
  enrolled_by?: string;
  created_at: string;
}

// Office Location
export interface OfficeLocationItem {
  id: string;
  name: string;
  address?: string | null;
  lat: number;
  lng: number;
  latitude?: number;
  longitude?: number;
  radius_meter: number;
  radius_meters?: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Attendance
export type AttendanceType = "checkin" | "checkout" | "check_in" | "check_out";
export type AttendanceStatus = "approved" | "pending_review" | "pending" | "rejected";
export type VerificationMethod = "face" | "fallback" | "manual";
export type GeofenceStatus = "inside" | "outside" | "unknown";

export interface AttendanceItem {
  id: string;
  employee_id: string;
  employee_number?: string;
  employee_name?: string;
  user_id?: string | null;
  department?: string;
  type: AttendanceType;
  check_type?: string | null;
  status: AttendanceStatus;
  verification_method: VerificationMethod;
  verification_mode?: string;
  geofence_status: GeofenceStatus;
  is_within_geofence?: boolean;
  work_date: string;
  server_timestamp: string;
  recorded_at?: string;
  client_timestamp?: string;
  clock_skew_seconds?: number;
  matched_similarity?: number | null;
  similarity_score?: number | null;
  threshold_used?: number | null;
  similarity_threshold?: number | null;
  model_version?: string | null;
  quality_score?: number | null;
  quality_hints?: string[] | null;
  distance_meter?: number | null;
  distance_meters?: number | null;
  gps_accuracy_meter?: number | null;
  lat?: number | null;
  lng?: number | null;
  latitude?: number | null;
  longitude?: number | null;
  office_location_id?: string | null;
  office_location_name?: string | null;
  office_latitude?: number | null;
  office_longitude?: number | null;
  office_radius_meters?: number | null;
  photo_path?: string | null;
  fallback_reason?: string | null;
  hints?: string[];
  notes?: string | null;
  review_note?: string | null;
  reviewed_by?: string | null;
  reviewed_by_name?: string | null;
  reviewed_at?: string | null;
  waiting_hours?: number;
  created_at: string;
}

export interface AttendanceSummaryItem {
  employee: {
    id: string;
    employee_number: string;
    full_name: string;
    department?: string | null;
  };
  total_days_present?: number;
  total_checkin?: number;
  total_checkout?: number;
  total_fallback?: number;
  total_pending?: number;
  total_rejected?: number;
  total_days?: number;
  present_days?: number;
  late_days?: number;
  early_leave_days?: number;
  absence_days?: number;
  pending_review?: number;
  face_verified?: number;
  fallback_count?: number;
}

export interface AttendanceSummaryResponse {
  from?: string;
  to?: string;
  total_employees?: number;
  items?: AttendanceSummaryItem[];
  data?: AttendanceSummaryItem[];
  meta?: any;
  total_present?: number;
  valid_count?: number;
  pending_count?: number;
  rejected_count?: number;
  late_count?: number;
  by_department?: Record<string, any>;
}

export interface AttendanceAttemptItem {
  id: string | number;
  employee_id?: string | null;
  employee_number?: string | null;
  employee_name?: string | null;
  type: AttendanceType;
  work_date?: string;
  outcome?: string;
  matched_similarity?: number | null;
  threshold_used?: number | null;
  model_version?: string | null;
  quality_score?: number | null;
  hints?: string[];
  geofence_status?: GeofenceStatus;
  distance_meter?: number | null;
  allow_fallback?: boolean;
  attendance_id?: string | null;
  failure_reason?: string | null;
  request_id?: string | null;
  ip?: string | null;
  user_agent?: string | null;
  server_timestamp: string;
  created_at?: string;
}

// Users & Roles
export interface UserItem {
  id: string;
  email: string;
  employee_id?: string | null;
  employee_number?: string | null;
  full_name?: string | null;
  is_active: boolean;
  must_change_password: boolean;
  last_login_at?: string | null;
  created_at: string;
  roles: { id: string; name: string; is_system?: boolean }[];
}

export interface RoleItem {
  id: string;
  name: string;
  description?: string | null;
  is_system: boolean;
  created_at: string;
  permissions?: { id: string; name: string; resource: string; action: string; description?: string }[];
}

export interface PermissionItem {
  id: string;
  name: string;
  resource: string;
  action: string;
  description?: string;
}

export interface GroupedPermissionsResponse {
  [resource: string]: PermissionItem[];
}

// Audit Log
export interface AuditLogItem {
  id: string | number;
  actor_user_id?: string | null;
  user_id?: string | null;
  user_email?: string | null;
  action: string;
  resource_type: string;
  resource_id?: string | null;
  metadata?: Record<string, unknown> | null;
  payload?: Record<string, unknown> | null;
  ip?: string | null;
  ip_address?: string | null;
  user_agent?: string | null;
  request_id?: string | null;
  created_at: string;
}

// Consent Document
export interface ConsentDocumentItem {
  id: string;
  version: string;
  title: string;
  body: string;
  content_hash: string;
  published_at: string;
}

// Face Reindex Job
export type ReindexJobStatus = "pending" | "running" | "completed" | "failed" | "cancelled";

export interface ReindexJobItem {
  id: string;
  target_model_version?: string;
  from_model_version?: string;
  to_model_version?: string;
  status: ReindexJobStatus;
  dry_run: boolean;
  total_items?: number;
  processed_items?: number;
  succeeded_items?: number;
  failed_items?: number;
  total_count?: number;
  processed_count?: number;
  succeeded_count?: number;
  failed_count?: number;
  employees_ready_count?: number;
  employees_incomplete_count: number;
  incomplete_employees?: Array<{
    employee_id: string;
    employee_number?: string;
    succeeded: number;
    required: number;
  }>;
  error_message?: string | null;
  created_by?: string | null;
  created_at: string;
  started_at?: string | null;
  finished_at?: string | null;
}

export type Attendance = AttendanceItem;
export type AttendanceSummary = AttendanceSummaryResponse;
export type Role = RoleItem & {
  display_name?: string;
  user_count?: number;
  permission_count?: number;
};
export type Permission = PermissionItem;
export type User = Omit<UserItem, "roles"> & {
  employee_name?: string | null;
  failed_login_count?: number;
  locked_until?: string | null;
  token_version?: number;
  roles?: any;
};
export type Employee = EmployeeItem;
export type FaceProfile = FaceReferenceItem;
export type OfficeLocation = OfficeLocationItem;
export type BiometricConsent = BiometricConsentItem;
export type Consent = BiometricConsentItem;
export type SettingItem = AppSettingItem;
export type FaceReindexJob = ReindexJobItem;
