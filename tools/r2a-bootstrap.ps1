#!/usr/bin/env pwsh
# R2-A bootstrap: register Seedance key + production_authorized, then optional smoke probe.
# Requires: SEEDANCE_API_KEY in env file (never commit the key).
# Usage:
#   $env:SUB2API_ENV_FILE = 'C:\tmp\sub2api-b1-dev.env'
#   pwsh -File tools/r2a-bootstrap.ps1

$ErrorActionPreference = 'Stop'
$Base = $env:SUB2API_BASE_URL
if (-not $Base) { $Base = 'http://127.0.0.1:18081' }
$EnvFile = $env:SUB2API_ENV_FILE
if (-not $EnvFile) { $EnvFile = 'C:\tmp\sub2api-b1-dev.env' }
if (-not (Test-Path $EnvFile)) { throw "Env file not found: $EnvFile" }

function Get-EnvValue([string]$Name) {
    foreach ($line in Get-Content $EnvFile) {
        if ($line -match "^\s*$Name=(.*)$") { return $Matches[1].Trim() }
    }
    return ''
}

$adminEmail = Get-EnvValue 'ADMIN_EMAIL'
$adminPassword = Get-EnvValue 'ADMIN_PASSWORD'
if ([string]::IsNullOrWhiteSpace($adminEmail) -or [string]::IsNullOrWhiteSpace($adminPassword)) {
    throw 'Set ADMIN_EMAIL and ADMIN_PASSWORD in env file'
}
$seedanceKey = Get-EnvValue 'SEEDANCE_API_KEY'
if ([string]::IsNullOrWhiteSpace($seedanceKey)) {
    $seedanceKey = $env:SEEDANCE_API_KEY
}
if ([string]::IsNullOrWhiteSpace($seedanceKey)) {
    Write-Host 'BLOCKED: set SEEDANCE_API_KEY in env file before running R2-A bootstrap.'
    exit 2
}

$login = Invoke-RestMethod -Method Post -Uri "$Base/api/v1/auth/login" -ContentType 'application/json' -Body (@{
    email = $adminEmail
    password = $adminPassword
} | ConvertTo-Json)
$token = $login.data.access_token
if (-not $token) { $token = $login.data.token }
if (-not $token) { throw 'Admin login failed' }
$headers = @{ Authorization = "Bearer $token" }

$providers = Invoke-RestMethod -Uri "$Base/api/v1/admin/video/providers" -Headers $headers
$seedance = $providers.data.items | Where-Object { $_.provider -eq 'seedance' } | Select-Object -First 1
$payload = @{
    provider = 'seedance'
    display_name = 'Seedance 2.0 - R2 Production'
    enabled = $true
    api_key = $seedanceKey
    default_model = 'doubao-seedance-2-0-260128'
    metadata = @{
        production_authorized = $true
        single_smoke_authorized = $true
    }
}
if ($seedance -and $seedance.id) {
    Invoke-RestMethod -Method Put -Uri "$Base/api/v1/admin/video/providers/$($seedance.id)" -Headers $headers -ContentType 'application/json' -Body ($payload | ConvertTo-Json -Depth 5) | Out-Null
    Write-Host "Updated Seedance provider id=$($seedance.id)"
} else {
    Invoke-RestMethod -Method Post -Uri "$Base/api/v1/admin/video/providers" -Headers $headers -ContentType 'application/json' -Body ($payload | ConvertTo-Json -Depth 5) | Out-Null
    Write-Host 'Created Seedance provider'
}

$test = Invoke-RestMethod -Uri "$Base/api/v1/admin/video/providers" -Headers $headers
$ready = $test.data.items | Where-Object { $_.provider -eq 'seedance' -and $_.enabled -and $_.route_available }
if (-not $ready) {
    Write-Host 'Provider registered but route_available is still false; check key and env gates.'
    exit 3
}
Write-Host 'Seedance provider ready for R2-A. Run tools/r2a-smoke-probe.ps1 next.'
