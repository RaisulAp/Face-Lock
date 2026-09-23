// ============================================================
// BAHASA INDONESIA — nav namespace
// ============================================================
export interface NavTranslations {
    groups: {
        utama: string; manajemenKaryawan: string;
        konfigurasiOperasional: string; keamananAkses: string;
    };
    items: {
        dashboard: string; antrianReview: string; dataAbsensi: string; laporanRekap: string;
        daftarKaryawan: string; pendaftaranWajah: string; persetujuanBiometrik: string;
        lokasiKantor: string; pengaturanSistem: string; reindexModelWajah: string;
        pengguna: string; roleIzin: string; percobaanGagal: string; auditLog: string;
    };
}

export const nav: NavTranslations = {
    groups: {
        utama: "Utama",
        manajemenKaryawan: "Manajemen Karyawan",
        konfigurasiOperasional: "Konfigurasi & Operasional",
        keamananAkses: "Keamanan & Akses",
    },
    items: {
        dashboard: "Dashboard",
        antrianReview: "Antrian Review",
        dataAbsensi: "Data Absensi",
        laporanRekap: "Laporan & Rekap",
        daftarKaryawan: "Daftar Karyawan",
        pendaftaranWajah: "Pendaftaran Wajah",
        persetujuanBiometrik: "Persetujuan Biometrik",
        lokasiKantor: "Lokasi Kantor",
        pengaturanSistem: "Pengaturan Sistem",
        reindexModelWajah: "Reindex Model Wajah",
        pengguna: "Pengguna",
        roleIzin: "Role & Izin",
        percobaanGagal: "Percobaan Gagal",
        auditLog: "Audit Log",
    },
};
