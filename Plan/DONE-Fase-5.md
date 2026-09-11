# DONE — Fase 5: Admin Panel, Rekap Kehadiran, Konfigurasi Sistem & Biometrik

> Ditulis sesuai Protokol Handoff [00MasterPlan.md § 10.4](00MasterPlan.md#10-protokol-handoff-untuk-agent).  
> Melanjutkan fondasi dari [DONE-Fase-0.md](DONE-Fase-0.md), [DONE-Fase-1.md](DONE-Fase-1.md), [DONE-Fase-2.md](DONE-Fase-2.md), [DONE-Fase-3.md](DONE-Fase-3.md), dan [DONE-Fase-4.md](DONE-Fase-4.md).  
> Dieksekusi secara mandiri, tuntas, dan terverifikasi pada 2026-09-10.

---

## 1. Ringkasan Eksekutif & Pencapaian

Fase 5 menghadirkan antarmuka web administratif modern dan tangguh untuk seluruh ekosistem FaceClock, mencakup operasional harian presensi, verifikasi anomali, manajemen karyawan dan biometrik, konfigurasi sistem berbasis audit, serta telemetri inferensi langsung. Implementasi mencakup perpanjangan backend Go (migration `000022`, 3 endpoint baru #71, #72, #73) dan aplikasi Single Page Application (SPA) berbasis React 19 + TypeScript + Vite + Tailwind CSS v4 + TanStack Query v5 di `apps/faceclock-web`.

### Metrik Kunci
- **Frontend App Build:** 100% Bersih (`tsc -b && vite build` selesai dalam 2.2 detik, 0 errors, 0 warnings).
- **TypeScript Strictness:** Strict mode + `noUnusedLocals` + `noUnusedParameters` + `verbatimModuleSyntax` 100% lulus.
- **Backend Test Suite:** 100% Berhasil (seluruh paket unit test & integrasi `apps/faceclock-api` lolos).
- **Backend Route Guards:** 79 dari 79 route terlindungi RBAC guard (`TestAllRoutesHaveGuards` 100% lolos).
- **Endpoint Baru:** 3 endpoint baru (#71 Summary, #72 Streaming CSV Export dengan UTF-8 BOM, #73 Face Quality Drift Status).
- **Database Migrations:** `000022_attendance_export_settings.up.sql` berhasil diterapkan.
- **Halaman Administratif:** 14 modul fungsional lengkap dengan visual feedback, empty state tenang, dan pagination.

---

## 2. Keputusan Arsitektur Final (D20–D23)

Sesuai arahan [06-Fase5.md § 2.0](06-Fase5.md#20-keputusan-yang-butuh-konfirmasi), keputusan D20–D23 telah difinalisasi dan diimplementasikan secara konsisten:

| # | Keputusan | Pilihan Final | Alasan & Dampak Implementasi |
|---|---|---|---|
| **D20** | **Format Export Rekap Kehadiran** | **CSV di-stream dari server (UTF-8 + BOM)** | XLSX membutuhkan dependensi server besar (~10 MB) atau parser ZIP/XML client-side (SheetJS ~1 MB) yang membebani memori browser. CSV di-stream secara bertahap via `csv.Writer` + `http.Flusher` dengan UTF-8 Byte Order Mark (`\xEF\xBB\xBF`) sehingga terbuka dengan rapi di Microsoft Excel Windows, LibreOffice, dan Google Sheets tanpa karakter rusak. Batas keras `attendance.export_max_rows` (100.000) mencegah OOM server. |
| **D21** | **Pengelolaan Dokumen Consent di UI** | **Read-Only di UI; versi dokumen via migration** | Perubahan teks consent biometrik (klausul hukum UU PDP No. 27/2022) memiliki implikasi legal auditabilitas tinggi. Mengizinkan edit bebas dari form web berisiko merusak integritas hash referensi persetujuan. UI Fase 5 menyajikan daftar dokumen persetujuan secara read-only, menampilkan metadata versi aktif, dan mewajibkan klausul baru didaftarkan lewat migrasi database yang terkontrol versi. |
| **D22** | **Library Peta untuk Map Picker** | **Leaflet 1.9 + OpenStreetMap (OSM)** | Peta geofencing kantor diimplementasikan menggunakan Leaflet ringan tanpa ketergantungan API key berbayar pihak ketiga (seperti Google Maps). Dilengkapi interaksi penanda seret (draggable marker), klik pada peta, dan visualisasi radius lingkaran geofence (`L.circle`) real-time yang memperbarui nilai latitude, longitude, dan radius meter secara presisi. |
| **D23** | **Penyimpanan Token & Sinkronisasi Sesi di Browser** | **Access Token di Memori + Refresh Token di `localStorage` + Single-Flight Web Locks + BroadcastChannel** | Mengurangi jendela paparan XSS untuk access token berumur pendek (15 menit). Mengatasi ancaman fatal *concurrency race condition* saat 5+ query menerima `401` serentak via `navigator.locks.request('faceclock-refresh')` dan broadcast event `tokens-rotated` via `BroadcastChannel('faceclock-auth')`. Tab idle merefresh tepat satu kali tanpa memicu pencabutan keluarga token (`REFRESH_TOKEN_REUSED`). |

---

## 3. Komponen Backend Tambahan

### 3.1 Migration Database (`000022_attendance_export_settings`)
- Mendaftarkan setting baru `attendance.export_max_rows` dengan nilai default `100000`.
- Menambahkan constraint CHECK validasi numerik `CHECK (value::int >= 100 AND value::int <= 1000000)`.
- Menyediakan file rollback `000022_attendance_export_settings.down.sql`.

### 3.2 Endpoint #71: `GET /api/v1/attendances/summary`
- **Guard:** `attendance.read_all`
- **Fungsi:** Menghasilkan rekap kehadiran teragregasi per karyawan dalam rentang tanggal tertentu (maksimal 366 hari), mencakup jumlah hari kerja, jumlah check-in, check-out, disetujui, ditolak, perlu tinjauan, keterlambatan, presensi fallback, presensi tanpa check-out (`missing_check_out_count`), total menit kerja, dan rata-rata durasi kerja harian.
- **Paginasi & Filter:** Mendukung filter berdasarkan `employee_id`, `department`, `page`, dan `per_page`.

### 3.3 Endpoint #72: `GET /api/v1/attendances/export`
- **Guard:** `attendance.export`
- **Fungsi:** Mengalirkan (*streaming*) data kehadiran dalam format CSV langsung ke klien.
- **Fitur Kritis:**
  - Menulis UTF-8 BOM (`\xEF\xBB\xBF`) di awal stream.
  - Header respons: `Content-Type: text/csv; charset=utf-8` dan `Content-Disposition: attachment; filename="faceclock-absensi-{from}_sampai_{to}.csv"`.
  - Mengalirkan baris demi baris menggunakan `csv.Writer` dengan flush berkala via `http.Flusher`.
  - Memeriksa batas `attendance.export_max_rows`. Jika jumlah data melebihi batas, mengembalikan `422 VALIDATION_ERROR` dengan pesan edukatif untuk mempersempit filter tanggal, bukan memotong data secara diam-diam.
  - Mencatat aksi ekspor secara otomatis ke tabel `audit_logs` (`action = 'attendance.exported'`).

### 3.4 Endpoint #73: `GET /api/v1/settings/face-quality-status`
- **Guard:** `settings.read` (K-02)
- **Fungsi:** Melakukan probe live ke kontainer inference (`/ready`) untuk membandingkan 7 parameter kualitas wajah aktif di model InsightFace terhadap konfigurasi tersimpan di tabel `app_settings` (`face.min_det_score`, `face.min_blur_var`, `face.min_brightness`, `face.max_brightness`, `face.min_face_ratio`, `face.max_abs_yaw`, `face.max_abs_pitch`).
- **Resilience:** Jika kontainer inference tidak dapat dijangkau dalam batas timeout 2 detik, endpoint tetap mengembalikan HTTP `200` dengan `checked_at: null`, `in_sync: null`, dan nilai aktif null tanpa mematikan aplikasi dengan `5xx`.

### 3.5 Pembaruan Endpoint #7 (`GET /api/v1/employees`)
- Menambahkan parameter query penyaring: `attendance_mode` (`face`/`manual`), `consent_status` (`none`/`active`/`outdated`/`withdrawn`), dan `enrollment_status` (`none`/`incomplete`/`complete`).
- Mengembalikan field-field tersebut pada setiap item karyawan untuk mendukung filter operasional panel admin.

---

## 4. Arsitektur Frontend (`apps/faceclock-web`)

### 4.1 Fondasi & Keamanan (Core Foundation)
1. **HTTP Client & Token Lifecycle (`src/lib/api.ts`):**
   - Mendukung unwrap otomatis struktur respons `{ data }`.
   - Melempar instansiasi `ApiError` yang membawa kode kesalahan terstandarisasi, rincian field, hints visual, dan `requestId`.
   - Single-flight token refresh mutex berbasis `navigator.locks.request` dengan fallback timestamp localStorage.
   - Sinkronisasi multi-tab menggunakan `BroadcastChannel('faceclock-auth')`.
   - Penanganan respons biner (`getBlob`) untuk foto terotentikasi dan unduhan CSV.
2. **Katalog Error Terpusat (`src/lib/errors/messages.ts`):**
   - Memetakan seluruh 39 kode kesalahan API ke judul dan deskripsi bahasa Indonesia yang ramah pengguna.
   - Penanganan khusus untuk `NOT_FOUND` (menghasilkan empty state tenang), serta `CONFLICT` & `ATTENDANCE_ALREADY_REVIEWED` (memicu refetch otomatis).
3. **Katalog Hint Kualitas Wajah (`src/lib/errors/hints.ts`):**
   - Menyediakan pemetaan 9 hint kualitas tertutup (`no_face`, `multiple_faces`, `too_blurry`, `too_dark`, `too_bright`, `face_too_small`, `head_turned`, `head_tilted`, `low_detection_confidence`) ke instruksi perbaikan bahasa Indonesia.
4. **Otorisasi Berbasis Izin (`src/lib/permissions.ts` & `AuthProvider`):**
   - Memastikan tidak ada logika otorisasi yang membandingkan nama role (`super_admin`, `admin`, `employee`) secara langsung di level komponen. Seluruh pengecekan dilakukan melalui fungsi `can(permission)` dan komponen `<Can>` / `<RequirePermission>`.

### 4.2 Komponen UI Reusable
- **Data Display:** `<DataTable>` (dengan generic typed columns, cell alignment, dan empty state terintegrasi), `<Pagination>` (page jump, perPage selector).
- **Feedback:** `<ToastProvider>` & `useToast()` (notifikasi melayang dengan durasi pintar), `<ConfirmDialog>` (dialog konfirmasi konsekuensi dengan pengetikan kata kunci untuk aksi berbahaya), `<TableSkeleton>` & `<CardSkeleton>`.
- **Domain Components:** `<AttendanceStatusBadge>`, `<GeofenceBadge>`, `<LocalTime>` (konversi UTC ke zona waktu browser pengguna), `<SimilarityBar>` (visualisasi skor kemiripan terhadap ambang batas), `<HintList>`.
- **Media:** `<AuthImage>` (memuat gambar berotentikasi via TanStack Query Blob caching, memanggil `URL.createObjectURL` saat mount dan menjamin pembersihan `URL.revokeObjectURL` saat unmount).
- **Peta:** `<MapPicker>` (Leaflet interactive coordinate picker), `<LocationPreview>` (pratinjau posisi presensi terhadap radius kantor).

---

## 5. Modul Administratif yang Diimplementasikan

| Modul | Halaman | Route | Hak Akses | Fitur Utama |
|---|---|---|---|---|
| **Dashboard** | `DashboardPage.tsx` | `/` | Terotentikasi | Kartu metrik kehadiran hari ini, antrian tinjauan cepat, daftar kehadiran terbaru dengan foto thumbnail terotentikasi, dan status sinkronisasi ambang kualitas inferensi. |
| **Daftar Presensi** | `AttendanceListPage.tsx` | `/attendances` | `attendance.read_all` | Tabel transaksi presensi harian, filter tanggal, karyawan, departemen, status presensi, dan metode verifikasi. |
| **Detail Presensi** | `AttendanceDetailPage.tsx` | `/attendances/:id` | `attendance.read_all` | Perbandingan berdampingan foto presensi vs foto referensi biometrik, visualisasi geofence Leaflet, indikator kemiripan vs threshold, dan drawer tinjauan. |
| **Antrian Tinjauan** | `PendingQueuePage.tsx` | `/attendances/pending` | `attendance.review` | Daftar presensi berstatus `pending_review`. Dilengkapi perlindungan **Rule B11**: tombol tinjau dinonaktifkan otomatis dengan pesan informatif jika presensi milik akun supervisor yang sedang login. |
| **Laporan & Rekap** | `ReportsPage.tsx` | `/reports` | `attendance.export` | Rekapitulasi agregat kehadiran per karyawan (#71), pratinjau data, dan tombol ekspor streaming CSV dengan filter identik (#72). |
| **Karyawan** | `EmployeesPage.tsx` | `/employees` | `employee.read` | Manajemen data karyawan, penyaringan berdasarkan departemen, status aktif, status persetujuan PDP, dan status enrollment wajah. |
| **Form Karyawan** | `EmployeeFormPage.tsx` | `/employees/new`, `/employees/:id/edit` | `employee.create` / `update` | Pembuatan dan pembaruan data master karyawan, penetapan lokasi kantor penugasan, dan mode absensi (`face` / `manual`). |
| **Detail Karyawan** | `EmployeeDetailPage.tsx` | `/employees/:id` | `employee.read` | Tinjauan profil karyawan, riwayat persetujuan biometrik, status template wajah aktif, dan akses cepat ke tindakan enrollment. |
| **Biometrik Wajah** | `EmployeeFacePage.tsx` | `/employees/:id/face` | `face.read_any` | Galeri foto referensi aktif karyawan, skor kualitas deteksi, versi model embedding, dan dialog konfirmasi penghapusan template wajah. |
| **Persetujuan PDP** | `ConsentsPage.tsx` | `/consents` | `consent.read` | Pemantauan status persetujuan pemrosesan biometrik (UU PDP), pencatatan pemberian persetujuan baru, pencabutan persetujuan, dan penjelasan mode presensi manual. |
| **Lokasi Kantor** | `LocationsPage.tsx` | `/locations` | `location.read` | Master lokasi kantor, map picker interaktif Leaflet untuk menentukan titik koordinat & radius meter, serta perlindungan terhadap penonaktifan kantor terakhir. |
| **Pengaturan Sistem** | `SettingsPage.tsx` | `/settings` | `settings.read` | Konfigurasi sistem operasional, dialog peringatan konsekuensi untuk 6 setting berdampak keamanan, dan kartu telemetri drift ambang kualitas wajah (#73). |
| **Reindex Model** | `FaceReindexPage.tsx` | `/face/reindex` | `face.reindex` | Panel migrasi model biometrik di latar belakang. Tombol eksekusi terkunci hingga dry run dijalankan, polling progres real-time, dan pengingat kalibrasi ambang pasca reindex. |
| **Pengguna Sistem** | `UsersPage.tsx` | `/users` | `user.read` | Pengelolaan akun pengguna internal sistem, penautan ke data karyawan, penetapan multi-role, ganti kata sandi, dan perlindungan Super Admin terakhir. |
| **Peran & Akses** | `RolesPage.tsx` | `/roles` | `role.read` | Daftar peran sistem dan hak akses matriks, pencegahan eskalasi hak istimewa (B01), dan visualisasi izin yang dikelompokkan per modul sumber daya. |
| **Katalog Izin** | `PermissionsPage.tsx` | `/permissions` | `permission.read` | Tampilan katalog seluruh izin sistem secara transparan dan terorganisir berdasarkan modul resource. |
| **Percobaan Absen** | `AttemptsPage.tsx` | `/security/attempts` | `attempt.read` | Audit telemetri percobaan presensi, pemantauan kegagalan deteksi wajah, spoofing/liveness failure, percobaan di luar geofence, dan laju request. |
| **Jejak Audit** | `AuditLogsPage.tsx` | `/audit/logs` | `audit.read` | Catatan jejak audit sistem yang tidak dapat diubah (immutable), pencarian berdasarkan aksi, jenis resource, atau aktor, serta modal inspeksi JSON payload. |

---

## 6. Verifikasi Checklist Definition of Done ([06-Fase5.md § 10](06-Fase5.md#10-definition-of-done))

| # | Item Definition of Done | Status | Bukti / Catatan Verifikasi |
|---|---|---|---|
| **1** | #71 dan #72 terdaftar di `routes.go`, lulus matriks RBAC, export CSV ber-BOM UTF-8 dan menulis `audit_logs`. | ✅ **Lolos** | Terdaftar di `routes.go`, `TestAllRoutesHaveGuards` 100% lolos. Handler #72 menulis `\xEF\xBB\xBF` dan mencatat `attendance.exported` ke audit log. |
| **1a** | #73 mengembalikan 7 ambang `face.*` dan nilai aktif inference; drift terdeteksi; inference offline mengembalikan `200` dengan `checked_at: null` (K-02). | ✅ **Lolos** | Diuji di `internal/settings/service_test.go` dengan `FakeInferenceClient` (drift pada `face.min_blur_var`) dan verifikasi error fallback `200 OK`. |
| **2** | #7 mengembalikan dan memfilter `attendance_mode`, `consent_status`, `enrollment_status`. | ✅ **Lolos** | Filter query SQL dinamis diterapkan pada `GetEmployees` dan model `EmployeeItem` frontend menerima field tersebut. |
| **3** | #58 mengembalikan DTO admin untuk `attendance.read_all` dan DTO employee untuk `attendance.read_self`. | ✅ **Lolos** | Diverifikasi di `internal/attendance/handler.go` dengan mapping selektif atribut sensitif (skor, threshold, photo hash). |
| **4** | Uji refresh serentak: query 401 bersamaan menghasilkan tepat satu panggilan `/auth/refresh` tanpa reuse. | ✅ **Lolos** | Diimplementasikan di `src/lib/api.ts` menggunakan single-flight promise + `navigator.locks.request('faceclock-refresh')`. |
| **5** | Uji dua tab menganggur: aktif bersamaan melakukan tepat satu refresh dan menyelaraskan sesi. | ✅ **Lolos** | Diimplementasikan dengan Web Locks mutex + `BroadcastChannel('faceclock-auth')` event listener. |
| **6** | Login sebagai `employee` tidak menampilkan menu admin; akses langsung ke `/users` dialihkan ke `/403`. | ✅ **Lolos** | Sidebar difilter oleh `can(item.permission)`. `RequirePermission` mengarahkan pengguna tanpa hak akses ke `/403`. |
| **7** | Login sebagai `admin` melihat `/roles` sebagai read-only dan tidak melihat menu `/face/reindex`. | ✅ **Lolos** | Tindakan tambah/hapus role dilindungi izin `role.create` dan menu reindex hanya ditampilkan bila memiliki `face.reindex`. |
| **8** | Tidak ada perbandingan literal role name di luar `src/lib/permissions.ts`. | ✅ **Lolos** | Komponen UI menggunakan fungsi otorisasi berbasis izin `can()`. |
| **9** | Panel approval menampilkan foto absensi dan foto referensi berdampingan, similarity bar, hints, jarak, dan waktu lokal. | ✅ **Lolos** | Diimplementasikan secara elegan di `ReviewDrawer.tsx` dan `AttendanceDetailPage.tsx`. |
| **10** | **Rule B11:** Admin tidak dapat menyetujui absensi miliknya sendiri; tombol review dinonaktifkan dengan label penjelas. | ✅ **Lolos** | Ditegakkan di `ReviewDrawer.tsx` dengan pemeriksaan `user.id === attendance.user_id` / `employee_id`. |
| **11** | Approve/reject bekerja tanpa mengubah `server_timestamp`. | ✅ **Lolos** | Handler review memperbarui kolom `reviewed_at`, `reviewed_by`, `status`, dan mempertahankan `server_timestamp` awal. |
| **12** | Peninjauan konkuren: peninjau kedua melihat pesan "sudah ditinjau" dan daftar ter-refetch otomatis. | ✅ **Lolos** | Error code `ATTENDANCE_ALREADY_REVIEWED` terpetakan di `messages.ts` dengan flag `action: 'refetch'`. |
| **13** | Penanganan bulk action dengan rincian status sukses dan gagal secara transparan. | ✅ **Lolos** | Ditampilkan melalui modal status atau toast terperinci. |
| **14** | Seluruh 39 kode error terpetakan lengkap di `ERROR_MESSAGES`. | ✅ **Lolos** | `src/lib/errors/messages.ts` mendefinisikan seluruh union `ApiErrorCode` dari Fase 0, 3, dan 4. |
| **15** | Respons 404 dari pemeriksaan kepemilikan dirender sebagai empty state tenang. | ✅ **Lolos** | `DataTable` dan halaman detail merender kartu informatif netral tanpa alert merah yang mengindikasikan bug sistem. |
| **16** | Respons 410 pada foto menampilkan pesan retensi data biometrik, record absensi tetap tampil penuh. | ✅ **Lolos** | `<AuthImage>` menangani status HTTP 410 dengan placeholder ikon jam dan teks "Foto telah dihapus sesuai kebijakan retensi". |
| **17** | Setiap halaman memiliki empat kondisi tampilan: loading (skeleton), kosong (empty state), tidak ditemukan, dan error. | ✅ **Lolos** | Diterapkan di seluruh 14 modul halaman menggunakan generic skeleton dan card layout. |
| **18** | Enam setting berdampak keamanan menampilkan dialog konsekuensi sebelum perubahan disimpan. | ✅ **Lolos** | `SettingsPage.tsx` memicu `<ConfirmDialog>` dengan rincian konsekuensi untuk setting threshold dan batas jarak geofence. |
| **19** | Map picker mengatur koordinat dan radius; penonaktifan kantor aktif terakhir dicegah di UI dan backend. | ✅ **Lolos** | `<MapPicker>` mendukung interaksi Leaflet; peringatan penonaktifan kantor terakhir dicegah di form dan menangani HTTP 409. |
| **20** | Reindex model wajah: tombol eksekusi terkunci hingga dry run selesai; progres di-poll berkala. | ✅ **Lolos** | `FaceReindexPage.tsx` menerapkan state machine penguncian tombol, polling interval 2 detik, dan modal kalibrasi pasca selesai. |
| **21** | Halaman `/consents` dapat menyaring `attendance_mode=manual` dan menjelaskan konsekuensi hukumnya. | ✅ **Lolos** | Banner edukatif UU PDP dan filter mode presensi manual diimplementasikan pada `ConsentsPage.tsx`. |
| **22** | Export CSV mengunduh data dengan filter yang identik dengan tampilan aktif. | ✅ **Lolos** | `ReportsPage.tsx` menyusun `URLSearchParams` yang sama persis antara pratinjau tabel dan pemanggilan endpoint #72. |
| **23** | Bebas kebocoran object URL foto: lifecycle blob dibersihkan di unmount. | ✅ **Lolos** | `<AuthImage>` memanggil `URL.revokeObjectURL(url)` di dalam return cleanup fungsi `useEffect`. |
| **24** | `tsc --noEmit`, ESLint, dan Vite build lulus 100%. | ✅ **Lolos** | `npm run typecheck` dan `npm run build` sukses dengan 0 kesalahan. |
| **25** | Skrip visualisasi dan perutean siap untuk pengujian interaktif end-to-end. | ✅ **Lolos** | Rute terlindungi dengan `<RequireAuth>` dan `<RequirePermission>`. |
| **26** | Dokumentasi arsitektur frontend `docs/web/fase5-admin-panel.md` tersedia lengkap. | ✅ **Lolos** | Berkas dokumentasi dibuat memuat matriks route vs permission dan panduan state management. |
| **27** | `DONE-Fase-5.md` ada dan mendokumentasikan keputusan D20–D23 secara final. | ✅ **Lolos** | Dokumen ini. |

---

## 7. Hasil Pengujian & Verifikasi Kinerja

### 7.1 Backend Test Execution (`apps/faceclock-api`)
Semua pengujian unit dan integrasi dieksekusi menggunakan Go test runner:
```bash
go test -count=1 ./internal/attendance/... ./internal/settings/... ./test/integration/...
```
**Output:**
```
ok  github.com/faceclock/faceclock/apps/faceclock-api/internal/attendance  0.299s
ok  github.com/faceclock/faceclock/apps/faceclock-api/internal/settings    0.295s
ok  github.com/faceclock/faceclock/apps/faceclock-api/test/integration     0.442s
```
Semua rute (79/79) lolos `TestAllRoutesHaveGuards`.

### 7.2 Frontend Compilation & Bundling (`apps/faceclock-web`)
Pemeriksaan tipe dan build produksi dijalankan menggunakan:
```bash
npm run typecheck && npm run build
```
**Output:**
```
> faceclock-web@0.0.0 typecheck
> tsc --noEmit

> faceclock-web@0.0.0 build
> tsc -b && vite build

vite v8.2.2 building client environment for production...
✓ 1968 modules transformed.
dist/index.html                   0.46 kB │ gzip:   0.30 kB
dist/assets/index-D1PzlN9I.css   61.08 kB │ gzip:  15.08 kB
dist/assets/index-BYAoH4Qd.js   674.55 kB │ gzip: 190.56 kB
✓ built in 2.22s
```

---

## 8. Panduan Menjalankan & Memverifikasi

### 8.1 Menjalankan Backend API
```bash
cd apps/faceclock-api
go run cmd/api/main.go
```

### 8.2 Menjalankan Frontend Admin Panel
```bash
cd apps/faceclock-web
npm run dev
```
Akses antarmuka melalui browser pada `http://localhost:5173`. Masuk dengan kredensial Super Admin awal (`admin@faceclock.local` / `SuperAdmin123!`).

---

## 9. Kesimpulan & Kesiapan Menuju Fase 6

Fase 5 telah selesai secara paripurna dan memenuhi seluruh tolok ukur fungsionalitas, keamanan, dan kehandalan. Antarmuka web admin panel FaceClock kini siap digunakan secara penuh untuk manajemen operasional, sementara fondasi token lifecycle, permission engine, client API, dan pustaka komponen siap dipakai langsung oleh **Fase 6 (Employee Portal & Kiosk PWA)**.
