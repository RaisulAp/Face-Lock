# Indeks Kontrak API — Faceclock

Dokumen kontrak lengkap per fase ada di `Plan/`. Folder ini (`docs/api/`)
menyimpan **berkas kanonik** yang dibaca lebih dari satu bahasa/service
sekaligus, supaya tidak ada dua sumber kebenaran yang bisa menyimpang.

| Berkas | Dipakai oleh | Diisi di |
|---|---|---|
| [`hints.json`](hints.json) | `internal/inference/hints.go` (Go), `lib/errors/hints.ts` (TS), `core/messages/hints.dart` (Dart, Fase 7) | Fase 2 (kosakata `server_hints`), Fase 7 (`client_coach`) |
| [`geo-testcases.json`](geo-testcases.json) | `internal/geo` (Go), `lib/geo` (TS) | Fase 4 |

Kedua berkas ini adalah konsekuensi langsung dari **D1 = monorepo**
(dikunci 2026-09-04): satu berkas sumber dibaca semua sisi, tanpa
distribusi paket berversi terpisah. Lihat
[REV-CTR-01/02/03](../../Plan/09-Revisions-Log.md#j-artefak-kontrak-bersama-berkas-kanonik).

## Ringkasan endpoint per fase

| Fase | Endpoint | Dokumen |
|---|---|---|
| 0 | `GET /healthz`, `GET /readyz`, `GET /api/v1/version` | [Plan/01-Fase0.md § 4](../../Plan/01-Fase0.md#4-daftar-endpoint) |
| 1 | Auth, user, role, permission, settings, audit (#1–#31) | [Plan/02-Fase1.md](../../Plan/02-Fase1.md) |
| 2 | `faceclock-inference` internal (`/health`, `/ready`, `/v1/embed`, dst.) | [Plan/03-Fase2.md](../../Plan/03-Fase2.md) |
| 3 | Consent, enrollment wajah (#32–#52) | [Plan/04-Fase3.md](../../Plan/04-Fase3.md) |
| 4 | Attendance (#53–#70), `/attendances/context` | [Plan/05-Fase4.md](../../Plan/05-Fase4.md) |
| 5 | Admin panel: rekap, export, `/settings/face-quality-status` (#71–#73) | [Plan/06-Fase5.md](../../Plan/06-Fase5.md) |
| 6 | Konsumen murni — tidak ada endpoint baru | [Plan/07-Fase6.md](../../Plan/07-Fase6.md) |
| 7 | Konsumen murni — tidak ada endpoint baru | [Plan/08-Fase7.md](../../Plan/08-Fase7.md) |

## Endpoint pasca-fase

Ditambahkan setelah Fase 6, sebagai konsekuensi dari pendaftaran satu-langkah
(migration `000025`), yang membuat HR tidak lagi mengisi seluruh data karyawan
sehingga karyawan perlu melengkapi sendiri.

| Endpoint | Guard | Ditambahkan |
|---|---|---|
| `PATCH /api/v1/employees/me/profile` | `employee.update_self` | migration `000026` |

**Kontrak**: hanya menerima `employee_number`, `position`, `join_date`, dan
`phone`. Field admin-only (`employment_status`, `department`, `email`,
`office_location_id`) **ditolak dengan 400**, bukan diabaikan diam-diam, karena
`httpx.DecodeAndValidate` mengaktifkan `DisallowUnknownFields`. Aturan
`profile_completed` menjadi `true` saat NIP, jabatan, dan tanggal masuk terisi —
sama persis dengan aturan pada `PATCH /api/v1/employees/{id}` milik admin.

Katalog endpoint lengkap dan revisinya ada di
[Plan/09-Revisions-Log.md § 2 B](../../Plan/09-Revisions-Log.md#b-perubahan-katalog-endpoint).
