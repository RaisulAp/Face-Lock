import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { AppShell } from "./AppShell";
import { RequireAuth, RequirePermission } from "../lib/auth/RequireAuth";

// Auth & Shell Pages
import { LoginPage } from "../features/auth/pages/LoginPage";
import { AccountPage } from "../features/auth/pages/AccountPage";
import { SmartRedirect } from "../features/auth/pages/SmartRedirect";
import { ForbiddenPage } from "../features/auth/pages/ForbiddenPage";
import { NotFoundPage } from "../features/auth/pages/NotFoundPage";
import { EmployeePlaceholderPage } from "../features/auth/pages/EmployeePlaceholderPage";

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
    element: (
      <RequireAuth>
        <EmployeePlaceholderPage />
      </RequireAuth>
    ),
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
          <RequirePermission permission={["attendance.read_all", "attendance.review", "employee.read"]}>
            <DashboardPage />
          </RequirePermission>
        ),
      },
      {
        path: "/attendance/pending",
        element: (
          <RequirePermission permission="attendance.review">
            <PendingQueuePage />
          </RequirePermission>
        ),
      },
      {
        path: "/attendance/records",
        element: (
          <RequirePermission permission="attendance.read_all">
            <AttendanceListPage />
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
        path: "/attendance/reports",
        element: (
          <RequirePermission permission="attendance.export">
            <ReportsPage />
          </RequirePermission>
        ),
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
          <RequirePermission permission="consent.read">
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
        path: "/settings",
        element: (
          <RequirePermission permission="setting.read">
            <SettingsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/face/reindex",
        element: (
          <RequirePermission permission="face.reindex">
            <FaceReindexPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/users",
        element: (
          <RequirePermission permission="user.read">
            <UsersPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/roles",
        element: (
          <RequirePermission permission="role.read">
            <RolesPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/permissions",
        element: (
          <RequirePermission permission="role.read">
            <PermissionsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/attempts",
        element: (
          <RequirePermission permission="attendance.read_all">
            <AttemptsPage />
          </RequirePermission>
        ),
      },
      {
        path: "/security/audit-logs",
        element: (
          <RequirePermission permission="audit.read">
            <AuditLogsPage />
          </RequirePermission>
        ),
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
    path: "*",
    element: <NotFoundPage />,
  },
]);

export function AppRouter() {
  return <RouterProvider router={router} />;
}
