import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import type { Permission } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { KeyRound, Search, Shield } from "lucide-react";

export function PermissionsPage() {
  const [search, setSearch] = useState("");

  const { data: perms, isLoading } = useQuery<Permission[]>({
    queryKey: ["permissions"],
    queryFn: () => api.get<Permission[]>("/api/v1/permissions"),
  });

  const filteredPerms = (perms || []).filter(
    (p) =>
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.resource.toLowerCase().includes(search.toLowerCase()) ||
      p.action.toLowerCase().includes(search.toLowerCase()) ||
      (p.description ? p.description.toLowerCase().includes(search.toLowerCase()) : false),
  );

  // Group by resource
  const permsByResource: Record<string, Permission[]> = {};
  filteredPerms.forEach((p) => {
    const list = permsByResource[p.resource] || [];
    list.push(p);
    permsByResource[p.resource] = list;
  });

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Header */}
      <div>
        <div className="flex items-center gap-2">
          <KeyRound className="w-5 h-5 text-indigo-600" />
          <h1 className="text-xl font-bold text-gray-900">Katalog Izin Sistem (Permissions)</h1>
        </div>
        <p className="text-xs text-gray-500 mt-0.5">
          Daftar referensi seluruh hak akses (permissions) granular yang ada pada sistem FaceClock.
        </p>
      </div>

      {/* Search filter */}
      <Card>
        <CardContent className="py-3">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <Input
              placeholder="Cari izin berdasarkan nama, modul sumber daya, atau deskripsi..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
        </CardContent>
      </Card>

      {/* Permissions List Grouped */}
      {isLoading ? (
        <div className="py-12 text-center text-xs text-gray-400">Memuat katalog izin sistem...</div>
      ) : Object.keys(permsByResource).length === 0 ? (
        <div className="py-12 text-center text-xs text-gray-400">
          Tidak ada izin sistem yang cocok dengan kriteria pencarian.
        </div>
      ) : (
        <div className="space-y-6">
          {Object.entries(permsByResource).map(([resource, list]) => (
            <Card key={resource}>
              <CardHeader className="py-3 bg-gray-50/70 border-b border-gray-200">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Shield className="w-4 h-4 text-indigo-600" />
                    <CardTitle className="text-xs font-bold uppercase tracking-wider text-gray-900">
                      Modul: {resource}
                    </CardTitle>
                  </div>
                  <Badge variant="neutral" size="sm">
                    {list.length} Izin
                  </Badge>
                </div>
              </CardHeader>

              <CardContent className="p-0">
                <div className="overflow-x-auto">
                  <table className="w-full text-xs text-left">
                    <thead className="bg-gray-50/40 text-gray-500 border-b border-gray-100">
                      <tr>
                        <th className="px-4 py-2 font-semibold w-64">Nama Izin</th>
                        <th className="px-4 py-2 font-semibold w-32">Aksi</th>
                        <th className="px-4 py-2 font-semibold">Deskripsi Wewenang</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {list.map((p) => (
                        <tr key={p.id} className="hover:bg-gray-50/50">
                          <td className="px-4 py-2.5 font-mono font-semibold text-indigo-900">{p.name}</td>
                          <td className="px-4 py-2.5">
                            <Badge variant="outline" size="sm">
                              {p.action}
                            </Badge>
                          </td>
                          <td className="px-4 py-2.5 text-gray-600">{p.description}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
