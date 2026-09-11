import type { OfficeLocationTarget } from "../../lib/geo/haversine";

export type AttendanceActionType = "check_in" | "check_out";

export interface EmployeeAttendanceItem {
    id: string;
    employee_id: string;
    work_date: string;
    type: "check_in" | "check_out" | "in" | "out";
    status: "approved" | "pending_review" | "rejected";
    method: "face" | "fallback" | "manual";
    geofence_status: "inside" | "outside" | "unknown";
    recorded_at: string;
    server_timestamp: string;
    office_location_name?: string | null;
    waiting_hours?: number;
    review_note?: string | null;
    photo_path?: string | null;
}

export interface AttendanceContextData {
    server_time: string;
    work_date: string;
    today_check_in: EmployeeAttendanceItem | null;
    today_check_out: EmployeeAttendanceItem | null;
    can_check_in: boolean;
    can_check_out: boolean;
    has_enrolled_face: boolean;
    work_start_time?: string;
    work_end_time?: string;
    late_tolerance_minutes?: number;
    early_leave_minutes?: number;
    active_offices: OfficeLocationTarget[];
}

export interface ClockSubmitPayload {
    photo: Blob;
    type: AttendanceActionType;
    latitude?: number;
    longitude?: number;
    accuracy?: number;
    note?: string;
    allow_fallback?: boolean;
    fallback_reason?: string;
    idempotency_key: string;
}

export interface ClockSubmitResult {
    status: "approved" | "pending_review" | "rejected";
    method: "face" | "fallback" | "manual";
    geofence_status: "inside" | "outside" | "unknown";
    recorded_at: string;
    hints?: string[];
    waiting_hours?: number;
    allow_fallback?: boolean;
    attendance?: EmployeeAttendanceItem;
    message?: string;
}

// PDP Biometric Consent
export interface ConsentDocumentData {
    version: string;
    title: string;
    body: string;
    content_hash: string;
    published_at: string | null;
}

export interface ConsentMeData {
    status: "granted" | "withdrawn" | "none" | "outdated";
    document_version: string;
    granted_at: string | null;
    is_current_version: boolean;
    method: string;
}

export interface WithdrawConsentResponse {
    status: string;
    withdrawn_at: string;
    deactivated_reference_count: number;
    attendance_mode_hint: string;
}

// Enrollment Session
export interface EnrollmentStatusMeData {
    consent: {
        status: string;
        document_version: string;
        is_current_version: boolean;
    };
    attendance_mode: string;
    is_enrolled: boolean;
    active_reference_count: number;
    required_photos: number;
    max_photos: number;
    model_version_matches: boolean;
    needs_re_enrollment: boolean;
    draft_session_id: string | null;
}

export interface SessionPhotoItem {
    id: string;
    position: number;
    quality_score: number;
    det_score: number;
    photo_bytes: number;
    capture_source: string;
    created_at: string;
}

export interface EnrollmentSessionData {
    id: string;
    employee_id: string;
    status: "draft" | "committed" | "cancelled" | "expired";
    mode: "replace" | "append";
    required_photos: number;
    max_photos: number;
    model_version: string;
    photos: SessionPhotoItem[];
    expires_at: string;
}

export interface UploadEnrollmentPhotoResult {
    id: string;
    position: number;
    quality_score: number;
    det_score: number;
    hints: string[];
    photo_bytes: number;
    accepted_count: number;
    required_photos: number;
    can_commit: boolean;
}

export interface CommitSessionResult {
    session_id: string;
    employee_id: string;
    committed_at: string;
    model_version: string;
    created_reference_ids: string[];
    deactivated_reference_ids: string[];
    active_reference_count: number;
    mean_quality_score: number;
}
