// Permission catalog and evaluator.
// Rule: Code outside of this file must NEVER compare role names (e.g. "admin", "super_admin").
// All authorization in the frontend is strictly permission-based.

export type Permission =
    // User management
    | "user.create"
    | "user.read"
    | "user.update"
    | "user.delete"
    | "user.reset_password"
    | "user.assign_role"

    // Role management
    | "role.create"
    | "role.read"
    | "role.update"
    | "role.delete"
    | "role.assign_permission"

    // Permission management
    | "permission.read"

    // Employee management
    | "employee.create"
    | "employee.read"
    | "employee.read_self"
    | "employee.update"
    | "employee.delete"

    // Settings & System
    | "settings.read"
    | "settings.update"
    | "audit.read"

    // Biometrics & Face Engine
    | "face.enroll_self"
    | "face.enroll_any"
    | "face.read_self"
    | "face.read_any"
    | "face.delete_self"
    | "face.delete_any"
    | "face.reindex"

    // Attendance
    | "attendance.checkin"
    | "attendance.checkout"
    | "attendance.read_self"
    | "attendance.read_all"
    | "attendance.approve"
    | "attendance.export"

    // Office Locations & Geofence
    | "location.create"
    | "location.read"
    | "location.update"
    | "location.delete"

    // Consents & Reviews & Misc
    | "consent.create"
    | "consent.read"
    | "consent.revoke"
    | "attendance.review"
    | "attendance.override"
    | "attendance.stats"
    | "setting.read"
    | "setting.update"
    | "attempt.read"
    | (string & {});

export type PermissionName = Permission;
export type PermissionCheck = Permission | Permission[];

export function checkPermission(
    userPermissions: string[] | undefined,
    required: PermissionCheck,
    mode: "all" | "any" = "all",
): boolean {
    if (!userPermissions || userPermissions.length === 0) {
        return false;
    }

    const list = Array.isArray(required) ? required : [required];
    if (list.length === 0) return true;

    if (mode === "any") {
        return list.some((p) => userPermissions.includes(p));
    }
    return list.every((p) => userPermissions.includes(p));
}
