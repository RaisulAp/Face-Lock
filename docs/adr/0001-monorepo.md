# ADR 0001 — Monorepo

**Status:** ✅ Terkunci — dikonfirmasi user 2026-09-04.
**Rujukan:** [Plan/01-Fase0.md § 2.1](../../Plan/01-Fase0.md#21-d1--monorepo-vs-multi-repo--terkunci)

## Keputusan

Faceclock dikembangkan sebagai **satu monorepo**, root di folder project ini,
dengan tiap service (`faceclock-api`, `faceclock-web`, `faceclock-inference`)
tetap *independently deployable* lewat context Docker-nya masing-masing.

## Alasan

1. **Kontrak antar-service masih bergerak.** Kontrak `faceclock-api` ↔
   `faceclock-inference` didefinisikan Fase 2 dan hampir pasti berubah di
   Fase 3–4. Di monorepo, satu commit atomik mengubah server dan client
   sekaligus; CI menguji keduanya bersama.
2. **Tim kecil.** Keuntungan utama multi-repo — isolasi izin akses dan
   siklus rilis terpisah antar-tim — tidak berlaku di sini.
3. **Satu `docker compose up`** adalah deliverable Fase 0 itu sendiri.
4. **Satu sumber kebenaran** untuk `Plan/` dan `DONE-Fase-N.md`.
5. **Onboarding**: satu clone, satu `make up`.

## Risiko dan mitigasi

| Risiko | Mitigasi |
|---|---|
| Build context Docker jadi raksasa | Setiap `Dockerfile` punya context sendiri (`apps/faceclock-api/`, bukan root) + `.dockerignore` ketat |
| Coupling tidak sengaja lintas bahasa | Tidak mungkin lintas bahasa; antar-modul Go dijaga `internal/` |
| CI jalan semua padahal cuma satu folder berubah | `paths:` filter per job di GitHub Actions |
| `faceclock-inference` harus bisa dideploy sendiri | Folder-nya self-contained: `Dockerfile`, `requirements.txt`, `.env.example` sendiri |

## Alternatif yang ditolak — multi-repo

Akan dipilih bila `faceclock-inference` dikelola tim berbeda, atau ada
kebutuhan compliance yang melarang kode yang menyentuh data biometrik berada
satu repo dengan kode aplikasi. Keduanya tidak berlaku di project ini.
Pemisahan tetap mungkin di masa depan, tapi harus jadi keputusan sadar
dengan ADR baru — bukan default.
