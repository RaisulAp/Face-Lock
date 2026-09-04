# DONE — Fase 0: Fondasi & Setup

> Ditulis sesuai Protokol Handoff [00MasterPlan.md § 10.4](00MasterPlan.md#10-protokol-handoff-untuk-agent).
> Tidak ada `DONE-Fase-N.md` sebelumnya — diverifikasi eksplisit (ini fase pertama yang dieksekusi; sebelum sesi ini repo belum punya kode maupun git history).

---

## 1. Apa yang dibangun

### `apps/faceclock-api` (Go 1.25, module `github.com/faceclock/faceclock/apps/faceclock-api`)

- **`cmd/api/main.go`** — `config.Load()` (fail-fast) → logger → `postgres.Connect()` (retry 1s/2s/4s/8s/16s) → `runMigrations()` (dev only, via `golang-migrate` + `pgx/v5` stdlib adapter) → `storage.NewLocalStore()` → `inference.New()` → router → `http.Server` dengan graceful shutdown (SIGINT/SIGTERM, drain 15s).
- **`internal/config`** — struct config dari env, validasi *fail-fast* yang melaporkan **semua** variabel yang kurang/tidak valid sekaligus (bukan satu per satu), termasuk aturan maju untuk `JWT_SECRET`/`INFERENCE_TOKEN` (wajib mulai Fase 1/2 di luar `development`) dan larangan `CORS_ALLOWED_ORIGINS=*` di luar `development`.
- **`internal/platform/logger`** — `slog` JSON handler dengan `ReplaceAttr` yang me-redact key apa pun yang mengandung `password`, `token`, `authorization`, `embedding`, `image`, `photo`, `base64`, atau `note` (case-insensitive substring match, sengaja luas). `FromContext(ctx)` menambahkan `request_id` otomatis.
- **`internal/platform/postgres`** — `pgxpool` dengan retry/backoff terkunci, `Ping()` dengan budget 2 detik untuk `/readyz`.
- **`internal/httpx`** — `response.go` (`OK`/`Created`/`NoContent`/`Paginated`/`Status`/`Fail`/`FailWithStatus`), `errors.go` (13 kode katalog + `StatusFor()` yang **panic** kalau kode belum terdaftar — supaya kode baru yang lupa didaftarkan ketahuan saat compile/test, bukan jadi 500 diam-diam), `decode.go` (decode+validate body, membedakan 413/400/422), `router.go` (chi + middleware chain terkunci + slot `/api/v1` untuk Fase 1).
- **`internal/httpx/middleware`** — `requestid.go` (ULID, monotonic), `logger.go` (StructuredLogger — hanya method/path/status/duration/bytes, tidak pernah membaca body/header), `recoverer.go`, `cors.go`, `timeout.go`, `bodylimit.go`, `ratelimit.go` (fixed-window per-IP, dasar).
- **`internal/storage`** — interface `Store` (`Put`/`Get`/`Delete`/`SignedURL`) + `LocalStore`. `SignedURL` sengaja mengembalikan error (lihat § 2 "Deviasi" — REV-INF-04).
- **`internal/inference`** — `Client.Health(ctx)` saja, sesuai batas Fase 0.
- **`internal/health`** — handler `/healthz`, `/readyz` (cek DB + inference paralel secara berurutan, masing-masing timeout 2 detik).
- **`internal/version`** — `Version`/`Commit`/`BuildTime` (diisi `-ldflags`) + `MinSupportedClient` (REV-EP-10) + handler `/api/v1/version`.
- **`migrations/000001_enable_extensions`** — `pgcrypto`, `citext`, `vector`, dengan down yang benar-benar membalikkan.
- **`.golangci.yml`**, **`Dockerfile`** (multi-stage, `golang:1.25-alpine` → `alpine:3.20`, non-root, `HEALTHCHECK`), **`.dockerignore`**.

### `apps/faceclock-web` (Vite + React + TypeScript, di-scaffold via `npm create vite@latest -- --template react-ts`)

- **`src/types/api.ts`** — tipe envelope `{data}`/`{error}`, union `ApiErrorCode` (13 kode).
- **`src/lib/api.ts`** — wrapper `fetch`: baca `VITE_API_BASE_URL`, seam `getAuthToken()` untuk Fase 1, unwrap `{data}`, lempar `ApiError` (kelas, bukan interface) dari `{error}`.
- **`src/app/providers.tsx`** — `QueryClientProvider` (TanStack Query v5). **`src/app/router.tsx`** — React Router v6, satu route `/`.
- **`src/pages/HealthPage.tsx`** — memanggil `GET /readyz` lewat `useQuery`, menampilkan status database + inference.
- Tailwind **v4** (lewat `@tailwindcss/vite`, bukan PostCSS — lihat § 2 "Deviasi").
- ESLint (flat config, `typescript-eslint` + `react-hooks` + `react-refresh`) + Prettier + `tsc --noEmit` (strict + `noUncheckedIndexedAccess`).
- **`Dockerfile`** tiga stage: `dev` (node + `vite --host`, dipakai compose), `build`, `prod` (nginx, tersedia untuk deployment nanti).
- Dependency Fase 5/6/7 dipasang sejak sekarang sesuai **REV-INF-05/06**: `react-hook-form`, `zod`, `leaflet`, `dompurify`, `ulid`, `msw`, `@playwright/test` (belum dipakai — tidak ada fitur yang membutuhkannya di Fase 0).

### `services/faceclock-inference` (stub FastAPI)

- `app/main.py` — `GET /health` (`200 {"status":"ok","model_version":"stub","stub":true}`), `POST /v1/embed` (`501 NOT_IMPLEMENTED`). `docs_url`/`redoc_url`/`openapi_url` dimatikan.
- `requirements.txt` — hanya `fastapi` + `uvicorn[standard]`. `requirements-dev.txt` untuk `pytest`/`httpx`/`ruff`.
- `Dockerfile` — `python:3.11-slim`, non-root, `HEALTHCHECK`.
- 2 test (`tests/test_health.py`).

### `deploy/`

- **`docker-compose.yml`** — 4 service (`postgres` `pgvector/pgvector:pg17`, `faceclock-inference` tanpa `ports:`, `faceclock-api`, `faceclock-web` target `dev`), network internal, volume `faceclock-pgdata` + `faceclock-uploads`, healthcheck di keempat service.
- **`docker-compose.override.yml`** — bind-mount `faceclock-web/src` untuk hot reload (dipakai manual, lihat § 2).
- **`.env.example`** — lengkap sesuai § 2.9, ditambah blok `STORAGE_S3_*` **dikomentari** (REV-INF-03, lihat § 7 "Celah").

### `docs/`

- `docs/adr/0001-monorepo.md`, `0002-inference-prior-art.md`, `0003-api-conventions.md` (termasuk konfirmasi D2–D5).
- `docs/api/README.md` + kerangka kanonik `hints.json` (REV-CTR-01/02) dan `geo-testcases.json` (REV-CTR-03), keduanya kosong-terstruktur, diisi Fase 2/4/7.

### Root

- `.gitignore` (ditulis **sebelum** `git init`), `.gitattributes` (paksa LF), `.editorconfig`, `Makefile`, `README.md`, `.github/workflows/{api,web,inference,compose-smoke}.yml`.

---

## 2. Keputusan teknis & deviasi dari plan

| # | Keputusan/Deviasi | Alasan |
|---|---|---|
| 1 | **D2–D5 dikonfirmasi user langsung di prompt eksekusi** ("D2-D5 ... DIKONFIRMASI dipakai sebagai default") — dicatat resmi di [`docs/adr/0003-api-conventions.md`](../docs/adr/0003-api-conventions.md#konfirmasi-d2d5) karena `Plan/01-Fase0.md` sendiri tidak diubah (larangan eksplisit di prompt) | DoD § 10 poin 13 mensyaratkan D2–D5 "diisi hasil konfirmasi" di ADR |
| 2 | **`sqlc` (bagian dari D5) belum di-setup** — hanya `pgx/v5` + `pgxpool` | Tidak ada satu query bisnis pun di Fase 0 untuk digenerate (migration 000001 cuma `CREATE EXTENSION`). Masuk Fase 1 bersama skema pertama |
| 3 | **Vite 8 + React 19 + `oxlint`→ESLint**, bukan Vite 5 + React 18 seperti tertulis di D6 | `npm create vite@latest` hari ini menginstal versi current-stable, bukan versi yang ditulis di dokumen (ditulis saat Vite 5 masih current). ESLint + `typescript-eslint` + Prettier dipasang eksplisit menggantikan `oxlint` bawaan scaffold supaya sesuai D7 |
| 4 | **Tailwind v4 lewat `@tailwindcss/vite`**, bukan v3 + PostCSS seperti tersirat oleh `tailwind.config.ts` di struktur folder dokumen | `npm install tailwindcss` hari ini memasang v4, yang arsitekturnya berbeda (tanpa config file wajib, integrasi lewat plugin Vite). Fungsional setara untuk kebutuhan Fase 0 |
| 5 | **`golang-migrate` CLI tidak dibundel di image `faceclock-api`** — `make migrate-up/down/db-status` menjalankan `go run -tags 'pgx5' .../cmd/migrate` **dari host**, terhubung ke port Postgres yang dipublish, bukan `docker compose exec` ke dalam container | Image runtime sengaja minimal (§ 2.6: hanya binary + migrations). `faceclock-api` sendiri tetap migrate otomatis saat startup dev — target Makefile ini untuk penggunaan manual/CI |
| 6 | **`POSTGRES_PORT` di `deploy/.env` = 5434, bukan 5433** (default dokumen) | Port 5433 di mesin dev sudah dipakai container project lain (`simrs-pg-dev`) yang **tidak boleh diganggu**. Ini justru membuktikan E10 (semua port host dibaca dari env) bekerja — tidak ada yang di-hardcode di compose |
| 7 | **405 Method Not Allowed dijawab lewat `FailWithStatus`**, bukan lewat katalog `StatusFor` biasa | Tidak ada kode 405 di 13 kode dasar Fase 0 (dicatat eksplisit di ADR 0003). Envelope tetap `{error:{code:"BAD_REQUEST",...}}` tapi status HTTP asli 405, sesuai § 11.5 |
| 8 | **`faceclock-web` di compose base memakai target `dev`** (bukan `prod`) — `docker-compose.override.yml` (dev bind-mount) **tidak** otomatis ter-apply oleh `make up`, karena Makefile-nya sendiri pin `-f deploy/docker-compose.yml` tanpa override | Base compose sudah cukup untuk memuaskan DoD (menyajikan halaman `/`); override untuk kenyamanan hot-reload harus disertakan manual dengan `-f` kedua. Dicatat eksplisit di komentar `docker-compose.override.yml` sendiri supaya tidak dianggap otomatis aktif |
| 9 | **Test E6 (413) tidak dijalankan secara literal via `curl -X POST /api/v1/version`** seperti contoh di § 11.5 | `/api/v1/version` sengaja `GET`-only (§ 4) — POST ke situ menghasilkan `405` dari router sebelum body pernah dibaca, bukan `413`. Ditulis ulang sebagai test Go (`TestBodyLimit_TriggersPayloadTooLarge`) yang mengeksekusi `BodyLimit`+`DecodeAndValidate` langsung — jalur kode yang sama yang akan dipakai endpoint `POST` pertama di Fase 1. Dilaporkan di sini karena ini kemungkinan inkonsistensi kecil di `01-Fase0.md § 11.5` sendiri (dua skenario, 405 dan 413, digabung ke satu endpoint yang hanya bisa menghasilkan salah satunya) |

---

## 3. Bukti verifikasi

Semua dijalankan sungguhan pada sesi ini (Windows, Docker Desktop, Go 1.25.5,
Node 22.20.0, Python 3.14.3), dari kondisi **bersih**
(`docker compose down -v` lalu `up -d --build`).

### 3.1 `docker compose up` dari kondisi bersih

```
$ docker compose -f deploy/docker-compose.yml down -v
 Container faceclock-faceclock-api-1 Removed
 Container faceclock-postgres-1 Removed
 Container faceclock-faceclock-inference-1 Removed
 Volume faceclock_faceclock-pgdata Removed
 Volume faceclock_faceclock-uploads Removed
 Network faceclock_faceclock-internal Removed

$ docker compose -f deploy/docker-compose.yml up -d --build
 ...
 Container faceclock-postgres-1 Healthy
 Container faceclock-faceclock-api-1 Started
 Container faceclock-faceclock-web-1 Started
real  0m10.941s   (build cache warm)

$ docker compose -f deploy/docker-compose.yml ps
NAME                              STATUS
faceclock-faceclock-api-1         Up (healthy)
faceclock-faceclock-inference-1   Up (healthy)
faceclock-faceclock-web-1         Up (healthy)
faceclock-postgres-1              Up (healthy)
```

**Empat dari empat container `healthy`.**

### 3.2 Tiga endpoint

```
$ curl -s -i http://localhost:8080/healthz
HTTP/1.1 200 OK
{"data":{"status":"ok"}}

$ curl -s http://localhost:8080/readyz
{"data":{"checks":{"database":{"latency_ms":0,"status":"ok"},"inference":{"latency_ms":3,"status":"ok"}},"status":"ok"}}

$ curl -s http://localhost:8080/api/v1/version
{"data":{"build_time":"2026-09-04T17:56:27Z","commit":"522cbfe","min_supported_client":{"mobile":"0.0.0","web":"0.0.0"},"version":"0.1.0"}}
```

> **Catatan commit hash:** `522cbfe` di atas adalah commit pertama sesi ini
> (`feat: Fase 0 — fondasi monorepo ...`). Sesi ini menambah dua commit lagi
> setelahnya (fix lint/format, lalu commit ini sendiri). Jalankan
> `docker compose -f deploy/docker-compose.yml up -d --build faceclock-api`
> setelah `git log -1` untuk membakar hash commit **final** ke dalam image —
> langkah yang sama seperti yang dilakukan tiga kali di sesi ini setiap kali
> kode berubah setelah commit.

### 3.3 Degradasi dependency (DoD 6, § 11.2)

```
$ docker compose stop faceclock-inference
$ curl -s http://localhost:8080/readyz   # 503
{"data":{"checks":{"database":{"status":"ok",...},
  "inference":{"status":"fail","error":"...context deadline exceeded"}},"status":"degraded"}}
$ curl -o /dev/null -w "%{http_code}" http://localhost:8080/healthz   # 200
200
$ docker compose start faceclock-inference
$ curl -s http://localhost:8080/readyz   # kembali "ok" setelah restart
{"data":{...,"status":"ok"}}
```

Diuji juga untuk `postgres` stop/start — hasil identik (readyz 503 dengan
`checks.database.status="fail"`, lalu pulih ke `"ok"`).

### 3.4 Ekstensi Postgres, timezone, migration table

```
$ psql -c "SELECT extname, extversion FROM pg_extension ORDER BY extname;"
 citext   | 1.6
 pgcrypto | 1.3
 plpgsql  | 1.0
 vector   | 0.8.6
(4 rows)

$ psql -c "SHOW timezone;"     → UTC
$ psql -c "SELECT * FROM schema_migrations;"   → version=1, dirty=f
```

### 3.5 `migrate down` lalu `migrate up` (DoD 8)

```
$ go run -tags 'pgx5' .../cmd/migrate -path migrations -database "pgx5://...localhost:5434.../faceclock?sslmode=disable" down -all
1/d enable_extensions (34.98ms)
$ ... up
1/u enable_extensions (50.94ms)
$ ... version
1
```

Extension terverifikasi masih ada (`citext`, `pgcrypto`, `vector`) setelah
siklus down/up.

### 3.6 Envelope error (405, 404, request_id di log)

```
$ curl -i -X POST http://localhost:8080/api/v1/version
HTTP/1.1 405 Method Not Allowed
{"error":{"code":"BAD_REQUEST","message":"Method tidak diizinkan untuk endpoint ini","request_id":"01M1PRPHK6BZDMZY76J1K6F0MX"}}

$ curl -i http://localhost:8080/api/v1/tidak-ada
HTTP/1.1 404 Not Found
{"error":{"code":"NOT_FOUND","message":"Endpoint tidak ditemukan","request_id":"01M1PRPHMNN4RJV5Z46WEKH9MA"}}

$ docker compose logs faceclock-api | grep 01M1PRPHMNN4RJV5Z46WEKH9MA
{"msg":"request started","request_id":"01M1PRPHMNN4RJV5Z46WEKH9MA",...}
{"msg":"request completed","request_id":"01M1PRPHMNN4RJV5Z46WEKH9MA","status":404,...}
```

`request_id` di response **ditemukan** di log — DoD § 11.5 syarat terakhir.

413 (E6) diuji lewat `TestBodyLimit_TriggersPayloadTooLarge` (Go, § 2 poin 9),
bukan curl literal — lihat penjelasan deviasi di atas.

### 3.7 Fail-fast config (DoD § 11.4)

```
$ docker compose run --rm -e DATABASE_URL= faceclock-api
faceclock-api: fatal config error: config: invalid or missing environment variables: DATABASE_URL
exit code: 1
```

### 3.8 Unit test anti-biometrik (§ 11.6) — dijalankan, lulus

```
$ go test ./internal/httpx/middleware/... -v
=== RUN   TestStructuredLogger_NeverLeaksSecretsFromRealRequest
--- PASS: TestStructuredLogger_NeverLeaksSecretsFromRealRequest (0.00s)
=== RUN   TestRedact_CatchesForbiddenKeysEvenWhenHandlerMisbehaves
--- PASS: TestRedact_CatchesForbiddenKeysEvenWhenHandlerMisbehaves (0.00s)
=== RUN   TestRedact_AllowsSafeFields
--- PASS: TestRedact_AllowsSafeFields (0.00s)
PASS
```

Test pertama mengirim request sungguhan lewat `StructuredLogger` dengan
header `Authorization: Bearer rahasia123` dan body `{"password":"rahasia456"}`,
lalu meng-assert output log tidak memuat kedua substring — **persis** skenario
yang diminta dokumen. Test kedua membuktikan pertahanan lapis kedua
(`ReplaceAttr` di `internal/platform/logger`) bekerja bahkan kalau seorang
handler di masa depan secara keliru mencoba men-log field terlarang secara
langsung.

### 3.9 Seluruh test suite Go

```
$ go build ./... && go vet ./... && go test ./...
Go build: Success
Go vet: No issues found
Go test: 14 passed in 10 packages
```

14 test: 5 config, 3 middleware (termasuk § 11.6), 3 httpx (413/400/422),
3 storage (put/get/delete, path traversal, `SignedURL` sengaja gagal).

### 3.10 Web: typecheck, lint, format, build

```
$ npx tsc --noEmit          → No errors found
$ npx eslint .              → 0 errors, 1 warning (react-refresh, non-blocking)
$ npx prettier --check .    → All files formatted correctly
$ npm run build              → tsc -b && vite build — sukses, dist/ 318 KB (gzip 100 KB)
```

Diuji juga build stage `prod` (nginx) langsung via `docker build --target prod`
— sukses.

### 3.11 Inference: ruff + pytest (virtualenv lokal)

```
$ pytest -v
tests/test_health.py::test_health_ok PASSED
tests/test_health.py::test_embed_not_implemented PASSED
2 passed

$ ruff check .     → All checks passed!
$ ruff format --check .   → 3 files already formatted
```

### 3.12 Kebersihan repo (§ 11.7)

```
$ git count-objects -vH
size-pack: 375.75 KiB   (jauh di bawah ambang "wajar")

$ git ls-files | grep -c '^face-engine/'
0

$ git ls-files | grep '\.env$'
(kosong — tidak ada .env yang ter-commit)
```

---

## 4. Checklist Definition of Done ([01-Fase0.md § 10](01-Fase0.md#10-definition-of-done))

| # | Poin | Status | Bukti |
|---|---|---|---|
| 1 | `cp .env.example` → isi password → `make up` berhasil bersih | ✅ **Terpenuhi** — tapi lewat `docker compose -f deploy/docker-compose.yml up -d --build` langsung, bukan `make up` (`make` tidak terinstal di mesin dev ini — lihat § 7 celah). Perintah yang persis sama dijalankan manual dan berhasil dari `down -v` | § 3.1, § 3.2 |
| 2 | 4 container `healthy` | ✅ **Terpenuhi** | § 3.1 |
| 3 | `GET /healthz` → 200 | ✅ **Terpenuhi** | § 3.2 |
| 4 | `GET /readyz` → 200, database+inference `ok` | ✅ **Terpenuhi** | § 3.2 |
| 5 | `GET /api/v1/version` mengembalikan commit asli | ✅ **Terpenuhi** | § 3.2 |
| 6 | Stop inference → `/readyz` 503, `/healthz` tetap 200 | ✅ **Terpenuhi** | § 3.3 |
| 7 | 3 ekstensi Postgres | ✅ **Terpenuhi** | § 3.4 |
| 8 | `migrate-down` lalu `migrate-up` tanpa error | ✅ **Terpenuhi** (dijalankan manual dengan invokasi persis yang ditulis di `Makefile`, bukan lewat `make` itu sendiri) | § 3.5 |
| 9 | `faceclock-web` di :5173 menampilkan status readiness | ⚠️ **Terpenuhi secara kontrak, tidak diverifikasi visual di browser** — server merespons 200, CORS terverifikasi benar (§ 3.2 tersirat via header `Access-Control-Allow-Origin`), kontrak JSON `HealthPage.tsx` cocok persis dengan response `/readyz`, tapi sesi ini tidak membuka browser sungguhan untuk screenshot render React. Direkomendasikan diverifikasi manual sebelum menandai 100% |
| 10 | `make lint` dan `make test` lulus ketiga bahasa | ⚠️ **Perintah dasarnya lulus (Go/TS/Python masing-masing diverifikasi langsung — § 3.9–3.11), tapi TIDAK lewat `make` itu sendiri** — `make` tidak terinstal di mesin dev ini. `golangci-lint` juga tidak terinstal lokal (Makefile-nya sudah punya fallback pesan untuk kasus ini, dan CI menjalankannya) | § 3.9, § 3.10, § 3.11, § 7 |
| 11 | CI hijau semua workflow termasuk `compose-smoke` | ❌ **Belum diverifikasi** — empat workflow ditulis (`api.yml`, `web.yml`, `inference.yml`, `compose-smoke.yml`) sesuai spek, tapi repo belum punya remote GitHub dan tidak ada runner CI di sesi ini untuk benar-benar menjalankannya | § 7 |
| 12 | Tidak ada file `face-engine/`, `.env` rahasia, atau file >10MB ter-commit | ✅ **Terpenuhi** | § 3.12 |
| 13 | 3 ADR mencatat D1–D5 final | ✅ **Terpenuhi** — D1 dari sesi sebelumnya, D2–D5 ditambahkan di `docs/adr/0003-api-conventions.md` sesi ini | `docs/adr/` |
| 14 | `DONE-Fase-0.md` ada | ✅ **Terpenuhi** | Dokumen ini |

**Skor: 11 dari 14 sepenuhnya terpenuhi dengan bukti langsung; 2 (poin 1, 10)
terpenuhi secara substansi tapi lewat perintah manual, bukan `make` itu
sendiri karena `make` tidak terinstal; 1 (poin 11) belum bisa diverifikasi
karena tidak ada CI runner di sesi ini.** Tidak ada yang ditandai selesai
tanpa command/output nyata di § 3.

---

## 5. Revisi dari `09-Revisions-Log.md` yang diterapkan di fase ini

Diterapkan **di kode**, dengan bukti konkret (bukan cuma disebut di dokumen
plan):

| ID | Apa yang diterapkan | Bukti |
|---|---|---|
| **REV-MW-01** | Slot `RequireConsent` dicetak (komentar) di `internal/httpx/router.go` setelah posisi `RequirePermission`, belum diimplementasikan | `router.go` § "mountAPIv1" |
| **REV-ERR-04** | Status 410 dicatat di `docs/adr/0003-api-conventions.md`, belum dipakai kode (memang belum ada endpoint foto) | ADR 0003 |
| **REV-CONV-01/02/03** | Tiga pengecualian konvensi dicatat di `docs/adr/0003-api-conventions.md` — belum ada tabelnya (Fase 1/3 yang membuat `audit_logs`/`attendance_attempts`/`face_references`) | ADR 0003 |
| **REV-INF-05/06** | `react-hook-form`, `zod`, `leaflet`, `dompurify`, `ulid`, `msw`, `@playwright/test` terpasang di `package.json` | `apps/faceclock-web/package.json` |
| **REV-EP-10** | `min_supported_client` ada di response `GET /api/v1/version` | § 3.2, `internal/version/version.go` |
| **REV-CTR-01/03** | Kerangka `docs/api/hints.json` (namespace `server_hints`/`client_coach` — juga menutup **REV-CTR-02**) dan `docs/api/geo-testcases.json` dibuat, kosong-terstruktur | `docs/api/` |
| **REV-INF-04** | `Store.SignedURL()` di-implementasikan tapi sengaja mengembalikan error dengan pesan yang merujuk balik ke keputusan D14 | `internal/storage/local.go` + test |

**Belum diterapkan sepenuhnya (gap, lihat § 7):**

| ID | Status |
|---|---|
| **REV-INF-01** (HTTPS untuk `faceclock-web`) | Tidak diimplementasikan (tidak relevan untuk dev `localhost`) — dicatat sebagai prasyarat produksi, bukan blocker Fase 0 |
| **REV-INF-03** (MinIO + `STORAGE_S3_*`) | `STORAGE_S3_*` dicatat **dikomentari** di `deploy/.env.example`; MinIO **tidak** ditambahkan ke `docker-compose.yml` — lihat § 7 |

**Konfirmasi eksplisit sesuai instruksi:** ID-ID di atas **tidak** saya ubah
statusnya di `09-Revisions-Log.md` — perubahan status "belum dieksekusi" →
"dieksekusi" untuk item-item ini adalah prompt terpisah setelah Fase 0–2
selesai semua, supaya konsisten dengan pola konsolidasi sebelumnya.

---

## 6. Cara menjalankan & menguji ulang

```bash
git clone <repo>
cd "03 FaceClock"
cp deploy/.env.example deploy/.env
# edit deploy/.env:
#   - POSTGRES_PASSWORD wajib diisi
#   - bila port 5433 di host sudah dipakai proses lain, ubah POSTGRES_PORT
#   - opsional: COMMIT=$(git rev-parse --short HEAD) supaya /api/v1/version akurat

docker compose -f deploy/docker-compose.yml up -d --build
# atau, bila `make` terinstal:  make up

docker compose -f deploy/docker-compose.yml ps        # keempat harus "healthy"
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/version
open http://localhost:5173   # atau buka manual di browser

# Test
cd apps/faceclock-api && go test ./...
cd ../faceclock-web && npm run typecheck && npm run lint && npm run build
cd ../../services/faceclock-inference && python -m venv .venv && \
  .venv/bin/pip install -r requirements-dev.txt && .venv/bin/pytest
```

## 7. Hal yang masih tertunda / celah yang diketahui

1. **`make` tidak terinstal di mesin dev yang dipakai sesi ini** (Windows,
   tanpa Git Bash `make`/WSL/choco). Setiap target `Makefile` sudah
   diverifikasi dengan menjalankan perintah persis yang ada di dalamnya
   secara manual (§ 3), tapi `make up`/`make lint`/`make test`/`make
   migrate-*` sendiri belum pernah benar-benar dieksekusi. Rekomendasi:
   install `make` (via `choco install make` atau WSL) sebelum Fase 1, atau
   verifikasi di CI (yang berjalan di `ubuntu-latest`, sudah punya `make`).
2. **CI belum pernah benar-benar jalan** — empat workflow ditulis sesuai
   spek tapi repo belum di-push ke GitHub remote apa pun di sesi ini.
   `golangci-lint` dan integrasi `-race` (butuh CGO/compiler C, tidak
   terinstal di mesin dev ini) juga baru akan teruji pertama kali di CI.
3. **REV-INF-03 (MinIO) sengaja tidak dieksekusi** — bergantung pada **D13**
   (Fase 3, keputusan penyimpanan foto referensi) yang belum dikonfirmasi.
   Menambahkan MinIO ke compose sekarang berarti menebak keputusan yang
   belum diambil. `STORAGE_S3_*` sudah dicatat (dikomentari) di
   `deploy/.env.example` supaya Fase 3 tinggal mengaktifkan, bukan
   menemukan dari nol.
4. **REV-INF-01 (HTTPS) tidak relevan di Fase 0** — deployment dev ini
   `localhost` murni. Prasyarat produksinya sudah dicatat di
   `09-Revisions-Log.md § 5` (harus ada sebelum Fase 6 diuji di perangkat
   nyata), tidak diulang implementasinya di sini.
5. **`faceclock-web` di browser sungguhan belum di-screenshot** — hanya
   diverifikasi lewat `curl` (server merespons, CORS header benar) dan
   pembacaan kode (kontrak data `HealthPage.tsx` cocok dengan response
   `/readyz`). Rekomendasi: buka `http://localhost:5173` di browser sebelum
   menganggap DoD poin 9 100% tuntas secara visual.
6. **`sqlc` belum di-setup** (bagian dari D5) — akan masuk di Fase 1 (§ 2
   poin 2).
7. **Docker image belum di-scan** untuk kerentanan (mis. `trivy`/`grype`) —
   tidak diminta dokumen Fase 0, dicatat sebagai kandidat hardening
   produksi, bukan gap Fase 0.

Semua slot yang disiapkan untuk Fase 1 (`internal/httpx/router.go`'s
`mountAPIv1`, middleware chain yang berhenti di `RateLimit`, `sqlc` yang
belum di-setup) **sengaja** dibiarkan sebagai slot kosong, bukan
diimplementasikan sebagian — sesuai batas scope Fase 0 § 1 "Yang TIDAK
dikerjakan di fase ini".
