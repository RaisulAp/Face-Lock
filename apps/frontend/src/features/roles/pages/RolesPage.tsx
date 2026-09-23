import { useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { Role, Permission, Module } from "../../../types/api";
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
  Table,
} from "lucide-react";
import { PermissionMatrixModal } from "../components/PermissionMatrixModal";
import { MODULE_DEFINITIONS } from "../roleModules";

export function RolesPage() {
  const { t } = useTranslation(["role", "common"]);
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

  // Fetch system modules catalog
  const { data: modules } = useQuery<Module[]>({
    queryKey: ["modules"],
    queryFn: () => api.get<Module[]>("/api/v1/modules"),
  });

  // Open Permission Matrix Modal
  const openPermMatrixModal = async (r: Role) => {
    setSelectedRole(r);
    setMatrixOpen(true);
    try {
      const roleDetail = await api.get<Role>(`/api/v1/roles/${r.id}`);
      setSelectedRole(roleDetail);
      const currentIds = (roleDetail.permissions || []).map((p) => p.id);
      setSelectedPermIds(currentIds);
    } catch {
      const currentIds = (r.permissions || []).map((p) => p.id);
      setSelectedPermIds(currentIds);
    }
  };

  // Switch role inside matrix modal
  const handleSelectRoleInModal = async (r: Role) => {
    setSelectedRole(r);
    try {
      const roleDetail = await api.get<Role>(`/api/v1/roles/${r.id}`);
      setSelectedRole(roleDetail);
      const currentIds = (roleDetail.permissions || []).map((p) => p.id);
      setSelectedPermIds(currentIds);
    } catch {
      const currentIds = (r.permissions || []).map((p) => p.id);
      setSelectedPermIds(currentIds);
    }
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
              {t("page.title", { ns: "role" })}
            </h1>
          </div>
        </div>

        {roles && roles.length > 0 && (
          <Button
            type="button"
            variant="primary"
            size="sm"
            onClick={() => {
              const firstRole = roles[0];
              if (firstRole) openPermMatrixModal(firstRole);
            }}
            className="self-start sm:self-auto shadow-xs"
          >
            <Table className="w-4 h-4 mr-1.5" />
            {t("card.viewMatrix", { ns: "role" })}
          </Button>
        )}
      </div>

      {/* Search Bar */}
      <div className="p-3 sm:p-4 bg-white rounded-xl border border-gray-200 shadow-2xs flex flex-col sm:flex-row items-center justify-between gap-3">
        <div className="relative w-full sm:w-80">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <Input
            type="text"
            placeholder={t("page.searchPlaceholder", { ns: "role" })}
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
            <p className="text-sm font-semibold text-gray-700">{t("page.noRolesFound", { ns: "role" })}</p>
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
            const activeModulesCount = r.module_count ?? r.modules?.length ?? 0;
            const totalModulesCount = r.total_modules ?? modules?.length ?? 8;

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
                      {t("card.systemRole", { ns: "role" })}
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
                        {activeModulesCount}/{totalModulesCount} Modul ({coveragePercent}% • {permCount}/{totalPerms} Izin)
                      </span>
                    </div>

                    {r.modules && r.modules.length > 0 ? (
                      <div className="flex flex-wrap gap-1.5">
                        {r.modules.map((mod) => (
                          <span
                            key={mod.code}
                            className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[10px] font-medium bg-slate-100 text-slate-800 border border-slate-200 hover:bg-slate-200/80 transition-colors"
                            title={`${mod.name}: ${mod.active_permissions} dari ${mod.total_permissions} wewenang aktif`}
                          >
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
                            <span className="font-semibold text-gray-800">{mod.name}</span>
                            <span className="font-mono text-[9px] text-indigo-700 font-bold bg-white px-1 py-0.2 rounded border border-gray-200">
                              {mod.active_permissions}/{mod.total_permissions}
                            </span>
                          </span>
                        ))}
                      </div>
                    ) : coveredModules.length > 0 ? (
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
                      {t("card.viewMatrix", { ns: "role" })}
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
        roles={roles || []}
        onSelectRole={handleSelectRoleInModal}
        allPermissions={allPerms || []}
        selectedPermIds={selectedPermIds}
      />
    </div>
  );
}
