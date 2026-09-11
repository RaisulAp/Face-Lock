import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useToast } from "../../../components/ui/Toast";
import type { BiometricConsent, Employee, SuccessEnvelope } from "../../../types/api";
import { DataTable } from "../../../components/data/DataTable";
import type { Column } from "../../../components/data/DataTable";
import { Pagination } from "../../../components/data/Pagination";
import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogContent,
  DialogFooter,
} from "../../../components/ui/Dialog";
import { Select } from "../../../components/ui/Select";
import { Input } from "../../../components/ui/Input";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { LocalTime } from "../../../components/domain/LocalTime";
import { FileCheck, Plus, XCircle, ShieldAlert } from "lucide-react";

export function ConsentsPage() {
  const toast = useToast();
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);

  const [isRecordModalOpen, setIsRecordModalOpen] = useState(false);
  const [selectedEmployeeId, setSelectedEmployeeId] = useState("");
  const [documentRef, setDocumentRef] = useState("");
  const [notes, setNotes] = useState("");
  const [isRecording, setIsRecording] = useState(false);

  const [revokeConsentId, setRevokeConsentId] = useState<string | null>(null);
  const [isRevoking, setIsRevoking] = useState(false);

  // Fetch Consents
  const { data, isLoading, refetch } = useQuery<SuccessEnvelope<BiometricConsent[]>>({
    queryKey: ["consents", "list", page, perPage],
    queryFn: () => api.getWithMeta<BiometricConsent[]>(`/api/v1/consents?page=${page}&per_page=${perPage}`),
  });

  // Fetch Employees for dropdown
  const { data: employeesData } = useQuery<SuccessEnvelope<Employee[]>>({
    queryKey: ["employees", "list-for-consent"],
    queryFn: () => api.getWithMeta<Employee[]>("/api/v1/employees?per_page=100"),
    enabled: isRecordModalOpen,
  });

  const consents = data?.data ?? [];
  const meta = data?.meta;
  const employees = employeesData?.data ?? [];

  const handleRecordOfflineConsent = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedEmployeeId) {
      toast.show({
        type: "error",
        title: "Validasi Gagal",
        message: "Pilih karyawan terlebih dahulu.",
      });
      return;
    }

    setIsRecording(true);
    try {
      await api.post("/api/v1/consents", {
        employee_id: selectedEmployeeId,
        consent_type: "biometric",
        status: "granted",
        document_reference: documentRef.trim() || undefined,
        notes: notes.trim() || "Persetujuan formulir fisik offline",
      });

      toast.show({
        type: "success",
        title: "Persetujuan Dicatat",
        message: "Dokumentasi persetujuan biometrik berhasil disimpan.",
      });

      setIsRecordModalOpen(false);
      setSelectedEmployeeId("");
      setDocumentRef("");
      setNotes("");
      refetch();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Mencatat Persetujuan",
        message: "Terjadi kesalahan saat menyimpan formulir persetujuan.",
      });
    } finally {
      setIsRecording(false);
    }
  };

  const handleRevokeConsent = async () => {
    if (!revokeConsentId) return;
    setIsRevoking(true);
    try {
      await api.put(`/api/v1/consents/${revokeConsentId}/revoke`, {});
      toast.show({
        type: "success",
        title: "Persetujuan Dicabut",
        message: "Persetujuan biometrik telah dicabut. Data wajah karyawan terkait dinonaktifkan.",
      });
      setRevokeConsentId(null);
      refetch();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Mencabut Persetujuan",
        message: "Terjadi kesalahan saat memproses pencabutan persetujuan.",
      });
    } finally {
      setIsRevoking(false);
    }
  };

  const columns: Column<BiometricConsent>[] = [
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
      key: "status",
      header: "Status Persetujuan",
      render: (row) => (
        <Badge
          variant={row.status === "granted" ? "success" : row.status === "revoked" ? "danger" : "neutral"}
          size="sm"
        >
          {row.status === "granted" ? "Disetujui" : row.status === "revoked" ? "Dicabut" : "Pending"}
        </Badge>
      ),
    },
    {
      key: "document_reference",
      header: "Dokumen Referensi",
      render: (row) => (
        <span className="text-xs text-gray-600 font-mono">
          {row.document_reference || "Digital (In-App)"}
        </span>
      ),
    },
    {
      key: "created_at",
      header: "Waktu Pencatatan",
      render: (row) => <LocalTime value={row.created_at} format="datetime" />,
    },
    {
      key: "actions",
      header: "Aksi",
      align: "right",
      render: (row) =>
        row.status === "granted" ? (
          <Button
            variant="ghost"
            size="sm"
            className="text-rose-600 hover:bg-rose-50"
            onClick={() => setRevokeConsentId(row.id)}
          >
            <XCircle className="w-3.5 h-3.5 mr-1" />
            Cabut
          </Button>
        ) : null,
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <FileCheck className="w-5 h-5 text-indigo-600" />
            <h1 className="text-xl font-bold text-gray-900">Persetujuan Biometrik (Consent)</h1>
          </div>
          <p className="text-xs text-gray-500 mt-0.5">
            Kepatuhan privasi UU PDP: Dokumentasi persetujuan pemrosesan data wajah karyawan.
          </p>
        </div>

        <Button variant="primary" size="sm" onClick={() => setIsRecordModalOpen(true)}>
          <Plus className="w-4 h-4 mr-1.5" />
          Catat Persetujuan Manual / Fisik
        </Button>
      </div>

      {/* Table */}
      <DataTable
        columns={columns}
        data={consents}
        keyExtractor={(item) => item.id}
        isLoading={isLoading}
        emptyTitle="Belum Ada Rekaman Persetujuan"
        emptyDescription="Catat formulir persetujuan fisik atau tunggu karyawan menyetujui melalui aplikasi."
      />

      {/* Pagination */}
      {meta && (
        <Pagination
          page={page}
          perPage={perPage}
          total={meta.total}
          onPageChange={setPage}
          onPerPageChange={(size) => {
            setPerPage(size);
            setPage(1);
          }}
        />
      )}

      {/* Record Offline Consent Modal */}
      <Dialog open={isRecordModalOpen} onClose={() => setIsRecordModalOpen(false)} size="md">
        <DialogHeader>
          <DialogTitle>Pencatatan Persetujuan Biometrik Fisik</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleRecordOfflineConsent}>
          <DialogContent className="space-y-3.5">
            <div className="p-3 bg-indigo-50 border border-indigo-200 rounded-xl flex items-start gap-2 text-indigo-900 text-xs">
              <ShieldAlert className="w-4 h-4 text-indigo-600 shrink-0 mt-0.5" />
              <div>
                Pastikan karyawan telah menandatangani formulir fisik persetujuan pemrosesan data biometrik
                wajah sesuai standar UU Pelindungan Data Pribadi (UU PDP).
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Pilih Karyawan *</label>
              <Select
                required
                value={selectedEmployeeId}
                onChange={(e) => setSelectedEmployeeId(e.target.value)}
              >
                <option value="">-- Pilih Karyawan --</option>
                {employees.map((emp) => (
                  <option key={emp.id} value={emp.id}>
                    {emp.nip} - {emp.name} ({emp.department || "Umum"})
                  </option>
                ))}
              </Select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">
                Nomor / Kode Formulir Fisik (Opsional)
              </label>
              <Input
                placeholder="Contoh: DOC-PDP-2026-081"
                value={documentRef}
                onChange={(e) => setDocumentRef(e.target.value)}
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Catatan Tambahan</label>
              <Input
                placeholder="Keterangan arsip atau lokasi lemari dokumen"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </div>
          </DialogContent>

          <DialogFooter>
            <Button variant="outline" size="sm" type="button" onClick={() => setIsRecordModalOpen(false)}>
              Batal
            </Button>
            <Button variant="primary" size="sm" type="submit" isLoading={isRecording}>
              Simpan Persetujuan
            </Button>
          </DialogFooter>
        </form>
      </Dialog>

      {/* Revoke Consent Confirm Dialog */}
      <ConfirmDialog
        open={Boolean(revokeConsentId)}
        title="Cabut Persetujuan Biometrik?"
        description="Mencabut persetujuan ini akan menghapus hak sistem untuk memproses foto dan embedding wajah karyawan. Pendaftaran wajah aktif karyawan terkait akan dinonaktifkan."
        confirmText="Ya, Cabut Persetujuan"
        variant="danger"
        isLoading={isRevoking}
        onConfirm={handleRevokeConsent}
        onClose={() => setRevokeConsentId(null)}
      />
    </div>
  );
}
