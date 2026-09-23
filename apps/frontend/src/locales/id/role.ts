// ============================================================
// BAHASA INDONESIA — role namespace
// ============================================================
export interface RoleTranslations {
    page: { title: string; subtitle: string; createRole: string; searchPlaceholder: string; noRolesFound: string; loading: string; };
    card: { permissions: string; modules: string; viewMatrix: string; editRole: string; deleteRole: string; systemRole: string; createdAt: string; };
    matrix: {
        title: string; subtitle: string; roleLabel: string; searchPlaceholder: string;
        legend: { title: string; granted: string; notGranted: string; };
        columns: { feature: string; create: string; read: string; update: string; delete: string; export: string; };
        modules: { attendance: string; employee: string; face: string; user: string; role: string; location: string; settings: string; audit: string; };
        noResults: string; totalPermissions: string; granted: string;
    };
    form: { createTitle: string; editTitle: string; nameLabel: string; namePlaceholder: string; descriptionLabel: string; descriptionPlaceholder: string; permissionsLabel: string; saveButton: string; cancelButton: string; };
    delete: { title: string; message: string; confirm: string; cancel: string; systemRoleWarning: string; };
    messages: { createSuccess: string; updateSuccess: string; deleteSuccess: string; createError: string; updateError: string; deleteError: string; };
}

export const role: RoleTranslations = {
    page: {
        title: "Role & Hak Akses",
        subtitle: "Kelola peran dan izin akses pengguna sistem.",
        createRole: "Buat Role Baru",
        searchPlaceholder: "Cari nama role...",
        noRolesFound: "Tidak ada role yang ditemukan.",
        loading: "Memuat role...",
    },
    card: {
        permissions: "izin",
        modules: "modul",
        viewMatrix: "Buka Matriks Hak Akses",
        editRole: "Edit Role",
        deleteRole: "Hapus Role",
        systemRole: "Role Sistem",
        createdAt: "Dibuat",
    },
    matrix: {
        title: "Matriks Hak Akses",
        subtitle: "Tampilan visual izin per modul untuk role ini.",
        roleLabel: "Role",
        searchPlaceholder: "Cari modul atau fitur...",
        legend: { title: "Keterangan", granted: "Diberikan", notGranted: "Tidak Diberikan" },
        columns: {
            feature: "Fitur / Modul",
            create: "Tambah / Create",
            read: "Lihat / Read",
            update: "Ubah / Update",
            delete: "Hapus / Delete",
            export: "Ekspor / Import",
        },
        modules: {
            attendance: "Absensi", employee: "Karyawan", face: "Wajah", user: "Pengguna",
            role: "Role & Izin", location: "Lokasi", settings: "Pengaturan", audit: "Audit",
        },
        noResults: "Tidak ada modul sesuai pencarian.",
        totalPermissions: "Total Izin",
        granted: "Diberikan",
    },
    form: {
        createTitle: "Buat Role Baru",
        editTitle: "Edit Role",
        nameLabel: "Nama Role",
        namePlaceholder: "Contoh: Manajer HR",
        descriptionLabel: "Deskripsi",
        descriptionPlaceholder: "Deskripsi singkat tentang role ini...",
        permissionsLabel: "Izin Akses",
        saveButton: "Simpan Role",
        cancelButton: "Batal",
    },
    delete: {
        title: "Hapus Role",
        message: "Apakah Anda yakin ingin menghapus role \"{{name}}\"? Tindakan ini tidak dapat dibatalkan.",
        confirm: "Ya, Hapus",
        cancel: "Batal",
        systemRoleWarning: "Role sistem tidak dapat dihapus.",
    },
    messages: {
        createSuccess: "Role berhasil dibuat.",
        updateSuccess: "Role berhasil diperbarui.",
        deleteSuccess: "Role berhasil dihapus.",
        createError: "Gagal membuat role.",
        updateError: "Gagal memperbarui role.",
        deleteError: "Gagal menghapus role.",
    },
};
