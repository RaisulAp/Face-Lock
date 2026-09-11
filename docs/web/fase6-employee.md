# Dokumentasi Arsitektur Frontend — FaceClock Web Employee Portal (Fase 6)

## 1. Ikhtisar & Tujuan

Portal Karyawan FaceClock (`/portal/*`) adalah antarmuka web khusus karyawan yang dirancang *mobile-first* untuk memfasilitasi seluruh kebutuhan presensi harian berbasis biometrik wajah, pendaftaran wajah multi-pose mandiri (*self-service enrollment*), pengelolaan persetujuan data biometrik sesuai UU PDP No. 27/2022, serta riwayat dan profil personal.

### Stack & Fondasi Kunci
- **Framework UI:** React 19 + TypeScript (~6.0)
- **Styling:** Tailwind CSS v4 (*mobile-first responsive layout*)
- **State Management & Server Sync:** TanStack Query v5
- **Routing:** React Router v7
- **Media Capture:** HTML5 WebRTC `navigator.mediaDevices.getUserMedia` + unmirrored `<canvas>`
- **Sanitasi Konten:** DOMPurify 3.4
- **Idempotensi Presensi:** ULID (`ulid`)

---

## 2. Struktur Navigasi & Matriks Rute Portal

Aplikasi karyawan dibungkus dalam `EmployeeShell` dengan container terpusat (`max-w-md sm:max-w-lg mx-auto`) dan navigasi mobile 5 tab di bagian bawah layar:

| Path | Halaman / Komponen | Kategori | Deskripsi Fungsional |
|---|---|---|---|
| `/portal/attendance` | `AttendancePage` | Presensi | Layar presensi biometrik harian (Check-In & Check-Out) |
| `/portal/enrollment` | `EnrollmentPage` | Biometrik | Wizard pendaftaran template wajah multi-pose mandiri |
| `/portal/consents` | `ConsentPage` | UU PDP | Pembacaan dan penandatanganan dokumen persetujuan biometrik |
| `/portal/history` | `HistoryPage` | Riwayat | Daftar log kehadiran personal per bulan |
| `/portal/history/:id` | `HistoryDetailPage` | Riwayat | Tinjauan detail presensi (foto, jam, status, catatan) |
| `/portal/profile` | `ProfilePage` | Profil | Informasi akun karyawan dan formulir ganti password |

Pengguna yang login langsung diarahkan oleh `SmartRedirect` (`/`):
- Pengguna dengan role `employee` (tanpa permission administratif) langsung diarahkan ke `/portal/attendance`.
- Pengguna dengan permission administratif diarahkan ke dashboard admin (`/`).

---

## 3. Media Pipeline & Penangkapan Kamera (WebRTC)

### 3.1 Pencegahan File Input (Anti-File-Input Compliance)
Sesuai arahan arsitektur keamanan presensi biometrik, seluruh portal karyawan **sama sekali tidak menyediakan opsi unggah file** (`<input type="file">` = 0). Foto wajah wajib diambil langsung dari sensor kamera fisik perangkat pengguna menggunakan WebRTC stream guna memitigasi risiko *presentation attack* berbasis injeksi file statis.

### 3.2 Viewfinder Bercermin vs Piksel Asli Kanvas
- **Viewfinder Video:** Ditampilkan kepada pengguna dengan CSS horizontal mirror (`-scale-x-100`) agar gerakan pengguna terasa alami selayaknya bercermin di kaca.
- **Piksel Tangkapan Kanvas (`src/lib/media/capture.ts`):** Kanvas off-screen mengambil piksel murni dari video **tanpa transformasi mirror** (`ctx.drawImage(video, 0, 0, width, height)`). Hal ini memastikan geometri asimetri wajah alami tetap terjaga untuk model inferensi biometrik InsightFace Buffalo_L.

### 3.3 Kompresi JPEG Adaptif (Keputusan D24)
Untuk mematuhi batasan payload backend `photo.max_bytes` (800 KB) tanpa mengurangi resolusi wajah, fungsi `captureVideoFrame` menerapkan tangga kompresi adaptif 4 tingkat:
1. Skala awal kanvas dibatasi maksimal lebar 960px (menjaga aspek rasio).
2. Kualitas kompresi diuji bertahap: `0.85` $\rightarrow$ `0.75` $\rightarrow$ `0.65` $\rightarrow$ `0.55`.
3. Jika pada kualitas `0.55` ukuran masih $> 800\text{ KB}$, kanvas diperkecil ke lebar 800px dan kualitas diuji ulang.

### 3.4 Evaluasi Kecerahan Real-Time (Keputusan D25)
Modul `src/lib/media/luminance.ts` melakukan analisis frame secara periodik (setiap 300 ms) pada kanvas mini 64×64 piksel menggunakan rumus standar ITU-R BT.601:
$$Y = 0.299R + 0.587G + 0.114B$$
- **$Y < 42$:** Menampilkan indikator "Pencahayaan Redup" (*too_dark*).
- **$Y > 225$:** Menampilkan indikator "Pencahayaan Terlalu Terang" (*too_bright*).
- **$42 \le Y \le 225$:** Menampilkan indikator "Pencahayaan Optimal".
Evaluasi ini bersifat *non-blocking advisory* (tidak mematikan tombol rana), mengedukasi pengguna agar menyesuaikan posisi pencahayaan sebelum menekan tombol ambil foto.

---

## 4. Engine Geolocation & Paritas Geofence

### 4.1 Client-Side Haversine ($R = 6371008.8\text{ m}$)
Modul `src/lib/geo/haversine.ts` menerapkan rumus *spherical Haversine* dengan akurasi presisi tinggi yang identik dengan implementasi Go backend di `apps/faceclock-api/internal/geo/haversine.go`:
- Konstanta radius bumi: $6.371.008,8\text{ meter}$.
- Penanganan selisih bujur melewati batas antimeridian ($-180^\circ$ s/d $+180^\circ$).
- Pemotongan numerik domain akar kuadrat $a \in [0, 1]$.

### 4.2 Deteksi Kantor Terdekat & Visualisasi Radius
Hook `useGeolocation` secara otomatis mengkalkulasi jarak pengguna ke seluruh kantor aktif yang dikirimkan oleh backend pada endpoint `/api/v1/attendances/context`. UI menampilkan:
- Nama kantor terdekat dan radius validasi.
- Status berada di dalam (*inside*) atau di luar (*outside*) area kantor.
- Peringatan akurasi GPS jika sinyal satelit melampaui batas toleransi ($> 150\text{ m}$).

---

## 5. Protokol Idempotensi Presensi

Siklus hidup pengiriman presensi diatur oleh hook `useCheckInFlow.ts` menggunakan pengenal ULID (*Universally Unique Lexicographically Sortable Identifier*):
1. **Inisialisasi Transaksi:** ULID baru dibuat saat foto berhasil ditangkap.
2. **Pengiriman Data:** ULID dikirimkan melalui header HTTP `Idempotency-Key` dan field formulir multipart `idempotency_key`.
3. **Penanganan Retry Jaringan:** Jika koneksi terputus atau server mengembalikan galat sementara (misalnya `504 Gateway Timeout`), pengiriman ulang tetap menggunakan ULID yang sama untuk menjamin data tidak terduplikasi di database.
4. **Pengambilan Foto Baru (Retake):** Jika pengguna memilih mengambil ulang foto, ULID lama dibuang dan ULID baru di-generate.

---

## 6. Kepatuhan Regulasi UU PDP & Zero Metric Leakage

### 6.1 Persetujuan Pemrosesan Data Biometrik (UU PDP No. 27/2022)
Halaman `/portal/consents` menjamin pemenuhan asas legalitas pemrosesan data spesifik:
- **Scroll-to-Bottom Gate:** Tombol persetujuan terkunci hingga pengguna membaca seluruh isi klausul hukum sampai batas terbawah dokumen.
- **Persetujuan Eksplisit:** Mewajibkan konfirmasi centang persetujuan sebelum mengirimkan mutasi persetujuan ke server.
- **Pencabutan Hak (Withdrawal):** Pengguna dapat mencabut persetujuan sewaktu-waktu; sistem akan menonaktifkan seluruh referensi foto wajah secara permanen dan mengembalikan mode presensi ke manual.

### 6.2 Perlindungan Metrik Biometrik (Zero Metric Leakage)
Sesuai rancangan keamanan anti-spoofing [Plan/07-Fase6.md § 2.8], antarmuka karyawan sama sekali tidak mengekspos angka metrik inferensi:
- Tidak ada field `matched_similarity`, `threshold_used`, `similarity_score`, `model_version`, `quality_score`, atau koordinat numerik mentah di layer tampilan.
- Hasil verifikasi hanya dikomunikasikan dalam bentuk status operasional (`Disetujui`, `Perlu Verifikasi`, `Ditolak`) dan panduan perbaikan pose yang konstruktif melalui `HintCoach`.

---

## 7. Penanganan Kesalahan & Fallback Presensi

Jika karyawan mengalami kegagalan pengenalan wajah berturut-turut ($\ge 3$ kali):
1. Tombol **"Opsi Fallback"** muncul secara visual di kartu presensi.
2. Membuka `FallbackModal` untuk memilih alasan kendala (misal: "Iritasi / cedera mata", "Pencahayaan darurat", dsb) serta catatan klarifikasi.
3. Mengirimkan presensi dengan flag `allow_fallback: true`.
4. Transaksi masuk ke antrean verifikasi manual (`status = pending_review`) untuk diverifikasi oleh supervisor di panel admin.
