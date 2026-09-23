// ============================================================
// ID Locale — barrel export (all namespaces)
// ============================================================
export { common } from "./common";
export { nav } from "./nav";
export { auth } from "./auth";
export { role } from "./role";
export { user } from "./user";
export { attendance } from "./attendance";
export { employee } from "./employee";
export { location } from "./location";
export { settings } from "./settings";
export { audit } from "./audit";
export { face } from "./face";

// Re-export all types so EN can use them for `satisfies`
export type { CommonTranslations } from "./common";
export type { NavTranslations } from "./nav";
export type { AuthTranslations } from "./auth";
export type { RoleTranslations } from "./role";
export type { UserTranslations } from "./user";
export type { AttendanceTranslations } from "./attendance";
export type { EmployeeTranslations } from "./employee";
export type { LocationTranslations } from "./location";
export type { SettingsTranslations } from "./settings";
export type { AuditTranslations } from "./audit";
export type { FaceTranslations } from "./face";
