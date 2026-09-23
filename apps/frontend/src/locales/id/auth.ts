// ============================================================
// BAHASA INDONESIA — auth namespace
// ============================================================
export interface AuthTranslations {
    login: {
        title: string; subtitle: string; emailLabel: string; emailPlaceholder: string;
        passwordLabel: string; passwordPlaceholder: string; submitButton: string;
        submittingButton: string; forgotPassword: string; errorDefault: string;
    };
    account: {
        title: string; subtitle: string; changePassword: string; currentPassword: string;
        newPassword: string; confirmNewPassword: string; passwordMismatch: string;
        passwordChanged: string; activeSessions: string; revokeSession: string;
        revokeAllOther: string; thisSession: string; lastActive: string;
    };
    forbidden: { title: string; subtitle: string; backToDashboard: string; };
    notFound: { title: string; subtitle: string; backToHome: string; };
}

export const auth: AuthTranslations = {
    login: {
        title: "Masuk ke Akun",
        subtitle: "Gunakan email dan kata sandi yang telah terdaftar.",
        emailLabel: "Email",
        emailPlaceholder: "nama@perusahaan.com",
        passwordLabel: "Kata Sandi",
        passwordPlaceholder: "••••••••",
        submitButton: "Masuk",
        submittingButton: "Sedang Masuk...",
        forgotPassword: "Lupa kata sandi?",
        errorDefault: "Gagal masuk. Silakan periksa koneksi atau kredensial Anda.",
    },
    account: {
        title: "Akun & Sesi",
        subtitle: "Kelola kata sandi dan sesi login Anda.",
        changePassword: "Ganti Kata Sandi",
        currentPassword: "Kata Sandi Saat Ini",
        newPassword: "Kata Sandi Baru",
        confirmNewPassword: "Konfirmasi Kata Sandi Baru",
        passwordMismatch: "Kata sandi baru tidak cocok.",
        passwordChanged: "Kata sandi berhasil diubah.",
        activeSessions: "Sesi Aktif",
        revokeSession: "Cabut Sesi",
        revokeAllOther: "Cabut Semua Sesi Lain",
        thisSession: "Sesi ini",
        lastActive: "Terakhir aktif",
    },
    forbidden: {
        title: "Akses Ditolak",
        subtitle: "Anda tidak memiliki izin untuk mengakses halaman ini.",
        backToDashboard: "Kembali ke Dashboard",
    },
    notFound: {
        title: "Halaman Tidak Ditemukan",
        subtitle: "Halaman yang Anda cari tidak tersedia.",
        backToHome: "Kembali ke Beranda",
    },
};
