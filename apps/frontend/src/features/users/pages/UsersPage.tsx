import { useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useAuth } from "../../../lib/auth/useAuth";
import { useToast } from "../../../components/ui/Toast";
import type { User, Role, SuccessEnvelope } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { Pagination } from "../../../components/data/Pagination";
import { LocalTime } from "../../../components/domain/LocalTime";

// Icons
import {
  Users as UsersIcon,
  UserPlus,
  Shield,
  KeyRound,
  Trash2,
  Search,
  CheckCircle2,
  XCircle,
  Eye,
  Edit2,
  RotateCcw,
  UserCheck,
  Building2,
  HelpCircle,
} from "lucide-react";

// Subcomponents
import { UserOnboardingGuide } from "../components/UserOnboardingGuide";
import { UserFormModal } from "../components/UserFormModal";
import { RegisterEmployeeModal } from "../components/RegisterEmployeeModal";
import { UserRolesModal } from "../components/UserRolesModal";
import { ResetPasswordModal } from "../components/ResetPasswordModal";
import { UserCredentialsModal } from "../components/UserCredentialsModal";
import { UserDetailModal } from "../components/UserDetailModal";

export function UsersPage() {
  const { user: currentUser, can } = useAuth();
  const { t } = useTranslation(["user", "common"]);
  const isSuperAdmin =
    currentUser?.roles?.some((r: any) =>
      typeof r === "string" ? r === "super_admin" : r?.name === "super_admin",
    ) ?? false;
  const toast = useToast();

  // Filters & Pagination
  const [search, setSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "inactive">("all");
  const [typeFilter, setTypeFilter] = useState<"all" | "employee" | "standalone">("all");
  const [page, setPage] = useState(1);

  // Guide visibility toggle state
  const [showGuide, setShowGuide] = useState(() => {
    return localStorage.getItem("faceclock_hide_user_guide") !== "true";
  });

  // Modal States
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [rolesModalOpen, setRolesModalOpen] = useState(false);
  const [resetPwdModalOpen, setResetPwdModalOpen] = useState(false);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);

  // Selected User for Modals
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  // Credentials Modal state (for newly created user or reset password)
  const [credentialsModalOpen, setCredentialsModalOpen] = useState(false);
  const [credentialsData, setCredentialsData] = useState<{
    email: string;
    temporaryPassword?: string;
    actionTitle: string;
  }>({
    email: "",
    temporaryPassword: "",
    actionTitle: "",
  });

  const [isActionSubmitting, setIsActionSubmitting] = useState(false);

  // Fetch Users with Meta
  const {
    data: usersData,
    isLoading,
    refetch,
  } = useQuery<SuccessEnvelope<User[]>>({
    queryKey: ["users", page, search, roleFilter, statusFilter],
    queryFn: async () => {
      const p = new URLSearchParams();
      p.set("page", String(page));
      p.set("per_page", "15");
      if (search) p.set("search", search);
      if (roleFilter) p.set("role", roleFilter);
      if (statusFilter === "active") p.set("is_active", "true");
      if (statusFilter === "inactive") p.set("is_active", "false");

      return api.getWithMeta<User[]>(`/api/v1/users?${p.toString()}`);
    },
  });

  // Fetch Roles for filters and modal assignment
  const { data: rolesList = [] } = useQuery<Role[]>({
    queryKey: ["roles"],
    queryFn: () => api.get<Role[]>("/api/v1/roles"),
  });

  const users: User[] = usersData?.data ?? [];
  const meta = usersData?.meta;

  // Filter client-side by employee type if requested
  const filteredUsers = useMemo(() => {
    if (typeFilter === "employee") {
      return users.filter((u: User) => Boolean(u.employee_id));
    }
    if (typeFilter === "standalone") {
      return users.filter((u: User) => !u.employee_id);
    }
    return users;
  }, [users, typeFilter]);

  // Summary Metrics
  const totalCount = meta?.total || users.length;
  const activeCount = users.filter((u) => u.is_active).length;
  const employeeLinkedCount = users.filter((u) => Boolean(u.employee_id)).length;
  const standaloneCount = users.filter((u) => !u.employee_id).length;

  // Handle Save User Roles
  const handleSaveRoles = async (roleIds: string[]) => {
    if (!selectedUser) return;
    setIsActionSubmitting(true);
    try {
      await api.put(`/api/v1/users/${selectedUser.id}/roles`, {
        role_ids: roleIds,
      });

      toast.show({
        type: "success",
        title: "Peran Berhasil Diperbarui",
        message: `Hak akses untuk ${selectedUser.email} telah disesuaikan.`,
      });
      refetch();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal mengubah peran pengguna.";
      toast.show({
        type: "error",
        title: "Gagal Mengubah Peran",
        message: msg,
      });
      throw err;
    } finally {
      setIsActionSubmitting(false);
    }
  };

  // Handle Reset Password Submit
  const handleResetPasswordSubmit = async (customPassword?: string) => {
    if (!selectedUser) return;
    setIsActionSubmitting(true);
    try {
      const res = await api.post<{ data: { temporary_password?: string } } | { temporary_password?: string }>(
        `/api/v1/users/${selectedUser.id}/reset-password`,
        {
          password: customPassword || undefined,
        }
      );

      const returnedTempPwd =
        (res as any)?.data?.temporary_password || (res as any)?.temporary_password || customPassword;

      toast.show({
        type: "success",
        title: "Kata Sandi Berhasil Direset",
        message: `Kata sandi untuk ${selectedUser.email} telah diperbarui.`,
      });

      // Show credentials modal so admin can copy the new password
      setCredentialsData({
        email: selectedUser.email,
        temporaryPassword: returnedTempPwd,
        actionTitle: "Kata Sandi Baru Siap Digunakan",
      });
      setCredentialsModalOpen(true);
      refetch();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal mereset kata sandi pengguna.";
      toast.show({
        type: "error",
        title: "Gagal Reset Kata Sandi",
        message: msg,
      });
      throw err;
    } finally {
      setIsActionSubmitting(false);
    }
  };

  // Handle Toggle Active Status
  const handleConfirmToggleStatus = async () => {
    if (!selectedUser) return;
    setIsActionSubmitting(true);
    const newStatus = !selectedUser.is_active;

    try {
      await api.patch(`/api/v1/users/${selectedUser.id}/status`, {
        is_active: newStatus,
      });

      toast.show({
        type: "success",
        title: newStatus ? "Akun Diaktifkan" : "Akun Dinonaktifkan",
        message: `Akun ${selectedUser.email} sekarang berstatus ${newStatus ? "Aktif" : "Nonaktif"}.`,
      });

      setStatusDialogOpen(false);
      refetch();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal mengubah status akun.";
      toast.show({
        type: "error",
        title: "Gagal Mengubah Status",
        message: msg,
      });
    } finally {
      setIsActionSubmitting(false);
    }
  };

  // Handle Delete User
  const handleDeleteUser = async () => {
    if (!selectedUser) return;
    setIsActionSubmitting(true);
    try {
      await api.delete(`/api/v1/users/${selectedUser.id}`);
      toast.show({
        type: "success",
        title: "Akun Dihapus",
        message: `Akun ${selectedUser.email} telah dinonaktifkan permanen dari sistem.`,
      });
      setDeleteDialogOpen(false);
      refetch();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal menghapus akun pengguna.";
      toast.show({
        type: "error",
        title: "Gagal Menghapus Akun",
        message: msg,
      });
    } finally {
      setIsActionSubmitting(false);
    }
  };

  // Action handlers from table
  const openDetail = (u: User) => {
    setSelectedUser(u);
    setDetailModalOpen(true);
  };

  const openEdit = (u: User) => {
    setSelectedUser(u);
    setEditModalOpen(true);
  };

  const openRoles = (u: User) => {
    setSelectedUser(u);
    setRolesModalOpen(true);
  };

  const openResetPassword = (u: User) => {
    setSelectedUser(u);
    setResetPwdModalOpen(true);
  };

  const promptToggleStatus = (u: User) => {
    setSelectedUser(u);
    setStatusDialogOpen(true);
  };

  const promptDelete = (u: User) => {
    setSelectedUser(u);
    setDeleteDialogOpen(true);
  };

  const handleResetFilters = () => {
    setSearch("");
    setRoleFilter("");
    setStatusFilter("all");
    setTypeFilter("all");
    setPage(1);
  };

  const hasActiveFilters =
    Boolean(search) || Boolean(roleFilter) || statusFilter !== "all" || typeFilter !== "all";

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-indigo-50 border border-indigo-100 text-indigo-700">
              <UsersIcon className="w-5 h-5" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-gray-900">{t("page.title", { ns: "user" })}</h1>
              <p className="text-xs text-gray-500 mt-0.5">
                {t("page.subtitle", { ns: "user" })}
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowGuide(!showGuide)}
            className="text-xs text-gray-700"
          >
            <HelpCircle className="w-3.5 h-3.5 mr-1 text-indigo-600" />
            {showGuide ? t("onboarding.hideGuide", { ns: "user" }) : t("onboarding.showGuide", { ns: "user" })}
          </Button>

          {can("user.create") && (
            <Button
              variant="primary"
              size="sm"
              onClick={() => setCreateModalOpen(true)}
              className="text-xs shadow-xs"
            >
              <UserPlus className="w-4 h-4 mr-1.5" />
              {t("page.createUser", { ns: "user" })}
            </Button>
          )}
        </div>
      </div>

      {/* Onboarding Guide Banner */}
      {showGuide && (
        <UserOnboardingGuide
          onOpenCreateModal={() => setCreateModalOpen(true)}
          canCreateUser={can("user.create")}
        />
      )}

      {/* Summary Metrics */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <div className="p-4 bg-white rounded-xl border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Total Pengguna</span>
            <div className="p-1.5 bg-indigo-50 text-indigo-600 rounded-lg">
              <UsersIcon className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-gray-900 mt-2">{totalCount}</div>
          <p className="text-[11px] text-gray-400 mt-0.5">Akun terdaftar di database</p>
        </div>

        <div className="p-4 bg-white rounded-xl border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Akun Aktif</span>
            <div className="p-1.5 bg-emerald-50 text-emerald-600 rounded-lg">
              <UserCheck className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-emerald-700 mt-2">{activeCount}</div>
          <p className="text-[11px] text-emerald-600/80 mt-0.5">Dapat login ke aplikasi</p>
        </div>

        <div className="p-4 bg-white rounded-xl border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Taut Karyawan</span>
            <div className="p-1.5 bg-blue-50 text-blue-600 rounded-lg">
              <Building2 className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-blue-700 mt-2">{employeeLinkedCount}</div>
          <p className="text-[11px] text-blue-600/80 mt-0.5">Terhubung profil pegawai</p>
        </div>

        <div className="p-4 bg-white rounded-xl border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Akun Khusus</span>
            <div className="p-1.5 bg-purple-50 text-purple-600 rounded-lg">
              <Shield className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-purple-700 mt-2">{standaloneCount}</div>
          <p className="text-[11px] text-purple-600/80 mt-0.5">Admin / Non-Karyawan</p>
        </div>
      </div>

      {/* Filter Toolbar */}
      <Card>
        <CardContent className="py-3">
          <div className="flex flex-col lg:flex-row items-stretch lg:items-center gap-3">
            {/* Search */}
            <div className="relative flex-1">
              <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <Input
                placeholder="Cari email pengguna, nama karyawan, atau NIK..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(1);
                }}
                className="pl-9 text-xs"
              />
            </div>

            {/* Role Filter */}
            <div className="w-full lg:w-48">
              <select
                value={roleFilter}
                onChange={(e) => {
                  setRoleFilter(e.target.value);
                  setPage(1);
                }}
                className="w-full px-3 py-2 text-xs border border-gray-300 rounded-lg bg-white text-gray-700 focus:outline-hidden focus:ring-2 focus:ring-indigo-500"
              >
                <option value="">Semua Peran</option>
                {rolesList.map((r) => (
                  <option key={r.id} value={r.name}>
                    {r.display_name || r.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Status Filter */}
            <div className="w-full lg:w-40">
              <select
                value={statusFilter}
                onChange={(e) => {
                  setStatusFilter(e.target.value as any);
                  setPage(1);
                }}
                className="w-full px-3 py-2 text-xs border border-gray-300 rounded-lg bg-white text-gray-700 focus:outline-hidden focus:ring-2 focus:ring-indigo-500"
              >
                <option value="all">Semua Status</option>
                <option value="active">Hanya Aktif</option>
                <option value="inactive">Hanya Nonaktif</option>
              </select>
            </div>

            {/* Employee Type Filter */}
            <div className="w-full lg:w-44">
              <select
                value={typeFilter}
                onChange={(e) => {
                  setTypeFilter(e.target.value as any);
                  setPage(1);
                }}
                className="w-full px-3 py-2 text-xs border border-gray-300 rounded-lg bg-white text-gray-700 focus:outline-hidden focus:ring-2 focus:ring-indigo-500"
              >
                <option value="all">Semua Tipe Akun</option>
                <option value="employee">Taut Karyawan</option>
                <option value="standalone">Non-Karyawan</option>
              </select>
            </div>

            {/* Reset Button */}
            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleResetFilters}
                className="text-xs text-gray-500 hover:text-gray-900 shrink-0"
              >
                <RotateCcw className="w-3.5 h-3.5 mr-1" />
                Reset
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Users Table Card */}
      <Card>
        <CardHeader className="py-3.5 px-4 border-b border-gray-100 flex flex-row items-center justify-between">
          <CardTitle className="text-xs font-bold text-gray-800 flex items-center gap-2">
            <span>Daftar Akun Pengguna</span>
            <span className="px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700 font-semibold text-[11px]">
              {meta ? `${meta.total} Akun` : `${filteredUsers.length} Akun`}
            </span>
          </CardTitle>
          <div className="text-[11px] text-gray-400">
            Halaman {meta?.page || 1} dari {meta?.total_pages || 1}
          </div>
        </CardHeader>

        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-xs text-left">
              <thead className="bg-gray-50 text-gray-600 border-b border-gray-200">
                <tr>
                  <th className="px-4 py-3 font-semibold">Pengguna & Email</th>
                  <th className="px-4 py-3 font-semibold">Relasi Karyawan</th>
                  <th className="px-4 py-3 font-semibold">Peran Akses (Role)</th>
                  <th className="px-4 py-3 font-semibold">Status Akun</th>
                  <th className="px-4 py-3 font-semibold">Terdaftar</th>
                  <th className="px-4 py-3 font-semibold text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {isLoading ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-12 text-center text-gray-400">
                      <div className="flex flex-col items-center justify-center gap-2">
                        <div className="w-6 h-6 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin" />
                        <span className="text-xs">Memuat daftar pengguna...</span>
                      </div>
                    </td>
                  </tr>
                ) : filteredUsers.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-12 text-center text-gray-400">
                      <div className="max-w-sm mx-auto space-y-2">
                        <div className="w-10 h-10 rounded-full bg-gray-100 text-gray-400 mx-auto flex items-center justify-center">
                          <UsersIcon className="w-5 h-5" />
                        </div>
                        <div className="font-semibold text-gray-700 text-xs">
                          Tidak Ada Pengguna Ditemukan
                        </div>
                        <p className="text-[11px] text-gray-500">
                          {hasActiveFilters
                            ? "Coba ubah kata kunci pencarian atau sesuaikan filter Anda."
                            : "Belum ada akun pengguna yang terdaftar di sistem FaceClock."}
                        </p>
                        {hasActiveFilters && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={handleResetFilters}
                            className="mt-2 text-xs"
                          >
                            Hapus Semua Filter
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ) : (
                  filteredUsers.map((u) => {
                    const initials = u.email.slice(0, 2).toUpperCase();
                    const isCurrentUser = u.id === currentUser?.id;

                    return (
                      <tr key={u.id} className="hover:bg-gray-50/70 transition">
                        {/* Email & Initial */}
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2.5">
                            <div className="w-8 h-8 rounded-lg bg-indigo-100 text-indigo-700 font-bold text-xs flex items-center justify-center shrink-0">
                              {initials}
                            </div>
                            <div className="min-w-0">
                              <div className="font-semibold text-gray-900 font-mono flex items-center gap-1.5">
                                <span className="truncate max-w-[200px] sm:max-w-[260px]">
                                  {u.email}
                                </span>
                                {isCurrentUser && (
                                  <span className="px-1.5 py-0.5 rounded text-[9px] font-bold bg-indigo-100 text-indigo-800">
                                    Anda
                                  </span>
                                )}
                              </div>
                              <div className="flex items-center gap-2 mt-0.5">
                                {u.must_change_password && (
                                  <span className="text-[10px] text-amber-600 font-medium bg-amber-50 px-1.5 py-0.2 rounded border border-amber-200">
                                    Wajib Ganti Sandi
                                  </span>
                                )}
                                {u.failed_login_count && u.failed_login_count > 0 ? (
                                  <span className="text-[10px] text-rose-600 font-medium">
                                    {u.failed_login_count}x gagal login
                                  </span>
                                ) : null}
                              </div>
                            </div>
                          </div>
                        </td>

                        {/* Employee relation */}
                        <td className="px-4 py-3">
                          {u.employee_name ? (
                            <div className="min-w-0">
                              <div className="font-semibold text-gray-900 truncate max-w-[200px]">
                                {u.employee_name}
                              </div>
                              <div className="text-[11px] text-gray-400 font-mono">
                                NIK: {u.employee_number || "-"}
                              </div>
                            </div>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-[11px] text-gray-500 bg-gray-100 px-2 py-0.5 rounded-md">
                              <Shield className="w-3 h-3 text-gray-400" />
                              Non-Karyawan
                            </span>
                          )}
                        </td>

                        {/* Role */}
                        <td className="px-4 py-3">
                          {u.roles && u.roles.length > 0 ? (() => {
                            const firstRole = u.roles[0];
                            const roleStr = typeof firstRole === "string" ? firstRole : firstRole.name;
                            const isSuper = roleStr === "super_admin";
                            const isAdmin = roleStr === "admin";
                            const label = isSuper ? "Super Administrator" : isAdmin ? "Administrator" : roleStr === "employee" ? "Pegawai" : roleStr;

                            return (
                              <Badge
                                variant={isSuper ? "purple" : isAdmin ? "info" : "neutral"}
                                size="sm"
                              >
                                {label}
                              </Badge>
                            );
                          })() : (
                            <span className="text-[11px] text-gray-400 italic">Tanpa peran</span>
                          )}
                        </td>

                        {/* Status */}
                        <td className="px-4 py-3">
                          <button
                            type="button"
                            disabled={!can("user.update")}
                            onClick={() => promptToggleStatus(u)}
                            className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold transition ${u.is_active
                              ? "bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                              : "bg-rose-50 text-rose-700 hover:bg-rose-100"
                              } ${!can("user.update") ? "cursor-default opacity-80" : "cursor-pointer"}`}
                            title={
                              can("user.update")
                                ? "Klik untuk mengubah status akun"
                                : undefined
                            }
                          >
                            {u.is_active ? (
                              <>
                                <CheckCircle2 className="w-3 h-3 text-emerald-600" />
                                <span>Aktif</span>
                              </>
                            ) : (
                              <>
                                <XCircle className="w-3 h-3 text-rose-600" />
                                <span>Nonaktif</span>
                              </>
                            )}
                          </button>
                        </td>

                        {/* Date Registered */}
                        <td className="px-4 py-3 text-gray-500 whitespace-nowrap">
                          <LocalTime value={u.created_at} format="date" />
                        </td>

                        {/* Action buttons */}
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-1">
                            {/* View Detail */}
                            <Button
                              variant="ghost"
                              size="sm"
                              title="Lihat Rincian Akun"
                              onClick={() => openDetail(u)}
                              className="text-gray-600 hover:text-indigo-600"
                            >
                              <Eye className="w-3.5 h-3.5" />
                            </Button>

                            {/* Edit User */}
                            {can("user.update") && (
                              <Button
                                variant="ghost"
                                size="sm"
                                title="Edit Akun"
                                onClick={() => openEdit(u)}
                                className="text-gray-600 hover:text-indigo-600"
                              >
                                <Edit2 className="w-3.5 h-3.5" />
                              </Button>
                            )}

                            {/* Assign Roles */}
                            {can("user.assign_role") && (
                              <Button
                                variant="ghost"
                                size="sm"
                                title="Kelola Peran (Hak Akses)"
                                onClick={() => openRoles(u)}
                                className="text-indigo-600 hover:bg-indigo-50"
                              >
                                <Shield className="w-3.5 h-3.5" />
                              </Button>
                            )}

                            {/* Reset Password */}
                            {can("user.reset_password") && (
                              <Button
                                variant="ghost"
                                size="sm"
                                title="Reset Kata Sandi"
                                onClick={() => openResetPassword(u)}
                                className="text-amber-600 hover:bg-amber-50"
                              >
                                <KeyRound className="w-3.5 h-3.5" />
                              </Button>
                            )}

                            {/* Delete User */}
                            {can("user.delete") && (
                              <Button
                                variant="ghost"
                                size="sm"
                                title="Hapus Akun"
                                disabled={isCurrentUser}
                                onClick={() => promptDelete(u)}
                                className={`text-rose-600 hover:bg-rose-50 ${isCurrentUser ? "opacity-30 cursor-not-allowed" : ""}`}
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                              </Button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {meta && meta.total_pages > 1 && (
            <div className="p-4 border-t border-gray-100 flex items-center justify-between">
              <div className="text-xs text-gray-500">
                Menampilkan {(meta.page - 1) * meta.per_page + 1} -{" "}
                {Math.min(meta.page * meta.per_page, meta.total)} dari {meta.total} pengguna
              </div>
              <Pagination page={page} totalPages={meta.total_pages} onPageChange={setPage} />
            </div>
          )}
        </CardContent>
      </Card>

      {/* Modal: Single-step registration (creates employee + login account) */}
      <RegisterEmployeeModal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onSuccess={(result) => {
          toast.show({
            type: "success",
            title: "Karyawan Berhasil Didaftarkan",
            message: `Akun ${result.email} dan data karyawan dibuat sekaligus.`,
          });
          refetch();

          // Show the one-time credentials so the admin can hand them over.
          setCredentialsData({
            email: result.email,
            temporaryPassword: result.temporary_password,
            actionTitle: "Akun Karyawan Baru Berhasil Dibuat",
          });
          setCredentialsModalOpen(true);
        }}
      />

      {/* Modal: Edit User */}
      <UserFormModal
        open={editModalOpen}
        onClose={() => {
          setEditModalOpen(false);
          setSelectedUser(null);
        }}
        userToEdit={selectedUser}
        rolesList={rolesList}
        isSuperAdmin={isSuperAdmin}
        onSuccessCreate={() => { }}
        onSuccessUpdate={(updatedUser) => {
          toast.show({
            type: "success",
            title: "Pengguna Diperbarui",
            message: `Data akun ${updatedUser.email} berhasil disimpan.`,
          });
          refetch();
        }}
      />

      {/* Modal: Manage Roles */}
      <UserRolesModal
        open={rolesModalOpen}
        onClose={() => {
          setRolesModalOpen(false);
          setSelectedUser(null);
        }}
        user={selectedUser}
        rolesList={rolesList}
        isSuperAdmin={isSuperAdmin}
        isSubmitting={isActionSubmitting}
        onSave={handleSaveRoles}
      />

      {/* Modal: Reset Password */}
      <ResetPasswordModal
        open={resetPwdModalOpen}
        onClose={() => {
          setResetPwdModalOpen(false);
          setSelectedUser(null);
        }}
        user={selectedUser}
        isSubmitting={isActionSubmitting}
        onSubmit={handleResetPasswordSubmit}
      />

      {/* Modal: Credentials Display (with 1-click clipboard) */}
      <UserCredentialsModal
        open={credentialsModalOpen}
        onClose={() => setCredentialsModalOpen(false)}
        email={credentialsData.email}
        temporaryPassword={credentialsData.temporaryPassword}
        actionTitle={credentialsData.actionTitle}
      />

      {/* Modal: User Detail */}
      <UserDetailModal
        open={detailModalOpen}
        onClose={() => {
          setDetailModalOpen(false);
          setSelectedUser(null);
        }}
        user={selectedUser}
        rolesList={rolesList}
        canEdit={can("user.update")}
        canManageRoles={can("user.assign_role")}
        canResetPassword={can("user.reset_password")}
        onOpenEdit={(u) => openEdit(u)}
        onOpenRoles={(u) => openRoles(u)}
        onOpenResetPassword={(u) => openResetPassword(u)}
        onToggleStatus={(u) => promptToggleStatus(u)}
      />

      {/* Confirm Dialog: Toggle Status */}
      <ConfirmDialog
        open={statusDialogOpen}
        title={selectedUser?.is_active ? "Nonaktifkan Akun Pengguna?" : "Aktifkan Akun Pengguna?"}
        description={
          selectedUser?.is_active
            ? `Apakah Anda yakin ingin menonaktifkan akun ${selectedUser?.email}? Sesi login aktif pengguna akan dicabut dan pengguna tidak dapat masuk kembali sampai diaktifkan.`
            : `Apakah Anda yakin ingin mengaktifkan kembali akun ${selectedUser?.email}? Pengguna akan diizinkan login ke sistem.`
        }
        confirmText={selectedUser?.is_active ? "Nonaktifkan Akun" : "Aktifkan Akun"}
        variant={selectedUser?.is_active ? "danger" : "info"}
        isLoading={isActionSubmitting}
        onConfirm={handleConfirmToggleStatus}
        onClose={() => setStatusDialogOpen(false)}
      />

      {/* Confirm Dialog: Delete User */}
      <ConfirmDialog
        open={deleteDialogOpen}
        title="Hapus Akun Pengguna Permanen?"
        description={`Apakah Anda yakin ingin menghapus akun ${selectedUser?.email}? Tindakan ini akan menghapus akun secara permanen (soft-delete). Catatan: Sistem akan menolak jika akun ini merupakan Super Admin terakhir.`}
        confirmText="Hapus Pengguna"
        variant="danger"
        isLoading={isActionSubmitting}
        onConfirm={handleDeleteUser}
        onClose={() => setDeleteDialogOpen(false)}
      />
    </div>
  );
}

