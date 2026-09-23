// ============================================================
// BAHASA INDONESIA — location namespace
// ============================================================
export interface LocationTranslations {
    page: { title: string; subtitle: string; createLocation: string; searchPlaceholder: string; noData: string; loading: string; };
    table: { name: string; address: string; radius: string; latitude: string; longitude: string; status: string; actions: string; };
    form: { createTitle: string; editTitle: string; nameLabel: string; namePlaceholder: string; addressLabel: string; addressPlaceholder: string; latitudeLabel: string; longitudeLabel: string; radiusLabel: string; radiusHelper: string; pickOnMap: string; statusLabel: string; saveButton: string; cancelButton: string; };
    map: { title: string; searchPlaceholder: string; confirmButton: string; currentLocation: string; };
    delete: { title: string; message: string; confirm: string; cancel: string; };
    messages: { createSuccess: string; updateSuccess: string; deleteSuccess: string; error: string; };
}

export const location: LocationTranslations = {
    page: {
        title: "Lokasi Kantor",
        subtitle: "Kelola lokasi yang diizinkan untuk absensi.",
        createLocation: "Tambah Lokasi",
        searchPlaceholder: "Cari nama lokasi...",
        noData: "Tidak ada lokasi ditemukan.",
        loading: "Memuat lokasi...",
    },
    table: {
        name: "Nama Lokasi",
        address: "Alamat",
        radius: "Radius (m)",
        latitude: "Latitude",
        longitude: "Longitude",
        status: "Status",
        actions: "Aksi",
    },
    form: {
        createTitle: "Tambah Lokasi Baru",
        editTitle: "Edit Lokasi",
        nameLabel: "Nama Lokasi",
        namePlaceholder: "Contoh: Kantor Pusat Jakarta",
        addressLabel: "Alamat",
        addressPlaceholder: "Alamat lengkap lokasi",
        latitudeLabel: "Latitude",
        longitudeLabel: "Longitude",
        radiusLabel: "Radius (meter)",
        radiusHelper: "Jarak maksimum dari titik pusat yang diizinkan untuk absensi.",
        pickOnMap: "Pilih di Peta",
        statusLabel: "Status",
        saveButton: "Simpan Lokasi",
        cancelButton: "Batal",
    },
    map: {
        title: "Pilih Lokasi di Peta",
        searchPlaceholder: "Cari alamat...",
        confirmButton: "Konfirmasi Lokasi",
        currentLocation: "Lokasi Saya",
    },
    delete: {
        title: "Hapus Lokasi",
        message: "Hapus lokasi \"{{name}}\"? Tindakan ini tidak dapat dibatalkan.",
        confirm: "Ya, Hapus",
        cancel: "Batal",
    },
    messages: {
        createSuccess: "Lokasi berhasil ditambahkan.",
        updateSuccess: "Lokasi berhasil diperbarui.",
        deleteSuccess: "Lokasi berhasil dihapus.",
        error: "Terjadi kesalahan. Silakan coba lagi.",
    },
};
