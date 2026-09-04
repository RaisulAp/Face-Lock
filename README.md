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
