#!/usr/bin/env pwsh
# R2-A bootstrap: register Seedance key + production_authorized.
# Requires: SEEDANCE_API_KEY in env file (never commit the key).

$ErrorActionPreference = 'Stop'
$Base = if ($env:SUB2API_BASE_URL) { $env:SUB2API_BASE_URL } else { 'http://127.0.0.1:18081' }
$EnvFile = if ($env:SUB2API_ENV_FILE) { $env:SUB2API_ENV_FILE } else { 'C:\tmp\sub2api-b1-dev.env' }
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
if ([string]::IsNullOrWhiteSpace($seedanceKey)) { $seedanceKey = $env:SEEDANCE_API_KEY }
if ([string]::IsNullOrWhiteSpace($seedanceKey)) {
    Write-Host 'BLOCKED: set SEEDANCE_API_KEY in env file before running R2-A bootstrap.'
    exit 2
}

$login = Invoke-RestMethod -Method Post -Uri "$Base/api/v1/auth/login" -ContentType 'application/json' -Body (@{
    email = $adminEmail; password = $adminPassword
} | ConvertTo-Json)
$token = $login.data.access_token
if (-not $token) { $token = $login.data.token }
if (-not $token) { throw 'Admin login failed' }
$headers = @{ Authorization = "Bearer $token" }

$providers = Invoke-RestMethod -Uri "$Base/api/v1/admin/video/providers" -Headers $headers
$seedance = $providers.data.items | Where-Object { $_.provider -eq 'seedance' } | Select-Object -First 1
$payload = @{
    display_name = 'Seedance 2.0 - R2 Production'
    enabled = $true
    api_key = $seedanceKey
    default_model = 'doubao-seedance-2-0-260128'
    metadata_json = @{
        production_authorized = $true
        single_smoke_authorized = $true
    }
}
$body = $payload | ConvertTo-Json -Depth 5
$utf8 = New-Object System.Text.UTF8Encoding $false
$tmp = Join-Path $env:TEMP 'r2a-patch-provider.json'
[System.IO.File]::WriteAllText($tmp, $body, $utf8)

if ($seedance -and $seedance.id) {
    curl.exe -s -X PATCH "$Base/api/v1/admin/video/providers/$($seedance.id)" -H "Authorization: Bearer $token" -H "Content-Type: application/json" --data-binary "@$tmp" | Out-Null
    Write-Host "Updated Seedance provider id=$($seedance.id)"
} else {
    throw 'No seedance provider row found'
}

$test = Invoke-RestMethod -Uri "$Base/api/v1/admin/video/providers" -Headers $headers
$ready = $test.data.items | Where-Object { $_.provider -eq 'seedance' -and $_.enabled -and $_.route_available }
if (-not $ready) {
    Write-Host 'Provider registered but route_available is still false; check key and env gates.'
    exit 3
}
Write-Host 'Seedance provider ready. Run tools/r2a-smoke-probe.ps1 next.'
Remove-Item -Force $tmp -ErrorAction SilentlyContinue
