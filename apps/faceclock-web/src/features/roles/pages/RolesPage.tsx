import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useAuth } from "../../../lib/auth/useAuth";
import { useToast } from "../../../components/ui/Toast";
import type { Role, Permission } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { Modal } from "../../../components/ui/Modal";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { LocalTime } from "../../../components/domain/LocalTime";
import { ShieldCheck, Plus, Key, Trash2, Lock, Users, Info } from "lucide-react";

export function RolesPage() {
  const { user: currentUser, can } = useAuth();
  const toast = useToast();

  // Dialogs
  const [createOpen, setCreateOpen] = useState(false);
  const [permOpen, setPermOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);

  // Create form state
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Permission selection state
  const [selectedPermIds, setSelectedPermIds] = useState<string[]>([]);

  // Fetch Roles
  const {
    data: roles,
    isLoading,
    refetch,
  } = useQuery<Role[]>({
    queryKey: ["roles"],
    queryFn: () => api.get<Role[]>("/api/v1/roles"),
  });

  // Fetch all permissions catalog
  const { data: allPerms } = useQuery<Permission[]>({
    queryKey: ["permissions"],
    queryFn: () => api.get<Permission[]>("/api/v1/permissions"),
  });

  // Group permissions by resource
  const permsByResource: Record<string, Permission[]> = {};
  (allPerms || []).forEach((p) => {
    const list = permsByResource[p.resource] || [];
    list.push(p);
    permsByResource[p.resource] = list;
  });

  // Handle Create Role
  const handleCreateRole = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !displayName.trim()) return;

    setIsSubmitting(true);
    try {
      await api.post("/api/v1/roles", {
        name: name.trim().toLowerCase(),
        display_name: displayName.trim(),
        description: description.trim() || undefined,
      });

      toast.show({
        type: "success",
        title: "Peran Berhasil Dibuat",
        message: `Peran ${displayName} telah ditambahkan ke sistem.`,
      });

      setCreateOpen(false);
      setName("");
      setDisplayName("");
      setDescription("");
      refetch();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Membuat Peran",
        message: "Periksa kembali format nama peran (harus unik).",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  // Open Permission Assignment Modal
  const openPermModal = async (r: Role) => {
    setSelectedRole(r);
    try {
      // Fetch current permissions for role
      const roleDetail = await api.get<Role>(`/api/v1/roles/${r.id}`);
      const currentIds = (roleDetail.permissions || []).map((p) => p.id);
      setSelectedPermIds(currentIds);
    } catch {
      // Fallback to empty if not fetched
      setSelectedPermIds([]);
    }
    setPermOpen(true);
  };

  // Save Permissions Replacement (PUT /roles/{id}/permissions)
  const handleSavePermissions = async () => {
    if (!selectedRole) return;
    setIsSubmitting(true);
    try {
      await api.put(`/api/v1/roles/${selectedRole.id}/permissions`, {
        permission_ids: selectedPermIds,
      });

      toast.show({
        type: "success",
        title: "Izin Peran Diperbarui",
        message: `Hak akses untuk peran ${selectedRole.display_name} berhasil disimpan.`,
      });

      setPermOpen(false);
      refetch();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Mengubah Izin",
        message:
          "Operasi ditolak. Anda tidak memiliki izin untuk memberikan hak yang Anda sendiri tidak miliki.",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle Delete Role
  const handleDeleteRole = async () => {
    if (!selectedRole) return;
    setIsSubmitting(true);
    try {
      await api.delete(`/api/v1/roles/${selectedRole.id}`);
      toast.show({
        type: "success",
        title: "Peran Dihapus",
        message: `Peran ${selectedRole.display_name} telah dihapus.`,
      });
      setDeleteOpen(false);
      refetch();
    } catch {
      toast.show({
        type: "error",
        title: "Gagal Menghapus Peran",
        message: "Peran sistem atau peran yang sedang digunakan oleh user tidak dapat dihapus.",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const isSuperAdmin = currentUser?.roles?.some((r: any) =>
    typeof r === "string" ? r === "super_admin" : r.name === "super_admin",
  );

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-5 h-5 text-indigo-600" />
            <h1 className="text-xl font-bold text-gray-900">Manajemen Peran & Hak Akses</h1>
          </div>
          <p className="text-xs text-gray-500 mt-0.5">
            Definisikan peran sistem (RBAC) dan atur matriks izin yang diperbolehkan untuk setiap tingkatan
            pengguna.
          </p>
        </div>

        {can("role.create") && (
          <Button variant="primary" size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="w-4 h-4 mr-1.5" />
            Tambah Peran Baru
          </Button>
        )}
      </div>

      {/* Privilege Escalation Info */}
      <div className="p-3.5 bg-blue-50 border border-blue-200 rounded-xl flex items-center gap-3 text-xs text-blue-900">
        <Info className="w-4 h-4 text-blue-600 shrink-0" />
        <div>
          <strong>Aturan Privilege Escalation (Rule B01/REV-PERM-01):</strong> Anda hanya dapat memberikan
          izin kepada peran lain sejauh izin tersebut dimiliki oleh akun Anda. Izin di luar wewenang Anda akan
          otomatis dinonaktifkan.
        </div>
      </div>

      {/* Roles Grid / Table */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full py-12 text-center text-xs text-gray-400">Memuat daftar peran...</div>
        ) : !roles || roles.length === 0 ? (
          <div className="col-span-full py-12 text-center text-xs text-gray-400">
            Belum ada peran terdaftar.
          </div>
        ) : (
          roles.map((r) => (
            <Card key={r.id} className="flex flex-col justify-between hover:border-gray-300 transition">
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <CardTitle className="text-sm font-bold text-gray-900">{r.display_name}</CardTitle>
                    <div className="font-mono text-[11px] text-gray-400 mt-0.5">{r.name}</div>
                  </div>

                  {r.is_system ? (
                    <Badge variant="neutral" size="sm">
                      <Lock className="w-3 h-3 mr-1" />
                      Sistem
                    </Badge>
                  ) : (
                    <Badge variant="info" size="sm">
                      Kustom
                    </Badge>
                  )}
                </div>
              </CardHeader>

              <CardContent className="space-y-3 pt-0">
                <p className="text-xs text-gray-600 min-h-[32px]">
                  {r.description || "Tidak ada deskripsi tambahan."}
                </p>

                <div className="flex items-center gap-4 text-xs text-gray-500 py-2 border-y border-gray-100">
                  <div className="flex items-center gap-1.5">
                    <Users className="w-3.5 h-3.5 text-gray-400" />
                    <span>{r.user_count ?? 0} Pengguna</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Key className="w-3.5 h-3.5 text-gray-400" />
                    <span>{r.permission_count ?? 0} Izin</span>
                  </div>
                </div>

                <div className="flex items-center justify-between pt-1">
                  <span className="text-[10px] text-gray-400">
                    Dibuat: <LocalTime value={r.created_at} format="date" />
                  </span>

                  <div className="flex items-center gap-1">
                    {can("role.assign_permission") && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openPermModal(r)}
                        className="text-xs"
                      >
                        <Key className="w-3.5 h-3.5 mr-1 text-indigo-600" />
                        Kelola Izin
                      </Button>
                    )}

                    {can("role.delete") && !r.is_system && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setSelectedRole(r);
                          setDeleteOpen(true);
                        }}
                        className="text-rose-600 hover:bg-rose-50"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </Button>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>

      {/* Modal: Create Role */}
      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="Buat Peran Kustom Baru">
        <form onSubmit={handleCreateRole} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1">
              Kode Unik Peran (Internal Name) *
            </label>
            <Input
              required
              placeholder="e.g. finance_staff, branch_supervisor"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <span className="text-[10px] text-gray-400 block mt-1">
              Gunakan huruf kecil dan garis bawah (underscore). Tidak bisa diubah setelah dibuat.
            </span>
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1">
              Nama Tampilan (Display Name) *
            </label>
            <Input
              required
              placeholder="e.g. Staf Keuangan, Supervisor Cabang"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-1">Deskripsi Tugas</label>
            <textarea
              className="w-full text-xs border border-gray-300 rounded-lg p-2.5 bg-white text-gray-800 focus:ring-1 focus:ring-indigo-500"
              rows={3}
              placeholder="Jelaskan ruang lingkup wewenang peran ini..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t border-gray-100">
            <Button type="button" variant="outline" size="sm" onClick={() => setCreateOpen(false)}>
              Batal
            </Button>
            <Button type="submit" variant="primary" size="sm" isLoading={isSubmitting}>
              Buat Peran
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal: Manage Permissions (Grouped by resource, REPLACE contract) */}
      <Modal
        open={permOpen}
        onClose={() => setPermOpen(false)}
        title={`Kelola Izin: ${selectedRole?.display_name}`}
      >
        <div className="space-y-4">
          <p className="text-xs text-gray-500">
            Centang hak akses yang ingin diberikan ke peran <strong>{selectedRole?.display_name}</strong>.
            Daftar izin ini menggantikan (replace) seluruh izin peran tersebut.
          </p>

          <div className="max-h-[60vh] overflow-y-auto space-y-4 pr-1">
            {Object.entries(permsByResource).map(([resource, perms]) => (
              <div key={resource} className="border border-gray-200 rounded-xl p-3 bg-gray-50/50">
                <div className="flex items-center justify-between mb-2 pb-1.5 border-b border-gray-200">
                  <span className="font-bold text-xs uppercase tracking-wider text-indigo-900">
                    Modul: {resource}
                  </span>
                  <span className="text-[10px] text-gray-400">{perms.length} opsi wewenang</span>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {perms.map((p) => {
                    // Check if current user has this permission to prevent privilege escalation
                    const userHasPerm = isSuperAdmin || can(p.name as any);
                    const isChecked = selectedPermIds.includes(p.id);

                    return (
                      <label
                        key={p.id}
                        className={`flex items-start gap-2 p-2 rounded-lg border text-xs transition ${
                          !userHasPerm
                            ? "bg-gray-100 border-gray-200 opacity-60 cursor-not-allowed"
                            : isChecked
                              ? "bg-indigo-50/60 border-indigo-200 text-indigo-950"
                              : "bg-white border-gray-200 text-gray-700 hover:border-gray-300 cursor-pointer"
                        }`}
                        title={
                          !userHasPerm ? "Anda tidak bisa memberikan izin yang tidak Anda miliki." : undefined
                        }
                      >
                        <input
                          type="checkbox"
                          disabled={!userHasPerm}
                          checked={isChecked}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedPermIds([...selectedPermIds, p.id]);
                            } else {
                              setSelectedPermIds(selectedPermIds.filter((id) => id !== p.id));
                            }
                          }}
                          className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 mt-0.5"
                        />
                        <div>
                          <div className="font-semibold text-[11px] font-mono">{p.name}</div>
                          <div className="text-[10px] text-gray-500 line-clamp-2">{p.description}</div>
                        </div>
                      </label>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t border-gray-100">
            <Button type="button" variant="outline" size="sm" onClick={() => setPermOpen(false)}>
              Batal
            </Button>
            <Button
              type="button"
              variant="primary"
              size="sm"
              isLoading={isSubmitting}
              onClick={handleSavePermissions}
            >
              Simpan Matriks Izin
            </Button>
          </div>
        </div>
      </Modal>

      {/* Confirm Delete Role */}
      <ConfirmDialog
        open={deleteOpen}
        title="Hapus Peran Pengguna"
        description={`Apakah Anda yakin ingin menghapus peran "${selectedRole?.display_name}"? Peran hanya dapat dihapus bila tidak ada pengguna yang sedang memakainya.`}
        confirmText="Hapus Peran"
        variant="danger"
        isLoading={isSubmitting}
        onConfirm={handleDeleteRole}
        onClose={() => setDeleteOpen(false)}
      />
    </div>
  );
}
