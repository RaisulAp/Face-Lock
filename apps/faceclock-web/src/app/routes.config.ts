import type { LucideIcon } from "lucide-react";
import {
    LayoutDashboard,
    Clock,
    CheckSquare,
    BarChart3,
    Users,
    FileCheck,
    MapPin,
    UserCheck,
    Shield,
    Sliders,
    RefreshCw,
    ShieldAlert,
    History,
} from "lucide-react";
import type { Permission } from "../lib/permissions";

export interface NavItemConfig {
    path: string;
    label: string;
    icon: LucideIcon;
    permission: Permission;
    badgeKey?: "pendingCount";
}

export interface NavGroupConfig {
    title: string;
    items: NavItemConfig[];
}

export const NAVIGATION_CONFIG: NavGroupConfig[] = [
    {
        title: "Utama",
        items: [
            {
                path: "/dashboard",
                label: "Dashboard",
                icon: LayoutDashboard,
                permission: "attendance.read_all",
            },
            {
                path: "/attendances/pending",
                label: "Antrian Review",
                icon: CheckSquare,
                permission: "attendance.approve",
                badgeKey: "pendingCount",
            },
            {
                path: "/attendances",
                label: "Data Absensi",
                icon: Clock,
                permission: "attendance.read_all",
            },
            {
                path: "/reports",
                label: "Laporan & Rekap",
                icon: BarChart3,
                permission: "attendance.read_all",
            },
        ],
    },
    {
        title: "Manajemen Karyawan",
        items: [
            {
                path: "/employees",
                label: "Daftar Karyawan",
                icon: Users,
                permission: "employee.read",
            },
            {
                path: "/consents",
                label: "Persetujuan Biometrik",
                icon: FileCheck,
                permission: "face.read_any",
            },
        ],
    },
    {
        title: "Konfigurasi & Operasional",
        items: [
            {
                path: "/office-locations",
                label: "Lokasi Kantor",
                icon: MapPin,
                permission: "location.read",
            },
            {
                path: "/settings",
                label: "Pengaturan Sistem",
                icon: Sliders,
                permission: "settings.read",
            },
            {
                path: "/face/reindex",
                label: "Reindex Model Wajah",
                icon: RefreshCw,
                permission: "face.reindex",
            },
        ],
    },
    {
        title: "Keamanan & Akses",
        items: [
            {
                path: "/users",
                label: "Pengguna",
                icon: UserCheck,
                permission: "user.read",
            },
            {
                path: "/roles",
                label: "Role & Izin",
                icon: Shield,
                permission: "role.read",
            },
            {
                path: "/security/attempts",
                label: "Percobaan Gagal",
                icon: ShieldAlert,
                permission: "attendance.read_all",
            },
            {
                path: "/audit-logs",
                label: "Audit Log",
                icon: History,
                permission: "audit.read",
            },
        ],
    },
];
