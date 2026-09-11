# DONE — Fase 4: Attendance Engine, Geofencing, Biometric Verification & Review Workflow

> Ditulis sesuai Protokol Handoff [00MasterPlan.md § 10.4](00MasterPlan.md#10-protokol-handoff-untuk-agent).  
> Melanjutkan fondasi dari [DONE-Fase-0.md](DONE-Fase-0.md), [DONE-Fase-1.md](DONE-Fase-1.md), dan Fase 2 & 3.  
> Dieksekusi secara tuntas, presisi, dan mandiri pada 2026-09-10.

---

## 1. Apa yang Dibangun

Seluruh komponen backend Fase 4 di `apps/faceclock-api` telah diimplementasikan, dimigrasikan, dan diverifikasi secara menyeluruh sesuai spesifikasi [05-Fase4.md](05-Fase4.md) dan [09-Revisions-Log.md](09-Revisions-Log.md):

### 1.1 Database Migrations (`migrations/`)
- **`000018_create_office_locations.{up,down}.sql`** — Tabel master lokasi kantor `office_locations` dengan kolom `id` (uuid PK), `name`, `address`, `latitude` (CHECK $-90 \le \text{lat} \le 90$), `longitude` (CHECK $-180 \le \text{lon} \le 180$), `radius_meter` (CHECK $10 \le \text{radius} \le 5000$), `is_active`, timestamps, `deleted_at` (soft-delete), dan seed default 'Kantor Pusat' (-6.2088, 106.8456, 100m).
- **`000019_create_attendances.{up,down}.sql`** — Tabel transaksi presensi `attendances` dengan 6 domain check constraints:
  - `attendances_type_chk`: `type IN ('check_in', 'check_out')`
  - `attendances_status_chk`: `status IN ('approved', 'rejected', 'pending_review')`
  - `attendances_method_chk`: `method IN ('face_verified', 'fallback_manual', 'fallback_offline', 'system_timeout')`
  - `attendances_fallback_chk`: Ketika `method = 'fallback_manual'`, `fallback_reason` wajib salah satu dari `('camera_failure', 'outside_geofence', 'face_unrecognized', 'device_unsupported', 'system_timeout', 'liveness_failed', 'location_mocked', 'other')`
  - `attendances_review_chk`: Menegakkan keberadaan `reviewed_by` dan `reviewed_at` ketika `status IN ('approved', 'rejected')` dan `method = 'fallback_manual'`
  - `attendances_hash_chk`: `photo_sha256 ~ '^[0-9a-f]{64}$'`
  - Partial unique indexes untuk mencegah duplikasi presensi yang sah pada hari yang sama:
    - `attendances_one_checkin_per_day`: `(employee_id, work_date) WHERE type = 'check_in' AND status <> 'rejected'`
    - `attendances_one_checkout_per_day`: `(employee_id, work_date) WHERE type = 'check_out' AND status <> 'rejected'`
  - Dilengkapi kolom liveness (**REV-DB-04**) dan mock location (**REV-DB-05**).
- **`000020_create_attendance_attempts.{up,down}.sql`** — Tabel telemetri percobaan presensi `attendance_attempts` (pengecualian sadar **REV-CONV-02**: PK `bigint identity`) dengan 10 outcome terstandarisasi (`'matched'`, `'below_threshold'`, `'face_not_usable'`, `'inference_unavailable'`, `'no_reference'`, `'geofence_rejected'`, `'rule_rejected'`, `'duplicate_photo'`, `'rate_limited'`, `'manual_mode'`). Dilengkapi index multi-kolom untuk optimasi sliding window rate limiting.
- **`000021_attendance_settings.{up,down}.sql`** — Penambahan 20 konfigurasi sistem `attendance.*` ke `app_settings` (**REV-SET-03**, **REV-SET-06**, **REV-SET-07**), rekonsiliasi deskripsi `attendance.max_distance_meter` (**REV-SET-04**), serta pendaftaran dan penetapan permissions absensi baru (`attendance.create`, `attendance.review`, `attendance.read_team`, `location.manage`) ke role `super_admin`, `admin`, dan `employee`.

---

### 1.2 Core Packages (`apps/faceclock-api/internal/`)

#### A. `internal/geo/` (Geodesy Engine WGS-84)
- **`haversine.go`**: Perhitungan jarak *Great-Circle* Haversine secara murni fungsional berbasis WGS-84 ($R = 6.371.000\text{ meter}$) dengan optimasi clamping nilai floating-point $\in [-1.0, 1.0]$.
- **`geofence.go`**: Evaluasi status geofence terhadap seluruh kantor aktif non-terhapus, menghasilkan status (`"inside"`, `"outside"`, `"unavailable"`), jarak dalam meter, dan identifikasi kantor terdekat (`nearest_office`).
- **`haversine_test.go`**: Suite pengujian unit yang memverifikasi formula terhadap seluruh kasus uji di `docs/api/geo-testcases.json` (**REV-CTR-03**) dengan presisi tinggi.

#### B. `internal/location/` (Master Data Kantor)
- **`location.go`**: Repository, Service, dan HTTP Handler untuk pengelolaan master lokasi kantor (#66–#70: `GET /locations`, `POST /locations`, `GET /locations/{id}`, `PUT /locations/{id}`, `DELETE /locations/{id}`) dengan otorisasi berbasis `location.manage`.

#### C. `internal/attendance/` (Attendance Core Engine)
- **`rules.go` & `rules_test.go`**: Modul murni fungsional untuk aturan bisnis presensi:
  - `ComputeWorkDate`: Penentuan tanggal kerja berdasarkan ambang batas `workday_cutoff_hour` (default 04:00).
  - `EvaluateLate`: Perhitungan keterlambatan masuk kerja dan selisih menit.
  - `EvaluateEarlyLeave`: Perhitungan pulang mendahului jadwal dan selisih menit.
  - `EvaluateClockSkew`: Deteksi manipulasi waktu pada jam perangkat pengguna ($> 300$ detik).
  - `ComputePhotoSHA256`: Hashing foto biner untuk deteksi replay serangan visual.
- **`models.go` & `dto.go`**: Struktur entitas dan representasi data anti-bocor:
  - **`EmployeeAttendanceDTO`**: Disediakan khusus untuk response karyawan (mengecualikan `matched_similarity`, `threshold_used`, `photo_sha256`) guna mematuhi **Anti-Score Leakage (Rule B7 / D16)**.
  - **`AdminAttendanceDTO`**: Disediakan khusus untuk panel supervisor/admin dengan informasi lengkap (termasuk skor kemiripan, hash foto, data reviewer, dan catatan peninjauan).
- **`attempts.go`**: Repository telemetri percobaan presensi dan penegakan pembatasan laju percobaan gagal (*Sliding-Window Rate Limiting*, **Rule B12**) dengan isolasi JSON marshaling pada hints dan parsing alamat IP `inet`.
- **`service.go`**: Orkestrasi alur presensi end-to-end:
  - `Clock`: Pipeline validasi lengkap (rate limiting -> skew time -> idempotensi -> duplicate photo hash -> aturan jadwal check-in/check-out -> geofence -> biometric 1:1 cosine verification via pgvector -> penyimpanan foto terenkripsi di storage -> transaksi database atomik -> telemetri attempt).
  - `GetContext`: Penyediaan metadata status hari ini, daftar kantor aktif, dan kebijakan sistem.
  - `GetMyToday`, `GetMyHistory`, `GetMySummary`: Query riwayat absensi mandiri karyawan.
  - `GetAdminRecords`, `GetRecordByID`, `GetAttendancePhoto`: Monitoring absensi dan pengaliran foto aman via streaming biner HTTP.
  - `ReviewRecord` & `BulkReviewRecords`: Alur persetujuan/penolakan absensi manual dengan penegakan **Anti-Self Review (Rule B11)**.
- **`retention.go`**: Daemon pembersih foto kadaluarsa (*Background Photo Retention Worker*) berbasis PostgreSQL Advisory Lock `20260904`.
- **`handler.go`**: Controller HTTP untuk endpoints #53–#65 dengan validasi input multipart/form-data dan JSON, serta integrasi audit log (`attendance.clock_in`, `attendance.clock_out`).

#### D. Integrasi Sistem & Routing (`internal/httpx/` & `cmd/api/`)
- Pendaftaran 18 endpoint HTTP di `internal/httpx/routes.go` dan `router.go` dengan RBAC guards ketat (`attendance.checkin`, `attendance.read_self`, `attendance.read_all`, `attendance.read_team`, `attendance.review`, `location.manage`).
- Dependency injection lengkap di `cmd/api/main.go`, termasuk start dan graceful shutdown worker retensi foto harian.

---

## 2. Kepatuhan Aturan Bisnis & Revisi Kontrak

| Aturan / Revisi | Implementasi & Penegakan di Kode |
|---|---|
| **Anti-Score Leakage (Rule B7 / D16)** | Ditegakkan melalui `EmployeeAttendanceDTO`. Skor `matched_similarity` dan `threshold_used` **tidak pernah** dikirimkan kepada karyawan biasa pada endpoint presensi atau riwayat mandiri. |
| **Anti-Replay Photo SHA-256 (Rule B4 / § 2.7c)** | Dihitung via `ComputePhotoSHA256` dan diverifikasi terhadap database `attendances` dalam rentang jendela geser (`duplicate_photo_window_days`, default 7 hari) per karyawan. Replay ditolak dengan 409 `DUPLICATE_PHOTO`. |
| **Anti-Self Review (Rule B11)** | Pada `ReviewRecord` dan `BulkReviewRecords`, sistem memvalidasi bahwa `reviewer_user_id` tidak sama dengan `employee.user_id`. Pelanggaran ditolak keras dengan HTTP 403 `SELF_REVIEW_NOT_ALLOWED`. |
| **Sliding-Window Rate Limiting (Rule B12)** | `CheckRateLimit` menghitung jumlah kegagalan pada `attendance_attempts` dalam 1 jam terakhir. Melebihi `max_failed_attempts_per_hour` (default 5) ditolak dengan HTTP 429 `RATE_LIMITED`. |
| **Fail-Secure Manual Fallback (Rule B2 / D17)** | Ketika inferensi tidak tersedia atau wajah gagal dicocokkan, presensi manual hanya diizinkan jika `allow_fallback = true` dan `attendance.fallback_enabled = true`. Hasilnya berstatus `pending_review` dan wajib ditinjau manusia. |
| **Aturan Interval Presensi (Rule B6)** | Check-out sebelum check-in ditolak dengan 409 `CHECKOUT_WITHOUT_CHECKIN`. Check-out dalam waktu kurang dari 5 menit pasca check-in ditolak dengan 409 `CHECKOUT_TOO_SOON`. |
| **Pembersihan Foto Harian (Rule B10)** | `RetentionWorker` berjalan setiap 24 jam dengan Postgres advisory lock `20260904`, menghapus file foto dari storage fisik, men-NULL-kan `photo_path`, namun mempertahankan `photo_sha256` untuk integritas audit anti-replay. |
| **REV-AUTH-02 (HttpOnly Cookie + Anti-CSRF)** | Seluruh endpoint presensi dan review yang memutasi state (`POST`, `PUT`, `DELETE`) dilindungi oleh Anti-CSRF Shield (`X-Requested-With` / `X-CSRF-Token` + Origin check) dan mendukung sesi HttpOnly cookie murni. |

---

## 3. Bukti Verifikasi

Semua pengujian dan verifikasi dijalankan langsung pada environment pengujian aktif (PostgreSQL 17 di Docker container `faceclock-postgres-1` port 5434, Go 1.26.0 toolchain).

### 3.1 Migrasi Database Bersih (000001–000021)
Seluruh 21 migrasi berhasil diterapkan dari awal:
```
000018_create_office_locations.up.sql
000019_create_attendances.up.sql
000020_create_attendance_attempts.up.sql
000021_attendance_settings.up.sql
```

### 3.2 Hasil Pengujian Suite Unit Test & Integration Test
Eksekusi `go test -p 1 -count=1 ./...` di folder `apps/faceclock-api`:

```
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/attendance   0.516s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/geo          0.042s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx        0.055s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/location     0.112s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/settings     0.068s
ok      github.com/faceclock/faceclock/apps/faceclock-api/test/integration      11.046s
```
**Status: 100% PASS di seluruh paket pengujian.**

### 3.3 Verifikasi Integrasi End-to-End (`test/integration/face_attendance_test.go`)
1. **Flow Check-In Sukses Biometrik**: Foto wajah valid + geofence didalam radius -> Menghasilkan `status: "approved"`, `method: "face_verified"`, dan `hints` kualitatif.
2. **Anti-Score Leakage**: Memverifikasi bahwa response karyawan tidak memuat `matched_similarity` atau `threshold_used`.
3. **Pencegahan Double Check-In**: Percobaan check-in kedua pada work date yang sama ditolak dengan HTTP 409 `ALREADY_CHECKED_IN`.
4. **Anti-Replay Attack**: Menggunakan foto byte yang sama untuk presensi berikutnya ditolak dengan HTTP 409 `DUPLICATE_PHOTO`.
5. **Geofence Enforcement**: Check-in dengan koordinat di luar radius kantor (mis. Monas vs Kantor Pusat) ditolak dengan HTTP 422 `OUTSIDE_GEOFENCE`.
6. **Biometric Mismatch & Fallback Pending Review**: Foto wajah yang tidak cocok dialihkan ke `method: "fallback_manual"` dengan `status: "pending_review"` saat fallback diizinkan.
7. **Anti-Self Review**: Supervisor mencoba menyetujui absensi miliknya sendiri ditolak dengan HTTP 403 `SELF_REVIEW_NOT_ALLOWED`.
8. **Admin Approval Flow**: Admin menyetujui absensi pending milik karyawan lain -> `status` berubah menjadi `"approved"`, `reviewed_by` terisi.
9. **HttpOnly Cookie & CSRF**: Seluruh mutating request presensi dan review terbukti dilindungi Anti-CSRF shield dan kompatibel dengan HttpOnly cookies.

---

## 4. State Database Pasca Fase 4

Kondisi skema di PostgreSQL (`faceclock`):
- `office_locations`: 1 tabel master lokasi kantor (seed default: 'Kantor Pusat' Jakarta)
- `attendances`: 1 tabel transaksi presensi dengan 6 domain constraints dan 2 partial unique indexes
- `attendance_attempts`: 1 tabel telemetri percobaan presensi dengan 10 allowed outcomes
- `app_settings`: 50 konfigurasi sistem terdaftar (termasuk 20 settings absensi & liveness baru)
- `permissions`: 38 permissions terdaftar (ditambah `attendance.create`, `attendance.review`, `attendance.read_team`, `location.manage`)
- Seluruh tabel sebelumnya (`employees`, `users`, `roles`, `refresh_tokens`, `face_references`, `biometric_consents`, dll.) berfungsi terintegrasi secara penuh.

---

## 5. Kesiapan Melangkah ke Fase 5 (Admin Panel Web UI)

Fase 4 (Attendance Engine) telah selesai 100% tanpa catatan hutang teknis.
Fondasi backend untuk Fase 5 (Admin Portal Dashboard, Review Queue, Rekap Presensi, dan Pengelolaan Master Data Lokasi) telah siap digunakan seutuhnya melalui endpoints #53–#70 dan endpoint analitik #71–#73.
