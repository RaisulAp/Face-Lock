# Fase 7 — Flutter Mobile (`faceclock-mobile`)

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 7**. Ukuran: 🔴 **Besar**.
> Mengikuti Protokol Handoff § 10.2: dokumen ini di-review sebelum eksekusi.
>
> **Depends on:** milestone **"Web fully working"** ([Fase 6](07-Fase6.md)) — lihat
> peringatan verifikasi di § 0.
>
> **Status:** DRAFT — 7 keputusan butuh konfirmasi user (§ 2.0), dan **11 revisi
> kontrak** yang harus disepakati sebelum eksekusi (§ 14).

---

## 0. ⛔ Verifikasi prasyarat — milestone "Web fully working" BELUM terpenuhi

Master plan § 7 menyatakan Fase 7 baru dimulai setelah web tuntas. Sebelum menulis
dokumen ini, prasyarat itu diverifikasi terhadap isi repo, bukan terhadap dokumen.

**Hasil verifikasi (2026-09-04):**

| Yang diperiksa | Hasil |
|---|---|
| `apps/faceclock-api/` | ❌ tidak ada |
| `apps/faceclock-web/` | ❌ tidak ada |
| `services/faceclock-inference/` | ❌ tidak ada |
| `deploy/docker-compose.yml` | ❌ tidak ada |
| `docs/` | ❌ tidak ada |
| `Makefile`, `go.mod`, `package.json` | ❌ tidak ada |
| Repositori git (`.git/`) | ❌ belum diinisialisasi |
| `DONE-Fase-0.md` … `DONE-Fase-6.md` | ❌ tidak ada satu pun |
| Isi root project | Hanya `Plan/` (8 dokumen) dan `face-engine/` (prior art dari project lain) |

**Kesimpulan: Definition of Done Fase 0–6 terpenuhi 0%.** Yang ada adalah
**rencana**, bukan implementasi. Tidak satu pun DoD Fase 5 atau Fase 6 dapat
dinyatakan terbukti, karena tidak ada kode yang bisa diuji:

- DoD Fase 5 poin 4 (uji refresh serentak → tepat satu panggilan refresh) — belum diuji.
- DoD Fase 5 poin 8 (nol literal nama role di `src/`) — tidak ada `src/`.
- DoD Fase 6 poin 1 (nol `<input type="file">`) — tidak ada halaman.
- DoD Fase 6 poin 3 (frame terkirim tidak tercermin) — belum pernah dibuktikan.
- DoD Fase 6 poin 23 (uji perangkat nyata) — belum dijalankan.

**Konsekuensi untuk Fase 7:**

1. Dokumen ini **boleh** ditulis sekarang — merencanakan lebih awal tidak merugikan,
   dan sebagian temuannya (§ 14) justru mengubah kontrak fase sebelumnya *sebelum*
   fase itu dikerjakan, yang jauh lebih murah daripada setelahnya.
2. **Eksekusi Fase 7 tidak boleh dimulai** sampai Fase 0–6 benar-benar dibangun dan
   DoD-nya terbukti. Aplikasi mobile ini adalah klien murni: tanpa API yang berjalan,
   tidak ada satu pun layar yang bisa diselesaikan, dan tidak ada satu pun asumsi
   kontrak yang bisa diverifikasi.
3. Beberapa temuan Fase 7 (§ 14) menyentuh Fase 1 dan Fase 4 — dua fase yang belum
   dieksekusi. Menyepakatinya sekarang berarti perubahan itu masuk ke implementasi
   pertama, bukan menjadi migration tambalan di kemudian hari.

> Seluruh sisa dokumen ini ditulis dengan asumsi eksplisit: **Fase 0–6 akan
> dieksekusi lebih dulu dan DoD-nya terbukti.** Setiap rujukan "sesuai Fase N"
> berarti "sesuai kontrak Fase N sebagaimana direncanakan".

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Dua alasan, dan hanya yang kedua yang tidak bisa digantikan browser:

1. **Absensi lapangan.** Karyawan yang tidak duduk di depan laptop — teknisi,
   sales, petugas cabang — butuh alat yang selalu di saku.
2. **Menutup R3 (spoofing) sejauh yang bisa ditutup.**
   [Fase 6 § 6.5 E29/E30](07-Fase6.md#65-keamanan--kepatuhan) menyatakan terus terang
   bahwa web **tidak** mendeteksi foto-dari-layar maupun kamera virtual, dan
   mendelegasikan penutupannya ke fase ini. Liveness on-device adalah satu-satunya
   kontrol anti-spoofing nyata yang dimiliki seluruh sistem.

### Hasil akhir yang diharapkan
- APK/IPA yang bisa dipasang, tempat karyawan bisa login, memberi consent,
  mendaftarkan wajah, check-in/out dengan verifikasi wajah + liveness, dan melihat
  riwayatnya sendiri.
- Liveness berjalan **on-device sebelum foto dikirim**, dengan tantangan acak,
  dan hasilnya dilaporkan jujur ke server.
- Refresh token disimpan di Keychain/Keystore — lebih baik daripada `localStorage`
  di web, bukan sekadar menyalin keterbatasannya.
- Perilaku jaringan lapangan yang tidak stabil ditangani tanpa menciptakan celah
  keamanan baru.

### Yang TIDAK dikerjakan di fase ini
- **Seluruh fitur admin** (approval, konfigurasi, laporan, kelola user/role) —
  tetap hanya di web (§ 2.1, D27).
- Endpoint backend baru. Nol, dan itu diverifikasi di § 4.6.
- Antrean absensi offline penuh (§ 2.7, D31).
- Push notification.
- Presentation Attack Detection bersertifikat (§ 6.7, D28 alternatif).

---

## 2. Scope & Batas

### 2.0 Keputusan yang butuh konfirmasi

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D27 | Batas scope aplikasi | **Murni employee-facing.** Fitur admin tetap hanya di web | ⚠️ **BUTUH KONFIRMASI** |
| D28 | Mekanisme liveness | `google_mlkit_face_detection` + tantangan acak kedip/senyum, on-device | ⚠️ **BUTUH KONFIRMASI** |
| D29 | Asal frame yang dikirim | **Frame dari sesi liveness itu sendiri** (anti-swap), bukan `takePicture()` terpisah | ⚠️ **BUTUH KONFIRMASI** |
| D30 | Perilaku saat liveness gagal / tidak didukung | Server yang memutuskan lewat `attendance.liveness_policy`; default `preferred` → `pending_review` | ⚠️ **BUTUH KONFIRMASI** |
| D31 | Offline | **Retry-only**, bukan antrean absensi offline | ⚠️ **BUTUH KONFIRMASI** |
| D32 | Jalur distribusi | **Internal** (Managed Google Play private app + Apple Business Manager custom app), bukan store publik | ⚠️ **BUTUH KONFIRMASI** |
| D33 | State management Flutter | Riverpod v2 + `go_router` + `freezed` | ⚠️ **BUTUH KONFIRMASI** |

---

### 2.1 D27 — Murni employee-facing ⚠️ BUTUH KONFIRMASI

Master plan § Fase 7 mendeskripsikannya sebagai *"absensi lapangan yang praktis dari
HP"*, dan daftar pekerjaannya hanya menyebut auth, enroll, check-in/out, riwayat,
liveness, geolocation, dan offline ringan. Tidak ada satu pun fitur admin.

**Rekomendasi: pertahankan batas itu dengan tegas.** Aplikasi ini hanya memakai
permission yang dimiliki role `employee`
([Fase 1 § 2.4](02-Fase1.md#24-role-default)): `employee.read_self`,
`face.enroll_self`, `face.read_self`, `attendance.checkin`, `attendance.read_self`,
`settings.read`.

Alasan menolak "sekalian tambahkan approval di mobile":

1. **Panel approval butuh bukti yang tidak muat di layar ponsel.**
   [Fase 5 § 2.5](06-Fase5.md#25-antrian-approval--pekerjaan-inti-fase-ini) menuntut
   reviewer melihat foto absensi berdampingan dengan tiga foto referensi, bar
   similarity relatif ambang, peta, dan konteks riwayat. Memampatkannya ke layar 6
   inci menghasilkan approval yang dilakukan tanpa benar-benar melihat — persis
   kegagalan yang seluruh rantai verifikasi ini dirancang untuk mencegah.
2. **Melipatgandakan permukaan yang harus dijaga.** Setiap layar admin di mobile
   adalah salinan kedua dari aturan seperti B11 (tidak boleh meninjau absensi
   sendiri) yang bisa menyimpang dari versi web.
3. **Admin sudah punya alat.** Panel web bisa dibuka dari browser ponsel bila
   sesekali diperlukan; ia tidak butuh kamera maupun liveness.

Bila organisasi benar-benar membutuhkan approval di mobile, itu **fase tersendiri**
dengan desainnya sendiri — bukan tambahan diam-diam ke Fase 7.

> Perluasan scope ke fitur admin **tidak dilakukan** tanpa konfirmasi eksplisit.

### 2.2 Prasyarat infrastruktur: HTTPS tetap wajib

Aplikasi native tidak butuh *secure context* seperti `getUserMedia` di web
([Fase 6 § 2.1](07-Fase6.md#21--prasyarat-infrastruktur-https)), tetapi itu
**bukan** alasan untuk melonggarkan transport.

- `API_BASE_URL` **wajib** `https://` di semua environment kecuali dev lokal.
- Android: `usesCleartextTraffic=false` di `AndroidManifest.xml`, dengan
  `network_security_config.xml` yang hanya mengizinkan cleartext untuk host dev
  tertentu (`10.0.2.2`) pada build debug — tidak pernah pada build release.
- iOS: ATS dibiarkan pada default ketat; **tidak ada** `NSAllowsArbitraryLoads`.
- **Certificate pinning:** direkomendasikan untuk build release, dengan cadangan
  pin kedua supaya rotasi sertifikat tidak mematikan seluruh armada perangkat.
  Kalau pinning dianggap terlalu berisiko operasional, catat keputusannya —
  jangan dilewatkan diam-diam.

Aplikasi ini mengirim foto wajah mentah lewat jaringan seluler dan Wi-Fi publik.
Transport yang tidak terenkripsi berarti data biometrik terekspos di setiap hop.

### 2.3 Reuse kontrak, bukan penerjemahan ulang

Aturan pemakaian ulang yang mengikat seluruh fase ini:

| Yang di-reuse | Dari | Aturan |
|---|---|---|
| Kosakata `hints` + kalimat Indonesia | [Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints) | Dibaca dari berkas kanonik `docs/api/hints.json` (usulan [Fase 6 § 12](07-Fase6.md#12-kontrak-fase-05-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam)), **tidak** diterjemahkan ulang di Dart |
| Katalog 39 error code + pesannya | [Fase 5 § 2.3](06-Fase5.md#23-pemetaan-error--dari-code-bukan-pesan-buatan-sendiri) | Dibaca dari berkas kanonik yang sama; Dart hanya memetakan `code` → teks |
| Kapan fallback boleh ditawarkan | [Fase 4](05-Fase4.md#210-error-code-tambahan-registrasi-resmi-ke-katalog-fase-0) | Dari field `can_fallback` di response error. **Tidak** disimpulkan sendiri di client |
| Seluruh ambang & kebijakan | [Fase 4 § 4.4](05-Fase4.md#44-get-attendancescontext) | Dari `/attendances/context` (#55). Tidak ada konstanta di app |
| Alur enrollment | [Fase 3 D15](04-Fase3.md#23-d15--alur-enrollment-bertahap--butuh-konfirmasi) | Sesi bertahap, sama persis |
| Alur fallback | [Fase 4 D17](05-Fase4.md#23-d17--bentuk-alur-fallback--butuh-konfirmasi) | Wajah dulu, fallback sebagai konsekuensi |
| Panduan foto enrollment | [Fase 6 D26](07-Fase6.md#d26--panduan-variasi-foto--butuh-konfirmasi) | Variasi ekspresi/pencahayaan/kacamata — **bukan** hadap kiri/kanan |

Dua sumber kebenaran untuk kalimat yang sama akan menyimpang. Yang menyimpang
paling cepat adalah teks yang dibaca pengguna setiap hari.

### 2.4 Penyimpanan token — menyempurnakan D23, bukan mewarisi keterbatasannya

[Fase 5 D23](06-Fase5.md#c-d23--penyimpanan-token--butuh-konfirmasi) memilih
`localStorage` untuk refresh token di web, dengan pengakuan jujur bahwa itu **tidak**
menghilangkan risiko XSS — hanya mempersempitnya. Mobile tidak punya keterbatasan itu.

| Item | Penyimpanan | Alasan |
|---|---|---|
| Access token (15 menit) | **Memori proses saja** | Hilang saat app di-kill; tidak pernah menyentuh disk |
| Refresh token (30 hari) | **`flutter_secure_storage`** → iOS Keychain (`kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`), Android EncryptedSharedPreferences dengan kunci di Keystore | Terenkripsi kunci perangkat; tidak ikut backup ke cloud |
| Cache DTO (riwayat, context) | Memori + cache disk **tanpa** foto | Lihat § 2.7 |
| Foto wajah | **Memori saja, tidak pernah persisten** | § 2.7 dan § 8.3 |

`ThisDeviceOnly` disengaja: refresh token yang ikut restore iCloud ke perangkat baru
adalah sesi yang berpindah tanpa sepengetahuan siapa pun.

**Ini bukan perubahan kontrak backend** — server tetap menerbitkan token yang sama.
Yang berubah hanya tempat client menyimpannya. Satu hal yang **memang** perlu
perubahan backend ada di § 2.5.

### 2.5 Bahaya `single-flight refresh` — lebih tajam di mobile

[Fase 5 § 2.2b](06-Fase5.md#b-refresh-token-dengan-single-flight-lock--wajib-bukan-optimasi)
menemukan bahwa rotasi + deteksi reuse ([Fase 1 § 2.5](02-Fase1.md#25-strategi-token))
membuat beberapa `401` bersamaan mencabut seluruh sesi. **Bahaya yang sama berlaku
penuh di mobile**, dan mobile menambah satu varian yang lebih buruk.

#### a. Lock dalam proses — padanan Web Locks

Flutter hanya punya satu isolate UI, jadi tidak ada masalah lintas-tab. Tapi `Dio`
mengirim beberapa request paralel, dan semuanya bisa gagal `401` bersamaan.

```dart
class AuthInterceptor extends QueuedInterceptor {
  Completer<void>? _refreshing;

  Future<void> _refreshOnce() {
    final existing = _refreshing;
    if (existing != null) return existing.future;   // ikut antre, jangan mulai baru
    final c = _refreshing = Completer<void>();
    _doRefresh().then((_) => c.complete())
                .catchError(c.completeError)
                .whenComplete(() => _refreshing = null);
    return c.future;
  }
}
```

`QueuedInterceptor` (bukan `Interceptor`) dipilih karena ia menahan request
berikutnya sampai handler sebelumnya selesai — setengah dari masalah selesai di
sana. `Completer` menutup sisanya.

#### b. Varian mobile: response refresh yang hilang di jalan

Ini yang tidak dialami web dengan frekuensi berarti, dan ia **merusak sesi secara
permanen**:

```
1. App mengirim POST /auth/refresh dengan token T1
2. Server MEROTASI: T1.used_at = now(), menerbitkan T2, mengirim response
3. Sinyal hilang. Response tidak pernah sampai.
4. App masih memegang T1 (ia tidak pernah menerima T2)
5. Sinyal kembali. App mencoba lagi dengan T1
6. Server: T1.used_at != NULL  ⇒  DETEKSI REUSE
      → seluruh family di-revoke, token_version dinaikkan
      → karyawan dipaksa login ulang, di lapangan, mungkin tanpa mengingat password
```

Ini bukan kasus tepi untuk aplikasi yang namanya saja "absensi lapangan". Ia akan
terjadi setiap minggu di area sinyal lemah.

**Perbaikan yang diusulkan — grace window di sisi server** (revisi kontrak, § 14 R3):

```
Saat token T dengan used_at != NULL dipresentasikan:
  bila (now() - T.used_at) <= auth.refresh_reuse_grace_seconds   (default 30)
   DAN anak langsung T (T.child) BELUM pernah dipakai (child.used_at IS NULL)
   DAN family belum di-revoke
  ⇒ ini pengiriman ulang, bukan pencurian.
     Terbitkan ULANG pasangan token yang sama (kembalikan T.child), JANGAN revoke.
  selain itu ⇒ perlakukan sebagai reuse: revoke seluruh family (perilaku Fase 1 tetap)
```

Sifat keamanan yang dipertahankan: token yang benar-benar dicuri dan dipakai
belakangan (di luar 30 detik) atau setelah anaknya terpakai **tetap** memicu
pencabutan family. Yang berubah hanya jendela sangat pendek yang secara fisik
hanya bisa dihasilkan oleh response yang hilang.

Sisi client tetap wajib: **jangan** mengulang `/auth/refresh` secara agresif.
Maksimum 1 pengulangan, dengan jeda 2 detik, dan hanya untuk kegagalan transport
(timeout/koneksi), **tidak pernah** untuk `401`.

### 2.6 Idempotensi di jaringan lapangan

[Fase 4 § 2.8](05-Fase4.md#28-idempotensi) menyediakan `Idempotency-Key` dan
menyatakan Fase 6/7 *"diwajibkan mengirimkannya"*. Di mobile ia berhenti menjadi
kenyamanan dan menjadi kebutuhan.

Aturan siklus hidup (sama dengan [Fase 6 § 2.4](07-Fase6.md#24-alur-check-in--check-out-d17)):

| Kejadian | Kunci |
|---|---|
| Kirim pertama | ULID baru |
| Timeout/koneksi putus → "Coba kirim lagi" | **Sama** |
| App di-kill lalu dibuka lagi, percobaan yang sama | **Sama** — kunci + `type` + `work_date` disimpan sementara (§ 2.7) |
| Pengguna mengambil foto baru | Baru |
| Kirim fallback dengan foto baru | Baru |

Baris ketiga adalah alasan kunci ini perlu bertahan melewati restart proses: di
lapangan, "aplikasi tertutup saat menunggu" adalah kejadian biasa.

> **Batas idempotensi ([resolusi K-03](09-Revisions-Log.md#k-03--idempotensi-hanya-berlaku-untuk-permintaan-yang-berhasil--celah--resolved)):**
> kunci yang sama hanya dijamin mengembalikan response yang sama **bila percobaan
> sebelumnya BERHASIL** membuat record ([Fase 4 § 2.8](05-Fase4.md#28-idempotensi)
> sudah diperbaiki kalimatnya). Untuk kegagalan (`422`, `409`, `502`/`504` di
> tabel § 2.5b), server **tidak** menyimpan apa pun terhadap kunci itu — "Coba
> kirim lagi" pada E4 dan "Lanjutkan pengiriman" pada E5 (§ 6.4) berarti
> **menjalankan ulang seluruh percobaan dari awal**, termasuk mengirim ulang
> byte foto yang sama dan memanggil inference lagi, **bukan** menerima response
> yang di-cache dari percobaan gagal sebelumnya. Baris tabel di atas ("Timeout/
> koneksi putus", "App di-kill") benar untuk kasus yang paling sering terjadi di
> lapangan — **response sukses yang hilang di jaringan** — tapi tidak berarti
> percobaan yang benar-benar gagal menjadi aman diulang tanpa efek tambahan
> (mis. tetap kena hitungan `attendance_attempts` / rate limit B12 setiap kali).

### 2.7 D31 — Offline: retry-only, bukan antrean ⚠️ BUTUH KONFIRMASI

Master plan § Fase 7 menyebut *"Handling offline/retry ringan bila perlu"*. Kata
**ringan** itu penting, dan rekomendasi ini mengambilnya secara harfiah.

**Rekomendasi: dukung *retry*, tolak *antrean*.**

| Skenario | Perilaku |
|---|---|
| Request sudah terkirim, response hilang | **Retry** dengan `Idempotency-Key` yang sama, sampai 15 menit, otomatis saat konektivitas kembali bila app terbuka. Server mengembalikan record yang sama |
| Tidak ada koneksi **sebelum** mengirim | **Tolak dengan jujur**: *"Tidak ada koneksi. Absensi memerlukan koneksi ke server."* Foto **tidak** disimpan, absensi **tidak** diantrekan |
| Koneksi putus di tengah unggah | Sama dengan baris pertama — kunci dipertahankan |

Tiga alasan menolak antrean absensi offline:

1. **Merusak janji utama sistem.** Master plan § 9: *"Timestamp server-side."*
   Absensi yang diambil pukul 08:00 dan terkirim pukul 12:00 akan tercatat
   `server_timestamp = 12:00`. Menyimpan waktu client sebagai waktu resmi berarti
   mempercayai jam perangkat — hal yang seluruh sistem ini dibangun untuk hindari.
2. **Menciptakan celah yang persis ingin ditutup.** Antrean offline adalah izin
   untuk "ambil foto di rumah, kirim di kantor". Liveness memang berjalan saat
   capture, tapi geofence dievaluasi saat *submit* — dan lokasi saat submit adalah
   lokasi kantor.
3. **Menyimpan foto biometrik di penyimpanan perangkat.** Retensinya tidak jelas,
   perangkat bisa hilang, dan tidak ada mekanisme penghapusan jarak jauh. Ini
   liabilitas UU PDP yang tidak sebanding dengan manfaatnya.

**Apa yang boleh persisten di perangkat:**

| Data | Persisten? | Catatan |
|---|---|---|
| Refresh token | ✅ Secure storage | § 2.4 |
| Metadata percobaan tertunda (`idempotency_key`, `type`, `work_date`, waktu) | ✅ ≤ 15 menit | **Tanpa** foto, tanpa koordinat |
| Cache riwayat & `context` (untuk tampilan saat offline) | ✅ | Read-only, tanpa foto |
| **Byte foto** | ❌ **Tidak pernah** | Memori saja; dilepas setelah kirim berhasil atau dibatalkan |
| Berkas sementara kamera (`XFile`) | ❌ | **Dihapus segera** setelah dibaca (§ 8.3) |

> Bila organisasi benar-benar memerlukan absensi di lokasi tanpa sinyal sama sekali
> (mis. tambang, perkebunan), itu **bukan** penyesuaian kecil di Fase 7. Ia butuh
> desain sendiri: `client_reported_at` menjadi semi-dipercaya, setiap record offline
> **wajib** `pending_review`, dan ada kebijakan retensi foto lokal yang eksplisit.
> Angkat sebagai fase terpisah, jangan diselipkan.

---

## 3. Struktur Layar & Navigasi

### 3.1 Peta layar

```
/splash                     cek sesi + prefetch context → arahkan
/login                      publik
/change-password            wajib bila must_change_password = true

── Setelah login (bottom navigation, 3 tab) ──────────────────────
/attendance   [tab 1]       check-in / check-out          attendance.checkin
/history      [tab 2]       riwayat pribadi               attendance.read_self
/profile      [tab 3]       data karyawan + akun          employee.read_self

── Alur penuh-layar (di luar tab) ────────────────────────────────
/consent                    baca / setujui / cabut        auth
/enrollment                 wizard sesi bertahap          face.enroll_self
/enrollment/camera          layar kamera enrollment
/attendance/liveness        layar liveness + capture
/attendance/result          hasil percobaan
/history/:id                detail absensi
/app-settings               pilih kamera, hapus cache, info versi
/unsupported                perangkat tidak memenuhi syarat
```

Layar liveness dan kamera dibuat **penuh layar tanpa bottom navigation**: ia adalah
alur yang harus diselesaikan atau dibatalkan, dan tab yang masih bisa ditekan di
tengah proses menghasilkan stream kamera yang menggantung.

### 3.2 Guard navigasi (`go_router` redirect)

```dart
redirect: (ctx, state) {
  if (!bootDone)                 return '/splash';
  if (!deviceSupported)          return '/unsupported';
  if (!hasSession)               return '/login';
  if (mustChangePassword)        return '/change-password';
  return null;   // sisanya diputuskan per-layar dari /attendances/context
}
```

Keadaan seperti "belum consent" atau "belum enroll" **tidak** ditangani redirect,
melainkan sebagai **keadaan layar** `/attendance` (§ 6.6) — sama seperti
[Fase 6 § 5.1](07-Fase6.md#51-membuka-meattendance). Redirect otomatis ke layar
enrollment membuat karyawan tidak pernah melihat penjelasan mengapa ia diarahkan.

### 3.3 Permission catalog sebagai lapisan kenyamanan

Meski aplikasi ini hanya untuk `employee`, gating tetap dari `permissions` di
`GET /auth/me` (#5), **bukan** dari nama role — konsisten dengan
[Fase 1 § 2.2](02-Fase1.md#22-model-otorisasi-permission-based-bukan-role-based)
dan [Fase 5 § 2.2d](06-Fase5.md#d-gating-permission-di-ui--lapisan-kenyamanan-bukan-keamanan).

Alasan praktisnya nyata: super admin yang membuka aplikasi ini juga punya
`attendance.checkin`, dan role kustom buatan admin bisa saja tidak punya
`face.enroll_self`. Aplikasi yang mengasumsikan "pengguna app = role employee" akan
menampilkan tombol yang selalu gagal.

```dart
if (!auth.can('face.enroll_self')) {
  // sembunyikan tombol "Daftarkan wajah", tampilkan
  // "Hubungi admin untuk mengaktifkan pendaftaran wajah."
}
```

### 3.4 D33 — Stack teknis ⚠️ BUTUH KONFIRMASI

| Kebutuhan | Paket | Catatan |
|---|---|---|
| State management | `flutter_riverpod` v2 | Provider dapat diuji tanpa widget; `AsyncValue` memetakan langsung ke keadaan loading/error/data yang dituntut § 6 |
| Navigasi | `go_router` | Redirect deklaratif (§ 3.2) |
| HTTP | `dio` | `QueuedInterceptor` untuk auth (§ 2.5a) |
| DTO | `freezed` + `json_serializable` | `@JsonKey(name: 'employee_id')` — wire tetap `snake_case` ([Fase 0 D4](01-Fase0.md#24-d4--konvensi-json-snake_case--butuh-konfirmasi)) |
| Secure storage | `flutter_secure_storage` | § 2.4 |
| Kamera | `camera` | Preview + `startImageStream` untuk liveness |
| Liveness | `google_mlkit_face_detection` | § 6 |
| Lokasi | `geolocator` | Termasuk `Position.isMocked` (§ 14 R5) |
| Izin | `permission_handler` | § 7 |
| Kompresi | `flutter_image_compress` | Membakar orientasi EXIF (§ 5.3) |
| Konektivitas | `connectivity_plus` | Banner offline, pemicu retry |
| Id | `ulid` | `Idempotency-Key` |
| Info perangkat | `package_info_plus`, `device_info_plus` | `User-Agent` bermakna |
| Markdown consent | `flutter_markdown` | Tanpa risiko XSS (render ke widget, bukan HTML), tapi **nonaktifkan pembukaan tautan sembarang** |

`bloc` adalah alternatif yang setara bila tim lebih terbiasa; seluruh dokumen ini
tetap berlaku, yang berubah hanya bentuk provider.

---

## 4. Endpoint yang Dikonsumsi

**Nol endpoint baru.** Seluruhnya sudah ada sejak Fase 1/3/4 dan dipakai persis
seperti Fase 6.

### 4.1 Auth — Fase 1

| # | Method | Path | Dipakai di | DTO / catatan |
|---|---|---|---|---|
| 1 | POST | `/auth/login` | `/login` | `{access_token, refresh_token, user{roles,permissions}}` |
| 2 | POST | `/auth/refresh` | `AuthInterceptor` | § 2.5 — **single-flight wajib** |
| 3 | POST | `/auth/logout` | `/profile` | body `{refresh_token}` |
| 4 | POST | `/auth/logout-all` | `/profile` | "Keluar dari semua perangkat" |
| 5 | GET | `/auth/me` | boot + `refreshOnFocus` | sumber `permissions` (§ 3.3) |
| 6 | POST | `/auth/change-password` | `/change-password` | mencabut semua sesi → login ulang |

### 4.2 Profil & setting — Fase 1

| # | Method | Path | Dipakai di | Catatan |
|---|---|---|---|---|
| 9 | GET | `/employees/me` | `/profile` | `EmployeeDTO` |
| 28 | GET | `/settings` | **Tidak dipakai untuk layar absensi.** `max_note_length`, `require_face_for_checkout`, `checkout_without_face_status` semuanya dibaca dari `#55 context` (§ 14 R1, sudah final) — bukan dari sini | Hanya baris `is_public = true`; `checkout_without_face_status` sengaja **tidak pernah** ada di respons ini (K-01) |

### 4.3 Consent & enrollment — Fase 3

| # | Method | Path | Dipakai di | Catatan |
|---|---|---|---|---|
| 32 | GET | `/consents/document` | `/consent` | `body` Markdown → `flutter_markdown` |
| 33 | GET | `/consents/me` | `/consent`, `/profile` | |
| 34 | POST | `/consents` | `/consent` | `{document_version, agreed:true}` |
| 35 | POST | `/consents/withdraw` | `/consent` | `{reason}` + dialog akibat |
| 38 | POST | `/face/enrollments` | wizard L2 | `{mode:"replace"}`; tangani `409` + `existing_session_id` |
| 39 | GET | `/face/enrollments/{id}` | wizard | **sumber kebenaran langkah** |
| 40 | POST | `/face/enrollments/{id}/photos` | wizard L3 | multipart `image` + `capture_source=mobile_camera` |
| 41 | DELETE | `/face/enrollments/{id}/photos/{pid}` | wizard | |
| 42 | POST | `/face/enrollments/{id}/commit` | wizard L5 | tanpa `force_duplicate` |
| 43 | DELETE | `/face/enrollments/{id}` | batal | |
| 44 | GET | `/employees/{id}/face-references` | `/profile` | milik sendiri, `face.read_self` |
| 45 | GET | `/face/references/{id}/photo` | `AuthImage` Dart (§ 5.5) | |
| 48 | GET | `/face/enrollment-status/me` | wizard L0, `/attendance` | |

### 4.4 Absensi — Fase 4

| # | Method | Path | Dipakai di | DTO |
|---|---|---|---|---|
| 53 | POST | `/attendances/check-in` | `/attendance` | multipart + `Idempotency-Key` + field liveness (§ 14 R4) |
| 54 | POST | `/attendances/check-out` | `/attendance` | idem |
| 55 | GET | `/attendances/context` | `/attendance`, boot | **satu-satunya sumber konfigurasi** |
| 56 | GET | `/attendances/me` | `/history` | **`AttendanceEmployeeDTO`** |
| 57 | GET | `/attendances/me/today` | kartu hari ini | idem |
| 58 | GET | `/attendances/{id}` | `/history/:id` | DTO employee |
| 59 | GET | `/attendances/{id}/photo` | `AuthImage` Dart | tangani `410` |

### 4.5 Yang TIDAK dipakai

#60, #61, #62, #63, #64, #65, #66–#70, #71, #72, dan seluruh #7–#27, #29–#31,
#36, #37, #46, #47, #49–#52 — semuanya dijaga permission yang tidak dimiliki role
`employee` ([Fase 1 § 2.4](02-Fase1.md#24-role-default)), dan semuanya adalah
fitur admin yang secara sadar tidak diduplikasi (D27).

Ini sekaligus verifikasi D27: aplikasi hanya menyentuh 24 dari 72 endpoint, dan
tidak satu pun di antaranya membutuhkan permission di luar milik `employee`.

### 4.6 Verifikasi "nol endpoint baru"

Setiap kebutuhan layar ditelusuri ke endpoint yang sudah ada:

| Kebutuhan | Terpenuhi oleh |
|---|---|
| Tahu harus check-in atau check-out | #55 `today` + `next_action` |
| Tahu sudah enroll / perlu enroll ulang | #55 `is_enrolled`, `needs_re_enrollment` |
| Tahu status consent | #55 `consent_status` |
| Tahu mode manual | #55 `attendance_mode` |
| Tahu ambang GPS & kebijakan geofence | #55 `geofence.*` |
| Tahu batas foto | #55 `photo.*` |
| Tahu jam server | #55 `server_time` |
| Kirim absensi | #53 / #54 |
| Riwayat + foto | #56 / #57 / #58 / #59 |
| Enrollment penuh | #38–#43, #48 |
| Consent penuh | #32–#35 |

**Kesimpulan: tidak ada endpoint baru.** Yang dibutuhkan adalah **field tambahan
pada endpoint yang sudah ada** — seluruhnya didaftarkan di § 14.

---

## 5. Flow per Fitur

### 5.1 Boot

```
/splash
 ├─ deviceSupported()? (§ 6.6)  → tidak: /unsupported
 ├─ baca refresh token dari secure storage
 │     tidak ada ⇒ /login
 ├─ GET #5 /auth/me   (memicu refresh bila access token belum ada)
 │     401 ⇒ hapus token, /login (tanpa pesan error — ini normal)
 ├─ prefetch #55 /attendances/context  (di-cache; dipakai seluruh app)
 ├─ ambil hints.json + error-messages dari asset bundel (§ 2.3)
 └─ mustChangePassword ⇒ /change-password  |  selain itu ⇒ /attendance
```

### 5.2 Check-in — alur lengkap

```
/attendance
 1. #55 context (stale ≤ 30 dtk; refetch saat app kembali ke foreground)
 2. Evaluasi keadaan BERURUTAN — yang pertama cocok menang (§ 6.6):
      consent belum/ dicabut & mode face  → layar consent
      attendance_mode = "manual"          → mode manual (tanpa liveness, tanpa janji wajah)
      !is_enrolled || needs_re_enrollment → ajakan enroll
      today lengkap                       → layar "absensi hari ini lengkap"
      selain itu                          → layar siap
 3. Izin: kamera + lokasi (§ 7). Ditolak permanen ⇒ layar panduan ke Pengaturan
 4. Ambil lokasi lebih dulu (geolocator, desiredAccuracy: high, timeLimit 10 dtk)
      akurasi > context.geofence.max_gps_accuracy_meter
        DAN missing_location_policy = "reject"  ⇒ blokir + [Ambil ulang lokasi]
      isMocked = true                            ⇒ § 14 R5
 5. [Check-in] ditekan → /attendance/liveness
 6. LIVENESS on-device (§ 6). Hasil:
      passed  → frame dari sesi liveness dipakai sebagai foto (D29)
      failed / unsupported → sesuai context.liveness.policy (§ 6.5)
 7. Kompresi adaptif sesuai context.photo.* (§ 5.3)
 8. idempotencyKey = ulid(); simpan metadata percobaan (§ 2.7)
 9. POST #53 allow_fallback=false, + field liveness (§ 14 R4)
10. Percabangan — SAMA PERSIS dengan Fase 6 § 2.4:
      201 approved        → layar berhasil (jam server, lokasi)
      201 pending_review  → layar menunggu persetujuan
      422 FACE_NOT_MATCHED | FACE_NOT_USABLE | 502 | 504
                          → layar gagal + hints + [Coba lagi]
                            + [Kirim untuk ditinjau] HANYA bila error.can_fallback
                              → WAJIB liveness + foto BARU → POST allow_fallback=true
      409 ALREADY_CHECKED_IN → layar "sudah absen"
      422 FACE_NOT_ENROLLED  → ajakan enroll
      422 OUTSIDE_GEOFENCE   → layar di luar area + jarak
      403 CONSENT_REQUIRED   → layar consent
      429 TOO_MANY_FAILED_ATTEMPTS → layar jeda + hitung mundur Retry-After
      503 FACE_SERVICE_NOT_CONFIGURED
      503 ATTENDANCE_NOT_CONFIGURED → keduanya: "Sistem belum dikonfigurasi.
                                       Hubungi admin." (dua kode, satu akar
                                       masalah berbeda — K-04)
      timeout/koneksi        → § 2.7 retry dengan kunci yang sama
11. Sukses ⇒ hapus metadata percobaan, lepas byte foto dari memori
```

**Tidak ada tombol absen manual yang berdiri sendiri** — aturan
[Fase 6 § 2.4](07-Fase6.md#24-alur-check-in--check-out-d17) berlaku identik, dan
diuji dengan cara yang sama (§ 13.4).

### 5.3 Pipeline capture — dan bug cermin versi Flutter

[Fase 6 § 2.2](07-Fase6.md#22-pipeline-capture-kamera--dan-satu-kesalahan-yang-harus-dihindari)
menemukan bahwa mengirim frame tercermin menurunkan similarity **tanpa satu pun
error**. Di Flutter, bug yang sama punya dua jalur masuk tambahan yang lebih
berbahaya karena tidak terlihat di kode kita sendiri:

| Jalur | Risiko |
|---|---|
| `Transform.scale(scaleX: -1)` diterapkan ke widget yang salah | Sama dengan `ctx.scale(-1,1)` di web — hanya boleh menyentuh **preview**, tidak pernah byte |
| **Plugin `camera` sendiri** | Perilaku cermin kamera depan **berbeda antar platform dan antar versi plugin**. iOS (`AVCaptureConnection.isVideoMirrored`) dan Android (Camera2) tidak selalu sepakat, dan hasil `takePicture()` bisa berbeda dari yang tampak di preview |
| Frame `startImageStream` | Orientasi & cermin bisa berbeda lagi dari `takePicture()` — dan D29 memakai jalur ini |

**Karena itu mitigasinya harus empiris, bukan asumsi:**

1. Buat fixture uji dengan **penanda asimetris** (mis. karyawan memegang kartu
   bertanda di sisi kiri, atau target cetak).
2. Jalankan di **perangkat Android nyata dan iOS nyata**, untuk **kedua** jalur
   (`takePicture()` dan frame stream).
3. Periksa byte yang benar-benar dikirim (dari log request, bukan dari preview).
4. Bila salah satu platform mencerminkan, **unmirror satu kali di sumbernya**,
   di satu fungsi bernama `normalizeCapturedFrame()`, dengan komentar yang menjelaskan
   platform mana dan versi plugin mana.
5. Kunci temuan itu dengan **golden test** yang gagal bila perilakunya berubah
   setelah upgrade plugin.

Poin 5 bukan formalitas: upgrade minor `camera` bisa membalik perilaku ini, dan
gejalanya hanya "akurasi menurun" beberapa minggu kemudian.

**Orientasi & kompresi.** `takePicture()` menghasilkan JPEG dengan EXIF orientation;
frame stream menghasilkan YUV tanpa EXIF tapi dengan rotasi sensor. Server memang
menangani EXIF ([Fase 2 § 2.7a](03-Fase2.md#27-penanganan-gambar--dua-celah-yang-harus-ditutup)),
tapi setelah `flutter_image_compress` orientasi sudah **dibakar** ke piksel dan EXIF
hilang. Aturan: setelah kompresi, verifikasi wajah tegak — sekali, di test perangkat
nyata, per platform.

**Parameter capture** dibaca dari `#55 context.photo.*`, disesuaikan ke kamera perangkat:

```
target = min(context.photo.recommended_dimension_px ?? 1280,
             resolusi terbaik yang didukung perangkat)
kualitas awal 85 → turun 75 → 65 sampai ukuran ≤ context.photo.max_bytes
tetap terlalu besar ⇒ perkecil sisi terpanjang ke 960 lalu ulangi
```

Ponsel 108 MP yang mengirim foto 12 MB lewat jaringan seluler adalah pengalaman
buruk dan pemborosan; server pun akan memperkecilnya
([Fase 2 § 2.7d](03-Fase2.md#27-penanganan-gambar--dua-celah-yang-harus-ditutup)).
`recommended_dimension_px` adalah field baru yang diminta di § 14 R2.

### 5.4 Enrollment — sesi bertahap

Identik dengan [Fase 6 § 2.5](07-Fase6.md#25-enrollment--sesi-bertahap-d15), dengan
tiga perbedaan mobile:

1. `capture_source = 'mobile_camera'`.
2. **Liveness juga dijalankan saat enrollment.** Referensi wajah adalah dasar seluruh
   pencocokan berikutnya; membiarkan seseorang mendaftarkan foto dari layar berarti
   memasang wajah palsu sebagai kebenaran permanen. Kebijakan liveness saat enrollment
   mengikuti `context.liveness.policy` dengan satu perbedaan: bila `preferred` dan
   liveness gagal, foto **ditolak** (bukan diterima dengan tanda) — tidak ada jalur
   "pending review" untuk referensi.
3. Pratinjau memakai byte lokal di memori, tidak mengunduh ulang.

Panduan foto mengikuti **D26** ([Fase 6](07-Fase6.md#d26--panduan-variasi-foto--butuh-konfirmasi))
persis:

| Foto | Panduan |
|---|---|
| 1 | *"Hadap lurus ke kamera, ekspresi netral."* |
| 2 | *"Tetap hadap lurus, tersenyum tipis."* |
| 3 | *"Hadap lurus, ubah sedikit posisi atau pencahayaan."* |
| 4–5 | *"Bila biasa memakai kacamata, ambil satu dengan dan satu tanpa."* |

> **Jangan** menulis "hadap kiri / hadap kanan". Gate `face.max_abs_yaw` (default
> 0,35) akan menolaknya, dan pengguna akan menyimpulkan aplikasinya rusak. Ini juga
> alasan tantangan liveness **tidak** memakai gerakan menoleh sebagai tantangan
> terakhir sebelum capture (§ 6.2).

### 5.5 Menampilkan foto — padanan `<AuthImage>`

[Fase 4 D14](04-Fase3.md#22-d14--akses-foto-streaming-lewat-api-bukan-signed-url)
melarang signed URL; foto hanya bisa diambil dengan header `Authorization`.
Konsekuensinya di Flutter: **`Image.network` tidak boleh dipakai** untuk foto absensi
maupun referensi — ia tidak bisa membawa header (dan `headers:` yang ada tidak
melewati interceptor auth, tidak ikut refresh 401, dan tidak menangani 410).

```dart
class AuthImage extends ConsumerWidget {
  // 1. Dio.get(path, responseType: ResponseType.bytes)   → lewat AuthInterceptor
  // 2. Uint8List di-cache LRU (maks ~12 entri / ~8 MB) — perangkat kelas bawah
  //    punya RAM terbatas; daftar riwayat 50 baris tidak boleh menahan 50 foto
  // 3. Image.memory(bytes, gaplessPlayback: true)
  // 4. Status: loading (placeholder rasio tetap) / 401 (refresh sekali) /
  //    403 / 404 (empty tenang) / 410 ("foto dihapus sesuai kebijakan retensi")
}
```

Sama seperti [Fase 5 § 2.4](06-Fase5.md#24-komponen-authimage--konsekuensi-langsung-dari-d14):
**daftar riwayat tidak memuat foto**; foto hanya di layar detail.

### 5.6 Riwayat

#56/#57/#58 dengan **`AttendanceEmployeeDTO`** — tanpa `matched_similarity`,
`threshold_used`, `model_version`, `quality_score`, `distance_meter`,
`gps_accuracy_meter` ([Fase 4 § 2.7](05-Fase4.md#27-anti-penyalahgunaan)).

Di Dart ini ditegakkan oleh tipe: kelas `freezed` `AttendanceEmployeeDto` **tidak
memiliki** field tersebut, sehingga mengaksesnya adalah error kompilasi — padanan
dari penegakan `tsc` di [Fase 6 § 2.8](07-Fase6.md#28-riwayat-pribadi--dan-yang-sengaja-tidak-ada-di-sana).

`review_note` **ditampilkan** pada status `rejected` — karyawan berhak tahu alasannya.

---

## 6. Liveness — Desain Lengkap

### 6.1 Apa yang sedang dan tidak sedang diklaim

Bagian ini ditulis lebih dulu supaya tidak ada yang salah membaca sisanya.

**Yang ditutup liveness ini:**

| Serangan | Ditutup? | Bagaimana |
|---|---|---|
| Foto cetak | ✅ | Kertas tidak berkedip atau tersenyum |
| Foto statis di layar ponsel lain | ✅ | Sama |
| Foto dari galeri | ✅ | Tidak ada jalur galeri sama sekali (§ 7.3) |
| "Titip absen" dengan foto tersimpan | ✅ | Sama |
| Wajah orang lain masuk frame di tengah proses | ✅ | Kontinuitas `trackingId` (§ 6.3) |
| Liveness lolos lalu kamera diarahkan ke foto | ✅ | Frame yang dikirim **berasal dari sesi liveness** (D29) |

**Yang TIDAK ditutup — dan harus dinyatakan, bukan disembunyikan:**

| Serangan | Kenapa tidak tertutup |
|---|---|
| Video replay berkualitas tinggi di layar besar | Wajah dalam video berkedip dan tersenyum. Tantangan acak menaikkan biayanya (butuh operator yang memutar segmen yang tepat secara langsung), tapi tidak menutupnya |
| Deepfake real-time | ML Kit tidak mengukur tekstur, kedalaman, atau artefak sintetis |
| APK dimodifikasi / kamera virtual di perangkat di-root | Liveness berjalan di client; client yang dikuasai penyerang bisa berbohong (§ 6.6) |
| Masker cetak 3D | Di luar jangkauan deteksi berbasis landmark |

**Kesimpulan jujur: R3 ditutup untuk serangan oportunistik (yang mencakup hampir
seluruh kecurangan absensi di praktik), dan TIDAK ditutup untuk penyerang yang
termotivasi dan punya perkakas.** Penutupan sesungguhnya membutuhkan PAD
bersertifikat (FaceTec, iProov, Aware) atau sensor kedalaman (TrueDepth/ARCore) —
keduanya di luar scope dan berbiaya lisensi. Risiko sisa ini didokumentasikan di
`docs/mobile/liveness-threat-model.md` dan **tetap terbuka** sebagai R3-residual.

### 6.2 D28 — Mekanisme ⚠️ BUTUH KONFIRMASI

`google_mlkit_face_detection` dengan opsi:

```dart
FaceDetectorOptions(
  performanceMode: FaceDetectorMode.fast,   // stream realtime
  enableClassification: true,               // smilingProbability, eyeOpenProbability
  enableTracking: true,                     // trackingId — kunci anti-swap
  enableLandmarks: false,                   // tidak dibutuhkan
  enableContours: false,                    // mahal, tidak dibutuhkan
  minFaceSize: 0.15,
)
```

**Tantangan yang dipakai: kedip dan senyum — keduanya bisa dilakukan sambil
menghadap lurus.**

| Tantangan | Sinyal | Ambang indikatif (kalibrasi di lapangan) |
|---|---|---|
| Kedip | `leftEyeOpenProbability` & `rightEyeOpenProbability` turun < 0,25 lalu naik > 0,70 | dalam satu jendela ≤ 4 detik |
| Senyum | `smilingProbability` naik dari < 0,25 ke > 0,70 | dalam ≤ 4 detik |

**Menoleh sengaja TIDAK dipakai sebagai tantangan.** Alasannya sama dengan D26:
frame yang akhirnya dikirim harus lolos gate `max_abs_yaw` (0,35) dan
`max_abs_pitch` (0,30). Tantangan yang membuat kepala menoleh menciptakan risiko
frame terpilih diambil saat kepala belum kembali lurus, lalu ditolak `head_turned` —
pengguna melakukan persis yang diminta lalu ditolak, dan itu cara tercepat membuat
orang tidak percaya pada aplikasinya.

**Urutan dan jumlah tantangan diacak** per sesi (default 2 dari 2 jenis, urutan
acak, jeda acak 0,5–1,5 detik). Pengacakan adalah satu-satunya hal yang membedakan
ini dari "putar video rekaman": video yang direkam sebelumnya tidak tahu urutan
yang akan diminta.

### 6.3 Prasyarat yang harus terpenuhi sepanjang sesi

Diperiksa pada **setiap frame**; pelanggaran mengakhiri sesi dan mengulanginya:

| Prasyarat | Alasan |
|---|---|
| Tepat **satu** wajah | Lebih dari satu = orang lain di frame; sama dengan `multiple_faces` Fase 2 |
| **`trackingId` tidak berubah** dari awal sampai frame terpilih | Menutup "orang A melakukan liveness, orang B difoto" |
| `boundingBox.height / imageHeight` ≥ `min_face_ratio` context | Mencegah wajah terlalu kecil (penyebab `face_too_small` paling sering) |
| `|headEulerAngleY|` ≤ ambang, `|headEulerAngleX|` ≤ ambang | Frontal — supaya frame terpilih lolos gate server |
| Wajah terdeteksi terus-menerus (jeda hilang ≤ 500 ms) | Menutup pergantian sumber gambar di tengah proses |

Timeout total sesi: `context.liveness.timeout_seconds` (default 20). Percobaan
maksimum: `context.liveness.max_attempts` (default 3).

### 6.4 D29 — Frame yang dikirim berasal dari sesi liveness ⚠️ BUTUH KONFIRMASI

Dua pilihan:

| Opsi | Cara | Masalah |
|---|---|---|
| A. Liveness selesai → `takePicture()` | Kualitas JPEG terbaik dari plugin | **Ada jeda** antara "terbukti hidup" dan "foto diambil". Kamera bisa diarahkan ke foto dalam jeda itu. Pada beberapa versi plugin, `takePicture()` sambil `startImageStream` juga bermasalah di Android |
| **B. Frame dari stream liveness itu sendiri** (rekomendasi) | Simpan frame terbaik (skor kualitas tertinggi, frontal, `trackingId` sama) dari jendela ≤ 1 detik setelah tantangan terakhir, konversi YUV→JPEG | Butuh konversi YUV420/NV21 → JPEG (~100–300 ms untuk 720p); kualitas sedikit di bawah `takePicture()` |

**Rekomendasi: B.** Jeda pada opsi A adalah bypass yang sepele — liveness dilakukan
dengan wajah asli, lalu kamera diarahkan ke foto. Opsi B menghilangkan jendela itu
sepenuhnya: yang dikirim adalah frame dari orang yang barusan terbukti berkedip.

Resolusi stream (`ResolutionPreset.high` ≈ 720p) memadai — [Fase 6 D24](07-Fase6.md#d24--parameter-capture--butuh-konfirmasi)
memakai 1280×720 untuk web dan [Fase 2 § 2.8](03-Fase2.md#28-budget-performa)
mengukur pada ukuran itu.

**Cadangan:** bila di perangkat tertentu kualitas hasil konversi menyebabkan
`too_blurry` berulang, jatuh ke opsi A dengan syarat jeda liveness→capture **< 1
detik** dan `trackingId` tetap sama. Keputusan ini dicatat per-model perangkat di
`docs/mobile/device-matrix.md`, bukan dijadikan default diam-diam.

### 6.5 D30 — Perilaku saat liveness gagal atau tidak didukung ⚠️ BUTUH KONFIRMASI

**Keputusan ada di server, bukan di app** — supaya bisa diubah admin tanpa merilis
ulang aplikasi, konsisten dengan seluruh setting lain.

Setting baru `attendance.liveness_policy` (§ 14 R4), dibaca dari `#55 context`:

| Nilai | Liveness gagal / tidak didukung | Liveness berhasil |
|---|---|---|
| `off` | Diabaikan seluruhnya | Diabaikan |
| **`preferred`** (default) | Absensi tetap dikirim dengan `liveness_passed=false` → server memaksa `pending_review`, `fallback_reason='liveness_failed'` | Normal |
| `required` | Server menolak: `422 LIVENESS_REQUIRED` dengan `can_fallback` sesuai `attendance.fallback_enabled` | Normal |

**Rekomendasi default `preferred`.** Alasannya: `required` mengunci karyawan dengan
perangkat lama dari absensi sama sekali — dan orang yang tidak bisa absen akan
mencari jalan lain, biasanya dengan meminjam ponsel rekan, yang justru menghasilkan
kecurangan yang ingin dicegah. `preferred` menempatkan kasus itu di antrian manusia,
tempat ia bisa dinilai.

**Peringatan yang harus dinyatakan dalam desain, bukan disembunyikan:**
`liveness_passed` adalah **klaim client**, sama statusnya dengan `capture_source`
([Fase 4 E28](05-Fase4.md#64-keamanan--integritas)). Server tidak dapat
memverifikasinya. Karena itu:

- App yang **jujur** dan gagal liveness akan mendapat perlakuan yang benar
  (`pending_review`) — ini nilai utamanya.
- App yang **dimodifikasi** cukup mengirim `liveness_passed=true` dan mendapat
  perlakuan seperti tanpa liveness. Ia **tidak** mendapat lebih dari yang sudah
  bisa dilakukan lewat web hari ini.

Jadi liveness ini adalah **penghalang, bukan kontrol yang dapat diverifikasi**.
Menaikkannya menjadi kontrol yang dapat diverifikasi memerlukan attestation
(Play Integrity API / Apple App Attest) yang membuktikan aplikasi tidak dimodifikasi.
Itu **di luar scope Fase 7** dan dicatat sebagai opsi lanjutan di
`docs/mobile/liveness-threat-model.md`.

### 6.6 Perangkat yang tidak didukung

```
deviceSupported() =
     ada kamera depan
 AND Android SDK ≥ 24  |  iOS ≥ 13.0
 AND ML Kit face detector berhasil diinisialisasi
```

Gagal inisialisasi ML Kit (mis. Play Services tidak tersedia di perangkat Android
tanpa GMS) → aplikasi **tetap berfungsi**, tapi `liveness_supported=false` dikirim
ke server dan perlakuannya mengikuti `liveness_policy` (§ 6.5). Layar `/unsupported`
hanya untuk kasus benar-benar fatal (tanpa kamera depan, OS di bawah minimum).

### 6.7 Alur layar liveness

```
/attendance/liveness
 ├─ Preview penuh layar (dicerminkan secara visual saja — § 5.3)
 ├─ Oval panduan + instruksi "Posisikan wajah di dalam bingkai"
 ├─ Prasyarat terpenuhi (§ 6.3) selama 1 detik → mulai tantangan
 ├─ Tantangan 1 (acak): "Kedipkan mata"      [progres 1/2]  timeout 4 dtk
 ├─ Jeda acak 0,5–1,5 dtk
 ├─ Tantangan 2 (acak): "Tersenyum tipis"    [progres 2/2]  timeout 4 dtk
 ├─ "Hadap lurus, tahan sebentar" → pilih frame terbaik (D29)
 ├─ Berhasil → lanjut ke pengiriman (§ 5.2 langkah 7)
 └─ Gagal:
      percobaan < max_attempts ⇒ ulangi dengan tantangan acak BARU
      percobaan habis          ⇒ sesuai liveness_policy (§ 6.5)
```

Kalimat instruksi liveness **bukan** bagian dari kosakata `hints` Fase 2 — kosakata
itu tertutup dan milik server. Kalimat liveness masuk namespace terpisah di berkas
kanonik yang sama (§ 14 R6).

Umpan balik harus **seketika dan spesifik**: "Wajah terlalu jauh", "Wajah tidak
terlihat", "Terdeteksi lebih dari satu wajah", "Terlalu gelap". Liveness yang hanya
menampilkan "Gagal, coba lagi" akan gagal terus tanpa pengguna tahu sebabnya.

---

## 7. Permission & Privacy

### 7.1 Izin yang diminta — dan yang sengaja tidak

| Izin | Platform | Wajib? | Alasan |
|---|---|---|---|
| Kamera | keduanya | ✅ | Verifikasi wajah |
| Lokasi saat digunakan | keduanya | ✅ (bila `geofence_enabled`) | Geofence |
| Internet | Android | ✅ | — |
| **Lokasi latar belakang** | — | ❌ **TIDAK diminta** | Tidak dibutuhkan; memicu review Play yang berat dan menaikkan kekhawatiran privasi tanpa manfaat |
| **Akses galeri / media** | — | ❌ **TIDAK diminta** | Tidak ada jalur galeri (§ 7.3). Ketiadaannya di manifest adalah **bukti** aturan master plan § 9 ditegakkan |
| Notifikasi | keduanya | ❌ (opsional, di luar scope) | — |
| Penyimpanan | — | ❌ | Foto tidak pernah ditulis ke penyimpanan bersama |

### 7.2 String izin

**iOS — `Info.plist`:**

```xml
<key>NSCameraUsageDescription</key>
<string>Faceclock menggunakan kamera untuk memverifikasi wajah Anda saat melakukan
absensi. Foto dikirim ke server perusahaan dan tidak dibagikan ke pihak lain.</string>

<key>NSLocationWhenInUseUsageDescription</key>
<string>Faceclock menggunakan lokasi Anda hanya saat melakukan absensi, untuk
memastikan Anda berada di area kantor yang telah ditentukan.</string>
```

**Android — `AndroidManifest.xml`:**

```xml
<uses-permission android:name="android.permission.CAMERA" />
<uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
<uses-permission android:name="android.permission.ACCESS_COARSE_LOCATION" />
<uses-permission android:name="android.permission.INTERNET" />
<uses-feature android:name="android.hardware.camera.front" android:required="true" />
```

Penjelasan sebenarnya ditampilkan **sebelum** dialog sistem (priming screen), bukan
hanya di string sistem. Dialog izin yang muncul tanpa konteks lebih sering ditolak,
dan penolakan permanen di iOS hanya bisa dipulihkan lewat Pengaturan.

### 7.3 Tidak ada jalur galeri — penegakan master plan § 9

Master plan § 9 mensyaratkan capture langsung dari kamera. Di mobile ini ditegakkan
**tiga lapis**:

1. Tidak ada `image_picker` di `pubspec.yaml`.
2. Tidak ada izin galeri di manifest/plist.
3. Test build yang memindai manifest APK/IPA hasil build untuk memastikan izin
   media tidak muncul (§ 13.4) — termasuk yang mungkin ditarik diam-diam oleh
   dependensi transitif.

Lapis ketiga yang menangkap kasus nyata: satu paket pihak ketiga yang menambahkan
`READ_MEDIA_IMAGES` lewat manifest merge, tanpa satu baris pun kode kita berubah.

### 7.4 Data biometrik di perangkat (UU PDP)

| Data | Di perangkat | Retensi |
|---|---|---|
| Byte foto wajah | Memori saja | Dilepas segera setelah kirim berhasil / dibatalkan |
| Berkas sementara kamera (`XFile`) | Sesaat | **Dihapus segera** setelah dibaca — lihat catatan di bawah |
| Frame stream liveness | Memori saja | Tidak pernah ditulis |
| Embedding | Tidak pernah ada di perangkat | Server-only |
| Foto yang ditampilkan (riwayat/referensi) | Cache memori LRU | Dibersihkan saat logout & saat app ke background lama |
| Metadata percobaan tertunda | Penyimpanan biasa | ≤ 15 menit, tanpa foto/koordinat |

**Catatan berkas sementara kamera.** `camera.takePicture()` **menulis JPEG ke berkas
temporer di disk** dan mengembalikan `XFile` yang menunjuk ke sana. Ini persis kelas
masalah yang ditemukan prior art di
[Fase 2 § 2.1](03-Fase2.md#21-d12--memanfaatkan-face-engine-yang-sudah-ada--butuh-konfirmasi)
(Starlette menumpahkan frame biometrik ke temp file tanpa error dan tanpa log).
Mitigasi wajib:

```dart
final xfile = await controller.takePicture();
final bytes = await xfile.readAsBytes();
await File(xfile.path).delete();        // segera, dalam try/finally
```

Dengan D29 (frame dari stream), jalur ini bahkan tidak terpakai pada alur utama —
tapi ia tetap ada pada jalur cadangan, dan penghapusannya diuji (§ 13.5).

**Logout** menghapus: refresh token dari secure storage, seluruh cache foto,
metadata percobaan, dan cache DTO.

---

## 8. Edge Case & Validasi

### 8.1 Sesi & jaringan

| # | Kondisi | Penanganan |
|---|---|---|
| E1 | Banyak request `401` bersamaan | `QueuedInterceptor` + `Completer` (§ 2.5a). **Tepat satu** `/auth/refresh` |
| E2 | Response refresh hilang di jaringan | Grace window server (§ 2.5b / § 14 R3). Client: maks 1 pengulangan, jeda 2 dtk, hanya untuk kegagalan transport |
| E3 | `REFRESH_TOKEN_REUSED` sungguhan | Hapus token, ke `/login` dengan *"Sesi berakhir. Silakan login kembali."* — bukan "Terjadi kesalahan" |
| E4 | Jaringan hilang saat mengirim absensi | Retry dengan kunci sama (§ 2.7). Pesan: *"Koneksi terputus. Absensi mungkin sudah tercatat."* |
| E5 | App di-kill saat mengirim | Saat dibuka lagi ≤ 15 menit: tawarkan *"Lanjutkan pengiriman absensi tadi?"* dengan kunci sama |
| E6 | Tidak ada koneksi sebelum mengirim | Tolak jujur; foto tidak disimpan (§ 2.7) |
| E7 | Jam perangkat salah | Tidak berpengaruh. Semua waktu dari `context.server_time` |
| E8 | Perangkat berpindah zona waktu | `work_date` selalu memakai `attendance.timezone` organisasi ([Fase 4 E8](05-Fase4.md#61-waktu--aturan-harian)) |
| E9 | App lama, backend sudah berubah | Boot memeriksa `/api/v1/version`; ketidakcocokan mayor → layar "Perbarui aplikasi" (§ 14 R11) |

### 8.2 Kamera & liveness

| # | Kondisi | Penanganan |
|---|---|---|
| E10 | Izin kamera ditolak | Layar penjelasan + tombol ke Pengaturan sistem. Bukan blank |
| E11 | Izin ditolak permanen (iOS "Don't Allow") | Hanya bisa dipulihkan lewat Pengaturan; app mengatakannya, tidak memunculkan dialog yang tidak akan pernah muncul lagi |
| E12 | Kamera dipakai app lain | Pesan spesifik + coba lagi |
| E13 | ML Kit gagal inisialisasi | `liveness_supported=false`; perlakuan sesuai `liveness_policy` (§ 6.6) |
| E14 | Liveness gagal 3× | Sesuai `liveness_policy`; layar menjelaskan pilihan berikutnya |
| E15 | Wajah hilang di tengah tantangan | Sesi liveness diulang dengan tantangan baru, bukan dilanjutkan |
| E16 | `trackingId` berubah | Sesi **dibatalkan** — ini indikasi pergantian wajah (§ 6.3) |
| E17 | Terlalu gelap | Peringatan seketika dari luminansi frame (setara [Fase 6 D25](07-Fase6.md#d25--pra-pemeriksaan-di-browser--butuh-konfirmasi)) — **saran**, bukan blokir. Server tetap otoritas |
| E18 | App ke background saat kamera hidup | `WidgetsBindingObserver` → hentikan stream & `dispose()` controller; saat kembali, mulai ulang sesi dari awal |
| E19 | Frame terkirim tercermin | Dicegah + diuji per platform (§ 5.3, § 13.3) |
| E20 | Rotasi perangkat | Layar liveness dikunci portrait; menghindari kombinasi orientasi yang tidak diuji |

### 8.3 Lokasi

| # | Kondisi | Penanganan |
|---|---|---|
| E21 | Izin lokasi ditolak | Bila `missing_location_policy = reject`: blokir + panduan. Bila `pending_review`: izinkan dengan peringatan |
| E22 | GPS mati | Ajakan menyalakan + `Geolocator.openLocationSettings()` |
| E23 | Akurasi buruk | Sesuai `context.geofence.max_gps_accuracy_meter` (§ 5.2 langkah 4) |
| E24 | Timeout perolehan lokasi | 10 detik → [Ambil ulang lokasi]; jangan mengirim koordinat kosong |
| E25 | **Mock location terdeteksi** (Android) | `Position.isMocked` → dikirim ke server; server yang memutuskan (§ 14 R5). App **tidak** memblokir sendiri |
| E26 | iOS — tidak ada deteksi mock | Dinyatakan sebagai keterbatasan platform di `docs/mobile/limitations.md` |
| E27 | Di luar radius | Jarak indikatif ditampilkan sebelum kirim; blokir bila `outside_policy = reject` |

### 8.4 Alur bisnis

| # | Kondisi | Penanganan |
|---|---|---|
| E28 | Sudah check-in hari ini | Dari `context.today` → layar "sudah check-in" + tombol check-out |
| E29 | Dua perangkat menekan kirim bersamaan | Satu `201`, satu `409 ALREADY_CHECKED_IN` → layar "sudah absen", bukan error |
| E30 | Sesi enrollment kedaluwarsa | `409 ENROLLMENT_SESSION_EXPIRED` → tawarkan mulai ulang, katakan foto sebelumnya hilang |
| E31 | Model berganti di tengah sesi | `409 ENROLLMENT_MODEL_CHANGED` → mulai ulang |
| E32 | Reindex berjalan | `409 REINDEX_IN_PROGRESS` → pesan pemeliharaan |
| E33 | Wajah sudah terdaftar di karyawan lain | `409 FACE_BELONGS_TO_ANOTHER_EMPLOYEE` → pesan **tanpa** menyebut siapa ([Fase 3 § 2.6](04-Fase3.md#26-d16--deteksi-wajah-duplikat-antar-karyawan--butuh-konfirmasi)) |
| E34 | Consent dicabut dari perangkat lain | `403 CONSENT_REQUIRED` → layar consent |
| E35 | `404` karena resource milik orang lain | Empty state tenang, **bukan** error ([Fase 1 § 5.4](02-Fase1.md#54-pola-akses-self)) |

### 8.5 Tujuh keadaan layar `/attendance` — port lengkap dari Fase 6

Seluruh tujuh keadaan [Fase 6 § 6.6](07-Fase6.md#66-keadaan-kosong--jujur) diport,
tidak sebagian:

| Keadaan | Kalimat |
|---|---|
| Mode manual | *"Akun Anda diatur pada mode absensi manual. Foto Anda disimpan sebagai bukti dan absensi akan ditinjau admin. Tidak ada pencocokan wajah otomatis."* |
| Belum enroll | *"Anda belum mendaftarkan wajah. Daftarkan wajah dulu agar bisa absen."* + [Daftarkan wajah] |
| Perlu enroll ulang | *"Sistem pengenalan wajah telah diperbarui. Silakan daftarkan ulang wajah Anda."* |
| Consent belum ada | *"Diperlukan persetujuan pemrosesan data biometrik sebelum Anda dapat absen."* |
| Consent dicabut | *"Anda telah mencabut persetujuan. Absensi wajah tidak tersedia. Hubungi HR untuk jalur absensi alternatif."* |
| Sudah lengkap hari ini | *"Absensi hari ini sudah lengkap."* + ringkasan jam & durasi |
| Belum dikonfigurasi (`503`) | *"Sistem absensi belum dikonfigurasi. Hubungi admin."* |

Ditambah dua keadaan khas mobile:

| Keadaan | Kalimat |
|---|---|
| Tidak ada koneksi | *"Tidak ada koneksi. Absensi memerlukan koneksi ke server."* + [Coba lagi] |
| Liveness tidak didukung perangkat | *"Perangkat ini tidak mendukung pemeriksaan wajah langsung. Absensi Anda akan ditinjau admin."* (bila `policy=preferred`) |

Tidak satu pun berbunyi "Terjadi kesalahan".

---

## 9. Struktur Folder

```
apps/faceclock-mobile/                       # monorepo (D1 terkunci 2026-09-04)
├── lib/
│   ├── main.dart
│   ├── app.dart                             # MaterialApp.router + tema
│   ├── core/
│   │   ├── config/
│   │   │   ├── env.dart                     # API_BASE_URL per flavor
│   │   │   └── flavors.dart                 # dev | staging | prod
│   │   ├── network/
│   │   │   ├── dio_client.dart
│   │   │   ├── auth_interceptor.dart        # QueuedInterceptor + Completer (§ 2.5a)
│   │   │   ├── token_store.dart             # secure storage + memori (§ 2.4)
│   │   │   ├── api_error.dart               # envelope {error} → ApiError
│   │   │   ├── error_codes.dart             # 39 + kode baru (§ 14 R4)
│   │   │   └── idempotency.dart             # § 2.6
│   │   ├── messages/
│   │   │   ├── hints.dart                   # DIBACA dari assets/contracts/hints.json
│   │   │   ├── error_messages.dart          # dari assets/contracts/error-messages.json
│   │   │   └── liveness_messages.dart       # namespace terpisah (§ 14 R6)
│   │   ├── datetime/local_time.dart         # UTC → attendance.timezone
│   │   ├── permissions/permission_service.dart
│   │   └── device/device_support.dart       # § 6.6
│   ├── data/
│   │   ├── dto/                             # freezed + json_serializable, snake_case
│   │   │   ├── auth_dto.dart  employee_dto.dart
│   │   │   ├── attendance_employee_dto.dart # TANPA field skor (§ 5.6)
│   │   │   ├── attendance_context_dto.dart
│   │   │   ├── enrollment_dto.dart  consent_dto.dart
│   │   └── repositories/
│   │       ├── auth_repository.dart         # #1–#6
│   │       ├── attendance_repository.dart   # #53–#59
│   │       ├── enrollment_repository.dart   # #38–#44, #48
│   │       ├── consent_repository.dart      # #32–#35
│   │       └── profile_repository.dart      # #9, #28
│   ├── features/
│   │   ├── auth/{login_screen,change_password_screen,auth_provider}.dart
│   │   ├── attendance/
│   │   │   ├── attendance_screen.dart       # 9 keadaan (§ 8.5)
│   │   │   ├── liveness_screen.dart
│   │   │   ├── result_screen.dart           # berhasil / menunggu / gagal
│   │   │   ├── fallback_prompt.dart         # HANYA dari layar gagal (D17)
│   │   │   ├── check_in_controller.dart     # mesin keadaan § 5.2
│   │   │   └── widgets/{server_clock,location_status,today_card}.dart
│   │   ├── enrollment/
│   │   │   ├── enrollment_screen.dart       # wizard, langkah dari #39
│   │   │   ├── photo_slot.dart              # kosong/proses/diterima/ditolak+hints
│   │   │   ├── photo_guidance.dart          # D26
│   │   │   └── session_countdown.dart
│   │   ├── consent/{consent_screen,withdraw_dialog}.dart
│   │   ├── history/{history_screen,history_detail_screen}.dart
│   │   └── profile/{profile_screen,app_settings_screen}.dart
│   ├── liveness/                            # ← inti Fase 7
│   │   ├── liveness_engine.dart             # mesin keadaan tantangan (§ 6.7)
│   │   ├── challenge.dart                   # blink | smile, pengacakan
│   │   ├── face_gate.dart                   # prasyarat per-frame (§ 6.3)
│   │   ├── frame_selector.dart              # pilih frame terbaik (D29)
│   │   ├── yuv_to_jpeg.dart                 # konversi + orientasi
│   │   └── liveness_result.dart
│   ├── camera/
│   │   ├── camera_controller_provider.dart
│   │   ├── capture_pipeline.dart            # kompresi adaptif (§ 5.3)
│   │   ├── normalize_frame.dart             # ← titik tunggal penanganan cermin
│   │   └── camera_preview_mirrored.dart     # cermin HANYA visual
│   └── ui/
│       ├── auth_image.dart                  # padanan <AuthImage> (§ 5.5)
│       ├── empty_state.dart  error_state.dart  loading_state.dart
│       └── confirm_dialog.dart
├── assets/
│   └── contracts/                           # DISALIN dari docs/api/ saat build
│       ├── hints.json
│       └── error-messages.json
├── test/                                    # unit + widget
│   ├── network/auth_interceptor_test.dart   # single-flight
│   ├── liveness/{challenge_test,face_gate_test,frame_selector_test}.dart
│   ├── camera/normalize_frame_test.dart
│   └── dto/attendance_employee_dto_test.dart
├── integration_test/
│   ├── checkin_flow_test.dart
│   ├── enrollment_flow_test.dart
│   └── permission_denied_test.dart
├── android/  ios/
├── pubspec.yaml
└── README.md
```

Dua catatan:

- **`assets/contracts/` disalin, bukan ditulis ulang.** Skrip build menyalin
  `docs/api/hints.json` dan `docs/api/error-messages.json` dari repo ke asset
  bundel, dan CI **gagal** bila keduanya berbeda. Ini yang mencegah kalimat mobile
  menyimpang dari web (§ 2.3). Penyalinan langsung ini dimungkinkan karena D1
  terkunci **monorepo** (2026-09-04): berkas sumber dan aplikasi mobile berada di
  satu repo, jadi tidak perlu distribusi paket berversi.
- **`camera/normalize_frame.dart` adalah satu-satunya tempat** yang boleh menyentuh
  orientasi/cermin byte. Semua jalur capture melewatinya.

---

## 10. Checklist Task

### 10.0 Prasyarat
- [ ] ⛔ **Fase 0–6 sudah dieksekusi dan DoD-nya terbukti** (§ 0)
- [ ] **Konfirmasi D27–D33** (§ 2.0)
- [ ] **Sepakati 11 revisi kontrak** (§ 14) — beberapa mengubah migration Fase 1/4
- [ ] `API_BASE_URL` HTTPS tersedia untuk dev/staging/prod (§ 2.2)
- [ ] Perangkat uji fisik tersedia (§ 13.1)

### 10.1 Fondasi
- [ ] Proyek Flutter + flavors (dev/staging/prod) + `env.dart`
- [ ] `dio_client` + `auth_interceptor` (`QueuedInterceptor` + `Completer`)
- [ ] Test single-flight: 6 request `401` bersamaan → **tepat satu** refresh
- [ ] `token_store` — access di memori, refresh di `flutter_secure_storage` dengan
      `ThisDeviceOnly`
- [ ] `api_error` + envelope `{data}`/`{error}` + `can_fallback` + `hints` + `request_id`
- [ ] Salin `hints.json` & `error-messages.json` ke asset + cek kesamaan di CI
- [ ] `local_time` — konversi ke `attendance.timezone`, bukan zona perangkat
- [ ] `device_support` + layar `/unsupported`
- [ ] `go_router` + guard (§ 3.2)
- [ ] DTO `freezed` — `AttendanceEmployeeDto` **tanpa** field skor
- [ ] `User-Agent` bermakna (`Faceclock-Mobile/x.y.z (Android 14; SM-G991B)`)
- [ ] Cert pinning (atau keputusan tertulis untuk tidak memakainya)

### 10.2 Izin
- [ ] String `Info.plist` + `AndroidManifest.xml` (§ 7.2)
- [ ] Priming screen sebelum dialog sistem
- [ ] Alur ditolak / ditolak permanen → panduan ke Pengaturan
- [ ] Verifikasi: **tidak ada** izin galeri/media di APK & IPA hasil build
- [ ] Verifikasi: **tidak ada** izin lokasi latar belakang

### 10.3 Kamera & liveness
- [ ] `camera_preview_mirrored` — cermin **hanya** visual
- [ ] `normalize_frame` — titik tunggal, dengan hasil uji per platform
- [ ] **Golden test cermin** di Android & iOS nyata (§ 13.3)
- [ ] `yuv_to_jpeg` + verifikasi orientasi setelah kompresi
- [ ] `capture_pipeline` — kompresi adaptif dari `context.photo.*`
- [ ] `face_gate` — 5 prasyarat per-frame (§ 6.3), termasuk kontinuitas `trackingId`
- [ ] `challenge` — kedip & senyum, urutan + jeda acak
- [ ] `liveness_engine` — mesin keadaan, timeout, `max_attempts`
- [ ] `frame_selector` — frame terbaik dari jendela ≤ 1 dtk (D29)
- [ ] Umpan balik seketika & spesifik (§ 6.7)
- [ ] Peringatan gelap (saran, bukan blokir)
- [ ] Hapus `XFile` temp segera di jalur cadangan (§ 7.4)
- [ ] `WidgetsBindingObserver` → hentikan stream saat background

### 10.4 Absensi
- [ ] `attendance_screen` — **9 keadaan** (§ 8.5)
- [ ] `server_clock` dari `context.server_time`
- [ ] `location_status` + jarak indikatif + `isMocked`
- [ ] `check_in_controller` — mesin keadaan § 5.2
- [ ] Kirim #53/#54 + `Idempotency-Key` + field liveness
- [ ] `result_screen` — berhasil / menunggu / gagal
- [ ] `fallback_prompt` — **hanya** bila `error.can_fallback`, **wajib** liveness+foto baru
- [ ] **Verifikasi: tidak ada tombol absen manual berdiri sendiri**
- [ ] Mode manual — satu kali kirim, tanpa liveness, pesan berbeda
- [ ] Check-out: baca `require_face_for_checkout`; `CHECKOUT_TOO_SOON` dengan sisa menit
- [ ] Retry dengan kunci sama + pemulihan setelah app di-kill (E5)

### 10.5 Enrollment & consent
- [ ] Wizard — langkah dari #39, bukan state lokal
- [ ] Liveness saat enrollment (§ 5.4 poin 2)
- [ ] `photo_slot` 4 keadaan + `hints`
- [ ] `photo_guidance` sesuai D26 — **bukan** hadap kiri/kanan
- [ ] `session_countdown` + peringatan 5 menit
- [ ] Lanjutkan sesi dari `409` + `existing_session_id`
- [ ] Tangani 6 error enrollment (E30–E33)
- [ ] `consent_screen` — Markdown + gate scroll + checkbox eksplisit
- [ ] Cabut consent + dialog akibat + arahan ke HR

### 10.6 Riwayat & profil
- [ ] `history_screen` (#56) + filter + `AttendanceEmployeeDto`
- [ ] `history_detail` (#58) + `auth_image` (#59) + `410`
- [ ] `review_note` pada status `rejected`
- [ ] `profile_screen` (#9, #33, #44) + ganti password (#6) + logout-all (#4)
- [ ] Logout menghapus secure storage + seluruh cache
- [ ] Verifikasi: nol field skor di seluruh app

### 10.7 Build & rilis
- [ ] Android `minSdk 24`, `targetSdk` terbaru, `usesCleartextTraffic=false`
- [ ] iOS deployment target `13.0`, ATS ketat
- [ ] Build AAB + IPA; ukuran app dicatat (ML Kit menambah ~20 MB; pertimbangkan
      model ML Kit tanpa bundel di Android)
- [ ] Halaman kebijakan privasi publik (URL wajib kedua store)
- [ ] Deklarasi data: Play Data Safety (Foto, Lokasi presisi) + Apple Privacy Nutrition Label
- [ ] Jalur distribusi sesuai D32
- [ ] `docs/mobile/liveness-threat-model.md`, `device-matrix.md`, `limitations.md`
- [ ] `DONE-Fase-7.md` sesuai Protokol Handoff master plan § 10.4

---

## 11. Dependencies

| Dari | Yang dibutuhkan |
|---|---|
| **Fase 0–6 dieksekusi** | ⛔ Blocker mutlak (§ 0) |
| Fase 0 | Envelope `{data}`/`{error}`, katalog error, konvensi `snake_case` |
| Fase 1 | #1–#6, #9, #28; `permissions` di `/auth/me`; rotasi refresh + deteksi reuse (yang memaksa § 2.5); pola ownership → 404 |
| Fase 2 | Kosakata `hints` + kalimat Indonesia; `face.max_image_bytes`; gate kualitas sebagai otoritas |
| Fase 3 | #32–#35, #38–#44, #48; sesi bertahap; aturan pesan duplikat |
| Fase 4 | #53–#59; `context` (#55); D17; `can_fallback`; `Idempotency-Key`; `AttendanceEmployeeDTO` |
| Fase 5 | Pola `<AuthImage>`; pemetaan error; pelajaran single-flight refresh; panel approval yang akan menampilkan info liveness (§ 14 R8) |
| Fase 6 | **Rujukan perilaku**: alur fallback, 7 keadaan layar, D24/D25/D26, bug cermin |
| Infrastruktur | HTTPS; perangkat uji fisik; akun Play Console & Apple Developer |

**Status risiko warisan:**

| Risiko | Status setelah Fase 7 |
|---|---|
| **R3** — spoofing | 🟡 **Ditutup sebagian.** Foto cetak, foto statis di layar, galeri, dan swap saat capture ditutup. Video replay berkualitas, deepfake, dan APK dimodifikasi **tetap terbuka** (§ 6.1). Disebut R3-residual |
| **R1** — lisensi model InsightFace | ⛔ **Tetap terbuka.** Fase 7 tidak menyentuh model sama sekali. Blocker rilis produksi |
| **R2** — kalibrasi Dataset B | ⛔ **Tetap terbuka.** Blocker rilis produksi. Fase 7 justru memperbesar taruhannya: absensi lapangan berarti variasi pencahayaan jauh lebih besar daripada kantor, sehingga threshold yang belum divalidasi akan lebih sering salah |
| R4, R5, R7 | ✅ Selesai (Fase 3/4) |
| **R6** — monorepo vs multi-repo | ✅ **Ditutup** — dikunci monorepo 2026-09-04 ([Fase 0 § 2.1](01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)). Berkas kontrak bersama (§ 9) memakai keuntungannya langsung |

---

## 12. Definition of Done

1. ⛔ **Fase 0–6 selesai dan DoD-nya terbukti** — prasyarat, bukan bagian DoD ini.
2. Sebelas revisi kontrak (§ 14) disepakati dan diimplementasikan di backend.
3. **Uji single-flight refresh:** 6 request gagal `401` bersamaan → tepat **satu**
   `/auth/refresh`, tidak ada `REFRESH_TOKEN_REUSED`, semua berhasil setelah retry.
4. **Uji grace window:** response refresh dibuang (proxy uji), app mengulang dengan
   token lama → server mengembalikan pasangan token yang sama, sesi **tidak** dicabut.
5. Refresh token tersimpan di Keychain/Keystore — diverifikasi dengan memeriksa
   bahwa ia **tidak** muncul di `SharedPreferences`/`NSUserDefaults` polos, dan
   **tidak** ikut backup perangkat.
6. **Frame terkirim tidak tercermin**, diverifikasi pada **Android nyata dan iOS
   nyata**, untuk jalur stream (D29) dan jalur cadangan.
7. Wajah pada foto terkirim **tegak** setelah kompresi, pada kedua platform.
8. Liveness: tantangan acak berjalan; foto cetak dan foto statis di layar **gagal**
   liveness pada uji nyata; video replay dicatat sebagai **berhasil lolos**
   (dinyatakan, bukan disembunyikan).
9. Kontinuitas `trackingId` terbukti: mengganti wajah di tengah sesi membatalkan sesi.
10. Frame yang dikirim berasal dari sesi liveness yang sama (D29) — dibuktikan dengan
    menandai frame dan memeriksa byte yang terkirim.
11. `liveness_policy` bekerja untuk ketiga nilai (`off`/`preferred`/`required`),
    **diubah dari panel admin web tanpa merilis ulang aplikasi**.
12. Perangkat tanpa ML Kit tetap bisa absen (`policy=preferred`) dan hasilnya
    `pending_review` dengan `fallback_reason='liveness_failed'`.
13. Check-in berhasil: `201 approved`, layar menampilkan **jam server**, bukan jam perangkat.
14. Wajah tidak cocok → layar gagal + `hints`; [Kirim untuk ditinjau] muncul **hanya**
    bila `can_fallback`; menekannya **mengulang liveness dan mengambil foto baru**.
15. Inference dimatikan → tidak ada jalur yang menghasilkan `approved`.
16. **Tidak ada tombol absen manual berdiri sendiri** — diuji.
17. **Tidak ada izin galeri/media** di APK dan IPA hasil build — diuji dengan
    memindai manifest hasil merge.
18. **Tidak ada izin lokasi latar belakang**.
19. Sembilan keadaan layar `/attendance` (§ 8.5) tampil benar; tidak satu pun
    berbunyi "Terjadi kesalahan".
20. Tanpa koneksi → pesan jujur, **foto tidak disimpan ke penyimpanan**.
21. Retry setelah koneksi putus memakai `Idempotency-Key` yang sama → **satu** record.
22. Pemulihan setelah app di-kill (≤ 15 menit) menawarkan lanjut dengan kunci sama.
23. **Byte foto tidak pernah persisten**: setelah 20 siklus absensi, tidak ada berkas
    gambar tersisa di direktori app — diuji (§ 13.5).
24. `XFile` temporer dihapus setelah dibaca — diuji.
25. Riwayat pribadi **tidak memuat** field skor apa pun — diuji pada DOM widget dan
    pada body response.
26. `404` pola ownership dirender sebagai empty state tenang.
27. Logout menghapus secure storage, cache foto, dan metadata percobaan.
28. Semua ambang berasal dari `#55 context` — diuji dengan mengubah setting di web
    lalu memverifikasi perilaku app berubah **tanpa rilis ulang**.
29. Berkas `hints.json` dan `error-messages.json` di asset **identik** dengan yang di
    `docs/api/` — dijaga CI.
30. Diuji pada matriks perangkat nyata (§ 13.1), **bukan hanya emulator**.
31. Halaman kebijakan privasi terbit; deklarasi data store terisi.
32. `docs/mobile/liveness-threat-model.md` memuat batasan § 6.1 apa adanya.
33. `DONE-Fase-7.md` ada, memuat keputusan D27–D33 final.

---

## 13. Cara Test / Verifikasi

### 13.1 Matriks perangkat nyata — emulator tidak cukup

Emulator memberi kamera sintetis dan GPS sempurna. Keduanya **tidak** merepresentasikan
kondisi lapangan, dan justru dua hal itulah yang paling sering gagal.

| Kelas | Contoh | Yang diuji |
|---|---|---|
| Android kelas bawah | RAM 3 GB, Android 10 | Performa liveness, memori `AuthImage`, waktu konversi YUV→JPEG |
| Android kelas menengah | Android 13–14 | Baseline |
| Android tanpa GMS (bila relevan) | — | ML Kit gagal init → `liveness_supported=false` |
| iPhone lama | iPhone SE, iOS 15 | Perilaku cermin kamera depan, ATS |
| iPhone baru | iOS 17+ | Baseline; izin lokasi presisi |

Kondisi lapangan yang wajib diuji, bukan disimulasikan:
- Cahaya sore membelakangi jendela (backlight) — penyebab `too_dark` paling sering
- Di dalam gedung bertingkat — akurasi GPS 50–200 m
- Sinyal seluler 1 bar — timeout dan retry
- Berjalan sambil absen — stabilitas `trackingId`

### 13.2 Uji single-flight & grace window

```dart
test('hanya satu refresh untuk banyak 401 bersamaan', () async {
  var refreshCalls = 0;
  mockAdapter.onPost('/auth/refresh', (_) { refreshCalls++; return tokensResponse; });
  mockAdapter.on401Once(['/attendances/context', '/auth/me', '/attendances/me', ...6]);

  await Future.wait([...6 permintaan...]);
  expect(refreshCalls, 1);            // BUKAN 6
});

test('response refresh hilang → tidak mencabut sesi', () async {
  // proxy uji: server MEROTASI tapi response dibuang
  // app mengulang dengan token lama dalam < 30 dtk
  // harapan: pasangan token yang sama dikembalikan, sesi hidup  (§ 2.5b)
});
```

### 13.3 Uji cermin — pada perangkat nyata, per platform

```
1. Cetak target dengan penanda asimetris (kotak hitam di sisi KIRI subjek)
2. Jalankan alur check-in di Android nyata
3. Intersep body request (proxy debug / log ukuran+hash, bukan foto di log)
4. Decode JPEG terkirim → periksa penanda ada di sisi KIRI
5. Ulangi di iOS nyata
6. Ulangi untuk jalur cadangan takePicture()
7. Simpan hasil sebagai golden; CI menggagalkan build bila plugin di-upgrade dan
   golden tidak diperbarui
```

Ditambah pemeriksaan statis: `grep -rn "scaleX: -1\|Transform.scale" lib/` — hanya
boleh muncul di `camera_preview_mirrored.dart`.

### 13.4 Uji penegakan aturan master plan § 9

```bash
# 1. Tidak ada izin galeri/media di APK hasil build (setelah manifest merge)
$ANDROID_HOME/build-tools/*/aapt2 dump permissions build/app/outputs/bundle/release/app.aab \
  | grep -iE 'READ_MEDIA|READ_EXTERNAL_STORAGE|WRITE_EXTERNAL_STORAGE'
# harus KOSONG

# 2. Tidak ada lokasi latar belakang
... | grep -i 'ACCESS_BACKGROUND_LOCATION'      # harus KOSONG

# 3. iOS: tidak ada NSPhotoLibraryUsageDescription
/usr/libexec/PlistBuddy -c "Print" Runner.app/Info.plist | grep -i PhotoLibrary
# harus KOSONG

# 4. Tidak ada image_picker di dependency tree
flutter pub deps --style=compact | grep -i image_picker      # harus KOSONG
```

```dart
testWidgets('tidak ada jalur absen manual berdiri sendiri', (t) async {
  await t.pumpWidget(app(state: ready));
  expect(find.textContaining(RegExp('manual|tinjau|review', caseSensitive: false)),
         findsNothing);          // tombol fallback baru ada SETELAH kegagalan
});
```

### 13.5 Uji tidak ada foto persisten

```dart
testWidgets('byte foto tidak pernah ditulis ke penyimpanan', (t) async {
  final dir = await getApplicationDocumentsDirectory();
  final before = listAllFiles(dir) + listAllFiles(await getTemporaryDirectory());

  for (var i = 0; i < 20; i++) { await runCheckInFlow(t); }

  final after = listAllFiles(dir) + listAllFiles(await getTemporaryDirectory());
  final baru = after.difference(before).where(isImageFile);
  expect(baru, isEmpty);
});
```

Ini padanan langsung dari `test_no_disk_write.py` di
[Fase 2 § 11.3](03-Fase2.md#113-test-tidak-menyentuh-disk) — dan alasannya sama:
tumpahan berkas biometrik ke disk terjadi tanpa error dan tanpa log.

### 13.6 Uji liveness

| Uji | Cara | Harapan |
|---|---|---|
| Foto cetak | Arahkan kamera ke foto cetak wajah | **Gagal** — tidak berkedip |
| Foto di layar ponsel | Tampilkan foto di ponsel lain | **Gagal** |
| Video replay | Putar video subjek berkedip & tersenyum | **Kemungkinan lolos** — dicatat sebagai batasan yang diketahui, bukan bug |
| Ganti wajah di tengah | Orang A memulai, orang B masuk frame | Sesi **dibatalkan** (`trackingId` berubah) |
| Swap setelah liveness | Liveness selesai, kamera diarahkan ke foto | **Tidak mungkin** dengan D29 — frame sudah terpilih |
| Wajah jauh | Mundur 2 m | Umpan balik "Wajah terlalu jauh", tantangan tidak dimulai |
| Gelap total | Tutup lampu | Peringatan gelap; bila tetap dikirim, server yang menolak dengan `too_dark` |
| Perangkat tanpa ML Kit | Android tanpa GMS | `liveness_supported=false`, absensi tetap bisa (`policy=preferred`) |

Uji video replay **wajib dijalankan dan hasilnya dicatat apa adanya** di
`liveness-threat-model.md`. Melewatkannya berarti mengklaim penutupan yang tidak
pernah diverifikasi.

### 13.7 Uji setting berubah tanpa rilis ulang

```
1. Web (Fase 5) → /settings → ubah attendance.max_gps_accuracy_meter 100 → 30
2. Mobile: tutup & buka app (context di-refetch)
3. Harapan: ambang blokir akurasi berubah TANPA rilis ulang app
4. Ulangi untuk: outside_geofence_policy, require_face_for_checkout,
   liveness_policy, photo.max_bytes
```

Uji ini yang membuktikan tidak ada asumsi ter-hardcode — kegagalan di sini berarti
setiap perubahan kebijakan butuh rilis app baru, yang di lapangan berarti berminggu-minggu.

### 13.8 Uji lapangan terkendali (sebelum rilis)

Pilot 10–15 karyawan selama 2 minggu, mengukur:
- Tingkat keberhasilan check-in percobaan pertama (target ≥ 90%)
- Tingkat masuk `pending_review` (target ≤ 10% — bila lebih, threshold atau gate
  kualitas perlu ditinjau)
- Distribusi `hints` yang paling sering muncul → memandu perbaikan panduan UI
- Tingkat kegagalan liveness pada pengguna sah (target ≤ 5%)
- Konsumsi baterai & data per absensi

Angka-angka ini juga menjadi masukan nyata untuk **R2** (kalibrasi) — pilot ini
adalah kesempatan paling murah untuk mengumpulkan Dataset B yang selama ini tertunda.

---

## 14. Kontrak yang Perlu Direvisi

Mengikuti pola Fase 3/4 dan 5/6: setiap penyesuaian dilaporkan di sini, bukan diubah
diam-diam. **Bagian ini lengkap — tidak ada item yang sengaja ditunda ke dokumen lain.**

| # | Kontrak | Revisi | Sifat | Wajib? |
|---|---|---|---|---|
| **R1** | Fase 4 #55 `/attendances/context` | Sertakan `require_face_for_checkout`, `checkout_without_face_status`, dan `max_note_length` — sudah diminta [Fase 6 § 12 no. 1](07-Fase6.md#12-kontrak-fase-05-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam), naik jadi **Wajib** karena kontradiksi K-01 (`checkout_without_face_status` tidak pernah terbaca lewat #28 karena `is_public=false`). **Kontrak sudah final** — [Fase 4 § 4.4](05-Fase4.md#44-get-attendancescontext), [REV-EP-04](09-Revisions-Log.md#b-perubahan-katalog-endpoint) — implementasi kode menunggu eksekusi Fase 4 | Tambahan | **Wajib** |
| **R2** | Fase 4 #55 `context.photo` | Tambah `recommended_dimension_px` (default 1280) dan `max_dimension_px` (default 1920). Web mengasumsikan 1280×720 tetap ([Fase 6 D24](07-Fase6.md#d24--parameter-capture--butuh-konfirmasi)); kamera ponsel bervariasi dari 5 MP sampai 108 MP dan butuh target yang dinyatakan server | Tambahan | **Wajib** |
| **R3** | Fase 1 § 2.5 & § 5.2 — rotasi refresh token | Tambah **grace window**: token dengan `used_at` dalam `auth.refresh_reuse_grace_seconds` (setting baru, default 30) yang anaknya belum terpakai → terbitkan ulang pasangan yang sama, **jangan** revoke family. Di luar itu, perilaku deteksi reuse Fase 1 tidak berubah. Diperlukan karena response yang hilang di jaringan lapangan akan mencabut sesi karyawan secara permanen (§ 2.5b) | **Perubahan perilaku** + setting baru + kolom tidak berubah | **Wajib** |
| **R4** | Fase 4 — dukungan liveness | (a) `attendances`: kolom baru `liveness_passed boolean NULL`, `liveness_supported boolean NULL`, `liveness_method text NULL`, `liveness_challenges jsonb NOT NULL DEFAULT '[]'`. (b) `fallback_reason` CHECK: tambah `'liveness_failed'`. (c) `attendance_attempts.outcome` CHECK: tambah `'liveness_failed'`. (d) #53/#54 menerima field `liveness_passed`, `liveness_supported`, `liveness_method`, `liveness_challenges`. (e) `app_settings` baru: `attendance.liveness_policy` (`off`\|`preferred`\|`required`, default `preferred`), `attendance.liveness_max_attempts` (3), `attendance.liveness_challenge_count` (2), `attendance.liveness_timeout_seconds` (20) — semuanya diekspos di #55 | Migration baru + perubahan CHECK + setting | **Wajib** |
| **R5** | Fase 4 — deteksi mock location | (a) `attendances.location_is_mocked boolean NULL`. (b) `fallback_reason` CHECK: tambah `'location_mocked'`. (c) `attendance_attempts.outcome`: tambah `'location_mocked'`. (d) #53/#54 menerima `location_is_mocked`. (e) `app_settings.attendance.mocked_location_policy` (`reject`\|`pending_review`, default `reject`), diekspos di #55. [Fase 4 E21](05-Fase4.md#63-lokasi) menyatakan mock location "tidak terdeteksi"; Android **bisa** mendeteksinya lewat `Position.isMocked`, jadi informasi itu sekarang tersedia dan sayang dibuang. iOS tetap tidak bisa — dinyatakan sebagai keterbatasan | Migration + setting | Direkomendasikan |
| **R6** | Berkas kanonik pesan (usulan [Fase 6 § 12 no. 5](07-Fase6.md#12-kontrak-fase-05-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam)) | Strukturkan `docs/api/hints.json` dengan **namespace terpisah**: `server_hints` (kosakata tertutup Fase 2 — **tidak boleh** ditambah dari client) dan `client_coach` (kalimat liveness khas mobile: `liveness_blink`, `liveness_smile`, `liveness_hold_still`, `liveness_face_lost`, `liveness_multiple_faces`, `liveness_too_far`, `liveness_timeout`, `liveness_unsupported`). Tanpa pemisahan ini, menambah kalimat liveness akan mencemari kosakata tertutup [Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints) | Perubahan struktur berkas | **Wajib** |
| **R7** | Fase 0 katalog error | Kode baru: **`LIVENESS_REQUIRED`** — HTTP **422**, muncul hanya bila `liveness_policy='required'` dan `liveness_passed != true`. Response **wajib** menyertakan `can_fallback` (nilainya `attendance.fallback_enabled`), konsisten dengan aturan `can_fallback` di R8. Tidak ada kode untuk "perangkat tidak didukung" — itu ditangani `liveness_policy`, bukan error tersendiri | Tambahan katalog | **Wajib** |
| **R8** | Fase 4 — cakupan `can_fallback` | Sudah diminta [Fase 6 § 12 no. 2](07-Fase6.md#12-kontrak-fase-05-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam) dan **masih terbuka**. Tegaskan berlaku pada `FACE_NOT_MATCHED`, `FACE_NOT_USABLE`, `FACE_NOT_ENROLLED`, `LIVENESS_REQUIRED` (baru), `502 UPSTREAM_ERROR`, `504 UPSTREAM_TIMEOUT`. Mobile membaca field ini dan **tidak** menyimpulkan sendiri | Klarifikasi + tambahan | **Wajib** |
| **R9** | Fase 5 § 2.5 — panel bukti approval | Tampilkan `liveness_passed`, `liveness_supported`, `liveness_method`, dan `location_is_mocked` di panel detail. Tanpa ini, reviewer menyetujui record `fallback_reason='liveness_failed'` tanpa tahu apa yang gagal — dan justru record itulah yang paling perlu dinilai manusia | Tambahan UI | **Wajib** |
| **R10** | Fase 3 — `capture_source` | Nilai `'mobile_camera'` **sudah** ada di CHECK `face_references` ([Fase 3 § 3.4](04-Fase3.md#34-face_references--migration-000014)) dan `attendances` ([Fase 4 § 3.2](05-Fase4.md#32-attendances--migration-000019)). **Diverifikasi: tidak perlu perubahan.** Dicatat di sini supaya tidak diperiksa ulang | — | Tidak perlu |
| **R11** | Fase 0 #`GET /api/v1/version` | Tambah `min_supported_client` (mis. `{"mobile":"1.0.0"}`) supaya app bisa menampilkan "Perbarui aplikasi" saat kontrak berubah tidak kompatibel (E9). Tanpa ini, app lama akan gagal dengan error yang membingungkan setelah backend di-deploy | Tambahan | Direkomendasikan |

### 14.1 Yang diverifikasi dan **tidak** perlu revisi

Diperiksa satu per satu supaya tidak menjadi keraguan yang terbawa:

| Diperiksa | Hasil |
|---|---|
| Endpoint baru | **Tidak ada.** Verifikasi lengkap di § 4.6 |
| Permission baru | **Tidak ada.** App hanya memakai 6 permission milik role `employee` ([Fase 1 § 2.4](02-Fase1.md#24-role-default)) |
| **D23 → apakah berubah jadi perubahan kontrak backend?** | **Tidak.** Penyimpanan token adalah keputusan client. Server tetap menerbitkan token yang sama, TTL yang sama (`JWT_REFRESH_TTL=720h`). **Tidak** direkomendasikan TTL berbeda per platform — ia menambah cabang kebijakan tanpa manfaat nyata, dan 30 hari sudah memadai untuk pemakaian harian. Yang **memang** perlu perubahan backend adalah R3 (grace window), dan itu berlaku untuk web maupun mobile |
| `Idempotency-Key` 24 jam ([Fase 4 § 2.8](05-Fase4.md#28-idempotensi)) | **Cukup.** Dengan D31 (retry-only, jendela 15 menit), 24 jam jauh melampaui kebutuhan |
| `AttendanceEmployeeDTO` | **Tidak perlu varian mobile.** Field yang sama persis |
| Katalog `hints` Fase 2 | **Tidak ditambah.** Kalimat liveness masuk namespace terpisah (R6) |
| Alur enrollment Fase 3 | **Tidak berubah.** Liveness ditambahkan di sisi client sebelum `POST .../photos`, tanpa mengubah kontrak endpoint |
| Endpoint #71/#72 (Fase 5) | **Tidak dipakai** — fitur admin, di luar D27 |

### 14.2 Catatan konsolidasi

Berkas gabungan revisi (`09-Revisions-Log.md`) **belum dibuat** dan memang belum
diminta. Sampai dokumen itu ada, revisi kontrak tersebar di empat tempat dan
semuanya masih **terbuka** (belum ada backend yang mengimplementasikannya):

| Sumber | Jumlah item |
|---|---|
| [04-Fase3.md](04-Fase3.md) § 2.9 + § 3.8 | Error code & permission baru Fase 3 |
| [05-Fase4.md](05-Fase4.md) § 2.1 + § 2.10 | Rekonsiliasi setting + error code Fase 4 |
| [06-Fase5.md § 12](06-Fase5.md#12-kontrak-fase-04-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam) | 7 item |
| [07-Fase6.md § 12](07-Fase6.md#12-kontrak-fase-05-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam) | 6 item (2 di antaranya diulang di sini sebagai R1 & R8 karena masih terbuka) |
| **08-Fase7.md § 14** (dokumen ini) | **11 item** |

---

## 15. Referensi Silang ke "Isu Lintas-Fase" (Master Plan § 9)

| Isu lintas-fase | Bagaimana Fase 7 memenuhinya |
|---|---|
| **Anti-spoofing / liveness** | Deliverable utama fase ini (§ 6). Ditutup untuk serangan oportunistik; **tidak** ditutup untuk video replay, deepfake, dan APK dimodifikasi — dinyatakan apa adanya di § 6.1, bukan diklaim selesai |
| **Keamanan fallback** | Jalur wajah + liveness selalu dicoba lebih dulu; fallback hanya sebagai konsekuensi kegagalan dan hanya bila `can_fallback` dari server. Liveness gagal → `pending_review`, tidak pernah `approved` |
| **Timestamp server-side** | `server_timestamp` tetap dari server; jam perangkat tidak pernah dipercaya; **antrean offline ditolak** justru karena akan merusak janji ini (§ 2.7) |
| **Geofence server-side** | Seluruh evaluasi tetap di server; app hanya menampilkan jarak indikatif; `isMocked` dilaporkan ke server, bukan dinilai sendiri (R5) |
| **Threshold configurable** | Seluruh ambang — termasuk kebijakan liveness yang baru — dibaca dari `#55 context`; diuji bahwa perubahan setting di web berlaku **tanpa rilis ulang app** (§ 13.7) |
| **Data biometrik = data sensitif (UU PDP)** | Foto hanya di memori, tidak pernah persisten (§ 7.4, diuji § 13.5); `XFile` temp dihapus segera; tidak ada izin galeri; refresh token di Keychain/Keystore tanpa backup cloud; logout membersihkan semuanya; deklarasi data biometrik di kedua store |
| **RBAC konsisten** | App hanya klien; gating UI dari `permissions`, bukan nama role (§ 3.3); enforcement tetap di API |
