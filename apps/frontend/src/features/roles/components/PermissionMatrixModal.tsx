import { useState, useMemo } from "react";
import {
    Clock,
    Users,
    ScanFace,
    MapPin,
    UserCheck,
    Shield,
    Sliders,
    FileText,
    Search,
    CheckCircle2,
    XCircle,
    Info,
    Layers,
    Lock,
} from "lucide-react";
import type { Role, Permission } from "../../../types/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import { groupPermissionsByDefinedModules } from "../roleModules";

interface PermissionMatrixModalProps {
    open: boolean;
    onClose: () => void;
    role: Role | null;
    allPermissions: Permission[];
    selectedPermIds: string[];
}

export function PermissionMatrixModal({
    open,
    onClose,
    role,
    allPermissions,
    selectedPermIds,
}: PermissionMatrixModalProps) {
    const [search, setSearch] = useState("");

    const grouped = useMemo(
        () => groupPermissionsByDefinedModules(allPermissions),
        [allPermissions]
    );

    const getModuleIcon = (key: string) => {
        switch (key) {
            case "attendance":
                return <Clock className="w-5 h-5 text-emerald-600" />;
            case "employee":
                return <Users className="w-5 h-5 text-blue-600" />;
            case "face":
                return <ScanFace className="w-5 h-5 text-violet-600" />;
            case "location":
                return <MapPin className="w-5 h-5 text-amber-600" />;
            case "user":
                return <UserCheck className="w-5 h-5 text-indigo-600" />;
            case "role":
                return <Shield className="w-5 h-5 text-purple-600" />;
            case "settings":
                return <Sliders className="w-5 h-5 text-slate-600" />;
            case "audit":
                return <FileText className="w-5 h-5 text-rose-600" />;
            default:
                return <Layers className="w-5 h-5 text-gray-600" />;
        }
    };

    // Filter groups based on search query
    const filteredGroups = useMemo(() => {
        const q = search.trim().toLowerCase();
        if (!q) return grouped;

        return grouped
            .map((g) => {
                const matchesModule =
                    g.module.title.toLowerCase().includes(q) ||
                    g.module.description.toLowerCase().includes(q);

                const matchedPerms = g.permissions.filter(
                    (p) =>
                        p.name.toLowerCase().includes(q) ||
                        p.friendlyLabel?.toLowerCase().includes(q) ||
                        p.friendlyDesc?.toLowerCase().includes(q)
                );

                if (matchesModule) {
                    return g;
                }

                if (matchedPerms.length > 0) {
                    return {
                        ...g,
                        permissions: matchedPerms,
                    };
                }

                return null;
            })
            .filter(Boolean) as typeof grouped;
    }, [grouped, search]);

    const totalPermsCount = allPermissions.length;
    const activeCount = selectedPermIds.length;
    const percentActive = totalPermsCount > 0 ? Math.round((activeCount / totalPermsCount) * 100) : 0;

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={`Hak Akses: ${role?.display_name || role?.name}`}
            description="Daftar wewenang operasional untuk peran sistem ini. Hak akses peran sistem bersifat baku dan terlindungi."
            maxWidth="3xl"
        >
            <div className="space-y-4 max-h-[75vh] flex flex-col -mx-2 sm:-mx-6 -mb-6">
                {/* Header Summary Bar */}
                <div className="px-4 sm:px-6 space-y-3 shrink-0">
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3.5 bg-gradient-to-r from-indigo-50/70 via-purple-50/50 to-blue-50/50 rounded-xl border border-indigo-100">
                        <div className="flex items-center gap-3">
                            <div className="w-12 h-12 rounded-xl bg-indigo-600 text-white flex items-center justify-center font-bold text-sm shadow-xs shrink-0">
                                {percentActive}%
                            </div>
                            <div>
                                <div className="text-sm font-bold text-gray-900 flex items-center gap-2">
                                    <span>{activeCount} dari {totalPermsCount} Izin Sistem Aktif</span>
                                    <Badge variant="neutral" size="sm">
                                        <Lock className="w-3 h-3 mr-1" />
                                        Peran Sistem
                                    </Badge>
                                </div>
                                <div className="text-xs text-gray-500 mt-0.5">
                                    Kode Peran: <span className="font-mono font-semibold text-indigo-700">{role?.name}</span>
                                </div>
                            </div>
                        </div>

                        <div className="text-xs text-gray-500 bg-white/80 px-3 py-2 rounded-lg border border-indigo-100">
                            {role?.description || "Peran sistem baku FaceClock."}
                        </div>
                    </div>

                    {/* Search bar */}
                    <div className="relative">
                        <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                        <Input
                            type="text"
                            placeholder="Cari nama izin atau nama modul (misal: 'presensi', 'karyawan', 'ekspor')..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="pl-9 text-xs"
                        />
                    </div>

                    <div className="flex items-center gap-2 p-2.5 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-700">
                        <Info className="w-4 h-4 text-slate-500 shrink-0" />
                        <span>
                            <strong>Peran Akses Baku:</strong> Peran ini adalah peran sistem baku yang hak aksesnya diatur langsung oleh arsitektur platform demi menjamin integritas & keamanan.
                        </span>
                    </div>
                </div>

                {/* Modules Permission List (Scrollable Area) */}
                <div className="flex-1 overflow-y-auto px-4 sm:px-6 space-y-4 pr-3">
                    {filteredGroups.length === 0 ? (
                        <div className="py-12 text-center text-gray-400 text-xs">
                            Tidak ada modul atau izin yang sesuai dengan pencarian &quot;{search}&quot;.
                        </div>
                    ) : (
                        filteredGroups.map(({ module: mod, permissions }) => {
                            const modulePermIds = permissions.map((p) => p.id);
                            const grantedCount = modulePermIds.filter((id) =>
                                selectedPermIds.includes(id)
                            ).length;
                            const totalInMod = modulePermIds.length;
                            const allModGranted = totalInMod > 0 && grantedCount === totalInMod;

                            return (
                                <div
                                    key={mod.key}
                                    className={`border rounded-xl transition overflow-hidden bg-white ${allModGranted
                                        ? "border-indigo-300 shadow-2xs"
                                        : grantedCount > 0
                                            ? "border-gray-300"
                                            : "border-gray-200"
                                        }`}
                                >
                                    {/* Module Header Bar */}
                                    <div className={`p-3 sm:p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b ${allModGranted ? "bg-indigo-50/40 border-indigo-100" : "bg-gray-50/70 border-gray-100"
                                        }`}>
                                        <div className="flex items-center gap-3">
                                            <div className="p-2 rounded-lg bg-white border border-gray-200 shadow-2xs">
                                                {getModuleIcon(mod.key)}
                                            </div>
                                            <div>
                                                <div className="flex items-center gap-2 flex-wrap">
                                                    <h3 className="font-bold text-xs sm:text-sm text-gray-900">
                                                        Modul {mod.title}
                                                    </h3>
                                                    {allModGranted ? (
                                                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-100 text-emerald-800">
                                                            <CheckCircle2 className="w-3 h-3" />
                                                            Akses Penuh ({grantedCount}/{totalInMod})
                                                        </span>
                                                    ) : grantedCount > 0 ? (
                                                        <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-blue-100 text-blue-800">
                                                            Akses Sebagian ({grantedCount}/{totalInMod})
                                                        </span>
                                                    ) : (
                                                        <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-gray-100 text-gray-500">
                                                            Tidak Ada Akses (0/{totalInMod})
                                                        </span>
                                                    )}
                                                </div>
                                                <p className="text-[11px] text-gray-500 mt-0.5">
                                                    {mod.description}
                                                </p>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Module Permissions Grid */}
                                    <div className="p-3 sm:p-4 grid grid-cols-1 md:grid-cols-2 gap-2.5">
                                        {permissions.map((p) => {
                                            const isGranted = selectedPermIds.includes(p.id);

                                            return (
                                                <div
                                                    key={p.id}
                                                    className={`p-3 rounded-xl border text-xs flex items-start gap-3 ${isGranted
                                                        ? "bg-indigo-50/50 border-indigo-200 shadow-2xs"
                                                        : "bg-gray-50/50 border-gray-200 opacity-60"
                                                        }`}
                                                >
                                                    <div className="mt-0.5 shrink-0">
                                                        {isGranted ? (
                                                            <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                                                        ) : (
                                                            <XCircle className="w-4 h-4 text-gray-400" />
                                                        )}
                                                    </div>
                                                    <div className="flex-1 min-w-0">
                                                        <div className="flex items-center justify-between gap-1">
                                                            <span className={`font-semibold text-xs ${isGranted ? "text-gray-900" : "text-gray-500"}`}>
                                                                {p.friendlyLabel || p.name}
                                                            </span>
                                                            <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${isGranted
                                                                ? "bg-emerald-100 text-emerald-800"
                                                                : "bg-gray-200 text-gray-600"
                                                                }`}>
                                                                {isGranted ? "Diberikan" : "Tidak Aktif"}
                                                            </span>
                                                        </div>
                                                        <div className="font-mono text-[10px] text-gray-400 mt-0.5">
                                                            {p.name}
                                                        </div>
                                                        <p className="text-[11px] text-gray-500 mt-1 leading-snug">
                                                            {p.friendlyDesc || p.description}
                                                        </p>
                                                    </div>
                                                </div>
                                            );
                                        })}
                                    </div>
                                </div>
                            );
                        })
                    )}
                </div>

                {/* Footer Actions */}
                <div className="px-4 sm:px-6 pt-3 pb-4 border-t border-gray-100 flex items-center justify-between bg-gray-50/50 shrink-0">
                    <div className="text-xs text-gray-500">
                        Total wewenang aktif: <span className="font-bold text-gray-900">{activeCount}</span> izin.
                    </div>
                    <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={onClose}
                    >
                        Tutup
                    </Button>
                </div>
            </div>
        </Modal>
    );
}
