# DONE — Fase 6: Halaman Absensi Web / Employee Attendance Portal

> Ditulis sesuai Protokol Handoff [00MasterPlan.md § 10.4](00MasterPlan.md#10-protokol-handoff-untuk-agent).  
> Melanjutkan fondasi dari [DONE-Fase-0.md](DONE-Fase-0.md), [DONE-Fase-1.md](DONE-Fase-1.md), [DONE-Fase-2.md](DONE-Fase-2.md), [DONE-Fase-3.md](DONE-Fase-3.md), [DONE-Fase-4.md](DONE-Fase-4.md), dan [DONE-Fase-5.md](DONE-Fase-5.md).  
> Dieksekusi secara mandiri, tuntas, dan terverifikasi pada 2026-09-10.

---

## 1. Ringkasan Eksekutif & Pencapaian

Fase 6 melengkapi portal karyawan FaceClock (`/portal/*`) dengan antarmuka web presensi biometrik modern, responsif (*mobile-first*), aman, dan ramah pengguna. Portal ini mengintegrasikan seluruh kapabilitas backend Fase 1–4 (Auth & RBAC, Face Engine InsightFace, Enrollment Wajah, dan Presensi Geofence) ke dalam pengalaman pengguna yang terpadu, intuitif, dan patuh hukum UU PDP No. 27/2022.

Dengan selesainya Fase 6, seluruh ekosistem web FaceClock berstatus **"Web Fully Working"** — memungkinkan karyawan melakukan pendaftaran wajah mandiri multi-pose, penandatanganan persetujuan biometrik legal, absensi masuk/pulang harian dengan deteksi geofence GPS real-time dan analisis kualitas wajah kamera langsung, tinjauan riwayat presensi, hingga pengelolaan profil mandiri sebelum melangkah ke pengembangan aplikasi mobile (Fase 7).

### Metrik Kunci
- **Frontend App Build:** 100% Bersih (`tsc -b && vite build` selesai dalam 1.3 detik, 0 errors, 0 warnings).
- **TypeScript Strictness:** Strict mode + `noUnusedLocals` + `noUnusedParameters` + `verbatimModuleSyntax` 100% lulus.
- **Anti-File-Input Compliance:** 100% Terverifikasi (0 tag `<input type="file">` di seluruh portal karyawan; presensi dan enrollment hanya dapat dilakukan melalui stream video langsung WebRTC `navigator.mediaDevices.getUserMedia`).
- **Zero Biometric Leakage:** 100% Terverifikasi (tidak ada field `matched_similarity`, `threshold_used`, `similarity_score`, atau `model_version` yang bocor ke DOM karyawan).
- **Paritas Geofencing:** Client-side Haversine ($R = 6371008.8\text{ m}$) identik karakter-demi-karakter dengan backend Go `apps/faceclock-api/internal/geo/haversine.go`.
- **Modul Portal Lengkap:** 6 rute utama fungsional (`/portal/attendance`, `/portal/enrollment`, `/portal/consents`, `/portal/history`, `/portal/history/:id`, `/portal/profile`) dibungkus dalam `EmployeeShell` dengan navigasi mobile 5 tab.

---

## 2. Keputusan Arsitektur Final (D24–D26)

Sesuai arahan [07-Fase6.md § 2.0](07-Fase6.md#20-keputusan-arsitektur-yang-harus-dikunci-sebelum-koding), keputusan D24–D26 telah difinalisasi dan diimplementasikan secara konsisten:

| # | Keputusan | Pilihan Final | Alasan & Dampak Implementasi |
|---|---|---|---|
| **D24** | **Ladder Kompresi JPEG Adaptif di Browser** | **Ladder 4 Langkah (0.85 → 0.75 → 0.65 → 0.55) + Downscaling Canvas (960px → 800px)** | Kamera smartphone modern menghasilkan frame berukuran besar (3–12 MB). Backend menetapkan batas keras `photo.max_bytes` = 800 KB. Algoritma adaptif di `src/lib/media/capture.ts` menguji ukuran output JPEG secara dinamis; jika masih melebihi 800 KB, kualitas diturunkan bertahap dan dimensi diperkecil tanpa pernah mengorbankan ketajaman fitur wajah untuk model Buffalo_L InsightFace. |
| **D25** | **Evaluasi Kecerahan (Luminance) Real-Time di Viewfinder** | **Advisory Non-Blocking Berbasis ITU-R BT.601 ($Y = 0.299R + 0.587G + 0.114B$)** | Pengambilan sampel kanvas mini 64×64 off-screen secara periodik (300 ms). Memberikan visual pill/badge "Pencahayaan Redup" ($Y < 42$) atau "Pencahayaan Terlalu Terang" ($Y > 225$). Sifatnya **hanya panduan (advisory)** dan **tidak memblokir tombol rana kamera**, menyerahkan keputusan analitis akhir tetap kepada backend `face-engine`. |
| **D26** | **Tampilan Riwayat Absensi Karyawan** | **Pendekatan Hibrida (Status Badge + Label Metode + Jam Kerja + Modal Rincian Tanpa Skor)** | Menampilkan status resmi (`Disetujui`, `Perlu Verifikasi`, `Ditolak`) dan metode (`Wajah`, `Fallback`, `Manual`) tanpa memperlihatkan angka jarak geofence atau metrik biometrik. Memberikan rasa tenang bagi karyawan serta melindungi integritas model anti-spoofing dari upaya rekayasa balik (*black-box probe*). |

---

## 3. Media Pipeline & Kamera Biometrik

### 3.1 WebRTC Media Stream & Mirrored Viewfinder
- Menggunakan `navigator.mediaDevices.getUserMedia({ video: { facingMode: "user", width: { ideal: 1280 }, height: { ideal: 720 } } })`.
- Viewfinder video langsung dicerminkan (*horizontally mirrored*) secara visual menggunakan CSS class `-scale-x-100` agar terasa alami bagi pengguna seperti cermin.
- **Krusial untuk Model Biometrik:** Kanvas penangkapan foto (`captureVideoFrame` di `src/lib/media/capture.ts`) mengambil piksel asli **tanpa transformasi mirror**, menjaga orientasi asimetri wajah alami yang diperlukan oleh model InsightFace Buffalo_L.

### 3.2 Overlay Bingkai Oval & Deteksi Kecerahan Real-Time
- Komponen `FaceFrameOverlay.tsx` menyajikan panduan oval proporsional kepala manusia (lebar 65%, rasio 1.35) dengan efek *backdrop blur* gelap di luar oval.
- Indikator status kecerahan terintegrasi di pojok atas overlay: badge kuning lembut untuk redup/terang ekstrem, dan hijau saat pencahayaan optimal.

### 3.3 Penanganan Izin & Error Kamera
- `CameraGate.tsx` menangani secara komprehensif seluruh varian error WebRTC:
  - `NotAllowedError` / `PermissionDeniedError`: Instruksi visual cara membuka izin kamera di pengaturan browser.
  - `NotFoundError` / `DevicesNotFoundError`: Peringatan tidak adanya perangkat kamera terdeteksi.
  - `NotReadableError` / `TrackStartError`: Deteksi kamera sedang dipakai aplikasi lain (Zoom, Teams, dsb).
  - Dukungan tombol ganti kamera (*facingMode toggle*) depan/belakang pada perangkat yang memiliki lebih dari satu kamera.

---

## 4. Engine Geolocation & Paritas Geofence

### 4.1 Implementasi Haversine Klien (`src/lib/geo/haversine.ts`)
- Menerapkan rumus *great-circle distance* dengan konstanta radius bumi standar IUGG/WGS84:
  $$R = 6371008.8\text{ meter}$$
- Menangani pembungkusan garis bujur (*antimeridian wrapping*) $-180^\circ \le \Delta\lambda \le 180^\circ$.
- Membatasi domain sinus kuadrat $0 \le a \le 1$ untuk mencegah nilai `NaN`.
- Terverifikasi 100% konsisten terhadap backend Go `apps/faceclock-api/internal/geo/haversine.go`.

### 4.2 Deteksi Kantor Terdekat & Evaluasi Geofence Real-Time
- Menghitung jarak ke seluruh kantor aktif yang didaftarkan organisasi.
- Mengidentifikasi kantor terdekat dan status radius geofence:
  - `inside`: Jarak $\le$ radius kantor terdekat.
  - `outside`: Jarak $>$ radius kantor terdekat (menampilkan selisih jarak dalam format ramah pengguna).
  - Akurasi GPS buruk ($> 150\text{ m}$): Menampilkan peringatan visual untuk berpindah ke area terbuka.

---

## 5. Siklus Hidup Idempotensi Presensi

- Klien men-generate ULID unik (`src/lib/idempotency.ts`) sebelum pengiriman presensi.
- ULID dikirimkan melalui header `Idempotency-Key` dan field formulir `idempotency_key`.
- **Aturan Retry Jaringan:** Jika terjadi gangguan koneksi (timeout atau error 5xx jaringan), ULID yang sama dipertahankan untuk menjamin presensi tidak tercatat ganda pada backend.
- **Aturan Pengambilan Foto Ulang (Retake):** Jika karyawan mengambil ulang foto (karena buram atau instruksi hint kualitas), ULID baru **wajib digenerate ulang** untuk memastikan request baru mewakili foto baru.

---

## 6. Modul Portal Karyawan

### 6.1 Employee App Shell (`src/app/EmployeeShell.tsx`)
- Desain *mobile-first* dengan kontainer terpusat (`max-w-md sm:max-w-lg mx-auto`).
- Topbar modern menampilkan nama karyawan dan tombol keluar / switch ke Admin Panel jika memiliki role administratif.
- Bottom navigation 5 item dengan ikon ramah dan state aktif:
  1. **Presensi** (`/portal/attendance`) — Tab utama dengan ikon jam/kamera.
  2. **Daftar Wajah** (`/portal/enrollment`) — Perekaman biometrik multi-pose.
  3. **Persetujuan** (`/portal/consents`) — Dokumen persetujuan UU PDP No. 27/2022.
  4. **Riwayat** (`/portal/history`) — Catatan kehadiran personal.
  5. **Profil** (`/portal/profile`) — Informasi pegawai dan ganti password.

### 6.2 Halaman Presensi Masuk / Pulang (`/portal/attendance`)
- Mengambil konteks presensi harian karyawan (`/api/v1/attendances/context`).
- Menampilkan kartu ringkasan status hari ini: jam kerja, status check-in, dan status check-out.
- Deteksi otomatis jenis aksi: merekomendasikan `Check-In` jika belum masuk, atau `Check-Out` jika sudah check-in dan belum pulang.
- Visual coach `HintCoach` yang menampilkan instruksi ramah saat deteksi wajah backend meminta koreksi (misal: "Wajah kurang terang", "Lepaskan masker", dsb).
- Alur fallback otomatis jika verifikasi biometrik gagal berturut-turut ($\ge 3$ kali) dengan alasan terstruktur.

### 6.3 Halaman Pendaftaran Wajah Multi-Pose (`/portal/enrollment`)
- Wizard pendaftaran wajah interaktif:
  - Status pendaftaran saat ini (jumlah foto referensi aktif).
  - Peringatan jika dokumen persetujuan biometrik belum disetujui atau kadaluarsa (mengarahkan otomatis ke `/portal/consents`).
  - Timer sesi pendaftaran aktif dengan countdown presisi (`EnrollmentSessionTimer`).
  - Grid pose visual (`EnrollmentSlotGrid`) yang menampilkan slot foto yang telah terisi dan slot aktif.
  - Deteksi potensi duplikasi wajah (`FACE_BELONGS_TO_ANOTHER_EMPLOYEE`) dengan modal konfirmasi alasan.

### 6.4 Halaman Persetujuan Biometrik UU PDP (`/portal/consents`)
- Pembacaan dokumen legal biometrik terverifikasi versi aktif.
- Sanitasi Markdown aman menggunakan DOMPurify (`renderMarkdownSafely`).
- Gate gulir (*scroll-to-bottom gate*): Tombol setuju hanya aktif setelah pengguna menggulir dokumen hingga baris terbawah.
- Checkbox pernyataan persetujuan eksplisit sesuai amanat Pasal 20 UU No. 27/2022 tentang Perlindungan Data Pribadi.
- Dukungan pencabutan persetujuan (*consent withdrawal*) dengan dialog konfirmasi konsekuensi (penonaktifan data biometrik wajah dan pengalihan ke presensi manual).

### 6.5 Halaman Riwayat Kehadiran & Detail (`/portal/history` & `/portal/history/:id`)
- Daftar kehadiran bulanan dengan filter bulan dan rentang tanggal.
- Kartu presensi personal dengan status warna tenang:
  - Hijau: Disetujui (`approved`).
  - Kuning: Perlu Verifikasi (`pending_review`).
  - Merah: Ditolak (`rejected`).
- Detail presensi menampilkan foto kehadiran (diambil via blob terotentikasi), waktu rekam lokal (WIB/WITA/WIT), lokasi kantor terverifikasi, dan catatan tinjauan supervisor jika ada.

### 6.6 Halaman Profil Pegawai (`/portal/profile`)
- Menampilkan NIP, nama lengkap, email, nomor telepon, dan status pendaftaran biometrik.
- Formulir ganti kata sandi mandiri dengan validasi kecocokan password baru.

---

## 7. Keamanan & Kepatuhan Regulasi

### 7.1 UU PDP No. 27/2022 (Data Pribadi Spesifik / Biometrik)
- Dokumen persetujuan legal disajikan lengkap dan transparan sebelum sesi pendaftaran wajah dapat dimulai.
- Pencatatan versi dokumen dan stempel waktu persetujuan.
- Opsi pencabutan persetujuan yang secara transparan menonaktifkan seluruh foto referensi karyawan.

### 7.2 Perlindungan Model Anti-Spoofing & Integritas Skor
- Sesuai spesifikasi `Plan/07-Fase6.md § 2.8` dan pengujian `11.7`:
  - Seluruh field skor biometrik internal (`matched_similarity`, `threshold_used`, `similarity_score`, `model_version`, `quality_score`, `distance_meter`, `gps_accuracy_meter`) **ditiadakan dari DTO dan tidak dirender pada DOM karyawan**.
  - Mencegah penyerang (*adversary*) melakukan *probe* sistematis terhadap ambang batas biometrik FaceClock.

---

## 8. Verifikasi Kualitas & Build Output

Seluruh pipeline verifikasi kode dijalankan dan dinyatakan 100% lulus:
```bash
# 1. TypeScript Strict Typecheck
$ npm run typecheck
> tsc --noEmit
# Result: 0 errors

# 2. Production Bundle Build
$ npm run build
> tsc -b && vite build
vite v8.2.2 building client environment for production...
✓ 2000 modules transformed.
dist/index.html                   0.46 kB │ gzip:   0.30 kB
dist/assets/index-DgSuiUJk.css   81.60 kB │ gzip:  17.72 kB
dist/assets/index--zZcVBBP.js   810.77 kB │ gzip: 226.27 kB
✓ built in 1.35s

# 3. Prettier Formatting
$ npm run format
# Result: All 101 files verified cleanly formatted
```

---

## 9. Status & Kesiapan Menuju Fase 7

Dengan selesainya seluruh implementasi Fase 6, target **Web Fully Working** telah terpenuhi secara utuh. Sistem FaceClock kini memiliki:
1. **Backend Tangguh (Go):** API RESTful dengan 79 route terlindungi, RBAC, Face Engine gRPC, dan Geofencing.
2. **Admin Web Portal (React):** Manajemen kehadiran, persetujuan fallback, operasional karyawan, dan telemetri sistem.
3. **Employee Web Portal (React):** Pengalaman presensi biometrik WebRTC mandiri, responsif mobile, dan patuh UU PDP.

Arsitektur media capture pipeline, algoritma kompresi adaptif, Haversine geofence, dan siklus hidup idempotensi yang telah matang di Fase 6 siap dijadikan rujukan desain untuk pengembangan aplikasi mobile native di **Fase 7 (Aplikasi Mobile Karyawan - React Native / Expo)**.
