"""faceclock-inference — Fase 0 stub.

This is NOT the real inference service. It exists so faceclock-api's
/readyz can prove connectivity and compose wiring now, without waiting on
the InsightFace model (~330MB) that Fase 2 adds. See Plan/01-Fase0.md § 2.8
and Plan/03-Fase2.md for the real implementation, which replaces this file
without changing the /health and /v1/embed contract paths.
"""

from fastapi import FastAPI, Response
from fastapi.responses import JSONResponse

app = FastAPI(
    title="faceclock-inference (stub)",
    docs_url=None,
    redoc_url=None,
    openapi_url=None,
)


@app.get("/health")
def health() -> dict:
    return {"status": "ok", "model_version": "stub", "stub": True}


@app.post("/v1/embed")
def embed() -> Response:
    return JSONResponse(
        status_code=501,
        content={
            "error": {
                "code": "NOT_IMPLEMENTED",
                "message": "faceclock-inference is running the Fase 0 stub — "
                "real embedding arrives in Fase 2",
                "request_id": "stub",
            }
        },
    )
