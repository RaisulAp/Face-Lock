import { api } from "../../lib/api";
import type { PaginatedEnvelope } from "../../types/api";
import type {
    AttendanceActionType,
    AttendanceContextData,
    ClockSubmitPayload,
    ClockSubmitResult,
    CommitSessionResult,
    ConsentDocumentData,
    ConsentMeData,
    EmployeeAttendanceItem,
    EnrollmentSessionData,
    EnrollmentStatusMeData,
    UploadEnrollmentPhotoResult,
    WithdrawConsentResponse,
} from "./types";

export interface AttendanceHistoryParams {
    page?: number;
    per_page?: number;
    start_date?: string;
    end_date?: string;
    status?: string;
    type?: string;
}

export interface EmployeeProfileData {
    id: string;
    user_id: string;
    employee_number: string;
    name: string;
    email: string;
    department: string | null;
    position: string | null;
    phone: string | null;
    status: "active" | "inactive" | "suspended";
    created_at: string;
}

export const employeeApi = {
    // Attendance Context & Today
    getContext: () => api.get<AttendanceContextData>("/api/v1/attendances/context"),

    getToday: () =>
        api.get<{ check_in: EmployeeAttendanceItem | null; check_out: EmployeeAttendanceItem | null }>(
            "/api/v1/attendances/me/today",
        ),

    // Check-In / Check-Out
    submitClock: async (
        action: AttendanceActionType,
        payload: ClockSubmitPayload,
    ): Promise<ClockSubmitResult> => {
        const formData = new FormData();
        formData.append("photo", payload.photo, "attendance.jpg");

        if (payload.latitude !== undefined) formData.append("latitude", payload.latitude.toString());
        if (payload.longitude !== undefined) formData.append("longitude", payload.longitude.toString());
        if (payload.accuracy !== undefined) formData.append("accuracy", payload.accuracy.toString());
        formData.append("client_reported_at", new Date().toISOString());

        if (payload.note) formData.append("notes", payload.note);
        if (payload.allow_fallback) {
            formData.append("allow_fallback", "true");
            if (payload.fallback_reason) formData.append("fallback_reason", payload.fallback_reason);
            if (payload.note) formData.append("fallback_note", payload.note);
        }

        const endpoint = action === "check_in" ? "/api/v1/attendances/clock-in" : "/api/v1/attendances/clock-out";

        return api.postForm<ClockSubmitResult>(endpoint, formData, {
            headers: {
                "Idempotency-Key": payload.idempotency_key,
            },
        });
    },

    // Attendance History
    getMyHistory: (params: AttendanceHistoryParams = {}) => {
        const query = new URLSearchParams();
        if (params.page) query.set("page", params.page.toString());
        if (params.per_page) query.set("per_page", params.per_page.toString());
        if (params.start_date) query.set("start_date", params.start_date);
        if (params.end_date) query.set("end_date", params.end_date);
        if (params.status) query.set("status", params.status);
        if (params.type) query.set("type", params.type);

        const qs = query.toString();
        return api.get<PaginatedEnvelope<EmployeeAttendanceItem>>(`/api/v1/attendances/me${qs ? `?${qs}` : ""}`);
    },

    getAttendanceDetail: (id: string) => api.get<EmployeeAttendanceItem>(`/api/v1/attendances/${id}`),

    // Biometric Consent (UU PDP No. 27/2022)
    getConsentDocument: () => api.get<ConsentDocumentData>("/api/v1/consents/document"),

    getMyConsent: () => api.get<ConsentMeData>("/api/v1/consents/me"),

    grantConsent: (documentVersion: string) =>
        api.post<{ status: string; document_version: string; granted_at: string }>("/api/v1/consents", {
            document_version: documentVersion,
            agreed: true,
        }),

    withdrawConsent: (reason: string) => api.post<WithdrawConsentResponse>("/api/v1/consents/withdraw", { reason }),

    // Face Enrollment
    getMyEnrollmentStatus: () => api.get<EnrollmentStatusMeData>("/api/v1/face/enrollment-status/me"),

    createEnrollmentSession: (mode: "replace" | "append" = "replace") =>
        api.post<EnrollmentSessionData>("/api/v1/face/enrollments", { mode }),

    getEnrollmentSession: (sessionId: string) =>
        api.get<EnrollmentSessionData>(`/api/v1/face/enrollments/${sessionId}`),

    uploadEnrollmentPhoto: (sessionId: string, photo: Blob, captureSource = "web_camera") => {
        const formData = new FormData();
        formData.append("image", photo, "face.jpg");
        formData.append("capture_source", captureSource);

        return api.postForm<UploadEnrollmentPhotoResult>(`/api/v1/face/enrollments/${sessionId}/photos`, formData);
    },

    deleteEnrollmentPhoto: (sessionId: string, photoId: string) =>
        api.delete<void>(`/api/v1/face/enrollments/${sessionId}/photos/${photoId}`),

    commitEnrollmentSession: (sessionId: string, forceDuplicate = false, reason?: string) =>
        api.post<CommitSessionResult>(`/api/v1/face/enrollments/${sessionId}/commit`, {
            force_duplicate: forceDuplicate,
            reason,
        }),

    cancelEnrollmentSession: (sessionId: string) => api.delete<void>(`/api/v1/face/enrollments/${sessionId}`),

    // Profile & Security
    getMyProfile: () => api.get<EmployeeProfileData>("/api/v1/employees/me"),

    changePassword: (currentPassword: string, newPassword: string) =>
        api.post<void>("/api/v1/auth/change-password", {
            current_password: currentPassword,
            new_password: newPassword,
        }),
};
