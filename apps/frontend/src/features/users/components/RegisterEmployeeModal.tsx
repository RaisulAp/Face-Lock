import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import {
    AlertCircle,
    Sparkles,
    Lock,
    KeyRound,
    Building2,
    MapPin,
    UserPlus,
    Phone,
    Briefcase,
    Info,
} from "lucide-react";
import type { Role, OfficeLocationItem } from "../../../types/api";
import { api } from "../../../lib/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";

/**
 * The single registration form. One submission creates BOTH the employee row
 * and the login account, which removes the old two-step flow where an admin had
 * to build an employee first and then attach a user to it.
 *
 * The admin only fills what they actually know: identity (name, email, phone)
 * and placement (department text, office location, optional job title). The
 * employee completes the rest themselves from the portal: NIP, position and
 * join date, plus face enrollment.
 */
interface RegisterEmployeeResult {
    user?: { id: string; email: string };
    employee_id: string;
    employee_name: string;
    email: string;
    department?: string | null;
    office_location?: string | null;
    temporary_password?: string;
}

interface RegisterEmployeeModalProps {
    open: boolean;
    onClose: () => void;
    onSuccess: (result: RegisterEmployeeResult) => void;
}

export function RegisterEmployeeModal({ open, onClose, onSuccess }: RegisterEmployeeModalProps) {
    const [fullName, setFullName] = useState("");
    const [email, setEmail] = useState("");
    const [phone, setPhone] = useState("");
    const [department, setDepartment] = useState("");
    const [officeLocationId, setOfficeLocationId] = useState("");
    const [jobTitle, setJobTitle] = useState("");

    const [passwordMode, setPasswordMode] = useState<"auto" | "manual">("auto");
    const [manualPassword, setManualPassword] = useState("");
    const [mustChangePassword, setMustChangePassword] = useState(true);

    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Office locations drive the attendance geofence, so the selection matters.
    const { data: officeLocations = [], isLoading: isLoadingLocations } = useQuery({
        queryKey: ["office-locations-for-register"],
        queryFn: async () => {
            try {
                return await api.get<OfficeLocationItem[]>("/api/v1/office-locations");
            } catch {
                return [];
            }
        },
        enabled: open,
    });

    // Reset the form every time the modal is opened so a previous attempt does
    // not leak values into a fresh registration.
    useEffect(() => {
        if (open) {
            setFullName("");
            setEmail("");
            setPhone("");
            setDepartment("");
            setOfficeLocationId("");
            setJobTitle("");
            setPasswordMode("auto");
            setManualPassword("");
            setMustChangePassword(true);
            setError(null);
            setIsSubmitting(false);
        }
    }, [open]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        const trimmedName = fullName.trim();
        const trimmedEmail = email.trim();

        if (!trimmedName) {
            setError("Nama lengkap wajib diisi.");
            return;
        }
        if (!trimmedEmail) {
            setError("Email wajib diisi.");
            return;
        }
        if (passwordMode === "manual" && manualPassword.length < 10) {
            setError("Kata sandi manual harus minimal 10 karakter.");
            return;
        }

        const payload: Record<string, unknown> = {
            full_name: trimmedName,
            email: trimmedEmail,
            must_change_password: mustChangePassword,
        };
        if (phone.trim()) payload.phone = phone.trim();
        if (department.trim()) payload.department = department.trim();
        if (officeLocationId) payload.office_location_id = officeLocationId;
        if (jobTitle.trim()) payload.job_title = jobTitle.trim();
        if (passwordMode === "manual" && manualPassword) payload.password = manualPassword;

        setIsSubmitting(true);
        try {
            const res = await api.post<{ data: RegisterEmployeeResult }>(
                "/api/v1/users/register-employee",
                payload,
            );
            const result = (res as any).data || res;
            onSuccess(result as RegisterEmployeeResult);
            onClose();
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : "Gagal mendaftarkan karyawan.";
            setError(msg);
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <Modal
            open={open}
            onClose={onClose}
            title="Daftarkan Karyawan Baru"
            description="Satu formulir membuat data karyawan sekaligus akun login-nya. Karyawan melengkapi NIP, jabatan, dan presensi wajah sendiri dari portal."
            maxWidth="lg"
        >
            <form onSubmit={handleSubmit} className="space-y-4">
                {error && (
                    <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 rounded-lg text-xs flex items-center gap-2">
                        <AlertCircle className="w-4 h-4 shrink-0 text-rose-500" />
                        <span>{error}</span>
                    </div>
                )}

                <div className="p-3 bg-indigo-50/50 border border-indigo-100 rounded-xl flex gap-2 text-[11px] text-indigo-900 leading-relaxed">
                    <Info className="w-4 h-4 shrink-0 text-indigo-500 mt-0.5" />
                    <span>
                        Isi hanya data yang Anda ketahui. Sistem akan membuat NIP sementara secara otomatis,
                        lalu karyawan mengisi NIP, jabatan, dan tanggal masuk yang sebenarnya saat login pertama.
                    </span>
                </div>

                {/* Identitas */}
                <div className="space-y-3">
                    <h4 className="text-xs font-bold text-gray-900 flex items-center gap-1.5">
                        <UserPlus className="w-3.5 h-3.5 text-indigo-600" />
                        Identitas
                    </h4>

                    <div className="space-y-1.5">
                        <label className="block text-xs font-semibold text-gray-700">
                            Nama Lengkap <span className="text-rose-500">*</span>
                        </label>
                        <Input
                            required
                            placeholder="contoh: Budi Santoso"
                            value={fullName}
                            onChange={(e) => setFullName(e.target.value)}
                            className="text-xs"
                        />
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div className="space-y-1.5">
                            <label className="block text-xs font-semibold text-gray-700">
                                Email Login <span className="text-rose-500">*</span>
                            </label>
                            <Input
                                type="email"
                                required
                                placeholder="contoh: budi@perusahaan.com"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                className="text-xs"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <label className="block text-xs font-semibold text-gray-700 flex items-center gap-1">
                                <Phone className="w-3 h-3" /> Nomor Telepon
                            </label>
                            <Input
                                placeholder="contoh: 08123456789"
                                value={phone}
                                onChange={(e) => setPhone(e.target.value)}
                                className="text-xs"
                            />
                        </div>
                    </div>
                </div>

                {/* Penempatan */}
                <div className="space-y-3 pt-1 border-t border-gray-100">
                    <h4 className="text-xs font-bold text-gray-900 flex items-center gap-1.5 pt-3">
                        <Building2 className="w-3.5 h-3.5 text-indigo-600" />
                        Penempatan Kerja
                    </h4>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div className="space-y-1.5">
                            <label className="block text-xs font-semibold text-gray-700">Departemen</label>
                            <Input
                                placeholder="contoh: Engineering"
                                value={department}
                                onChange={(e) => setDepartment(e.target.value)}
                                className="text-xs"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <label className="block text-xs font-semibold text-gray-700 flex items-center gap-1">
                                <Briefcase className="w-3 h-3" /> Jabatan (opsional)
                            </label>
                            <Input
                                placeholder="contoh: Staff Finance"
                                value={jobTitle}
                                onChange={(e) => setJobTitle(e.target.value)}
                                className="text-xs"
                            />
                        </div>
                    </div>

                    <div className="space-y-1.5">
                        <label className="block text-xs font-semibold text-gray-700 flex items-center gap-1">
                            <MapPin className="w-3 h-3" /> Kantor Utama (lokasi presensi)
                        </label>
                        <select
                            value={officeLocationId}
                            onChange={(e) => setOfficeLocationId(e.target.value)}
                            disabled={isLoadingLocations}
                            className="w-full text-xs rounded-lg border border-gray-300 bg-white py-2 px-3 focus:outline-hidden focus:ring-2 focus:ring-indigo-500"
                        >
                            <option value="">-- Belum ditentukan (dapat diubah nanti) --</option>
                            {officeLocations.map((loc) => (
                                <option key={loc.id} value={loc.id}>
                                    {loc.name}
                                    {loc.address ? ` — ${loc.address}` : ""}
                                </option>
                            ))}
                        </select>
                        <p className="text-[11px] text-gray-500">
                            Kantor ini menentukan area geofence saat karyawan melakukan presensi.
                        </p>
                    </div>
                </div>

                {/* Kata sandi */}
                <div className="p-3.5 bg-gray-50 border border-gray-200 rounded-xl space-y-3">
                    <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-gray-900 flex items-center gap-1.5">
                            <KeyRound className="w-3.5 h-3.5 text-indigo-600" />
                            Kata Sandi Awal
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
                            Sistem membuat kata sandi acak yang aman. Kredensial ditampilkan sekali setelah
                            pendaftaran berhasil agar Anda dapat menyalinnya.
                        </p>
                    ) : (
                        <Input
                            type="password"
                            showPasswordToggle
                            autoComplete="new-password"
                            placeholder="Ketik kata sandi manual (min. 10 karakter)"
                            value={manualPassword}
                            onChange={(e) => setManualPassword(e.target.value)}
                            className="text-xs bg-white"
                        />
                    )}

                    <label className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
                        <input
                            type="checkbox"
                            checked={mustChangePassword}
                            onChange={(e) => setMustChangePassword(e.target.checked)}
                            className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                        />
                        <span>Wajibkan ganti kata sandi saat login pertama</span>
                    </label>
                </div>

                <div className="flex items-center justify-between gap-2 pt-2 border-t border-gray-100">
                    <p className="text-[11px] text-gray-500">
                        Peran akses otomatis: <strong className="text-gray-700">Employee</strong>
                    </p>
                    <div className="flex items-center gap-2">
                        <Button type="button" variant="outline" size="sm" onClick={onClose} disabled={isSubmitting}>
                            Batal
                        </Button>
                        <Button type="submit" variant="primary" size="sm" isLoading={isSubmitting}>
                            Daftarkan Sekarang
                        </Button>
                    </div>
                </div>
            </form>
        </Modal>
    );
}

/** Kept as a named export for tests that assert the default role label. */
export const DEFAULT_REGISTER_ROLE = "employee" as const;

export type { RegisterEmployeeResult };
export type RegisterRole = Role;
