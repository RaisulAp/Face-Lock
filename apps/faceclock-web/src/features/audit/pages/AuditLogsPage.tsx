import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { AuditLogItem, PaginatedResponse } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { Modal } from "../../../components/ui/Modal";
import { Pagination } from "../../../components/data/Pagination";
import { LocalTime } from "../../../components/domain/LocalTime";
import {
    FileText,
    Eye,
} from "lucide-react";

export function AuditLogsPage() {
    const [page, setPage] = useState(1);
    const [action, setAction] = useState("");
    const [resourceType, setResourceType] = useState("");
    const [dateFrom, setDateFrom] = useState("");
    const [dateTo, setDateTo] = useState("");

    const [selectedLog, setSelectedLog] = useState<AuditLogItem | null>(null);
    const [detailOpen, setDetailOpen] = useState(false);

    const { data: logsRes, isLoading } = useQuery<PaginatedResponse<AuditLogItem>>({
        queryKey: ["audit-logs", page, action, resourceType, dateFrom, dateTo],
        queryFn: () => {
            const p = new URLSearchParams();
            p.set("page", String(page));
            p.set("per_page", "15");
            if (action) p.set("action", action);
            if (resourceType) p.set("resource_type", resourceType);
            if (dateFrom) p.set("date_from", dateFrom);
            if (dateTo) p.set("date_to", dateTo);
            return api.get<PaginatedResponse<AuditLogItem>>(`/api/v1/audit-logs?${p.toString()}`);
        },
    });

    return (
        <div className="space-y-6 max-w-7xl mx-auto">
            {/* Header */}
            <div>
                <div className="flex items-center gap-2">
                    <FileText className="w-5 h-5 text-indigo-600" />
                    <h1 className="text-xl font-bold text-gray-900">Jejak Audit Sistem (Audit Logs)</h1>
                </div>
                <p className="text-xs text-gray-500 mt-0.5">
                    Catatan tidak dapat diubah (append-only) dari setiap aktivitas mutasi data dan perubahan konfigurasi sistem.
                </p>
            </div>

            {/* Filter Bar */}
            <Card>
                <CardContent className="py-3">
                    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3">
                        <div>
                            <label className="block text-[10px] font-semibold text-gray-500 mb-1">
                                Aksi (Action)
                            </label>
                            <Input
                                placeholder="e.g. employee.create, setting.update"
                                value={action}
                                onChange={(e) => {
                                    setAction(e.target.value);
                                    setPage(1);
                                }}
                                className="text-xs"
                            />
                        </div>

                        <div>
                            <label className="block text-[10px] font-semibold text-gray-500 mb-1">
                                Tipe Sumber Daya (Resource)
                            </label>
                            <Input
                                placeholder="e.g. employee, setting, user, role"
                                value={resourceType}
                                onChange={(e) => {
                                    setResourceType(e.target.value);
                                    setPage(1);
                                }}
                                className="text-xs"
                            />
                        </div>

                        <div>
                            <label className="block text-[10px] font-semibold text-gray-500 mb-1">
                                Dari Tanggal
                            </label>
                            <Input
                                type="date"
                                value={dateFrom}
                                onChange={(e) => {
                                    setDateFrom(e.target.value);
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
                                value={dateTo}
                                onChange={(e) => {
                                    setDateTo(e.target.value);
                                    setPage(1);
                                }}
                                className="text-xs"
                            />
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Audit Logs Table */}
            <Card>
                <CardHeader>
                    <CardTitle className="text-xs">
                        Daftar Jejak Audit ({logsRes?.meta?.total || 0})
                    </CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                    <div className="overflow-x-auto">
                        <table className="w-full text-xs text-left">
                            <thead className="bg-gray-50 text-gray-600 border-b border-gray-200">
                                <tr>
                                    <th className="px-4 py-3 font-semibold">Waktu Kejadian</th>
                                    <th className="px-4 py-3 font-semibold">Aksi</th>
                                    <th className="px-4 py-3 font-semibold">Sumber Daya</th>
                                    <th className="px-4 py-3 font-semibold">Pelaku (Actor)</th>
                                    <th className="px-4 py-3 font-semibold">Alamat IP</th>
                                    <th className="px-4 py-3 font-semibold text-right">Detail</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-100">
                                {isLoading ? (
                                    <tr>
                                        <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                                            Memuat data jejak audit...
                                        </td>
                                    </tr>
                                ) : !logsRes?.data || logsRes.data.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                                            Tidak ada catatan audit yang cocok.
                                        </td>
                                    </tr>
                                ) : (
                                    logsRes.data.map((log) => (
                                        <tr key={log.id} className="hover:bg-gray-50/50">
                                            <td className="px-4 py-3 text-gray-600 whitespace-nowrap">
                                                <LocalTime value={log.created_at} format="datetime" />
                                            </td>
                                            <td className="px-4 py-3">
                                                <Badge variant="neutral" size="sm">
                                                    {log.action}
                                                </Badge>
                                            </td>
                                            <td className="px-4 py-3">
                                                <span className="font-semibold text-gray-800">{log.resource_type}</span>
                                                {log.resource_id && (
                                                    <span className="text-[10px] text-gray-400 block font-mono truncate max-w-[140px]">
                                                        {log.resource_id}
                                                    </span>
                                                )}
                                            </td>
                                            <td className="px-4 py-3 font-mono text-[11px] text-gray-700">
                                                {log.user_email || log.actor_user_id || log.user_id || "System"}
                                            </td>
                                            <td className="px-4 py-3 font-mono text-[11px] text-gray-500">
                                                {log.ip || log.ip_address || "—"}
                                            </td>
                                            <td className="px-4 py-3 text-right">
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() => {
                                                        setSelectedLog(log);
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

                    {logsRes?.meta && logsRes.meta.total_pages > 1 && (
                        <div className="p-4 border-t border-gray-100">
                            <Pagination
                                page={page}
                                totalPages={logsRes.meta.total_pages}
                                onPageChange={setPage}
                            />
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Modal: Audit Log Payload Inspection */}
            <Modal
                open={detailOpen}
                onClose={() => setDetailOpen(false)}
                title="Detail Aktivitas Audit Log"
            >
                {selectedLog && (
                    <div className="space-y-4 text-xs">
                        <div className="p-3 bg-gray-50 border border-gray-200 rounded-xl space-y-1.5 font-mono text-[11px]">
                            <div className="flex justify-between">
                                <span className="text-gray-400">Aksi:</span>
                                <span className="font-bold text-indigo-700">{selectedLog.action}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-400">Resource:</span>
                                <span className="text-gray-800">
                                    {selectedLog.resource_type} ({selectedLog.resource_id || "N/A"})
                                </span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-400">Actor ID:</span>
                                <span className="text-gray-800">
                                    {selectedLog.actor_user_id || selectedLog.user_id || "System"}
                                </span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-400">Request ID:</span>
                                <span className="text-gray-800">{selectedLog.request_id || "—"}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-400">Alamat IP:</span>
                                <span className="text-gray-800">{selectedLog.ip || selectedLog.ip_address || "—"}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-400">User Agent:</span>
                                <span className="text-gray-800 max-w-xs truncate" title={selectedLog.user_agent || ""}>
                                    {selectedLog.user_agent || "—"}
                                </span>
                            </div>
                        </div>

                        {/* Payload JSON */}
                        <div>
                            <span className="text-[10px] text-gray-400 uppercase font-semibold block mb-1.5">
                                Payload Perubahan / Parameter (JSON)
                            </span>
                            <pre className="p-3 bg-gray-900 text-gray-100 rounded-xl overflow-x-auto text-[11px] font-mono leading-relaxed max-h-60">
                                {JSON.stringify(
                                    selectedLog.metadata || selectedLog.payload || {},
                                    null,
                                    2
                                )}
                            </pre>
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
