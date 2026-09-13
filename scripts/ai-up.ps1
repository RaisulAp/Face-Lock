<#
.SYNOPSIS
    Hybrid Development: start faceclock-inference (Python + ONNX) on demand.
    Equivalent to `make ai-up`. See deploy/HYBRID_DEV.md.
    Reachable afterwards at http://localhost:8000 (INFERENCE_PORT in deploy/.env).
#>
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

& docker compose -f "$root/deploy/docker-compose.yml" -f "$root/deploy/docker-compose.override.yml" --profile ai up -d --build faceclock-inference
