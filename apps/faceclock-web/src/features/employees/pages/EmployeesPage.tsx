import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useAuth } from "../../../lib/auth/useAuth";
import type { Employee, SuccessEnvelope } from "../../../types/api";
import { DataTable } from "../../../components/data/DataTable";
import type { Column } from "../../../components/data/DataTable";
import { Pagination } from "../../../components/data/Pagination";
import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Users, UserPlus, Search, Edit2, ScanFace } from "lucide-react";

export function EmployeesPage() {
    const { can } = useAuth();
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(20);
    const [search, setSearch] = useState("");

    const queryParams = new URLSearchParams();
    queryParams.set("page", page.toString());
    queryParams.set("per_page", perPage.toString());
    if (search.trim()) queryParams.set("search", search.trim());

    const { data, isLoading } = useQuery<SuccessEnvelope<Employee[]>>({
        queryKey: ["employees", "list", page, perPage, search],
        queryFn: () => api.getWithMeta<Employee[]>(`/api/v1/employees?${queryParams.toString()}`),
    });

    const employees = data?.data ?? [];
    const meta = data?.meta;

    const columns: Column<Employee>[] = [
        {
            key: "nip",
            header: "NIP",
            render: (row) => (
                <span className="font-mono font-semibold text-gray-900 text-xs">{row.nip}</span>
            ),
        },
        {
            key: "name",
            header: "Nama Lengkap",
            render: (row) => (
                <div>
                    <p className="font-semibold text-gray-900">{row.name}</p>
                    <span className="text-[11px] text-gray-400">{row.email || "-"}</span>
                </div>
            ),
        },
        {
            key: "department",
            header: "Departemen / Jabatan",
            render: (row) => (
                <div>
                    <p className="text-gray-800 font-medium">{row.department || "-"}</p>
                    <span className="text-[11px] text-gray-400">{row.position || "-"}</span>
                </div>
            ),
        },
        {
            key: "status",
            header: "Status",
            render: (row) => (
                <Badge variant={row.is_active ? "success" : "neutral"} size="sm">
                    {row.is_active ? "Aktif" : "Non-aktif"}
                </Badge>
            ),
        },
        {
            key: "actions",
            header: "Aksi",
            align: "right",
            render: (row) => (
                <div className="flex items-center justify-end gap-1.5">
                    {can("face.read_any") && (
                        <Link to={`/employees/${row.id}/face`}>
                            <Button variant="ghost" size="sm" title="Status Biometrik Wajah">
                                <ScanFace className="w-3.5 h-3.5 text-indigo-600 mr-1" />
                                Biometrik
                            </Button>
                        </Link>
                    )}

                    {can("employee.update") && (
                        <Link to={`/employees/${row.id}/edit`}>
                            <Button variant="outline" size="sm">
                                <Edit2 className="w-3.5 h-3.5 mr-1" />
                                Edit
                            </Button>
                        </Link>
                    )}
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
                        <Users className="w-5 h-5 text-indigo-600" />
                        <h1 className="text-xl font-bold text-gray-900">Manajemen Karyawan</h1>
                    </div>
                    <p className="text-xs text-gray-500 mt-0.5">
                        Daftar profil karyawan, data identitas, dan status pendaftaran wajah.
                    </p>
                </div>

                {can("employee.create") && (
                    <Link to="/employees/new">
                        <Button variant="primary" size="sm">
                            <UserPlus className="w-4 h-4 mr-1.5" />
                            Tambah Karyawan
                        </Button>
                    </Link>
                )}
            </div>

            {/* Search Toolbar */}
            <div className="bg-white p-4 rounded-xl border border-gray-200/80 shadow-2xs flex items-center gap-3">
                <div className="relative flex-1 max-w-sm">
                    <Input
                        placeholder="Cari berdasarkan NIP atau nama..."
                        value={search}
                        onChange={(e) => {
                            setSearch(e.target.value);
                            setPage(1);
                        }}
                        className="pl-9"
                    />
                    <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                </div>
            </div>

            {/* Table */}
            <DataTable
                columns={columns}
                data={employees}
                keyExtractor={(item) => item.id}
                isLoading={isLoading}
                emptyTitle="Belum Ada Karyawan"
                emptyDescription="Karyawan yang didaftarkan akan muncul di sini."
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
