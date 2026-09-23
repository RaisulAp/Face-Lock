import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import {
    Shield,
    KeyRound,
    AlertCircle,
    Sparkles,
    Lock,
    UserCheck,
    Building2,
    Briefcase,
} from "lucide-react";
import type { User, Role, EmployeeItem } from "../../../types/api";
import { api } from "../../../lib/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";

/**
 * Account editing and standalone account creation.
 *
 * Note: creating an account that is linked to an employee is now handled by
 * RegisterEmployeeModal through POST /api/v1/users/register-employee, which
 * creates the employee row and the login account in one step. This modal keeps
 * only the two remaining jobs:
 *   1. editing an existing account (email, linked employee, active flag), and
 *   2. creating a non-employee "standalone" system account.
 */
interface UserFormModalProps {
    open: boolean;
    onClose: () => void;
    userToEdit?: User | null;
    rolesList: Role[];
    isSuperAdmin: boolean;
    onSuccessCreate: (created: { user: User; temporaryPassword?: string }) => void;
    onSuccessUpdate: (user: User) => void;
}

export function UserFormModal({
    open,
    onClose,
    userToEdit,
    rolesList,
    isSuperAdmin,
    onSuccessCreate,
    onSuccessUpdate,
}: UserFormModalProps) {
    const { t } = useTranslation(["user", "common"]);
    const isEdit = Boolean(userToEdit);

    // Form states
    const [selectedEmployeeId, setSelectedEmployeeId] = useState<string>("");
    const [email, setEmail] = useState("");
    const [passwordMode, setPasswordMode] = useState<"auto" | "manual">("auto");
    const [manualPassword, setManualPassword] = useState("");
    const [mustChangePassword, setMustChangePassword] = useState(true);
    const [isActive, setIsActive] = useState(true);
    const [selectedRoleId, setSelectedRoleId] = useState<string>("");
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // The employee linker is only offered in edit mode for existing employee
    // records. Standalone accounts never need the list, so it stays disabled
    // unless we are explicitly editing an employee-linked account.
    const [enableEmployeeLink, setEnableEmployeeLink] = useState(false);

    // Query all active employees, used by the edit-mode employee relinker.
    const { data: allEmployees = [], isLoading: isLoadingEmployees } = useQuery({
        queryKey: ["employees-all-active"],
        queryFn: async () => {
            const res = await api.getWithMeta<EmployeeItem[]>("/api/v1/employees?per_page=100");
            return res.data;
        },
        enabled: open && isEdit && enableEmployeeLink,
    });

    // Reset or initialize on open
    useEffect(() => {
        if (open) {
            setError(null);
            if (userToEdit) {
                setEmail(userToEdit.email);
                setSelectedEmployeeId(userToEdit.employee_id || "");
                setIsActive(userToEdit.is_active);
                setEnableEmployeeLink(false);
            } else {
                setEmail("");
                setSelectedEmployeeId("");
                setPasswordMode("auto");
                setManualPassword("");
                setMustChangePassword(true);
                setIsActive(true);
                setEnableEmployeeLink(false);

                // Standalone accounts are the only creation path left here, and
                // the employee role makes no sense for them, so prefer a
                // non-employee role when one is available.
                const fallbackRole =
                    rolesList.find((r) => r.name !== "employee" && r.name !== "super_admin") ||
                    rolesList.find((r) => r.name === "employee") ||
                    rolesList[0];
                setSelectedRoleId(fallbackRole ? fallbackRole.id : "");
            }
        }
    }, [open, userToEdit, rolesList]);

    const handleSelectRole = (roleId: string, roleName: string) => {
        if (roleName === "super_admin" && !isSuperAdmin) {
            return;
        }
        setSelectedRoleId(roleId);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        const trimmedEmail = email.trim();
        if (!trimmedEmail) {
            setError("Email wajib diisi.");
            return;
        }

        if (!isEdit && !selectedRoleId) {
            setError("Peran sistem wajib dipilih (1 akun harus memiliki tepat 1 peran akses).");
            return;
        }

        if (!isEdit && passwordMode === "manual") {
            if (!manualPassword || manualPassword.length < 10) {
                setError("Kata sandi manual harus minimal 10 karakter.");
                return;
            }
        }

        setIsSubmitting(true);
        try {
            if (isEdit && userToEdit) {
                const payload: Record<string, unknown> = {
                    email: trimmedEmail,
                    is_active: isActive,
                };
                // Only touch employee_id when the admin explicitly opted in to
                // relinking, so a plain email change never detaches an account.
                if (enableEmployeeLink) {
                    payload.employee_id = selectedEmployeeId || null;
                }

                const res = await api.patch<{ data: User }>(`/api/v1/users/${userToEdit.id}`, payload);
                const updated = (res as any).data || res;
                onSuccessUpdate(updated);
                onClose();
            } else {
                // Standalone (non-employee) system account. Employee-linked
                // accounts are created through RegisterEmployeeModal instead.
                const payload: Record<string, unknown> = {
                    email: trimmedEmail,
                    role_ids: [selectedRoleId],
                    must_change_password: mustChangePassword,
                };

                if (passwordMode === "manual" && manualPassword) {
                    payload.password = manualPassword;
                }

                const res = await api.post<any>("/api/v1/users", payload);
                const createdUser: User = res.data?.user || res.user || res;
                const tempPassword = res.data?.temporary_password || res.temporary_password;

                onSuccessCreate({
                    user: createdUser,
                    temporaryPassword: tempPassword,
                });
                onClose();
            }
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : "Gagal menyimpan akun pengguna.";
            setError(msg);
        } finally {
            setIsSubmitting(false);
        }
    };

    const selectedEmployeeObj = allEmployees.find((e) => e.id === selectedEmployeeId);

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={isEdit ? t("form.editTitle", { ns: "user" }) : t("form.createTitle", { ns: "user" })}
            description={
                isEdit
                    ? "Perbarui email atau relasi karyawan untuk akun ini."
                    : "Buat akun sistem tanpa data karyawan. Untuk mendaftarkan karyawan beserta akun login-nya, gunakan tombol Daftarkan Karyawan."
            }
            maxWidth="lg"
        >
            <form onSubmit={handleSubmit} className="space-y-4">
                {error && (
                    <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 rounded-lg text-xs flex items-center gap-2">
                        <AlertCircle className="w-4 h-4 shrink-0 text-rose-500" />
                        <span>{error}</span>
                    </div>
                )}

                {!isEdit && (
                    <div className="p-3 bg-indigo-50/50 border border-indigo-100 rounded-xl flex gap-2 text-[11px] text-indigo-900 leading-relaxed">
                        <Shield className="w-4 h-4 shrink-0 text-indigo-500 mt-0.5" />
                        <span>
                            Formulir ini hanya membuat <strong>akun sistem non-karyawan</strong> (mis. admin
                            atau auditor). Untuk karyawan, gunakan <strong>Daftarkan Karyawan</strong> agar data
                            karyawan dan akun login dibuat sekaligus.
                        </span>
                    </div>
                )}

                {/* Employee Link (edit mode only, opt-in) */}
                {isEdit && (
                    <div className="space-y-2">
                        <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
                            <input
                                type="checkbox"
                                checked={enableEmployeeLink}
                                onChange={(e) => {
                                    setEnableEmployeeLink(e.target.checked);
                                    if (!e.target.checked) setSelectedEmployeeId("");
                                }}
                                className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                            />
                            <span>Ubah relasi karyawan untuk akun ini</span>
                        </label>

                        {enableEmployeeLink && (
                            <div className="space-y-2 p-3 bg-indigo-50/40 border border-indigo-100 rounded-xl">
                                <select
                                    value={selectedEmployeeId}
                                    onChange={(e) => setSelectedEmployeeId(e.target.value)}
                                    className="w-full text-xs rounded-lg border border-gray-300 bg-white py-2 px-3 focus:outline-hidden focus:ring-2 focus:ring-indigo-500"
                                    disabled={isLoadingEmployees}
                                >
                                    <option value="">
                                        {isLoadingEmployees ? "-- Memuat data karyawan --" : "-- Lepas relasi karyawan (jadikan akun sistem) --"}
                                    </option>
                                    {allEmployees.map((emp) => (
                                        <option key={emp.id} value={emp.id}>
                                            {emp.full_name} ({emp.employee_number}) - {emp.department || "Tanpa Divisi"}
                                        </option>
                                    ))}
                                </select>

                                {selectedEmployeeObj && (
                                    <div className="p-2.5 bg-white rounded-lg border border-indigo-100 text-xs space-y-1">
                                        <div className="font-semibold text-gray-900 flex items-center gap-1.5">
                                            <UserCheck className="w-3.5 h-3.5 text-emerald-600" />
                                            {selectedEmployeeObj.full_name}
                                        </div>
                                        <div className="flex items-center gap-3 text-[11px] text-gray-500">
                                            <span>NIK: {selectedEmployeeObj.employee_number}</span>
                                            {selectedEmployeeObj.department && (
                                                <span className="flex items-center gap-1">
                                                    <Building2 className="w-3 h-3" /> {selectedEmployeeObj.department}
                                                </span>
                                            )}
                                            {selectedEmployeeObj.position && (
                                                <span className="flex items-center gap-1">
                                                    <Briefcase className="w-3 h-3" /> {selectedEmployeeObj.position}
                                                </span>
                                            )}
                                        </div>
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                )}

                {/* Email Field */}
                <div className="space-y-1.5">
                    <label className="block text-xs font-semibold text-gray-700">
                        Alamat Email Login <span className="text-rose-500">*</span>
                    </label>
                    <Input
                        type="email"
                        required
                        placeholder="contoh: budi@perusahaan.com"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        className="text-xs"
                    />
                    <p className="text-[11px] text-gray-500">
                        Email ini digunakan untuk autentikasi dan pengiriman notifikasi/kata sandi.
                    </p>
                </div>

                {/* Password Handling (Create Mode Only) */}
                {!isEdit && (
                    <div className="p-3.5 bg-gray-50 border border-gray-200 rounded-xl space-y-3">
                        <div className="flex items-center justify-between">
                            <span className="text-xs font-bold text-gray-900 flex items-center gap-1.5">
                                <KeyRound className="w-3.5 h-3.5 text-indigo-600" />
                                Konfigurasi Kata Sandi
                            </span>
                            <div className="flex items-center gap-2">
                                <button
                                    type="button"
                                    onClick={() => setPasswordMode("auto")}
                                    className={`text-[11px] font-semibold px-2 py-1 rounded transition ${passwordMode === "auto"
                                        ? "bg-indigo-600 text-white"
                                        : "bg-gray-200 text-gray-700 hover:bg-gray-300"
                                        }`}
                                >
                                    <Sparkles className="w-3 h-3 inline mr-1" />
                                    Otomatis
                                </button>
                                <button
                                    type="button"
                                    onClick={() => setPasswordMode("manual")}
                                    className={`text-[11px] font-semibold px-2 py-1 rounded transition ${passwordMode === "manual"
                                        ? "bg-indigo-600 text-white"
                                        : "bg-gray-200 text-gray-700 hover:bg-gray-300"
                                        }`}
                                >
                                    <Lock className="w-3 h-3 inline mr-1" />
                                    Manual
                                </button>
                            </div>
                        </div>

                        {passwordMode === "auto" ? (
                            <p className="text-[11px] text-gray-600 leading-relaxed">
                                Sistem akan membuat kata sandi acak yang aman. Anda dapat melihat dan menyalin kredensial tersebut langsung ke clipboard setelah akun berhasil dibuat.
                            </p>
                        ) : (
                            <div className="space-y-1">
                                <Input
                                    type="password"
                                    showPasswordToggle
                                    autoComplete="new-password"
                                    placeholder="Ketik kata sandi manual (min. 10 karakter)"
                                    value={manualPassword}
                                    onChange={(e) => setManualPassword(e.target.value)}
                                    className="text-xs bg-white"
                                />
                            </div>
                        )}

                        <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer pt-1">
                            <input
                                type="checkbox"
                                checked={mustChangePassword}
                                onChange={(e) => setMustChangePassword(e.target.checked)}
                                className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                            />
                            <span>Wajibkan ganti kata sandi pada saat pertama kali login</span>
                        </label>
                    </div>
                )}

                {/* Roles Assignment (Create Mode Only) */}
                {!isEdit && (
                    <div className="space-y-2">
                        <div className="flex items-center justify-between">
                            <label className="block text-xs font-semibold text-gray-700">
                                Peran Akses Sistem (Pilih Tepat 1 Peran) <span className="text-rose-500">*</span>
                            </label>
                            <span className="text-[11px] text-gray-500">
                                1 peran per pengguna
                            </span>
                        </div>

                        <div className="grid grid-cols-1 gap-2">
                            {rolesList.map((r) => {
                                const isChecked = selectedRoleId === r.id;
                                const isSuperAdminRole = r.name === "super_admin";
                                const isDisabled = isSuperAdminRole && !isSuperAdmin;

                                return (
                                    <label
                                        key={r.id}
                                        onClick={() => {
                                            if (!isDisabled) handleSelectRole(r.id, r.name);
                                        }}
                                        className={`flex items-start gap-3 p-3 rounded-xl border text-xs transition cursor-pointer ${isDisabled
                                            ? "bg-gray-50 border-gray-200 opacity-60 cursor-not-allowed"
                                            : isChecked
                                                ? "bg-indigo-50/80 border-indigo-300 ring-1 ring-indigo-500 shadow-2xs"
                                                : "bg-white border-gray-200 hover:bg-gray-50"
                                            }`}
                                    >
                                        <input
                                            type="radio"
                                            name="userRoleChoice"
                                            disabled={isDisabled}
                                            checked={isChecked}
                                            onChange={() => handleSelectRole(r.id, r.name)}
                                            onClick={(e) => e.stopPropagation()}
                                            className="mt-0.5 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                                        />
                                        <div className="min-w-0 flex-1">
                                            <div className="flex items-center justify-between">
                                                <div className="font-semibold text-gray-900 leading-tight">
                                                    {r.display_name || r.name}
                                                </div>
                                                <div className="text-[10px] text-gray-400 font-mono">{r.name}</div>
                                            </div>
                                            <p className="text-[11px] text-gray-500 mt-1 leading-snug">
                                                {r.description || "Peran akses sistem baku."}
                                            </p>
                                        </div>
                                    </label>
                                );
                            })}
                        </div>
                    </div>
                )}

                {/* Active Toggle (Edit Mode Only) */}
                {isEdit && (
                    <div className="p-3 bg-gray-50 rounded-xl border border-gray-200">
                        <label className="flex items-center gap-2.5 text-xs text-gray-900 font-medium cursor-pointer">
                            <input
                                type="checkbox"
                                checked={isActive}
                                onChange={(e) => setIsActive(e.target.checked)}
                                className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                            />
                            <span>Akun Aktif (Dapat Masuk ke Aplikasi)</span>
                        </label>
                        <p className="text-[11px] text-gray-500 mt-1 pl-6">
                            Jika dinonaktifkan, seluruh token login pengguna akan langsung dicabut dan sesi dikeluarkan.
                        </p>
                    </div>
                )}

                {/* Modal Footer */}
                <div className="flex items-center justify-end gap-2 pt-2 border-t border-gray-100">
                    <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={onClose}
                        disabled={isSubmitting}
                    >
                        Batal
                    </Button>
                    <Button
                        type="submit"
                        variant="primary"
                        size="sm"
                        isLoading={isSubmitting}
                    >
                        {isEdit ? "Simpan Perubahan" : "Buat Akun Sekarang"}
                    </Button>
                </div>
            </form>
        </Modal>
    );
}
