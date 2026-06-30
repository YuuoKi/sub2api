# After agent completes, nudge verification when Go/Vue files may have changed.
param()

$ErrorActionPreference = 'SilentlyContinue'

$repoRoot = if ($env:CURSOR_PROJECT_DIR) { $env:CURSOR_PROJECT_DIR } else { (Get-Location).Path }
Set-Location $repoRoot

$changed = git diff --name-only HEAD 2>$null
if (-not $changed) {
    $changed = git diff --name-only 2>$null
}
if (-not $changed) {
    exit 0
}

$needsBackend = $false
$needsFrontend = $false
foreach ($line in ($changed -split "`n")) {
    if ($line -match '^backend/') { $needsBackend = $true }
    if ($line -match '^frontend/') { $needsFrontend = $true }
}

if (-not $needsBackend -and -not $needsFrontend) {
    exit 0
}

$parts = @()
if ($needsBackend) { $parts += 'make test-backend (or cd backend && golangci-lint run ./...)' }
if ($needsFrontend) { $parts += 'make test-frontend (lint + typecheck + critical vitest)' }

$message = "Sub2API quality gate: diff touches " + ($parts -join ' and ') + ". Run check-compiler-errors / verification-before-completion before marking done. Before push, use /review-bugbot."

@{
    followup_message = $message
} | ConvertTo-Json -Compress
exit 0
