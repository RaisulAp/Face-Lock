import { Navigate } from "react-router-dom";
import { useAuth } from "../../../lib/auth/useAuth";

export function SmartRedirect() {
    const { can, isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="flex h-screen items-center justify-center">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent" />
            </div>
        );
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    if (
        can([
            "attendance.read_all",
            "attendance.review",
            "employee.read",
            "user.read",
            "role.read",
            "location.read",
            "settings.read",
            "audit.read",
            "face.reindex",
        ], "any")
    ) {
        return <Navigate to="/dashboard" replace />;
    }

    if (can("attendance.checkin") || can("attendance.read_self") || can("face.enroll_self")) {
        return <Navigate to="/portal/attendance" replace />;
    }

    // User without operational permissions lands on /account to manage password
    return <Navigate to="/account" replace />;
}
