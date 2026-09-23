import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
    BadgeCheck,
    ClipboardList,
} from "lucide-react";
import { employeeApi } from "../api";
import { ApiError } from "../../../lib/api";

export const ProfilePage: React.FC = () => {
    const navigate = useNavigate();
    const queryClient = useQueryClient();

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

    // HR profile completion form.
    //
    // The form is seeded from the loaded profile, but an effect that copies
    // server data into state would fight the user's typing and trigger
    // cascading renders. Instead we keep ONLY the user's edits here, and fall
    // back to the fetched value when a field has not been touched yet. That
    // makes the fields correct on first paint without any sync effect.
    type ProfileEdits = {
        employee_number?: string;
        position?: string;
        join_date?: string;
        phone?: string;
    };
    const [profileEdits, setProfileEdits] = useState<ProfileEdits>({});
    const [profileFormError, setProfileFormError] = useState<string | null>(null);
    const [profileSaved, setProfileSaved] = useState(false);

    const employeeNumber = profileEdits.employee_number ?? profile?.employee_number ?? "";
    const position = profileEdits.position ?? profile?.position ?? "";
    const joinDate = profileEdits.join_date ?? profile?.join_date ?? "";
    const phoneNumber = profileEdits.phone ?? profile?.phone ?? "";

    const completeProfileMutation = useMutation({
        mutationFn: () =>
            employeeApi.completeMyProfile({
                employee_number: employeeNumber.trim(),
                position: position.trim(),
                join_date: joinDate,
                phone: phoneNumber.trim(),
            }),
        onSuccess: () => {
            setProfileFormError(null);
            setProfileSaved(true);
            queryClient.invalidateQueries({ queryKey: ["employee-profile-me"] });
        },
        onError: (err: unknown) => {
            setProfileSaved(false);
            // Prefer a field-level message when the backend told us which
            // field was wrong, otherwise fall back to the top-level message.
            if (err instanceof ApiError) {
                const detail = err.details?.[0];
                setProfileFormError(detail ? `${detail.field}: ${detail.message}` : err.message);
            } else {
                setProfileFormError(err instanceof Error ? err.message : "Gagal menyimpan profil.");
            }
        },
    });

    const handleProfileSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        setProfileFormError(null);
        setProfileSaved(false);

        if (employeeNumber.trim().length < 2) {
            setProfileFormError("NIP minimal 2 karakter.");
            return;
        }
        if (position.trim().length < 2) {
            setProfileFormError("Jabatan minimal 2 karakter.");
            return;
        }
        if (!joinDate) {
            setProfileFormError("Tanggal masuk wajib diisi.");
            return;
        }

        completeProfileMutation.mutate();
    };

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
                        {profile?.full_name?.charAt(0) || "K"}
                    </div>
                    <div>
                        <h1 className="text-lg font-bold text-slate-900">{profile?.full_name || "Karyawan"}</h1>
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

            {/* 2. HR Profile Completion Card */}
            <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs space-y-4">
                <div className="flex items-start justify-between gap-3">
                    <div className="flex items-center space-x-2.5">
                        <div className="w-8 h-8 rounded-lg bg-primary-50 text-primary-700 flex items-center justify-center">
                            <ClipboardList className="w-5 h-5" />
                        </div>
                        <div>
                            <h2 className="text-sm font-bold text-slate-900">Data Kepegawaian</h2>
                            <p className="text-xs text-slate-500">Lengkapi data yang belum diisi oleh HR</p>
                        </div>
                    </div>

                    {profile?.profile_completed ? (
                        <span className="inline-flex items-center space-x-1.5 px-2.5 py-1 rounded-lg bg-emerald-50 border border-emerald-200 text-[11px] font-bold text-emerald-700 shrink-0">
                            <BadgeCheck className="w-3.5 h-3.5" />
                            <span>Lengkap</span>
                        </span>
                    ) : (
                        <span className="inline-flex items-center space-x-1.5 px-2.5 py-1 rounded-lg bg-amber-50 border border-amber-200 text-[11px] font-bold text-amber-700 shrink-0">
                            <AlertTriangle className="w-3.5 h-3.5" />
                            <span>Belum Lengkap</span>
                        </span>
                    )}
                </div>

                <form onSubmit={handleProfileSubmit} className="space-y-3.5 pt-1">
                    {profileFormError && (
                        <div className="bg-rose-50 border border-rose-200 rounded-xl p-3 text-xs text-rose-800 flex items-center space-x-2">
                            <AlertTriangle className="w-4 h-4 shrink-0 text-rose-600" />
                            <span>{profileFormError}</span>
                        </div>
                    )}

                    {profileSaved && (
                        <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-3 text-xs text-emerald-800 flex items-center space-x-2">
                            <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-600" />
                            <span>Data kepegawaian berhasil disimpan.</span>
                        </div>
                    )}

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div>
                            <label className="block text-xs font-semibold text-slate-700 mb-1">
                                NIP / Nomor Karyawan: <span className="text-rose-500">*</span>
                            </label>
                            <input
                                type="text"
                                value={employeeNumber}
                                onChange={(e) => setProfileEdits((prev) => ({ ...prev, employee_number: e.target.value }))}
                                maxLength={50}
                                placeholder="Contoh: EMP-00123"
                                className="w-full text-xs rounded-xl border border-slate-300 px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500"
                            />
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-slate-700 mb-1">
                                Jabatan: <span className="text-rose-500">*</span>
                            </label>
                            <input
                                type="text"
                                value={position}
                                onChange={(e) => setProfileEdits((prev) => ({ ...prev, position: e.target.value }))}
                                maxLength={100}
                                placeholder="Contoh: Staf Keuangan"
                                className="w-full text-xs rounded-xl border border-slate-300 px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500"
                            />
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-slate-700 mb-1">
                                Tanggal Masuk: <span className="text-rose-500">*</span>
                            </label>
                            <input
                                type="date"
                                value={joinDate}
                                onChange={(e) => setProfileEdits((prev) => ({ ...prev, join_date: e.target.value }))}
                                className="w-full text-xs rounded-xl border border-slate-300 px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500"
                            />
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-slate-700 mb-1">Nomor HP:</label>
                            <input
                                type="tel"
                                value={phoneNumber}
                                onChange={(e) => setProfileEdits((prev) => ({ ...prev, phone: e.target.value }))}
                                maxLength={20}
                                placeholder="Contoh: 081234567890"
                                className="w-full text-xs rounded-xl border border-slate-300 px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500"
                            />
                        </div>
                    </div>

                    <p className="text-[11px] text-slate-400 leading-relaxed">
                        Departemen, email, status kepegawaian dan lokasi kerja hanya dapat diubah oleh HR.
                    </p>

                    <div className="flex justify-end pt-1">
                        <button
                            type="submit"
                            disabled={completeProfileMutation.isPending}
                            className="px-5 py-2.5 rounded-xl bg-primary-600 hover:bg-primary-700 text-white text-xs font-bold transition-all shadow-xs flex items-center space-x-2 disabled:opacity-50"
                        >
                            {completeProfileMutation.isPending ? (
                                <>
                                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                    <span>Menyimpan...</span>
                                </>
                            ) : (
                                <span>Simpan Data Kepegawaian</span>
                            )}
                        </button>
                    </div>
                </form>
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
