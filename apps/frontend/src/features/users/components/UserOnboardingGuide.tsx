import { useState } from "react";
import { Users, Shield, KeyRound, Sparkles, X, ChevronDown, ChevronUp, Info, ArrowRight } from "lucide-react";
import { Button } from "../../../components/ui/Button";

interface UserOnboardingGuideProps {
    onOpenCreateModal?: () => void;
    canCreateUser?: boolean;
}

export function UserOnboardingGuide({ onOpenCreateModal, canCreateUser = true }: UserOnboardingGuideProps) {
    const [isDismissed, setIsDismissed] = useState<boolean>(() => {
        return localStorage.getItem("fc_users_guide_dismissed") === "true";
    });
    const [isExpanded, setIsExpanded] = useState(true);

    const handleDismiss = () => {
        setIsDismissed(true);
        localStorage.setItem("fc_users_guide_dismissed", "true");
    };

    const handleRestore = () => {
        setIsDismissed(false);
        localStorage.removeItem("fc_users_guide_dismissed");
    };

    if (isDismissed) {
        return (
            <div className="flex justify-end">
                <button
                    type="button"
                    onClick={handleRestore}
                    className="inline-flex items-center gap-1.5 text-xs font-semibold text-indigo-600 hover:text-indigo-800 bg-indigo-50/80 px-3 py-1.5 rounded-lg border border-indigo-100 transition"
                >
                    <Info className="w-3.5 h-3.5" />
                    Pelajari Perbedaan Karyawan vs Akun Pengguna
                </button>
            </div>
        );
    }

    return (
        <div className="rounded-2xl bg-gradient-to-r from-indigo-900 via-indigo-800 to-blue-900 text-white p-5 shadow-sm relative overflow-hidden transition-all duration-200">
            {/* Decorative backdrop shapes */}
            <div className="absolute right-0 top-0 translate-x-12 -translate-y-8 w-64 h-64 bg-white/5 rounded-full blur-2xl pointer-events-none" />
            <div className="absolute left-1/3 bottom-0 translate-y-12 w-48 h-48 bg-purple-500/10 rounded-full blur-xl pointer-events-none" />

            <div className="relative z-10">
                <div className="flex items-start justify-between gap-4">
                    <div className="flex items-center gap-2.5">
                        <div className="p-2 rounded-xl bg-white/10 text-indigo-200 border border-white/10 shrink-0">
                            <Sparkles className="w-5 h-5 text-amber-300" />
                        </div>
                        <div>
                            <h2 className="font-bold text-base sm:text-lg text-white">
                                Panduan Memulai: Pengguna & Hak Akses
                            </h2>
                            <p className="text-xs text-indigo-200 mt-0.5">
                                Baru pertama kali menggunakan FaceClock? Pahami konsep akun agar pengaturan akses berjalan tepat.
                            </p>
                        </div>
                    </div>

                    <div className="flex items-center gap-1 shrink-0">
                        <button
                            type="button"
                            onClick={() => setIsExpanded(!isExpanded)}
                            className="p-1 text-indigo-300 hover:text-white rounded-lg transition hover:bg-white/10"
                            title={isExpanded ? "Ciutkan" : "Perluas"}
                        >
                            {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                        </button>
                        <button
                            type="button"
                            onClick={handleDismiss}
                            className="p-1 text-indigo-300 hover:text-white rounded-lg transition hover:bg-white/10"
                            title="Tutup panduan ini"
                        >
                            <X className="w-4 h-4" />
                        </button>
                    </div>
                </div>

                {isExpanded && (
                    <div className="mt-4 space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
                            {/* Card 1: Karyawan vs Pengguna */}
                            <div className="p-3.5 rounded-xl bg-white/10 border border-white/10 backdrop-blur-xs space-y-2">
                                <div className="flex items-center gap-2 font-bold text-white">
                                    <Users className="w-4 h-4 text-emerald-300" />
                                    <span>1. Karyawan vs Pengguna</span>
                                </div>
                                <p className="text-indigo-100 text-[11px] leading-relaxed">
                                    <strong>Karyawan:</strong> Identitas fisik SDM yang melakukan presensi wajah di kantor.
                                    <br />
                                    <strong>Pengguna:</strong> Akun login (Email & Sandi) untuk mengelola web dashboard FaceClock.
                                </p>
                            </div>

                            {/* Card 2: Hubungkan Otomatis */}
                            <div className="p-3.5 rounded-xl bg-white/10 border border-white/10 backdrop-blur-xs space-y-2">
                                <div className="flex items-center gap-2 font-bold text-white">
                                    <Shield className="w-4 h-4 text-amber-300" />
                                    <span>2. Hubungkan ke Pegawai</span>
                                </div>
                                <p className="text-indigo-100 text-[11px] leading-relaxed">
                                    Pilih pegawai yang belum memiliki akun pengguna. Sistem akan otomatis mengisi email dan mengaitkan riwayat login ke data pegawai tersebut.
                                </p>
                            </div>

                            {/* Card 3: Peran & Kata Sandi */}
                            <div className="p-3.5 rounded-xl bg-white/10 border border-white/10 backdrop-blur-xs space-y-2">
                                <div className="flex items-center gap-2 font-bold text-white">
                                    <KeyRound className="w-4 h-4 text-purple-300" />
                                    <span>3. Kata Sandi & Peran (RBAC)</span>
                                </div>
                                <p className="text-indigo-100 text-[11px] leading-relaxed">
                                    Tentukan peran (misal: HR, Supervisor) dan dapatkan kata sandi acak yang aman. User akan diminta mengganti sandi saat pertama login.
                                </p>
                            </div>
                        </div>

                        <div className="flex items-center justify-between pt-1 text-xs">
                            <span className="text-[11px] text-indigo-200">
                                Tip: Anda dapat menyalin kredensial login dengan 1 klik setelah akun pengguna dibuat.
                            </span>

                            {onOpenCreateModal && canCreateUser && (
                                <Button
                                    size="sm"
                                    onClick={onOpenCreateModal}
                                    className="bg-white text-indigo-900 hover:bg-indigo-50 font-semibold text-xs border-0 shadow-sm"
                                >
                                    Mulai Buat Pengguna Baru
                                    <ArrowRight className="w-3.5 h-3.5 ml-1" />
                                </Button>
                            )}
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
