# ADR 0003 — Konvensi API & Skema DB

**Status:** Terkunci di Fase 0, ditegakkan di semua fase berikutnya.
**Rujukan:** [Plan/01-Fase0.md § 2.6](../../Plan/01-Fase0.md#26-scaffolding-faceclock-api-go), § 3.

## Konfirmasi D2–D5

Dikonfirmasi user pada sesi eksekusi Fase 0 (2026-09-04/05): rekomendasi
[Plan/01-Fase0.md § 2.0](../../Plan/01-Fase0.md#20-keputusan-teknis-yang-harus-dikunci-di-fase-0)
dipakai sebagai default, **tidak di-override**:

| # | Keputusan | Final |
|---|---|---|
| D2 | HTTP router Go | **`go-chi/chi/v5`** — dipakai di `internal/httpx/router.go` |
| D3 | Tooling migration | **`golang-migrate/migrate/v4`**, file SQL polos, dijalankan otomatis saat startup dev (`cmd/api/main.go`) dan lewat CLI manual (`make migrate-*`) |
| D4 | Konvensi JSON | **`snake_case`** — lihat bagian terpisah di bawah |
| D5 | Akses database | **`pgx/v5` + `pgxpool`**, **tanpa `sqlc` di Fase 0** — lihat catatan di bawah |

**Deviasi D5 dari rekomendasi:** rekomendasi § 2.5 adalah `pgx/v5` + `sqlc`.
Fase 0 hanya mengimplementasikan `pgx/v5` — `sqlc` belum di-setup karena
belum ada satu query bisnis pun untuk digenerate (migration 000001 hanya
mengaktifkan ekstensi, tidak membuat tabel). Menambahkan tooling `sqlc`
sekarang berarti mengonfigurasi generator tanpa `queries/*.sql` apa pun
untuk dijalankan terhadap — pekerjaan yang tidak menghasilkan apa-apa untuk
diverifikasi. `sqlc` masuk di **Fase 1**, bersamaan dengan skema `users`/
`employees`/`roles` dan query pertama yang sungguh membutuhkannya.

## Envelope response

Sukses: `{"data": ...}`, opsional `"meta": {page, per_page, total, total_pages}`.
Error: `{"error": {"code", "message", "details"?, "request_id"}}`.

`message` boleh dibaca manusia; `code` yang dipakai program. `details` hanya
untuk error validasi. `request_id` **selalu** ada di error.

Implementasi: `apps/faceclock-api/internal/httpx/response.go`. Tidak ada
handler yang boleh memanggil `json.Encode` langsung.

## Katalog error code

13 kode dasar dikunci di Fase 0 (`internal/httpx/errors.go`):

| HTTP | code |
|---|---|
| 400 | `BAD_REQUEST` |
| 401 | `UNAUTHENTICATED` |
| 403 | `FORBIDDEN` |
| 404 | `NOT_FOUND` |
| 409 | `CONFLICT` |
| 413 | `PAYLOAD_TOO_LARGE` |
| 415 | `UNSUPPORTED_MEDIA_TYPE` |
| 422 | `VALIDATION_ERROR` |
| 429 | `RATE_LIMITED` |
| 500 | `INTERNAL_ERROR` |
| 502 | `UPSTREAM_ERROR` |
| 503 | `SERVICE_UNAVAILABLE` |
| 504 | `UPSTREAM_TIMEOUT` |

Fase berikutnya **hanya menambah**, tidak pernah mengubah makna kode yang
sudah ada — setiap tambahan dilaporkan sebagai revisi kontrak di
[Plan/09-Revisions-Log.md](../../Plan/09-Revisions-Log.md).

**405 Method Not Allowed** sengaja tidak punya kode di katalog (chi
menghasilkannya langsung dari routing, sebelum handler mana pun dipanggil).
`FailWithStatus` menuliskannya dengan status HTTP asli 405 dan `code:
"BAD_REQUEST"`, supaya envelope tetap konsisten tanpa memperluas katalog
untuk satu kasus routing.

**[REV-ERR-04](../../Plan/09-Revisions-Log.md#c-perubahan-katalog-error-code)
dicatat di sini sejak Fase 0** meski belum dipakai: status HTTP **410 Gone**
akan masuk katalog di Fase 3, khusus untuk foto yang sudah dihapus kebijakan
retensi, dengan `code: "NOT_FOUND"` supaya client lama tetap menanganinya
tanpa perubahan.

## `snake_case` di semua service

Setiap struct Go yang menyeberang HTTP punya tag eksplisit
(`json:"employee_id"`), tidak pernah default PascalCase Go. Model Pydantic
di `faceclock-inference` memakai nama field `snake_case` apa adanya.

## Konvensi skema database

| Aturan | Nilai |
|---|---|
| Primary key | `uuid DEFAULT gen_random_uuid()` |
| Nama tabel | jamak, `snake_case` |
| Kolom waktu | `timestamptz`, selalu |
| Default waktu | `now()` di sisi DB |
| Audit kolom | `created_at`, `updated_at` di semua tabel bisnis |
| Soft delete | `deleted_at` untuk `users`, `employees`, `roles`. Tabel append-only **tidak** punya soft delete |
| Enum | `text` + `CHECK`, bukan tipe `ENUM` |
| FK | `ON DELETE` eksplisit; default `RESTRICT`, `CASCADE` hanya untuk pivot |
| Koordinat | `double precision` |

### Pengecualian konvensi yang disetujui

Tiga pengecualian berikut menyimpang dari tabel di atas dan **sudah
disetujui** ([09-Revisions-Log.md § 2 G](../../Plan/09-Revisions-Log.md#g-perubahan-konvensi-umum--pengecualian-yang-disetujui)),
dicatat di sini sejak Fase 0 supaya Fase 1/3/4 tidak dianggap melanggar
konvensi saat menerapkannya:

| ID | Pengecualian | Alasan |
|---|---|---|
| REV-CONV-01 | `audit_logs.id` memakai `bigint GENERATED ALWAYS AS IDENTITY`, bukan `uuid` | Tabel append-only, sangat banyak baris, urutan sisip bermakna |
| REV-CONV-02 | `attendance_attempts.id` memakai `bigint identity` | Alasan identik dengan REV-CONV-01 |
| REV-CONV-03 | `face_references` tanpa `deleted_at` | Siklus hidupnya `is_active`; penghapusan permanen adalah hard delete lewat prosedur PDP, bukan soft delete |

## Middleware chain (terkunci)

```
RequestID → RealIP → StructuredLogger → Recoverer → CORS
  → Timeout(30s) → BodyLimit(10MB) → RateLimit
    → [Fase 1] Authenticate → [Fase 1] RequirePermission(...)
      → [Fase 3] RequireConsent(...)   ← REV-MW-01
        → handler
```

`RequestID` paling luar supaya semua log punya korelasi. `Recoverer` di
dalam `StructuredLogger` supaya panic tetap tercatat. `Authenticate` di
dalam `Timeout` supaya query DB untuk auth ikut terpotong deadline.

**[REV-MW-01](../../Plan/09-Revisions-Log.md#f-perubahan-middleware-chain):**
slot `RequireConsent` dicetak di sini sejak Fase 0 (belum ada implementasinya
— itu pekerjaan Fase 3), diletakkan **setelah** `RequirePermission` dengan
sengaja: pihak yang tidak berhak sama sekali harus menerima `403 FORBIDDEN`,
bukan `403 CONSENT_REQUIRED` yang membocorkan keberadaan seorang karyawan
kepada pemanggil yang bahkan tidak punya permission untuk melihatnya.

Implementasi Fase 0 berhenti di `RateLimit` — lihat
`apps/faceclock-api/internal/httpx/router.go` untuk titik mount yang sudah
disiapkan.
