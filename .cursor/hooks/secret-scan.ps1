# Runs tools/secret_scan.py from the repository root before `git push`.
# Emits beforeShellExecution JSON: allow on clean scan, deny when secrets/errors.
param()

$ErrorActionPreference = 'Stop'

function Write-HookResult {
    param(
        [Parameter(Mandatory = $true)][ValidateSet('allow', 'deny', 'ask')][string]$Permission,
        [string]$UserMessage = '',
        [string]$AgentMessage = ''
    )
    $payload = [ordered]@{ permission = $Permission }
    if ($UserMessage) { $payload.user_message = $UserMessage }
    if ($AgentMessage) { $payload.agent_message = $AgentMessage }
    $payload | ConvertTo-Json -Compress
}

# Consume Cursor hook stdin JSON (command metadata); ignore contents for this gate.
$null = [Console]::In.ReadToEnd()

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
    Write-HookResult -Permission deny `
        -UserMessage 'secret-scan requires Python (python3/python/py). Install Python and retry, or run: make secret-scan' `
        -AgentMessage 'Push blocked: secret-scan hook could not find a non-WindowsApps Python.'
    exit 0
}

$scanner = Join-Path $repoRoot 'tools\secret_scan.py'
if (-not (Test-Path $scanner)) {
    Write-HookResult -Permission deny `
        -UserMessage 'secret-scan scanner missing at tools/secret_scan.py' `
        -AgentMessage 'Push blocked: tools/secret_scan.py is missing.'
    exit 0
}

# Capture scanner stdout/stderr so only JSON reaches Cursor on this script's stdout.
$output = & $python $scanner --include-untracked 2>&1 | Out-String
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    Write-HookResult -Permission deny `
        -UserMessage 'secret-scan found potential secrets. Run: make secret-scan' `
        -AgentMessage ("Push blocked by secret-scan.`n" + $output.Trim())
    exit 0
}

Write-HookResult -Permission allow
exit 0
