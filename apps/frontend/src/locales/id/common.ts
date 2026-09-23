// ============================================================
// BAHASA INDONESIA — common namespace (SOURCE OF TRUTH)
// ============================================================

// Explicit interface — values are string so EN can satisfy it with any string
export interface CommonTranslations {
    appName: string;
    appTagline: string;
    adminPanel: string;
    actions: {
        save: string; cancel: string; close: string; delete: string; edit: string;
        add: string; create: string; update: string; search: string; filter: string;
        reset: string; refresh: string; export: string; import: string; view: string;
        back: string; next: string; previous: string; submit: string; confirm: string;
        yes: string; no: string; ok: string; apply: string; loading: string;
        retry: string; download: string; upload: string; select: string; clear: string;
    };
    status: {
        active: string; inactive: string; pending: string; approved: string; rejected: string;
        all: string; enabled: string; disabled: string; success: string; failed: string;
        loading: string; error: string; empty: string; unknown: string;
    };
    pagination: {
        page: string; of: string; perPage: string; showing: string; to: string;
        total: string; first: string; last: string; entries: string;
    };
    time: {
        today: string; yesterday: string; thisWeek: string; thisMonth: string;
        date: string; time: string; dateTime: string; createdAt: string; updatedAt: string;
    };
    confirm: {
        deleteTitle: string; deleteMessage: string; yesDelete: string; noCancel: string;
    };
    messages: {
        saveSuccess: string; deleteSuccess: string; updateSuccess: string; createSuccess: string;
        errorGeneric: string; noData: string; networkError: string;
    };
    environment: { development: string; production: string; };
    nav: {
        system: string; changePassword: string; employeePortal: string;
        auditLog: string; logout: string;
    };
}

export const common: CommonTranslations = {
    appName: "FaceClock",
    appTagline: "Sistem Absensi Biometrik & Manajemen Kehadiran",
    adminPanel: "Admin Panel",
    actions: {
        save: "Simpan", cancel: "Batal", close: "Tutup", delete: "Hapus", edit: "Ubah",
        add: "Tambah", create: "Buat", update: "Perbarui", search: "Cari", filter: "Filter",
        reset: "Reset", refresh: "Segarkan", export: "Ekspor", import: "Impor", view: "Lihat",
        back: "Kembali", next: "Lanjut", previous: "Sebelumnya", submit: "Kirim", confirm: "Konfirmasi",
        yes: "Ya", no: "Tidak", ok: "OK", apply: "Terapkan", loading: "Memuat...",
        retry: "Coba Lagi", download: "Unduh", upload: "Unggah", select: "Pilih", clear: "Hapus Filter",
    },
    status: {
        active: "Aktif", inactive: "Nonaktif", pending: "Menunggu", approved: "Disetujui", rejected: "Ditolak",
        all: "Semua", enabled: "Diaktifkan", disabled: "Dinonaktifkan", success: "Berhasil", failed: "Gagal",
        loading: "Memuat", error: "Error", empty: "Tidak ada data", unknown: "Tidak diketahui",
    },
    pagination: {
        page: "Halaman", of: "dari", perPage: "per halaman", showing: "Menampilkan", to: "–",
        total: "total", first: "Pertama", last: "Terakhir", entries: "data",
    },
    time: {
        today: "Hari ini", yesterday: "Kemarin", thisWeek: "Minggu ini", thisMonth: "Bulan ini",
        date: "Tanggal", time: "Jam", dateTime: "Tanggal & Jam", createdAt: "Dibuat pada", updatedAt: "Diperbarui pada",
    },
    confirm: {
        deleteTitle: "Konfirmasi Hapus", deleteMessage: "Tindakan ini tidak dapat dibatalkan. Lanjutkan?",
        yesDelete: "Ya, Hapus", noCancel: "Tidak, Batal",
    },
    messages: {
        saveSuccess: "Berhasil disimpan.", deleteSuccess: "Berhasil dihapus.", updateSuccess: "Berhasil diperbarui.",
        createSuccess: "Berhasil dibuat.", errorGeneric: "Terjadi kesalahan. Silakan coba lagi.",
        noData: "Tidak ada data ditemukan.", networkError: "Gagal terhubung ke server.",
    },
    environment: { development: "Development", production: "Production" },
    nav: {
        system: "FaceClock System", changePassword: "Ganti Password & Sesi",
        employeePortal: "Portal Absensi Karyawan", auditLog: "Audit Log Aktivitas", logout: "Keluar (Logout)",
    },
};
