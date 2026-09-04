# Fase 0 — Fondasi & Setup

> Turunan detail dari **[00MasterPlan.md](00MasterPlan.md) § Fase 0**.
> Ukuran: 🟢 Kecil (boleh langsung eksekusi), tapi dokumen ini tetap dibuat karena
> keputusan struktural yang diambil di sini mengunci seluruh fase berikutnya.
>
> **Status:** DRAFT — ada 5 keputusan yang butuh konfirmasi user (lihat § 2.0).

---

## 1. Tujuan Fase

### Kenapa fase ini ada
Fase 1–7 semuanya menambahkan kode di atas kerangka yang sama. Kalau kerangka itu
belum ada saat Fase 1 dimulai, setiap fase akan mengarang strukturnya sendiri —
dan biaya menyatukannya kembali jauh lebih mahal daripada menyepakatinya sekarang.

Fase 0 **tidak menulis fitur apa pun**. Fase 0 menjawab pertanyaan-pertanyaan yang
akan ditanyakan berulang kali oleh fase lain:
- Di mana kode diletakkan?
- Bagaimana cara menjalankan seluruh sistem dengan satu perintah?
- Bagaimana bentuk response API (sukses maupun error)?
- Bagaimana cara menambah tabel database?
- Dari mana config dibaca, dan mana yang rahasia?
- Bagaimana bentuk log, dan apa yang **haram** masuk log?

### Hasil akhir yang diharapkan
Seseorang yang baru clone repo ini bisa menjalankan:

```bash
cp deploy/.env.example deploy/.env
make up
curl http://localhost:8080/healthz     # → 200 {"status":"ok"}
curl http://localhost:8080/readyz      # → 200, database + inference terkoneksi
```

…dan mendapat 4 container sehat (`postgres`, `faceclock-api`, `faceclock-web`,
`faceclock-inference`), database dengan ekstensi `pgvector` aktif, serta linter
dan test yang lulus di semua bahasa.

### Yang TIDAK dikerjakan di fase ini
- Tabel bisnis apa pun (users/employees/roles → Fase 1).
- Autentikasi / JWT (→ Fase 1).
- Model InsightFace sungguhan — di Fase 0 `faceclock-inference` hanya **stub**
  yang mengembalikan `/health` dan `/v1/embed` dummy, supaya `/readyz` di Go bisa
  diuji tanpa menunggu Fase 2.
- UI apa pun selain halaman placeholder yang membuktikan Vite dev server dan
  proxy ke API berjalan.

---

## 2. Scope Detail

### 2.0 Keputusan teknis yang harus dikunci di Fase 0

Semua fase berikutnya mengasumsikan keputusan di tabel ini. Kolom **Status**
menandai mana yang sudah aman diasumsikan dan mana yang menunggu user.

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D1 | Struktur repo | **Monorepo** (lihat § 2.1) | ✅ **TERKUNCI** — dikonfirmasi user 2026-09-04 |
| D2 | HTTP router Go | `go-chi/chi/v5` | ⚠️ **BUTUH KONFIRMASI** |
| D3 | Tooling migration | `golang-migrate` (file SQL `.up`/`.down`) | ⚠️ **BUTUH KONFIRMASI** |
| D4 | Konvensi JSON di wire | `snake_case` di **semua** service | ⚠️ **BUTUH KONFIRMASI** |
| D5 | DB access layer | `pgx/v5` + `sqlc` (query SQL, kode generated) | ⚠️ **BUTUH KONFIRMASI** |
| D6 | Frontend tooling | Vite + React 18 + TypeScript + Tailwind + TanStack Query + React Router | Diasumsikan |
| D7 | Logging | `log/slog` (stdlib) handler JSON | Diasumsikan |
| D8 | Timezone | Semua kolom waktu `timestamptz`, proses & DB berjalan di **UTC**; konversi ke `Asia/Jakarta` hanya di layer presentasi | Diasumsikan |
| D9 | Penyimpanan foto | Local volume di dev, di balik interface `storage.Store` supaya bisa diganti S3/MinIO | Diasumsikan |

Alasan tiap rekomendasi ada di § 2.1–2.9.

---

### 2.1 D1 — Monorepo vs Multi-repo ✅ TERKUNCI

> ✅ **KEPUTUSAN FINAL: Monorepo.** Dikonfirmasi user pada **2026-09-04**.
> Analisis di bawah dipertahankan apa adanya sebagai catatan alasan — bukan lagi
> sebagai pilihan terbuka. Seluruh dokumen fase (01–08) mengasumsikan monorepo
> secara pasti; tidak ada lagi cabang multi-repo yang perlu dipertimbangkan.
> Rekamannya wajib ditulis di `docs/adr/0001-monorepo.md` saat Fase 0 dieksekusi.

Master plan sengaja menggantung keputusan ini. Berikut analisis yang mendasarinya.

#### Keputusan: **Monorepo**, dengan tiap service tetap *independently deployable*.

**Alasan mendukung monorepo di project ini:**

1. **Kontrak antar-service masih bergerak.** Fase 2 mendefinisikan kontrak
   `faceclock-api` ↔ `faceclock-inference`, dan Fase 3–4 hampir pasti mengubahnya
   (tambah field quality, tambah hint). Di multi-repo, satu perubahan kontrak =
   2 PR di 2 repo + koordinasi versi. Di monorepo = 1 commit atomik yang mengubah
   server dan client sekaligus, dan CI menguji keduanya bersama.
2. **Tim kecil.** Keuntungan utama multi-repo adalah isolasi izin akses dan
   siklus rilis terpisah antar-tim. Keduanya tidak berlaku di sini.
3. **Satu `docker compose up`.** Master plan menjadikan ini deliverable Fase 0.
   Di multi-repo, compose file harus hidup di salah satu repo dan mereferensi
   image yang sudah ter-publish, atau di repo "infra" ketiga.
4. **Satu sumber kebenaran untuk dokumen fase.** Folder `Plan/` dan
   `DONE-Fase-N.md` (Protokol Handoff § 10) sudah ada di root project ini.
5. **Onboarding.** Satu clone, satu `make up`.

**Risiko monorepo dan mitigasinya:**

| Risiko | Mitigasi |
|---|---|
| Build context Docker jadi raksasa | Setiap `Dockerfile` punya context sendiri (`apps/faceclock-api/`, bukan root) + `.dockerignore` ketat |
| Coupling tidak sengaja (Go meng-import kode Python, dsb.) | Tidak mungkin lintas bahasa; antar-modul Go dijaga lewat `internal/` dan review |
| CI jalan semua padahal yang berubah cuma web | Path filter di GitHub Actions (`paths:` per job) |
| `faceclock-inference` harus bisa dideploy sendiri ke host lain | Folder service-nya self-contained: punya `Dockerfile`, `requirements.txt`, `.env.example`, dan compose file standalone sendiri. Tidak ada satu pun import ke luar foldernya |

**Alternatif yang ditolak — multi-repo.** Ia akan dipilih bila `faceclock-inference`
dikelola pihak/tim berbeda, atau bila ada kebutuhan compliance yang melarang kode
yang menyentuh data biometrik berada satu repo dengan kode aplikasi. Kedua kondisi
itu **tidak berlaku** di project ini, dan keputusan sudah dikunci. Bila salah satunya
muncul di kemudian hari, memisahkan repo tetap mungkin — tapi biayanya naik seiring
bertambahnya kode, jadi perubahan itu harus jadi keputusan sadar dengan ADR baru.

> ✅ **Terkunci.** Seluruh § 7 (struktur folder) dan seluruh dokumen fase 01–08
> mengasumsikan monorepo. Tidak ada lagi percabangan yang perlu dipertimbangkan
> saat eksekusi.

#### Struktur monorepo yang diusulkan

Root = folder project saat ini (`03 FaceClock/`).

```
faceclock/                          # = root project sekarang
├── Plan/                           # sudah ada — dokumen fase
├── apps/
│   ├── faceclock-api/              # Go
│   └── faceclock-web/              # React
├── services/
│   └── faceclock-inference/        # Python (diisi Fase 2)
├── deploy/
│   ├── docker-compose.yml
│   ├── docker-compose.override.yml # override khusus dev (hot reload, port publish)
│   └── .env.example
├── docs/                           # ADR + kontrak API
├── .github/workflows/
├── Makefile
├── .gitignore
├── .editorconfig
└── README.md
```

#### Catatan tentang folder `face-engine/` yang sudah ada

Di root project sudah ada folder `face-engine/` — inference service Python/FastAPI
+ InsightFace milik project lain (SIMRS/VitaCore) yang sudah matang. Folder ini
adalah **prior art yang sangat berharga untuk Fase 2** (lihat [03-Fase2.md](03-Fase2.md)).

Aturan di Fase 0:
- **Jangan diubah, jangan dihapus, jangan dijadikan `services/faceclock-inference/`.**
  Porting-nya adalah pekerjaan Fase 2 yang dilakukan secara sadar, bukan `mv`.
- **Wajib di-exclude dari git.** Isinya `.venv/`, `models/*.onnx` (~330 MB), dan
  `spike/dataset/lfw/` (dataset LFW, ratusan MB). Kalau ini ter-commit, repo rusak
  permanen. Tambahkan ke `.gitignore` **sebelum** `git init` / commit pertama.
- Task Fase 0 hanya: catat keberadaannya di `docs/adr/0002-inference-prior-art.md`.

---

### 2.2 D2 — HTTP router Go ⚠️ BUTUH KONFIRMASI

**Rekomendasi: `go-chi/chi/v5`.**

- Handler-nya `http.HandlerFunc` biasa → middleware apa pun dari ekosistem
  `net/http` bisa dipakai, dan handler bisa diuji dengan `httptest` tanpa
  framework apa pun.
- Middleware RBAC di Fase 1 akan menempel di level *route group* — chi punya
  `r.Group()` / `r.Route()` yang membuat ini eksplisit dan mudah di-audit.
  Ini penting karena master plan § 9 mensyaratkan RBAC **konsisten di level API**;
  yang paling sering gagal adalah endpoint baru yang lupa dipasangi guard.
- Alternatif `gin` / `echo` lebih cepat ditulis tapi mengunci signature handler ke
  tipe framework, dan `gin` punya kebiasaan panic-recovery yang menelan error.

Kalau tim lebih familiar `gin`/`echo`, silakan override — seluruh dokumen ini
tetap valid, yang berubah hanya signature handler dan cara mount middleware.

---

### 2.3 D3 — Tooling migration ⚠️ BUTUH KONFIRMASI

**Rekomendasi: `golang-migrate/migrate`** dengan file SQL polos.

```
apps/faceclock-api/migrations/
├── 000001_enable_extensions.up.sql
├── 000001_enable_extensions.down.sql
└── ...
```

- SQL mentah, bukan DSL. Ini penting karena Fase 1 memakai fitur Postgres spesifik
  (`citext`, partial unique index, `ON DELETE RESTRICT`) dan Fase 3 memakai tipe
  `vector(512)` dari pgvector — DSL ORM biasanya tidak punya tipe ini.
- Ada CLI dan library. Library dipakai untuk `migrate up` otomatis saat container
  API start di **dev**; di production migration dijalankan sebagai step terpisah.
- Alternatif `pressly/goose` setara; pilih salah satu dan konsisten.

**Aturan migration (berlaku semua fase):**
- Satu perubahan logis = satu pasang file. Jangan pernah edit migration yang sudah
  pernah jalan di mesin orang lain — buat migration baru.
- Setiap `.up.sql` **wajib** punya `.down.sql` yang benar-benar membalikkan.
- Migration tidak boleh berisi data bisnis. Seed = program terpisah (§ 2.7 / Fase 1).

---

### 2.4 D4 — Konvensi JSON `snake_case` ⚠️ BUTUH KONFIRMASI

**Rekomendasi: `snake_case` untuk semua field JSON, di semua service.**

Alasan: kolom database `snake_case`, service Python `snake_case`, dan payload Go
di-tag eksplisit. Satu konvensi = nol layer pemetaan = nol tempat field bisa
hilang diam-diam. React membayar ongkosnya di satu tempat saja (tipe TypeScript
yang ditulis mengikuti wire format), dan itu jauh lebih murah daripada dua
konvensi yang harus diterjemahkan di tiga batas service.

Konsekuensi yang harus ditegakkan sejak Fase 0:
- Setiap struct Go yang menyeberang HTTP punya tag eksplisit:
  `json:"employee_id"`. Tidak ada yang mengandalkan default Go (PascalCase).
- Model Pydantic di `faceclock-inference` memakai nama field `snake_case` apa
  adanya (tidak seperti `face-engine/app/schemas.py` yang meng-alias ke camelCase
  karena consumer-nya Node).

---

### 2.5 D5 — Akses database ⚠️ BUTUH KONFIRMASI

**Rekomendasi: `jackc/pgx/v5` (pool) + `sqlc` untuk generate kode dari SQL.**

- `pgx` adalah driver Postgres native — ini **wajib**, bukan preferensi: dukungan
  tipe kustom pgx yang membuat `vector(512)` pgvector bisa di-scan langsung
  (`pgvector-go` menyediakan tipe ini untuk pgx).
- `sqlc` membaca `queries/*.sql` + schema, lalu meng-generate struct dan method
  Go yang type-safe. Query tetap SQL yang bisa dibaca dan di-`EXPLAIN`.
- Alternatif: `pgx` + query manual (paling sedikit tooling, paling banyak
  boilerplate) atau GORM (paling cepat menulis, tapi query yang dihasilkan sulit
  dikontrol — buruk untuk query rekap absensi di Fase 5 dan similarity search di
  Fase 4).

Kalau `sqlc` dianggap tooling berlebih, fallback yang aman: `pgx` + repository
manual. Yang **tidak** direkomendasikan adalah ORM full-featured.

---

### 2.6 Scaffolding `faceclock-api` (Go)

Yang dibangun di Fase 0 — kerangka, bukan fitur:

| Komponen | Isi |
|---|---|
| `cmd/api/main.go` | Baca config → init logger → connect DB → run migration (dev) → build router → `http.Server` dengan graceful shutdown (SIGTERM, drain 15s) |
| `internal/config` | Struct config dibaca dari env, dengan validasi *fail-fast*: proses **exit non-zero** kalau ada env wajib yang kosong, bukan jalan dengan default diam-diam |
| `internal/platform/logger` | `slog` JSON handler; helper `logger.FromContext(ctx)` yang otomatis membawa `request_id` |
| `internal/platform/postgres` | `pgxpool` dengan `MaxConns`, `HealthCheckPeriod`; `Ping(ctx)` untuk `/readyz` |
| `internal/httpx` | Router, middleware stack, response envelope, error mapping |
| `internal/httpx/middleware` | RequestID, RealIP, Recoverer, StructuredLogger, CORS, Timeout, BodyLimit, RateLimit (dasar) |
| `internal/storage` | Interface `Store { Put(ctx, key, r, contentType) (string, error); Get; Delete; SignedURL }` + implementasi `LocalStore` |
| `internal/inference` | Client HTTP ke `faceclock-inference` — di Fase 0 hanya `Health(ctx)`; diselesaikan di Fase 2 |
| `internal/version` | `var Version, Commit, BuildTime string` diisi lewat `-ldflags` |

**Middleware order (ditetapkan sekarang, tidak boleh diacak fase lain):**

```
RequestID → RealIP → StructuredLogger → Recoverer → CORS
  → Timeout(30s) → BodyLimit(10MB) → RateLimit
    → [Fase 1] Authenticate → [Fase 1] RequirePermission(...)
      → handler
```

Alasan urutan: `RequestID` harus paling luar supaya semua log punya korelasi;
`Recoverer` harus **di dalam** logger supaya panic tetap tercatat; `Authenticate`
harus di dalam `Timeout` supaya query DB untuk auth ikut terpotong deadline.

**Response envelope (dikunci di Fase 0, dipakai semua fase):**

Sukses:
```json
{
  "data": { }
}
```

Sukses berpaginasi:
```json
{
  "data": [ ],
  "meta": { "page": 1, "per_page": 20, "total": 137, "total_pages": 7 }
}
```

Error:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request tidak valid",
    "details": [
      { "field": "email", "message": "format email tidak valid" }
    ],
    "request_id": "01JD8Z0X9K7Q..."
  }
}
```

Aturan: `message` boleh dibaca manusia, `code` yang dipakai program.
`details` hanya diisi untuk error validasi. `request_id` **selalu** ada di error
supaya user bisa melaporkan dan kita bisa mencarinya di log.

**Katalog `code` awal** (fase lain menambah, tidak mengubah yang ada):

| HTTP | code | Kapan |
|---|---|---|
| 400 | `BAD_REQUEST` | Body tidak bisa di-parse |
| 401 | `UNAUTHENTICATED` | Token tidak ada / tidak valid / expired |
| 403 | `FORBIDDEN` | Terautentikasi tapi permission kurang |
| 404 | `NOT_FOUND` | Resource tidak ada / tidak boleh dilihat |
| 409 | `CONFLICT` | Pelanggaran unique / state tidak sesuai |
| 413 | `PAYLOAD_TOO_LARGE` | Body / file melebihi batas |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | Content-Type salah |
| 422 | `VALIDATION_ERROR` | Body ter-parse tapi isinya tidak valid |
| 429 | `RATE_LIMITED` | Terlalu banyak request |
| 500 | `INTERNAL_ERROR` | Bug — detail tidak pernah dibocorkan ke client |
| 502 | `UPSTREAM_ERROR` | `faceclock-inference` error |
| 503 | `SERVICE_UNAVAILABLE` | Dependency belum siap |
| 504 | `UPSTREAM_TIMEOUT` | `faceclock-inference` timeout |

**Aturan logging (implementasi konkret dari master plan § 9 "data biometrik"):**

Ditegakkan sejak Fase 0, karena kalau baru diingat di Fase 4 sudah telanjur
tersebar. Yang **tidak boleh pernah** masuk log, di service mana pun:
- byte gambar / base64 gambar / nama file upload asli;
- nilai embedding (utuh maupun potongan);
- password, password hash, access token, refresh token, isi header `Authorization`;
- `note` absensi (bisa berisi info pribadi).

Yang **boleh** dan memang harus dicatat: `request_id`, `user_id`, `employee_id`,
method, path, status, durasi, ukuran byte (angka saja), `hints`, `usable`,
`similarity` (angka), `model_version`.

Implementasi: satu helper `logger.Redact()` + unit test yang gagal kalau field
terlarang muncul di output handler log.

---

### 2.7 Scaffolding `faceclock-web` (React)

| Komponen | Isi |
|---|---|
| Build | Vite 5 + React 18 + TypeScript strict (`"strict": true`, `noUncheckedIndexedAccess`) |
| Styling | TailwindCSS |
| Routing | React Router v6, dengan struktur route sudah disiapkan untuk guard permission (Fase 5) |
| Data fetching | TanStack Query v5 |
| HTTP client | `src/lib/api.ts` — wrapper `fetch` yang: membaca `VITE_API_BASE_URL`, memasang `Authorization` (Fase 1), meng-unwrap `{data}`, dan melempar `ApiError` bertipe dari `{error}` |
| Env | `.env.example` dengan `VITE_API_BASE_URL=http://localhost:8080` |
| Quality | ESLint (`typescript-eslint`) + Prettier + `tsc --noEmit` di CI |
| Halaman Fase 0 | Satu halaman `/` yang memanggil `GET /readyz` dan menampilkan status — pembuktian bahwa proxy & CORS jalan |

Dev proxy: `vite.config.ts` mem-proxy `/api` → `http://faceclock-api:8080` di dalam
compose, dan `http://localhost:8080` di luar compose.

---

### 2.8 `faceclock-inference` — stub Fase 0

Bukan implementasi sungguhan (itu Fase 2). Fase 0 hanya perlu container yang:
- `GET /health` → `200 {"status":"ok","model_version":"stub","stub":true}`
- `POST /v1/embed` → `501 {"error":{"code":"NOT_IMPLEMENTED",...}}`

> Catatan kesinambungan: di Fase 0, `/readyz` milik Go memeriksa inference lewat
> `GET /health`. [Fase 2 § 4.2](03-Fase2.md#42-get-ready--readiness--introspeksi-konfigurasi)
> menambahkan `GET /ready` (readiness sungguhan + introspeksi konfigurasi), dan
> pemeriksaan di `/readyz` **dipindah ke sana**. Path `/v1/...` sudah dipakai sejak
> stub supaya perpindahan itu tidak mengubah kontrak.

Tujuannya satu: `/readyz` di `faceclock-api` bisa memverifikasi konektivitas dan
wiring compose sekarang, tanpa menunggu model 330 MB. FastAPI + uvicorn saja,
tanpa `insightface`, tanpa `onnxruntime` — image-nya kecil dan build-nya detik.

Folder `services/faceclock-inference/` dibuat di Fase 0 dengan isi stub ini, lalu
**diganti isinya** di Fase 2.

---

### 2.9 Database & Docker Compose

**Postgres:** image `pgvector/pgvector:pg17` (Postgres resmi + ekstensi `vector`
sudah terpasang). Menggunakan `postgres:17` polos berarti harus compile pgvector
sendiri — tidak ada alasan untuk itu.

Init: volume named `faceclock-pgdata`, `POSTGRES_INITDB_ARGS="--data-checksums"`,
timezone container UTC.

**Compose services:**

| Service | Image / build | Port (dev) | Depends on |
|---|---|---|---|
| `postgres` | `pgvector/pgvector:pg17` | `5433:5432` | — |
| `faceclock-api` | build `apps/faceclock-api` | `8080:8080` | postgres (healthy) |
| `faceclock-web` | build `apps/faceclock-web` | `5173:5173` | — |
| `faceclock-inference` | build `services/faceclock-inference` | *tidak dipublish* | — |

Catatan penting:
- Port host Postgres `5433`, bukan `5432` — supaya tidak bentrok dengan Postgres
  lokal yang sudah terpasang di mesin dev.
- `faceclock-inference` **tidak** mem-publish port ke host. Service ini menerima
  gambar wajah mentah; satu-satunya yang boleh menghubunginya adalah
  `faceclock-api` lewat network internal compose. Ini mengikuti pola yang sudah
  terbukti di `face-engine/docker-compose.face-engine.yml`.
- `depends_on: condition: service_healthy` untuk postgres — tanpa ini `faceclock-api`
  crash-loop saat cold start.
- Semua secret lewat `deploy/.env` (git-ignored); `deploy/.env.example` di-commit
  berisi nama variabel + komentar, **tanpa nilai rahasia**.

**Variabel environment (`deploy/.env.example`):**

```
# ── Postgres ──────────────────────────────────────────────
POSTGRES_USER=faceclock
POSTGRES_PASSWORD=              # WAJIB diisi
POSTGRES_DB=faceclock
POSTGRES_PORT=5433

# ── faceclock-api ─────────────────────────────────────────
APP_ENV=development             # development | staging | production
APP_PORT=8080
APP_LOG_LEVEL=debug             # debug | info | warn | error
DATABASE_URL=postgres://faceclock:${POSTGRES_PASSWORD}@postgres:5432/faceclock?sslmode=disable
DATABASE_MAX_CONNS=10
CORS_ALLOWED_ORIGINS=http://localhost:5173
REQUEST_TIMEOUT_SECONDS=30
MAX_BODY_BYTES=10485760

# Storage foto (Fase 3/4)
STORAGE_DRIVER=local            # local | s3
STORAGE_LOCAL_PATH=/data/uploads
STORAGE_PUBLIC_BASE_URL=http://localhost:8080/files

# Inference (dipakai penuh mulai Fase 2)
INFERENCE_BASE_URL=http://faceclock-inference:8000
INFERENCE_TOKEN=                # WAJIB diisi mulai Fase 2
INFERENCE_TIMEOUT_MS=6000

# JWT (dipakai mulai Fase 1) — biarkan kosong di Fase 0
JWT_SECRET=
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# ── faceclock-inference ───────────────────────────────────
FACE_ENGINE_TOKEN=              # sama dengan INFERENCE_TOKEN

# ── faceclock-web ─────────────────────────────────────────
VITE_API_BASE_URL=http://localhost:8080
```

Aturan config: `internal/config` memvalidasi saat startup. Kalau `APP_ENV != development`
dan `JWT_SECRET` kosong (mulai Fase 1) atau `INFERENCE_TOKEN` kosong (mulai Fase 2)
→ **exit(1) dengan pesan jelas**. Service yang jalan tanpa auth secara diam-diam
adalah kegagalan yang baru ketahuan setelah bocor.

---

## 3. Skema Database

Fase 0 **tidak membuat tabel bisnis**. Hanya satu migration:

**`000001_enable_extensions.up.sql`**
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";     -- email case-insensitive (Fase 1)
CREATE EXTENSION IF NOT EXISTS "vector";     -- embedding 512-d (Fase 3)
```

**`000001_enable_extensions.down.sql`**
```sql
DROP EXTENSION IF EXISTS "vector";
DROP EXTENSION IF EXISTS "citext";
DROP EXTENSION IF EXISTS "pgcrypto";
```

**Konvensi skema yang dikunci sekarang** (ditegakkan di Fase 1 dan seterusnya):

| Aturan | Nilai |
|---|---|
| Primary key | `uuid` dengan `DEFAULT gen_random_uuid()` — bukan serial, supaya id tidak membocorkan jumlah karyawan dan aman kalau nanti ada sinkronisasi mobile |
| Nama tabel | jamak, `snake_case` (`face_references`) |
| Kolom waktu | `timestamptz`, **selalu**. Tidak pernah `timestamp` polos |
| Default waktu | `now()` di sisi DB, bukan dari aplikasi — implementasi konkret master plan § 9 "timestamp server-side" |
| Audit kolom | `created_at timestamptz NOT NULL DEFAULT now()`, `updated_at timestamptz NOT NULL DEFAULT now()` di semua tabel bisnis |
| Soft delete | `deleted_at timestamptz NULL` untuk `users`, `employees`, `roles`. Tabel append-only seperti `attendances` **tidak** punya soft delete |
| Enum | Kolom `text` + `CHECK (col IN (...))`, bukan tipe `ENUM` Postgres — menambah nilai ke tipe ENUM di migration itu menyakitkan |
| FK | Eksplisit `ON DELETE` di setiap FK. Default `RESTRICT`; `CASCADE` hanya untuk pivot |
| Koordinat | `lat` / `lng` sebagai `double precision` |

Ini ditulis di sini, bukan di Fase 1, supaya Fase 2–7 punya rujukan yang sama.

---

## 4. Daftar Endpoint

Fase 0 hanya punya 3 endpoint, semuanya publik (belum ada auth).

### `GET /healthz` — liveness

Tidak menyentuh dependency apa pun. Dipakai Docker/orchestrator untuk memutuskan
apakah proses perlu di-restart.

```
200 OK
{ "status": "ok" }
```

### `GET /readyz` — readiness

Memeriksa dependency. Dipakai untuk memutuskan apakah traffic boleh masuk.

```
200 OK
{
  "data": {
    "status": "ok",
    "checks": {
      "database":  { "status": "ok", "latency_ms": 3 },
      "inference": { "status": "ok", "latency_ms": 12 }
    }
  }
}
```

```
503 Service Unavailable
{
  "data": {
    "status": "degraded",
    "checks": {
      "database":  { "status": "ok",   "latency_ms": 2 },
      "inference": { "status": "fail", "error": "connection refused" }
    }
  }
}
```

Aturan: setiap check punya timeout sendiri **2 detik**; total `/readyz` tidak
boleh lebih dari 5 detik. Health check yang bisa menggantung adalah cara populer
untuk membuat load balancer menganggap seluruh armada mati.

### `GET /api/v1/version`

```
200 OK
{ "data": { "version": "0.1.0", "commit": "a1b2c3d", "build_time": "2026-09-04T10:00:00Z" } }
```

`commit` dan `build_time` diisi lewat `-ldflags` saat build, bukan hardcode.

---

## 5. Flow

### 5.1 Cold start `docker compose up`

```
1. postgres start
   └─ initdb (pertama kali saja) → healthcheck `pg_isready` mulai lulus
2. faceclock-inference start (stub) → /health 200
3. faceclock-api start (menunggu postgres healthy)
   ├─ config.Load()  → gagal ⇒ exit(1) dengan daftar env yang kurang
   ├─ logger.New()
   ├─ postgres.Connect() → retry 5×, backoff 1s/2s/4s/8s/16s → gagal ⇒ exit(1)
   ├─ migrate.Up()  [hanya bila APP_ENV=development]
   ├─ router.Build()
   └─ http.Server.ListenAndServe() → log "listening on :8080"
4. faceclock-web start → vite dev server :5173
```

### 5.2 Siklus satu request (kerangka yang dipakai semua fase)

```
Request masuk
 → RequestID: generate ULID, taruh di ctx + header response X-Request-Id
 → RealIP: baca X-Forwarded-For bila di belakang proxy
 → StructuredLogger: catat start
 → Recoverer: pasang defer recover → panic jadi 500 INTERNAL_ERROR
              (stack ke log, tidak pernah ke client)
 → CORS: preflight dijawab di sini
 → Timeout(30s): pasang ctx deadline
 → BodyLimit(10MB): bungkus r.Body dengan http.MaxBytesReader
 → RateLimit
 → [Fase 1+] Authenticate → RequirePermission
 → handler
 → response ditulis lewat httpx.OK / httpx.Fail (tidak pernah json.Encode langsung)
 → StructuredLogger: catat selesai {status, duration_ms, bytes}
```

### 5.3 Graceful shutdown

```
SIGTERM/SIGINT diterima
 → server.Shutdown(ctx deadline 15s)   # stop terima koneksi baru, tuntaskan yang jalan
 → pgxpool.Close()
 → log "shutdown complete"
 → exit(0)
```

Kalau 15 detik lewat dan masih ada request berjalan → log warning, exit(1).
Penting mulai sekarang karena Fase 4 akan punya request yang memanggil inference
service dan tidak boleh terputus di tengah penulisan record absensi.

---

## 6. Edge Case & Validasi

| # | Kondisi | Penanganan yang diharapkan |
|---|---|---|
| E1 | Env wajib kosong | Exit(1) saat startup dengan pesan menyebut **semua** nama env yang kurang sekaligus, bukan satu per satu |
| E2 | Postgres belum siap saat API start | Retry koneksi 5× dengan exponential backoff; baru menyerah setelah itu |
| E3 | Postgres mati saat runtime | `/readyz` → 503; request yang butuh DB → 503 `SERVICE_UNAVAILABLE`; `/healthz` tetap 200 (proses hidup, tidak perlu di-restart) |
| E4 | `faceclock-inference` mati | `/readyz` → 503 dengan `checks.inference.status = "fail"`. **Tidak** mematikan API — endpoint yang tidak butuh inference tetap jalan |
| E5 | Ekstensi `vector` tidak ada di image Postgres | Migration 000001 gagal → API exit(1) dengan pesan eksplisit menyuruh cek image Postgres |
| E6 | Body request > 10 MB | 413 `PAYLOAD_TOO_LARGE`, koneksi tidak digantung |
| E7 | JSON body rusak | 400 `BAD_REQUEST`, pesan generik — jangan pantulkan isi body ke response |
| E8 | Panic di handler | 500 `INTERNAL_ERROR`; stack trace hanya ke log; `request_id` diberikan ke client |
| E9 | Origin tidak diizinkan | Preflight ditolak. `CORS_ALLOWED_ORIGINS` **tidak boleh** `*` saat `APP_ENV != development` — validasi ini di config, bukan di review |
| E10 | Port host bentrok (5433/8080/5173 sudah dipakai) | Semua port host dibaca dari env, bisa diganti tanpa edit compose |
| E11 | Clock container ≠ clock host | Semua container `TZ=UTC`. Didokumentasikan supaya tidak dianggap bug di Fase 4 |
| E12 | Migration setengah jalan lalu gagal | `golang-migrate` menandai state `dirty`; `make db-status` menampilkannya, `make db-force VERSION=n` untuk memulihkan. Prosedur ditulis di README, bukan diingat-ingat |

---

## 7. Struktur Folder

```
03 FaceClock/                                  # root monorepo
├── .github/
│   └── workflows/
│       ├── api.yml                            # lint + test Go     (paths: apps/faceclock-api/**)
│       ├── web.yml                            # lint + tsc + build (paths: apps/faceclock-web/**)
│       └── inference.yml                      # ruff + pytest      (paths: services/**)
├── .editorconfig
├── .gitignore
├── Makefile
├── README.md
│
├── Plan/
│   ├── 00MasterPlan.md
│   ├── 01-Fase0.md                            # ← dokumen ini
│   ├── 02-Fase1.md
│   └── 03-Fase2.md
│
├── docs/
│   ├── adr/
│   │   ├── 0001-monorepo.md                   # keputusan D1 + alasan
│   │   ├── 0002-inference-prior-art.md        # catatan folder face-engine/
│   │   └── 0003-api-conventions.md            # envelope, error code, snake_case
│   └── api/
│       └── README.md                          # indeks kontrak API per fase
│
├── apps/
│   ├── faceclock-api/
│   │   ├── cmd/
│   │   │   └── api/
│   │   │       └── main.go
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   │   ├── config.go
│   │   │   │   └── config_test.go
│   │   │   ├── httpx/
│   │   │   │   ├── router.go
│   │   │   │   ├── response.go                # OK / Created / Fail / Paginated
│   │   │   │   ├── errors.go                  # katalog code → HTTP status
│   │   │   │   ├── decode.go                  # decode + validate body
│   │   │   │   └── middleware/
│   │   │   │       ├── requestid.go
│   │   │   │       ├── logger.go
│   │   │   │       ├── recoverer.go
│   │   │   │       ├── cors.go
│   │   │   │       ├── timeout.go
│   │   │   │       ├── bodylimit.go
│   │   │   │       └── ratelimit.go
│   │   │   ├── platform/
│   │   │   │   ├── logger/logger.go
│   │   │   │   ├── postgres/postgres.go
│   │   │   │   └── validator/validator.go
│   │   │   ├── storage/
│   │   │   │   ├── storage.go                 # interface Store
│   │   │   │   └── local.go
│   │   │   ├── inference/
│   │   │   │   └── client.go                  # Fase 0: hanya Health()
│   │   │   ├── health/
│   │   │   │   └── handler.go                 # /healthz, /readyz
│   │   │   └── version/version.go
│   │   ├── migrations/
│   │   │   ├── 000001_enable_extensions.up.sql
│   │   │   └── 000001_enable_extensions.down.sql
│   │   ├── .dockerignore
│   │   ├── .golangci.yml
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   └── faceclock-web/
│       ├── public/
│       ├── src/
│       │   ├── app/
│       │   │   ├── router.tsx
│       │   │   └── providers.tsx              # QueryClientProvider, dll
│       │   ├── components/ui/
│       │   ├── features/                      # diisi Fase 5/6
│       │   ├── hooks/
│       │   ├── lib/
│       │   │   ├── api.ts                     # fetch wrapper + ApiError
│       │   │   └── env.ts
│       │   ├── types/
│       │   │   └── api.ts                     # tipe envelope {data} / {error}
│       │   ├── pages/
│       │   │   └── HealthPage.tsx
│       │   ├── main.tsx
│       │   └── index.css
│       ├── .dockerignore
│       ├── .env.example
│       ├── .eslintrc.cjs
│       ├── .prettierrc
│       ├── Dockerfile
│       ├── index.html
│       ├── package.json
│       ├── tailwind.config.ts
│       ├── tsconfig.json
│       └── vite.config.ts
│
├── services/
│   └── faceclock-inference/                   # Fase 0: STUB. Diisi penuh di Fase 2
│       ├── app/
│       │   ├── __init__.py
│       │   └── main.py
│       ├── .dockerignore
│       ├── .env.example
│       ├── Dockerfile
│       ├── pyproject.toml
│       └── requirements.txt
│
├── deploy/
│   ├── docker-compose.yml
│   ├── docker-compose.override.yml
│   └── .env.example
│
└── face-engine/                               # PRIOR ART — jangan diubah, WAJIB di-gitignore
```

**`.gitignore` minimum (wajib ada sebelum commit pertama):**
```
# Prior art dari project lain — berisi .venv, model 330MB, dataset LFW
/face-engine/

# Secrets
deploy/.env
**/.env
!**/.env.example

# Go
apps/faceclock-api/bin/
apps/faceclock-api/tmp/

# Node
node_modules/
dist/

# Python
__pycache__/
*.pyc
.venv/
.pytest_cache/

# Data lokal
/data/
*.onnx
```

**Target Makefile:**

| Target | Aksi |
|---|---|
| `make up` | `docker compose -f deploy/docker-compose.yml up -d --build` |
| `make down` | stop, volume dipertahankan |
| `make reset` | `down -v` + `up` (drop database) |
| `make logs s=faceclock-api` | tail log satu service |
| `make migrate-up` / `make migrate-down` | jalankan migration |
| `make migrate-new name=create_users` | generate pasangan file migration |
| `make db-status` | versi migration + status dirty |
| `make psql` | shell psql ke container |
| `make lint` | golangci-lint + eslint + ruff |
| `make test` | go test ./... + vitest + pytest |
| `make fmt` | gofmt + prettier + ruff format |

---

## 8. Checklist Task

### 8.0 Prasyarat
- [x] **D1 (struktur repo) — TERKUNCI: monorepo**, dikonfirmasi 2026-09-04 (§ 2.1)
- [ ] **Konfirmasi user untuk D2–D5** (§ 2.0)

### 8.1 Repo & tooling
- [ ] `git init` di root project
- [ ] Tulis `.gitignore` (§ 7) — **sebelum** `git add` pertama; verifikasi `face-engine/` dan `deploy/.env` ter-exclude
- [ ] Pastikan tidak ada file > 10 MB yang ter-stage
- [ ] `.editorconfig` (LF, UTF-8, indent 2 untuk web/yaml, tab untuk Go)
- [ ] `README.md` root: apa ini, prasyarat, cara jalan, cara test, cara reset DB
- [ ] `Makefile` dengan semua target di § 7

### 8.2 Dokumentasi keputusan
- [ ] `docs/adr/0001-monorepo.md` — rekam keputusan D1 yang **sudah terkunci**
      (monorepo, 2026-09-04) + alasan + alternatif multi-repo yang ditolak
- [ ] `docs/adr/0002-inference-prior-art.md` — catat `face-engine/`, apa yang bisa dipakai ulang, lisensinya, dan bahwa porting-nya pekerjaan Fase 2
- [ ] `docs/adr/0003-api-conventions.md` — envelope, katalog error code, `snake_case`, konvensi skema DB (§ 3)

### 8.3 `faceclock-api`
- [ ] `go mod init github.com/<org>/faceclock/apps/faceclock-api`; Go 1.23+
- [ ] Dependency: `chi/v5`, `pgx/v5`, `pgxpool`, `golang-migrate`, `go-playground/validator/v10`, `oklog/ulid/v2`
- [ ] `internal/config` + validasi fail-fast + unit test untuk kasus env kurang
- [ ] `internal/platform/logger` (slog JSON) + helper `FromContext`
- [ ] `internal/platform/postgres` (pool + retry backoff + `Ping`)
- [ ] `internal/httpx/response.go` + `errors.go` (envelope § 2.6)
- [ ] Semua middleware di `internal/httpx/middleware/` dengan urutan terkunci
- [ ] Unit test: log **tidak** memuat field terlarang (§ 2.6)
- [ ] `internal/storage` interface + `LocalStore` + test
- [ ] `internal/inference/client.go` — `Health(ctx)` saja
- [ ] Handler `/healthz`, `/readyz`, `/api/v1/version`
- [ ] `cmd/api/main.go` dengan graceful shutdown
- [ ] Migration `000001_enable_extensions` (up + down), tes `down` benar-benar membalikkan
- [ ] `.golangci.yml`: `errcheck, govet, staticcheck, gosec, revive, ineffassign, bodyclose, sqlclosecheck`
- [ ] `Dockerfile` multi-stage: build `golang:1.23-alpine` → runtime distroless/alpine, **non-root user**, `HEALTHCHECK` ke `/healthz`
- [ ] `.dockerignore`

### 8.4 `faceclock-web`
- [ ] `npm create vite@latest -- --template react-ts`
- [ ] Tailwind + PostCSS
- [ ] React Router + TanStack Query di `src/app/providers.tsx`
- [ ] `src/lib/api.ts` (unwrap `{data}`, lempar `ApiError` dari `{error}`)
- [ ] `src/types/api.ts` (tipe envelope)
- [ ] Halaman `/` menampilkan hasil `GET /readyz`
- [ ] ESLint + Prettier + `tsc --noEmit` lulus
- [ ] `Dockerfile` (dev: node + `vite --host`; prod stage: build → nginx)
- [ ] `.env.example`

### 8.5 `faceclock-inference` (stub)
- [ ] `app/main.py` FastAPI: `GET /health`, `POST /v1/embed` → 501
- [ ] `requirements.txt`: `fastapi`, `uvicorn[standard]` saja
- [ ] `Dockerfile` non-root + `HEALTHCHECK`
- [ ] `.env.example` (`FACE_ENGINE_TOKEN`)
- [ ] `README.md` singkat: "ini stub, diganti di Fase 2"

### 8.6 Compose & database
- [ ] `deploy/docker-compose.yml` — 4 service, network internal, volume `faceclock-pgdata` + `faceclock-uploads`
- [ ] `postgres` pakai `pgvector/pgvector:pg17` + healthcheck `pg_isready`
- [ ] `faceclock-inference` **tanpa** `ports:` (hanya `expose`)
- [ ] `faceclock-api` `depends_on: postgres: condition: service_healthy`
- [ ] `docker-compose.override.yml` untuk dev (bind mount source, hot reload)
- [ ] `deploy/.env.example` lengkap (§ 2.9)
- [ ] Verifikasi `SELECT extversion FROM pg_extension WHERE extname='vector';` mengembalikan baris

### 8.7 CI
- [ ] `.github/workflows/api.yml` — `go build`, `go vet`, `golangci-lint`, `go test -race ./...`
- [ ] `.github/workflows/web.yml` — `npm ci`, `eslint`, `tsc --noEmit`, `vite build`
- [ ] `.github/workflows/inference.yml` — `ruff check`, `pytest`
- [ ] `paths:` filter agar tiap job hanya jalan saat foldernya berubah
- [ ] Satu job `compose-smoke`: `docker compose up -d`, tunggu healthy, `curl /readyz`, `docker compose down -v`

### 8.8 Penutup fase
- [ ] Tulis `DONE-Fase-0.md` sesuai Protokol Handoff master plan § 10.4

---

## 9. Dependencies

**Prasyarat sebelum fase ini:**
- Tidak ada fase yang mendahului.
- **D1 sudah terkunci** (monorepo, 2026-09-04). **Keputusan user untuk D2–D5**
  (§ 2.0) masih menjadi blocker nyata, bukan formalitas.
- Di mesin dev: Docker Desktop / Docker Engine + Compose v2, Go 1.23+, Node 20+,
  `git`. Python tidak wajib (stub jalan di container).

**Yang bergantung pada fase ini:** semuanya.

| Fase | Mengambil apa dari Fase 0 |
|---|---|
| Fase 1 | Router + middleware chain, envelope error, tooling migration, konvensi skema, config loader |
| Fase 2 | Folder `services/faceclock-inference/`, wiring compose, `internal/inference/client.go`, aturan logging (larangan mencatat gambar/embedding) |
| Fase 3 | Ekstensi `vector`, `storage.Store` |
| Fase 4 | Aturan `timestamptz` + `now()` sisi DB, graceful shutdown |
| Fase 5–6 | `faceclock-web` scaffolding, `lib/api.ts`, tipe envelope |

Peta lengkap artefak Fase 0–2 yang dipakai Fase 3–7 ada di
**[03-Fase2.md § 13 — Peta Kesinambungan Fase 0 → 7](03-Fase2.md#13-peta-kesinambungan-fase-0--7)**.

---

## 10. Definition of Done

Fase 0 selesai bila **semua** poin berikut benar:

1. `git clone` → `cp deploy/.env.example deploy/.env` → isi `POSTGRES_PASSWORD` →
   `make up` berhasil di mesin bersih, tanpa langkah manual tambahan.
2. Empat container berstatus `healthy` (`docker compose ps`).
3. `GET /healthz` → 200 `{"status":"ok"}`.
4. `GET /readyz` → 200 dengan `database.status = "ok"` **dan** `inference.status = "ok"`.
5. `GET /api/v1/version` mengembalikan commit hash asli, bukan placeholder.
6. Menghentikan `faceclock-inference` membuat `/readyz` → 503 sedangkan `/healthz` tetap 200.
7. `SELECT extversion FROM pg_extension WHERE extname IN ('vector','citext','pgcrypto');`
   mengembalikan 3 baris.
8. `make migrate-down` lalu `make migrate-up` berhasil tanpa error (down migration terbukti benar).
9. `faceclock-web` di `http://localhost:5173` menampilkan status readiness yang diambil dari API.
10. `make lint` dan `make test` lulus, exit code 0, untuk ketiga bahasa.
11. CI hijau di semua workflow, termasuk `compose-smoke`.
12. `git ls-files` tidak memuat satu pun file dari `face-engine/`, tidak ada file
    `.env` berisi rahasia, dan tidak ada file > 10 MB.
13. Tiga ADR di `docs/adr/` ada dan mencatat keputusan final D1–D5 (D1 sudah
    terkunci sejak 2026-09-04; D2–D5 diisi hasil konfirmasi).
14. `DONE-Fase-0.md` ada.

---

## 11. Cara Test / Verifikasi

### 11.1 Smoke test manual

```bash
cp deploy/.env.example deploy/.env
# isi POSTGRES_PASSWORD
make up
docker compose -f deploy/docker-compose.yml ps      # semua "healthy"

curl -i http://localhost:8080/healthz                # 200
curl -s http://localhost:8080/readyz | jq            # database + inference "ok"
curl -s http://localhost:8080/api/v1/version | jq
```

### 11.2 Uji degradasi dependency

```bash
docker compose -f deploy/docker-compose.yml stop faceclock-inference
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/readyz   # 503
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/healthz  # 200
docker compose -f deploy/docker-compose.yml start faceclock-inference
curl -s http://localhost:8080/readyz | jq '.data.status'                # "ok"
```

```bash
docker compose -f deploy/docker-compose.yml stop postgres
curl -s http://localhost:8080/readyz | jq '.data.checks.database'       # status "fail"
docker compose -f deploy/docker-compose.yml start postgres
```

### 11.3 Uji database & migration

```bash
make psql
faceclock=# SELECT extname, extversion FROM pg_extension ORDER BY extname;
faceclock=# \dt
faceclock=# SHOW timezone;         -- harus UTC
faceclock=# \q

make migrate-down && make migrate-up && make db-status
```

### 11.4 Uji fail-fast config

```bash
docker compose -f deploy/docker-compose.yml run --rm -e DATABASE_URL= faceclock-api
# harus exit non-zero dengan pesan menyebut DATABASE_URL, bukan panic nil pointer
```

### 11.5 Uji envelope error

```bash
curl -i -X POST http://localhost:8080/api/v1/version        # 405, envelope {error}
curl -i http://localhost:8080/api/v1/tidak-ada              # 404 NOT_FOUND + request_id

head -c 11000000 /dev/urandom > /tmp/big.bin
curl -i -X POST --data-binary @/tmp/big.bin http://localhost:8080/api/v1/version  # 413
```

Setiap response error harus punya `error.code`, `error.message`, dan `error.request_id`,
dan `request_id` itu harus bisa ditemukan di `make logs s=faceclock-api`.

### 11.6 Uji aturan logging (otomatis)

Unit test di `internal/httpx/middleware/logger_test.go`: kirim request dengan
header `Authorization: Bearer rahasia123` dan body berisi `"password":"rahasia456"`,
lalu assert output log **tidak** memuat substring `rahasia123` maupun `rahasia456`.
Test ini harus ada sejak Fase 0 karena ia menjaga aturan yang paling mudah
dilanggar tanpa sengaja di fase-fase berikutnya.

### 11.7 Uji kebersihan repo

```bash
git count-objects -vH                      # size-pack wajar (< 5 MB)
git ls-files | grep -c '^face-engine/'     # harus 0
```

---

## 12. Referensi Silang ke "Isu Lintas-Fase" (Master Plan § 9)

| Isu lintas-fase | Bagaimana Fase 0 menyiapkannya |
|---|---|
| Timestamp server-side | Konvensi skema § 3: `timestamptz` + `DEFAULT now()` sisi DB; semua container `TZ=UTC` |
| Geofence server-side | `lat`/`lng` `double precision`; belum ada logika (Fase 4) |
| RBAC di level API | Slot `Authenticate` → `RequirePermission` sudah tercetak di middleware chain § 2.6, jadi Fase 1 mengisi, bukan mendesain ulang |
| Data biometrik / UU PDP | Aturan logging § 2.6 + unit test-nya; `faceclock-inference` tidak mem-publish port; `.gitignore` mencegah foto/model masuk repo |
| Threshold configurable | Tabel `app_settings` dibuat di Fase 1; Fase 0 memastikan tooling migration + seeder ada |
| Fallback wajib `pending_review` | Belum relevan (Fase 4) |
