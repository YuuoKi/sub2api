# Runs tools/secret_scan.py from the repository root.
# Exits 0 when clean; exits 2 to block when secrets are found (Cursor hook convention).
param()

$ErrorActionPreference = 'Stop'

$stdin = [Console]::In.ReadToEnd()
$repoRoot = if ($env:CURSOR_PROJECT_DIR) { $env:CURSOR_PROJECT_DIR } else { (Get-Location).Path }
Set-Location $repoRoot

function Find-Python {
    foreach ($candidate in @('python3', 'python', 'py')) {
        $cmd = Get-Command $candidate -ErrorAction SilentlyContinue
        if ($null -ne $cmd -and $cmd.Source -notmatch 'WindowsApps\\python') {
            return $cmd.Source
        }
    }
    return $null
}

$python = Find-Python
if (-not $python) {
    # Python not available locally; do not block agent work.
    exit 0
}

$scanner = Join-Path $repoRoot 'tools\secret_scan.py'
if (-not (Test-Path $scanner)) {
    exit 0
}

& $python $scanner --include-untracked
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    Write-Error "secret-scan found potential secrets. Run: make secret-scan"
    exit 2
}
exit 0
