# Hybrid Development

Menjalankan seluruh stack (postgres + faceclock-api + faceclock-web +
faceclock-inference) sekaligus di Docker Desktop/WSL2 bisa memakan RAM cukup
besar untuk laptop dev — terutama `faceclock-inference` begitu fase model
ONNX-nya aktif. **Hybrid Development** memisahkan mana yang benar-benar perlu
Docker dan mana yang lebih ringan jalan langsung di host:

| Komponen | Nama Folder / Service | Hybrid mode | Kenapa |
|---|---|---|---|
| **Database (DB)** | `postgres` | Docker, selalu jalan | Butuh image `pgvector/pgvector`, tidak praktis di-install native di Windows |
| **Service AI / Face Engine** | `services/faceclock-inference` | Docker, **on-demand** | Python + ONNX, paling berat, cuma perlu nyala saat test enrollment/absensi wajah |
| **Backend (BE)** | `apps/backend` | Host (`go run`) | Binary Go REST API, ~40MB RAM, jauh lebih ringan & responsif di host |
| **Frontend (FE)** | `apps/frontend` | Host (`npm run dev`) | Vite React App, hot reload lebih cepat tanpa lapisan bind-mount Docker |

Ini murni mode pengembangan lokal.

## Setup sekali di awal

1. **Postgres**: pastikan `deploy/.env` sudah berisi `POSTGRES_PASSWORD` dan
   catat nilai `POSTGRES_PORT`-nya (pada setup ini: port `5434`).
2. **Backend (BE)**:
   ```
   cp apps/backend/.env.local.example apps/backend/.env.local
   ```
   Pastikan `DATABASE_URL` mengarah ke port `5434`. File ini sudah otomatis dibuat dan di-gitignore.
3. **Frontend (FE)**:
   ```
   cp apps/frontend/.env.example apps/frontend/.env
   ```
   Default `BACKEND_URL=http://localhost:8080` sudah benar untuk hybrid mode
   (Backend berjalan di host). File ini sudah otomatis dibuat dan di-dockerignore.
4. (Opsional tapi disarankan) Batasi RAM WSL2 — lihat `deploy/wslconfig.example`.

## Pemakaian sehari-hari

```bash
# 1) Nyalakan database (hanya PostgreSQL di Docker)
pwsh .\scripts\dev-db.ps1       # atau: make dev-db

# 2) Jalankan Backend (BE) & Frontend (FE) di dua terminal terpisah di host
pwsh .\scripts\dev-backend.ps1  # (atau: make dev-backend)  -> :8080
pwsh .\scripts\dev-frontend.ps1 # (atau: make dev-frontend) -> :5173

# 3) Butuh fitur wajah (enrollment/absensi)? Nyalakan AI saat itu saja
pwsh .\scripts\ai-up.ps1        # (atau: make ai-up)    -> :8000
# ... testing presensi/wajah ...
pwsh .\scripts\ai-down.ps1      # (atau: make ai-down)  -> matikan dan bebaskan RAM lagi
```

`make ai-up`/`ai-down` hanya menyalakan/mematikan container
`faceclock-inference` — postgres tidak ikut terganggu.

### faceclock-api saat AI mati

`faceclock-api` **tidak crash** kalau `faceclock-inference` sedang mati:
- Startup (`cmd/api/main.go`) tidak pernah memanggil inference — hanya
  membuat client HTTP-nya (`inference.New`, `face.NewRESTEngine`), tanpa
  dial apa pun ke `INFERENCE_BASE_URL` saat boot.
- `/healthz` selalu OK selama proses hidup (tidak menyentuh dependency apa
  pun — lihat `internal/health/handler.go`).
- `/readyz` akan melaporkan `"status": "degraded"` (HTTP 503) kalau AI mati,
  tapi endpoint lain (auth, employee, user, role, settings non-wajah, dst.)
  tetap berfungsi normal.
- Endpoint yang benar-benar butuh wajah (enrollment, clock-in/out,
  reindex) akan mengembalikan HTTP error biasa (mis. 502/500 dari handler)
  saat memanggil inference dan gagal — bukan panic/crash proses.

Ini bukan perubahan kode baru; ini memang bagaimana `internal/inference` dan
`internal/health` sudah didesain (semua pemanggilan inference mengembalikan
`error` lewat jalur HTTP client biasa).

## Fallback: jalankan semua di Docker seperti biasa

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.override.yml \
  --profile ai --profile docker-app up -d --build
```

Kedua profile harus disebut sekaligus — Compose menolak start kalau
`faceclock-api`/`faceclock-web` (`docker-app`) aktif tapi `faceclock-inference`
(`ai`, yang jadi `depends_on`-nya) tidak, jadi tidak ada pull-in otomatis
lintas profile.

## Kenapa `deploy/.env.example` sekarang punya `INFERENCE_PORT`

`docker-compose.yml` (base/prod/CI) **sengaja tanpa** `ports:` untuk
`faceclock-inference` — hanya `faceclock-api` di dalam network compose yang
boleh mengaksesnya. `INFERENCE_PORT` (default `8000`) hanya dipakai oleh
`docker-compose.override.yml` untuk publish port itu ke host, khusus supaya
`faceclock-api` yang jalan di host (bukan di container) bisa
menghubunginya di `localhost:8000`.

## Batas memori container

`docker-compose.override.yml` menambahkan `mem_limit`:
- `postgres`: 512MB
- `faceclock-inference`: 1.5GB

Ini pagar dev, bukan sizing produksi — kalau `faceclock-inference` sering
kena OOM-killed setelah model ONNX yang sebenarnya (bukan stub Fase 0) aktif,
naikkan angka ini di `docker-compose.override.yml` (referensi: prior-art
`face-engine/docker-compose.face-engine.yml` mengukur ~1.2GB committed untuk
detection+recognition, dengan limit 2GB).
