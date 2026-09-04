# Fase 1 — Auth, User & RBAC

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 1**. Ukuran: 🔴 **Besar**.
> Mengikuti Protokol Handoff § 10.2: dokumen ini harus di-review sebelum eksekusi.
>
> **Depends on:** [Fase 0](01-Fase0.md).
> **Blocker:** [Fase 3](#), [Fase 4](#), [Fase 5](#) — tidak ada satupun yang aman
> dikerjakan sebelum fase ini selesai.

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Setiap endpoint yang ditulis di Fase 3–6 akan menyentuh data pribadi atau data
biometrik. Kalau lapisan identitas dan otorisasi belum ada saat endpoint itu
ditulis, otorisasinya akan ditambal belakangan — dan tambalan belakangan selalu
bocor di satu-dua endpoint yang terlewat.

Master plan § 9 menuntut RBAC ditegakkan **di level API berbasis permission**,
bukan sekadar menyembunyikan menu di UI. Fase 1 adalah tempat aturan itu jadi kode
yang tidak bisa dilewati.

### Hasil akhir yang diharapkan
- Seseorang bisa login dengan email + password dan mendapat access token + refresh token.
- Setiap endpoint terproteksi menolak request tanpa permission yang tepat, dengan
  403 yang konsisten.
- Ada 3 role default terisi (`super_admin`, `admin`, `employee`) beserta katalog
  permission yang lengkap dan bisa diperluas fase berikutnya.
- Admin bisa mengelola karyawan dan user, serta memberi/mencabut role.
- Semua aksi sensitif tercatat di `audit_logs`.
- `app_settings` sudah ada dan bisa dibaca/diubah — tempat threshold similarity
  akan tinggal (master plan § 9 "threshold configurable").

### Yang TIDAK dikerjakan di fase ini
- UI apa pun (→ Fase 5).
- Wajah, embedding, enrollment (→ Fase 2/3).
- Absensi (→ Fase 4).
- SSO / OAuth / 2FA. Kalau nanti dibutuhkan, `users` sudah cukup fleksibel untuk
  menambah kolom provider, tapi jangan dikerjakan sekarang.
- Reset password lewat email (butuh SMTP). Yang ada: **admin-initiated reset** yang
  mengembalikan password sementara + memaksa ganti saat login berikutnya.

---

## 2. Scope Detail

### 2.1 Model identitas: `users` vs `employees`

Dua tabel terpisah, bukan satu. Alasannya bukan kerapian:

- Ada karyawan yang **tidak** punya akun login (belum di-onboard, atau memang tidak
  perlu). Kalau digabung, setiap karyawan wajib punya `password_hash`.
- Ada akun yang **tidak** merepresentasikan karyawan — `super_admin` awal, akun
  integrasi. Kalau digabung, akun itu akan muncul di laporan absensi sebagai
  "karyawan" yang tidak pernah absen.
- Fase 3 dan 4 selalu berbicara tentang `employee_id`, bukan `user_id`. Absensi
  milik **karyawan**, bukan milik akun. Kalau akun dihapus/diganti, riwayat
  absensinya tidak boleh ikut hilang.

Relasi: `users.employee_id` → `employees.id`, **nullable**, **unique**
(satu karyawan maksimum satu akun), `ON DELETE RESTRICT`.

### 2.2 Model otorisasi: permission-based, bukan role-based

Guard di endpoint memeriksa **permission**, tidak pernah memeriksa nama role.

```go
// BENAR
r.With(RequirePermission("attendance.approve")).Post("/attendances/{id}/approve", h.Approve)

// SALAH — mengunci kebijakan ke nama role, dan bocor begitu ada role baru
if user.Role == "admin" { ... }
```

Alasan: master plan § 6 menyatakan Super Admin bisa mengelola role & permission.
Artinya himpunan role **berubah saat runtime**. Kode yang memeriksa nama role akan
salah setiap kali admin membuat role baru, dan tidak ada test yang menangkapnya.

Rantai: `user` → (`user_roles`) → `role` → (`role_permissions`) → `permission`.

### 2.3 Katalog permission

Format nama: `resource.action`, huruf kecil, `snake_case` untuk action majemuk.

Sufiks `_self` berarti: pemegangnya hanya boleh mengakses baris miliknya sendiri.
Guard di middleware tidak bisa menegakkan kepemilikan (ia tidak tahu isi resource),
jadi ada **pola ownership** khusus di § 5.4 yang wajib dipakai.

| Permission | Keterangan | Ditambahkan di fase |
|---|---|---|
| `user.read` | Lihat daftar & detail user | 1 |
| `user.create` | Buat user | 1 |
| `user.update` | Ubah user (termasuk aktif/nonaktif) | 1 |
| `user.delete` | Soft-delete user | 1 |
| `user.assign_role` | Set role milik user | 1 |
| `user.reset_password` | Reset password user lain | 1 |
| `employee.read` | Lihat semua karyawan | 1 |
| `employee.read_self` | Lihat data karyawan diri sendiri | 1 |
| `employee.create` | Tambah karyawan | 1 |
| `employee.update` | Ubah karyawan | 1 |
| `employee.delete` | Soft-delete karyawan | 1 |
| `role.read` | Lihat role | 1 |
| `role.create` | Buat role | 1 |
| `role.update` | Ubah role | 1 |
| `role.delete` | Hapus role | 1 |
| `role.assign_permission` | Set permission milik role | 1 |
| `permission.read` | Lihat katalog permission | 1 |
| `settings.read` | Baca app settings | 1 |
| `settings.update` | Ubah app settings | 1 |
| `audit.read` | Baca audit log | 1 |
| `face.enroll_self` | Enroll wajah sendiri | 3 |
| `face.read_self` | Lihat referensi wajah sendiri | 3 |
| `face.enroll_any` | Enroll wajah karyawan lain | 3 |
| `face.read_any` | Lihat referensi wajah siapa pun | 3 |
| `face.delete_any` | Nonaktifkan referensi wajah siapa pun | 3 |
| `attendance.checkin` | Melakukan check-in/out untuk diri sendiri | 4 |
| `attendance.read_self` | Lihat riwayat absensi sendiri | 4 |
| `attendance.read_all` | Lihat absensi semua karyawan | 4 |
| `attendance.approve` | Approve/reject absensi `pending_review` | 4 |
| `attendance.export` | Export rekap | 5 |
| `location.read` / `location.create` / `location.update` / `location.delete` | Kelola `office_locations` | 4 |

**Aturan yang mengikat semua fase berikutnya:** setiap migration yang menambah
baris `permissions` **wajib** di migration yang sama juga meng-`INSERT` baris
`role_permissions` untuk `super_admin`. Tanpa aturan ini, super admin akan
kehilangan akses ke fitur baru sampai ada yang ingat menjalankan seeder ulang.

Permission Fase 3–5 di tabel di atas **sudah di-seed di Fase 1** (barisnya ada di
tabel `permissions` dan sudah dipetakan ke role). Yang belum ada hanyalah endpoint
yang memakainya. Ini disengaja: supaya Fase 3/4 hanya menulis handler, tidak
mendesain ulang otorisasi.

### 2.4 Role default

| Role | `is_system` | Permission |
|---|---|---|
| `super_admin` | ✅ | **Semua** permission yang terdaftar |
| `admin` | ✅ | Semua kecuali: `role.create`, `role.update`, `role.delete`, `role.assign_permission`, `user.delete` |
| `employee` | ✅ | `employee.read_self`, `face.enroll_self`, `face.read_self`, `attendance.checkin`, `attendance.read_self`, `settings.read` |

`is_system = true` berarti role tidak bisa dihapus dan `name`-nya tidak bisa
diubah (permission-nya masih boleh disesuaikan oleh super admin).

Catatan `settings.read` untuk employee: employee memang perlu membaca sebagian
setting (jam kerja, radius geofence untuk ditampilkan di UI). Karena itu
`app_settings` punya kolom `is_public` — pemegang `settings.read` tanpa
`settings.update` hanya menerima baris `is_public = true`. Rinciannya di § 4.6.

### 2.5 Strategi token

#### Access token — JWT, HS256, TTL 15 menit

```json
{
  "iss": "faceclock-api",
  "aud": "faceclock",
  "sub": "018f4c2a-...",          // user_id
  "eid": "018f4c31-...",          // employee_id, null bila akun tanpa karyawan
  "typ": "access",
  "tv":  3,                        // token_version, lihat di bawah
  "jti": "01JD8Z0X9K7Q...",
  "iat": 1757000000,
  "nbf": 1757000000,
  "exp": 1757000900
}
```

**Permission TIDAK dimasukkan ke dalam token.** Ini keputusan sadar:

- Token yang membawa permission jadi basi begitu admin mengubah role. Pencabutan
  akses baru berlaku setelah token kedaluwarsa — jendela 15 menit di mana orang
  yang baru dipecat masih bisa approve absensi.
- Daftar permission super_admin (30+ item) membuat header setiap request
  membengkak.

Sebagai gantinya, middleware memuat permission dari database, dengan **cache
in-process** (`map[uuid]permissionSet`, TTL 30 detik) yang **di-invalidasi
eksplisit** saat `user_roles` atau `role_permissions` berubah. Biayanya satu query
join per user per 30 detik — tidak signifikan, dan pencabutan akses berlaku
maksimum 30 detik.

**`tv` (token_version)** dibandingkan dengan `users.token_version` setiap request
(nilainya ikut ter-cache bersama permission). Menaikkan `token_version` mematikan
**semua** access token milik user itu seketika. Dinaikkan saat: ganti password,
reset password oleh admin, user dinonaktifkan, user di-soft-delete, `logout-all`.

#### Refresh token — opaque, TTL 30 hari, dengan rotasi + deteksi reuse

- 32 byte acak dari `crypto/rand`, di-encode base64url. **Bukan JWT** — refresh
  token harus bisa dicabut satu per satu, dan itu butuh state di database.
- Yang disimpan di DB adalah `sha256(token)`, bukan tokennya. Dump database tidak
  langsung berarti sesi orang bisa dibajak.
- **Rotasi:** setiap `/auth/refresh` menandai token lama `used_at = now()` dan
  menerbitkan token baru dengan `family_id` yang sama dan `parent_id` = id token lama.
- **Deteksi reuse:** kalau token yang sudah `used_at != NULL` dipakai lagi, itu
  indikasi token dicuri (dua pihak memegang token yang sama). Responsnya: revoke
  **seluruh family**, naikkan `token_version`, tulis `audit_logs` dengan action
  `auth.refresh.reuse_detected`, balas 401. User dipaksa login ulang.

#### Password

- **Argon2id** (`golang.org/x/crypto/argon2`), parameter: `memory=64MB`, `time=3`,
  `threads=2`, `saltLen=16`, `keyLen=32`. Disimpan dalam format encoded standar
  `$argon2id$v=19$m=65536,t=3,p=2$<salt-b64>$<hash-b64>` supaya parameter bisa
  dinaikkan nanti tanpa memaksa semua orang reset.
- Alternatif yang dapat diterima bila tim lebih nyaman: `bcrypt` cost 12.
- Verifikasi selalu memakai perbandingan constant-time.
- Kebijakan password: minimum 10 karakter, wajib memuat huruf dan angka, ditolak
  bila ada di daftar 200 password terlemah (list statik di repo, bukan panggilan
  API eksternal).

#### Perlindungan brute force

Dua lapis, keduanya wajib:

1. **Per akun:** `users.failed_login_count`. 5 kegagalan berturut-turut →
   `locked_until = now() + 15 menit`. Login sukses mereset counter.
2. **Per IP:** rate limit 10 request/menit pada `POST /auth/login` dan
   `POST /auth/refresh` (middleware `RateLimit` dari Fase 0, dengan konfigurasi
   khusus route ini).

Lapis per-akun saja bisa dipakai untuk mengunci akun orang lain (denial of
service); lapis per-IP saja tidak menahan botnet. Keduanya ada, dan responsnya
sengaja **tidak** membedakan "akun terkunci" dari "password salah" — lihat E4.

---

## 3. Skema Database

Semua tabel mengikuti konvensi yang dikunci di [Fase 0 § 3](01-Fase0.md#3-skema-database).
Migration Fase 1: `000002` – `000010`.

### 3.1 `employees` — migration `000002`

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `employee_number` | `citext` | `NOT NULL` | NIK/NIP internal; unique lewat partial index |
| `full_name` | `text` | `NOT NULL`, `CHECK (length(btrim(full_name)) BETWEEN 2 AND 120)` | |
| `department` | `text` | `NULL` | Sengaja teks bebas di Fase 1; jadi tabel sendiri bila Fase 5 butuh filter hierarkis |
| `position` | `text` | `NULL` | |
| `phone` | `text` | `NULL`, `CHECK (phone IS NULL OR phone ~ '^[0-9+][0-9 +()-]{6,19}$')` | |
| `email` | `citext` | `NULL` | Email kantor; **berbeda** dari `users.email` (email login) |
| `join_date` | `date` | `NULL` | |
| `employment_status` | `text` | `NOT NULL DEFAULT 'active'`, `CHECK (employment_status IN ('active','inactive','resigned'))` | |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |
| `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |
| `deleted_at` | `timestamptz` | `NULL` | soft delete |

```sql
CREATE UNIQUE INDEX employees_employee_number_uniq
  ON employees (employee_number) WHERE deleted_at IS NULL;
CREATE INDEX employees_department_idx  ON employees (department) WHERE deleted_at IS NULL;
CREATE INDEX employees_full_name_idx   ON employees (lower(full_name)) WHERE deleted_at IS NULL;
CREATE INDEX employees_status_idx      ON employees (employment_status) WHERE deleted_at IS NULL;
```

Unique-nya **partial** (`WHERE deleted_at IS NULL`) supaya nomor karyawan bisa
dipakai ulang setelah karyawan lama dihapus — dan supaya soft-delete tidak gagal
karena bentrok dengan dirinya sendiri.

### 3.2 `users` — migration `000003`

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `employee_id` | `uuid` | `NULL`, FK → `employees(id)` `ON DELETE RESTRICT` | unique lewat partial index |
| `email` | `citext` | `NOT NULL`, `CHECK (email ~ '^[^@\s]+@[^@\s]+\.[^@\s]+$')` | email login; case-insensitive berkat `citext` |
| `password_hash` | `text` | `NOT NULL` | Argon2id encoded string |
| `is_active` | `boolean` | `NOT NULL DEFAULT true` | |
| `must_change_password` | `boolean` | `NOT NULL DEFAULT false` | true setelah admin reset |
| `token_version` | `integer` | `NOT NULL DEFAULT 0` | lihat § 2.5 |
| `failed_login_count` | `smallint` | `NOT NULL DEFAULT 0` | |
| `locked_until` | `timestamptz` | `NULL` | |
| `last_login_at` | `timestamptz` | `NULL` | |
| `password_changed_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |
| `created_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` | |
| `created_at` / `updated_at` / `deleted_at` | `timestamptz` | | |

```sql
CREATE UNIQUE INDEX users_email_uniq       ON users (email)       WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_employee_id_uniq ON users (employee_id) WHERE deleted_at IS NULL AND employee_id IS NOT NULL;
CREATE INDEX        users_is_active_idx    ON users (is_active)   WHERE deleted_at IS NULL;
```

### 3.3 `roles` — migration `000004`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` |
| `name` | `citext` | `NOT NULL`, `CHECK (name ~ '^[a-z][a-z0-9_]{2,49}$')` |
| `display_name` | `text` | `NOT NULL` |
| `description` | `text` | `NULL` |
| `is_system` | `boolean` | `NOT NULL DEFAULT false` |
| `created_at` / `updated_at` / `deleted_at` | `timestamptz` | |

```sql
CREATE UNIQUE INDEX roles_name_uniq ON roles (name) WHERE deleted_at IS NULL;
```

### 3.4 `permissions` — migration `000005`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` |
| `name` | `text` | `NOT NULL UNIQUE`, `CHECK (name ~ '^[a-z_]+\.[a-z_]+$')` |
| `resource` | `text` | `NOT NULL` — bagian sebelum titik |
| `action` | `text` | `NOT NULL` — bagian sesudah titik |
| `description` | `text` | `NOT NULL` |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` |

`permissions` **tidak** punya `deleted_at`: permission adalah katalog yang
ditentukan kode, bukan data yang dikelola user. Menambah/menghapus baris di sini
selalu lewat migration.

```sql
CREATE INDEX permissions_resource_idx ON permissions (resource);
```

### 3.5 `role_permissions` — migration `000006`

| Kolom | Tipe | Constraint |
|---|---|---|
| `role_id` | `uuid` | FK → `roles(id)` `ON DELETE CASCADE` |
| `permission_id` | `uuid` | FK → `permissions(id)` `ON DELETE CASCADE` |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` |

`PRIMARY KEY (role_id, permission_id)` + `CREATE INDEX ON role_permissions (permission_id);`
(index kedua dibutuhkan untuk pertanyaan "role mana saja yang punya permission X").

### 3.6 `user_roles` — migration `000007`

| Kolom | Tipe | Constraint |
|---|---|---|
| `user_id` | `uuid` | FK → `users(id)` `ON DELETE CASCADE` |
| `role_id` | `uuid` | FK → `roles(id)` `ON DELETE RESTRICT` |
| `assigned_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` |
| `assigned_at` | `timestamptz` | `NOT NULL DEFAULT now()` |

`PRIMARY KEY (user_id, role_id)` + `CREATE INDEX ON user_roles (role_id);`

`ON DELETE RESTRICT` pada `role_id` disengaja: menghapus role yang masih dipakai
harus **gagal dengan pesan jelas**, bukan diam-diam mencabut akses sekelompok orang.

### 3.7 `refresh_tokens` — migration `000008`

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `user_id` | `uuid` | `NOT NULL`, FK → `users(id)` `ON DELETE CASCADE` | |
| `token_hash` | `text` | `NOT NULL UNIQUE` | `sha256` hex dari token |
| `family_id` | `uuid` | `NOT NULL` | konstan sepanjang rantai rotasi |
| `parent_id` | `uuid` | `NULL`, FK → `refresh_tokens(id)` `ON DELETE SET NULL` | |
| `expires_at` | `timestamptz` | `NOT NULL` | |
| `used_at` | `timestamptz` | `NULL` | diisi saat dirotasi |
| `revoked_at` | `timestamptz` | `NULL` | |
| `revoked_reason` | `text` | `NULL`, `CHECK (... IN ('logout','logout_all','rotated','reuse_detected','password_changed','user_deactivated','admin_revoked'))` | |
| `user_agent` | `text` | `NULL` | dipotong maks 255 char |
| `ip` | `inet` | `NULL` | |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |

```sql
CREATE INDEX refresh_tokens_user_idx    ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_family_idx  ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_expires_idx ON refresh_tokens (expires_at)
  WHERE revoked_at IS NULL;
```

Housekeeping: job harian menghapus baris dengan
`expires_at < now() - interval '30 days'`. Di Fase 1 cukup goroutine ticker di
`cmd/api`; kalau nanti ada scheduler sungguhan, dipindah ke sana.

### 3.8 `audit_logs` — migration `000009`

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `bigint` | `GENERATED ALWAYS AS IDENTITY`, PK | **Pengecualian sadar** dari konvensi UUID: tabel ini append-only, sangat banyak, dan urutan sisipnya bermakna |
| `actor_user_id` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` | null untuk aksi sistem/seeder |
| `action` | `text` | `NOT NULL` | mis. `user.create`, `auth.login.failed` |
| `resource_type` | `text` | `NULL` | `user`, `role`, `employee`, … |
| `resource_id` | `text` | `NULL` | teks, karena tidak semua PK uuid |
| `metadata` | `jsonb` | `NOT NULL DEFAULT '{}'::jsonb` | **wajib bebas dari data sensitif** — lihat larangan di bawah |
| `ip` | `inet` | `NULL` | |
| `user_agent` | `text` | `NULL` | |
| `request_id` | `text` | `NULL` | korelasi ke log aplikasi |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |

```sql
CREATE INDEX audit_logs_actor_idx    ON audit_logs (actor_user_id, created_at DESC);
CREATE INDEX audit_logs_action_idx   ON audit_logs (action, created_at DESC);
CREATE INDEX audit_logs_resource_idx ON audit_logs (resource_type, resource_id, created_at DESC);
CREATE INDEX audit_logs_created_idx  ON audit_logs (created_at DESC);
```

`metadata` **tidak boleh** memuat: password (plain/hash), token, embedding, byte
gambar. Yang boleh: field mana yang berubah (nama field saja, atau nilai lama/baru
untuk field non-sensitif seperti `department`), daftar role id, alasan.

Aksi yang wajib dicatat di Fase 1:
`auth.login.success`, `auth.login.failed`, `auth.logout`, `auth.logout_all`,
`auth.refresh.reuse_detected`, `auth.password.changed`, `auth.password.reset_by_admin`,
`user.create`, `user.update`, `user.delete`, `user.status.changed`, `user.roles.changed`,
`employee.create`, `employee.update`, `employee.delete`,
`role.create`, `role.update`, `role.delete`, `role.permissions.changed`,
`settings.updated`.

### 3.9 `app_settings` — migration `000010`

| Kolom | Tipe | Constraint |
|---|---|---|
| `key` | `text` | PK, `CHECK (key ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$')` |
| `value` | `jsonb` | `NOT NULL` |
| `value_type` | `text` | `NOT NULL`, `CHECK (value_type IN ('string','number','boolean','json'))` |
| `description` | `text` | `NOT NULL` |
| `is_public` | `boolean` | `NOT NULL DEFAULT false` |
| `updated_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` |
| `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` |

Seed awal (nilai final di-tuning Fase 2/4):

| key | value | type | is_public |
|---|---|---|---|
| `face.similarity_threshold` | `0.45` | number | `false` |
| `face.min_reference_photos` | `3` | number | `true` |
| `face.model_version` | `"unset"` | string | `false` |
| `attendance.geofence_enabled` | `true` | boolean | `true` |
| `attendance.max_distance_meter` | `100` | number | `true` |
| `attendance.workday_start` | `"08:00"` | string | `true` |
| `attendance.workday_end` | `"17:00"` | string | `true` |
| `attendance.timezone` | `"Asia/Jakarta"` | string | `true` |
| `security.password_min_length` | `10` | number | `true` |
| `security.max_failed_login` | `5` | number | `false` |
| `security.lockout_minutes` | `15` | number | `false` |

`face.similarity_threshold` sengaja diberi nilai awal yang konservatif dan ditandai
`"unset"` pada `face.model_version` — Fase 2 yang mengisinya berdasarkan kalibrasi
nyata. Angka 0.45 di sini **bukan** rekomendasi, hanya placeholder yang tidak
akan meloloskan siapa pun secara tidak sengaja bila lupa dikalibrasi.

### 3.10 Diagram relasi

```
employees 1 ──── 0..1 users
                    │ 1
                    │
                    ├──── * user_roles * ──── 1 roles
                    │                            │ 1
                    │                            │
                    ├──── * refresh_tokens       └──── * role_permissions * ──── 1 permissions
                    │
                    └──── * audit_logs (actor)
```

---

## 4. Daftar Endpoint / Kontrak API

Base path: `/api/v1`. Semua response memakai envelope dari
[Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go).
Semua field JSON `snake_case`.

Kolom **Guard**: `—` publik, `auth` cukup terautentikasi, sisanya nama permission.

### 4.1 Auth

| Method | Path | Guard |
|---|---|---|
| POST | `/auth/login` | — |
| POST | `/auth/refresh` | — |
| POST | `/auth/logout` | auth |
| POST | `/auth/logout-all` | auth |
| GET | `/auth/me` | auth |
| POST | `/auth/change-password` | auth |

#### `POST /auth/login`

```json
// request
{ "email": "hr@faceclock.local", "password": "RahasiaKuat123" }
```

```json
// 200
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900,
    "refresh_token": "9pQ2...base64url...",
    "refresh_expires_in": 2592000,
    "must_change_password": false,
    "user": {
      "id": "018f4c2a-...",
      "email": "hr@faceclock.local",
      "employee": {
        "id": "018f4c31-...",
        "employee_number": "EMP-0001",
        "full_name": "Sari Rahmawati",
        "department": "HR"
      },
      "roles": ["admin"],
      "permissions": ["employee.read", "employee.create", "..."]
    }
  }
}
```

`permissions` dikirim di response login **hanya untuk kebutuhan UI** (menyembunyikan
menu). Ia bukan sumber otorisasi — server tetap memeriksa ulang di setiap request.
Ini harus ditulis sebagai komentar di kode supaya tidak ada yang "mengoptimalkan"
dengan mempercayainya.

Error:

| Status | code | Kondisi |
|---|---|---|
| 422 | `VALIDATION_ERROR` | email/password kosong atau format email salah |
| 401 | `INVALID_CREDENTIALS` | email tidak ada, password salah, user nonaktif, **atau akun terkunci** (sengaja tidak dibedakan) |
| 429 | `RATE_LIMITED` | >10 percobaan/menit dari satu IP |

#### `POST /auth/refresh`

```json
// request
{ "refresh_token": "9pQ2..." }
```

```json
// 200 — token lama otomatis tidak berlaku lagi (rotasi)
{
  "data": {
    "access_token": "...",
    "token_type": "Bearer",
    "expires_in": 900,
    "refresh_token": "<token baru>",
    "refresh_expires_in": 2592000
  }
}
```

| Status | code | Kondisi |
|---|---|---|
| 401 | `INVALID_REFRESH_TOKEN` | tidak ditemukan / expired / sudah di-revoke |
| 401 | `REFRESH_TOKEN_REUSED` | token sudah pernah dirotasi → seluruh family di-revoke |
| 401 | `UNAUTHENTICATED` | user sudah nonaktif atau soft-deleted |

#### `POST /auth/logout`
Body `{ "refresh_token": "..." }`. Me-revoke satu token (`revoked_reason='logout'`).
Selalu `204`, bahkan bila tokennya sudah tidak valid — logout tidak boleh gagal.

#### `POST /auth/logout-all`
Revoke semua refresh token milik user + `token_version += 1`. `204`.

#### `GET /auth/me`
Mengembalikan objek `user` yang sama persis dengan yang ada di response login
(termasuk `roles` dan `permissions` terkini). Endpoint ini yang dipanggil web saat
reload halaman.

#### `POST /auth/change-password`

```json
{ "current_password": "...", "new_password": "...", "new_password_confirmation": "..." }
```

Efek: `password_hash` diganti, `password_changed_at = now()`,
`must_change_password = false`, `token_version += 1`, **semua** refresh token
di-revoke (`revoked_reason='password_changed'`). Response `204`; client wajib
login ulang. Ini disengaja — ganti password yang tidak mencabut sesi lain tidak
berguna saat akun memang sedang disusupi.

| Status | code | Kondisi |
|---|---|---|
| 401 | `INVALID_CREDENTIALS` | `current_password` salah |
| 422 | `VALIDATION_ERROR` | konfirmasi tidak cocok, tidak memenuhi kebijakan, atau sama dengan password lama |

### 4.2 Employees

| Method | Path | Guard |
|---|---|---|
| GET | `/employees` | `employee.read` |
| POST | `/employees` | `employee.create` |
| GET | `/employees/me` | `employee.read_self` |
| GET | `/employees/{id}` | `employee.read` **atau** `employee.read_self` + pemilik |
| PATCH | `/employees/{id}` | `employee.update` |
| DELETE | `/employees/{id}` | `employee.delete` |

`GET /employees` — query param: `page` (default 1), `per_page` (default 20, maks 100),
`q` (cari di `full_name` / `employee_number`), `department`, `employment_status`,
`has_user` (`true`/`false`), `sort` (`full_name`, `-created_at`, …).

```json
// 200
{
  "data": [
    {
      "id": "018f4c31-...",
      "employee_number": "EMP-0001",
      "full_name": "Sari Rahmawati",
      "department": "HR",
      "position": "HR Manager",
      "phone": "081234567890",
      "email": "sari@perusahaan.co.id",
      "join_date": "2024-02-01",
      "employment_status": "active",
      "has_user_account": true,
      "created_at": "2026-09-01T02:11:04Z",
      "updated_at": "2026-09-01T02:11:04Z"
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 137, "total_pages": 7 }
}
```

`POST /employees` — body: `employee_number` (wajib), `full_name` (wajib),
`department`, `position`, `phone`, `email`, `join_date`, `employment_status`.
`201` + objek employee.

| Status | code | Kondisi |
|---|---|---|
| 409 | `EMPLOYEE_NUMBER_TAKEN` | `employee_number` sudah dipakai karyawan aktif |
| 422 | `VALIDATION_ERROR` | field tidak valid |

`DELETE /employees/{id}` — soft delete. Ditolak bila karyawan masih punya user
aktif (`409 EMPLOYEE_HAS_ACTIVE_USER`) — nonaktifkan akunnya dulu, supaya tidak
ada akun hidup yang menggantung tanpa identitas karyawan.

> Catatan Fase 3/4: setelah `face_references` dan `attendances` ada, soft-delete
> employee **tidak** menghapus keduanya. Riwayat absensi adalah catatan yang harus
> tetap bisa diaudit; embedding-nya dinonaktifkan (`is_active=false`), tidak dihapus,
> kecuali ada permintaan penghapusan data dari yang bersangkutan (UU PDP —
> prosedurnya didefinisikan di Fase 3).

### 4.3 Users

| Method | Path | Guard |
|---|---|---|
| GET | `/users` | `user.read` |
| POST | `/users` | `user.create` |
| GET | `/users/{id}` | `user.read` |
| PATCH | `/users/{id}` | `user.update` |
| DELETE | `/users/{id}` | `user.delete` |
| PATCH | `/users/{id}/status` | `user.update` |
| PUT | `/users/{id}/roles` | `user.assign_role` |
| POST | `/users/{id}/reset-password` | `user.reset_password` |

`POST /users`

```json
// request
{
  "email": "budi@faceclock.local",
  "password": "PasswordAwal123",
  "employee_id": "018f4c31-...",
  "role_ids": ["018f4c40-..."],
  "must_change_password": true
}
```

```json
// 201
{
  "data": {
    "id": "018f4c50-...",
    "email": "budi@faceclock.local",
    "is_active": true,
    "must_change_password": true,
    "last_login_at": null,
    "employee": { "id": "018f4c31-...", "employee_number": "EMP-0002", "full_name": "Budi Santoso" },
    "roles": [{ "id": "018f4c40-...", "name": "employee", "display_name": "Karyawan" }],
    "created_at": "2026-09-04T03:00:00Z"
  }
}
```

`password_hash` **tidak pernah** muncul di response mana pun. Tulis test yang
memverifikasi ini, bukan sekadar mengandalkan struct response yang benar.

| Status | code | Kondisi |
|---|---|---|
| 409 | `EMAIL_TAKEN` | email sudah dipakai user aktif |
| 409 | `EMPLOYEE_ALREADY_HAS_USER` | `employee_id` sudah punya akun |
| 404 | `NOT_FOUND` | `employee_id` / `role_ids` tidak ada |
| 403 | `ROLE_ESCALATION_DENIED` | pemberi mencoba memberi role yang permission-nya melebihi miliknya (§ 5.5) |
| 422 | `VALIDATION_ERROR` | password tidak memenuhi kebijakan |

`PUT /users/{id}/roles` — **replace**, bukan tambah:

```json
{ "role_ids": ["018f4c40-...", "018f4c41-..."] }
```

Response `200` dengan objek user terbaru. Efek samping: cache permission user itu
di-invalidasi seketika, `audit_logs` action `user.roles.changed` dengan metadata
`{"before": [...role names], "after": [...role names]}`.

`PATCH /users/{id}/status` — `{ "is_active": false }`.
Menonaktifkan berarti: `is_active=false`, `token_version += 1`, semua refresh token
di-revoke (`user_deactivated`). Efeknya langsung, bukan menunggu token expired.

`POST /users/{id}/reset-password`

```json
// 200 — password sementara hanya ditampilkan SEKALI, tidak disimpan plain di mana pun
{ "data": { "temporary_password": "Xk8-2mQ-vT4z", "must_change_password": true } }
```

Efek: `must_change_password=true`, `token_version += 1`, semua refresh token
di-revoke. Password sementara: 12 karakter dari `crypto/rand`, alfabet tanpa
karakter ambigu (`0/O`, `1/l/I`).

### 4.4 Roles

| Method | Path | Guard |
|---|---|---|
| GET | `/roles` | `role.read` |
| POST | `/roles` | `role.create` |
| GET | `/roles/{id}` | `role.read` |
| PATCH | `/roles/{id}` | `role.update` |
| DELETE | `/roles/{id}` | `role.delete` |
| PUT | `/roles/{id}/permissions` | `role.assign_permission` |

`GET /roles`

```json
{
  "data": [
    {
      "id": "018f4c40-...",
      "name": "admin",
      "display_name": "Admin / HR",
      "description": "Kelola karyawan, approve absensi, konfigurasi",
      "is_system": true,
      "user_count": 3,
      "permission_count": 24
    }
  ]
}
```

`GET /roles/{id}` menyertakan `permissions: [{id, name, resource, action, description}]`.

`PUT /roles/{id}/permissions` — replace penuh, body `{ "permission_ids": [...] }`.
Efek: cache permission **semua** user yang punya role ini di-invalidasi.

| Status | code | Kondisi |
|---|---|---|
| 409 | `ROLE_NAME_TAKEN` | nama role sudah ada |
| 409 | `ROLE_IN_USE` | `DELETE` pada role yang masih dipakai ≥1 user |
| 403 | `SYSTEM_ROLE_IMMUTABLE` | mencoba hapus / rename role `is_system` |
| 403 | `ROLE_ESCALATION_DENIED` | memberi permission yang tidak dimiliki pemberi |
| 409 | `LAST_SUPER_ADMIN` | mencabut permission kritikal dari satu-satunya jalur super admin |

### 4.5 Permissions

| Method | Path | Guard |
|---|---|---|
| GET | `/permissions` | `permission.read` |

Read-only. Mendukung `?group_by=resource`:

```json
{
  "data": {
    "employee": [ { "id": "...", "name": "employee.read", "action": "read", "description": "..." } ],
    "attendance": [ ... ]
  }
}
```

### 4.6 App settings

| Method | Path | Guard |
|---|---|---|
| GET | `/settings` | `settings.read` |
| GET | `/settings/{key}` | `settings.read` |
| PUT | `/settings/{key}` | `settings.update` |

Aturan visibilitas: pemegang `settings.read` **tanpa** `settings.update` hanya
menerima baris dengan `is_public = true`. Ini mencegah karyawan biasa membaca
`face.similarity_threshold` — angka yang, kalau diketahui, memberi petunjuk seberapa
longgar sistem bisa ditipu.

```json
// GET /settings (sebagai admin)
{
  "data": [
    { "key": "attendance.max_distance_meter", "value": 100, "value_type": "number",
      "description": "Radius maksimum dari lokasi kantor (meter)", "is_public": true,
      "updated_at": "2026-09-04T03:00:00Z" }
  ]
}
```

`PUT /settings/{key}` — body `{ "value": 150 }`. Server memvalidasi terhadap
`value_type` dan terhadap validator per-key (mis. `face.similarity_threshold` harus
`number` dalam rentang 0–1; `attendance.max_distance_meter` integer 10–10000).
Validator per-key hidup di kode, bukan di DB. Setiap perubahan menulis `audit_logs`
dengan nilai lama dan baru.

| Status | code | Kondisi |
|---|---|---|
| 404 | `NOT_FOUND` | key tidak dikenal (key baru hanya lewat migration) |
| 422 | `VALIDATION_ERROR` | tipe/rentang tidak valid |

### 4.7 Audit logs

| Method | Path | Guard |
|---|---|---|
| GET | `/audit-logs` | `audit.read` |

Query: `page`, `per_page` (maks 100), `actor_user_id`, `action`, `resource_type`,
`resource_id`, `from`, `to` (RFC3339). Urut `created_at DESC`.
Tidak ada endpoint create/update/delete — audit log yang bisa diedit bukan audit log.

### 4.8 Ringkasan seluruh endpoint Fase 1

| # | Method | Path | Guard |
|---|---|---|---|
| 1 | POST | `/api/v1/auth/login` | — |
| 2 | POST | `/api/v1/auth/refresh` | — |
| 3 | POST | `/api/v1/auth/logout` | auth |
| 4 | POST | `/api/v1/auth/logout-all` | auth |
| 5 | GET | `/api/v1/auth/me` | auth |
| 6 | POST | `/api/v1/auth/change-password` | auth |
| 7 | GET | `/api/v1/employees` | `employee.read` |
| 8 | POST | `/api/v1/employees` | `employee.create` |
| 9 | GET | `/api/v1/employees/me` | `employee.read_self` |
| 10 | GET | `/api/v1/employees/{id}` | `employee.read` \| self |
| 11 | PATCH | `/api/v1/employees/{id}` | `employee.update` |
| 12 | DELETE | `/api/v1/employees/{id}` | `employee.delete` |
| 13 | GET | `/api/v1/users` | `user.read` |
| 14 | POST | `/api/v1/users` | `user.create` |
| 15 | GET | `/api/v1/users/{id}` | `user.read` |
| 16 | PATCH | `/api/v1/users/{id}` | `user.update` |
| 17 | DELETE | `/api/v1/users/{id}` | `user.delete` |
| 18 | PATCH | `/api/v1/users/{id}/status` | `user.update` |
| 19 | PUT | `/api/v1/users/{id}/roles` | `user.assign_role` |
| 20 | POST | `/api/v1/users/{id}/reset-password` | `user.reset_password` |
| 21 | GET | `/api/v1/roles` | `role.read` |
| 22 | POST | `/api/v1/roles` | `role.create` |
| 23 | GET | `/api/v1/roles/{id}` | `role.read` |
| 24 | PATCH | `/api/v1/roles/{id}` | `role.update` |
| 25 | DELETE | `/api/v1/roles/{id}` | `role.delete` |
| 26 | PUT | `/api/v1/roles/{id}/permissions` | `role.assign_permission` |
| 27 | GET | `/api/v1/permissions` | `permission.read` |
| 28 | GET | `/api/v1/settings` | `settings.read` |
| 29 | GET | `/api/v1/settings/{key}` | `settings.read` |
| 30 | PUT | `/api/v1/settings/{key}` | `settings.update` |
| 31 | GET | `/api/v1/audit-logs` | `audit.read` |

---

## 5. Flow

### 5.1 Login

```
POST /auth/login {email, password}
 1. Rate limit per IP (10/menit) ─ lampaui ⇒ 429
 2. Validasi bentuk body ─ gagal ⇒ 422
 3. SELECT user WHERE email=$1 AND deleted_at IS NULL
      tidak ada ⇒ tetap jalankan Argon2id verify terhadap hash dummy
                  (menyamakan waktu respons; tanpa ini, selisih waktu
                   membocorkan email mana yang terdaftar) ⇒ 401
 4. locked_until > now()  ⇒ 401 INVALID_CREDENTIALS (tidak dibedakan)
 5. is_active = false     ⇒ 401 INVALID_CREDENTIALS
 6. argon2id.Verify(password, user.password_hash)
      gagal ⇒ failed_login_count += 1
              bila mencapai security.max_failed_login:
                 locked_until = now() + security.lockout_minutes
              audit: auth.login.failed
              ⇒ 401
 7. Sukses:
      failed_login_count = 0, locked_until = NULL, last_login_at = now()
      access  = JWT{sub, eid, tv: user.token_version, exp: +15m}
      raw     = crypto/rand 32 byte → base64url
      INSERT refresh_tokens{token_hash: sha256(raw), family_id: uuid baru,
                            expires_at: +30d, user_agent, ip}
      audit: auth.login.success
 8. 200 dengan access_token, refresh_token(raw), dan objek user + permissions
```

### 5.2 Rotasi refresh token & deteksi reuse

```
POST /auth/refresh {refresh_token}
 1. h = sha256(refresh_token)
 2. SELECT * FROM refresh_tokens WHERE token_hash = h
      tidak ada           ⇒ 401 INVALID_REFRESH_TOKEN
 3. revoked_at IS NOT NULL ⇒ 401 INVALID_REFRESH_TOKEN
 4. expires_at < now()     ⇒ 401 INVALID_REFRESH_TOKEN
 5. used_at IS NOT NULL    ⇒ TOKEN DIPAKAI ULANG
        BEGIN
          UPDATE refresh_tokens SET revoked_at=now(), revoked_reason='reuse_detected'
            WHERE family_id = $family AND revoked_at IS NULL
          UPDATE users SET token_version = token_version + 1 WHERE id = $user
        COMMIT
        audit: auth.refresh.reuse_detected
        ⇒ 401 REFRESH_TOKEN_REUSED
 6. Muat user; nonaktif / soft-deleted ⇒ revoke family ⇒ 401
 7. BEGIN
      UPDATE refresh_tokens SET used_at=now() WHERE id=$old
      INSERT refresh_tokens{family_id: $old.family_id, parent_id: $old.id, ...}
    COMMIT
 8. 200 dengan access_token baru + refresh_token baru
```

Langkah 5 dan 7 dijalankan dalam satu transaksi dengan `SELECT ... FOR UPDATE`
pada baris token, supaya dua request refresh yang datang bersamaan tidak keduanya
lolos. Balapan seperti ini nyata: aplikasi mobile yang membuka dua tab akan
melakukannya sendiri, dan tanpa lock, satu request sah akan salah dituduh reuse.

### 5.3 Middleware `Authenticate` → `RequirePermission`

```
Authenticate:
 1. Baca header Authorization: Bearer <jwt>
      tidak ada ⇒ 401 UNAUTHENTICATED
 2. Parse + verifikasi tanda tangan HS256, iss, aud, exp, nbf
      gagal ⇒ 401 UNAUTHENTICATED   (pesan generik, tidak menyebut alasan spesifik)
 3. typ != "access" ⇒ 401  (refresh token tidak boleh dipakai sebagai access token)
 4. Muat konteks user dari cache (TTL 30 detik), miss ⇒ query:
      SELECT u.id, u.employee_id, u.is_active, u.token_version,
             array_agg(DISTINCT r.name)  AS roles,
             array_agg(DISTINCT p.name)  AS permissions
      FROM users u
      LEFT JOIN user_roles ur       ON ur.user_id = u.id
      LEFT JOIN roles r             ON r.id = ur.role_id AND r.deleted_at IS NULL
      LEFT JOIN role_permissions rp ON rp.role_id = r.id
      LEFT JOIN permissions p       ON p.id = rp.permission_id
      WHERE u.id = $1 AND u.deleted_at IS NULL
      GROUP BY u.id
 5. user tidak ada / is_active=false ⇒ 401
 6. jwt.tv != user.token_version     ⇒ 401  (token sudah dicabut)
 7. Simpan Principal{user_id, employee_id, roles, permissions} ke ctx

RequirePermission("x.y"):
 8. principal.Has("x.y") ⇒ lanjut
    tidak ⇒ 403 FORBIDDEN, audit TIDAK ditulis (403 rutin akan membanjiri audit;
            yang dicatat hanya percobaan di endpoint bertanda sensitif)
```

**Aturan default-deny.** Router dibangun sedemikian rupa sehingga route di bawah
`/api/v1` yang tidak secara eksplisit dibungkus `RequirePermission` atau ditandai
publik akan **gagal di test**, bukan diam-diam terbuka. Implementasinya: registry
route (`routes.go`) yang mendaftarkan tiap route beserta guard-nya, lalu satu test
yang menelusuri router chi (`chi.Walk`) dan memastikan setiap pola route ada di
registry dan punya guard non-kosong. Ini menutup lubang paling umum di sistem RBAC:
endpoint baru yang lupa dipasangi guard.

### 5.4 Pola akses "self"

Endpoint yang bisa diakses baik oleh admin (semua data) maupun karyawan (data
sendiri) memakai pola ini — bukan dua endpoint terpisah:

```go
// Guard: cukup punya SALAH SATU
r.With(RequireAnyPermission("employee.read", "employee.read_self")).
  Get("/employees/{id}", h.GetEmployee)

// Di dalam handler:
func (h *Handler) GetEmployee(w http.ResponseWriter, r *http.Request) {
    p := authctx.Principal(r.Context())
    id := chi.URLParam(r, "id")

    // Kepemilikan diperiksa HANYA bila ia tidak punya izin baca menyeluruh.
    if !p.Has("employee.read") {
        if p.EmployeeID == nil || *p.EmployeeID != id {
            // 404, bukan 403: 403 mengonfirmasi bahwa id itu ada.
            httpx.Fail(w, r, httpx.NotFound())
            return
        }
    }
    ...
}
```

Dua hal yang wajib diikuti fase berikutnya:
1. Cek kepemilikan ada di **handler/service**, bukan di middleware — middleware
   tidak tahu resource-nya milik siapa.
2. Akses ke resource milik orang lain dibalas **404**, bukan 403. 403 memberi tahu
   penyerang bahwa id yang ia tebak itu ada.

### 5.5 Pencegahan privilege escalation

Dua aturan, keduanya ditegakkan di service layer:

**A. Tidak bisa memberi apa yang tidak dimiliki.**
Saat `PUT /users/{id}/roles` atau `PUT /roles/{id}/permissions`, kumpulan permission
efektif yang akan diberikan harus merupakan **himpunan bagian** dari permission
milik pelaku. Kalau tidak → `403 ROLE_ESCALATION_DENIED`.

Tanpa aturan ini, `admin` yang punya `user.assign_role` bisa memberi dirinya sendiri
role `super_admin` — dan itu membuat seluruh pemisahan role di master plan § 6 tidak
ada artinya. (Karena itu pula `admin` default **tidak** diberi `role.assign_permission`.)

**B. Selalu ada minimal satu super admin aktif.**
Ditolak dengan `409 LAST_SUPER_ADMIN` bila operasi akan menghasilkan nol user aktif
yang memiliki permission `role.assign_permission`:
- mencabut role `super_admin` dari user terakhir yang memilikinya,
- menonaktifkan / soft-delete user super admin terakhir,
- mencabut `role.assign_permission` dari role `super_admin` saat itu satu-satunya jalur.

Pemeriksaan dilakukan **di dalam transaksi yang sama** dengan perubahannya, dengan
`SELECT ... FOR UPDATE` pada baris user yang bersangkutan. Kalau tidak, dua admin
yang menonaktifkan satu sama lain bersamaan akan berhasil keduanya dan sistem
terkunci permanen.

### 5.6 Seeding

Seeder adalah **program terpisah** (`cmd/seed`), bukan migration — supaya bisa
dijalankan ulang dengan aman dan tidak mencampur data dengan struktur.

```
cmd/seed run:
 1. UPSERT semua baris `permissions` dari katalog di kode (§ 2.3)
       ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description
 2. UPSERT roles: super_admin, admin, employee (is_system = true)
 3. Sinkronkan role_permissions sesuai peta di § 2.4
       super_admin selalu mendapat SEMUA permission yang ada saat itu
 4. UPSERT app_settings (hanya INSERT bila key belum ada — jangan menimpa
    nilai yang sudah di-tuning admin)
 5. Bila belum ada user sama sekali:
       buat super admin awal dari env SEED_ADMIN_EMAIL + SEED_ADMIN_PASSWORD
       must_change_password = true
       cetak email-nya ke stdout (password TIDAK dicetak — ia berasal dari env
       yang sudah dipegang operator)
    Bila sudah ada user: langkah ini dilewati, dengan pesan jelas.
```

Idempoten: aman dijalankan berkali-kali. Di dev, `cmd/api` memanggilnya otomatis
setelah migration bila `APP_ENV=development`. Di production dijalankan manual.

Kalau `APP_ENV != development` dan `SEED_ADMIN_PASSWORD` kosong sementara belum ada
user → **exit(1)**. Sistem tanpa jalan masuk lebih baik daripada sistem dengan
password default.

---

## 6. Edge Case & Validasi

### 6.1 Autentikasi

| # | Kondisi | Penanganan |
|---|---|---|
| E1 | Email tidak terdaftar | 401 `INVALID_CREDENTIALS`. Tetap jalankan verify Argon2id terhadap hash dummy agar waktu respons seragam |
| E2 | Password salah | 401 sama persis dengan E1 — pesan, code, dan waktu tidak boleh membedakan |
| E3 | User `is_active = false` | 401 `INVALID_CREDENTIALS` (bukan 403 — jangan konfirmasi keberadaan akun) |
| E4 | Akun terkunci | 401 `INVALID_CREDENTIALS`. Alasan kunci hanya masuk audit log, tidak ke client |
| E5 | User soft-deleted lalu login | 401. Query login selalu `AND deleted_at IS NULL` |
| E6 | Access token kedaluwarsa | 401 `UNAUTHENTICATED`; client memakai `/auth/refresh` |
| E7 | Token dengan `alg: none` atau algoritma lain | Ditolak. Parser dikonfigurasi hanya menerima HS256 — jangan pakai default library |
| E8 | Refresh token dipakai sebagai `Authorization: Bearer` | 401 (klaim `typ` diperiksa) |
| E9 | Dua request refresh bersamaan (race) | `SELECT FOR UPDATE`; satu berhasil, satu dapat 401 reuse. Client menangani dengan login ulang |
| E10 | Refresh token dicuri lalu dipakai penyerang | Rotasi + reuse detection mencabut seluruh family; korban dipaksa login ulang. Terdeteksi, tidak diam-diam |
| E11 | Password diganti saat ada 3 sesi aktif | Semua 3 dicabut (`token_version += 1` + revoke semua refresh token) |
| E12 | Role user diubah saat ia sedang aktif | Cache di-invalidasi eksplisit → berlaku pada request berikutnya (bukan menunggu TTL) |
| E13 | Clock skew antar-container | JWT diberi leeway 30 detik saat verifikasi `exp`/`nbf` |
| E14 | `JWT_SECRET` diganti | Semua token menjadi tidak valid. Perilaku yang benar; didokumentasikan sebagai cara "cabut semua sesi darurat" |
| E15 | `JWT_SECRET` < 32 byte | Config gagal saat startup, exit(1) |

### 6.2 Otorisasi

| # | Kondisi | Penanganan |
|---|---|---|
| E16 | Endpoint baru lupa dipasangi guard | Test `TestAllRoutesHaveGuards` gagal (§ 5.3) |
| E17 | Employee mengakses `/employees/{id}` milik orang lain | **404**, bukan 403 (§ 5.4) |
| E18 | Admin memberi role `super_admin` ke dirinya | 403 `ROLE_ESCALATION_DENIED` (§ 5.5 A) |
| E19 | Mencabut role super admin terakhir | 409 `LAST_SUPER_ADMIN` (§ 5.5 B) |
| E20 | Menonaktifkan super admin terakhir | 409 `LAST_SUPER_ADMIN` |
| E21 | Menghapus role yang masih dipakai | 409 `ROLE_IN_USE`, dengan `details` berisi jumlah user terdampak |
| E22 | Menghapus / rename role `is_system` | 403 `SYSTEM_ROLE_IMMUTABLE` |
| E23 | `role_ids` memuat id yang tidak ada | 404 `NOT_FOUND`, seluruh operasi dibatalkan (transaksi) |
| E24 | `PUT /roles/{id}/permissions` dengan array kosong | Diizinkan (role tanpa hak), **kecuali** melanggar E19 |
| E25 | User tanpa role sama sekali | Login berhasil, tapi `permissions` kosong → semua endpoint terproteksi 403. Perilaku yang benar, bukan bug |
| E26 | Employee memanggil `GET /settings` | 200, tapi hanya baris `is_public = true` |

### 6.3 Data & konkurensi

| # | Kondisi | Penanganan |
|---|---|---|
| E27 | `email` beda kapitalisasi (`Budi@x.com` vs `budi@x.com`) | Dianggap sama — kolom `citext` + unique index |
| E28 | Email berspasi di ujung | Di-`trim` sebelum validasi & simpan |
| E29 | Dua request `POST /users` dengan email sama, bersamaan | Satu berhasil; yang lain kena unique violation → dipetakan ke 409 `EMAIL_TAKEN` (jangan andalkan cek `SELECT` sebelumnya — itu bocor) |
| E30 | `employee_number` dipakai ulang setelah karyawan lama di-soft-delete | Diizinkan (unique index partial) |
| E31 | Hapus employee yang masih punya user aktif | 409 `EMPLOYEE_HAS_ACTIVE_USER` |
| E32 | Hapus user yang punya refresh token | Berhasil; `ON DELETE CASCADE` (tapi ini soft-delete, jadi token dicabut eksplisit, bukan dihapus) |
| E33 | `per_page=100000` | Di-clamp ke 100. Jangan balas error — clamp diam-diam lebih ramah dan tetap aman |
| E34 | `page=0` atau negatif | Di-clamp ke 1 |
| E35 | `sort` dengan kolom yang tidak dikenal | 422. **Whitelist** kolom sort; jangan pernah menyusun `ORDER BY` dari input mentah |
| E36 | `q` memuat `%` atau `_` | Di-escape sebelum `ILIKE` |
| E37 | Update konkuren pada user yang sama | Update terakhir menang (Fase 1 tidak memakai optimistic locking). Didokumentasikan; kalau jadi masalah nyata, tambahkan kolom `version` di fase lain |
| E38 | Body `PATCH` kosong `{}` | 422 `VALIDATION_ERROR` — tidak ada yang diubah |
| E39 | `PATCH` mengirim field yang tidak dikenal | Diabaikan diam-diam (forward compatible), tapi dicatat di log level debug |

### 6.4 Validasi input (ringkas)

| Field | Aturan |
|---|---|
| `email` | wajib, ≤ 254 char, regex email, di-trim, di-lowercase |
| `password` | wajib, 10–128 char, mengandung huruf **dan** angka, tidak ada di daftar password lemah, bukan sama dengan email |
| `employee_number` | wajib, 1–50 char, `^[A-Za-z0-9._/-]+$` |
| `full_name` | wajib, 2–120 char setelah trim |
| `phone` | opsional, `^[0-9+][0-9 +()-]{6,19}$` |
| `join_date` | opsional, format `YYYY-MM-DD`, tidak boleh > hari ini |
| `employment_status` | salah satu dari `active`, `inactive`, `resigned` |
| `role.name` | `^[a-z][a-z0-9_]{2,49}$` |
| `role_ids` / `permission_ids` | array UUID valid, maks 50 item, tanpa duplikat |
| `page` / `per_page` | integer; di-clamp (E33/E34) |

Validasi dilakukan di dua tempat dan itu disengaja: `go-playground/validator` di
batas HTTP untuk pesan yang ramah, dan `CHECK` constraint di database sebagai
jaring pengaman terakhir. Yang di database berlaku bahkan untuk data yang masuk
lewat seeder atau perbaikan manual.

---

## 7. Struktur Folder

Yang **ditambahkan** ke `apps/faceclock-api/` (struktur dasar dari Fase 0):

```
apps/faceclock-api/
├── cmd/
│   ├── api/main.go                      # (diubah) mount modul auth/user/role
│   └── seed/main.go                      # ← BARU: seeder idempoten
├── internal/
│   ├── auth/                             # ← BARU
│   │   ├── handler.go                    # login, refresh, logout, me, change-password
│   │   ├── service.go                    # orkestrasi login & rotasi token
│   │   ├── repository.go
│   │   ├── jwt.go                        # issue + verify access token (HS256 saja)
│   │   ├── refresh.go                    # generate, hash, rotate, reuse detection
│   │   ├── password.go                   # Argon2id hash/verify + kebijakan
│   │   ├── weak_passwords.txt            # daftar statik 200 password terlemah
│   │   ├── principal.go                  # tipe Principal + helper Has()/HasAny()
│   │   ├── context.go                    # simpan/ambil Principal dari ctx
│   │   ├── middleware.go                 # Authenticate
│   │   ├── jwt_test.go
│   │   ├── password_test.go
│   │   ├── refresh_test.go
│   │   └── service_test.go
│   ├── rbac/                             # ← BARU
│   │   ├── permissions.go                # katalog permission sebagai konstanta Go
│   │   ├── middleware.go                 # RequirePermission, RequireAnyPermission
│   │   ├── cache.go                      # cache permission per user + invalidasi
│   │   ├── escalation.go                 # aturan subset (§ 5.5 A)
│   │   ├── guardrails.go                 # aturan last-super-admin (§ 5.5 B)
│   │   ├── middleware_test.go
│   │   └── escalation_test.go
│   ├── user/                             # ← BARU
│   │   ├── handler.go  service.go  repository.go  dto.go  *_test.go
│   ├── employee/                         # ← BARU
│   │   ├── handler.go  service.go  repository.go  dto.go  *_test.go
│   ├── role/                             # ← BARU
│   │   ├── handler.go  service.go  repository.go  dto.go  *_test.go
│   ├── settings/                         # ← BARU
│   │   ├── handler.go  service.go  repository.go
│   │   ├── validators.go                 # validator per-key
│   │   └── *_test.go
│   ├── audit/                            # ← BARU
│   │   ├── handler.go                    # GET /audit-logs
│   │   ├── recorder.go                   # Record(ctx, action, resource, metadata)
│   │   ├── actions.go                    # konstanta nama action
│   │   └── recorder_test.go
│   ├── httpx/
│   │   ├── routes.go                     # ← BARU: registry route + guard-nya
│   │   └── routes_test.go                # ← BARU: TestAllRoutesHaveGuards
│   └── ... (dari Fase 0)
├── migrations/
│   ├── 000002_create_employees.{up,down}.sql
│   ├── 000003_create_users.{up,down}.sql
│   ├── 000004_create_roles.{up,down}.sql
│   ├── 000005_create_permissions.{up,down}.sql
│   ├── 000006_create_role_permissions.{up,down}.sql
│   ├── 000007_create_user_roles.{up,down}.sql
│   ├── 000008_create_refresh_tokens.{up,down}.sql
│   ├── 000009_create_audit_logs.{up,down}.sql
│   └── 000010_create_app_settings.{up,down}.sql
├── test/
│   ├── integration/
│   │   ├── auth_test.go
│   │   ├── rbac_matrix_test.go           # matriks role × endpoint (§ 11.3)
│   │   ├── user_test.go
│   │   ├── employee_test.go
│   │   ├── role_test.go
│   │   └── helpers.go                    # spin-up DB (testcontainers), login as role
│   └── fixtures/
└── docs/
    └── api/
        └── fase1-auth-rbac.md            # kontrak API yang di-generate/ditulis manual
```

Catatan struktur: tiap modul (`auth`, `user`, `employee`, `role`, `settings`,
`audit`) berisi `handler → service → repository` dengan arah dependensi satu arah.
Handler tidak pernah menyentuh `pgxpool` langsung. Ini bukan selera — ini yang
membuat aturan di § 5.5 (escalation, last-super-admin) bisa hidup di satu tempat
dan tidak terlewat.

---

## 8. Checklist Task

### 8.1 Migration & skema
- [ ] `000002_create_employees` (+ index partial unique, index department/nama/status)
- [ ] `000003_create_users` (+ partial unique email & employee_id)
- [ ] `000004_create_roles`
- [ ] `000005_create_permissions`
- [ ] `000006_create_role_permissions`
- [ ] `000007_create_user_roles`
- [ ] `000008_create_refresh_tokens`
- [ ] `000009_create_audit_logs`
- [ ] `000010_create_app_settings`
- [ ] Semua `.down.sql` ditulis dan diuji (`migrate down` sampai versi 1, lalu `up` lagi)
- [ ] Trigger/`updated_at` otomatis (function `set_updated_at()` + trigger per tabel)

### 8.2 Password & token
- [ ] `internal/auth/password.go`: Argon2id hash + verify + parse encoded string
- [ ] Kebijakan password + `weak_passwords.txt`
- [ ] Unit test: hash ≠ hash untuk password sama (salt acak), verify benar/salah
- [ ] `internal/auth/jwt.go`: issue + verify, **hanya** HS256, cek `iss`/`aud`/`typ`, leeway 30s
- [ ] Unit test: token `alg:none` ditolak, token dengan `aud` salah ditolak, token expired ditolak
- [ ] `internal/auth/refresh.go`: generate (crypto/rand 32B), sha256, rotate, reuse detection
- [ ] Unit test rotasi: token lama tidak berlaku setelah rotasi
- [ ] Unit test reuse: memakai token yang sudah dirotasi mencabut seluruh family

### 8.3 Auth endpoints
- [ ] `POST /auth/login` termasuk verify dummy saat email tidak ada (E1)
- [ ] Lockout per akun (5×/15 menit) + reset counter saat sukses
- [ ] Rate limit per IP di `/auth/login` dan `/auth/refresh`
- [ ] `POST /auth/refresh` dengan `SELECT FOR UPDATE` di dalam transaksi
- [ ] `POST /auth/logout` (idempoten, selalu 204)
- [ ] `POST /auth/logout-all`
- [ ] `GET /auth/me`
- [ ] `POST /auth/change-password` (mencabut semua sesi)

### 8.4 RBAC
- [ ] `internal/rbac/permissions.go` — katalog lengkap § 2.3 sebagai konstanta Go
- [ ] Middleware `Authenticate` (§ 5.3)
- [ ] Middleware `RequirePermission` / `RequireAnyPermission`
- [ ] Cache permission per user (TTL 30s) + invalidasi eksplisit saat role/permission berubah
- [ ] `internal/httpx/routes.go` — registry route + guard
- [ ] `TestAllRoutesHaveGuards` — gagal bila ada route `/api/v1/*` tanpa guard
- [ ] `escalation.go` — aturan subset permission (§ 5.5 A) + test
- [ ] `guardrails.go` — aturan last-super-admin (§ 5.5 B) + test konkurensi
- [ ] Helper pola ownership (§ 5.4) + contoh terpakai di `/employees/{id}`

### 8.5 CRUD
- [ ] Modul `employee`: list (filter/sort/paginate), create, get, get-me, patch, soft-delete
- [ ] Modul `user`: list, create, get, patch, soft-delete, status, roles, reset-password
- [ ] Modul `role`: list, create, get, patch, delete, set-permissions
- [ ] `GET /permissions` (+ `group_by=resource`)
- [ ] Modul `settings`: list (dengan filter `is_public`), get, put + validator per-key
- [ ] Modul `audit`: recorder + `GET /audit-logs` dengan filter
- [ ] Verifikasi: tidak ada satupun response yang memuat `password_hash` / `token_hash`

### 8.6 Seeder
- [ ] `cmd/seed` idempoten (§ 5.6)
- [ ] Seed permissions (termasuk permission Fase 3–5)
- [ ] Seed 3 role sistem + peta permission-nya
- [ ] Seed `app_settings` (INSERT-only, tidak menimpa)
- [ ] Super admin awal dari `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD`, `must_change_password=true`
- [ ] Exit(1) bila production tanpa `SEED_ADMIN_PASSWORD` dan belum ada user
- [ ] Dipanggil otomatis oleh `cmd/api` hanya saat `APP_ENV=development`

### 8.7 Housekeeping & config
- [ ] Env baru di `deploy/.env.example`: `JWT_SECRET`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`
- [ ] Validasi `JWT_SECRET` ≥ 32 byte, wajib saat `APP_ENV != development`
- [ ] Goroutine ticker: hapus refresh token kedaluwarsa > 30 hari

### 8.8 Test & dokumentasi
- [ ] Integration test harness (testcontainers Postgres + migrate + seed)
- [ ] `rbac_matrix_test.go` — matriks role × endpoint (§ 11.3)
- [ ] Test alur lengkap: login → akses → refresh → logout
- [ ] Test: `password_hash` tidak pernah bocor di response mana pun
- [ ] `docs/api/fase1-auth-rbac.md` (atau OpenAPI spec) berisi 31 endpoint § 4.8
- [ ] Koleksi Postman/Bruno/`.http` untuk uji manual
- [ ] `DONE-Fase-1.md` sesuai Protokol Handoff master plan § 10.4

---

## 9. Dependencies

**Prasyarat:**

| Dari | Yang dibutuhkan |
|---|---|
| Fase 0 | Router chi + middleware chain (slot `Authenticate`/`RequirePermission` sudah tersedia) |
| Fase 0 | Response envelope + katalog error code |
| Fase 0 | Tooling migration + konvensi skema (uuid PK, `timestamptz`, soft delete, partial unique) |
| Fase 0 | `internal/config` fail-fast, `internal/platform/postgres`, logger dengan aturan redaksi |
| Fase 0 | Ekstensi `citext` dan `pgcrypto` aktif |
| — | Keputusan D5 (akses DB) dari Fase 0 sudah final |

**Library baru:**
`golang.org/x/crypto/argon2`, `github.com/golang-jwt/jwt/v5`,
`github.com/testcontainers/testcontainers-go` (test saja).

**Yang bergantung pada fase ini:**

| Fase | Mengambil apa |
|---|---|
| Fase 3 | `employees`, `Principal.EmployeeID`, permission `face.*`, pola ownership § 5.4 |
| Fase 4 | Permission `attendance.*`, `app_settings` untuk threshold & geofence, `audit_logs` untuk jejak approval |
| Fase 5 | Semua endpoint user/role/permission/settings; `GET /auth/me` untuk guard UI |
| Fase 6 | Login + `attendance.checkin` + `face.enroll_self` |
| Fase 7 | API auth yang sama persis (mobile tidak punya endpoint sendiri) |

**Bisa paralel:** [Fase 2](03-Fase2.md) tidak bergantung pada Fase 1 dan boleh
dikerjakan bersamaan. Satu-satunya titik temunya: Fase 2 mengisi
`app_settings.face.similarity_threshold` dan `face.model_version` — dua baris yang
sudah di-seed Fase 1 (§ 3.9) — dan menambahkan migration `000011`, tepat setelah
`000010` milik fase ini.

Peta lengkap artefak Fase 0–2 yang dipakai Fase 3–7 ada di
**[03-Fase2.md § 13 — Peta Kesinambungan Fase 0 → 7](03-Fase2.md#13-peta-kesinambungan-fase-0--7)**.

---

## 10. Definition of Done

1. Sembilan migration (`000002`–`000010`) jalan bersih dari database kosong, dan
   `down` mengembalikannya ke keadaan Fase 0 tanpa sisa.
2. `cmd/seed` idempoten: dijalankan 3× berturut-turut menghasilkan state identik,
   tanpa error dan tanpa duplikat.
3. Tiga role sistem ada dengan peta permission sesuai § 2.4; `super_admin`
   memiliki **semua** baris `permissions`.
4. Login dengan akun super admin awal berhasil dan mengembalikan
   `must_change_password: true`.
5. Alur token lengkap terbukti: login → panggil endpoint terproteksi → tunggu
   access token kedaluwarsa → refresh → panggil lagi → logout → token lama ditolak.
6. Reuse detection terbukti: memakai ulang refresh token yang sudah dirotasi
   membalas 401, mencabut seluruh family, dan menulis `audit_logs`.
7. **Matriks RBAC lulus penuh** (§ 11.3): untuk setiap kombinasi
   (role × 31 endpoint), hasilnya sesuai yang diharapkan — 3 role × 31 endpoint.
8. `TestAllRoutesHaveGuards` lulus: tidak ada route `/api/v1/*` tanpa guard eksplisit.
9. Privilege escalation ditolak: `admin` tidak bisa memberi `super_admin` ke siapa pun.
10. Super admin terakhir tidak bisa dinonaktifkan/dicabut (409 `LAST_SUPER_ADMIN`),
    termasuk saat dua request berjalan bersamaan.
11. Employee yang mengakses `/employees/{id}` milik orang lain menerima **404**.
12. Tidak ada response API yang memuat `password_hash` atau `token_hash` — dibuktikan
    dengan test yang memindai seluruh body response di integration test.
13. Tidak ada log yang memuat password, token, atau isi header `Authorization` —
    test dari Fase 0 § 11.6 masih lulus.
14. `GET /settings` sebagai `employee` hanya mengembalikan baris `is_public = true`.
15. Semua aksi di daftar § 3.8 tercatat di `audit_logs` dengan `actor_user_id` benar.
16. Coverage unit test modul `auth` dan `rbac` ≥ 80%.
17. `make lint` dan `make test` lulus; CI hijau.
18. `docs/api/fase1-auth-rbac.md` lengkap dan cocok dengan implementasi.
19. `DONE-Fase-1.md` ada.

---

## 11. Cara Test / Verifikasi

### 11.1 Bootstrap

```bash
make reset            # database bersih
make up
docker compose -f deploy/docker-compose.yml exec faceclock-api /app/seed
```

### 11.2 Alur manual (curl)

```bash
API=http://localhost:8080/api/v1

# 1) Login super admin
LOGIN=$(curl -s -X POST $API/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@faceclock.local","password":"<SEED_ADMIN_PASSWORD>"}')
AT=$(echo "$LOGIN" | jq -r .data.access_token)
RT=$(echo "$LOGIN" | jq -r .data.refresh_token)

# 2) Identitas & permission
curl -s $API/auth/me -H "Authorization: Bearer $AT" | jq '.data.roles, (.data.permissions|length)'

# 3) Buat karyawan
EMP=$(curl -s -X POST $API/employees -H "Authorization: Bearer $AT" \
  -H 'Content-Type: application/json' \
  -d '{"employee_number":"EMP-0002","full_name":"Budi Santoso","department":"IT"}')
EMP_ID=$(echo "$EMP" | jq -r .data.id)

# 4) Buat user employee untuk karyawan itu
ROLE_EMP=$(curl -s "$API/roles" -H "Authorization: Bearer $AT" \
  | jq -r '.data[] | select(.name=="employee") | .id')
curl -s -X POST $API/users -H "Authorization: Bearer $AT" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"budi@faceclock.local\",\"password\":\"PasswordAwal123\",
       \"employee_id\":\"$EMP_ID\",\"role_ids\":[\"$ROLE_EMP\"]}" | jq .data.id

# 5) Login sebagai employee, lalu coba akses endpoint admin → harus 403
BAT=$(curl -s -X POST $API/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"budi@faceclock.local","password":"PasswordAwal123"}' | jq -r .data.access_token)
curl -s -o /dev/null -w "%{http_code}\n" $API/users     -H "Authorization: Bearer $BAT"   # 403
curl -s -o /dev/null -w "%{http_code}\n" $API/employees -H "Authorization: Bearer $BAT"   # 403
curl -s -o /dev/null -w "%{http_code}\n" $API/employees/me -H "Authorization: Bearer $BAT" # 200

# 6) Rotasi refresh token
NEW=$(curl -s -X POST $API/auth/refresh -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$RT\"}")
echo "$NEW" | jq -r .data.access_token > /dev/null

# 7) Reuse detection — pakai RT lama sekali lagi → 401 REFRESH_TOKEN_REUSED
curl -s -X POST $API/auth/refresh -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$RT\"}" | jq .error.code
```

### 11.3 Matriks RBAC (test otomatis, wajib)

`test/integration/rbac_matrix_test.go` menjalankan **setiap** endpoint di § 4.8
sebagai masing-masing dari 4 principal, dan membandingkan status yang diperoleh
dengan tabel harapan yang ditulis eksplisit di file test:

| Principal | Contoh harapan |
|---|---|
| Tanpa token | Semua endpoint kecuali `/auth/login` & `/auth/refresh` → **401** |
| `employee` | `/employees/me` 200; `/users` 403; `/roles` 403; `/settings` 200 (hanya public); `/audit-logs` 403 |
| `admin` | `/users` 200; `/employees` 200; `/settings` 200 (semua); `POST /roles` **403**; `PUT /roles/{id}/permissions` **403** |
| `super_admin` | Semua 2xx |

Tabel harapan ditulis sebagai data, bukan sebagai `if`. Menambah endpoint baru
tanpa menambah barisnya di tabel akan membuat test gagal — itu tujuannya.

### 11.4 Test keamanan spesifik

| Yang diuji | Cara |
|---|---|
| Timing attack pada login | 200 request email tidak ada vs 200 request password salah; selisih median < 20 ms |
| `alg: none` | Susun JWT dengan header `{"alg":"none"}` dan signature kosong → harus 401 |
| Token milik user lain | Ambil token user A, panggil resource user B → 404/403, tidak pernah 200 |
| `token_version` | Login → `logout-all` di sesi lain → access token lama langsung 401 |
| Escalation | Sebagai `admin`, `PUT /users/{me}/roles` dengan `super_admin` → 403 |
| Last super admin | Sebagai satu-satunya super admin, `PATCH /users/{me}/status {is_active:false}` → 409 |
| Last super admin (race) | Dua goroutine menonaktifkan dua super admin terakhir bersamaan → tepat satu berhasil |
| Lockout | 5 login gagal → login ke-6 dengan password **benar** tetap 401; setelah 15 menit berhasil |
| SQL injection di `sort`/`q` | `?sort=id;DROP TABLE users` → 422; `?q=%' OR '1'='1` → hasil kosong, tabel utuh |
| Kebocoran hash | Grep seluruh body response integration test untuk `argon2` / `$2a$` / `password_hash` → nol hit |

### 11.5 Verifikasi database

```sql
-- super_admin punya SEMUA permission
SELECT (SELECT count(*) FROM permissions) AS total,
       count(*) AS granted
FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
WHERE r.name = 'super_admin';    -- total harus = granted

-- tidak ada permission yatim
SELECT p.name FROM permissions p
LEFT JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.permission_id IS NULL;

-- tidak ada user aktif tanpa role
SELECT u.email FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
WHERE u.deleted_at IS NULL AND u.is_active AND ur.user_id IS NULL;

-- audit log tercatat
SELECT action, count(*) FROM audit_logs GROUP BY action ORDER BY 2 DESC;

-- tidak ada refresh token yang belum dirotasi tapi punya anak (integritas rantai)
SELECT c.id FROM refresh_tokens c
JOIN refresh_tokens p ON p.id = c.parent_id
WHERE p.used_at IS NULL;
```

### 11.6 Verifikasi idempotensi seeder

```bash
docker compose -f deploy/docker-compose.yml exec faceclock-api /app/seed
docker compose -f deploy/docker-compose.yml exec faceclock-api /app/seed
docker compose -f deploy/docker-compose.yml exec faceclock-api /app/seed
make psql -c "SELECT (SELECT count(*) FROM roles), (SELECT count(*) FROM permissions), (SELECT count(*) FROM users);"
# angkanya harus identik dengan setelah run pertama
```

---

## 12. Referensi Silang ke "Isu Lintas-Fase" (Master Plan § 9)

| Isu lintas-fase | Bagaimana Fase 1 memenuhinya |
|---|---|
| **RBAC konsisten di level API** | Guard berbasis permission di setiap route (§ 5.3), default-deny yang ditegakkan `TestAllRoutesHaveGuards`, matriks RBAC 3 role × 31 endpoint (§ 11.3), pola ownership eksplisit (§ 5.4) |
| **Threshold configurable** | Tabel `app_settings` + endpoint baca/ubah + validator per-key; `face.similarity_threshold` sudah ada barisnya dengan `face.model_version = "unset"` menunggu kalibrasi Fase 2 |
| **Timestamp server-side** | Semua `timestamptz` dengan `DEFAULT now()` sisi DB; `last_login_at`, `assigned_at`, `created_at` tidak pernah menerima nilai dari client |
| **Data biometrik = data sensitif (UU PDP)** | Belum ada data biometrik di fase ini, tapi fondasinya disiapkan: `audit_logs` untuk jejak siapa mengakses apa, `employee.*` vs `face.*` dipisah sehingga akses ke data wajah bisa diberikan tanpa memberi akses lain, dan larangan isi `metadata` audit (§ 3.8) |
| **Fallback wajib `pending_review`** | Belum relevan (Fase 4); permission `attendance.approve` sudah disiapkan dan sudah dipetakan ke `admin` |
| **Anti-spoofing / liveness** | Belum relevan (Fase 4/6/7) |
