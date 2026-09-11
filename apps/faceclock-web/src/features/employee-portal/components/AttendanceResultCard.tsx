import React from "react";
import {
    CheckCircle2,
    AlertTriangle,
    XCircle,
    MapPin,
    Clock,
    ShieldCheck,
    RotateCcw,
    ArrowRight,
} from "lucide-react";
import type { ClockSubmitResult, AttendanceActionType } from "../types";

interface AttendanceResultCardProps {
    result: ClockSubmitResult;
    actionType: AttendanceActionType;
    onReset: () => void;
    onViewHistory: () => void;
}

export const AttendanceResultCard: React.FC<AttendanceResultCardProps> = ({
    result,
    actionType,
    onReset,
    onViewHistory,
}) => {
    const isApproved = result.status === "approved";
    const isPending = result.status === "pending_review";

    const actionTitle = actionType === "check_in" ? "Check-In (Masuk)" : "Check-Out (Pulang)";

    const formattedTime = new Date(result.recorded_at).toLocaleTimeString("id-ID", {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
    });

    return (
        <div className="bg-white rounded-2xl shadow-md border border-slate-200 overflow-hidden text-center max-w-lg mx-auto">
            {/* Header Banner */}
            <div
                className={`p-6 ${isApproved
                        ? "bg-emerald-600 text-white"
                        : isPending
                            ? "bg-amber-500 text-white"
                            : "bg-rose-600 text-white"
                    }`}
            >
                <div className="w-16 h-16 mx-auto rounded-full bg-white/20 flex items-center justify-center mb-3">
                    {isApproved ? (
                        <CheckCircle2 className="w-10 h-10 text-white" />
                    ) : isPending ? (
                        <AlertTriangle className="w-10 h-10 text-white" />
                    ) : (
                        <XCircle className="w-10 h-10 text-white" />
                    )}
                </div>

                <h2 className="text-xl font-bold">
                    {isApproved
                        ? `${actionTitle} Berhasil!`
                        : isPending
                            ? `${actionTitle} Menunggu Review`
                            : `${actionTitle} Gagal`}
                </h2>

                <p className="text-sm text-white/90 mt-1">
                    {isApproved
                        ? "Kehadiran Anda telah terverifikasi secara akurat oleh sistem biometrik."
                        : isPending
                            ? "Absensi telah dicatat dan membutuhkan tinjauan manual oleh atasan/admin."
                            : "Presensi tidak dapat diterima oleh sistem. Silakan coba kembali."}
                </p>
            </div>

            {/* Detail Breakdown */}
            <div className="p-6 space-y-4 text-left">
                <div className="bg-slate-50 rounded-xl p-4 space-y-3 border border-slate-100">
                    <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-500 flex items-center">
                            <Clock className="w-4 h-4 mr-1.5 text-slate-400" />
                            Waktu Server
                        </span>
                        <span className="font-semibold text-slate-900">{formattedTime} WIB</span>
                    </div>

                    <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-500 flex items-center">
                            <ShieldCheck className="w-4 h-4 mr-1.5 text-slate-400" />
                            Metode Presensi
                        </span>
                        <span className="font-semibold text-slate-900 capitalize">
                            {result.method === "face"
                                ? "Verifikasi Biometrik Wajah"
                                : result.method === "fallback"
                                    ? "Pengajuan Fallback"
                                    : "Manual"}
                        </span>
                    </div>

                    <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-500 flex items-center">
                            <MapPin className="w-4 h-4 mr-1.5 text-slate-400" />
                            Area Lokasi (Geofence)
                        </span>
                        <span
                            className={`font-semibold ${result.geofence_status === "inside"
                                    ? "text-emerald-700"
                                    : result.geofence_status === "outside"
                                        ? "text-rose-600"
                                        : "text-slate-700"
                                }`}
                        >
                            {result.geofence_status === "inside"
                                ? "Di Dalam Radius Kantor"
                                : result.geofence_status === "outside"
                                    ? "Di Luar Radius Kantor"
                                    : "Tidak Diketahui"}
                        </span>
                    </div>
                </div>

                {/* Pending explanation if applicable */}
                {isPending && (
                    <div className="bg-amber-50 border border-amber-200 rounded-xl p-3.5 text-xs text-amber-800 space-y-1">
                        <p className="font-semibold">Catatan Verifikasi Manual:</p>
                        <p>
                            Data presensi Anda disimpan dengan aman. Estimasi waktu peninjauan sekitar{" "}
                            <strong>{result.waiting_hours || 24} jam kerja</strong>. Anda tidak perlu mengulang presensi
                            kecuali diminta oleh atasan.
                        </p>
                    </div>
                )}

                {/* Action Buttons */}
                <div className="flex flex-col sm:flex-row gap-3 pt-2">
                    {!isApproved && (
                        <button
                            type="button"
                            onClick={onReset}
                            className="flex-1 inline-flex items-center justify-center py-2.5 px-4 rounded-xl border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 font-medium text-sm transition-colors shadow-xs"
                        >
                            <RotateCcw className="w-4 h-4 mr-2 text-slate-500" />
                            Coba Lagi
                        </button>
                    )}

                    <button
                        type="button"
                        onClick={onViewHistory}
                        className="flex-1 inline-flex items-center justify-center py-2.5 px-4 rounded-xl bg-primary-600 hover:bg-primary-700 text-white font-medium text-sm transition-colors shadow-xs"
                    >
                        Lihat Riwayat
                        <ArrowRight className="w-4 h-4 ml-2" />
                    </button>
                </div>
            </div>
        </div>
    );
};
