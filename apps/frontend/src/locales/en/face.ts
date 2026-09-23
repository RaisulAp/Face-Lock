import type { FaceTranslations } from "../id/face";

export const face = {
    reindex: {
        title: "Reindex Face Model",
        subtitle: "This process will update the face index from all registered biometric data.",
        description: "Use this feature if face data is inaccurate or after adding a large number of new employees.",
        startButton: "Start Reindex",
        running: "Reindex in Progress...",
        success: "Reindex completed successfully.",
        error: "Reindex failed. Please try again.",
        lastRun: "Last run",
        status: {
            idle: "Ready",
            running: "Running",
            completed: "Completed",
            failed: "Failed",
        },
        stats: {
            totalFaces: "Total Enrolled Faces",
            indexed: "Successfully Indexed",
            failed: "Failed",
            duration: "Duration",
        },
    },
    consent: {
        page: {
            title: "Biometric Consent",
            subtitle: "Manage employee biometric data usage consent.",
            loading: "Loading consent data...",
            noData: "No consent records found.",
        },
        status: {
            given: "Consent Given",
            pending: "Pending",
            revoked: "Revoked",
        },
        actions: {
            giveConsent: "Give Consent",
            revokeConsent: "Revoke Consent",
        },
        messages: {
            giveSuccess: "Consent given successfully.",
            revokeSuccess: "Consent revoked successfully.",
            error: "Failed to process consent.",
        },
    },
} satisfies FaceTranslations;
