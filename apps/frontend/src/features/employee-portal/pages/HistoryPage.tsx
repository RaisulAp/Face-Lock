import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
    Clock,
    CheckCircle2,
    AlertTriangle,
    XCircle,
    Filter,
    ShieldCheck,
    ChevronLeft,
    ChevronRight,
    Loader2,
    Calendar,
} from "lucide-react";
import { employeeApi, type AttendanceHistoryParams } from "../api";
import type { EmployeeAttendanceItem } from "../types";

export const HistoryPage: React.FC = () => {
    const navigate = useNavigate();

    const [filters, setFilters] = useState<AttendanceHistoryParams>({
        page: 1,
        per_page: 10,
        status: "",
        type: "",
        start_date: "",
        end_date: "",
    });

    const { data, isLoading, isPlaceholderData } = useQuery({
        queryKey: ["attendance-history-me", filters],
        queryFn: () => employeeApi.getMyHistory(filters),
        placeholderData: (previousData) => previousData,
    });

    const records = data?.data || [];
    const meta = data?.meta;

    const handleFilterChange = (key: keyof AttendanceHistoryParams, value: string) => {
        setFilters((prev) => ({
            ...prev,
            [key]: value,
            page: 1, // reset page on filter change
        }));
    };

    const formatDateTime = (dateStr: string) => {
        try {
            const d = new Date(dateStr);
            return {
                date: d.toLocaleDateString("id-ID", {
                    day: "numeric",
                    month: "short",
                    year: "numeric",
                }),
                time: d.toLocaleTimeString("id-ID", {
                    hour: "2-digit",
                    minute: "2-digit",
                }),
            };
        } catch {
            return { date: dateStr, time: "" };
        }
    };

    return (
        <div className="max-w-4xl mx-auto space-y-6">
            {/* Header */}
            <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs flex flex-wrap items-center justify-between gap-4">
                <div>
                    <h1 className="text-lg font-bold text-slate-900">Riwayat Presensi Mandiri</h1>
                    <p className="text-xs text-slate-500">Daftar seluruh catatan check-in dan check-out kerja Anda</p>
                </div>
            </div>

            {/* Filter Bar */}
            <div className="bg-white rounded-2xl p-4 border border-slate-200 shadow-xs">
                <div className="flex items-center space-x-2 text-xs font-bold text-slate-700 mb-3">
                    <Filter className="w-3.5 h-3.5" />
                    <span>Filter Riwayat</span>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
                    <div>
                        <label className="block text-[11px] font-semibold text-slate-500 mb-1">Dari Tanggal:</label>
                        <input
                            type="date"
                            value={filters.start_date || ""}
                            onChange={(e) => handleFilterChange("start_date", e.target.value)}
                            className="w-full text-xs rounded-lg border border-slate-300 p-2 focus:outline-none focus:ring-1 focus:ring-primary-500"
                        />
                    </div>

                    <div>
                        <label className="block text-[11px] font-semibold text-slate-500 mb-1">Sampai Tanggal:</label>
                        <input
                            type="date"
                            value={filters.end_date || ""}
                            onChange={(e) => handleFilterChange("end_date", e.target.value)}
                            className="w-full text-xs rounded-lg border border-slate-300 p-2 focus:outline-none focus:ring-1 focus:ring-primary-500"
                        />
                    </div>

                    <div>
                        <label className="block text-[11px] font-semibold text-slate-500 mb-1">Status:</label>
                        <select
                            value={filters.status || ""}
                            onChange={(e) => handleFilterChange("status", e.target.value)}
                            className="w-full text-xs rounded-lg border border-slate-300 p-2 focus:outline-none focus:ring-1 focus:ring-primary-500"
                        >
                            <option value="">Semua Status</option>
                            <option value="approved">Disetujui</option>
                            <option value="pending_review">Menunggu Verifikasi</option>
                            <option value="rejected">Ditolak</option>
                        </select>
                    </div>

                    <div>
                        <label className="block text-[11px] font-semibold text-slate-500 mb-1">Tipe:</label>
                        <select
                            value={filters.type || ""}
                            onChange={(e) => handleFilterChange("type", e.target.value)}
                            className="w-full text-xs rounded-lg border border-slate-300 p-2 focus:outline-none focus:ring-1 focus:ring-primary-500"
                        >
                            <option value="">Semua Tipe</option>
                            <option value="check_in">Check-In</option>
                            <option value="check_out">Check-Out</option>
                        </select>
                    </div>
                </div>
            </div>

            {/* Records List / Table */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-xs overflow-hidden">
                {isLoading ? (
                    <div className="p-12 text-center space-y-3">
                        <Loader2 className="w-8 h-8 animate-spin text-primary-600 mx-auto" />
                        <p className="text-xs text-slate-500">Memuat data presensi...</p>
                    </div>
                ) : records.length === 0 ? (
                    <div className="p-12 text-center space-y-2">
                        <Calendar className="w-10 h-10 text-slate-300 mx-auto" />
                        <p className="text-sm font-semibold text-slate-700">Belum Ada Catatan Presensi</p>
                        <p className="text-xs text-slate-500">
                            Catatan presensi Anda akan tampil di sini setelah melakukan check-in atau check-out.
                        </p>
                    </div>
                ) : (
                    <div className="divide-y divide-slate-100">
                        {records.map((item: EmployeeAttendanceItem) => {
                            const { date, time } = formatDateTime(item.server_timestamp || item.recorded_at);
                            const isApproved = item.status === "approved";
                            const isPending = item.status === "pending_review";
                            const isCheckIn = item.type === "check_in" || item.type === "in";

                            return (
                                <div
                                    key={item.id}
                                    onClick={() => navigate(`/portal/history/${item.id}`)}
                                    className="p-4 sm:p-5 hover:bg-slate-50/70 transition-colors cursor-pointer flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                                >
                                    <div className="flex items-start space-x-3.5">
                                        <div
                                            className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${isApproved
                                                    ? "bg-emerald-100 text-emerald-700"
                                                    : isPending
                                                        ? "bg-amber-100 text-amber-700"
                                                        : "bg-rose-100 text-rose-700"
                                                }`}
                                        >
                                            {isApproved ? (
                                                <CheckCircle2 className="w-5 h-5" />
                                            ) : isPending ? (
                                                <AlertTriangle className="w-5 h-5" />
                                            ) : (
                                                <XCircle className="w-5 h-5" />
                                            )}
                                        </div>

                                        <div className="space-y-1">
                                            <div className="flex items-center space-x-2">
                                                <span className="text-xs font-bold text-slate-900">
                                                    {isCheckIn ? "Check-In (Masuk)" : "Check-Out (Pulang)"}
                                                </span>
                                                <span className="text-xs text-slate-400">•</span>
                                                <span className="text-xs font-medium text-slate-600">{date}</span>
                                            </div>

                                            <div className="flex flex-wrap items-center gap-1.5 text-[11px] text-slate-500">
                                                <span className="flex items-center text-slate-700 font-semibold">
                                                    <Clock className="w-3 h-3 mr-1 text-slate-400" />
                                                    {time} WIB
                                                </span>
                                                <span>•</span>
                                                {item.method === "face" && (
                                                    <span className="inline-flex items-center text-blue-600 font-medium">
                                                        <ShieldCheck className="w-3 h-3 mr-0.5" /> Biometrik
                                                    </span>
                                                )}
                                                {item.method === "fallback" && (
                                                    <span className="text-amber-700 font-medium">Fallback</span>
                                                )}
                                                {item.method === "manual" && (
                                                    <span className="text-slate-600 font-medium">Manual</span>
                                                )}
                                                <span>•</span>
                                                <span
                                                    className={
                                                        item.geofence_status === "inside"
                                                            ? "text-emerald-700 font-medium"
                                                            : "text-rose-600 font-medium"
                                                    }
                                                >
                                                    {item.geofence_status === "inside" ? "Dalam Radius" : "Luar Radius"}
                                                </span>
                                            </div>
                                        </div>
                                    </div>

                                    <div className="flex items-center justify-between sm:justify-end space-x-3 pt-2 sm:pt-0 border-t sm:border-0 border-slate-100">
                                        <span
                                            className={`text-xs px-2.5 py-1 rounded-full font-bold ${isApproved
                                                    ? "bg-emerald-100 text-emerald-800"
                                                    : isPending
                                                        ? "bg-amber-100 text-amber-800"
                                                        : "bg-rose-100 text-rose-800"
                                                }`}
                                        >
                                            {isApproved ? "Disetujui" : isPending ? "Menunggu Verifikasi" : "Ditolak"}
                                        </span>
                                        <ChevronRight className="w-4 h-4 text-slate-400" />
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                )}

                {/* Pagination Footer */}
                {meta && meta.total_pages > 1 && (
                    <div className="p-4 border-t border-slate-100 bg-slate-50 flex items-center justify-between text-xs">
                        <span className="text-slate-500">
                            Menampilkan Halaman <strong>{meta.page}</strong> dari <strong>{meta.total_pages}</strong> (Total{" "}
                            {meta.total} data)
                        </span>

                        <div className="flex items-center space-x-2">
                            <button
                                type="button"
                                disabled={meta.page <= 1 || isPlaceholderData}
                                onClick={() => setFilters((p) => ({ ...p, page: (p.page || 1) - 1 }))}
                                className="p-1.5 rounded-lg border border-slate-300 bg-white text-slate-700 disabled:opacity-40"
                            >
                                <ChevronLeft className="w-4 h-4" />
                            </button>
                            <button
                                type="button"
                                disabled={meta.page >= meta.total_pages || isPlaceholderData}
                                onClick={() => setFilters((p) => ({ ...p, page: (p.page || 1) + 1 }))}
                                className="p-1.5 rounded-lg border border-slate-300 bg-white text-slate-700 disabled:opacity-40"
                            >
                                <ChevronRight className="w-4 h-4" />
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};
