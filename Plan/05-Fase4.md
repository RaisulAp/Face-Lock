# Fase 4 — Attendance Engine

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 4**. Ukuran: 🔴 **Besar**.
> Mengikuti Protokol Handoff § 10.2: dokumen ini di-review sebelum eksekusi.
>
> **Depends on:** [Fase 1](02-Fase1.md), [Fase 2](03-Fase2.md), [Fase 3](04-Fase3.md).
> **Blocker untuk:** Fase 5 (panel admin & approval) dan Fase 6 (halaman absensi web).
>
> **Status:** DRAFT — 3 keputusan butuh konfirmasi user (§ 2.0).
> ⛔ **Satu blocker yang diwarisi:** Dataset B (kalibrasi pada populasi nyata) —
> lihat § 9 dan § 10.

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Ini inti sistem. Semua fase sebelumnya membangun bahan; Fase 4 adalah tempat
keputusan "kehadiran ini sah atau tidak" benar-benar dibuat, dan tempat setiap
asumsi keamanan yang ditulis di fase lain diuji oleh orang yang punya insentif
untuk melanggarnya.

Tiga hal yang harus benar di sini, dan tidak ada yang bisa memperbaikinya
belakangan:
1. **Waktu berasal dari server.** Kalau tidak, seluruh sistem hanya seremonial.
2. **Kegagalan tidak pernah berujung `approved`.** Inference mati, wajah tidak
   cocok, GPS palsu — semuanya berujung ditolak atau `pending_review`, tidak pernah
   lolos otomatis.
3. **Setiap record bisa diaudit.** Bukti wajah, lokasi, waktu server, similarity,
   threshold yang berlaku saat itu, dan siapa yang menyetujui.

### Hasil akhir yang diharapkan
- Karyawan bisa check-in dan check-out dengan verifikasi wajah lewat API.
- Absensi yang lolos verifikasi berstatus `approved`; yang lewat jalur fallback
  **selalu** `pending_review` dan menunggu persetujuan manusia.
- Geofence divalidasi di server terhadap `office_locations`, dengan dukungan
  multi-lokasi.
- Aturan bisnis ditegakkan: satu check-in per hari kerja, check-out butuh check-in,
  urutan waktu benar.
- Admin bisa melihat antrian `pending_review` dan menyetujui/menolak dengan alasan.
- Setiap percobaan gagal tercatat sebagai telemetri keamanan, tanpa menumpuk foto.

### Yang TIDAK dikerjakan di fase ini
- UI apa pun (→ Fase 5 admin, Fase 6 karyawan).
- Laporan, rekap agregat, dan export CSV/Excel (→ Fase 5). Fase 4 menyediakan
  daftar dan filter yang menjadi dasarnya; permission `attendance.export` sudah
  di-seed sejak Fase 1 tapi belum dipakai.
- Perhitungan keterlambatan, lembur, cuti. `attendance.workday_start/end` sudah ada
  di `app_settings` sejak Fase 1, tapi Fase 4 hanya **menyimpan** waktu — tidak
  menafsirkannya.
- Liveness (→ Fase 7). Fase 4 menyimpan `capture_source` sebagai klaim, dan
  mengatakan terus terang bahwa ia klaim (§ 6 E28).

---

## 2. Scope Detail

### 2.0 Keputusan yang butuh konfirmasi

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D17 | Bentuk alur fallback | **Satu endpoint dengan flag `allow_fallback`**, bukan endpoint terpisah (§ 2.3) | ⚠️ **BUTUH KONFIRMASI** |
| D18 | Perlakuan absensi di luar geofence | **Tolak** secara default, dengan setting `attendance.outside_geofence_policy` untuk mengubahnya ke `pending_review` (§ 2.5) | ⚠️ **BUTUH KONFIRMASI** |
| D19 | Apakah check-out juga wajib verifikasi wajah | **Ya** (`attendance.require_face_for_checkout = true`), bisa dimatikan lewat setting (§ 2.6) | ⚠️ **BUTUH KONFIRMASI** |

---

### 2.1 Rekonsiliasi setting yang sudah ada

[Fase 1 § 3.9](02-Fase1.md#39-app_settings--migration-000010) sudah men-seed
`attendance.max_distance_meter = 100`, sementara Fase 4 memperkenalkan
`office_locations.radius_meter` per-lokasi. Keduanya tidak boleh menjadi dua
sumber kebenaran.

**Aturan yang ditetapkan:**
- `office_locations.radius_meter` adalah **satu-satunya** yang menentukan apakah
  seseorang berada di dalam geofence.
- `attendance.max_distance_meter` diubah maknanya menjadi **batas atas dan nilai
  default**: nilai awal `radius_meter` saat membuat lokasi baru, dan `radius_meter`
  tidak boleh melebihinya (`422 VALIDATION_ERROR`).
- Deskripsi baris `app_settings` itu diperbarui di migration Fase 4 agar tidak ada
  yang salah membacanya.

> Ini termasuk **revisi kecil kontrak Fase 1** dan dilaporkan sebagai revisi, bukan
> diubah diam-diam.

Demikian pula `attendance.workday_start` / `workday_end`: Fase 4 **tidak
memakainya** untuk apa pun. Keduanya disimpan untuk Fase 5 (perhitungan
keterlambatan). Yang dipakai Fase 4 adalah `attendance.timezone` dan
`attendance.workday_cutoff_hour` (§ 2.4).

### 2.2 Waktu server & konsep `work_date`

Master plan § 9: *"Timestamp server-side: jangan percaya data waktu dari client."*

Penegakannya:
- `attendances.server_timestamp` diisi `now()` **di dalam statement `INSERT`**,
  bukan dari variabel Go. Tidak ada jalur kode yang bisa menuliskan nilai lain.
- `client_reported_at` dari client **disimpan** di kolom terpisah, tapi tidak
  pernah dipakai untuk apa pun kecuali telemetri: selisih besar antara jam client
  dan jam server adalah sinyal yang berguna (`clock_skew_seconds`), dan
  membuangnya berarti membuang bukti.

**`work_date`** adalah kunci semua aturan bisnis harian. Ia dihitung server, dalam
statement yang sama dengan `now()` (yang stabil sepanjang satu transaksi):

```sql
work_date := ((now() AT TIME ZONE $tz) - make_interval(hours => $cutoff))::date
```

- `$tz` = `app_settings.attendance.timezone` (default `Asia/Jakarta`).
- `$cutoff` = `app_settings.attendance.workday_cutoff_hour` (default `0`).

`workday_cutoff_hour` menangani shift malam tanpa menambah tabel jadwal: dengan
nilai `4`, check-out pukul 02:00 masih terhitung hari kerja sebelumnya. Nilai
default `0` berarti perilakunya persis "tanggal kalender lokal", yang benar untuk
kantor jam kerja normal.

Menyimpan `work_date` sebagai kolom (bukan menghitungnya saat query) disengaja:
ia dipakai oleh dua index unik parsial yang menegakkan "satu check-in per hari"
(§ 3.3), dan index tidak bisa dibangun di atas ekspresi yang bergantung pada
setting yang berubah-ubah.

### 2.3 D17 — Bentuk alur fallback ⚠️ BUTUH KONFIRMASI

Master plan: *"Fallback: bila gagal match, izinkan upload foto → record `pending_review`."*
dan § 9: *"Jalur fallback **tidak boleh** langsung `approved`. Selalu `pending_review`."*

Tiga bentuk yang mungkin:

| Bentuk | Cara kerja | Masalah |
|---|---|---|
| A. Endpoint terpisah `/attendances/fallback` | Client memanggilnya setelah check-in gagal | Endpoint yang **selalu** menghasilkan `pending_review` tanpa pernah mencoba wajah. Siapa pun bisa memanggilnya langsung dan melewati verifikasi sepenuhnya. Jalur wajah menjadi opsional |
| B. Otomatis: gagal wajah → langsung `pending_review` | Satu panggilan | Karyawan tidak pernah tahu wajahnya gagal, dan tidak punya kesempatan mencoba ulang dengan pencahayaan lebih baik. Antrian approval akan penuh oleh hal yang seharusnya bisa diperbaiki sendiri |
| **C. Satu endpoint, flag `allow_fallback`** (rekomendasi) | Percobaan pertama `allow_fallback=false`; gagal → `422` + `hints`; client menawarkan "kirim untuk direview?"; percobaan kedua `allow_fallback=true` → `pending_review` | Butuh foto dikirim dua kali (~200 KB) |

**Rekomendasi: C.** Alasannya:
- Jalur wajah **selalu dicoba lebih dulu**, apa pun nilai `allow_fallback`. Tidak
  ada pintu yang melewatinya. Ini yang membuat bentuk A gugur — dan bentuk A
  adalah yang paling mudah ditulis, jadi kegagalannya perlu dinyatakan eksplisit.
- Karyawan mendapat `hints` yang bisa ditindaklanjuti ("terlalu gelap") dan
  kesempatan mencoba ulang. Jumlah `pending_review` yang harus di-review manusia
  tetap kecil — dan sistem approval yang kebanjiran adalah sistem approval yang
  di-klik "setujui semua".
- Foto kedua adalah **capture baru**, bukan pengulangan foto yang sama. Itu bukti
  yang lebih baik, bukan lebih buruk.

Konsekuensi yang harus ditegakkan: `allow_fallback=true` **tidak** melewati
verifikasi. Ia hanya mengubah apa yang terjadi **setelah** verifikasi gagal.
Server tetap memanggil inference, tetap membandingkan, dan tetap mencatat
`matched_similarity` bila ada. Kalau ternyata cocok, hasilnya `approved` — bendera
`allow_fallback` tidak menurunkan derajat absensi yang sah.

### 2.4 Aturan bisnis

| Kode | Aturan | Ditegakkan oleh |
|---|---|---|
| B1 | Satu `check_in` per `(employee_id, work_date)` yang tidak `rejected` | Index unik parsial (§ 3.3) — bukan `SELECT` lebih dulu |
| B2 | Satu `check_out` per `(employee_id, work_date)` yang tidak `rejected` | Index unik parsial |
| B3 | `check_out` butuh `check_in` di `work_date` yang sama | Service; bisa dimatikan `attendance.allow_checkout_without_checkin` |
| B4 | `check_out` tidak boleh lebih awal dari `check_in`-nya | Service (perbandingan `server_timestamp`) |
| B5 | Jeda minimum antara `check_in` dan `check_out` | `attendance.min_minutes_between_checkin_checkout` (default 1) |
| B6 | Karyawan harus `employment_status='active'` dan tidak `deleted_at` | Service → `403 EMPLOYEE_INACTIVE` |
| B7 | Karyawan mode `face` harus punya ≥ `face.min_reference_photos` referensi aktif **dengan `model_version` yang aktif** | Service → `422 FACE_NOT_ENROLLED` |
| B8 | Karyawan mode `manual` melewati verifikasi wajah dan **selalu** `pending_review` | Service |
| B9 | Foto yang sama tidak boleh dipakai dua kali (anti-replay) | Unique parsial + cek 7 hari (§ 2.7) |
| B10 | Record yang sudah `approved`/`rejected` tidak bisa di-review ulang | Service → `409 ATTENDANCE_ALREADY_REVIEWED` |
| B11 | Tidak boleh me-review absensi milik diri sendiri | Service → `403 SELF_REVIEW_DENIED` |
| B12 | Terlalu banyak percobaan gagal → dijeda | `attendance.max_failed_attempts_per_hour` → `429` |

**B11 pantas dijelaskan.** Admin/HR hampir selalu juga karyawan, dan
[Fase 1 § 2.4](02-Fase1.md#24-role-default) memberi role `admin` permission
`attendance.approve` **dan** `attendance.checkin`. Tanpa B11, seorang admin bisa
sengaja menggagalkan verifikasi wajahnya sendiri, masuk jalur fallback, lalu
menyetujui dirinya sendiri — dan record-nya akan terlihat sah lengkap dengan
`reviewed_by`. Ini bukan skenario hipotetis; ini kombinasi permission yang memang
sudah ada di sistem sejak Fase 1.

**B1/B2 memakai index, bukan pemeriksaan aplikasi.** Ini pelajaran yang sudah
ditulis di [Fase 1 E29](02-Fase1.md#63-data--konkurensi): `SELECT` lalu `INSERT`
bisa dibalap dua tab yang menekan tombol bersamaan. Predikat `status <> 'rejected'`
membuat karyawan yang absensinya ditolak admin bisa mencatat ulang — itu memang
yang diinginkan, dan itulah sebabnya `rejected` bersifat terminal (B10): kalau
penolakan bisa dibatalkan, index ini bisa bentrok dengan record baru.

### 2.5 D18 — Geofence ⚠️ BUTUH KONFIRMASI

Master plan § 2: *"Geofencing: validasi radius maksimal dari lokasi kantor."*
Perhitungan seluruhnya di server; koordinat dari client adalah masukan, bukan
kesimpulan.

**Algoritma (`internal/geo`, fungsi murni, tanpa I/O):**

```
Evaluate(lat, lng, accuracy, locations[]) → GeofenceResult
 1. geofence_enabled = false                      ⇒ status = "disabled"
 2. lat atau lng nil                              ⇒ status = "unavailable"
 3. accuracy > attendance.max_gps_accuracy_meter  ⇒ status = "unavailable"
 4. untuk setiap lokasi aktif: d = haversine(lat, lng, loc.lat, loc.lng)
 5. covering = { loc : d ≤ loc.radius_meter }
      covering tidak kosong ⇒ status = "inside",
                              office_location = argmin(d) di antara covering
      covering kosong       ⇒ status = "outside",
                              office_location = argmin(d) di antara SEMUA lokasi
                              (dilaporkan supaya karyawan tahu ia jauh dari mana)
```

Langkah 5 sengaja memilih dari **himpunan yang mencakup**, bukan dari lokasi
terdekat saja. Kalau tidak, karyawan yang berdiri di dalam radius gudang (radius
300 m) tapi kebetulan lebih dekat ke kantor pusat yang radiusnya 50 m akan
dinyatakan di luar geofence — padahal ia berada di dalam salah satu lokasi resmi.

Haversine dengan `R = 6.371.008,8 m`. Tidak memakai PostGIS maupun ekstensi
`earthdistance`: jumlah lokasi kantor dihitung dengan jari, perhitungannya beberapa
mikrodetik di Go, dan fungsi murni jauh lebih mudah diuji daripada ekspresi SQL.

**Perlakuan hasilnya:**

| `geofence_status` | Perlakuan default | Setting |
|---|---|---|
| `inside` | Lanjut normal | — |
| `disabled` | Lanjut normal | `attendance.geofence_enabled = false` |
| `outside` | **Tolak** `422 OUTSIDE_GEOFENCE` | `attendance.outside_geofence_policy` = `reject` \| `pending_review` |
| `unavailable` | **Tolak** `422 LOCATION_REQUIRED` / `LOCATION_INACCURATE` | `attendance.missing_location_policy` = `reject` \| `pending_review` |

**Rekomendasi default `reject`** untuk keduanya. Lokasi adalah aturan yang tegas
dan bukan soal kualitas foto: kalau di luar radius dibiarkan masuk sebagai
`pending_review`, jalur itu akan menjadi cara normal untuk absen dari rumah, dan
antrian approval akan menjadi stempel. Organisasi yang memang butuh absen dinas
luar bisa mengubahnya ke `pending_review` secara sadar lewat setting — dan
perubahan itu tercatat di `audit_logs`.

Geofence dievaluasi **sebelum** foto disimpan dan sebelum inference dipanggil.
Menolak lebih awal berarti tidak ada foto wajah yang disimpan untuk request yang
memang tidak akan pernah sah.

### 2.6 D19 — Verifikasi wajah saat check-out ⚠️ BUTUH KONFIRMASI

Master plan § 4 menyebut check-in **dan** check-out melalui alur verifikasi yang
sama. Rekomendasi: pertahankan (`attendance.require_face_for_checkout = true`).

Alasan: check-out yang tidak diverifikasi membuat "titip pulang" menjadi gratis,
dan pada banyak organisasi jam pulang justru yang menentukan perhitungan lembur.
Bila di lapangan ternyata memberatkan (mis. antrean di pintu keluar), setting ini
bisa dimatikan — dan konsekuensinya menjadi keputusan sadar organisasi, bukan
celah yang tidak disengaja. Saat dimatikan, check-out tetap menyimpan foto dan
lokasi, hanya tidak membandingkan wajah; statusnya `pending_review` bila
`attendance.checkout_without_face_status = 'pending_review'` (default) atau
`approved` bila organisasi memilih demikian.

### 2.7 Anti-penyalahgunaan

Tiga kontrol yang murah dan menutup celah nyata:

**a. Telemetri percobaan.** Setiap percobaan — berhasil maupun gagal — menulis satu
baris `attendance_attempts` (§ 3.4). Baris ini **tidak menyimpan foto**: menyimpan
foto setiap percobaan gagal berarti menumpuk data biometrik dari kejadian yang
justru paling tidak berguna, dan itu bertentangan dengan minimalisasi data. Yang
disimpan: `outcome`, `matched_similarity`, `threshold_used`, `hints`,
`quality_score`, `distance_meter`.

**b. Pembatasan percobaan gagal.** Lebih dari
`attendance.max_failed_attempts_per_hour` (default 10) percobaan tidak berhasil
dalam 1 jam → `429 TOO_MANY_FAILED_ATTEMPTS` dengan `Retry-After`. Ini menahan
orang yang mencoba menembus threshold dengan variasi foto, dan memberi sinyal jelas
di dashboard. Batasnya cukup longgar agar karyawan yang wajahnya memang sulit
dikenali di pagi hari tidak terkunci.

**c. Anti-replay foto.** Foto yang byte-nya identik (`sha256` sama) dengan absensi
karyawan itu dalam 7 hari terakhir → `422 DUPLICATE_PHOTO`. Ini menutup serangan
paling sederhana: menyimpan satu foto yang pernah lolos, lalu mengirimkannya ulang
setiap hari. Retry yang sah dilindungi header `Idempotency-Key` (§ 2.8), sehingga
pengiriman ulang karena jaringan putus tidak tertolak.

**Yang sengaja TIDAK diungkap ke karyawan:** `matched_similarity` dan
`threshold_used` **tidak** muncul di response check-in maupun di
`GET /attendances/me`. Keduanya hanya terlihat oleh pemegang `attendance.read_all`.
Memberi tahu seseorang bahwa skornya "0,41 sedangkan ambangnya 0,45" adalah
memberi umpan balik terukur kepada orang yang sedang mencoba menipu sistem.
Karyawan tetap mendapat `hints` yang bisa ditindaklanjuti — itu cukup untuk
memperbaiki foto, dan tidak cukup untuk menyetel serangan.

### 2.8 Idempotensi

Check-in dan check-out **membuat data** dan berjalan di jaringan seluler yang tidak
andal. Tanpa idempotensi, satu timeout di sisi client menghasilkan dua percobaan,
dan yang kedua akan gagal `409 ALREADY_CHECKED_IN` — membuat karyawan mengira
absensinya gagal padahal berhasil.

- Client boleh mengirim header `Idempotency-Key` (ULID/UUID).
- Server menyimpan `(employee_id, idempotency_key)` beserta `attendance_id` hasilnya
  selama 24 jam — **hanya untuk permintaan yang berhasil membuat record**
  (`attendances.idempotency_key` diisi saat insert sukses, § 3.2). Permintaan
  berikutnya dengan kunci yang sama, selama record itu ada, **mengembalikan
  response yang sama persis**, tanpa memanggil inference dan tanpa membuat record baru.
- **Permintaan yang GAGAL (`422 FACE_NOT_MATCHED`, `409`, `502`, dst.) tidak
  menyimpan apa pun terhadap kunci itu.** Mengulang dengan kunci yang sama setelah
  kegagalan akan **dijalankan ulang penuh** — termasuk memanggil inference lagi —
  bukan menerima response gagal yang di-cache. Ini cukup untuk kasus yang penting
  (response **sukses** yang hilang di jaringan, § 6.4 E4/E5 Fase 7), tapi client
  **tidak boleh** berasumsi kunci yang sama membuat sebuah percobaan gagal menjadi
  "aman untuk diulang tanpa efek" secara umum — ia hanya aman untuk diulang, bukan
  dijamin idempoten, kecuali percobaan sebelumnya sukses.
  ([Resolusi K-03](09-Revisions-Log.md#k-03--idempotensi-hanya-berlaku-untuk-permintaan-yang-berhasil--celah--resolved) —
  ini perbaikan kalimat, mekanismenya tidak berubah.)
- Tanpa header, perilakunya seperti biasa. Fase 6/7 diwajibkan mengirimkannya.

### 2.9 Retensi foto absensi

Utang R7 dari [Fase 2 § 13.3](03-Fase2.md#133-risiko-lintas-fase-yang-belum-terselesaikan),
kini jatuh tempo. Setiap check-in/out menyimpan satu foto; 200 karyawan × 2 ×
250 hari kerja ≈ 100.000 foto/tahun ≈ 25–30 GB.

Kebijakan:
- `attendance.photo_retention_days` (default 365). Job harian menghapus objek foto
  untuk record yang `server_timestamp` melewati ambang, mengisi `photo_purged_at`,
  dan mengosongkan `photo_key` — dalam transaksi yang sama.
- **Record absensinya tidak pernah dihapus.** Ia adalah catatan kehadiran yang
  mungkin dibutuhkan bertahun-tahun; yang dihapus hanyalah bukti biometriknya.
- Record `pending_review` **tidak** dipurge selama masih `pending_review` —
  menghapus bukti dari perkara yang belum diputus tidak masuk akal. Ambangnya
  dihitung dari `reviewed_at` untuk record yang sudah di-review.
- Bucket `faceclock-attendance` terpisah dari `faceclock-face`
  ([Fase 3 § 2.1](04-Fase3.md#21-d13--penyimpanan-foto-referensi--butuh-konfirmasi))
  justru supaya masa retensinya bisa berbeda.

### 2.10 Error code tambahan (registrasi resmi ke katalog Fase 0)

Semua memakai envelope dan status HTTP yang sudah ada.

| HTTP | code | Kapan |
|---|---|---|
| 422 | `FACE_NOT_ENROLLED` | Belum punya cukup referensi aktif dengan `model_version` aktif |
| 422 | `FACE_NOT_MATCHED` | Similarity di bawah threshold dan `allow_fallback=false` |
| 422 | `OUTSIDE_GEOFENCE` | Di luar radius semua lokasi aktif, kebijakan `reject` |
| 422 | `LOCATION_REQUIRED` | Koordinat tidak dikirim sementara geofence aktif |
| 422 | `LOCATION_INACCURATE` | `gps_accuracy_meter` melebihi ambang |
| 409 | `ALREADY_CHECKED_IN` | Sudah ada check-in non-`rejected` di `work_date` ini |
| 409 | `ALREADY_CHECKED_OUT` | Sudah ada check-out non-`rejected` di `work_date` ini |
| 409 | `CHECKOUT_WITHOUT_CHECKIN` | Check-out tanpa check-in di hari yang sama |
| 409 | `CHECKOUT_TOO_SOON` | Jeda < `min_minutes_between_checkin_checkout` |
| 409 | `ATTENDANCE_ALREADY_REVIEWED` | Record sudah `approved`/`rejected` |
| 403 | `SELF_REVIEW_DENIED` | Me-review absensi milik sendiri (B11) |
| 403 | `EMPLOYEE_INACTIVE` | Karyawan tidak aktif / resigned |
| 429 | `TOO_MANY_FAILED_ATTEMPTS` | Melebihi batas percobaan gagal per jam |
| 503 | `FACE_SERVICE_NOT_CONFIGURED` | `face.model_version = "unset"` — Fase 2 belum dikalibrasi |
| 503 | `ATTENDANCE_NOT_CONFIGURED` | Geofence aktif tanpa satu pun `office_locations` aktif |

`FACE_NOT_USABLE` dan `DUPLICATE_PHOTO` dipakai ulang dari
[Fase 3 § 2.9](04-Fase3.md#29-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0) —
tidak didefinisikan ulang.

> **Resolusi [K-04](09-Revisions-Log.md#k-04--dua-kode-berbeda-untuk-satu-akar-masalah--inkonsistensi-kecil--resolved):**
> sebelumnya baris ini menggabungkan **dua akar masalah berbeda** di bawah satu
> kode `ATTENDANCE_NOT_CONFIGURED` (model belum dikalibrasi **dan** geofence tanpa
> lokasi aktif), sementara `#38 POST /face/enrollments` ([Fase 3 § 4.3](04-Fase3.md#43-enrollment))
> memakai `SERVICE_UNAVAILABLE` untuk akar masalah yang **sama** dengan yang
> pertama. Dipisah jadi dua kode: `FACE_SERVICE_NOT_CONFIGURED` khusus akar
> masalah "kalibrasi Fase 2 belum selesai" — **dipakai bersama oleh `#38`, `#53`,
> dan `#54`** karena akar masalahnya identik — dan `ATTENDANCE_NOT_CONFIGURED`
> dipersempit khusus untuk kondisi geofence tanpa lokasi aktif, yang murni
> masalah konfigurasi lokasi kantor dan tidak ada hubungannya dengan wajah.
> Menyamakan **semuanya** jadi `ATTENDANCE_NOT_CONFIGURED` (opsi paling literal)
> ditolak karena kode itu tidak pas dipakai dari endpoint *enrollment*, yang
> bukan bagian dari alur absensi. Efeknya: katalog error bertambah **satu** kode
> baru dari yang direncanakan semula ([lihat 09-Revisions-Log.md § 2 C](09-Revisions-Log.md#c-perubahan-katalog-error-code)),
> bukan sekadar mengganti nama tanpa menambah — dilaporkan eksplisit karena
> menyimpang dari instruksi awal "jangan buat kode baru".

---

## 3. Skema Database

Konvensi mengikuti [Fase 0 § 3](01-Fase0.md#3-skema-database). Migration Fase 4:
`000018` – `000021`.

### 3.1 `office_locations` — migration `000018`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` |
| `name` | `text` | `NOT NULL`, `CHECK (length(btrim(name)) BETWEEN 2 AND 120)` |
| `address` | `text` | `NULL` |
| `lat` | `double precision` | `NOT NULL`, `CHECK (lat BETWEEN -90 AND 90)` |
| `lng` | `double precision` | `NOT NULL`, `CHECK (lng BETWEEN -180 AND 180)` |
| `radius_meter` | `integer` | `NOT NULL`, `CHECK (radius_meter BETWEEN 10 AND 10000)` |
| `is_active` | `boolean` | `NOT NULL DEFAULT true` |
| `created_by` / `updated_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` |
| `created_at` / `updated_at` / `deleted_at` | `timestamptz` | |

```sql
CREATE UNIQUE INDEX office_locations_name_uniq ON office_locations (lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX office_locations_active_idx ON office_locations (is_active) WHERE deleted_at IS NULL;
```

Batas atas `radius_meter` juga divalidasi terhadap `attendance.max_distance_meter`
di service (§ 2.1) — `CHECK` di sini adalah pagar terakhir, bukan aturan bisnisnya.

Tidak ada tipe geometry/PostGIS: perhitungan jarak ada di Go (§ 2.5).

### 3.2 `attendances` — migration `000019`

Tabel inti. **Append-only: tidak ada `deleted_at`**, sesuai konvensi Fase 0 untuk
tabel catatan.

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `employee_id` | `uuid` | `NOT NULL`, FK → `employees(id)` `ON DELETE RESTRICT` | |
| `type` | `text` | `NOT NULL`, `CHECK (type IN ('check_in','check_out'))` | |
| `server_timestamp` | `timestamptz` | `NOT NULL DEFAULT now()` | **satu-satunya** waktu yang dipercaya |
| `work_date` | `date` | `NOT NULL` | dihitung server (§ 2.2) |
| `method` | `text` | `NOT NULL`, `CHECK (method IN ('face','fallback'))` | |
| `status` | `text` | `NOT NULL`, `CHECK (status IN ('approved','pending_review','rejected'))` | |
| `fallback_reason` | `text` | `NULL`, `CHECK (fallback_reason IN ('below_threshold','face_not_usable','inference_unavailable','no_reference','outside_geofence','location_unavailable','manual_mode','checkout_without_face'))` | |
| `matched_similarity` | `real` | `NULL`, `CHECK (matched_similarity BETWEEN -1 AND 1)` | |
| `matched_reference_id` | `uuid` | `NULL`, FK → `face_references(id)` `ON DELETE SET NULL` | |
| `threshold_used` | `real` | `NULL` | snapshot; wajib untuk audit saat threshold berubah |
| `model_version` | `text` | `NULL` | |
| `quality_score` | `real` | `NULL` | |
| `hints` | `jsonb` | `NOT NULL DEFAULT '[]'::jsonb` | |
| `photo_key` | `text` | `NULL` | `NULL` hanya setelah purge |
| `photo_sha256` | `text` | `NOT NULL`, `CHECK (photo_sha256 ~ '^[0-9a-f]{64}$')` | tetap ada setelah purge (anti-replay) |
| `photo_bytes` | `integer` | `NOT NULL` | |
| `photo_mime` | `text` | `NOT NULL` | |
| `photo_purged_at` | `timestamptz` | `NULL` | |
| `lat` / `lng` | `double precision` | `NULL`, CHECK rentang | |
| `gps_accuracy_meter` | `real` | `NULL`, `CHECK (>= 0)` | |
| `geofence_status` | `text` | `NOT NULL`, `CHECK (geofence_status IN ('inside','outside','disabled','unavailable'))` | |
| `office_location_id` | `uuid` | `NULL`, FK → `office_locations(id)` `ON DELETE SET NULL` | |
| `distance_meter` | `real` | `NULL` | |
| `note` | `text` | `NULL`, `CHECK (note IS NULL OR length(note) <= 500)` | |
| `capture_source` | `text` | `NOT NULL`, `CHECK (capture_source IN ('web_camera','mobile_camera'))` | klaim client (E28) |
| `client_reported_at` | `timestamptz` | `NULL` | telemetri saja, **tidak pernah** dipakai |
| `clock_skew_seconds` | `integer` | `NULL` | `client_reported_at - server_timestamp` |
| `attendance_mode` | `text` | `NOT NULL`, `CHECK (attendance_mode IN ('face','manual'))` | snapshot dari `employees` |
| `request_id` | `text` | `NULL` | korelasi ke log |
| `idempotency_key` | `text` | `NULL` | § 2.8 |
| `reviewed_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` | |
| `reviewed_at` | `timestamptz` | `NULL` | |
| `review_note` | `text` | `NULL`, `CHECK (review_note IS NULL OR length(review_note) <= 500)` | |
| `created_at` / `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |

**Constraint integritas (yang membuat aturan tidak bisa dilanggar lewat SQL langsung):**

```sql
-- Jalur fallback WAJIB pending_review atau rejected — TIDAK PERNAH approved.
-- Ini implementasi harfiah master plan § 9 di level database.
ALTER TABLE attendances ADD CONSTRAINT attendances_fallback_never_approved_chk
  CHECK (method <> 'fallback' OR status <> 'approved' OR reviewed_by IS NOT NULL);

-- method='fallback' wajib punya alasan; method='face' tidak boleh punya.
ALTER TABLE attendances ADD CONSTRAINT attendances_fallback_reason_chk
  CHECK ((method = 'fallback') = (fallback_reason IS NOT NULL));

-- Status approved lewat jalur face wajib membawa bukti angkanya.
ALTER TABLE attendances ADD CONSTRAINT attendances_face_evidence_chk
  CHECK (method <> 'face' OR (matched_similarity IS NOT NULL
                              AND threshold_used IS NOT NULL
                              AND model_version IS NOT NULL));

-- Review konsisten.
ALTER TABLE attendances ADD CONSTRAINT attendances_review_chk
  CHECK ((reviewed_by IS NULL) = (reviewed_at IS NULL));
ALTER TABLE attendances ADD CONSTRAINT attendances_review_required_chk
  CHECK (status = 'pending_review' OR method = 'face' OR reviewed_by IS NOT NULL);

-- Foto: ada key, atau sudah di-purge.
ALTER TABLE attendances ADD CONSTRAINT attendances_photo_chk
  CHECK ((photo_key IS NOT NULL) OR (photo_purged_at IS NOT NULL));
```

Constraint pertama pantas dibaca dua kali. Ia menyatakan: sebuah record `fallback`
boleh berstatus `approved` **hanya bila ada manusia yang menyetujuinya**
(`reviewed_by IS NOT NULL`). Tidak ada jalur kode — dan tidak ada `UPDATE` manual —
yang bisa menghasilkan absensi fallback yang `approved` tanpa penyetuju.

**Index:**

```sql
-- B1 & B2: aturan harian, ditegakkan database, bukan aplikasi.
CREATE UNIQUE INDEX attendances_one_checkin_per_day
  ON attendances (employee_id, work_date)
  WHERE type = 'check_in' AND status <> 'rejected';
CREATE UNIQUE INDEX attendances_one_checkout_per_day
  ON attendances (employee_id, work_date)
  WHERE type = 'check_out' AND status <> 'rejected';

-- Idempotensi (§ 2.8)
CREATE UNIQUE INDEX attendances_idempotency_uniq
  ON attendances (employee_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Riwayat pribadi & rekap (dasar laporan Fase 5)
CREATE INDEX attendances_employee_date_idx
  ON attendances (employee_id, work_date DESC, server_timestamp DESC);
CREATE INDEX attendances_work_date_idx  ON attendances (work_date DESC);

-- Antrian approval: index parsial, hanya sebesar antriannya
CREATE INDEX attendances_pending_idx
  ON attendances (server_timestamp DESC) WHERE status = 'pending_review';

-- Anti-replay (§ 2.7c)
CREATE INDEX attendances_photo_sha_idx
  ON attendances (employee_id, photo_sha256, server_timestamp DESC);

-- Job retensi
CREATE INDEX attendances_purge_idx
  ON attendances (server_timestamp) WHERE photo_purged_at IS NULL;
```

### 3.3 Catatan tentang dua index unik harian

`attendances_one_checkin_per_day` adalah jantung aturan B1. Tiga sifatnya:

1. **Parsial pada `status <> 'rejected'`** — record yang ditolak admin tidak lagi
   memblokir; karyawan bisa mencatat ulang. Inilah mengapa `rejected` harus
   terminal (B10): kalau bisa dibatalkan, dua record hidup akan bentrok.
2. **Menutup balapan.** Dua request bersamaan → satu berhasil, satu kena unique
   violation yang dipetakan ke `409 ALREADY_CHECKED_IN`. Service **tetap** melakukan
   pemeriksaan lebih dulu, tapi hanya demi pesan error yang ramah — kebenarannya
   dijamin index.
3. **`work_date`, bukan `server_timestamp::date`.** Karena `work_date` sudah
   memperhitungkan timezone dan `workday_cutoff_hour` (§ 2.2), index bekerja benar
   untuk shift malam tanpa perubahan apa pun.

### 3.4 `attendance_attempts` — migration `000020`

Telemetri keamanan. **Tanpa foto, tanpa embedding.**

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | `bigint` | `GENERATED ALWAYS AS IDENTITY`, PK — sama alasannya dengan `audit_logs` ([Fase 1 § 3.8](02-Fase1.md#38-audit_logs--migration-000009)) |
| `employee_id` | `uuid` | `NOT NULL`, FK → `employees(id)` `ON DELETE CASCADE` |
| `type` | `text` | `NOT NULL`, `CHECK (type IN ('check_in','check_out'))` |
| `server_timestamp` | `timestamptz` | `NOT NULL DEFAULT now()` |
| `work_date` | `date` | `NOT NULL` |
| `outcome` | `text` | `NOT NULL`, `CHECK (outcome IN ('matched','below_threshold','face_not_usable','inference_unavailable','no_reference','geofence_rejected','rule_rejected','duplicate_photo','rate_limited','manual_mode'))` |
| `matched_similarity` | `real` | `NULL` |
| `threshold_used` | `real` | `NULL` |
| `model_version` | `text` | `NULL` |
| `quality_score` | `real` | `NULL` |
| `hints` | `jsonb` | `NOT NULL DEFAULT '[]'::jsonb` |
| `geofence_status` | `text` | `NULL` |
| `distance_meter` | `real` | `NULL` |
| `allow_fallback` | `boolean` | `NOT NULL DEFAULT false` |
| `attendance_id` | `uuid` | `NULL`, FK → `attendances(id)` `ON DELETE SET NULL` |
| `request_id` | `text` | `NULL` |
| `ip` | `inet` | `NULL` |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` |

```sql
CREATE INDEX attendance_attempts_employee_idx ON attendance_attempts (employee_id, server_timestamp DESC);
CREATE INDEX attendance_attempts_outcome_idx  ON attendance_attempts (outcome, server_timestamp DESC);
CREATE INDEX attendance_attempts_ratelimit_idx
  ON attendance_attempts (employee_id, server_timestamp DESC) WHERE outcome <> 'matched';
```

Index terakhir melayani pemeriksaan B12 (§ 2.7b) tanpa memindai seluruh riwayat.

Retensi: `attendance.attempt_retention_days` (default 90). Job yang sama dengan
retensi foto menghapus baris lama.

### 3.5 `app_settings` tambahan — migration `000021`

```sql
INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
  ('attendance.workday_cutoff_hour',              '0',      'number',  'Jam pemisah hari kerja (0-11); >0 untuk shift malam', true),
  ('attendance.fallback_enabled',                 'true',   'boolean', 'Izinkan jalur fallback saat verifikasi wajah gagal', true),
  ('attendance.outside_geofence_policy',          '"reject"','string', 'Perlakuan absensi di luar radius: reject | pending_review', true),
  ('attendance.missing_location_policy',          '"reject"','string', 'Perlakuan saat lokasi tidak tersedia/tidak akurat: reject | pending_review', true),
  ('attendance.max_gps_accuracy_meter',           '100',    'number',  'Akurasi GPS terburuk yang masih diterima (meter)', true),
  ('attendance.allow_checkout_without_checkin',   'false',  'boolean', 'Izinkan check-out tanpa check-in di hari yang sama', true),
  ('attendance.min_minutes_between_checkin_checkout','1',   'number',  'Jeda minimum check-in ke check-out (menit)', true),
  ('attendance.require_face_for_checkout',        'true',   'boolean', 'Wajibkan verifikasi wajah saat check-out', true),
  ('attendance.checkout_without_face_status',     '"pending_review"','string','Status check-out saat verifikasi wajah dimatikan', false),
  ('attendance.allow_fallback_without_enrollment','false',  'boolean', 'Izinkan absen fallback bagi karyawan yang belum enroll', false),
  ('attendance.max_note_length',                  '500',    'number',  'Panjang maksimum catatan absensi', true),
  ('attendance.max_failed_attempts_per_hour',     '10',     'number',  'Batas percobaan gagal per karyawan per jam', false),
  ('attendance.photo_retention_days',             '365',    'number',  'Hari sebelum foto absensi dihapus', false),
  ('attendance.attempt_retention_days',           '90',     'number',  'Hari sebelum telemetri percobaan dihapus', false),
  ('attendance.duplicate_photo_window_days',      '7',      'number',  'Rentang deteksi foto identik (anti-replay)', false)
ON CONFLICT (key) DO NOTHING;

-- Rekonsiliasi makna (§ 2.1) — bukan mengubah nilainya, hanya deskripsinya.
UPDATE app_settings
SET description = 'Batas atas dan nilai default radius_meter saat membuat office_location baru. Radius efektif ditentukan per lokasi.'
WHERE key = 'attendance.max_distance_meter';
```

> **`attendance.checkout_without_face_status` tetap `is_public = false` — ini
> disengaja, bukan celah.** Karyawan tidak membacanya lewat `#28 GET /settings`
> (baris `is_public=false` tidak pernah sampai ke pemegang `settings.read` saja,
> [Fase 1 § 4.6](02-Fase1.md#46-app-settings)); ia dibaca lewat `#55 GET
> /attendances/context` (§ 4.4), yang menyuntikkan nilainya secara eksplisit di
> handler. Resolusi [K-01](09-Revisions-Log.md#k-01--checkout_without_face_status-tidak-dapat-dibaca-karyawan--kontradiksi--resolved).
> **Jangan** mengubah `is_public` menjadi `true` sebagai "perbaikan" — itu
> membocorkan kebijakan ini ke `#28` untuk siapa pun yang punya `settings.read`.

**Tidak ada permission baru di Fase 4.** Seluruh 18 endpoint memakai
`attendance.checkin`, `attendance.read_self`, `attendance.read_all`,
`attendance.approve`, dan `location.*` — semuanya sudah di-seed di
[Fase 1 § 2.3](02-Fase1.md#23-katalog-permission).

Validator per-key ditambahkan di `internal/settings/validators.go`:
`outside_geofence_policy` dan `missing_location_policy` hanya menerima
`reject`/`pending_review`; `workday_cutoff_hour` integer 0–11;
`max_gps_accuracy_meter` 5–1000.

### 3.6 Diagram relasi (tambahan Fase 4)

```
employees 1 ──── * attendances ──┬── matched_reference_id ──► face_references  [Fase 3]
    │                            ├── office_location_id ────► office_locations
    │                            └── reviewed_by ───────────► users            [Fase 1]
    └──── * attendance_attempts ──── attendance_id ─────────► attendances
```

---

## 4. Daftar Endpoint / Kontrak API

Base path `/api/v1`. Envelope, `snake_case`, dan katalog error mengikuti
[Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go) + § 2.10.

### 4.1 Ringkasan

| # | Method | Path | Guard |
|---|---|---|---|
| 53 | POST | `/attendances/check-in` | `attendance.checkin` |
| 54 | POST | `/attendances/check-out` | `attendance.checkin` |
| 55 | GET | `/attendances/context` | `attendance.checkin` |
| 56 | GET | `/attendances/me` | `attendance.read_self` |
| 57 | GET | `/attendances/me/today` | `attendance.read_self` |
| 58 | GET | `/attendances/{id}` | `attendance.read_all` \| `attendance.read_self`+self |
| 59 | GET | `/attendances/{id}/photo` | `attendance.read_all` \| `attendance.read_self`+self |
| 60 | GET | `/attendances` | `attendance.read_all` |
| 61 | GET | `/attendances/pending` | `attendance.approve` |
| 62 | POST | `/attendances/{id}/approve` | `attendance.approve` |
| 63 | POST | `/attendances/{id}/reject` | `attendance.approve` |
| 64 | POST | `/attendances/reviews` | `attendance.approve` |
| 65 | GET | `/attendances/attempts` | `attendance.read_all` |
| 66 | GET | `/office-locations` | `location.read` |
| 67 | POST | `/office-locations` | `location.create` |
| 68 | GET | `/office-locations/{id}` | `location.read` |
| 69 | PATCH | `/office-locations/{id}` | `location.update` |
| 70 | DELETE | `/office-locations/{id}` | `location.delete` |

Semua didaftarkan di `internal/httpx/routes.go` sehingga
`TestAllRoutesHaveGuards` ([Fase 1 § 5.3](02-Fase1.md#53-middleware-authenticate--requirepermission))
tetap menjadi jaring pengaman.

> Role `employee` **tidak** punya `location.read`
> ([Fase 1 § 2.4](02-Fase1.md#24-role-default)). Karena itu informasi lokasi yang
> dibutuhkan halaman absensi disajikan lewat `GET /attendances/context` (#55) yang
> dijaga `attendance.checkin` — bukan dengan memperluas permission. Karyawan
> mendapat lokasi terdekat miliknya, bukan seluruh daftar kantor.

### 4.2 `POST /attendances/check-in`

**Request** — `multipart/form-data`:

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `image` | file | ✅ | JPEG/PNG/WebP, ≤ `face.max_image_bytes` |
| `lat` | float | kondisional | Wajib bila `attendance.geofence_enabled` |
| `lng` | float | kondisional | idem |
| `gps_accuracy_meter` | float | ➖ | Dari `GeolocationPosition.coords.accuracy` |
| `capture_source` | text | ✅ | `web_camera` \| `mobile_camera` |
| `note` | text | ➖ | ≤ `attendance.max_note_length` |
| `allow_fallback` | bool | ➖ | Default `false` (§ 2.3) |
| `client_reported_at` | RFC3339 | ➖ | Telemetri saja |

Header opsional: `Idempotency-Key` (§ 2.8).

**Response 201 — verifikasi wajah lolos:**

```json
{
  "data": {
    "id": "018f7a01-...",
    "type": "check_in",
    "status": "approved",
    "method": "face",
    "server_timestamp": "2026-09-04T01:12:47Z",
    "work_date": "2026-09-04",
    "local_time": "08:12:47",
    "timezone": "Asia/Jakarta",
    "geofence_status": "inside",
    "office_location": { "id": "018f70a1-...", "name": "Kantor Pusat" },
    "distance_meter": 23.4,
    "note": null,
    "photo_url": "/api/v1/attendances/018f7a01-.../photo",
    "requires_review": false
  }
}
```

`matched_similarity`, `threshold_used`, dan `model_version` **tidak** ada di
response ini — hanya terlihat oleh pemegang `attendance.read_all` (§ 2.7).

**Response 201 — masuk jalur fallback (`allow_fallback=true`):**

```json
{
  "data": {
    "id": "018f7a02-...",
    "type": "check_in",
    "status": "pending_review",
    "method": "fallback",
    "fallback_reason": "below_threshold",
    "server_timestamp": "2026-09-04T01:14:02Z",
    "work_date": "2026-09-04",
    "geofence_status": "inside",
    "office_location": { "id": "018f70a1-...", "name": "Kantor Pusat" },
    "distance_meter": 23.4,
    "hints": [],
    "photo_url": "/api/v1/attendances/018f7a02-.../photo",
    "requires_review": true,
    "message": "Absensi tercatat dan menunggu persetujuan admin."
  }
}
```

**Response 422 — verifikasi gagal, `allow_fallback=false`:**

```json
{
  "error": {
    "code": "FACE_NOT_MATCHED",
    "message": "Wajah tidak dikenali. Coba lagi dengan pencahayaan lebih baik, atau kirim untuk ditinjau admin.",
    "hints": [],
    "can_fallback": true,
    "request_id": "01JD..."
  }
}
```

`can_fallback` memberi tahu client apakah menawarkan tombol "kirim untuk ditinjau"
masuk akal (nilainya `attendance.fallback_enabled` DAN kebijakan geofence
mengizinkan). Tanpa field ini, Fase 6 harus menebak.

**Katalog error endpoint ini:**

| Status | code | Kondisi |
|---|---|---|
| 403 | `EMPLOYEE_INACTIVE` | B6 |
| 403 | `CONSENT_REQUIRED` | Mode `face` tanpa consent aktif (guard Fase 3) |
| 409 | `ALREADY_CHECKED_IN` | B1 |
| 409 | `DUPLICATE_PHOTO` | Anti-replay (§ 2.7c) |
| 422 | `FACE_NOT_ENROLLED` | B7 |
| 422 | `FACE_NOT_USABLE` | `usable=false`, `allow_fallback=false` — `hints` disertakan |
| 422 | `FACE_NOT_MATCHED` | Di bawah threshold, `allow_fallback=false` |
| 422 | `OUTSIDE_GEOFENCE` | Kebijakan `reject` |
| 422 | `LOCATION_REQUIRED` / `LOCATION_INACCURATE` | Geofence aktif, koordinat tidak ada / akurasi buruk |
| 422 | `VALIDATION_ERROR` | `note` terlalu panjang, `capture_source` tidak valid, dll. |
| 413 / 415 | `PAYLOAD_TOO_LARGE` / `UNSUPPORTED_MEDIA_TYPE` | Diteruskan dari inference |
| 429 | `TOO_MANY_FAILED_ATTEMPTS` | B12; `Retry-After` disertakan |
| 502 / 504 | `UPSTREAM_ERROR` / `UPSTREAM_TIMEOUT` | Inference bermasalah **dan** `allow_fallback=false` |
| 503 | `FACE_SERVICE_NOT_CONFIGURED` | `face.model_version = "unset"` (§ 2.10, K-04) |
| 503 | `ATTENDANCE_NOT_CONFIGURED` | Geofence aktif tanpa lokasi aktif (§ 2.10) |

### 4.3 `POST /attendances/check-out`

Request identik dengan check-in. Perbedaan hanya pada aturan bisnis (B2–B5) dan
pada kemungkinan `require_face_for_checkout = false` (§ 2.6).

Response menambahkan:

```json
"work_duration_minutes": 512,
"checkin_id": "018f7a01-..."
```

| Status | code | Kondisi |
|---|---|---|
| 409 | `ALREADY_CHECKED_OUT` | B2 |
| 409 | `CHECKOUT_WITHOUT_CHECKIN` | B3 |
| 409 | `CHECKOUT_TOO_SOON` | B5; `details` menyebut sisa menitnya |

`work_duration_minutes` dihitung dari selisih `server_timestamp` kedua record —
angka mentah, tanpa penafsiran lembur/keterlambatan (itu Fase 5).

### 4.4 `GET /attendances/context`

Satu panggilan yang menjawab semua yang dibutuhkan halaman absensi (Fase 6/7)
**sebelum** kamera dinyalakan.

```json
{
  "data": {
    "server_time": "2026-09-04T01:10:00Z",
    "work_date": "2026-09-04",
    "timezone": "Asia/Jakarta",
    "attendance_mode": "face",
    "is_enrolled": true,
    "active_reference_count": 3,
    "needs_re_enrollment": false,
    "consent_status": "granted",
    "today": {
      "check_in":  { "id": "018f7a01-...", "server_timestamp": "...", "status": "approved" },
      "check_out": null
    },
    "next_action": "check_out",
    "geofence": {
      "enabled": true,
      "outside_policy": "reject",
      "max_gps_accuracy_meter": 100,
      "nearest_location": {
        "id": "018f70a1-...", "name": "Kantor Pusat",
        "lat": -6.2001, "lng": 106.8166, "radius_meter": 100
      }
    },
    "photo": { "max_bytes": 6291456, "accepted_mime_types": ["image/jpeg","image/png","image/webp"] },
    "fallback_enabled": true,
    "require_face_for_checkout": true,
    "checkout_without_face_status": "pending_review",
    "max_note_length": 500
  }
}
```

`server_time` di sini adalah cara client mengoreksi tampilannya tanpa pernah
mengirimkan waktu yang dipercaya — sekaligus memungkinkan Fase 6 menampilkan jam
yang benar meski jam perangkat salah.

`nearest_location` dihitung dari koordinat **terakhir yang diketahui** bila client
mengirim `?lat=&lng=`; tanpa itu, dikembalikan lokasi aktif pertama. Daftar lengkap
kantor tidak pernah dikirim ke karyawan.

`require_face_for_checkout`, `checkout_without_face_status`, dan `max_note_length`
**diinjeksikan eksplisit oleh handler `context`** dari `app_settings`, bukan lewat
mekanisme generik `#28 GET /settings`. Ini keputusan yang disengaja
([REV-EP-04](09-Revisions-Log.md#b-perubahan-katalog-endpoint), resolusi
[K-01](09-Revisions-Log.md#k-01--checkout_without_face_status-tidak-dapat-dibaca-karyawan--kontradiksi--resolved)):
`attendance.checkout_without_face_status` **tetap** `is_public = false` di
`app_settings` (§ 3.5) — karyawan **tidak pernah** membacanya lewat `#28`, hanya
lewat endpoint ini, yang dijaga `attendance.checkin` dan hanya membocorkan tiga
field yang memang dibutuhkan halaman absensi, bukan seluruh baris setting.
**Jangan** "memperbaiki" ini dengan mengubah `is_public` menjadi `true` di
kemudian hari — itu membocorkan detail kebijakan verifikasi ke `#28` untuk siapa
pun yang punya `settings.read`, termasuk yang bukan karyawan yang login sebagai
dirinya sendiri.

### 4.5 Riwayat pribadi

#### `GET /attendances/me` — guard `attendance.read_self`

Query: `page`, `per_page` (maks 100), `from`, `to` (tanggal), `type`, `status`.

```json
{
  "data": [
    {
      "id": "018f7a01-...",
      "type": "check_in",
      "status": "approved",
      "method": "face",
      "fallback_reason": null,
      "server_timestamp": "2026-09-04T01:12:47Z",
      "work_date": "2026-09-04",
      "geofence_status": "inside",
      "office_location": { "id": "...", "name": "Kantor Pusat" },
      "note": null,
      "photo_url": "/api/v1/attendances/018f7a01-.../photo",
      "reviewed_at": null,
      "review_note": null
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 84, "total_pages": 5 }
}
```

Tanpa `matched_similarity`, `threshold_used`, `model_version`, `quality_score`,
maupun `distance_meter` (§ 2.7).

#### `GET /attendances/me/today`
Bentuk ringkas untuk widget: `{check_in, check_out, work_duration_minutes, next_action}`.

### 4.6 Admin

#### `GET /attendances` — guard `attendance.read_all`

Query: `page`, `per_page`, `employee_id`, `department`, `from`, `to`, `type`,
`status`, `method`, `geofence_status`, `office_location_id`,
`sort` (whitelist: `server_timestamp`, `-server_timestamp`, `work_date`, `-work_date`).

Item response **lengkap** — menambahkan yang disembunyikan dari karyawan:

```json
{
  "matched_similarity": 0.6721,
  "threshold_used": 0.42,
  "model_version": "buffalo_l@v1",
  "quality_score": 0.78,
  "hints": [],
  "distance_meter": 23.4,
  "gps_accuracy_meter": 12.0,
  "capture_source": "web_camera",
  "clock_skew_seconds": -3,
  "attendance_mode": "face",
  "employee": { "id": "...", "employee_number": "EMP-0002", "full_name": "Budi Santoso", "department": "IT" },
  "reviewed_by": { "id": "...", "email": "hr@faceclock.local" }
}
```

`sort` memakai whitelist kolom, tidak pernah menyusun `ORDER BY` dari input mentah
([Fase 1 E35](02-Fase1.md#63-data--konkurensi)).

#### `GET /attendances/pending` — guard `attendance.approve`
Sama seperti #60 tapi terkunci `status = 'pending_review'`, diurutkan tertua dulu
(antrian, bukan tumpukan), dan menyertakan `waiting_hours`.

#### `POST /attendances/{id}/approve`

```json
{ "review_note": "Verifikasi manual: wajah cocok, pencahayaan buruk saat capture" }
```

```json
// 200
{ "data": { "id": "...", "status": "approved", "reviewed_by": {...}, "reviewed_at": "..." } }
```

Efek: `status='approved'`, `reviewed_by`, `reviewed_at=now()`.
`server_timestamp` **tidak pernah** berubah — persetujuan mengubah status
kehadiran, bukan kapan kehadiran itu terjadi.

| Status | code | Kondisi |
|---|---|---|
| 409 | `ATTENDANCE_ALREADY_REVIEWED` | B10 |
| 403 | `SELF_REVIEW_DENIED` | B11 |
| 404 | `NOT_FOUND` | |

#### `POST /attendances/{id}/reject`
Body: `{ "review_note": "..." }` — **wajib**, 3–500 karakter. Menolak tanpa alasan
membuat karyawan tidak tahu harus berbuat apa, dan membuat penolakan tidak bisa
diaudit.

Efek tambahan: karena `rejected` tidak lagi terhitung di index unik harian (§ 3.3),
karyawan **bisa** mencatat ulang di hari yang sama. Response menyebutkannya
eksplisit (`"employee_can_retry": true`) supaya admin sadar konsekuensinya.

#### `POST /attendances/reviews` — bulk

```json
{
  "items": [
    { "id": "018f7a02-...", "action": "approve", "review_note": "OK" },
    { "id": "018f7a03-...", "action": "reject",  "review_note": "Foto bukan wajah karyawan" }
  ]
}
```

Maksimum 50 item. Diproses per item, **bukan** all-or-nothing:

```json
{
  "data": {
    "succeeded": 1,
    "failed": 1,
    "results": [
      { "id": "018f7a02-...", "status": "approved" },
      { "id": "018f7a03-...", "error": { "code": "ATTENDANCE_ALREADY_REVIEWED" } }
    ]
  }
}
```

Satu item yang sudah di-review oleh admin lain tidak boleh menggagalkan 49 lainnya.
Setiap item tetap menjalankan B10 dan B11 secara penuh — bulk bukan jalan pintas
melewati aturan.

#### `GET /attendances/attempts` — guard `attendance.read_all`
Telemetri percobaan (§ 2.7a). Filter: `employee_id`, `outcome`, `from`, `to`.
Dasar bagi dashboard "karyawan dengan percobaan gagal terbanyak" di Fase 5.

### 4.7 `office_locations`

| Method | Path | Guard | Catatan |
|---|---|---|---|
| GET | `/office-locations` | `location.read` | filter `is_active`, `include_deleted` |
| POST | `/office-locations` | `location.create` | `radius_meter` default dari `attendance.max_distance_meter` |
| GET | `/office-locations/{id}` | `location.read` | |
| PATCH | `/office-locations/{id}` | `location.update` | |
| DELETE | `/office-locations/{id}` | `location.delete` | soft delete |

```json
// POST request
{ "name": "Kantor Pusat", "address": "Jl. ...", "lat": -6.2001, "lng": 106.8166, "radius_meter": 100 }
```

| Status | code | Kondisi |
|---|---|---|
| 409 | `CONFLICT` | Nama sudah dipakai lokasi aktif |
| 422 | `VALIDATION_ERROR` | `radius_meter` > `attendance.max_distance_meter`, atau koordinat di luar rentang |
| 409 | `CONFLICT` | `DELETE`/nonaktifkan lokasi **terakhir** yang aktif sementara `attendance.geofence_enabled = true` — akan mengunci seluruh organisasi dari absensi |

Constraint terakhir sejenis dengan aturan "super admin terakhir"
[Fase 1 § 5.5 B](02-Fase1.md#55-pencegahan-privilege-escalation): diperiksa di
dalam transaksi yang sama dengan perubahannya, dengan `FOR UPDATE`, supaya dua
admin yang menghapus dua lokasi terakhir bersamaan tidak berhasil keduanya.

Perubahan `lat`/`lng`/`radius_meter` menulis `audit_logs` dengan nilai lama dan
baru — menggeser lokasi kantor 500 meter adalah cara paling halus untuk membuat
absensi dari rumah menjadi sah.

---

## 5. Flow

### 5.1 Check-in — alur lengkap

```
POST /attendances/check-in
──────────────────────────────────────────────────────────────────────
 0. Authenticate → RequirePermission("attendance.checkin")   [Fase 0/1]
    → RequireConsent(self)  (hanya bila employees.attendance_mode='face')  [Fase 3]

 1. Idempotency-Key ada dan sudah pernah dipakai (< 24 jam)?
       ⇒ kembalikan response tersimpan. TIDAK memanggil inference, TIDAK insert.

 2. Muat employee dari Principal.EmployeeID
       tidak ada / deleted_at / employment_status ≠ 'active'  ⇒ 403 EMPLOYEE_INACTIVE

 3. Kesiapan konfigurasi
       app_settings.face.model_version = "unset"                     ⇒ 503 FACE_SERVICE_NOT_CONFIGURED
       geofence_enabled DAN nol office_locations aktif               ⇒ 503 ATTENDANCE_NOT_CONFIGURED

 4. Rate limit B12
       count(attendance_attempts WHERE employee_id=$1
             AND outcome <> 'matched' AND server_timestamp > now()-'1h')
             >= attendance.max_failed_attempts_per_hour
       ⇒ attempt{outcome:'rate_limited'} ; 429 + Retry-After

 5. Validasi bentuk: MIME dari magic bytes, ukuran, panjang note, capture_source
       gagal ⇒ 4xx. Belum ada apa pun yang disimpan.

 6. Waktu server (satu query, nilai dipakai seluruh transaksi):
       SELECT now() AS ts,
              ((now() AT TIME ZONE $tz) - make_interval(hours=>$cutoff))::date AS work_date

 7. Aturan harian (pemeriksaan ramah; kebenaran dijamin index di langkah 14)
       check_in sudah ada non-rejected di work_date ⇒ attempt{'rule_rejected'} ; 409 ALREADY_CHECKED_IN

 8. Anti-replay: sha256(foto) sudah ada di attendances employee ini
       dalam attendance.duplicate_photo_window_days hari
       ⇒ attempt{'duplicate_photo'} ; 409 DUPLICATE_PHOTO

 9. GEOFENCE — dievaluasi SEBELUM foto disimpan & SEBELUM inference dipanggil
       geo.Evaluate(lat, lng, accuracy, lokasi aktif)  → status, office_location, distance
       status 'unavailable' & policy 'reject' ⇒ attempt{'geofence_rejected'} ; 422 LOCATION_REQUIRED|LOCATION_INACCURATE
       status 'outside'     & policy 'reject' ⇒ attempt{'geofence_rejected'} ; 422 OUTSIDE_GEOFENCE
       status 'outside'/'unavailable' & policy 'pending_review'
             ⇒ tandai forced_pending = true, fallback_reason = 'outside_geofence' | 'location_unavailable'

10. MODE MANUAL  (employees.attendance_mode = 'manual')
       lewati langkah 11–13 seluruhnya
       method='fallback', status='pending_review', fallback_reason='manual_mode'
       lompat ke langkah 14

11. Referensi wajah
       SELECT count(*) FROM face_references
       WHERE employee_id=$1 AND is_active AND model_version = app_settings.face.model_version
       < face.min_reference_photos ?
          allow_fallback_without_enrollment = false ⇒ attempt{'no_reference'} ; 422 FACE_NOT_ENROLLED
          true                                      ⇒ fallback_reason='no_reference' (butuh allow_fallback)

12. INFERENCE  [klien Go Fase 2]
       result, err := inference.Embed(ctx, bytes)
       err (timeout / 5xx / circuit terbuka):
            allow_fallback = false ⇒ attempt{'inference_unavailable'} ; 502/504
            allow_fallback = true  ⇒ fallback_reason='inference_unavailable' ; lompat ke 14
       result.Usable == false:
            allow_fallback = false ⇒ attempt{'face_not_usable', hints} ; 422 FACE_NOT_USABLE + hints
            allow_fallback = true  ⇒ fallback_reason='face_not_usable' ; lompat ke 14
       result.Usable == true && result.Embedding == nil:
            pelanggaran kontrak Fase 2 ⇒ 502 UPSTREAM_ERROR, log error, metrik naik
       result.ModelVersion ≠ app_settings.face.model_version:
            ⇒ perlakukan seperti 'no_reference' (vektornya tidak sebanding)

13. PERBANDINGAN — di Postgres, sesuai keputusan D11 Fase 2
       SELECT id, 1 - (embedding <=> $1::vector) AS similarity
       FROM face_references
       WHERE employee_id = $2 AND is_active = true AND model_version = $3
       ORDER BY embedding <=> $1::vector
       LIMIT 1;                          ← "similarity TERTINGGI", master plan § 4

       threshold := app_settings.face.similarity_threshold
       similarity >= threshold ?
          ya    ⇒ method='face' ; status = forced_pending ? 'pending_review' : 'approved'
          tidak ⇒ allow_fallback = false ⇒ attempt{'below_threshold', similarity} ; 422 FACE_NOT_MATCHED
                  allow_fallback = true  ⇒ method='fallback', fallback_reason='below_threshold',
                                           status='pending_review'
                                           (matched_similarity TETAP disimpan — buktinya berharga)

14. PENULISAN
       a. Put foto ke storage: attendance/{work_date}/{employee_id}/{ulid}.jpg
       b. BEGIN
            INSERT INTO attendances (...) VALUES (..., now(), $work_date, ...)
              unique violation pada attendances_one_checkin_per_day
                 ⇒ ROLLBACK ; hapus objek foto ; 409 ALREADY_CHECKED_IN
          COMMIT
       c. INSERT attendance_attempts { outcome, attendance_id }
       d. audit_logs: attendance.checked_in { type, status, method, fallback_reason }
                       (TANPA similarity mentah? tidak — similarity BOLEH, ia angka;
                        yang dilarang adalah embedding dan byte foto)

15. Response 201
```

Tiga sifat yang perlu dicatat karena mudah hilang saat implementasi:

- **Foto disimpan hanya bila record akan dibuat.** Semua penolakan di langkah 2–13
  terjadi sebelum langkah 14a. Request yang ditolak tidak meninggalkan foto wajah
  di object storage.
- **Geofence dievaluasi sebelum inference.** Menolak lebih awal menghemat panggilan
  inference dan, lebih penting, tidak memproses data biometrik untuk permintaan
  yang memang tidak akan sah.
- **`allow_fallback=true` tidak melewati verifikasi.** Langkah 12 dan 13 tetap
  berjalan penuh. Bendera itu hanya mengubah percabangan setelah kegagalan (§ 2.3).

### 5.2 Check-out

Identik sampai langkah 9, lalu:

```
 7'. Aturan harian
       check_out non-rejected sudah ada di work_date ⇒ 409 ALREADY_CHECKED_OUT
       cari check_in non-rejected di work_date:
          tidak ada:
             attendance.allow_checkout_without_checkin = false ⇒ 409 CHECKOUT_WITHOUT_CHECKIN
             true ⇒ lanjut, tandai fallback_reason bila diperlukan
          ada:
             now() - checkin.server_timestamp < min_minutes ⇒ 409 CHECKOUT_TOO_SOON
                (details: "sisa X menit")
12'. attendance.require_face_for_checkout = false ?
       ⇒ lewati 12–13
         method='fallback', fallback_reason='checkout_without_face'
         status = attendance.checkout_without_face_status  (default 'pending_review')
```

`work_duration_minutes` dihitung saat menyusun response, tidak disimpan — ia
turunan dari dua `server_timestamp` dan menyimpannya menciptakan kemungkinan
tidak sinkron bila salah satu record di-review ulang.

### 5.3 Review

```
POST /attendances/{id}/approve | /reject
 1. Authenticate → RequirePermission("attendance.approve")
 2. BEGIN
      SELECT * FROM attendances WHERE id=$1 FOR UPDATE
        tidak ada                         ⇒ 404
        status <> 'pending_review'        ⇒ 409 ATTENDANCE_ALREADY_REVIEWED
        employee_id = Principal.EmployeeID ⇒ 403 SELF_REVIEW_DENIED     [B11]
      UPDATE attendances
        SET status = $2, reviewed_by = $3, reviewed_at = now(),
            review_note = $4, updated_at = now()
        WHERE id = $1
      -- server_timestamp TIDAK disentuh
    COMMIT
 3. audit_logs: attendance.approved | attendance.rejected
      { attendance_id, employee_id, from_status, to_status, review_note }
```

`FOR UPDATE` menutup balapan dua admin yang menekan tombol bersamaan: yang kedua
membaca status yang sudah berubah dan mendapat `409`.

### 5.4 Job retensi (harian)

```
worker retensi (pg_try_advisory_lock, sekali sehari):
  A. Foto absensi
       SELECT id, photo_key FROM attendances
       WHERE photo_purged_at IS NULL
         AND photo_key IS NOT NULL
         AND status <> 'pending_review'
         AND coalesce(reviewed_at, server_timestamp)
             < now() - make_interval(days => attendance.photo_retention_days)
       LIMIT 500
       per baris:
         BEGIN
           UPDATE attendances SET photo_key = NULL, photo_purged_at = now() WHERE id=$1
         COMMIT
         hapus objek dari storage (best-effort; gagal ⇒ disapu job objek yatim)
  B. Telemetri percobaan
       DELETE FROM attendance_attempts
       WHERE server_timestamp < now() - make_interval(days => attendance.attempt_retention_days)
  C. audit_logs: attendance.photos_purged { count, oldest, newest }
```

Urutan di A disengaja (DB dulu, storage kemudian) — alasannya sama dengan
[Fase 3 § 2.8](04-Fase3.md#28-penghapusan-data-biometrik-uu-pdp): baris yang
menunjuk ke objek yang sudah hilang lebih buruk daripada objek yatim yang bisa
disapu.

`status <> 'pending_review'` melindungi bukti untuk perkara yang belum diputus.

---

## 6. Edge Case & Validasi

### 6.1 Waktu & aturan harian

| # | Kondisi | Penanganan |
|---|---|---|
| E1 | Client mengirim `client_reported_at` yang dimanipulasi | Disimpan sebagai telemetri; `clock_skew_seconds` dihitung. **Tidak pernah** memengaruhi `server_timestamp` maupun `work_date`. Skew > 300 detik menulis log `warn` |
| E2 | Jam perangkat mundur/maju drastis | Tidak berpengaruh sama sekali — server tidak membaca waktu client |
| E3 | Check-in pukul 23:58, check-out pukul 00:05 | Dengan `workday_cutoff_hour = 0` keduanya jatuh di `work_date` berbeda → `CHECKOUT_WITHOUT_CHECKIN`. Solusinya bukan tambalan kode, melainkan menyetel `workday_cutoff_hour` (mis. `4`). Didokumentasikan sebagai konfigurasi, bukan bug |
| E4 | Dua tab menekan check-in bersamaan | Index unik parsial → satu `201`, satu `409`. Foto request yang kalah dihapus (langkah 14b) |
| E5 | Check-in ditolak admin, lalu karyawan check-in lagi hari itu | Diizinkan — index tidak menghitung `rejected`. Response reject sudah memberi tahu admin (`employee_can_retry`) |
| E6 | Admin mencoba membatalkan `rejected` | Ditolak `409` — `rejected` terminal (B10). Kalau perlu diperbaiki, karyawan mencatat ulang dan admin menyetujui yang baru; jejaknya jadi lebih jujur daripada mengedit record lama |
| E7 | `attendance.timezone` diubah di tengah hari | Record lama tetap memakai `work_date` yang sudah tersimpan. Perubahan hanya memengaruhi record baru. Perubahan setting ini menulis audit dan menampilkan peringatan di panel (Fase 5) |
| E8 | Karyawan pindah zona waktu (dinas luar kota) | `work_date` selalu memakai `attendance.timezone` organisasi, bukan zona perangkat. Konsisten dan bisa dijelaskan |

### 6.2 Wajah & inference

| # | Kondisi | Penanganan |
|---|---|---|
| E9 | Inference mati, `allow_fallback=false` | `504`/`502`. **Tidak ada** record dibuat. Tidak pernah `approved` — aturan [Fase 2 § 2.6](03-Fase2.md#26-klien-go-internalinference) |
| E10 | Inference mati, `allow_fallback=true` | `pending_review`, `fallback_reason='inference_unavailable'`. Manusia yang memutuskan |
| E11 | Circuit breaker terbuka | Sama seperti E9/E10, tapi gagal dalam < 50 ms alih-alih menunggu 6 detik |
| E12 | `usable=true` tapi `embedding=nil` | Pelanggaran kontrak Fase 2 → `502 UPSTREAM_ERROR`, metrik `inference_contract_violation_total`. **Tidak pernah** dianggap "tidak cocok" — membedakan bug dari kegagalan verifikasi itu penting |
| E13 | `model_version` inference ≠ `app_settings.face.model_version` | Diperlakukan seperti `no_reference`. Vektor dari model berbeda tidak pernah dibandingkan ([Fase 2 § 2.2](03-Fase2.md#22-d10--pilihan-model--butuh-konfirmasi)) |
| E14 | Karyawan punya referensi, tapi semuanya `model_version` lama (reindex gagal untuknya) | Query langkah 11 memfilter `model_version` → hitungannya 0 → `FACE_NOT_ENROLLED`. Karyawan diarahkan enroll ulang ([Fase 3 § 4.4](04-Fase3.md#44-manajemen-referensi) `needs_re_enrollment`) |
| E15 | Similarity persis sama dengan threshold | `>=` → lolos. Ditulis eksplisit di kode dan diuji, supaya tidak berubah diam-diam saat refactor |
| E16 | Karyawan kembar identik | Bisa saling lolos. Dimitigasi di Fase 3 (deteksi duplikat saat enroll, [§ 2.6](04-Fase3.md#26-d16--deteksi-wajah-duplikat-antar-karyawan--butuh-konfirmasi)) dan oleh `force_duplicate` yang tercatat di audit. Fase 4 **tidak** bisa membedakannya — keterbatasan yang dinyatakan, bukan disembunyikan |
| E17 | Consent dicabut pagi ini, karyawan absen siang | Guard `RequireConsent` → `403 CONSENT_REQUIRED`. Referensinya juga sudah dinonaktifkan Fase 3 |

### 6.3 Lokasi

| # | Kondisi | Penanganan |
|---|---|---|
| E18 | `lat`/`lng` tidak dikirim, geofence aktif | `422 LOCATION_REQUIRED` (kebijakan default) |
| E19 | Akurasi GPS 800 m (indoor) | `geofence_status='unavailable'` → `422 LOCATION_INACCURATE`. Ambangnya `attendance.max_gps_accuracy_meter` supaya bisa dilonggarkan di gedung bertingkat |
| E20 | Koordinat di luar rentang valid (lat 200) | `422 VALIDATION_ERROR` sebelum perhitungan apa pun |
| E21 | GPS palsu (mock location) | **Tidak terdeteksi di Fase 4.** Server tidak punya cara membedakannya. Mitigasi: deteksi mock location on-device di Fase 7 + telemetri (koordinat yang persis sama berulang kali terlihat di `attendance_attempts`). Dinyatakan sebagai keterbatasan |
| E22 | Berada di dalam radius gudang tapi lebih dekat ke kantor pusat berradius kecil | `inside`, dengan `office_location` = gudang (§ 2.5 langkah 5) |
| E23 | Semua lokasi dinonaktifkan sementara geofence aktif | `503 ATTENDANCE_NOT_CONFIGURED` saat absen, dan `409` saat mencoba menonaktifkan yang terakhir (§ 4.7) |
| E24 | Admin menggeser koordinat kantor | Diizinkan, tapi menulis `audit_logs` dengan nilai lama & baru. Absensi lama tidak dihitung ulang — `distance_meter` yang tersimpan adalah fakta saat itu |

### 6.4 Keamanan & integritas

| # | Kondisi | Penanganan |
|---|---|---|
| E25 | Admin menyetujui absensinya sendiri | `403 SELF_REVIEW_DENIED` (B11) |
| E26 | Dua admin me-review record yang sama bersamaan | `FOR UPDATE`; yang kedua `409 ATTENDANCE_ALREADY_REVIEWED` |
| E27 | Foto lama dikirim ulang setiap hari | `409 DUPLICATE_PHOTO` (§ 2.7c) |
| E28 | Foto dari galeri, bukan kamera | **Tidak bisa dibuktikan server.** `capture_source` adalah klaim. Mitigasi: Fase 6 (`getUserMedia` tanpa input file), Fase 7 (kamera in-app + liveness). Nilainya disimpan supaya audit bisa menyaringnya |
| E29 | Karyawan mengakses `/attendances/{id}` milik orang lain | `404` (pola ownership [Fase 1 § 5.4](02-Fase1.md#54-pola-akses-self)) |
| E30 | Karyawan menebak-nebak untuk membaca `matched_similarity` | Field-nya tidak ada di response yang dijaga `attendance.read_self` — bukan disembunyikan di UI, tapi tidak diserialisasi sama sekali (§ 2.7). Diuji |
| E31 | Percobaan berulang untuk mencari foto yang lolos | `429 TOO_MANY_FAILED_ATTEMPTS` + jejak lengkap di `attendance_attempts` |
| E32 | `UPDATE` manual di database membuat fallback `approved` tanpa penyetuju | Ditolak `CHECK attendances_fallback_never_approved_chk` (§ 3.2) |
| E33 | Log memuat byte foto / embedding | Dilarang [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go); diuji ulang di fase ini |
| E34 | `note` berisi data pribadi | `note` dilarang masuk log (aturan Fase 0), disimpan di DB, dan ikut terhapus bila record dihapus |

### 6.5 Data & operasional

| # | Kondisi | Penanganan |
|---|---|---|
| E35 | Storage penuh saat menyimpan foto | `503`; **tidak ada** record dibuat. Absensi yang tercatat tanpa bukti foto lebih buruk daripada absensi yang gagal dan bisa diulang |
| E36 | Insert gagal setelah foto tersimpan | Objek dihapus (best-effort) + job objek yatim menyapunya |
| E37 | Karyawan di-soft-delete setelah punya absensi | `ON DELETE RESTRICT` pada `employee_id`; riwayat absensi tidak pernah ikut hilang |
| E38 | Foto sudah di-purge lalu diminta | `410 Gone` dengan `code: NOT_FOUND` (pola sama dengan [Fase 3 § 4.4](04-Fase3.md#44-manajemen-referensi)) |
| E39 | `matched_reference_id` menunjuk referensi yang kemudian dinonaktifkan | `ON DELETE SET NULL` hanya berlaku untuk hard delete; penonaktifan tidak mengubah apa pun. Record absensi tetap menyimpan bukti referensi mana yang cocok saat itu |
| E40 | `per_page=100000` / `page=-1` | Di-clamp (100 / 1), tidak error — konsisten dengan [Fase 1 E33/E34](02-Fase1.md#63-data--konkurensi) |
| E41 | Retensi menghapus foto record yang masih `pending_review` | Tidak terjadi — filter `status <> 'pending_review'` (§ 5.4) |
| E42 | Threshold diubah admin | Record lama tetap menyimpan `threshold_used` saat itu; tidak ada perhitungan ulang. Perubahan menulis audit |

### 6.6 Validasi input

| Field | Aturan |
|---|---|
| `image` | wajib; MIME dari magic bytes; ≤ `face.max_image_bytes` |
| `lat` | wajib bila geofence aktif; `-90..90` |
| `lng` | wajib bila geofence aktif; `-180..180` |
| `gps_accuracy_meter` | opsional; `>= 0`; `> max_gps_accuracy_meter` → `unavailable` |
| `capture_source` | wajib; `web_camera` \| `mobile_camera` |
| `note` | opsional; ≤ `attendance.max_note_length`; di-trim; dilarang masuk log |
| `allow_fallback` | boolean; default `false` |
| `client_reported_at` | opsional; RFC3339; hanya telemetri |
| `Idempotency-Key` | opsional; ULID/UUID; berlaku 24 jam |
| `review_note` | wajib untuk `reject`, 3–500 karakter; opsional untuk `approve` |
| `radius_meter` | 10..`attendance.max_distance_meter` |
| `sort` | whitelist kolom saja |

---

## 7. Struktur Folder

```
apps/faceclock-api/
├── internal/
│   ├── attendance/                     # ← BARU
│   │   ├── handler.go                  # #53–#65
│   │   ├── service.go                  # orkestrasi
│   │   ├── checkin.go                  # alur § 5.1 (dipakai check-in & check-out)
│   │   ├── rules.go                    # B1–B12, fungsi murni sedapat mungkin
│   │   ├── verify.go                   # panggil inference + query pgvector (§ 5.1 langkah 12-13)
│   │   ├── review.go                   # approve / reject / bulk (§ 5.3)
│   │   ├── attempts.go                 # telemetri + rate limit B12
│   │   ├── idempotency.go              # § 2.8
│   │   ├── context.go                  # #55
│   │   ├── photo.go                    # streaming foto (#59)
│   │   ├── retention.go                # job harian (§ 5.4)
│   │   ├── repository.go
│   │   ├── dto.go                      # DTO employee-view vs admin-view TERPISAH (§ 2.7)
│   │   ├── rules_test.go
│   │   ├── checkin_test.go
│   │   ├── verify_test.go
│   │   ├── review_test.go
│   │   └── idempotency_test.go
│   ├── geo/                            # ← BARU
│   │   ├── haversine.go                # fungsi murni, tanpa I/O
│   │   ├── geofence.go                 # Evaluate() (§ 2.5)
│   │   ├── haversine_test.go
│   │   └── geofence_test.go
│   ├── location/                       # ← BARU
│   │   ├── handler.go                  # #66–#70
│   │   ├── service.go                  # termasuk penjagaan "lokasi aktif terakhir"
│   │   ├── repository.go
│   │   └── service_test.go
│   ├── settings/
│   │   └── validators.go               # (diubah) validator 15 key baru
│   ├── storage/
│   │   └── keys.go                     # (diubah) skema key attendance/
│   └── httpx/
│       └── routes.go                   # (diubah) daftarkan #53–#70 + guard
├── migrations/
│   ├── 000018_create_office_locations.{up,down}.sql
│   ├── 000019_create_attendances.{up,down}.sql
│   ├── 000020_create_attendance_attempts.{up,down}.sql
│   └── 000021_attendance_settings.{up,down}.sql
└── test/
    ├── integration/
    │   ├── checkin_test.go
    │   ├── checkout_test.go
    │   ├── fallback_test.go
    │   ├── geofence_test.go
    │   ├── review_test.go
    │   ├── attendance_concurrency_test.go
    │   ├── attendance_security_test.go
    │   ├── location_test.go
    │   └── rbac_matrix_test.go         # (diperluas) + 18 endpoint Fase 4
    └── fixtures/
```

Dua catatan struktur:

- **`internal/geo` sengaja bebas I/O.** Ia menerima angka dan daftar lokasi,
  mengembalikan hasil. Itu membuat seluruh logika geofence — bagian yang paling
  mudah salah dan paling sulit direproduksi di lapangan — bisa diuji dengan tabel
  kasus, termasuk garis khatulistiwa, meridian 180°, dan jarak nol.
- **`dto.go` memisahkan tampilan karyawan dan admin sebagai dua tipe berbeda**,
  bukan satu tipe dengan field yang dikosongkan. Field yang tidak ada di struct
  tidak bisa bocor karena lupa dikosongkan di satu cabang kode (§ 2.7, E30).

Dokumen yang dihasilkan:

```
docs/
├── api/
│   └── fase4-attendance.md              # kontrak #53–#70
└── attendance/
    ├── business-rules.md                # B1–B12 + alasan tiap aturan
    ├── geofence-policy.md               # algoritma, kebijakan, cara tuning
    └── attendance-retention-policy.md   # § 2.9
```

---

## 8. Checklist Task

### 8.0 Prasyarat
- [ ] **Konfirmasi D17–D19** (§ 2.0)
- [ ] Fase 3 selesai: minimal satu karyawan uji punya 3 referensi aktif
- [ ] ⛔ **Dataset B sudah dikumpulkan dan `face.similarity_threshold` bukan angka provisional** (§ 9)

### 8.1 Migration & skema
- [ ] `000018_create_office_locations`
- [ ] `000019_create_attendances` — termasuk **6 CHECK constraint** integritas (§ 3.2)
- [ ] Dua index unik parsial harian + index idempotensi + index parsial `pending_review`
- [ ] `000020_create_attendance_attempts`
- [ ] `000021_attendance_settings` — 15 key + rekonsiliasi deskripsi `max_distance_meter`
- [ ] Semua `.down.sql` ditulis & diuji (turun ke 000017, naik lagi)
- [ ] Test SQL langsung: `INSERT` fallback+approved tanpa `reviewed_by` **harus ditolak** database

### 8.2 Geofence
- [ ] `internal/geo/haversine.go` + tabel uji (jarak nol, khatulistiwa, meridian 180°, antipoda)
- [ ] `internal/geo/geofence.go` — `Evaluate()` sesuai § 2.5, termasuk aturan "himpunan yang mencakup"
- [ ] Test: di dalam gudang radius besar meski lebih dekat ke kantor radius kecil → `inside` (E22)
- [ ] Test: akurasi buruk → `unavailable`

### 8.3 Check-in / check-out
- [ ] `internal/attendance/rules.go` — B1–B12 sebagai fungsi yang bisa diuji terpisah
- [ ] `checkin.go` — alur § 5.1 lengkap, urutan langkah **persis** (geofence sebelum inference)
- [ ] `verify.go` — panggil `inference.Embed`, lalu query pgvector "similarity tertinggi"
- [ ] Penegakan kontrak Fase 2: `usable=true` + `embedding=nil` → `502` (E12)
- [ ] Penanganan `model_version` tidak cocok (E13)
- [ ] `POST /attendances/check-in` (#53)
- [ ] `POST /attendances/check-out` (#54) + `work_duration_minutes`
- [ ] `require_face_for_checkout = false` (D19) + `checkout_without_face_status`
- [ ] Mode `manual` melewati verifikasi dan selalu `pending_review` (B8)
- [ ] `idempotency.go` + penyimpanan hasil 24 jam
- [ ] Anti-replay foto (§ 2.7c) dengan pengecualian idempotency key
- [ ] Rate limit percobaan gagal (B12) + header `Retry-After`
- [ ] Hapus objek foto saat insert gagal (E36)

### 8.4 Telemetri & audit
- [ ] `attempts.go` — satu baris per percobaan, **tanpa foto**
- [ ] Semua 10 nilai `outcome` benar-benar dipakai di jalur yang sesuai
- [ ] Action audit baru di `internal/audit/actions.go`
- [ ] `GET /attendances/attempts` (#65)

### 8.5 Riwayat & admin
- [ ] `GET /attendances/me` (#56), `/me/today` (#57) — **DTO employee** tanpa similarity
- [ ] `GET /attendances/context` (#55)
- [ ] `GET /attendances/{id}` (#58) + pola ownership → 404
- [ ] `GET /attendances/{id}/photo` (#59) + `410` untuk yang sudah di-purge
- [ ] `GET /attendances` (#60) — filter lengkap + `sort` whitelist, **DTO admin**
- [ ] `GET /attendances/pending` (#61) — tertua dulu + `waiting_hours`
- [ ] `POST /{id}/approve` (#62), `/reject` (#63) — `FOR UPDATE`, B10, B11
- [ ] `POST /attendances/reviews` (#64) — per item, maks 50, tetap menegakkan B10/B11

### 8.6 Lokasi kantor
- [ ] CRUD `/office-locations` (#66–#70)
- [ ] Validasi `radius_meter` ≤ `attendance.max_distance_meter`
- [ ] Penjagaan "lokasi aktif terakhir" dalam transaksi + `FOR UPDATE`
- [ ] Audit perubahan koordinat/radius dengan nilai lama & baru

### 8.7 Retensi
- [ ] `retention.go` — job harian + advisory lock
- [ ] Purge foto (kecuali `pending_review`), isi `photo_purged_at`, kosongkan `photo_key`
- [ ] Purge `attendance_attempts`
- [ ] Bucket `faceclock-attendance` terpisah + lifecycle rule MinIO sebagai lapis kedua

### 8.8 Integrasi & lintas-fase
- [ ] Daftarkan 18 route di `routes.go`; `TestAllRoutesHaveGuards` tetap lulus
- [ ] Perluas `rbac_matrix_test.go`: 18 endpoint × 4 principal
- [ ] Daftarkan 14 error code baru (§ 2.10) di `internal/httpx/errors.go`
- [ ] Validator per-key untuk 15 setting baru
- [ ] `docs/api/fase4-attendance.md`, `docs/attendance/business-rules.md`,
      `docs/attendance/geofence-policy.md`, `docs/attendance/attendance-retention-policy.md`
- [ ] Perbarui `docs/adr/0003-api-conventions.md` dengan katalog error terbaru
- [ ] `DONE-Fase-4.md` sesuai Protokol Handoff master plan § 10.4

---

## 9. Dependencies

**Prasyarat:**

| Dari | Yang dibutuhkan |
|---|---|
| Fase 0 | Envelope + katalog error; middleware chain; `storage.Store`; konvensi skema (`timestamptz` + `now()` sisi DB); aturan logging; graceful shutdown |
| Fase 1 | `employees` (+ `employment_status`); `Principal{employee_id}`; `RequirePermission`; pola ownership → 404; `app_settings` + validator; `audit_logs`; permission `attendance.*` & `location.*` **sudah di-seed**; pelajaran "gunakan index, bukan SELECT-lalu-INSERT" |
| Fase 2 | Klien Go `inference.Embed` + retry + circuit breaker; `FakeClient`; kosakata `hints` + `hints.go`; **aturan "inference gagal ≠ approved"**; query pgvector § 2.4; `face.similarity_threshold` & `face.model_version` hasil kalibrasi |
| Fase 3 | `face_references` (embedding, `is_active`, `model_version`) + index `face_references_lookup_idx`; `employees.attendance_mode`; guard `RequireConsent`; driver storage S3/MinIO |

**Utang lintas-fase yang jatuh tempo di sini**
([Fase 2 § 13.3](03-Fase2.md#133-risiko-lintas-fase-yang-belum-terselesaikan)):

| Risiko | Status di Fase 4 |
|---|---|
| **R2** — Dataset B untuk kalibrasi threshold | ⛔ **Blocker DoD** (§ 10 poin 1). Fase 2 menetapkan: bila threshold hanya dari LFW, ia provisional dan pengumpulan Dataset B menjadi blocker Fase 4. Menjalankan Fase 4 di produksi dengan threshold yang belum divalidasi pada populasi nyata berarti FAR yang tidak diketahui |
| **R7** — volume penyimpanan foto absensi | ✅ Diselesaikan: bucket terpisah + `attendance.photo_retention_days` + job (§ 2.9, § 5.4) |
| **R3** — spoofing foto-dari-layar | ⛔ Fase 6/7. Fase 4 menyimpan `capture_source` dan telemetri percobaan (E28) |
| **R1** — lisensi model InsightFace | ⛔ Masih terbuka; harus dijawab sebelum produksi |
| R4, R5 | ✅ Sudah diselesaikan di Fase 3 |
| **R6** — monorepo vs multi-repo | ✅ **Ditutup** — dikunci monorepo 2026-09-04 ([Fase 0 § 2.1](01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)) |

**Yang bergantung pada fase ini:**

| Fase | Mengambil apa |
|---|---|
| Fase 5 | Antrian `pending_review` (#61) + approve/reject/bulk; daftar & filter absensi (#60) sebagai dasar rekap dan export; CRUD lokasi + map picker; panel setting threshold & geofence; dashboard dari `attendance_attempts` |
| Fase 6 | `GET /attendances/context` untuk menyiapkan halaman; alur `allow_fallback` dua langkah; peta `hints` → coach text; riwayat pribadi |
| Fase 7 | Endpoint yang sama persis; `capture_source='mobile_camera'`; `Idempotency-Key` wajib karena jaringan seluler; liveness on-device sebagai lapis tambahan **sebelum** foto dikirim |

---

## 10. Definition of Done

1. ⛔ **`app_settings.face.similarity_threshold` berasal dari kalibrasi Dataset B**
   (populasi karyawan nyata), bukan angka provisional dari LFW, dan
   `docs/face/fase2-calibration-report.md` sudah tidak bertanda *provisional*.
2. Empat migration (`000018`–`000021`) jalan bersih dari state Fase 3; `down`
   mengembalikannya tanpa sisa.
3. **Uji constraint langsung di SQL:** `INSERT`/`UPDATE` yang menghasilkan
   `method='fallback' AND status='approved' AND reviewed_by IS NULL` **ditolak
   database**. Ini bukti bahwa aturan master plan § 9 tidak bergantung pada
   disiplin kode.
4. Check-in dengan wajah cocok → `201`, `status='approved'`, `method='face'`, dan
   `matched_similarity`, `threshold_used`, `model_version` semuanya terisi.
5. `server_timestamp` selalu berasal dari `now()` database — dibuktikan dengan
   mengirim `client_reported_at` yang berselisih 3 jam dan memverifikasi record
   tetap memakai waktu server, dengan `clock_skew_seconds ≈ -10800`.
6. Check-in kedua di hari yang sama → `409 ALREADY_CHECKED_IN`, termasuk saat dua
   request dikirim **bersamaan** (tepat satu `201`, tepat satu record di database).
7. Check-out tanpa check-in → `409 CHECKOUT_WITHOUT_CHECKIN`; check-out terlalu
   cepat → `409 CHECKOUT_TOO_SOON`.
8. Wajah tidak cocok + `allow_fallback=false` → `422 FACE_NOT_MATCHED`, **nol
   record**, **nol objek foto di storage**, satu baris `attendance_attempts`.
9. Wajah tidak cocok + `allow_fallback=true` → `201` `pending_review`,
   `method='fallback'`, `fallback_reason='below_threshold'`, dan
   `matched_similarity` tetap tersimpan.
10. **Inference dimatikan** → check-in dengan `allow_fallback=false` menghasilkan
    `504`/`502` dan **tidak ada** record; dengan `allow_fallback=true` menghasilkan
    `pending_review`. **Tidak ada satu pun jalur** yang menghasilkan `approved`.
    Diuji sebagai test eksplisit bernama, bukan sebagai efek samping.
11. Di luar radius → `422 OUTSIDE_GEOFENCE` dengan kebijakan default; berubah
    menjadi `pending_review` setelah setting diubah, tanpa restart.
12. Karyawan mode `manual` bisa absen, selalu `pending_review`, dan inference
    **tidak pernah dipanggil** (diverifikasi lewat `FakeClient.EmbedCallCount()`).
13. Karyawan tanpa referensi aktif dengan `model_version` aktif →
    `422 FACE_NOT_ENROLLED`.
14. Admin **tidak bisa** menyetujui absensinya sendiri → `403 SELF_REVIEW_DENIED`.
15. Dua admin me-review record yang sama bersamaan → tepat satu berhasil.
16. Record `approved`/`rejected` tidak bisa di-review ulang → `409`.
17. Approve **tidak mengubah** `server_timestamp` — diverifikasi di test.
18. `GET /attendances/me` **tidak pernah** memuat `matched_similarity`,
    `threshold_used`, `model_version`, `quality_score`, maupun `distance_meter` —
    diuji dengan memindai body response.
19. Karyawan mengakses absensi orang lain → `404`.
20. Anti-replay: mengirim ulang foto yang sama → `409 DUPLICATE_PHOTO`; mengirim
    ulang dengan `Idempotency-Key` yang sama **setelah sukses** → response identik,
    satu record saja.
20a. **(K-03)** `Idempotency-Key` yang sama dikirim ulang **setelah percobaan
    gagal** (mis. `422 FACE_NOT_MATCHED`) → inference **dipanggil lagi**, bukan
    dikembalikan dari cache; diverifikasi lewat jumlah panggilan `FakeClient`
    bertambah, dan `attendance_attempts` mencatat baris baru.
21. Rate limit: percobaan gagal ke-11 dalam satu jam → `429` dengan `Retry-After`.
22. Menonaktifkan lokasi aktif terakhir sementara geofence aktif → `409`.
23. Job retensi menghapus foto record lama, **tidak** menyentuh yang masih
    `pending_review`, dan record absensinya tetap ada dengan `photo_purged_at` terisi.
24. `TestAllRoutesHaveGuards` lulus dengan 18 route baru; matriks RBAC diperluas
    dan lulus (`employee` tidak bisa `GET /attendances`, `POST /{id}/approve`,
    maupun `GET /office-locations`).
25. Semua integration test berjalan dengan `FakeClient` — **tidak ada** yang
    membutuhkan container ONNX; test end-to-end sungguhan bertanda
    `//go:build inference_e2e` dan dijalankan terpisah.
26. Log tidak memuat byte foto, embedding, maupun `note` — test Fase 0 § 11.6
    diperluas dan tetap lulus.
27. `make lint` & `make test` lulus; CI hijau.
28. Empat dokumen di § 7 lengkap.
29. `DONE-Fase-4.md` ada, memuat keputusan D17–D19 final beserta alasannya.

---

## 11. Cara Test / Verifikasi

### 11.1 Persiapan

```bash
make reset && make up
docker compose -f deploy/docker-compose.yml exec faceclock-api /app/seed
API=http://localhost:8080/api/v1
# Karyawan uji sudah ter-enroll di Fase 3
AT=$(curl -s -X POST $API/auth/login -H 'Content-Type: application/json' \
     -d '{"email":"budi@faceclock.local","password":"PasswordAwal123"}' | jq -r .data.access_token)
SA=$(curl -s -X POST $API/auth/login -H 'Content-Type: application/json' \
     -d '{"email":"admin@faceclock.local","password":"..."}' | jq -r .data.access_token)

# Lokasi kantor
curl -s -X POST $API/office-locations -H "Authorization: Bearer $SA" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Kantor Pusat","lat":-6.2001,"lng":106.8166,"radius_meter":100}' | jq .data.id
```

### 11.2 Alur check-in

```bash
# Konteks sebelum absen
curl -s $API/attendances/context -H "Authorization: Bearer $AT" \
| jq '{mode:.data.attendance_mode, enrolled:.data.is_enrolled, next:.data.next_action}'

# Check-in berhasil
curl -s -X POST $API/attendances/check-in -H "Authorization: Bearer $AT" \
  -H "Idempotency-Key: 01JD8Z0X9K7QAAAA" \
  -F "image=@fixtures/faces/budi_live.jpg" \
  -F "lat=-6.2002" -F "lng=106.8167" -F "gps_accuracy_meter=12" \
  -F "capture_source=web_camera" \
| jq '{status:.data.status, method:.data.method, geo:.data.geofence_status, ts:.data.server_timestamp}'

# BUKTI similarity TIDAK bocor ke karyawan
curl -s $API/attendances/me -H "Authorization: Bearer $AT" \
| jq '.data[0] | has("matched_similarity")'        # false

# Admin melihat angkanya
curl -s "$API/attendances?employee_id=$EMP" -H "Authorization: Bearer $SA" \
| jq '.data[0] | {sim:.matched_similarity, th:.threshold_used, mv:.model_version}'

# Check-in kedua → 409
curl -s -X POST $API/attendances/check-in -H "Authorization: Bearer $AT" \
  -F "image=@fixtures/faces/budi_live2.jpg" -F "lat=-6.2002" -F "lng=106.8167" \
  -F "capture_source=web_camera" | jq .error.code      # ALREADY_CHECKED_IN

# Idempotensi: kunci sama → record yang sama, bukan 409
curl -s -X POST $API/attendances/check-in -H "Authorization: Bearer $AT" \
  -H "Idempotency-Key: 01JD8Z0X9K7QAAAA" \
  -F "image=@fixtures/faces/budi_live.jpg" -F "lat=-6.2002" -F "lng=106.8167" \
  -F "capture_source=web_camera" | jq .data.id         # id yang sama
```

### 11.3 Alur fallback (dua langkah, D17)

```bash
# Wajah orang lain → 422, TIDAK ada record
curl -s -X POST $API/attendances/check-in -H "Authorization: Bearer $AT2" \
  -F "image=@fixtures/faces/orang_lain.jpg" -F "lat=-6.2002" -F "lng=106.8167" \
  -F "capture_source=web_camera" \
| jq '{code:.error.code, can_fallback:.error.can_fallback}'   # FACE_NOT_MATCHED, true

# Kirim untuk ditinjau
curl -s -X POST $API/attendances/check-in -H "Authorization: Bearer $AT2" \
  -F "image=@fixtures/faces/orang_lain.jpg" -F "lat=-6.2002" -F "lng=106.8167" \
  -F "capture_source=web_camera" -F "allow_fallback=true" \
| jq '{status:.data.status, method:.data.method, reason:.data.fallback_reason}'
# pending_review / fallback / below_threshold

# Admin approve
AID=$(curl -s "$API/attendances/pending" -H "Authorization: Bearer $SA" | jq -r .data[0].id)
curl -s -X POST $API/attendances/$AID/approve -H "Authorization: Bearer $SA" \
  -H 'Content-Type: application/json' -d '{"review_note":"Verifikasi manual OK"}' \
| jq '{status:.data.status, by:.data.reviewed_by.email}'
```

### 11.4 Uji "inference mati tidak pernah approved"

Ini uji terpenting di seluruh fase, karena ia menguji aturan yang paling menggoda
untuk dilanggar demi kenyamanan.

```bash
docker compose -f deploy/docker-compose.yml stop faceclock-inference

# allow_fallback=false → 502/504, NOL record
curl -s -o /dev/null -w "%{http_code}\n" -X POST $API/attendances/check-in \
  -H "Authorization: Bearer $AT3" -F "image=@fixtures/faces/x.jpg" \
  -F "lat=-6.2002" -F "lng=106.8167" -F "capture_source=web_camera"    # 502 atau 504

# allow_fallback=true → pending_review, TIDAK PERNAH approved
curl -s -X POST $API/attendances/check-in -H "Authorization: Bearer $AT3" \
  -F "image=@fixtures/faces/x.jpg" -F "lat=-6.2002" -F "lng=106.8167" \
  -F "capture_source=web_camera" -F "allow_fallback=true" \
| jq '{status:.data.status, reason:.data.fallback_reason}'
# pending_review / inference_unavailable

docker compose -f deploy/docker-compose.yml start faceclock-inference
```

```sql
-- Invariant yang harus SELALU nol baris:
SELECT id FROM attendances
WHERE method = 'fallback' AND status = 'approved' AND reviewed_by IS NULL;
```

### 11.5 Uji constraint database langsung

Membuktikan aturan hidup di database, bukan hanya di kode:

```sql
-- Harus GAGAL: fallback approved tanpa penyetuju
INSERT INTO attendances (employee_id, type, work_date, method, status, fallback_reason,
                         photo_sha256, photo_bytes, photo_mime, photo_key,
                         geofence_status, capture_source, attendance_mode)
VALUES ('<emp>', 'check_in', current_date, 'fallback', 'approved', 'below_threshold',
        repeat('a',64), 1000, 'image/jpeg', 'k', 'inside', 'web_camera', 'face');
-- ERROR: attendances_fallback_never_approved_chk

-- Harus GAGAL: method='face' tanpa bukti angka
INSERT INTO attendances (..., method, status, matched_similarity, threshold_used, model_version, ...)
VALUES (..., 'face', 'approved', NULL, NULL, NULL, ...);
-- ERROR: attendances_face_evidence_chk

-- Harus GAGAL: dua check-in di hari yang sama
-- ERROR: duplicate key value violates unique constraint "attendances_one_checkin_per_day"
```

### 11.6 Uji geofence (unit, tanpa database)

```go
func TestGeofenceEvaluate(t *testing.T) {
    locs := []geo.Location{
        {ID: "pusat",  Lat: -6.2001, Lng: 106.8166, Radius: 50},
        {ID: "gudang", Lat: -6.2010, Lng: 106.8180, Radius: 300},
    }
    cases := []struct {
        name string; lat, lng, acc float64; want string; wantLoc string
    }{
        {"tepat di kantor",        -6.2001, 106.8166, 10,  "inside",      "pusat"},
        {"di dalam gudang saja",   -6.2008, 106.8176, 10,  "inside",      "gudang"},  // E22
        {"jauh dari keduanya",     -6.3000, 106.9000, 10,  "outside",     "gudang"},
        {"akurasi buruk",          -6.2001, 106.8166, 800, "unavailable", ""},
        {"khatulistiwa",            0.0,      0.0,     10,  "outside",     "pusat"},
    }
    ...
}

func TestHaversineKnownDistances(t *testing.T) {
    // Jakarta–Bandung ≈ 118 km; toleransi 0,5%
    // Jarak nol = 0
    // Meridian 180°: (0,179.999) ke (0,-179.999) ≈ 222 m, bukan setengah bumi
}
```

### 11.7 Uji dengan `FakeClient` (tanpa ONNX)

```go
func TestInferenceFailureNeverApproves(t *testing.T) {
    for _, mode := range []string{"timeout", "500", "breaker_open"} {
        t.Run(mode, func(t *testing.T) {
            fake := inference.NewFake(); fake.FailWith(mode)
            // allow_fallback=false → 502/504, nol record
            // allow_fallback=true  → pending_review
            // TIDAK ADA kombinasi yang menghasilkan approved
        })
    }
}

func TestManualModeSkipsInference(t *testing.T) {
    fake := inference.NewFake()
    // employee.attendance_mode = 'manual'
    // assert: 201 pending_review DAN fake.EmbedCallCount() == 0
}

func TestSimilarityExactlyAtThreshold(t *testing.T) {
    // threshold 0.42, similarity 0.42 → approved (E15)
}

func TestGeofenceRejectedBeforeInference(t *testing.T) {
    // koordinat jauh → 422 OUTSIDE_GEOFENCE DAN fake.EmbedCallCount() == 0
    // DAN nol objek di bucket attendance
}
```

### 11.8 Uji konkurensi

| Skenario | Harapan |
|---|---|
| 10 goroutine check-in bersamaan untuk satu karyawan | Tepat 1 × `201`, 9 × `409`; tepat 1 baris di `attendances`; nol objek foto yatim |
| Dua admin approve record yang sama | Tepat 1 × `200`, 1 × `409` |
| Check-in dan check-out bersamaan | Check-out kalah dengan `409 CHECKOUT_TOO_SOON` atau `CHECKOUT_WITHOUT_CHECKIN` — tidak pernah keduanya sukses dengan urutan waktu terbalik |
| Dua admin menghapus dua lokasi aktif terakhir | Tepat 1 berhasil; selalu tersisa ≥ 1 lokasi aktif |
| Reject + check-in ulang bersamaan | Index unik tidak bentrok; tepat 1 record hidup |

### 11.9 Verifikasi database

```sql
-- Invariant 1: fallback tidak pernah approved tanpa penyetuju
SELECT count(*) FROM attendances
WHERE method='fallback' AND status='approved' AND reviewed_by IS NULL;      -- 0

-- Invariant 2: setiap approved lewat jalur face membawa buktinya
SELECT count(*) FROM attendances
WHERE method='face' AND (matched_similarity IS NULL OR threshold_used IS NULL);  -- 0

-- Invariant 3: satu check-in hidup per hari per karyawan
SELECT employee_id, work_date, count(*) FROM attendances
WHERE type='check_in' AND status <> 'rejected'
GROUP BY 1,2 HAVING count(*) > 1;                                          -- nol baris

-- Invariant 4: check-out tidak pernah mendahului check-in
SELECT o.id FROM attendances o
JOIN attendances i ON i.employee_id=o.employee_id AND i.work_date=o.work_date
                  AND i.type='check_in' AND i.status <> 'rejected'
WHERE o.type='check_out' AND o.status <> 'rejected'
  AND o.server_timestamp < i.server_timestamp;                             -- nol baris

-- Invariant 5: semua similarity dihitung terhadap referensi model yang sama
SELECT a.id FROM attendances a
JOIN face_references f ON f.id = a.matched_reference_id
WHERE a.model_version IS DISTINCT FROM f.model_version;                    -- nol baris

-- Sebaran hasil (bahan tuning threshold di Fase 5)
SELECT status, method, fallback_reason, count(*),
       round(avg(matched_similarity)::numeric, 4) AS avg_sim
FROM attendances GROUP BY 1,2,3 ORDER BY 4 DESC;

-- Telemetri percobaan gagal per karyawan
SELECT e.employee_number, a.outcome, count(*)
FROM attendance_attempts a JOIN employees e ON e.id = a.employee_id
WHERE a.outcome <> 'matched' AND a.server_timestamp > now() - interval '7 days'
GROUP BY 1,2 ORDER BY 3 DESC LIMIT 20;
```

Invariant 3 dan 4 layak dijalankan berkala di production, bukan hanya di test —
keduanya murah dan keduanya mendeteksi kelas bug yang tidak akan pernah dilaporkan
pengguna.

### 11.10 Uji kebocoran data

```go
func TestEmployeeViewNeverLeaksScores(t *testing.T) {
    forbidden := []string{"matched_similarity", "threshold_used", "model_version",
                          "quality_score", "distance_meter", "gps_accuracy_meter"}
    for _, path := range []string{"/attendances/me", "/attendances/me/today",
                                   "/attendances/{id}"} {
        body := callAs(t, employeeToken, path)
        for _, f := range forbidden {
            assert.NotContains(t, body, f, "%s membocorkan %s", path, f)
        }
    }
}
```

Ditambah pemeriksaan log untuk `note`, byte foto, dan array embedding.

---

## 12. Referensi Silang ke "Isu Lintas-Fase" (Master Plan § 9)

| Isu lintas-fase | Bagaimana Fase 4 memenuhinya |
|---|---|
| **Timestamp server-side** | `server_timestamp DEFAULT now()` diisi di dalam `INSERT`; `work_date` dihitung dari `now()` yang sama; `client_reported_at` disimpan terpisah sebagai telemetri dan tidak pernah dipakai; diuji dengan client yang jamnya berselisih 3 jam (§ 10 poin 5) |
| **Geofence server-side** | Seluruh perhitungan di `internal/geo` (fungsi murni); koordinat client adalah masukan; kebijakan `outside`/`unavailable` default `reject`; dievaluasi sebelum foto disimpan (§ 2.5, § 5.1 langkah 9) |
| **Keamanan fallback** | Ditegakkan tiga lapis: alur (§ 2.3 — jalur wajah selalu dicoba lebih dulu), kode (§ 5.1), dan **CHECK constraint database** yang menolak fallback+approved tanpa `reviewed_by` (§ 3.2). Diuji sebagai DoD terpisah (§ 10 poin 3 & 10) |
| **Threshold configurable** | Dibaca dari `app_settings.face.similarity_threshold` setiap request; `threshold_used` disnapshot di setiap record supaya perubahan threshold tidak menghapus jejak keputusan lama |
| **RBAC konsisten di level API** | 18 route terdaftar di `routes.go` + `TestAllRoutesHaveGuards`; matriks RBAC diperluas; pola ownership → 404; **DTO terpisah** untuk employee dan admin sehingga field sensitif tidak bisa bocor karena lupa dikosongkan; B11 menutup konflik kepentingan admin-yang-juga-karyawan |
| **Data biometrik = data sensitif (UU PDP)** | Foto hanya disimpan bila record dibuat; percobaan gagal tidak menyimpan foto; retensi otomatis + bucket terpisah; foto hanya bisa diakses lewat API dengan permission; embedding tidak pernah disimpan di `attendances` (hanya `matched_reference_id`); `note` dilarang masuk log |
| **Anti-spoofing / liveness** | Tidak diselesaikan di sini dan dikatakan terus terang (E21, E28). Yang disediakan: anti-replay foto, telemetri percobaan, rate limit, dan `capture_source` sebagai bahan audit. Penegakan sesungguhnya di Fase 6 (paksa kamera) dan Fase 7 (liveness on-device) |
