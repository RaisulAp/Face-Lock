# Master Plan — Faceclock (Sistem Absensi Face Recognition)

> **Faceclock** — clock in/out pakai wajah.
>
> Dokumen ini adalah **peta besar** proyek. Tujuannya untuk dipegang oleh agent yang akan
> mendetailkan tiap fase. Baca bagian **Protokol Handoff** sebelum mulai fase apa pun.

---

## 1. Ringkasan Proyek

### Apa yang dibuat
Sistem absensi karyawan berbasis **face recognition** sebagai upgrade dari absensi QR/manual.
Karyawan melakukan check-in / check-out dengan verifikasi wajah secara langsung; jika wajah
cocok dengan foto referensi yang sudah terdaftar, absensi langsung sah. Jika gagal cocok,
tersedia jalur fallback (upload foto) yang statusnya **pending review** dan wajib disetujui admin.

### Tujuan
- Menghilangkan kecurangan absensi (titip absen / foto orang lain).
- Absensi yang **audit-able**: setiap record punya bukti wajah, lokasi, waktu server, dan status.
- Satu sistem yang bisa diakses lewat **web** (setting + absensi) dan **mobile** (absensi lapangan).
- Pemisahan hak akses yang jelas via **RBAC** (siapa yang bisa setting, siapa yang cuma absen).

### Non-goals (di luar scope awal)
- Continuous CCTV monitoring / deteksi wajah real-time dari banyak kamera.
- Payroll / penggajian otomatis (cukup sediakan data & export, integrasi menyusul).
- Multi-tenant / multi-perusahaan (fokus single organisasi dulu).

---

## 2. Fitur Utama

**Untuk Karyawan**
- Enrollment wajah: upload/capture **minimal 3 foto** referensi.
- Check-in & check-out dengan verifikasi wajah langsung dari kamera.
- Fallback upload foto bila verifikasi gagal (masuk status pending review).
- Riwayat absensi pribadi.

**Untuk Admin / HR**
- Manajemen karyawan (CRUD, assign role).
- Antrian approval untuk absensi berstatus pending review.
- Konfigurasi: threshold similarity, lokasi kantor & geofence, jam kerja.
- Laporan & rekap absensi (filter tanggal/karyawan/departemen, export).

**Untuk Super Admin**
- Manajemen role & permission.
- Konfigurasi sistem tingkat tinggi.

**Cross-cutting**
- Timestamp diambil dari **server**, bukan device (anti-manipulasi jam).
- **Geofencing**: validasi radius maksimal dari lokasi kantor.
- Consent form biometrik saat onboarding (kepatuhan UU PDP).

---

## 3. Tech Stack (Confirmed)

| Layer | Teknologi | Catatan |
|---|---|---|
| Backend | **Go** | REST API, orchestration, RBAC, business logic |
| Frontend Web | **React** | Panel admin + halaman absensi web |
| Mobile | **Flutter** | Absensi lapangan (dikerjakan setelah web siap) |
| Database | **PostgreSQL** | + ekstensi `pgvector` untuk simpan embedding |
| Face Inference | **Python + InsightFace** | Microservice terpisah (detection + embedding) |
| Auth | **JWT** | Access + refresh token |
| Deploy (dev) | **Docker Compose** | Go, React, Postgres, inference service |

### Kenapa inference dipisah jadi microservice Python
InsightFace (model `buffalo_s` ringan untuk CPU) hidup di ekosistem Python. Dipisah dari core Go
supaya: (1) Go tetap ringan & tidak perlu CGO/binding ribet, (2) inference bisa di-scale sendiri,
(3) kontrak antar service jelas via HTTP/gRPC. Go bertindak sebagai orchestrator.

### Naming convention (repo / service)
Pakai prefix `faceclock-` biar konsisten lintas komponen:

| Komponen | Nama |
|---|---|
| Backend Go | `faceclock-api` |
| Frontend Web | `faceclock-web` |
| Mobile | `faceclock-mobile` |
| Inference Service | `faceclock-inference` |

Untuk mobile app id / package: `com.faceclock.app` (sesuaikan bila domain final berbeda).

---

## 4. Arsitektur High-Level

```
[React Web]  [Flutter Mobile]
      \            /
       \          /
        v        v
     [Go Backend API]  <--- RBAC, auth, business logic, timestamp server, geofence
        |         |
        |         +---> [PostgreSQL + pgvector]  (users, employees, embeddings, attendance)
        |
        +---> [Python Inference Service]  (generate embedding, compare)
```

**Alur verifikasi absensi (garis besar):**
1. Client capture wajah langsung dari kamera → kirim foto + lat/long + employee_id + tipe (checkin/checkout) + note.
2. Go validasi auth, geofence, ambil embedding referensi milik employee tsb.
3. Go kirim foto ke inference service → dapat embedding.
4. Bandingkan embedding baru vs 3 embedding referensi (ambil **similarity tertinggi**, bukan rata-rata).
5. Di atas threshold → record `approved`. Di bawah threshold → tawarkan fallback → record `pending_review`.

---

## 5. Model Data (Draft — didetailkan di Fase 1)

- **users** — auth: id, email, password_hash, employee_id (FK), is_active
- **employees** — id, employee_number, name, department, position, dll
- **roles** — id, name (super_admin, admin/hr, employee)
- **permissions** — id, name
- **role_permissions** — pivot (role ↔ permission)
- **user_roles** — pivot (user ↔ role)
- **face_references** — id, employee_id, photo_url, embedding (vector), is_active, created_at
- **attendances** — id, employee_id, type, server_timestamp, lat, long, note, method (face/fallback),
  status (approved/pending/rejected), matched_similarity, photo_url, reviewed_by, reviewed_at
- **office_locations** — id, name, lat, long, radius_meter (support multi-lokasi)
- **app_settings** — threshold similarity, jam kerja, dll (configurable, tersimpan di DB)

---

## 6. RBAC — Definisi Role

| Role | Hak Akses |
|---|---|
| **Super Admin** | Semua akses; kelola role & permission; konfigurasi sistem |
| **Admin / HR** | Kelola karyawan; approve pending attendance; konfigurasi lokasi & threshold; lihat laporan |
| **Employee** | Enroll wajah sendiri; check-in/out; lihat riwayat absensi sendiri |

> Web dipakai untuk **setting + absensi**. Yang membedakan bukan platform-nya, tapi RBAC-nya.
> Endpoint & menu difilter berdasarkan permission, bukan berdasarkan web vs mobile.

---

## 7. Rincian Fase

Urutan eksekusi: **Web dulu (Fase 0–6, Go + React) sampai fully working**, baru Flutter (Fase 7).

Legenda ukuran:
- 🔴 **Besar** — agent wajib membuat plan detail per fase sebelum eksekusi.
- 🟢 **Kecil** — langsung eksekusi, tapi tetap tinggalkan dokumen hasil (lihat Protokol Handoff).

---

### Fase 0 — Fondasi & Setup 🟢
**Tujuan:** Menyiapkan kerangka proyek agar semua fase berikutnya punya pijakan yang konsisten.
**Garis besar pekerjaan:**
- Struktur repo — ✅ **sudah diputuskan: MONOREPO** (dikunci 2026-09-04, lihat
  [01-Fase0.md § 2.1](01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)),
  scaffolding project Go & React.
- Setup PostgreSQL + `pgvector`, tooling migration.
- Docker Compose untuk dev (Go, React, Postgres, inference service placeholder).
- Manajemen config/env, linter, base logging.
**Deliverable:** Proyek bisa `docker compose up`, ada healthcheck endpoint, DB terkoneksi.
**Depends on:** —

---

### Fase 1 — Auth, User & RBAC 🔴
**Tujuan:** Pondasi keamanan & identitas; tanpa ini fase lain tidak aman dikerjakan.
**Garis besar pekerjaan:**
- Finalisasi & migrasi skema: users, employees, roles, permissions, pivot tables.
- Autentikasi JWT (login, refresh, logout).
- Middleware RBAC (permission-based guard di setiap endpoint).
- CRUD user & employee, assign role.
- Seeder role default (super_admin, admin, employee) + akun awal.
**Deliverable:** API auth + RBAC berfungsi, teruji dengan minimal 3 role.
**Depends on:** Fase 0.

---

### Fase 2 — Face Inference Service 🔴
**Tujuan:** Menyediakan "otak" pengenalan wajah sebagai service independen.
**Garis besar pekerjaan:**
- Service Python + InsightFace (model `buffalo_s`).
- Endpoint `generate-embedding` (foto → vektor) & `compare` (opsional bila logika compare ditaruh di sini).
- Definisikan **kontrak API** yang dipakai Go (format request/response, error, ukuran/format foto).
- Handling kasus: wajah tidak terdeteksi, banyak wajah dalam 1 foto, foto blur.
**Deliverable:** Service jalan di Docker, Go bisa memanggilnya, kontrak terdokumentasi.
**Depends on:** Fase 0 (bisa paralel dengan Fase 1).

---

### Fase 3 — Enrollment Wajah 🔴
**Tujuan:** Karyawan bisa mendaftarkan wajah referensi (min. 3 foto).
**Garis besar pekerjaan:**
- Endpoint upload/capture foto referensi → generate embedding via Fase 2 → simpan ke `face_references`.
- Validasi: minimal 3 foto, tiap foto harus terdeteksi tepat 1 wajah.
- Simpan **3 embedding terpisah** (jangan di-average).
- Kelola foto referensi (ganti, nonaktifkan, re-enroll).
- Consent form biometrik saat pertama enroll.
**Deliverable:** Karyawan punya 3+ embedding referensi tersimpan & bisa dikelola.
**Depends on:** Fase 1, Fase 2.

---

### Fase 4 — Attendance Engine 🔴
**Tujuan:** Inti sistem — proses check-in/out dengan verifikasi wajah.
**Garis besar pekerjaan:**
- Endpoint check-in & check-out: terima foto + lat/long + tipe + note.
- Ambil embedding referensi employee → compare (ambil similarity tertinggi) vs threshold.
- **Timestamp dari server.** Validasi **geofence** terhadap `office_locations`.
- Logika status: `approved` (lolos face) / `pending_review` (lewat fallback).
- Fallback: bila gagal match, izinkan upload foto → record `pending_review`.
- Aturan bisnis: cegah double check-in, urutan checkin→checkout, dll.
**Deliverable:** Absensi end-to-end berfungsi lewat API, semua field tersimpan benar.
**Depends on:** Fase 1, Fase 2, Fase 3.

---

### Fase 5 — Admin Panel & Konfigurasi (React) 🔴
**Tujuan:** Antarmuka bagi Admin/HR & Super Admin untuk mengatur & mengawasi sistem.
**Garis besar pekerjaan:**
- UI login + guard berbasis permission.
- Manajemen karyawan & role.
- Antrian approval pending attendance (approve/reject + alasan).
- Konfigurasi: threshold, lokasi kantor + geofence (map picker), jam kerja.
- Laporan & rekap absensi (filter + export CSV/Excel).
- (Super Admin) kelola role & permission.
**Deliverable:** Panel web fungsional sesuai RBAC.
**Depends on:** Fase 1–4.

---

### Fase 6 — Halaman Absensi Web (React) 🟢
**Tujuan:** Karyawan bisa absen langsung dari web (capture kamera browser).
**Garis besar pekerjaan:**
- Halaman check-in/out dengan akses kamera (getUserMedia), disable upload dari file untuk foto harian.
- Ambil lokasi via browser geolocation.
- Halaman enrollment wajah versi web.
- Riwayat absensi pribadi.
**Deliverable:** Karyawan bisa enroll & absen penuh lewat web.
**Depends on:** Fase 3, Fase 4, Fase 5.

> ✅ **Milestone: Web fully working.** Setelah titik ini, Flutter baru dimulai.

---

### Fase 7 — Flutter Mobile 🔴 *(setelah web siap)*
**Tujuan:** Absensi lapangan yang praktis dari HP, dengan keamanan ekstra.
**Garis besar pekerjaan:**
- App Flutter konsumsi API yang sama (auth, enroll, check-in/out, riwayat).
- Capture wajah langsung dari kamera in-app (disable galeri untuk foto verifikasi).
- **Liveness ringan on-device** via `google_ml_kit` (mis. deteksi kedip/senyum) sebelum kirim ke server.
- Geolocation + note.
- Handling offline/retry ringan bila perlu.
**Deliverable:** APK/iOS build yang bisa enroll & absen, dengan liveness dasar.
**Depends on:** Milestone web fully working.

---

## 8. Peta Ketergantungan Fase

```
Fase 0 ──┬─> Fase 1 ──┬─> Fase 3 ──> Fase 4 ──> Fase 5 ──> Fase 6 ──[Web done]──> Fase 7
         └─> Fase 2 ──┘                 ^
                                        └── (Fase 2 juga dipakai di sini)
```

---

## 9. Isu Lintas-Fase yang Wajib Diingat

- **Anti-spoofing / liveness:** Web minimal paksa capture langsung dari kamera (bukan upload galeri).
  Liveness sungguhan (kedip/senyum) diprioritaskan di Fase 7 (mobile).
- **Keamanan fallback:** Jalur fallback **tidak boleh** langsung `approved`. Selalu `pending_review`.
- **Threshold configurable:** Simpan di `app_settings`, jangan hardcode — butuh tuning berkelanjutan.
- **Timestamp server-side & geofence server-side:** Jangan percaya data waktu/lokasi mentah dari client.
- **Data biometrik = data sensitif (UU PDP):** Consent saat onboarding, kebijakan retensi foto & embedding.
- **RBAC konsisten:** Filter di level API (permission), bukan sekadar sembunyikan menu di UI.

---

## 10. Protokol Handoff untuk Agent

1. **Kerjakan fase sesuai urutan ketergantungan** (Bagian 8). Web (Fase 0–6) tuntas dulu, baru Fase 7.
2. **Fase 🔴 Besar:** buat dulu dokumen `PLAN-Fase-N.md` berisi detail (skema final, daftar endpoint,
   flow, edge case, checklist) → minta review bila perlu → baru eksekusi.
3. **Fase 🟢 Kecil:** boleh langsung eksekusi tanpa plan detail.
4. **Semua fase (besar maupun kecil)** wajib meninggalkan dokumen hasil `DONE-Fase-N.md` berisi:
   apa yang dikerjakan, keputusan teknis yang diambil, cara menjalankan/menguji, dan hal yang tertunda.
5. Update peta ketergantungan / master plan ini bila ada perubahan scope.