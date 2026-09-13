import { useState } from "react";
import { useAuth } from "../../../lib/auth/useAuth";
import { api } from "../../../lib/api";
import { useToast } from "../../../components/ui/Toast";
import type { Attendance } from "../../../types/api";
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogContent,
  DialogFooter,
} from "../../../components/ui/Dialog";
import { Button } from "../../../components/ui/Button";
import { AuthImage } from "../../../components/media/AuthImage";
import { SimilarityBar } from "../../../components/domain/SimilarityBar";
import { GeofenceBadge } from "../../../components/domain/GeofenceBadge";
import { HintList } from "../../../components/domain/HintList";
import { LocalTime } from "../../../components/domain/LocalTime";
import { AlertTriangle, CheckCircle, XCircle } from "lucide-react";

interface ReviewDrawerProps {
  attendance: Attendance | null;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function ReviewDrawer({ attendance, isOpen, onClose, onSuccess }: ReviewDrawerProps) {
  const { user } = useAuth();
  const toast = useToast();

  const [notes, setNotes] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [actionType, setActionType] = useState<"approve" | "reject" | null>(null);

  if (!attendance) return null;

  // Rule B11: Guard against self-review
  const isSelfReview = Boolean(
    (user?.employee_id && user.employee_id === attendance.employee_id) ||
    (user?.id && user.id === attendance.user_id),
  );

  const handleAction = async (decision: "approve" | "reject") => {
    if (isSelfReview) {
      toast.show({
        type: "error",
        title: "Pelanggaran Aturan B11",
        message: "Anda tidak dapat menyetujui atau menolak absensi Anda sendiri.",
      });
      return;
    }

    setIsSubmitting(true);
    setActionType(decision);

    try {
      const endpoint = `/api/v1/attendances/${attendance.id}/${decision}`;
      await api.put(endpoint, {
        notes: notes.trim() || undefined,
      });

      toast.show({
        type: "success",
        title: decision === "approve" ? "Absensi Disetujui" : "Absensi Ditolak",
        message: `Absensi untuk ${attendance.employee_name || "karyawan"} telah diproses.`,
      });

      onSuccess();
      onClose();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Memproses Review",
        message: "Terjadi kesalahan saat memproses keputusan. Coba beberapa saat lagi.",
      });
    } finally {
      setIsSubmitting(false);
      setActionType(null);
    }
  };

  return (
    <Dialog open={isOpen} onClose={onClose} size="lg">
      <DialogHeader>
        <DialogTitle>Tinjau Permohonan Absensi</DialogTitle>
      </DialogHeader>

      <DialogContent className="space-y-4 max-h-[75vh] overflow-y-auto">
        {/* Rule B11 Alert if reviewer is attendance owner */}
        {isSelfReview && (
          <div className="p-3 bg-amber-50 border border-amber-300 rounded-xl flex items-start gap-2 text-amber-900 text-xs">
            <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
            <div>
              <strong className="font-semibold block">Pencegahan Konflik Kepentingan (B11)</strong>
              Anda terdeteksi sebagai pemilik catatan kehadiran ini. Aturan sistem melarang supervisor
              menyetujui atau menolak absensi miliknya sendiri. Mohon minta supervisor lain untuk meninjau.
            </div>
          </div>
        )}

        {/* Employee Info Header */}
        <div className="flex items-center justify-between p-3 bg-gray-50 rounded-xl border border-gray-100 text-xs">
          <div>
            <p className="font-bold text-gray-900 text-sm">{attendance.employee_name || "Karyawan"}</p>
            <p className="text-gray-500">ID: {attendance.employee_id}</p>
          </div>
          <div className="text-right">
            <LocalTime
              value={attendance.recorded_at}
              format="datetime"
              className="font-medium text-gray-700"
            />
            <p className="text-gray-400 capitalize">{attendance.check_type || "Masuk"}</p>
          </div>
        </div>

        {/* Comparison Photos */}
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5 text-center">
            <span className="text-[11px] font-semibold text-gray-600 block">Foto Absensi</span>
            <div className="aspect-square rounded-xl overflow-hidden border border-gray-200 bg-gray-100 flex items-center justify-center">
              <AuthImage
                src={`/api/v1/attendances/${attendance.id}/photo`}
                alt="Foto Absensi"
                className="w-full h-full object-cover"
              />
            </div>
          </div>

          <div className="space-y-1.5 text-center">
            <span className="text-[11px] font-semibold text-gray-600 block">Foto Terdaftar (Referensi)</span>
            <div className="aspect-square rounded-xl overflow-hidden border border-gray-200 bg-gray-100 flex items-center justify-center">
              <AuthImage
                src={`/api/v1/employees/${attendance.employee_id}/face/photo`}
                alt="Foto Referensi"
                className="w-full h-full object-cover"
              />
            </div>
          </div>
        </div>

        {/* Metrics: Similarity & Geofence */}
        <div className="space-y-3 pt-2">
          <div>
            <span className="text-xs font-semibold text-gray-700 block mb-1">Skor Kemiripan Wajah</span>
            <SimilarityBar score={attendance.similarity_score} threshold={attendance.similarity_threshold} />
          </div>

          <div className="flex items-center justify-between text-xs pt-1">
            <span className="text-gray-600 font-medium">Status Lokasi (Geofence):</span>
            <GeofenceBadge
              isWithin={attendance.is_within_geofence}
              distanceMeters={attendance.distance_meters}
            />
          </div>

          {attendance.quality_hints && attendance.quality_hints.length > 0 && (
            <div>
              <span className="text-xs font-semibold text-gray-700 block mb-1">Catatan Kualitas Wajah:</span>
              <HintList hints={attendance.quality_hints} />
            </div>
          )}
        </div>

        {/* Supervisor Review Notes */}
        <div className="pt-2">
          <label className="block text-xs font-semibold text-gray-700 mb-1">
            Catatan Persetujuan / Penolakan (Opsional)
          </label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            disabled={isSelfReview || isSubmitting}
            placeholder="Tuliskan alasan jika menolak, atau keterangan validasi..."
            rows={2}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-xs focus:border-indigo-500 focus:outline-none disabled:bg-gray-50 disabled:text-gray-400"
          />
        </div>
      </DialogContent>

      <DialogFooter>
        <Button variant="outline" size="sm" onClick={onClose} disabled={isSubmitting}>
          Batal
        </Button>

        <Button
          variant="danger"
          size="sm"
          disabled={isSelfReview || isSubmitting}
          isLoading={isSubmitting && actionType === "reject"}
          onClick={() => handleAction("reject")}
        >
          <XCircle className="w-4 h-4 mr-1" /> Tolak
        </Button>

        <Button
          variant="primary"
          size="sm"
          disabled={isSelfReview || isSubmitting}
          isLoading={isSubmitting && actionType === "approve"}
          onClick={() => handleAction("approve")}
        >
          <CheckCircle className="w-4 h-4 mr-1" /> Setujui
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
