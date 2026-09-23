# Faceclock

Sistem absensi karyawan berbasis face recognition. Lihat
[`Plan/00MasterPlan.md`](Plan/00MasterPlan.md) untuk gambaran produk lengkap,
dan [`Plan/01-Fase0.md`](Plan/01-Fase0.md) untuk spesifikasi fondasi ini.

**Status:** Fase 0 (Fondasi & Setup) — lihat
[`Plan/DONE-Fase-0.md`](Plan/DONE-Fase-0.md).

## Struktur monorepo

```
apps/faceclock-api/          Go — REST API, RBAC, business logic
apps/faceclock-web/          React — panel admin + halaman absensi web
services/faceclock-inference/ Python/FastAPI — face embedding microservice
deploy/                      docker-compose + env
docs/adr/                    Keputusan arsitektur (ADR)
docs/api/                    Kontrak API lintas-bahasa (kanonik)
Plan/                        Spesifikasi per fase + log revisi kontrak
```

## Prasyarat

- Docker Desktop / Docker Engine + Compose v2
- Go 1.25+ (untuk development di luar container)
- Node 20+
- Python tidak wajib di host — `faceclock-inference` jalan di container

## Menjalankan

Ada dua cara menjalankan stack: **Hybrid Development** (disarankan untuk dev
sehari-hari, lihat [`deploy/HYBRID_DEV.md`](deploy/HYBRID_DEV.md)) dan
**semua-di-Docker**. Untuk target `make` lihat [`Makefile`](Makefile).

### Peta port & folder

| Komponen | Folder / Service | URL default |
|---|---|---|
| Database (Postgres 17 + pgvector) | `postgres` (Docker) | `localhost:5434` |
| Backend (Go REST API) | `apps/backend` | http://localhost:8080 |
| Frontend (Vite + React) | `apps/frontend` | http://localhost:5173 |
| Service AI / face engine | `services/faceclock-inference` (Docker, on-demand) | http://localhost:8000 |

Login super admin (hasil seeder dev): `admin@faceclock.local` / `123456789`.

### ⚠️ Dua hal yang paling sering bikin gagal

1. **Semua perintah dijalankan dari ROOT repo**, bukan dari `apps\...` atau
   `scripts\`, karena script memakai path relatif `.\scripts\...`:

   ```powershell
   cd "D:\WORK\Website\09 September 2026\03 FaceClock"
   Get-Location   # wajib berakhir di ...\03 FaceClock
   ```

2. **Windows: gunakan `powershell` (bukan `pwsh`) jika PowerShell 7 belum
   terpasang.** Windows hanya punya Windows PowerShell 5.1. Panggil script
   dengan `.\` atau `powershell -File`:

   ```powershell
   .\scripts\dev-db.ps1
   # atau, kalau execution policy memblokir:
   powershell -ExecutionPolicy Bypass -File .\scripts\dev-db.ps1
   ```

### Setup sekali di awal

#### Jalur Docker (semua komponen)

```bash
cp deploy/.env.example deploy/.env
# edit deploy/.env — isi POSTGRES_PASSWORD (wajib).
# Jika port 5433 di host sudah dipakai proses lain, ubah POSTGRES_PORT.

make up
docker compose -f deploy/docker-compose.yml ps      # keempat service harus "healthy"

curl http://localhost:8080/healthz
curl http://localhost:8080/readyz | jq
curl http://localhost:8080/api/v1/version | jq
```

Buka `http://localhost:5173` — halaman menampilkan status `/readyz` yang
diambil lewat CORS dari `faceclock-api`.

#### Jalur Hybrid Development (disarankan untuk dev lokal)

Hanya Postgres (dan AI saat dibutuhkan) yang jalan di Docker; Backend & Frontend
jalan langsung di host agar lebih ringan dan hot-reload lebih cepat.
Setup sekali:

```powershell
# 1) env untuk docker compose (Postgres + AI)
Copy-Item deploy/.env.example deploy/.env
#    → edit deploy/.env, isi POSTGRES_PASSWORD & catat POSTGRES_PORT

# 2) env untuk Backend (host) — DATABASE_URL wajib cocok dengan deploy/.env
Copy-Item apps/backend/.env.local.example apps/backend/.env.local

# 3) env untuk Frontend (host) — BACKEND_URL default sudah benar (localhost:8080)
Copy-Item apps/frontend/.env.example apps/frontend/.env
```

> File di atas sudah ada di-gitignore/dockerignore dan **tidak** dibaca otomatis
> oleh Go/Vite — helper script & target `make dev-*` yang meng-export-nya.

### Menjalankan sehari-hari (Hybrid) — pakai terminal terpisah

Dari **root repo**:

```powershell
# Terminal 1 — Database (detached, hanya Postgres di Docker)
.\scripts\dev-db.ps1                 # setara: make dev-db

# Terminal 2 — Backend (blocking, biarkan terbuka)
.\scripts\dev-backend.ps1            # setara: make dev-backend   -> :8080

# Terminal 3 — Frontend (blocking, biarkan terbuka)
.\scripts\dev-frontend.ps1           # setara: make dev-frontend  -> :5173

# Terminal 4 (opsional) — Service AI hanya saat uji wajah (RAM-heavy)
.\scripts\ai-up.ps1                  # setara: make ai-up         -> :8000
.\scripts\ai-down.ps1                # setara: make ai-down  (matikan & bebaskan RAM)
```

Target `make` alternatif (Git Bash / WSL / Linux/macOS):

```bash
make dev-db
make dev-backend
make dev-frontend
make ai-up
make ai-down
```

### Verifikasi cepat

```powershell
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"   # postgres "healthy"
curl.exe http://localhost:8080/healthz                            # {"status":"ok",...}
curl.exe http://localhost:8080/readyz                             # 503 "degraded" bila AI mati — normal
curl.exe http://localhost:8080/api/v1/version
```

Lalu buka **http://localhost:5173** dan login dengan kredensial super admin di atas.

### Catatan: Backend saat Service AI mati

`faceclock-api` **tidak crash** kalau `faceclock-inference` mati. `/healthz`
selalu OK; `/readyz` melaporkan `degraded` (HTTP 503). Endpoint non-wajah
(auth, employee, user, role, settings) tetap normal; endpoint yang butuh wajah
akan mengembalikan HTTP error biasa (bukan panic). Karena itu AI cukup
dinyalakan on-demand lewat `ai-up`/`ai-down`.

### Menghentikan

- **Backend & Frontend:** di terminalnya tekan `Ctrl + C`.
- **DB / AI (Docker):**
  ```powershell
  docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.override.yml stop
  ```
- **Semua container:** `docker stop $(docker ps -q)`

### Troubleshooting (hybrid)

| Gejala | Penyebab umum |
|---|---|
| `pwsh : not recognized` | PowerShell 7 tidak terpasang → pakai `.\script.ps1` atau `powershell -ExecutionPolicy Bypass -File .\scripts\xxx.ps1` |
| `Missing ...\.env.local` | Perintah dijalankan bukan dari root repo, atau file env belum dibuat (lihat Setup) |
| `running scripts is disabled on this system` | `Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass` lalu ulangi |
| `dev-frontend` port 5173 sudah dipakai | Ada sesi Vite lain — tutup/`Ctrl+C` dulu |
| `/readyz` `degraded` 503 | Normal saat AI mati — jalankan `.\scripts\ai-up.ps1` kalau butuh fitur wajah |

## Test & lint

```bash
make test     # go test -race + web typecheck + pytest
make lint     # go vet/golangci-lint + eslint/tsc + ruff
make fmt      # gofmt + prettier + ruff format
```

## Database & migration

```bash
make psql                    # shell psql ke container
make migrate-down             # revert migration terakhir (proof bahwa down valid)
make migrate-up                # apply lagi
make db-status                 # versi migration + status dirty
make migrate-new name=create_x # generate pasangan file migration baru
```

`faceclock-api` sendiri menjalankan migration otomatis saat startup ketika
`APP_ENV=development` — target `make migrate-*` di atas untuk penggunaan
manual (mis. reset paksa, atau CI).

## Reset total (drop database)

```bash
make reset   # docker compose down -v lalu up lagi — SEMUA DATA DI VOLUME HILANG
```

## Troubleshooting

| Gejala | Penyebab umum |
|---|---|
| `Bind for 0.0.0.0:5433 failed: port is already allocated` | Port Postgres bentrok dengan service lain di mesin dev. Ubah `POSTGRES_PORT` di `deploy/.env` |
| `faceclock-api` exit(1) saat start dengan pesan menyebut nama env var | Fail-fast config (E1) — env wajib kosong/tidak valid. Pesan menyebut **semua** yang kurang sekaligus |
| Migration menyebut `dirty` | `make db-status` untuk melihat versinya, lalu perbaiki manual sebelum `make migrate-up` lagi — jangan hapus baris `schema_migrations` secara manual |

## Dokumen terkait

- [`docs/adr/0001-monorepo.md`](docs/adr/0001-monorepo.md) — kenapa monorepo
- [`docs/adr/0002-inference-prior-art.md`](docs/adr/0002-inference-prior-art.md) — soal `face-engine/`
- [`docs/adr/0003-api-conventions.md`](docs/adr/0003-api-conventions.md) — envelope, katalog error, konvensi skema
- [`docs/api/README.md`](docs/api/README.md) — indeks kontrak API per fase
- [`Plan/09-Revisions-Log.md`](Plan/09-Revisions-Log.md) — seluruh revisi kontrak lintas-fase
