import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useAuth } from "../../../lib/auth/useAuth";
import { useToast } from "../../../components/ui/Toast";
import type { Employee, FaceProfile } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { AuthImage } from "../../../components/media/AuthImage";
import { LocalTime } from "../../../components/domain/LocalTime";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { ArrowLeft, ScanFace, Trash2, ShieldCheck, ShieldAlert, Sparkles } from "lucide-react";

export function EmployeeFacePage() {
  const { t } = useTranslation(["employee", "common"]);
  const { id } = useParams<{ id: string }>();
  const { can } = useAuth();
  const toast = useToast();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  // 1. Fetch Employee info
  const { data: employee } = useQuery<Employee>({
    queryKey: ["employees", "detail", id],
    queryFn: () => api.get<Employee>(`/api/v1/employees/${id}`),
    enabled: Boolean(id),
  });

  // 2. Fetch Face Profile
  const {
    data: faceProfile,
    isLoading: isFaceLoading,
    refetch: refetchFace,
    error: faceError,
  } = useQuery<FaceProfile>({
    queryKey: ["employees", "face", id],
    queryFn: () => api.get<FaceProfile>(`/api/v1/employees/${id}/face`),
    enabled: Boolean(id),
    retry: false,
  });

  const isEnrolled = Boolean(faceProfile && !faceError);

  const handleDeleteFace = async () => {
    setIsDeleting(true);
    try {
      await api.delete(`/api/v1/employees/${id}/face`);
      toast.show({
        type: "success",
        title: "Data Biometrik Dihapus",
        message: "Data wajah dan embedding karyawan telah dihapus dari sistem.",
      });
      setDeleteDialogOpen(false);
      refetchFace();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Menghapus",
        message: "Terjadi kesalahan saat menghapus data biometrik.",
      });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="space-y-6 max-w-3xl mx-auto">
      <div className="flex items-center gap-3">
        <Link to="/employees">
          <Button variant="outline" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" /> {t("actions.back", { ns: "common" })}
          </Button>
        </Link>
        <div>
          <h1 className="text-xl font-bold text-gray-900">{t("face.title", { ns: "employee" })}: {employee?.name || "Karyawan"}</h1>
          <p className="text-xs text-gray-500 mt-0.5">
            {t("face.subtitle", { ns: "employee" })}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Status & Photo Card */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <ScanFace className="w-4 h-4 text-indigo-600" />
              <CardTitle>Foto Template Biometrik</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {isFaceLoading ? (
              <div className="aspect-square bg-gray-50 rounded-xl flex items-center justify-center">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent" />
              </div>
            ) : isEnrolled ? (
              <div className="space-y-3">
                <div className="aspect-square rounded-2xl overflow-hidden border border-gray-200 bg-gray-100 flex items-center justify-center shadow-xs">
                  <AuthImage
                    src={`/api/v1/employees/${id}/face/photo`}
                    alt="Foto Biometrik"
                    className="w-full h-full object-cover"
                  />
                </div>
                <div className="flex items-center justify-center gap-1.5 text-xs font-semibold text-emerald-700 bg-emerald-50 py-1.5 rounded-lg border border-emerald-200">
                  <ShieldCheck className="w-4 h-4" />
                  <span>Wajah Terdaftar & Aktif</span>
                </div>
              </div>
            ) : (
              <div className="aspect-square rounded-2xl border-2 border-dashed border-gray-200 bg-gray-50 flex flex-col items-center justify-center p-6 text-center">
                <ShieldAlert className="w-10 h-10 text-amber-500 mb-2" />
                <p className="text-xs font-semibold text-gray-700">Belum Terdaftar</p>
                <p className="text-[11px] text-gray-400 mt-1 leading-relaxed">
                  Karyawan ini belum memiliki pendaftaran template wajah. Pendaftaran mandiri dapat dilakukan
                  lewat portal absensi.
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Details & Actions Card */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-xs">Informasi Profil Biometrik</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-xs">
              <div className="flex justify-between py-1 border-b border-gray-100">
                <span className="text-gray-500">NIP:</span>
                <span className="font-mono font-semibold text-gray-900">{employee?.nip}</span>
              </div>

              <div className="flex justify-between py-1 border-b border-gray-100">
                <span className="text-gray-500">Model AI:</span>
                <span className="font-semibold text-indigo-700">
                  {faceProfile?.model_version || "buffalo_l (ArcFace)"}
                </span>
              </div>

              <div className="flex justify-between py-1 border-b border-gray-100">
                <span className="text-gray-500">Dimensi Vektor:</span>
                <span className="font-mono text-gray-800">512-D float32</span>
              </div>

              <div className="flex justify-between py-1 border-b border-gray-100">
                <span className="text-gray-500">Terdaftar Pada:</span>
                <LocalTime
                  value={faceProfile?.created_at}
                  format="datetime"
                  className="font-medium text-gray-800"
                />
              </div>

              <div className="pt-3">
                <Link
                  to="/consents"
                  className="inline-flex items-center gap-1.5 text-xs text-indigo-600 hover:underline font-medium"
                >
                  <Sparkles className="w-3.5 h-3.5" />
                  Periksa Persetujuan Biometrik (Consent)
                </Link>
              </div>
            </CardContent>
          </Card>

          {/* Delete Danger Zone */}
          {isEnrolled && can("face.delete_any") && (
            <Card className="border-rose-200 bg-rose-50/30">
              <CardContent className="p-4 space-y-2">
                <span className="text-xs font-bold text-rose-800 block">Hapus Data Biometrik</span>
                <p className="text-[11px] text-rose-600 leading-relaxed">
                  Menghapus template wajah akan membuat karyawan tidak dapat melakukan absensi pengenalan
                  wajah sampai dilakukan pendaftaran ulang.
                </p>
                <Button variant="danger" size="sm" onClick={() => setDeleteDialogOpen(true)} className="mt-2">
                  <Trash2 className="w-3.5 h-3.5 mr-1" />
                  Hapus Data Wajah
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Confirm Delete Dialog */}
      <ConfirmDialog
        open={deleteDialogOpen}
        title="Hapus Data Biometrik Wajah?"
        description={`Anda yakin ingin menghapus template wajah untuk karyawan "${employee?.name}"? Foto dan embedding 512-dimensi akan dihapus permanen.`}
        confirmText="Hapus Permanen"
        variant="danger"
        isLoading={isDeleting}
        onConfirm={handleDeleteFace}
        onClose={() => setDeleteDialogOpen(false)}
      />
    </div>
  );
}
