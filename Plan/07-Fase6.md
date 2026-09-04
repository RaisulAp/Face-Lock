# Fase 6 — Halaman Absensi Web (`faceclock-web`)

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 6**. Ukuran: 🟢 Kecil
> menurut master plan — tetapi dokumen ini tetap dibuat karena fase ini memegang
> satu-satunya penegakan anti-spoofing yang dimiliki versi web (paksa kamera, larang
> galeri), dan itu terlalu mudah dilanggar tanpa spesifikasi.
>
> **Depends on:** [Fase 3](04-Fase3.md), [Fase 4](05-Fase4.md), dan
> [Fase 5](06-Fase5.md) (app shell: auth, refresh lock, `<Can>`, `<AuthImage>`,
> katalog error, `<LocalTime>`).
>
> ✅ **Milestone setelah fase ini: "Web fully working"** — Fase 7 (Flutter) baru
> dimulai setelah milestone ini dikonfirmasi.
>
> **Status:** DRAFT — 3 keputusan butuh konfirmasi user (§ 2.0), dan **1 prasyarat
> infrastruktur yang memblokir pengujian nyata** (§ 2.1).

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Fase 3 dan 4 membangun kemampuan; Fase 6 adalah satu-satunya tempat karyawan
benar-benar memakainya. Dua hal yang **hanya** bisa diselesaikan di sini:

1. **Penegakan "capture langsung dari kamera".** Master plan § 9 menyatakan web
   minimal harus memaksa capture kamera, bukan unggah galeri. Backend **tidak bisa
   memaksakan ini** — [Fase 3 E23](04-Fase3.md#64-keamanan--integritas) dan
   [Fase 4 E28](05-Fase4.md#64-keamanan--integritas) sudah menyatakan terus terang
   bahwa `capture_source` hanyalah klaim client. Satu-satunya penegakan yang ada di
   versi web adalah tidak menyediakan jalan lain. Itu pekerjaan fase ini.
2. **Menerjemahkan `hints` menjadi tindakan.** [Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints)
   merancang kosakata `hints` justru supaya client bisa memandu orang memperbaiki
   fotonya sendiri. Kalau panduan itu tidak ada, setiap kegagalan kualitas berakhir
   di antrian `pending_review` — dan antrian yang membengkak adalah antrian yang
   akhirnya disetujui tanpa dilihat.

### Hasil akhir yang diharapkan
- Karyawan bisa login, membaca dan menyetujui dokumen consent biometrik.
- Karyawan bisa mendaftarkan wajah lewat sesi bertahap dengan umpan balik per foto.
- Karyawan bisa check-in dan check-out dengan kamera browser dan lokasi browser.
- Jalur wajah selalu dicoba lebih dulu; opsi "kirim untuk ditinjau" hanya muncul
  **setelah** jalur wajah gagal.
- Karyawan bisa melihat riwayat absensinya sendiri — tanpa satu pun angka skor.
- Setiap keadaan tidak biasa (mode manual, consent dicabut, belum enroll, kamera
  ditolak, model berganti) dijelaskan dengan kalimat yang bisa ditindaklanjuti,
  bukan pesan teknis.

### Yang TIDAK dikerjakan di fase ini
- Endpoint backend baru. Fase 6 murni konsumen — dan itu sekaligus pembuktian bahwa
  kontrak Fase 3/4 memang dirancang untuk client ini (§ 4.5).
- Liveness sungguhan. Master plan § 9 menaruhnya di Fase 7 (on-device). Fase 6
  hanya menaikkan biayanya, dan keterbatasannya dinyatakan (§ 6.6 E29).
- Mode offline / antrean absensi tertunda (→ Fase 7).
- Mode kiosk multi-karyawan pada satu perangkat (§ 6.6 E31).
- PWA / installable app.

---

## 2. Scope Detail

### 2.0 Keputusan yang butuh konfirmasi

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D24 | Parameter capture (resolusi, kompresi) | Capture `1280×720`, JPEG kualitas adaptif, target ≤ 800 KB; batas dibaca dari `/attendances/context` | ⚠️ **BUTUH KONFIRMASI** |
| D25 | Pra-pemeriksaan kualitas di browser | **Hanya peringatan gelap/terang berbasis luminansi kanvas — bersifat saran, tidak pernah memblokir.** Tidak memakai `FaceDetector` API | ⚠️ **BUTUH KONFIRMASI** |
| D26 | Panduan variasi foto enrollment | Variasi **ekspresi & pencahayaan**, bukan variasi arah kepala (§ 2.5) | ⚠️ **BUTUH KONFIRMASI** |

---

### 2.1 ⚠️ Prasyarat infrastruktur: HTTPS

`navigator.mediaDevices.getUserMedia` dan `navigator.geolocation` hanya tersedia di
**secure context**: `https://` atau `http://localhost`.

Artinya: begitu `faceclock-web` diakses dari perangkat lain lewat IP LAN
(`http://192.168.1.10:5173`), **kamera dan lokasi tidak akan berfungsi sama sekali** —
`getUserMedia` melempar `SecurityError` dan `navigator.mediaDevices` bahkan bisa
`undefined`.

Ini bukan detail yang bisa ditunda ke saat deploy: seluruh pengujian lapangan Fase 6
membutuhkannya. Sudah dilaporkan sebagai revisi kontrak di
[Fase 5 § 12 nomor 6](06-Fase5.md#12-kontrak-fase-04-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam);
di sini ia menjadi **prasyarat eksekusi**.

Pilihan yang tersedia untuk lingkungan dev/staging:

| Opsi | Catatan |
|---|---|
| `mkcert` + sertifikat lokal, Vite `server.https` | Paling cepat untuk dev di LAN; sertifikat harus dipasang di tiap perangkat uji |
| Reverse proxy (Caddy/nginx) dengan Let's Encrypt di staging | Paling mendekati produksi; butuh domain |
| Tunnel (`cloudflared`, `ngrok`) | Praktis untuk uji cepat di HP; jangan dipakai membawa data biometrik nyata |

UI **wajib** mendeteksi keadaan ini dan menjelaskannya, bukan menampilkan kegagalan
kamera yang membingungkan:

```ts
if (!window.isSecureContext) {
  // "Halaman ini harus diakses lewat HTTPS agar kamera dan lokasi dapat digunakan.
  //  Hubungi admin sistem."
}
```

---

### 2.2 Pipeline capture kamera — dan satu kesalahan yang harus dihindari

#### Alur teknis

```
1. getUserMedia({ video: { facingMode: "user",
                           width:  { ideal: 1280 },
                           height: { ideal: 720 } },
                  audio: false })
2. stream → <video autoPlay playsInline muted>
     preview DICERMINKAN lewat CSS:  transform: scaleX(-1)
3. Saat tombol ambil ditekan:
     canvas.width/height = video.videoWidth/videoHeight
     ctx.drawImage(video, 0, 0)          ← TANPA ctx.scale(-1, 1)
4. canvas.toBlob(blob => …, "image/jpeg", quality)
     quality mulai 0.85; bila blob > target, turunkan 0.75 → 0.65;
     bila masih besar, perkecil kanvas ke 960px sisi terpanjang lalu ulangi
5. Hentikan seluruh track saat komponen dilepas:
     stream.getTracks().forEach(t => t.stop())
```

#### Kesalahan yang harus dihindari: mengirim frame yang tercermin

Preview **harus** dicerminkan — orang mengharapkan gerakannya seperti di cermin, dan
preview yang tidak dicerminkan membuat orang menggeser kepala ke arah yang salah.

Tetapi **frame yang dikirim ke server tidak boleh dicerminkan.** Cara paling umum
"memperbaiki" cermin adalah menerapkan `ctx.scale(-1, 1)` sebelum `drawImage` — dan
itu justru menghasilkan bug yang sangat sulit dilacak: embedding dari wajah tercermin
**tidak identik** dengan embedding dari wajah aslinya. Kalau enrollment dilakukan
lewat unggahan admin (`capture_source='admin_upload'`, tidak tercermin) sementara
check-in memakai frame tercermin, similarity akan turun sistematis untuk semua orang —
tanpa satu pun error, tanpa satu pun `hint`, dan gejalanya hanya "akurasinya kurang bagus".

**Aturan:** cerminkan hanya CSS pada elemen `<video>`. Kanvas selalu menggambar frame
mentah. Aturan ini ditegakkan test (§ 11.4).

#### Catatan EXIF

`canvas.toBlob()` menghasilkan JPEG **tanpa EXIF**, jadi masalah orientasi yang
ditangani [Fase 2 § 2.7a](03-Fase2.md#27-penanganan-gambar--dua-celah-yang-harus-ditutup)
tidak muncul dari jalur ini. Penanganan EXIF di server tetap dibutuhkan untuk unggahan
admin dan untuk Fase 7 — bukan untuk Fase 6.

#### D24 — parameter capture ⚠️ BUTUH KONFIRMASI

- `1280×720` dipilih karena [Fase 2 § 2.7d](03-Fase2.md#27-penanganan-gambar--dua-celah-yang-harus-ditutup)
  menyatakan server memperkecil frame di atas 1920 px sebelum deteksi. Mengirim 4K
  hanya menambah waktu unggah dan waktu decode tanpa menambah akurasi sedikit pun.
- Target ≤ 800 KB, jauh di bawah `face.max_image_bytes` (default 6 MB). Marginnya
  lebar karena jaringan kantor bukan selalu cepat, dan `413` di depan kamera adalah
  pengalaman yang buruk.
- **Batas keras dibaca dari `/attendances/context` (#55)** — `photo.max_bytes` dan
  `photo.accepted_mime_types` — bukan konstanta di frontend. Kalau di-hardcode,
  mengubah setting di server akan membuat client menolak foto yang sebenarnya sah.

#### D25 — pra-pemeriksaan di browser ⚠️ BUTUH KONFIRMASI

Yang **dilakukan**: menghitung luminansi rata-rata kanvas sebelum unggah, dan
menampilkan peringatan lembut *"Ruangan tampak gelap — hasil bisa ditolak"* bila
sangat rendah. Bersifat **saran**; tombol kirim tetap aktif.

Yang **tidak dilakukan**: deteksi wajah di browser (`FaceDetector` API atau
face-api.js). Alasannya:
- `FaceDetector` hanya ada di sebagian Chromium dan di belakang flag; hasilnya tidak
  konsisten antar-perangkat.
- face-api.js menambah beberapa megabyte model ke bundle untuk pekerjaan yang server
  sudah lakukan dengan model yang jauh lebih baik.
- Yang lebih penting: **server adalah otoritas kualitas**. Gate kualitas di
  [Fase 2](03-Fase2.md#25-kontrak-kualitas--kosakata-hints) yang menentukan `usable`.
  Pra-pemeriksaan client yang lebih ketat akan menolak foto yang sebenarnya diterima
  server; yang lebih longgar tidak berguna. Dua otoritas kualitas yang berbeda pasti
  akan berselisih, dan selisihnya muncul sebagai "kadang bisa, kadang tidak".

Panduan yang benar-benar membantu bukan deteksi client, melainkan **overlay bingkai
wajah** (oval panduan) dan **penerjemahan `hints` setelah percobaan** (§ 2.4).

---

### 2.3 Lokasi browser

```ts
navigator.geolocation.getCurrentPosition(ok, err, {
  enableHighAccuracy: true,
  timeout: 10_000,
  maximumAge: 0,          // jangan pakai cache; absensi butuh posisi SAAT INI
});
```

`maximumAge: 0` disengaja: posisi ter-cache dari 10 menit lalu bisa berasal dari
lokasi yang berbeda, dan geofence yang divalidasi terhadap posisi basi adalah
geofence yang bisa ditipu tanpa alat apa pun.

Lokasi diambil **saat halaman dibuka**, bukan saat tombol kirim ditekan — perolehan
GPS bisa memakan beberapa detik, dan menunggunya setelah foto diambil membuat orang
mengira aplikasinya menggantung. Nilainya di-*refresh* bila lebih tua dari 60 detik
saat kirim.

**Perilaku terhadap akurasi** membaca setting, bukan mengasumsikan
([D18](05-Fase4.md#25-d18--geofence--butuh-konfirmasi)):

```
accuracy ≤ context.geofence.max_gps_accuracy_meter          ⇒ lanjut normal
accuracy >  ambang  DAN  missing_location_policy = "reject" ⇒ blokir kirim,
      tampilkan "Akurasi lokasi belum cukup (±N m). Dekati jendela lalu coba lagi."
      + tombol [Ambil ulang lokasi]
accuracy >  ambang  DAN  policy = "pending_review"          ⇒ izinkan kirim,
      peringatan "Absensi akan masuk antrian persetujuan karena lokasi kurang akurat."
```

Memblokir di client saat kebijakannya `reject` bukan duplikasi logika — ia mencegah
pengunggahan foto wajah yang **pasti** akan ditolak. [Fase 4 § 5.1](05-Fase4.md#51-check-in--alur-lengkap)
mengevaluasi geofence sebelum inference justru dengan alasan yang sama: jangan
memproses data biometrik untuk permintaan yang tidak akan pernah sah.

UI juga menampilkan **jarak perkiraan** ke `context.geofence.nearest_location`
(dihitung client dengan haversine yang sama), sehingga orang tahu ia kurang 30 meter,
bukan sekadar "di luar area". Angka ini **indikatif**; yang menentukan tetap server.

---

### 2.4 Alur check-in / check-out (D17)

[Fase 4 D17](05-Fase4.md#23-d17--bentuk-alur-fallback--butuh-konfirmasi) menetapkan
fallback sebagai **flag pada alur yang sama**, bukan endpoint terpisah. Terjemahannya
ke UI adalah satu aturan keras:

> **Tidak boleh ada tombol "absen manual" yang berdiri sendiri di mana pun.**
> Opsi "kirim untuk ditinjau" hanya muncul sebagai **konsekuensi** dari percobaan
> wajah yang gagal, di layar hasil percobaan itu.

Kalau tombol itu berdiri sendiri, ia akan menjadi jalur yang dipakai setiap orang
setiap hari — lebih cepat, tidak pernah gagal — dan verifikasi wajah menjadi hiasan.

```
[Layar siap]  kamera hidup, lokasi didapat, tombol "Check-in" aktif
   │
   ├─ ambil foto → pratinjau → [Kirim]
   │     POST #53 allow_fallback=false, Idempotency-Key = K1
   │
   ├─ 201 status=approved  ──────────────► [Layar berhasil]
   │                                        jam server, lokasi, "Tercatat"
   │
   ├─ 201 status=pending_review ─────────► [Layar menunggu persetujuan]
   │     (terjadi bila mode manual, atau kebijakan geofence pending_review)
   │
   ├─ 422 FACE_NOT_MATCHED  ─────────────┐
   ├─ 422 FACE_NOT_USABLE (+hints) ──────┤
   ├─ 502 / 504 (inference bermasalah) ──┤
   │                                     ▼
   │                            [Layar percobaan gagal]
   │                              • alasan dalam bahasa manusia
   │                              • daftar hints → kalimat tindakan
   │                              • [Coba lagi]  → kembali ke layar siap,
   │                                                 Idempotency-Key BARU
   │                              • [Kirim untuk ditinjau admin]
   │                                   muncul HANYA bila error.can_fallback
   │                                   → WAJIB ambil foto BARU
   │                                   → POST #53 allow_fallback=true, key K2
   │                                   → 201 pending_review → layar menunggu
   │
   ├─ 409 ALREADY_CHECKED_IN ────────────► [Layar sudah absen] + ringkasan hari ini
   ├─ 422 FACE_NOT_ENROLLED ─────────────► [Ajakan enroll] + tautan /me/enrollment
   ├─ 422 OUTSIDE_GEOFENCE ──────────────► [Layar di luar area] + jarak + lokasi kantor
   ├─ 403 CONSENT_REQUIRED ──────────────► [Layar consent] (§ 2.6)
   ├─ 429 TOO_MANY_FAILED_ATTEMPTS ──────► [Layar jeda] + hitung mundur Retry-After
   ├─ 503 FACE_SERVICE_NOT_CONFIGURED ───► "Sistem belum dikonfigurasi.
   │                                        Hubungi admin." (bukan error teknis)
   └─ 503 ATTENDANCE_NOT_CONFIGURED ─────► "Sistem belum dikonfigurasi.
                                             Hubungi admin." (bukan error teknis; pesan
                                             sama, dua kode — lihat K-04)
```

**Foto baru untuk fallback** mengikuti [Fase 4 § 2.3](05-Fase4.md#23-d17--bentuk-alur-fallback--butuh-konfirmasi):
*"Foto kedua adalah capture baru, bukan pengulangan foto yang sama. Itu bukti yang
lebih baik, bukan lebih buruk."* Biayanya satu capture ulang; manfaatnya adalah bukti
yang segar untuk reviewer di Fase 5.

**`Idempotency-Key`** ([Fase 4 § 2.8](05-Fase4.md#28-idempotensi)) dibuat sebagai ULID
saat tombol Kirim ditekan, dan dipertahankan **hanya** untuk pengulangan akibat
kegagalan jaringan pada percobaan yang sama. Setiap capture baru = kunci baru.
Aturannya:

| Kejadian | Kunci |
|---|---|
| Kirim pertama | Buat baru |
| Timeout jaringan → tombol "Coba kirim lagi" | **Sama** — ini yang membuat idempotensi ada gunanya |
| Pengguna menekan "Coba lagi" (capture ulang) | Baru |
| Kirim fallback dengan foto baru | Baru |

**Mode manual.** Bila `context.attendance_mode === "manual"`
([Fase 4 § 5.1 langkah 10](05-Fase4.md#51-check-in--alur-lengkap)), server melewati
pencocokan wajah seluruhnya dan selalu menghasilkan `pending_review`. UI:
- tetap memakai kamera (foto tetap disimpan sebagai bukti untuk reviewer);
- **tidak** menampilkan janji "verifikasi wajah";
- menjelaskan di depan: *"Akun Anda diatur pada mode absensi manual. Foto Anda
  disimpan sebagai bukti dan absensi akan ditinjau admin. Tidak ada pencocokan
  wajah otomatis."*
- mengirim satu kali saja — tidak ada tarian fallback, karena tidak ada yang gagal.

**Check-out** memakai alur yang sama, dengan dua perbedaan yang dibaca dari context:
- `attendance.require_face_for_checkout = false` (D19) → tidak ada percobaan wajah;
  UI mengatakan apa adanya bahwa check-out tidak diverifikasi wajah, dan menampilkan
  status yang akan dihasilkan (`checkout_without_face_status`).
- Error khusus `409 CHECKOUT_WITHOUT_CHECKIN` dan `409 CHECKOUT_TOO_SOON` (yang
  `details`-nya menyebut sisa menit) ditampilkan sebagai keadaan, bukan kesalahan.

---

### 2.5 Enrollment — sesi bertahap (D15)

Mengikuti [Fase 3 § 2.3](04-Fase3.md#23-d15--alur-enrollment-bertahap--butuh-konfirmasi)
persis.

```
Langkah 0  Status        #48 GET /face/enrollment-status/me
              consent.status ≠ granted        → Langkah 1
              attendance_mode = "manual"      → layar penjelasan, berhenti
              draft_session_id ada            → lanjutkan sesi itu
              is_enrolled && !needs_re_enrollment → layar "sudah terdaftar"
                                                   + opsi [Daftar ulang]
Langkah 1  Consent       #32 dokumen → #34 setujui           (§ 2.6)
Langkah 2  Buat sesi     #38 POST /face/enrollments {mode}
                            409 CONFLICT + existing_session_id ⇒ lanjutkan sesi itu
                            503 ⇒ "Sistem belum siap menerima pendaftaran wajah."
                            409 REINDEX_IN_PROGRESS ⇒ "Sedang ada pemeliharaan data
                                  wajah. Coba lagi beberapa saat lagi."
Langkah 3  Ambil foto    ulangi sampai `required_photos` terpenuhi:
                            capture → #40 POST .../photos (capture_source=web_camera)
                              201  ⇒ kartu foto HIJAU, tampilkan quality_score
                              422 FACE_NOT_USABLE ⇒ kartu MERAH + hints → kalimat aksi,
                                    foto TIDAK tersimpan di mana pun, ambil ulang
                              409 DUPLICATE_PHOTO ⇒ "Foto ini sama persis dengan yang
                                    sudah diambil. Ubah ekspresi lalu ambil lagi."
                              409 ENROLLMENT_LIMIT_REACHED ⇒ tombol ambil dinonaktifkan
                            [Hapus] pada kartu → #41 DELETE
Langkah 4  Tinjau        pratinjau seluruh foto diterima (dari blob lokal, bukan server)
Langkah 5  Simpan        #42 POST .../commit
                            200 ⇒ layar berhasil + tautan ke /me/attendance
                            422 ENROLLMENT_INCOMPLETE ⇒ "Butuh N foto lagi."
                            409 ENROLLMENT_MODEL_CHANGED ⇒ "Sistem pengenalan wajah
                                  telah diperbarui. Sesi ini harus diulang." + [Mulai ulang]
                            409 FACE_BELONGS_TO_ANOTHER_EMPLOYEE ⇒ § 2.7
                            403 CONSENT_REQUIRED ⇒ kembali ke Langkah 1
                            409 REINDEX_IN_PROGRESS ⇒ pesan pemeliharaan
```

**Pratinjau memakai blob lokal**, bukan mengunduh ulang dari server. Foto staging
memang belum punya endpoint baca di [Fase 3](04-Fase3.md#41-ringkasan) — dan memang
tidak perlu: client baru saja membuat gambarnya. Object URL-nya di-*revoke* saat
kartu dilepas atau sesi selesai.

**Hitung mundur sesi.** `expires_at` (default 30 menit) ditampilkan; pada 5 menit
terakhir muncul peringatan. Sesi kedaluwarsa → `409 ENROLLMENT_SESSION_EXPIRED` →
tawarkan mulai ulang.

#### D26 — panduan variasi foto ⚠️ BUTUH KONFIRMASI

Ini catatan yang mencegah UI berkelahi dengan backend-nya sendiri.

Panduan enrollment yang lazim adalah *"hadap kiri, hadap kanan, hadap depan"*.
**Panduan itu salah untuk sistem ini.** Gate kualitas Fase 2 menolak
`head_turned` bila `|yaw| > face.max_abs_yaw` (default `0.35`) dan `head_tilted`
bila `|pitch| > 0.30`
([Fase 2 § 2.5](03-Fase2.md#25-kontrak-kualitas--kosakata-hints)). Menyuruh orang
menoleh berarti menyuruhnya menghasilkan foto yang akan ditolak — lalu ia akan
mencoba berkali-kali dan menyimpulkan sistemnya rusak.

Panduan yang benar meminta variasi pada dimensi yang **tidak** dijaga gate:

| Foto | Panduan |
|---|---|
| 1 | *"Hadap lurus ke kamera, ekspresi netral."* |
| 2 | *"Tetap hadap lurus, tersenyum tipis."* |
| 3 | *"Hadap lurus, sedikit ubah posisi duduk/berdiri atau pencahayaan."* |
| 4–5 (opsional) | *"Bila Anda biasa memakai kacamata, ambil satu dengan dan satu tanpa kacamata."* |

Variasi kacamata adalah yang paling bernilai secara praktis, karena itulah perbedaan
harian yang paling sering menurunkan similarity.

Overlay oval panduan ditampilkan pada preview untuk membantu orang mengisi bingkai —
menangani `face_too_small`, yang merupakan penyebab penolakan paling sering pada
webcam laptop.

---

### 2.6 Consent

Layar consent adalah gerbang yang [Fase 3 § 2.4](04-Fase3.md#24-consent-biometrik--di-mana-disimpan--bagaimana-ditegakkan)
tegakkan di server lewat `RequireConsent`. Di client, ia harus **sungguh-sungguh
dibaca**, bukan sekadar diklik.

```
#32 GET /consents/document → { version, title, body (Markdown), content_hash }
 ├─ render `body` sebagai Markdown TERSANITASI (§ 6.5 E24)
 ├─ tombol setuju NONAKTIF sampai:
 │     (a) area teks di-scroll sampai bawah, DAN
 │     (b) checkbox "Saya telah membaca dan menyetujui" dicentang
 ├─ #34 POST /consents { document_version, agreed: true }
 │     409 CONSENT_VERSION_OUTDATED ⇒ muat ulang dokumen (versi berubah saat dibaca)
 │     409 CONSENT_ALREADY_GRANTED  ⇒ lanjut saja; ini bukan kegagalan
 │     404 NOT_FOUND ⇒ "Akun Anda belum terhubung ke data karyawan. Hubungi HR."
 └─ 201 ⇒ lanjut ke sesi enrollment
```

`agreed: true` harus berasal dari tindakan eksplisit
([Fase 3 E3](04-Fase3.md#61-consent)) — checkbox tidak boleh tercentang secara default.

Halaman `/me/consent` juga menampilkan status saat ini (#33) dan menyediakan
**pencabutan** (#35), dengan dialog yang menyebut akibatnya apa adanya:

> *"Mencabut persetujuan akan menonaktifkan seluruh data wajah Anda. Anda tidak akan
> bisa absen dengan wajah sampai mendaftar ulang. Hubungi HR bila Anda memerlukan
> jalur absensi alternatif."*

Kalimat terakhir penting: [Fase 3 § 2.4d](04-Fase3.md#24-consent-biometrik--di-mana-disimpan--bagaimana-ditegakkan)
sengaja **tidak** memindahkan karyawan ke `attendance_mode='manual'` secara otomatis —
itu keputusan HR. Tanpa kalimat itu, karyawan yang mencabut consent akan menemukan
dirinya tidak bisa absen sama sekali tanpa tahu harus ke mana.

Setelah mencabut, `/me/attendance` akan menerima `403 CONSENT_REQUIRED` dan menampilkan
keadaan yang menjelaskannya (§ 6.3 E14) — bukan pesan error.

---

### 2.7 Menangani `FACE_BELONGS_TO_ANOTHER_EMPLOYEE`

[Fase 3 § 2.6](04-Fase3.md#26-d16--deteksi-wajah-duplikat-antar-karyawan--butuh-konfirmasi)
menetapkan bahwa response **tidak menyebut** karyawan lain itu siapa — membocorkannya
berarti memberi informasi kepada orang yang mungkin justru pelakunya.

UI mengikuti persis:

> *"Wajah ini sudah terdaftar pada akun karyawan lain. Pendaftaran tidak dapat
> dilanjutkan. Silakan hubungi HR untuk pemeriksaan."*

Tanpa nama, tanpa nomor karyawan, tanpa tingkat kemiripan. Tidak ada tombol "paksa
lanjutkan" di UI karyawan — `force_duplicate` hanya tersedia bagi pemegang
`face.enroll_any`, dan itu berarti lewat admin di Fase 5.

---

### 2.8 Riwayat pribadi — dan yang sengaja tidak ada di sana

Memakai #56 dan #57, yang mengembalikan **DTO employee**.
[Fase 4 § 2.7](05-Fase4.md#27-anti-penyalahgunaan) menyatakan alasannya: memberi tahu
seseorang bahwa skornya "0,41 sedangkan ambangnya 0,45" adalah memberi umpan balik
terukur kepada orang yang sedang mencoba menipu sistem.

Konsekuensi di kode: `types/dto/attendance.ts` mendefinisikan
`AttendanceEmployeeDTO` **tanpa** field `matched_similarity`, `threshold_used`,
`model_version`, `quality_score`, `distance_meter`, dan `gps_accuracy_meter`. Halaman
riwayat mengetik ulang field itu akan **gagal di `tsc`**, bukan lolos diam-diam.

Yang ditampilkan: tanggal kerja, tipe, jam lokal (`<LocalTime>`), status, alasan
fallback yang diterjemahkan, lokasi kantor, catatan, foto (`<AuthImage>`), serta
`review_note` bila ditolak — karena karyawan berhak tahu alasan penolakan.

Ringkasan bulan berjalan dihitung dari `meta.total` dengan filter, bukan dari endpoint
agregat baru.

---

## 3. Struktur Halaman, Routing & State

### 3.1 Route

Semua di dalam `AppShell` milik [Fase 5 § 3.4](06-Fase5.md#34-layout--komponen-bersama),
dengan varian layout lebih sederhana (tanpa sidebar admin) untuk pengguna yang hanya
punya permission karyawan.

```
/me                     redirect → /me/attendance
/me/attendance          attendance.checkin      check-in & check-out
/me/enrollment          face.enroll_self        wizard sesi bertahap
/me/consent             auth                    baca / setujui / cabut
/me/history             attendance.read_self    riwayat pribadi
/me/profile             employee.read_self      data karyawan (read-only) + /account
```

Placeholder `/me/*` yang dibuat Fase 5 (§ 3.2) diganti isinya di fase ini.

Admin yang juga karyawan (punya `attendance.checkin`) mendapat tautan "Absensi Saya"
di topbar — ia memang perlu absen juga.

### 3.2 State

| State | Alat |
|---|---|
| `/attendances/context` (#55) | TanStack Query, `staleTime: 30s`, `refetchOnWindowFocus` |
| Status enrollment (#48) | TanStack Query, invalidasi setelah setiap mutasi sesi |
| Stream kamera | `useCamera()` — hook dengan `useRef<MediaStream>`, **bukan** state global |
| Posisi GPS | `useGeolocation()` — di-*refresh* bila > 60 detik |
| Blob foto & object URL | State lokal komponen + `revokeObjectURL` di cleanup |
| Sesi enrollment | Server sebagai sumber kebenaran (#39); client hanya menyimpan blob pratinjau |
| Langkah wizard | Turunan dari status server, **bukan** state langkah independen (§ 6.4 E19) |

Kamera **tidak** disimpan di context global. Stream yang hidup lebih lama dari
halamannya adalah lampu kamera yang menyala di laptop orang setelah ia pindah
halaman — dan itu wajar dianggap sebagai pelanggaran privasi.

```ts
useEffect(() => {
  let stream: MediaStream | null = null;
  let cancelled = false;
  (async () => {
    stream = await navigator.mediaDevices.getUserMedia(constraints);
    if (cancelled) { stream.getTracks().forEach(t => t.stop()); return; }
    videoRef.current!.srcObject = stream;
  })();
  return () => { cancelled = true; stream?.getTracks().forEach(t => t.stop()); };
}, []);
```

Cabang `cancelled` menangani unmount cepat: `getUserMedia` bisa selesai **setelah**
komponen dilepas, dan tanpa cabang itu stream-nya tidak akan pernah dihentikan.

### 3.3 Komponen

| Komponen | Fungsi |
|---|---|
| `<CameraCapture>` | Preview tercermin, overlay oval, tombol ambil, kompresi adaptif (§ 2.2) |
| `<CameraGate>` | Menangani `isSecureContext` + seluruh error `getUserMedia` (§ 6.2) |
| `<LocationGate>` | Perolehan & status akurasi GPS (§ 2.3) |
| `<CapturePreview>` | Pratinjau blob lokal + [Ambil ulang] / [Kirim] |
| `<HintCoach>` | `hints` → kalimat tindakan; memakai `lib/errors/hints.ts` Fase 5 |
| `<AttemptResultScreen>` | Layar hasil percobaan: berhasil / menunggu / gagal + opsi fallback |
| `<EnrollmentWizard>` | Orkestrasi Langkah 0–5 (§ 2.5) |
| `<PhotoSlot>` | Kartu foto: kosong / diproses / diterima / ditolak+hints |
| `<SessionCountdown>` | Hitung mundur `expires_at` |
| `<ConsentReader>` | Markdown tersanitasi + gate scroll + checkbox |
| `<TodayStatusCard>` | Ringkasan hari ini dari #57 |
| `<AttendanceHistoryList>` | Riwayat (#56), DTO employee |

Dipakai ulang dari Fase 5 tanpa perubahan: `<AuthImage>`, `<LocalTime>`,
`<EmptyState>`, `<ErrorState>`, `<ConfirmDialog>`, `<Can>`, `ERROR_MESSAGES`,
`hints.ts`, `api.ts` + refresh lock.

---

## 4. Endpoint yang Dikonsumsi

**Tidak ada endpoint baru.** Seluruhnya sudah ada sejak Fase 1/3/4.

### 4.1 Auth & profil — Fase 1

| # | Method | Path | Dipakai di | DTO |
|---|---|---|---|---|
| 1–5 | — | `/auth/*` | Shell Fase 5 | — |
| 6 | POST | `/auth/change-password` | `/me/profile` | — |
| 9 | GET | `/employees/me` | `/me/profile` | `EmployeeDTO` |
| 28 | GET | `/settings` | jam kerja tampilan | hanya baris `is_public = true` ([Fase 1 § 4.6](02-Fase1.md#46-app-settings)) |

### 4.2 Consent & enrollment — Fase 3

| # | Method | Path | Dipakai di | Catatan |
|---|---|---|---|---|
| 32 | GET | `/consents/document` | `<ConsentReader>` | `body` Markdown → **sanitasi wajib** |
| 33 | GET | `/consents/me` | `/me/consent` | |
| 34 | POST | `/consents` | `<ConsentReader>` | `{document_version, agreed:true}` |
| 35 | POST | `/consents/withdraw` | `/me/consent` | `{reason}`; dialog akibat (§ 2.6) |
| 38 | POST | `/face/enrollments` | Wizard L2 | `{mode:"replace"}`; tangani `409` + `existing_session_id` |
| 39 | GET | `/face/enrollments/{id}` | Wizard | sumber kebenaran langkah |
| 40 | POST | `/face/enrollments/{id}/photos` | Wizard L3 | multipart `image` + `capture_source=web_camera` |
| 41 | DELETE | `/face/enrollments/{id}/photos/{pid}` | `<PhotoSlot>` | |
| 42 | POST | `/face/enrollments/{id}/commit` | Wizard L5 | tanpa `force_duplicate` (§ 2.7) |
| 43 | DELETE | `/face/enrollments/{id}` | Wizard batal | |
| 44 | GET | `/employees/{id}/face-references` | `/me/enrollment` (milik sendiri) | guard `face.read_self` + pola ownership |
| 45 | GET | `/face/references/{id}/photo` | `<AuthImage>` | |
| 48 | GET | `/face/enrollment-status/me` | Wizard L0, `/me/attendance` | satu panggilan penentu langkah |

### 4.3 Absensi — Fase 4

| # | Method | Path | Dipakai di | DTO |
|---|---|---|---|---|
| 53 | POST | `/attendances/check-in` | `/me/attendance` | multipart; `allow_fallback`; `Idempotency-Key` |
| 54 | POST | `/attendances/check-out` | `/me/attendance` | idem |
| 55 | GET | `/attendances/context` | `/me/attendance` | **sumber seluruh konfigurasi UI** — jangan hardcode apa pun darinya |
| 56 | GET | `/attendances/me` | `/me/history` | **`AttendanceEmployeeDTO`** |
| 57 | GET | `/attendances/me/today` | `<TodayStatusCard>` | idem |
| 58 | GET | `/attendances/{id}` | detail riwayat | DTO employee (pemanggil hanya punya `attendance.read_self`) |
| 59 | GET | `/attendances/{id}/photo` | `<AuthImage>` | tangani `410` |

**Tidak dipakai Fase 6:** #60, #61, #62, #63, #64, #65, #66–#70, #71, #72 — seluruhnya
dijaga permission yang tidak dimiliki role `employee`
([Fase 1 § 2.4](02-Fase1.md#24-role-default)).

### 4.4 Semua yang dibaca dari `/attendances/context` (#55)

Ditulis eksplisit karena inilah yang mencegah asumsi ter-hardcode:

| Field | Dipakai untuk |
|---|---|
| `server_time`, `timezone` | Jam yang ditampilkan; **tidak pernah** memakai jam perangkat |
| `work_date`, `today.*`, `next_action` | Menentukan tombol: Check-in / Check-out / sudah lengkap |
| `attendance_mode` | Mode manual (§ 2.4) |
| `is_enrolled`, `active_reference_count`, `needs_re_enrollment` | Ajakan enroll / enroll ulang |
| `consent_status` | Gerbang consent |
| `geofence.enabled` | Meminta lokasi atau tidak |
| `geofence.outside_policy` **(D18)** | Peringatan sebelum kirim saat di luar radius |
| `geofence.max_gps_accuracy_meter` | Ambang blokir/peringatan akurasi (§ 2.3) |
| `geofence.nearest_location` | Peta kecil + jarak indikatif |
| `photo.max_bytes`, `photo.accepted_mime_types` **(D24)** | Target kompresi & tipe MIME |
| `fallback_enabled` | Apakah tombol "kirim untuk ditinjau" mungkin muncul |
| `require_face_for_checkout` **(D19)** | Untuk check-out: apakah kamera perlu dinyalakan sama sekali |
| `checkout_without_face_status` | Menampilkan status yang **akan** dihasilkan (`pending_review` atau `approved`) sebelum karyawan menekan tombol, saat `require_face_for_checkout = false` |
| `max_note_length` | Validasi client sebelum kirim, mencerminkan validator server |

`require_face_for_checkout`, `checkout_without_face_status`, dan
`max_note_length` dibaca **dari `#55` ini, bukan dari `#28 GET /settings`** —
[REV-EP-04](09-Revisions-Log.md#b-perubahan-katalog-endpoint) sudah dieksekusi
sebagai resolusi [K-01](09-Revisions-Log.md#k-01--checkout_without_face_status-tidak-dapat-dibaca-karyawan--kontradiksi--resolved):
`attendance.checkout_without_face_status` tetap `is_public = false`, sehingga
`#28` **tidak pernah** mengembalikannya untuk karyawan biasa. Halaman ini
**tidak boleh** memanggil `#28` untuk field ini.

### 4.5 Pembuktian kontrak

Fase 6 tidak membutuhkan satu pun endpoint baru, satu pun permission baru, dan satu
pun error code baru. Semua yang dibutuhkan halaman karyawan sudah tersedia dari
Fase 1/3/4 — termasuk #55 yang memang dirancang
[Fase 4 § 4.4](05-Fase4.md#44-get-attendancescontext) untuk "menjawab semua yang
dibutuhkan halaman absensi sebelum kamera dinyalakan". Ini menjadi verifikasi bahwa
kontrak backend disusun dengan client ini dalam pikiran, bukan sekadar diasumsikan cukup.

---

## 5. Flow per Fitur

### 5.1 Membuka `/me/attendance`

```
1. isSecureContext = false ⇒ layar penjelasan HTTPS, berhenti (§ 2.1)
2. #55 GET /attendances/context  (paralel dengan langkah 3)
3. Minta izin lokasi bila context.geofence.enabled
4. Evaluasi keadaan, berurutan — yang pertama cocok menang:
     consent_status ≠ "granted" && attendance_mode = "face"
         ⇒ layar "Persetujuan diperlukan" + tautan /me/consent
     attendance_mode = "manual"
         ⇒ mode manual (§ 2.4): kamera tetap, pesan berbeda
     !is_enrolled || needs_re_enrollment
         ⇒ layar ajakan enroll + tautan /me/enrollment
     today.check_in && today.check_out
         ⇒ layar "Absensi hari ini lengkap" + ringkasan
     next_action = "check_in" | "check_out"
         ⇒ layar siap
5. Nyalakan kamera (§ 3.2) — hanya setelah langkah 4 memutuskan layar siap.
   Menyalakan kamera pada layar yang tidak membutuhkannya adalah lampu kamera
   yang menyala tanpa alasan.
6. Tampilkan: jam server (dari server_time, berjalan lokal), lokasi + jarak,
   status geofence, tombol aksi
```

Urutan langkah 4 penting: karyawan mode manual yang belum enroll harus melihat pesan
mode manual (ia memang tidak perlu enroll), bukan ajakan enroll yang menyesatkan.

### 5.2 Mengirim check-in

```
[Kirim] ditekan
 ├─ Validasi client:
 │    lokasi belum didapat & geofence aktif & policy reject ⇒ blokir + [Ambil ulang lokasi]
 │    akurasi > ambang & policy reject                      ⇒ blokir + penjelasan
 │    blob > photo.max_bytes                                ⇒ kompresi ulang (§ 2.2)
 ├─ idempotencyKey = ulid()
 ├─ FormData: image, lat, lng, gps_accuracy_meter, capture_source="web_camera",
 │            note?, allow_fallback=false, client_reported_at=<ISO lokal>
 │            header Idempotency-Key
 ├─ Kirim #53. Indikator: "Memverifikasi wajah…" (bukan spinner tanpa teks —
 │    proses ini bisa 1–2 detik dan orang perlu tahu ada yang berjalan)
 └─ Percabangan sesuai § 2.4
```

`client_reported_at` dikirim meski server mengabaikannya untuk penentuan waktu —
[Fase 4 § 2.2](05-Fase4.md#22-waktu-server--konsep-work_date) memakainya untuk
menghitung `clock_skew_seconds` sebagai telemetri. Client tidak perlu tahu lebih dari itu.

### 5.3 Layar percobaan gagal

```
<AttemptResultScreen variant="failed">
  Judul       ← ERROR_MESSAGES[code].title
  Penjelasan  ← ERROR_MESSAGES[code].body
  <HintCoach hints={error.hints} />        // mis. too_dark → "Cari tempat lebih terang"
  [Coba lagi]                              // selalu ada; kembali ke layar siap
  [Kirim untuk ditinjau admin]             // HANYA bila error.can_fallback === true
       → tooltip: "Absensi akan tercatat dengan status menunggu persetujuan."
       → menekan ini membuka kamera lagi untuk foto BARU (§ 2.4)
</AttemptResultScreen>
```

Urutan tombol disengaja: "Coba lagi" lebih menonjol daripada "Kirim untuk ditinjau".
Memperbaiki pencahayaan lalu berhasil `approved` lebih baik bagi semua pihak daripada
menambah satu baris ke antrian reviewer.

### 5.4 Wizard enrollment

Alur di § 2.5. Yang perlu ditegaskan di implementasi:

- **Langkah wizard adalah turunan dari `GET /face/enrollments/{id}` (#39)**, bukan
  penghitung lokal. Kalau ada state langkah tersendiri, *reload* di tengah sesi akan
  mengembalikan pengguna ke langkah 1 sementara server masih menyimpan 2 foto —
  dan pengguna akan mengambil foto ketiga lalu bingung karena commit-nya berhasil
  dengan foto yang tidak ia ingat.
- Setiap `422 FACE_NOT_USABLE` **tidak** menyisakan apa pun di server
  ([Fase 3 E10](04-Fase3.md#62-kualitas--inference)); UI menampilkan kartu merah
  yang bisa langsung diambil ulang, tanpa perlu menghapus apa pun.
- Tombol "Simpan" aktif hanya bila `accepted_count >= required_photos`; nilai
  keduanya dibaca dari server, tidak dihitung ulang.

### 5.5 Riwayat

```
/me/history
 ├─ <TodayStatusCard>  ← #57
 ├─ Filter: bulan (default bulan berjalan), tipe, status — tersimpan di URL
 ├─ Daftar ← #56, DTO employee
 │     baris: tanggal, jam lokal, tipe, badge status, lokasi, ikon foto
 │     status=rejected ⇒ tampilkan review_note (karyawan berhak tahu alasannya)
 │     status=pending_review ⇒ "Menunggu persetujuan admin"
 └─ Klik baris → detail (#58) + foto (#59, <AuthImage>, tangani 410)
```

---

## 6. Edge Case, Validasi & State

### 6.1 Prasyarat lingkungan

| # | Kondisi | Penanganan |
|---|---|---|
| E1 | `isSecureContext === false` | Layar penjelasan HTTPS (§ 2.1). **Jangan** memanggil `getUserMedia` lalu menampilkan error mentahnya |
| E2 | `navigator.mediaDevices === undefined` | Sama seperti E1 — browser lama atau konteks tidak aman |
| E3 | `navigator.geolocation === undefined` | Bila geofence aktif: jelaskan browser tidak mendukung lokasi; bila tidak aktif: lanjut |
| E4 | Browser sangat lama (tanpa `canvas.toBlob`) | Layar "Browser tidak didukung" dengan daftar browser yang didukung |
| E5 | `navigator.locks` tidak ada | Fallback lock Fase 5 (§ 2.2b Fase 5); tidak memengaruhi Fase 6 |

### 6.2 Kamera

| # | Kondisi (`error.name`) | Penanganan |
|---|---|---|
| E6 | `NotAllowedError` | *"Akses kamera ditolak. Izinkan kamera pada ikon gembok di address bar, lalu muat ulang."* + panduan singkat per browser |
| E7 | `NotFoundError` / `DevicesNotFoundError` | *"Tidak ada kamera terdeteksi pada perangkat ini."* |
| E8 | `NotReadableError` / `TrackStartError` | *"Kamera sedang dipakai aplikasi lain (mis. Zoom/Meet). Tutup aplikasi itu lalu coba lagi."* — penyebab paling sering di laptop kantor |
| E9 | `OverconstrainedError` | Ulangi tanpa constraint resolusi; bila tetap gagal, tampilkan pesan |
| E10 | `AbortError` | Coba sekali lagi otomatis, lalu tampilkan pesan |
| E11 | Izin dicabut saat halaman terbuka | Track berhenti (`onended`) → kembali ke `<CameraGate>` dengan pesan E6 |
| E12 | Pengguna berpindah halaman saat kamera hidup | Cleanup `useEffect` menghentikan semua track (§ 3.2). **Diuji** (§ 11.5) |
| E13 | Beberapa kamera (webcam + kamera USB) | `enumerateDevices()` → pemilih perangkat bila > 1; pilihan disimpan di `localStorage` |
| E14 | Preview tercermin ikut terkirim | Dilarang (§ 2.2). Diuji dengan membandingkan piksel penanda (§ 11.4) |

### 6.3 Lokasi

| # | Kondisi | Penanganan |
|---|---|---|
| E15 | `PERMISSION_DENIED` | Bila `missing_location_policy = reject`: blokir kirim + panduan mengizinkan lokasi. Bila `pending_review`: izinkan dengan peringatan |
| E16 | `POSITION_UNAVAILABLE` | Tombol [Ambil ulang lokasi]; jangan mengirim `lat`/`lng` kosong secara diam-diam |
| E17 | `TIMEOUT` (10 dtk) | Sama seperti E16, dengan saran mendekati jendela |
| E18 | Akurasi buruk | § 2.3 — perilaku mengikuti `missing_location_policy` |
| E19 | Lokasi basi (> 60 dtk saat kirim) | Ambil ulang otomatis sebelum mengirim |
| E20 | Di luar radius | Jarak indikatif ditampilkan **sebelum** kirim; bila `outside_policy = reject`, tombol kirim dinonaktifkan dengan penjelasan; server tetap penentu |

### 6.4 Alur absensi & enrollment

| # | Kondisi | Penanganan |
|---|---|---|
| E21 | Sudah check-in hari ini, halaman dibuka lagi | Dari #55 `today.check_in` → layar "sudah check-in" + tombol Check-out. `409` tidak akan pernah terjadi dalam pemakaian normal |
| E22 | Dua tab menekan kirim bersamaan | Satu `201`, satu `409 ALREADY_CHECKED_IN` → layar "sudah absen" + `refetch`. **Bukan** toast merah |
| E23 | Jaringan putus saat mengirim | *"Koneksi terputus. Absensi mungkin sudah tercatat."* + [Coba kirim lagi] dengan **`Idempotency-Key` yang sama** (§ 2.4). Server mengembalikan record yang sama, bukan duplikat |
| E24 | Sesi enrollment kedaluwarsa saat memotret | `409 ENROLLMENT_SESSION_EXPIRED` → tawarkan mulai ulang; foto yang sudah diterima hilang, dan UI mengatakannya sebelum pengguna mengulang |
| E25 | Model berubah di tengah sesi | `409 ENROLLMENT_MODEL_CHANGED` → *"Sistem pengenalan wajah telah diperbarui. Pendaftaran harus diulang."* |
| E26 | *Reload* di tengah wizard | Langkah dipulihkan dari #39 (§ 5.4). Blob pratinjau lokal hilang — kartu foto yang sudah diterima ditampilkan sebagai "Foto N — diterima" tanpa gambar, dan itu jujur |
| E27 | Consent dicabut di perangkat lain | Request berikutnya `403 CONSENT_REQUIRED` → layar consent |
| E28 | Reindex berjalan | `409 REINDEX_IN_PROGRESS` → pesan pemeliharaan, bukan error |

### 6.5 Keamanan & kepatuhan

| # | Kondisi | Penanganan |
|---|---|---|
| E29 | Foto dari layar/foto cetak (spoof) | **Tidak terdeteksi Fase 6.** Yang ada: paksa kamera langsung, tanpa input berkas, dan telemetri percobaan Fase 4. Liveness sungguhan di Fase 7. Dinyatakan, bukan disembunyikan |
| E30 | Kamera virtual (OBS, ManyCam) | Tidak dapat dibedakan dari kamera nyata oleh `getUserMedia`. Sama seperti E29 — batasan yang diketahui |
| E31 | Satu perangkat dipakai banyak karyawan (kios) | Setiap orang harus login. Sesi sebelumnya **wajib** dibersihkan: `/me/attendance` menyediakan [Selesai & keluar] yang memanggil #3 dan mengosongkan cache. Mode kios sungguhan di luar scope |
| E32 | Markdown consent memuat HTML/skrip | **Sanitasi wajib** (mis. `dompurify`) sebelum render, dengan allowlist tag. Ini satu-satunya tempat di aplikasi yang merender HTML dari server; ia menjadi permukaan XSS yang paling masuk akal, dan token refresh ada di `localStorage` ([Fase 5 D23](06-Fase5.md#c-d23--penyimpanan-token--butuh-konfirmasi)) |
| E33 | Blob foto tertinggal di memori | `revokeObjectURL` di cleanup; blob dilepas setelah kirim berhasil. Tidak pernah ditulis ke `localStorage`/IndexedDB |
| E34 | `matched_similarity` bocor ke halaman karyawan | Tidak ada di `AttendanceEmployeeDTO`; mengetiknya gagal `tsc` (§ 2.8). Diuji juga di runtime (§ 11.7) |
| E35 | Ada `<input type="file">` di jalur foto harian | **Dilarang** master plan § 9. Diuji: nol elemen file input di `/me/attendance` dan `/me/enrollment` (§ 11.3) |

### 6.6 Keadaan kosong & jujur

| Layar | Keadaan | Kalimat |
|---|---|---|
| `/me/attendance` | mode manual | *"Akun Anda diatur pada mode absensi manual. Foto Anda disimpan sebagai bukti dan absensi akan ditinjau admin. Tidak ada pencocokan wajah otomatis."* |
| `/me/attendance` | belum enroll | *"Anda belum mendaftarkan wajah. Daftarkan wajah dulu agar bisa absen."* + [Daftarkan wajah] |
| `/me/attendance` | `needs_re_enrollment` | *"Sistem pengenalan wajah telah diperbarui. Silakan daftarkan ulang wajah Anda."* |
| `/me/attendance` | consent belum ada | *"Diperlukan persetujuan pemrosesan data biometrik sebelum Anda dapat absen."* + [Baca persetujuan] |
| `/me/attendance` | consent dicabut | *"Anda telah mencabut persetujuan. Absensi wajah tidak tersedia. Hubungi HR untuk jalur absensi alternatif."* (§ 2.6) |
| `/me/attendance` | sudah lengkap hari ini | *"Absensi hari ini sudah lengkap."* + ringkasan jam & durasi |
| `/me/attendance` | `503 FACE_SERVICE_NOT_CONFIGURED` \| `ATTENDANCE_NOT_CONFIGURED` | *"Sistem belum dikonfigurasi. Hubungi admin."* — dua kode, satu pesan (K-04) |
| `/me/history` | belum ada riwayat | *"Belum ada riwayat absensi."* |
| `/me/history` | filter tidak menemukan | *"Tidak ada absensi pada rentang ini."* + [Reset filter] |
| `/me/enrollment` | sudah terdaftar | *"Wajah Anda sudah terdaftar (N foto)."* + [Daftar ulang] |
| `/me/enrollment` | mode manual | *"Akun Anda pada mode absensi manual, sehingga pendaftaran wajah tidak diperlukan."* |

Tidak satu pun di antaranya berbunyi "Terjadi kesalahan". Semuanya adalah keadaan
yang sah dalam desain sistem ini, dan menyebutnya kesalahan akan membuat karyawan
melapor ke IT untuk hal yang bekerja sebagaimana mestinya.

### 6.7 Validasi client

| Field | Aturan |
|---|---|
| `image` | Wajib; JPEG hasil `canvas.toBlob`; ≤ `context.photo.max_bytes` |
| `note` | ≤ `attendance.max_note_length` dari `/settings`, penghitung karakter |
| `lat` / `lng` | Dari Geolocation API saja; tidak pernah bisa diketik pengguna |
| `capture_source` | Selalu `web_camera` literal; tidak pernah dari input |
| `allow_fallback` | Hanya `true` lewat tombol eksplisit di layar gagal |
| `Idempotency-Key` | ULID; aturan siklus hidup di § 2.4 |
| `agreed` (consent) | Harus `true` dari checkbox + gate scroll |

Baris `lat`/`lng` layak ditegaskan: tidak ada field input koordinat di UI karyawan.
Menyediakannya berarti membuat pemalsuan lokasi tidak memerlukan alat sama sekali.

---

## 7. Struktur Folder

Melanjutkan struktur [Fase 5 § 7](06-Fase5.md#7-struktur-folder).

```
apps/faceclock-web/
├── src/
│   ├── app/
│   │   ├── routes.config.ts           # (diubah) route /me/* jadi nyata
│   │   └── EmployeeShell.tsx           # ← BARU: layout ringkas untuk karyawan
│   │
│   ├── lib/
│   │   ├── media/                      # ← BARU
│   │   │   ├── camera.ts               # constraints, pemetaan error getUserMedia
│   │   │   ├── capture.ts              # canvas → JPEG + kompresi adaptif (§ 2.2)
│   │   │   ├── luminance.ts            # pra-peringatan gelap/terang (D25)
│   │   │   └── objectUrl.ts            # helper create/revoke terlacak
│   │   ├── geo/                        # ← BARU
│   │   │   ├── geolocation.ts          # getCurrentPosition + pemetaan error
│   │   │   └── haversine.ts            # jarak indikatif (cerminan internal/geo Go)
│   │   ├── idempotency.ts              # ← BARU: ULID + siklus hidup kunci
│   │   └── markdown.ts                 # ← BARU: render + sanitasi (E32)
│   │
│   ├── hooks/
│   │   ├── useCamera.ts                # ← BARU: stream + cleanup ketat (§ 3.2)
│   │   ├── useGeolocation.ts           # ← BARU
│   │   ├── useCapture.ts               # ← BARU: blob + object URL + revoke
│   │   └── useCountdown.ts             # ← BARU
│   │
│   ├── components/
│   │   ├── camera/                     # ← BARU
│   │   │   ├── CameraGate.tsx
│   │   │   ├── CameraCapture.tsx
│   │   │   ├── CameraDevicePicker.tsx
│   │   │   ├── FaceFrameOverlay.tsx
│   │   │   └── CapturePreview.tsx
│   │   ├── location/                   # ← BARU
│   │   │   ├── LocationGate.tsx
│   │   │   └── LocationStatus.tsx
│   │   └── domain/
│   │       ├── HintCoach.tsx           # ← BARU
│   │       └── AttendanceStatusBadge.tsx   # (dipakai ulang dari Fase 5)
│   │
│   ├── features/
│   │   ├── me-attendance/              # ← BARU
│   │   │   ├── pages/AttendancePage.tsx
│   │   │   ├── components/
│   │   │   │   ├── ReadyScreen.tsx
│   │   │   │   ├── AttemptResultScreen.tsx
│   │   │   │   ├── FallbackPrompt.tsx
│   │   │   │   ├── TodayStatusCard.tsx
│   │   │   │   ├── ManualModeNotice.tsx
│   │   │   │   └── ServerClock.tsx
│   │   │   ├── useCheckInFlow.ts       # mesin keadaan alur § 2.4
│   │   │   └── api.ts                  # #53, #54, #55, #57
│   │   ├── me-enrollment/              # ← BARU
│   │   │   ├── pages/EnrollmentPage.tsx
│   │   │   ├── components/
│   │   │   │   ├── EnrollmentWizard.tsx
│   │   │   │   ├── PhotoSlot.tsx
│   │   │   │   ├── PhotoGuidance.tsx   # panduan D26
│   │   │   │   └── SessionCountdown.tsx
│   │   │   ├── useEnrollmentSession.ts
│   │   │   └── api.ts                  # #38–#44, #48
│   │   ├── me-consent/                 # ← BARU
│   │   │   ├── pages/ConsentPage.tsx
│   │   │   ├── components/{ConsentReader,WithdrawDialog}.tsx
│   │   │   └── api.ts                  # #32, #33, #34, #35
│   │   ├── me-history/                 # ← BARU
│   │   │   ├── pages/{HistoryPage,HistoryDetailPage}.tsx
│   │   │   └── api.ts                  # #56, #57, #58, #59
│   │   └── me-profile/                 # ← BARU  (#9, #6)
│   │
│   ├── types/dto/
│   │   ├── attendance.ts               # (diubah) AttendanceEmployeeDTO dipakai di sini
│   │   └── enrollment.ts               # ← BARU: sesi, foto, status
│   │
│   └── test/msw/handlers/
│       ├── fase3.ts                    # ← BARU
│       └── fase4-employee.ts           # ← BARU
│
├── e2e/
│   ├── checkin-happy.spec.ts           # ← BARU
│   ├── checkin-fallback.spec.ts        # ← BARU
│   ├── enrollment-wizard.spec.ts       # ← BARU
│   ├── consent.spec.ts                 # ← BARU
│   ├── camera-errors.spec.ts           # ← BARU
│   ├── no-file-input.spec.ts           # ← BARU  (penegakan master plan § 9)
│   └── fixtures/
│       ├── face-ok.y4m                 # video palsu untuk Chromium
│       ├── face-dark.y4m
│       └── no-face.y4m
└── playwright.config.ts                # (diubah) flag kamera palsu
```

`lib/geo/haversine.ts` sengaja mencerminkan `internal/geo/haversine.go`
([Fase 4 § 7](05-Fase4.md#7-struktur-folder)). Keduanya harus memberi jawaban yang
sama untuk masukan yang sama — kalau tidak, jarak yang ditampilkan sebelum kirim akan
berbeda dari yang dihitung server, dan pengguna akan melihat "23 m" lalu ditolak
karena di luar radius. Test membandingkan keduanya pada set koordinat yang sama (§ 11.6).

---

## 8. Checklist Task

### 8.0 Prasyarat
- [ ] **Konfirmasi D24–D26** (§ 2.0)
- [ ] **HTTPS tersedia** untuk dev/staging (§ 2.1) — memblokir seluruh pengujian nyata
- [ ] Fase 5 selesai: app shell, refresh lock, `<AuthImage>`, `ERROR_MESSAGES`, `hints.ts`
- [ ] Data uji: karyawan mode `face` (ter-enroll & belum), karyawan mode `manual`,
      karyawan dengan consent dicabut

### 8.1 Fondasi media & lokasi
- [ ] `lib/media/camera.ts` — constraints + pemetaan 5 `error.name` ke pesan (E6–E10)
- [ ] `lib/media/capture.ts` — kanvas **tanpa** `ctx.scale(-1,1)`, kompresi adaptif (§ 2.2)
- [ ] `lib/media/luminance.ts` (D25) — peringatan saja, tidak memblokir
- [ ] `lib/media/objectUrl.ts` — pembuatan & pelepasan terlacak
- [ ] `lib/geo/geolocation.ts` — `maximumAge: 0`, pemetaan 3 kode error
- [ ] `lib/geo/haversine.ts` + test kesetaraan dengan implementasi Go
- [ ] `lib/idempotency.ts` — ULID + aturan siklus hidup (§ 2.4)
- [ ] `lib/markdown.ts` — render + **sanitasi** (E32)
- [ ] `useCamera` dengan cleanup ketat + cabang `cancelled` (§ 3.2)
- [ ] `useGeolocation`, `useCapture`, `useCountdown`
- [ ] Deteksi `isSecureContext` + layar penjelasan (E1/E2)

### 8.2 Komponen kamera
- [ ] `<CameraGate>` — semua keadaan izin/perangkat
- [ ] `<CameraCapture>` — preview tercermin CSS, tombol ambil
- [ ] `<FaceFrameOverlay>` — oval panduan (mengurangi `face_too_small`)
- [ ] `<CameraDevicePicker>` bila > 1 perangkat (E13)
- [ ] `<CapturePreview>` — [Ambil ulang] / [Kirim]
- [ ] `<HintCoach>` — 9 hint → kalimat tindakan, memakai `hints.ts` Fase 5

### 8.3 Check-in / check-out
- [ ] `/me/attendance` + `<EmployeeShell>`
- [ ] Evaluasi keadaan berurutan (§ 5.1 langkah 4) — 7 keadaan di § 6.6
- [ ] `<ServerClock>` dari `context.server_time` (**bukan** jam perangkat)
- [ ] `<LocationStatus>` — jarak indikatif + status akurasi (§ 2.3)
- [ ] `useCheckInFlow` — mesin keadaan § 2.4
- [ ] Kirim #53/#54 dengan `Idempotency-Key`
- [ ] `<AttemptResultScreen>` untuk berhasil / menunggu / gagal
- [ ] `<FallbackPrompt>` — **hanya** bila `error.can_fallback`, dan **wajib** foto baru
- [ ] **Verifikasi: tidak ada tombol absen manual yang berdiri sendiri** (§ 2.4)
- [ ] Mode manual: satu kali kirim, pesan berbeda, tanpa tarian fallback
- [ ] Check-out: baca `require_face_for_checkout` (D19); tangani `CHECKOUT_TOO_SOON`
      dengan sisa menit dari `details`
- [ ] Retry jaringan memakai **kunci idempotensi yang sama** (E23)

### 8.4 Enrollment
- [ ] `/me/enrollment` + wizard 6 langkah (§ 2.5)
- [ ] Langkah diturunkan dari #39, **bukan** state lokal (E26)
- [ ] `<PhotoSlot>` 4 keadaan: kosong / diproses / diterima / ditolak+hints
- [ ] `<PhotoGuidance>` sesuai D26 — **bukan** "hadap kiri/kanan"
- [ ] `<SessionCountdown>` + peringatan 5 menit terakhir
- [ ] Lanjutkan sesi dari `409 CONFLICT` + `existing_session_id`
- [ ] Tangani `DUPLICATE_PHOTO`, `ENROLLMENT_LIMIT_REACHED`, `ENROLLMENT_INCOMPLETE`,
      `ENROLLMENT_MODEL_CHANGED`, `ENROLLMENT_SESSION_EXPIRED`, `REINDEX_IN_PROGRESS`
- [ ] `FACE_BELONGS_TO_ANOTHER_EMPLOYEE` — pesan **tanpa** menyebut karyawan lain (§ 2.7)
- [ ] Daftar referensi aktif (#44) + foto (#45, `<AuthImage>`)
- [ ] Alur daftar ulang (`mode=replace`)

### 8.5 Consent
- [ ] `/me/consent` — status (#33) + dokumen (#32)
- [ ] `<ConsentReader>` — Markdown tersanitasi + gate scroll + checkbox eksplisit
- [ ] Setujui (#34); tangani `CONSENT_VERSION_OUTDATED`, `CONSENT_ALREADY_GRANTED`
- [ ] Cabut (#35) dengan dialog akibat + arahan ke HR (§ 2.6)

### 8.6 Riwayat & profil
- [ ] `/me/history` — #56 dengan filter ter-URL, **DTO employee**
- [ ] `<TodayStatusCard>` (#57)
- [ ] Detail (#58) + foto (#59) + penanganan `410`
- [ ] `review_note` ditampilkan pada status `rejected`
- [ ] `/me/profile` (#9) + ganti password (#6)
- [ ] Verifikasi: tidak ada field skor di seluruh ruang `/me/*`

### 8.7 Kualitas & penutup
- [ ] MSW handlers untuk seluruh endpoint § 4
- [ ] Unit test: `capture.ts` (tidak tercermin, kompresi), `camera.ts` (pemetaan error),
      `haversine` (setara Go), `idempotency`
- [ ] Playwright: kamera palsu + geolocation (§ 11.2)
- [ ] E2E: happy path, fallback, enrollment, consent, error kamera, **no-file-input**
- [ ] Uji di perangkat nyata lewat HTTPS: Chrome desktop, Chrome Android, Safari iOS
- [ ] `tsc --noEmit`, ESLint, Prettier lulus; CI hijau
- [ ] `docs/web/fase6-employee.md`
- [ ] `DONE-Fase-6.md` sesuai Protokol Handoff master plan § 10.4
- [ ] **Konfirmasi milestone "Web fully working"** sebelum Fase 7 dimulai

---

## 9. Dependencies

**Prasyarat:**

| Dari | Yang dibutuhkan |
|---|---|
| Fase 5 | Seluruh app shell: `api.ts` + refresh lock, `AuthProvider`, `<Can>`, `<RequirePermission>`, `<AuthImage>`, `ERROR_MESSAGES` (39 code), `hints.ts`, `<LocalTime>`, `<EmptyState>`, `<ErrorState>`, `queryClient` |
| Fase 4 | #53–#59; `context` (#55) sebagai sumber konfigurasi; D17 (flag fallback), D18, D19; `Idempotency-Key`; `AttendanceEmployeeDTO` |
| Fase 3 | #32–#35, #38–#44, #48; sesi bertahap (D15); aturan pesan duplikat (§ 2.7) |
| Fase 2 | Kosakata 9 `hints` + kalimat Indonesia; `face.max_image_bytes`, `accepted_mime_types` |
| Fase 1 | #1–#6, #9, #28; permission `attendance.checkin`, `attendance.read_self`, `face.enroll_self`, `face.read_self`, `employee.read_self` — **semuanya sudah dimiliki role `employee`** |
| Infrastruktur | **HTTPS** (§ 2.1) — blocker |
| Eksternal | `dompurify` (sanitasi Markdown), `ulid`; Playwright dengan flag kamera palsu |

**Utang lintas-fase:**

| Risiko | Status di Fase 6 |
|---|---|
| **R3** — spoofing foto-dari-layar | ✅ **Sebagian, sejauh yang bisa dilakukan web.** Paksa `getUserMedia`, tanpa input berkas, tanpa jalur unggah. Kamera virtual dan foto-dari-layar **tetap tidak terdeteksi** (E29/E30) — penutupannya di Fase 7 |
| **R1** — lisensi model InsightFace | ⛔ Masih terbuka. Tidak memblokir pengerjaan Fase 6, **memblokir rilis produksi** |
| **R2** — kalibrasi Dataset B | ⛔ Masih blocker rilis. Fase 6 justru memperbesar dampaknya: begitu karyawan memakainya setiap hari, threshold yang belum dikalibrasi akan langsung terlihat sebagai penolakan palsu massal atau — lebih buruk karena tidak terlihat — penerimaan palsu |
| R4, R5, R7 | ✅ Selesai (Fase 3/4), UI-nya di Fase 5 |
| **R6** — monorepo vs multi-repo | ✅ **Ditutup** — dikunci monorepo 2026-09-04 ([Fase 0 § 2.1](01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)) |

**Yang bergantung pada fase ini:**

| Fase | Mengambil apa |
|---|---|
| Fase 7 (Flutter) | Alur yang sama persis: sesi enrollment bertahap, jalur wajah-lalu-fallback, `Idempotency-Key`, kalimat `hints` yang sama, keadaan kosong yang sama. Fase 6 menjadi **rujukan perilaku** — perbedaan mobile hanya pada liveness on-device, kamera in-app, dan penanganan offline |

---

## 10. Definition of Done

1. **Tidak ada satu pun `<input type="file">`** di `/me/attendance` dan
   `/me/enrollment` — diuji otomatis (§ 11.3). Ini penegakan langsung master plan § 9.
2. **Tidak ada tombol absen manual yang berdiri sendiri.** Opsi "kirim untuk ditinjau"
   hanya dapat dicapai dari layar hasil percobaan wajah yang gagal — diuji dengan
   memeriksa bahwa tombol itu tidak ada di layar siap.
3. Frame yang dikirim **tidak tercermin**, sementara preview tercermin — diuji dengan
   video palsu berpenanda asimetris (§ 11.4).
4. Check-in berhasil: foto + lokasi terkirim, `201 approved`, layar berhasil
   menampilkan **jam server** (bukan jam perangkat) dan lokasi.
5. Wajah tidak cocok → layar gagal dengan alasan + `hints` yang diterjemahkan;
   tombol "kirim untuk ditinjau" muncul **hanya** bila `can_fallback`; menekannya
   **meminta foto baru**, lalu menghasilkan `pending_review`.
6. Inference dimatikan → layar gagal yang menjelaskan layanan bermasalah; fallback
   ditawarkan bila `can_fallback`; **tidak ada** jalur yang menghasilkan `approved`.
7. Semua 7 keadaan `/me/attendance` (§ 6.6) tampil benar, tanpa satu pun berbunyi
   "Terjadi kesalahan".
8. Mode manual: pesan yang benar, satu kali kirim, `pending_review`, dan **tidak ada**
   janji verifikasi wajah di layar.
9. Consent dicabut → `/me/attendance` menampilkan keadaan yang menjelaskannya beserta
   arahan ke HR, bukan pesan error.
10. Wizard enrollment: 3 foto diterima → commit → `is_enrolled: true`. Foto ditolak
    menampilkan `hints` dan **tidak** menambah hitungan.
11. *Reload* di tengah wizard memulihkan langkah dari server (#39), tidak kembali ke awal.
12. Panduan foto **tidak** menyuruh menoleh (D26) — diverifikasi manual terhadap
    `face.max_abs_yaw`; foto yang mengikuti panduan lolos gate kualitas.
13. `FACE_BELONGS_TO_ANOTHER_EMPLOYEE` tidak menyebut identitas siapa pun.
14. `/me/history` **tidak memuat** `matched_similarity`, `threshold_used`,
    `model_version`, `quality_score`, maupun `distance_meter` — diuji dengan memindai
    seluruh DOM dan seluruh body response.
15. Semua error kamera (E6–E10) menampilkan pesan spesifik yang bisa ditindaklanjuti,
    bukan `error.message` mentah.
16. Berpindah halaman saat kamera hidup **menghentikan seluruh track** — diuji.
17. Akurasi GPS buruk: memblokir bila `missing_location_policy = reject`, memperingatkan
    bila `pending_review`. Perilakunya **berubah** saat setting diubah di Fase 5, tanpa
    deploy ulang — diuji dengan mengubah setting lalu memuat ulang.
18. Retry setelah kegagalan jaringan memakai `Idempotency-Key` yang sama dan **tidak**
    menghasilkan record ganda.
19. Markdown consent tersanitasi: dokumen berisi `<script>` dan `<img onerror>`
    dirender tanpa mengeksekusi apa pun.
20. Tombol setuju consent nonaktif sampai teks di-scroll penuh **dan** checkbox dicentang.
21. Halaman dibuka lewat `http://` (non-localhost) menampilkan penjelasan HTTPS,
    bukan kegagalan kamera.
22. Tidak ada kebocoran object URL setelah 30 siklus ambil-batal.
23. Diuji di perangkat nyata lewat HTTPS: **Chrome desktop, Chrome Android, Safari iOS** —
    kamera, lokasi, unggah, dan seluruh alur berfungsi di ketiganya.
24. `tsc --noEmit`, ESLint, Prettier lulus; CI hijau; seluruh E2E lulus.
25. `docs/web/fase6-employee.md` ada.
26. `DONE-Fase-6.md` ada.
27. ✅ **Milestone "Web fully working" dikonfirmasi user** sebelum Fase 7 dimulai.

---

## 11. Cara Test / Verifikasi

### 11.1 Persiapan

```bash
# HTTPS untuk dev (§ 2.1)
mkcert -install && mkcert localhost 192.168.1.10
# vite.config.ts: server.https = { key, cert }, server.host = true

make up
./scripts/seed-fase6-scenario.sh   # 4 karyawan: enrolled, belum-enroll, manual, consent-dicabut
cd apps/faceclock-web && npm run dev
```

### 11.2 Konfigurasi Playwright (kamera & lokasi palsu)

```ts
// playwright.config.ts
use: {
  launchOptions: {
    args: [
      "--use-fake-ui-for-media-stream",      // otomatis izinkan kamera
      "--use-fake-device-for-media-stream",
      "--use-file-for-fake-video-capture=e2e/fixtures/face-ok.y4m",
    ],
  },
  permissions: ["geolocation"],
  geolocation: { latitude: -6.2002, longitude: 106.8167, accuracy: 12 },
  ignoreHTTPSErrors: true,
}
```

Untuk **jalur error** kamera, jangan mengandalkan flag — stub API-nya:

```ts
await page.addInitScript(() => {
  navigator.mediaDevices.getUserMedia = () =>
    Promise.reject(Object.assign(new Error("denied"), { name: "NotAllowedError" }));
});
```

Ini membuat E6–E10 dapat diuji satu per satu secara deterministik.

### 11.3 Uji penegakan master plan § 9 — wajib

```ts
test("tidak ada input berkas di jalur foto harian", async ({ page }) => {
  for (const path of ["/me/attendance", "/me/enrollment"]) {
    await page.goto(path);
    await expect(page.locator('input[type="file"]')).toHaveCount(0);
  }
});

test("tidak ada jalur absen manual yang berdiri sendiri", async ({ page }) => {
  await page.goto("/me/attendance");
  await expect(page.getByRole("button", { name: /manual|tinjau|review/i })).toHaveCount(0);
  // tombol fallback baru muncul SETELAH percobaan wajah gagal
});
```

Dua test ini adalah pembuktian paling langsung bahwa aturan master plan § 9 hidup
di kode, bukan hanya di dokumen.

### 11.4 Uji frame tidak tercermin

Fixture `face-ok.y4m` dibuat dengan penanda asimetris (mis. kotak putih di sudut
kiri-atas frame).

```ts
test("frame terkirim tidak tercermin", async ({ page }) => {
  const upload = page.waitForRequest(r => r.url().includes("/attendances/check-in"));
  await page.goto("/me/attendance");
  await page.getByRole("button", { name: "Ambil foto" }).click();
  await page.getByRole("button", { name: "Kirim" }).click();
  const buf = extractImagePart(await (await upload).postDataBuffer()!);
  const px = await decodeJpegPixel(buf, { x: 10, y: 10 });      // sudut kiri-atas
  expect(isWhite(px)).toBe(true);        // penanda tetap di KIRI; bila tercermin, ia ada di kanan
});
```

Ditambah pemeriksaan statis sederhana: `grep -r "scale(-1" src/lib/media/` harus nol
hasil.

### 11.5 Uji siklus hidup kamera

```ts
test("track dihentikan saat berpindah halaman", async ({ page }) => {
  await page.addInitScript(() => {
    (window as any).__stopped = 0;
    const orig = MediaStreamTrack.prototype.stop;
    MediaStreamTrack.prototype.stop = function () { (window as any).__stopped++; return orig.call(this); };
  });
  await page.goto("/me/attendance");
  await page.waitForSelector("video");
  await page.goto("/me/history");
  expect(await page.evaluate(() => (window as any).__stopped)).toBeGreaterThan(0);
});
```

### 11.6 Uji kesetaraan haversine Go ↔ TS

```ts
// Fixture koordinat dibagi dengan test Go (internal/geo/haversine_test.go)
const CASES = loadJson("../../../docs/api/geo-testcases.json");
for (const c of CASES) {
  expect(haversine(c.a, c.b)).toBeCloseTo(c.expected_meters, 0);   // ±1 m
}
```

Berkas kasus uji yang sama dibaca kedua sisi — inilah cara memastikan jarak yang
ditampilkan sebelum kirim tidak berbeda dari yang dihitung server.

### 11.7 Uji kebocoran field sensitif

```ts
test("halaman karyawan tidak menampilkan skor", async ({ page }) => {
  const bodies: string[] = [];
  page.on("response", async r => { if (r.url().includes("/api/")) bodies.push(await r.text().catch(() => "")); });
  await page.goto("/me/history");
  await page.getByRole("row").first().click();

  const forbidden = ["matched_similarity", "threshold_used", "model_version",
                     "quality_score", "distance_meter", "gps_accuracy_meter"];
  const dom = await page.content();
  for (const f of forbidden) {
    expect(dom).not.toContain(f);
    expect(bodies.join("\n")).not.toContain(f);   // membuktikan DTO server juga benar
  }
});
```

Baris terakhir menguji dua hal sekaligus: UI tidak menampilkannya, **dan** server
memang tidak mengirimkannya ([Fase 4 § 10 poin 18](05-Fase4.md#10-definition-of-done)).

### 11.8 Uji alur end-to-end

| Spec | Skenario |
|---|---|
| `checkin-happy` | Login → `/me/attendance` → ambil → kirim → `approved` → jam server tampil → `/me/history` memuat barisnya |
| `checkin-fallback` | MSW/backend balas `FACE_NOT_MATCHED` + `can_fallback:true` → layar gagal → hints tampil → [Kirim untuk ditinjau] → **kamera terbuka lagi** → kirim → `pending_review` |
| `checkin-inference-down` | Matikan `faceclock-inference` → `502/504` → layar gagal → fallback → `pending_review`. Assert **tidak pernah** `approved` |
| `checkin-geofence` | `geolocation` jauh → tombol kirim nonaktif + jarak tampil; ubah `outside_geofence_policy` ke `pending_review` → kirim diizinkan dengan peringatan |
| `checkin-manual-mode` | Karyawan mode manual → pesan mode manual → satu kali kirim → `pending_review` |
| `checkin-double` | Check-in dua kali → layar "sudah check-in", **bukan** toast merah |
| `enrollment-wizard` | Consent → sesi → 2 foto bagus + 1 gelap (fixture `face-dark.y4m`) → hints tampil → ulangi → commit → `is_enrolled` |
| `enrollment-resume` | Reload setelah 2 foto → langkah dipulihkan, hitungan tetap 2 |
| `consent` | Tombol setuju nonaktif → scroll → checkbox → setuju → cabut → `/me/attendance` menampilkan keadaan consent dicabut |
| `camera-errors` | Stub 5 `error.name` → 5 pesan berbeda dan spesifik |
| `no-file-input` | § 11.3 |

### 11.9 Uji perangkat nyata (manual, lewat HTTPS)

| Perangkat | Yang diperiksa |
|---|---|
| Chrome desktop (webcam laptop) | Alur penuh; `face_too_small` sering muncul → overlay oval membantu |
| Chrome Android | Kamera depan, akurasi GPS di dalam gedung, ukuran unggah di jaringan seluler |
| Safari iOS | `playsInline` wajib (tanpa itu video masuk fullscreen); `canvas.toBlob` tersedia; izin lokasi berperilaku berbeda |
| Firefox desktop | `getUserMedia` & `enumerateDevices` |

Safari iOS layak diuji lebih awal, bukan terakhir: ia yang paling sering berbeda,
dan `playsInline` yang terlupa membuat seluruh halaman tidak dapat dipakai di iPhone.

### 11.10 Daftar periksa manual penutup

- [ ] Buka lewat `http://<ip-lan>` → penjelasan HTTPS, bukan error kamera
- [ ] Tolak izin kamera → pesan spesifik + panduan
- [ ] Buka Zoom lalu buka halaman absensi → pesan `NotReadableError` yang tepat
- [ ] Semua jam tampil dalam `attendance.timezone`, sama dengan yang dilihat admin di Fase 5
- [ ] Ubah `attendance.max_note_length` di Fase 5 → penghitung karakter di Fase 6 ikut berubah
- [ ] Ubah `require_face_for_checkout` → layar check-out berubah tanpa deploy ulang
- [ ] DevTools → `localStorage` tidak memuat foto, blob, maupun access token
- [ ] Salin URL foto dari Network → buka di tab baru → `401`, bukan gambar (verifikasi D14)

---

## 12. Kontrak Fase 0–5 yang Perlu Revisi (dilaporkan, bukan diubah diam-diam)

| # | Kontrak | Revisi | Sifat |
|---|---|---|---|
| 1 | Fase 4 #55 `GET /attendances/context` | Sertakan `require_face_for_checkout` dan `checkout_without_face_status` di objek response, agar halaman absensi cukup satu panggilan alih-alih menambah #28 | Tambahan kecil, kompatibel mundur — ✅ **Diterapkan di kontrak** ([Fase 4 § 4.4](05-Fase4.md#44-get-attendancescontext), [REV-EP-04](09-Revisions-Log.md#b-perubahan-katalog-endpoint), resolusi [K-01](09-Revisions-Log.md#k-01--checkout_without_face_status-tidak-dapat-dibaca-karyawan--kontradiksi--resolved)) |
| 2 | Fase 4 — field `can_fallback` pada error | Fase 4 § 4.2 menampilkannya pada contoh `422 FACE_NOT_MATCHED`. Perlu **dinyatakan berlaku juga** pada `422 FACE_NOT_USABLE`, `422 FACE_NOT_ENROLLED`, `502 UPSTREAM_ERROR`, dan `504 UPSTREAM_TIMEOUT` — semua keadaan di mana fallback adalah langkah berikutnya yang sah. Tanpa itu, UI harus menebak | Klarifikasi + tambahan |
| 3 | Fase 0/5 — penyajian `faceclock-web` | **HTTPS wajib** untuk semua lingkungan selain `localhost` (§ 2.1). Sudah dilaporkan di [Fase 5 § 12 nomor 6](06-Fase5.md#12-kontrak-fase-04-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam); di sini menjadi prasyarat eksekusi | ⚠️ Infrastruktur |
| 4 | Fase 4 § 7 `internal/geo` | Ekspor kasus uji haversine ke `docs/api/geo-testcases.json` supaya sisi Go dan TS diuji terhadap data yang sama (§ 11.6) | Tambahan; test saja |
| 5 | Fase 2/5 — kalimat `hints` | Sekali lagi menegaskan saran [Fase 5 § 12 nomor 7](06-Fase5.md#12-kontrak-fase-04-yang-perlu-revisi-dilaporkan-bukan-diubah-diam-diam): satu berkas kanonik `docs/api/hints.json`. Di Fase 6 kalimat ini menjadi teks yang dibaca karyawan setiap hari, jadi perbedaan antara Go dan TS akan langsung terasa | Saran |
| 6 | Fase 0 § 2.7 (dependensi web) | Bertambah: `dompurify`, `ulid`. Playwright butuh flag kamera palsu di config | Tambahan |

Nomor 2 pantas diputuskan sebelum eksekusi: tanpa `can_fallback` yang konsisten,
UI harus menyimpulkan sendiri dari `fallback_enabled` + kebijakan geofence — yaitu
menduplikasi logika yang sudah ada di server, dan duplikasi itu akan menyimpang
begitu ada setting yang berubah.
