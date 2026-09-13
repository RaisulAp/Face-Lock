import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation } from "@tanstack/react-query";
import {
    User,
    Mail,
    Building2,
    Briefcase,
    Phone,
    ShieldCheck,
    KeyRound,
    Eye,
    EyeOff,
    CheckCircle2,
    AlertTriangle,
    Loader2,
    ArrowRight,
    Shield,
} from "lucide-react";
import { employeeApi } from "../api";
import { ApiError } from "../../../lib/api";

export const ProfilePage: React.FC = () => {
    const navigate = useNavigate();

    // 1. Fetch Profile
    const { data: profile, isLoading: isLoadingProfile } = useQuery({
        queryKey: ["employee-profile-me"],
        queryFn: employeeApi.getMyProfile,
    });

    // 2. Fetch Enrollment Status
    const { data: enrollmentStatus } = useQuery({
        queryKey: ["enrollment-status-me"],
        queryFn: employeeApi.getMyEnrollmentStatus,
    });

    // Password Change Form State
    const [currentPassword, setCurrentPassword] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [showCurrent, setShowCurrent] = useState(false);
    const [showNew, setShowNew] = useState(false);
    const [showConfirm, setShowConfirm] = useState(false);
    const [formError, setFormError] = useState<string | null>(null);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    const changePasswordMutation = useMutation({
        mutationFn: () => employeeApi.changePassword(currentPassword, newPassword),
        onSuccess: () => {
            setSuccessMessage("Kata sandi berhasil diperbarui dengan aman.");
            setFormError(null);
            setCurrentPassword("");
            setNewPassword("");
            setConfirmPassword("");
        },
        onError: (err: unknown) => {
            setSuccessMessage(null);
            if (err instanceof ApiError) {
                setFormError(err.message);
            } else {
                setFormError(err instanceof Error ? err.message : "Gagal mengganti kata sandi.");
            }
        },
    });

    const handlePasswordSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        setFormError(null);
        setSuccessMessage(null);

        if (!currentPassword) {
            setFormError("Kata sandi saat ini wajib diisi.");
            return;
        }
        if (newPassword.length < 8) {
            setFormError("Kata sandi baru minimal harus 8 karakter.");
            return;
        }
        if (newPassword !== confirmPassword) {
            setFormError("Konfirmasi kata sandi tidak cocok dengan kata sandi baru.");
            return;
        }

        changePasswordMutation.mutate();
    };

    if (isLoadingProfile) {
        return (
            <div className="max-w-2xl mx-auto p-12 text-center space-y-3">
                <Loader2 className="w-8 h-8 animate-spin text-primary-600 mx-auto" />
                <p className="text-xs text-slate-500">Memuat profil karyawan...</p>
            </div>
        );
    }

    return (
        <div className="max-w-2xl mx-auto space-y-6">
            {/* 1. Profile Overview Card */}
            <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs space-y-5">
                <div className="flex items-center space-x-4">
                    <div className="w-14 h-14 rounded-2xl bg-primary-600 text-white flex items-center justify-center font-bold text-xl shadow-xs">
                        {profile?.name?.charAt(0) || "K"}
                    </div>
                    <div>
                        <h1 className="text-lg font-bold text-slate-900">{profile?.name || "Karyawan"}</h1>
                        <p className="text-xs text-slate-500 font-mono">NIK: {profile?.employee_number || "-"}</p>
                    </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 text-xs border-t border-slate-100">
                    <div className="flex items-center space-x-2.5 text-slate-600 p-2.5 rounded-xl bg-slate-50 border border-slate-100">
                        <Mail className="w-4 h-4 text-slate-400 shrink-0" />
                        <span className="truncate">{profile?.email || "-"}</span>
                    </div>

                    <div className="flex items-center space-x-2.5 text-slate-600 p-2.5 rounded-xl bg-slate-50 border border-slate-100">
                        <Building2 className="w-4 h-4 text-slate-400 shrink-0" />
                        <span className="truncate">{profile?.department || "Departemen Umum"}</span>
                    </div>

                    <div className="flex items-center space-x-2.5 text-slate-600 p-2.5 rounded-xl bg-slate-50 border border-slate-100">
                        <Briefcase className="w-4 h-4 text-slate-400 shrink-0" />
                        <span className="truncate">{profile?.position || "Staf"}</span>
                    </div>

                    <div className="flex items-center space-x-2.5 text-slate-600 p-2.5 rounded-xl bg-slate-50 border border-slate-100">
                        <Phone className="w-4 h-4 text-slate-400 shrink-0" />
                        <span className="truncate">{profile?.phone || "Belum ada nomor HP"}</span>
                    </div>
                </div>
            </div>

            {/* 2. Biometric Security Summary Card */}
            <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs space-y-4">
                <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2.5">
                        <div className="w-8 h-8 rounded-lg bg-emerald-100 text-emerald-700 flex items-center justify-center">
                            <ShieldCheck className="w-5 h-5" />
                        </div>
                        <div>
                            <h2 className="text-sm font-bold text-slate-900">Keamanan Biometrik & PDP</h2>
                            <p className="text-xs text-slate-500">Status verifikasi wajah dan konsen pribadi</p>
                        </div>
                    </div>
                </div>

                <div className="bg-slate-50 rounded-xl p-4 border border-slate-100 space-y-3 text-xs">
                    <div className="flex items-center justify-between">
                        <span className="text-slate-600">Status Pendaftaran Wajah:</span>
                        <span
                            className={`font-bold ${enrollmentStatus?.is_enrolled ? "text-emerald-700" : "text-amber-700"}`}
                        >
                            {enrollmentStatus?.is_enrolled
                                ? `Aktif (${enrollmentStatus.active_reference_count} Foto)`
                                : "Belum Terdaftar"}
                        </span>
                    </div>

                    <div className="flex items-center justify-between">
                        <span className="text-slate-600">Persetujuan UU PDP:</span>
                        <span
                            className={`font-bold ${enrollmentStatus?.consent?.status === "granted" ? "text-emerald-700" : "text-rose-600"
                                }`}
                        >
                            {enrollmentStatus?.consent?.status === "granted" ? "Telah Disetujui" : "Belum Disetujui"}
                        </span>
                    </div>
                </div>

                <div className="flex flex-col sm:flex-row gap-2.5 pt-1">
                    <button
                        type="button"
                        onClick={() => navigate("/portal/enrollment")}
                        className="flex-1 py-2 px-3 rounded-xl border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-xs font-semibold inline-flex items-center justify-center space-x-1.5 transition-colors"
                    >
                        <User className="w-3.5 h-3.5 text-slate-500" />
                        <span>Kelola Biometrik Wajah</span>
                        <ArrowRight className="w-3.5 h-3.5 text-slate-400" />
                    </button>

                    <button
                        type="button"
                        onClick={() => navigate("/portal/consent")}
                        className="flex-1 py-2 px-3 rounded-xl border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-xs font-semibold inline-flex items-center justify-center space-x-1.5 transition-colors"
                    >
                        <Shield className="w-3.5 h-3.5 text-slate-500" />
                        <span>Lembar Konsen PDP</span>
                        <ArrowRight className="w-3.5 h-3.5 text-slate-400" />
                    </button>
                </div>
            </div>

            {/* 3. Change Password Card */}
            <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs space-y-4">
                <div className="flex items-center space-x-2.5">
                    <div className="w-8 h-8 rounded-lg bg-primary-50 text-primary-700 flex items-center justify-center">
                        <KeyRound className="w-5 h-5" />
                    </div>
                    <div>
                        <h2 className="text-sm font-bold text-slate-900">Perbarui Kata Sandi</h2>
                        <p className="text-xs text-slate-500">Gunakan kombinasi minimal 8 karakter yang kuat</p>
                    </div>
                </div>

                <form onSubmit={handlePasswordSubmit} className="space-y-3.5 pt-1">
                    {formError && (
                        <div className="bg-rose-50 border border-rose-200 rounded-xl p-3 text-xs text-rose-800 flex items-center space-x-2">
                            <AlertTriangle className="w-4 h-4 shrink-0 text-rose-600" />
                            <span>{formError}</span>
                        </div>
                    )}

                    {successMessage && (
                        <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-3 text-xs text-emerald-800 flex items-center space-x-2">
                            <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-600" />
                            <span>{successMessage}</span>
                        </div>
                    )}

                    <div>
                        <label className="block text-xs font-semibold text-slate-700 mb-1">
                            Kata Sandi Saat Ini: <span className="text-rose-500">*</span>
                        </label>
                        <div className="relative">
                            <input
                                type={showCurrent ? "text" : "password"}
                                value={currentPassword}
                                onChange={(e) => setCurrentPassword(e.target.value)}
                                required
                                className="w-full text-xs rounded-xl border border-slate-300 px-3 py-2.5 pr-10 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500"
                            />
                            <button
                                type="button"
                                onClick={() => setShowCurrent(!showCurrent)}
                                className="absolute right-3 top-2.5 text-slate-400 hover:text-slate-600"
                            >
                                {showCurrent ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-xs font-semibold text-slate-700 mb-1">
                            Kata Sandi Baru: <span className="text-rose-500">*</span>
                        </label>
                        <div className="relative">
                            <input
                                type={showNew ? "text" : "password"}
                                value={newPassword}
                                onChange={(e) => setNewPassword(e.target.value)}
                                required
                                minLength={8}
                                className="w-full text-xs rounded-xl border border-slate-300 px-3 py-2.5 pr-10 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500"
                            />
                            <button
                                type="button"
                                onClick={() => setShowNew(!showNew)}
                                className="absolute right-3 top-2.5 text-slate-400 hover:text-slate-600"
                            >
                                {showNew ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-xs font-semibold text-slate-700 mb-1">
                            Konfirmasi Kata Sandi Baru: <span className="text-rose-500">*</span>
                        </label>
                        <div className="relative">
                            <input
                                type={showConfirm ? "text" : "password"}
                                value={confirmPassword}
                                onChange={(e) => setConfirmPassword(e.target.value)}
                                required
                                minLength={8}
                                className="w-full text-xs rounded-xl border border-slate-300 px-3 py-2.5 pr-10 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500"
                            />
                            <button
                                type="button"
                                onClick={() => setShowConfirm(!showConfirm)}
                                className="absolute right-3 top-2.5 text-slate-400 hover:text-slate-600"
                            >
                                {showConfirm ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                            </button>
                        </div>
                    </div>

                    <div className="flex justify-end pt-2">
                        <button
                            type="submit"
                            disabled={changePasswordMutation.isPending}
                            className="px-5 py-2.5 rounded-xl bg-primary-600 hover:bg-primary-700 text-white text-xs font-bold transition-all shadow-xs flex items-center space-x-2 disabled:opacity-50"
                        >
                            {changePasswordMutation.isPending ? (
                                <>
                                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                    <span>Menyimpan...</span>
                                </>
                            ) : (
                                <span>Simpan Perubahan</span>
                            )}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};
