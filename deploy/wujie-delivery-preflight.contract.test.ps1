[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$scriptPath = Join-Path $PSScriptRoot 'wujie-delivery-preflight.ps1'
$failures = [System.Collections.Generic.List[string]]::new()

function Assert-True {
    param(
        [Parameter(Mandatory)]
        [bool]$Condition,

        [Parameter(Mandatory)]
        [string]$Message
    )

    if (-not $Condition) {
        $failures.Add($Message)
    }
}

function Assert-Match {
    param(
        [Parameter(Mandatory)]
        [string]$Content,

        [Parameter(Mandatory)]
        [string]$Pattern,

        [Parameter(Mandatory)]
        [string]$Message
    )

    Assert-True -Condition ([regex]::IsMatch($Content, $Pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)) -Message $Message
}

function Assert-NotMatch {
    param(
        [Parameter(Mandatory)]
        [string]$Content,

        [Parameter(Mandatory)]
        [string]$Pattern,

        [Parameter(Mandatory)]
        [string]$Message
    )

    Assert-True -Condition (-not [regex]::IsMatch($Content, $Pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)) -Message $Message
}

Assert-True -Condition (Test-Path -LiteralPath $scriptPath -PathType Leaf) -Message 'Missing deploy/wujie-delivery-preflight.ps1.'

if (Test-Path -LiteralPath $scriptPath -PathType Leaf) {
    $scriptContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $scriptPath
    $parseErrors = $null
    [void][System.Management.Automation.Language.Parser]::ParseFile($scriptPath, [ref]$null, [ref]$parseErrors)

    Assert-True -Condition ($parseErrors.Count -eq 0) -Message 'Delivery preflight script has syntax errors.'
    Assert-Match -Content $scriptContent -Pattern "ValidateSet\s*\(\s*'Check'\s*,\s*'Build'\s*\)" -Message 'Action must be restricted to Check and Build.'
    Assert-Match -Content $scriptContent -Pattern '\[Parameter\(Mandatory\)\][\r\n\s]+\[string\]\$RepoRoot' -Message 'RepoRoot must be explicit and mandatory.'
    Assert-Match -Content $scriptContent -Pattern '\[Parameter\(Mandatory\)\][\r\n\s]+\[string\]\$ReleaseRoot' -Message 'ReleaseRoot must be explicit and mandatory.'
    Assert-Match -Content $scriptContent -Pattern '\[Parameter\(Mandatory\)\][\r\n\s]+\[string\]\$ExpectedBranch' -Message 'ExpectedBranch must be explicit and mandatory.'
    Assert-Match -Content $scriptContent -Pattern '\[Parameter\(Mandatory\)\][\r\n\s]+\[ValidatePattern' -Message 'RequiredProductCommit must be explicit and validated.'
    Assert-NotMatch -Content $scriptContent -Pattern 'console-unification|wujie-console-unification-20260717|4c6111502eb59e83e2c5d750a2a724aaf1f70b55' -Message 'Stale hard-coded delivery worktree/branch/commit remains.'
    Assert-Match -Content $scriptContent -Pattern 'Resolve-RequiredDirectory' -Message 'Missing explicit path existence checks.'
    Assert-Match -Content $scriptContent -Pattern 'merge-base[^\r\n]+--is-ancestor' -Message 'Product commit ancestry check is missing.'
    Assert-Match -Content $scriptContent -Pattern 'status[^\r\n]+--untracked-files=no' -Message 'Tracked worktree cleanliness check is missing.'
    Assert-Match -Content $scriptContent -Pattern '\.cache/' -Message 'Root cache dockerignore verification is missing.'
    Assert-Match -Content $scriptContent -Pattern 'backend/\.cache/' -Message 'Backend cache dockerignore verification is missing.'
    Assert-Match -Content $scriptContent -Pattern 'GetActiveTcpListeners' -Message 'Non-elevated port check is missing.'
    Assert-NotMatch -Content $scriptContent -Pattern 'Get-NetTCPConnection' -Message 'Port checks must not require elevated CIM access.'
    Assert-Match -Content $scriptContent -Pattern "CanonicalImage\s*=\s*'wujie-sub2api:local'" -Message 'Canonical delivery image is not pinned.'
    Assert-Match -Content $scriptContent -Pattern 'docker\s+build[^\r\n]+--progress=plain' -Message 'Deterministic Docker build command is missing.'
    Assert-Match -Content $scriptContent -Pattern 'docker\s+image\s+inspect' -Message 'Built image identity verification is missing.'
    Assert-NotMatch -Content $scriptContent -Pattern 'compose\s+(up|start)|docker\s+run|\bdeploy\b' -Message 'Preflight must not start or deploy services.'
    Assert-NotMatch -Content $scriptContent -Pattern '\.Config\.Env|Get-Content[^\r\n]*\.env|\bcat\b[^\r\n]*\.env|compose\s+config' -Message 'Preflight must not read or print secrets.'
    Assert-NotMatch -Content $scriptContent -Pattern 'seedance|kling|tiny.real|provider' -Message 'Preflight must not reference real provider execution.'

    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $invalidActionOutput = & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $scriptPath InvalidAction 2>&1
    $invalidActionExitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorActionPreference
    Assert-True -Condition ($invalidActionExitCode -ne 0) -Message 'Unsupported action must fail before any Docker operation.'
    Assert-Match -Content ($invalidActionOutput -join "`n") -Pattern 'ValidateSet|InvalidAction' -Message 'Unsupported action failure must explain parameter validation.'
}

if ($failures.Count -gt 0) {
    Write-Error ("Delivery preflight contract failed:`n- " + ($failures -join "`n- "))
    exit 1
}

Write-Host 'Delivery preflight offline contract passed.'
