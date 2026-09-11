import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../../../lib/api";
import type { AttendanceSummary, Attendance, FaceQualityStatusResponse } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { AttendanceStatusBadge } from "../../../components/domain/AttendanceStatusBadge";
import { AuthImage } from "../../../components/media/AuthImage";
import { LocalTime } from "../../../components/domain/LocalTime";
import {
    Users,
    CheckCircle2,
    ClockAlert,
    XCircle,
    Activity,
    ArrowRight,
    Sparkles,
} from "lucide-react";

export function DashboardPage() {
    const todayStr = new Date().toISOString().split("T")[0];

    // 1. Fetch attendance summary for today (#71)
    const { data: summary, isLoading: isSummaryLoading } = useQuery<AttendanceSummary>({
        queryKey: ["attendances", "summary", todayStr],
        queryFn: () => api.get<AttendanceSummary>(`/api/v1/attendances/summary?date=${todayStr}`),
        refetchInterval: 30_000,
    });

    // 2. Fetch face quality status (#73)
    const { data: qualityStatus, isLoading: isQualityLoading } = useQuery<FaceQualityStatusResponse>({
        queryKey: ["settings", "face-quality-status"],
        queryFn: () => api.get<FaceQualityStatusResponse>("/api/v1/settings/face-quality-status"),
        refetchInterval: 60_000,
    });

    // 3. Fetch recent attendances
    const { data: recentAttendances, isLoading: isRecentLoading } = useQuery<Attendance[]>({
        queryKey: ["attendances", "recent"],
        queryFn: () => api.get<Attendance[]>("/api/v1/attendances?per_page=5"),
        refetchInterval: 15_000,
    });

    return (
        <div className="space-y-6">
            {/* Header Banner */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                <div>
                    <h1 className="text-xl font-bold text-gray-900 tracking-tight">
                        Ringkasan Operasional Kehadiran
                    </h1>
                    <p className="text-xs text-gray-500 mt-0.5">
                        Pantau kehadiran karyawan dan status verifikasi biometrik hari ini ({todayStr}).
                    </p>
                </div>

                {/* Live Engine Status Indicator */}
                <div className="flex items-center gap-2 bg-white px-3 py-1.5 rounded-xl border border-gray-200/80 shadow-2xs">
                    <Activity className="w-3.5 h-3.5 text-indigo-600" />
                    <span className="text-[11px] font-medium text-gray-600">Engine AI:</span>
                    {isQualityLoading ? (
                        <span className="text-[11px] text-gray-400">Memeriksa...</span>
                    ) : qualityStatus?.inference_ready ? (
                        <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-600">
                            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                            Siap ({qualityStatus.model_name})
                        </span>
                    ) : (
                        <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-rose-600">
                            <span className="w-2 h-2 rounded-full bg-rose-500" />
                            Offline
                        </span>
                    )}
                </div>
            </div>

            {/* Metric Cards Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {/* Total Hadir */}
                <Card className="border-l-4 border-l-indigo-500">
                    <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold text-gray-500">Total Hadir</span>
                            <Users className="w-4 h-4 text-indigo-500" />
                        </div>
                        <p className="text-2xl font-bold text-gray-900 mt-2">
                            {isSummaryLoading ? "..." : summary?.total_present ?? 0}
                        </p>
                        <p className="text-[10px] text-gray-400 mt-1">Hari ini</p>
                    </CardContent>
                </Card>

                {/* Valid */}
                <Card className="border-l-4 border-l-emerald-500">
                    <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold text-emerald-700">Valid</span>
                            <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                        </div>
                        <p className="text-2xl font-bold text-emerald-600 mt-2">
                            {isSummaryLoading ? "..." : summary?.valid_count ?? 0}
                        </p>
                        <p className="text-[10px] text-gray-400 mt-1">Terverifikasi otomatis</p>
                    </CardContent>
                </Card>

                {/* Pending Review */}
                <Card className="border-l-4 border-l-amber-500">
                    <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold text-amber-700">Perlu Review</span>
                            <ClockAlert className="w-4 h-4 text-amber-500" />
                        </div>
                        <p className="text-2xl font-bold text-amber-600 mt-2">
                            {isSummaryLoading ? "..." : summary?.pending_count ?? 0}
                        </p>
                        <p className="text-[10px] text-gray-400 mt-1">Menunggu persetujuan</p>
                    </CardContent>
                </Card>

                {/* Rejected */}
                <Card className="border-l-4 border-l-rose-500">
                    <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold text-rose-700">Ditolak</span>
                            <XCircle className="w-4 h-4 text-rose-500" />
                        </div>
                        <p className="text-2xl font-bold text-rose-600 mt-2">
                            {isSummaryLoading ? "..." : summary?.rejected_count ?? 0}
                        </p>
                        <p className="text-[10px] text-gray-400 mt-1">Gagal verifikasi</p>
                    </CardContent>
                </Card>
            </div>

            {/* Operational Highlights & Actions */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Recent Attendances Table */}
                <div className="lg:col-span-2 space-y-3">
                    <div className="flex items-center justify-between">
                        <h2 className="text-sm font-bold text-gray-900">Aktivitas Absensi Terbaru</h2>
                        <Link
                            to="/attendances"
                            className="text-xs font-semibold text-indigo-600 hover:text-indigo-700 inline-flex items-center gap-1"
                        >
                            Lihat Semua <ArrowRight className="w-3.5 h-3.5" />
                        </Link>
                    </div>

                    <Card>
                        <div className="overflow-x-auto">
                            <table className="w-full text-xs text-left">
                                <thead className="bg-gray-50/80 text-gray-600 border-b border-gray-200/80">
                                    <tr>
                                        <th className="px-4 py-2.5 font-semibold">Foto</th>
                                        <th className="px-4 py-2.5 font-semibold">Karyawan</th>
                                        <th className="px-4 py-2.5 font-semibold">Waktu</th>
                                        <th className="px-4 py-2.5 font-semibold">Status</th>
                                        <th className="px-4 py-2.5 font-semibold text-right">Aksi</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-gray-100">
                                    {isRecentLoading ? (
                                        <tr>
                                            <td colSpan={5} className="px-4 py-6 text-center text-gray-400">
                                                Memuat aktivitas terbaru...
                                            </td>
                                        </tr>
                                    ) : !recentAttendances || recentAttendances.length === 0 ? (
                                        <tr>
                                            <td colSpan={5} className="px-4 py-6 text-center text-gray-400">
                                                Belum ada data absensi tercatat hari ini.
                                            </td>
                                        </tr>
                                    ) : (
                                        recentAttendances.map((item) => (
                                            <tr key={item.id} className="hover:bg-gray-50/50">
                                                <td className="px-4 py-2">
                                                    <AuthImage
                                                        src={`/api/v1/attendances/${item.id}/photo`}
                                                        alt="Absensi"
                                                        className="w-8 h-8 rounded-md object-cover border border-gray-200"
                                                    />
                                                </td>
                                                <td className="px-4 py-2">
                                                    <p className="font-semibold text-gray-900">
                                                        {item.employee_name || "Karyawan"}
                                                    </p>
                                                    <span className="text-[11px] text-gray-400">
                                                        {item.employee_id.slice(0, 8)}...
                                                    </span>
                                                </td>
                                                <td className="px-4 py-2 text-gray-600">
                                                    <LocalTime value={item.recorded_at} format="time" />
                                                </td>
                                                <td className="px-4 py-2">
                                                    <AttendanceStatusBadge status={item.status} />
                                                </td>
                                                <td className="px-4 py-2 text-right">
                                                    <Link
                                                        to={`/attendances/${item.id}`}
                                                        className="text-[11px] font-semibold text-indigo-600 hover:underline"
                                                    >
                                                        Detail
                                                    </Link>
                                                </td>
                                            </tr>
                                        ))
                                    )}
                                </tbody>
                            </table>
                        </div>
                    </Card>
                </div>

                {/* Quick Review & Engine Info Side Card */}
                <div className="space-y-4">
                    <h2 className="text-sm font-bold text-gray-900">Pusat Tindakan Cepat</h2>

                    <Card className="bg-gradient-to-br from-indigo-900 to-slate-900 text-white border-0 shadow-md">
                        <CardHeader>
                            <div className="flex items-center gap-2">
                                <Sparkles className="w-4 h-4 text-amber-400" />
                                <CardTitle className="text-white text-xs">Antrian Butuh Persetujuan</CardTitle>
                            </div>
                        </CardHeader>
                        <CardContent className="space-y-3">
                            <p className="text-xs text-indigo-200 leading-relaxed">
                                Terdapat {summary?.pending_count ?? 0} permohonan kehadiran dengan anomali
                                geofence atau kemiripan wajah sedang menunggu tinjauan supervisor.
                            </p>
                            <Link to="/attendances/pending" className="block">
                                <Button
                                    variant="primary"
                                    size="sm"
                                    className="w-full bg-indigo-500 hover:bg-indigo-400 text-white font-semibold"
                                >
                                    Buka Antrian Review ({summary?.pending_count ?? 0})
                                </Button>
                            </Link>
                        </CardContent>
                    </Card>

                    {/* Department Breakdown */}
                    {summary?.by_department && Object.keys(summary.by_department).length > 0 && (
                        <Card padding="sm">
                            <CardHeader>
                                <CardTitle className="text-xs">Distribusi per Departemen</CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-2">
                                {Object.entries(summary.by_department).map(([dept, count]) => (
                                    <div key={dept} className="flex items-center justify-between text-xs">
                                        <span className="text-gray-600">{dept || "Umum"}</span>
                                        <span className="font-semibold text-gray-900">{count} orang</span>
                                    </div>
                                ))}
                            </CardContent>
                        </Card>
                    )}
                </div>
            </div>
        </div>
    );
}
