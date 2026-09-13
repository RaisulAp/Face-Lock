import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useToast } from "../../../components/ui/Toast";
import type { OfficeLocation } from "../../../types/api";
import { DataTable } from "../../../components/data/DataTable";
import type { Column } from "../../../components/data/DataTable";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogContent,
  DialogFooter,
} from "../../../components/ui/Dialog";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { MapPicker } from "../../../components/map/MapPicker";
import { MapPin, Plus, Edit2, Trash2, Crosshair } from "lucide-react";

export function LocationsPage() {
  const toast = useToast();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingLocation, setEditingLocation] = useState<OfficeLocation | null>(null);

  // Form states
  const [name, setName] = useState("");
  const [address, setAddress] = useState("");
  const [latitude, setLatitude] = useState<number>(-6.2088);
  const [longitude, setLongitude] = useState<number>(106.8456);
  const [radiusMeters, setRadiusMeters] = useState<number>(50);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Delete state
  const [deleteLocationId, setDeleteLocationId] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  // Fetch locations
  const {
    data: locations,
    isLoading,
    refetch,
  } = useQuery<OfficeLocation[]>({
    queryKey: ["office-locations", "list"],
    queryFn: () => api.get<OfficeLocation[]>("/api/v1/office-locations"),
  });

  const handleOpenCreate = () => {
    setEditingLocation(null);
    setName("");
    setAddress("");
    setLatitude(-6.2088);
    setLongitude(106.8456);
    setRadiusMeters(50);
    setIsModalOpen(true);
  };

  const handleOpenEdit = (loc: OfficeLocation) => {
    setEditingLocation(loc);
    setName(loc.name);
    setAddress(loc.address || "");
    setLatitude(loc.latitude ?? loc.lat ?? -6.2088);
    setLongitude(loc.longitude ?? loc.lng ?? 106.8456);
    setRadiusMeters(loc.radius_meters ?? loc.radius_meter ?? 100);
    setIsModalOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    const payload = {
      name: name.trim(),
      address: address.trim() || undefined,
      latitude,
      longitude,
      radius_meters: radiusMeters,
    };

    try {
      if (editingLocation) {
        await api.put(`/api/v1/office-locations/${editingLocation.id}`, payload);
        toast.show({
          type: "success",
          title: "Lokasi Diperbarui",
          message: `Lokasi "${name}" berhasil disimpan.`,
        });
      } else {
        await api.post("/api/v1/office-locations", payload);
        toast.show({
          type: "success",
          title: "Lokasi Ditambahkan",
          message: `Lokasi "${name}" berhasil dibuat.`,
        });
      }
      setIsModalOpen(false);
      refetch();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Menyimpan",
        message: "Periksa kembali koordinat dan data formulir.",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteLocationId) return;
    setIsDeleting(true);
    try {
      await api.delete(`/api/v1/office-locations/${deleteLocationId}`);
      toast.show({
        type: "success",
        title: "Lokasi Dihapus",
        message: "Data lokasi kantor telah dihapus dari sistem.",
      });
      setDeleteLocationId(null);
      refetch();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Menghapus",
        message: "Lokasi mungkin sedang ditautkan pada data karyawan aktif.",
      });
    } finally {
      setIsDeleting(false);
    }
  };

  const columns: Column<OfficeLocation>[] = [
    {
      key: "name",
      header: "Nama Kantor",
      render: (row) => (
        <div>
          <p className="font-semibold text-gray-900">{row.name}</p>
          <span className="text-[11px] text-gray-400">{row.address || "-"}</span>
        </div>
      ),
    },
    {
      key: "coordinates",
      header: "Titik Koordinat (Lat, Lng)",
      render: (row) => {
        const lat = row.latitude ?? row.lat;
        const lng = row.longitude ?? row.lng;
        return (
          <span className="font-mono text-xs text-gray-700">
            {lat != null ? lat.toFixed(6) : "-"}, {lng != null ? lng.toFixed(6) : "-"}
          </span>
        );
      },
    },
    {
      key: "radius",
      header: "Radius Geofence",
      render: (row) => (
        <span className="inline-flex items-center gap-1 font-semibold text-xs text-indigo-700 bg-indigo-50 px-2 py-0.5 rounded-md border border-indigo-200">
          <Crosshair className="w-3 h-3" />
          {row.radius_meters} meter
        </span>
      ),
    },
    {
      key: "actions",
      header: "Aksi",
      align: "right",
      render: (row) => (
        <div className="flex items-center justify-end gap-1.5">
          <Button variant="ghost" size="sm" onClick={() => handleOpenEdit(row)}>
            <Edit2 className="w-3.5 h-3.5 mr-1" /> Edit
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-rose-600 hover:bg-rose-50"
            onClick={() => setDeleteLocationId(row.id)}
          >
            <Trash2 className="w-3.5 h-3.5" />
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <MapPin className="w-5 h-5 text-indigo-600" />
            <h1 className="text-xl font-bold text-gray-900">Lokasi Kantor & Geofence</h1>
          </div>
          <p className="text-xs text-gray-500 mt-0.5">
            Daftar titik koordinat absensi resmi dan radius toleransi jarak (Decision D22).
          </p>
        </div>

        <Button variant="primary" size="sm" onClick={handleOpenCreate}>
          <Plus className="w-4 h-4 mr-1.5" />
          Tambah Lokasi Kantor
        </Button>
      </div>

      {/* Table */}
      <DataTable
        columns={columns}
        data={locations ?? []}
        keyExtractor={(item) => item.id}
        isLoading={isLoading}
        emptyTitle="Belum Ada Lokasi Kantor"
        emptyDescription="Tambahkan kantor atau titik presensi untuk memvalidasi geolokasi absensi."
      />

      {/* Modal Create / Edit with Leaflet MapPicker */}
      <Dialog open={isModalOpen} onClose={() => setIsModalOpen(false)} size="lg">
        <DialogHeader>
          <DialogTitle>{editingLocation ? "Edit Lokasi Kantor" : "Tambah Lokasi Kantor Baru"}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <DialogContent className="space-y-4 max-h-[75vh] overflow-y-auto">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Nama Lokasi *</label>
                <Input
                  required
                  placeholder="Contoh: Kantor Pusat Sudirman"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">
                  Radius Geofence (Meter) *
                </label>
                <Input
                  type="number"
                  required
                  min={5}
                  max={5000}
                  value={radiusMeters}
                  onChange={(e) => setRadiusMeters(Number(e.target.value))}
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Alamat / Keterangan</label>
              <Input
                placeholder="Jl. Jenderal Sudirman No. Kav 1..."
                value={address}
                onChange={(e) => setAddress(e.target.value)}
              />
            </div>

            {/* Interactive Leaflet Map Picker (D22) */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="block text-xs font-semibold text-gray-700">
                  Titik Peta & Radius Toleransi (Geser Marker atau Klik Peta)
                </label>
                <span className="font-mono text-[11px] text-indigo-600 bg-indigo-50 px-2 py-0.5 rounded border border-indigo-200">
                  {latitude.toFixed(6)}, {longitude.toFixed(6)}
                </span>
              </div>

              <MapPicker
                latitude={latitude}
                longitude={longitude}
                radiusMeters={radiusMeters}
                onChange={(lat, lng) => {
                  setLatitude(lat);
                  setLongitude(lng);
                }}
                className="h-64 rounded-xl border border-gray-300 overflow-hidden"
              />
            </div>
          </DialogContent>

          <DialogFooter>
            <Button variant="outline" size="sm" type="button" onClick={() => setIsModalOpen(false)}>
              Batal
            </Button>
            <Button variant="primary" size="sm" type="submit" isLoading={isSubmitting}>
              {editingLocation ? "Simpan Perubahan" : "Tambah Lokasi"}
            </Button>
          </DialogFooter>
        </form>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={Boolean(deleteLocationId)}
        title="Hapus Lokasi Kantor?"
        description="Apakah Anda yakin ingin menghapus lokasi kantor ini? Karyawan yang terikat dengan lokasi ini tidak lagi memiliki batasan geofence default."
        confirmText="Hapus Lokasi"
        variant="danger"
        isLoading={isDeleting}
        onConfirm={handleDelete}
        onClose={() => setDeleteLocationId(null)}
      />
    </div>
  );
}
