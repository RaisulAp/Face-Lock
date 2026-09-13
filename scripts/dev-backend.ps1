<#
.SYNOPSIS
    FaceClock BACKEND (BE): Jalankan backend Go (apps/backend) di host Windows.
    Penggunaan: .\scripts\dev-backend.ps1 atau make dev-backend
#>
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$apiDir = Join-Path $root "apps\backend"
$envFile = Join-Path $apiDir ".env.local"

if (-not (Test-Path $envFile)) {
    Write-Error "Missing $envFile - copy it from .env.local.example first, then adjust DATABASE_URL/port to match deploy/.env."
    exit 1
}

# Muat environment variables dari .env.local
Get-Content $envFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -eq "" -or $line.StartsWith("#")) { return }
    $idx = $line.IndexOf("=")
    if ($idx -lt 1) { return }
    $key = $line.Substring(0, $idx).Trim()
    $value = $line.Substring($idx + 1).Trim()
    if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
        if ($value.Length -ge 2) {
            $value = $value.Substring(1, $value.Length - 2)
        }
    }
    Set-Item -Path "env:$key" -Value $value
}

Write-Host "==================================================" -ForegroundColor Cyan
Write-Host " [FACECLOCK] Menjalankan BACKEND (BE) - Go REST API" -ForegroundColor Green
Write-Host " Lokasi Folder : apps/backend" -ForegroundColor DarkGray
Write-Host " API URL       : http://localhost:$($env:APP_PORT)" -ForegroundColor Yellow
Write-Host " Database      : $($env:DATABASE_URL)" -ForegroundColor DarkGray
Write-Host "==================================================" -ForegroundColor Cyan

Push-Location $apiDir
try {
    & go run ./cmd/api
}
finally {
    Pop-Location
}
