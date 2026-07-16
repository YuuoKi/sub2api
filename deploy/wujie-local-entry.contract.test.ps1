[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$scriptPath = Join-Path $PSScriptRoot 'wujie-local-entry.ps1'
$sopPath = Join-Path $PSScriptRoot 'WUJIE_BOSS_OPERATIONS_SOP.md'
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

function ConvertFrom-UnicodeEscape {
    param(
        [Parameter(Mandatory)]
        [string]$Value
    )

    return ConvertFrom-Json -InputObject ('"' + $Value + '"')
}

Assert-True -Condition (Test-Path -LiteralPath $scriptPath -PathType Leaf) -Message 'Missing deploy/wujie-local-entry.ps1.'
Assert-True -Condition (Test-Path -LiteralPath $sopPath -PathType Leaf) -Message 'Missing deploy/WUJIE_BOSS_OPERATIONS_SOP.md.'

if (Test-Path -LiteralPath $scriptPath -PathType Leaf) {
    $scriptContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $scriptPath
    $parseErrors = $null
    [void][System.Management.Automation.Language.Parser]::ParseFile($scriptPath, [ref]$null, [ref]$parseErrors)

    Assert-True -Condition ($parseErrors.Count -eq 0) -Message 'PowerShell entry script has syntax errors.'
    Assert-Match -Content $scriptContent -Pattern "ValidateSet\s*\(\s*'Start'\s*,\s*'Status'\s*,\s*'Stop'\s*\)" -Message 'Action must be restricted to Start, Status, and Stop.'
    Assert-Match -Content $scriptContent -Pattern "wujie-sub2api:local" -Message 'Canonical image is not pinned.'
    Assert-Match -Content $scriptContent -Pattern "wujie-single-entry-sub2api" -Message 'Canonical container is not pinned.'
    Assert-Match -Content $scriptContent -Pattern "127\.0\.0\.1:8080" -Message 'Loopback HTTP endpoint is not pinned.'
    Assert-Match -Content $scriptContent -Pattern "CanonicalTitle.+u65e0.+u754c.+u00b7" -Message 'Wujie title verification is missing.'
    Assert-Match -Content $scriptContent -Pattern "GetActiveTcpListeners" -Message 'Non-elevated port 3000 listener verification is missing.'
    Assert-NotMatch -Content $scriptContent -Pattern "Get-NetTCPConnection" -Message 'Port verification must not require elevated CIM access.'
    Assert-Match -Content $scriptContent -Pattern "HostIp" -Message 'Docker loopback binding verification is missing.'
    Assert-Match -Content $scriptContent -Pattern "HostPort" -Message 'Docker host port verification is missing.'
    Assert-Match -Content $scriptContent -Pattern "CanonicalService\s*=\s*'sub2api'" -Message 'Canonical compose service is not pinned.'
    Assert-Match -Content $scriptContent -Pattern "ComposePath\s*=\s*'deploy/docker-compose\.yml'" -Message 'Compose path must remain relative for the Windows-to-WSL Docker wrapper.'
    Assert-Match -Content $scriptContent -Pattern 'Push-Location\s+-LiteralPath\s+\$RepoRoot' -Message 'Compose commands must run from the repository root.'
    Assert-Match -Content $scriptContent -Pattern "Invoke-CanonicalCompose[^\r\n]+\('up'[^\r\n]+'--no-build'[^\r\n]+'--no-deps'" -Message 'Start must use only the canonical app service without rebuilding dependencies.'
    Assert-Match -Content $scriptContent -Pattern "Invoke-CanonicalCompose[^\r\n]+\('stop'" -Message 'Stop must target only the canonical app service.'
    Assert-Match -Content $scriptContent -Pattern "docker image inspect" -Message 'Status must resolve the exact canonical image ID.'
    Assert-Match -Content $scriptContent -Pattern "\.Image" -Message 'Status must compare the running container image ID.'

    Assert-NotMatch -Content $scriptContent -Pattern "compose\s+config" -Message 'The script must not print expanded compose configuration.'
    Assert-NotMatch -Content $scriptContent -Pattern "\bdown\b" -Message 'The script must not run docker compose down.'
    Assert-NotMatch -Content $scriptContent -Pattern "\.Config\.Env|Get-Content[^\r\n]*\.env|\bcat\b[^\r\n]*\.env" -Message 'The script must not read or print environment secrets.'
    Assert-NotMatch -Content $scriptContent -Pattern "(?:docker\s+compose|compose)[^\r\n]*\s-v(?:\s|$)" -Message 'The script must not use destructive volume flags.'

    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $invalidActionOutput = & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $scriptPath InvalidAction 2>&1
    $invalidActionExitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorActionPreference
    Assert-True -Condition ($invalidActionExitCode -ne 0) -Message 'An unsupported action must fail before any Docker operation.'
    Assert-Match -Content ($invalidActionOutput -join "`n") -Pattern 'ValidateSet|InvalidAction' -Message 'Unsupported action failure must explain parameter validation.'
}

if (Test-Path -LiteralPath $sopPath -PathType Leaf) {
    $sopContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $sopPath
    foreach ($requiredHeading in @(
        (ConvertFrom-UnicodeEscape '\u5458\u5de5\u989d\u5ea6\u6a21\u677f'),
        ((ConvertFrom-UnicodeEscape '\u5bc6\u94a5\u8f6e\u6362\u4eba\u5de5') + ' Gate'),
        (ConvertFrom-UnicodeEscape '\u5931\u8d25\u4efb\u52a1\u5b9a\u4f4d'),
        (ConvertFrom-UnicodeEscape '\u8d39\u7528\u5f02\u5e38\u6838\u5bf9'),
        (ConvertFrom-UnicodeEscape '\u542f\u52a8\u3001\u72b6\u6001\u4e0e\u505c\u6b62'),
        (ConvertFrom-UnicodeEscape '\u5b89\u5168\u56de\u6eda'),
        (ConvertFrom-UnicodeEscape '\u9694\u79bb DEV \u6062\u590d\u6f14\u7ec3')
    )) {
        Assert-Match -Content $sopContent -Pattern ([regex]::Escape($requiredHeading)) -Message "SOP missing section: $requiredHeading"
    }

    Assert-Match -Content $sopContent -Pattern ([regex]::Escape((ConvertFrom-UnicodeEscape '\u4e0d\u5f97\u89e6\u78b0\u751f\u4ea7'))) -Message 'SOP must forbid production access.'
    Assert-Match -Content $sopContent -Pattern ([regex]::Escape((ConvertFrom-UnicodeEscape '\u4e0d\u5f97\u5220\u9664\u5907\u4efd'))) -Message 'SOP must forbid backup deletion.'
    Assert-Match -Content $sopContent -Pattern ([regex]::Escape((ConvertFrom-UnicodeEscape '\u4e0d\u5f97\u53d1\u8d77\u771f\u5b9e\u8c03\u7528'))) -Message 'SOP must forbid real provider calls.'
}

if ($failures.Count -gt 0) {
    Write-Error ("Gate 4A contract failed:`n- " + ($failures -join "`n- "))
    exit 1
}

Write-Host 'Gate 4A offline contract passed.'
