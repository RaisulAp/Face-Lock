"""faceclock-inference - Fase 0 stub.

This is the inference service stub for faceclock-api connectivity.
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


@app.get("/ready")
def ready() -> dict:
    return {
        "data": {
            "status": "ready",
            "model_name": "buffalo_l",
            "model_version": "buffalo_l",
            "embedding_dim": 512,
            "quality_thresholds": {
                "min_det_score": 0.60,
                "min_blur_var": 40.0,
                "min_brightness": 55.0,
                "max_brightness": 215.0,
                "min_face_ratio": 0.18,
                "max_abs_yaw": 0.35,
                "max_abs_pitch": 0.30,
            },
        }
    }


@app.post("/v1/embed")
def embed() -> Response:
    return JSONResponse(
        status_code=501,
        content={
            "error": {
                "code": "NOT_IMPLEMENTED",
                "message": "faceclock-inference is running the Fase 0 stub - real embedding arrives in Fase 2",
                "request_id": "stub",
            }
        },
    )
