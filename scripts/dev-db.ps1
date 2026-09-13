<#
.SYNOPSIS
    Hybrid Development: start ONLY Postgres (pgvector) in Docker.
    Equivalent to `make dev-db`. See deploy/HYBRID_DEV.md.
#>
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

& docker compose -f "$root/deploy/docker-compose.yml" -f "$root/deploy/docker-compose.override.yml" up -d postgres
