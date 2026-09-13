import type { ApiErrorCode } from "./codes";

export interface ErrorPresentation {
  title: string;
  body: string;
  severity: "info" | "warning" | "error";
  action?: "relogin" | "refetch";
  showRequestId?: boolean;
}

export function getErrorMessage(code?: string, fallback?: string): string {
  if (!code) return fallback || "Terjadi kesalahan pada sistem.";
  const msg = ERROR_MESSAGES[code as ApiErrorCode];
  return msg ? `${msg.title}: ${msg.body}` : fallback || code;
}

export const ERROR_MESSAGES: Record<ApiErrorCode, ErrorPresentation> = {
  // Fase 0 Base Catalog
  BAD_REQUEST: {
    title: "Permintaan tidak valid",
    body: "Format data atau parameter yang dikirim tidak sesuai.",
    severity: "warning",
  },
  UNAUTHENTICATED: {
    title: "Sesi berakhir",
    body: "Sesi Anda telah kedaluwarsa. Silakan login kembali.",
    severity: "warning",
    action: "relogin",
  },
  FORBIDDEN: {
    title: "Tidak diizinkan",
    body: "Anda tidak memiliki hak akses untuk tindakan ini.",
    severity: "warning",
  },
  NOT_FOUND: {
    title: "Tidak ditemukan",
    body: "Data yang Anda cari tidak ada atau bukan milik Anda.",
    severity: "info",
  },
  CONFLICT: {
    title: "Data berubah",
    body: "Data sudah berubah atau telah diperbarui pengguna lain. Muat ulang untuk melihat data terbaru.",
    severity: "warning",
    action: "refetch",
  },
  PAYLOAD_TOO_LARGE: {
    title: "Ukuran berkas terlalu besar",
    body: "Ukuran data atau berkas yang diunggah melebihi batas yang diizinkan.",
    severity: "warning",
  },
  UNSUPPORTED_MEDIA_TYPE: {
    title: "Format berkas tidak didukung",
    body: "Gunakan format berkas gambar yang didukung (JPEG atau PNG).",
    severity: "warning",
  },
  VALIDATION_ERROR: {
    title: "Validasi gagal",
    body: "Periksa kembali isian formulir sesuai instruksi pada tiap kolom.",
    severity: "warning",
  },
  RATE_LIMITED: {
    title: "Terlalu banyak permintaan",
    body: "Batas permintaan tercapai. Silakan tunggu beberapa saat sebelum mencoba kembali.",
    severity: "warning",
  },
  INTERNAL_ERROR: {
    title: "Terjadi kesalahan sistem",
    body: "Terjadi gangguan internal pada server. Laporkan ID permintaan ke admin jika berlanjut.",
    severity: "error",
    showRequestId: true,
  },
  UPSTREAM_ERROR: {
    title: "Layanan perantara gagal",
    body: "Gagal berkomunikasi dengan layanan pemrosesan internal.",
    severity: "error",
    showRequestId: true,
  },
  SERVICE_UNAVAILABLE: {
    title: "Layanan tidak tersedia",
    body: "Sistem sedang dalam pemeliharaan atau kapasitas penuh. Coba lagi sebentar lagi.",
    severity: "error",
    showRequestId: true,
  },
  UPSTREAM_TIMEOUT: {
    title: "Layanan wajah tidak merespons",
    body: "Layanan pemrosesan biometrik melebihi batas waktu tunggu. Coba beberapa saat lagi.",
    severity: "error",
    showRequestId: true,
  },

  // Fase 1 Auth & RBAC
  INVALID_CREDENTIALS: {
    title: "Email atau password salah",
    body: "Periksa kembali email dan password yang Anda masukkan.",
    severity: "warning",
  },
  INVALID_REFRESH_TOKEN: {
    title: "Sesi tidak valid",
    body: "Token sesi tidak dikenali atau telah kedaluwarsa. Silakan login kembali.",
    severity: "warning",
    action: "relogin",
  },
  REFRESH_TOKEN_REUSED: {
    title: "Sesi terdeteksi di perangkat lain",
    body: "Terdeteksi penggunaan sesi ganda. Demi keamanan, silakan login ulang.",
    severity: "warning",
    action: "relogin",
  },
  EMPLOYEE_NUMBER_TAKEN: {
    title: "Nomor karyawan sudah digunakan",
    body: "Nomor karyawan ini telah terdaftar untuk karyawan lain.",
    severity: "warning",
  },
  EMPLOYEE_HAS_ACTIVE_USER: {
    title: "Karyawan masih memiliki akun aktif",
    body: "Nonaktifkan atau hapus akun pengguna terkait sebelum menghapus karyawan.",
    severity: "warning",
  },
  EMAIL_TAKEN: {
    title: "Email sudah digunakan",
    body: "Alamat email ini telah terdaftar untuk pengguna lain.",
    severity: "warning",
  },
  EMPLOYEE_ALREADY_HAS_USER: {
    title: "Karyawan sudah memiliki akun",
    body: "Satu karyawan hanya dapat ditautkan ke satu akun pengguna.",
    severity: "warning",
  },
  ROLE_NAME_TAKEN: {
    title: "Nama role sudah ada",
    body: "Gunakan nama peran yang berbeda.",
    severity: "warning",
  },
  ROLE_IN_USE: {
    title: "Role sedang digunakan",
    body: "Role ini masih dimiliki oleh beberapa pengguna aktif dan tidak dapat dihapus.",
    severity: "warning",
  },
  SYSTEM_ROLE_IMMUTABLE: {
    title: "Peran bawaan sistem",
    body: "Peran sistem tidak dapat dihapus atau diganti namanya.",
    severity: "warning",
  },
  ROLE_ESCALATION_DENIED: {
    title: "Eskalasi hak akses ditolak",
    body: "Anda tidak dapat memberikan peran atau izin yang tidak Anda miliki.",
    severity: "warning",
  },
  LAST_SUPER_ADMIN: {
    title: "Super Admin terakhir",
    body: "Sistem harus mempertahankan minimal satu akun Super Admin aktif.",
    severity: "warning",
  },
  CSRF_HEADER_MISSING: {
    title: "Permintaan tidak aman",
    body: "Header keamanan peramban tidak ditemukan.",
    severity: "error",
  },
  CSRF_UNTRUSTED_ORIGIN: {
    title: "Asal permintaan tidak dipercaya",
    body: "Permintaan ditolak karena berasal dari domain yang tidak terdaftar.",
    severity: "error",
  },

  // Fase 3 Biometrik & Consent
  CONSENT_REQUIRED: {
    title: "Persetujuan biometrik diperlukan",
    body: "Karyawan harus menyetujui pemrosesan data biometrik sebelum melanjutkan.",
    severity: "warning",
  },
  CONSENT_ALREADY_GRANTED: {
    title: "Persetujuan aktif sudah ada",
    body: "Persetujuan biometrik versi saat ini sudah diberikan sebelumnya.",
    severity: "info",
  },
  CONSENT_VERSION_OUTDATED: {
    title: "Versi dokumen persetujuan lama",
    body: "Persetujuan harus menggunakan versi dokumen kebijakan biometrik terbaru.",
    severity: "warning",
  },
  FACE_NOT_USABLE: {
    title: "Kualitas foto wajah belum memenuhi syarat",
    body: "Foto wajah tidak lolos standar kualitas. Perhatikan panduan pencahayaan dan posisi.",
    severity: "warning",
  },
  DUPLICATE_PHOTO: {
    title: "Foto identik terdeteksi",
    body: "Foto yang sama persis sudah digunakan sebelumnya.",
    severity: "warning",
  },
  ENROLLMENT_INCOMPLETE: {
    title: "Pendaftaran wajah belum selesai",
    body: "Jumlah foto referensi masih di bawah batas minimal yang ditetapkan.",
    severity: "warning",
  },
  ENROLLMENT_LIMIT_REACHED: {
    title: "Batas foto tercapai",
    body: "Jumlah foto referensi sudah mencapai batas maksimum yang diizinkan.",
    severity: "warning",
  },
  ENROLLMENT_SESSION_EXPIRED: {
    title: "Sesi pendaftaran kedaluwarsa",
    body: "Waktu sesi pendaftaran wajah telah habis. Silakan mulai sesi baru.",
    severity: "warning",
    action: "refetch",
  },
  ENROLLMENT_MODEL_CHANGED: {
    title: "Versi model AI berubah",
    body: "Model verifikasi wajah telah diperbarui. Silakan ulangi sesi pendaftaran.",
    severity: "warning",
    action: "refetch",
  },
  FACE_BELONGS_TO_ANOTHER_EMPLOYEE: {
    title: "Wajah mirip dengan karyawan lain",
    body: "Pemeriksaan kemiripan mendeteksi kecocokan dengan data karyawan lain yang sudah terdaftar.",
    severity: "warning",
  },
  REINDEX_IN_PROGRESS: {
    title: "Sedang dilakukan penataan ulang model",
    body: "Proses reindex wajah sedang berjalan. Pendaftaran wajah baru ditangguhkan sementara.",
    severity: "warning",
  },
  ATTENDANCE_MODE_MANUAL: {
    title: "Jalur absensi manual",
    body: "Karyawan ini berada pada moda absensi manual tanpa pemrosesan biometrik.",
    severity: "info",
  },

  // Fase 4 & 7 Absensi & Verifikasi
  FACE_NOT_ENROLLED: {
    title: "Wajah belum terdaftar",
    body: "Karyawan belum memiliki foto referensi aktif untuk model yang berlaku.",
    severity: "warning",
  },
  FACE_NOT_MATCHED: {
    title: "Wajah tidak cocok",
    body: "Tingkat kemiripan wajah di bawah batas ambang dan jalur persetujuan manual tidak diaktifkan.",
    severity: "warning",
  },
  FACE_MISMATCH: {
    title: "Wajah tidak sesuai",
    body: "Kemiripan wajah tidak mencapai ambang batas yang ditentukan.",
    severity: "warning",
  },
  OUTSIDE_GEOFENCE: {
    title: "Di luar area kantor",
    body: "Posisi Anda berada di luar radius kantor yang diizinkan.",
    severity: "warning",
  },
  LOCATION_REQUIRED: {
    title: "Lokasi GPS diperlukan",
    body: "Pengambilan koordinat lokasi wajib diaktifkan saat geofence menyala.",
    severity: "warning",
  },
  LOCATION_INACCURATE: {
    title: "Akurasi GPS kurang baik",
    body: "Sinyal GPS belum cukup akurat. Tunggu akurasi membaik atau berada di ruang terbuka.",
    severity: "warning",
  },
  ALREADY_CHECKED_IN: {
    title: "Sudah melakukan check-in",
    body: "Check-in hari ini sudah tercatat sebelumnya.",
    severity: "info",
    action: "refetch",
  },
  ALREADY_CHECKED_OUT: {
    title: "Sudah melakukan check-out",
    body: "Check-out hari ini sudah tercatat sebelumnya.",
    severity: "info",
    action: "refetch",
  },
  CHECKOUT_WITHOUT_CHECKIN: {
    title: "Belum check-in",
    body: "Tidak dapat melakukan check-out sebelum ada catatan check-in hari ini.",
    severity: "warning",
  },
  CHECKOUT_TOO_SOON: {
    title: "Terlalu cepat untuk check-out",
    body: "Jeda waktu minimal antara jam masuk dan jam pulang belum terpenuhi.",
    severity: "warning",
  },
  ATTENDANCE_ALREADY_REVIEWED: {
    title: "Absensi sudah ditinjau",
    body: "Absensi ini telah disetujui atau ditolak oleh peninjau lain.",
    severity: "info",
    action: "refetch",
  },
  SELF_REVIEW_DENIED: {
    title: "Tidak bisa meninjau absensi sendiri",
    body: "Aturan integritas mengharuskan absensi Anda ditinjau oleh administrator lain.",
    severity: "warning",
  },
  EMPLOYEE_INACTIVE: {
    title: "Status karyawan tidak aktif",
    body: "Karyawan telah dinonaktifkan atau berstatus keluar.",
    severity: "warning",
  },
  TOO_MANY_FAILED_ATTEMPTS: {
    title: "Batas percobaan terlampaui",
    body: "Terlalu banyak percobaan gagal dalam satu jam terakhir. Silakan coba kembali nanti.",
    severity: "warning",
  },
  ATTENDANCE_NOT_CONFIGURED: {
    title: "Konfigurasi kantor belum lengkap",
    body: "Geofence aktif namun belum ada lokasi kantor yang aktif.",
    severity: "error",
  },
  FACE_SERVICE_NOT_CONFIGURED: {
    title: "Model biometrik belum dikonfigurasi",
    body: "Kalibrasi model wajah belum diselesaikan oleh administrator.",
    severity: "error",
  },
  LIVENESS_REQUIRED: {
    title: "Pemeriksaan keaslian diperlukan",
    body: "Deteksi keaslian wajah (liveness) diperlukan untuk menyelesaikan proses.",
    severity: "warning",
  },
};

export function getErrorPresentation(code: string): ErrorPresentation {
  const typedCode = code as ApiErrorCode;
  if (typedCode in ERROR_MESSAGES) {
    return ERROR_MESSAGES[typedCode];
  }
  return {
    title: "Terjadi kesalahan",
    body: "Operasi tidak dapat diselesaikan. Silakan coba lagi.",
    severity: "error",
    showRequestId: true,
  };
}
