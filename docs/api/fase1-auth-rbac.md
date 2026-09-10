# FaceClock API Specification — Fase 1: Autentikasi, RBAC, User, Employee, Settings & Audit

Dokumen ini mendefinisikan seluruh kontrak 31 endpoint HTTP API Fase 1 untuk backend FaceClock (`apps/faceclock-api`).
Sesuai konvensi standar arsitektur sistem (`docs/adr/0003-api-conventions.md`), semua endpoint:
- Berakar di prefix `/api/v1`
- Menggunakan JSON envelope standar: `{ "data": ... }` untuk sukses, dan `{ "error": { "code": "...", "message": "...", "details": [...] } }` untuk gagal
- Mengirimkan header korelasi `X-Request-ID` di setiap response
- Menegakkan otorisasi berbasis Role-Based Access Control (RBAC) dengan prinsip **default-deny**
- Menjamin **zero secret leakage**: field hash seperti `password_hash`, `token_hash`, dan string internal tidak pernah diekspos di response mana pun.

---

## Arsitektur Autentikasi: HttpOnly Cookie-Based + Anti-CSRF Shield

Sistem autentikasi FaceClock menerapkan standar keamanan browser perusahaan untuk mencegah pencurian token melalui serangan Cross-Site Scripting (XSS) serta memblokir Cross-Site Request Forgery (CSRF):

### 1. Kebijakan Cookie
- **`access_token`**:
  - `Path`: `/`
  - `HttpOnly`: `true` (tidak dapat diakses oleh skrip JavaScript)
  - `SameSite`: `Lax` (cookie dikirim pada navigasi top-level yang aman)
  - `Secure`: `true` di lingkungan non-development (`isDev = false`), `false` pada development lokal
  - `Max-Age`: `900` detik (15 menit)
- **`refresh_token`**:
  - `Path`: `/api/v1/auth` (terisolasi hanya pada endpoint autentikasi)
  - `HttpOnly`: `true`
  - `SameSite`: `Lax`
  - `Secure`: `true` di lingkungan non-development, `false` pada development lokal
  - `Max-Age`: `2592000` detik (30 hari)

### 2. Dual-Mode Compatibility (Browser & Non-Browser Client)
- **Ekstraksi Token**: Middleware `Authenticate` memprioritaskan pembacaan cookie `access_token`. Jika cookie tidak ditemukan, middleware secara mulus beralih (*fallback*) ke header `Authorization: Bearer <token>`.
- **Response Payload**: Endpoint login dan refresh tetap menyertakan token di JSON envelope untuk menjamin kompatibilitas penuh dengan client non-browser (mobile app, daemon/CLI, automated test harness).

### 3. Anti-CSRF Shield
Semua request dengan metode mutasi status (*mutating methods*: `POST`, `PUT`, `PATCH`, `DELETE`) dilindungi oleh middleware anti-CSRF global:
- **Custom Header Requirement**: Setiap request mutasi wajib menyertakan salah satu dari:
  - `X-Requested-With: XMLHttpRequest` (standar AJAX browser yang tidak bisa dikirim otomatis oleh cross-site form submission)
  - `X-CSRF-Token: <token-non-empty>`
- **Verifikasi Origin**: Jika header `Origin` dikirim oleh browser, nilainya harus cocok dengan daftar origin terpercaya yang dikonfigurasi (`allowedOrigins` atau `*` pada mode dev).
- **Error Response**:
  - Permintaan mutasi tanpa custom header yang valid ditolak dengan HTTP 403 `CSRF_HEADER_MISSING`.
  - Permintaan dari origin yang tidak dikenal/tidak terdaftar ditolak dengan HTTP 403 `CSRF_UNTRUSTED_ORIGIN`.
- **Safe Methods**: Metode `GET`, `HEAD`, `OPTIONS`, dan `TRACE` dikecualikan dari pemeriksaan CSRF.

---

## Daftar Isi
1. [Autentikasi & Sesi (#1–#6)](#1-autentikasi--sesi)
2. [Manajemen Karyawan (#7–#12)](#2-manajemen-karyawan)
3. [Manajemen Akun User (#13–#20)](#3-manajemen-akun-user)
4. [Role & Permission (#21–#27)](#4-role--permission)
5. [Pengaturan Aplikasi (#28–#30)](#5-pengaturan-aplikasi)
6. [Audit Trail & Logging (#31)](#6-audit-trail--logging)
7. [Katalog Kode Error Fase 1](#7-katalog-kode-error-fase-1)

---

## 1. Autentikasi & Sesi

### 1.1 POST `/api/v1/auth/login`
- **Guard**: Publik (tanpa autentikasi, rate-limited per IP 10 req/menit)
- **CSRF**: Wajib menyertakan header `X-Requested-With: XMLHttpRequest` atau `X-CSRF-Token`
- **Deskripsi**: Autentikasi email dan password menggunakan Argon2id. Memiliki proteksi timing attack via constant-time dummy verification saat email tidak terdaftar, serta lockout otomatis setelah 5 kali gagal berturut-turut.
- **Request Body**:
```json
{
  "email": "user@faceclock.local",
  "password": "SecurePassword123!"
}
```
- **Response Headers**:
  - `Set-Cookie`: `access_token=<jwt>; Path=/; Max-Age=900; HttpOnly; SameSite=Lax` (ditambah `Secure` di production)
  - `Set-Cookie`: `refresh_token=<token>; Path=/api/v1/auth; Max-Age=2592000; HttpOnly; SameSite=Lax` (ditambah `Secure` di production)
- **Response 200 OK**:
```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "c7a8b9d0-1234-4567-89ab-cdef01234567",
      "employee_id": "e8a9b0c1-2345-6789-01bc-defa23456789",
      "email": "user@faceclock.local",
      "is_active": true,
      "must_change_password": false,
      "roles": ["employee"],
      "permissions": ["attendance.checkin", "employee.read_self"]
    }
  }
}
```
- **Error Responses**:
  - `401 INVALID_CREDENTIALS`: Kredensial tidak valid, akun tidak aktif, atau akun sedang terkunci.
  - `403 CSRF_HEADER_MISSING`: Header anti-CSRF tidak disertakan pada request mutasi.
  - `403 CSRF_UNTRUSTED_ORIGIN`: Header `Origin` tidak sesuai dengan daftar origin terpercaya.
  - `422 VALIDATION_ERROR`: Format email/password tidak sesuai schema.
  - `429 RATE_LIMITED`: Melebihi batas percobaan login per menit.
- **Audit Log**: Menulis `auth.login.success` atau `auth.login.failed`.

---

### 1.2 POST `/api/v1/auth/refresh`
- **Guard**: Publik
- **CSRF**: Wajib menyertakan header `X-Requested-With: XMLHttpRequest` atau `X-CSRF-Token`
- **Deskripsi**: Melakukan rotasi refresh token rantai tunggal. Menggunakan `SELECT ... FOR UPDATE` untuk mencegah race condition. Jika refresh token yang sudah pernah digunakan dikirim ulang di luar batas toleransi grace period, sistem mendeteksi reuse dan otomatis mencabut seluruh sesi satu keluarga token (`family_id`).
- **Ekstraksi Token**: Membaca cookie `refresh_token` secara otomatis dari browser. Jika cookie tidak ada, menerima body JSON fallback.
- **Request Body (Opsional bila cookie dikirim)**:
```json
{
  "refresh_token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
}
```
- **Response Headers**:
  - `Set-Cookie`: `access_token=<jwt_baru>; Path=/; Max-Age=900; HttpOnly; SameSite=Lax`
  - `Set-Cookie`: `refresh_token=<token_baru>; Path=/api/v1/auth; Max-Age=2592000; HttpOnly; SameSite=Lax`
- **Response 200 OK**:
```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "z9y8x7w6v5u4t3s2r1q0p9o8n7m6l5k4",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```
- **Error Responses**:
  - `401 INVALID_REFRESH_TOKEN`: Refresh token tidak ditemukan, kadaluwarsa, atau sudah direvoke.
  - `401 REFRESH_TOKEN_REUSED`: Refresh token yang telah terpakai dikirim ulang (indikasi pencurian token). Seluruh token keluarga direvoke dan `token_version` dinaikkan.
  - `403 CSRF_HEADER_MISSING` / `403 CSRF_UNTRUSTED_ORIGIN`: Pelanggaran kebijakan anti-CSRF.
- **Audit Log**: Menulis `auth.token.refreshed` atau `auth.refresh.reuse_detected`.

---

### 1.3 POST `/api/v1/auth/logout`
- **Guard**: `Authenticated` (via cookie `access_token` atau header `Authorization: Bearer`)
- **CSRF**: Wajib menyertakan header `X-Requested-With: XMLHttpRequest` atau `X-CSRF-Token`
- **Deskripsi**: Melakukan pencabutan refresh token sesi yang sedang aktif dan menghapus cookie autentikasi di browser client. Idempoten.
- **Request Body (Opsional bila cookie dikirim)**:
```json
{
  "refresh_token": "z9y8x7w6v5u4t3s2r1q0p9o8n7m6l5k4"
}
```
- **Response Headers**:
  - `Set-Cookie`: `access_token=; Path=/; Max-Age=-1; HttpOnly`
  - `Set-Cookie`: `refresh_token=; Path=/api/v1/auth; Max-Age=-1; HttpOnly`
- **Response 204 No Content**
- **Audit Log**: Menulis `auth.logout`.

---

### 1.4 POST `/api/v1/auth/logout-all`
- **Guard**: `Authenticated` (via cookie `access_token` atau header `Authorization: Bearer`)
- **CSRF**: Wajib menyertakan header `X-Requested-With: XMLHttpRequest` atau `X-CSRF-Token`
- **Deskripsi**: Mencabut seluruh sesi login user di seluruh perangkat dengan menginkrementasi `token_version` pada tabel `users` dan membatalkan semua refresh token milik user tersebut, serta menghapus cookie sesi di browser pemohon.
- **Request Body**: Kosong `{}`
- **Response Headers**:
  - `Set-Cookie`: `access_token=; Path=/; Max-Age=-1; HttpOnly`
  - `Set-Cookie`: `refresh_token=; Path=/api/v1/auth; Max-Age=-1; HttpOnly`
- **Response 204 No Content**
- **Audit Log**: Menulis `auth.logout_all`.

---

### 1.5 GET `/api/v1/auth/me`
- **Guard**: `Authenticated`
- **Deskripsi**: Mengambil data profil user yang sedang login beserta role dan permissions aktif (dilayani dari permission cache layer).
- **Response 200 OK**:
```json
{
  "data": {
    "id": "c7a8b9d0-1234-4567-89ab-cdef01234567",
    "employee_id": "e8a9b0c1-2345-6789-01bc-defa23456789",
    "email": "user@faceclock.local",
    "is_active": true,
    "must_change_password": false,
    "roles": ["employee"],
    "permissions": ["attendance.checkin", "employee.read_self"],
    "created_at": "2026-09-10T12:00:00Z"
  }
}
```
- **Error Responses**: `401 UNAUTHENTICATED`.

---

### 1.6 POST `/api/v1/auth/change-password`
- **Guard**: `Authenticated`
- **Deskripsi**: Mengganti password user saat ini. Mengharuskan verifikasi password lama, memvalidasi kompleksitas password baru terhadap blacklist kata sandi umum, mengupdate `token_version += 1`, dan membatalkan seluruh refresh token aktif.
- **Request Body**:
```json
{
  "old_password": "OldPassword123!",
  "new_password": "NewSecurePassword456!"
}
```
- **Response 200 OK**:
```json
{
  "data": {
    "message": "Password updated successfully. All other active sessions have been revoked."
  }
}
```
- **Error Responses**:
  - `401 INVALID_CREDENTIALS`: Password lama tidak cocok.
  - `422 VALIDATION_ERROR`: Password baru kurang dari 10 karakter, tidak memiliki kombinasi huruf & angka, sama dengan email, atau terdaftar dalam daftar password lemah (`weak_passwords.txt`).
- **Audit Log**: Menulis `auth.password.changed`.

---

## 2. Manajemen Karyawan

### 2.1 GET `/api/v1/employees`
- **Guard**: `employee.read`
- **Query Params**:
  - `page` (int, default 1)
  - `per_page` (int, default 20, max 100)
  - `status` (string, `active` | `inactive` | `resigned`)
  - `department` (string)
  - `q` (string, pencarian nama atau nomor induk karyawan)
  - `sort` (string, format `full_name:asc`, `employee_number:desc`, dll.)
- **Response 200 OK**:
```json
{
  "data": [
    {
      "id": "e8a9b0c1-2345-6789-01bc-defa23456789",
      "employee_number": "EMP-001",
      "full_name": "Budi Santoso",
      "department": "Engineering",
      "position": "Staff",
      "phone": "+6281234567890",
      "join_date": "2026-01-15",
      "employment_status": "active",
      "face_enrolled_at": null,
      "created_at": "2026-09-10T12:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

---

### 2.2 POST `/api/v1/employees`
- **Guard**: `employee.create`
- **Request Body**:
```json
{
  "employee_number": "EMP-002",
  "full_name": "Siti Aminah",
  "department": "Human Resources",
  "position": "Specialist",
  "phone": "+6281987654321",
  "join_date": "2026-09-01",
  "employment_status": "active"
}
```
- **Response 201 Created**: Mengembalikan objek karyawan yang baru dibuat.
- **Error Responses**:
  - `409 EMPLOYEE_NUMBER_TAKEN`: Nomor induk karyawan sudah terdaftar pada karyawan aktif.
  - `422 VALIDATION_ERROR`: Format data tidak valid (nama < 2 karakter, format tanggal salah, dll.).
- **Audit Log**: Menulis `employee.create`.

---

### 2.3 GET `/api/v1/employees/me`
- **Guard**: `employee.read_self`
- **Deskripsi**: Mengambil data profil karyawan dari user yang sedang login (berdasarkan klaim `employee_id` di token).
- **Response 200 OK**: Objek profil karyawan.
- **Error Responses**:
  - `404 NOT_FOUND`: Akun user tidak terhubung dengan profil karyawan mana pun.

---

### 2.4 GET `/api/v1/employees/{id}`
- **Guard**: `employee.read` ATAU `employee.read_self` (Ownership Pattern)
- **Deskripsi**: Pemegang `employee.read` dapat membaca karyawan mana saja. Pemegang `employee.read_self` hanya dapat membaca profilnya sendiri; jika mencoba mengakses ID karyawan lain, sistem mengembalikan **404 NOT_FOUND** (bukan 403) untuk mencegah konfirmasi keberadaan ID (user enumeration protection).
- **Response 200 OK**: Objek profil karyawan.
- **Error Responses**: `404 NOT_FOUND`.

---

### 2.5 PATCH `/api/v1/employees/{id}`
- **Guard**: `employee.update`
- **Request Body**: Sebagian field yang ingin diubah.
```json
{
  "full_name": "Siti Aminah, S.Psi",
  "department": "People Operations",
  "employment_status": "active"
}
```
- **Response 200 OK**: Objek karyawan terupdate.
- **Error Responses**:
  - `404 NOT_FOUND`: Karyawan tidak ditemukan.
  - `409 EMPLOYEE_NUMBER_TAKEN`: Jika nomor induk diubah ke nomor yang sudah aktif digunakan karyawan lain.
- **Audit Log**: Menulis `employee.update`.

---

### 2.6 DELETE `/api/v1/employees/{id}`
- **Guard**: `employee.delete`
- **Deskripsi**: Melakukan soft-delete pada karyawan (`deleted_at = now()`). Gagal jika karyawan masih terhubung dengan user aktif.
- **Response 204 No Content**
- **Error Responses**:
  - `404 NOT_FOUND`: Karyawan tidak ditemukan.
  - `409 EMPLOYEE_HAS_ACTIVE_USER`: Karyawan memiliki akun pengguna yang masih berstatus aktif.
- **Audit Log**: Menulis `employee.delete`.

---

## 3. Manajemen Akun User

### 3.1 GET `/api/v1/users`
- **Guard**: `user.read`
- **Query Params**: `page`, `per_page`, `role`, `is_active`, `q` (email search).
- **Response 200 OK**:
```json
{
  "data": [
    {
      "id": "c7a8b9d0-1234-4567-89ab-cdef01234567",
      "employee_id": "e8a9b0c1-2345-6789-01bc-defa23456789",
      "email": "staff@faceclock.local",
      "is_active": true,
      "must_change_password": false,
      "roles": ["employee"],
      "last_login_at": "2026-09-10T14:30:00Z",
      "created_at": "2026-09-10T12:00:00Z"
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 1, "total_pages": 1 }
}
```

---

### 3.2 POST `/api/v1/users`
- **Guard**: `user.create`
- **Deskripsi**: Membuat user baru. Jika password tidak disediakan, sistem akan otomatis men-generate password sementara yang aman dari `crypto/rand` dan menandai `must_change_password = true`. Melindungi privilege escalation: pelaku tidak dapat menetapkan role yang memiliki permissions melampaui permission milik pelaku.
- **Request Body**:
```json
{
  "email": "newuser@faceclock.local",
  "employee_id": "e8a9b0c1-2345-6789-01bc-defa23456789",
  "role_ids": ["f1a2b3c4-d5e6-7890-abcd-ef1234567890"]
}
```
- **Response 201 Created**:
```json
{
  "data": {
    "user": {
      "id": "d0e1f2a3-b4c5-6789-01de-fa2345678901",
      "email": "newuser@faceclock.local",
      "is_active": true,
      "must_change_password": true,
      "roles": ["employee"]
    },
    "temporary_password": "k9X-2vM-qL4p"
  }
}
```
- **Error Responses**:
  - `403 ROLE_ESCALATION_DENIED`: Mencoba memberikan role dengan izin di luar kepemilikan pembuat.
  - `409 EMAIL_TAKEN`: Email sudah terdaftar.
  - `422 VALIDATION_ERROR`: Format email/role_ids tidak valid.
- **Audit Log**: Menulis `user.create`.

---

### 3.3 GET `/api/v1/users/{id}`
- **Guard**: `user.read`
- **Response 200 OK**: Objek detail user bersangkutan.
- **Error Responses**: `404 NOT_FOUND`.

---

### 3.4 PATCH `/api/v1/users/{id}`
- **Guard**: `user.update`
- **Request Body**: Field yang ingin diubah (`email`, `is_active`, dll.).
- **Response 200 OK**: Objek user terupdate.
- **Error Responses**:
  - `404 NOT_FOUND`
  - `409 EMAIL_TAKEN`
  - `409 LAST_SUPER_ADMIN`: Ditolak jika menonaktifkan super admin aktif terakhir.
- **Audit Log**: Menulis `user.update`.

---

### 3.5 DELETE `/api/v1/users/{id}`
- **Guard**: `user.delete`
- **Deskripsi**: Soft-delete user (`deleted_at = now()`), mencabut seluruh refresh token, dan meningkatkan `token_version`.
- **Response 204 No Content**
- **Error Responses**:
  - `404 NOT_FOUND`
  - `409 LAST_SUPER_ADMIN`: Ditolak jika mencoba menghapus super admin aktif terakhir.
- **Audit Log**: Menulis `user.delete`.

---

### 3.6 PATCH `/api/v1/users/{id}/status`
- **Guard**: `user.update`
- **Request Body**:
```json
{
  "is_active": false
}
```
- **Response 200 OK**: Objek user dengan status terbaru.
- **Error Responses**:
  - `409 LAST_SUPER_ADMIN`: Mencoba menonaktifkan super admin terakhir.
- **Audit Log**: Menulis `user.status.update`.

---

### 3.7 PUT `/api/v1/users/{id}/roles`
- **Guard**: `user.assign_role`
- **Deskripsi**: Mengganti daftar role user. Memeriksa privilege escalation (pemberi harus memiliki superset permission dari role yang diberikan) dan memeriksa guardrail last super admin (tidak boleh mencabut role `super_admin` dari super admin aktif terakhir).
- **Request Body**:
```json
{
  "role_ids": ["f1a2b3c4-d5e6-7890-abcd-ef1234567890"]
}
```
- **Response 200 OK**: Daftar role terbaru user.
- **Error Responses**:
  - `403 ROLE_ESCALATION_DENIED`: Pemberi mencoba memberikan role dengan wewenang yang tidak dimilikinya.
  - `409 LAST_SUPER_ADMIN`: Mencabut role super admin terakhir.
- **Audit Log**: Menulis `user.roles.assign`.

---

### 3.8 POST `/api/v1/users/{id}/reset-password`
- **Guard**: `user.reset_password`
- **Deskripsi**: Reset password oleh administrator. Membuat password sementara baru yang acak, mengaktifkan `must_change_password = true`, menginkrementasi `token_version`, dan membatalkan semua refresh token user.
- **Request Body**:
```json
{
  "password": "OpsionalCustomPassword123!"
}
```
*(Catatan: Jika `password` tidak disediakan, sistem otomatis membuat password sementara acak 12 karakter).*
- **Response 200 OK**:
```json
{
  "data": {
    "temporary_password": "p9W-3xK-mN8q",
    "must_change_password": true
  }
}
```
- **Audit Log**: Menulis `user.password.reset`.

---

## 4. Role & Permission

### 4.1 GET `/api/v1/roles`
- **Guard**: `role.read`
- **Response 200 OK**:
```json
{
  "data": [
    {
      "id": "11111111-2222-3333-4444-555555555555",
      "name": "admin",
      "display_name": "Administrator",
      "description": "Pengelola operasional, HR, dan pengguna",
      "is_system": true,
      "user_count": 5,
      "permission_count": 27,
      "created_at": "2026-09-10T00:00:00Z"
    }
  ]
}
```

---

### 4.2 POST `/api/v1/roles`
- **Guard**: `role.create`
- **Request Body**:
```json
{
  "name": "branch_manager",
  "display_name": "Manajer Cabang",
  "description": "Monitoring absensi unit cabang",
  "permission_ids": ["22222222-3333-4444-5555-666666666666"]
}
```
- **Response 201 Created**: Objek role baru.
- **Error Responses**:
  - `403 ROLE_ESCALATION_DENIED`: Pemberi mencoba menyematkan permission yang tidak dimilikinya ke role baru.
  - `409 ROLE_NAME_TAKEN`: Nama role sudah digunakan.
- **Audit Log**: Menulis `role.create`.

---

### 4.3 GET `/api/v1/roles/{id}`
- **Guard**: `role.read`
- **Response 200 OK**: Objek role lengkap beserta array `permissions`.
- **Error Responses**: `404 NOT_FOUND`.

---

### 4.4 PATCH `/api/v1/roles/{id}`
- **Guard**: `role.update`
- **Deskripsi**: Mengupdate atribut display_name atau description role. Role sistem (`is_system = true`) tidak dapat diubah namanya.
- **Request Body**:
```json
{
  "display_name": "HR Supervisor",
  "description": "Supervisor departemen HR"
}
```
- **Response 200 OK**: Objek role terupdate.
- **Error Responses**:
  - `403 SYSTEM_ROLE_IMMUTABLE`: Mencoba mengubah nama role sistem.
  - `409 ROLE_NAME_TAKEN`: Nama baru sudah digunakan.
- **Audit Log**: Menulis `role.update`.

---

### 4.5 DELETE `/api/v1/roles/{id}`
- **Guard**: `role.delete`
- **Deskripsi**: Menghapus custom role. Role sistem dilarang dihapus (`is_system = true`), dan role yang masih memiliki user aktif tidak boleh dihapus (`ON DELETE RESTRICT`).
- **Response 204 No Content**
- **Error Responses**:
  - `403 SYSTEM_ROLE_IMMUTABLE`: Role sistem tidak dapat dihapus.
  - `409 ROLE_IN_USE`: Role masih ditugaskan kepada 1 atau lebih user.
- **Audit Log**: Menulis `role.delete`.

---

### 4.6 PUT `/api/v1/roles/{id}/permissions`
- **Guard**: `role.assign_permission`
- **Deskripsi**: Menetapkan ulang daftar permissions pada role. Menginvaliasi cache RBAC untuk semua user pemegang role tersebut. Memeriksa privilege escalation (pemberi harus memiliki permission yang dipasangkan) dan memeriksa guardrail last super admin.
- **Request Body**:
```json
{
  "permission_ids": [
    "22222222-3333-4444-5555-666666666666",
    "33333333-4444-5555-6666-777777777777"
  ]
}
```
- **Response 200 OK**: Objek role beserta permissions terbaru.
- **Error Responses**:
  - `403 ROLE_ESCALATION_DENIED`
  - `409 LAST_SUPER_ADMIN`
- **Audit Log**: Menulis `role.permissions.assign`.

---

### 4.7 GET `/api/v1/permissions`
- **Guard**: `permission.read`
- **Query Params**: `group_by=resource` (opsional)
- **Response 200 OK (Flat)**:
```json
{
  "data": [
    {
      "id": "22222222-3333-4444-5555-666666666666",
      "name": "employee.read",
      "resource": "employee",
      "action": "read",
      "description": "Lihat data karyawan"
    }
  ]
}
```
- **Response 200 OK (Grouped by resource)**:
```json
{
  "data": {
    "employee": [
      {
        "id": "22222222-3333-4444-5555-666666666666",
        "name": "employee.read",
        "resource": "employee",
        "action": "read",
        "description": "Lihat data karyawan"
      }
    ]
  }
}
```

---

## 5. Pengaturan Aplikasi

### 5.1 GET `/api/v1/settings`
- **Guard**: `settings.read`
- **Aturan Visibilitas**: User dengan permission `settings.read` **tanpa** `settings.update` hanya akan menerima baris konfigurasi yang bertanda `is_public = true`. Pengaturan privat seperti `face.similarity_threshold` hanya terlihat oleh pemegang izin `settings.update`.
- **Response 200 OK**:
```json
{
  "data": [
    {
      "key": "attendance.max_distance_meter",
      "value": 100,
      "value_type": "number",
      "description": "Radius maksimum presensi (meter)",
      "is_public": true,
      "updated_at": "2026-09-10T12:00:00Z"
    }
  ]
}
```

---

### 5.2 GET `/api/v1/settings/{key}`
- **Guard**: `settings.read`
- **Aturan Visibilitas**: Jika key yang diminta bertanda `is_public = false` dan pemohon tidak memiliki `settings.update`, sistem membalas dengan **404 NOT_FOUND** (bukan 403).
- **Response 200 OK**: Objek konfigurasi per-key.
- **Error Responses**: `404 NOT_FOUND`.

---

### 5.3 PUT `/api/v1/settings/{key}`
- **Guard**: `settings.update`
- **Deskripsi**: Mengubah nilai konfigurasi aplikasi. Memvalidasi tipe data (`number`, `string`, `boolean`, `json`) dan menjalankan validator domain spesifik (contoh: `face.similarity_threshold` wajib angka antara 0.0 sampai 1.0; `attendance.max_distance_meter` wajib integer positif antara 10 sampai 10000 meter).
- **Request Body**:
```json
{
  "value": 150
}
```
- **Response 200 OK**: Objek konfigurasi terupdate.
- **Error Responses**:
  - `404 NOT_FOUND`: Kunci pengaturan tidak dikenal (kunci baru hanya ditambahkan via database migration).
  - `422 VALIDATION_ERROR`: Tipe nilai atau rentang angka tidak valid.
- **Audit Log**: Menulis `settings.update` dengan metadata nilai lama dan baru.

---

## 6. Audit Trail & Logging

### 6.1 GET `/api/v1/audit-logs`
- **Guard**: `audit.read`
- **Deskripsi**: Mengambil log aktivitas sistem yang append-only. Catatan audit tidak menyediakan operasi create/update/delete lewat API.
- **Query Params**:
  - `page` (int, default 1)
  - `per_page` (int, default 20, max 100)
  - `actor_user_id` (UUID)
  - `action` (string, misal: `user.create`, `auth.login.failed`)
  - `resource_type` (string, misal: `user`, `role`, `employee`)
  - `resource_id` (string)
  - `from` (RFC3339 timestamp)
  - `to` (RFC3339 timestamp)
- **Response 200 OK**:
```json
{
  "data": [
    {
      "id": 1042,
      "actor_user_id": "c7a8b9d0-1234-4567-89ab-cdef01234567",
      "action": "user.create",
      "resource_type": "user",
      "resource_id": "d0e1f2a3-b4c5-6789-01de-fa2345678901",
      "metadata": {
        "email": "newuser@faceclock.local",
        "roles": ["employee"]
      },
      "ip": "127.0.0.1",
      "user_agent": "Mozilla/5.0...",
      "request_id": "01J7ABCDEF1234567890",
      "created_at": "2026-09-10T14:30:00Z"
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 1, "total_pages": 1 }
}
```

---

## 7. Katalog Kode Error Fase 1

Semua error response dikembalikan dalam envelope:
```json
{
  "error": {
    "code": "ERROR_CODE_STRING",
    "message": "Pesan deskriptif yang aman untuk ditampilkan",
    "details": []
  }
}
```

Daftar kode error aktif di Fase 1:
- `BAD_REQUEST` (400)
- `INVALID_CREDENTIALS` (401)
- `UNAUTHENTICATED` (401)
- `INVALID_REFRESH_TOKEN` (401)
- `REFRESH_TOKEN_REUSED` (401)
- `FORBIDDEN` (403)
- `CSRF_HEADER_MISSING` (403)
- `CSRF_UNTRUSTED_ORIGIN` (403)
- `ROLE_ESCALATION_DENIED` (403)
- `SYSTEM_ROLE_IMMUTABLE` (403)
- `NOT_FOUND` (404)
- `CONFLICT` (409)
- `EMAIL_TAKEN` (409)
- `EMPLOYEE_NUMBER_TAKEN` (409)
- `ROLE_NAME_TAKEN` (409)
- `ROLE_IN_USE` (409)
- `EMPLOYEE_HAS_ACTIVE_USER` (409)
- `LAST_SUPER_ADMIN` (409)
- `PAYLOAD_TOO_LARGE` (413)
- `VALIDATION_ERROR` (422)
- `RATE_LIMITED` (429)
- `INTERNAL_ERROR` (500)
- `SERVICE_UNAVAILABLE` (503)
