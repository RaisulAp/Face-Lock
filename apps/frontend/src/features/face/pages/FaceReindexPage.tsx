import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useToast } from "../../../components/ui/Toast";
import type { FaceReindexJob } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { LocalTime } from "../../../components/domain/LocalTime";
import { RefreshCw, Play, FileSearch, Clock, StopCircle } from "lucide-react";

interface DryRunResult {
  from_model_version: string;
  to_model_version: string;
  total_count: number;
  affected_employee_count: number;
}

export function FaceReindexPage() {
  const toast = useToast();
  const { t } = useTranslation(["face", "common"]);
  const [targetModel, setTargetModel] = useState("buffalo_l");
  const [dryRunResult, setDryRunResult] = useState<DryRunResult | null>(null);
  const [isDryRunning, setIsDryRunning] = useState(false);

  // Active Job ID to monitor
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  const [isStartingActual, setIsStartingActual] = useState(false);
  const [confirmActualOpen, setConfirmActualOpen] = useState(false);

  // Fetch list of past jobs
  const { data: pastJobsData, refetch: refetchJobs } = useQuery<{ jobs: FaceReindexJob[] } | FaceReindexJob[]>({
    queryKey: ["face", "reindex-jobs"],
    queryFn: () => api.get<{ jobs: FaceReindexJob[] } | FaceReindexJob[]>("/api/v1/face/reindex-jobs"),
  });
  const pastJobs = Array.isArray(pastJobsData)
    ? pastJobsData
    : (pastJobsData as { jobs?: FaceReindexJob[] })?.jobs ?? [];

  // Poll current active job every 3s if pending/running
  const { data: activeJobDetail, refetch: refetchActiveJob } = useQuery<FaceReindexJob>({
    queryKey: ["face", "reindex-job", activeJobId],
    queryFn: () => api.get<FaceReindexJob>(`/api/v1/face/reindex-jobs/${activeJobId}`),
    enabled: Boolean(activeJobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "pending" || status === "running" ? 3000 : false;
    },
  });

  // Handle Dry Run
  const handleDryRun = async () => {
    setIsDryRunning(true);
    try {
      const res = await api.post<DryRunResult>("/api/v1/face/reindex-jobs", {
        to_model_version: targetModel.trim(),
        dry_run: true,
      });

      setDryRunResult(res);
      toast.show({
        type: "info",
        title: "Dry-Run Selesai",
        message: `Ditemukan ${res.total_count} foto referensi pada ${res.affected_employee_count} karyawan terdampak.`,
      });
    } catch {
      toast.show({
        type: "error",
        title: "Dry-Run Gagal",
        message: "Gagal menganalisis data foto referensi wajah.",
      });
    } finally {
      setIsDryRunning(false);
    }
  };

  // Handle Actual Reindex Start
  const handleStartActual = async () => {
    setIsStartingActual(true);
    try {
      const res = await api.post<{ id: string }>("/api/v1/face/reindex-jobs", {
        to_model_version: targetModel.trim(),
        dry_run: false,
      });

      toast.show({
        type: "success",
        title: "Pekerjaan Reindex Dimulai",
        message: "Background worker sedang memproses regenerasi embedding.",
      });

      setConfirmActualOpen(false);
      setActiveJobId(res.id);
      refetchJobs();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Memulai Reindex",
        message: "Pastikan tidak ada pekerjaan reindex lain yang sedang berjalan.",
      });
    } finally {
      setIsStartingActual(false);
    }
  };

  // Handle Cancel Job
  const handleCancelJob = async (jobId: string) => {
    try {
      await api.post(`/api/v1/face/reindex-jobs/${jobId}/cancel`, {});
      toast.show({
        type: "info",
        title: "Pekerjaan Dibatalkan",
        message: "Pekerjaan reindex telah dihentikan.",
      });
      refetchActiveJob();
      refetchJobs();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Membatalkan",
        message: "Pekerjaan mungkin telah selesai atau gagal.",
      });
    }
  };

  const isRunning = activeJobDetail?.status === "pending" || activeJobDetail?.status === "running";

  const progressPercent = activeJobDetail?.total_count
    ? Math.round(((activeJobDetail.processed_count || 0) / activeJobDetail.total_count) * 100)
    : 0;

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      {/* Header */}
      <div>
        <div className="flex items-center gap-2">
          <RefreshCw className="w-5 h-5 text-indigo-600" />
          <h1 className="text-xl font-bold text-gray-900">{t("reindex.title", { ns: "face" })}</h1>
        </div>
        <p className="text-xs text-gray-500 mt-0.5">
          {t("reindex.subtitle", { ns: "face" })}
        </p>
      </div>

      {/* Warning Banner */}
      <div className="p-4 bg-amber-50 border border-amber-300 rounded-xl flex items-start gap-3 text-amber-900 text-xs">
        <Clock className="w-5 h-5 text-amber-600 shrink-0 mt-0.5" />
        <div className="space-y-1">
          <strong className="font-semibold block">Aturan Keamanan Sistem & Non-Blocking Reindex:</strong>
          <ul className="list-disc list-inside space-y-0.5 text-amber-800">
            <li>
              <strong>Dry-Run Wajib:</strong> Anda harus menjalankan simulasi (dry-run) terlebih dahulu untuk
              melihat dampak sebelum eksekusi aktual diaktifkan.
            </li>
            <li>
              <strong>Selama proses berjalan:</strong> Karyawan tidak dapat melakukan pendaftaran wajah baru
              (HTTP 409 REINDEX_IN_PROGRESS).
            </li>
            <li>
              Setelah selesai, periksa kembali kecocokan <em>face.similarity_threshold</em> di menu
              Pengaturan.
            </li>
          </ul>
        </div>
      </div>

      {/* Control Card: Dry-Run & Trigger */}
      <Card>
        <CardHeader>
          <CardTitle className="text-xs">Konfigurasi Target Model</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Target Versi Model AI</label>
              <Input
                value={targetModel}
                onChange={(e) => {
                  setTargetModel(e.target.value);
                  setDryRunResult(null); // Reset dry run if target changes
                }}
                placeholder="buffalo_l"
              />
            </div>

            <div className="flex items-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleDryRun}
                isLoading={isDryRunning}
                disabled={isRunning}
                className="flex-1"
              >
                <FileSearch className="w-4 h-4 mr-1 text-indigo-600" />
                1. Jalankan Analisis Dry-Run
              </Button>

              <Button
                variant="primary"
                size="sm"
                onClick={() => setConfirmActualOpen(true)}
                disabled={!dryRunResult || isRunning}
                className="flex-1 bg-amber-600 hover:bg-amber-700 text-white"
              >
                <Play className="w-4 h-4 mr-1" />
                2. Eksekusi Reindex Aktual
              </Button>
            </div>
          </div>

          {/* Dry Run Outcome Banner */}
          {dryRunResult && (
            <div className="mt-4 p-3.5 bg-indigo-50 border border-indigo-200 rounded-xl text-xs space-y-1">
              <span className="font-bold text-indigo-900 block">Hasil Analisis Dry-Run Terverifikasi:</span>
              <p className="text-indigo-800">
                Model Sumber: <strong>{dryRunResult.from_model_version || "Aktif"}</strong> ➔ Target:{" "}
                <strong>{dryRunResult.to_model_version}</strong>
              </p>
              <p className="text-indigo-800">
                Total Foto Ditemukan: <strong>{dryRunResult.total_count}</strong> foto referensi pada{" "}
                <strong>{dryRunResult.affected_employee_count}</strong> karyawan.
              </p>
              <p className="text-[11px] text-indigo-600 pt-1">
                ✔ Aman untuk dieksekusi. Klik "2. Eksekusi Reindex Aktual" untuk memulai antrian background.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Active Job Monitor */}
      {activeJobDetail && (
        <Card className="border-indigo-300 shadow-sm">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <RefreshCw className={`w-4 h-4 text-indigo-600 ${isRunning ? "animate-spin" : ""}`} />
                <CardTitle className="text-xs">
                  Progres Pekerjaan Aktif: {activeJobDetail.id.slice(0, 8)}...
                </CardTitle>
              </div>

              <div className="flex items-center gap-2">
                <Badge
                  variant={
                    activeJobDetail.status === "completed"
                      ? "success"
                      : activeJobDetail.status === "failed"
                        ? "danger"
                        : "warning"
                  }
                  size="sm"
                >
                  {activeJobDetail.status.toUpperCase()}
                </Badge>

                {isRunning && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-rose-600 hover:bg-rose-50"
                    onClick={() => handleCancelJob(activeJobDetail.id)}
                  >
                    <StopCircle className="w-3.5 h-3.5 mr-1" />
                    Batalkan
                  </Button>
                )}
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Progress Bar */}
            <div>
              <div className="flex justify-between text-xs font-semibold text-gray-700 mb-1">
                <span>
                  Progres Pemrosesan: {activeJobDetail.processed_count ?? 0} /{" "}
                  {activeJobDetail.total_count ?? 0} foto
                </span>
                <span>{progressPercent}%</span>
              </div>
              <div className="w-full h-3 bg-gray-100 rounded-full overflow-hidden border border-gray-200">
                <div
                  className="h-full bg-indigo-600 rounded-full transition-all duration-300"
                  style={{ width: `${progressPercent}%` }}
                />
              </div>
            </div>

            {/* Metrics Breakdown */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
              <div className="p-2.5 bg-gray-50 rounded-lg border border-gray-100">
                <span className="text-gray-400 block text-[10px]">Berhasil Diperbarui</span>
                <span className="text-emerald-700 font-bold text-base">
                  {activeJobDetail.succeeded_count ?? 0}
                </span>
              </div>

              <div className="p-2.5 bg-gray-50 rounded-lg border border-gray-100">
                <span className="text-gray-400 block text-[10px]">Gagal Diekstrak</span>
                <span className="text-rose-700 font-bold text-base">{activeJobDetail.failed_count ?? 0}</span>
              </div>

              <div className="p-2.5 bg-gray-50 rounded-lg border border-gray-100">
                <span className="text-gray-400 block text-[10px]">Karyawan Siap</span>
                <span className="text-indigo-700 font-bold text-base">
                  {activeJobDetail.employees_ready_count ?? 0}
                </span>
              </div>

              <div className="p-2.5 bg-gray-50 rounded-lg border border-gray-100">
                <span className="text-gray-400 block text-[10px]">Karyawan Incomplete</span>
                <span className="text-amber-700 font-bold text-base">
                  {activeJobDetail.employees_incomplete_count ?? 0}
                </span>
              </div>
            </div>

            {/* Incomplete Employees List if any */}
            {activeJobDetail.incomplete_employees && activeJobDetail.incomplete_employees.length > 0 && (
              <div className="p-3 bg-rose-50 border border-rose-200 rounded-xl space-y-2 text-xs">
                <span className="font-bold text-rose-900 block">Karyawan Yang Perlu Pendaftaran Ulang:</span>
                <div className="divide-y divide-rose-100 max-h-36 overflow-y-auto">
                  {activeJobDetail.incomplete_employees.map((inc) => (
                    <div key={inc.employee_id} className="py-1.5 flex items-center justify-between">
                      <span className="font-mono text-gray-800">
                        {inc.employee_number || inc.employee_id}
                      </span>
                      <span className="text-rose-700 font-medium">
                        {inc.succeeded} / {inc.required} foto valid
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Past Jobs History Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-xs">Riwayat Pekerjaan Reindex</CardTitle>
        </CardHeader>
        <CardContent>
          {!pastJobs || pastJobs.length === 0 ? (
            <p className="text-center py-6 text-xs text-gray-400">Belum ada riwayat pekerjaan reindex.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-xs text-left">
                <thead className="bg-gray-50 text-gray-600 border-b border-gray-200">
                  <tr>
                    <th className="px-3 py-2 font-semibold">ID</th>
                    <th className="px-3 py-2 font-semibold">Target Model</th>
                    <th className="px-3 py-2 font-semibold">Waktu Mulai</th>
                    <th className="px-3 py-2 font-semibold">Status</th>
                    <th className="px-3 py-2 font-semibold text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {pastJobs.map((j) => (
                    <tr key={j.id} className="hover:bg-gray-50/50">
                      <td className="px-3 py-2 font-mono text-gray-700">{j.id.slice(0, 8)}...</td>
                      <td className="px-3 py-2 font-medium text-gray-900">{j.to_model_version}</td>
                      <td className="px-3 py-2 text-gray-500">
                        <LocalTime value={j.created_at} format="datetime" />
                      </td>
                      <td className="px-3 py-2">
                        <Badge
                          variant={
                            j.status === "completed"
                              ? "success"
                              : j.status === "failed"
                                ? "danger"
                                : "warning"
                          }
                          size="sm"
                        >
                          {j.status}
                        </Badge>
                      </td>
                      <td className="px-3 py-2 text-right">
                        <Button variant="ghost" size="sm" onClick={() => setActiveJobId(j.id)}>
                          Pantau
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Actual Reindex Consequence Dialog */}
      <ConfirmDialog
        open={confirmActualOpen}
        title="Konfirmasi Reindex Aktual Biometrik"
        description="Reindex aktual akan menggantikan seluruh representasi embedding wajah karyawan aktif di database. Selama proses berjalan, pendaftaran wajah baru ditolak. Apakah Anda yakin ingin melanjutkan?"
        confirmText="Mulai Reindex Sekarang"
        variant="danger"
        isLoading={isStartingActual}
        onConfirm={handleStartActual}
        onClose={() => setConfirmActualOpen(false)}
      />
    </div>
  );
}
