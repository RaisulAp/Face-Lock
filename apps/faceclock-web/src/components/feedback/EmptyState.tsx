import React from "react";
import { FolderOpen, SearchX, ShieldAlert } from "lucide-react";
import { Button } from "../ui/Button";

interface EmptyStateProps {
    variant?: "empty" | "not-found" | "forbidden";
    title?: string;
    description?: string;
    actionLabel?: string;
    onAction?: () => void;
    children?: React.ReactNode;
}

export function EmptyState({
    variant = "empty",
    title,
    description,
    actionLabel,
    onAction,
    children,
}: EmptyStateProps) {
    const defaults = {
        empty: {
            title: "Belum ada data",
            desc: "Data belum tersedia untuk kriteria yang dipilih.",
            icon: <FolderOpen className="w-10 h-10 text-gray-400" />,
        },
        "not-found": {
            title: "Tidak ditemukan",
            desc: "Data yang Anda cari tidak ada atau bukan milik Anda.",
            icon: <SearchX className="w-10 h-10 text-gray-400" />,
        },
        forbidden: {
            title: "Akses terbatas",
            desc: "Anda tidak memiliki izin yang cukup untuk melihat data ini.",
            icon: <ShieldAlert className="w-10 h-10 text-amber-500" />,
        },
    }[variant];

    return (
        <div className="flex flex-col items-center justify-center p-8 text-center bg-white rounded-xl border border-gray-100 my-4">
            <div className="p-3 bg-gray-50 rounded-full mb-3">{defaults.icon}</div>
            <h3 className="text-sm font-semibold text-gray-900">{title || defaults.title}</h3>
            <p className="text-xs text-gray-500 max-w-sm mt-1">{description || defaults.desc}</p>
            {actionLabel && onAction && (
                <div className="mt-4">
                    <Button size="sm" variant="secondary" onClick={onAction}>
                        {actionLabel}
                    </Button>
                </div>
            )}
            {children && <div className="mt-4">{children}</div>}
        </div>
    );
}
