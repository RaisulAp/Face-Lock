import React from "react";
import { Camera, AlertCircle, RefreshCw, Lock, ShieldAlert } from "lucide-react";
import type { CameraErrorDetails } from "../../lib/media/camera";

interface CameraGateProps {
    isLoading: boolean;
    error: CameraErrorDetails | null;
    onRetry: () => void;
    children: React.ReactNode;
}

export const CameraGate: React.FC<CameraGateProps> = ({ isLoading, error, onRetry, children }) => {
    if (isLoading) {
        return (
            <div className="w-full aspect-[3/4] sm:aspect-video rounded-2xl bg-slate-900 flex flex-col items-center justify-center p-6 text-white text-center shadow-inner">
                <div className="w-12 h-12 rounded-full border-4 border-indigo-500/30 border-t-indigo-500 animate-spin mb-4" />
                <p className="text-sm font-medium text-slate-300">Mengakses kamera perangkat...</p>
                <p className="text-xs text-slate-500 mt-1">Harap berikan izin jika diminta oleh browser.</p>
            </div>
        );
    }

    if (error) {
        let icon = <AlertCircle className="w-12 h-12 text-rose-500 mb-3" />;
        let title = "Kendala Kamera";
        let advice = "Pastikan browser Anda memiliki izin untuk menggunakan kamera.";

        if (error.kind === "PERMISSION_DENIED") {
            icon = <Lock className="w-12 h-12 text-amber-500 mb-3" />;
            title = "Izin Kamera Ditolak";
            advice =
                "Klik ikon gembok atau kamera di address bar browser Anda, ubah izin Kamera menjadi 'Izinkan' (Allow), lalu klik tombol coba lagi di bawah.";
        } else if (error.kind === "NOT_SECURE") {
            icon = <ShieldAlert className="w-12 h-12 text-rose-500 mb-3" />;
            title = "Koneksi Tidak Aman";
            advice =
                "Browser membatasi fitur kamera hanya pada sambungan HTTPS atau localhost. Harap gunakan URL yang aman.";
        } else if (error.kind === "NOT_FOUND") {
            icon = <Camera className="w-12 h-12 text-slate-400 mb-3" />;
            title = "Kamera Tidak Terdeteksi";
            advice = "Periksa apakah webcam terhubung dengan benar atau tidak ditutup oleh sakelar fisik.";
        } else if (error.kind === "IN_USE") {
            icon = <Camera className="w-12 h-12 text-amber-500 mb-3" />;
            title = "Kamera Sedang Digunakan";
            advice =
                "Kamera mungkin sedang digunakan oleh tab lain atau aplikasi lain (Zoom/Meet). Tutup aplikasi tersebut lalu coba lagi.";
        }

        return (
            <div className="w-full rounded-2xl bg-rose-50/80 dark:bg-rose-950/20 border border-rose-200 dark:border-rose-900/50 p-6 flex flex-col items-center justify-center text-center shadow-sm">
                {icon}
                <h3 className="text-base font-semibold text-slate-900 dark:text-white mb-2">{title}</h3>
                <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 max-w-sm mb-2">{error.message}</p>
                <p className="text-xs text-slate-500 dark:text-slate-400 max-w-xs mb-5 bg-white/60 dark:bg-slate-900/60 p-2.5 rounded-lg border border-slate-200/50 dark:border-slate-800">
                    💡 {advice}
                </p>
                <button
                    type="button"
                    onClick={onRetry}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium shadow-sm transition-all"
                >
                    <RefreshCw className="w-4 h-4" />
                    Coba Lagi
                </button>
            </div>
        );
    }

    return <>{children}</>;
};
