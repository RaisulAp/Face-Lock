# Alur Input Data FaceClock (dari Nol sampai Siap Absen)

> Dokumen ini menjawab pertanyaan: **"Saya harus input data dari mana dulu?"**
>
> **Status: sudah SATU LANGKAH.** Admin cukup membuka **Pengguna → Daftarkan Karyawan**.
> Satu form itu membuat **data karyawan + akun login sekaligus**. Karyawan kemudian
> login sendiri dengan email tersebut dan melengkapi sisanya (NIP, jabatan, tanggal
> masuk, enrollment wajah) dari portal karyawan.
>
> Semua alur di bawah sudah diverifikasi langsung dari kode:
> - Skema DB: `apps/backend/migrations/000002_create_employees.up.sql`, `000003_create_users.up.sql`, `000025_single_step_registration.up.sql`
> - API: `apps/backend/internal/httpx/router.go`, `apps/backend/internal/user/user.go`, `POST /api/v1/users/register-employee`
> - UI: `apps/frontend/src/app/router.tsx`, `routes.config.ts`, `features/users/components/RegisterEmployeeModal.tsx`

---

## 0. Ringkasan Alur Baru (Quick Start)

```
Admin  →  Pengguna  →  [Daftarkan Karyawan]
                         ├─ Nama Lengkap
                         ├─ Email           (sekaligus jadi email login)
                         ├─ Nomor Telepon
                         ├─ Departemen
                         └─ Kantor
                                  │
                                  ▼
                    Sistem membuat 2 baris sekaligus dalam 1 transaksi:
                      • employees  (nama, email, telepon, departemen, kantor)
                      • users      (email login + password) + role "employee"
                                  │
                                  ▼
                    Layar menampilkan kredensial sekali:
                      email + kata sandi sementara  → salin, kirim ke karyawan
                                  │
                                  ▼
         Karyawan login  →  melengkapi sendiri dari portal:
                      NIP, Jabatan, Tanggal Masuk, Persetujuan, Wajah
```

Field yang **tidak** diisi admin: `employee_number` (NIP), `position`,
`join_date`, `employment_status`, `attendance_mode` — semuanya kosong/NULL
saat registrasi dan diisi karyawan sendiri (atau admin menyusul) di portal.

---

## 1. Konsep yang Harus Dipahami Dulu

FaceClock tetap menyimpan data dalam **dua entitas berbeda**:

```
┌─────────────────────────┐          ┌─────────────────────────┐
│       EMPLOYEES         │          │         USERS           │
│   (profil kepegawaian)  │          │   (profil akun login)   │
├─────────────────────────┤          ├─────────────────────────┤
│ id (uuid)               │◄─────────│ employee_id (FK, NULL)  │
│ employee_number (NIP)   │   1 : 0..1│ email (unik)            │
│ full_name               │          │ password_hash           │
│ department              │          │ is_active               │
│ position                │          │ must_change_password    │
│ phone, email            │          │ last_login_at           │
│ join_date               │          │                         │
│ employment_status       │          └─────────────────────────┘
│ attendance_mode         │                      │
└─────────────────────────┘                      │
          │                                      │
          │ 1 : N                                │ N : 1
          ▼                                      ▼
┌─────────────────────────┐          ┌─────────────────────────┐
│   face_embeddings       │          │      user_roles         │
│   attendances           │          └─────────────────────────┘
│   consents              │
└─────────────────────────┘
```

### Aturan penting (dari skema DB)

| Aturan | Bukti di kode |
|---|---|
| **Dua entitas tetap dipisah** | `users.employee_id` adalah FK ke `employees.id` |
| User **boleh** tidak punya Employee | kolom `employee_id` NULLABLE (akun sistem murni) |
| 1 Employee maksimal **1** User | `CREATE UNIQUE INDEX users_employee_id_uniq` |
| Employee yang punya User **tidak bisa dihapus** | `ON DELETE RESTRICT` |
| Wajah & absensi menempel ke **Employee**, bukan User | `face_embeddings`, `attendances` → `employee_id` |
| NIP **boleh kosong** saat registrasi | migrasi `000025`: `employee_number` → NULLABLE |
| Beberapa kolom employee **boleh kosong** | `000025`: `full_name`, `department`, `attendance_mode` → NULLABLE |
| Employee tahu dia dibuat oleh admin | kolom `profile_created_by_admin` |
| Form belum dilengkapi karyawan | kolom `profile_completed_at` masih NULL |

### ✅ Sekarang ada orkestrasi satu-langkah

`POST /api/v1/users/register-employee` menjalankan **satu transaksi Postgres**:

```
BEGIN
  INSERT INTO employees (full_name, email, phone, department, location_id, profile_created_by_admin=true)
  INSERT INTO users     (employee_id, email, password_hash, must_change_password)
  INSERT INTO user_roles(user_id, role_id)   -- role "employee"
COMMIT
```

Kalau salah satu langkah gagal (mis. email sudah dipakai), **seluruh transaksi
di-rollback** sehingga tidak pernah ada employee "yatim" tanpa akun.

> Catatan: form **Pengguna → Buat Akun** yang lama kini khusus untuk
> **akun sistem non-karyawan** (admin/auditor). Akun itu memang tanpa `employee_id`.

---

## 2. Alur Lengkap: Setup Sistem Pertama Kali

Lakukan **sekali saja** saat aplikasi baru dipasang.

```
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 0 — Seeder (otomatis, tidak perlu input manual)            │
├──────────────────────────────────────────────────────────────────┤
│  • Super Admin: admin@faceclock.local / 123456789                │
│    └─ dibuat TANPA employee_id (murni akun admin)                │
│  • Roles: super_admin, admin_hr, supervisor, employee            │
│  • Permissions: lengkap untuk semua modul                        │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                   Login: admin@faceclock.local
```

Setelah login sebagai super admin, urutannya:

```
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 1 — Pengaturan Sistem              Menu: Pengaturan Sistem │
│ URL: /settings                                                   │
├──────────────────────────────────────────────────────────────────┤
│  • Jam masuk / jam pulang                                        │
│  • Toleransi keterlambatan                                       │
│  • Radius geofence default                                       │
│  • Mode attendance (face / manual)                               │
│                                                                  │
│  Kerjakan ini DULU agar absensi punya acuan waktu yang benar.    │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 2 — Lokasi Kantor                  Menu: Lokasi Kantor     │
│ URL: /locations                                                  │
├──────────────────────────────────────────────────────────────────┤
│  • Nama lokasi  (contoh: "Kantor Pusat Jakarta")                 │
│  • Latitude / Longitude                                          │
│  • Radius (meter)                                                │
│                                                                  │
│  Wajib ada SEBELUM karyawan absen, karena clock-in butuh lokasi.  │
│  Bisa >1 lokasi jika ada beberapa cabang.                        │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 3 — Daftarkan Karyawan (SATU LANGKAH)                      │
│ Menu: Pengguna  →  tombol "Daftarkan Karyawan"                   │
│ URL: /users                                                      │
├──────────────────────────────────────────────────────────────────┤
│  Admin cukup mengisi 5 kolom:                                    │
│   • Nama Lengkap             (contoh: Budi Pratama)              │
│   • Email                    ← SEKALIGUS jadi email login        │
│   • Nomor Telepon                                                │
│   • Departemen               (contoh: Engineering / HRD)         │
│   • Kantor                   (pilih dari Tahap 2)                │
│                                                                  │
│  Opsional: metode kata sandi (otomatis / manual).                │
│                                                                  │
│  Yang terjadi di belakang layar (1 transaksi Postgres):          │
│   • employees ← dibuat; NIP & jabatan masih KOSONG               │
│   • users     ← dibuat dengan email di atas + role "employee"    │
│                                                                  │
│  Setelah sukses, kredensial ditampilkan SEKALI:                  │
│   email + kata sandi sementara  → salin & kirim ke karyawan       │
│                                                                  │
│  ❌ Tidak ada lagi dropdown "pilih karyawan belum punya akun".   │
│     Tidak ada risiko dropdown kosong.                            │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 3b — Karyawan Melengkapi Profilnya (self-service)          │
│ Portal karyawan → URL: /portal/profile                           │
├──────────────────────────────────────────────────────────────────┤
│  Karyawan login pakai email + kata sandi dari Tahap 3, lalu      │
│  mengisi sendiri:                                                │
│   • NIP / employee_number                                        │
│   • Jabatan                                                      │
│   • Tanggal Masuk                                                │
│   • Status kepegawaian & mode absensi                            │
│                                                                  │
│  Admin tetap boleh mengisi/mengoreksi dari /employees.           │
│  Setelah lengkap, kolom `profile_completed_at` terisi.           │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 4 — Persetujuan Biometrik (Consent)                        │
│ Menu: Persetujuan Biometrik  |  URL: /consents                   │
├──────────────────────────────────────────────────────────────────┤
│  Dua cara:                                                       │
│   a) Admin mencatat persetujuan karyawan dari halaman ini        │
│   b) Karyawan menyetujui sendiri via portal → /portal/consent    │
│                                                                  │
│  Wajib sebelum enrollment wajah (syarat kepatuhan data pribadi). │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 5 — Enrollment Wajah                                       │
│ Dua jalur berbeda (pilih sesuai kebutuhan):                      │
├──────────────────────────────────────────────────────────────────┤
│  A) ADMIN / HR                                                   │
│     Menu: Daftar Karyawan → tombol "Biometrik" pada baris        │
│     URL: /employees/:id/face                                     │
│     → Melihat status biometrik, menghapus data wajah             │
│     (butuh permission face.read_any)                             │
│                                                                  │
│  B) KARYAWAN SENDIRI (self-enrollment)                           │
│     Portal karyawan → URL: /portal/enrollment                    │
│     → Ambil 3-5 foto wajah via kamera                            │
│     (butuh permission face.enroll_self)                          │
│                                                                  │
│  ⚠️  SERVICE AI HARUS MENYALA. Tanpa itu enrollment gagal.        │
│     Jalankan: .\scripts\ai-up.ps1                                │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ TAHAP 6 — SETELAH KARYAWAN PERTAMA LULUS ENROLLMENT              │
│                                                                  │
│  Karyawan bisa:                                                  │
│   • Login sendiri → diarahkan ke /portal/attendance              │
│   • Absen clock-in di halaman Absensi                            │
│   • Lihat riwayat di /portal/history                             │
│                                                                  │
│  Admin bisa:                                                     │
│   • Pantau di /dashboard                                         │
│   • Review absensi di /attendances/pending                       │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3. Checklist Praktis (Centang Satu-satu)

Untuk **setiap karyawan baru**, urutan wajibnya:

```
□ 1. Sudah ada minimal 1 Lokasi Kantor?          (/locations)
□ 2. Daftarkan Karyawan                          (/users → Daftarkan Karyawan)
       - Nama, Email, Telepon, Departemen, Kantor
       - 1 form → employee + akun login dibuat sekaligus
□ 3. Salin kredensial (email + kata sandi) → kirim ke karyawan
□ 4. Karyawan login → lengkapi NIP/jabatan/tgl masuk  (/portal/profile)
□ 5. Catat persetujuan biometrik                  (/consents)
□ 6. Enrollment wajah (service AI harus nyala)    (/portal/enrollment)
□ 7. Test login & absen pakai akun karyawan tersebut
```

---

## 4. Pembagian Field: Siapa Mengisi Apa (TIDAK ADA DUPLIKASI)

| Field | Admin saat daftar | Karyawan di portal | Catatan |
|---|---|---|---|
| `full_name` | ✅ wajib | ✏️ boleh ubah | |
| `email` (profil) | ✅ wajib | ✏️ | Sekaligus jadi email login |
| `phone` | ✅ | ✏️ | |
| `department` | ✅ | ✏️ | Teks bebas — belum ada tabel departemen |
| `location_id` (kantor) | ✅ | ✏️ | Pilih dari master Lokasi Kantor |
| `employee_number` (NIP) | ❌ | ✅ | Nullable sejak migrasi `000025` |
| `position` (jabatan) | ❌ | ✅ | Nullable |
| `join_date` | ❌ | ✅ | Nullable |
| `employment_status` | ❌ | ✅ | Nullable |
| `attendance_mode` | ❌ | ✅ | Nullable, ikut default Pengaturan |
| `password` | ✅ auto/manual | 🔒 ganti sendiri | `must_change_password` |
| `role` | otomatis `employee` | ❌ | Admin bisa ubah di /users |
| **`employee_id` (penghubung)** | otomatis dibuat | — | Dibuat oleh `RegisterEmployee` |

Kesimpulan: **admin mengisi 5 kolom saja**, sisanya diisi karyawan sendiri.
Tidak ada satupun field yang harus diketik dua kali.

---

## 5. Dropdown "Karyawan Belum Punya Akun" Sudah Dihapus

Dulu form `UserFormModal.tsx` memuat daftar karyawan dengan filter
`?has_user=false`, dan dropdown ini sering kosong sehingga membingungkan.

Sekarang dropdown itu **hanya muncul di mode edit** (opsi lanjutan
"Ubah relasi karyawan"), sedangkan pembuatan akun karyawan baru dilakukan
lewat `RegisterEmployeeModal.tsx` → `POST /api/v1/users/register-employee`.
Jadi kasus "dropdown kosong" tidak bisa terjadi lagi pada alur normal.

Jika kamu memang ingin melepas/mengganti relasi karyawan sebuah akun,
buka **Pengguna → Edit → centang "Ubah relasi karyawan untuk akun ini"**.
Membiarkan kotak itu tidak dicentang berarti `employee_id` tidak diubah sama
sekali, sehingga mengganti email tidak akan pernah melepas relasi karyawan
secara tidak sengaja.

---

## 6. Kenapa Masih Dipisah Jadi Dua Tabel?

Meskipun input-nya kini satu kali, datanya tetap disimpan di dua tabel karena:

| Alasan | Penjelasan |
|---|---|
| Karyawan **bisa** eksis tanpa akun login | mis. staff lapangan yang diabsen oleh supervisor |
| Akun **bisa** eksis tanpa karyawan | mis. admin, auditor, super admin |
| Relasi 1:1 tapi opsional | `users.employee_id` UNIQUE dan NULLABLE |
| Pemisahan domain | modul `employee` dan `user` punya permission & lifecycle sendiri |
| Audit & kepatuhan | penghapusan karyawan yang punya akun diblokir (`ON DELETE RESTRICT`) |

Yang berubah hanyalah **cara input**-nya: satu form, satu transaksi.
Pemisahan tabel tetap dipertahankan sebagai model data.


### ✅ Yang dipakai sekarang: 1 langkah (Opsin B — orchestration)

```
/users → "Daftarkan Karyawan"
   └─ POST /api/v1/users/register-employee
        └─ 1 transaksi: employees + users + user_roles
```

Implementasi:
- Backend: service `RegisterEmployee` di `internal/user/user.go` (handler
  `UserRegisterEmployee` di `internal/user/handler.go`), didaftarkan di
  `internal/httpx/router.go` dengan permission **`user.create`**, karena
  permintaan datang dari modul Pengguna.
- Frontend: `features/users/components/RegisterEmployeeModal.tsx`,
  dibuka dari `features/users/pages/UsersPage.tsx`.

Keunggulan: tidak ada dropdown yang bisa kosong, tidak ada employee "yatim",
dan admin hanya mengisi 5 kolom. Karyawan melengkapi sisanya sendiri.

### Opsi lain yang tidak dipilih

| Opsi | Cara kerja | Alasan tidak dipakai |
|---|---|---|
| **A. Tombol "Buat Akun" di detail Employee** | Dari employee → form user dengan `employee_id` ter-prefill | Tetap 2 langkah, 2 menu |
| **C. Bulk import CSV** | 1 file berisi employee + user sekaligus | Berguna nanti, tapi butuh endpoint & validasi baru |

---

## 7. Referensi Cepat: URL Penting

| Halaman | URL | Permission |
|---|---|---|
| Dashboard | `/dashboard` | banyak (lihat routes.config.ts) |
| Daftar Karyawan | `/employees` | `employee.read` |
| Tambah Karyawan | `/employees/new` | `employee.create` |
| **Daftarkan Karyawan + Akun** | **`/users` → tombol "Daftarkan Karyawan"** | **`user.create`** |
| Biometrik Karyawan | `/employees/:id/face` | `employee.read` |
| Pengguna | `/users` | `user.read` |
| Role & Izin | `/roles` | `role.read` |
| Persetujuan Biometrik | `/consents` | `face.read_any` / `consent.read` |
| Lokasi Kantor | `/locations` | `location.read` |
| Pengaturan Sistem | `/settings` | `settings.read` |
| Data Absensi | `/attendances` | `attendance.read_all` |
| Antrian Review | `/attendances/pending` | `attendance.review` |
| Audit Log | `/audit-logs` | `audit.read` |
| **Portal Karyawan** | | |
| Absen | `/portal/attendance` | (self) |
| Enrollment Wajah | `/portal/enrollment` | `face.enroll_self` |
| Riwayat Absen | `/portal/history` | (self) |
| Persetujuan | `/portal/consent` | (self) |
| Profil | `/portal/profile` | (self) |

---

## 8. Ringkasan Satu Kalimat

> **Admin → Pengguna → "Daftarkan Karyawan" (1 form, 5 kolom) → employee + akun
> login dibuat sekaligus → kredensial ditampilkan sekali → karyawan melengkapi
> NIP/jabatan dari portal → consent → enrollment wajah → siap absen.**

Satu kali input. Tidak ada field yang diketik dua kali. Tabel `employees` dan
`users` tetap terpisah hanya sebagai model data, bukan sebagai langkah kerja.
