[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$scriptPath = Join-Path $PSScriptRoot 'wujie-isolated-restore-drill.ps1'
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

Assert-True -Condition (Test-Path -LiteralPath $scriptPath -PathType Leaf) -Message 'Missing deploy/wujie-isolated-restore-drill.ps1.'

if (Test-Path -LiteralPath $scriptPath -PathType Leaf) {
    $scriptContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $scriptPath
    $parseErrors = $null
    [void][System.Management.Automation.Language.Parser]::ParseFile($scriptPath, [ref]$null, [ref]$parseErrors)

    Assert-True -Condition ($parseErrors.Count -eq 0) -Message 'PowerShell restore drill script has syntax errors.'
    Assert-Match -Content $scriptContent -Pattern 'Parameter\s*\(\s*Mandatory' -Message 'BackupPath must be mandatory.'
    Assert-Match -Content $scriptContent -Pattern '\[string\]\s*\$BackupPath' -Message 'BackupPath parameter is missing.'
    Assert-Match -Content $scriptContent -Pattern '\.sql\\?\.gz' -Message 'Only the real *.sql.gz backup format may be accepted.'
    Assert-Match -Content $scriptContent -Pattern 'Test-Path.+PathType\s+Leaf' -Message 'BackupPath must be an existing file.'
    Assert-Match -Content $scriptContent -Pattern 'Get-FileHash.+SHA256' -Message 'Backup SHA-256 calculation is missing.'
    Assert-Match -Content $scriptContent -Pattern '\.Length' -Message 'Backup size calculation is missing.'
    Assert-Match -Content $scriptContent -Pattern '\[guid\]::NewGuid' -Message 'Unique isolation suffix is missing.'
    Assert-Match -Content $scriptContent -Pattern 'postgres:18-alpine' -Message 'Restore image must match the repository PostgreSQL major version.'
    Assert-Match -Content $scriptContent -Pattern 'docker.+volume.+create' -Message 'Unique Docker volume creation is missing.'
    Assert-Match -Content $scriptContent -Pattern 'docker.+run.+--detach' -Message 'Unique isolated Docker container creation is missing.'
    Assert-Match -Content $scriptContent -Pattern '127\.0\.0\.1::5432' -Message 'PostgreSQL must bind only to a random loopback port.'
    Assert-Match -Content $scriptContent -Pattern 'type=volume.+target=/var/lib/postgresql' -Message 'PostgreSQL 18 root data directory must use the unique named volume.'
    Assert-Match -Content $scriptContent -Pattern 'PGDATA=/var/lib/postgresql/data' -Message 'PostgreSQL 18 must write into the named drill volume instead of its default anonymous volume.'
    Assert-Match -Content $scriptContent -Pattern 'POSTGRES_PASSWORD' -Message 'The PostgreSQL image must receive the fixed drill-only credential.'
    Assert-Match -Content $scriptContent -Pattern 'gzip\s+-dc' -Message 'The real gzip backup restore path is missing.'
    Assert-Match -Content $scriptContent -Pattern 'psql.+ON_ERROR_STOP=1.+single-transaction' -Message 'Restore must fail atomically on SQL errors.'
    foreach ($table in @('users', 'groups', 'accounts', 'api_keys', 'usage_logs')) {
        Assert-Match -Content $scriptContent -Pattern ("COUNT\(\*\).+FROM\s+" + [regex]::Escape($table)) -Message "Missing safe count check for table $table."
    }
    Assert-Match -Content $scriptContent -Pattern 'restore_status=' -Message 'Restore status output is missing.'
    Assert-Match -Content $scriptContent -Pattern 'backup_sha256=' -Message 'Backup hash output is missing.'
    Assert-Match -Content $scriptContent -Pattern 'table_counts=' -Message 'Safe table count output is missing.'
    Assert-Match -Content $scriptContent -Pattern 'stop_action=' -Message 'Failure/success stop guidance is missing.'
    Assert-Match -Content $scriptContent -Pattern 'preserve_action=' -Message 'Failure/success evidence preservation guidance is missing.'

    Assert-NotMatch -Content $scriptContent -Pattern 'compose\s+config|Get-Content[^\r\n]*\.env|\bcat\b[^\r\n]*\.env' -Message 'The drill must not read or expand secret configuration.'
    Assert-NotMatch -Content $scriptContent -Pattern 'docker\s+(?:container\s+)?rm|docker\s+volume\s+rm|\bprune\b|\bdown\b|Remove-Item' -Message 'The drill must never auto-delete containers, volumes, or backups.'
    Assert-NotMatch -Content $scriptContent -Pattern 'SELECT\s+\*' -Message 'The drill must never output restored rows.'
    Assert-NotMatch -Content $scriptContent -Pattern 'Write-(?:Host|Output|Information|Warning)[^\r\n]*(?:ExercisePassword|POSTGRES_PASSWORD)' -Message 'The fixed drill-only credential must never be printed.'
    Assert-NotMatch -Content $scriptContent -Pattern 'wujie-single-entry-sub2api|sub2api-postgres(?:-dev)?' -Message 'The drill must not reference canonical containers.'

    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'

    $missingOutput = & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $scriptPath -BackupPath (Join-Path $PSScriptRoot 'missing.sql.gz') 2>&1
    $missingExitCode = $LASTEXITCODE
    Assert-True -Condition ($missingExitCode -ne 0) -Message 'A missing backup must fail before Docker operations.'
    Assert-Match -Content ($missingOutput -join "`n") -Pattern 'existing file' -Message 'Missing backup failure must explain the file requirement.'

    $wrongTypeOutput = & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $scriptPath -BackupPath $PSCommandPath 2>&1
    $wrongTypeExitCode = $LASTEXITCODE
    Assert-True -Condition ($wrongTypeExitCode -ne 0) -Message 'An existing non-*.sql.gz file must be rejected before Docker operations.'
    Assert-Match -Content ($wrongTypeOutput -join "`n") -Pattern '\.sql\.gz' -Message 'Wrong backup format failure must name the accepted format.'

    $ErrorActionPreference = $previousErrorActionPreference
}

if ($failures.Count -gt 0) {
    Write-Error ("Gate 4 isolated restore contract failed:`n- " + ($failures -join "`n- "))
    exit 1
}

Write-Host 'Gate 4 isolated restore offline contract passed.'
