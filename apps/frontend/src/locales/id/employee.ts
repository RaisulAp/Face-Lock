// ============================================================
// BAHASA INDONESIA — employee namespace
// ============================================================
export interface EmployeeTranslations {
    page: { title: string; subtitle: string; createEmployee: string; searchPlaceholder: string; noData: string; loading: string; };
    filters: { allDepartments: string; allStatus: string; active: string; inactive: string; };
    table: { nik: string; name: string; department: string; position: string; email: string; phone: string; joinDate: string; status: string; actions: string; faceEnrolled: string; hasUser: string; };
    form: { createTitle: string; editTitle: string; personalInfo: string; fullNameLabel: string; fullNamePlaceholder: string; nikLabel: string; nikPlaceholder: string; emailLabel: string; emailPlaceholder: string; phoneLabel: string; phonePlaceholder: string; departmentLabel: string; departmentPlaceholder: string; positionLabel: string; positionPlaceholder: string; joinDateLabel: string; statusLabel: string; saveButton: string; cancelButton: string; };
    face: { title: string; subtitle: string; enrolled: string; notEnrolled: string; enrollButton: string; reEnrollButton: string; deleteEnrollment: string; capturePhoto: string; uploadPhoto: string; previewPhoto: string; qualityScore: string; qualityGood: string; qualityPoor: string; enrollSuccess: string; enrollError: string; deleteSuccess: string; deleteConfirm: string; };
    delete: { title: string; message: string; confirm: string; cancel: string; };
    messages: { createSuccess: string; updateSuccess: string; deleteSuccess: string; error: string; };
}

export const employee: EmployeeTranslations = {
    page: {
        title: "Daftar Karyawan",
        subtitle: "Kelola data karyawan perusahaan.",
        createEmployee: "Tambah Karyawan",
        searchPlaceholder: "Cari nama atau NIK...",
        noData: "Tidak ada karyawan ditemukan.",
        loading: "Memuat data karyawan...",
    },
    filters: {
        allDepartments: "Semua Departemen",
        allStatus: "Semua Status",
        active: "Aktif",
        inactive: "Nonaktif",
    },
    table: {
        nik: "NIK",
        name: "Nama",
        department: "Departemen",
        position: "Jabatan",
        email: "Email",
        phone: "Telepon",
        joinDate: "Tanggal Bergabung",
        status: "Status",
        actions: "Aksi",
        faceEnrolled: "Wajah Terdaftar",
        hasUser: "Punya Akun",
    },
    form: {
        createTitle: "Tambah Karyawan Baru",
        editTitle: "Edit Karyawan",
        personalInfo: "Informasi Pribadi",
        fullNameLabel: "Nama Lengkap",
        fullNamePlaceholder: "Nama lengkap karyawan",
        nikLabel: "NIK (Nomor Induk Karyawan)",
        nikPlaceholder: "Contoh: EMP001",
        emailLabel: "Email",
        emailPlaceholder: "email@perusahaan.com",
        phoneLabel: "No. Telepon",
        phonePlaceholder: "Contoh: 08123456789",
        departmentLabel: "Departemen",
        departmentPlaceholder: "Pilih atau ketik departemen",
        positionLabel: "Jabatan",
        positionPlaceholder: "Contoh: Software Engineer",
        joinDateLabel: "Tanggal Bergabung",
        statusLabel: "Status",
        saveButton: "Simpan Karyawan",
        cancelButton: "Batal",
    },
    face: {
        title: "Pendaftaran Wajah",
        subtitle: "Daftarkan data biometrik wajah karyawan untuk absensi.",
        enrolled: "Wajah Terdaftar",
        notEnrolled: "Belum Terdaftar",
        enrollButton: "Daftarkan Wajah",
        reEnrollButton: "Daftarkan Ulang",
        deleteEnrollment: "Hapus Data Wajah",
        capturePhoto: "Ambil Foto",
        uploadPhoto: "Unggah Foto",
        previewPhoto: "Pratinjau Foto",
        qualityScore: "Skor Kualitas",
        qualityGood: "Kualitas Baik",
        qualityPoor: "Kualitas Kurang",
        enrollSuccess: "Wajah berhasil didaftarkan.",
        enrollError: "Gagal mendaftarkan wajah.",
        deleteSuccess: "Data wajah berhasil dihapus.",
        deleteConfirm: "Hapus data biometrik wajah karyawan ini?",
    },
    delete: {
        title: "Hapus Karyawan",
        message: "Apakah Anda yakin ingin menghapus karyawan \"{{name}}\"?",
        confirm: "Ya, Hapus",
        cancel: "Batal",
    },
    messages: {
        createSuccess: "Karyawan berhasil ditambahkan.",
        updateSuccess: "Karyawan berhasil diperbarui.",
        deleteSuccess: "Karyawan berhasil dihapus.",
        error: "Terjadi kesalahan. Silakan coba lagi.",
    },
};
