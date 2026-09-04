# faceclock-inference — Fase 0 stub

**Ini bukan implementasi sungguhan.** `app/main.py` hanya menyediakan
`GET /health` (selalu `200 {"stub": true}`) dan `POST /v1/embed` (selalu
`501 NOT_IMPLEMENTED`), supaya `faceclock-api`'s `/readyz` bisa membuktikan
konektivitas dan wiring `docker compose` sekarang, tanpa menunggu model
InsightFace (~330 MB).

Diganti isinya sepenuhnya di **Fase 2** ([Plan/03-Fase2.md](../../Plan/03-Fase2.md)),
memakai `face-engine/` (di root repo) sebagai prior art. Kontrak path
(`/health`, `/v1/embed`) sengaja dipakai sejak stub ini supaya perpindahan ke
implementasi sungguhan tidak mengubah apa yang sudah dipanggil `faceclock-api`.

## Jalankan lokal (tanpa Docker)

```bash
cd services/faceclock-inference
python -m venv .venv && source .venv/bin/activate   # Windows: .venv\Scripts\activate
pip install -r requirements-dev.txt
uvicorn app.main:app --reload --port 8000
```

## Test

```bash
pytest
ruff check .
```
