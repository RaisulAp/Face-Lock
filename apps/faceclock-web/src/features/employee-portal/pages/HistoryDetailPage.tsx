import React from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
    ArrowLeft,
    CheckCircle2,
    AlertTriangle,
    XCircle,
    Clock,
    MapPin,
    ShieldCheck,
    FileText,
    Loader2,
    AlertCircle,
} from "lucide-react";
import { employeeApi } from "../api";

export const HistoryDetailPage: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();

    const {
        data: record,
        isLoading,
        error,
    } = useQuery({
        queryKey: ["attendance-detail", id],
        queryFn: () => employeeApi.getAttendanceDetail(id!),
        enabled: Boolean(id),
    });

    if (isLoading) {
        return (
            <div className="max-w-2xl mx-auto p-12 text-center space-y-3">
                <Loader2 className="w-8 h-8 animate-spin text-primary-600 mx-auto" />
                <p className="text-xs text-slate-500">Memuat rincian presensi...</p>
            </div>
        );
    }

    if (error || !record) {
        return (
            <div className="max-w-2xl mx-auto p-8 bg-white rounded-2xl border border-slate-200 text-center space-y-4">
                <AlertCircle className="w-10 h-10 text-rose-500 mx-auto" />
                <h2 className="text-base font-bold text-slate-900">Data Presensi Tidak Ditemukan</h2>
                <p className="text-xs text-slate-500">
                    Catatan presensi ini mungkin sudah kedaluwarsa atau tidak dapat diakses.
                </p>
                <button
                    type="button"
                    onClick={() => navigate("/portal/history")}
                    className="px-4 py-2 rounded-xl bg-primary-600 text-white text-xs font-semibold"
                >
                    Kembali ke Riwayat
                </button>
            </div>
        );
    }

    const isApproved = record.status === "approved";
    const isPending = record.status === "pending_review";
    const isCheckIn = record.type === "check_in" || record.type === "in";

    const recordedDate = new Date(record.server_timestamp || record.recorded_at).toLocaleDateString("id-ID", {
        weekday: "long",
        day: "numeric",
        month: "long",
        year: "numeric",
    });

    const recordedTime = new Date(record.server_timestamp || record.recorded_at).toLocaleTimeString("id-ID", {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
    });

    return (
        <div className="max-w-2xl mx-auto space-y-6">
            {/* Top Bar Back Button */}
            <button
                type="button"
                onClick={() => navigate("/portal/history")}
                className="inline-flex items-center text-xs font-semibold text-slate-600 hover:text-slate-900 transition-colors"
            >
                <ArrowLeft className="w-4 h-4 mr-1.5" />
                Kembali ke Daftar Riwayat
            </button>

            {/* Main Card */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-xs overflow-hidden">
                {/* Status Header */}
                <div
                    className={`p-6 text-white flex items-center space-x-4 ${isApproved ? "bg-emerald-600" : isPending ? "bg-amber-500" : "bg-rose-600"
                        }`}
                >
                    <div className="w-12 h-12 rounded-xl bg-white/20 flex items-center justify-center shrink-0">
                        {isApproved ? (
                            <CheckCircle2 className="w-7 h-7" />
                        ) : isPending ? (
                            <AlertTriangle className="w-7 h-7" />
                        ) : (
                            <XCircle className="w-7 h-7" />
                        )}
                    </div>
                    <div>
                        <span className="text-xs uppercase tracking-wider font-semibold opacity-90">
                            {isCheckIn ? "Presensi Masuk (Check-In)" : "Presensi Pulang (Check-Out)"}
                        </span>
                        <h1 className="text-lg font-bold">
                            {isApproved
                                ? "Presensi Disetujui"
                                : isPending
                                    ? "Menunggu Verifikasi Manual"
                                    : "Presensi Ditolak"}
                        </h1>
                    </div>
                </div>

                {/* Detailed Fields */}
                <div className="p-6 space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
                        <div className="bg-slate-50 p-3.5 rounded-xl border border-slate-100 space-y-1">
                            <span className="text-slate-500 flex items-center">
                                <Clock className="w-3.5 h-3.5 mr-1 text-slate-400" />
                                Tanggal Kerja
                            </span>
                            <p className="font-bold text-slate-800">{recordedDate}</p>
                        </div>

                        <div className="bg-slate-50 p-3.5 rounded-xl border border-slate-100 space-y-1">
                            <span className="text-slate-500 flex items-center">
                                <Clock className="w-3.5 h-3.5 mr-1 text-slate-400" />
                                Waktu Tercatat (Server)
                            </span>
                            <p className="font-bold text-slate-800">{recordedTime} WIB</p>
                        </div>

                        <div className="bg-slate-50 p-3.5 rounded-xl border border-slate-100 space-y-1">
                            <span className="text-slate-500 flex items-center">
                                <ShieldCheck className="w-3.5 h-3.5 mr-1 text-slate-400" />
                                Metode Verifikasi
                            </span>
                            <p className="font-bold text-slate-800 capitalize">
                                {record.method === "face"
                                    ? "Biometrik Wajah"
                                    : record.method === "fallback"
                                        ? "Pengajuan Fallback"
                                        : "Manual"}
                            </p>
                        </div>

                        <div className="bg-slate-50 p-3.5 rounded-xl border border-slate-100 space-y-1">
                            <span className="text-slate-500 flex items-center">
                                <MapPin className="w-3.5 h-3.5 mr-1 text-slate-400" />
                                Area Kantor (Geofence)
                            </span>
                            <p
                                className={`font-bold ${record.geofence_status === "inside" ? "text-emerald-700" : "text-rose-600"
                                    }`}
                            >
                                {record.geofence_status === "inside"
                                    ? `Dalam Radius (${record.office_location_name || "Kantor"})`
                                    : "Di Luar Radius Kantor"}
                            </p>
                        </div>
                    </div>

                    {/* Pending Review Notice */}
                    {isPending && (
                        <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 text-xs text-amber-900 space-y-1">
                            <p className="font-bold flex items-center">
                                <AlertTriangle className="w-4 h-4 mr-1.5 text-amber-600" />
                                Dalam Proses Peninjauan
                            </p>
                            <p>Presensi ini sedang menunggu evaluasi dan persetujuan dari tim HR / atasan langsung.</p>
                        </div>
                    )}

                    {/* Review Notes from Admin */}
                    {record.review_note && (
                        <div className="bg-slate-50 border border-slate-200 rounded-xl p-4 text-xs text-slate-700 space-y-1">
                            <span className="font-bold text-slate-900 flex items-center">
                                <FileText className="w-3.5 h-3.5 mr-1 text-slate-500" />
                                Catatan Peninjau / Atasan:
                            </span>
                            <p className="italic text-slate-600">{record.review_note}</p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};
