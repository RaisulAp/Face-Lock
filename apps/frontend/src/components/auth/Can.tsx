import React from "react";
import { useAuth } from "../../lib/auth/useAuth";
import type { PermissionCheck } from "../../lib/permissions";

interface CanProps {
  permission: PermissionCheck;
  mode?: "all" | "any";
  fallback?: React.ReactNode;
  children: React.ReactNode;
}

export function Can({ permission, mode = "all", fallback = null, children }: CanProps) {
  const { can } = useAuth();

  if (!can(permission, mode)) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
}
