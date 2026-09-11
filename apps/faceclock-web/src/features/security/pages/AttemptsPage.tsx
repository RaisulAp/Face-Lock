import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { AttendanceAttemptItem, PaginatedResponse } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { Modal } from "../../../components/ui/Modal";
import { Pagination } from "../../../components/data/Pagination";
import { LocalTime } from "../../../components/domain/LocalTime";
import { HintList } from "../../../components/domain/HintList";
import {
    ShieldAlert,
    Eye,
} from "lucide-react";

export function AttemptsPage() {
    const [page, setPage] = useState(1);
    const [outcome, setOutcome] = useState("");
    const [type, setType] = useState("");
    const [startDate, setStartDate] = useState("");
    const [endDate, setEndDate] = useState("");

    const [selectedAttempt, setSelectedAttempt] = useState<AttendanceAttemptItem | null>(null);
    const [detailOpen, setDetailOpen] = useState(false);

    const { data: attemptsRes, isLoading } = useQuery<PaginatedResponse<AttendanceAttemptItem>>({
        queryKey: ["attendance-attempts", page, outcome, type, startDate, endDate],
        queryFn: () => {
            const p = new URLSearchParams();
            p.set("page", String(page));
            p.set("per_page", "15");
            if (outcome) p.set("outcome", outcome);
            if (type) p.set("type", type);
            if (startDate) p.set("start_date", startDate);
            if (endDate) p.set("end_date", endDate);
            return api.get<PaginatedResponse<AttendanceAttemptItem>>(
                `/api/v1/attendances/attempts?${p.toString()}`
            );
        },
    });

    const getOutcomeBadge = (out?: string) => {
        switch (out) {
            case "matched":
                return <Badge variant="success" size="sm">MATCHED</Badge>;
            case "below_threshold":
                return <Badge variant="danger" size="sm">BELOW THRESHOLD</Badge>;
            case "geofence_rejected":
                return <Badge variant="warning" size="sm">GEOFENCE REJECTED</Badge>;
            case "face_not_usable":
                return <Badge variant="danger" size="sm">FACE NOT USABLE</Badge>;
            case "duplicate_photo":
                return <Badge variant="danger" size="sm">DUPLICATE PHOTO</Badge>;
            case "rate_limited":
                return <Badge variant="danger" size="sm">RATE LIMITED</Badge>;
            case "liveness_failed":
                return <Badge variant="danger" size="sm">LIVENESS FAILED</Badge>;
            case "location_mocked":
                return <Badge variant="danger" size="sm">MOCKED GPS</Badge>;
            default:
                return <Badge variant="neutral" size="sm">{out || "UNKNOWN"}</Badge>;
        }
    };

    return (
        <div className="space-y-6 max-w-7xl mx-auto">
            {/* Header */}
            <div>
                <div className="flex items-center gap-2">
                    <ShieldAlert className="w-5 h-5 text-indigo-600" />
                    <h1 className="text-xl font-bold text-gray-900">Log Percobaan Presensi (Security Telemetry)</h1>
                </div>
                <p className="text-xs text-gray-500 mt-0.5">
                    Audit telemetri setiap percobaan check-in/check-out oleh karyawan, kegagalan pencocokan wajah, dan anomali lokasi.
                </p>
            </div>

            {/* Filter Bar */}
            <Card>
                <CardContent className="py-3">
                    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3">
                        <div>
                            <label className="block text-[10px] font-semibold text-gray-500 mb-1">
                                Hasil Verifikasi (Outcome)
                            </label>
                            <select
                                value={outcome}
                                onChange={(e) => {
                                    setOutcome(e.target.value);
                                    setPage(1);
                                }}
                                className="w-full px-2.5 py-1.5 text-xs border border-gray-300 rounded-lg bg-white text-gray-700"
                            >
                                <option value="">Semua Hasil</option>
                                <option value="matched">Matched (Berhasil)</option>
                                <option value="below_threshold">Below Threshold (Wajah Tidak Cocok)</option>
                                <option value="geofence_rejected">Geofence Rejected (Di Luar Radius)</option>
                                <option value="face_not_usable">Face Not Usable (Kualitas Rendah)</option>
                                <option value="duplicate_photo">Duplicate Photo (Anti-Spoofing)</option>
                                <option value="rate_limited">Rate Limited (B12)</option>
                                <option value="liveness_failed">Liveness Failed</option>
                                <option value="location_mocked">Mocked GPS</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-[10px] font-semibold text-gray-500 mb-1">
                                Tipe Presensi
                            </label>
                            <select
                                value={type}
                                onChange={(e) => {
                                    setType(e.target.value);
                                    setPage(1);
                                }}
                                className="w-full px-2.5 py-1.5 text-xs border border-gray-300 rounded-lg bg-white text-gray-700"
                            >
                                <option value="">Semua Tipe</option>
                                <option value="check_in">Check-In (Masuk)</option>
                                <option value="check_out">Check-Out (Pulang)</option>
                            </select>
                        </div>

                        <div>
                            <label className="block text-[10px] font-semibold text-gray-500 mb-1">
                                Dari Tanggal
                            </label>
                            <Input
                                type="date"
                                value={startDate}
                                onChange={(e) => {
                                    setStartDate(e.target.value);
                                    setPage(1);
                                }}
                                className="text-xs"
                            />
                        </div>

                        <div>
                            <label className="block text-[10px] font-semibold text-gray-500 mb-1">
                                Sampai Tanggal
                            </label>
                            <Input
                                type="date"
                                value={endDate}
                                onChange={(e) => {
                                    setEndDate(e.target.value);
                                    setPage(1);
                                }}
                                className="text-xs"
                            />
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Attempts Table */}
            <Card>
                <CardHeader>
                    <CardTitle className="text-xs">
                        Daftar Percobaan Presensi ({attemptsRes?.meta?.total || 0})
                    </CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                    <div className="overflow-x-auto">
                        <table className="w-full text-xs text-left">
                            <thead className="bg-gray-50 text-gray-600 border-b border-gray-200">
                                <tr>
                                    <th className="px-4 py-3 font-semibold">Waktu Server</th>
                                    <th className="px-4 py-3 font-semibold">Karyawan</th>
                                    <th className="px-4 py-3 font-semibold">Tipe</th>
                                    <th className="px-4 py-3 font-semibold">Hasil (Outcome)</th>
                                    <th className="px-4 py-3 font-semibold">Skor Kemiripan</th>
                                    <th className="px-4 py-3 font-semibold">Geofence</th>
                                    <th className="px-4 py-3 font-semibold text-right">Detail</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-100">
                                {isLoading ? (
                                    <tr>
                                        <td colSpan={7} className="px-4 py-8 text-center text-gray-400">
                                            Memuat telemetri percobaan presensi...
                                        </td>
                                    </tr>
                                ) : !attemptsRes?.data || attemptsRes.data.length === 0 ? (
                                    <tr>
                                        <td colSpan={7} className="px-4 py-8 text-center text-gray-400">
                                            Tidak ada telemetri percobaan yang cocok.
                                        </td>
                                    </tr>
                                ) : (
                                    attemptsRes.data.map((att) => (
                                        <tr key={att.id} className="hover:bg-gray-50/50">
                                            <td className="px-4 py-3 text-gray-600 whitespace-nowrap">
                                                <LocalTime value={att.server_timestamp} format="datetime" />
                                            </td>
                                            <td className="px-4 py-3">
                                                <div className="font-semibold text-gray-900">
                                                    {att.employee_name || "Tidak Terdaftar"}
                                                </div>
                                                <div className="text-[10px] text-gray-400 font-mono">
                                                    {att.employee_number || "—"}
                                                </div>
                                            </td>
                                            <td className="px-4 py-3">
                                                <Badge variant="outline" size="sm">
                                                    {att.type === "check_in" ? "Check-In" : "Check-Out"}
                                                </Badge>
                                            </td>
                                            <td className="px-4 py-3 whitespace-nowrap">
                                                {getOutcomeBadge(att.outcome)}
                                            </td>
                                            <td className="px-4 py-3 font-mono text-gray-700">
                                                {att.matched_similarity != null ? (
                                                    <span>
                                                        {(att.matched_similarity * 100).toFixed(1)}%
                                                        <span className="text-[10px] text-gray-400 ml-1">
                                                            / {((att.threshold_used ?? 0.72) * 100).toFixed(0)}%
                                                        </span>
                                                    </span>
                                                ) : (
                                                    <span className="text-gray-400">—</span>
                                                )}
                                            </td>
                                            <td className="px-4 py-3">
                                                <span
                                                    className={
                                                        att.geofence_status === "inside"
                                                            ? "text-emerald-700 font-medium"
                                                            : att.geofence_status === "outside"
                                                                ? "text-rose-700 font-medium"
                                                                : "text-gray-500"
                                                    }
                                                >
                                                    {att.geofence_status || "unknown"}
                                                    {att.distance_meter != null && (
                                                        <span className="text-[10px] text-gray-400 block font-mono">
                                                            {Math.round(att.distance_meter)}m
                                                        </span>
                                                    )}
                                                </span>
                                            </td>
                                            <td className="px-4 py-3 text-right">
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() => {
                                                        setSelectedAttempt(att);
                                                        setDetailOpen(true);
                                                    }}
                                                >
                                                    <Eye className="w-3.5 h-3.5 mr-1" />
                                                    Inspeksi
                                                </Button>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {attemptsRes?.meta && attemptsRes.meta.total_pages > 1 && (
                        <div className="p-4 border-t border-gray-100">
                            <Pagination
                                page={page}
                                totalPages={attemptsRes.meta.total_pages}
                                onPageChange={setPage}
                            />
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Modal: Telemetry Details */}
            <Modal
                open={detailOpen}
                onClose={() => setDetailOpen(false)}
                title="Detail Telemetri Percobaan Presensi"
            >
                {selectedAttempt && (
                    <div className="space-y-4 text-xs">
                        {/* Status overview */}
                        <div className="p-3 bg-gray-50 border border-gray-200 rounded-xl space-y-1">
                            <div className="flex items-center justify-between">
                                <span className="font-semibold text-gray-700">Hasil Akhir:</span>
                                {getOutcomeBadge(selectedAttempt.outcome)}
                            </div>
                            {selectedAttempt.failure_reason && (
                                <p className="text-rose-700 font-medium pt-1">
                                    Alasan Kegagalan: {selectedAttempt.failure_reason}
                                </p>
                            )}
                        </div>

                        {/* Biometric & Geofence metrics */}
                        <div className="grid grid-cols-2 gap-3">
                            <div className="p-3 bg-gray-50 rounded-lg border border-gray-100 space-y-1">
                                <span className="text-[10px] text-gray-400 uppercase font-semibold block">
                                    Biometrik Wajah
                                </span>
                                <div className="text-gray-700">
                                    Kemiripan:{" "}
                                    <strong>
                                        {selectedAttempt.matched_similarity != null
                                            ? `${(selectedAttempt.matched_similarity * 100).toFixed(1)}%`
                                            : "—"}
                                    </strong>
                                </div>
                                <div className="text-gray-700">
                                    Threshold:{" "}
                                    <strong>
                                        {selectedAttempt.threshold_used != null
                                            ? `${(selectedAttempt.threshold_used * 100).toFixed(1)}%`
                                            : "—"}
                                    </strong>
                                </div>
                                <div className="text-gray-700">
                                    Model AI:{" "}
                                    <strong className="font-mono">{selectedAttempt.model_version || "—"}</strong>
                                </div>
                                <div className="text-gray-700">
                                    Skor Kualitas:{" "}
                                    <strong>{selectedAttempt.quality_score ?? "—"}</strong>
                                </div>
                            </div>

                            <div className="p-3 bg-gray-50 rounded-lg border border-gray-100 space-y-1">
                                <span className="text-[10px] text-gray-400 uppercase font-semibold block">
                                    Lokasi Geofence
                                </span>
                                <div className="text-gray-700">
                                    Status: <strong>{selectedAttempt.geofence_status || "—"}</strong>
                                </div>
                                <div className="text-gray-700">
                                    Jarak ke Kantor:{" "}
                                    <strong>
                                        {selectedAttempt.distance_meter != null
                                            ? `${Math.round(selectedAttempt.distance_meter)} meter`
                                            : "—"}
                                    </strong>
                                </div>
                                <div className="text-gray-700">
                                    Fallback Diizinkan:{" "}
                                    <strong>{selectedAttempt.allow_fallback ? "Ya" : "Tidak"}</strong>
                                </div>
                            </div>
                        </div>

                        {/* Hints list */}
                        {selectedAttempt.hints && selectedAttempt.hints.length > 0 && (
                            <div>
                                <span className="text-[10px] text-gray-400 uppercase font-semibold block mb-1.5">
                                    Petunjuk Kualitas Wajah (Hints)
                                </span>
                                <HintList hints={selectedAttempt.hints} />
                            </div>
                        )}

                        {/* Network / Client info */}
                        <div className="p-3 bg-gray-50 rounded-lg border border-gray-100 space-y-1 font-mono text-[11px] text-gray-600">
                            <div className="flex justify-between">
                                <span className="text-gray-400">Request ID:</span>
                                <span className="text-gray-800">{selectedAttempt.request_id || "—"}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-400">Alamat IP:</span>
                                <span className="text-gray-800">{selectedAttempt.ip || "—"}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-400">User Agent:</span>
                                <span className="text-gray-800 max-w-xs truncate" title={selectedAttempt.user_agent || ""}>
                                    {selectedAttempt.user_agent || "—"}
                                </span>
                            </div>
                        </div>

                        <div className="flex justify-end pt-2 border-t border-gray-100">
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setDetailOpen(false)}
                            >
                                Tutup
                            </Button>
                        </div>
                    </div>
                )}
            </Modal>
        </div>
    );
}
