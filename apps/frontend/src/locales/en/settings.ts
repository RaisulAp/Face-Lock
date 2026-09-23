import type { SettingsTranslations } from "../id/settings";

export const settings = {
    page: {
        title: "System Settings",
        subtitle: "Configure global application parameters.",
        loading: "Loading settings...",
        saveAll: "Save All Changes",
        saved: "Settings saved successfully.",
        error: "Failed to save settings.",
    },
    sections: {
        general: "General",
        attendance: "Attendance",
        face: "Face Recognition",
        notification: "Notifications",
        security: "Security",
    },
    general: {
        companyName: "Company Name",
        companyNamePlaceholder: "e.g. Acme Corporation",
        timezone: "Timezone",
        dateFormat: "Date Format",
        language: "System Language",
    },
    attendance: {
        workStartTime: "Work Start Time",
        workEndTime: "Work End Time",
        lateThreshold: "Late Threshold (minutes)",
        earlyLeaveThreshold: "Early Leave Threshold (minutes)",
        requireApproval: "Require Admin Approval",
        allowRemote: "Allow Remote Attendance",
    },
    face: {
        similarityThreshold: "Face Similarity Threshold",
        qualityThreshold: "Photo Quality Threshold",
        livenessCheck: "Liveness Check",
        maxRetries: "Maximum Retries",
    },
    security: {
        maxLoginAttempts: "Max Login Attempts",
        lockoutDuration: "Lockout Duration (minutes)",
        sessionTimeout: "Session Timeout (hours)",
        requireMfa: "Require MFA",
    },
} satisfies SettingsTranslations;
