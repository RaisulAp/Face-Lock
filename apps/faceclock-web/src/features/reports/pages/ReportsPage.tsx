import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { downloadFile } from "../../../lib/download";
import { useToast } from "../../../components/ui/Toast";
import type { AttendanceSummary, Attendance, SuccessEnvelope } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Select } from "../../../components/ui/Select";
import { DataTable } from "../../../components/data/DataTable";
import type { Column } from "../../../components/data/DataTable";
import { AttendanceStatusBadge } from "../../../components/domain/AttendanceStatusBadge";
import { LocalTime } from "../../../components/domain/LocalTime";
import { BarChart3, Download, Calendar, Users, Building, FileSpreadsheet } from "lucide-react";

export function ReportsPage() {
  const toast = useToast();
  const todayStr = new Date().toISOString().split("T")[0];

  const [activeTab, setActiveTab] = useState<"summary" | "export">("summary");

  // Summary Tab State
  const [summaryDate, setSummaryDate] = useState(todayStr);

  // Export Tab State
  const [fromDate, setFromDate] = useState(todayStr);
  const [toDate, setToDate] = useState(todayStr);
  const [statusFilter, setStatusFilter] = useState("");
  const [isExporting, setIsExporting] = useState(false);

  // 1. Fetch Summary (Endpoint #71)
  const { data: summary, isLoading: isSummaryLoading } = useQuery<AttendanceSummary>({
    queryKey: ["attendances", "summary", summaryDate],
    queryFn: () => api.get<AttendanceSummary>(`/api/v1/attendances/summary?date=${summaryDate}`),
    enabled: activeTab === "summary",
  });

  // 2. Fetch Preview for Export Tab (Endpoint #60)
  const exportPreviewParams = new URLSearchParams();
  if (fromDate) exportPreviewParams.set("from_date", fromDate);
  if (toDate) exportPreviewParams.set("to_date", toDate);
  if (statusFilter) exportPreviewParams.set("status", statusFilter);
  exportPreviewParams.set("per_page", "50");

  const { data: exportPreviewData, isLoading: isPreviewLoading } = useQuery<SuccessEnvelope<Attendance[]>>({
    queryKey: ["attendances", "export-preview", fromDate, toDate, statusFilter],
    queryFn: () => api.getWithMeta<Attendance[]>(`/api/v1/attendances?${exportPreviewParams.toString()}`),
    enabled: activeTab === "export",
  });

  const previewAttendances = exportPreviewData?.data ?? [];

  // Handle CSV Download (Endpoint #72)
  const handleExportCsv = async () => {
    setIsExporting(true);
    try {
      const params = new URLSearchParams();
      if (fromDate) params.set("from_date", fromDate);
      if (toDate) params.set("to_date", toDate);
      if (statusFilter) params.set("status", statusFilter);

      const filename = `laporan-absensi_${fromDate || "awal"}_sampai_${toDate || "akhir"}.csv`;
      const { blob } = await api.getBlob(`/api/v1/attendances/export?${params.toString()}`);
      downloadFile(blob, filename);

      toast.show({
        type: "success",
        title: "Ekspor Berhasil",
        message: `File CSV berhasil diunduh (${filename}).`,
      });
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Mengunduh",
        message: "Terjadi kesalahan saat memproses ekspor data CSV.",
      });
    } finally {
      setIsExporting(false);
    }
  };

  const previewColumns: Column<Attendance>[] = [
    {
      key: "employee",
      header: "Karyawan",
      render: (row) => (
        <div>
          <p className="font-semibold text-gray-900">{row.employee_name || "Karyawan"}</p>
          <span className="text-[11px] text-gray-400 font-mono">{row.employee_id.slice(0, 8)}...</span>
        </div>
      ),
    },
    {
      key: "recorded_at",
      header: "Waktu Absen",
      render: (row) => <LocalTime value={row.recorded_at} format="datetime" />,
    },
    {
      key: "status",
      header: "Status",
      render: (row) => <AttendanceStatusBadge status={row.status} />,
    },
    {
      key: "similarity",
      header: "Skor Wajah",
      render: (row) => (
        <span className="font-mono text-xs">
          {row.similarity_score != null ? `${(row.similarity_score * 100).toFixed(1)}%` : "-"}
        </span>
      ),
    },
    {
      key: "geofence",
      header: "Jarak GPS",
      render: (row) => (
        <span className="text-xs text-gray-600 font-mono">
          {row.distance_meters != null ? `${Math.round(row.distance_meters)} m` : "-"}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <div className="flex items-center gap-2">
          <BarChart3 className="w-5 h-5 text-indigo-600" />
          <h1 className="text-xl font-bold text-gray-900">Laporan & Rekapitulasi Kehadiran</h1>
        </div>
        <p className="text-xs text-gray-500 mt-0.5">
          Agregasi statistik absensi per tanggal dan ekspor data presensi ke format CSV.
        </p>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-200">
        <button
          type="button"
          onClick={() => setActiveTab("summary")}
          className={`px-4 py-2.5 text-xs font-semibold border-b-2 cursor-pointer transition-colors ${
            activeTab === "summary"
              ? "border-indigo-600 text-indigo-600"
              : "border-transparent text-gray-500 hover:text-gray-900"
          }`}
        >
          <div className="flex items-center gap-2">
            <Calendar className="w-4 h-4" />
            <span>Rekapitulasi Harian (#71)</span>
          </div>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab("export")}
          className={`px-4 py-2.5 text-xs font-semibold border-b-2 cursor-pointer transition-colors ${
            activeTab === "export"
              ? "border-indigo-600 text-indigo-600"
              : "border-transparent text-gray-500 hover:text-gray-900"
          }`}
        >
          <div className="flex items-center gap-2">
            <FileSpreadsheet className="w-4 h-4" />
            <span>Ekspor CSV Data Lengkap (#72)</span>
          </div>
        </button>
      </div>

      {/* TAB 1: SUMMARY */}
      {activeTab === "summary" && (
        <div className="space-y-6">
          {/* Date Selector */}
          <div className="flex items-center gap-3 bg-white p-4 rounded-xl border border-gray-200/80 shadow-2xs">
            <span className="text-xs font-semibold text-gray-700">Pilih Tanggal:</span>
            <Input
              type="date"
              value={summaryDate}
              onChange={(e) => setSummaryDate(e.target.value)}
              className="w-44"
            />
          </div>

          {/* Metric Summary Cards */}
          <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
            <Card padding="sm" className="bg-slate-50 border-slate-200">
              <CardContent className="p-3">
                <span className="text-[11px] font-semibold text-gray-500">Total Hadir</span>
                <p className="text-xl font-bold text-gray-900 mt-1">
                  {isSummaryLoading ? "..." : (summary?.total_present ?? 0)}
                </p>
              </CardContent>
            </Card>

            <Card padding="sm" className="bg-emerald-50 border-emerald-200">
              <CardContent className="p-3">
                <span className="text-[11px] font-semibold text-emerald-700">Valid</span>
                <p className="text-xl font-bold text-emerald-700 mt-1">
                  {isSummaryLoading ? "..." : (summary?.valid_count ?? 0)}
                </p>
              </CardContent>
            </Card>

            <Card padding="sm" className="bg-amber-50 border-amber-200">
              <CardContent className="p-3">
                <span className="text-[11px] font-semibold text-amber-700">Perlu Review</span>
                <p className="text-xl font-bold text-amber-700 mt-1">
                  {isSummaryLoading ? "..." : (summary?.pending_count ?? 0)}
                </p>
              </CardContent>
            </Card>

            <Card padding="sm" className="bg-rose-50 border-rose-200">
              <CardContent className="p-3">
                <span className="text-[11px] font-semibold text-rose-700">Ditolak</span>
                <p className="text-xl font-bold text-rose-700 mt-1">
                  {isSummaryLoading ? "..." : (summary?.rejected_count ?? 0)}
                </p>
              </CardContent>
            </Card>

            <Card padding="sm" className="bg-indigo-50 border-indigo-200">
              <CardContent className="p-3">
                <span className="text-[11px] font-semibold text-indigo-700">Terlambat</span>
                <p className="text-xl font-bold text-indigo-700 mt-1">
                  {isSummaryLoading ? "..." : (summary?.late_count ?? 0)}
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Department Breakdown Table */}
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Building className="w-4 h-4 text-indigo-600" />
                <CardTitle className="text-xs">Kehadiran per Departemen ({summaryDate})</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              {isSummaryLoading ? (
                <p className="text-center py-6 text-xs text-gray-400">Memuat rekap...</p>
              ) : !summary?.by_department || Object.keys(summary.by_department).length === 0 ? (
                <p className="text-center py-6 text-xs text-gray-400">
                  Tidak ada data absensi untuk tanggal {summaryDate}.
                </p>
              ) : (
                <div className="divide-y divide-gray-100">
                  {Object.entries(summary.by_department).map(([dept, count]) => (
                    <div key={dept} className="flex items-center justify-between py-2.5 text-xs">
                      <div className="flex items-center gap-2">
                        <Users className="w-3.5 h-3.5 text-gray-400" />
                        <span className="font-semibold text-gray-800">{dept || "Umum"}</span>
                      </div>
                      <span className="font-bold text-gray-900 bg-gray-100 px-2 py-0.5 rounded-full">
                        {count} orang
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* TAB 2: EXPORT CSV */}
      {activeTab === "export" && (
        <div className="space-y-6">
          {/* Controls Bar */}
          <div className="bg-white p-4 rounded-xl border border-gray-200/80 shadow-2xs flex flex-wrap items-center justify-between gap-4">
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold text-gray-700">Periode:</span>
                <Input
                  type="date"
                  value={fromDate}
                  onChange={(e) => setFromDate(e.target.value)}
                  className="w-36"
                />
                <span className="text-xs text-gray-400">s/d</span>
                <Input
                  type="date"
                  value={toDate}
                  onChange={(e) => setToDate(e.target.value)}
                  className="w-36"
                />
              </div>

              <div className="w-40">
                <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
                  <option value="">Semua Status</option>
                  <option value="valid">Valid</option>
                  <option value="pending">Pending</option>
                  <option value="rejected">Ditolak</option>
                </Select>
              </div>
            </div>

            <Button
              variant="primary"
              size="sm"
              isLoading={isExporting}
              onClick={handleExportCsv}
              className="bg-emerald-600 hover:bg-emerald-700"
            >
              <Download className="w-4 h-4 mr-1.5" />
              Unduh CSV
            </Button>
          </div>

          {/* Preview Table */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-xs font-bold text-gray-700 uppercase tracking-wider">
                Pratinjau Data (Maks. 50 Baris)
              </h3>
              <span className="text-xs text-gray-400">{previewAttendances.length} baris ditampilkan</span>
            </div>

            <DataTable
              columns={previewColumns}
              data={previewAttendances}
              keyExtractor={(item) => item.id}
              isLoading={isPreviewLoading}
              emptyTitle="Tidak Ada Rekaman"
              emptyDescription="Sesuaikan filter rentang tanggal di atas untuk melihat data."
            />
          </div>
        </div>
      )}
    </div>
  );
}
