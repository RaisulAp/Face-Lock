import type { AuthTranslations } from "../id/auth";

export const auth = {
    login: {
        title: "Sign In",
        subtitle: "Use your registered email and password.",
        emailLabel: "Email",
        emailPlaceholder: "name@company.com",
        passwordLabel: "Password",
        passwordPlaceholder: "••••••••",
        submitButton: "Sign In",
        submittingButton: "Signing In...",
        forgotPassword: "Forgot password?",
        errorDefault: "Sign in failed. Please check your connection or credentials.",
    },
    account: {
        title: "Account & Sessions",
        subtitle: "Manage your password and login sessions.",
        changePassword: "Change Password",
        currentPassword: "Current Password",
        newPassword: "New Password",
        confirmNewPassword: "Confirm New Password",
        passwordMismatch: "New passwords do not match.",
        passwordChanged: "Password changed successfully.",
        activeSessions: "Active Sessions",
        revokeSession: "Revoke Session",
        revokeAllOther: "Revoke All Other Sessions",
        thisSession: "This session",
        lastActive: "Last active",
    },
    forbidden: {
        title: "Access Denied",
        subtitle: "You do not have permission to access this page.",
        backToDashboard: "Back to Dashboard",
    },
    notFound: {
        title: "Page Not Found",
        subtitle: "The page you are looking for is not available.",
        backToHome: "Back to Home",
    },
} satisfies AuthTranslations;
