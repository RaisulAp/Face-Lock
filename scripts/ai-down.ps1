<#
.SYNOPSIS
    Hybrid Development: stop faceclock-inference and free its RAM.
    Equivalent to `make ai-down`. See deploy/HYBRID_DEV.md.
#>
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

& docker compose -f "$root/deploy/docker-compose.yml" -f "$root/deploy/docker-compose.override.yml" --profile ai stop faceclock-inference
& docker compose -f "$root/deploy/docker-compose.yml" -f "$root/deploy/docker-compose.override.yml" --profile ai rm -f faceclock-inference
