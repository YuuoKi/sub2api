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
    # Cursor hook shells may start before a fresh install refreshes PATH.
    $local = [Environment]::GetFolderPath('LocalApplicationData')
    foreach ($rel in @(
        'Programs\Python\Python312\python.exe',
        'Programs\Python\Python311\python.exe',
        'Programs\Python\Python313\python.exe'
    )) {
        $path = Join-Path $local $rel
        if (Test-Path -LiteralPath $path) {
            return $path
        }
    }
    return $null
}

$python = Find-Python
if (-not $python) {
    $ErrorActionPreference = 'Continue'
    Write-Error "secret-scan requires Python (python3/python/py). Install Python and retry, or run: make secret-scan"
    exit 2
}

$scanner = Join-Path $repoRoot 'tools\secret_scan.py'
if (-not (Test-Path $scanner)) {
    $ErrorActionPreference = 'Continue'
    Write-Error "secret-scan scanner missing at tools\secret_scan.py"
    exit 2
}

& $python $scanner --include-untracked
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    Write-Error "secret-scan found potential secrets. Run: make secret-scan"
    exit 2
}
exit 0
