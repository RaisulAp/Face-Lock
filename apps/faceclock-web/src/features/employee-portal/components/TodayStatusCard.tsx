import React, { useEffect, useState } from "react";
import { CheckCircle2, Clock, AlertTriangle, XCircle, ShieldCheck } from "lucide-react";
import type { AttendanceContextData, EmployeeAttendanceItem } from "../types";

interface TodayStatusCardProps {
    context?: AttendanceContextData;
    isLoading?: boolean;
}

export const TodayStatusCard: React.FC<TodayStatusCardProps> = ({ context, isLoading }) => {
    const [currentTime, setCurrentTime] = useState<string>("");

    useEffect(() => {
        const updateTime = () => {
            const now = new Date();
            setCurrentTime(
                now.toLocaleTimeString("id-ID", {
                    hour: "2-digit",
                    minute: "2-digit",
                    second: "2-digit",
                }),
            );
        };
        updateTime();
        const interval = setInterval(updateTime, 1000);
        return () => clearInterval(interval);
    }, []);

    const formatDate = (dateStr?: string) => {
        if (!dateStr) return "";
        try {
            const date = new Date(dateStr);
            return date.toLocaleDateString("id-ID", {
                weekday: "long",
                day: "numeric",
                month: "long",
                year: "numeric",
            });
        } catch {
            return dateStr;
        }
    };

    const renderRecordItem = (
        label: string,
        record: EmployeeAttendanceItem | null | undefined,
        defaultEmptyText: string,
    ) => {
        if (!record) {
            return (
                <div className="flex items-center justify-between p-3 rounded-lg bg-slate-50 border border-dashed border-slate-200">
                    <div className="flex items-center space-x-3">
                        <Clock className="w-5 h-5 text-slate-400" />
                        <div>
                            <p className="text-xs font-medium text-slate-500">{label}</p>
                            <p className="text-sm font-semibold text-slate-400">{defaultEmptyText}</p>
                        </div>
                    </div>
                    <span className="text-xs px-2 py-0.5 rounded-full bg-slate-100 text-slate-500 font-medium">
                        Belum Tercatat
                    </span>
                </div>
            );
        }

        const timeStr = new Date(record.server_timestamp || record.recorded_at).toLocaleTimeString("id-ID", {
            hour: "2-digit",
            minute: "2-digit",
        });

        const isApproved = record.status === "approved";
        const isPending = record.status === "pending_review";

        return (
            <div
                className={`flex items-center justify-between p-3 rounded-lg border ${isApproved
                        ? "bg-emerald-50/60 border-emerald-200"
                        : isPending
                            ? "bg-amber-50/60 border-amber-200"
                            : "bg-rose-50/60 border-rose-200"
                    }`}
            >
                <div className="flex items-center space-x-3">
                    {isApproved ? (
                        <CheckCircle2 className="w-5 h-5 text-emerald-600" />
                    ) : isPending ? (
                        <AlertTriangle className="w-5 h-5 text-amber-600" />
                    ) : (
                        <XCircle className="w-5 h-5 text-rose-600" />
                    )}
                    <div>
                        <p className="text-xs font-medium text-slate-500">{label}</p>
                        <p className="text-base font-bold text-slate-900">{timeStr} WIB</p>
                    </div>
                </div>

                <div className="flex flex-col items-end space-y-1">
                    <span
                        className={`text-xs px-2 py-0.5 rounded-full font-semibold ${isApproved
                                ? "bg-emerald-100 text-emerald-800"
                                : isPending
                                    ? "bg-amber-100 text-amber-800"
                                    : "bg-rose-100 text-rose-800"
                            }`}
                    >
                        {isApproved ? "Disetujui" : isPending ? "Menunggu Verifikasi" : "Ditolak"}
                    </span>
                    <div className="flex items-center space-x-1.5 text-[11px] text-slate-500">
                        {record.method === "face" && (
                            <span className="flex items-center text-blue-600 font-medium">
                                <ShieldCheck className="w-3 h-3 mr-0.5" /> Biometrik
                            </span>
                        )}
                        {record.method === "fallback" && <span className="text-amber-700 font-medium">Fallback</span>}
                        {record.geofence_status === "inside" && (
                            <span className="text-emerald-700 font-medium">• Dalam Radius</span>
                        )}
                        {record.geofence_status === "outside" && (
                            <span className="text-rose-600 font-medium">• Luar Radius</span>
                        )}
                    </div>
                </div>
            </div>
        );
    };

    if (isLoading) {
        return (
            <div className="bg-white rounded-xl shadow-xs border border-slate-200 p-5 animate-pulse space-y-4">
                <div className="h-6 bg-slate-200 rounded w-1/3"></div>
                <div className="space-y-3">
                    <div className="h-14 bg-slate-100 rounded-lg"></div>
                    <div className="h-14 bg-slate-100 rounded-lg"></div>
                </div>
            </div>
        );
    }

    return (
        <div className="bg-white rounded-xl shadow-xs border border-slate-200 p-5 space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                <div>
                    <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">
                        Status Kehadiran Hari Ini
                    </p>
                    <h3 className="text-sm font-bold text-slate-800">{formatDate(context?.work_date) || "Hari Ini"}</h3>
                </div>
                <div className="text-right">
                    <span className="inline-flex items-center font-mono font-bold text-base text-primary-700 bg-primary-50 px-2.5 py-1 rounded-md border border-primary-200">
                        {currentTime}
                    </span>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {renderRecordItem("Jam Masuk (Check-In)", context?.today_check_in, "Belum Check-In")}
                {renderRecordItem("Jam Pulang (Check-Out)", context?.today_check_out, "Belum Check-Out")}
            </div>

            {context?.work_start_time && context?.work_end_time && (
                <div className="flex items-center justify-between text-xs text-slate-500 pt-1">
                    <span>
                        Jadwal: <strong className="text-slate-700">{context.work_start_time}</strong> -{" "}
                        <strong className="text-slate-700">{context.work_end_time}</strong>
                    </span>
                    {context.late_tolerance_minutes !== undefined && (
                        <span>Toleransi telat: {context.late_tolerance_minutes} menit</span>
                    )}
                </div>
            )}
        </div>
    );
};
