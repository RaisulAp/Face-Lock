import React from "react";
import { TableSkeleton } from "../feedback/Skeletons";
import { EmptyState } from "../feedback/EmptyState";

export interface Column<T> {
    key: string;
    header: React.ReactNode;
    render?: (row: T) => React.ReactNode;
    className?: string;
    align?: "left" | "center" | "right";
}

interface DataTableProps<T> {
    columns: Column<T>[];
    data: T[];
    keyExtractor: (row: T) => string;
    isLoading?: boolean;
    emptyTitle?: string;
    emptyDescription?: string;
    onRowClick?: (row: T) => void;
    selectable?: boolean;
    selectedIds?: string[];
    onToggleSelect?: (id: string) => void;
    onToggleSelectAll?: () => void;
    isAllSelected?: boolean;
}

export function DataTable<T>({
    columns,
    data,
    keyExtractor,
    isLoading = false,
    emptyTitle,
    emptyDescription,
    onRowClick,
    selectable = false,
    selectedIds = [],
    onToggleSelect,
    onToggleSelectAll,
    isAllSelected = false,
}: DataTableProps<T>) {
    if (isLoading) {
        return (
            <div className="p-4 bg-white rounded-xl border border-gray-200/80">
                <TableSkeleton rows={5} cols={columns.length + (selectable ? 1 : 0)} />
            </div>
        );
    }

    if (data.length === 0) {
        return (
            <EmptyState
                title={emptyTitle || "Tidak ada data"}
                description={emptyDescription || "Belum ada rekaman yang sesuai dengan filter."}
            />
        );
    }

    return (
        <div className="overflow-x-auto rounded-xl border border-gray-200/80 bg-white shadow-xs">
            <table className="min-w-full divide-y divide-gray-200 text-left text-sm">
                <thead className="bg-gray-50/80 text-xs font-semibold text-gray-600">
                    <tr>
                        {selectable && (
                            <th className="w-10 px-4 py-3">
                                <input
                                    type="checkbox"
                                    checked={isAllSelected}
                                    onChange={onToggleSelectAll}
                                    className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                                />
                            </th>
                        )}
                        {columns.map((col) => (
                            <th key={col.key} className={`px-4 py-3 ${col.className || ""}`}>
                                {col.header}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                    {data.map((row) => {
                        const id = keyExtractor(row);
                        const isSelected = selectedIds.includes(id);

                        return (
                            <tr
                                key={id}
                                onClick={() => onRowClick?.(row)}
                                className={`transition-colors ${onRowClick ? "cursor-pointer hover:bg-indigo-50/30" : "hover:bg-gray-50/50"
                                    } ${isSelected ? "bg-indigo-50/50" : ""}`}
                            >
                                {selectable && (
                                    <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                                        <input
                                            type="checkbox"
                                            checked={isSelected}
                                            onChange={() => onToggleSelect?.(id)}
                                            className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                                        />
                                    </td>
                                )}
                                {columns.map((col) => (
                                    <td key={col.key} className={`px-4 py-3 text-gray-700 ${col.className || ""}`}>
                                        {col.render ? col.render(row) : (row as Record<string, unknown>)[col.key] as React.ReactNode}
                                    </td>
                                ))}
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
}
