import { useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { Attendance, SuccessEnvelope } from "../../../types/api";
import { DataTable } from "../../../components/data/DataTable";
import type { Column } from "../../../components/data/DataTable";
import { Pagination } from "../../../components/data/Pagination";
import { AuthImage } from "../../../components/media/AuthImage";
import { AttendanceStatusBadge } from "../../../components/domain/AttendanceStatusBadge";
import { GeofenceBadge } from "../../../components/domain/GeofenceBadge";
import { SimilarityBar } from "../../../components/domain/SimilarityBar";
import { LocalTime } from "../../../components/domain/LocalTime";
import { Input } from "../../../components/ui/Input";
import { Select } from "../../../components/ui/Select";
import { Button } from "../../../components/ui/Button";
import { Clock, Filter, Eye } from "lucide-react";

export function AttendanceListPage() {
  const { t } = useTranslation(["attendance", "common"]);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [fromDate, setFromDate] = useState<string>("");
  const [toDate, setToDate] = useState<string>("");

  const queryParams = new URLSearchParams();
  queryParams.set("page", page.toString());
  queryParams.set("per_page", perPage.toString());
  if (statusFilter) queryParams.set("status", statusFilter);
  if (fromDate) queryParams.set("from_date", fromDate);
  if (toDate) queryParams.set("to_date", toDate);

  const { data, isLoading } = useQuery<SuccessEnvelope<Attendance[]>>({
    queryKey: ["attendances", "list", page, perPage, statusFilter, fromDate, toDate],
    queryFn: () => api.getWithMeta<Attendance[]>(`/api/v1/attendances?${queryParams.toString()}`),
  });

  const attendances = data?.data ?? [];
  const meta = data?.meta;

  const columns: Column<Attendance>[] = [
    {
      key: "photo",
      header: "Foto",
      render: (row) => (
        <AuthImage
          src={`/api/v1/attendances/${row.id}/photo`}
          alt="Foto Absen"
          className="w-8 h-8 rounded-lg object-cover border border-gray-200"
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
      key: "status",
      header: "Status",
      render: (row) => <AttendanceStatusBadge status={row.status} />,
    },
    {
      key: "verification_mode",
      header: "Metode",
      render: (row) => (
        <span className="text-xs text-gray-600 capitalize font-medium">
          {row.verification_mode || "Face"}
        </span>
      ),
    },
    {
      key: "similarity",
      header: "Kemiripan",
      render: (row) => (
        <div className="w-24">
          <SimilarityBar score={row.similarity_score} threshold={row.similarity_threshold} />
        </div>
      ),
    },
    {
      key: "geofence",
      header: "Geofence",
      render: (row) => (
        <GeofenceBadge isWithin={row.is_within_geofence} distanceMeters={row.distance_meters} />
      ),
    },
    {
      key: "actions",
      header: "Aksi",
      align: "right",
      render: (row) => (
        <Link to={`/attendances/${row.id}`}>
          <Button variant="ghost" size="sm">
            <Eye className="w-3.5 h-3.5 mr-1 text-gray-500" />
            Detail
          </Button>
        </Link>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <div className="flex items-center gap-2">
          <Clock className="w-5 h-5 text-indigo-600" />
          <h1 className="text-xl font-bold text-gray-900">{t("page.title", { ns: "attendance" })}</h1>
        </div>
        <p className="text-xs text-gray-500 mt-0.5">
          {t("page.subtitle", { ns: "attendance" })}
        </p>
      </div>

      {/* Filter Toolbar */}
      <div className="bg-white p-4 rounded-xl border border-gray-200/80 shadow-2xs flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-1.5 text-xs text-gray-500 font-medium">
          <Filter className="w-3.5 h-3.5 text-indigo-600" />
          <span>Filter:</span>
        </div>

        <div className="w-40">
          <Select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
          >
            <option value="">Semua Status</option>
            <option value="valid">Valid (Disetujui)</option>
            <option value="pending">Menunggu Review</option>
            <option value="rejected">Ditolak</option>
          </Select>
        </div>

        <div className="flex items-center gap-2">
          <Input
            type="date"
            value={fromDate}
            onChange={(e) => {
              setFromDate(e.target.value);
              setPage(1);
            }}
            placeholder="Dari Tanggal"
            className="w-36"
          />
          <span className="text-xs text-gray-400">s/d</span>
          <Input
            type="date"
            value={toDate}
            onChange={(e) => {
              setToDate(e.target.value);
              setPage(1);
            }}
            placeholder="Sampai Tanggal"
            className="w-36"
          />
        </div>

        {(statusFilter || fromDate || toDate) && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setStatusFilter("");
              setFromDate("");
              setToDate("");
              setPage(1);
            }}
            className="text-xs text-gray-500 hover:text-gray-900"
          >
            Reset Filter
          </Button>
        )}
      </div>

      {/* Table */}
      <DataTable
        columns={columns}
        data={attendances}
        keyExtractor={(item) => item.id}
        isLoading={isLoading}
        emptyTitle="Tidak Ada Data Absensi"
        emptyDescription="Belum ada rekaman absensi yang cocok dengan kriteria filter."
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
    </div>
  );
}
