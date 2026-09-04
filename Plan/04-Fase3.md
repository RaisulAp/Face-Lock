# Fase 3 — Enrollment Wajah

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 3**. Ukuran: 🔴 **Besar**.
> Mengikuti Protokol Handoff § 10.2: dokumen ini di-review sebelum eksekusi.
>
> **Depends on:** [Fase 1](02-Fase1.md) (identitas, RBAC, `app_settings`, audit) dan
> [Fase 2](03-Fase2.md) (kontrak `/v1/embed`, kontrak data `vector(512)`, `FakeClient`).
> **Blocker untuk:** [Fase 4](05-Fase4.md) — tanpa referensi wajah, tidak ada yang bisa dibandingkan.
>
> **Status:** DRAFT — 4 keputusan butuh konfirmasi user (§ 2.0).

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Fase 4 membandingkan wajah yang datang dengan wajah yang sudah dikenal. Fase 3
adalah satu-satunya tempat "wajah yang sudah dikenal" itu lahir. Kualitas seluruh
sistem absensi ditentukan di sini: referensi yang buruk menghasilkan penolakan
palsu setiap hari, dan referensi yang salah orang menghasilkan kecurangan yang
tidak akan pernah terdeteksi — karena sistem akan mencocokkannya dengan benar.

Fase 3 juga tempat kepatuhan UU PDP menjadi kode, bukan niat: **tidak ada satu pun
embedding boleh lahir tanpa consent yang tercatat.**

### Hasil akhir yang diharapkan
- Karyawan bisa membaca dokumen consent biometrik dan menyetujuinya; persetujuan
  itu tersimpan dengan versi dokumen dan hash isinya.
- Karyawan (atau admin atas namanya) bisa mendaftarkan minimal 3 foto wajah, dengan
  umpan balik kualitas **per foto** sebelum apa pun disimpan permanen.
- Tersimpan **3 embedding terpisah** — tidak pernah di-average — beserta foto
  aslinya, `quality_score`, dan `model_version` yang memproduksinya.
- Referensi bisa dikelola: tambah, nonaktifkan, ganti (re-enroll), dan dihapus
  permanen lewat prosedur penghapusan data PDP.
- Ada job **reindex** yang meregenerasi seluruh embedding saat model berganti
  (utang dari [Fase 2 § 13.3 R5](03-Fase2.md#133-risiko-lintas-fase-yang-belum-terselesaikan)).
- Ada jalur yang sah bagi karyawan yang **menolak** consent (utang dari
  [Fase 2 § 13.3 R4](03-Fase2.md#133-risiko-lintas-fase-yang-belum-terselesaikan)).

### Yang TIDAK dikerjakan di fase ini
- Check-in / check-out (→ Fase 4).
- UI apa pun (→ Fase 5 untuk admin, Fase 6 untuk karyawan).
- Liveness. Foto enrollment memang berasal dari kamera, tapi Fase 3 tidak bisa
  membuktikannya — pembuktiannya ada di client (Fase 6/7). Keterbatasan ini
  ditulis eksplisit, bukan diasumsikan tertangani (§ 6 E23).

---

## 2. Scope Detail

### 2.0 Keputusan yang butuh konfirmasi

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D13 | Di mana file foto referensi disimpan | **MinIO (S3-compatible) di compose; `LocalStore` untuk dev.** Driver S3 yang sama bisa diarahkan ke cloud tanpa ubah kode | ⚠️ **BUTUH KONFIRMASI** |
| D14 | Cara client mengakses foto | **Streaming lewat API** (`GET .../photo`) dengan cek permission per-request — **bukan** signed URL | ⚠️ **BUTUH KONFIRMASI** |
| D15 | Bentuk alur enrollment | **Sesi bertahap** (buat sesi → unggah foto satu per satu → commit) | ⚠️ **BUTUH KONFIRMASI** |
| D16 | Cek wajah kembar/duplikat antar-karyawan saat enroll | **Aktif secara default** (§ 2.6) | ⚠️ **BUTUH KONFIRMASI** |

---

### 2.1 D13 — Penyimpanan foto referensi ⚠️ BUTUH KONFIRMASI

[Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go) sudah menyediakan
abstraksinya:

```go
type Store interface {
    Put(ctx context.Context, key string, r io.Reader, contentType string) (string, error)
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}
```

Jadi pertanyaannya bukan "bagaimana kodenya", tapi **driver mana yang dipakai di
production**. Empat opsi:

| Opsi | Kelebihan | Kekurangan |
|---|---|---|
| **A. Disk lokal / volume Docker** (`LocalStore`, sudah ada) | Nol dependensi baru; paling cepat jalan | Tidak ada replikasi; backup harus diurus terpisah dari backup DB; tidak ada lifecycle rule untuk retensi otomatis; sulit di-scale ke >1 replica API |
| **B. MinIO self-hosted** (S3 API, satu container) | API S3 → kode yang sama bisa dipakai ke cloud nanti; versioning, lifecycle rule (retensi otomatis), server-side encryption; bucket privat by default; data tetap di infrastruktur sendiri | Satu container lagi untuk dioperasikan & di-backup |
| **C. Cloud object storage** (S3/GCS/R2) | Paling andal, nol operasi | Data biometrik keluar dari infrastruktur organisasi — butuh kajian PDP & kontrak pemrosesan data; biaya; butuh internet |
| **D. `bytea` di Postgres** | Satu backup untuk semua; transaksi atomik dengan metadata | Ukuran database membengkak (3 foto × N karyawan × ~300 KB); setiap dump/restore ikut berat; `SELECT *` yang ceroboh menarik megabyte; bukan tempat blob |

**Rekomendasi: B (MinIO), dengan `LocalStore` tetap dipakai di dev.**

Alasan utamanya bukan performa, melainkan **retensi dan enkripsi**. Data biometrik
menuntut kebijakan retensi yang bisa dibuktikan berjalan. Di opsi A, "hapus foto
karyawan yang sudah resign lebih dari 1 tahun" adalah cron buatan sendiri yang bisa
diam-diam berhenti jalan. Di MinIO, itu adalah *lifecycle rule* pada bucket plus
job aplikasi — dua lapis, dan yang satu tidak bergantung pada kode kita.

Alasan kedua: opsi B dan C berbagi driver yang sama. Memilih B sekarang tidak
menutup pintu ke C nanti — hanya mengubah endpoint dan kredensial.

Opsi D ditolak karena embedding **memang** di Postgres (itu perlu, untuk pgvector),
tapi foto tidak perlu berada di sana dan hanya memperberat setiap operasi database.

**Konfigurasi yang menyertai rekomendasi ini:**

```
STORAGE_DRIVER=s3                       # local | s3
STORAGE_S3_ENDPOINT=http://minio:9000
STORAGE_S3_REGION=us-east-1
STORAGE_S3_BUCKET_FACE=faceclock-face
STORAGE_S3_BUCKET_ATTENDANCE=faceclock-attendance   # dipakai Fase 4
STORAGE_S3_ACCESS_KEY=
STORAGE_S3_SECRET_KEY=
STORAGE_S3_FORCE_PATH_STYLE=true        # wajib untuk MinIO
STORAGE_S3_SSE=AES256                   # server-side encryption
```

Dua bucket terpisah (referensi vs absensi) disengaja: masa retensinya berbeda,
dan kebijakan aksesnya berbeda. Bucket **tidak pernah** publik.

**Skema key** — deterministik, tidak bocorkan identitas di nama file:

```
face/{employee_id}/{face_reference_id}.jpg
face-staging/{session_id}/{photo_id}.jpg      # dihapus setelah commit / kedaluwarsa
```

`employee_id` adalah UUID, bukan nomor karyawan — orang yang melihat daftar objek
di bucket tidak langsung tahu itu siapa.

---

### 2.2 D14 — Akses foto: streaming lewat API, bukan signed URL ⚠️ BUTUH KONFIRMASI

Signed URL terlihat menarik (API tidak perlu mengalirkan byte), tapi untuk foto
biometrik ia punya satu sifat yang menggagalkan RBAC: **begitu URL diterbitkan,
siapa pun yang memegangnya bisa membukanya** sampai kedaluwarsa. URL itu akan
masuk riwayat browser, log proxy, dan screenshot WhatsApp.

Rekomendasi: **stream lewat API**.

```
GET /api/v1/face/references/{id}/photo
```

- Permission diperiksa **di setiap request**, memakai pola ownership
  [Fase 1 § 5.4](02-Fase1.md#54-pola-akses-self) — akses ke foto milik orang lain
  dibalas **404**, bukan 403.
- Response header: `Cache-Control: private, max-age=60`, `X-Content-Type-Options: nosniff`,
  `Content-Disposition: inline`.
- Setiap pembacaan foto orang lain (`face.read_any`) menulis `audit_logs`
  `face.reference.photo_viewed`. Membaca foto sendiri tidak dicatat — kalau dicatat,
  tabel audit akan penuh oleh hal yang tidak menarik dan yang menarik jadi
  tenggelam.

`Store.SignedURL()` tetap ada di interface (Fase 0), tapi **tidak dipakai** di
Fase 3/4. Ia disimpan untuk kemungkinan CDN di masa depan.

---

### 2.3 D15 — Alur enrollment bertahap ⚠️ BUTUH KONFIRMASI

Dua bentuk yang mungkin:

**Bentuk 1 — satu request, tiga foto sekaligus.** Sederhana di server, buruk di
lapangan: kalau foto ke-3 terlalu gelap, seluruh request gagal dan karyawan
mengulang ketiganya. Dengan gate kualitas Fase 2 yang ketat, ini akan sering terjadi.

**Bentuk 2 — sesi bertahap (rekomendasi).**

```
POST   /face/enrollments                    → buat sesi (draft), 30 menit
POST   /face/enrollments/{id}/photos        → 1 foto: embed + validasi + umpan balik SEKARANG
DELETE /face/enrollments/{id}/photos/{pid}  → buang foto yang jelek, ambil ulang
POST   /face/enrollments/{id}/commit        → tulis ke face_references dalam SATU transaksi
DELETE /face/enrollments/{id}               → batal
```

Keuntungan yang menentukan: karyawan tahu **saat itu juga** kenapa fotonya ditolak
("terlalu gelap"), dan hanya mengulang satu foto. Ini persis yang dibutuhkan UI
Fase 6, dan persis yang dimungkinkan oleh kosakata `hints`
[Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints).

Konsekuensinya: ada **staging** — tabel `face_enrollment_photos` dan prefix
`face-staging/` di object storage, keduanya dibersihkan saat commit atau kedaluwarsa.

**Embedding dihitung saat foto diunggah, bukan saat commit.** Menghitung ulang di
commit berarti memanggil inference dua kali dan — lebih buruk — bisa menghasilkan
vektor berbeda kalau model sempat berganti di tengah sesi. Karena itu sesi
menyimpan `model_version` sebagai snapshot, dan commit menolak (409) bila model
aktif sudah berbeda (§ 6 E12).

---

### 2.4 Consent biometrik — di mana disimpan & bagaimana ditegakkan

Master plan § 2 dan § 9 menuntut consent saat onboarding. Yang membuatnya nyata
bukan formulir, melainkan tiga hal:

**a. Isi yang disetujui harus bisa dibuktikan.** Menyimpan `consent = true` tidak
bernilai apa-apa kalau teks yang disetujui sudah berubah sejak itu. Karena itu ada
dua tabel: `consent_documents` (teks berversi, dengan `content_hash`) dan
`biometric_consents` (siapa menyetujui versi mana, kapan, dan hash isinya saat itu).

**b. Penegakan di satu tempat, bukan di setiap handler.** Sebuah guard
`RequireConsent(employeeIDFrom)` dipasang di semua route yang menghasilkan atau
membaca data biometrik. Ia bekerja seperti `RequirePermission`
([Fase 0 middleware chain](01-Fase0.md#26-scaffolding-faceclock-api-go)) tetapi
berjalan **setelah** otorisasi:

```
Authenticate → RequirePermission("face.enroll_self") → RequireConsent(self) → handler
```

Karena consent adalah properti **karyawan**, bukan properti pemanggil, guard-nya
menerima fungsi yang menunjukkan employee mana yang datanya akan disentuh. Admin
yang meng-enroll atas nama orang lain tetap terhalang bila karyawan itu belum
memberi consent — dan itu memang yang benar.

**c. Pencabutan harus punya akibat.** Mencabut consent bukan sekadar mengubah
kolom: ia **menonaktifkan semua `face_references` milik karyawan itu** dalam
transaksi yang sama (`deactivated_reason = 'consent_withdrawn'`). Efeknya di Fase 4:
karyawan tidak lagi punya referensi aktif, sehingga verifikasi wajah tidak mungkin —
persis konsekuensi yang diharapkan dari mencabut izin.

**d. Jalur bagi yang menolak consent** — utang R4 dari Fase 2. Karyawan yang
menolak **tidak boleh** kehilangan kemampuan mencatat kehadiran; kalau ia kehilangan,
consent-nya bukan sukarela dan secara hukum tidak sah. Jalur yang disediakan:
kolom `employees.attendance_mode` (`face` | `manual`). Karyawan `manual` melewati
seluruh jalur wajah dan absensinya selalu `pending_review` untuk disetujui atasan.
Perubahan mode dicatat di audit log dan hanya bisa dilakukan pemegang
`employee.update`.

> Menempatkan `attendance_mode` di `employees` (bukan menyimpulkannya dari ada/tidaknya
> consent) disengaja: "belum sempat enroll" dan "menolak enroll" adalah dua keadaan
> berbeda yang butuh perlakuan berbeda, dan menyamakannya akan membuat karyawan baru
> otomatis masuk jalur manual.

---

### 2.5 Aturan penyimpanan embedding

Mengikuti kontrak yang sudah dikunci di
[Fase 2 § 3.2](03-Fase2.md#32-kontrak-data-untuk-fase-3-belum-dibuat-di-sini):

| Aturan | Nilai |
|---|---|
| Tipe | `vector(512)` |
| Normalisasi | Sudah ter-L2-normalize dari inference. **Tidak** dinormalisasi ulang saat baca maupun tulis |
| Jumlah minimum | `app_settings.face.min_reference_photos` (default 3) |
| Jumlah maksimum | `app_settings.face.max_reference_photos` (default 5) — batas baru di fase ini |
| Penyimpanan | **Satu baris per foto.** Tidak pernah di-average, tidak pernah digabung |
| Penyertaan | `model_version` wajib; vektor dari model berbeda tidak pernah dibandingkan |
| Index | B-tree `(employee_id, is_active, model_version)`. **Tidak ada** HNSW/IVFFlat |

Kenapa tidak di-average — ini pantas ditulis sekali dengan jelas, karena
"rata-ratakan saja biar hemat" adalah optimasi yang terlihat masuk akal: tiga foto
diambil pada pose, pencahayaan, dan ekspresi berbeda. Vektornya menempati tiga
titik berbeda di ruang embedding, dan rata-ratanya adalah titik yang **tidak
merepresentasikan foto mana pun** — sering kali lebih jauh dari wajah asli
dibanding ketiganya. Master plan § 4 juga menuntut perbandingan mengambil
**similarity tertinggi**, yang hanya mungkin bila ketiganya masih utuh.

---

### 2.6 D16 — Deteksi wajah duplikat antar-karyawan ⚠️ BUTUH KONFIRMASI

Saat karyawan B mendaftarkan wajah, apakah sistem memeriksa bahwa wajah itu belum
terdaftar sebagai karyawan A?

**Rekomendasi: ya, aktif secara default.**

Tanpa pemeriksaan ini, seluruh sistem bisa dilewati satu kali di titik enrollment:
seseorang mendaftarkan wajahnya pada dua akun karyawan, lalu setiap hari absen
untuk keduanya — dan setiap absensi akan **lolos dengan benar**, `approved`, tanpa
jejak mencurigakan apa pun. Ini kegagalan paling mahal yang mungkin terjadi pada
sistem ini, dan pemeriksaannya murah karena enrollment jarang.

Mekanisme:

```sql
SELECT fr.employee_id, max(1 - (fr.embedding <=> $1::vector)) AS similarity
FROM face_references fr
WHERE fr.is_active = true
  AND fr.model_version = $2
  AND fr.employee_id <> $3
GROUP BY fr.employee_id
HAVING max(1 - (fr.embedding <=> $1::vector)) >= $4   -- face.duplicate_threshold
ORDER BY similarity DESC
LIMIT 5;
```

- Dijalankan **saat commit**, bukan saat tiap foto diunggah — sekali per sesi.
- `face.duplicate_threshold` (default: `face.similarity_threshold + 0.05`, tapi
  disimpan sebagai angka tersendiri agar bisa di-tuning).
- Ditemukan kecocokan → `409 FACE_BELONGS_TO_ANOTHER_EMPLOYEE`. Response **tidak
  menyebut nama karyawan lain** (itu kebocoran data ke orang yang mungkin justru
  pelakunya); yang disebut hanya bahwa terjadi konflik dan nomor tiket audit.
  Detail lengkapnya masuk `audit_logs` untuk admin.
- Pemegang `face.enroll_any` bisa memaksa lewat `force_duplicate: true` (untuk
  kasus kembar identik yang nyata), yang **wajib** disertai `reason` dan dicatat
  di audit dengan action `face.enrollment.duplicate_override`.

**Catatan skala.** Query di atas adalah pemindaian 1:N — persis yang dilarang
[Fase 2 § 2.4](03-Fase2.md#24-d11--di-mana-perbandingan-terjadi--butuh-konfirmasi)
di **jalur verifikasi**. Larangan itu tetap berlaku dan tidak dilanggar di sini:
larangan itu tentang jalur check-in yang berjalan ratusan kali sehari dan harus
eksak. Pemindaian ini berjalan sekali per enrollment, boleh memakan ratusan
milidetik, dan tetap eksak (sequential scan, bukan ANN). Pada ~5.000 referensi
ini masih di bawah 100 ms. Di atas ~50.000 referensi, tinjau ulang — dan itu
berarti organisasi dengan >15.000 karyawan, yang jauh di luar asumsi master plan.

---

### 2.7 Job reindex saat model berganti

Utang R5 dari [Fase 2 § 13.3](03-Fase2.md#133-risiko-lintas-fase-yang-belum-terselesaikan).

Ketika `face.model_version` berubah (ganti model, ganti `det_size`, naik versi
`insightface`), semua embedding lama menjadi tidak sebanding. [Fase 2 § 2.2](03-Fase2.md#22-d10--pilihan-model--butuh-konfirmasi)
sudah menetapkan: query perbandingan **memfilter** `model_version`, sehingga vektor
lama tidak ikut dibandingkan. Akibatnya sistem **gagal aman** (orang tidak dikenali)
alih-alih gagal berbahaya — tapi artinya seluruh karyawan tidak bisa absen sampai
di-reindex.

Job reindex:

```
1. Buat job {from_model_version, to_model_version}
2. Snapshot: semua face_references dengan model_version = from  →  face_reindex_items
3. Untuk tiap item, dalam batch 16 (batas /v1/embed-batch):
     ambil foto asli dari storage  →  POST /v1/embed-batch
     usable=true  → INSERT baris BARU face_references {embedding baru, model_version = to,
                    photo_key SAMA, is_active = false}
     usable=false → tandai item 'failed' + hints; foto lama TETAP ada
4. Setelah semua item selesai DAN succeeded_count >= min_reference_photos untuk
   setiap karyawan yang terdampak:
     dalam SATU transaksi per karyawan:
       UPDATE lama SET is_active=false, deactivated_reason='model_reindex'
       UPDATE baru SET is_active=true
5. Karyawan yang referensi barunya < min_reference_photos → TIDAK di-swap;
   ia masuk daftar "perlu enroll ulang" yang ditampilkan admin (Fase 5)
6. app_settings.face.model_version diubah ke `to` HANYA setelah job selesai
```

Dua sifat yang penting:
- **Foto asli wajib disimpan.** Tanpa foto, reindex tidak mungkin dan pergantian
  model berarti seluruh perusahaan enroll ulang. Ini alasan teknis paling kuat
  mengapa D13 (§ 2.1) tidak boleh berujung pada "embedding saja, foto dibuang".
- **Swap terjadi per karyawan, atomik**, dan hanya kalau hasilnya memenuhi minimum.
  Karyawan tidak boleh berada di keadaan "referensi lama sudah mati, yang baru
  belum cukup".

Job berjalan sebagai goroutine worker di `faceclock-api` dengan advisory lock
Postgres (`pg_try_advisory_lock`) supaya dua replica tidak menjalankan job yang
sama. Bisa di-cancel; item yang sudah selesai tidak diulang saat dijalankan lagi.

---

### 2.8 Penghapusan data biometrik (UU PDP)

Berbeda dari `is_active = false`. Penghapusan permanen dipakai saat karyawan
menggunakan haknya untuk meminta penghapusan data, atau saat kebijakan retensi
terpenuhi.

```
DELETE /api/v1/employees/{id}/face-data   (guard: face.delete_any, wajib `reason`)
```

Dalam satu transaksi:
1. Catat `audit_logs` `face.data.erased` dengan `{employee_id, reason, reference_count}` —
   **tanpa** embedding, tanpa key foto.
2. `DELETE FROM face_references WHERE employee_id = $1` (hard delete, bukan soft).
3. Setelah transaksi commit: hapus objek foto dari storage.
4. `biometric_consents` **tidak** dihapus — ia adalah bukti bahwa pemrosesan dulu
   sah, dan menghapusnya justru menghilangkan pertanggungjawaban. Status consent
   diubah menjadi `withdrawn` bila belum.

Urutan langkah 2–3 disengaja: kalau penghapusan storage gagal, database sudah
bersih dan ada job pembersih objek yatim yang menyapunya. Kebalikannya — storage
terhapus tapi DB masih menunjuk ke sana — menghasilkan baris yang fotonya 404 selamanya.

Kebijakan retensi otomatis (karyawan `resigned` > `face.retention_days_after_resign`,
default 365) dijalankan job harian yang memakai prosedur yang sama.

---

### 2.9 Error code tambahan (registrasi resmi ke katalog Fase 0)

Bukan format baru — semuanya memakai envelope `{error: {code, message, details, request_id}}`
dan status HTTP yang sudah ada di
[katalog Fase 0](01-Fase0.md#26-scaffolding-faceclock-api-go).

| HTTP | code | Kapan |
|---|---|---|
| 403 | `CONSENT_REQUIRED` | Karyawan belum memberi consent biometrik |
| 409 | `CONSENT_ALREADY_GRANTED` | Consent aktif sudah ada |
| 409 | `CONSENT_VERSION_OUTDATED` | Menyetujui versi dokumen yang bukan versi aktif |
| 422 | `FACE_NOT_USABLE` | Foto tidak lolos gate kualitas; `details` berisi `hints` |
| 409 | `DUPLICATE_PHOTO` | Foto identik (sha256 sama) sudah ada di sesi/referensi |
| 422 | `ENROLLMENT_INCOMPLETE` | Commit dengan foto < `face.min_reference_photos` |
| 409 | `ENROLLMENT_LIMIT_REACHED` | Melebihi `face.max_reference_photos` |
| 409 | `ENROLLMENT_SESSION_EXPIRED` | Sesi sudah kedaluwarsa / sudah di-commit |
| 409 | `ENROLLMENT_MODEL_CHANGED` | `model_version` berubah di tengah sesi |
| 409 | `FACE_BELONGS_TO_ANOTHER_EMPLOYEE` | Deteksi duplikat (§ 2.6) |
| 409 | `REINDEX_IN_PROGRESS` | Enroll/commit ditolak selagi reindex berjalan |
| 422 | `ATTENDANCE_MODE_MANUAL` | Karyawan berada di jalur manual, tidak bisa enroll |

> `#38 POST /face/enrollments` (§ 4.3) juga memakai `503 FACE_SERVICE_NOT_CONFIGURED`
> untuk kondisi "kalibrasi Fase 2 belum selesai". Kode itu **tidak** didaftarkan di
> sini — registrasi resminya ada di [Fase 4 § 2.10](05-Fase4.md#210-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0),
> dipakai lebih dulu di Fase 3 karena akar masalahnya sama dengan yang dihadapi
> check-in/check-out ([resolusi K-04](09-Revisions-Log.md#k-04--dua-kode-berbeda-untuk-satu-akar-masalah--inkonsistensi-kecil--resolved)).
> **Dampak urutan eksekusi:** kode ini harus sudah ada di `internal/httpx/errors.go`
> saat Fase 3 dieksekusi, meski definisinya "milik" dokumen Fase 4.

`details` untuk `FACE_NOT_USABLE` memakai bentuk yang sudah ada di Fase 0
(array `{field, message}`) ditambah field `hints`:

```json
{
  "error": {
    "code": "FACE_NOT_USABLE",
    "message": "Foto belum memenuhi syarat",
    "details": [{ "field": "image", "message": "Terlalu gelap. Cari tempat yang lebih terang." }],
    "hints": ["too_dark"],
    "request_id": "01JD..."
  }
}
```

Teks Indonesia diambil dari `internal/inference/hints.go`
([Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints)) sehingga web
(Fase 6) dan mobile (Fase 7) memakai kalimat yang sama persis.

---

## 3. Skema Database

Konvensi mengikuti [Fase 0 § 3](01-Fase0.md#3-skema-database). Migration Fase 3:
`000012` – `000017` (melanjutkan `000011` milik Fase 2).

### 3.1 `consent_documents` — migration `000012`

| Kolom | Tipe | Constraint |
|---|---|---|
| `version` | `text` | PK, `CHECK (version ~ '^[0-9]{4}-[0-9]{2}-v[0-9]+$')` — mis. `2026-09-v1` |
| `title` | `text` | `NOT NULL` |
| `body` | `text` | `NOT NULL` — teks lengkap (Markdown) |
| `content_hash` | `text` | `NOT NULL` — `sha256` dari `body` |
| `is_active` | `boolean` | `NOT NULL DEFAULT false` |
| `published_at` | `timestamptz` | `NULL` |
| `created_at` / `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` |

```sql
CREATE UNIQUE INDEX consent_documents_active_uniq ON consent_documents (is_active) WHERE is_active;
```

Index unik pada nilai boolean yang selalu `true` adalah cara menegakkan "hanya
boleh ada satu dokumen aktif" di level database, bukan di level niat baik.

Dokumen tidak pernah diedit setelah dipublikasikan — versi baru berarti baris baru.
Kalau boleh diedit, `content_hash` yang tersimpan di `biometric_consents` menjadi
bohong.

### 3.2 `biometric_consents` — migration `000012`

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `employee_id` | `uuid` | `NOT NULL`, FK → `employees(id)` `ON DELETE RESTRICT` | |
| `document_version` | `text` | `NOT NULL`, FK → `consent_documents(version)` `ON DELETE RESTRICT` | |
| `content_hash` | `text` | `NOT NULL` | disalin saat menyetujui — bukti isi apa yang disetujui |
| `status` | `text` | `NOT NULL`, `CHECK (status IN ('granted','withdrawn'))` | |
| `method` | `text` | `NOT NULL`, `CHECK (method IN ('self_web','self_mobile','admin_recorded'))` | |
| `granted_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |
| `withdrawn_at` | `timestamptz` | `NULL` | |
| `withdrawn_reason` | `text` | `NULL` | |
| `recorded_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` | diisi bila `method='admin_recorded'` |
| `ip` | `inet` | `NULL` | |
| `user_agent` | `text` | `NULL` | |
| `created_at` / `updated_at` | `timestamptz` | | |

```sql
CREATE UNIQUE INDEX biometric_consents_active_uniq
  ON biometric_consents (employee_id) WHERE status = 'granted';
CREATE INDEX biometric_consents_employee_idx ON biometric_consents (employee_id, granted_at DESC);

ALTER TABLE biometric_consents ADD CONSTRAINT biometric_consents_withdraw_chk
  CHECK ((status = 'withdrawn') = (withdrawn_at IS NOT NULL));
```

**Tidak ada soft delete.** Ini catatan kepatuhan; ia bertambah, tidak pernah
berkurang. Mencabut consent membuat baris baru berstatus `withdrawn`, bukan
menghapus yang lama.

### 3.3 `employees.attendance_mode` — migration `000013`

```sql
ALTER TABLE employees
  ADD COLUMN attendance_mode text NOT NULL DEFAULT 'face'
    CHECK (attendance_mode IN ('face','manual'));
CREATE INDEX employees_attendance_mode_idx ON employees (attendance_mode)
  WHERE deleted_at IS NULL;
```

Jalur bagi karyawan yang menolak consent (§ 2.4d) dan bagi kasus khusus (cedera
wajah, alasan keagamaan). Dipakai Fase 4.

### 3.4 `face_references` — migration `000014`

Tabel inti fase ini. Memenuhi kontrak [Fase 2 § 3.2](03-Fase2.md#32-kontrak-data-untuk-fase-3-belum-dibuat-di-sini).

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `employee_id` | `uuid` | `NOT NULL`, FK → `employees(id)` `ON DELETE RESTRICT` | |
| `embedding` | `vector(512)` | `NOT NULL` | sudah L2-normalized; jangan dinormalisasi ulang |
| `model_version` | `text` | `NOT NULL` | mis. `buffalo_l@v1` |
| `quality_score` | `real` | `NOT NULL`, `CHECK (quality_score BETWEEN 0 AND 1)` | |
| `det_score` | `real` | `NULL`, `CHECK (det_score IS NULL OR det_score BETWEEN 0 AND 1)` | |
| `photo_key` | `text` | `NULL` | key di object storage; `NULL` hanya setelah retensi |
| `photo_sha256` | `text` | `NOT NULL`, `CHECK (photo_sha256 ~ '^[0-9a-f]{64}$')` | dedup + integritas |
| `photo_bytes` | `integer` | `NOT NULL`, `CHECK (photo_bytes > 0)` | |
| `photo_mime` | `text` | `NOT NULL`, `CHECK (photo_mime IN ('image/jpeg','image/png','image/webp'))` | |
| `photo_purged_at` | `timestamptz` | `NULL` | diisi saat foto dihapus karena retensi |
| `capture_source` | `text` | `NOT NULL`, `CHECK (capture_source IN ('web_camera','mobile_camera','admin_upload'))` | |
| `position` | `smallint` | `NOT NULL`, `CHECK (position BETWEEN 1 AND 20)` | urutan dalam sesi, untuk tampilan |
| `is_active` | `boolean` | `NOT NULL DEFAULT true` | |
| `enrollment_session_id` | `uuid` | `NULL`, FK → `face_enrollment_sessions(id)` `ON DELETE SET NULL` | |
| `enrolled_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` | |
| `superseded_by` | `uuid` | `NULL`, FK → `face_references(id)` `ON DELETE SET NULL` | diisi saat reindex |
| `deactivated_at` | `timestamptz` | `NULL` | |
| `deactivated_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` | |
| `deactivated_reason` | `text` | `NULL`, `CHECK (deactivated_reason IN ('replaced','re_enroll','admin_removed','consent_withdrawn','model_reindex','quality_review','employee_resigned'))` | |
| `created_at` / `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |

```sql
-- Index utama; dipakai Fase 4 (query verifikasi) dan § 2.6 (deteksi duplikat).
CREATE INDEX face_references_lookup_idx
  ON face_references (employee_id, is_active, model_version);

CREATE INDEX face_references_active_model_idx
  ON face_references (model_version) WHERE is_active;

CREATE INDEX face_references_session_idx
  ON face_references (enrollment_session_id);

-- Foto yang sama persis tidak boleh terdaftar dua kali sebagai referensi aktif.
CREATE UNIQUE INDEX face_references_photo_uniq
  ON face_references (employee_id, photo_sha256) WHERE is_active;

ALTER TABLE face_references ADD CONSTRAINT face_references_deactivation_chk
  CHECK ((is_active = false) = (deactivated_at IS NOT NULL));
ALTER TABLE face_references ADD CONSTRAINT face_references_photo_chk
  CHECK ((photo_key IS NOT NULL) OR (photo_purged_at IS NOT NULL));
```

> **Tidak ada index HNSW/IVFFlat**, sesuai [Fase 2 § 2.4](03-Fase2.md#24-d11--di-mana-perbandingan-terjadi--butuh-konfirmasi):
> jalur verifikasi Fase 4 sudah difilter `employee_id` dan hanya menyentuh 3–5 baris,
> dan index *approximate* bisa melewatkan tetangga terdekat yang sebenarnya — kesalahan
> yang tidak boleh terjadi pada verifikasi identitas.

**Tidak ada `deleted_at`.** Siklus hidupnya `is_active`; penghapusan permanen
adalah hard delete lewat prosedur PDP (§ 2.8). Ini pengecualian sadar dari konvensi
soft-delete Fase 0, dan alasannya: baris yang "dihapus" tapi masih menyimpan
embedding bukan penghapusan data biometrik dalam pengertian mana pun.

### 3.5 `face_enrollment_sessions` — migration `000015`

| Kolom | Tipe | Constraint | Catatan |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `employee_id` | `uuid` | `NOT NULL`, FK → `employees(id)` `ON DELETE CASCADE` | |
| `status` | `text` | `NOT NULL DEFAULT 'draft'`, `CHECK (status IN ('draft','committed','cancelled','expired'))` | |
| `mode` | `text` | `NOT NULL DEFAULT 'replace'`, `CHECK (mode IN ('replace','append'))` | `replace` = re-enroll |
| `required_photos` | `smallint` | `NOT NULL` | snapshot `face.min_reference_photos` |
| `max_photos` | `smallint` | `NOT NULL` | snapshot `face.max_reference_photos` |
| `model_version` | `text` | `NOT NULL` | snapshot; commit menolak bila sudah berbeda |
| `created_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` | |
| `expires_at` | `timestamptz` | `NOT NULL` | `now() + face.enrollment_session_ttl_minutes` |
| `committed_at` | `timestamptz` | `NULL` | |
| `cancelled_at` | `timestamptz` | `NULL` | |
| `created_at` / `updated_at` | `timestamptz` | | |

```sql
-- Satu sesi draft aktif per karyawan; mencegah dua tab membuat sesi bersamaan.
CREATE UNIQUE INDEX face_enrollment_sessions_draft_uniq
  ON face_enrollment_sessions (employee_id) WHERE status = 'draft';
CREATE INDEX face_enrollment_sessions_expiry_idx
  ON face_enrollment_sessions (expires_at) WHERE status = 'draft';
```

### 3.6 `face_enrollment_photos` — migration `000015`

Staging. Isinya berumur pendek dan dibersihkan saat commit/kedaluwarsa.

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` |
| `session_id` | `uuid` | `NOT NULL`, FK → `face_enrollment_sessions(id)` `ON DELETE CASCADE` |
| `position` | `smallint` | `NOT NULL`, `CHECK (position BETWEEN 1 AND 20)` |
| `embedding` | `vector(512)` | `NOT NULL` |
| `quality_score` | `real` | `NOT NULL` |
| `det_score` | `real` | `NULL` |
| `hints` | `jsonb` | `NOT NULL DEFAULT '[]'::jsonb` |
| `photo_key` | `text` | `NOT NULL` — prefix `face-staging/` |
| `photo_sha256` | `text` | `NOT NULL` |
| `photo_bytes` | `integer` | `NOT NULL` |
| `photo_mime` | `text` | `NOT NULL` |
| `capture_source` | `text` | `NOT NULL` |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` |

```sql
CREATE UNIQUE INDEX face_enrollment_photos_pos_uniq ON face_enrollment_photos (session_id, position);
CREATE UNIQUE INDEX face_enrollment_photos_sha_uniq ON face_enrollment_photos (session_id, photo_sha256);
```

Hanya foto `usable = true` yang masuk sini — foto yang gagal gate kualitas
**tidak disimpan sama sekali**, tidak di DB maupun di storage. Menyimpannya berarti
menumpuk data biometrik yang tidak akan pernah dipakai, dan itu bertentangan dengan
prinsip minimalisasi data.

### 3.7 `face_reindex_jobs` & `face_reindex_items` — migration `000016`

**`face_reindex_jobs`**

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` |
| `from_model_version` | `text` | `NOT NULL` |
| `to_model_version` | `text` | `NOT NULL`, `CHECK (to_model_version <> from_model_version)` |
| `status` | `text` | `NOT NULL DEFAULT 'pending'`, `CHECK (status IN ('pending','running','completed','failed','cancelled'))` |
| `total_count` / `processed_count` / `succeeded_count` / `failed_count` | `integer` | `NOT NULL DEFAULT 0` |
| `employees_ready_count` / `employees_incomplete_count` | `integer` | `NOT NULL DEFAULT 0` |
| `error` | `text` | `NULL` |
| `created_by` | `uuid` | `NULL`, FK → `users(id)` `ON DELETE SET NULL` |
| `started_at` / `finished_at` | `timestamptz` | `NULL` |
| `created_at` / `updated_at` | `timestamptz` | |

```sql
CREATE UNIQUE INDEX face_reindex_jobs_running_uniq
  ON face_reindex_jobs ((true)) WHERE status IN ('pending','running');
```

Satu job berjalan pada satu waktu — di level database, bukan di level kode.

**`face_reindex_items`**

| Kolom | Tipe | Constraint |
|---|---|---|
| `job_id` | `uuid` | `NOT NULL`, FK → `face_reindex_jobs(id)` `ON DELETE CASCADE` |
| `face_reference_id` | `uuid` | `NOT NULL`, FK → `face_references(id)` `ON DELETE CASCADE` |
| `status` | `text` | `NOT NULL DEFAULT 'pending'`, `CHECK (status IN ('pending','ok','failed','skipped'))` |
| `new_reference_id` | `uuid` | `NULL`, FK → `face_references(id)` `ON DELETE SET NULL` |
| `hints` | `jsonb` | `NOT NULL DEFAULT '[]'::jsonb` |
| `reason` | `text` | `NULL` |
| `processed_at` | `timestamptz` | `NULL` |

`PRIMARY KEY (job_id, face_reference_id)` + `CREATE INDEX ON face_reindex_items (job_id, status);`

Adanya tabel item inilah yang membuat job bisa dilanjutkan setelah container
restart di tengah jalan — tanpa itu, reindex 5.000 foto yang terputus di menit ke-40
harus diulang dari nol.

### 3.8 `app_settings` & permission baru — migration `000017`

```sql
INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
  ('face.max_reference_photos',        '5',    'number',  'Jumlah maksimum foto referensi aktif per karyawan', true),
  ('face.enrollment_session_ttl_minutes','30', 'number',  'Masa berlaku sesi enrollment (menit)',              true),
  ('face.min_quality_score',           '0.35', 'number',  'Skor kualitas minimum agar foto diterima sebagai referensi', false),
  ('face.duplicate_check_enabled',     'true', 'boolean', 'Periksa apakah wajah sudah terdaftar di karyawan lain', false),
  ('face.duplicate_threshold',         '0.50', 'number',  'Ambang similarity untuk deteksi wajah duplikat',    false),
  ('face.retention_days_after_resign', '365',  'number',  'Hari sebelum data wajah karyawan resign dihapus',   false),
  ('face.consent_required',            'true', 'boolean', 'Wajibkan consent biometrik sebelum enrollment',     true)
ON CONFLICT (key) DO NOTHING;

-- Permission BARU. Sesuai aturan Fase 1 § 2.3: setiap migration yang menambah
-- permission WAJIB meng-grant-nya ke super_admin di migration yang sama.
INSERT INTO permissions (name, resource, action, description) VALUES
  ('face.reindex', 'face', 'reindex', 'Menjalankan job regenerasi embedding saat model berganti')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'super_admin' AND p.name = 'face.reindex'
ON CONFLICT DO NOTHING;
```

`face.reindex` **tidak** diberikan ke `admin` secara default: reindex mengubah
basis pencocokan seluruh organisasi dan bisa membuat semua orang gagal absen bila
salah dijalankan. Super admin dapat memberikannya ke `admin` lewat
`PUT /roles/{id}/permissions` bila memang diinginkan.

`face.min_quality_score` **berbeda** dari gate `usable` milik inference. `usable`
menjawab "apakah foto ini layak diproses"; `min_quality_score` menjawab "apakah
foto ini cukup baik untuk dijadikan **referensi permanen**" — dan standarnya
memang lebih tinggi, karena referensi dipakai berulang selama bertahun-tahun.

### 3.9 Diagram relasi (tambahan Fase 3)

```
employees 1 ──── * biometric_consents * ──── 1 consent_documents
    │ 1
    ├──── * face_references ──┬── superseded_by (self-FK, saat reindex)
    │         │               └── * face_reindex_items * ──── 1 face_reindex_jobs
    │         └── enrollment_session_id
    │ 1
    └──── * face_enrollment_sessions 1 ──── * face_enrollment_photos
                                              (staging, berumur pendek)
```

---

## 4. Daftar Endpoint / Kontrak API

Base path `/api/v1`. Envelope, `snake_case`, dan katalog error mengikuti
[Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go), ditambah kode di § 2.9.

Kolom **Guard** memakai permission yang **sudah di-seed di Fase 1** — tidak ada
permission baru selain `face.reindex` (§ 3.8).

### 4.1 Ringkasan

| # | Method | Path | Guard | Consent |
|---|---|---|---|---|
| 32 | GET | `/consents/document` | auth | — |
| 33 | GET | `/consents/me` | auth | — |
| 34 | POST | `/consents` | `face.enroll_self` | — |
| 35 | POST | `/consents/withdraw` | auth (self) | — |
| 36 | GET | `/employees/{id}/consent` | `face.read_any` \| `face.read_self`+self | — |
| 37 | POST | `/employees/{id}/consent` | `face.enroll_any` | — |
| 38 | POST | `/face/enrollments` | `face.enroll_self` \| `face.enroll_any` | ✅ |
| 39 | GET | `/face/enrollments/{id}` | pemilik sesi \| `face.read_any` | — |
| 40 | POST | `/face/enrollments/{id}/photos` | idem #38 | ✅ |
| 41 | DELETE | `/face/enrollments/{id}/photos/{pid}` | idem #38 | — |
| 42 | POST | `/face/enrollments/{id}/commit` | idem #38 | ✅ |
| 43 | DELETE | `/face/enrollments/{id}` | idem #38 | — |
| 44 | GET | `/employees/{id}/face-references` | `face.read_any` \| `face.read_self`+self | — |
| 45 | GET | `/face/references/{id}/photo` | `face.read_any` \| `face.read_self`+self | — |
| 46 | PATCH | `/face/references/{id}` | `face.delete_any` | — |
| 47 | DELETE | `/employees/{id}/face-data` | `face.delete_any` | — |
| 48 | GET | `/face/enrollment-status/me` | `face.read_self` | — |
| 49 | POST | `/face/reindex-jobs` | `face.reindex` | — |
| 50 | GET | `/face/reindex-jobs` | `face.reindex` | — |
| 51 | GET | `/face/reindex-jobs/{id}` | `face.reindex` | — |
| 52 | POST | `/face/reindex-jobs/{id}/cancel` | `face.reindex` | — |

Kolom **Consent** = route dibungkus guard `RequireConsent` (§ 2.4b).

Setiap route di atas didaftarkan di `internal/httpx/routes.go` sehingga
`TestAllRoutesHaveGuards` ([Fase 1 § 5.3](02-Fase1.md#53-middleware-authenticate--requirepermission))
tidak gagal — dan sebaliknya, endpoint Fase 3 yang lupa dipasangi guard **akan**
membuatnya gagal.

### 4.2 Consent

#### `GET /consents/document`
Dokumen consent yang sedang aktif. Semua user terautentikasi boleh membacanya —
orang harus bisa membaca apa yang akan ia setujui.

```json
{
  "data": {
    "version": "2026-09-v1",
    "title": "Persetujuan Pemrosesan Data Biometrik",
    "body": "## Persetujuan...\n\nData yang kami proses...",
    "content_hash": "3f2a...",
    "published_at": "2026-09-01T00:00:00Z"
  }
}
```

| Status | code | Kondisi |
|---|---|---|
| 404 | `NOT_FOUND` | Belum ada dokumen aktif — sistem belum siap menerima enrollment |

#### `GET /consents/me`

```json
{
  "data": {
    "status": "granted",
    "document_version": "2026-09-v1",
    "granted_at": "2026-09-02T01:20:33Z",
    "is_current_version": true,
    "method": "self_web"
  }
}
```

`status` bernilai `"none"` bila belum pernah. `is_current_version = false` berarti
karyawan menyetujui versi lama dan perlu menyetujui ulang — tapi **tidak**
membatalkan consent yang sudah ada (lihat E5).

#### `POST /consents` — guard `face.enroll_self`

```json
// request
{ "document_version": "2026-09-v1", "agreed": true }
```

```json
// 201
{ "data": { "status": "granted", "document_version": "2026-09-v1", "granted_at": "..." } }
```

Server menyimpan `content_hash` dari dokumen versi itu, plus `ip` dan `user_agent`.
`method` disimpulkan dari `User-Agent` (`self_web` / `self_mobile`).

| Status | code | Kondisi |
|---|---|---|
| 422 | `VALIDATION_ERROR` | `agreed` bukan `true` — persetujuan harus eksplisit |
| 409 | `CONSENT_VERSION_OUTDATED` | `document_version` bukan versi aktif |
| 409 | `CONSENT_ALREADY_GRANTED` | Sudah ada consent aktif untuk versi yang sama |
| 404 | `NOT_FOUND` | User tidak terhubung ke karyawan (`employee_id` null) |

#### `POST /consents/withdraw` — guard auth (self)

```json
{ "reason": "Tidak lagi bersedia data wajah saya diproses" }
```

Dalam satu transaksi: baris consent aktif → `status='withdrawn'`, dan **semua**
`face_references` aktif milik karyawan itu → `is_active=false`,
`deactivated_reason='consent_withdrawn'`.

```json
// 200
{
  "data": {
    "status": "withdrawn",
    "withdrawn_at": "...",
    "deactivated_reference_count": 3,
    "attendance_mode_hint": "manual"
  }
}
```

`attendance_mode_hint` memberi tahu client bahwa karyawan sekarang perlu
dipindahkan ke jalur manual oleh admin. Perpindahannya **tidak** otomatis —
mengubah `attendance_mode` adalah keputusan HR (butuh `employee.update`), dan
mengotomatiskannya berarti siapa pun bisa memindahkan dirinya sendiri ke jalur
yang tidak diverifikasi wajah.

Audit: `consent.withdrawn` + `face.references.deactivated`.

#### `POST /employees/{id}/consent` — guard `face.enroll_any`
Mencatat consent yang diberikan **di luar sistem** (formulir kertas saat onboarding).

```json
{ "document_version": "2026-09-v1", "signed_at": "2026-09-01T09:00:00Z", "note": "Formulir fisik arsip HR/2026/0142" }
```

`method = 'admin_recorded'`, `recorded_by` = user admin. `signed_at` yang dikirim
client **dicatat sebagai `note`**, sedangkan `granted_at` tetap `now()` dari server —
konsisten dengan aturan timestamp server-side
([Fase 0 § 3](01-Fase0.md#3-skema-database)). Kapan kertasnya ditandatangani adalah
klaim; kapan sistem mencatatnya adalah fakta.

### 4.3 Enrollment

#### `POST /face/enrollments` — buat sesi

```json
// request (semua opsional)
{ "employee_id": "018f4c31-...", "mode": "replace" }
```

- Tanpa `employee_id` → sesi untuk diri sendiri, butuh `face.enroll_self`.
- Dengan `employee_id` → butuh `face.enroll_any`.
- `mode`: `replace` (default — referensi lama dinonaktifkan saat commit) atau
  `append` (menambah ke yang sudah ada, sampai `face.max_reference_photos`).

```json
// 201
{
  "data": {
    "id": "018f5a10-...",
    "employee_id": "018f4c31-...",
    "status": "draft",
    "mode": "replace",
    "required_photos": 3,
    "max_photos": 5,
    "model_version": "buffalo_l@v1",
    "photos": [],
    "expires_at": "2026-09-04T04:05:00Z"
  }
}
```

| Status | code | Kondisi |
|---|---|---|
| 403 | `CONSENT_REQUIRED` | Karyawan belum memberi consent aktif |
| 422 | `ATTENDANCE_MODE_MANUAL` | Karyawan berada di jalur manual |
| 409 | `CONFLICT` | Sudah ada sesi `draft` aktif — response menyertakan `existing_session_id` |
| 409 | `REINDEX_IN_PROGRESS` | Ada job reindex berjalan |
| 409 | `ENROLLMENT_LIMIT_REACHED` | `mode=append` sementara sudah di batas maksimum |
| 503 | `FACE_SERVICE_NOT_CONFIGURED` | `face.model_version` masih `"unset"` — Fase 2 belum dikalibrasi. Kode yang sama dipakai `#53`/`#54` (§ 2.9, K-04) |

Baris terakhir penting: menerima enrollment sebelum kalibrasi selesai berarti
menyimpan embedding yang tidak akan pernah bisa dibandingkan dengan threshold
yang bermakna.

#### `POST /face/enrollments/{id}/photos` — unggah satu foto

`multipart/form-data`:

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `image` | file | ✅ | JPEG/PNG/WebP, ≤ `face.max_image_bytes` |
| `capture_source` | text | ✅ | `web_camera` \| `mobile_camera` \| `admin_upload` |

```json
// 201 — foto diterima
{
  "data": {
    "id": "018f5a11-...",
    "position": 1,
    "quality_score": 0.812,
    "det_score": 0.941,
    "hints": [],
    "photo_bytes": 214088,
    "accepted_count": 1,
    "required_photos": 3,
    "can_commit": false
  }
}
```

```json
// 422 — foto ditolak, TIDAK ADA yang disimpan
{
  "error": {
    "code": "FACE_NOT_USABLE",
    "message": "Foto belum memenuhi syarat",
    "details": [{ "field": "image", "message": "Terlalu gelap. Cari tempat yang lebih terang." }],
    "hints": ["too_dark"],
    "request_id": "01JD..."
  }
}
```

| Status | code | Kondisi |
|---|---|---|
| 422 | `FACE_NOT_USABLE` | `usable=false` dari inference, atau `quality_score < face.min_quality_score` |
| 409 | `DUPLICATE_PHOTO` | `photo_sha256` sudah ada di sesi ini |
| 409 | `ENROLLMENT_LIMIT_REACHED` | Sudah mencapai `max_photos` |
| 409 | `ENROLLMENT_SESSION_EXPIRED` | Sesi kedaluwarsa / sudah di-commit / dibatalkan |
| 413 | `PAYLOAD_TOO_LARGE` | Diteruskan dari inference |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | Diteruskan dari inference |
| 502 / 504 | `UPSTREAM_ERROR` / `UPSTREAM_TIMEOUT` | Inference bermasalah |

`capture_source = 'admin_upload'` hanya diterima bila pemanggil punya
`face.enroll_any`. Karyawan tidak bisa mengaku mengunggah sebagai admin — dan di
Fase 6, halaman enrollment karyawan hanya mengirim `web_camera`.

#### `DELETE /face/enrollments/{id}/photos/{pid}`
Menghapus foto dari staging (DB + objek storage). `204`. Posisi foto sisanya
**tidak** dirapikan ulang — `position` hanya untuk urutan tampilan, dan menyusun
ulangnya menciptakan balapan yang tidak perlu.

#### `POST /face/enrollments/{id}/commit`

```json
// request (opsional)
{ "force_duplicate": false, "reason": null }
```

```json
// 200
{
  "data": {
    "session_id": "018f5a10-...",
    "employee_id": "018f4c31-...",
    "committed_at": "2026-09-04T03:48:12Z",
    "model_version": "buffalo_l@v1",
    "created_reference_ids": ["018f5b01-...", "018f5b02-...", "018f5b03-..."],
    "deactivated_reference_ids": ["018f4f01-...", "018f4f02-..."],
    "active_reference_count": 3,
    "mean_quality_score": 0.784
  }
}
```

| Status | code | Kondisi |
|---|---|---|
| 422 | `ENROLLMENT_INCOMPLETE` | Foto diterima < `required_photos`; `details` menyebut kekurangannya |
| 409 | `ENROLLMENT_MODEL_CHANGED` | `session.model_version` ≠ `app_settings.face.model_version` |
| 409 | `ENROLLMENT_SESSION_EXPIRED` | Sesi tidak lagi `draft` |
| 409 | `FACE_BELONGS_TO_ANOTHER_EMPLOYEE` | Deteksi duplikat (§ 2.6) |
| 403 | `CONSENT_REQUIRED` | Consent dicabut di tengah sesi |
| 409 | `REINDEX_IN_PROGRESS` | Reindex mulai berjalan di tengah sesi |

`force_duplicate: true` hanya berlaku bagi pemegang `face.enroll_any` dan **wajib**
disertai `reason` non-kosong; tanpanya → `422 VALIDATION_ERROR`.

#### `DELETE /face/enrollments/{id}`
Membatalkan sesi: `status='cancelled'`, staging dibersihkan. `204`.

### 4.4 Manajemen referensi

#### `GET /employees/{id}/face-references`

Query: `include_inactive` (default `false`).

```json
{
  "data": [
    {
      "id": "018f5b01-...",
      "position": 1,
      "quality_score": 0.812,
      "det_score": 0.941,
      "model_version": "buffalo_l@v1",
      "capture_source": "web_camera",
      "is_active": true,
      "photo_url": "/api/v1/face/references/018f5b01-.../photo",
      "enrolled_at": "2026-09-04T03:48:12Z",
      "enrolled_by": { "id": "018f4c50-...", "email": "hr@faceclock.local" },
      "deactivated_at": null,
      "deactivated_reason": null
    }
  ],
  "meta": { "active_count": 3, "required_count": 3, "is_enrolled": true }
}
```

**`embedding` tidak pernah muncul di response mana pun.** Ia adalah data biometrik;
tidak ada layar yang butuh melihat 512 angka. Ini diuji, bukan sekadar disepakati
(§ 11.5).

`photo_url` adalah path relatif ke endpoint streaming (§ 4.2 D14), bukan URL
object storage.

#### `GET /face/references/{id}/photo`
Mengalirkan byte foto. `Content-Type` dari `photo_mime`,
`Cache-Control: private, max-age=60`.

| Status | code | Kondisi |
|---|---|---|
| 404 | `NOT_FOUND` | Tidak ada, atau milik karyawan lain sementara pemanggil hanya punya `face.read_self` |
| 410 | `NOT_FOUND` (410 Gone) | `photo_purged_at` terisi — foto sudah dihapus karena retensi |

> Registrasi tambahan ke katalog Fase 0: **410 Gone** dipakai khusus untuk
> "pernah ada, sudah dihapus sesuai kebijakan retensi", dengan `code: "NOT_FOUND"`
> agar client yang tidak peduli bedanya tetap menanganinya sebagai tidak ada.

#### `PATCH /face/references/{id}` — guard `face.delete_any`

```json
{ "is_active": false, "reason": "quality_review" }
```

Hanya menonaktifkan; mengaktifkan kembali **tidak diizinkan** (`422`) — referensi
yang pernah dinonaktifkan karena kualitas atau pencabutan consent tidak boleh
hidup lagi tanpa melewati enrollment baru.

Ditolak (`409 ENROLLMENT_INCOMPLETE`) bila penonaktifan akan membuat jumlah
referensi aktif karyawan turun di bawah `face.min_reference_photos` **kecuali**
`allow_below_minimum: true` disertakan — dengan konsekuensi karyawan itu tidak bisa
absen sampai enroll ulang. Konfirmasi eksplisit lebih baik daripada admin yang
tidak sadar baru saja mengunci seseorang dari sistem absensi.

> Karyawan **tidak** bisa menonaktifkan referensinya sendiri satu per satu
> (guard-nya `face.delete_any`, yang tidak dimiliki role `employee`). Kalau bisa,
> ia dapat membuang referensi terbaiknya sampai pencocokan selalu gagal dan setiap
> absensinya masuk jalur `pending_review` yang lebih longgar. Yang bisa ia lakukan
> adalah **re-enroll** (sesi `mode=replace`), yang mengganti semuanya sekaligus.

#### `DELETE /employees/{id}/face-data` — guard `face.delete_any`
Penghapusan permanen (§ 2.8). Body wajib: `{ "reason": "..." }`.

```json
// 200
{ "data": { "deleted_reference_count": 3, "photos_deleted": 3, "consent_status": "withdrawn" } }
```

#### `GET /face/enrollment-status/me` — guard `face.read_self`
Satu panggilan yang menjawab semua yang dibutuhkan UI enrollment (Fase 6):

```json
{
  "data": {
    "consent": { "status": "granted", "document_version": "2026-09-v1", "is_current_version": true },
    "attendance_mode": "face",
    "is_enrolled": true,
    "active_reference_count": 3,
    "required_photos": 3,
    "max_photos": 5,
    "model_version_matches": true,
    "needs_re_enrollment": false,
    "draft_session_id": null
  }
}
```

`model_version_matches = false` + `needs_re_enrollment = true` adalah cara karyawan
mengetahui bahwa reindex gagal untuk dirinya dan ia harus mendaftar ulang.

### 4.5 Reindex — guard `face.reindex`

#### `POST /face/reindex-jobs`

```json
{ "to_model_version": "buffalo_s@v1", "dry_run": false }
```

```json
// 202
{
  "data": {
    "id": "018f6a01-...",
    "from_model_version": "buffalo_l@v1",
    "to_model_version": "buffalo_s@v1",
    "status": "pending",
    "total_count": 417,
    "affected_employee_count": 139
  }
}
```

`dry_run: true` hanya menghitung dan melaporkan tanpa memanggil inference.

| Status | code | Kondisi |
|---|---|---|
| 409 | `REINDEX_IN_PROGRESS` | Sudah ada job `pending`/`running` |
| 422 | `VALIDATION_ERROR` | `to_model_version` sama dengan yang aktif, atau tidak cocok dengan `GET /ready` inference |
| 503 | `SERVICE_UNAVAILABLE` | Inference tidak siap |

Validasi bahwa `to_model_version` **cocok dengan yang benar-benar dilaporkan
inference** lewat `GET /ready` ([Fase 2 § 4.2](03-Fase2.md#42-get-ready--readiness--introspeksi-konfigurasi))
adalah pengaman utama di sini: menjalankan reindex ke versi yang tidak sesuai
dengan model yang benar-benar berjalan menghasilkan ribuan vektor yang salah label.

#### `GET /face/reindex-jobs/{id}`

```json
{
  "data": {
    "id": "018f6a01-...",
    "status": "running",
    "total_count": 417, "processed_count": 208,
    "succeeded_count": 201, "failed_count": 7,
    "employees_ready_count": 64, "employees_incomplete_count": 3,
    "started_at": "...", "finished_at": null,
    "incomplete_employees": [
      { "employee_id": "018f...", "employee_number": "EMP-0043", "succeeded": 1, "required": 3 }
    ]
  }
}
```

#### `POST /face/reindex-jobs/{id}/cancel`
`status='cancelled'`. Item yang sudah selesai tetap tersimpan (referensi baru
`is_active=false`, tidak mengganggu apa pun) sehingga job berikutnya melewatinya.

---

## 5. Flow

### 5.1 Enrollment mandiri (happy path)

```
Karyawan login  [Fase 1]
 │
 ├─ GET /face/enrollment-status/me
 │     consent.status = "none"  ⇒ tampilkan dokumen consent
 │
 ├─ GET /consents/document → tampilkan teks lengkap
 ├─ POST /consents {document_version, agreed:true}
 │     INSERT biometric_consents {status:'granted', content_hash, ip, user_agent}
 │     audit: consent.granted
 │
 ├─ POST /face/enrollments {mode:"replace"}
 │     RequireConsent lolos
 │     snapshot required_photos, max_photos, model_version
 │     expires_at = now() + 30 menit
 │
 ├─ POST /face/enrollments/{id}/photos  (foto 1)
 │     1. baca file, hitung sha256 → sudah ada di sesi? ⇒ 409 DUPLICATE_PHOTO
 │     2. inference.Embed(ctx, bytes)          [klien Go Fase 2]
 │          error/timeout ⇒ 502/504, TIDAK ada yang disimpan
 │     3. usable == false ⇒ 422 FACE_NOT_USABLE + hints, TIDAK ada yang disimpan
 │     4. quality_score < face.min_quality_score ⇒ 422 FACE_NOT_USABLE
 │          hints diperkaya "quality_below_reference_standard"
 │     5. embedding == nil padahal usable == true ⇒ 502 UPSTREAM_ERROR
 │          (pelanggaran kontrak Fase 2; dicatat error, tidak pernah diabaikan)
 │     6. Put foto ke storage  key = face-staging/{session}/{photo_id}.jpg
 │     7. INSERT face_enrollment_photos {embedding, quality_score, photo_key, ...}
 │
 ├─ (ulangi untuk foto 2 dan 3)
 │
 └─ POST /face/enrollments/{id}/commit
       BEGIN  (SELECT sesi FOR UPDATE)
         a. status masih 'draft'? tidak ⇒ 409
         b. session.model_version == app_settings.face.model_version? tidak ⇒ 409
         c. consent masih 'granted'? tidak ⇒ 403
         d. count(photos) >= required_photos? tidak ⇒ 422
         e. deteksi duplikat (§ 2.6) ⇒ 409 bila ketemu & !force_duplicate
         f. mode='replace' ⇒ UPDATE face_references SET is_active=false,
              deactivated_reason='re_enroll' WHERE employee_id=$1 AND is_active
         g. INSERT face_references (satu baris per foto staging)
              photo_key dipindah: face-staging/... → face/{employee_id}/{ref_id}.jpg
         h. UPDATE session status='committed', committed_at=now()
         i. DELETE face_enrollment_photos WHERE session_id=$1
       COMMIT
       (setelah commit) hapus objek staging yang tersisa dari storage
       audit: face.enrollment.committed {reference_count, mean_quality_score}
```

Langkah (g) menyalin objek di storage lalu menghapus yang lama — bukan mengubah
`photo_key` staging menjadi permanen. Alasannya: kalau transaksi di-rollback
setelah objek dipindah, referensi ke objek staging sudah hilang dan foto menjadi
yatim. Menyalin dulu membuat kegagalan hanya meninggalkan salinan yang disapu job
pembersih.

### 5.2 Enrollment oleh admin

Sama, kecuali:
- `POST /face/enrollments {employee_id}` butuh `face.enroll_any`.
- Guard `RequireConsent` memeriksa consent **karyawan yang bersangkutan**, bukan
  consent admin. Admin tidak bisa "menyetujui atas nama" — kalau consent-nya di
  kertas, ia dicatat lebih dulu lewat `POST /employees/{id}/consent`.
- `capture_source = 'admin_upload'` diizinkan.
- Audit mencatat `enrolled_by` = user admin.

### 5.3 Guard `RequireConsent`

```
RequireConsent(resolve func(*http.Request) (uuid.UUID, error)):
 1. app_settings.face.consent_required == false ⇒ lanjut (jalan keluar untuk dev/demo)
 2. employeeID = resolve(r)      // dari path, body, atau Principal.EmployeeID
 3. SELECT 1 FROM biometric_consents
      WHERE employee_id=$1 AND status='granted'
      (memakai index unik parsial — satu baris atau nol)
 4. tidak ada ⇒ 403 CONSENT_REQUIRED
 5. lanjut
```

Dipasang **setelah** `RequirePermission`, karena urutannya bermakna: orang yang
tidak berhak sama sekali harus mendapat `403 FORBIDDEN`, bukan `403 CONSENT_REQUIRED`
yang membocorkan bahwa karyawan itu ada dan belum memberi consent.

Hasil query di-cache 60 detik per `employee_id`, dengan invalidasi eksplisit saat
consent diberikan/dicabut — pola yang sama dengan cache permission
[Fase 1 § 2.5](02-Fase1.md#25-strategi-token).

### 5.4 Pencabutan consent

```
POST /consents/withdraw
  BEGIN
    UPDATE biometric_consents SET status='withdrawn', withdrawn_at=now(), withdrawn_reason=$2
      WHERE employee_id=$1 AND status='granted'
      RETURNING id                       -- nol baris ⇒ 409 CONFLICT
    UPDATE face_references
      SET is_active=false, deactivated_at=now(), deactivated_reason='consent_withdrawn'
      WHERE employee_id=$1 AND is_active
      RETURNING id                       -- dihitung untuk response & audit
  COMMIT
  invalidasi cache consent
  audit: consent.withdrawn, face.references.deactivated
```

**Foto tidak dihapus di sini.** Mencabut consent menghentikan pemrosesan; menghapus
data adalah permintaan terpisah (`DELETE /employees/{id}/face-data`). Membedakan
keduanya penting karena karyawan mungkin ingin berhenti dipakai tanpa menghapus,
atau menghapus tanpa pernah mencabut — dan mencampurnya menghilangkan pilihan itu.

### 5.5 Job reindex

```
worker (goroutine di faceclock-api, interval 10 detik):
  pg_try_advisory_lock(FACE_REINDEX_LOCK_ID)  -- gagal ⇒ replica lain sedang jalan
  SELECT job WHERE status IN ('pending','running') LIMIT 1
  job.status = 'running', started_at = now()

  loop batch 16 item 'pending':
    ambil foto dari storage (photo_key)
      objek hilang ⇒ item 'failed', reason='photo_missing'
    inference.EmbedBatch(fotos)
      per item:
        usable=true  ⇒ INSERT face_references BARU
                        {embedding baru, model_version=to, photo_key SAMA,
                         is_active=false, quality_score baru}
                        UPDATE item {status:'ok', new_reference_id}
                        UPDATE lama SET superseded_by = baru.id
        usable=false ⇒ item 'failed', hints disimpan
    UPDATE job counters
    job.status == 'cancelled' ⇒ keluar loop

  setelah semua item selesai:
    untuk setiap employee terdampak, dalam SATU transaksi:
      n = count(referensi baru yang berhasil)
      n >= face.min_reference_photos ?
        ya    ⇒ UPDATE lama  SET is_active=false, deactivated_reason='model_reindex'
                 UPDATE baru  SET is_active=true
                 employees_ready_count++
        tidak ⇒ tidak ada yang di-swap; employees_incomplete_count++
                 (karyawan tetap memakai referensi lama, dan Fase 4 akan
                  memfilternya keluar karena model_version tidak cocok →
                  ia jatuh ke jalur fallback sampai enroll ulang)
  job.status='completed', finished_at=now()
  audit: face.reindex.completed {counters}
```

Perubahan `app_settings.face.model_version` ke nilai baru adalah **langkah manual
terpisah** oleh super admin setelah membaca laporan job. Mengotomatiskannya berarti
sistem bisa memindahkan seluruh organisasi ke model baru sambil menyisakan 3 orang
yang tidak bisa absen, tanpa ada manusia yang melihatnya.

---

## 6. Edge Case & Validasi

### 6.1 Consent

| # | Kondisi | Penanganan |
|---|---|---|
| E1 | Enrollment tanpa consent | `403 CONSENT_REQUIRED` dari guard, sebelum satu byte pun dikirim ke inference |
| E2 | Consent dicabut saat sesi enrollment sedang berjalan | Commit memeriksa ulang (§ 5.1c) → `403`. Staging dibersihkan |
| E3 | `agreed: false` atau field tidak ada | `422` — persetujuan harus tindakan eksplisit, bukan default |
| E4 | Menyetujui versi dokumen yang bukan versi aktif | `409 CONSENT_VERSION_OUTDATED` |
| E5 | Dokumen consent versi baru dipublikasikan | Consent lama **tetap berlaku** (`status='granted'`), tapi `is_current_version=false`. Admin melihat daftar yang perlu menyetujui ulang (Fase 5). Membatalkan consent lama secara otomatis akan mengunci seluruh perusahaan dari absensi pada hari publikasi |
| E6 | Karyawan tanpa akun user diberi consent oleh admin | Diizinkan — `method='admin_recorded'`, `recorded_by` diisi. Consent milik **karyawan**, bukan milik akun |
| E7 | Consent dicabut, lalu diberikan lagi | Baris baru `granted`. Referensi lama tetap `is_active=false` — harus enroll ulang. Menghidupkan kembali referensi yang dinonaktifkan karena pencabutan akan membuat pencabutan itu tidak berarti |
| E8 | Belum ada `consent_documents` aktif | `POST /face/enrollments` → `503`; `GET /consents/document` → `404`. Seeder Fase 3 menyediakan dokumen awal |
| E9 | Dua tab mencabut consent bersamaan | `UPDATE ... WHERE status='granted' RETURNING id`; yang kedua mengembalikan nol baris → `409`. Tidak ada `SELECT` sebelumnya yang bisa dibalap |

### 6.2 Kualitas & inference

| # | Kondisi | Penanganan |
|---|---|---|
| E10 | `usable=false` (hint apa pun) | `422 FACE_NOT_USABLE` + `hints`; **tidak ada** yang disimpan ke DB maupun storage |
| E11 | `usable=true` tapi `embedding = null` | Pelanggaran kontrak [Fase 2 § 4.3](03-Fase2.md#43-post-v1embed). `502 UPSTREAM_ERROR`, log level `error`, metrik `inference_contract_violation_total`. **Tidak pernah** diperlakukan sebagai foto valid |
| E12 | `embedding_dim ≠ 512` | `502 UPSTREAM_ERROR`. Diperiksa di client Go sebelum menyentuh database — `vector(512)` akan menolaknya juga, tapi pesan errornya tidak akan menjelaskan apa pun |
| E13 | `model_version` dari response ≠ `session.model_version` | Foto ditolak `409 ENROLLMENT_MODEL_CHANGED`; sesi ditandai perlu dibuat ulang |
| E14 | Inference mati saat unggah foto | `504 UPSTREAM_TIMEOUT` / `502`. Foto tidak disimpan; karyawan mengulang unggah. Tidak ada jalur "simpan dulu, embed nanti" — referensi tanpa embedding tidak berguna dan hanya menumpuk data biometrik |
| E15 | `quality_score` lolos `usable` tapi di bawah `face.min_quality_score` | `422 FACE_NOT_USABLE` dengan hint tambahan; standar referensi memang lebih tinggi (§ 3.8) |
| E16 | Semua 3 foto identik (orang menekan tombol 3× tanpa bergerak) | Foto kedua & ketiga tertolak `409 DUPLICATE_PHOTO` bila sha256 sama. Bila berbeda beberapa byte, tetap diterima — variasi pose tidak bisa ditegakkan secara teknis, jadi UI Fase 6 yang memandu ("hadap kiri", "hadap kanan"), dan itu ditulis sebagai keterbatasan yang diketahui |

### 6.3 Sesi & konkurensi

| # | Kondisi | Penanganan |
|---|---|---|
| E17 | Dua tab membuat sesi bersamaan | Index unik parsial `face_enrollment_sessions_draft_uniq` → yang kedua `409` dengan `existing_session_id` |
| E18 | Sesi kedaluwarsa | Job pembersih tiap 5 menit: `status='expired'`, staging DB + objek storage dihapus. `POST .../photos` pada sesi kedaluwarsa → `409` |
| E19 | Commit dua kali (double-click) | `SELECT ... FOR UPDATE` + cek `status='draft'` → yang kedua `409 ENROLLMENT_SESSION_EXPIRED`. Aman untuk di-retry client |
| E20 | Container mati setelah objek disalin, sebelum COMMIT | Transaksi rollback; objek permanen menjadi yatim → disapu job pembersih objek yatim (harian, membandingkan bucket dengan `face_references.photo_key`) |
| E21 | `mode=append` melampaui `max_photos` di tengah sesi | Foto ditolak `409 ENROLLMENT_LIMIT_REACHED` saat diunggah, bukan saat commit |
| E22 | Reindex mulai berjalan saat ada sesi draft | Commit → `409 REINDEX_IN_PROGRESS`. Sesi tetap ada dan bisa di-commit setelah reindex selesai, **kecuali** `model_version` sudah berubah (E13) |

### 6.4 Keamanan & integritas

| # | Kondisi | Penanganan |
|---|---|---|
| E23 | Foto diunggah dari galeri, bukan kamera | **Fase 3 tidak bisa membuktikannya.** `capture_source` adalah klaim client. Mitigasi ada di Fase 6 (`getUserMedia`, tanpa input file) dan Fase 7 (kamera in-app + liveness). Ditulis eksplisit sebagai keterbatasan, dan `capture_source` disimpan supaya audit bisa menyaring `admin_upload` bila ada kecurigaan |
| E24 | Karyawan A meng-enroll wajah karyawan B | Ditangkap deteksi duplikat (§ 2.6) → `409 FACE_BELONGS_TO_ANOTHER_EMPLOYEE`. Ini alasan utama D16 direkomendasikan aktif |
| E25 | Karyawan mengakses `/face/references/{id}/photo` milik orang lain | `404`, bukan `403` — pola ownership [Fase 1 § 5.4](02-Fase1.md#54-pola-akses-self) |
| E26 | `embedding` bocor lewat response | Diuji: integration test memindai seluruh body untuk array float berpanjang 512 (§ 11.5) |
| E27 | `employee_id` di body dipakai untuk enroll orang lain tanpa `face.enroll_any` | Handler mengabaikan `employee_id` bila pemanggil hanya punya `face.enroll_self`, dan memakai `Principal.EmployeeID`. Tidak dibalas error — parameter yang tidak berhak diisi cukup diabaikan, sesuai pola [Fase 1 E39](02-Fase1.md#63-data--konkurensi) |
| E28 | Karyawan `deleted_at IS NOT NULL` atau `employment_status='resigned'` | `POST /face/enrollments` → `404` / `422`. Query selalu menyertakan `deleted_at IS NULL` |
| E29 | Admin menonaktifkan referensi sampai karyawan tidak bisa absen | `409 ENROLLMENT_INCOMPLETE` kecuali `allow_below_minimum: true` dikirim eksplisit; keduanya masuk audit |
| E30 | Log memuat byte foto / nilai embedding / nama file asli | Dilarang [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go); diuji ulang di fase ini dengan fixture nyata |

### 6.5 Reindex & retensi

| # | Kondisi | Penanganan |
|---|---|---|
| E31 | Objek foto hilang dari storage saat reindex | Item `failed`, `reason='photo_missing'`. Job tetap jalan; karyawan masuk daftar `incomplete_employees` |
| E32 | Reindex menghasilkan < minimum untuk satu karyawan | Tidak ada swap untuk karyawan itu (§ 5.5). Referensi lama tetap, tapi `model_version` tidak lagi cocok → Fase 4 memperlakukannya sebagai tidak ter-enroll |
| E33 | Container restart di tengah reindex | `face_reindex_items` menyimpan progres; job dilanjutkan dari item `pending` |
| E34 | Dua replica menjalankan worker reindex | `pg_try_advisory_lock` + index unik parsial pada job aktif |
| E35 | `to_model_version` tidak cocok dengan `GET /ready` inference | `422` saat membuat job — sebelum satu vektor pun dibuat |
| E36 | Job retensi menghapus foto yang masih dirujuk referensi aktif | Job hanya menyentuh karyawan `employment_status='resigned'` dengan `updated_at` melewati ambang; setelah menghapus objek, ia mengisi `photo_purged_at` dan **mengosongkan** `photo_key` dalam transaksi yang sama |
| E37 | Foto sudah di-purge lalu ada permintaan reindex | Item `skipped`, `reason='photo_purged'` — bukan `failed`. Ini keadaan yang wajar, bukan kesalahan |

### 6.6 Validasi input

| Field | Aturan |
|---|---|
| `image` | wajib; MIME dari magic bytes; ≤ `face.max_image_bytes`; format sesuai `face.accepted_mime_types` |
| `capture_source` | wajib; salah satu dari `web_camera`, `mobile_camera`, `admin_upload`; `admin_upload` butuh `face.enroll_any` |
| `mode` | `replace` \| `append`; default `replace` |
| `employee_id` | UUID valid; diabaikan bila pemanggil tidak punya `face.enroll_any` (E27) |
| `document_version` | wajib; harus versi aktif |
| `agreed` | wajib, harus `true` literal |
| `reason` (withdraw / delete / force_duplicate) | wajib untuk `force_duplicate` dan `DELETE face-data`; 3–500 karakter |
| `to_model_version` | wajib; ≠ versi aktif; harus cocok dengan `/ready` inference |
| `include_inactive` | boolean; default `false` |

---

## 7. Struktur Folder

Yang **ditambahkan** ke `apps/faceclock-api/`:

```
apps/faceclock-api/
├── internal/
│   ├── consent/                        # ← BARU
│   │   ├── handler.go                  # #32–#37
│   │   ├── service.go                  # grant, withdraw (+ menonaktifkan referensi)
│   │   ├── repository.go
│   │   ├── middleware.go               # RequireConsent (§ 5.3)
│   │   ├── cache.go                    # cache consent 60s + invalidasi
│   │   ├── dto.go
│   │   ├── service_test.go
│   │   └── middleware_test.go
│   ├── face/                           # ← BARU
│   │   ├── enrollment/
│   │   │   ├── handler.go              # #38–#43
│   │   │   ├── service.go              # sesi, unggah foto, commit
│   │   │   ├── repository.go
│   │   │   ├── commit.go               # transaksi commit (§ 5.1)
│   │   │   ├── duplicate.go            # deteksi wajah duplikat (§ 2.6)
│   │   │   ├── cleanup.go              # job sesi kedaluwarsa
│   │   │   ├── dto.go
│   │   │   ├── service_test.go
│   │   │   ├── commit_test.go
│   │   │   └── duplicate_test.go
│   │   ├── reference/
│   │   │   ├── handler.go              # #44–#48
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── photo.go                # streaming foto (§ 4.2 D14)
│   │   │   ├── erasure.go              # penghapusan PDP (§ 2.8)
│   │   │   ├── retention.go            # job retensi
│   │   │   └── *_test.go
│   │   └── reindex/
│   │       ├── handler.go              # #49–#52
│   │       ├── service.go
│   │       ├── worker.go               # goroutine + advisory lock (§ 5.5)
│   │       ├── repository.go
│   │       └── worker_test.go
│   ├── storage/
│   │   ├── storage.go                  # (dari Fase 0) interface Store
│   │   ├── local.go                    # (dari Fase 0)
│   │   ├── s3.go                       # ← BARU: driver MinIO/S3 (D13)
│   │   ├── keys.go                     # ← BARU: skema key (§ 2.1)
│   │   ├── orphan.go                   # ← BARU: job penyapu objek yatim
│   │   └── s3_test.go
│   └── httpx/
│       └── routes.go                   # (diubah) daftarkan #32–#52 + guard-nya
├── cmd/
│   └── seed/main.go                    # (diubah) seed consent_documents awal
├── migrations/
│   ├── 000012_create_consents.{up,down}.sql
│   ├── 000013_add_employee_attendance_mode.{up,down}.sql
│   ├── 000014_create_face_references.{up,down}.sql
│   ├── 000015_create_face_enrollment.{up,down}.sql
│   ├── 000016_create_face_reindex.{up,down}.sql
│   └── 000017_face_settings_and_permission.{up,down}.sql
└── test/
    ├── integration/
    │   ├── consent_test.go
    │   ├── enrollment_test.go
    │   ├── enrollment_concurrency_test.go
    │   ├── face_reference_test.go
    │   ├── reindex_test.go
    │   └── rbac_matrix_test.go         # (diperluas) + 21 endpoint Fase 3
    └── fixtures/
        └── faces/                       # foto uji; TIDAK di-commit bila wajah nyata
```

Perubahan di `services/faceclock-inference/`: **tidak ada.** Fase 3 hanya
mengonsumsi kontrak Fase 2. Kalau ternyata ada yang perlu diubah di sana, itu
perubahan kontrak yang harus dicatat di `docs/api/fase2-inference.md`, bukan
tambalan diam-diam.

Dokumen yang dihasilkan:

```
docs/
├── api/
│   └── fase3-enrollment.md              # kontrak #32–#52
└── face/
    ├── consent-policy.md                # teks dokumen consent + kebijakan versi
    ├── biometric-retention-policy.md    # retensi, penghapusan, siapa berwenang
    └── model-version-and-reindex.md     # (dari Fase 2, diperbarui dengan prosedur nyata)
```

---

## 8. Checklist Task

### 8.0 Prasyarat
- [ ] **Konfirmasi D13–D16** (§ 2.0)
- [ ] Fase 2 selesai: `app_settings.face.model_version` **bukan** `"unset"`
- [ ] Teks dokumen consent disetujui pihak yang berwenang (HR/legal), bukan dikarang developer

### 8.1 Migration & skema
- [ ] `000012_create_consents` — `consent_documents` + `biometric_consents` + index unik parsial
- [ ] `000013_add_employee_attendance_mode`
- [ ] `000014_create_face_references` — termasuk `vector(512)`, CHECK, dan index B-tree (**tanpa** HNSW)
- [ ] `000015_create_face_enrollment` — sesi + staging
- [ ] `000016_create_face_reindex` — jobs + items
- [ ] `000017_face_settings_and_permission` — 7 setting + permission `face.reindex` + grant ke `super_admin`
- [ ] Semua `.down.sql` ditulis & diuji (`down` sampai 000011, lalu `up` lagi)
- [ ] Trigger `set_updated_at()` dipasang di semua tabel baru
- [ ] Verifikasi `pgvector-go` bisa scan/serialize `vector(512)` lewat `pgx`

### 8.2 Storage
- [ ] `internal/storage/s3.go` — driver MinIO/S3, `force_path_style`, SSE
- [ ] `internal/storage/keys.go` — skema key deterministik (§ 2.1)
- [ ] MinIO ditambahkan ke `deploy/docker-compose.yml` + bucket dibuat otomatis saat init
- [ ] Bucket **tidak** publik; diverifikasi dengan `curl` langsung ke endpoint MinIO
- [ ] `internal/storage/orphan.go` — job penyapu objek yatim (harian)
- [ ] Env baru di `deploy/.env.example` (§ 2.1)

### 8.3 Consent
- [ ] Seeder: dokumen consent versi awal (`is_active = true`), teks dari HR
- [ ] `GET /consents/document`, `GET /consents/me`
- [ ] `POST /consents` — simpan `content_hash`, `ip`, `user_agent`
- [ ] `POST /consents/withdraw` — transaksi + menonaktifkan referensi (§ 5.4)
- [ ] `GET/POST /employees/{id}/consent`
- [ ] Middleware `RequireConsent` + cache + invalidasi
- [ ] Test: enrollment tanpa consent → `403 CONSENT_REQUIRED` sebelum memanggil inference

### 8.4 Enrollment
- [ ] `POST /face/enrollments` — snapshot setting, cek consent/mode/reindex/model_version
- [ ] `POST /face/enrollments/{id}/photos` — sha256, inference, gate kualitas, staging
- [ ] Penegakan kontrak Fase 2: `usable=true` tapi `embedding=nil` → `502` (E11)
- [ ] Penegakan `embedding_dim == 512` (E12)
- [ ] `DELETE .../photos/{pid}` — hapus DB + objek staging
- [ ] `POST .../commit` — transaksi penuh (§ 5.1), termasuk salin objek sebelum commit
- [ ] `DELETE /face/enrollments/{id}`
- [ ] Deteksi duplikat (§ 2.6) + `force_duplicate` + audit override
- [ ] Job pembersih sesi kedaluwarsa (tiap 5 menit)
- [ ] Test konkurensi: dua commit bersamaan, dua sesi bersamaan

### 8.5 Manajemen referensi
- [ ] `GET /employees/{id}/face-references` — dengan `meta.active_count` / `is_enrolled`
- [ ] `GET /face/references/{id}/photo` — streaming, header cache privat, audit saat `face.read_any`
- [ ] `PATCH /face/references/{id}` — nonaktifkan saja, dengan penjagaan minimum
- [ ] `DELETE /employees/{id}/face-data` — prosedur PDP (§ 2.8)
- [ ] `GET /face/enrollment-status/me`
- [ ] Job retensi (`face.retention_days_after_resign`)
- [ ] Test: `embedding` tidak pernah muncul di response mana pun

### 8.6 Reindex
- [ ] `POST /face/reindex-jobs` + validasi terhadap `GET /ready` inference
- [ ] `GET /face/reindex-jobs`, `GET /{id}`, `POST /{id}/cancel`
- [ ] Worker + `pg_try_advisory_lock` + batch 16 (`/v1/embed-batch`)
- [ ] Swap per karyawan, atomik, hanya bila ≥ minimum (§ 5.5)
- [ ] Resume setelah restart (E33)
- [ ] Test dengan `FakeClient`: 100 referensi, 5 gagal, verifikasi swap parsial benar

### 8.7 Integrasi & lintas-fase
- [ ] Daftarkan 21 route baru di `internal/httpx/routes.go` — `TestAllRoutesHaveGuards` tetap lulus
- [ ] Perluas `rbac_matrix_test.go` dengan 21 endpoint Fase 3 × 4 principal
- [ ] Daftarkan 12 error code baru (§ 2.9) di `internal/httpx/errors.go`
- [ ] Tambah action audit baru di `internal/audit/actions.go`
- [ ] Perbarui `docs/adr/0003-api-conventions.md` dengan kode & status 410
- [ ] `docs/api/fase3-enrollment.md`
- [ ] `docs/face/consent-policy.md`, `docs/face/biometric-retention-policy.md`
- [ ] `DONE-Fase-3.md` sesuai Protokol Handoff master plan § 10.4

---

## 9. Dependencies

**Prasyarat:**

| Dari | Yang dibutuhkan |
|---|---|
| Fase 0 | Envelope + katalog error; middleware chain; `storage.Store`; konvensi skema; aturan logging; ekstensi `vector` |
| Fase 1 | `employees`, `users`; `Principal{employee_id, permissions}`; `RequirePermission` / `RequireAnyPermission`; pola ownership → 404; `app_settings` + validator per-key; `audit_logs` + recorder; `TestAllRoutesHaveGuards`; permission `face.*` **sudah di-seed** |
| Fase 2 | Kontrak `/v1/embed` & `/v1/embed-batch`; kontrak data `vector(512)` L2-normalized; kosakata `hints` + `hints.go`; klien Go + `FakeClient`; `GET /ready` untuk validasi `model_version`; `face.model_version` sudah terisi hasil kalibrasi |
| Eksternal | Teks dokumen consent dari HR/legal; MinIO (bila D13 disetujui) |

**Utang lintas-fase yang jatuh tempo di fase ini**
([Fase 2 § 13.3](03-Fase2.md#133-risiko-lintas-fase-yang-belum-terselesaikan)):

| Risiko | Status di Fase 3 |
|---|---|
| **R4** — karyawan menolak consent | ✅ Diselesaikan: `employees.attendance_mode = 'manual'` (§ 2.4d, § 3.3) |
| **R5** — reindex saat model berganti | ✅ Diselesaikan: job + tabel + worker (§ 2.7, § 3.7, § 5.5). UI-nya Fase 5 |
| **R7** — volume penyimpanan foto | ✅ Sebagian: D13 memilih object storage + retensi; volume foto **absensi** ditangani Fase 4 |
| **R1** — lisensi model InsightFace | ⛔ Masih terbuka; bukan pekerjaan Fase 3 tapi harus sudah dijawab sebelum produksi |
| **R2** — Dataset B untuk kalibrasi | ⛔ Blocker DoD Fase 4, bukan Fase 3 |
| **R3** — spoofing foto-dari-layar | ⛔ Fase 6/7. Fase 3 mencatat `capture_source` sebagai bahan audit (E23) |
| **R6** — monorepo vs multi-repo | ✅ **Ditutup** — dikunci monorepo 2026-09-04 ([Fase 0 § 2.1](01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)) |

**Yang bergantung pada fase ini:**

| Fase | Mengambil apa |
|---|---|
| Fase 4 | `face_references` (embedding + `model_version` + `is_active`), `employees.attendance_mode`, guard `RequireConsent`, `face_references_lookup_idx` |
| Fase 5 | Antrian "perlu enroll ulang" dari reindex; pemantauan job; kelola dokumen consent; lihat foto referensi |
| Fase 6 | Alur sesi bertahap + `hints` untuk coaching kamera; `GET /face/enrollment-status/me` |
| Fase 7 | Endpoint yang sama persis; `capture_source='mobile_camera'` |

---

## 10. Definition of Done

1. Enam migration (`000012`–`000017`) jalan bersih dari state Fase 2, dan `down`
   mengembalikannya tanpa sisa.
2. Seeder menghasilkan satu `consent_documents` aktif; menjalankannya 3× tetap
   menghasilkan satu.
3. Permission `face.reindex` ada dan ter-grant ke `super_admin`; query "super_admin
   punya semua permission" ([Fase 1 § 11.5](02-Fase1.md#115-verifikasi-database))
   masih menghasilkan `total = granted`.
4. `TestAllRoutesHaveGuards` lulus dengan 21 route baru terdaftar.
5. Matriks RBAC diperluas dan lulus: `employee` bisa `POST /face/enrollments`
   untuk dirinya sendiri, **tidak bisa** untuk orang lain, dan **tidak bisa**
   `PATCH /face/references/{id}` maupun `POST /face/reindex-jobs`.
6. Enrollment tanpa consent → `403 CONSENT_REQUIRED`, dan **inference tidak pernah
   dipanggil** (diverifikasi lewat `FakeClient` yang menghitung panggilan).
7. Alur lengkap berhasil: consent → sesi → 3 foto → commit → 3 baris
   `face_references` dengan `is_active=true`, `model_version` benar, dan **3
   embedding berbeda** (bukan hasil rata-rata — diuji dengan membandingkan
   ketiganya satu sama lain).
8. Foto yang `usable=false` **tidak meninggalkan jejak apa pun**: nol baris di
   `face_enrollment_photos`, nol objek di bucket staging.
9. Commit dengan 2 foto → `422 ENROLLMENT_INCOMPLETE`; tidak ada baris tertulis.
10. Re-enroll (`mode=replace`) menonaktifkan referensi lama dan mengaktifkan yang
    baru dalam satu transaksi; tidak pernah ada momen di mana karyawan punya nol
    referensi aktif (diuji dengan pembacaan konkuren).
11. Deteksi duplikat menangkap wajah yang sudah terdaftar di karyawan lain →
    `409`, dan response **tidak** menyebut nama/id karyawan lain itu.
12. Pencabutan consent menonaktifkan semua referensi dalam transaksi yang sama;
    `GET /face/enrollment-status/me` langsung melaporkan `is_enrolled: false`.
13. `DELETE /employees/{id}/face-data` menghapus baris **dan** objek storage;
    `biometric_consents` tetap ada sebagai bukti.
14. `embedding` tidak pernah muncul di response API mana pun — dibuktikan test
    yang memindai seluruh body integration test untuk array numerik ≥ 512 elemen.
15. Log tidak memuat byte foto, nama file asli, maupun nilai embedding — test
    Fase 0 § 11.6 diperluas dengan fixture foto nyata dan tetap lulus.
16. Foto hanya bisa diakses lewat API dengan permission yang benar; akses langsung
    ke endpoint MinIO tanpa kredensial → ditolak.
17. Job reindex: 100 referensi dengan 5 foto rusak → karyawan yang hasilnya
    memenuhi minimum ter-swap, yang tidak memenuhi **tidak** ter-swap dan muncul
    di `incomplete_employees`.
18. Reindex bisa dilanjutkan setelah proses di-kill di tengah jalan.
19. Semua test integration berjalan dengan `FakeClient` — **tidak ada** test yang
    membutuhkan container ONNX. Ada satu test bertanda `//go:build inference_e2e`
    yang memakai service sungguhan, dijalankan terpisah.
20. `make lint` & `make test` lulus; CI hijau.
21. `docs/api/fase3-enrollment.md`, `docs/face/consent-policy.md`,
    `docs/face/biometric-retention-policy.md` lengkap.
22. `DONE-Fase-3.md` ada, memuat keputusan D13–D16 final beserta alasannya.

---

## 11. Cara Test / Verifikasi

### 11.1 Persiapan

```bash
make reset && make up
docker compose -f deploy/docker-compose.yml exec faceclock-api /app/seed
API=http://localhost:8080/api/v1
AT=$(curl -s -X POST $API/auth/login -H 'Content-Type: application/json' \
     -d '{"email":"budi@faceclock.local","password":"PasswordAwal123"}' | jq -r .data.access_token)
```

### 11.2 Alur enrollment (curl)

```bash
# 1) Status awal
curl -s $API/face/enrollment-status/me -H "Authorization: Bearer $AT" | jq

# 2) Enrollment tanpa consent → 403 CONSENT_REQUIRED
curl -s -X POST $API/face/enrollments -H "Authorization: Bearer $AT" \
  -H 'Content-Type: application/json' -d '{}' | jq .error.code

# 3) Baca & setujui consent
VER=$(curl -s $API/consents/document -H "Authorization: Bearer $AT" | jq -r .data.version)
curl -s -X POST $API/consents -H "Authorization: Bearer $AT" \
  -H 'Content-Type: application/json' \
  -d "{\"document_version\":\"$VER\",\"agreed\":true}" | jq .data.status

# 4) Buat sesi
SID=$(curl -s -X POST $API/face/enrollments -H "Authorization: Bearer $AT" \
  -H 'Content-Type: application/json' -d '{"mode":"replace"}' | jq -r .data.id)

# 5) Unggah 3 foto
for i in 1 2 3; do
  curl -s -X POST $API/face/enrollments/$SID/photos -H "Authorization: Bearer $AT" \
    -F "image=@fixtures/faces/budi_$i.jpg" -F "capture_source=web_camera" \
  | jq '{pos:.data.position, qs:.data.quality_score, can_commit:.data.can_commit}'
done

# 6) Foto gelap → 422 FACE_NOT_USABLE
curl -s -X POST $API/face/enrollments/$SID/photos -H "Authorization: Bearer $AT" \
  -F "image=@fixtures/faces/dark.jpg" -F "capture_source=web_camera" \
| jq '{code:.error.code, hints:.error.hints}'

# 7) Commit
curl -s -X POST $API/face/enrollments/$SID/commit -H "Authorization: Bearer $AT" \
  -H 'Content-Type: application/json' -d '{}' \
| jq '{refs:(.data.created_reference_ids|length), mean:.data.mean_quality_score}'

# 8) Verifikasi
curl -s $API/face/enrollment-status/me -H "Authorization: Bearer $AT" \
| jq '{enrolled:.data.is_enrolled, count:.data.active_reference_count}'
```

### 11.3 Verifikasi database — inti fase ini

```sql
-- Tepat 3 referensi aktif, model_version benar
SELECT employee_id, count(*) AS aktif, min(model_version), round(avg(quality_score)::numeric,3)
FROM face_references WHERE is_active GROUP BY employee_id;

-- BUKTI ketiga embedding TERPISAH, bukan rata-rata:
-- similarity antar-referensi milik orang yang sama harus tinggi TAPI tidak 1.0.
-- Kalau ada yang persis 1.0, berarti foto identik atau vektornya di-average.
SELECT a.position, b.position, round((1 - (a.embedding <=> b.embedding))::numeric, 4) AS sim
FROM face_references a
JOIN face_references b ON a.employee_id = b.employee_id AND a.id < b.id
WHERE a.is_active AND b.is_active AND a.employee_id = '<employee_id>';

-- Semua vektor ter-L2-normalize (kontrak Fase 2 § 3.2)
SELECT id, round((embedding <#> embedding * -1)::numeric, 6) AS norm_kuadrat
FROM face_references WHERE is_active LIMIT 10;   -- harus ≈ 1.000000

-- Tidak ada referensi aktif tanpa consent aktif — invariant utama fase ini
SELECT fr.employee_id
FROM face_references fr
LEFT JOIN biometric_consents bc
  ON bc.employee_id = fr.employee_id AND bc.status = 'granted'
WHERE fr.is_active AND bc.id IS NULL;            -- harus nol baris

-- Tidak ada staging yang tertinggal setelah commit
SELECT count(*) FROM face_enrollment_photos p
JOIN face_enrollment_sessions s ON s.id = p.session_id
WHERE s.status <> 'draft';                       -- harus 0

-- Tidak ada index ANN (kontrak Fase 2 § 2.4)
SELECT indexname, indexdef FROM pg_indexes
WHERE tablename = 'face_references' AND indexdef ILIKE '%hnsw%' OR indexdef ILIKE '%ivfflat%';
                                                 -- harus nol baris
```

### 11.4 Test dengan `FakeClient` (tanpa ONNX)

Semua integration test memakai `inference.FakeClient` dari
[Fase 2 § 7.2](03-Fase2.md#72-yang-ditambahkan-di-appsfaceclock-api):

```go
func TestCommitRequiresMinimumPhotos(t *testing.T) {
    fake := inference.NewFake()
    fake.OnEmbed(func(_ []byte) (*inference.EmbedResult, error) {
        return inference.Usable(randomUnitVector(512), 0.80, "buffalo_l@v1"), nil
    })
    ...
    // hanya 2 foto → commit harus 422 ENROLLMENT_INCOMPLETE
}

func TestUnusablePhotoLeavesNoTrace(t *testing.T) {
    fake.OnEmbed(func(_ []byte) (*inference.EmbedResult, error) {
        return inference.Unusable([]string{"too_dark"}, "buffalo_l@v1"), nil
    })
    // assert: 422, nol baris face_enrollment_photos, nol objek di bucket staging
}

func TestContractViolationUsableButNilEmbedding(t *testing.T) {
    fake.OnEmbed(func(_ []byte) (*inference.EmbedResult, error) {
        return &inference.EmbedResult{Usable: true, Embedding: nil,
                                      ModelVersion: "buffalo_l@v1"}, nil
    })
    // assert: 502 UPSTREAM_ERROR, metrik inference_contract_violation_total naik,
    //         nol baris tertulis
}

func TestConsentGateRunsBeforeInference(t *testing.T) {
    // tanpa consent: assert 403 DAN fake.EmbedCallCount() == 0
}
```

### 11.5 Test kebocoran data biometrik

```go
func TestEmbeddingNeverLeaks(t *testing.T) {
    for _, resp := range captureAllResponses(t) {   // semua response integration test
        var any map[string]any
        json.Unmarshal(resp.Body, &any)
        assert.False(t, containsNumericArrayOfLength(any, 512),
            "response %s %s membocorkan embedding", resp.Method, resp.Path)
    }
}
```

Ditambah pemeriksaan `grep` pada output log untuk pola base64 panjang dan untuk
nama file fixture.

### 11.6 Test konkurensi

| Skenario | Harapan |
|---|---|
| Dua `POST /face/enrollments` bersamaan | Satu `201`, satu `409` dengan `existing_session_id` |
| Dua `commit` bersamaan pada sesi yang sama | Satu `200`, satu `409`; tepat 3 referensi tertulis |
| Commit sementara consent dicabut di transaksi lain | Salah satu gagal; tidak pernah ada referensi aktif tanpa consent aktif |
| Pembacaan `GET .../face-references` selama re-enroll berjalan | Tidak pernah mengembalikan `active_count = 0` |
| Dua worker reindex (dua replica) | Hanya satu yang memegang advisory lock |

### 11.7 Verifikasi storage

```bash
# Bucket tidak boleh bisa dibaca tanpa kredensial
curl -s -o /dev/null -w "%{http_code}\n" \
  http://localhost:9000/faceclock-face/face/<employee_id>/<ref_id>.jpg     # 403

# Foto hanya lewat API, dan hanya oleh yang berhak
curl -s -o /dev/null -w "%{http_code}\n" \
  $API/face/references/$REF/photo -H "Authorization: Bearer $AT"           # 200 (pemilik)
curl -s -o /dev/null -w "%{http_code}\n" \
  $API/face/references/$REF/photo -H "Authorization: Bearer $AT_ORANG_LAIN" # 404

# Staging bersih setelah commit
docker compose -f deploy/docker-compose.yml exec minio \
  mc ls --recursive local/faceclock-face/face-staging/                      # kosong
```

### 11.8 Test reindex

```bash
# Dry run dulu
curl -s -X POST $API/face/reindex-jobs -H "Authorization: Bearer $SA_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"to_model_version":"buffalo_s@v1","dry_run":true}' | jq

# Job sungguhan, lalu pantau
JID=$(curl -s -X POST $API/face/reindex-jobs -H "Authorization: Bearer $SA_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"to_model_version":"buffalo_s@v1"}' | jq -r .data.id)
watch -n2 "curl -s $API/face/reindex-jobs/$JID -H 'Authorization: Bearer $SA_TOKEN' \
  | jq '{s:.data.status,p:.data.processed_count,t:.data.total_count,inc:.data.employees_incomplete_count}'"
```

Uji ketahanan: jalankan job pada 100 referensi, `docker compose restart faceclock-api`
di tengah jalan, pastikan job dilanjutkan dan hasil akhirnya sama dengan job yang
tidak diinterupsi.

---

## 12. Referensi Silang ke "Isu Lintas-Fase" (Master Plan § 9)

| Isu lintas-fase | Bagaimana Fase 3 memenuhinya |
|---|---|
| **Data biometrik = data sensitif (UU PDP)** | Consent berversi dengan `content_hash` sebagai bukti isi yang disetujui; guard `RequireConsent` di semua route biometrik; pencabutan yang berakibat nyata (referensi dinonaktifkan); penghapusan permanen dengan prosedur & audit; retensi otomatis; foto di bucket privat terenkripsi; embedding tidak pernah keluar lewat API; foto yang gagal gate tidak pernah disimpan (minimalisasi data) |
| **Threshold configurable** | `face.min_quality_score`, `face.duplicate_threshold`, `face.min/max_reference_photos` semuanya di `app_settings`, tidak ada yang hardcode |
| **RBAC konsisten di level API** | 21 route baru terdaftar di `routes.go` dan diuji `TestAllRoutesHaveGuards`; matriks RBAC diperluas; pola ownership → 404; `face.reindex` sengaja tidak diberikan ke `admin` |
| **Timestamp server-side** | `granted_at`, `created_at`, `committed_at` semuanya `now()` sisi DB. `signed_at` dari admin diperlakukan sebagai catatan, bukan waktu resmi (§ 4.2) |
| **Keamanan fallback** | Belum relevan (Fase 4). Yang disiapkan: `employees.attendance_mode='manual'` sebagai jalur sah bagi penolak consent, yang di Fase 4 **selalu** `pending_review` |
| **Anti-spoofing / liveness** | Tidak diselesaikan di sini dan dikatakan terus terang (E23). `capture_source` disimpan sebagai bahan audit; penegakan ada di Fase 6/7 |
