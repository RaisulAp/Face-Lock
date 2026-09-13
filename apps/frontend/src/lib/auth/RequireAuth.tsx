import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "./useAuth";
import type { PermissionName } from "../permissions";
import { ForbiddenPage } from "../../features/auth/pages/ForbiddenPage";

interface RequireAuthProps {
  children: ReactNode;
}

export function RequireAuth({ children }: RequireAuthProps) {
  const { isAuthenticated, isLoading, user } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 text-gray-500">
        <div className="w-8 h-8 border-3 border-indigo-600 border-t-transparent rounded-full animate-spin mb-3" />
        <span className="text-xs">Memeriksa sesi login...</span>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Enforce password change if flagged and not already on account page
  if (user?.must_change_password && location.pathname !== "/account") {
    return <Navigate to="/account" replace />;
  }

  return <>{children}</>;
}

interface RequirePermissionProps {
  permission: PermissionName | PermissionName[];
  children: ReactNode;
}

export function RequirePermission({ permission, children }: RequirePermissionProps) {
  const { can, anyCan, user } = useAuth();

  // Super admin bypasses all permission checks
  const isSuperAdmin = user?.roles?.some((r: any) =>
    typeof r === "string" ? r === "super_admin" : r.name === "super_admin",
  );

  if (isSuperAdmin) {
    return <>{children}</>;
  }

  const hasAccess = Array.isArray(permission) ? anyCan(permission) : can(permission);

  if (!hasAccess) {
    return <ForbiddenPage />;
  }

  return <>{children}</>;
}
