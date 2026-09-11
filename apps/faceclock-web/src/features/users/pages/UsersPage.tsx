import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useAuth } from "../../../lib/auth/useAuth";
import { useToast } from "../../../components/ui/Toast";
import type { User, Role, PaginatedResponse } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { Modal } from "../../../components/ui/Modal";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { Pagination } from "../../../components/data/Pagination";
import { LocalTime } from "../../../components/domain/LocalTime";
import {
    Users as UsersIcon,
    UserPlus,
    Shield,
    KeyRound,
    Trash2,
    Search,
    CheckCircle2,
    XCircle,
} from "lucide-react";

export function UsersPage() {
    const { user: currentUser, can } = useAuth();
    const toast = useToast();

    const [search, setSearch] = useState("");
    const [roleFilter, setRoleFilter] = useState("");
    const [page, setPage] = useState(1);

    // Modals state
    const [createOpen, setCreateOpen] = useState(false);
    const [rolesOpen, setRolesOpen] = useState(false);
    const [resetPwdOpen, setResetPwdOpen] = useState(false);
    const [deleteOpen, setDeleteOpen] = useState(false);

    const [selectedUser, setSelectedUser] = useState<User | null>(null);

    // Form states
    const [createEmail, setCreateEmail] = useState("");
    const [createPassword, setCreatePassword] = useState("");
    const [createSelectedRoleIds, setCreateSelectedRoleIds] = useState<string[]>([]);
    const [isSubmitting, setIsSubmitting] = useState(false);

    // Assign roles state
    const [userRoleIds, setUserRoleIds] = useState<string[]>([]);

    // Reset password state
    const [newPassword, setNewPassword] = useState("");

    // Fetch Users
    const {
        data: usersRes,
        isLoading,
        refetch,
    } = useQuery<PaginatedResponse<User>>({
        queryKey: ["users", page, search, roleFilter],
        queryFn: () => {
            const p = new URLSearchParams();
            p.set("page", String(page));
            p.set("per_page", "15");
            if (search) p.set("search", search);
            if (roleFilter) p.set("role", roleFilter);
            return api.get<PaginatedResponse<User>>(`/api/v1/users?${p.toString()}`);
        },
    });

    // Fetch Roles for assignment checklists
    const { data: rolesList } = useQuery<Role[]>({
        queryKey: ["roles"],
        queryFn: () => api.get<Role[]>("/api/v1/roles"),
    });

    // Handle Create User
    const handleCreateUser = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!createEmail.trim()) return;

        setIsSubmitting(true);
        try {
            await api.post("/api/v1/users", {
                email: createEmail.trim(),
                password: createPassword || undefined,
                role_ids: createSelectedRoleIds,
                is_active: true,
            });

            toast.show({
                type: "success",
                title: "User Dibuat",
                message: `Akun untuk ${createEmail} berhasil didaftarkan.`,
            });

            setCreateOpen(false);
            setCreateEmail("");
            setCreatePassword("");
            setCreateSelectedRoleIds([]);
            refetch();
        } catch {
            toast.show({
                type: "error",
                title: "Gagal Membuat User",
                message: "Periksa kelengkapan input dan pastikan email belum terdaftar.",
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    // Open Roles Modal
    const openRolesModal = (u: User) => {
        setSelectedUser(u);
        // Find role IDs for existing user role names
        const currentRoleIds = (rolesList || [])
            .filter((r) => u.roles?.includes(r.name))
            .map((r) => r.id);
        setUserRoleIds(currentRoleIds);
        setRolesOpen(true);
    };

    // Save Role Replacement (PUT /users/{id}/roles is REPLACE)
    const handleSaveRoles = async () => {
        if (!selectedUser) return;
        setIsSubmitting(true);
        try {
            await api.put(`/api/v1/users/${selectedUser.id}/roles`, {
                role_ids: userRoleIds,
            });

            toast.show({
                type: "success",
                title: "Peran Diperbarui",
                message: `Peran untuk user ${selectedUser.email} telah diganti.`,
            });

            setRolesOpen(false);
            refetch();
        } catch {
            toast.show({
                type: "error",
                title: "Gagal Mengubah Peran",
                message: "Operasi ditolak. Anda tidak dapat menetapkan hak akses di atas izin Anda sendiri.",
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    // Handle Reset Password
    const handleResetPassword = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!selectedUser) return;
        setIsSubmitting(true);
        try {
            await api.post(`/api/v1/users/${selectedUser.id}/reset-password`, {
                password: newPassword || undefined,
            });

            toast.show({
                type: "success",
                title: "Kata Sandi Direset",
                message: `Kata sandi untuk ${selectedUser.email} berhasil diperbarui.`,
            });

            setResetPwdOpen(false);
            setNewPassword("");
        } catch {
            toast.show({
                type: "error",
                title: "Gagal Reset Kata Sandi",
                message: "Gagal mereset kata sandi pengguna.",
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    // Handle Delete User
    const handleDeleteUser = async () => {
        if (!selectedUser) return;
        setIsSubmitting(true);
        try {
            await api.delete(`/api/v1/users/${selectedUser.id}`);
            toast.show({
                type: "success",
                title: "User Dihapus",
                message: `Akun ${selectedUser.email} telah dinonaktifkan permanen.`,
            });
            setDeleteOpen(false);
            refetch();
        } catch {
            toast.show({
                type: "error",
                title: "Gagal Menghapus User",
                message: "Tidak dapat menghapus super admin terakhir atau user sedang aktif.",
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <div className="space-y-6 max-w-7xl mx-auto">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                <div>
                    <div className="flex items-center gap-2">
                        <UsersIcon className="w-5 h-5 text-indigo-600" />
                        <h1 className="text-xl font-bold text-gray-900">Manajemen Pengguna Sistem</h1>
                    </div>
                    <p className="text-xs text-gray-500 mt-0.5">
                        Kelola akun login admin, staf HR, supervisor, dan peran akses sistem FaceClock.
                    </p>
                </div>

                {can("user.create") && (
                    <Button variant="primary" size="sm" onClick={() => setCreateOpen(true)}>
                        <UserPlus className="w-4 h-4 mr-1.5" />
                        Tambah Pengguna
                    </Button>
                )}
            </div>

            {/* Filter Card */}
            <Card>
                <CardContent className="py-3">
                    <div className="flex flex-col sm:flex-row gap-3">
                        <div className="relative flex-1">
                            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                            <Input
                                placeholder="Cari email, nama, atau NIK..."
                                value={search}
                                onChange={(e) => {
                                    setSearch(e.target.value);
                                    setPage(1);
                                }}
                                className="pl-9"
                            />
                        </div>

                        <select
                            value={roleFilter}
                            onChange={(e) => {
                                setRoleFilter(e.target.value);
                                setPage(1);
                            }}
                            className="px-3 py-2 text-xs border border-gray-300 rounded-lg bg-white text-gray-700"
                        >
                            <option value="">Semua Peran</option>
                            {(rolesList || []).map((r) => (
                                <option key={r.id} value={r.name}>
                                    {r.display_name} ({r.name})
                                </option>
                            ))}
                        </select>
                    </div>
                </CardContent>
            </Card>

            {/* Users Table */}
            <Card>
                <CardHeader>
                    <CardTitle className="text-xs">
                        Daftar Akun Pengguna ({usersRes?.meta?.total || 0})
                    </CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                    <div className="overflow-x-auto">
                        <table className="w-full text-xs text-left">
                            <thead className="bg-gray-50 text-gray-600 border-b border-gray-200">
                                <tr>
                                    <th className="px-4 py-3 font-semibold">Pengguna & Email</th>
                                    <th className="px-4 py-3 font-semibold">Karyawan Terkait</th>
                                    <th className="px-4 py-3 font-semibold">Peran (Roles)</th>
                                    <th className="px-4 py-3 font-semibold">Status</th>
                                    <th className="px-4 py-3 font-semibold">Terdaftar</th>
                                    <th className="px-4 py-3 font-semibold text-right">Aksi</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-100">
                                {isLoading ? (
                                    <tr>
                                        <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                                            Memuat daftar pengguna...
                                        </td>
                                    </tr>
                                ) : !usersRes?.data || usersRes.data.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                                            Tidak ada pengguna ditemukan.
                                        </td>
                                    </tr>
                                ) : (
                                    usersRes.data.map((u) => (
                                        <tr key={u.id} className="hover:bg-gray-50/50">
                                            <td className="px-4 py-3">
                                                <div className="font-semibold text-gray-900">{u.email}</div>
                                                {u.must_change_password && (
                                                    <span className="text-[10px] text-amber-600 font-medium">
                                                        Wajib ganti sandi
                                                    </span>
                                                )}
                                            </td>
                                            <td className="px-4 py-3 text-gray-600">
                                                {u.employee_name ? (
                                                    <div>
                                                        <span className="font-medium text-gray-900">{u.employee_name}</span>
                                                        <span className="block text-[10px] text-gray-400 font-mono">
                                                            {u.employee_number}
                                                        </span>
                                                    </div>
                                                ) : (
                                                    <span className="text-gray-400 italic">Non-Karyawan</span>
                                                )}
                                            </td>
                                            <td className="px-4 py-3">
                                                <div className="flex flex-wrap gap-1">
                                                    {u.roles && u.roles.length > 0 ? (
                                                        u.roles.map((r: any) => {
                                                            const roleStr = typeof r === "string" ? r : r.name;
                                                            return (
                                                                <Badge key={roleStr} variant="neutral" size="sm">
                                                                    {roleStr}
                                                                </Badge>
                                                            );
                                                        })
                                                    ) : (
                                                        <span className="text-gray-400 italic">Tanpa peran</span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-3">
                                                {u.is_active ? (
                                                    <span className="inline-flex items-center gap-1 text-emerald-700 font-semibold">
                                                        <CheckCircle2 className="w-3.5 h-3.5" />
                                                        Aktif
                                                    </span>
                                                ) : (
                                                    <span className="inline-flex items-center gap-1 text-rose-700 font-semibold">
                                                        <XCircle className="w-3.5 h-3.5" />
                                                        Nonaktif
                                                    </span>
                                                )}
                                            </td>
                                            <td className="px-4 py-3 text-gray-500">
                                                <LocalTime value={u.created_at} format="date" />
                                            </td>
                                            <td className="px-4 py-3 text-right">
                                                <div className="flex items-center justify-end gap-1">
                                                    {can("user.update") && (
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            title="Kelola Peran"
                                                            onClick={() => openRolesModal(u)}
                                                        >
                                                            <Shield className="w-3.5 h-3.5 text-indigo-600" />
                                                        </Button>
                                                    )}

                                                    {can("user.update") && (
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            title="Reset Password"
                                                            onClick={() => {
                                                                setSelectedUser(u);
                                                                setResetPwdOpen(true);
                                                            }}
                                                        >
                                                            <KeyRound className="w-3.5 h-3.5 text-amber-600" />
                                                        </Button>
                                                    )}

                                                    {can("user.delete") && (
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            title="Hapus Akun"
                                                            disabled={u.id === currentUser?.id}
                                                            onClick={() => {
                                                                setSelectedUser(u);
                                                                setDeleteOpen(true);
                                                            }}
                                                        >
                                                            <Trash2 className="w-3.5 h-3.5 text-rose-600" />
                                                        </Button>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {usersRes?.meta && usersRes.meta.total_pages > 1 && (
                        <div className="p-4 border-t border-gray-100">
                            <Pagination
                                page={page}
                                totalPages={usersRes.meta.total_pages}
                                onPageChange={setPage}
                            />
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Modal: Create User */}
            <Modal
                open={createOpen}
                onClose={() => setCreateOpen(false)}
                title="Tambah Akun Pengguna Baru"
            >
                <form onSubmit={handleCreateUser} className="space-y-4">
                    <div>
                        <label className="block text-xs font-semibold text-gray-700 mb-1">
                            Alamat Email (Login) *
                        </label>
                        <Input
                            type="email"
                            required
                            value={createEmail}
                            onChange={(e) => setCreateEmail(e.target.value)}
                            placeholder="admin@perusahaan.com"
                        />
                    </div>

                    <div>
                        <label className="block text-xs font-semibold text-gray-700 mb-1">
                            Kata Sandi Sementara (Opsional)
                        </label>
                        <Input
                            type="password"
                            value={createPassword}
                            onChange={(e) => setCreatePassword(e.target.value)}
                            placeholder="Kosongkan untuk kata sandi acak otomatis"
                        />
                        <p className="text-[10px] text-gray-400 mt-1">
                            Jika dikosongkan, sistem akan meng-generate password awal dan user wajib menggantinya saat login.
                        </p>
                    </div>

                    <div>
                        <label className="block text-xs font-semibold text-gray-700 mb-1">
                            Tetapkan Peran (Roles)
                        </label>
                        <div className="space-y-1.5 max-h-40 overflow-y-auto border border-gray-200 rounded-lg p-2.5 bg-gray-50">
                            {(rolesList || []).map((r) => (
                                <label key={r.id} className="flex items-center gap-2 text-xs text-gray-700">
                                    <input
                                        type="checkbox"
                                        checked={createSelectedRoleIds.includes(r.id)}
                                        onChange={(e) => {
                                            if (e.target.checked) {
                                                setCreateSelectedRoleIds([...createSelectedRoleIds, r.id]);
                                            } else {
                                                setCreateSelectedRoleIds(
                                                    createSelectedRoleIds.filter((id) => id !== r.id)
                                                );
                                            }
                                        }}
                                        className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                                    />
                                    <span>
                                        <strong className="font-semibold">{r.display_name}</strong> ({r.name})
                                    </span>
                                </label>
                            ))}
                        </div>
                    </div>

                    <div className="flex justify-end gap-2 pt-2 border-t border-gray-100">
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => setCreateOpen(false)}
                        >
                            Batal
                        </Button>
                        <Button
                            type="submit"
                            variant="primary"
                            size="sm"
                            isLoading={isSubmitting}
                        >
                            Simpan Pengguna
                        </Button>
                    </div>
                </form>
            </Modal>

            {/* Modal: Replace Roles */}
            <Modal
                open={rolesOpen}
                onClose={() => setRolesOpen(false)}
                title={`Tetapkan Peran: ${selectedUser?.email}`}
            >
                <div className="space-y-4">
                    <p className="text-xs text-gray-500">
                        Pilih peran yang ingin diberikan kepada pengguna ini. Seluruh daftar peran lama akan
                        digantikan sesuai pilihan di bawah (Replace).
                    </p>

                    <div className="space-y-2 border border-gray-200 rounded-lg p-3 bg-gray-50">
                        {(rolesList || []).map((r) => (
                            <label
                                key={r.id}
                                className="flex items-start gap-2.5 text-xs text-gray-800 cursor-pointer p-1.5 rounded hover:bg-white transition"
                            >
                                <input
                                    type="checkbox"
                                    checked={userRoleIds.includes(r.id)}
                                    onChange={(e) => {
                                        if (e.target.checked) {
                                            setUserRoleIds([...userRoleIds, r.id]);
                                        } else {
                                            setUserRoleIds(userRoleIds.filter((id) => id !== r.id));
                                        }
                                    }}
                                    className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 mt-0.5"
                                />
                                <div>
                                    <div className="font-semibold text-gray-900">{r.display_name}</div>
                                    <div className="text-[11px] text-gray-500">{r.description || r.name}</div>
                                </div>
                            </label>
                        ))}
                    </div>

                    <div className="flex justify-end gap-2 pt-2 border-t border-gray-100">
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => setRolesOpen(false)}
                        >
                            Batal
                        </Button>
                        <Button
                            type="button"
                            variant="primary"
                            size="sm"
                            isLoading={isSubmitting}
                            onClick={handleSaveRoles}
                        >
                            Simpan Perubahan Peran
                        </Button>
                    </div>
                </div>
            </Modal>

            {/* Modal: Reset Password */}
            <Modal
                open={resetPwdOpen}
                onClose={() => setResetPwdOpen(false)}
                title={`Reset Kata Sandi: ${selectedUser?.email}`}
            >
                <form onSubmit={handleResetPassword} className="space-y-4">
                    <div>
                        <label className="block text-xs font-semibold text-gray-700 mb-1">
                            Kata Sandi Baru (Opsional)
                        </label>
                        <Input
                            type="password"
                            value={newPassword}
                            onChange={(e) => setNewPassword(e.target.value)}
                            placeholder="Minimal 10 karakter..."
                        />
                        <p className="text-[10px] text-gray-400 mt-1">
                            Kosongkan jika ingin sistem men-generate kata sandi sementara secara otomatis.
                        </p>
                    </div>

                    <div className="flex justify-end gap-2 pt-2 border-t border-gray-100">
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => setResetPwdOpen(false)}
                        >
                            Batal
                        </Button>
                        <Button
                            type="submit"
                            variant="primary"
                            size="sm"
                            isLoading={isSubmitting}
                        >
                            Simpan Kata Sandi
                        </Button>
                    </div>
                </form>
            </Modal>

            {/* Modal: Delete User Confirmation */}
            <ConfirmDialog
                open={deleteOpen}
                title="Hapus Akun Pengguna"
                description={`Apakah Anda yakin ingin menghapus akun ${selectedUser?.email}? Pengguna tidak akan dapat login lagi ke sistem.`}
                confirmText="Hapus Akun"
                variant="danger"
                isLoading={isSubmitting}
                onConfirm={handleDeleteUser}
                onClose={() => setDeleteOpen(false)}
            />
        </div>
    );
}
