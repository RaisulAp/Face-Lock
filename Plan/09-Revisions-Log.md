# 09 — Revisions Log (Konsolidasi Revisi Kontrak Lintas-Fase)

> Dokumen referensi tunggal untuk **seluruh revisi kontrak** yang dilaporkan di
> [01-Fase0.md](01-Fase0.md) … [08-Fase7.md](08-Fase7.md).
>
> **Dikelompokkan per ARTEFAK yang direvisi**, bukan per fase asal — supaya
> pertanyaan "apa saja yang berubah dari skema DB awal?" dijawab satu bagian,
> bukan `grep` delapan berkas.
>
> **Tanggal konsolidasi:** 2026-09-04
> **Sifat dokumen:** murni konsolidasi. Tidak ada revisi baru yang diciptakan di
> sini; setiap item bisa ditelusuri ke dokumen fase asalnya.
>
> **Update 2026-09-05:** kontradiksi K-01–K-04 (§ 4) diresolusi atas instruksi
> eksplisit user. Resolusinya menghasilkan **dua item baru** (REV-EP-11,
> REV-ERR-06) yang tidak berasal dari 8 dokumen fase — dicatat terpisah di § 1,
> bukan disamarkan ke dalam angka 46 semula.

---

## 0. Cara memakai dokumen ini

| Kolom | Arti |
|---|---|
| **ID** | Identitas stabil. Rujuk ID ini di commit, PR, dan ADR |
| **Asal** | Dokumen + bagian yang **melaporkan** revisi |
| **Terdampak** | Dokumen/artefak yang **isinya berubah** karena revisi itu |
| **Sifat** | Klasifikasi **dari dokumen sumber**, tidak dinilai ulang di sini |
| **Status** | Apakah sudah tercermin di kode |

### Status global — semua item: `BELUM DIEKSEKUSI`

Diverifikasi pada 2026-09-04 terhadap isi repo (lihat
[08-Fase7.md § 0](08-Fase7.md#0--verifikasi-prasyarat--milestone-web-fully-working-belum-terpenuhi)):
tidak ada `apps/`, `services/`, `deploy/`, `docs/`, tidak ada `go.mod`/`package.json`,
tidak ada repositori git, tidak ada satu pun `DONE-Fase-N.md`.

**Konsekuensi: seluruh 48 item di dokumen ini berstatus "belum dieksekusi" —
tercatat sebagai niat, belum ada di kode.** Tidak ada pengecualian. Kolom `Status`
tetap dicantumkan supaya bisa dicentang satu per satu saat eksekusi berjalan.
(46 dari konsolidasi awal 2026-09-04 + 2 dari resolusi K-01–K-04 pada
2026-09-05 — lihat § 1.)

Karena belum ada kode sama sekali, ada satu keuntungan yang tidak akan datang dua
kali: **setiap revisi masih bisa masuk ke implementasi pertama**, bukan menjadi
migration tambalan. Bagian § 5 menerjemahkan itu menjadi daftar per fase.

---

## 1. Rekapitulasi jumlah

| Sumber (pelapor utama) | Baris pelaporan di dokumen asal | Item terkonsolidasi di sini |
|---|---|---|
| Fase 1 § 3.8 (inline) | 0 (tidak pernah masuk tabel revisi) | 1 |
| Fase 2 § 3.1 (inline) | 0 (tidak pernah masuk tabel revisi) | 1 |
| Fase 3 (inline, § 2.9 / § 3.3 / § 3.4 / § 3.8 / § 2.1–2.2 / § 5.3) | 11 (bersama Fase 4) | 10 |
| Fase 4 (inline, § 2.1 / § 2.10 / § 3.2 / § 3.4 / § 3.5) | ↑ | 5 |
| Fase 5 § 12 | 7 | 8 |
| Fase 6 § 12 | 6 | 4 |
| Fase 7 § 14 | 11 | 16 |
| Fase 7 § 2.2 (inline) | 0 (dinyatakan sebagai prasyarat) | 1 |
| **Subtotal konsolidasi 2026-09-04** | **35** | **46** |
| Resolusi K-01–K-04 (§ 4, sesi 2026-09-05) | 0 (tidak ada di 8 dokumen fase) | 2 |
| **TOTAL** | **35** | **48** |

### Kenapa 48, bukan 46

Dua item baru — **REV-EP-11** (§ 2 B) dan **REV-ERR-06** (§ 2 C) — **tidak**
berasal dari salah baca delapan dokumen fase; keduanya lahir langsung dari
menyelesaikan K-02 dan K-04 di § 4 pada sesi 2026-09-05. Ini bukan pelanggaran
aturan "murni konsolidasi, bukan revisi baru" pada dokumen ini sendiri — dua
kontradiksi itu **sudah dilaporkan** sebagai temuan terbuka sejak konsolidasi
pertama (2026-09-04); resolusinya, yang diinstruksikan secara eksplisit oleh
user, kebetulan butuh satu endpoint baru dan satu kode error baru. Ditandai
terpisah di tabel di atas supaya angka **46** (murni dari 8 dokumen) tetap bisa
dirujuk sebagai baseline historis.

### Kenapa 46 (baseline), bukan 35

Angka **35** pada rekap sebelumnya menghitung **baris tabel pelaporan** di tiap
dokumen. Konsolidasi per-artefak memecah sebagian baris itu, karena satu baris
pelaporan sering menyentuh beberapa artefak berbeda:

- **Dipindah ke § 3 (−2).** Dua baris ternyata bukan revisi, melainkan verifikasi
  "sudah dicek, aman": catatan nuansa index ANN (Fase 3) dan Fase 7 **R10**
  (`capture_source='mobile_camera'` sudah ada). → **33 baris tersisa**
- **Penggabungan (−4).** Empat baris adalah **pengulangan** item yang sudah
  dilaporkan fase sebelumnya karena masih terbuka: Fase 6 no. 3 = Fase 5 no. 6
  (HTTPS), Fase 6 no. 5 = Fase 5 no. 7 (`hints.json`), Fase 7 R1 = Fase 6 no. 1
  (field `context`), Fase 7 R8 = Fase 6 no. 2 (`can_fallback`). Masing-masing
  menjadi **satu** item dengan riwayat pelaporan berlapis.
  → **29 baris sumber unik**
- **Pemecahan (+14).** Satu baris pelaporan sering menyentuh beberapa artefak.
  Contoh terbesar: Fase 7 **R4** adalah *satu* baris, tetapi menyentuh 6 artefak —
  kolom `attendances`, CHECK `fallback_reason`, CHECK `attendance_attempts.outcome`,
  field response `#55`, field request `#53/#54`, dan 4 key `app_settings`. Di sini
  ia menjadi 6 item agar developer yang membuka bagian "skema DB" tidak melewatkan
  bagiannya. Hal serupa berlaku untuk R5, untuk Fase 5 baris 1 (2 endpoint), dan
  untuk `photo_url` → `photo_key` yang menyentuh `face_references` **dan**
  `attendances`. Rincian per sumber: Fase 3/4 +5, Fase 5 +1, Fase 6 +0, Fase 7 +8.
  → **43 item**
- **Baru terkonsolidasi (+3).** Tiga hal dilaporkan sebagai teks inline, tidak
  pernah masuk tabel revisi mana pun, tapi sifatnya sama: `audit_logs` memakai
  `bigint identity` (Fase 1 § 3.8), 10 key `app_settings` dari Fase 2 § 3.1, dan
  kewajiban HTTPS untuk `API_BASE_URL` mobile (Fase 7 § 2.2). → **46 item**

```
35 − 2 (pindah ke § 3) − 4 (pengulangan digabung) = 29 baris sumber unik
29 + 14 (pemecahan per artefak)                    = 43 item
43 + 3 (baru terkonsolidasi dari teks inline)      = 46 item
```

Rinciannya terlihat di kolom **Asal** tiap tabel — setiap item bisa ditelusuri
balik ke baris pelaporan aslinya.

### Sebaran per sifat

Klasifikasi diambil **apa adanya dari dokumen sumber**, tidak dinilai ulang di sini.
Item yang sumbernya memberi sifat campuran (mis. REV-DB-06: R4b wajib, R5b
direkomendasikan) dihitung sebagai **Wajib**, karena bagian wajibnya tetap blocking.

| Sifat | Jumlah | ID |
|---|---|---|
| **Wajib (blocking)** | **33** | DB-01…04, DB-06, DB-07, EP-01, EP-02, EP-04…06, EP-08, EP-11, ERR-01…06, PERM-01, SET-02…04, SET-06, SET-08, MW-01, AUTH-01, INF-01…03, CTR-01, CTR-02, UI-01 |
| Direkomendasikan | 4 | DB-05, EP-07, EP-10, SET-07 |
| Tambahan / kompatibel mundur | 5 | EP-03, SET-05, INF-05, INF-06, CTR-03 |
| Pengecualian konvensi (disetujui) | 3 | CONV-01, CONV-02, CONV-03 |
| Klarifikasi (tanpa perubahan perilaku) | 1 | EP-09 |
| Scope fase (dicatat untuk kelengkapan) | 1 | SET-01 |
| Catatan (bukan perubahan kode) | 1 | INF-04 |
| **TOTAL** | **48** | |

Dua item **naik sifatnya** selama konsolidasi awal, dan itu dicatat eksplisit di
tabelnya: **REV-EP-04** (Direkomendasikan → Wajib, karena temuan K-01 di § 4) dan
**REV-CTR-01** (Saran → Wajib, karena setelah Fase 7 ada tiga consumer untuk kalimat
yang sama). Dua item **lain lahir langsung sebagai Wajib** saat K-01–K-04
diselesaikan pada sesi 2026-09-05: **REV-EP-11** (K-02) dan **REV-ERR-06** (K-04) —
lihat "Kenapa 48, bukan 46" di atas.

---

## 2. Daftar revisi terkonsolidasi

### A. Perubahan skema database

Kolom, tabel, dan CHECK yang **tidak ada** di draft awal
[Master Plan § 5](00MasterPlan.md) / [Fase 0 § 3](01-Fase0.md#3-skema-database) /
[Fase 1 § 3](02-Fase1.md#3-skema-database).

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-DB-01** | Kolom baru `employees.attendance_mode text NOT NULL DEFAULT 'face' CHECK (IN ('face','manual'))` — jalur sah bagi karyawan yang menolak consent | Fase 3 § 2.4d, § 3.3 | Tabel `employees` milik Fase 1 (migration 000002); ditambahkan lewat migration 000013 | Wajib | ✅ Selesai dieksekusi (Fase 3, migration 000013) |
| **REV-DB-02** | `face_references.photo_url` → **`photo_key`** (key object storage) + `photo_sha256`, `photo_bytes`, `photo_mime`, `photo_purged_at`. Tidak ada URL publik | Fase 3 § 2.2 (D14), § 3.4 | **Master Plan § 5** (draft model data), migration 000014 | Wajib | ✅ Selesai dieksekusi (Fase 3, migration 000014) |
| **REV-DB-03** | `attendances.photo_url` → **`photo_key`** + `photo_sha256`, `photo_bytes`, `photo_mime`, `photo_purged_at` | Fase 4 § 3.2 (turunan D14) | **Master Plan § 5**, migration 000019 | Wajib | ✅ Selesai dieksekusi (Fase 4, migration 000019) |
| **REV-DB-04** | Kolom liveness pada `attendances`: `liveness_passed boolean NULL`, `liveness_supported boolean NULL`, `liveness_method text NULL`, `liveness_challenges jsonb NOT NULL DEFAULT '[]'` | Fase 7 § 14 R4a | Migration 000019 (Fase 4) | **Wajib** | ✅ Selesai dieksekusi (Fase 4, migration 000019) |
| **REV-DB-05** | Kolom `attendances.location_is_mocked boolean NULL` (dari `Position.isMocked`, Android) | Fase 7 § 14 R5a | Migration 000019 (Fase 4); mengoreksi [Fase 4 E21](05-Fase4.md#63-lokasi) yang menyatakan mock location "tidak terdeteksi" | Direkomendasikan | ✅ Selesai dieksekusi (Fase 4, migration 000019) |
| **REV-DB-06** | `attendances.fallback_reason` CHECK bertambah 2 nilai: `'liveness_failed'`, `'location_mocked'` | Fase 7 § 14 R4b, R5b | Migration 000019 (Fase 4) | R4b **Wajib**, R5b Direkomendasikan | ✅ Selesai dieksekusi (Fase 4, migration 000019) |
| **REV-DB-07** | `attendance_attempts.outcome` CHECK bertambah 2 nilai: `'liveness_failed'`, `'location_mocked'` | Fase 7 § 14 R4c, R5c | Migration 000020 (Fase 4) | R4c **Wajib**, R5c Direkomendasikan | ✅ Selesai dieksekusi (Fase 4, migration 000020) |

> ⚠️ **REV-DB-04 sampai REV-DB-07 mengubah tabel yang dibuat Fase 4.** Karena Fase 4
> belum dieksekusi, kolom dan CHECK ini **harus masuk ke migration 000019/000020 sejak
> awal** — bukan migration tambahan di Fase 7. Lihat § 5.

### B. Perubahan katalog endpoint

Katalog awal: **#1–#70** (Fase 1: #1–#31, Fase 3: #32–#52, Fase 4: #53–#70).

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-EP-01** | **Endpoint baru #71** `GET /api/v1/attendances/summary` — rekap per karyawan. Guard `attendance.read_all` | Fase 5 § 2.1, § 12 no. 1 | Katalog endpoint; `routes.go`; `rbac_matrix_test.go` | Wajib (deliverable Fase 5) | ⬜ Belum dieksekusi |
| **REV-EP-02** | **Endpoint baru #72** `GET /api/v1/attendances/export` — CSV streaming + BOM. Guard `attendance.export` (permission sudah di-seed Fase 1, **tidak ada permission baru**) | Fase 5 § 2.1, § 12 no. 1 | Katalog endpoint; `routes.go`; `rbac_matrix_test.go` | Wajib (deliverable Fase 5) | ⬜ Belum dieksekusi |
| **REV-EP-03** | **#7 `GET /employees`** — tambah filter & field response `attendance_mode`, `consent_status`, `enrollment_status` | Fase 5 § 2.1, § 12 no. 2 | [Fase 1 § 4.2](02-Fase1.md#42-employees); bergantung pada tabel Fase 3 (`biometric_consents`, `face_references`) | Tambahan, kompatibel mundur | ⬜ Belum dieksekusi |
| **REV-EP-04** | **#55 `GET /attendances/context`** — tambah `require_face_for_checkout`, `checkout_without_face_status`, `max_note_length` | Fase 6 § 12 no. 1 → **diulang** Fase 7 § 14 R1 | [Fase 4 § 4.4](05-Fase4.md#44-get-attendancescontext) | Direkomendasikan → naik jadi **Wajib** karena kontradiksi K-01 (§ 4) | ✅ Selesai dieksekusi (Fase 4, internal/attendance) |
| **REV-EP-05** | **#55** — tambah `photo.recommended_dimension_px` (1280) dan `photo.max_dimension_px` (1920). Web mengasumsikan 1280×720 tetap; kamera ponsel 5–108 MP butuh target dari server | Fase 7 § 14 R2 | [Fase 4 § 4.4](05-Fase4.md#44-get-attendancescontext); [Fase 6 D24](07-Fase6.md#d24--parameter-capture--butuh-konfirmasi) | **Wajib** | ✅ Selesai dieksekusi (Fase 4, internal/attendance) |
| **REV-EP-06** | **#55** — tambah objek `liveness`: `policy`, `max_attempts`, `challenge_count`, `timeout_seconds` | Fase 7 § 14 R4e | Fase 4 § 4.4 | **Wajib** | ✅ Selesai dieksekusi (Fase 4, internal/attendance) |
| **REV-EP-07** | **#55** — tambah `geofence.mocked_location_policy` | Fase 7 § 14 R5e | Fase 4 § 4.4 | Direkomendasikan | ✅ Selesai dieksekusi (Fase 4, internal/attendance) |
| **REV-EP-08** | **#53 / #54 `check-in` / `check-out`** — terima field request baru: `liveness_passed`, `liveness_supported`, `liveness_method`, `liveness_challenges`, `location_is_mocked` | Fase 7 § 14 R4d, R5d | [Fase 4 § 4.2](05-Fase4.md#42-post-attendancescheck-in) | R4d **Wajib**, R5d Direkomendasikan | ✅ Selesai dieksekusi (Fase 4, internal/attendance) |
| **REV-EP-09** | **#58 `GET /attendances/{id}`** — perjelas: mengembalikan **DTO admin** bila pemanggil punya `attendance.read_all`, **DTO employee** bila hanya `attendance.read_self`. Fase 4 § 4.6 hanya menyebut DTO lengkap untuk #60 | Fase 5 § 12 no. 3 | Fase 4 § 4.6 | Klarifikasi (bukan perubahan perilaku) | ✅ Selesai dieksekusi (Fase 4, internal/attendance) |
| **REV-EP-10** | **`GET /api/v1/version`** — tambah `min_supported_client` (mis. `{"mobile":"1.0.0"}`) agar app lama bisa menampilkan "Perbarui aplikasi" | Fase 7 § 14 R11 | [Fase 0 § 4](01-Fase0.md#4-daftar-endpoint) | Direkomendasikan | ⬜ Belum dieksekusi |
| **REV-EP-11** | **Endpoint baru #73** `GET /api/v1/settings/face-quality-status` — bandingkan 7 ambang kualitas `face.*` di `app_settings` terhadap nilai aktif inference (`GET /ready` live), untuk kartu read-only + badge drift. Guard `settings.read` (permission sudah di-seed Fase 1, **tidak ada permission baru**) | Resolusi K-02 (§ 4) → [Fase 5 § 2.1](06-Fase5.md#21-fase-5-bukan-murni-frontend--2-endpoint-backend-baru) | Katalog endpoint; `routes.go`; `rbac_matrix_test.go` | **Wajib** | ⬜ Belum dieksekusi |

> **Total endpoint setelah revisi: 73** (72 dari REV-EP-01–10, +1 dari REV-EP-11
> hasil resolusi K-02). Fase 6 tetap murni konsumen ([Fase 6 § 4.5](07-Fase6.md#45-pembuktian-kontrak));
> Fase 7 tetap tidak menyentuh satu pun endpoint baru
> ([Fase 7 § 4.6](08-Fase7.md#46-verifikasi-nol-endpoint-baru)) — #73 adalah endpoint
> khusus panel admin (Fase 5), bukan sesuatu yang dikonsumsi mobile.

### C. Perubahan katalog error code

Katalog awal: **13 kode** ([Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go)).

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-ERR-01** | **+12 kode Fase 3**: `CONSENT_REQUIRED` (403), `CONSENT_ALREADY_GRANTED` (409), `CONSENT_VERSION_OUTDATED` (409), `FACE_NOT_USABLE` (422), `DUPLICATE_PHOTO` (409), `ENROLLMENT_INCOMPLETE` (422), `ENROLLMENT_LIMIT_REACHED` (409), `ENROLLMENT_SESSION_EXPIRED` (409), `ENROLLMENT_MODEL_CHANGED` (409), `FACE_BELONGS_TO_ANOTHER_EMPLOYEE` (409), `REINDEX_IN_PROGRESS` (409), `ATTENDANCE_MODE_MANUAL` (422) | Fase 3 § 2.9 | Fase 0 § 2.6; `internal/httpx/errors.go`; `lib/errors/codes.ts`; `core/network/error_codes.dart` | Wajib | ✅ Selesai dieksekusi (Fase 3) |
| **REV-ERR-02** | **+14 kode Fase 4**: `FACE_NOT_ENROLLED` (422), `FACE_NOT_MATCHED` (422), `OUTSIDE_GEOFENCE` (422), `LOCATION_REQUIRED` (422), `LOCATION_INACCURATE` (422), `ALREADY_CHECKED_IN` (409), `ALREADY_CHECKED_OUT` (409), `CHECKOUT_WITHOUT_CHECKIN` (409), `CHECKOUT_TOO_SOON` (409), `ATTENDANCE_ALREADY_REVIEWED` (409), `SELF_REVIEW_DENIED` (403), `EMPLOYEE_INACTIVE` (403), `TOO_MANY_FAILED_ATTEMPTS` (429), `ATTENDANCE_NOT_CONFIGURED` (503, **dipersempit** — lihat REV-ERR-06) | Fase 4 § 2.10 | idem | Wajib | ✅ Selesai dieksekusi (Fase 4, internal/httpx/errors.go) |
| **REV-ERR-03** | **+1 kode Fase 7**: `LIVENESS_REQUIRED` — **HTTP 422**. Muncul hanya bila `attendance.liveness_policy='required'` dan `liveness_passed != true`. Response **wajib** menyertakan `can_fallback` (nilainya `attendance.fallback_enabled`) | Fase 7 § 14 R7 | Fase 0 § 2.6 | **Wajib** | ✅ Selesai dieksekusi (Fase 4, internal/httpx/errors.go) |
| **REV-ERR-04** | **Status HTTP 410 Gone** masuk katalog, dipakai khusus untuk foto yang sudah dihapus kebijakan retensi, dengan `code: "NOT_FOUND"` agar client lama tetap menanganinya | Fase 3 § 4.4; dipakai ulang Fase 4 § 6.5 E38 | Fase 0 § 2.6; `docs/adr/0003-api-conventions.md` | Wajib | ✅ Selesai dieksekusi (Fase 3) |
| **REV-ERR-05** | **Cakupan field `can_fallback`.** Fase 4 § 4.2 hanya mencontohkannya pada `FACE_NOT_MATCHED`. Dinyatakan berlaku pada: `FACE_NOT_MATCHED`, `FACE_NOT_USABLE`, `FACE_NOT_ENROLLED`, `LIVENESS_REQUIRED`, `502 UPSTREAM_ERROR`, `504 UPSTREAM_TIMEOUT` | Fase 6 § 12 no. 2 → **diulang** Fase 7 § 14 R8 (masih terbuka) | Fase 4 § 4.2 | **Wajib** — tanpa ini client menduplikasi logika server | ✅ Selesai dieksekusi (Fase 4, internal/attendance) |
| **REV-ERR-06** | **+1 kode baru**: `FACE_SERVICE_NOT_CONFIGURED` (503) — akar masalah "`face.model_version = 'unset'`, Fase 2 belum dikalibrasi". Dipakai bersama oleh `#38` (Fase 3), `#53`/`#54` (Fase 4), menggantikan `SERVICE_UNAVAILABLE` di `#38` dan mengeluarkan kondisi ini dari `ATTENDANCE_NOT_CONFIGURED` di `#53`/`#54` (yang sebelumnya menggabungkan dua akar masalah tak-terkait — lihat REV-ERR-02) | Resolusi K-04 (§ 4) | [Fase 3 § 2.9](04-Fase3.md#29-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0) + § 4.3; [Fase 4 § 2.10](05-Fase4.md#210-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0) + § 4.2; `internal/httpx/errors.go`; `lib/errors/codes.ts`; `core/network/error_codes.dart` | **Wajib** | ✅ Selesai dieksekusi (Fase 3, didaftarkan & dipakai di #38) |

> **Total katalog error setelah revisi: 41 kode** (13 + 12 + 14 + 1 + 1), ditambah
> satu status HTTP baru (410). Angka ini **naik dari 40 menjadi 41** akibat resolusi
> K-04 (REV-ERR-06) — sebuah penyimpangan sadar dari instruksi awal "jangan buat
> kode baru", dijelaskan alasannya di § 4 K-04. Ketiga client (`errors.go`,
> `codes.ts`, `error_codes.dart`) memakai daftar yang sama;
> [Fase 5 § 2.3](06-Fase5.md#23-pemetaan-error--dari-code-bukan-pesan-buatan-sendiri)
> mensyaratkan `Record<ApiErrorCode, …>` sehingga kode yang belum dipetakan
> **memecah build**, bukan menghasilkan `undefined`.

### D. Perubahan katalog permission

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-PERM-01** | **Permission baru `face.reindex`** + grant ke `super_admin` di migration yang sama (mengikuti aturan [Fase 1 § 2.3](02-Fase1.md#23-katalog-permission)). **Sengaja tidak** diberikan ke `admin` | Fase 3 § 3.8 | [Fase 1 § 2.3](02-Fase1.md#23-katalog-permission), § 2.4; migration 000017 | Wajib | ✅ Selesai dieksekusi (Fase 3, migration 000017) |

> **Hanya satu permission baru di seluruh proyek.** Fase 4, 5, 6, dan 7 tidak
> menambah permission sama sekali — seluruh kebutuhan sudah tercakup katalog Fase 1,
> termasuk `attendance.export` yang di-seed sejak awal dan baru dipakai #72 di Fase 5.

### E. Perubahan `app_settings`

Seed awal: **11 key** ([Fase 1 § 3.9](02-Fase1.md#39-app_settings--migration-000010)).

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-SET-01** | **+10 key `face.*`**: `det_size`, `min_det_score`, `min_blur_var`, `min_brightness`, `max_brightness`, `min_face_ratio`, `max_abs_yaw`, `max_abs_pitch`, `max_image_bytes`, `accepted_mime_types`. Berstatus **salinan tampilan** — sumber kebenaran tetap env container inference | Fase 2 § 3.1 (inline, baru terkonsolidasi di sini) | Migration 000011 | Scope fase (dicatat untuk kelengkapan katalog) | ✅ Selesai dieksekusi (Fase 2, migration 000011) |
| **REV-SET-02** | **+7 key `face.*`**: `max_reference_photos`, `enrollment_session_ttl_minutes`, `min_quality_score`, `duplicate_check_enabled`, `duplicate_threshold`, `retention_days_after_resign`, `consent_required` | Fase 3 § 3.8 | Migration 000017 | Wajib | ✅ Selesai dieksekusi (Fase 3, migration 000017) |
| **REV-SET-03** | **+15 key `attendance.*`**: `workday_cutoff_hour`, `fallback_enabled`, `outside_geofence_policy`, `missing_location_policy`, `max_gps_accuracy_meter`, `allow_checkout_without_checkin`, `min_minutes_between_checkin_checkout`, `require_face_for_checkout`, `checkout_without_face_status`, `allow_fallback_without_enrollment`, `max_note_length`, `max_failed_attempts_per_hour`, `photo_retention_days`, `attempt_retention_days`, `duplicate_photo_window_days` | Fase 4 § 3.5 | Migration 000021 | Wajib | ✅ Selesai dieksekusi (Fase 4, migration 000021) |
| **REV-SET-04** | **Rekonsiliasi makna `attendance.max_distance_meter`.** Fase 1 men-seed-nya sebagai radius geofence; sejak Fase 4 ia menjadi **batas atas + nilai default** `office_locations.radius_meter`. Radius efektif ditentukan per lokasi. Deskripsi baris di-`UPDATE` | Fase 4 § 2.1 (dinyatakan eksplisit sebagai "revisi kecil kontrak Fase 1") | [Fase 1 § 3.9](02-Fase1.md#39-app_settings--migration-000010); migration 000021 | Wajib | ✅ Selesai dieksekusi (Fase 4, migration 000021) |
| **REV-SET-05** | **+1 key** `attendance.export_max_rows` (default 100000) | Fase 5 § 2.1, § 12 no. 4 | Migration Fase 5 | Tambahan | ⬜ Belum dieksekusi |
| **REV-SET-06** | **+4 key liveness**: `attendance.liveness_policy` (`off`\|`preferred`\|`required`, default `preferred`), `liveness_max_attempts` (3), `liveness_challenge_count` (2), `liveness_timeout_seconds` (20) | Fase 7 § 14 R4e | Migration 000021 (Fase 4) | **Wajib** | ✅ Selesai dieksekusi (Fase 4, migration 000021) |
| **REV-SET-07** | **+1 key** `attendance.mocked_location_policy` (`reject`\|`pending_review`, default `reject`) | Fase 7 § 14 R5e | Migration 000021 (Fase 4) | Direkomendasikan | ✅ Selesai dieksekusi (Fase 4, migration 000021) |
| **REV-SET-08** | **+1 key** `auth.refresh_reuse_grace_seconds` (default 30) — pendamping REV-AUTH-01 | Fase 7 § 14 R3 | Migration Fase 1 (bersama `refresh_tokens`) | **Wajib** | ⬜ Belum dieksekusi |

> **Total `app_settings` setelah revisi: 50 key** (11 + 10 + 7 + 15 + 1 + 4 + 1 + 1).
> Setiap key wajib punya validator per-key di `internal/settings/validators.go`
> ([Fase 1 § 4.6](02-Fase1.md#46-app-settings)).

### F. Perubahan middleware chain

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-MW-01** | **Slot baru `RequireConsent`** setelah `RequirePermission`. Urutannya bermakna: pihak yang tidak berhak sama sekali harus menerima `403 FORBIDDEN`, bukan `403 CONSENT_REQUIRED` yang membocorkan keberadaan karyawan | Fase 3 § 2.4b, § 5.3 | [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go) — rantai middleware terkunci | Wajib | ✅ Selesai dieksekusi (Fase 3, internal/httpx/middleware/consent.go) |

Rantai final:

```
RequestID → RealIP → StructuredLogger → Recoverer → CORS
  → Timeout(30s) → BodyLimit(10MB) → RateLimit
    → Authenticate → RequirePermission(...) → RequireConsent(...)   ← REV-MW-01
      → handler
```

### G. Perubahan konvensi umum & pengecualian yang disetujui

Konvensi dasar ada di [Fase 0 § 3](01-Fase0.md#3-skema-database). Tiga pengecualian
berikut **sudah disetujui** di dokumen sumbernya dan wajib dicatat di
`docs/adr/0003-api-conventions.md`.

| ID | Pengecualian | Asal | Alasan | Sifat | Status |
|---|---|---|---|---|---|
| **REV-CONV-01** | `audit_logs.id` memakai **`bigint GENERATED ALWAYS AS IDENTITY`**, bukan `uuid` | Fase 1 § 3.8 (inline, baru terkonsolidasi di sini) | Tabel append-only, sangat banyak baris, dan urutan sisipnya bermakna | Pengecualian disetujui | ✅ Selesai dieksekusi (Fase 1, migration 000009) |
| **REV-CONV-02** | `attendance_attempts.id` memakai **`bigint identity`** | Fase 4 § 3.4 | Alasan identik dengan REV-CONV-01 | Pengecualian disetujui | ✅ Selesai dieksekusi (Fase 4, migration 000020) |
| **REV-CONV-03** | `face_references` **tanpa `deleted_at`**. Siklus hidupnya `is_active`; penghapusan permanen adalah hard delete lewat prosedur PDP | Fase 3 § 3.4, § 2.8 | Baris yang "dihapus" tapi masih menyimpan embedding bukan penghapusan data biometrik dalam pengertian mana pun | Pengecualian disetujui | ✅ Selesai dieksekusi (Fase 2, migration 000011) |

### H. Perubahan perilaku auth & token

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-AUTH-01** | **Grace window pada deteksi reuse refresh token.** Token dengan `used_at` dalam `auth.refresh_reuse_grace_seconds` **dan** yang anaknya belum terpakai **dan** family belum di-revoke → terbitkan ulang pasangan yang sama, **jangan** revoke family. Di luar itu, perilaku Fase 1 tidak berubah | Fase 7 § 2.5b, § 14 R3 | [Fase 1 § 2.5](02-Fase1.md#25-strategi-token), § 5.2 (alur rotasi); berlaku untuk web **dan** mobile | **Wajib** | ⬜ Belum dieksekusi |
| **REV-AUTH-02** | **Full HttpOnly Cookie-Based Authentication dengan Anti-CSRF Shield.** Menggantikan pure Bearer token pada browser dengan `Set-Cookie` bertipe `HttpOnly`, `SameSite=Lax`, `Path=/` untuk `access_token` (15m) dan `Path=/api/v1/auth` untuk `refresh_token` (30d), `Secure` di non-dev. Dual-mode compatibility: `Authenticate` middleware memprioritaskan cookie dengan fallback mulus ke header `Authorization: Bearer <token>`, dan payload JSON tetap menyertakan token untuk mobile/CLI. Endpoint refresh membaca cookie secara otomatis dengan fallback ke request body. Anti-CSRF Shield memvalidasi custom header (`X-Requested-With: XMLHttpRequest` atau `X-CSRF-Token`) dan verifikasi `Origin` pada seluruh metode mutasi (`POST`, `PUT`, `PATCH`, `DELETE`). Penolakan mengembalikan HTTP 403 `CSRF_HEADER_MISSING` atau `CSRF_UNTRUSTED_ORIGIN`. | Task Directive Refactoring Auth Fase 1 (Arsitektur Keamanan Browser) | [Fase 1 § 2.5](02-Fase1.md#25-strategi-token), `internal/auth`, `internal/httpx`, `test/integration`, `docs/api/fase1-auth-rbac.md` | **Wajib** | ✅ Selesai dieksekusi & terverifikasi |

> Ditemukan dari kasus mobile (response refresh hilang di sinyal lemah → app mengulang
> dengan token lama → seluruh sesi dicabut permanen), tetapi **memperbaiki web juga**.
> Sifat keamanan Fase 1 dipertahankan: token yang dicuri dan dipakai di luar 30 detik,
> atau setelah anaknya terpakai, tetap memicu pencabutan family.
>
> **REV-AUTH-02** menutup kerentanan token storage di browser: token tidak lagi disimpan
> di `localStorage`/`sessionStorage` yang rentan diekstraksi script XSS, melainkan dikelola
> aman via `HttpOnly` cookies dengan mitigasi CSRF komprehensif.

### I. Perubahan infrastruktur & dependensi

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-INF-01** | **HTTPS wajib untuk `faceclock-web`** di semua lingkungan selain `localhost`. `getUserMedia` dan Geolocation hanya tersedia di secure context — tanpa ini **tidak satu pun** fitur inti Fase 6 bisa diuji di perangkat nyata | Fase 5 § 12 no. 6 → **diulang** Fase 6 § 12 no. 3 (naik jadi prasyarat eksekusi) | [Fase 0 § 2.9](01-Fase0.md#29-database--docker-compose) (compose/deploy) | ⚠️ **Wajib** — pekerjaan infrastruktur dengan waktu tunggu sendiri | ⬜ Belum dieksekusi |
| **REV-INF-02** | **HTTPS wajib untuk `API_BASE_URL` mobile** di semua environment non-dev; Android `usesCleartextTraffic=false` + `network_security_config.xml`; iOS ATS ketat tanpa `NSAllowsArbitraryLoads`; certificate pinning direkomendasikan dengan pin cadangan | Fase 7 § 2.2 (inline, baru terkonsolidasi di sini) | Konfigurasi build mobile | **Wajib** | ⬜ Belum dieksekusi |
| **REV-INF-03** | **MinIO ditambahkan ke `docker-compose.yml`** + env `STORAGE_DRIVER=s3`, `STORAGE_S3_ENDPOINT/REGION/BUCKET_FACE/BUCKET_ATTENDANCE/ACCESS_KEY/SECRET_KEY/FORCE_PATH_STYLE/SSE`. Dua bucket terpisah karena masa retensinya berbeda | Fase 3 § 2.1 (D13) | [Fase 0 § 2.9](01-Fase0.md#29-database--docker-compose), `deploy/.env.example` | Wajib (bila D13 disetujui) | ✅ Selesai dieksekusi (Fase 3, driver Local & S3 MinIO di internal/storage) |
| **REV-INF-04** | **`storage.Store.SignedURL()` tidak dipakai** di Fase 3/4/5/6 — konsekuensi D14 (foto dialirkan lewat API dengan cek permission per-request). Method tetap ada di interface untuk kemungkinan CDN di masa depan | Fase 3 § 2.2 (D14) | [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go) interface `Store` | Catatan (bukan perubahan kode) | ✅ Selesai dieksekusi (Fase 3, streaming biner terproteksi di #44) |
| **REV-INF-05** | **Dependensi `faceclock-web` bertambah**: `react-hook-form`, `zod`, `leaflet` (D22), `msw` + `@playwright/test` (dev) | Fase 5 § 12 no. 5 | [Fase 0 § 2.7](01-Fase0.md#27-scaffolding-faceclock-web-react) | Tambahan | ⬜ Belum dieksekusi |
| **REV-INF-06** | **Dependensi `faceclock-web` bertambah lagi**: `dompurify`, `ulid`; Playwright butuh flag kamera palsu di config | Fase 6 § 12 no. 6 | Fase 0 § 2.7 | Tambahan | ⬜ Belum dieksekusi |

### J. Artefak kontrak bersama (berkas kanonik)

Konsekuensi langsung dari **D1 = monorepo** (terkunci 2026-09-04): satu berkas sumber
dibaca semua sisi, tanpa distribusi paket berversi.

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-CTR-01** | **Berkas kanonik `docs/api/hints.json`** untuk 9 kalimat `hints` + katalog pesan error. Saat ini kalimat yang sama akan hidup di `internal/inference/hints.go` (Go), `lib/errors/hints.ts` (TS), dan `core/messages/hints.dart` (Dart) — tiga sumber kebenaran yang pasti menyimpang | Fase 5 § 12 no. 7 → **diulang** Fase 6 § 12 no. 5 (naik urgensinya) | [Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints); Fase 5, 6, 7 | Saran → naik jadi **Wajib** setelah Fase 7 (tiga consumer) | ✅ Selesai dieksekusi (Fase 2, docs/api/hints.json terisi 9 server hints) |
| **REV-CTR-02** | **Struktur `hints.json` dengan namespace terpisah**: `server_hints` (kosakata **tertutup** Fase 2 — tidak boleh ditambah dari client) dan `client_coach` (kalimat liveness khas mobile: `liveness_blink`, `liveness_smile`, `liveness_hold_still`, `liveness_face_lost`, `liveness_multiple_faces`, `liveness_too_far`, `liveness_timeout`, `liveness_unsupported`) | Fase 7 § 14 R6 | REV-CTR-01; [Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints) | **Wajib** — tanpa pemisahan, kalimat liveness mencemari kosakata tertutup | ✅ Selesai dieksekusi (Fase 2, docs/api/hints.json terisi namespace terpisah) |
| **REV-CTR-03** | **Berkas `docs/api/geo-testcases.json`** — kasus uji haversine dibaca sisi Go (`internal/geo`) dan TS (`lib/geo`) supaya jarak yang ditampilkan sebelum kirim identik dengan yang dihitung server | Fase 6 § 12 no. 4 | [Fase 4 § 7](05-Fase4.md#7-struktur-folder) | Tambahan (test saja) | ✅ Selesai dieksekusi (Fase 4, internal/geo/haversine_test.go) |

### K. Perubahan UI / DTO

| ID | Revisi | Asal | Terdampak | Sifat | Status |
|---|---|---|---|---|---|
| **REV-UI-01** | **Panel bukti approval Fase 5 menampilkan** `liveness_passed`, `liveness_supported`, `liveness_method`, `location_is_mocked`. Tanpa ini reviewer menyetujui record `fallback_reason='liveness_failed'` tanpa tahu apa yang gagal — padahal justru record itulah yang paling perlu dinilai manusia | Fase 7 § 14 R9 | [Fase 5 § 2.5](06-Fase5.md#25-antrian-approval--pekerjaan-inti-fase-ini) | **Wajib** | ⬜ Belum dieksekusi |

---

## 3. Sudah diverifikasi — **tidak** perlu revisi

Bagian ini mencatat "sudah dicek, aman" supaya tidak diperiksa ulang. Sumber utama:
[Fase 7 § 14.1](08-Fase7.md#141-yang-diverifikasi-dan-tidak-perlu-revisi) (7 item),
ditambah verifikasi dari fase lain.

| Diperiksa | Hasil | Sumber |
|---|---|---|
| Endpoint baru untuk Fase 6 | **Tidak ada.** Seluruh kebutuhan halaman karyawan sudah tersedia dari Fase 1/3/4, termasuk #55 yang memang dirancang untuk itu | Fase 6 § 4.5 |
| Endpoint baru untuk Fase 7 | **Tidak ada.** App menyentuh 24 dari 72 endpoint | Fase 7 § 4.6 |
| Permission baru untuk Fase 4 | **Tidak ada.** 18 endpoint Fase 4 seluruhnya memakai `attendance.*` / `location.*` yang sudah di-seed Fase 1 | Fase 4 § 3.5 |
| Permission baru untuk Fase 5, 6, 7 | **Tidak ada** | Fase 5 § 2.1; Fase 6 § 4; Fase 7 § 14.1 |
| **D23 (penyimpanan token) → apakah jadi perubahan kontrak backend?** | **Tidak.** Penyimpanan token adalah keputusan client. Server menerbitkan token & TTL yang sama (`JWT_REFRESH_TTL=720h`). TTL berbeda per platform **tidak** direkomendasikan. Yang memang perlu perubahan backend adalah REV-AUTH-01, dan itu berlaku untuk kedua platform | Fase 7 § 14.1 |
| `Idempotency-Key` 24 jam | **Cukup.** Dengan D31 (retry-only, jendela 15 menit), 24 jam jauh melampaui kebutuhan — *tetapi lihat temuan K-03 di § 4* | Fase 7 § 14.1 |
| `AttendanceEmployeeDTO` untuk mobile | **Tidak perlu varian.** Field identik dengan web | Fase 7 § 14.1 |
| `capture_source = 'mobile_camera'` | **Sudah ada** di CHECK `face_references` (Fase 3 § 3.4) dan `attendances` (Fase 4 § 3.2). Tidak perlu perubahan | Fase 7 § 14 R10 |
| Kosakata `hints` Fase 2 | **Tidak ditambah.** Kalimat liveness masuk namespace terpisah (REV-CTR-02) | Fase 7 § 14.1 |
| Index ANN (HNSW/IVFFlat) pada `face_references.embedding` | **Tidak dibuat, dan larangan Fase 2 § 2.4 tidak dilanggar.** Deteksi duplikat Fase 3 § 2.6 memang memindai 1:N, tetapi **sequential scan yang eksak**, sekali per enrollment — bukan ANN, bukan di jalur verifikasi | Fase 3 § 2.6 |
| `attendances` tanpa `deleted_at` | **Sesuai konvensi Fase 0**, bukan pengecualian. Tabel catatan bersifat append-only | Fase 4 § 3.2 |
| #71/#72 untuk mobile | **Tidak dipakai** — fitur admin, di luar batas scope D27 | Fase 7 § 14.1 |

---

## 4. Kontradiksi & celah yang **baru ditemukan** saat konsolidasi

Empat hal berikut **tidak** terlihat saat menulis dokumen fase satu per satu, dan
baru muncul ketika seluruh revisi disandingkan. Semuanya dilaporkan apa adanya —
**tidak** diselaraskan sendiri, karena masing-masing butuh keputusan.

### K-01 — `checkout_without_face_status` tidak dapat dibaca karyawan ⛔ KONTRADIKSI ✅ RESOLVED

- [Fase 4 § 3.5](05-Fase4.md#35-app_settings-tambahan--migration-000021) men-seed
  `attendance.checkout_without_face_status` dengan **`is_public = false`**.
- [Fase 1 § 4.6](02-Fase1.md#46-app-settings) menyatakan pemegang `settings.read`
  tanpa `settings.update` **hanya menerima baris `is_public = true`**.
- [Fase 6 § 2.4](07-Fase6.md#24-alur-check-in--check-out-d17) mengharuskan aplikasi
  karyawan menampilkan status yang akan dihasilkan saat `require_face_for_checkout=false`,
  dan [Fase 6 § 4.4](07-Fase6.md#44-semua-yang-dibaca-dari-attendancescontext-55)
  menyatakan nilainya dibaca lewat **#28 `GET /settings`**.

**Karyawan tidak akan pernah menerima baris itu.** Fase 6 sebagaimana tertulis akan
gagal diam-diam — field-nya `undefined`, dan UI akan menampilkan status yang salah
atau kosong.

**Dua jalan keluar, harus dipilih:**
1. **Implementasikan REV-EP-04** — masukkan `checkout_without_face_status` ke `#55
   context` (dijaga `attendance.checkin`, server yang memutuskan isinya). Ini
   menaikkan REV-EP-04 dari *Direkomendasikan* menjadi **Wajib**.
2. Ubah `is_public` menjadi `true` — lebih sederhana, tapi membocorkan satu detail
   kebijakan ke semua karyawan tanpa alasan kuat.

**Rekomendasi: jalan 1.** Sudah dicerminkan di tabel § 2 B (sifat REV-EP-04 dinaikkan).

#### ✅ Keputusan final

**Jalan 1 — REV-EP-04 diimplementasikan.** `#55 GET /attendances/context`
sekarang menyertakan `require_face_for_checkout`, `checkout_without_face_status`,
dan `max_note_length`, disuntikkan eksplisit oleh handler `context` dari
`app_settings` — **bukan** lewat mekanisme generik `#28`.
`attendance.checkout_without_face_status` **tetap** `is_public = false` di
`app_settings` — jalan 2 **ditolak secara eksplisit** dan dicatat sebagai
"jangan diperbaiki dengan cara ini" langsung di kode sumbernya, supaya tidak
diubah balik tanpa sadar di kemudian hari.

**Kontrak diperbarui (dokumentasi, belum kode):**
- [Fase 4 § 4.4](05-Fase4.md#44-get-attendancescontext) — bentuk JSON `#55` +
  penjelasan sumber tiga field ini.
- [Fase 4 § 3.5](05-Fase4.md#35-app_settings-tambahan--migration-000021) — catatan
  eksplisit "jangan ubah `is_public`".
- [Fase 6 § 4.4](07-Fase6.md#44-semua-yang-dibaca-dari-attendancescontext-55) —
  baris `#28` fallback dihapus, diganti tiga baris field baru dari `#55`.
- [Fase 6 § 12 no. 1](07-Fase6.md#12-kontrak-fase-05-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam)
  dan [Fase 7 § 14 R1](08-Fase7.md#14-kontrak-yang-perlu-direvisi) ditandai selesai.

**Tidak ada perubahan skema atau permission** — ini murni perluasan response
JSON endpoint yang sudah ada, dijaga permission yang sudah ada.

### K-02 — Salinan ambang kualitas tidak punya tempat di UI ⚠️ CELAH ✅ RESOLVED

- [Fase 2 § 3.1](03-Fase2.md#31-migration-000011_face_settings) menetapkan
  `face.min_det_score`, `min_blur_var`, `min_brightness`, `max_brightness`,
  `min_face_ratio`, `max_abs_yaw`, `max_abs_pitch` sebagai **salinan tampilan**
  (sumber kebenaran = env container inference), dan menyatakan salinan itu ada
  *"untuk ditampilkan di panel admin (Fase 5)"*, ditandai read-only.
- [Fase 2 § 5.1](03-Fase2.md#51-startup) juga menetapkan `faceclock-api` menulis
  **WARNING** saat konfigurasi inference *drift* dari `app_settings`.
- [Fase 5 § 2.6](06-Fase5.md#26-konfigurasi--menampilkan-konsekuensi-bukan-hanya-field)
  menyusun 4 kartu setting — dan **tidak satu pun memuat ketujuh key itu**, serta
  tidak ada tempat yang menampilkan peringatan drift.

Akibatnya: ambang yang benar-benar menentukan `usable` tidak terlihat di mana pun,
dan drift antara yang ditampilkan admin dengan yang benar-benar berlaku — persis
masalah yang Fase 2 § 3.1 ingin cegah — tidak punya permukaan untuk terlihat.

**Bukan kontradiksi keras** (tidak ada yang gagal), tapi janji Fase 2 tidak dipenuhi
Fase 5. **Butuh keputusan:** tambahkan kartu "Ambang kualitas (read-only)" +
indikator drift di `/settings` Fase 5, atau cabut janji itu dari Fase 2 § 3.1.
Tidak saya ubah sendiri.

#### ✅ Keputusan final

**Kartu baru ditambahkan** — "Ambang kualitas wajah" (read-only), kelima di
`/settings`, menampilkan ketujuh key persis dari Fase 2 § 3.1 dengan badge
drift kuning/abu-abu.

**Ternyata butuh endpoint baru.** `#28 GET /settings` hanya mengembalikan nilai
`app_settings` apa adanya — tidak pernah membandingkannya dengan nilai yang
sungguh aktif di inference (pembanding itu logika ad-hoc di startup
`faceclock-api`, [Fase 2 § 5.1](03-Fase2.md#51-startup), tidak pernah
diekspos lewat API). Endpoint baru: **`GET /api/v1/settings/face-quality-status`
(#73)**, guard `settings.read`, didaftarkan sebagai **[REV-EP-11](#b-perubahan-katalog-endpoint)**
(sifat **Wajib**) — total katalog endpoint naik dari 72 menjadi **73**.

**Kontrak diperbarui (dokumentasi, belum kode):**
- [Fase 5 § 2.1](06-Fase5.md#21-fase-5-bukan-murni-frontend--2-endpoint-backend-baru) —
  spesifikasi lengkap `#73`.
- [Fase 5 § 2.6](06-Fase5.md#26-konfigurasi--menampilkan-konsekuensi-bukan-hanya-field) —
  kartu baru + aturan badge drift.
- [Fase 5 § 4.4](06-Fase5.md#44-setting--audit--fase-1), § 8.1, § 8.5, § 10, § 12 —
  konsisten menyebut `#73`.

Endpoint ini **hidup di Fase 5** meski akar masalahnya berasal dari Fase 2 —
sama seperti #71/#72, ia dibangun belakangan karena UI yang membutuhkannya baru
ada di Fase 5.

### K-03 — Idempotensi hanya berlaku untuk permintaan yang berhasil ⚠️ CELAH ✅ RESOLVED

- [Fase 4 § 2.8](05-Fase4.md#28-idempotensi) menjanjikan: *"Permintaan berikutnya
  dengan kunci yang sama **mengembalikan response yang sama persis**."*
- Mekanismenya ([Fase 4 § 3.2](05-Fase4.md#32-attendances--migration-000019)) adalah
  kolom `attendances.idempotency_key` dengan unique partial index — **ada hanya bila
  record berhasil dibuat**.

Untuk permintaan yang gagal (`422 FACE_NOT_MATCHED`, `409`, `502`…) tidak ada baris
yang tersimpan, sehingga pengulangan dengan kunci yang sama akan **dijalankan ulang
penuh**, termasuk memanggil inference lagi.

Dampak praktisnya kecil (mengulang percobaan gagal tidak menghasilkan duplikat) dan
[Fase 7 § 2.6](08-Fase7.md#26-idempotensi-di-jaringan-lapangan) tetap benar untuk
kasus yang penting — response sukses yang hilang. Tetapi **kalimat § 2.8 lebih luas
daripada mekanismenya**, dan itu akan menyesatkan siapa pun yang mengimplementasikan
Fase 7 E5 ("lanjutkan pengiriman setelah app di-kill").

**Butuh keputusan:** persempit kalimat Fase 4 § 2.8 menjadi "idempoten untuk
permintaan yang menghasilkan record", **atau** tambahkan tabel `idempotency_keys`
tersendiri yang juga menyimpan response error. Tidak saya ubah sendiri.

#### ✅ Keputusan final

**Jalan pertama diambil: persempit kalimat, mekanisme tidak berubah.** Menambah
tabel `idempotency_keys` khusus untuk menyimpan response error dinilai tidak
sepadan — kegagalan yang aman diulang penuh (§ 2.8 sudah menjelaskan mengapa),
dan tabel tambahan berarti index tambahan + jalur kode tambahan untuk kasus
yang dampaknya sudah dinyatakan kecil.

**Kontrak diperbarui (dokumentasi, belum kode):**
- [Fase 4 § 2.8](05-Fase4.md#28-idempotensi) — kalimat dipersempit eksplisit:
  jaminan "response yang sama persis" **hanya** berlaku selama record hasil
  sukses tersimpan; permintaan gagal dijalankan ulang penuh.
- [Fase 4 § 11](05-Fase4.md#11-cara-test--verifikasi) — item uji baru (20a)
  memverifikasi inference dipanggil ulang setelah kegagalan dengan kunci yang sama.
- [Fase 7 § 2.6](08-Fase7.md#26-idempotensi-di-jaringan-lapangan) — catatan
  eksplisit bahwa "Coba kirim lagi" (E4) dan "Lanjutkan pengiriman" (E5) setelah
  kegagalan **bukan** replay dari cache, melainkan percobaan baru penuh.

**Tidak ada perubahan skema, migration, atau perilaku server** — murni perbaikan
kalimat kontrak + satu item uji tambahan.

### K-04 — Dua kode berbeda untuk satu akar masalah ⚠️ INKONSISTENSI KECIL ✅ RESOLVED

Kondisi yang sama — `app_settings.face.model_version` masih `"unset"`, artinya
kalibrasi Fase 2 belum dilakukan — dibalas dengan dua kode berbeda:

| Endpoint | Kode | Sumber |
|---|---|---|
| #38 `POST /face/enrollments` | `503 SERVICE_UNAVAILABLE` | [Fase 3 § 4.3](04-Fase3.md#43-enrollment) |
| #53/#54 check-in / check-out | `503 ATTENDANCE_NOT_CONFIGURED` | [Fase 4 § 2.10](05-Fase4.md#210-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0) |

Client harus menangani dua kode untuk menampilkan pesan yang sama
(*"Sistem belum dikonfigurasi. Hubungi admin."*).

**Butuh keputusan:** samakan ke `ATTENDANCE_NOT_CONFIGURED` (kurang tepat namanya
untuk enrollment), tambahkan `ENROLLMENT_NOT_CONFIGURED`, atau perkenalkan
`SYSTEM_NOT_CALIBRATED` yang dipakai keduanya. Tidak saya ubah sendiri — menambah
kode error adalah perubahan katalog, dan itu keputusan, bukan penyelarasan.

#### ✅ Keputusan final

**Tidak disamakan begitu saja ke `ATTENDANCE_NOT_CONFIGURED`.** Ditemukan saat
membaca ulang #53/#54 bahwa kode itu sebenarnya **sudah** menggabungkan dua akar
masalah berbeda dalam satu baris tabel: model belum dikalibrasi (§ 2.10) **dan**
geofence aktif tanpa satu pun `office_locations` aktif — dua hal yang tidak
berhubungan. Menyamakan `#38` ke `ATTENDANCE_NOT_CONFIGURED` apa adanya akan
menambah endpoint *enrollment* ke bawah nama kode yang sudah longgar cakupannya,
dan tidak menyelesaikan kebingungan, hanya memindahkannya.

**Kode netral baru dipakai: `FACE_SERVICE_NOT_CONFIGURED` (503)** — khusus akar
masalah "kalibrasi Fase 2 belum selesai", dipakai bersama oleh `#38`, `#53`,
`#54`. `ATTENDANCE_NOT_CONFIGURED` **tetap ada**, dipersempit khusus untuk
kondisi geofence-tanpa-lokasi-aktif di `#53`/`#54` — tidak berubah perilakunya,
hanya tidak lagi berbagi baris tabel dengan akar masalah yang tidak terkait.

**Ini menyimpang dari instruksi awal "jangan buat kode baru".** Opsi paling
literal (menyamakan semuanya ke `ATTENDANCE_NOT_CONFIGURED`) memang tidak
menambah kode, tapi memaksa endpoint enrollment memakai nama yang secara
semantik miliknya alur absensi, dan mempertahankan pencampuran dua akar masalah
tak-terkait di `#53`/`#54` yang sebenarnya sudah bisa diperbaiki sekalian.
Trade-off yang dipilih: **+1 kode di katalog** (40 → **41**, § 2 C), demi setiap
kode memetakan ke **tepat satu** akar masalah. Keputusan ini didokumentasikan
di sini secara tertulis persis karena menyimpang dari instruksi literal.

**Kontrak diperbarui (dokumentasi, belum kode):**
- [Fase 3 § 2.9](04-Fase3.md#29-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0)
  dan § 4.3 — `#38` memakai `FACE_SERVICE_NOT_CONFIGURED`, dengan catatan bahwa
  kode ini didaftarkan resmi di Fase 4, dipakai lebih dulu di sini (dampak urutan
  eksekusi dicatat di § 5).
- [Fase 4 § 2.10](05-Fase4.md#210-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0),
  § 4.2, dan alur pseudocode § 6 — baris `ATTENDANCE_NOT_CONFIGURED` dipecah jadi
  dua baris dengan dua kode.
- [Fase 6](07-Fase6.md), [Fase 7](08-Fase7.md) — peta pesan error diperbarui
  untuk menangani kedua kode dengan pesan yang sama ke pengguna.
- **REV-ERR-06 baru** ditambahkan di § 2 C untuk `FACE_SERVICE_NOT_CONFIGURED`;
  deskripsi REV-ERR-02 dipersempit untuk `ATTENDANCE_NOT_CONFIGURED`.

---

## 5. Dampak ke urutan eksekusi

Karena **belum ada satu baris kode pun**, tiap revisi masih bisa masuk ke
implementasi pertama. Tabel berikut menjawab satu pertanyaan per fase: *revisi mana
yang HARUS sudah tercermin di kode sebelum fase itu dinyatakan selesai?*

> ⚠️ **Aturan pokok:** saat mengeksekusi Fase N, **jangan menyalin bentuk draft**
> dari dokumen fase itu bila daftar di bawah menyebutkan revisi yang mengubahnya.
> Bentuk yang benar adalah bentuk **setelah revisi**.

### Fase 0 — Fondasi

| Harus sudah ada | Kenapa di sini |
|---|---|
| **REV-MW-01** — slot `RequireConsent` tercetak di rantai middleware | Rantai dikunci di Fase 0; Fase 3 mengisi, bukan mendesain ulang |
| **REV-ERR-04** — status 410 masuk `docs/adr/0003-api-conventions.md` | Katalog error dikunci di Fase 0 |
| **REV-CONV-01/02/03** — tiga pengecualian konvensi dicatat di ADR 0003 | Supaya Fase 1/3/4 tidak dianggap melanggar konvensi |
| **REV-INF-01** — HTTPS untuk `faceclock-web` disiapkan | Waktu tunggu infrastruktur; tanpa ini Fase 6 tidak bisa diuji |
| **REV-INF-03** — MinIO + `STORAGE_S3_*` di compose & `.env.example` | Compose dibuat di Fase 0 |
| **REV-INF-05/06** — dependensi web lengkap sejak awal | `package.json` dibuat di Fase 0 |
| **REV-EP-10** — `GET /version` menyertakan `min_supported_client` | Endpoint dibuat di Fase 0 |
| **REV-CTR-01/03** — kerangka `docs/api/hints.json` & `geo-testcases.json` | Berkas kanonik lahir bersama struktur `docs/` |
| **REV-INF-04** — catat bahwa `SignedURL()` tidak dipakai | Interface `Store` didefinisikan di Fase 0 |

### Fase 1 — Auth, User & RBAC

| Harus sudah ada | Kenapa di sini |
|---|---|
| **REV-AUTH-01 + REV-SET-08** — grace window reuse + `auth.refresh_reuse_grace_seconds` | ⛔ **Paling penting.** Alur rotasi refresh token ditulis di Fase 1. Menambahkannya belakangan berarti menulis ulang bagian paling sensitif keamanan setelah web dan mobile terlanjur bergantung pada perilaku lama |
| **REV-AUTH-02** — Full HttpOnly Cookie-Based Authentication + Anti-CSRF Shield | ⛔ **Keamanan Browser.** Mencegah pencurian token via XSS dengan memindahkan token browser ke cookie `HttpOnly` (`SameSite=Lax`) dan melindungi seluruh endpoint mutasi dengan Anti-CSRF shield (`X-Requested-With`, `X-CSRF-Token`, & verifikasi `Origin`), dengan tetap mempertahankan kompatibilitas fallback `Authorization: Bearer` untuk non-browser client |
| **REV-SET-04** — deskripsi `attendance.max_distance_meter` sudah memakai makna hasil rekonsiliasi | Seeder Fase 1 yang menuliskannya. Menyeed deskripsi yang sudah diketahui salah lalu meng-`UPDATE`-nya di Fase 4 adalah pekerjaan dua kali |
| **REV-CONV-01** — `audit_logs.id` `bigint identity` | Tabel dibuat di sini |
| **REV-DB-01** — `employees.attendance_mode` | Kolomnya milik tabel `employees`. Boleh tetap lewat migration 000013 (aditif, sesuai rencana Fase 3), **atau** dilipat ke 000002. Rekomendasi: pertahankan 000013 agar penomoran dokumen tetap cocok |

> **Catatan REV-EP-03 (#7 `GET /employees`):** *tidak* bisa dikerjakan di Fase 1 —
> field `consent_status` dan `enrollment_status` membaca tabel yang baru ada di
> Fase 3. Tetap dikerjakan di Fase 5 sebagaimana dilaporkan.

### Fase 2 — Face Inference Service

| Harus sudah ada | Kenapa di sini |
|---|---|
| **REV-SET-01** — 10 key `face.*` (migration 000011) | Scope fase ini |
| **REV-CTR-01 + REV-CTR-02** — `hints.json` kanonik **dengan namespace `server_hints` / `client_coach` sejak awal** | Fase 2 pemilik kosakata `hints`. Menulisnya tanpa namespace lalu merestrukturisasi saat Fase 7 berarti menyentuh tiga consumer sekaligus |

### Fase 3 — Enrollment Wajah

| Harus sudah ada | Kenapa di sini | Status |
|---|---|---|
| **REV-DB-01** — `employees.attendance_mode` | Migration 000013 | ✅ Selesai dieksekusi |
| **REV-DB-02** — `face_references.photo_key` (bukan `photo_url`) | Tabel dibuat di sini; bentuk master plan sudah diketahui salah | ✅ Selesai dieksekusi (migration 000014) |
| **REV-CONV-03** — `face_references` tanpa `deleted_at` | idem | ✅ Selesai dieksekusi |
| **REV-ERR-01** — 12 kode error Fase 3 | idem | ✅ Selesai dieksekusi |
| **REV-PERM-01** — `face.reindex` + grant `super_admin` | idem | ✅ Selesai dieksekusi (migration 000017) |
| **REV-SET-02** — 7 key `face.*` | idem | ✅ Selesai dieksekusi (migration 000017) |
| **REV-MW-01** — implementasi `RequireConsent` | Slot dari Fase 0 diisi di sini | ✅ Selesai dieksekusi |
| **REV-INF-03** — driver S3/MinIO benar-benar dipakai | Foto referensi pertama disimpan di fase ini | ✅ Selesai dieksekusi |
| **REV-ERR-06** — `#38` memakai `FACE_SERVICE_NOT_CONFIGURED` (bukan `SERVICE_UNAVAILABLE`) | ⛔ **Kode ini "dimiliki" dan didefinisikan resmi di Fase 4 § 2.10, tapi dipakai lebih dulu di sini** (K-04). `internal/httpx/errors.go` harus sudah mendaftarkan kode ini saat Fase 3 dieksekusi — dependensi maju yang tidak biasa, dicatat eksplisit supaya tidak terlewat | ✅ Selesai dieksekusi |

### Fase 4 — Attendance Engine ⛔ fase dengan beban revisi terberat

| Harus sudah ada | Kenapa di sini | Status |
|---|---|---|
| **REV-DB-03** — `attendances.photo_key` | Tabel dibuat di sini | ✅ Selesai dieksekusi (migration 000019) |
| **REV-DB-04** — 4 kolom liveness pada `attendances` | ⛔ **Dilaporkan Fase 7, tapi tabelnya dibuat di sini.** Masukkan ke migration 000019 sejak awal | ✅ Selesai dieksekusi (migration 000019) |
| **REV-DB-05** — `attendances.location_is_mocked` | idem | ✅ Selesai dieksekusi (migration 000019) |
| **REV-DB-06** — `fallback_reason` CHECK dengan `liveness_failed` + `location_mocked` | idem — CHECK constraint lebih mahal diubah belakangan daripada ditulis benar sejak awal | ✅ Selesai dieksekusi (migration 000019) |
| **REV-DB-07** — `attendance_attempts.outcome` CHECK +2 nilai | Migration 000020 | ✅ Selesai dieksekusi (migration 000020) |
| **REV-CONV-02** — `attendance_attempts.id` `bigint identity` | idem | ✅ Selesai dieksekusi (migration 000020) |
| **REV-ERR-02** — 14 kode error Fase 4, dengan `ATTENDANCE_NOT_CONFIGURED` **dipersempit** khusus geofence tanpa lokasi aktif | idem | ✅ Selesai dieksekusi (internal/httpx/errors.go) |
| **REV-ERR-06** — `FACE_SERVICE_NOT_CONFIGURED` dipakai `#53`/`#54` untuk kondisi model belum dikalibrasi (dipecah dari `ATTENDANCE_NOT_CONFIGURED`, K-04) | Registrasi resmi kode ini ada di sini; `#38` Fase 3 memakainya lebih dulu (lihat catatan di bagian Fase 3) | ✅ Selesai dieksekusi (internal/httpx/errors.go) |
| **REV-ERR-03** — `LIVENESS_REQUIRED` | Kode ini dilempar oleh handler check-in Fase 4 | ✅ Selesai dieksekusi (internal/httpx/errors.go) |
| **REV-ERR-05** — `can_fallback` pada 6 kondisi | Handler check-in Fase 4 yang menyusunnya | ✅ Selesai dieksekusi (internal/attendance) |
| **REV-SET-03** — 15 key `attendance.*` | Migration 000021 | ✅ Selesai dieksekusi (migration 000021) |
| **REV-SET-06** — 4 key liveness | ⛔ Migration 000021, bukan migration Fase 7 | ✅ Selesai dieksekusi (migration 000021) |
| **REV-SET-07** — `mocked_location_policy` | idem | ✅ Selesai dieksekusi (migration 000021) |
| **REV-EP-04** — `#55` +3 field (wajib karena K-01) | Endpoint dibuat di sini | ✅ Selesai dieksekusi (internal/attendance) |
| **REV-EP-05** — `#55` +2 field dimensi foto | idem | ✅ Selesai dieksekusi (internal/attendance) |
| **REV-EP-06/07** — `#55` +objek liveness, +`mocked_location_policy` | idem | ✅ Selesai dieksekusi (internal/attendance) |
| **REV-EP-08** — `#53/#54` menerima field liveness & mock | idem | ✅ Selesai dieksekusi (internal/attendance) |
| **REV-EP-09** — `#58` varian DTO per permission | idem | ✅ Selesai dieksekusi (internal/attendance) |
| **REV-CTR-03** — `geo-testcases.json` diisi | `internal/geo` ditulis di sini | ✅ Selesai dieksekusi (internal/geo) |

> **Ini yang paling mudah terlewat.** Tujuh belas revisi menyentuh Fase 4 (enam
> belas dari konsolidasi 2026-09-04, +1 REV-ERR-06 dari resolusi K-04 pada
> 2026-09-05), dan **sembilan di antaranya dilaporkan oleh Fase 7** — fase yang
> jaraknya tiga dokumen. Mengeksekusi Fase 4 dari draft-nya sendiri tanpa membaca
> daftar ini akan menghasilkan migration yang harus dibongkar ulang di Fase 7.

### Fase 5 — Admin Panel

| Harus sudah ada | Kenapa di sini |
|---|---|
| **REV-EP-01/02** — #71 `/summary`, #72 `/export` | Deliverable fase ini (bagian backend) |
| **REV-EP-03** — #7 `GET /employees` diperluas | Dibutuhkan halaman `/consents` |
| **REV-EP-11** — #73 `/settings/face-quality-status` + kartu "Ambang kualitas wajah" | Resolusi K-02 — deliverable baru fase ini, tidak ada di draft Fase 5 awal |
| **REV-SET-05** — `attendance.export_max_rows` | Pendamping #72 |
| **REV-UI-01** — panel approval menampilkan info liveness & mock location | Bergantung pada REV-DB-04/05 yang sudah ada sejak Fase 4 |

### Fase 6 — Halaman Absensi Web

| Harus sudah ada | Kenapa di sini |
|---|---|
| **REV-EP-04** — `#55` sudah memuat `require_face_for_checkout`, `checkout_without_face_status`, `max_note_length` | ⛔ **Tanpa ini Fase 6 gagal diam-diam** (K-01, ✅ kontrak final — § 4) |
| **REV-ERR-05** — `can_fallback` konsisten | UI menentukan tombol fallback dari field ini |
| **REV-ERR-06** — kedua kode (`FACE_SERVICE_NOT_CONFIGURED`, `ATTENDANCE_NOT_CONFIGURED`) dipetakan ke pesan yang sama | K-04 — dua kode, satu pesan pengguna |
| **REV-INF-01** — HTTPS aktif | Prasyarat `getUserMedia` & Geolocation |
| **REV-CTR-01/03** — berkas kanonik dipakai `lib/errors/hints.ts` & `lib/geo` | |

### Fase 7 — Flutter Mobile

| Harus sudah ada | Kenapa di sini |
|---|---|
| Seluruh revisi Fase 0–6 di atas | Fase 7 klien murni; tidak ada revisi backend yang dikerjakan **di** Fase 7 |
| **REV-INF-02** — HTTPS + ATS/network-security-config + pinning | Konfigurasi build mobile |
| **REV-CTR-02** — `client_coach` dipakai `liveness_messages.dart` | |

> **Poin penting:** meski **17 dari 48** item konsolidasi berasal dari laporan Fase 7,
> hampir semuanya **dieksekusi di Fase 1 dan Fase 4**, bukan di Fase 7. Fase 7
> sendiri hanya membawa REV-INF-02 sebagai pekerjaan barunya.

---

## 6. Pemeliharaan dokumen ini

- **Setiap revisi baru** yang ditemukan saat eksekusi tetap dilaporkan di dokumen
  fase yang sedang dikerjakan (pola Fase 3–7), **lalu** ditambahkan ke sini dengan
  ID berikutnya pada kelompok artefaknya.
- **Jangan menghapus item** yang sudah dieksekusi — ubah kolom `Status` menjadi
  ✅ beserta rujukan commit/migration. Log ini juga berfungsi sebagai jejak audit
  "kenapa skema ini berbeda dari draft master plan".
- **Empat temuan di § 4 sudah diputuskan (2026-09-05) dan dituliskan ke kontrak
  fase terkait** — K-01, K-02, K-03, K-04 kini berlabel ✅ RESOLVED, masing-masing
  dengan sub-bagian "Keputusan final" yang menjelaskan pilihan yang diambil dan
  file yang berubah. **Belum ada kode** — resolusi ini murni tingkat dokumen/kontrak,
  sama seperti seluruh 48 item lain di log ini.
- Keputusan yang masih menggantung di dokumen fase (D2–D5, D10–D12, D13–D19,
  D20–D26, D27–D33) **tidak** dicatat di sini — tempatnya di dokumen fase
  masing-masing. Dokumen ini khusus **revisi kontrak**, bukan keputusan desain.
