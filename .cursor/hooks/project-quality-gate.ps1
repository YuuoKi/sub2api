# Sub2API-specific quality gate (invoked by ~/.cursor/hooks/agent-stop-quality.ps1).
param([string[]]$Changed = @())

$needsBackend = $false
$needsFrontend = $false
foreach ($line in $Changed) {
    if ($line -match '^backend/') { $needsBackend = $true }
    if ($line -match '^frontend/') { $needsFrontend = $true }
}

if (-not $needsBackend -and -not $needsFrontend) { exit 0 }

$parts = @()
if ($needsBackend) { $parts += 'make test-backend (or cd backend && golangci-lint run ./...)' }
if ($needsFrontend) { $parts += 'make test-frontend (lint + typecheck + critical vitest)' }

$message = "Sub2API quality gate: diff touches " + ($parts -join ' and ') + ". Run check-compiler-errors / verification-before-completion before marking done. Before push, use /review-bugbot."

@{ followup_message = $message } | ConvertTo-Json -Compress
exit 0
