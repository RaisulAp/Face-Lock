import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { Attendance, SuccessEnvelope } from "../../../types/api";
import { DataTable } from "../../../components/data/DataTable";
import type { Column } from "../../../components/data/DataTable";
import { ReviewDrawer } from "../components/ReviewDrawer";
import { AuthImage } from "../../../components/media/AuthImage";
import { SimilarityBar } from "../../../components/domain/SimilarityBar";
import { GeofenceBadge } from "../../../components/domain/GeofenceBadge";
import { LocalTime } from "../../../components/domain/LocalTime";
import { Button } from "../../../components/ui/Button";
import { CheckSquare, RefreshCw } from "lucide-react";

export function PendingQueuePage() {
  const [selectedAttendance, setSelectedAttendance] = useState<Attendance | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const { data, isLoading, refetch, isFetching } = useQuery<SuccessEnvelope<Attendance[]>>({
    queryKey: ["attendances", "pending"],
    queryFn: () => api.getWithMeta<Attendance[]>("/api/v1/attendances/pending"),
    refetchInterval: 15_000,
  });

  const attendances = data?.data ?? [];

  const columns: Column<Attendance>[] = [
    {
      key: "photo",
      header: "Foto",
      render: (row) => (
        <AuthImage
          src={`/api/v1/attendances/${row.id}/photo`}
          alt="Foto Absen"
          className="w-9 h-9 rounded-lg object-cover border border-gray-200"
        />
      ),
    },
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
      key: "similarity",
      header: "Kemiripan Wajah",
      render: (row) => (
        <div className="w-32">
          <SimilarityBar score={row.similarity_score} threshold={row.similarity_threshold} />
        </div>
      ),
    },
    {
      key: "geofence",
      header: "Status Geofence",
      render: (row) => (
        <GeofenceBadge isWithin={row.is_within_geofence} distanceMeters={row.distance_meters} />
      ),
    },
    {
      key: "actions",
      header: "Tindakan",
      align: "right",
      render: (row) => (
        <Button
          variant="primary"
          size="sm"
          onClick={() => {
            setSelectedAttendance(row);
            setIsDrawerOpen(true);
          }}
        >
          Tinjau Absensi
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <CheckSquare className="w-5 h-5 text-indigo-600" />
            <h1 className="text-xl font-bold text-gray-900">Antrian Review Kehadiran</h1>
          </div>
          <p className="text-xs text-gray-500 mt-0.5">
            Daftar absensi yang tertahan untuk verifikasi manual supervisor (anomali kemiripan atau geofence).
          </p>
        </div>

        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
          <RefreshCw className={`w-3.5 h-3.5 mr-1 ${isFetching ? "animate-spin" : ""}`} />
          Segarkan
        </Button>
      </div>

      {/* Table */}
      <DataTable
        columns={columns}
        data={attendances}
        keyExtractor={(item) => item.id}
        isLoading={isLoading}
        emptyTitle="Antrian Bersih"
        emptyDescription="Tidak ada permohonan kehadiran yang menunggu tinjauan manual saat ini."
      />

      {/* Review Drawer / Modal */}
      <ReviewDrawer
        attendance={selectedAttendance}
        isOpen={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        onSuccess={() => refetch()}
      />
    </div>
  );
}
