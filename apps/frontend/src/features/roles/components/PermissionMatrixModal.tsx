import { useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
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
    Check,
    CheckCircle2,
    XCircle,
    Info,
    Layers,
    Lock,
    Table as TableIcon,
    LayoutGrid,
} from "lucide-react";
import type { Role, Permission } from "../../../types/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Badge } from "../../../components/ui/Badge";
import {
    MATRIX_ACTION_COLUMNS,
    MATRIX_MODULE_GROUPS,
    getMatrixCellsForSubFeature,
    groupPermissionsByDefinedModules,
} from "../roleModules";

interface PermissionMatrixModalProps {
    open: boolean;
    onClose: () => void;
    role: Role | null;
    roles?: Role[];
    onSelectRole?: (role: Role) => void;
    allPermissions: Permission[];
    selectedPermIds?: string[];
}

export function PermissionMatrixModal({
    open,
    onClose,
    role,
    roles = [],
    onSelectRole,
    allPermissions,
    selectedPermIds = [],
}: PermissionMatrixModalProps) {
    const [search, setSearch] = useState("");
    const [selectedModuleFilter, setSelectedModuleFilter] = useState<string>("all");
    const [viewMode, setViewMode] = useState<"matrix" | "cards">("matrix");
    const { t } = useTranslation(["role", "common"]);

    // Identify current active role
    const activeRole = role;

    // Build Set of granted permission names for active role
    const assignedPermNames = useMemo(() => {
        if (activeRole?.permissions && activeRole.permissions.length > 0) {
            return new Set(activeRole.permissions.map((p) => p.name));
        }
        if (selectedPermIds && selectedPermIds.length > 0) {
            const idSet = new Set(selectedPermIds);
            return new Set(allPermissions.filter((p) => idSet.has(p.id)).map((p) => p.name));
        }
        return new Set<string>();
    }, [activeRole, selectedPermIds, allPermissions]);

    // Icon helper
    const getModuleIcon = (key: string, className = "w-4 h-4") => {
        switch (key) {
            case "attendance":
                return <Clock className={`${className} text-emerald-600`} />;
            case "employee":
                return <Users className={`${className} text-blue-600`} />;
            case "face":
                return <ScanFace className={`${className} text-violet-600`} />;
            case "location":
                return <MapPin className={`${className} text-amber-600`} />;
            case "user":
                return <UserCheck className={`${className} text-indigo-600`} />;
            case "role":
                return <Shield className={`${className} text-purple-600`} />;
            case "settings":
                return <Sliders className={`${className} text-slate-600`} />;
            case "audit":
                return <FileText className={`${className} text-rose-600`} />;
            default:
                return <Layers className={`${className} text-gray-600`} />;
        }
    };

    // Filter matrix module groups by search and module filter
    const filteredMatrixGroups = useMemo(() => {
        const q = search.trim().toLowerCase();

        return MATRIX_MODULE_GROUPS.map((group) => {
            // Check module tab filter
            if (selectedModuleFilter !== "all" && group.code !== selectedModuleFilter) {
                return null;
            }

            if (!q) {
                return group;
            }

            const moduleMatches =
                group.name.toLowerCase().includes(q) ||
                group.code.toLowerCase().includes(q) ||
                group.description.toLowerCase().includes(q);

            const matchedSubFeatures = group.subFeatures.filter((sub) => {
                const subMatches =
                    sub.name.toLowerCase().includes(q) ||
                    sub.description.toLowerCase().includes(q);

                const permMatches = Object.values(sub.permissionsByAction).some((pName) => {
                    return pName.toLowerCase().includes(q);
                });

                return subMatches || permMatches;
            });

            if (moduleMatches) {
                return group;
            }

            if (matchedSubFeatures.length > 0) {
                return {
                    ...group,
                    subFeatures: matchedSubFeatures,
                };
            }

            return null;
        }).filter(Boolean) as typeof MATRIX_MODULE_GROUPS;
    }, [search, selectedModuleFilter]);

    // For cards view fallback
    const groupedCards = useMemo(
        () => groupPermissionsByDefinedModules(allPermissions),
        [allPermissions]
    );

    const filteredCards = useMemo(() => {
        const q = search.trim().toLowerCase();
        if (!q) return groupedCards;

        return groupedCards
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

                if (matchesModule) return g;
                if (matchedPerms.length > 0) {
                    return {
                        ...g,
                        permissions: matchedPerms,
                    };
                }
                return null;
            })
            .filter(Boolean) as typeof groupedCards;
    }, [groupedCards, search]);

    // Statistics
    const totalPermsCount = allPermissions.length || 39;
    const activeCount = assignedPermNames.size;
    const percentActive = totalPermsCount > 0 ? Math.round((activeCount / totalPermsCount) * 100) : 0;

    // Active modules count
    const activeModulesCount = useMemo(() => {
        return MATRIX_MODULE_GROUPS.filter((group) => {
            return group.subFeatures.some((sub) =>
                Object.values(sub.permissionsByAction).some((pName) =>
                    assignedPermNames.has(pName)
                )
            );
        }).length;
    }, [assignedPermNames]);

    const totalModulesCount = MATRIX_MODULE_GROUPS.length;

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={t("matrix.title", { ns: "role" })}
            description={t("matrix.subtitle", { ns: "role" })}
            maxWidth="6xl"
        >
            <div className="space-y-4 max-h-[82vh] flex flex-col -mx-2 sm:-mx-6 -mb-6">
                {/* 1. Header & Role Selector Bar */}
                <div className="px-4 sm:px-6 space-y-3 shrink-0">
                    {/* Role Switcher Toolbar */}
                    {roles.length > 0 && (
                        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-2.5 bg-slate-50 rounded-xl border border-slate-200">
                            <div className="flex items-center gap-2">
                                <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
                                    {t("matrix.roleLabel", { ns: "role" })}:
                                </span>
                                <div className="inline-flex rounded-lg border border-gray-200 bg-white p-0.5 shadow-2xs">
                                    {roles.map((r) => {
                                        const isCurrent =
                                            r.id === activeRole?.id || r.name === activeRole?.name;
                                        const rolePermCount =
                                            r.permission_count ?? r.permissions?.length ?? 0;

                                        return (
                                            <button
                                                key={r.id}
                                                type="button"
                                                onClick={() => onSelectRole?.(r)}
                                                className={`flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-all cursor-pointer ${isCurrent
                                                    ? "bg-indigo-600 text-white font-semibold shadow-xs"
                                                    : "text-gray-600 hover:text-gray-900 hover:bg-gray-100"
                                                    }`}
                                            >
                                                <Shield
                                                    className={`w-3.5 h-3.5 ${isCurrent ? "text-white" : "text-gray-400"
                                                        }`}
                                                />
                                                <span>{r.display_name || r.name}</span>
                                                <span
                                                    className={`text-[10px] px-1.5 py-0.2 rounded-full font-mono ${isCurrent
                                                        ? "bg-indigo-700/80 text-white"
                                                        : "bg-gray-100 text-gray-600"
                                                        }`}
                                                >
                                                    {rolePermCount}/{totalPermsCount}
                                                </span>
                                            </button>
                                        );
                                    })}
                                </div>
                            </div>

                            {/* View Switcher (Matrix vs Cards) */}
                            <div className="flex items-center gap-1 border border-gray-200 bg-white rounded-lg p-0.5 shadow-2xs self-start sm:self-auto">
                                <button
                                    type="button"
                                    onClick={() => setViewMode("matrix")}
                                    className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer ${viewMode === "matrix"
                                        ? "bg-indigo-50 text-indigo-700 font-semibold border border-indigo-200"
                                        : "text-gray-500 hover:text-gray-800"
                                        }`}
                                    title="Tampilan Kisi-Kisi Matriks (Matrix Grid)"
                                >
                                    <TableIcon className="w-3.5 h-3.5" />
                                    <span>Tabel Matriks</span>
                                </button>
                                <button
                                    type="button"
                                    onClick={() => setViewMode("cards")}
                                    className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer ${viewMode === "cards"
                                        ? "bg-indigo-50 text-indigo-700 font-semibold border border-indigo-200"
                                        : "text-gray-500 hover:text-gray-800"
                                        }`}
                                    title="Tampilan Rincian Kartu Per Modul"
                                >
                                    <LayoutGrid className="w-3.5 h-3.5" />
                                    <span>Kartu Modul</span>
                                </button>
                            </div>
                        </div>
                    )}

                    {/* Active Role Summary Banner & Matrix Legend */}
                    <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 p-3.5 bg-gradient-to-r from-indigo-50/70 via-purple-50/50 to-blue-50/50 rounded-xl border border-indigo-100">
                        <div className="flex items-center gap-3.5">
                            <div className="w-12 h-12 rounded-xl bg-indigo-600 text-white flex flex-col items-center justify-center font-bold text-sm shadow-xs shrink-0">
                                <span>{percentActive}%</span>
                                <span className="text-[9px] font-normal text-indigo-200 uppercase -mt-0.5">Izin</span>
                            </div>
                            <div>
                                <div className="text-sm font-bold text-gray-900 flex items-center gap-2">
                                    <span>{activeRole?.display_name || activeRole?.name}</span>
                                    <Badge variant="neutral" size="sm">
                                        <Lock className="w-3 h-3 mr-1" />
                                        Peran Sistem
                                    </Badge>
                                </div>
                                <div className="text-xs text-gray-600 mt-0.5 flex items-center gap-2 flex-wrap">
                                    <span>
                                        Kode:{" "}
                                        <span className="font-mono font-semibold text-indigo-700">
                                            {activeRole?.name}
                                        </span>
                                    </span>
                                    <span>•</span>
                                    <span className="font-semibold text-emerald-700 bg-emerald-50 px-1.5 py-0.5 rounded border border-emerald-200">
                                        {activeModulesCount} dari {totalModulesCount} Modul Bisnis Aktif
                                    </span>
                                    <span>•</span>
                                    <span className="font-semibold text-indigo-700 bg-indigo-50 px-1.5 py-0.5 rounded border border-indigo-200">
                                        {activeCount} dari {totalPermsCount} Total Wewenang
                                    </span>
                                </div>
                            </div>
                        </div>

                        {/* Legend Guide */}
                        <div className="flex items-center gap-3 text-xs bg-white/90 px-3 py-2 rounded-lg border border-indigo-100 self-start md:self-auto shrink-0 shadow-2xs">
                            <span className="text-[11px] font-semibold text-gray-500 uppercase tracking-wider">
                                {t("matrix.legend.title", { ns: "role" })}:
                            </span>
                            <div className="flex items-center gap-1.5">
                                <div className="w-4 h-4 rounded bg-indigo-600 text-white flex items-center justify-center">
                                    <Check className="w-3 h-3 stroke-[3]" />
                                </div>
                                <span className="text-[11px] text-gray-700 font-medium">{t("matrix.legend.granted", { ns: "role" })}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <div className="w-4 h-4 rounded border border-gray-300 bg-white" />
                                <span className="text-[11px] text-gray-500">{t("matrix.legend.notGranted", { ns: "role" })}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <span className="text-gray-300 font-mono text-xs font-bold">—</span>
                                <span className="text-[11px] text-gray-400">N/A</span>
                            </div>
                        </div>
                    </div>

                    {/* Search & Quick Filter Pills */}
                    <div className="flex flex-col sm:flex-row sm:items-center gap-2.5">
                        <div className="relative flex-1">
                            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                            <Input
                                type="text"
                                placeholder={t("matrix.searchPlaceholder", { ns: "role" })}
                                value={search}
                                onChange={(e) => setSearch(e.target.value)}
                                className="pl-9 pr-8 text-xs"
                            />
                            {search && (
                                <button
                                    type="button"
                                    onClick={() => setSearch("")}
                                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs font-semibold cursor-pointer"
                                >
                                    ✕
                                </button>
                            )}
                        </div>

                        {/* Quick filter pills */}
                        <div className="flex items-center gap-1 overflow-x-auto pb-1 sm:pb-0 text-xs">
                            <button
                                type="button"
                                onClick={() => setSelectedModuleFilter("all")}
                                className={`px-2.5 py-1 rounded-md text-[11px] font-medium whitespace-nowrap transition-colors cursor-pointer ${selectedModuleFilter === "all"
                                    ? "bg-gray-800 text-white"
                                    : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                                    }`}
                            >
                                Semua Modul
                            </button>
                            {MATRIX_MODULE_GROUPS.map((m) => (
                                <button
                                    key={m.code}
                                    type="button"
                                    onClick={() => setSelectedModuleFilter(m.code)}
                                    className={`px-2 py-1 rounded-md text-[11px] font-medium whitespace-nowrap transition-colors cursor-pointer ${selectedModuleFilter === m.code
                                        ? "bg-indigo-600 text-white"
                                        : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                                        }`}
                                >
                                    {m.name.split("&")[0]?.trim() || m.name}
                                </button>
                            ))}
                        </div>
                    </div>
                </div>

                {/* 2. Main Content Area */}
                <div className="flex-1 overflow-y-auto px-4 sm:px-6 space-y-4 pr-3">
                    {viewMode === "matrix" ? (
                        /* MATRIX GRID TABLE VIEW */
                        <div className="border border-gray-200 rounded-xl overflow-hidden shadow-2xs bg-white">
                            <div className="overflow-x-auto">
                                <table className="w-full text-left border-collapse min-w-[760px]">
                                    {/* Action Columns Table Header */}
                                    <thead className="bg-slate-50 sticky top-0 z-20 border-b border-gray-200 text-[11px] font-semibold text-gray-700 tracking-wider">
                                        <tr>
                                            <th className="py-3 px-4 w-72 min-w-[250px] sticky left-0 bg-slate-50 z-30 border-r border-gray-200 shadow-[2px_0_4px_-2px_rgba(0,0,0,0.08)]">
                                                Modul & Fitur Operasional
                                            </th>
                                            {MATRIX_ACTION_COLUMNS.map((col) => (
                                                <th
                                                    key={col.key}
                                                    className="py-3 px-2 text-center w-32 min-w-[105px] border-r border-gray-200 last:border-r-0"
                                                    title={col.description}
                                                >
                                                    <div className="flex flex-col items-center">
                                                        <span className="font-bold text-gray-800">
                                                            {col.label}
                                                        </span>
                                                        <span className="text-[10px] font-normal text-gray-400 mt-0.5 line-clamp-1">
                                                            {col.shortLabel}
                                                        </span>
                                                    </div>
                                                </th>
                                            ))}
                                        </tr>
                                    </thead>

                                    {filteredMatrixGroups.length === 0 ? (
                                        <tbody className="text-xs">
                                            <tr>
                                                <td
                                                    colSpan={MATRIX_ACTION_COLUMNS.length + 1}
                                                    className="py-12 text-center text-gray-400 text-xs"
                                                >
                                                    Tidak ada modul atau aksi yang cocok dengan kata kunci &quot;{search}&quot;.
                                                </td>
                                            </tr>
                                        </tbody>
                                    ) : (
                                        filteredMatrixGroups.map((group) => {
                                            // Count granted in this module
                                            const allGroupPermNames = group.subFeatures.flatMap((s) =>
                                                Object.values(s.permissionsByAction)
                                            );
                                            const grantedInGroup = allGroupPermNames.filter((p) =>
                                                assignedPermNames.has(p)
                                            ).length;
                                            const totalInGroup = allGroupPermNames.length;
                                            const isFullAccess =
                                                totalInGroup > 0 && grantedInGroup === totalInGroup;

                                            return (
                                                <tbody key={group.code} className="border-t-2 border-gray-200 divide-y divide-gray-100 text-xs">
                                                    {/* Module Group Banner Row */}
                                                    <tr className="bg-slate-100/90 font-semibold text-gray-800 border-b border-gray-200">
                                                        <td
                                                            colSpan={MATRIX_ACTION_COLUMNS.length + 1}
                                                            className="py-2.5 px-4 sticky left-0 bg-slate-100/90 z-10"
                                                        >
                                                            <div className="flex items-center justify-between">
                                                                <div className="flex items-center gap-2.5">
                                                                    <div className="p-1 rounded bg-white border border-gray-200 shadow-2xs">
                                                                        {getModuleIcon(group.code, "w-4 h-4")}
                                                                    </div>
                                                                    <span className="font-bold text-xs text-gray-900">
                                                                        Modul {group.name}
                                                                    </span>
                                                                    <span className="text-[11px] font-normal text-gray-500 hidden sm:inline">
                                                                        — {group.description}
                                                                    </span>
                                                                </div>

                                                                <div className="flex items-center gap-2">
                                                                    <span
                                                                        className={`text-[10px] px-2 py-0.5 rounded-full font-semibold border ${isFullAccess
                                                                            ? "bg-emerald-100 text-emerald-800 border-emerald-200"
                                                                            : grantedInGroup > 0
                                                                                ? "bg-blue-100 text-blue-800 border-blue-200"
                                                                                : "bg-gray-200 text-gray-600 border-gray-300"
                                                                            }`}
                                                                    >
                                                                        {grantedInGroup}/{totalInGroup} Wewenang Aktif
                                                                    </span>
                                                                </div>
                                                            </div>
                                                        </td>
                                                    </tr>

                                                    {/* Sub-Feature Rows */}
                                                    {group.subFeatures.map((sub, sIdx) => {
                                                        const cells = getMatrixCellsForSubFeature(
                                                            sub,
                                                            allPermissions,
                                                            assignedPermNames
                                                        );

                                                        return (
                                                            <tr
                                                                key={sub.id}
                                                                className={`hover:bg-indigo-50/40 transition-colors border-b border-gray-100 ${sIdx % 2 === 1
                                                                    ? "bg-slate-50/40"
                                                                    : "bg-white"
                                                                    }`}
                                                            >
                                                                {/* Left Column: Sub-feature Name & Desc */}
                                                                <td className="py-2.5 px-4 sticky left-0 bg-inherit z-10 border-r border-gray-200 shadow-[2px_0_4px_-2px_rgba(0,0,0,0.06)]">
                                                                    <div className="font-semibold text-gray-900 text-xs">
                                                                        {sub.name}
                                                                    </div>
                                                                    <div className="text-[10px] text-gray-500 mt-0.5 line-clamp-1 leading-tight">
                                                                        {sub.description}
                                                                    </div>
                                                                </td>

                                                                {/* Action Columns Cells */}
                                                                {cells.map((cell) => (
                                                                    <td
                                                                        key={cell.actionKey}
                                                                        className="py-2.5 px-2 text-center align-middle border-r border-gray-100 last:border-r-0"
                                                                    >
                                                                        {cell.exists ? (
                                                                            <div className="flex justify-center items-center">
                                                                                {cell.isGranted ? (
                                                                                    /* Checkbox Checked / Granted */
                                                                                    <div
                                                                                        className="inline-flex items-center justify-center w-5 h-5 rounded bg-indigo-600 text-white shadow-2xs cursor-help transition-transform hover:scale-110"
                                                                                        title={`${cell.label || cell.permissionName} (${cell.permissionName}) - Diberikan untuk ${activeRole?.display_name || activeRole?.name}`}
                                                                                    >
                                                                                        <Check className="w-3.5 h-3.5 stroke-[3]" />
                                                                                    </div>
                                                                                ) : (
                                                                                    /* Checkbox Unchecked / Disabled */
                                                                                    <div
                                                                                        className="inline-flex items-center justify-center w-5 h-5 rounded border border-gray-300 bg-white hover:border-gray-400 cursor-help transition-colors"
                                                                                        title={`${cell.label || cell.permissionName} (${cell.permissionName}) - Tidak Aktif untuk ${activeRole?.display_name || activeRole?.name}`}
                                                                                    />
                                                                                )}
                                                                            </div>
                                                                        ) : (
                                                                            /* Not Applicable */
                                                                            <span
                                                                                className="text-gray-300 font-mono text-xs select-none"
                                                                                title="Aksi tidak berlaku pada sub-fitur ini"
                                                                            >
                                                                                —
                                                                            </span>
                                                                        )}
                                                                    </td>
                                                                ))}
                                                            </tr>
                                                        );
                                                    })}
                                                </tbody>
                                            );
                                        })
                                    )}
                                </table>
                            </div>
                        </div>
                    ) : (
                        /* CARDS VIEW (ALTERNATIVE VIEW) */
                        <div className="space-y-4">
                            {filteredCards.length === 0 ? (
                                <div className="py-12 text-center text-gray-400 text-xs">
                                    Tidak ada modul atau izin yang sesuai dengan pencarian &quot;{search}&quot;.
                                </div>
                            ) : (
                                filteredCards.map(({ module: mod, permissions }) => {
                                    const grantedCount = permissions.filter((p) =>
                                        assignedPermNames.has(p.name)
                                    ).length;
                                    const totalInMod = permissions.length;
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
                                            <div
                                                className={`p-3 sm:p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b ${allModGranted
                                                    ? "bg-indigo-50/40 border-indigo-100"
                                                    : "bg-gray-50/70 border-gray-100"
                                                    }`}
                                            >
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

                                            <div className="p-3 sm:p-4 grid grid-cols-1 md:grid-cols-2 gap-2.5">
                                                {permissions.map((p) => {
                                                    const isGranted = assignedPermNames.has(p.name);

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
                                                                    <span
                                                                        className={`font-semibold text-xs ${isGranted ? "text-gray-900" : "text-gray-500"
                                                                            }`}
                                                                    >
                                                                        {p.friendlyLabel || p.name}
                                                                    </span>
                                                                    <span
                                                                        className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${isGranted
                                                                            ? "bg-emerald-100 text-emerald-800"
                                                                            : "bg-gray-200 text-gray-600"
                                                                            }`}
                                                                    >
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
                    )}

                    <div className="flex items-center gap-2 p-2.5 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-700">
                        <Info className="w-4 h-4 text-slate-500 shrink-0" />
                        <span>
                            <strong>Peran Sistem Baku:</strong> Wewenang pada peran ini dikontrol langsung oleh arsitektur platform FaceClock untuk menjamin integritas presensi, biometrik, dan keamanan data.
                        </span>
                    </div>
                </div>

                {/* 3. Footer Actions Bar */}
                <div className="px-4 sm:px-6 pt-3 pb-4 border-t border-gray-100 flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-gray-50/50 shrink-0">
                    <div className="text-xs text-gray-500 flex items-center gap-2">
                        <span>
                            Peran: <strong className="text-gray-900">{activeRole?.display_name || activeRole?.name}</strong>
                        </span>
                        <span>•</span>
                        <span>
                            Total Wewenang Aktif:{" "}
                            <span className="font-bold text-indigo-700 font-mono">{activeCount}</span> dari{" "}
                            <span className="font-mono">{totalPermsCount}</span> izin ({percentActive}%)
                        </span>
                    </div>
                    <Button type="button" variant="outline" size="sm" onClick={onClose}>
                        Tutup
                    </Button>
                </div>
            </div>
        </Modal>
    );
}

