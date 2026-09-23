import { useTranslation } from "react-i18next";
import {
    Users,
    Shield,
    KeyRound,
    CheckCircle2,
    XCircle,
    Edit2,
    ShieldCheck,
} from "lucide-react";
import type { User, Role } from "../../../types/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { LocalTime } from "../../../components/domain/LocalTime";

interface UserDetailModalProps {
    open: boolean;
    onClose: () => void;
    user: User | null;
    rolesList: Role[];
    onOpenEdit: (u: User) => void;
    onOpenRoles: (u: User) => void;
    onOpenResetPassword: (u: User) => void;
    onToggleStatus: (u: User) => void;
    canEdit: boolean;
    canManageRoles: boolean;
    canResetPassword: boolean;
}

export function UserDetailModal({
    open,
    onClose,
    user,
    rolesList,
    onOpenEdit,
    onOpenRoles,
    onOpenResetPassword,
    onToggleStatus,
    canEdit,
    canManageRoles,
    canResetPassword,
}: UserDetailModalProps) {
    const { t } = useTranslation(["user", "common"]);
    if (!user) return null;

    // Find user's role objects
    const userRoleNames = new Set(
        (user.roles || []).map((r: any) => (typeof r === "string" ? r : r.name))
    );
    const matchedRoles = rolesList.filter((r) => userRoleNames.has(r.name));

    const initials = user.email.slice(0, 2).toUpperCase();

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={t("detail.title", { ns: "user" })}
            description="Informasi lengkap identitas login, relasi karyawan, dan konfigurasi hak akses."
            maxWidth="lg"
        >
            <div className="space-y-5">
                {/* Profile Card Header */}
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 bg-gradient-to-r from-gray-50 to-indigo-50/40 rounded-xl border border-gray-200">
                    <div className="flex items-center gap-3">
                        <div className="w-12 h-12 rounded-xl bg-indigo-600 text-white flex items-center justify-center font-bold text-base shadow-xs shrink-0">
                            {initials}
                        </div>
                        <div>
                            <div className="flex items-center gap-2 flex-wrap">
                                <h3 className="font-bold text-sm sm:text-base text-gray-900 font-mono">
                                    {user.email}
                                </h3>
                                {user.is_active ? (
                                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-100 text-emerald-800">
                                        <CheckCircle2 className="w-3 h-3" />
                                        Aktif
                                    </span>
                                ) : (
                                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-rose-100 text-rose-800">
                                        <XCircle className="w-3 h-3" />
                                        Nonaktif
                                    </span>
                                )}
                            </div>
                            <div className="text-xs text-gray-500 mt-0.5">
                                ID Pengguna: <span className="font-mono text-[10px]">{user.id}</span>
                            </div>
                        </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                        {canEdit && (
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                    onClose();
                                    onOpenEdit(user);
                                }}
                                className="text-xs"
                            >
                                <Edit2 className="w-3.5 h-3.5 mr-1" />
                                Edit
                            </Button>
                        )}
                        {canEdit && (
                            <Button
                                type="button"
                                variant={user.is_active ? "outline" : "primary"}
                                size="sm"
                                onClick={() => onToggleStatus(user)}
                                className={`text-xs ${user.is_active ? "text-rose-600 hover:bg-rose-50 border-rose-200" : ""}`}
                            >
                                {user.is_active ? "Nonaktifkan" : "Aktifkan Akun"}
                            </Button>
                        )}
                    </div>
                </div>

                {/* Linked Employee Information */}
                <div className="p-4 rounded-xl border border-gray-200 bg-white space-y-2">
                    <span className="text-[11px] font-semibold text-gray-500 uppercase tracking-wider block">
                        Relasi Karyawan
                    </span>
                    {user.employee_name ? (
                        <div className="flex items-center gap-3 p-2 bg-blue-50/60 border border-blue-100 rounded-lg">
                            <div className="p-2 bg-blue-100 text-blue-700 rounded-lg">
                                <Users className="w-4 h-4" />
                            </div>
                            <div>
                                <div className="font-bold text-xs text-blue-950">
                                    {user.employee_name}
                                </div>
                                <div className="text-[11px] text-blue-800 font-mono">
                                    NIK: {user.employee_number || "Tidak ada NIK"}
                                </div>
                            </div>
                        </div>
                    ) : (
                        <div className="text-xs text-gray-500 italic p-2 bg-gray-50 rounded-lg border border-gray-100">
                            Akun Khusus / Non-Karyawan (Tidak ditautkan ke data profil pegawai mana pun).
                        </div>
                    )}
                </div>

                {/* Roles & Permissions */}
                <div className="p-4 rounded-xl border border-gray-200 bg-white space-y-3">
                    <div className="flex items-center justify-between">
                        <span className="text-[11px] font-semibold text-gray-500 uppercase tracking-wider">
                            Peran Akses Sistem
                        </span>
                        {canManageRoles && (
                            <button
                                type="button"
                                onClick={() => {
                                    onClose();
                                    onOpenRoles(user);
                                }}
                                className="text-xs font-semibold text-indigo-600 hover:text-indigo-800 flex items-center gap-1"
                            >
                                <Shield className="w-3.5 h-3.5" />
                                Ubah Peran
                            </button>
                        )}
                    </div>

                    {matchedRoles.length > 0 ? (
                        <div className="grid grid-cols-1 gap-2">
                            {matchedRoles.map((r) => (
                                <div
                                    key={r.id}
                                    className="p-3 rounded-lg border border-gray-200 bg-gray-50/50 flex items-start gap-3 text-xs"
                                >
                                    <ShieldCheck className="w-5 h-5 text-indigo-600 shrink-0 mt-0.5" />
                                    <div className="min-w-0 flex-1">
                                        <div className="flex items-center justify-between">
                                            <div className="font-semibold text-gray-900">
                                                {r.display_name || r.name}
                                            </div>
                                            <span className="text-[10px] text-gray-400 font-mono">{r.name}</span>
                                        </div>
                                        <div className="text-[11px] text-gray-500 mt-0.5">
                                            {r.description || "Peran akses sistem baku."}
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    ) : (
                        <div className="text-xs text-gray-400 italic">
                            Pengguna belum memiliki peran akses apa pun.
                        </div>
                    )}
                </div>

                {/* Security & Activity Stats */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                    <div className="p-3 bg-gray-50 rounded-xl border border-gray-200">
                        <div className="text-[10px] text-gray-500 font-semibold uppercase">Wajib Ganti Sandi</div>
                        <div className="text-xs font-bold mt-1">
                            {user.must_change_password ? (
                                <span className="text-amber-600">Ya (Saat Login)</span>
                            ) : (
                                <span className="text-gray-700">Tidak</span>
                            )}
                        </div>
                    </div>

                    <div className="p-3 bg-gray-50 rounded-xl border border-gray-200">
                        <div className="text-[10px] text-gray-500 font-semibold uppercase">Gagal Login</div>
                        <div className="text-xs font-bold text-gray-900 mt-1">
                            {user.failed_login_count ?? 0} kali
                        </div>
                    </div>

                    <div className="p-3 bg-gray-50 rounded-xl border border-gray-200">
                        <div className="text-[10px] text-gray-500 font-semibold uppercase">Versi Token</div>
                        <div className="text-xs font-bold text-gray-900 font-mono mt-1">
                            v{user.token_version ?? 1}
                        </div>
                    </div>

                    <div className="p-3 bg-gray-50 rounded-xl border border-gray-200">
                        <div className="text-[10px] text-gray-500 font-semibold uppercase">Terdaftar Pada</div>
                        <div className="text-xs font-medium text-gray-700 mt-1">
                            <LocalTime value={user.created_at} format="date" />
                        </div>
                    </div>
                </div>

                {/* Footer Actions */}
                <div className="flex items-center justify-between pt-3 border-t border-gray-100">
                    <div>
                        {canResetPassword && (
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                    onClose();
                                    onOpenResetPassword(user);
                                }}
                                className="text-xs text-amber-700 hover:bg-amber-50 border-amber-200"
                            >
                                <KeyRound className="w-3.5 h-3.5 mr-1 text-amber-600" />
                                Reset Kata Sandi
                            </Button>
                        )}
                    </div>

                    <Button type="button" variant="primary" size="sm" onClick={onClose}>
                        Tutup
                    </Button>
                </div>
            </div>
        </Modal>
    );
}
