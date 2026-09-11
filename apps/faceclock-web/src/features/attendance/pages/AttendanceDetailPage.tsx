import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { Attendance } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { AuthImage } from "../../../components/media/AuthImage";
import { AttendanceStatusBadge } from "../../../components/domain/AttendanceStatusBadge";
import { GeofenceBadge } from "../../../components/domain/GeofenceBadge";
import { SimilarityBar } from "../../../components/domain/SimilarityBar";
import { HintList } from "../../../components/domain/HintList";
import { LocalTime } from "../../../components/domain/LocalTime";
import { LocationPreview } from "../../../components/map/LocationPreview";
import { ReviewDrawer } from "../components/ReviewDrawer";
import { ArrowLeft, CheckSquare, Clock, MapPin, Shield } from "lucide-react";

export function AttendanceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const {
    data: attendance,
    isLoading,
    refetch,
  } = useQuery<Attendance>({
    queryKey: ["attendances", "detail", id],
    queryFn: () => api.get<Attendance>(`/api/v1/attendances/${id}`),
    enabled: Boolean(id),
  });

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent" />
      </div>
    );
  }

  if (!attendance) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500 text-sm">Data absensi tidak ditemukan.</p>
        <Link to="/attendances" className="mt-4 inline-block">
          <Button variant="outline" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" /> Kembali ke Daftar
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      {/* Top action bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/attendances">
            <Button variant="outline" size="sm">
              <ArrowLeft className="w-4 h-4 mr-1" /> Kembali
            </Button>
          </Link>
          <div>
            <h1 className="text-lg font-bold text-gray-900">
              Detail Absensi: {attendance.employee_name || "Karyawan"}
            </h1>
            <p className="text-[11px] text-gray-400 font-mono">ID: {attendance.id}</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <AttendanceStatusBadge status={attendance.status} />

          {attendance.status === "pending" && (
            <Button variant="primary" size="sm" onClick={() => setIsDrawerOpen(true)}>
              <CheckSquare className="w-4 h-4 mr-1" /> Tinjau Absensi
            </Button>
          )}
        </div>
      </div>

      {/* Main detail grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Verification & Biometric Card */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-indigo-600" />
              <CardTitle>Verifikasi Biometrik Wajah</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Photos comparison */}
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5 text-center">
                <span className="text-[11px] font-semibold text-gray-600 block">Foto Saat Absen</span>
                <div className="aspect-square rounded-xl overflow-hidden border border-gray-200 bg-gray-50 flex items-center justify-center">
                  <AuthImage
                    src={`/api/v1/attendances/${attendance.id}/photo`}
                    alt="Foto Absen"
                    className="w-full h-full object-cover"
                  />
                </div>
              </div>

              <div className="space-y-1.5 text-center">
                <span className="text-[11px] font-semibold text-gray-600 block">
                  Foto Referensi Terdaftar
                </span>
                <div className="aspect-square rounded-xl overflow-hidden border border-gray-200 bg-gray-50 flex items-center justify-center">
                  <AuthImage
                    src={`/api/v1/employees/${attendance.employee_id}/face/photo`}
                    alt="Foto Referensi"
                    className="w-full h-full object-cover"
                  />
                </div>
              </div>
            </div>

            {/* Similarity Score vs Threshold */}
            <div>
              <span className="text-xs font-semibold text-gray-700 block mb-1">
                Skor Kemiripan (Cosine Similarity)
              </span>
              <SimilarityBar
                score={attendance.similarity_score}
                threshold={attendance.similarity_threshold}
              />
            </div>

            {/* Quality hints */}
            {attendance.quality_hints && attendance.quality_hints.length > 0 && (
              <div>
                <span className="text-xs font-semibold text-gray-700 block mb-1">
                  Catatan Kualitas Wajah (Hints)
                </span>
                <HintList hints={attendance.quality_hints} />
              </div>
            )}
          </CardContent>
        </Card>

        {/* Timestamp & Location Card */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-indigo-600" />
                <CardTitle>Waktu & Detail Catatan</CardTitle>
              </div>
            </CardHeader>
            <CardContent className="space-y-3 text-xs">
              <div className="flex justify-between py-1 border-b border-gray-100">
                <span className="text-gray-500">Waktu Absen:</span>
                <LocalTime
                  value={attendance.recorded_at}
                  format="datetime"
                  className="font-semibold text-gray-900"
                />
              </div>

              <div className="flex justify-between py-1 border-b border-gray-100">
                <span className="text-gray-500">Tipe Absen:</span>
                <span className="font-semibold text-gray-900 capitalize">
                  {attendance.check_type || "Masuk"}
                </span>
              </div>

              <div className="flex justify-between py-1 border-b border-gray-100">
                <span className="text-gray-500">Metode Verifikasi:</span>
                <span className="font-semibold text-gray-900 capitalize">
                  {attendance.verification_mode || "Face"}
                </span>
              </div>

              {attendance.reviewed_at && (
                <div className="flex justify-between py-1 border-b border-gray-100 bg-amber-50/50 px-2 rounded">
                  <span className="text-amber-800">Ditinjau pada:</span>
                  <LocalTime
                    value={attendance.reviewed_at}
                    format="datetime"
                    className="font-semibold text-amber-900"
                  />
                </div>
              )}

              {attendance.notes && (
                <div className="pt-2">
                  <span className="text-gray-500 block mb-0.5">Catatan Supervisor:</span>
                  <p className="bg-gray-50 p-2 rounded border border-gray-100 text-gray-800 italic">
                    "{attendance.notes}"
                  </p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Location & Map Preview */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <MapPin className="w-4 h-4 text-indigo-600" />
                  <CardTitle>Lokasi Absen & Geofence</CardTitle>
                </div>
                <GeofenceBadge
                  isWithin={attendance.is_within_geofence}
                  distanceMeters={attendance.distance_meters}
                />
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {attendance.latitude != null && attendance.longitude != null ? (
                <LocationPreview
                  attendanceLat={attendance.latitude}
                  attendanceLng={attendance.longitude}
                  officeLat={attendance.office_latitude ?? attendance.latitude}
                  officeLng={attendance.office_longitude ?? attendance.longitude}
                  officeRadius={attendance.office_radius_meters ?? 50}
                  className="h-44 w-full"
                />
              ) : (
                <div className="h-32 bg-gray-50 rounded-xl flex items-center justify-center text-xs text-gray-400 border border-gray-100">
                  Data koordinat GPS tidak tersedia.
                </div>
              )}

              <div className="grid grid-cols-2 gap-2 text-[11px] text-gray-500 pt-1">
                <div>
                  <span>Koordinat Absen:</span>
                  <p className="font-mono text-gray-800">
                    {attendance.latitude?.toFixed(6) || "-"}, {attendance.longitude?.toFixed(6) || "-"}
                  </p>
                </div>
                <div>
                  <span>Jarak ke Kantor:</span>
                  <p className="font-mono text-gray-800">
                    {attendance.distance_meters != null
                      ? `${Math.round(attendance.distance_meters)} meter`
                      : "-"}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Review Drawer Modal */}
      <ReviewDrawer
        attendance={attendance}
        isOpen={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        onSuccess={() => refetch()}
      />
    </div>
  );
}
