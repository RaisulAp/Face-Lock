import type { NavTranslations } from "../id/nav";

export const nav = {
    groups: {
        utama: "Main",
        manajemenKaryawan: "Employee Management",
        konfigurasiOperasional: "Configuration & Operations",
        keamananAkses: "Security & Access",
    },
    items: {
        dashboard: "Dashboard",
        antrianReview: "Review Queue",
        dataAbsensi: "Attendance Data",
        laporanRekap: "Reports & Summary",
        daftarKaryawan: "Employee List",
        pendaftaranWajah: "Face Registration",
        persetujuanBiometrik: "Biometric Consent",
        lokasiKantor: "Office Locations",
        pengaturanSistem: "System Settings",
        reindexModelWajah: "Reindex Face Model",
        pengguna: "Users",
        roleIzin: "Roles & Permissions",
        percobaanGagal: "Failed Attempts",
        auditLog: "Audit Log",
    },
} satisfies NavTranslations;
