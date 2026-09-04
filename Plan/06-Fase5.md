# Fase 5 — Admin Panel & Konfigurasi (`faceclock-web`)

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 5**. Ukuran: 🔴 **Besar**.
> Mengikuti Protokol Handoff § 10.2: dokumen ini di-review sebelum eksekusi.
>
> **Depends on:** [Fase 0](01-Fase0.md) (scaffolding React), [Fase 1](02-Fase1.md),
> [Fase 3](04-Fase3.md), [Fase 4](05-Fase4.md).
> **Blocker untuk:** [Fase 6](07-Fase6.md) — Fase 5 membangun **app shell** (auth,
> permission, layout, error handling) yang dipakai bersama.
>
> **Status:** DRAFT — 4 keputusan butuh konfirmasi user (§ 2.0), dan **2 endpoint
> backend baru** yang harus disepakati (§ 2.1).

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Sampai Fase 4, seluruh sistem hanya bisa dioperasikan lewat `curl`. Itu cukup untuk
membuktikan kebenarannya, tidak cukup untuk dipakai. Ada dua hal yang **hanya** bisa
diselesaikan di fase ini:

1. **Antrian `pending_review` butuh manusia.** Fase 4 sengaja mengarahkan setiap
   kegagalan verifikasi ke sana. Tanpa antarmuka yang membuat peninjauan cepat dan
   informatif, antrian itu akan menumpuk dan berakhir di-*approve* massal tanpa
   dilihat — yang berarti seluruh rantai verifikasi wajah menjadi teater.
2. **Semua angka yang menentukan perilaku sistem ada di `app_settings`.** Threshold,
   radius geofence, kebijakan di luar geofence, `workday_cutoff_hour`. Selama hanya
   bisa diubah lewat SQL, "configurable" hanya benar di atas kertas.

Fase 5 juga membangun **app shell**: autentikasi, refresh token, gating permission,
layout, dan pemetaan error. Fase 6 menumpang di atasnya. Karena itu urutannya
Fase 5 lebih dulu, meski secara fitur Fase 6 terasa lebih mendasar.

### Hasil akhir yang diharapkan
- Admin/HR bisa login dan melihat hanya menu yang benar-benar boleh ia akses.
- Antrian approval menampilkan bukti yang cukup untuk memutuskan: foto, lokasi,
  `matched_similarity` vs `threshold_used`, `hints`, dan riwayat karyawan.
- Manajemen karyawan, user, dan role berfungsi penuh.
- Konfigurasi bisa diubah dari UI: threshold, lokasi kantor dengan map picker,
  kebijakan geofence (D18), kewajiban wajah saat check-out (D19), `workday_cutoff_hour`.
- Status consent dan enrollment tiap karyawan terlihat, termasuk siapa yang berada
  di jalur `manual`.
- Job reindex bisa dipicu dan dipantau.
- Rekap absensi bisa difilter dan diekspor.
- Super Admin bisa mengelola role & permission.

### Yang TIDAK dikerjakan di fase ini
- Halaman untuk **karyawan** (check-in, enrollment, riwayat pribadi) → Fase 6.
  Fase 5 hanya menyediakan placeholder jujur bagi user yang tidak punya permission admin.
- Perhitungan keterlambatan/lembur/cuti. Rekap Fase 5 menyajikan **fakta yang
  tersimpan** (jam masuk, jam pulang, durasi, status), bukan penafsiran kebijakan HR.
  `attendance.workday_start/end` dipakai hanya sebagai **penanda visual**, bukan
  sebagai perhitungan yang tersimpan.
- Mobile (→ Fase 7).
- Notifikasi push/email.

---

## 2. Scope Detail

### 2.0 Keputusan yang butuh konfirmasi

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D20 | Format export rekap | **CSV** di-*stream* dari server (UTF-8 + BOM). XLSX ditunda | ⚠️ **BUTUH KONFIRMASI** |
| D21 | Pengelolaan dokumen consent lewat UI | **Read-only di UI**; versi baru tetap lewat migration | ⚠️ **BUTUH KONFIRMASI** |
| D22 | Library peta untuk map picker | **Leaflet + tile OpenStreetMap** | ⚠️ **BUTUH KONFIRMASI** |
| D23 | Penyimpanan token di browser | Access token **di memori**, refresh token di `localStorage` | ⚠️ **BUTUH KONFIRMASI** |

---

### 2.1 Fase 5 bukan murni frontend — 2 endpoint backend baru

Ini perlu dinyatakan di depan supaya tidak menjadi kejutan saat estimasi.

Master plan § Fase 5 menuntut *"Laporan & rekap absensi (filter + export CSV/Excel)"*.
[Fase 4 § 1](05-Fase4.md#1-tujuan-fase) secara eksplisit menunda rekap dan export ke
fase ini, dan [Fase 1 § 2.3](02-Fase1.md#23-katalog-permission) sudah men-*seed*
permission `attendance.export` yang sampai sekarang belum dipakai endpoint mana pun.

**Endpoint baru:**

| # | Method | Path | Guard |
|---|---|---|---|
| 71 | GET | `/api/v1/attendances/summary` | `attendance.read_all` |
| 72 | GET | `/api/v1/attendances/export` | `attendance.export` |
| 73 | GET | `/api/v1/settings/face-quality-status` | `settings.read` |

**Perubahan kecil pada endpoint yang sudah ada** (dilaporkan sebagai revisi, § 12):

| Endpoint | Perubahan |
|---|---|
| #7 `GET /employees` | Tambah filter `attendance_mode`, `consent_status`, `enrollment_status`; tambah field yang sama di item response |
| #58 `GET /attendances/{id}` | Perjelas: mengembalikan **DTO admin** bila pemanggil punya `attendance.read_all`, **DTO employee** bila hanya `attendance.read_self` |

**Yang sengaja TIDAK ditambahkan:** endpoint `dashboard/stats`. Semua angka kartu
dashboard bisa diambil dari `meta.total` endpoint daftar yang sudah ada
(`/attendances?work_date=...`, `/attendances/pending`, `/employees?enrollment_status=none`).
Menambah endpoint agregat berarti satu tempat lagi yang bisa tidak sinkron dengan
daftar yang ditampilkan di sebelahnya.

#### `GET /attendances/summary` (#71)

Rekap per karyawan untuk satu rentang tanggal.

Query: `from`, `to` (wajib, maks 366 hari), `employee_id`, `department`, `status`,
`page`, `per_page`.

```json
{
  "data": [
    {
      "employee": { "id": "...", "employee_number": "EMP-0002",
                    "full_name": "Budi Santoso", "department": "IT" },
      "work_days": 21,
      "check_in_count": 21,
      "check_out_count": 20,
      "approved_count": 39,
      "pending_review_count": 2,
      "rejected_count": 0,
      "fallback_count": 2,
      "missing_check_out_count": 1,
      "total_work_minutes": 10230,
      "avg_work_minutes": 511
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 137, "total_pages": 7,
            "from": "2026-09-01", "to": "2026-09-30" }
}
```

`missing_check_out_count` adalah `work_date` yang punya `check_in` non-`rejected`
tanpa `check_out` non-`rejected` — fakta yang tersimpan, bukan penilaian.

#### `GET /attendances/export` (#72)

Query: sama persis dengan #60 `GET /attendances`, ditambah `format` (`csv`) dan
`scope` (`detail` | `summary`).

- Response: `text/csv; charset=utf-8`, diawali **BOM** (`﻿`) supaya Excel di
  Windows membaca UTF-8 dengan benar tanpa diminta.
- Header: `Content-Disposition: attachment; filename="faceclock-absensi-2026-09-01_2026-09-30.csv"`.
- Di-*stream* baris demi baris (`csv.Writer` + `http.Flusher`), bukan disusun di
  memori. Rekap setahun untuk 500 karyawan adalah ~250.000 baris; menyusunnya di
  memori lebih dulu adalah cara paling mudah mematikan container API.
- Batas keras `attendance.export_max_rows` (setting baru, default 100.000). Melebihi
  → `422 VALIDATION_ERROR` dengan pesan yang menyebut jumlah baris dan menyarankan
  mempersempit rentang. **Bukan** memotong diam-diam — export yang terpotong tanpa
  pemberitahuan adalah laporan yang salah.
- Setiap export menulis `audit_logs` `attendance.exported` `{from, to, filters, row_count}`.
  Export adalah pemindahan data kehadiran ke luar sistem; ia pantas dicatat.

Kolom CSV `scope=detail`: `work_date, employee_number, full_name, department, type,
server_timestamp_local, status, method, fallback_reason, geofence_status,
office_location, distance_meter, matched_similarity, threshold_used, model_version,
capture_source, note, reviewed_by, reviewed_at, review_note`.

> `matched_similarity` dan `threshold_used` **ada** di CSV karena endpoint ini dijaga
> `attendance.export`, yang di [Fase 1 § 2.4](02-Fase1.md#24-role-default) hanya
> dimiliki `admin` dan `super_admin`. Ia tidak pernah sampai ke karyawan.

#### D20 — kenapa CSV, bukan XLSX ⚠️ BUTUH KONFIRMASI

XLSX butuh library penulis di server (menyusun ZIP + XML, sulit di-*stream*) atau
di client (SheetJS ~1 MB, dan seluruh dataset harus masuk memori browser). CSV
bisa di-*stream*, bisa dibuka Excel, Google Sheets, dan LibreOffice, dan tidak
menambah dependensi apa pun.

Kalau XLSX benar-benar diminta (mis. karena butuh beberapa sheet atau format
angka), jalur yang direkomendasikan adalah menambahkannya **nanti** sebagai
`format=xlsx` pada endpoint yang sama, bukan mengganti CSV.

#### `GET /settings/face-quality-status` (#73)

**Endpoint baru ketiga**, lahir dari resolusi
[K-02](09-Revisions-Log.md#k-02--salinan-ambang-kualitas-tidak-punya-tempat-di-ui--celah--resolved),
bukan dari master plan langsung. [Fase 2 § 3.1](03-Fase2.md#31-migration-000011_face_settings)
menjanjikan tujuh ambang kualitas `face.*` "ditampilkan di panel admin" dan
menjanjikan WARNING saat konfigurasi inference *drift* dari `app_settings` — tapi
tidak ada endpoint yang mengeksposnya. `#28 GET /settings` **tidak cukup**: ia
mengembalikan nilai `app_settings` apa adanya, tanpa pernah membandingkannya
dengan nilai yang **sungguh aktif** di container inference.

```json
// 200
{
  "data": {
    "checked_at": "2026-09-04T01:00:00Z",
    "in_sync": false,
    "thresholds": [
      { "key": "face.min_det_score",  "app_settings_value": 0.60,  "inference_active_value": 0.60,  "in_sync": true  },
      { "key": "face.min_blur_var",   "app_settings_value": 40.0,  "inference_active_value": 35.0,  "in_sync": false },
      { "key": "face.min_brightness", "app_settings_value": 55.0,  "inference_active_value": 55.0,  "in_sync": true  },
      { "key": "face.max_brightness", "app_settings_value": 215.0, "inference_active_value": 215.0, "in_sync": true  },
      { "key": "face.min_face_ratio", "app_settings_value": 0.18,  "inference_active_value": 0.18,  "in_sync": true  },
      { "key": "face.max_abs_yaw",    "app_settings_value": 0.35,  "inference_active_value": 0.35,  "in_sync": true  },
      { "key": "face.max_abs_pitch",  "app_settings_value": 0.30,  "inference_active_value": 0.30,  "in_sync": true  }
    ]
  }
}
```

- Handler memanggil `GET /ready` inference **langsung saat request** (sama
  seperti pemeriksaan startup di [Fase 2 § 5.1](03-Fase2.md#51-startup) —
  budget timeout 2 detik, pola yang sama dengan `/readyz` [Fase 0 § 4](01-Fase0.md#4-daftar-endpoint)),
  membandingkan `quality_thresholds` terhadap ketujuh key `app_settings` di atas.
  **Live**, bukan snapshot startup — supaya drift yang terjadi setelah restart
  container inference (tanpa restart `faceclock-api`) tetap terlihat.
- Inference tidak terjangkau dalam 2 detik ⇒ `200` tetap dikembalikan dengan
  `checked_at: null`, `in_sync: null`, dan tiap baris `inference_active_value: null,
  in_sync: null` — **bukan** `5xx`. Ini halaman diagnostik read-only; kegagalannya
  sendiri adalah salah satu hal yang ingin ditampilkan, bukan alasan mematikan
  halaman `/settings`.
- Tidak menyimpan riwayat — setiap panggilan adalah pemeriksaan baru. Tanpa
  tabel baru, tanpa migration.

---

### 2.2 App shell — fondasi bersama Fase 5 & 6

Empat hal ini dibangun di Fase 5 dan dipakai apa adanya oleh Fase 6.

#### a. Klien API di atas kontrak Fase 0

[Fase 0 § 2.7](01-Fase0.md#27-scaffolding-faceclock-web-react) sudah menetapkan
`src/lib/api.ts` yang meng-*unwrap* `{data}` dan melempar `ApiError` dari `{error}`.
Fase 5 menyelesaikannya:

```ts
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: ApiErrorCode,          // union dari 39 code (§ 2.3)
    readonly message: string,
    readonly details?: { field: string; message: string }[],
    readonly hints?: HintCode[],          // Fase 2 § 2.5
    readonly requestId?: string,
    readonly extra?: Record<string, unknown>,  // mis. can_fallback, existing_session_id
  ) { super(message); }
}
```

`requestId` **selalu** ditampilkan pada error 5xx — [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go)
menjamin ia ada di setiap error, dan itu satu-satunya cara pengguna bisa melaporkan
masalah yang bisa ditelusuri.

#### b. Refresh token dengan single-flight lock — **wajib, bukan optimasi**

Ini bagian paling mudah salah di seluruh Fase 5, dan akibatnya paling merusak.

[Fase 1 § 2.5](02-Fase1.md#25-strategi-token) menetapkan refresh token dengan
**rotasi + deteksi reuse**: memakai ulang token yang sudah dirotasi akan
**mencabut seluruh family** dan menaikkan `token_version`. [Fase 1 § 6.1 E9](02-Fase1.md#61-autentikasi)
menyatakan dua refresh bersamaan menghasilkan satu sukses dan satu `401` reuse.

Di SPA, ini bukan kasus tepi — ini kasus **normal**. Dashboard memuat 5 query
sekaligus; access token kedaluwarsa; kelimanya menerima `401`; kelimanya memanggil
`/auth/refresh` dengan token yang sama. Satu berhasil, empat memicu deteksi reuse,
dan **seluruh sesi dicabut**. Pengguna terlempar ke login setiap kali membuka tab
setelah 15 menit menganggur.

**Solusi yang wajib diimplementasikan — tiga lapis:**

1. **Single-flight dalam satu tab.** Satu `Promise<void>` modul-level. Request yang
   menerima `401` menunggu promise yang sedang berjalan alih-alih memulai refresh baru.

2. **Lock lintas-tab.** Dua tab punya modul masing-masing, jadi lapis 1 tidak cukup.
   Pakai **Web Locks API**:
   ```ts
   await navigator.locks.request('faceclock-refresh', async () => {
     if (tokenIsStillFresh()) return;   // tab lain sudah merefresh duluan
     await doRefresh();
   });
   ```
   Pemeriksaan ulang di dalam lock itu yang menyelesaikannya: tab kedua masuk lock,
   melihat token sudah baru, dan tidak melakukan apa-apa.

3. **Penyebaran hasil.** `BroadcastChannel('faceclock-auth')` mengirim
   `{type:'tokens-rotated'}` (**tanpa** nilai tokennya) ke tab lain, yang lalu
   membaca refresh token terbaru dari `localStorage` dan mereset access token
   di memori. Pesan yang membawa token akan tersimpan di antrean channel — tidak perlu.

**Fallback** untuk browser tanpa Web Locks: lock berbasis `localStorage` dengan
timestamp dan TTL 5 detik. Kurang rapi, tapi mencegah kasus terburuk.

**Perilaku saat refresh benar-benar gagal** (`REFRESH_TOKEN_REUSED` atau
`INVALID_REFRESH_TOKEN`): hapus token, batalkan semua query berjalan, arahkan ke
`/login?reason=session_expired`, dan tampilkan pesan yang jujur — *"Sesi berakhir.
Silakan login kembali."* Bukan *"Terjadi kesalahan"*.

#### c. D23 — penyimpanan token ⚠️ BUTUH KONFIRMASI

| Opsi | Kelebihan | Kekurangan |
|---|---|---|
| Keduanya di `localStorage` | Paling sederhana | Access token ikut terekspos XSS selama 15 menit penuh |
| **Access di memori + refresh di `localStorage`** (rekomendasi) | Access token hilang saat tab ditutup; hanya refresh token yang bertahan | Refresh token tetap terekspos XSS |
| Keduanya di memori | Paling aman terhadap XSS | Logout setiap kali *reload* — tidak bisa dipakai |
| Cookie `httpOnly` | Kebal XSS | Backend Fase 1 mengembalikan token di **body JSON**, bukan `Set-Cookie`. Memilih ini berarti mengubah kontrak Fase 1 dan menambah perlindungan CSRF |

**Rekomendasi: opsi 2**, dengan catatan jujur bahwa ia **tidak** menghilangkan
risiko XSS — ia hanya mempersempitnya. Yang benar-benar melindungi adalah tidak
punya XSS: React yang meng-*escape* secara default, larangan
`dangerouslySetInnerHTML` (kecuali satu tempat terkendali di § 2.6), dan CSP ketat
di server yang menyajikan `faceclock-web`.

Kalau tim menganggap risiko XSS tidak dapat diterima, opsi 4 adalah jawabannya —
tapi itu **perubahan kontrak Fase 1** yang harus diputuskan sekarang, bukan setelah
Fase 6 selesai.

#### d. Gating permission di UI — lapisan kenyamanan, bukan keamanan

`GET /auth/me` (#5) mengembalikan `permissions`.
[Fase 1 § 4.1](02-Fase1.md#41-auth) sudah menyatakan tegas: field itu **hanya untuk
kebutuhan UI**, dan server tetap memeriksa ulang setiap request.

```tsx
const { can } = usePermissions();

<Can permission="attendance.approve">
  <ApproveButton />
</Can>

<Can permission={["employee.read", "employee.read_self"]} mode="any">…</Can>
```

Aturan yang ditegakkan lewat lint rule kustom + review:
- **Tidak boleh ada perbandingan nama role di seluruh `src/`.** ESLint rule
  `no-restricted-syntax` yang menolak literal `'super_admin'`, `'admin'`, `'employee'`
  di luar `src/lib/permissions.ts`. Alasannya sama dengan
  [Fase 1 § 2.2](02-Fase1.md#22-model-otorisasi-permission-based-bukan-role-based):
  himpunan role berubah saat runtime, jadi kode yang memeriksa nama role akan salah
  begitu super admin membuat role baru.
- Route dijaga `<RequirePermission>`; user tanpa izin diarahkan ke `/403`, **bukan**
  layar kosong.
- Menu sidebar dibangun dari daftar deklaratif `{path, label, icon, permission}`,
  disaring `can()`. Menambah halaman tanpa menyebut permission-nya akan membuat
  test `every nav item declares a permission` gagal.

**Yang tidak boleh dilakukan:** menyembunyikan tombol lalu menganggap aksinya aman.
Setiap aksi tetap bisa gagal `403` dari server, dan UI harus menanganinya dengan
pesan yang benar (§ 6).

---

### 2.3 Pemetaan error — dari `code`, bukan pesan buatan sendiri

Katalog lengkap: **13** dari [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go),
**12** dari [Fase 3 § 2.9](04-Fase3.md#29-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0),
**14** dari [Fase 4 § 2.10](05-Fase4.md#210-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0)
= **39 code**.

`src/lib/errors/messages.ts` memetakan setiap `code` ke teks Indonesia, dan
**tidak ada tempat lain** di aplikasi yang menulis pesan error.

```ts
export const ERROR_MESSAGES: Record<ApiErrorCode, ErrorPresentation> = {
  UNAUTHENTICATED:   { title: "Sesi berakhir", body: "Silakan login kembali.", severity: "warning", action: "relogin" },
  FORBIDDEN:         { title: "Tidak diizinkan", body: "Anda tidak punya izin untuk tindakan ini.", severity: "warning" },
  NOT_FOUND:         { title: "Tidak ditemukan", body: "Data yang Anda cari tidak ada atau bukan milik Anda.", severity: "info" },
  CONFLICT:          { title: "Bentrok", body: "Data sudah berubah. Muat ulang lalu coba lagi.", severity: "warning", action: "refetch" },
  LAST_SUPER_ADMIN:  { title: "Tidak bisa dilakukan", body: "Harus tetap ada minimal satu Super Admin aktif.", severity: "warning" },
  SELF_REVIEW_DENIED:{ title: "Tidak bisa meninjau sendiri", body: "Absensi milik Anda harus ditinjau admin lain.", severity: "warning" },
  ATTENDANCE_ALREADY_REVIEWED: { title: "Sudah ditinjau", body: "Absensi ini sudah diproses admin lain.", severity: "info", action: "refetch" },
  UPSTREAM_TIMEOUT:  { title: "Layanan wajah tidak merespons", body: "Coba lagi beberapa saat lagi.", severity: "error", showRequestId: true },
  INTERNAL_ERROR:    { title: "Terjadi kesalahan", body: "Laporkan kode berikut ke admin sistem.", severity: "error", showRequestId: true },
  // … 39 seluruhnya
};
```

Tiga aturan:
1. **`NOT_FOUND` bukan error sistem.** [Fase 1 § 5.4](02-Fase1.md#54-pola-akses-self)
   menetapkan bahwa mengakses resource milik orang lain dibalas **404, bukan 403** —
   itu disengaja, supaya id yang ditebak tidak terkonfirmasi. Konsekuensinya di UI:
   `404` dirender sebagai **empty state yang tenang** (*"Tidak ditemukan"*), bukan
   sebagai toast merah bertuliskan "Terjadi kesalahan". Halaman detail dengan id
   yang salah harus terasa seperti halaman kosong, bukan seperti aplikasi rusak.
2. **`CONFLICT` dan `ATTENDANCE_ALREADY_REVIEWED` memicu `refetch` otomatis**, lalu
   menampilkan keadaan terbaru. Keduanya hampir selalu berarti "admin lain sudah
   melakukannya", dan yang dibutuhkan pengguna adalah melihat hasilnya.
3. **`showRequestId: true`** menampilkan `request_id` dengan tombol salin.

Type-safety: `ApiErrorCode` adalah union literal; `Record<ApiErrorCode, …>` membuat
`tsc` gagal bila ada code yang belum dipetakan. Menambah error code di backend tanpa
menambahkannya di sini akan **memecah build**, bukan diam-diam menampilkan
`undefined`.

---

### 2.4 Komponen `<AuthImage>` — konsekuensi langsung dari D14

[Fase 4 D14](04-Fase3.md#22-d14--akses-foto-streaming-lewat-api-bukan-signed-url)
menetapkan foto diakses lewat API dengan cek permission **per request**, bukan
signed URL. Konsekuensinya di browser tidak sepele: `<img src="/api/...">` **tidak
bisa membawa header `Authorization`**.

Pola yang dipakai di seluruh aplikasi:

```
1. fetch(path, { headers: { Authorization: `Bearer ${accessToken}` } })
2. res.blob()                      → di-cache TanStack Query (gcTime 5 menit)
3. URL.createObjectURL(blob)       → dibuat di useEffect, per mount
4. <img src={objectUrl} />
5. URL.revokeObjectURL(objectUrl)  → cleanup useEffect
```

Yang di-cache adalah **`Blob`**, bukan object URL. Object URL punya siklus hidup
per-mount; kalau ia yang di-cache, URL yang sudah di-*revoke* akan dipakai ulang
oleh komponen berikutnya dan gambarnya kosong tanpa error apa pun.

**Batas memori — dan konsekuensi desainnya.** Setiap blob foto ~200–300 KB dan
object URL menahan memori sampai di-*revoke*. Daftar approval berisi 200 baris ×
1 foto = 40–60 MB yang tidak bisa dilepas browser.

Karena itu: **daftar tidak menampilkan foto.** Baris menampilkan ikon/placeholder;
foto dimuat di **panel detail** yang terbuka satu per satu. Ini bukan kompromi —
untuk memutuskan approve/reject, reviewer memang perlu melihat satu foto besar,
bukan 200 thumbnail 40×40 yang tidak bisa dinilai.

Penanganan status yang wajib ada:

| Status | Tampilan |
|---|---|
| Loading | Skeleton dengan rasio aspek tetap (mencegah pergeseran layout) |
| `401` | Refresh token lalu ulangi sekali; gagal → alur sesi berakhir |
| `403` | *"Anda tidak punya izin melihat foto ini."* |
| `404` | *"Foto tidak ditemukan."* — tenang, bukan merah |
| `410` | *"Foto sudah dihapus sesuai kebijakan retensi."* — [Fase 3 § 4.4](04-Fase3.md#44-manajemen-referensi) & [Fase 4 § 6.5 E38](05-Fase4.md#65-data--operasional) |
| Gagal jaringan | Tombol "Coba lagi" |

---

### 2.5 Antrian approval — pekerjaan inti fase ini

Ini layar yang paling menentukan apakah rantai verifikasi Fase 4 bermakna. Dua
prinsip desainnya:

**a. Reviewer harus melihat bukti, bukan hanya tombol.** Panel detail menampilkan:

| Bagian | Isi | Sumber |
|---|---|---|
| Foto absensi | besar, `<AuthImage>` | #59 |
| Foto referensi | 3 foto referensi aktif karyawan, berdampingan | #44 + #45 |
| Skor | `matched_similarity` vs `threshold_used`, sebagai bar dengan penanda ambang | #58 (DTO admin) |
| Alasan | `fallback_reason` diterjemahkan; `hints` diterjemahkan | #58 |
| Waktu | `server_timestamp` dalam `attendance.timezone`; `clock_skew_seconds` bila > 300 dtk | #58 |
| Lokasi | peta kecil: titik absensi + lingkaran radius kantor; `distance_meter`, `gps_accuracy_meter`, `geofence_status` | #58 + #66 |
| Konteks karyawan | 10 absensi terakhir + berapa kali fallback 30 hari terakhir | #60 (filter) |
| Telemetri | percobaan gagal karyawan ini hari itu | #65 |

Menyandingkan foto absensi dengan foto referensi adalah inti pekerjaan reviewer:
ia sedang menjawab "apakah ini orang yang sama", dan tanpa foto referensi ia tidak
punya dasar apa pun untuk menjawabnya.

Bar skor menampilkan `matched_similarity` relatif terhadap `threshold_used` — bukan
angka telanjang. `0.41` tidak berarti apa-apa; `0.41 dari ambang 0.42` berarti
"nyaris lolos", dan `0.11 dari ambang 0.42` berarti "kemungkinan besar orang lain".

**b. Aturan B11 ditegakkan di UI, bukan hanya di backend.**
[Fase 4 B11](05-Fase4.md#24-aturan-bisnis) melarang admin meninjau absensinya
sendiri, dan backend membalas `403 SELF_REVIEW_DENIED`. UI **tidak menampilkan
tombol approve/reject** pada baris yang `attendance.employee.id === me.employee.id`;
sebagai gantinya muncul label *"Absensi Anda — harus ditinjau admin lain."*

Ini defense in depth: menampilkan tombol yang pasti gagal adalah cara membuat
pengguna belajar mengabaikan pesan error.

**c. Yang sengaja TIDAK ada:** tombol *"Setujui semua"*. Bulk review (#64) tersedia,
tapi UI hanya mengizinkan bulk atas **baris yang dipilih manual** (maksimum 50,
sesuai batas #64), dan dialog konfirmasi menampilkan jumlah serta daftar nama.
Satu tombol yang menyetujui seluruh antrian tanpa dilihat akan menghapus nilai
antriannya.

Bulk reject **wajib** disertai alasan (satu alasan untuk semua yang dipilih), sesuai
kontrak #63.

Hasil bulk ditampilkan per item — #64 memang mengembalikan `results[]` dengan
kegagalan per item, dan menyembunyikannya berarti admin mengira 50 berhasil padahal 3 gagal.

---

### 2.6 Konfigurasi — menampilkan konsekuensi, bukan hanya field

Halaman `/settings` mengelompokkan `app_settings` (#28/#30) menjadi kartu:

| Kartu | Key |
|---|---|
| Verifikasi wajah | `face.similarity_threshold`, `face.model_version` (read-only), `face.min_reference_photos`, `face.max_reference_photos`, `face.min_quality_score`, `face.duplicate_check_enabled`, `face.duplicate_threshold`, `face.consent_required` |
| Lokasi & geofence | `attendance.geofence_enabled`, `attendance.max_distance_meter`, `attendance.outside_geofence_policy` **(D18)**, `attendance.missing_location_policy`, `attendance.max_gps_accuracy_meter` |
| Aturan absensi | `attendance.timezone`, `attendance.workday_cutoff_hour`, `attendance.workday_start/end`, `attendance.allow_checkout_without_checkin`, `attendance.min_minutes_between_checkin_checkout`, `attendance.require_face_for_checkout` **(D19)**, `attendance.checkout_without_face_status`, `attendance.fallback_enabled`, `attendance.allow_fallback_without_enrollment`, `attendance.max_note_length` |
| Keamanan & retensi | `attendance.max_failed_attempts_per_hour`, `attendance.duplicate_photo_window_days`, `attendance.photo_retention_days`, `attendance.attempt_retention_days`, `face.retention_days_after_resign`, `security.*` |
| **Ambang kualitas wajah** *(read-only)* | `face.min_det_score`, `face.min_blur_var`, `face.min_brightness`, `face.max_brightness`, `face.min_face_ratio`, `face.max_abs_yaw`, `face.max_abs_pitch` — sumber **`#73`**, bukan `#28` |

#### Kartu "Ambang kualitas wajah" — resolusi K-02

[K-02](09-Revisions-Log.md#k-02--salinan-ambang-kualitas-tidak-punya-tempat-di-ui--celah--resolved):
tujuh ambang ini adalah **salinan tampilan** ([Fase 2 § 3.1](03-Fase2.md#31-migration-000011_face_settings)) —
sumber kebenaran tetap env container inference, jadi kartu ini **selalu
read-only**, tidak seperti empat kartu di atasnya. Diberi tempat sendiri, bukan
dilebur ke kartu "Verifikasi wajah", karena ia satu-satunya kartu yang bisa
menampilkan **peringatan drift**.

- Data diambil dari `#73 GET /settings/face-quality-status` (§ 2.1), dipanggil
  ulang setiap halaman dibuka + tombol "Periksa ulang" manual (tidak di-poll
  otomatis — ini bukan data yang berubah tanpa admin mengubahnya).
- Tiap baris menampilkan nilai `app_settings` **dan** nilai yang sungguh aktif
  di inference berdampingan. Baris dengan `in_sync: false` diberi **badge kuning
  "Tidak sinkron"** + teks: *"Nilai yang tersimpan berbeda dari yang benar-benar
  berlaku. Nilai yang aktif adalah **X**, bukan yang ditampilkan di sini. Hubungi
  tim teknis — kemungkinan container inference perlu diperbarui atau di-restart."*
  Ini persis WARNING yang dijanjikan [Fase 2 § 5.1](03-Fase2.md#51-startup),
  sekarang punya permukaan UI, bukan cuma baris log yang tidak pernah dibaca admin.
- `checked_at: null` (inference tak terjangkau saat dipanggil) ⇒ badge abu-abu
  netral *"Tidak bisa diperiksa saat ini"* — **bukan** merah; ini keadaan
  sementara, bukan kesalahan konfigurasi.
- Tanpa tombol edit di kartu ini sama sekali. Mengubah ambang ini hanya lewat
  env var container inference + redeploy — di luar cakupan panel admin, sesuai
  batas yang sudah ditetapkan Fase 2.

**Setiap setting yang mengubah perilaku keamanan menampilkan konsekuensinya sebelum
disimpan**, lewat dialog konfirmasi yang menyebut akibatnya dengan kalimat biasa:

| Perubahan | Peringatan |
|---|---|
| `face.similarity_threshold` **diturunkan** | *"Menurunkan ambang membuat sistem lebih mudah menerima wajah yang mirip. Angka saat ini berasal dari kalibrasi (lihat laporan). Ubah hanya bila Anda punya data pengukuran baru."* + tautan ke `docs/face/fase2-calibration-report.md` |
| `attendance.outside_geofence_policy` → `pending_review` | *"Absensi dari luar radius kantor akan tetap tercatat dan masuk antrian persetujuan. Antrian akan bertambah banyak."* |
| `attendance.require_face_for_checkout` → `false` | *"Check-out tidak lagi diverifikasi wajah. Siapa pun yang memegang akun bisa mencatat jam pulang."* |
| `attendance.allow_fallback_without_enrollment` → `true` | *"Karyawan yang belum mendaftarkan wajah bisa absen lewat jalur persetujuan manual."* |
| `attendance.geofence_enabled` → `false` | *"Lokasi tidak lagi diperiksa. Absensi bisa dilakukan dari mana saja."* |
| `attendance.timezone` diubah | *"Absensi lama tetap memakai tanggal kerja yang sudah tersimpan. Perubahan hanya berlaku untuk absensi baru."* ([Fase 4 E7](05-Fase4.md#61-waktu--aturan-harian)) |

Peringatan ini bukan hiasan. Setiap satu di antaranya adalah cara nyata untuk
melemahkan sistem tanpa sadar, dan semuanya tercatat di `audit_logs` — tapi audit
log dibaca setelah kejadian, sedangkan dialog dibaca sebelum.

Validasi client mengikuti validator per-key backend
([Fase 1 § 4.6](02-Fase1.md#46-app-settings), [Fase 4 § 3.5](05-Fase4.md#35-app_settings-tambahan--migration-000021)),
tapi validasi server tetap yang menentukan — client hanya mempercepat umpan balik.

`face.model_version` ditampilkan **read-only** dengan penjelasan bahwa ia hanya
boleh berubah lewat job reindex (§ 2.8).

---

### 2.7 Lokasi kantor & map picker (D22)

CRUD `/office-locations` (#66–#70) dengan peta:

- Klik peta menetapkan `lat`/`lng`; lingkaran radius digambar dan berubah mengikuti
  input `radius_meter`.
- Marker bisa digeser; koordinat juga bisa diketik manual (kadang tim fasilitas
  sudah punya angkanya).
- `radius_meter` divalidasi ≤ `attendance.max_distance_meter`
  ([Fase 4 § 2.1](05-Fase4.md#21-rekonsiliasi-setting-yang-sudah-ada)); pesan errornya
  menyebut batas yang berlaku, bukan sekadar "tidak valid".
- Menonaktifkan/menghapus lokasi aktif terakhir sementara geofence menyala akan
  dibalas `409` oleh backend; UI **mendahuluinya** dengan menonaktifkan tombol dan
  menampilkan alasannya — tapi tetap menangani `409` bila admin lain menghapus
  duluan.
- Perubahan koordinat menampilkan dialog: *"Menggeser lokasi kantor mengubah siapa
  yang dianggap berada di dalam radius. Perubahan ini dicatat."*

**D22 — kenapa Leaflet + OpenStreetMap ⚠️ BUTUH KONFIRMASI**

| Opsi | Catatan |
|---|---|
| **Leaflet + tile OSM** (rekomendasi) | Tanpa API key, tanpa penagihan, ~40 KB. Tunduk pada *tile usage policy* OSM — untuk panel admin dengan lalu lintas rendah ini wajar, tapi harus disebut di dokumen |
| Google Maps | Peta & pencarian alamat terbaik; butuh API key, kartu kredit, dan mengirim koordinat kantor ke Google |
| MapLibre + tile berbayar | Fleksibel, butuh penyedia tile |
| Tanpa peta (input angka saja) | Nol dependensi; sangat rawan salah ketik koordinat, dan salah koordinat = seluruh kantor gagal absen |

Bila lalu lintas tile menjadi masalah, jalur pindahnya mudah: Leaflet bisa menunjuk
ke penyedia tile lain hanya dengan mengganti URL.

---

### 2.8 Consent, enrollment, dan reindex

#### Halaman `/consents` — guard `face.read_any`

Daftar karyawan dengan status consent & enrollment, memakai #7 yang diperluas (§ 2.1).
Filter cepat:

| Filter | Untuk menjawab |
|---|---|
| `consent_status=none` | Siapa yang belum memberi consent sama sekali |
| `consent_status=outdated` | Siapa yang menyetujui versi dokumen lama ([Fase 3 E5](04-Fase3.md#61-consent)) |
| `consent_status=withdrawn` | Siapa yang mencabut |
| `attendance_mode=manual` | **Penolak consent** — absensinya selalu `pending_review` |
| `enrollment_status=none` \| `incomplete` | Siapa yang belum/kurang mendaftarkan wajah |

Baris `attendance_mode='manual'` diberi penanda dan penjelasan: *"Karyawan ini berada
di jalur absensi manual. Setiap absensinya masuk antrian persetujuan."* Tanpa
penjelasan itu, admin akan mengira antrian yang tidak pernah habis adalah bug.

Aksi: mencatat consent luring (#37) dan mengubah `attendance_mode` (#11, butuh
`employee.update`). Perubahan mode meminta konfirmasi dan alasan — ia memindahkan
seseorang keluar dari verifikasi wajah, dan itu keputusan HR, bukan klik biasa.

**D21 — dokumen consent read-only di UI ⚠️ BUTUH KONFIRMASI.**
[Fase 3 § 3.1](04-Fase3.md#31-consent_documents--migration-000012) menetapkan dokumen
consent tidak pernah diedit setelah dipublikasikan, dan `content_hash` yang tersimpan
di tiap persetujuan bergantung pada itu. Menyediakan editor di panel admin
mengundang publikasi tidak sengaja atas teks hukum yang belum ditinjau — dan
publikasi itu akan membuat seluruh karyawan berstatus `outdated`.

Rekomendasi: UI **menampilkan** dokumen aktif dan daftar siapa yang menyetujui versi
mana; versi baru diterbitkan lewat migration, dengan proses review seperti perubahan
kode lain.

#### Halaman `/employees/:id/face` — guard `face.read_any`

Referensi wajah karyawan (#44): foto (`<AuthImage>`), `quality_score`,
`capture_source`, `model_version`, status aktif, dan siapa yang meng-enroll.
Aksi menonaktifkan referensi (#46, guard `face.delete_any`) menampilkan peringatan
bila akan membuat jumlah aktif turun di bawah minimum — kontrak #46 mensyaratkan
`allow_below_minimum: true` untuk itu, dan UI meminta konfirmasi eksplisit dengan
kalimat *"Karyawan ini tidak akan bisa absen dengan wajah sampai mendaftar ulang."*

Penghapusan permanen (#47) berada di balik dialog terpisah yang meminta pengetikan
ulang nomor karyawan — ini operasi PDP yang tidak bisa dibatalkan
([Fase 3 § 2.8](04-Fase3.md#28-penghapusan-data-biometrik-uu-pdp)).

#### Halaman `/face/reindex` — guard `face.reindex`

[Fase 1 § 2.4](02-Fase1.md#24-role-default) tidak memberi `face.reindex` ke `admin`,
jadi menu ini biasanya hanya terlihat oleh Super Admin — dan itu memang disengaja.

- **Dry run wajib lebih dulu.** Tombol "Jalankan" baru aktif setelah dry run (#49
  dengan `dry_run:true`) dijalankan dan hasilnya ditampilkan: berapa referensi,
  berapa karyawan terdampak.
- Progres di-*poll* setiap 3 detik (#51) selama `status ∈ {pending, running}`,
  menampilkan `processed/total`, `succeeded`, `failed`, dan
  `employees_incomplete_count`.
- Setelah selesai: daftar `incomplete_employees` dengan aksi lanjutan yang jelas —
  karyawan ini **tidak bisa absen dengan wajah** sampai enroll ulang
  ([Fase 4 E14](05-Fase4.md#62-wajah--inference)).
- Peringatan sebelum menjalankan: *"Selama reindex berjalan, karyawan tidak bisa
  memulai pendaftaran wajah baru."* (kontrak `409 REINDEX_IN_PROGRESS`).
- Setelah job selesai, UI mengingatkan bahwa `face.similarity_threshold`
  **kemungkinan perlu dikalibrasi ulang** untuk model baru
  ([Fase 2 § 2.9](03-Fase2.md#29-kalibrasi-threshold--deliverable-utama-fase-ini)) —
  mengganti model tanpa mengganti threshold adalah cara paling halus untuk merusak
  akurasi tanpa satu pun error muncul.

---

### 2.9 Manajemen user, role & permission

Semuanya memakai #13–#27 apa adanya.

Yang perlu perhatian khusus di UI karena kontraknya punya aturan yang tidak jelas
dari nama endpoint:

| Kontrak | Perlakuan UI |
|---|---|
| #19 `PUT /users/{id}/roles` adalah **replace**, bukan tambah | UI memakai checkbox list yang mengirim seluruh himpunan. Tidak ada tombol "tambah role" yang menyesatkan |
| #26 `PUT /roles/{id}/permissions` juga **replace** | Idem, dikelompokkan per `resource` memakai `?group_by=resource` (#27) |
| `403 ROLE_ESCALATION_DENIED` ([Fase 1 § 5.5 A](02-Fase1.md#55-pencegahan-privilege-escalation)) | UI **menonaktifkan** permission/role yang tidak dimiliki pemberi, dengan tooltip *"Anda tidak bisa memberikan izin yang tidak Anda miliki."* Backend tetap penentu |
| `409 LAST_SUPER_ADMIN` | Tombol dinonaktifkan bila UI tahu ini super admin terakhir; `409` tetap ditangani |
| `403 SYSTEM_ROLE_IMMUTABLE` | Role `is_system` menampilkan gembok pada nama & tombol hapus; permission-nya tetap bisa diubah |
| #20 reset password mengembalikan `temporary_password` **sekali** | Ditampilkan di dialog dengan tombol salin dan peringatan *"Password ini tidak akan ditampilkan lagi."* **Tidak** disimpan di state global, tidak masuk log, dan hilang saat dialog ditutup |

---

### 2.10 Rekap & laporan

Halaman `/reports`:
- Filter: rentang tanggal (wajib), karyawan, departemen, status, tipe, metode,
  lokasi. **Seluruh filter tersimpan di URL `searchParams`** — laporan yang
  difilter bisa dibagikan lewat tautan dan tombol *back* berperilaku benar.
- Dua tampilan: **Rekap** (#71, per karyawan) dan **Detail** (#60, per record).
- Export (#72) mengunduh sesuai tampilan aktif dan filter yang sama persis. Tombol
  export dijaga `<Can permission="attendance.export">`.
- Penanda visual `attendance.workday_start/end` (mis. jam masuk setelah
  `workday_start` diberi warna) dengan keterangan *"Penanda visual; sistem tidak
  menghitung keterlambatan."* — supaya tidak ada yang mengira angka ini sudah
  memperhitungkan kebijakan HR.

Unduhan memakai pola yang sama dengan `<AuthImage>`: `fetch` dengan header →
`blob()` → `createObjectURL` → `<a download>` sintetis → `revokeObjectURL`. Nama
file diambil dari `Content-Disposition`, dengan cadangan yang dibentuk client.

---

## 3. Struktur Halaman, Routing & State

### 3.1 Peta route

```
/login                              publik
/403                                publik (terautentikasi)
/404                                publik

── Shell admin (butuh auth) ──────────────────────────────────────────
/                                   redirect cerdas (§ 3.2)
/dashboard                          attendance.read_all
/attendances                        attendance.read_all
/attendances/pending                attendance.approve
/attendances/:id                    attendance.read_all
/reports                            attendance.read_all
/employees                          employee.read
/employees/new                      employee.create
/employees/:id                      employee.read
/employees/:id/edit                 employee.update
/employees/:id/face                 face.read_any
/consents                           face.read_any
/users                              user.read
/users/:id                          user.read
/roles                              role.read
/roles/:id                          role.read
/permissions                        permission.read
/office-locations                   location.read
/settings                           settings.read
/face/reindex                       face.reindex
/security/attempts                  attendance.read_all
/audit-logs                         audit.read
/account                            auth  (ganti password, #6)

── Ruang karyawan (kerangka Fase 5, diisi Fase 6) ────────────────────
/me/*                               placeholder → Fase 6
```

### 3.2 Redirect cerdas di `/`

Fase 5 dan Fase 6 berbagi satu aplikasi. Setelah login, tujuan ditentukan
permission, bukan role:

```ts
if (can("attendance.read_all") || can("employee.read")) return "/dashboard";
if (can("attendance.checkin"))                          return "/me/attendance";
return "/account";     // user tanpa permission apa pun — masih bisa ganti password
```

Baris ketiga bukan hiasan: [Fase 1 E25](02-Fase1.md#62-otorisasi) menyatakan user
tanpa role adalah keadaan yang sah, bukan bug. Ia harus mendarat di halaman yang
menjelaskan keadaannya (*"Akun Anda belum diberi hak akses. Hubungi admin."*),
bukan di layar kosong atau loop redirect.

Selama Fase 5 (sebelum Fase 6 selesai), `/me/*` merender placeholder jujur:
*"Halaman absensi karyawan belum tersedia."* — bukan 404, karena route-nya memang
akan ada.

### 3.3 Manajemen state

| Jenis state | Alat | Catatan |
|---|---|---|
| Server state | **TanStack Query v5** | Satu-satunya sumber data server. Tidak ada penyalinan ke state lokal |
| Sesi & permission | React Context (`AuthProvider`) | Diisi dari #5 `GET /auth/me`; di-*refetch* saat window fokus |
| Filter & paginasi | **URL `searchParams`** | Dapat dibagikan, tombol back benar, dan *refetch* konsisten |
| Form | `react-hook-form` + `zod` | Skema `zod` mencerminkan validasi backend (§ 6.4) |
| UI sementara (dialog, toast) | State lokal + `<ToastProvider>` | |

Konvensi TanStack Query:

```ts
// Query key selalu menyertakan seluruh parameter yang memengaruhi hasil
queryKey: ["attendances", { page, perPage, from, to, status, employeeId }]

staleTime:  30_000          // daftar
staleTime:  5 * 60_000      // settings, permissions, roles
gcTime:     5 * 60_000
retry: (count, err) =>
  err instanceof ApiError && err.status >= 500 && count < 2   // JANGAN retry 4xx
```

**Jangan pernah me-*retry* 4xx.** Mengulang `409 ALREADY_CHECKED_IN` atau
`403 SELF_REVIEW_DENIED` tidak akan mengubah jawabannya, dan pada endpoint yang
membuat data ia berisiko menghasilkan duplikat.

Invalidasi setelah mutasi:

| Mutasi | Yang di-invalidasi |
|---|---|
| Approve/reject (#62/#63/#64) | `["attendances"]`, `["attendances","pending"]`, `["dashboard"]` |
| Ubah setting (#30) | `["settings"]` + query yang membaca setting itu (mis. `["attendances","context"]`) |
| Ubah role user (#19) | `["users"]`, dan **`["auth","me"]` bila targetnya diri sendiri** |
| CRUD lokasi (#67/#69/#70) | `["office-locations"]`, `["settings"]` |
| Nonaktifkan referensi (#46) | `["employees", id, "face-references"]`, `["employees"]` |

Baris ketiga penting: mengubah role diri sendiri mengubah permission, dan
[Fase 1 § 5.3](02-Fase1.md#53-middleware-authenticate--requirepermission) menyatakan
cache permission server di-invalidasi seketika. UI harus mengikuti, atau menu akan
berbohong sampai halaman dimuat ulang.

### 3.4 Layout & komponen bersama

```
<AppShell>
  <Sidebar>         nav deklaratif, disaring can()
  <Topbar>          nama user, role, tombol logout, indikator lingkungan
  <Outlet/>         halaman
  <ToastRegion/>
  <GlobalErrorBoundary/>
```

Komponen yang dipakai lintas halaman (dan lintas fase — Fase 6 memakainya juga):

| Komponen | Fungsi |
|---|---|
| `<AuthImage>` | Foto lewat API bertoken (§ 2.4) |
| `<Can>` / `usePermissions()` | Gating permission |
| `<DataTable>` | Tabel + sort (whitelist backend) + paginasi ter-URL |
| `<EmptyState>` | Kosong / tidak ditemukan / tidak berizin — tiga varian berbeda |
| `<ErrorState>` | Render `ApiError` lewat `ERROR_MESSAGES` |
| `<ConfirmDialog>` | Konfirmasi dengan tingkat bahaya + teks konsekuensi |
| `<SimilarityBar>` | `matched_similarity` relatif `threshold_used` |
| `<HintList>` | `hints` → kalimat Indonesia |
| `<LocalTime>` | Render UTC → `attendance.timezone`, dengan tooltip UTC |
| `<Money>`/`<Duration>` | Format menit → "8j 32m" |

`<LocalTime>` layak disebut: seluruh backend menyimpan `timestamptz` UTC
([Fase 0 § 3](01-Fase0.md#3-skema-database)), sementara pengguna berpikir dalam
`attendance.timezone`. Kalau konversi ini tersebar di puluhan tempat, satu di
antaranya akan memakai zona waktu browser dan menghasilkan tanggal kerja yang
berbeda dari yang dihitung server.

---

## 4. Endpoint yang Dikonsumsi

Semua sudah ada kecuali #71/#72/#73 (§ 2.1). Kolom **DTO** menyebut varian yang dipakai.

### 4.1 Auth & sesi — Fase 1

| # | Method | Path | Dipakai di | DTO / catatan |
|---|---|---|---|---|
| 1 | POST | `/auth/login` | `/login` | `{access_token, refresh_token, user{roles,permissions}}` |
| 2 | POST | `/auth/refresh` | interceptor | **single-flight** (§ 2.2b) |
| 3 | POST | `/auth/logout` | Topbar | body `{refresh_token}`; selalu 204 |
| 4 | POST | `/auth/logout-all` | `/account` | |
| 5 | GET | `/auth/me` | `AuthProvider` | sumber `permissions` untuk gating |
| 6 | POST | `/auth/change-password` | `/account` | mencabut semua sesi → login ulang |

### 4.2 Karyawan & user — Fase 1

| # | Method | Path | Dipakai di | Catatan |
|---|---|---|---|---|
| 7 | GET | `/employees` | `/employees`, `/consents`, filter laporan | **diperluas** `attendance_mode`, `consent_status`, `enrollment_status` (§ 2.1) |
| 8 | POST | `/employees` | `/employees/new` | |
| 10 | GET | `/employees/{id}` | `/employees/:id` | |
| 11 | PATCH | `/employees/{id}` | `/employees/:id/edit`, ubah `attendance_mode` | |
| 12 | DELETE | `/employees/{id}` | `/employees/:id` | tangani `409 EMPLOYEE_HAS_ACTIVE_USER` |
| 13–20 | — | `/users*` | `/users`, `/users/:id` | #19 & #20 lihat § 2.9 |

### 4.3 Role & permission — Fase 1

| # | Method | Path | Dipakai di |
|---|---|---|---|
| 21–26 | — | `/roles*` | `/roles`, `/roles/:id` |
| 27 | GET | `/permissions?group_by=resource` | editor permission role |

### 4.4 Setting & audit — Fase 1

| # | Method | Path | Dipakai di | Catatan |
|---|---|---|---|---|
| 28 | GET | `/settings` | `/settings`, banyak halaman | Sebagai admin → seluruh baris |
| 30 | PUT | `/settings/{key}` | `/settings` | Dialog konsekuensi (§ 2.6) |
| **73** | GET | `/settings/face-quality-status` | `/settings` kartu "Ambang kualitas wajah" | **baru** (K-02); dipanggil ulang manual, tidak di-poll |
| 31 | GET | `/audit-logs` | `/audit-logs`, panel detail | filter `resource_type`/`resource_id` |

### 4.5 Consent, wajah & reindex — Fase 3

| # | Method | Path | Dipakai di | Catatan |
|---|---|---|---|---|
| 32 | GET | `/consents/document` | `/consents` | ditampilkan read-only (D21) |
| 36 | GET | `/employees/{id}/consent` | `/employees/:id` | |
| 37 | POST | `/employees/{id}/consent` | `/consents` | consent luring; `granted_at` tetap waktu server |
| 44 | GET | `/employees/{id}/face-references` | `/employees/:id/face`, panel approval | `include_inactive` |
| 45 | GET | `/face/references/{id}/photo` | `<AuthImage>` | |
| 46 | PATCH | `/face/references/{id}` | `/employees/:id/face` | `allow_below_minimum` |
| 47 | DELETE | `/employees/{id}/face-data` | `/employees/:id/face` | dialog ketik ulang |
| 49–52 | — | `/face/reindex-jobs*` | `/face/reindex` | poll #51 tiap 3 dtk |

### 4.6 Absensi — Fase 4

| # | Method | Path | Dipakai di | DTO |
|---|---|---|---|---|
| 58 | GET | `/attendances/{id}` | panel detail approval | **DTO admin** — perlu klarifikasi kontrak (§ 2.1, § 12) |
| 59 | GET | `/attendances/{id}/photo` | `<AuthImage>` | tangani `410` |
| 60 | GET | `/attendances` | `/attendances`, `/reports` detail, konteks karyawan | **DTO admin** — memuat `matched_similarity`, `threshold_used`, `model_version`, `quality_score`, `distance_meter` |
| 61 | GET | `/attendances/pending` | `/attendances/pending` | DTO admin + `waiting_hours` |
| 62 | POST | `/attendances/{id}/approve` | panel detail | `review_note` opsional |
| 63 | POST | `/attendances/{id}/reject` | panel detail | `review_note` **wajib** 3–500 |
| 64 | POST | `/attendances/reviews` | bulk terpilih | maks 50; render `results[]` per item |
| 65 | GET | `/attendances/attempts` | `/security/attempts`, panel detail | |
| 66–70 | — | `/office-locations*` | `/office-locations`, peta panel detail | |
| **71** | GET | `/attendances/summary` | `/reports` rekap | **baru** |
| **72** | GET | `/attendances/export` | tombol export | **baru**, `attendance.export` |

> **Yang TIDAK dipakai Fase 5:** #56 `/attendances/me` dan #57 `/attendances/me/today`.
> Keduanya mengembalikan **DTO employee** yang sengaja tidak memuat
> `matched_similarity`/`threshold_used` ([Fase 4 § 2.7](05-Fase4.md#27-anti-penyalahgunaan)).
> Panel approval memakai #58/#60/#61 yang dijaga `attendance.read_all`/`attendance.approve`.
> Memakai #56 di panel admin akan membuat reviewer memutuskan tanpa melihat skor —
> tepat informasi yang ia butuhkan.

---

## 5. Flow per Fitur

### 5.1 Login → shell siap

```
/login  → POST #1 {email, password}
  200 ⇒ simpan access token DI MEMORI, refresh token di localStorage (D23)
        prefetch #5 GET /auth/me → AuthProvider terisi (roles, permissions)
        must_change_password = true ⇒ paksa ke /account, blokir navigasi lain
        redirect cerdas (§ 3.2)
  401 INVALID_CREDENTIALS ⇒ pesan sama untuk email salah / password salah / akun
        terkunci — UI TIDAK boleh membedakan, karena backend sengaja tidak
        membedakannya ([Fase 1 E1–E4](02-Fase1.md#61-autentikasi))
  429 RATE_LIMITED ⇒ tampilkan hitung mundur dari header Retry-After
```

### 5.2 Siklus token

```
Setiap request → api.ts memasang Authorization dari memori
  401 UNAUTHENTICATED:
    navigator.locks.request('faceclock-refresh'):
      token di memori sudah lebih baru? ⇒ keluar, ulangi request
      POST #2 /auth/refresh {refresh_token}
         200 ⇒ simpan token baru; BroadcastChannel kirim {type:'tokens-rotated'}
                ulangi request asli SATU kali
         401 INVALID_REFRESH_TOKEN | REFRESH_TOKEN_REUSED ⇒
                hapus token, queryClient.clear(), redirect /login?reason=session_expired
Tab lain menerima 'tokens-rotated' ⇒ baca localStorage, reset access token di memori
```

Satu request asli hanya diulang **sekali**. Kalau percobaan kedua juga `401`,
sesinya memang berakhir — mengulang lagi hanya menunda pesan yang harus dilihat pengguna.

### 5.3 Meninjau satu absensi pending

```
/attendances/pending
 ├─ #61 GET /attendances/pending?page=1&per_page=20   (tertua dulu)
 ├─ tabel: karyawan, waktu lokal, tipe, fallback_reason, waiting_hours, geofence
 │    baris milik reviewer sendiri ⇒ TANPA tombol aksi, label "Absensi Anda" (B11)
 └─ klik baris → panel detail
      ├─ #58  detail (DTO admin)
      ├─ #59  foto absensi           → <AuthImage>
      ├─ #44  referensi wajah        → 3 foto berdampingan
      ├─ #45  foto referensi         → <AuthImage>
      ├─ #66  lokasi kantor          → peta kecil + lingkaran radius
      ├─ #60  10 absensi terakhir karyawan (konteks)
      ├─ #65  percobaan gagal hari itu
      │
      ├─ [Setujui] → #62 {review_note?}
      │     200 ⇒ toast, invalidasi, buka otomatis baris berikutnya di antrian
      │     409 ATTENDANCE_ALREADY_REVIEWED ⇒ refetch + info "sudah ditinjau <nama>"
      │     403 SELF_REVIEW_DENIED ⇒ seharusnya tidak terjadi (UI mendahului);
      │           bila terjadi, tampilkan apa adanya — jangan ditelan
      └─ [Tolak]  → dialog alasan WAJIB (3–500) → #63
            200 ⇒ toast + info "karyawan dapat mencatat ulang hari ini"
                  ([Fase 4 § 4.6](05-Fase4.md#46-admin) employee_can_retry)
```

"Buka otomatis baris berikutnya" adalah keputusan sadar: antrian yang harus
di-klik-kembali-lalu-klik-berikutnya membuat reviewer terburu-buru, dan reviewer
yang terburu-buru menyetujui tanpa melihat.

### 5.4 Mengubah threshold

```
/settings → kartu "Verifikasi wajah"
 ├─ #28 GET /settings
 ├─ ubah face.similarity_threshold 0.42 → 0.35
 ├─ ConfirmDialog bahaya-tinggi:
 │    "Menurunkan ambang membuat sistem lebih mudah menerima wajah yang mirip."
 │    menampilkan: nilai lama, nilai baru, model_version aktif, tautan laporan kalibrasi
 ├─ #30 PUT /settings/face.similarity_threshold {value: 0.35}
 │    422 VALIDATION_ERROR ⇒ tampilkan details[].message per field
 └─ 200 ⇒ invalidasi ["settings"] + ["attendances","context"]
          toast: "Berlaku untuk absensi berikutnya. Absensi lama tetap menyimpan
                  ambang yang berlaku saat itu." ([Fase 4 E42](05-Fase4.md#65-data--operasional))
```

### 5.5 Menjalankan reindex

```
/face/reindex  (face.reindex)
 ├─ #51/#50 cek job berjalan ⇒ bila ada, langsung tampilkan progres
 ├─ pilih to_model_version → [Dry run] → #49 {dry_run:true}
 │     tampilkan total_count, affected_employee_count
 ├─ [Jalankan] aktif hanya setelah dry run
 │     ConfirmDialog: "Selama reindex berjalan, pendaftaran wajah baru ditolak."
 │     #49 {dry_run:false} → 202
 │       409 REINDEX_IN_PROGRESS ⇒ arahkan ke job yang sedang berjalan
 │       422 VALIDATION_ERROR    ⇒ "Versi model tidak cocok dengan layanan inference
 │                                  yang sedang berjalan." ([Fase 3 § 4.5](04-Fase3.md#45-reindex--guard-facereindex))
 │       503 SERVICE_UNAVAILABLE ⇒ "Layanan inference belum siap."
 ├─ poll #51 tiap 3 detik selama running
 └─ selesai ⇒ daftar incomplete_employees + pengingat kalibrasi threshold (§ 2.8)
```

### 5.6 Export rekap

```
/reports (filter tersimpan di URL)
 ├─ tab Rekap  → #71  |  tab Detail → #60
 ├─ [Export CSV]  (<Can permission="attendance.export">)
 │     fetch #72 dengan filter identik + header Authorization
 │       422 VALIDATION_ERROR (melebihi export_max_rows) ⇒
 │             "Rentang terlalu besar (±N baris). Persempit tanggal atau filter."
 │       200 ⇒ blob → createObjectURL → <a download> → klik → revokeObjectURL
 └─ toast: "Export dicatat di audit log."
```

Tombol export menampilkan indikator progres selama unduhan berjalan dan
dinonaktifkan — export 100.000 baris butuh beberapa detik, dan tanpa indikator
pengguna akan menekannya lima kali.

---

## 6. Edge Case, Validasi & State

### 6.1 Tiga state yang wajib ada di setiap layar

Setiap halaman daftar dan detail **wajib** menangani empat keadaan, dan
membedakannya adalah bagian dari definisi selesai:

| Keadaan | Tampilan |
|---|---|
| Loading | Skeleton berbentuk sama dengan kontennya, bukan spinner di tengah layar |
| Kosong (query berhasil, `data: []`) | `<EmptyState>` dengan kalimat yang menjelaskan **kenapa** kosong dan aksi berikutnya |
| Tidak ditemukan (`404`) | `<EmptyState>` varian tenang — **bukan** error merah (§ 2.3 aturan 1) |
| Error | `<ErrorState>` dari `ERROR_MESSAGES`, dengan `request_id` untuk 5xx |

Contoh kalimat empty state yang benar:

| Layar | Kosong |
|---|---|
| Antrian approval | *"Tidak ada absensi yang menunggu persetujuan."* (bukan "Tidak ada data") |
| Referensi wajah karyawan | *"Karyawan ini belum mendaftarkan wajah."* + tautan panduan |
| Daftar lokasi kantor | *"Belum ada lokasi kantor. Geofence sedang aktif, sehingga karyawan belum bisa absen."* — ini keadaan `503 ATTENDANCE_NOT_CONFIGURED` di [Fase 4](05-Fase4.md#210-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0) yang terlihat dari sisi admin |
| Audit log terfilter | *"Tidak ada aktivitas pada rentang ini."* |

### 6.2 Autentikasi & sesi

| # | Kondisi | Penanganan |
|---|---|---|
| E1 | Dua tab, access token kedaluwarsa bersamaan | Web Locks + pemeriksaan ulang di dalam lock (§ 2.2b). **Tidak boleh** memicu `REFRESH_TOKEN_REUSED` |
| E2 | Refresh token dicabut (deteksi reuse dari perangkat lain) | `401 REFRESH_TOKEN_REUSED` → bersihkan, `/login?reason=session_expired` dengan pesan jujur |
| E3 | Admin mencabut role user saat ia membuka halaman | Request berikutnya `403 FORBIDDEN`; UI menampilkan `<ErrorState>` dan me-*refetch* #5 sehingga menu ikut menyusut |
| E4 | User dinonaktifkan saat aktif | `401` (backend menaikkan `token_version`) → logout dengan pesan *"Akun Anda dinonaktifkan."* |
| E5 | `must_change_password = true` | Navigasi diblokir ke `/account` sampai #6 berhasil; setelah itu **semua sesi dicabut** ([Fase 1 § 4.1](02-Fase1.md#41-auth)) → login ulang. UI mengatakannya di depan, bukan sesudah |
| E6 | Backend mati | `<ErrorState>` "Tidak dapat terhubung ke server" + tombol coba lagi. Bukan layar putih |
| E7 | Jam browser salah | Tidak berpengaruh — semua waktu dari server, dirender `<LocalTime>` dengan `attendance.timezone` |
| E8 | Login di tab lain sebagai user berbeda | `BroadcastChannel` menyiarkan `user-changed`; tab lama `queryClient.clear()` + reload. Tanpa ini, tab lama menampilkan data user lain dengan token baru |

### 6.3 Otorisasi & kepemilikan

| # | Kondisi | Penanganan |
|---|---|---|
| E9 | User membuka URL yang tidak boleh diakses | `<RequirePermission>` → `/403` dengan penjelasan izin apa yang kurang |
| E10 | Admin membuka `/attendances/:id` milik karyawan yang tidak boleh dilihat | `404` → empty state tenang (§ 2.3) |
| E11 | Admin mencoba menyetujui absensinya sendiri lewat URL langsung | Tombol tidak ada; kalau dipaksa, `403 SELF_REVIEW_DENIED` ditampilkan apa adanya |
| E12 | Reviewer tidak punya `face.read_any` | Panel detail menyembunyikan bagian foto referensi dengan catatan *"Anda tidak punya izin melihat referensi wajah."* — bukan kotak gambar rusak |
| E13 | User memberi role yang melebihi miliknya | Opsi dinonaktifkan + tooltip; `403 ROLE_ESCALATION_DENIED` tetap ditangani |
| E14 | User tanpa role sama sekali | Mendarat di `/account` dengan penjelasan (§ 3.2) |

### 6.4 Form & validasi

Skema `zod` mencerminkan aturan backend, tapi **backend tetap penentu**. Ketika
`422 VALIDATION_ERROR` datang, `details[]` dipetakan ke field form lewat
`setError(field, …)` — pesan server ditampilkan pada field yang benar, bukan
sebagai toast anonim.

| Field | Aturan client (mengikuti backend) |
|---|---|
| `email` | ≤ 254, format email, di-trim & lowercase |
| `password` | 10–128, huruf **dan** angka ([Fase 1 § 2.5](02-Fase1.md#25-strategi-token)) |
| `employee_number` | 1–50, `^[A-Za-z0-9._/-]+$` |
| `full_name` | 2–120 setelah trim |
| `phone` | `^[0-9+][0-9 +()-]{6,19}$` |
| `join_date` | ≤ hari ini |
| `radius_meter` | 10 … `attendance.max_distance_meter` (dibaca dari #28, **tidak** hardcode) |
| `lat` / `lng` | −90…90 / −180…180, maks 7 desimal |
| `review_note` (reject) | **wajib**, 3–500 |
| `reason` (hapus data wajah) | **wajib**, 3–500 |
| rentang tanggal laporan | `from ≤ to`, maks 366 hari |
| `face.similarity_threshold` | 0…1 |
| `workday_cutoff_hour` | integer 0…11 |

Ambang seperti `max_distance_meter` dan `max_note_length` **dibaca dari `/settings`**,
bukan ditulis ulang sebagai konstanta di frontend. Kalau ditulis ulang, mengubah
setting di server akan membuat UI menolak nilai yang sebenarnya sah — bug yang
sangat sulit dilacak karena semua yang dilihat pengguna terlihat benar.

### 6.5 Data & konkurensi

| # | Kondisi | Penanganan |
|---|---|---|
| E15 | Dua admin menyetujui record yang sama | Yang kedua `409 ATTENDANCE_ALREADY_REVIEWED` → auto-`refetch` + info siapa yang meninjau |
| E16 | Bulk review 50, 3 gagal | `results[]` dirender per item: 47 berhasil, 3 gagal dengan alasan masing-masing. **Tidak** ditampilkan sebagai "berhasil" |
| E17 | Admin lain menghapus lokasi terakhir | `409` ditangani; daftar di-*refetch* |
| E18 | Antrian bertambah selama reviewer bekerja | Poll #61 tiap 30 detik; badge "N baru" yang **tidak** memindahkan baris yang sedang dibuka. Antrian yang melompat-lompat sendiri membuat reviewer kehilangan tempat |
| E19 | `per_page` besar dari URL yang diedit | Backend meng-*clamp* ke 100 ([Fase 1 E33](02-Fase1.md#63-data--konkurensi)); UI membaca `meta` sebagai kebenaran, bukan parameter yang ia kirim |
| E20 | `sort` di URL bukan kolom whitelist | `422` → jatuh ke sort default + toast informatif |
| E21 | Job reindex selesai saat halaman terbuka | Poll berhenti otomatis saat `status` terminal; ringkasan ditampilkan tanpa perlu reload |
| E22 | Foto sudah di-purge retensi | `410` → *"Foto sudah dihapus sesuai kebijakan retensi."* Record absensi tetap ditampilkan penuh — hilangnya foto bukan hilangnya catatan kehadiran |

### 6.6 Kinerja & memori

| # | Kondisi | Penanganan |
|---|---|---|
| E23 | Daftar approval 200 baris | Tidak memuat foto di daftar (§ 2.4). Paginasi 20/halaman |
| E24 | Object URL bocor | `revokeObjectURL` di cleanup `useEffect`; test kebocoran (§ 11.6) |
| E25 | Export besar | Progres + tombol dinonaktifkan; batas `export_max_rows` dengan pesan jelas |
| E26 | Tab peta membuat ulang instance Leaflet | Instance di-*memo*; marker & lingkaran diperbarui, bukan dibuat ulang |
| E27 | Poll berjalan di tab latar | Poll berhenti saat `document.hidden`; TanStack `refetchIntervalInBackground: false` |

---

## 7. Struktur Folder

Melanjutkan scaffolding [Fase 0 § 2.7](01-Fase0.md#27-scaffolding-faceclock-web-react).

```
apps/faceclock-web/
├── src/
│   ├── app/
│   │   ├── providers.tsx              # (diubah) Query + Auth + Toast + Router
│   │   ├── router.tsx                 # (diubah) seluruh route § 3.1
│   │   ├── routes.config.ts           # ← BARU: {path, element, permission} deklaratif
│   │   └── AppShell.tsx               # ← BARU: sidebar + topbar + outlet
│   │
│   ├── lib/
│   │   ├── api.ts                     # (diselesaikan) fetch + envelope + ApiError
│   │   ├── auth/
│   │   │   ├── AuthProvider.tsx       # ← BARU: sesi + permissions dari #5
│   │   │   ├── tokenStore.ts          # ← BARU: memori + localStorage (D23)
│   │   │   ├── refreshLock.ts         # ← BARU: Web Locks + fallback (§ 2.2b)
│   │   │   ├── broadcast.ts           # ← BARU: BroadcastChannel lintas-tab
│   │   │   └── useAuth.ts
│   │   ├── permissions.ts             # ← BARU: katalog permission + can()
│   │   ├── errors/
│   │   │   ├── codes.ts               # ← BARU: union 39 ApiErrorCode
│   │   │   ├── messages.ts            # ← BARU: Record<ApiErrorCode, …>
│   │   │   └── hints.ts               # ← BARU: 9 hint → kalimat Indonesia
│   │   ├── query.ts                   # ← BARU: queryClient + kebijakan retry
│   │   ├── datetime.ts                # ← BARU: UTC → attendance.timezone
│   │   ├── download.ts                # ← BARU: unduh bertoken (§ 2.10)
│   │   └── format.ts
│   │
│   ├── components/
│   │   ├── ui/                        # Button, Input, Select, Dialog, Toast, Badge…
│   │   ├── auth/
│   │   │   ├── Can.tsx
│   │   │   └── RequirePermission.tsx
│   │   ├── data/
│   │   │   ├── DataTable.tsx
│   │   │   ├── Pagination.tsx
│   │   │   └── FilterBar.tsx
│   │   ├── media/
│   │   │   ├── AuthImage.tsx          # ← inti D14 (§ 2.4)
│   │   │   └── useAuthBlob.ts
│   │   ├── map/
│   │   │   ├── MapPicker.tsx          # ← D22
│   │   │   └── LocationPreview.tsx    # peta kecil read-only untuk panel approval
│   │   ├── feedback/
│   │   │   ├── EmptyState.tsx
│   │   │   ├── ErrorState.tsx
│   │   │   ├── Skeletons.tsx
│   │   │   └── ConfirmDialog.tsx
│   │   └── domain/
│   │       ├── SimilarityBar.tsx
│   │       ├── HintList.tsx
│   │       ├── AttendanceStatusBadge.tsx
│   │       ├── GeofenceBadge.tsx
│   │       └── LocalTime.tsx
│   │
│   ├── features/
│   │   ├── auth/                      # login, account, ganti password
│   │   ├── dashboard/
│   │   ├── attendance/
│   │   │   ├── pages/{AttendanceListPage,PendingQueuePage,AttendanceDetailPage}.tsx
│   │   │   ├── components/{ReviewPanel,ReviewDialog,BulkReviewBar,EvidencePanel}.tsx
│   │   │   └── api.ts                 # hooks #58,#59,#60,#61,#62,#63,#64,#65
│   │   ├── reports/
│   │   │   ├── pages/ReportsPage.tsx
│   │   │   ├── components/{SummaryTable,DetailTable,ExportButton}.tsx
│   │   │   └── api.ts                 # hooks #60,#71,#72
│   │   ├── employees/
│   │   │   ├── pages/{EmployeeListPage,EmployeeFormPage,EmployeeDetailPage,EmployeeFacePage}.tsx
│   │   │   └── api.ts                 # #7,#8,#10,#11,#12,#36,#44,#46,#47
│   │   ├── consents/
│   │   ├── users/
│   │   ├── roles/
│   │   ├── locations/
│   │   ├── settings/
│   │   │   ├── pages/SettingsPage.tsx
│   │   │   ├── components/{SettingCard,SettingField,ConsequenceDialog}.tsx
│   │   │   └── consequences.ts        # peta key → teks peringatan (§ 2.6)
│   │   ├── reindex/
│   │   ├── security/                  # /security/attempts
│   │   └── audit/
│   │
│   ├── types/
│   │   ├── api.ts                     # (dari Fase 0) envelope
│   │   ├── dto/                       # ← BARU: satu file per domain, snake_case
│   │   │   ├── auth.ts  employee.ts  user.ts  role.ts
│   │   │   ├── attendance.ts          # AttendanceAdminDTO vs AttendanceEmployeeDTO
│   │   │   ├── face.ts  consent.ts  location.ts  setting.ts
│   │   └── permissions.ts
│   │
│   ├── test/
│   │   ├── msw/
│   │   │   ├── handlers/              # satu file per fase kontrak
│   │   │   └── server.ts
│   │   └── utils/renderWithProviders.tsx
│   └── main.tsx
│
├── e2e/                               # ← BARU: Playwright
│   ├── admin-approval.spec.ts
│   ├── settings.spec.ts
│   ├── rbac-visibility.spec.ts
│   └── fixtures/
├── playwright.config.ts
└── vitest.config.ts
```

Dua catatan struktur:

- **`types/dto/attendance.ts` mendefinisikan dua tipe berbeda**, bukan satu tipe
  dengan field opsional. Ini mencerminkan pemisahan DTO di
  [Fase 4 § 7](05-Fase4.md#7-struktur-folder), dan membuat `tsc` mencegah halaman
  karyawan (Fase 6) membaca `matched_similarity` — field itu tidak ada di tipenya.
- **`features/*/api.ts`** adalah satu-satunya tempat yang memanggil `api.ts`.
  Komponen memakai hook, tidak pernah `fetch` langsung. Ini yang membuat penambahan
  header, penanganan `401`, dan pemetaan error berlaku di semua tempat sekaligus.

---

## 8. Checklist Task

### 8.0 Prasyarat
- [ ] **Konfirmasi D20–D23** (§ 2.0)
- [ ] **Sepakati 3 endpoint backend baru (#71, #72, #73) + 2 revisi kecil** (§ 2.1, § 12)
- [ ] Fase 4 selesai; ada data uji: karyawan ter-enroll, absensi `approved` dan `pending_review`

### 8.1 Backend (bagian Fase 5 yang bukan React)
- [ ] `GET /attendances/summary` (#71) + agregasi SQL + test
- [ ] `GET /attendances/export` (#72) — CSV streaming + BOM + `Content-Disposition`
- [ ] Setting baru `attendance.export_max_rows` (default 100000) + validator
- [ ] Audit `attendance.exported`
- [ ] Perluas #7 `GET /employees`: filter & field `attendance_mode`, `consent_status`, `enrollment_status`
- [ ] Perjelas & uji #58: DTO admin bila `attendance.read_all`, DTO employee bila hanya `read_self`
- [ ] `GET /settings/face-quality-status` (#73) — panggil `/ready` inference live (timeout 2 dtk), bandingkan 7 key `face.*`, `checked_at:null` bila inference tak terjangkau (K-02)
- [ ] Daftarkan #71/#72/#73 di `routes.go`; `TestAllRoutesHaveGuards` tetap lulus
- [ ] Perluas `rbac_matrix_test.go` dengan #71/#72/#73

### 8.2 App shell
- [ ] `tokenStore.ts` (D23) — access di memori, refresh di `localStorage`
- [ ] `refreshLock.ts` — Web Locks + pemeriksaan ulang di dalam lock + fallback
- [ ] `broadcast.ts` — `tokens-rotated`, `user-changed`, `logout`
- [ ] `api.ts` — envelope, `ApiError` lengkap (`hints`, `extra`, `request_id`), retry 401 sekali
- [ ] `query.ts` — `queryClient`, **tanpa retry 4xx**, `refetchIntervalInBackground: false`
- [ ] `AuthProvider` + `useAuth` + `usePermissions`
- [ ] `<Can>`, `<RequirePermission>`, `/403`
- [ ] ESLint rule: literal nama role dilarang di luar `lib/permissions.ts`
- [ ] `errors/codes.ts` (39 code) + `messages.ts` + `hints.ts`
- [ ] `datetime.ts` — konversi ke `attendance.timezone`, bukan zona browser
- [ ] `AppShell` + sidebar deklaratif dari `routes.config.ts`
- [ ] Redirect cerdas `/` (§ 3.2) + placeholder `/me/*`
- [ ] `GlobalErrorBoundary` + halaman `/404`

### 8.3 Komponen bersama
- [ ] `<AuthImage>` + `useAuthBlob` — cache `Blob`, revoke per-mount, 401/403/404/410
- [ ] `<DataTable>` + paginasi & sort ter-URL (whitelist backend)
- [ ] `<EmptyState>` (3 varian), `<ErrorState>`, `<Skeletons>`
- [ ] `<ConfirmDialog>` dengan tingkat bahaya + teks konsekuensi
- [ ] `<SimilarityBar>`, `<HintList>`, `<LocalTime>`, badge status & geofence
- [ ] `download.ts` — unduhan bertoken + revoke

### 8.4 Absensi & approval
- [ ] `/attendances` — filter lengkap ter-URL, DTO admin
- [ ] `/attendances/pending` — tertua dulu, `waiting_hours`, poll 30 dtk, badge "N baru"
- [ ] **B11 di UI**: baris milik reviewer tanpa tombol aksi + label penjelas
- [ ] Panel bukti: foto absensi + 3 foto referensi berdampingan + `<SimilarityBar>` + hints + peta kecil
- [ ] Konteks karyawan (#60 10 terakhir) + telemetri (#65)
- [ ] Approve (#62) / Reject (#63, alasan wajib) + buka baris berikutnya otomatis
- [ ] Bulk (#64) hanya atas baris terpilih, maks 50, hasil per item dirender
- [ ] `/security/attempts` (#65)

### 8.5 Konfigurasi
- [ ] `/settings` — 5 kartu, seluruh key dari Fase 1/2/3/4
- [ ] Kartu "Ambang kualitas wajah" (read-only) dari `#73` + badge drift kuning/abu-abu (K-02, § 2.6)
- [ ] `consequences.ts` — dialog peringatan untuk 6 setting berdampak keamanan (§ 2.6)
- [ ] `face.model_version` read-only + penjelasan
- [ ] Validasi client membaca ambang dari `/settings`, bukan konstanta
- [ ] `/office-locations` CRUD + `<MapPicker>` (D22) + lingkaran radius
- [ ] Penjagaan "lokasi aktif terakhir" di UI + tangani `409`
- [ ] Dialog peringatan saat koordinat digeser

### 8.6 Karyawan, consent, wajah, reindex
- [ ] `/employees` CRUD (#7,#8,#10,#11,#12)
- [ ] `/employees/:id` + status consent (#36) + ringkasan absensi
- [ ] `/employees/:id/face` (#44,#45,#46,#47) + dialog ketik-ulang untuk #47
- [ ] `/consents` — filter `consent_status` / `attendance_mode` / `enrollment_status`
- [ ] Penanda & penjelasan `attendance_mode='manual'`
- [ ] Catat consent luring (#37); ubah `attendance_mode` (#11) dengan konfirmasi
- [ ] Dokumen consent read-only (#32, D21) — render Markdown **tersanitasi**
- [ ] `/face/reindex` — dry run wajib, poll 3 dtk, daftar `incomplete_employees`, pengingat kalibrasi

### 8.7 User, role, laporan
- [ ] `/users` (#13–#20) + dialog password sementara sekali tampil
- [ ] `/roles` (#21–#26) + editor permission dikelompokkan (#27)
- [ ] Nonaktifkan opsi eskalasi + tangani `403 ROLE_ESCALATION_DENIED` / `409 LAST_SUPER_ADMIN`
- [ ] `/reports` — tab Rekap (#71) & Detail (#60), filter ter-URL
- [ ] Export (#72) + `<Can permission="attendance.export">` + progres
- [ ] `/audit-logs` (#31) + filter
- [ ] `/dashboard` — kartu dari `meta.total` endpoint yang ada (tanpa endpoint agregat baru)

### 8.8 Kualitas & penutup
- [ ] MSW handlers untuk seluruh endpoint yang dikonsumsi
- [ ] Unit test: `refreshLock`, `AuthImage`, pemetaan error, `datetime`
- [ ] Test: `Record<ApiErrorCode,…>` lengkap (dijaga `tsc`)
- [ ] E2E Playwright: approval, settings, visibilitas RBAC per role
- [ ] `tsc --noEmit`, ESLint, Prettier lulus di CI
- [ ] Lighthouse: tidak ada pergeseran layout besar pada daftar & panel detail
- [ ] `docs/web/fase5-admin-panel.md` — peta route × permission
- [ ] `DONE-Fase-5.md` sesuai Protokol Handoff master plan § 10.4

---

## 9. Dependencies

**Prasyarat:**

| Dari | Yang dibutuhkan |
|---|---|
| Fase 0 | Scaffolding Vite+React+TS+Tailwind+Router+TanStack Query; `lib/api.ts`; tipe envelope; dev proxy; `snake_case` di wire |
| Fase 1 | #1–#31; `permissions` di `/auth/me`; pola ownership → 404; **rotasi refresh token + deteksi reuse** (yang memaksa § 2.2b); aturan replace pada #19/#26 |
| Fase 2 | Kosakata `hints` + kalimat Indonesia; `face.model_version`; laporan kalibrasi yang dirujuk dialog threshold |
| Fase 3 | #32,#36,#37,#44–#47,#49–#52; `attendance_mode`; D14 (foto lewat API) |
| Fase 4 | #58–#70; DTO admin vs employee; B11; katalog error; setting D18/D19 |
| Eksternal | Leaflet + tile OSM (D22); MSW & Playwright (dev) |

**Utang lintas-fase** ([Fase 2 § 13.3](03-Fase2.md#133-risiko-lintas-fase-yang-belum-terselesaikan)):

| Risiko | Status di Fase 5 |
|---|---|
| **R1** — lisensi model InsightFace | ⛔ Masih terbuka. **Tidak memblokir pengerjaan Fase 5**, tapi memblokir rilis produksi |
| **R2** — kalibrasi Dataset B | ⛔ Blocker DoD Fase 4 yang masih berlaku. Fase 5 justru **memberi alatnya**: halaman `/settings` untuk menerapkan hasil kalibrasi, dan `/reports` + `/security/attempts` untuk mengamati sebaran similarity di lapangan |
| **R3** — spoofing foto-dari-layar | ⛔ Fase 6/7. Fase 5 menampilkan `capture_source` di panel bukti sehingga reviewer bisa curiga |
| R4, R5, R7 | ✅ Selesai di Fase 3/4; Fase 5 menyediakan UI-nya |
| **R6** — monorepo vs multi-repo | ✅ **Ditutup** — dikunci monorepo 2026-09-04 ([Fase 0 § 2.1](01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)). Fase 5 memakai keuntungannya: kalimat `hints` dan katalog error dibagikan satu repo dengan Go |

**Yang bergantung pada fase ini:**

| Fase | Mengambil apa |
|---|---|
| Fase 6 | **Seluruh app shell**: `api.ts`, refresh lock, `AuthProvider`, `<Can>`, `<AuthImage>`, `ERROR_MESSAGES`, `hints.ts`, `<LocalTime>`, `<EmptyState>`, layout, `queryClient` |
| Fase 7 | Kalimat error & hint yang sama (dibagikan lewat repo); pola alur approval sebagai rujukan |

---

## 10. Definition of Done

1. #71 dan #72 ada, terdaftar di `routes.go`, lulus `TestAllRoutesHaveGuards` dan
   matriks RBAC; export CSV membuka benar di Excel (UTF-8 BOM) dan menulis `audit_logs`.
1a. #73 ada, mengembalikan 7 ambang `face.*` beserta nilai aktif inference;
    diuji dengan `FakeClient` yang sengaja dibuat drift (satu nilai berbeda) →
    `in_sync:false` pada baris yang tepat; inference tak terjangkau → `200` dengan
    `checked_at:null`, bukan `5xx` (K-02).
2. #7 mengembalikan dan memfilter `attendance_mode`, `consent_status`, `enrollment_status`.
3. #58 terbukti mengembalikan **DTO admin** untuk pemegang `attendance.read_all` dan
   **DTO employee** untuk pemegang `attendance.read_self` saja — diuji.
4. **Uji refresh serentak:** 6 query gagal `401` bersamaan → tepat **satu** panggilan
   `/auth/refresh`, tidak ada `REFRESH_TOKEN_REUSED`, semua query berhasil setelah retry.
5. **Uji dua tab:** dua tab menganggur 16 menit lalu aktif bersamaan → tepat satu
   refresh, kedua tab tetap login.
6. Login sebagai `employee` → sidebar **tidak** menampilkan satu pun menu admin;
   membuka `/users` langsung → `/403`, bukan layar kosong atau data bocor.
7. Login sebagai `admin` → `/roles` terlihat read-only sesuai permission
   ([Fase 1 § 2.4](02-Fase1.md#24-role-default): `admin` tidak punya `role.create`/
   `role.assign_permission`), dan `/face/reindex` **tidak** muncul.
8. **Tidak ada literal nama role** (`'admin'`, `'super_admin'`, `'employee'`) di
   seluruh `src/` di luar `lib/permissions.ts` — ditegakkan ESLint, gagal build bila dilanggar.
9. Panel approval menampilkan foto absensi **dan** foto referensi berdampingan,
   `matched_similarity` relatif `threshold_used`, `hints`, jarak, dan waktu lokal.
10. **B11 di UI:** admin yang punya absensi `pending_review` miliknya sendiri melihat
    baris itu **tanpa tombol** approve/reject, dengan label penjelas.
11. Approve/reject bekerja; approve **tidak mengubah** `server_timestamp` (diverifikasi
    di UI dengan membandingkan sebelum/sesudah).
12. Dua admin menyetujui record yang sama → yang kedua melihat pesan "sudah ditinjau"
    dan daftar ter-*refetch*, **bukan** toast merah "Terjadi kesalahan".
13. Bulk 50 dengan 3 kegagalan menampilkan 47 berhasil + 3 gagal beserta alasannya.
14. Semua 39 error code terpetakan di `ERROR_MESSAGES`; menghapus satu entri
    membuat `tsc --noEmit` gagal.
15. `404` dari pola ownership dirender sebagai **empty state tenang**, bukan error merah.
16. `410` pada foto menampilkan pesan retensi, dan record absensinya tetap tampil penuh.
17. Setiap halaman daftar & detail punya empat keadaan (loading/kosong/tidak
    ditemukan/error) yang berbeda dan benar — diperiksa lewat MSW.
18. Enam setting berdampak keamanan menampilkan dialog konsekuensi sebelum disimpan.
19. Map picker menetapkan `lat`/`lng`/`radius_meter`; menonaktifkan lokasi aktif
    terakhir dicegah di UI **dan** `409` ditangani.
20. Reindex: tombol jalankan terkunci sampai dry run; progres ter-*poll*; daftar
    `incomplete_employees` tampil; pengingat kalibrasi threshold muncul setelah selesai.
21. Halaman `/consents` bisa menyaring `attendance_mode=manual` dan menjelaskan artinya.
22. Export mengunduh file dengan filter yang **identik** dengan tampilan aktif;
    melebihi batas → pesan yang menyarankan mempersempit, bukan file terpotong.
23. Tidak ada kebocoran object URL: setelah membuka & menutup 50 panel detail,
    jumlah object URL aktif kembali ke nol (§ 11.6).
24. `tsc --noEmit`, ESLint, Prettier lulus; CI hijau.
25. E2E Playwright lulus untuk: alur approval, ubah setting, dan visibilitas menu
    untuk 3 principal (`employee`, `admin`, `super_admin`).
26. `docs/web/fase5-admin-panel.md` memuat tabel route × permission yang cocok
    dengan `routes.config.ts`.
27. `DONE-Fase-5.md` ada, memuat keputusan D20–D23 final.

---

## 11. Cara Test / Verifikasi

### 11.1 Persiapan data

```bash
make reset && make up
docker compose -f deploy/docker-compose.yml exec faceclock-api /app/seed
# Skrip penyemai skenario: 5 karyawan, 3 ter-enroll, 1 attendance_mode=manual,
# 1 consent withdrawn, 12 absensi approved, 6 pending_review (2 milik akun admin uji)
./scripts/seed-fase5-scenario.sh
cd apps/faceclock-web && npm run dev
```

Baris "2 milik akun admin uji" disengaja — tanpanya B11 tidak bisa diuji.

### 11.2 Uji refresh serentak (unit/integration, MSW)

```ts
it("hanya satu refresh untuk banyak 401 bersamaan", async () => {
  let refreshCalls = 0;
  server.use(
    http.post("*/auth/refresh", () => { refreshCalls++; return HttpResponse.json({ data: newTokens }); }),
    http.get("*/attendances",  expiredOnceThenOk()),
    /* … 5 endpoint lain yang juga 401 sekali … */
  );
  await Promise.all([...6 query...]);
  expect(refreshCalls).toBe(1);          // BUKAN 6
});
```

Ini test terpenting di seluruh Fase 5: kalau ia gagal, produksi akan melempar
pengguna ke halaman login setiap 15 menit, dan penyebabnya (deteksi reuse Fase 1)
tidak akan terlihat dari gejalanya.

### 11.3 Uji dua tab (Playwright)

```ts
const a = await context.newPage(); const b = await context.newPage();
await login(a); await b.goto("/dashboard");
await context.clock.fastForward("16:00");        // lewati masa berlaku access token
await Promise.all([a.reload(), b.reload()]);
await expect(a.getByTestId("topbar-user")).toBeVisible();
await expect(b.getByTestId("topbar-user")).toBeVisible();   // tidak ada yang ter-logout
```

### 11.4 Uji visibilitas RBAC (Playwright, 3 principal)

| Principal | Harapan |
|---|---|
| `employee` | Sidebar hanya ruang karyawan; `/users`, `/settings`, `/attendances` → `/403` |
| `admin` | Semua menu operasional; `/roles` tanpa tombol buat/ubah permission; `/face/reindex` tidak ada di sidebar dan `/403` bila dibuka |
| `super_admin` | Semua terlihat, termasuk `/face/reindex` |

Tabel harapan ditulis sebagai data (mencerminkan
[matriks RBAC Fase 1 § 11.3](02-Fase1.md#113-matriks-rbac-test-otomatis-wajib)),
sehingga menambah halaman tanpa menambah barisnya membuat test gagal.

### 11.5 Uji alur approval (Playwright)

```
1. Login admin → /attendances/pending
2. Assert: baris milik admin sendiri TIDAK punya tombol Setujui/Tolak,
   dan menampilkan label "Absensi Anda"
3. Klik baris karyawan lain → panel detail
4. Assert: foto absensi dan foto referensi termuat (bukan alt/broken)
5. Assert: SimilarityBar menampilkan nilai DAN penanda ambang
6. Klik Tolak tanpa alasan → tombol kirim nonaktif
7. Isi alasan → kirim → toast + baris berikutnya terbuka otomatis
8. Setujui record yang sudah disetujui admin lain (MSW balas 409)
   → assert pesan "sudah ditinjau", daftar ter-refetch, TIDAK ada toast merah generik
```

### 11.6 Uji kebocoran object URL

```ts
it("merevoke seluruh object URL", async () => {
  const created: string[] = [], revoked: string[] = [];
  vi.spyOn(URL, "createObjectURL").mockImplementation((b) => { const u = `blob:${created.length}`; created.push(u); return u; });
  vi.spyOn(URL, "revokeObjectURL").mockImplementation((u) => { revoked.push(u); });

  for (let i = 0; i < 50; i++) { const { unmount } = render(<AuthImage path={`/api/v1/attendances/${i}/photo`} />); await screen.findByRole("img"); unmount(); }
  expect(revoked.sort()).toEqual(created.sort());
});
```

### 11.7 Uji state error/kosong/tidak-ditemukan (MSW)

| Skenario | Harapan |
|---|---|
| `/attendances/pending` → `data: []` | *"Tidak ada absensi yang menunggu persetujuan."* |
| `/attendances/:id` → `404 NOT_FOUND` | Empty state tenang, **bukan** merah |
| `/attendances/:id/photo` → `410` | Pesan retensi; sisa detail tetap tampil |
| `/attendances/:id/photo` → `403` | *"Anda tidak punya izin melihat foto ini."* |
| Endpoint apa pun → `500` | `<ErrorState>` + `request_id` + tombol salin |
| Backend mati (network error) | *"Tidak dapat terhubung ke server"* + coba lagi |
| `/office-locations` → `data: []` | Menyebut konsekuensinya: karyawan belum bisa absen |

### 11.8 Uji dialog konsekuensi

```
/settings → turunkan face.similarity_threshold 0.42 → 0.30
  assert: dialog muncul, memuat nilai lama & baru, model_version, dan tautan laporan
  assert: tombol simpan butuh konfirmasi eksplisit (bukan Enter tak sengaja)
  batalkan → nilai kembali 0.42, TIDAK ada request PUT terkirim
```

Diulang untuk `outside_geofence_policy`, `require_face_for_checkout`,
`geofence_enabled`, `allow_fallback_without_enrollment`, `timezone`.

### 11.9 Uji export

```
/reports?from=2026-09-01&to=2026-09-30&status=approved
 → Export CSV
   assert: request membawa query yang IDENTIK dengan tampilan
   assert: file diawali BOM, baris pertama header sesuai § 2.1
   assert: kolom matched_similarity ADA (endpoint dijaga attendance.export)
 → set MSW membalas 422 (melebihi batas)
   assert: pesan menyebut jumlah baris & saran mempersempit; tidak ada file terunduh
```

### 11.10 Verifikasi manual lintas peran

Daftar periksa singkat yang dijalankan sebelum menyatakan fase selesai:

- [ ] Login `employee` → tidak ada satu pun menu admin yang terlihat
- [ ] Semua waktu tampil dalam `attendance.timezone`, dan tooltip-nya menunjukkan UTC
- [ ] Ubah `attendance.timezone` → absensi lama tetap menampilkan `work_date` yang sama
- [ ] Matikan `faceclock-inference` → panel admin tetap berfungsi penuh (Fase 5 tidak
      memanggil inference sama sekali)
- [ ] Buka DevTools → `localStorage` **hanya** memuat refresh token, tidak ada access token
- [ ] Buka DevTools Network → tidak ada URL foto yang bisa dibuka di tab baru tanpa header

Baris terakhir adalah verifikasi langsung atas D14: menyalin URL foto dari Network
lalu membukanya di tab baru harus menghasilkan `401`, bukan gambar.

---

## 12. Kontrak Fase 0–4 yang Perlu Revisi (dilaporkan, bukan diubah diam-diam)

| # | Kontrak | Revisi | Sifat |
|---|---|---|---|
| 1 | Katalog endpoint | **+2 endpoint**: #71 `GET /attendances/summary`, #72 `GET /attendances/export` | Tambahan. Memakai permission `attendance.read_all` & `attendance.export` yang **sudah** di-seed Fase 1 — tidak ada permission baru |
| 1a | Katalog endpoint | **+1 endpoint**: #73 `GET /settings/face-quality-status` — resolusi [K-02](09-Revisions-Log.md#k-02--salinan-ambang-kualitas-tidak-punya-tempat-di-ui--celah--resolved), bukan kebutuhan Fase 5 sendiri. Memakai `settings.read` yang **sudah** di-seed Fase 1 | Tambahan (**Wajib** — [REV-EP-11](09-Revisions-Log.md#b-perubahan-katalog-endpoint)) — tidak ada permission baru |
| 2 | #7 `GET /employees` (Fase 1) | Tambah filter & field `attendance_mode`, `consent_status`, `enrollment_status` | Tambahan, kompatibel mundur |
| 3 | #58 `GET /attendances/{id}` (Fase 4) | Perjelas: DTO **admin** bila pemanggil punya `attendance.read_all`, DTO **employee** bila hanya `attendance.read_self`. Fase 4 § 4.6 hanya menyebutkan DTO lengkap untuk #60 | Klarifikasi, bukan perubahan perilaku |
| 4 | `app_settings` | Tambah `attendance.export_max_rows` (default 100000) | Tambahan; migration Fase 5 |
| 5 | Fase 0 § 2.7 (scaffolding web) | Dependensi bertambah: `react-hook-form`, `zod`, `leaflet` (D22), `msw` + `@playwright/test` (dev) | Tambahan |
| 6 | Fase 0 — penyajian `faceclock-web` produksi | **`getUserMedia` dan Geolocation butuh secure context.** Untuk Fase 6 di luar `localhost`, web **wajib** disajikan lewat **HTTPS**. Belum tercatat di Fase 0 | ⚠️ Implikasi deployment; harus diputuskan sebelum Fase 6 diuji di perangkat nyata |
| 7 | Sumber kalimat `hints` | Kalimat Indonesia untuk 9 `hints` ada di dua tempat: `internal/inference/hints.go` (Go) dan `lib/errors/hints.ts` (TS). Rekomendasi: satu berkas kanonik `docs/api/hints.json` yang keduanya baca/generate | Saran; memanfaatkan monorepo (D1) |

Nomor 6 pantas diperhatikan sekarang, bukan saat Fase 6: menyiapkan sertifikat dan
reverse proxy adalah pekerjaan infrastruktur yang punya waktu tunggu sendiri, dan
tanpa HTTPS **tidak satu pun** fitur inti Fase 6 bisa diuji di luar `localhost`.
