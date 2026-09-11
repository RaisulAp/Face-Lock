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

    if (can("attendance.read_all") || can("employee.read")) {
        return <Navigate to="/dashboard" replace />;
    }

    if (can("attendance.checkin")) {
        return <Navigate to="/me/attendance" replace />;
    }

    // User without operational permissions lands on /account to manage password
    return <Navigate to="/account" replace />;
}
