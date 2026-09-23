// ============================================================
// BAHASA INDONESIA — attendance namespace
// ============================================================
export interface AttendanceTranslations {
    page: { title: string; subtitle: string; loading: string; noData: string; searchPlaceholder: string; };
    pending: { title: string; subtitle: string; noData: string; loading: string; approve: string; reject: string; approveAll: string; viewDetail: string; };
    filters: { allStatus: string; dateFrom: string; dateTo: string; employee: string; location: string; };
    table: { employee: string; date: string; checkIn: string; checkOut: string; location: string; status: string; method: string; duration: string; actions: string; notes: string; };
    status: { present: string; late: string; absent: string; leave: string; approved: string; pending: string; rejected: string; };
    method: { face: string; manual: string; qr: string; };
    detail: { title: string; checkInPhoto: string; checkOutPhoto: string; checkInLocation: string; checkOutLocation: string; reviewNotes: string; reviewedBy: string; reviewedAt: string; approveButton: string; rejectButton: string; editButton: string; rejectReason: string; };
    messages: { approveSuccess: string; rejectSuccess: string; updateSuccess: string; error: string; };
}

export const attendance: AttendanceTranslations = {
    page: {
        title: "Data Absensi",
        subtitle: "Rekap kehadiran karyawan.",
        loading: "Memuat data absensi...",
        noData: "Tidak ada data absensi ditemukan.",
        searchPlaceholder: "Cari nama karyawan...",
    },
    pending: {
        title: "Antrian Review Absensi",
        subtitle: "Absensi yang menunggu persetujuan.",
        noData: "Tidak ada absensi yang menunggu review.",
        loading: "Memuat antrian...",
        approve: "Setujui",
        reject: "Tolak",
        approveAll: "Setujui Semua",
        viewDetail: "Lihat Detail",
    },
    filters: {
        allStatus: "Semua Status",
        dateFrom: "Dari Tanggal",
        dateTo: "Sampai Tanggal",
        employee: "Karyawan",
        location: "Lokasi",
    },
    table: {
        employee: "Karyawan",
        date: "Tanggal",
        checkIn: "Masuk",
        checkOut: "Keluar",
        location: "Lokasi",
        status: "Status",
        method: "Metode",
        duration: "Durasi",
        actions: "Aksi",
        notes: "Catatan",
    },
    status: {
        present: "Hadir",
        late: "Terlambat",
        absent: "Absen",
        leave: "Cuti",
        approved: "Disetujui",
        pending: "Menunggu",
        rejected: "Ditolak",
    },
    method: {
        face: "Wajah",
        manual: "Manual",
        qr: "QR Code",
    },
    detail: {
        title: "Detail Absensi",
        checkInPhoto: "Foto Masuk",
        checkOutPhoto: "Foto Keluar",
        checkInLocation: "Lokasi Masuk",
        checkOutLocation: "Lokasi Keluar",
        reviewNotes: "Catatan Review",
        reviewedBy: "Direview oleh",
        reviewedAt: "Direview pada",
        approveButton: "Setujui",
        rejectButton: "Tolak",
        editButton: "Edit",
        rejectReason: "Alasan Penolakan",
    },
    messages: {
        approveSuccess: "Absensi berhasil disetujui.",
        rejectSuccess: "Absensi berhasil ditolak.",
        updateSuccess: "Absensi berhasil diperbarui.",
        error: "Terjadi kesalahan. Silakan coba lagi.",
    },
};
