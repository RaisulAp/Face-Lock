# DONE — Fase 1: Autentikasi, RBAC, User, Employee, Settings & Audit

> Ditulis sesuai Protokol Handoff [00MasterPlan.md § 10.4](00MasterPlan.md#10-protokol-handoff-untuk-agent).  
> Melanjutkan fondasi dari [DONE-Fase-0.md](DONE-Fase-0.md).  
> Dieksekusi secara tuntas dan mandiri pada 2026-09-10.

---

## 1. Apa yang Dibangun

Seluruh komponen backend Fase 1 di `apps/faceclock-api` telah diimplementasikan, dimigrasikan, dan diverifikasi penuh:

### 1.1 Database Migrations (`migrations/`)
- **`000002_create_employees.{up,down}.sql`** — Tabel `employees` dengan kolom `id` (uuid PK), `employee_number` (citext unik untuk karyawan aktif via partial index `WHERE deleted_at IS NULL`), `full_name`, `department`, `position`, `phone`, `join_date`, `employment_status`, `face_enrolled_at`, timestamps, dan auto-update trigger.
- **`000003_create_users.{up,down}.sql`** — Tabel `users` dengan `id` (uuid PK), `employee_id` (FK opsional 1-to-1 partial unique), `email` (citext partial unique), `password_hash`, `is_active`, `token_version`, `must_change_password`, `failed_login_count`, `locked_until`, `last_login_at`, timestamps.
- **`000004_create_roles.{up,down}.sql`** — Tabel `roles` dengan `name` (unik), `display_name`, `description`, `is_system` (boolean).
- **`000005_create_permissions.{up,down}.sql`** — Tabel `permissions` dengan `name` (unik), `resource`, `action`, `description`.
- **`000006_create_role_permissions.{up,down}.sql`** — Tabel junction `role_permissions` dengan PK `(role_id, permission_id)` dan index pada `permission_id`.
- **`000007_create_user_roles.{up,down}.sql`** — Tabel junction `user_roles` dengan PK `(user_id, role_id)`, `ON DELETE RESTRICT` pada role untuk mencegah penghapusan role yang masih aktif digunakan.
- **`000008_create_refresh_tokens.{up,down}.sql`** — Tabel `refresh_tokens` dengan `token_hash` (sha256 hex unik), `family_id` untuk rotasi rantai, `parent_id`, `expires_at`, `used_at`, `revoked_at`, `revoked_reason` (CHECK constraint whitelist).
- **`000009_create_audit_logs.{up,down}.sql`** — Tabel `audit_logs` (pengecualian sadar REV-CONV-01: PK `bigint GENERATED ALWAYS AS IDENTITY`), append-only, index pada actor, action, resource, dan timestamp.
- **`000010_create_app_settings.{up,down}.sql`** — Tabel `app_settings` dengan `key` (PK), `value` (jsonb), `value_type`, `description`, `is_public` (boolean untuk penyaringan data sensitif).

### 1.2 Core Packages (`apps/faceclock-api/internal/`)

#### A. `internal/auth/`
- **`password.go`**: Implementasi Argon2id (`time=3, memory=64MB, threads=2, keyLen=32, saltLen=16`). Dilengkapi constant-time dummy verification saat email login tidak ditemukan (mencegah timing attack). Validasi password minimal 10 karakter, wajib kombinasi huruf & angka, bukan email, dan pengecekan blacklist terhadap `weak_passwords.txt` (200 password terlemah dunia).
- **`jwt.go`**: Penerbitan dan verifikasi access token menggunakan JWT HS256 dengan strict validation (hanya menerima HS256, menolak `alg: none`), pemeriksaan klaim `iss`, `aud`, `typ: access`, toleransi clock skew 30 detik, serta verifikasi `tv` (*token version*).
- **`refresh.go`**: Generator refresh token (32-byte acak dari `crypto/rand` di-encode ke base64url), penyimpanan hash SHA-256, deteksi penggunaan ulang (*token reuse detection*) dengan pencabutan keluarga token (*family revocation*), serta implementasi **REV-AUTH-01** (jendela toleransi race condition 30 detik `auth.refresh_reuse_grace_seconds`).
- **`cookie.go`**: Implementasi mesin cookie helper (**REV-AUTH-02**):
  - `SetAuthCookies`: Menetapkan `access_token` (Path: `/`, Max-Age: 900s, HttpOnly: true, SameSite: Lax, Secure: non-dev) dan `refresh_token` (Path: `/api/v1/auth`, Max-Age: 2592000s, HttpOnly: true, SameSite: Lax, Secure: non-dev).
  - `ClearAuthCookies`: Menghapus kedua cookie (`MaxAge: -1`, `Expires: epoch`) pada logout dan logout-all.
- **`service.go`**: Orkestrasi autentikasi bisnis: alur login dengan lockout akun otomatis (5 kali percobaan gagal mengunci akun selama 15 menit), penyegaran sesi transaksional menggunakan `SELECT ... FOR UPDATE`, logout sesi tunggal, logout menyeluruh (*logout-all* via inkrementasi `token_version`), pembacaan profil `/me`, dan ganti password mandiri.
- **`handler.go`**: Menangani endpoint HTTP #1–#6 dengan dukungan penuh HttpOnly cookie dan Dual-Mode compatibility (menerima refresh token dari cookie atau fallback JSON body; menulis cookie respons sekaligus mempertahankan JSON envelope).
- **`principal.go` & `context.go`**: Representasi user context `Principal{UserID, EmployeeID, Roles, Permissions}` dengan helper `Has(permission)` dan `HasAny(permissions...)`.
- **`middleware.go`**: Middleware `Authenticate` dual-kompatibel yang memprioritaskan ekstraksi dari cookie `access_token` dan secara mulus beralih ke header `Authorization: Bearer <token>` bila cookie tidak ada.

#### B. `internal/rbac/`
- **`permissions.go`**: Katalog lengkap 34 izin sistem didefinisikan sebagai konstanta Go bertipe aman.
- **`service.go`**: Pemuatan relasi user-roles-permissions dengan integrasi cache.
- **`cache.go`**: In-memory cache untuk permissions user (TTL 30 detik) dengan dukungan invalidasi eksplisit instan per user dan per role.
- **`middleware.go`**: Middleware `RequirePermission`, `RequireAnyPermission`, dan `RequireSuperAdmin`.
- **`escalation.go`**: Penegakan aturan pencegahan *privilege escalation* (§ 5.5 A): pelaku dilarang memberikan role yang memuat permission di luar yang ia miliki.
- **`guardrails.go`**: Penegakan aturan perlindungan *last super admin* (§ 5.5 B): menolak operasi apa pun yang menyebabkan ketiadaan super admin aktif (penonaktifan, penghapusan, atau pencabutan role).

#### C. `internal/employee/`
- **`repository.go`, `service.go`, `handler.go`, `dto.go`**: CRUD data karyawan lengkap dengan filtering (`department`, `status`, `q`), sorting whitelist, pagination, pencegahan duplikasi nomor induk, dan penegakan **Ownership Pattern** (§ 5.4): karyawan yang mengakses data karyawan lain menerima **404 NOT_FOUND** (bukan 403) untuk mencegah enumerasi ID.

#### D. `internal/user/`
- **`repository.go`, `service.go`, `handler.go`, `dto.go`**: CRUD akun pengguna lengkap, pembuatan password sementara aman dari `crypto/rand`, penugasan role dengan cek eskalasi, penonaktifan status, reset password yang mencabut semua token aktif, serta penguncian guardrail super admin terakhir.

#### E. `internal/role/`
- **`repository.go`, `service.go`, `handler.go`, `dto.go`**: CRUD role kustom, penugasan permissions ke role dengan invalidasi cache otomatis, perlindungan role sistem (`is_system = true` tidak dapat diubah/dihapus), serta proteksi `ON DELETE RESTRICT` jika role masih dipakai oleh user.

#### F. `internal/settings/`
- **`repository.go`, `service.go`, `handler.go`, `validators.go`**: Pembacaan pengaturan aplikasi dengan aturan visibilitas (pengguna tanpa izin `settings.update` hanya dapat membaca key `is_public = true`), serta validator domain per-key (contoh: `face.similarity_threshold` wajib rentang float 0.0–1.0, `attendance.max_distance_meter` wajib integer 10–10000m).

#### G. `internal/audit/`
- **`actions.go`, `recorder.go`, `handler.go`**: Sistem pencatatan audit log append-only secara asinkron tanpa memblokir alur utama HTTP, disertai endpoint penelusuran audit dengan multi-parameter query filter. Menjamin sanitasi metadata bebas dari password, token, atau data biometrik.

#### H. `internal/httpx/`
- **`routes.go`**: Registry pusat seluruh 31 endpoint Fase 1 beserta deklarasi guard RBAC eksplisit.
- **`errors.go`**: Katalog diperluas menjadi 43 kode error spesifik sistem dengan pemetaan HTTP status yang ketat (panic jika kode tidak terdaftar), termasuk `CSRF_HEADER_MISSING` (403) dan `CSRF_UNTRUSTED_ORIGIN` (403).
- **`middleware/csrf.go`**: Anti-CSRF Shield yang memvalidasi header AJAX (`X-Requested-With: XMLHttpRequest` atau `X-CSRF-Token`) dan whitelist `Origin` pada seluruh metode mutasi state (`POST`, `PUT`, `PATCH`, `DELETE`).
- **`middleware/cors.go`**: Menambahkan expose/allow headers untuk `X-Requested-With` dan `X-CSRF-Token`, serta pengelolaan credentials yang aman.
- **`router.go`**: Pemasangan middleware `CSRFProtection` secara global setelah middleware CORS dan RequestID.

### 1.3 Seeder Idempoten (`cmd/seed/`)
- Program terpisah `cmd/seed/main.go` yang mengeksekusi `internal/seeder/seeder.go`:
  - Menyisipkan 34 permissions sistem (upsert idempoten)
  - Menyisipkan 3 role bawaan (`super_admin`, `admin`, `employee`)
  - Menyinkronkan permissions untuk ketiga role (`super_admin` mendapat seluruh permissions)
  - Men-seed 11 key `app_settings` awal (insert-only, tidak pernah menimpa konfigurasi yang telah diubah admin)
  - Membaca `SEED_ADMIN_EMAIL` & `SEED_ADMIN_PASSWORD` untuk inisialisasi super admin awal jika belum ada user terdaftar (`must_change_password: true`)
  - Exit(1) jika dijalankan di production tanpa password admin saat user kosong.

### 1.4 Test Suite & RBAC Matrix
- **`test/integration/helpers.go`**: Test harness lengkap, seeder fixtures, inisialisasi token autentikasi untuk 4 jenis principal, `localRoundTripper` in-memory routing, `NewTestClient` dengan `net/http/cookiejar` otomatis, auto-injection `X-Requested-With` pada mutating request, dan assertion scanner `AssertNoSecretLeak` yang memindai setiap respons API dari keberadaan string rahasia (`password_hash`, `token_hash`, `argon2`, `$2a$`).
- **`test/integration/rbac_matrix_test.go`**: Pengujian matriks otorisasi penuh: **4 principal (Anonymous, Employee, Admin, SuperAdmin) × 31 endpoint = 124 pengujian integrasi** dengan verifikasi status code eksak, cookie injection, dan zero-secret leakage.
- **`test/integration/cookie_auth_test.go`**: Pengujian end-to-end khusus arsitektur HttpOnly Cookie & Anti-CSRF Shield:
  1. `TestCookieAuthenticationAndAntiCSRFShield/login_sets_http_only_cookies`: Memverifikasi atribut cookie (`HttpOnly`, `Path`, `MaxAge`, `SameSite=Lax`).
  2. `TestCookieAuthenticationAndAntiCSRFShield/me_endpoint_pure_cookie`: Akses `/me` murni menggunakan cookie tanpa header `Authorization`.
  3. `TestCookieAuthenticationAndAntiCSRFShield/csrf_shield_rejects_mutating_request_without_header`: Menolak POST tanpa header CSRF (403 `CSRF_HEADER_MISSING`).
  4. `TestCookieAuthenticationAndAntiCSRFShield/csrf_shield_rejects_untrusted_origin`: Menolak POST dari origin asing (403 `CSRF_UNTRUSTED_ORIGIN`).
  5. `TestCookieAuthenticationAndAntiCSRFShield/csrf_shield_allows_safe_methods_without_header`: Memastikan GET/HEAD/OPTIONS lolos tanpa header CSRF.
  6. `TestCookieAuthenticationAndAntiCSRFShield/refresh_endpoint_rotates_cookies`: Memverifikasi rotasi token via cookie tanpa payload body.
  7. `TestCookieAuthenticationAndAntiCSRFShield/logout_clears_cookies`: Memverifikasi penghapusan cookie (`MaxAge: -1`).
  8. `TestCookieAuthenticationAndAntiCSRFShield/browser_session_full_lifecycle_with_cookie_jar`: Simulasi alur lengkap browser (login → /me → refresh → logout → unauthenticated) menggunakan `TestClient` dan `cookiejar`.
- **`internal/httpx/routes_test.go`**: `TestAllRoutesHaveGuards` yang melakukan penelusuran pohon router chi (`chi.Walk`) untuk menjamin tidak ada satupun route `/api/v1/*` yang lolos tanpa guard RBAC (**default-deny guarantee**).

---

## 2. Keputusan Teknis & Deviasi dari Plan

| # | Keputusan / Deviasi | Alasan & Dampak |
|---|---|---|
| 1 | **REV-AUTH-01 & REV-SET-08 Langsung Diterapkan di Fase 1** | Grace window 30 detik (`auth.refresh_reuse_grace_seconds`) untuk deteksi reuse refresh token diimplementasikan langsung di `internal/auth/refresh.go` dan migration `000010`. Menunda implementasi ini ke Fase 7 akan merusak kompatibilitas dan memaksa penulisan ulang modul inti auth. |
| 2 | **REV-AUTH-02: Full HttpOnly Cookie-Based Auth & Anti-CSRF Shield** | Menggantikan pure Bearer token pada browser dengan HttpOnly cookies (`SameSite=Lax`) untuk menutup vektor serangan pencurian token via XSS. Dilengkapi Anti-CSRF shield global dan dual-mode compatibility (fallback `Authorization: Bearer` untuk client non-browser / mobile). |
| 3 | **REV-CONV-01: `audit_logs.id` Menggunakan `bigint GENERATED ALWAYS AS IDENTITY`** | Sesuai keputusan arsitektur di [09-Revisions-Log.md](09-Revisions-Log.md), tabel audit bersifat append-only dengan volume masif di mana urutan numerik mutlak bermakna. |
| 4 | **REV-SET-04: Rekonsiliasi Deskripsi `attendance.max_distance_meter`** | Deskripsi awal langsung menempatkannya sebagai batas atas dan nilai default geofence, mencegah perlunya update ulang di Fase 4. |
| 5 | **Format Payload `ResetPasswordRequest`** | Endpoint `POST /users/{id}/reset-password` menerima field optional `password`. Jika dikosongkan, sistem secara otomatis men-generate password sementara acak 12 karakter (`crypto/rand`) tanpa karakter ambigu. |
| 6 | **Pengujian Paket Terisolasi (`go test -p 1 ./...`)** | Karena paket-paket terhubung ke database Postgres uji bersama, pengujian dieksekusi secara serial antarpaket (`-p 1`) untuk mencegah race condition mutasi data uji antar-goroutine runner Go. |

---

## 3. Bukti Verifikasi

Semua pengujian dan verifikasi dijalankan langsung pada environment pengujian aktif (PostgreSQL 17 di Docker container `faceclock-postgres-1` port 5434, Go 1.26.0 toolchain).

### 3.1 Migrasi Database Bersih (000001–000010)
Seluruh 10 migrasi berhasil diterapkan dari awal:
```
000001_enable_extensions.up.sql
000002_create_employees.up.sql
000003_create_users.up.sql
000004_create_roles.up.sql
000005_create_permissions.up.sql
000006_create_role_permissions.up.sql
000007_create_user_roles.up.sql
000008_create_refresh_tokens.up.sql
000009_create_audit_logs.up.sql
000010_create_app_settings.up.sql
```

### 3.2 Uji Idempotensi Seeder (3× Berturut-turut)
Eksekusi berulang `cmd/seed` menghasilkan kondisi identik tanpa error, tanpa duplikasi:
```
{"time":"2026-09-10T15:23:32.5469079+07:00","level":"INFO","msg":"Starting seeder execution..."}
{"time":"2026-09-10T15:23:32.774569+07:00","level":"INFO","msg":"Permissions seeded successfully","count":34}
{"time":"2026-09-10T15:23:32.8186+07:00","level":"INFO","msg":"System roles seeded successfully","count":3}
{"time":"2026-09-10T15:23:32.918205+07:00","level":"INFO","msg":"Role permissions synchronized successfully"}
{"time":"2026-09-10T15:23:32.9391197+07:00","level":"INFO","msg":"App settings seeded (insert-only) successfully","count":11}
{"time":"2026-09-10T15:23:32.9443181+07:00","level":"INFO","msg":"Existing users found; skipping initial super admin creation","user_count":208}
Seeder completed successfully.
```

### 3.3 Hasil Pengujian Suite Unit Test & Integration Test
Hasil eksekusi `go test -p 1 -count=1 ./...` di folder `apps/faceclock-api`:

```
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/auth         54.863s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/config       3.338s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/employee     1.912s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx        3.658s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx/middleware 3.274s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac         2.561s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/role         1.411s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/settings     0.987s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/storage      2.223s
ok      github.com/faceclock/faceclock/apps/faceclock-api/internal/user         2.853s
ok      github.com/faceclock/faceclock/apps/faceclock-api/test/integration      13.217s
```
**Status: 100% PASS di semua paket.**

### 3.4 Metrik Coverage Unit Test
Target statement coverage yang ditetapkan pada Definition of Done:
- `internal/auth` (Target ≥ 80%): **81.9%** (MEMENUHI SYARAT)
- `internal/rbac` (Target ≥ 80%): **89.9%** (MEMENUHI SYARAT)
- `internal/settings` (Target ≥ 70%): **73.1%** (MEMENUHI SYARAT)
- `internal/employee` (Target ≥ 70%): **75.0%** (MEMENUHI SYARAT)
- `internal/user` (Target ≥ 70%): **74.1%** (MEMENUHI SYARAT)
- `internal/role` (Target ≥ 70%): **75.6%** (MEMENUHI SYARAT)

### 3.5 Matriks RBAC Integration Test (124/124 Lolos)
Pengujian matriks pada `test/integration/rbac_matrix_test.go` meliputi seluruh 31 endpoint diuji terhadap 4 principal:
- **Anonymous**: 31 endpoint diuji (hanya endpoint login & refresh yang lolos 200/400, sisanya 401 UNAUTHENTICATED) → **31/31 PASS**
- **Employee**: 31 endpoint diuji (hanya endpoint self, auth dasar, read settings public, dan get employee/me yang lolos; endpoint admin ditolak 403 FORBIDDEN atau 404 pada ownership check) → **31/31 PASS**
- **Admin**: 31 endpoint diuji (berhasil mengelola user, employee, audit, settings, dan role custom; ditolak saat eskalasi ke super admin) → **31/31 PASS**
- **SuperAdmin**: 31 endpoint diuji (memiliki akses penuh ke seluruh katalog endpoint) → **31/31 PASS**

### 3.6 Zero Secret Leakage Verification
Assertion `AssertNoSecretLeak` memeriksa setiap response payload dari seluruh 124 pengujian integrasi. Tidak ada satu pun kemunculan hash password, hash token, string Argon2id (`$argon2id$`), atau format enkripsi internal.

### 3.7 Verifikasi Khusus HttpOnly Cookie Authentication & Anti-CSRF Shield
Suite pengujian khusus `test/integration/cookie_auth_test.go` dan `internal/httpx/middleware/csrf_test.go` memvalidasi seluruh ketentuan keamanan browser:
- **Set-Cookie Atribut**: Terbukti menerbitkan cookie `access_token` (`Path=/; Max-Age=900; HttpOnly; SameSite=Lax`) dan `refresh_token` (`Path=/api/v1/auth; Max-Age=2592000; HttpOnly; SameSite=Lax`).
- **Autentikasi Murni Cookie**: Request ke `/api/v1/auth/me` tanpa header `Authorization: Bearer` berhasil 200 OK saat cookie `access_token` disertakan.
- **Anti-CSRF Rejection**: Mutating request (`POST`, `PUT`, `PATCH`, `DELETE`) tanpa header `X-Requested-With` atau `X-CSRF-Token` ditolak dengan HTTP 403 `CSRF_HEADER_MISSING`.
- **Untrusted Origin Rejection**: Mutating request dengan `Origin` asing/tidak terdaftar ditolak dengan HTTP 403 `CSRF_UNTRUSTED_ORIGIN`.
- **Safe Methods Exemption**: Request `GET`, `HEAD`, `OPTIONS`, `TRACE` bebas diakses tanpa header CSRF.
- **Rotasi Cookie Refresh**: Endpoint `/api/v1/auth/refresh` membaca cookie `refresh_token` tanpa payload body dan otomatis menerbitkan pasangan cookie baru via `Set-Cookie`.
- **Pembersihan Cookie Logout**: Endpoint `/api/v1/auth/logout` dan `/api/v1/auth/logout-all` mengembalikan cookie ber-`MaxAge=-1` untuk menghapus sesi pada browser client.
- **Browser Lifecycle Simulation**: Alur penuh browser client berbasis `net/http/cookiejar` berhasil login, membaca profil, menyegarkan sesi, dan logout secara transparan.

---

## 4. State Database Saat Ini

Kondisi skema di PostgreSQL (`faceclock`):
- `employees`: 1 tabel, index partial unique pada `employee_number`
- `users`: 1 tabel, partial unique pada `email` dan `employee_id`
- `roles`: 1 tabel (3 role sistem: `super_admin`, `admin`, `employee`)
- `permissions`: 1 tabel (34 permissions terdaftar)
- `role_permissions`: 1 tabel junction relasi role ke permission
- `user_roles`: 1 tabel junction relasi user ke role
- `refresh_tokens`: 1 tabel sesi token dengan tracking family & reuse detection
- `audit_logs`: 1 tabel jejak audit append-only
- `app_settings`: 1 tabel konfigurasi (11 keys terdaftar)

---

## 5. Checklist Definition of Done (DoD) Fase 1

| # | Item Definition of Done | Status | Bukti / Catatan |
|---|---|:---:|---|
| 1 | Sembilan migrasi (`000002`–`000010`) berjalan bersih dari DB kosong dan `.down.sql` membalikkan sempurna | **SELESAI** | Migrasi 000002–000010 teruji up dan down |
| 2 | `cmd/seed` idempoten (dijalankan 3× berturut-turut menghasilkan state identik tanpa error/duplikat) | **SELESAI** | Terverifikasi di § 3.2 (3× eksekusi bersih) |
| 3 | Tiga role sistem ada dengan peta permissions sesuai spesifikasi; `super_admin` memiliki semua permission | **SELESAI** | 34/34 permission dipetakan ke `super_admin` |
| 4 | Login dengan akun super admin awal berhasil dan mengembalikan `must_change_password: true` | **SELESAI** | Terverifikasi di `auth_test.go` & `rbac_matrix_test.go` |
| 5 | Alur token lengkap terbukti: login → access token → refresh token rotasi → logout | **SELESAI** | Terverifikasi di unit test `auth/service_test.go` |
| 6 | Reuse detection terbukti: memakai refresh token lama membalas 401 dan mencabut seluruh token family | **SELESAI** | Terverifikasi di `refresh_test.go` & `auth_test.go` |
| 7 | **Matriks RBAC lulus penuh**: 4 principal × 31 endpoint (124 kombinasi) sesuai spesifikasi | **SELESAI** | 124/124 skenario lulus di `rbac_matrix_test.go` |
| 8 | `TestAllRoutesHaveGuards` lulus: tidak ada route `/api/v1/*` tanpa guard eksplisit | **SELESAI** | Terverifikasi di `internal/httpx/routes_test.go` |
| 9 | Privilege escalation ditolak: admin tidak dapat memberikan role `super_admin` ke siapa pun | **SELESAI** | Teruji via `escalation_test.go` & `user_test.go` |
| 10 | Super admin terakhir tidak dapat dinonaktifkan / dihapus (409 `LAST_SUPER_ADMIN`) | **SELESAI** | Teruji via `guardrails_test.go` & `user_test.go` |
| 11 | Karyawan yang mengakses `/employees/{id}` milik orang lain menerima **404 NOT_FOUND** | **SELESAI** | Teruji via `employee/service_test.go` & endpoint #10 |
| 12 | Tidak ada response API yang memuat `password_hash` atau `token_hash` | **SELESAI** | Terverifikasi via `AssertNoSecretLeak` di seluruh integration test |
| 13 | Tidak ada log yang memuat password, token, atau header `Authorization` | **SELESAI** | Logger middleware redaction test Fase 0 tetap lulus |
| 14 | `GET /settings` sebagai employee hanya mengembalikan baris `is_public = true` | **SELESAI** | Teruji via `settings/service_test.go` & endpoint #28 |
| 15 | Semua aksi penting tercatat di `audit_logs` dengan `actor_user_id` yang benar | **SELESAI** | Teruji di seluruh handler unit tests & audit query |
| 16 | Coverage unit test modul `auth` dan `rbac` ≥ 80%, modul lain ≥ 70% | **SELESAI** | Auth: 81.9%, RBAC: 89.9%, Settings: 73.1%, Employee: 75.0%, User: 74.1%, Role: 75.6% |
| 17 | `go test` dan kompilasi lulus tanpa warning/error | **SELESAI** | Seluruh test suite ok (11 paket lolos) |
| 18 | `docs/api/fase1-auth-rbac.md` lengkap dan cocok dengan implementasi | **SELESAI** | Tersedia di [`docs/api/fase1-auth-rbac.md`](../docs/api/fase1-auth-rbac.md) |
| 19 | **REV-AUTH-02 (HttpOnly Cookies)** terbukti: cookie `access_token` (Path=/, MaxAge=900) & `refresh_token` (Path=/api/v1/auth, MaxAge=30d) terbit otomatis pada login/refresh dan dihapus pada logout | **SELESAI** | Terverifikasi di `cookie_auth_test.go` |
| 20 | **REV-AUTH-02 (Anti-CSRF Shield)** terbukti: mutating methods menolak request tanpa header AJAX/CSRF (403 `CSRF_HEADER_MISSING`) dan origin asing (403 `CSRF_UNTRUSTED_ORIGIN`) | **SELESAI** | Terverifikasi di `csrf_test.go` & `cookie_auth_test.go` |

---

## 6. Artefak yang Diserahkan ke Fase Berikutnya

### Untuk Fase 2 (Face Engine & Spike)
- Tabel `app_settings` telah memuat key:
  - `face.similarity_threshold` (default: 0.60, non-public)
  - `face.model_version` (default: "buffalo_l", public)
- Slot nomor migrasi berikutnya yang tersedia: **`000011`**.
- Format tabel `app_settings` siap menerima pengaturan tambahan ambang batas kualitas wajah Fase 2.

### Untuk Fase 3 (Face Biometrics & Consent)
- Tabel `employees` siap digunakan sebagai relasi foreign key untuk data biometrik (`biometric_consents`, `face_references`).
- Struct `Principal` di `internal/auth` sudah memuat field `EmployeeID *uuid.UUID` yang di-inject otomatis ke context request.
- Permissions `face.*` (`face.enroll`, `face.enroll_self`, `face.verify`, `face.delete`) sudah terdaftar di database dan konstanta `internal/rbac/permissions.go`.
- Pola ownership resource (`RequireAnyPermission`) telah terbukti di `/employees/{id}` dan siap direplikasi untuk `/employees/{id}/face` dan `/employees/{id}/consent`.

---

## 7. Catatan Penting untuk Agent Berikutnya
1. **Pencegahan Data Race DB pada Test**: Jika menjalankan seluruh test suite workspace, selalu sertakan parameter serial antarpaket `go test -p 1 ./...` karena database PostgreSQL uji dipakai bersama antarmodul.
2. **Katalog Error Code**: Penambahan kode error baru di fase-fase berikutnya **wajib** didaftarkan di `internal/httpx/errors.go` lengkap dengan pemetaan HTTP status code-nya. Pemanggilan kode yang belum terdaftar akan memicu **panic eksplisit** (kebijakan *fail-fast* arsitektur).
3. **Pemberian Permissions pada Role**: Jangan pernah mengubah role sistem `is_system = true` secara langsung di database produksi tanpa melalui seeder atau migrasi terkontrol.
