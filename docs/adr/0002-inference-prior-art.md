# ADR 0002 — `face-engine/` sebagai prior art untuk Fase 2

**Status:** Dicatat, tidak dieksekusi di Fase 0.
**Rujukan:** [Plan/01-Fase0.md § 2.1](../../Plan/01-Fase0.md#catatan-tentang-folder-face-engine-yang-sudah-ada)

## Konteks

Di root project ini sudah ada folder `face-engine/` — service inference
Python/FastAPI + InsightFace milik project lain (SIMRS/VitaCore) yang sudah
matang secara produksi. Folder ini **bukan** bagian dari Faceclock, tapi
prior art yang sangat berharga untuk [Fase 2](../../Plan/03-Fase2.md), yang
akan mem-porting pola-polanya (bukan memindahkan foldernya) ke
`services/faceclock-inference/`.

## Keputusan

- `face-engine/` **tidak diubah, tidak dihapus, tidak dijadikan**
  `services/faceclock-inference/` di Fase 0. Fase 0 hanya membuat
  `services/faceclock-inference/` sebagai stub baru dari nol (§ 2.8).
- `face-engine/` **wajib di-`.gitignore`** — sudah diverifikasi masuk
  `.gitignore` **sebelum** `git init` dilakukan di fase ini (isinya
  `.venv/`, model `.onnx` ~330 MB, dataset LFW ratusan MB; sekali ter-commit,
  ukuran repo rusak permanen dan riwayat git harus di-rewrite untuk
  membersihkannya).
- Pola yang akan diambil alih di Fase 2 (bukan kode literalnya): anti-spill
  `MultiPartParser.spool_max_size`, model yang di-bake ke image (bukan
  didownload saat runtime), container `read_only: true`, dependency version
  pinned persis.

## Lisensi

Belum diverifikasi di Fase 0 — ini bukan blocker untuk fondasi (tidak ada
kode dari `face-engine/` yang disalin di fase ini), tapi **wajib** diperiksa
sebelum Fase 2 mem-porting kode apa pun darinya. Dicatat sebagai item terbuka
untuk Fase 2, bukan diselesaikan di sini.
