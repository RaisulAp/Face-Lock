// ============================================================
// BAHASA INDONESIA — face namespace
// ============================================================
export interface FaceTranslations {
    reindex: {
        title: string; subtitle: string; description: string; startButton: string;
        running: string; success: string; error: string; lastRun: string;
        status: { idle: string; running: string; completed: string; failed: string; };
        stats: { totalFaces: string; indexed: string; failed: string; duration: string; };
    };
    consent: {
        page: { title: string; subtitle: string; loading: string; noData: string; };
        status: { given: string; pending: string; revoked: string; };
        actions: { giveConsent: string; revokeConsent: string; };
        messages: { giveSuccess: string; revokeSuccess: string; error: string; };
    };
}

export const face: FaceTranslations = {
    reindex: {
        title: "Reindex Model Wajah",
        subtitle: "Proses ini akan memperbarui indeks wajah dari semua data biometrik terdaftar.",
        description: "Gunakan fitur ini jika data wajah tidak akurat atau setelah penambahan karyawan baru dalam jumlah besar.",
        startButton: "Mulai Reindex",
        running: "Reindex Sedang Berjalan...",
        success: "Reindex berhasil diselesaikan.",
        error: "Reindex gagal. Silakan coba lagi.",
        lastRun: "Terakhir dijalankan",
        status: {
            idle: "Siap",
            running: "Berjalan",
            completed: "Selesai",
            failed: "Gagal",
        },
        stats: {
            totalFaces: "Total Wajah Terdaftar",
            indexed: "Berhasil Diindex",
            failed: "Gagal",
            duration: "Durasi",
        },
    },
    consent: {
        page: {
            title: "Persetujuan Biometrik",
            subtitle: "Kelola persetujuan penggunaan data biometrik karyawan.",
            loading: "Memuat data persetujuan...",
            noData: "Tidak ada data persetujuan.",
        },
        status: {
            given: "Sudah Diberikan",
            pending: "Menunggu",
            revoked: "Dicabut",
        },
        actions: {
            giveConsent: "Berikan Persetujuan",
            revokeConsent: "Cabut Persetujuan",
        },
        messages: {
            giveSuccess: "Persetujuan berhasil diberikan.",
            revokeSuccess: "Persetujuan berhasil dicabut.",
            error: "Gagal memproses persetujuan.",
        },
    },
};
