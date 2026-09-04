# Fase 2 — Face Inference Service (`faceclock-inference`)

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 2**. Ukuran: 🔴 **Besar**.
> Mengikuti Protokol Handoff § 10.2: dokumen ini harus di-review sebelum eksekusi.
>
> **Depends on:** [Fase 0](01-Fase0.md). **Boleh dikerjakan paralel dengan** [Fase 1](02-Fase1.md).
> **Blocker untuk:** Fase 3 (enrollment) dan Fase 4 (attendance engine).
>
> **Status:** DRAFT — 3 keputusan butuh konfirmasi user (§ 2.0).

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Fase 3 dan Fase 4 keduanya bertanya hal yang sama ke sistem: *"foto ini wajah siapa,
dan seberapa yakin?"* Kalau kemampuan itu tidak dipisah jadi satu service dengan
kontrak yang jelas, logikanya akan tersebar: sebagian di enrollment, sebagian di
check-in, dengan aturan kualitas yang berbeda-beda — dan sistem akan menerima foto
buram saat enroll lalu menolak orang yang sama saat absen.

Fase 2 membangun **satu-satunya tempat di sistem yang melihat piksel wajah**.
Semua fase lain hanya berurusan dengan angka.

### Hasil akhir yang diharapkan
- `faceclock-inference` berjalan di Docker, memuat model InsightFace sekali saat
  startup, dan menerima foto → mengembalikan embedding 512 dimensi ter-normalisasi
  L2 plus metrik kualitas.
- Kontrak API-nya **terdokumentasi dan versioned**, sehingga Fase 3 dan Fase 4
  tinggal memakai, bukan menebak.
- `faceclock-api` punya client Go yang matang: timeout, retry terbatas, pemetaan
  error, dan metrik.
- Ada **laporan kalibrasi** yang menghasilkan angka `face.similarity_threshold`
  berbasis pengukuran FAR/FRR, bukan tebakan — dan angka itu tersimpan di
  `app_settings` (master plan § 9 "threshold configurable").
- Ada aturan tertulis: kombinasi model + detektor + alignment mana yang dipakai,
  dan apa yang harus terjadi kalau kombinasi itu berubah.

### Yang TIDAK dikerjakan di fase ini
- Menyimpan apa pun. Service ini **tidak punya database**, tidak menulis ke disk,
  dan tidak tahu apa itu karyawan. Byte foto masuk, angka keluar, referensinya
  dilepas. (Ini persyaratan UU PDP, bukan pilihan desain.)
- Membandingkan dengan referensi karyawan — itu Fase 4, dan perbandingannya
  terjadi di Postgres (§ 2.4).
- Liveness / anti-spoofing sungguhan. Master plan § 9 menaruhnya di Fase 7
  (on-device). Fase 2 hanya menyediakan **metrik kualitas** yang membuat foto layar
  atau foto-dari-foto lebih mudah ditolak.
- 1:N identification ("wajah ini siapa dari 500 karyawan"). Sistem ini melakukan
  1:1 verification — employee_id sudah diketahui dari token.

---

## 2. Scope Detail

### 2.0 Keputusan yang butuh konfirmasi

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D10 | Model: `buffalo_s` (master plan) vs `buffalo_l` (prior art) | **Mulai dari `buffalo_l`, benchmark keduanya di § 2.9, turunkan ke `buffalo_s` hanya bila budget latensi terlampaui** | ⚠️ **BUTUH KONFIRMASI** |
| D11 | Logika compare: di Python atau di Go/Postgres | **Di Postgres (pgvector)**; `/v1/compare` hanya utilitas kalibrasi, bukan jalur produksi | ⚠️ **BUTUH KONFIRMASI** |
| D12 | Basis kode: port dari `face-engine/` atau tulis dari nol | **Port dari `face-engine/`** | ⚠️ **BUTUH KONFIRMASI** |

---

### 2.1 D12 — Memanfaatkan `face-engine/` yang sudah ada ⚠️ BUTUH KONFIRMASI

Di root project sudah ada `face-engine/` — inference service FastAPI + InsightFace
milik project SIMRS/VitaCore yang **sudah matang dan sudah diukur**. Isinya:

| File | Yang sudah terselesaikan di sana |
|---|---|
| `app/embedder.py` | SCRFD detect → ArcFace align (`norm_crop` 112×112) → embed → L2-normalize; kualitas diukur pada crop teraligned, bukan frame mentah |
| `app/quality.py` | `variance_of_laplacian` (blur), `mean_brightness`, proxy `yaw`/`pitch` dari 5 keypoint SCRFD, proxy oklusi, dan formula `score_quality` yang sudah dipikirkan (geometric mean ditarik ke komponen terburuk) |
| `app/config.py` | Semua threshold kualitas sebagai konstanta yang bisa di-override env |
| `app/security.py` | Bearer token internal dengan `hmac.compare_digest` |
| `app/main.py` | Model dimuat di `lifespan` (bukan lazy), penanganan **anti-spill** `MultiPartParser.spool_max_size` — foto biometrik tidak pernah menyentuh disk lewat temp file Starlette |
| `Dockerfile` | Multi-stage, model di-*bake* saat build (tidak download saat boot), model yang tidak dipakai (`1k3d68`, `genderage`, `2d106det`) dibuang, non-root, `insightface --no-deps` agar tidak menarik `opencv-python` GUI |
| `docker-compose.face-engine.yml` | `read_only: true`, `tmpfs /tmp`, `no-new-privileges`, port tidak dipublish, limit memori |
| `requirements.txt` | Versi **dipin ketat**, dengan alasan tertulis: bump minor `insightface` bisa mengubah transform alignment dan menggeser semua vektor tersimpan tanpa error apa pun |
| `spike/roc.py` | Harness kalibrasi ROC/FAR/FRR |

**Rekomendasi: port, jangan tulis ulang.** Yang ada di sana bukan boilerplate —
itu hasil dari orang yang sudah menabrak masalahnya (temp file spill, model yang
di-download saat boot, `libGL.so.1` di image slim). Menulis ulang dari nol berarti
menabrak ulang.

**Tapi "port" bukan `mv`.** Yang harus berubah saat dipindah ke `faceclock-inference`:

| Aspek | `face-engine` (asal) | `faceclock-inference` (target) |
|---|---|---|
| Konvensi JSON | camelCase (consumer Node) | **snake_case** ([Fase 0 D4](01-Fase0.md#24-d4--konvensi-json-snake_case--butuh-konfirmasi)) |
| Envelope response | telanjang / `{detail}` FastAPI | `{data}` / `{error}` ([Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go)) |
| Path | `/embed`, `/embed-batch` | `/v1/embed`, `/v1/embed-batch` (versioned) |
| Readiness | hanya `/health` | `/health` (liveness) + `/ready` (model benar-benar termuat) |
| Domain | pasien/SIMRS di komentar & docstring | karyawan/Faceclock |
| EXIF orientation | **tidak ditangani** | ditangani (§ 2.7 — gap nyata, lihat E9) |
| Decompression bomb | **tidak dibatasi** | dibatasi (§ 2.7 — gap nyata, lihat E10) |
| `request_id` | tidak ada | diteruskan dari `X-Request-Id` milik Go, masuk ke setiap log |
| Threshold kualitas | dikalibrasi untuk webcam kiosk rumah sakit | **wajib dikalibrasi ulang** untuk kondisi Faceclock (§ 2.9) |

**Lisensi & asal-usul.** Sebelum port, konfirmasi bahwa kode `face-engine/` boleh
dipakai di project ini. Ini pertanyaan untuk user, bukan asumsi teknis. Catat
jawabannya di `docs/adr/0002-inference-prior-art.md`.

> ⚠️ Model InsightFace `buffalo_*` sendiri berlisensi **non-komersial** untuk
> penggunaan model pre-trained-nya. Kalau Faceclock akan dipakai komersial, ini
> harus diselesaikan (lisensi komersial InsightFace, atau ganti ke model dengan
> lisensi permisif). **Angkat ini ke user sekarang, bukan di Fase 7.**

---

### 2.2 D10 — Pilihan model ⚠️ BUTUH KONFIRMASI

Master plan § 3 menyebut `buffalo_s` ("ringan untuk CPU"). Prior art memakai
`buffalo_l` dan seluruh threshold di sana diukur dengan itu.

| | `buffalo_s` | `buffalo_l` |
|---|---|---|
| Detektor | `det_500m.onnx` (SCRFD-500MF) | `det_10g.onnx` (SCRFD-10GF) |
| Recognizer | `w600k_mbf.onnx` (MobileFaceNet) | `w600k_r50.onnx` (ResNet50) |
| Dimensi embedding | **512** | **512** |
| Ukuran | ~16 MB | ~330 MB (~166 MB setelah model tak terpakai dibuang) |
| Latensi CPU | jauh lebih cepat | lebih lambat |
| Akurasi | lebih rendah, terutama pada wajah kecil/miring/gelap | lebih tinggi |

**Fakta yang menurunkan risiko keputusan ini:** keduanya menghasilkan embedding
**512-d**. Artinya skema database Fase 3 (`vector(512)`) **tidak berubah** apa pun
pilihannya. Yang berubah hanya: nilai threshold, dan keharusan reindex bila model
diganti setelah ada data.

**Rekomendasi:** mulai dengan `buffalo_l` karena bobotnya sudah ada di
`face-engine/models/buffalo_l/` dan threshold prior art sudah terukur dengannya,
lalu **jalankan benchmark § 2.9 pada keduanya**. Turun ke `buffalo_s` hanya bila
p95 latensi `buffalo_l` melampaui budget (§ 2.8). Untuk absensi kantor —
puluhan sampai ratusan request per hari, bukan per detik — akurasi lebih berharga
daripada 200 ms.

`FACE_MODEL_NAME` tetap env, jadi keputusan ini bisa dibalik dengan rebuild image,
bukan dengan menulis ulang kode.

**Aturan `model_version` (mengikat semua fase):**
- Setiap embedding yang disimpan **wajib** membawa `model_version` yang
  memproduksinya (kolom di `face_references` dan `attendances` — Fase 3/4).
- Vektor dari `model_version` berbeda **tidak pernah** dibandingkan. Query
  perbandingan selalu memfilter `model_version = <aktif>`.
- Mengganti model, mengganti `det_size`, atau menaikkan versi `insightface`
  = `model_version` baru = **wajib reindex** semua `face_references` lewat
  `/v1/embed-batch`. Prosedurnya didefinisikan di Fase 3.
- `face.model_version` di `app_settings` (sudah di-seed Fase 1 dengan nilai
  `"unset"`) diisi di akhir Fase 2 dengan nilai sesungguhnya.

---

### 2.3 Arsitektur service

```
faceclock-api (Go)
   │  POST /v1/embed   multipart: image
   │  Authorization: Bearer <INFERENCE_TOKEN>
   │  X-Request-Id: <ULID dari Fase 0>
   ▼
faceclock-inference (FastAPI, 1 worker)
   ├─ auth: hmac.compare_digest bearer token
   ├─ cap: max 6 MB, max 40 MP, mime whitelist
   ├─ decode: cv2.imdecode dari MEMORI (tidak pernah ke disk)
   ├─ EXIF orientation fix
   ├─ SCRFD detect  ──► 0 wajah? ► 200 {usable:false, hints:["no_face"]}
   │                 └► >1 wajah? ► hints:["multiple_faces"], usable:false
   ├─ align: norm_crop 112×112 dari 5 keypoint
   ├─ quality: det_score, blur, brightness, yaw, pitch, face_ratio, occlusion
   ├─ gate: hints[] → usable = len(hints)==0
   ├─ embed: ArcFace ──► L2-normalize
   └─ response 200 {embedding[512], quality{}, hints[], usable, model_version}
        (byte gambar dilepas; tidak ada yang ditulis, tidak ada yang di-log)
```

**Satu worker uvicorn, scale dengan replica.** Setiap worker memegang salinan
sesi ONNX sendiri (~450 MB resident, ~1.2 GB committed menurut pengukuran prior
art). Menaikkan `--workers` menggandakan memori tanpa menggandakan throughput di
mesin dengan core terbatas.

**Tidak ada port yang dipublish ke host.** Sudah ditetapkan di
[Fase 0 § 2.9](01-Fase0.md#29-database--docker-compose). Service ini menerima
byte wajah mentah; satu-satunya yang boleh menghubunginya adalah `faceclock-api`.

---

### 2.4 D11 — Di mana perbandingan terjadi ⚠️ BUTUH KONFIRMASI

Master plan menyebut endpoint `compare` "opsional bila logika compare ditaruh di sini".

**Rekomendasi: perbandingan produksi terjadi di Postgres, bukan di Python.**

Alasan:
1. Embedding referensi **sudah ada di Postgres** (Fase 3 menyimpannya di
   `face_references.embedding vector(512)`). Mengirimkannya ke Python setiap
   check-in berarti mengangkut vektor biometrik bolak-balik lewat jaringan untuk
   pekerjaan yang bisa dilakukan di tempat datanya berada.
2. Threshold hidup di `app_settings` (Fase 1), yaitu di Postgres juga. Menaruh
   perbandingan di Python berarti Python harus tahu tentang setting aplikasi —
   dan itu merusak sifat "tidak tahu apa-apa tentang domain" yang membuat service
   ini aman.
3. Karena semua embedding **sudah ter-L2-normalize**, operator `<=>` pgvector
   (cosine distance) memberi `similarity = 1 - (a <=> b)` secara eksak. Tidak ada
   matematika yang perlu ditulis ulang.

Query yang akan dipakai Fase 4 — ditulis di sini supaya Fase 3 tahu index dan
kolom apa yang harus disediakan:

```sql
-- master plan § 4 langkah 4: "ambil similarity TERTINGGI, bukan rata-rata"
SELECT id, 1 - (embedding <=> $1::vector) AS similarity
FROM face_references
WHERE employee_id   = $2
  AND is_active     = true
  AND model_version = $3
ORDER BY embedding <=> $1::vector
LIMIT 1;
```

**Catatan penting untuk Fase 3: JANGAN buat index HNSW/IVFFlat pada
`face_references.embedding`.** Query di atas sudah difilter `employee_id` dan hanya
menyentuh 3–5 baris; index ANN tidak memberi percepatan apa pun di sana, dan
index *approximate* justru bisa melewatkan tetangga terdekat yang sebenarnya —
tepat kesalahan yang tidak boleh terjadi pada verifikasi identitas. Index yang
dibutuhkan hanyalah B-tree biasa pada `(employee_id, is_active, model_version)`.
Index ANN baru relevan kalau suatu saat ada 1:N identification, dan itu bukan
scope sistem ini (master plan § 1 Non-goals).

**`/v1/compare` tetap dibuat**, tapi sebagai **utilitas kalibrasi dan test**:
menerima dua gambar (atau gambar + array embedding) dan mengembalikan cosine
similarity. Dipakai oleh `spike/roc.py`, oleh contract test, dan oleh admin saat
men-debug "kenapa si A tidak dikenali". Tidak dipanggil di jalur check-in.
Didokumentasikan sebagai `x-internal-tooling` di kontrak.

---

### 2.5 Kontrak kualitas — kosakata `hints`

`hints` adalah **kosakata tertutup**. Kode Go, UI web (Fase 6), dan app Flutter
(Fase 7) semuanya memetakannya ke teks Indonesia. Menambah hint baru = perubahan
kontrak = naikkan versi kontrak dan perbarui semua consumer.

| Hint | Arti | Blocking? | Saran teks UI (Fase 6/7) |
|---|---|---|---|
| `no_face` | Tidak ada wajah terdeteksi | ✅ | "Wajah tidak terdeteksi. Posisikan wajah di dalam bingkai." |
| `multiple_faces` | Lebih dari satu wajah dalam frame | ✅ | "Terdeteksi lebih dari satu wajah. Pastikan hanya Anda dalam bingkai." |
| `low_detection_confidence` | `det_score` di bawah ambang | ✅ | "Wajah kurang jelas. Hadapkan wajah lurus ke kamera." |
| `too_blurry` | Variance of Laplacian rendah | ✅ | "Foto buram. Tahan perangkat agar tidak bergerak." |
| `too_dark` | Kecerahan crop wajah rendah | ✅ | "Terlalu gelap. Cari tempat yang lebih terang." |
| `too_bright` | Kecerahan terlalu tinggi / overexposed | ✅ | "Terlalu terang. Hindari cahaya langsung dari belakang." |
| `face_too_small` | Tinggi bbox / tinggi frame di bawah ambang | ✅ | "Wajah terlalu jauh. Dekatkan wajah ke kamera." |
| `head_turned` | Proxy yaw melebihi ambang | ✅ | "Hadapkan wajah lurus ke kamera." |
| `head_tilted` | Proxy pitch melebihi ambang | ✅ | "Jangan menunduk atau mendongak." |

`usable = (len(hints) == 0)`.

**`multiple_faces` tidak pernah diselesaikan diam-diam dengan mengambil wajah
terbesar.** Service memang memilih bbox terbesar untuk diukur, tapi tetap
melaporkan `face_count > 1` dan menandai `usable = false`. Kalau tidak, orang yang
berdiri di belakang karyawan bisa ikut ter-enroll — dan tidak akan ada yang tahu.

**`occlusion_ratio` dilaporkan tapi TIDAK dijadikan hint blocking.** Prior art
sudah menandainya "NOT VALIDATED": rasionya juga turun untuk wajah bercukur bersih
di bawah cahaya rata. Ia diekspos supaya bisa dikalibrasi (atau diganti classifier
masker sungguhan) nanti — jangan dipakai menolak orang berdasarkan angka yang
belum divalidasi.

**Aturan penting: "tidak ada wajah" BUKAN error HTTP.** Ia dibalas `200` dengan
`usable: false`. Alasannya: pemanggil (Fase 3/4) butuh metrik kualitas untuk
memutuskan apa yang ditampilkan ke user dan apakah menawarkan fallback. Kalau
dibalas 422, informasi itu hilang dan Go hanya tahu "gagal". Yang dibalas 4xx
hanyalah request yang **salah bentuk** (bukan gambar, terlalu besar, token salah).

---

### 2.6 Klien Go (`internal/inference`)

Diselesaikan di fase ini (Fase 0 hanya menyediakan `Health()`).

```go
type Client interface {
    Health(ctx context.Context) (HealthInfo, error)
    Embed(ctx context.Context, img []byte, filename string) (*EmbedResult, error)
    EmbedBatch(ctx context.Context, imgs [][]byte) ([]BatchItem, error)
}

type EmbedResult struct {
    Embedding    []float32 `json:"embedding"`     // nil bila tidak ada wajah
    BBox         []float64 `json:"bbox"`
    DetScore     *float64  `json:"det_score"`
    Quality      Quality   `json:"quality"`
    QualityScore float64   `json:"quality_score"`
    Hints        []string  `json:"hints"`
    Usable       bool      `json:"usable"`
    ModelVersion string    `json:"model_version"`
    EmbeddingDim int       `json:"embedding_dim"`
    TookMs       int       `json:"took_ms"`
}
```

**Kebijakan timeout & retry:**

| Aspek | Nilai | Alasan |
|---|---|---|
| Timeout total | `INFERENCE_TIMEOUT_MS`, default **6000 ms** | Karyawan berdiri di depan kamera; di atas ~6 detik dia sudah menekan tombol lagi |
| Retry | **maks 1×**, hanya untuk connection error / `502` / `503` / `504` | Embed tidak punya efek samping, jadi aman diulang — tapi mengulang berkali-kali hanya memperpanjang waktu tunggu |
| Tidak di-retry | `400`, `401`, `413`, `415`, `422` | Mengirim ulang byte yang sama tidak akan mengubah jawabannya |
| Backoff | 250 ms tetap | Satu percobaan ulang tidak butuh exponential |
| Batas konkurensi | semaphore = `2 × jumlah replica` | 1 worker per replica; antrian yang lebih panjang hanya menaikkan latensi ekor |
| Circuit breaker | buka setelah 5 kegagalan koneksi berturut-turut, half-open setelah 30 detik | Mencegah setiap request check-in menunggu 6 detik saat service memang mati |

**Pemetaan error ke katalog Fase 0:**

| Dari inference | Go membalas ke client | Catatan |
|---|---|---|
| `200 usable=true` | lanjut ke logika bisnis | |
| `200 usable=false` | **keputusan bisnis**, bukan error | Fase 3: `422 FACE_NOT_USABLE` + `hints`. Fase 4: tawarkan fallback → `pending_review` |
| `401` | `500 INTERNAL_ERROR` | Token salah = salah konfigurasi kita, bukan salah user. Log level `error` |
| `413` / `415` | `413` / `415` diteruskan | |
| `422 IMAGE_DECODE_FAILED` | `422 VALIDATION_ERROR` | "File bukan gambar yang valid" |
| `503 MODEL_NOT_READY` | `503 SERVICE_UNAVAILABLE` | |
| `5xx` lain | `502 UPSTREAM_ERROR` | |
| timeout / connection refused | `504 UPSTREAM_TIMEOUT` | |

> 🔴 **Aturan lintas-fase yang tidak boleh dilanggar:** kegagalan inference
> (timeout, 5xx, circuit breaker terbuka) **tidak pernah** menghasilkan absensi
> `approved`. Di Fase 4 hasilnya adalah jalur fallback → `pending_review`, atau
> penolakan. Master plan § 9 menyatakan fallback tidak boleh langsung approved;
> "inference-nya mati" adalah bentuk fallback yang paling mudah dieksploitasi
> (matikan service → semua orang lolos). Ini ditulis sebagai test di Fase 4.

**Metrik yang diekspos client** (dipakai dashboard operasional nanti):
`inference_request_duration_ms` (histogram), `inference_requests_total{status}`,
`inference_usable_total{usable}`, `inference_hints_total{hint}`,
`inference_circuit_state`.

**Larangan logging** (dari [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go)):
client Go **tidak boleh** mencatat byte gambar, nama file asli, atau isi array
`embedding`. Yang dicatat: `request_id`, `employee_id`, `usable`, `hints`,
`quality_score`, `took_ms`, ukuran byte, `model_version`.

---

### 2.7 Penanganan gambar — dua celah yang harus ditutup

Dua hal yang **tidak** ada di prior art dan harus ditambahkan:

**a. Orientasi EXIF.** `cv2.imdecode` mengabaikan tag EXIF `Orientation`. Foto dari
kamera HP (Fase 7) dan sebagian upload browser tersimpan dalam orientasi sensor
dengan tag rotasi terpisah. Akibatnya wajah masuk dalam keadaan miring 90°,
SCRFD gagal mendeteksi, dan user melihat "wajah tidak terdeteksi" pada foto yang
jelas-jelas berisi wajah. Ini bug yang hanya muncul di perangkat tertentu — jenis
yang lolos semua test lokal.

Penanganan: baca tag EXIF sebelum decode (atau `PIL.ImageOps.exif_transpose`),
putar array sesuai orientasi, baru jalankan detektor. Tambahkan test dengan file
JPEG yang punya `Orientation=6`.

**b. Decompression bomb.** Batas 6 MB adalah batas *byte terkompresi*. PNG 3 MB
bisa mengembang jadi ratusan megapiksel di memori dan mematikan container.
Penanganan: setelah membaca header (tanpa decode penuh), tolak bila
`width × height > FACE_MAX_PIXELS` (default **40 MP**) dengan
`422 IMAGE_TOO_LARGE_PIXELS`. Batasi juga dimensi maksimum per sisi (12000 px).

**c. Format yang diterima:** `image/jpeg`, `image/png`, `image/webp`. HEIC/HEIF
**ditolak** dengan `415` dan pesan jelas — mendukungnya butuh `libheif` di image
dan itu bukan pekerjaan Fase 2. Konsekuensinya untuk Fase 6/7: client wajib
mengonversi ke JPEG sebelum upload. Ditulis di kontrak supaya tidak jadi kejutan.

**d. Downscale sebelum deteksi.** Frame di atas 1920 px pada sisi terpanjang
di-resize dulu (menjaga rasio). SCRFD tetap berjalan pada `det_size` 640×640, jadi
resolusi berlebih hanya menambah waktu decode tanpa menambah akurasi.

---

### 2.8 Budget performa

Ditetapkan sekarang supaya benchmark § 2.9 punya kriteria lulus/gagal.

| Metrik | Target | Diukur pada |
|---|---|---|
| Cold start sampai `/ready` 200 | ≤ 45 detik | container start |
| `/v1/embed` p50 | ≤ 400 ms | 2 vCPU, `FACE_ONNX_THREADS=2`, JPEG 1280×720 |
| `/v1/embed` p95 | ≤ 900 ms | idem |
| Total budget check-in server-side (Fase 4) | ≤ 2500 ms p95 | auth + geofence + embed + query pgvector + insert |
| Memori resident per replica | ≤ 1.5 GB | steady state |
| Throughput 1 replica | ≥ 2 req/detik | cukup untuk ~100 karyawan absen dalam 15 menit |

Kalau `buffalo_l` melampaui p95 900 ms pada hardware target → jalankan D10 ke
`buffalo_s` dan **kalibrasi ulang threshold**.

---

### 2.9 Kalibrasi threshold — deliverable utama fase ini

Master plan § 9: *"Threshold configurable: simpan di `app_settings`, jangan
hardcode — butuh tuning berkelanjutan."* Fase 2 adalah tempat angka pertamanya lahir.

**Prosedur:**

1. **Port `spike/roc.py`** dari `face-engine/` ke `services/faceclock-inference/calibration/`.
2. **Dataset A — publik (LFW).** Sudah ada di `face-engine/spike/dataset/lfw/`.
   Hitung distribusi similarity untuk pasangan *genuine* (orang sama) dan
   *impostor* (orang berbeda), lalu kurva FAR/FRR.
3. **Dataset B — internal.** 15–30 karyawan sukarelawan, masing-masing 3 foto
   enrollment + 3 foto "check-in" pada kondisi berbeda (pagi/sore, dengan/tanpa
   kacamata, berdiri/duduk). **Ini yang menentukan angka produksi.**
4. Pilih threshold pada **FAR ≤ 0.1%** pada Dataset B, lalu laporkan FRR yang
   menyertainya. FAR (orang lain diterima) jauh lebih berbahaya daripada FRR
   (orang benar ditolak) — FRR punya jalur pemulihan yang sudah dirancang, yaitu
   fallback `pending_review`; FAR tidak punya jalur pemulihan sama sekali karena
   tidak ada yang tahu itu terjadi.
5. Tulis `docs/face/fase2-calibration-report.md`: model, `det_size`, versi
   `insightface`, ukuran dataset, kurva, threshold terpilih, FAR & FRR di titik itu,
   dan tanggal.
6. Isi `app_settings`: `face.similarity_threshold` = angka terpilih,
   `face.model_version` = mis. `buffalo_l@v1`.

> ⚠️ **Peringatan yang harus masuk laporan:** LFW didominasi wajah Barat dan foto
> jurnalistik. Threshold yang diturunkan hanya dari LFW **tidak sah** untuk populasi
> karyawan Indonesia di kondisi kantor. Dataset B bukan formalitas — ia adalah
> satu-satunya sumber angka produksi yang valid. Kalau Dataset B belum bisa
> dikumpulkan saat Fase 2 berjalan, tandai `face.similarity_threshold` sebagai
> **provisional** di laporan, dan jadikan pengumpulannya sebagai blocker
> Definition of Done Fase 4 — bukan Fase 2.

**Pengumpulan Dataset B = pemrosesan data biometrik.** Butuh consent tertulis dari
tiap sukarelawan (master plan § 2 & § 9, UU PDP). Foto disimpan terenkripsi, di luar
git, dan dihapus setelah kalibrasi selesai. Prosedurnya ditulis di
`docs/face/dataset-b-consent-and-retention.md`.

---

## 3. Skema Database

`faceclock-inference` **tidak punya database**. Yang berubah di Postgres hanya
nilai `app_settings` yang sudah dibuat Fase 1.

### 3.1 Migration `000011_face_settings`

Menambah key baru + menegakkan aturan § 2.2 (setiap permission baru langsung
di-grant ke `super_admin` — di sini tidak ada permission baru, jadi hanya settings).

```sql
INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
  ('face.det_size',            '640',   'number',  'Ukuran input detektor SCRFD',                       false),
  ('face.min_det_score',       '0.60',  'number',  'Ambang minimum confidence deteksi wajah',           false),
  ('face.min_blur_var',        '40.0',  'number',  'Ambang minimum variance of Laplacian (ketajaman)',  false),
  ('face.min_brightness',      '55.0',  'number',  'Ambang minimum kecerahan crop wajah (0-255)',       false),
  ('face.max_brightness',      '215.0', 'number',  'Ambang maksimum kecerahan crop wajah (0-255)',      false),
  ('face.min_face_ratio',      '0.18',  'number',  'Rasio minimum tinggi wajah terhadap tinggi frame',  false),
  ('face.max_abs_yaw',         '0.35',  'number',  'Ambang maksimum proxy yaw (tak berdimensi)',        false),
  ('face.max_abs_pitch',       '0.30',  'number',  'Ambang maksimum proxy pitch (tak berdimensi)',      false),
  ('face.max_image_bytes',     '6291456','number', 'Ukuran maksimum satu foto (byte)',                  true),
  ('face.accepted_mime_types', '["image/jpeg","image/png","image/webp"]', 'json',
                                          'Format foto yang diterima', true)
ON CONFLICT (key) DO NOTHING;
```

**Catatan arsitektural penting:** ambang kualitas ini disimpan di `app_settings`
**untuk ditampilkan dan diaudit**, tetapi yang benar-benar menegakkannya adalah
env var di container inference (`FACE_MIN_DET_SCORE`, dst.). Dua sumber kebenaran
adalah resep bug. Aturan yang dipilih:

- **Sumber kebenaran = env container inference.** Ia yang menghitung `hints`.
- `app_settings` menyimpan salinan **untuk ditampilkan di panel admin (Fase 5)**
  dan untuk dokumentasi, ditandai read-only di UI pada Fase 2–4.
- `GET /ready` di inference mengembalikan ambang yang **sedang aktif**;
  `faceclock-api` membandingkannya dengan `app_settings` saat startup dan
  **menulis WARNING** bila berbeda. Selisih diam-diam antara yang ditampilkan
  admin dan yang benar-benar berlaku adalah tepat jenis masalah yang berujung
  pada "kok settingnya sudah saya ubah tapi tidak ada efeknya".
- Yang **memang** dipakai `faceclock-api` dari `app_settings`:
  `face.similarity_threshold`, `face.model_version`, `face.min_reference_photos`.
  Ketiganya milik logika bisnis Go, bukan milik inference.

### 3.2 Kontrak data untuk Fase 3 (belum dibuat di sini)

Ditulis sekarang supaya Fase 3 tidak mendesain ulang:

| Aspek | Nilai yang mengikat |
|---|---|
| Tipe kolom | `embedding vector(512)` — pgvector |
| Normalisasi | Vektor **sudah** ter-L2-normalize saat ditulis. **Jangan** normalisasi ulang saat baca |
| Metrik | Cosine. `similarity = 1 - (a <=> b)`, rentang efektif 0..1 untuk vektor ternormalisasi |
| Kolom wajib pendamping | `model_version text NOT NULL`, `quality_score real NOT NULL`, `det_score real`, `is_active boolean` |
| Index | B-tree `(employee_id, is_active, model_version)`. **Bukan** HNSW/IVFFlat (§ 2.4) |
| Jumlah referensi | minimal `face.min_reference_photos` (default 3), disimpan **terpisah**, tidak di-average (master plan § Fase 3) |
| Pemilihan saat compare | **similarity tertinggi**, bukan rata-rata (master plan § 4 langkah 4) |

---

## 4. Daftar Endpoint / Kontrak API

Base URL internal: `http://faceclock-inference:8000`.
Semua endpoint selain `/health` memerlukan `Authorization: Bearer <FACE_ENGINE_TOKEN>`.
Envelope dan `snake_case` mengikuti [Fase 0 § 2.6](01-Fase0.md#26-scaffolding-faceclock-api-go).

| # | Method | Path | Auth | Dipakai oleh |
|---|---|---|---|---|
| 1 | GET | `/health` | — | Docker healthcheck, `/readyz` Go |
| 2 | GET | `/ready` | Bearer | `/readyz` Go, cek konsistensi setting |
| 3 | POST | `/v1/embed` | Bearer | Fase 3 (enroll), Fase 4 (check-in) |
| 4 | POST | `/v1/embed-batch` | Bearer | Reindex / backfill (Fase 3) |
| 5 | POST | `/v1/compare` | Bearer | Kalibrasi & debugging saja (§ 2.4) |

`docs_url`, `redoc_url`, dan `openapi_url` FastAPI **dimatikan** (seperti prior art).
Kontraknya didokumentasikan di `docs/api/fase2-inference.md`, bukan diekspos oleh
service yang memegang data biometrik.

### 4.1 `GET /health` — liveness

Tidak butuh auth (dipanggil oleh Docker HEALTHCHECK dari dalam container).

```json
200
{ "status": "ok", "model_loaded": true, "uptime_seconds": 1284 }
```

`status` = `"loading"` selama model belum termuat. Tetap `200` — proses hidup,
tidak perlu di-restart.

### 4.2 `GET /ready` — readiness + introspeksi konfigurasi

```json
200
{
  "data": {
    "status": "ok",
    "model_name": "buffalo_l",
    "model_version": "buffalo_l@v1",
    "embedding_dim": 512,
    "det_size": 640,
    "det_thresh": 0.5,
    "insightface_version": "1.0.1",
    "onnxruntime_version": "1.29.0",
    "quality_thresholds": {
      "min_det_score": 0.60,
      "min_blur_var": 40.0,
      "min_brightness": 55.0,
      "max_brightness": 215.0,
      "min_face_ratio": 0.18,
      "max_abs_yaw": 0.35,
      "max_abs_pitch": 0.30
    },
    "limits": {
      "max_image_bytes": 6291456,
      "max_pixels": 40000000,
      "max_batch_images": 16,
      "accepted_mime_types": ["image/jpeg", "image/png", "image/webp"]
    }
  }
}
```

```json
503
{ "error": { "code": "MODEL_NOT_READY", "message": "model belum termuat", "request_id": "..." } }
```

Endpoint ini yang membuat aturan "satu sumber kebenaran" di § 3.1 bisa ditegakkan:
`faceclock-api` membacanya saat startup dan membandingkan dengan `app_settings`.

### 4.3 `POST /v1/embed`

**Request** — `multipart/form-data`:

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `image` | file | ✅ | JPEG/PNG/WebP, ≤ 6 MB, ≤ 40 MP |

Header: `Authorization: Bearer <token>`, `X-Request-Id: <ULID>` (opsional tapi
selalu dikirim oleh client Go — dipakai untuk korelasi log lintas service).

**Response 200 — wajah ditemukan & layak:**

```json
{
  "data": {
    "embedding": [0.0134, -0.0421, "... 512 float ..."],
    "bbox": [412.5, 188.2, 246.0, 301.7],
    "det_score": 0.941,
    "quality": {
      "det_score": 0.941,
      "blur": 132.4,
      "brightness": 118.2,
      "yaw": 0.031,
      "pitch": -0.052,
      "face_ratio": 0.342,
      "face_count": 1,
      "occlusion_ratio": 0.874
    },
    "quality_score": 0.812,
    "hints": [],
    "usable": true,
    "model_version": "buffalo_l@v1",
    "embedding_dim": 512,
    "took_ms": 187
  }
}
```

`bbox` berformat `[x, y, width, height]` dalam koordinat piksel gambar **setelah**
koreksi EXIF dan downscale. Client yang ingin menggambar kotak harus memakai
`bbox_frame` di bawah untuk menskalakan:

```json
"frame": { "width": 1280, "height": 720, "downscaled_from": [4032, 2268], "exif_rotated": true }
```

(field `frame` selalu ada; ditambahkan supaya Fase 6/7 bisa menggambar overlay dengan benar.)

**Response 200 — tidak layak (bukan error):**

```json
{
  "data": {
    "embedding": null,
    "bbox": null,
    "det_score": null,
    "quality": {
      "det_score": 0.0, "blur": 0.0, "brightness": 42.1,
      "yaw": 0.0, "pitch": 0.0, "face_ratio": 0.0,
      "face_count": 0, "occlusion_ratio": 1.0
    },
    "quality_score": 0.0,
    "hints": ["no_face"],
    "usable": false,
    "model_version": "buffalo_l@v1",
    "embedding_dim": 512,
    "took_ms": 96,
    "frame": { "width": 1280, "height": 720, "downscaled_from": null, "exif_rotated": false }
  }
}
```

**Aturan yang mengikat:** bila `usable = false`, `embedding` **selalu** `null`.
Service tidak pernah mengembalikan vektor dari frame yang tidak lolos gate —
kalau ia mengembalikannya, cepat atau lambat ada kode di fase lain yang
menyimpannya. Satu pengecualian yang dibolehkan: `hints = ["multiple_faces"]`
saja, di mana embedding wajah terbesar **tetap null** demi konsistensi aturan ini.

**Error:**

| Status | code | Kondisi |
|---|---|---|
| 400 | `BAD_REQUEST` | Field `image` tidak ada / bukan multipart |
| 401 | `UNAUTHENTICATED` | Bearer token hilang atau salah |
| 413 | `PAYLOAD_TOO_LARGE` | Byte melebihi `FACE_MAX_IMAGE_BYTES` |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | MIME di luar whitelist (termasuk HEIC) |
| 422 | `IMAGE_DECODE_FAILED` | Byte bukan gambar yang bisa didecode |
| 422 | `IMAGE_TOO_LARGE_PIXELS` | `width × height` > `FACE_MAX_PIXELS` |
| 503 | `MODEL_NOT_READY` | Model belum selesai dimuat |
| 500 | `INTERNAL_ERROR` | Bug — detail hanya ke log |

### 4.4 `POST /v1/embed-batch`

Untuk reindex/backfill (Fase 3), **tidak pernah** dipakai di jalur check-in.

Request: `multipart/form-data`, field `images` (berulang), maks
`FACE_MAX_BATCH_IMAGES` (default 16).

```json
200
{
  "data": {
    "items": [
      { "index": 0, "result": { "...sama seperti /v1/embed..." } },
      { "index": 1, "error": { "code": "IMAGE_DECODE_FAILED", "message": "..." } }
    ],
    "model_version": "buffalo_l@v1",
    "took_ms": 1842
  }
}
```

**Satu file rusak tidak menggagalkan seluruh batch** — ia menjadi entri `error` di
posisinya. Backfill 10.000 foto yang batal karena satu file korup adalah kegagalan
operasional yang mahal.

| Status | code | Kondisi |
|---|---|---|
| 413 | `PAYLOAD_TOO_LARGE` | Jumlah gambar melebihi batas, atau total byte melebihi batas |

### 4.5 `POST /v1/compare` — utilitas kalibrasi (bukan jalur produksi)

Dua bentuk request:

```json
// A. dua embedding (JSON)
{ "a": [0.01, ...512...], "b": [0.02, ...512...] }
```

```json
// B. satu embedding vs banyak (JSON)
{ "probe": [ ...512... ], "references": [ [ ...512... ], [ ...512... ] ] }
```

```json
200
{
  "data": {
    "similarities": [0.6721, 0.3140],
    "best_index": 0,
    "best_similarity": 0.6721,
    "metric": "cosine",
    "model_version": "buffalo_l@v1"
  }
}
```

| Status | code | Kondisi |
|---|---|---|
| 422 | `VALIDATION_ERROR` | Panjang vektor ≠ 512, atau ada `NaN`/`Inf` |
| 422 | `VECTOR_NOT_NORMALIZED` | ‖v‖ menyimpang > 1e-3 dari 1.0 — menolak diam-diam akan menghasilkan angka yang terlihat masuk akal tapi tidak berarti |

Ditandai `x-internal-tooling: true` di dokumentasi kontrak, dan `faceclock-api`
**tidak** memanggilnya dari handler mana pun.

---

## 5. Flow

### 5.1 Startup

```
container start
 └─ uvicorn --workers 1
     └─ lifespan startup
         ├─ settings.load() dari env
         ├─ bila FACE_ENGINE_TOKEN kosong DAN APP_ENV != development ⇒ exit(1)
         │     (prior art hanya WARNING; untuk Faceclock ini dinaikkan jadi fatal)
         ├─ MultiPartParser.spool_max_size = max_image_bytes + 1 MB
         │     ← mencegah Starlette menulis frame biometrik ke temp file di disk
         ├─ FaceAnalysis(allowed_modules=["detection","recognition"]).prepare()
         │     ← ~10–35 detik; sengaja di startup, bukan lazy saat request pertama
         └─ log "startup complete in Xs"  (tanpa detail model path)
 └─ HEALTHCHECK mulai lulus setelah start_period 90s
```

`faceclock-api` saat startup:
```
 ├─ GET /ready ke inference
 ├─ bandingkan quality_thresholds & model_version dengan app_settings
 ├─ selisih ⇒ log WARNING "inference config drift" beserta field yang berbeda
 └─ app_settings.face.model_version == "unset" ⇒ log WARNING
        "threshold belum dikalibrasi — lihat Plan/03-Fase2.md § 2.9"
```

### 5.2 Satu request `/v1/embed`

```
1.  Auth: hmac.compare_digest(token) ─ gagal ⇒ 401
2.  Baca file dengan batas byte ─ lampaui ⇒ 413
3.  Sniff MIME dari magic bytes (BUKAN dari Content-Type yang dikirim client)
        di luar whitelist ⇒ 415
4.  Baca header dimensi tanpa decode penuh
        w × h > FACE_MAX_PIXELS ⇒ 422 IMAGE_TOO_LARGE_PIXELS
5.  Decode dari memori (cv2.imdecode) ─ gagal ⇒ 422 IMAGE_DECODE_FAILED
6.  Koreksi orientasi EXIF
7.  Downscale bila sisi terpanjang > 1920 px
8.  SCRFD detect
      0 wajah  ⇒ 200 {usable:false, hints:["no_face"], embedding:null}
      >1 wajah ⇒ hints += "multiple_faces"
9.  Pilih bbox terbesar → norm_crop 112×112 dari 5 keypoint
10. Ukur kualitas PADA CROP TERALIGNED (bukan frame mentah):
      det_score, blur, brightness, yaw, pitch, face_ratio, occlusion_ratio
11. build_hints(metrics, thresholds); usable = (len(hints) == 0)
12. usable == false ⇒ 200 dengan embedding:null   ← tidak pernah bocorkan vektor
13. ArcFace embed → L2-normalize (defensif, meski insightface sudah melakukannya)
14. Susun response
15. finally: del data (lepas referensi byte gambar)
16. log: request_id, usable, hints, quality_score, took_ms, bytes
       TIDAK di-log: nama file, byte, nilai embedding
```

Langkah 3 penting: `Content-Type` datang dari client dan bisa dibohongi. Yang
dipercaya adalah magic bytes.

Langkah 10 penting dan diwarisi dari prior art: sebuah frame bisa terang secara
keseluruhan sementara wajahnya sendiri berada di bayangan. Yang dilihat recognizer
adalah crop-nya, jadi crop itu yang diukur.

### 5.3 Alur pemakaian oleh Fase 3 (enrollment) — pratinjau kontrak

```
Karyawan upload 3 foto
 └─ untuk setiap foto:
      POST /v1/embed
        usable=false ⇒ tolak foto itu, tampilkan teks dari peta hints (§ 2.5)
                       JANGAN simpan apa pun
        usable=true  ⇒ simpan {embedding, quality_score, det_score, model_version}
                       ke face_references  (TIGA BARIS TERPISAH, tidak di-average)
 └─ setelah semua: jumlah referensi aktif >= face.min_reference_photos ⇒ enrollment sah
```

### 5.4 Alur pemakaian oleh Fase 4 (check-in) — pratinjau kontrak

```
POST /attendances/check-in {photo, lat, lng, note}
 1. auth + permission attendance.checkin           [Fase 1]
 2. server_timestamp = now() dari DB               [master plan § 9]
 3. validasi geofence terhadap office_locations    [Fase 4]
 4. POST /v1/embed
       error/timeout/circuit open ⇒ JANGAN approved ⇒ jalur fallback → pending_review
       usable=false               ⇒ tampilkan hints, tawarkan fallback → pending_review
 5. query pgvector (§ 2.4) → similarity TERTINGGI dari referensi aktif
       (filter model_version = app_settings.face.model_version)
 6. similarity >= app_settings.face.similarity_threshold
       ya    ⇒ status = 'approved',       method = 'face'
       tidak ⇒ tawarkan fallback          method = 'fallback', status = 'pending_review'
 7. simpan attendances {matched_similarity, model_version, photo_url, status, ...}
```

Langkah 4 dan 6 adalah tempat aturan master plan § 9 ("fallback tidak boleh langsung
approved") ditegakkan. Ditulis di sini supaya Fase 4 mewarisi, bukan menemukan ulang.

---

## 6. Edge Case & Validasi

| # | Kondisi | Penanganan |
|---|---|---|
| E1 | Tidak ada wajah | `200`, `usable:false`, `hints:["no_face"]`, `embedding:null` |
| E2 | Lebih dari satu wajah | `200`, `hints` memuat `multiple_faces`, `face_count` dilaporkan, `usable:false`, `embedding:null`. **Tidak pernah** diam-diam memilih wajah terbesar sebagai jawaban |
| E3 | Foto buram / gelap / terang / wajah kecil / kepala miring | `200` dengan hint yang sesuai; pemanggil memutuskan (§ 2.5) |
| E4 | Wajah di tepi frame, bbox terpotong | Tetap diproses; `face_ratio` dan `det_score` yang menyaring. Koordinat bbox di-clamp ke batas frame sebelum dikirim |
| E5 | Foto dari layar HP (spoof sederhana) | Fase 2 **tidak** mendeteksinya. Dimitigasi di Fase 6 (paksa capture kamera, larang upload galeri) dan Fase 7 (liveness on-device). Ditulis eksplisit sebagai keterbatasan yang diketahui, bukan diasumsikan tertangani |
| E6 | Wajah bermasker | `occlusion_ratio` dilaporkan tapi **tidak** blocking (§ 2.5). Umumnya `det_score` dan pose gate yang akan menolak |
| E7 | File bukan gambar (`.pdf`, `.exe` di-rename `.jpg`) | Ditolak `415` di sniff magic bytes, sebelum decoder disentuh |
| E8 | Gambar korup separuh | `422 IMAGE_DECODE_FAILED` |
| E9 | JPEG dengan EXIF `Orientation=6` | Diputar sebelum deteksi (§ 2.7a). Test wajib ada |
| E10 | PNG 3 MB yang mengembang jadi 300 MP | `422 IMAGE_TOO_LARGE_PIXELS` sebelum decode penuh (§ 2.7b) |
| E11 | GIF animasi / TIFF multi-frame | `415`. Kalaupun frame pertama bisa didecode, formatnya di luar whitelist |
| E12 | Gambar 1×1 piksel | `no_face` — tidak crash |
| E13 | Body kosong / field `image` kosong | `400 BAD_REQUEST` |
| E14 | Upload 6.1 MB | `413`. Batas ditegakkan **sebelum** seluruh body dibaca ke memori |
| E15 | Token salah | `401` dengan `hmac.compare_digest` (tahan timing attack) |
| E16 | `FACE_ENGINE_TOKEN` kosong di production | **exit(1) saat startup** (perubahan dari prior art yang hanya WARNING) |
| E17 | Request masuk sebelum model selesai dimuat | `503 MODEL_NOT_READY`. Go memetakannya ke `503`, bukan `502` |
| E18 | Dua request bersamaan pada 1 worker | Diserialisasi; batas konkurensi ada di sisi Go (§ 2.6). Latensi naik, tidak ada yang gagal |
| E19 | Container kehabisan memori | Limit memori di compose membuat container ini yang mati, bukan Postgres di sebelahnya. Restart otomatis; `/readyz` Go mencerminkannya |
| E20 | Model tidak ada di image | Gagal saat `lifespan` → container tidak pernah healthy → terlihat di `docker compose ps`, bukan muncul sebagai error misterius saat request pertama |
| E21 | Versi `insightface` naik tanpa sengaja | Versi dipin ketat di `requirements.txt`; CI membandingkan hash `requirements.txt` dan menggagalkan build kalau berubah tanpa perubahan `FACE_MODEL_VERSION` |
| E22 | `model_version` di DB ≠ yang aktif di service | Query compare memfilter `model_version`, jadi vektor lama **tidak ikut** dibandingkan → orang tidak dikenali (aman) alih-alih dikenali salah (berbahaya). Fase 3 menyediakan job reindex |
| E23 | Vektor tidak ter-normalize masuk ke `/v1/compare` | `422 VECTOR_NOT_NORMALIZED` |
| E24 | `X-Request-Id` tidak dikirim | Service membuat sendiri; korelasi lintas service hilang tapi request tetap dilayani |
| E25 | Disk container read-only, ada library yang mencoba menulis cache | `tmpfs /tmp` disediakan; `insightface` diarahkan ke `FACE_MODEL_ROOT` yang sudah di-bake sehingga tidak pernah mencoba download |

---

## 7. Struktur Folder

### 7.1 `services/faceclock-inference/` (mengganti stub Fase 0)

```
services/faceclock-inference/
├── app/
│   ├── __init__.py
│   ├── main.py                 # FastAPI: lifespan, route, exception handler
│   ├── config.py               # Settings + QualityThresholds dari env
│   ├── security.py             # bearer token, compare_digest
│   ├── schemas.py              # Pydantic — snake_case, envelope {data}/{error}
│   ├── images.py               # ← BARU: sniff MIME, EXIF, batas piksel, downscale
│   ├── quality.py              # metrik + hints + score  (port dari prior art)
│   ├── embedder.py             # detect → align → embed → L2-normalize
│   ├── compare.py              # ← BARU: cosine + validasi normalisasi
│   ├── errors.go.md            # (bukan kode) tabel pemetaan error, dirujuk Go
│   └── logging.py              # ← BARU: formatter JSON + redaksi field terlarang
├── tests/
│   ├── conftest.py
│   ├── fixtures/
│   │   ├── one_face.jpg
│   │   ├── two_faces.jpg
│   │   ├── no_face.jpg
│   │   ├── blurry.jpg
│   │   ├── dark.jpg
│   │   ├── bright.jpg
│   │   ├── small_face.jpg
│   │   ├── turned_head.jpg
│   │   ├── exif_orientation_6.jpg     # E9
│   │   ├── decompression_bomb.png     # E10
│   │   ├── corrupt.jpg                # E8
│   │   └── not_an_image.bin           # E7
│   ├── test_quality.py         # murni fungsi, tanpa ONNX
│   ├── test_images.py          # EXIF, MIME sniff, batas piksel, downscale
│   ├── test_compare.py
│   ├── test_security.py
│   ├── test_api_contract.py    # bentuk response untuk setiap fixture
│   └── test_no_disk_write.py   # ← memantau /tmp & cwd selama request
├── calibration/
│   ├── README.md
│   ├── roc.py                  # port dari face-engine/spike/roc.py
│   ├── embed_dataset.py
│   ├── fetch_lfw.py
│   └── requirements-calibration.txt
├── models/                     # di-bake saat build; TIDAK di-commit
│   └── .gitkeep
├── .dockerignore
├── .env.example
├── Dockerfile
├── docker-compose.inference.yml   # standalone, untuk deploy terpisah
├── pyproject.toml                 # ruff + pytest config
├── requirements.txt               # versi dipin ketat
└── README.md
```

### 7.2 Yang ditambahkan di `apps/faceclock-api/`

```
apps/faceclock-api/
├── internal/
│   └── inference/
│       ├── client.go           # (diperluas) Embed, EmbedBatch, Ready
│       ├── types.go            # EmbedResult, Quality, HealthInfo
│       ├── errors.go           # pemetaan HTTP inference → error Fase 0
│       ├── retry.go            # kebijakan retry (§ 2.6)
│       ├── breaker.go          # circuit breaker
│       ├── metrics.go
│       ├── hints.go            # kosakata hints + peta ke pesan Indonesia
│       ├── fake.go             # implementasi Client untuk test Fase 3/4
│       ├── client_test.go      # httptest server memalsukan tiap status
│       └── breaker_test.go
├── migrations/
│   ├── 000011_face_settings.up.sql
│   └── 000011_face_settings.down.sql
└── cmd/api/main.go             # (diubah) cek config drift saat startup (§ 5.1)
```

`fake.go` penting untuk kesinambungan: Fase 3 dan Fase 4 harus bisa menulis
integration test tanpa menyalakan container ONNX 1,5 GB. `FakeClient` dapat
diprogram untuk mengembalikan `usable=false` dengan hint tertentu, embedding
tertentu, atau error — sehingga jalur fallback Fase 4 bisa diuji secara deterministik.

### 7.3 Dokumen yang dihasilkan

```
docs/
├── api/
│   └── fase2-inference.md                     # kontrak lengkap § 4
└── face/
    ├── fase2-calibration-report.md            # § 2.9
    ├── dataset-b-consent-and-retention.md     # § 2.9, UU PDP
    └── model-version-and-reindex.md           # prosedur ganti model
```

---

## 8. Checklist Task

### 8.0 Prasyarat
- [ ] **Konfirmasi D10 (model), D11 (lokasi compare), D12 (port prior art)**
- [ ] **Konfirmasi lisensi**: boleh memakai kode `face-engine/`? Lisensi model InsightFace untuk penggunaan ini?

### 8.1 Port & bersih-bersih
- [ ] Salin `app/embedder.py`, `app/quality.py`, `app/config.py`, `app/security.py` ke `services/faceclock-inference/`
- [ ] Ganti seluruh referensi domain SIMRS/pasien → Faceclock/karyawan di docstring & komentar
- [ ] Ubah `schemas.py` ke `snake_case` + envelope `{data}` / `{error}`
- [ ] Ubah path `/embed` → `/v1/embed`, `/embed-batch` → `/v1/embed-batch`
- [ ] Pertahankan blok anti-spill `MultiPartParser` (dengan komentar alasannya)
- [ ] Naikkan token kosong dari WARNING → fatal saat `APP_ENV != development` (E16)
- [ ] Salin `requirements.txt` dengan versi tetap dipin + komentar alasannya

### 8.2 Kemampuan baru
- [ ] `app/images.py`: sniff MIME dari magic bytes (E7)
- [ ] `app/images.py`: koreksi orientasi EXIF (E9) + fixture test
- [ ] `app/images.py`: batas megapiksel sebelum decode penuh (E10) + fixture test
- [ ] `app/images.py`: downscale > 1920 px, laporkan di field `frame`
- [ ] `app/compare.py` + endpoint `/v1/compare` dengan validasi normalisasi (E23)
- [ ] `GET /ready` dengan introspeksi konfigurasi lengkap (§ 4.2)
- [ ] `app/logging.py`: JSON log + `X-Request-Id` + redaksi (tanpa nama file/byte/embedding)
- [ ] Pastikan `embedding = null` setiap kali `usable = false` (§ 4.3)
- [ ] Field `frame` di setiap response embed

### 8.3 Container & keamanan
- [ ] `Dockerfile` multi-stage, model di-bake, model tak terpakai dibuang, non-root, `HEALTHCHECK`
- [ ] `docker-compose.inference.yml` standalone: `read_only: true`, `tmpfs /tmp`, `no-new-privileges`, limit memori 2G, **tanpa** `ports:`
- [ ] Integrasikan ke `deploy/docker-compose.yml` (mengganti stub Fase 0)
- [ ] `.env.example` lengkap
- [ ] Verifikasi: `docker exec` → tidak ada file baru di filesystem setelah 100 request

### 8.4 Klien Go
- [ ] `types.go` sesuai kontrak § 4
- [ ] `client.go`: `Ready`, `Embed`, `EmbedBatch` (multipart, streaming, tanpa buffer ganda)
- [ ] `errors.go`: pemetaan lengkap tabel § 2.6
- [ ] `retry.go`: maks 1 retry, hanya pada kelas error yang benar
- [ ] `breaker.go` + test
- [ ] `metrics.go`
- [ ] `hints.go`: kosakata + peta pesan Indonesia (dipakai Fase 6/7)
- [ ] `fake.go`: `FakeClient` yang bisa diprogram — **dependency Fase 3 & 4**
- [ ] `client_test.go`: `httptest` yang memalsukan 200-usable, 200-unusable, 401, 413, 415, 422, 503, timeout
- [ ] Test: client tidak pernah mencatat byte gambar atau nilai embedding

### 8.5 Integrasi ke `faceclock-api`
- [ ] `/readyz` memanggil `/ready` (bukan `/health`) dan melaporkan `model_version`
- [ ] Pemeriksaan config drift saat startup (§ 5.1) + WARNING
- [ ] Migration `000011_face_settings`
- [ ] WARNING saat `face.model_version == "unset"`

### 8.6 Kalibrasi
- [ ] Port `roc.py` + `embed_dataset.py` ke `calibration/`
- [ ] Jalankan pada LFW → kurva FAR/FRR baseline
- [ ] Kumpulkan Dataset B (15–30 karyawan) **dengan consent tertulis**
- [ ] `docs/face/dataset-b-consent-and-retention.md`
- [ ] Jalankan pada Dataset B → pilih threshold pada FAR ≤ 0.1%
- [ ] `docs/face/fase2-calibration-report.md`
- [ ] Set `app_settings.face.similarity_threshold` dan `face.model_version`
- [ ] Hapus Dataset B sesuai kebijakan retensi setelah kalibrasi

### 8.7 Benchmark (D10)
- [ ] Ukur `buffalo_l`: p50/p95/p99, memori, cold start
- [ ] Ukur `buffalo_s`: idem
- [ ] Bandingkan akurasi keduanya pada Dataset B
- [ ] Tulis hasil + keputusan final di laporan kalibrasi
- [ ] `docs/face/model-version-and-reindex.md`: prosedur ganti model

### 8.8 Dokumentasi & penutup
- [ ] `docs/api/fase2-inference.md` — kontrak 5 endpoint, lengkap dengan contoh & error
- [ ] `docs/adr/0002-inference-prior-art.md` diperbarui: apa yang di-port, apa yang diubah, lisensi
- [ ] `services/faceclock-inference/README.md`
- [ ] `DONE-Fase-2.md` sesuai Protokol Handoff master plan § 10.4

---

## 9. Dependencies

**Prasyarat:**

| Dari | Yang dibutuhkan |
|---|---|
| Fase 0 | Folder `services/faceclock-inference/` + wiring compose + network internal |
| Fase 0 | `internal/inference/client.go` (kerangka) dan `/readyz` |
| Fase 0 | Envelope `{data}`/`{error}` + katalog error code + konvensi `snake_case` (D4) |
| Fase 0 | Aturan logging (larangan mencatat gambar/embedding) |
| Fase 0 | Ekstensi `vector` aktif di Postgres |
| Fase 1 | Tabel `app_settings` + endpoint-nya (untuk menyimpan threshold & model_version) |
| — | Bobot model: sudah tersedia di `face-engine/models/buffalo_l/` |
| — | Consent + sukarelawan untuk Dataset B |

> Fase 2 **tidak** menunggu Fase 1 selesai untuk mulai. Yang butuh Fase 1 hanyalah
> langkah terakhir (§ 8.6 menyimpan threshold ke `app_settings`) dan migration
> `000011`. Sisanya — service, kontrak, client Go, kalibrasi — bisa berjalan paralel.

**Yang bergantung pada fase ini:**

| Fase | Mengambil apa |
|---|---|
| Fase 3 | `/v1/embed`, kontrak data § 3.2 (`vector(512)`, L2-normalized, `model_version`), kosakata `hints`, `FakeClient`, `/v1/embed-batch` untuk reindex |
| Fase 4 | Query pgvector § 2.4, threshold dari `app_settings`, aturan "inference gagal ≠ approved" (§ 2.6), `FakeClient` untuk test |
| Fase 5 | Menampilkan `matched_similarity`, `quality_score`, dan `hints` di antrian approval; panel setting threshold |
| Fase 6 | Peta `hints` → teks Indonesia (§ 2.5); batas MIME/ukuran (`face.accepted_mime_types`, `face.max_image_bytes`) untuk konfigurasi `getUserMedia` + kompresi sebelum upload |
| Fase 7 | Peta `hints` yang sama; konversi HEIC→JPEG wajib di sisi app; liveness on-device **melengkapi**, bukan menggantikan, gate kualitas server |

---

## 10. Definition of Done

1. `docker compose up` menghasilkan `faceclock-inference` `healthy` dalam ≤ 90 detik,
   tanpa mengunduh apa pun dari internet saat boot.
2. `GET /ready` mengembalikan `model_version`, `embedding_dim: 512`, dan seluruh
   `quality_thresholds` yang sedang aktif.
3. `POST /v1/embed` dengan foto wajah jelas → `200`, `usable: true`,
   `embedding` sepanjang **tepat 512**, dan ‖embedding‖ = 1.0 ± 1e-5.
4. Untuk **setiap** hint di § 2.5 ada fixture yang memicunya, dan test yang
   memastikan hint itu muncul.
5. `usable: false` **selalu** disertai `embedding: null` — diuji untuk semua fixture negatif.
6. E7–E14, E17, E23 semuanya punya test yang lulus.
7. JPEG dengan `Orientation=6` menghasilkan `usable: true` (bukan `no_face`).
8. `test_no_disk_write.py` lulus: setelah 100 request berisi gambar, tidak ada file
   baru di `/tmp` maupun di working directory container.
9. Log service tidak memuat nama file, byte gambar, maupun nilai embedding —
   dibuktikan dengan test yang memindai output log.
10. Container berjalan `read_only: true` dan sebagai user non-root.
11. Klien Go lulus test untuk setiap status yang mungkin (200-usable, 200-unusable,
    401, 413, 415, 422, 503, timeout, connection refused), dengan pemetaan error
    persis seperti tabel § 2.6.
12. Circuit breaker terbukti: 5 kegagalan berturut-turut membuka sirkuit, dan
    request berikutnya gagal cepat (< 50 ms) alih-alih menunggu 6 detik.
13. `FakeClient` tersedia dan sudah dipakai minimal satu test di `faceclock-api`.
14. `/readyz` Go melaporkan `model_version` inference, dan menampilkan WARNING
    di log bila konfigurasi drift dari `app_settings`.
15. Benchmark § 2.8 terpenuhi (p95 ≤ 900 ms) pada model yang dipilih, atau
    ada keputusan tertulis untuk pindah model.
16. `docs/face/fase2-calibration-report.md` ada, memuat kurva FAR/FRR, threshold
    terpilih, dan angka FAR/FRR di titik itu. Bila hanya berbasis LFW, ia ditandai
    **provisional** dan Dataset B tercatat sebagai blocker DoD Fase 4.
17. `app_settings.face.similarity_threshold` dan `face.model_version` sudah terisi
    nilai nyata (bukan `"unset"`).
18. `docs/api/fase2-inference.md` cocok dengan implementasi (diverifikasi contract test).
19. `make lint` & `make test` lulus (ruff + pytest + go test); CI hijau.
20. `DONE-Fase-2.md` ada, memuat keputusan D10/D11/D12 final beserta alasannya.

---

## 11. Cara Test / Verifikasi

### 11.1 Manual dari dalam network compose

Service tidak dipublish ke host, jadi pengujian dilakukan dari container lain:

```bash
CE="docker compose -f deploy/docker-compose.yml exec faceclock-api"
TOKEN=$(grep '^INFERENCE_TOKEN=' deploy/.env | cut -d= -f2)
BASE=http://faceclock-inference:8000

# health & ready
$CE curl -s $BASE/health | jq
$CE curl -s -H "Authorization: Bearer $TOKEN" $BASE/ready | jq

# embed satu foto
$CE curl -s -H "Authorization: Bearer $TOKEN" \
   -F "image=@/tmp/wajah.jpg" $BASE/v1/embed \
 | jq '{usable: .data.usable, hints: .data.hints,
        dim: (.data.embedding|length), qs: .data.quality_score,
        ms: .data.took_ms}'

# norma vektor harus 1.0
$CE curl -s -H "Authorization: Bearer $TOKEN" -F "image=@/tmp/wajah.jpg" $BASE/v1/embed \
 | jq '[.data.embedding[] | .*.] | add | sqrt'      # ≈ 1.0

# token salah
$CE curl -s -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer salah" \
   -F "image=@/tmp/wajah.jpg" $BASE/v1/embed        # 401

# bukan gambar
$CE curl -s -H "Authorization: Bearer $TOKEN" \
   -F "image=@/etc/hostname" $BASE/v1/embed | jq .error.code   # UNSUPPORTED_MEDIA_TYPE
```

### 11.2 Test kontrak otomatis (pytest)

Satu test per fixture, membandingkan bentuk response dengan tabel harapan yang
ditulis sebagai data:

| Fixture | `usable` | `hints` memuat | `embedding` |
|---|---|---|---|
| `one_face.jpg` | `true` | `[]` | 512 float |
| `two_faces.jpg` | `false` | `multiple_faces` | `null` |
| `no_face.jpg` | `false` | `no_face` | `null` |
| `blurry.jpg` | `false` | `too_blurry` | `null` |
| `dark.jpg` | `false` | `too_dark` | `null` |
| `bright.jpg` | `false` | `too_bright` | `null` |
| `small_face.jpg` | `false` | `face_too_small` | `null` |
| `turned_head.jpg` | `false` | `head_turned` | `null` |
| `exif_orientation_6.jpg` | `true` | `[]` | 512 float |
| `corrupt.jpg` | — | HTTP 422 `IMAGE_DECODE_FAILED` | — |
| `not_an_image.bin` | — | HTTP 415 | — |
| `decompression_bomb.png` | — | HTTP 422 `IMAGE_TOO_LARGE_PIXELS` | — |

### 11.3 Test "tidak menyentuh disk"

```python
# tests/test_no_disk_write.py — inti idenya
before = snapshot_files(["/tmp", os.getcwd()])
for _ in range(100):
    client.post("/v1/embed", files={"image": open("fixtures/one_face.jpg","rb")},
                headers={"Authorization": f"Bearer {TOKEN}"})
after = snapshot_files(["/tmp", os.getcwd()])
assert before == after      # nol file baru
```

Ini menguji hal yang sama yang ditemukan prior art: `MultiPartParser` Starlette
menumpahkan part > 1 MB ke temp file di disk **tanpa error dan tanpa log**. Tanpa
test ini, regresi akan lolos setiap review dan baru terlihat sebagai frame
biometrik di `/tmp` production.

### 11.4 Test properti embedding

```python
def test_same_photo_same_vector(client):
    a = embed("fixtures/one_face.jpg")["embedding"]
    b = embed("fixtures/one_face.jpg")["embedding"]
    assert cosine(a, b) > 0.9999          # deterministik

def test_norm_is_one(client):
    v = embed("fixtures/one_face.jpg")["embedding"]
    assert abs(norm(v) - 1.0) < 1e-5

def test_different_people_low_similarity(client):
    a = embed("fixtures/person_a.jpg")["embedding"]
    b = embed("fixtures/person_b.jpg")["embedding"]
    assert cosine(a, b) < 0.35            # jauh di bawah threshold mana pun
```

### 11.5 Test klien Go

`httptest.Server` memalsukan setiap status, tanpa menyalakan ONNX:

```go
func TestClientMapsErrors(t *testing.T) {
    cases := []struct{ upstream int; body string; wantCode string }{
        {401, `{"error":{"code":"UNAUTHENTICATED"}}`,   "INTERNAL_ERROR"},
        {413, `{"error":{"code":"PAYLOAD_TOO_LARGE"}}`, "PAYLOAD_TOO_LARGE"},
        {415, `{"error":{"code":"UNSUPPORTED_MEDIA_TYPE"}}`, "UNSUPPORTED_MEDIA_TYPE"},
        {422, `{"error":{"code":"IMAGE_DECODE_FAILED"}}`,    "VALIDATION_ERROR"},
        {503, `{"error":{"code":"MODEL_NOT_READY"}}`,        "SERVICE_UNAVAILABLE"},
        {500, `{"error":{"code":"INTERNAL_ERROR"}}`,         "UPSTREAM_ERROR"},
    }
    ...
}
```

Plus: test timeout (server sengaja tidur 10 detik → `UPSTREAM_TIMEOUT` dalam ~6 detik),
test retry (server gagal sekali lalu berhasil → client berhasil dengan tepat 2 panggilan),
test tidak-retry (server balas 422 → client memanggil tepat 1×).

### 11.6 Benchmark

```bash
# 200 request berurutan, satu foto yang sama
for i in $(seq 1 200); do
  $CE curl -s -o /dev/null -w "%{time_total}\n" -H "Authorization: Bearer $TOKEN" \
     -F "image=@/tmp/wajah.jpg" $BASE/v1/embed
done | sort -n | awk '{a[NR]=$1} END {print "p50", a[int(NR*0.5)], "p95", a[int(NR*0.95)], "p99", a[int(NR*0.99)]}'

docker stats --no-stream faceclock-inference
```

### 11.7 Kalibrasi

```bash
cd services/faceclock-inference/calibration
python fetch_lfw.py                       # atau pakai dataset di face-engine/spike/
python embed_dataset.py --dataset ./dataset/lfw   --out ./out/lfw.npz
python roc.py --embeddings ./out/lfw.npz  --report ../../../docs/face/roc-lfw.md

python embed_dataset.py --dataset ./dataset/internal --out ./out/internal.npz
python roc.py --embeddings ./out/internal.npz --target-far 0.001 \
              --report ../../../docs/face/fase2-calibration-report.md
```

Kemudian setel lewat API (butuh `settings.update`, Fase 1):

```bash
curl -X PUT $API/settings/face.similarity_threshold \
  -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
  -d '{"value": 0.42}'
curl -X PUT $API/settings/face.model_version \
  -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
  -d '{"value": "buffalo_l@v1"}'
```

---

## 12. Referensi Silang ke "Isu Lintas-Fase" (Master Plan § 9)

| Isu lintas-fase | Bagaimana Fase 2 memenuhinya |
|---|---|
| **Threshold configurable** | Deliverable utama § 2.9: angka lahir dari FAR/FRR terukur, tersimpan di `app_settings`, tidak pernah hardcode di Go maupun Python |
| **Data biometrik = data sensitif (UU PDP)** | Service tanpa database & tanpa tulis disk (dibuktikan test § 11.3); anti-spill multipart; port tidak dipublish; bearer token internal; log tanpa gambar/embedding/nama file; consent + retensi untuk Dataset B |
| **Keamanan fallback** | Aturan tegas di § 2.6: kegagalan inference **tidak pernah** menghasilkan `approved`. Diwariskan ke Fase 4 sebagai test |
| **Anti-spoofing / liveness** | Fase 2 secara eksplisit **tidak** menyelesaikannya (E5). Yang disediakan: metrik kualitas yang membuat foto-dari-layar lebih mudah gagal, dan kosakata `hints` yang dipakai Fase 6/7 |
| **Timestamp server-side** | Tidak relevan di sini — service ini tidak menghasilkan waktu yang dipercaya. `took_ms` murni telemetri |
| **RBAC di level API** | Inference tidak mengenal user sama sekali; otorisasi seluruhnya di `faceclock-api` (Fase 1). Bearer token di sini adalah autentikasi antar-service, bukan otorisasi pengguna |

---

## 13. Peta Kesinambungan Fase 0 → 7

Bagian ini memverifikasi bahwa apa yang dibangun di Fase 0–2 benar-benar menopang
seluruh sisa master plan, dan menandai apa yang masih menggantung.

### 13.1 Rantai artefak

| Artefak (dibuat di) | Fase 3 | Fase 4 | Fase 5 | Fase 6 | Fase 7 |
|---|---|---|---|---|---|
| Middleware chain + envelope error (F0) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Konvensi skema DB: uuid, `timestamptz`, `now()` sisi DB (F0) | ✅ | ✅ | ✅ | — | — |
| `storage.Store` untuk foto (F0) | ✅ foto referensi | ✅ foto absensi | ✅ tampilkan | ✅ upload | ✅ upload |
| Aturan logging anti-biometrik (F0) | ✅ | ✅ | ✅ | — | — |
| `snake_case` + tipe envelope TS (F0) | — | — | ✅ | ✅ | ✅ |
| `Principal{user_id, employee_id, permissions}` (F1) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Pola ownership `_self` → 404 (F1 § 5.4) | ✅ `face.*_self` | ✅ `attendance.read_self` | ✅ | ✅ | ✅ |
| `TestAllRoutesHaveGuards` (F1 § 5.3) | ✅ menangkap endpoint baru | ✅ | ✅ | — | — |
| Permission `face.*`, `attendance.*`, `location.*` sudah di-seed (F1 § 2.3) | ✅ | ✅ | ✅ | ✅ | ✅ |
| `app_settings` + validator per-key (F1 § 4.6) | ✅ `min_reference_photos` | ✅ threshold, geofence, jam kerja | ✅ panel setting | ✅ tampilkan radius | ✅ |
| `audit_logs` + recorder (F1 § 3.8) | ✅ enroll/re-enroll | ✅ approve/reject | ✅ | — | — |
| `GET /auth/me` (F1) | — | — | ✅ guard UI | ✅ | ✅ |
| Kontrak `/v1/embed` + kosakata `hints` (F2 § 2.5) | ✅ | ✅ | ✅ tampilkan alasan | ✅ coach text | ✅ coach text |
| Kontrak data `vector(512)` L2-normalized (F2 § 3.2) | ✅ definisi tabel | ✅ query similarity | — | — | — |
| Query pgvector "similarity tertinggi" (F2 § 2.4) | — | ✅ | — | — | — |
| `face.model_version` + prosedur reindex (F2 § 2.2) | ✅ kolom + job | ✅ filter query | ✅ tombol reindex | — | — |
| `inference.FakeClient` (F2 § 7.2) | ✅ test | ✅ test | — | — | — |
| Aturan "inference gagal ≠ approved" (F2 § 2.6) | — | ✅ **test wajib** | — | — | — |
| `/v1/embed-batch` (F2 § 4.4) | ✅ backfill | — | ✅ dipicu admin | — | — |
| Batas MIME & ukuran foto (F2 § 3.1) | ✅ validasi | ✅ validasi | — | ✅ kompresi client | ✅ konversi HEIC |

### 13.2 Yang sengaja ditunda ke fase berikutnya

| Hal | Ditunda ke | Kenapa aman ditunda |
|---|---|---|
| Tabel `face_references` | Fase 3 | Kontrak kolomnya sudah dikunci di § 3.2, jadi Fase 3 mengimplementasi, bukan mendesain |
| Tabel `attendances`, `office_locations` | Fase 4 | Permission `attendance.*` / `location.*` dan setting geofence sudah ada sejak Fase 1 |
| Consent form biometrik | Fase 3 | Master plan menaruhnya di onboarding enrollment. Kebijakan retensi Dataset B (F2 § 2.9) adalah latihannya |
| Kebijakan retensi foto absensi | Fase 4 | Butuh tahu volume nyata; `storage.Store` sudah menyediakan `Delete()` |
| Liveness sungguhan | Fase 7 | Fase 6 memitigasi dengan memaksa capture kamera; keterbatasannya ditulis eksplisit (E5) |
| Index ANN pada embedding | **Tidak pernah** | Sistem ini 1:1 verification, bukan 1:N (§ 2.4) |
| Optimistic locking (kolom `version`) | Bila jadi masalah nyata | Fase 1 E37 mencatatnya sebagai keputusan sadar |

### 13.3 Risiko lintas-fase yang belum terselesaikan

> Status terkini tiap risiko dilacak di [04-Fase3.md § 9](04-Fase3.md#9-dependencies)
> dan [05-Fase4.md § 9](05-Fase4.md#9-dependencies). Ringkas: R4, R5, R7 sudah
> diselesaikan di Fase 3/4; **R6 sudah ditutup** (D1 dikunci monorepo, 2026-09-04);
> R1, R2, R3 masih terbuka.

| # | Risiko | Fase yang harus menyelesaikan | Catatan |
|---|---|---|---|
| R1 | **Lisensi model InsightFace** untuk penggunaan komersial | **Sekarang** (§ 2.1) | Menemukan ini di Fase 7 berarti seluruh kalibrasi harus diulang dengan model lain |
| R2 | Dataset B (kalibrasi pada populasi nyata) belum tentu bisa dikumpulkan saat Fase 2 | Fase 2, jika tidak → blocker DoD Fase 4 | Threshold dari LFW saja tidak sah untuk produksi (§ 2.9) |
| R3 | Spoofing foto-dari-layar di web | Fase 6 (mitigasi) + Fase 7 (liveness) | Fase 2 hanya menaikkan biayanya, tidak menutupnya (E5) |
| R4 | Karyawan menolak consent biometrik | Fase 3 | Harus ada jalur absensi alternatif yang sah, bukan pengecualian ad-hoc |
| R5 | Reindex saat model diganti setelah ada ribuan referensi | Fase 3 (job) + Fase 5 (UI) | Prosedur ditulis di `docs/face/model-version-and-reindex.md` |
| R6 | ~~Monorepo vs multi-repo belum final (Fase 0 D1)~~ | ✅ **DITUTUP** | Dikunci **monorepo** pada 2026-09-04 ([Fase 0 § 2.1](01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)) |
| R7 | Volume foto absensi (setiap check-in menyimpan foto) | Fase 4 | `LocalStore` cukup untuk dev; production kemungkinan butuh S3/MinIO — `storage.Store` sudah menyiapkan jalurnya |
