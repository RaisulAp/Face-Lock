import React from "react";
import { Lock, RefreshCw } from "lucide-react";
import type { GeolocationErrorDetails } from "../../hooks/useGeolocation";

interface LocationGateProps {
    isLoading: boolean;
    error: GeolocationErrorDetails | null;
    onRetry: () => void;
    children: React.ReactNode;
}

export const LocationGate: React.FC<LocationGateProps> = ({ isLoading, error, onRetry, children }) => {
    if (isLoading) {
        return (
            <div className="p-8 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 flex flex-col items-center justify-center text-center shadow-sm">
                <div className="w-10 h-10 rounded-full border-3 border-indigo-600 border-t-transparent animate-spin mb-3" />
                <p className="text-sm font-medium text-slate-700 dark:text-slate-200">Mendeteksi lokasi Anda...</p>
                <p className="text-xs text-slate-400 mt-1">Pastikan GPS aktif dan berikan izin akses lokasi.</p>
            </div>
        );
    }

    if (error && error.kind === "PERMISSION_DENIED") {
        return (
            <div className="p-6 rounded-2xl bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/50 flex flex-col items-center justify-center text-center shadow-sm">
                <div className="w-12 h-12 rounded-full bg-amber-100 dark:bg-amber-900/50 flex items-center justify-center text-amber-600 dark:text-amber-400 mb-3">
                    <Lock className="w-6 h-6" />
                </div>
                <h3 className="text-base font-semibold text-slate-900 dark:text-white mb-1">
                    Izin Lokasi Diperlukan
                </h3>
                <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 max-w-sm mb-4">
                    Presensi FaceClock memerlukan data lokasi untuk memverifikasi kehadiran Anda di area kantor.
                </p>
                <div className="text-xs text-slate-500 bg-white dark:bg-slate-900 p-3 rounded-xl border border-slate-200 dark:border-slate-800 mb-4 max-w-xs text-left">
                    <p className="font-semibold text-slate-700 dark:text-slate-300 mb-1">Cara mengaktifkan:</p>
                    <ol className="list-decimal list-inside space-y-0.5">
                        <li>Klik ikon gembok di address bar browser.</li>
                        <li>Ubah izin "Lokasi" menjadi "Izinkan".</li>
                        <li>Klik tombol Muat Ulang di bawah.</li>
                    </ol>
                </div>
                <button
                    type="button"
                    onClick={onRetry}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-amber-600 hover:bg-amber-700 text-white text-sm font-semibold transition-all shadow-sm"
                >
                    <RefreshCw className="w-4 h-4" />
                    Muat Ulang Izin
                </button>
            </div>
        );
    }

    return <>{children}</>;
};
