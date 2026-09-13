import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { Role, Permission } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { LocalTime } from "../../../components/domain/LocalTime";
import {
  ShieldCheck,
  Key,
  Lock,
  Users,
  Search,
  Shield,
  Layers,
  CheckCircle2,
} from "lucide-react";
import { PermissionMatrixModal } from "../components/PermissionMatrixModal";
import { MODULE_DEFINITIONS } from "../roleModules";

export function RolesPage() {
  const [matrixOpen, setMatrixOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);

  // Search
  const [searchQuery, setSearchQuery] = useState("");

  // Permission selection state for matrix modal
  const [selectedPermIds, setSelectedPermIds] = useState<string[]>([]);

  // Fetch Roles
  const {
    data: roles,
    isLoading,
  } = useQuery<Role[]>({
    queryKey: ["roles"],
    queryFn: () => api.get<Role[]>("/api/v1/roles"),
  });

  // Fetch all permissions catalog
  const { data: allPerms } = useQuery<Permission[]>({
    queryKey: ["permissions"],
    queryFn: () => api.get<Permission[]>("/api/v1/permissions"),
  });

  // Open Permission Matrix Modal
  const openPermMatrixModal = async (r: Role) => {
    setSelectedRole(r);
    try {
      const roleDetail = await api.get<Role>(`/api/v1/roles/${r.id}`);
      const currentIds = (roleDetail.permissions || []).map((p) => p.id);
      setSelectedPermIds(currentIds);
    } catch {
      setSelectedPermIds([]);
    }
    setMatrixOpen(true);
  };

  // Filtered Roles List
  const filteredRoles = useMemo(() => {
    if (!roles) return [];
    return roles.filter((r) => {
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase();
        const matchesName = r.name.toLowerCase().includes(q);
        const matchesDisplay = (r.display_name || "").toLowerCase().includes(q);
        const matchesDesc = (r.description || "").toLowerCase().includes(q);
        return matchesName || matchesDisplay || matchesDesc;
      }
      return true;
    });
  }, [roles, searchQuery]);

  // Statistics
  const stats = useMemo(() => {
    if (!roles) return { total: 0, system: 0, users: 0 };
    return {
      total: roles.length,
      system: roles.filter((r) => r.is_system).length,
      users: roles.reduce((acc, r) => acc + (r.user_count || 0), 0),
    };
  }, [roles]);

  // Helper to find modules covered by role's permissions
  const getCoveredModules = (role: Role) => {
    if (!role.permissions || role.permissions.length === 0) return [];
    const permNames = new Set(role.permissions.map((p) => p.name));
    return MODULE_DEFINITIONS.filter((mod) =>
      mod.permissions.some((p) => permNames.has(p.name))
    );
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Header Banner */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-6 h-6 text-indigo-600" />
            <h1 className="text-xl sm:text-2xl font-bold text-gray-900">
              Tingkatan Peran & Hak Akses (RBAC)
            </h1>
          </div>
          <p className="text-xs sm:text-sm text-gray-500 mt-1">
            Arsitektur hak akses FaceClock menggunakan 3 peran sistem baku. Setiap akun pengguna memiliki tepat 1 peran akses.
          </p>
        </div>
      </div>

      {/* Metric Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        <div className="p-4 rounded-xl bg-white border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Peran Sistem</span>
            <Lock className="w-4 h-4 text-indigo-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900 mt-2">{stats.total}</div>
          <div className="text-[11px] text-gray-400 mt-0.5">Tingkatan wewenang baku platform</div>
        </div>

        <div className="p-4 rounded-xl bg-white border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Total Pengguna</span>
            <Users className="w-4 h-4 text-blue-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900 mt-2">{stats.users}</div>
          <div className="text-[11px] text-gray-400 mt-0.5">Pengguna terdaftar dengan peran aktif</div>
        </div>

        <div className="p-4 rounded-xl bg-white border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Katalog Izin Sistem</span>
            <Layers className="w-4 h-4 text-emerald-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900 mt-2">
            {allPerms?.length || 39} Izin
          </div>
          <div className="text-[11px] text-gray-400 mt-0.5">
            Dikelompokkan ke {MODULE_DEFINITIONS.length} modul bisnis
          </div>
        </div>

        <div className="p-4 rounded-xl bg-white border border-gray-200 shadow-2xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-500">Prinsip Akses</span>
            <CheckCircle2 className="w-4 h-4 text-purple-600" />
          </div>
          <div className="text-sm font-bold text-gray-900 mt-2">1 User = 1 Role</div>
          <div className="text-[11px] text-gray-400 mt-0.5">Akses tunggal, ketat & aman</div>
        </div>
      </div>

      {/* Search Bar */}
      <div className="p-3 sm:p-4 bg-white rounded-xl border border-gray-200 shadow-2xs flex flex-col sm:flex-row items-center justify-between gap-3">
        <div className="relative w-full sm:w-80">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <Input
            type="text"
            placeholder="Cari peran berdasarkan nama atau deskripsi..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9 text-xs"
          />
        </div>

        <div className="text-xs text-gray-500">
          Menampilkan <span className="font-semibold text-gray-900">{filteredRoles.length}</span> peran akses
        </div>
      </div>

      {/* Roles Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full py-16 text-center text-xs text-gray-400">
            Memuat daftar peran & hak akses...
          </div>
        ) : filteredRoles.length === 0 ? (
          <div className="col-span-full py-16 text-center bg-white rounded-xl border border-gray-200 p-8 space-y-2">
            <Shield className="w-8 h-8 text-gray-300 mx-auto" />
            <p className="text-sm font-semibold text-gray-700">Tidak ada peran yang cocok</p>
            <p className="text-xs text-gray-400 max-w-sm mx-auto">
              Tidak ditemukan peran dengan kata kunci &quot;{searchQuery}&quot;.
            </p>
          </div>
        ) : (
          filteredRoles.map((r) => {
            const coveredModules = getCoveredModules(r);
            const totalPerms = allPerms?.length || 39;
            const permCount = r.permission_count ?? r.permissions?.length ?? 0;
            const coveragePercent = Math.round((permCount / totalPerms) * 100);

            return (
              <Card
                key={r.id}
                className="flex flex-col justify-between hover:border-indigo-200 hover:shadow-xs transition duration-150"
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-2.5">
                      <div className="w-9 h-9 rounded-xl bg-indigo-50 border border-indigo-100 text-indigo-700 flex items-center justify-center font-bold text-xs shrink-0">
                        <Shield className="w-5 h-5 text-indigo-600" />
                      </div>
                      <div>
                        <CardTitle className="text-sm font-bold text-gray-900 leading-tight">
                          {r.display_name || r.name}
                        </CardTitle>
                        <div className="font-mono text-[10px] text-gray-400 mt-0.5">
                          {r.name}
                        </div>
                      </div>
                    </div>

                    <Badge variant="neutral" size="sm">
                      <Lock className="w-3 h-3 mr-1" />
                      Sistem Baku
                    </Badge>
                  </div>
                </CardHeader>

                <CardContent className="space-y-3.5 pt-0">
                  <p className="text-xs text-gray-600 min-h-[36px] line-clamp-2 leading-relaxed">
                    {r.description || "Tidak ada penjelasan wewenang khusus."}
                  </p>

                  {/* Module Access Badges */}
                  <div>
                    <div className="text-[11px] font-semibold text-gray-500 mb-1.5 flex items-center justify-between">
                      <span>Cakupan Modul Aktif:</span>
                      <span className="text-[10px] font-mono text-indigo-600 font-bold">
                        {permCount}/{totalPerms} Izin ({coveragePercent}%)
                      </span>
                    </div>

                    {coveredModules.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {coveredModules.map((mod) => (
                          <span
                            key={mod.key}
                            className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[10px] font-medium bg-gray-100 text-gray-700 border border-gray-200"
                            title={mod.description}
                          >
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                            {mod.title}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="text-[11px] text-gray-400 italic">
                        Belum ada modul yang terhubung
                      </span>
                    )}
                  </div>

                  {/* User count & timestamps */}
                  <div className="flex items-center justify-between text-xs text-gray-500 py-2 border-y border-gray-100">
                    <div className="flex items-center gap-1.5 font-medium">
                      <Users className="w-3.5 h-3.5 text-gray-400" />
                      <span>{r.user_count ?? 0} Pengguna Terdaftar</span>
                    </div>
                    <span className="text-[10px] text-gray-400">
                      Terdaftar: <LocalTime value={r.created_at} format="date" />
                    </span>
                  </div>

                  {/* Actions: View Permissions */}
                  <div className="pt-1">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openPermMatrixModal(r)}
                      className="text-xs w-full justify-center shadow-2xs hover:bg-indigo-50 hover:text-indigo-700 hover:border-indigo-300"
                    >
                      <Key className="w-3.5 h-3.5 mr-1 text-indigo-600" />
                      Lihat Rincian Hak Akses
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })
        )}
      </div>

      {/* Permission Matrix Modal (Read-Only Viewer) */}
      <PermissionMatrixModal
        open={matrixOpen}
        onClose={() => setMatrixOpen(false)}
        role={selectedRole}
        allPermissions={allPerms || []}
        selectedPermIds={selectedPermIds}
      />
    </div>
  );
}
