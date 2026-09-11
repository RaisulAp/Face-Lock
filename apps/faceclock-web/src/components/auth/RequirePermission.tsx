import React from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../../lib/auth/useAuth";
import type { PermissionCheck } from "../../lib/permissions";

interface RequirePermissionProps {
    permission: PermissionCheck;
    mode?: "all" | "any";
    children: React.ReactNode;
}

export function RequirePermission({
    permission,
    mode = "all",
    children,
}: RequirePermissionProps) {
    const { isAuthenticated, isLoading, can } = useAuth();
    const location = useLocation();

    if (isLoading) {
        return (
            <div className="flex h-64 items-center justify-center">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent" />
            </div>
        );
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" state={{ from: location }} replace />;
    }

    if (!can(permission, mode)) {
        return <Navigate to="/403" state={{ requiredPermission: permission }} replace />;
    }

    return <>{children}</>;
}
