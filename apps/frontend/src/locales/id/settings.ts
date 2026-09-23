// ============================================================
// BAHASA INDONESIA — settings namespace
// ============================================================
export interface SettingsTranslations {
    page: { title: string; subtitle: string; loading: string; saveAll: string; saved: string; error: string; };
    sections: { general: string; attendance: string; face: string; notification: string; security: string; };
    general: { companyName: string; companyNamePlaceholder: string; timezone: string; dateFormat: string; language: string; };
    attendance: { workStartTime: string; workEndTime: string; lateThreshold: string; earlyLeaveThreshold: string; requireApproval: string; allowRemote: string; };
    face: { similarityThreshold: string; qualityThreshold: string; livenessCheck: string; maxRetries: string; };
    security: { maxLoginAttempts: string; lockoutDuration: string; sessionTimeout: string; requireMfa: string; };
}

export const settings: SettingsTranslations = {
    page: {
        title: "Pengaturan Sistem",
        subtitle: "Konfigurasi parameter global aplikasi.",
        loading: "Memuat pengaturan...",
        saveAll: "Simpan Semua Perubahan",
        saved: "Pengaturan berhasil disimpan.",
        error: "Gagal menyimpan pengaturan.",
    },
    sections: {
        general: "Umum",
        attendance: "Absensi",
        face: "Pengenalan Wajah",
        notification: "Notifikasi",
        security: "Keamanan",
    },
    general: {
        companyName: "Nama Perusahaan",
        companyNamePlaceholder: "Contoh: PT Maju Bersama",
        timezone: "Zona Waktu",
        dateFormat: "Format Tanggal",
        language: "Bahasa Sistem",
    },
    attendance: {
        workStartTime: "Jam Masuk Kerja",
        workEndTime: "Jam Pulang Kerja",
        lateThreshold: "Batas Keterlambatan (menit)",
        earlyLeaveThreshold: "Batas Pulang Awal (menit)",
        requireApproval: "Perlu Persetujuan Admin",
        allowRemote: "Izinkan Absensi Remote",
    },
    face: {
        similarityThreshold: "Ambang Kesamaan Wajah",
        qualityThreshold: "Ambang Kualitas Foto",
        livenessCheck: "Pemeriksaan Liveness",
        maxRetries: "Maksimum Percobaan",
    },
    security: {
        maxLoginAttempts: "Maksimum Percobaan Login",
        lockoutDuration: "Durasi Kunci (menit)",
        sessionTimeout: "Timeout Sesi (jam)",
        requireMfa: "Wajib MFA",
    },
};
