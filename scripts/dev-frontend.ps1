<#
.SYNOPSIS
    FaceClock FRONTEND (FE): Jalankan frontend Vite (apps/frontend) di host Windows.
    Penggunaan: .\scripts\dev-frontend.ps1 atau make dev-frontend
#>
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$webDir = Join-Path $root "apps\frontend"
$envFile = Join-Path $webDir ".env"

if (Test-Path $envFile) {
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
}

if (-not $env:BACKEND_URL) {
    $env:BACKEND_URL = "http://localhost:8080"
}

Write-Host "==================================================" -ForegroundColor Cyan
Write-Host " [FACECLOCK] Menjalankan FRONTEND (FE) - Vite Web" -ForegroundColor Green
Write-Host " Lokasi Folder : apps/frontend" -ForegroundColor DarkGray
Write-Host " Web App URL   : http://localhost:5173" -ForegroundColor Yellow
Write-Host " Proxy /api ke : $($env:BACKEND_URL)" -ForegroundColor DarkGray
Write-Host "==================================================" -ForegroundColor Cyan

Push-Location $webDir
try {
    npm.cmd run dev
}
finally {
    Pop-Location
}
