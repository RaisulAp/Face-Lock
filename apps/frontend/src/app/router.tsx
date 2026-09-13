import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import { AppShell } from "./AppShell";
import { EmployeeShell } from "./EmployeeShell";
import { RequireAuth, RequirePermission } from "../lib/auth/RequireAuth";

// Auth & Shell Pages
import { LoginPage } from "../features/auth/pages/LoginPage";
import { AccountPage } from "../features/auth/pages/AccountPage";
import { SmartRedirect } from "../features/auth/pages/SmartRedirect";
import { ForbiddenPage } from "../features/auth/pages/ForbiddenPage";
import { NotFoundPage } from "../features/auth/pages/NotFoundPage";

// Employee Portal Pages (Fase 6)
import { AttendancePage } from "../features/employee-portal/pages/AttendancePage";
import { EnrollmentPage } from "../features/employee-portal/pages/EnrollmentPage";
import { ConsentPage } from "../features/employee-portal/pages/ConsentPage";
import { HistoryPage } from "../features/employee-portal/pages/HistoryPage";
import { HistoryDetailPage } from "../features/employee-portal/pages/HistoryDetailPage";
import { ProfilePage } from "../features/employee-portal/pages/ProfilePage";

// Features Pages
import { DashboardPage } from "../features/dashboard/pages/DashboardPage";
import { PendingQueuePage } from "../features/attendance/pages/PendingQueuePage";
import { AttendanceListPage } from "../features/attendance/pages/AttendanceListPage";
import { AttendanceDetailPage } from "../features/attendance/pages/AttendanceDetailPage";
import { ReportsPage } from "../features/reports/pages/ReportsPage";
import { EmployeesPage } from "../features/employees/pages/EmployeesPage";
import { EmployeeFormPage } from "../features/employees/pages/EmployeeFormPage";
import { EmployeeFacePage } from "../features/employees/pages/EmployeeFacePage";
import { ConsentsPage } from "../features/consents/pages/ConsentsPage";
import { LocationsPage } from "../features/locations/pages/LocationsPage";
import { SettingsPage } from "../features/settings/pages/SettingsPage";
import { FaceReindexPage } from "../features/face/pages/FaceReindexPage";
import { UsersPage } from "../features/users/pages/UsersPage";
import { RolesPage } from "../features/roles/pages/RolesPage";
import { PermissionsPage } from "../features/permissions/pages/PermissionsPage";
import { AttemptsPage } from "../features/security/pages/AttemptsPage";
import { AuditLogsPage } from "../features/audit/pages/AuditLogsPage";

const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/",
    element: (
      <RequireAuth>
        <SmartRedirect />
      </RequireAuth>
    ),
  },
  {
    path: "/employee/self",
    element: <Navigate to="/portal/attendance" replace />,
  },
  {
    path: "/me/attendance",
    element: <Navigate to="/portal/attendance" replace />,
  },
  {
    path: "/portal",
    element: (
      <RequireAuth>
        <EmployeeShell />
      </RequireAuth>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="/portal/attendance" replace />,
      },
      {
        path: "attendance",
        element: <AttendancePage />,
      },
      {
        path: "enrollment",
        element: <EnrollmentPage />,
      },
      {
        path: "consent",
        element: <ConsentPage />,
      },
      {
        path: "history",
        element: <HistoryPage />,
      },
      {
        path: "history/:id",
        element: <HistoryDetailPage />,
      },
      {
        path: "profile",
        element: <ProfilePage />,
      },
    ],
  },
  {
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      {
        path: "/dashboard",
        element: (
          <RequirePermission
            permission={[
              "attendance.read_all",
              "attendance.review",
              "employee.read",
              "user.read",
              "role.read",
              "location.read",
              "settings.read",
              "audit.read",
              "face.reindex",
            ]}
          >
            <DashboardPage />
          </RequirePermission>
        ),
      },
      {
        path: "/attendances/pending",
        element: (
          <RequirePermission permission={["attendance.review", "attendance.approve"]}>
            <PendingQueuePage />
          </RequirePermission>
        ),
      },
      {
        path: "/attendance/pending",
        element: <Navigate to="/attendances/pending" replace />,
      },
      {
        path: "/attendances",
        element: (
          <RequirePermission permission="attendance.read_all">
            <AttendanceListPage />
          </RequirePermission>
        ),
      },
      {
        path: "/attendance/records",
        element: <Navigate to="/attendances" replace />,
      },
      {
        path: "/attendance",
        element: <Navigate to="/attendances" replace />,
      },
      {
        path: "/attendances/:id",
        element: (
          <RequirePermission permission="attendance.read_all">
            <AttendanceDetailPage />
          </RequirePermission>
        ),
      },
      {
        path: "/attendance/records/:id",
        element: (
          <RequirePermission permission="attendance.read_all">
            <AttendanceDetailPage />
          </RequirePermission>
        ),
      },
      {
        path: "/reports",
        element: (
          <RequirePermission permission={["attendance.export", "attendance.read_all"]}>
            <ReportsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/attendance/reports",
        element: <Navigate to="/reports" replace />,
      },
      {
        path: "/employees",
        element: (
          <RequirePermission permission="employee.read">
            <EmployeesPage />
          </RequirePermission>
        ),
      },
      {
        path: "/employees/new",
        element: (
          <RequirePermission permission="employee.create">
            <EmployeeFormPage />
          </RequirePermission>
        ),
      },
      {
        path: "/employees/:id",
        element: (
          <RequirePermission permission="employee.read">
            <EmployeeFormPage />
          </RequirePermission>
        ),
      },
      {
        path: "/employees/:id/edit",
        element: (
          <RequirePermission permission="employee.read">
            <EmployeeFormPage />
          </RequirePermission>
        ),
      },
      {
        path: "/employees/:id/face",
        element: (
          <RequirePermission permission="employee.read">
            <EmployeeFacePage />
          </RequirePermission>
        ),
      },
      {
        path: "/consents",
        element: (
          <RequirePermission permission={["face.read_any", "consent.read"]}>
            <ConsentsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/locations",
        element: (
          <RequirePermission permission="location.read">
            <LocationsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/office-locations",
        element: <Navigate to="/locations" replace />,
      },
      {
        path: "/settings",
        element: (
          <RequirePermission permission={["settings.read", "setting.read"]}>
            <SettingsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/face/reindex",
        element: (
          <RequirePermission permission={["face.reindex", "system.reindex"]}>
            <FaceReindexPage />
          </RequirePermission>
        ),
      },
      {
        path: "/reindex",
        element: <Navigate to="/face/reindex" replace />,
      },
      {
        path: "/users",
        element: (
          <RequirePermission permission="user.read">
            <UsersPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/users",
        element: <Navigate to="/users" replace />,
      },
      {
        path: "/roles",
        element: (
          <RequirePermission permission="role.read">
            <RolesPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/roles",
        element: <Navigate to="/roles" replace />,
      },
      {
        path: "/permissions",
        element: (
          <RequirePermission permission="role.read">
            <PermissionsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/permissions",
        element: <Navigate to="/permissions" replace />,
      },
      {
        path: "/security/attempts",
        element: (
          <RequirePermission permission={["attendance.read_all", "attempt.read"]}>
            <AttemptsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/attempts",
        element: <Navigate to="/security/attempts" replace />,
      },
      {
        path: "/audit-logs",
        element: (
          <RequirePermission permission="audit.read">
            <AuditLogsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/audit/logs",
        element: <Navigate to="/audit-logs" replace />,
      },
      {
        path: "/security/audit-logs",
        element: <Navigate to="/audit-logs" replace />,
      },
      {
        path: "/account",
        element: <AccountPage />,
      },
    ],
  },
  {
    path: "/forbidden",
    element: <ForbiddenPage />,
  },
  {
    path: "/403",
    element: <Navigate to="/forbidden" replace />,
  },
  {
    path: "*",
    element: <NotFoundPage />,
  },
]);

export function AppRouter() {
  return <RouterProvider router={router} />;
}
