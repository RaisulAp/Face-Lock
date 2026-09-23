// ============================================================
// BAHASA INDONESIA — user namespace
// ============================================================
export interface UserTranslations {
    page: { title: string; subtitle: string; createUser: string; searchPlaceholder: string; noUsersFound: string; loading: string; };
    filters: { allRoles: string; allStatus: string; allTypes: string; active: string; inactive: string; employee: string; standalone: string; };
    table: { name: string; email: string; role: string; status: string; type: string; lastLogin: string; createdAt: string; actions: string; neverLoggedIn: string; linkedToEmployee: string; standaloneUser: string; };
    actions: { view: string; edit: string; assignRoles: string; resetPassword: string; activate: string; deactivate: string; delete: string; };
    form: { createTitle: string; editTitle: string; fullNameLabel: string; fullNamePlaceholder: string; emailLabel: string; emailPlaceholder: string; passwordLabel: string; passwordPlaceholder: string; typeLabel: string; statusLabel: string; employeeLabel: string; saveButton: string; cancelButton: string; };
    detail: { title: string; personalInfo: string; accountInfo: string; roles: string; sessions: string; linkedEmployee: string; noLinkedEmployee: string; };
    credentials: { title: string; subtitle: string; email: string; temporaryPassword: string; warning: string; copyEmail: string; copyPassword: string; copied: string; close: string; };
    roles: { title: string; currentRoles: string; availableRoles: string; noRoles: string; saveButton: string; };
    resetPassword: { title: string; subtitle: string; newPassword: string; confirmPassword: string; generateRandom: string; saveButton: string; };
    delete: { title: string; message: string; confirm: string; cancel: string; };
    status: {
        activate: { title: string; message: string; };
        deactivate: { title: string; message: string; };
        confirm: string; cancel: string;
    };
    messages: { createSuccess: string; updateSuccess: string; deleteSuccess: string; activateSuccess: string; deactivateSuccess: string; rolesUpdated: string; passwordReset: string; error: string; };
    onboarding: {
        title: string; hideGuide: string; showGuide: string;
        steps: {
            create: { title: string; description: string; };
            role: { title: string; description: string; };
            activate: { title: string; description: string; };
        };
    };
}

export const user: UserTranslations = {
    page: {
        title: "Manajemen Pengguna",
        subtitle: "Kelola akun pengguna sistem, role, dan hak akses.",
        createUser: "Tambah Pengguna",
        searchPlaceholder: "Cari nama atau email...",
        noUsersFound: "Tidak ada pengguna ditemukan.",
        loading: "Memuat pengguna...",
    },
    filters: {
        allRoles: "Semua Role", allStatus: "Semua Status", allTypes: "Semua Tipe",
        active: "Aktif", inactive: "Nonaktif", employee: "Karyawan", standalone: "Standalone",
    },
    table: {
        name: "Nama", email: "Email", role: "Role", status: "Status", type: "Tipe",
        lastLogin: "Login Terakhir", createdAt: "Dibuat", actions: "Aksi",
        neverLoggedIn: "Belum pernah login", linkedToEmployee: "Terhubung ke Karyawan", standaloneUser: "User Mandiri",
    },
    actions: {
        view: "Lihat Detail", edit: "Edit Pengguna", assignRoles: "Atur Role",
        resetPassword: "Reset Kata Sandi", activate: "Aktifkan", deactivate: "Nonaktifkan", delete: "Hapus Pengguna",
    },
    form: {
        createTitle: "Tambah Pengguna Baru", editTitle: "Edit Pengguna",
        fullNameLabel: "Nama Lengkap", fullNamePlaceholder: "Nama lengkap pengguna",
        emailLabel: "Email", emailPlaceholder: "email@perusahaan.com",
        passwordLabel: "Kata Sandi Sementara", passwordPlaceholder: "Akan digenerate otomatis jika kosong",
        typeLabel: "Tipe Pengguna", statusLabel: "Status", employeeLabel: "Tautkan ke Karyawan",
        saveButton: "Simpan", cancelButton: "Batal",
    },
    detail: {
        title: "Detail Pengguna", personalInfo: "Informasi Pribadi", accountInfo: "Informasi Akun",
        roles: "Role", sessions: "Sesi", linkedEmployee: "Karyawan Terhubung", noLinkedEmployee: "Tidak ada karyawan terhubung",
    },
    credentials: {
        title: "Kredensial Pengguna", subtitle: "Simpan informasi ini di tempat yang aman.",
        email: "Email", temporaryPassword: "Kata Sandi Sementara",
        warning: "Kata sandi ini hanya ditampilkan sekali. Pastikan sudah disalin.",
        copyEmail: "Salin Email", copyPassword: "Salin Kata Sandi", copied: "Tersalin!", close: "Tutup",
    },
    roles: {
        title: "Atur Role Pengguna", currentRoles: "Role Saat Ini", availableRoles: "Role Tersedia",
        noRoles: "Belum ada role diberikan.", saveButton: "Simpan Role",
    },
    resetPassword: {
        title: "Reset Kata Sandi", subtitle: "Reset kata sandi untuk pengguna ini.",
        newPassword: "Kata Sandi Baru", confirmPassword: "Konfirmasi Kata Sandi",
        generateRandom: "Generate Otomatis", saveButton: "Reset Kata Sandi",
    },
    delete: { title: "Hapus Pengguna", message: "Apakah Anda yakin ingin menghapus pengguna \"{{name}}\"?", confirm: "Ya, Hapus", cancel: "Batal" },
    status: {
        activate: { title: "Aktifkan Pengguna", message: "Aktifkan akun pengguna \"{{name}}\"?" },
        deactivate: { title: "Nonaktifkan Pengguna", message: "Nonaktifkan akun pengguna \"{{name}}\"? Pengguna tidak akan bisa login." },
        confirm: "Ya, Lanjutkan", cancel: "Batal",
    },
    messages: {
        createSuccess: "Pengguna berhasil dibuat.", updateSuccess: "Pengguna berhasil diperbarui.",
        deleteSuccess: "Pengguna berhasil dihapus.", activateSuccess: "Pengguna berhasil diaktifkan.",
        deactivateSuccess: "Pengguna berhasil dinonaktifkan.", rolesUpdated: "Role pengguna berhasil diperbarui.",
        passwordReset: "Kata sandi berhasil direset.", error: "Terjadi kesalahan. Silakan coba lagi.",
    },
    onboarding: {
        title: "Panduan Onboarding Pengguna", hideGuide: "Sembunyikan panduan", showGuide: "Tampilkan panduan",
        steps: {
            create: { title: "1. Buat Akun Pengguna", description: "Tambahkan pengguna baru dengan email dan tipe akun yang sesuai." },
            role: { title: "2. Atur Role", description: "Tetapkan role yang sesuai untuk mengatur izin akses pengguna." },
            activate: { title: "3. Aktifkan Akun", description: "Pastikan akun aktif agar pengguna bisa login ke sistem." },
        },
    },
};
