// ============================================================
// BAHASA INDONESIA — audit namespace
// ============================================================
export interface AuditTranslations {
    page: { title: string; subtitle: string; loading: string; noData: string; searchPlaceholder: string; };
    filters: { allActions: string; allActors: string; dateFrom: string; dateTo: string; resource: string; };
    table: { timestamp: string; actor: string; action: string; resource: string; resourceId: string; ipAddress: string; userAgent: string; details: string; status: string; };
    actions: { view: string; export: string; };
    detail: { title: string; before: string; after: string; metadata: string; noChanges: string; };
    messages: { exportSuccess: string; exportError: string; };
}

export const audit: AuditTranslations = {
    page: {
        title: "Audit Log Aktivitas",
        subtitle: "Rekam jejak semua aktivitas penting dalam sistem.",
        loading: "Memuat audit log...",
        noData: "Tidak ada audit log ditemukan.",
        searchPlaceholder: "Cari aktivitas, pengguna, atau resource...",
    },
    filters: {
        allActions: "Semua Aksi",
        allActors: "Semua Pengguna",
        dateFrom: "Dari Tanggal",
        dateTo: "Sampai Tanggal",
        resource: "Resource",
    },
    table: {
        timestamp: "Waktu",
        actor: "Pengguna",
        action: "Aksi",
        resource: "Resource",
        resourceId: "ID Resource",
        ipAddress: "IP Address",
        userAgent: "Browser/Device",
        details: "Detail",
        status: "Status",
    },
    actions: {
        view: "Lihat Detail",
        export: "Ekspor Log",
    },
    detail: {
        title: "Detail Audit Log",
        before: "Sebelum",
        after: "Sesudah",
        metadata: "Metadata",
        noChanges: "Tidak ada perubahan tercatat.",
    },
    messages: {
        exportSuccess: "Audit log berhasil diekspor.",
        exportError: "Gagal mengekspor audit log.",
    },
};
