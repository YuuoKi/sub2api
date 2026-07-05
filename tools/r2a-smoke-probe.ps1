#!/usr/bin/env pwsh
# R2-A smoke probe via API-key production path (no trial_mode).
# Prereq: tools/r2a-bootstrap.ps1 succeeded; employee API key assigned to group 1.

$ErrorActionPreference = 'Stop'
$Base = if ($env:SUB2API_BASE_URL) { $env:SUB2API_BASE_URL } else { 'http://127.0.0.1:18081' }
$BodyPath = Join-Path $PSScriptRoot 'r2a-probe-body.json'

$apiKey = (wsl -e docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -t -A -c "SELECT key FROM api_keys WHERE user_id=(SELECT id FROM users WHERE email='zoucha-test@wujie.local') AND deleted_at IS NULL LIMIT 1;").Trim()
if (-not $apiKey) { throw 'No employee API key found' }

$resp = curl.exe -s -w "`nHTTP:%{http_code}" -X POST "$Base/v1/video/tasks" `
    -H "Authorization: Bearer $apiKey" `
    -H "Content-Type: application/json" `
    --data-binary "@$BodyPath"
Write-Host $resp

if ($resp -match 'HTTP:201') {
    $json = ($resp -split "`n")[0] | ConvertFrom-Json
    $taskId = $json.data.id
    Write-Host "Task created id=$taskId — poll GET $Base/v1/video/tasks/$taskId"
}
