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
  label: string;       // kept for backwards compat / non-i18n usage
  labelKey: string;    // i18n key under nav.items.*
  icon: LucideIcon;
  permission: Permission | Permission[];
  badgeKey?: "pendingCount";
}

export interface NavGroupConfig {
  title: string;       // kept for backwards compat
  titleKey: string;    // i18n key under nav.groups.*
  items: NavItemConfig[];
}

export const NAVIGATION_CONFIG: NavGroupConfig[] = [
  {
    title: "Utama",
    titleKey: "groups.utama",
    items: [
      {
        path: "/dashboard",
        label: "Dashboard",
        labelKey: "items.dashboard",
        icon: LayoutDashboard,
        permission: [
          "attendance.read_all",
          "attendance.review",
          "employee.read",
          "user.read",
          "role.read",
          "location.read",
          "settings.read",
          "audit.read",
        ],
      },
      {
        path: "/attendances/pending",
        label: "Antrian Review",
        labelKey: "items.antrianReview",
        icon: CheckSquare,
        permission: ["attendance.review", "attendance.approve"],
        badgeKey: "pendingCount",
      },
      {
        path: "/attendances",
        label: "Data Absensi",
        labelKey: "items.dataAbsensi",
        icon: Clock,
        permission: "attendance.read_all",
      },
      {
        path: "/reports",
        label: "Laporan & Rekap",
        labelKey: "items.laporanRekap",
        icon: BarChart3,
        permission: ["attendance.export", "attendance.read_all"],
      },
    ],
  },
  {
    title: "Manajemen Karyawan",
    titleKey: "groups.manajemenKaryawan",
    items: [
      {
        path: "/employees",
        label: "Daftar Karyawan",
        labelKey: "items.daftarKaryawan",
        icon: Users,
        permission: "employee.read",
      },
      {
        path: "/consents",
        label: "Persetujuan Biometrik",
        labelKey: "items.persetujuanBiometrik",
        icon: FileCheck,
        permission: ["face.read_any", "consent.read"],
      },
    ],
  },
  {
    title: "Konfigurasi & Operasional",
    titleKey: "groups.konfigurasiOperasional",
    items: [
      {
        path: "/locations",
        label: "Lokasi Kantor",
        labelKey: "items.lokasiKantor",
        icon: MapPin,
        permission: "location.read",
      },
      {
        path: "/settings",
        label: "Pengaturan Sistem",
        labelKey: "items.pengaturanSistem",
        icon: Sliders,
        permission: ["settings.read", "setting.read"],
      },
      {
        path: "/face/reindex",
        label: "Reindex Model Wajah",
        labelKey: "items.reindexModelWajah",
        icon: RefreshCw,
        permission: "face.reindex",
      },
    ],
  },
  {
    title: "Keamanan & Akses",
    titleKey: "groups.keamananAkses",
    items: [
      {
        path: "/users",
        label: "Pengguna",
        labelKey: "items.pengguna",
        icon: UserCheck,
        permission: "user.read",
      },
      {
        path: "/roles",
        label: "Role & Izin",
        labelKey: "items.roleIzin",
        icon: Shield,
        permission: "role.read",
      },
      {
        path: "/security/attempts",
        label: "Percobaan Gagal",
        labelKey: "items.percobaanGagal",
        icon: ShieldAlert,
        permission: ["attendance.read_all", "attempt.read"],
      },
      {
        path: "/audit-logs",
        label: "Audit Log",
        labelKey: "items.auditLog",
        icon: History,
        permission: "audit.read",
      },
    ],
  },
];
