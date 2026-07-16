[CmdletBinding(SupportsShouldProcess = $true, ConfirmImpact = 'High')]
param(
    [Parameter(Mandatory, Position = 0)]
    [string]$BackupPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$PostgresImage = 'postgres:18-alpine'
$ExerciseUser = 'sub2api'
$ExercisePassword = 'restore-drill-only-not-for-production'
$ContainerBackupPath = '/tmp/wujie-restore-input.sql.gz'
$CoreTables = @('users', 'groups', 'accounts', 'api_keys', 'usage_logs')

if (-not (Test-Path -LiteralPath $BackupPath -PathType Leaf)) {
    Write-Error 'BackupPath must reference an existing file.'
    exit 2
}

$resolvedBackup = (Resolve-Path -LiteralPath $BackupPath -ErrorAction Stop).Path
if ([System.IO.Path]::GetFileName($resolvedBackup) -notmatch '\.sql\.gz$') {
    Write-Error 'BackupPath must use the repository backup format: *.sql.gz.'
    exit 2
}

$backupItem = Get-Item -LiteralPath $resolvedBackup -ErrorAction Stop
$backupHash = (Get-FileHash -LiteralPath $resolvedBackup -Algorithm SHA256 -ErrorAction Stop).Hash.ToLowerInvariant()
$backupSize = [int64]$backupItem.Length

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Error 'Docker CLI is unavailable.'
    exit 3
}

& docker image inspect $PostgresImage *> $null
if ($LASTEXITCODE -ne 0) {
    Write-Error "Required image $PostgresImage is unavailable. Pull it explicitly before the drill."
    exit 3
}

$suffix = [guid]::NewGuid().ToString('N').Substring(0, 12)
$ContainerName = "wujie-restore-drill-$suffix"
$VolumeName = "wujie-restore-drill-data-$suffix"
$ExerciseDatabase = "wujie_restore_drill_$suffix"
$volumeCreated = $false
$containerCreated = $false
$stage = 'approval'

function Write-PreservationActions {
    if ($containerCreated) {
        Write-Output "stop_action=docker stop $ContainerName"
        Write-Output "preserve_action=leave container $ContainerName and volume $VolumeName unchanged for inspection"
        return
    }

    Write-Output 'stop_action=not_applicable'
    if ($volumeCreated) {
        Write-Output "preserve_action=leave volume $VolumeName unchanged for inspection"
    } else {
        Write-Output 'preserve_action=no Docker resource was created'
    }
}

if (-not $PSCmdlet.ShouldProcess(
    'a new uniquely named PostgreSQL container, volume, and database',
    'restore the selected backup into an isolated DEV drill target'
)) {
    Write-Output "backup_sha256=$backupHash"
    Write-Output "backup_size_bytes=$backupSize"
    Write-Output 'restore_status=cancelled'
    exit 0
}

try {
    $stage = 'create_volume'
    $null = & docker volume create $VolumeName 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker could not create the isolated volume.'
    }
    $volumeCreated = $true

    $stage = 'create_container'
    & docker run --detach `
        --name $ContainerName `
        --publish '127.0.0.1::5432' `
        --mount "type=volume,source=$VolumeName,target=/var/lib/postgresql" `
        --env "POSTGRES_USER=$ExerciseUser" `
        --env "POSTGRES_PASSWORD=$ExercisePassword" `
        --env "POSTGRES_DB=$ExerciseDatabase" `
        --env 'PGDATA=/var/lib/postgresql/data' `
        $PostgresImage *> $null
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker could not start the isolated PostgreSQL container.'
    }
    $containerCreated = $true

    $stage = 'verify_isolation'
    $portBindingsJson = (& docker inspect '--format={{json .HostConfig.PortBindings}}' $ContainerName 2>$null | Out-String).Trim()
    $portBindings = ConvertFrom-Json -InputObject $portBindingsJson -ErrorAction Stop
    $postgresBindings = @($portBindings.'5432/tcp')
    $hostIp = if ($postgresBindings.Count -gt 0) { [string]$postgresBindings[0].HostIp } else { '' }
    if ($LASTEXITCODE -ne 0 -or $hostIp -ne '127.0.0.1') {
        throw 'The isolated PostgreSQL port is not restricted to loopback.'
    }

    $mountsJson = (& docker inspect '--format={{json .Mounts}}' $ContainerName 2>$null | Out-String).Trim()
    $mounts = @(ConvertFrom-Json -InputObject $mountsJson -ErrorAction Stop)
    $mountedVolume = ''
    foreach ($mount in $mounts) {
        if ([string]$mount.Destination -eq '/var/lib/postgresql') {
            $mountedVolume = [string]$mount.Name
            break
        }
    }
    if ($LASTEXITCODE -ne 0 -or $mountedVolume -ne $VolumeName) {
        throw 'The isolated PostgreSQL data mount does not use the unique drill volume.'
    }

    $stage = 'wait_for_postgres'
    $ready = $false
    for ($attempt = 1; $attempt -le 60; $attempt++) {
        & docker exec $ContainerName pg_isready --username $ExerciseUser --dbname $ExerciseDatabase *> $null
        if ($LASTEXITCODE -eq 0) {
            $ready = $true
            break
        }
        Start-Sleep -Seconds 1
    }
    if (-not $ready) {
        throw 'The isolated PostgreSQL container did not become ready within 60 seconds.'
    }

    $stage = 'copy_backup'
    Push-Location -LiteralPath $backupItem.DirectoryName
    try {
        $relativeBackup = "./$($backupItem.Name)"
        & docker cp -- $relativeBackup "${ContainerName}:$ContainerBackupPath" *> $null
        if ($LASTEXITCODE -ne 0) {
            throw 'Docker could not copy the selected backup into the isolated container.'
        }
    } finally {
        Pop-Location
    }

    $stage = 'restore_backup'
    $restoreCommand = 'gzip -dc "$1" | psql --set=ON_ERROR_STOP=1 --single-transaction --username="$2" --dbname="$3"'
    & docker exec $ContainerName sh -ceu $restoreCommand restore-drill $ContainerBackupPath $ExerciseUser $ExerciseDatabase *> $null
    if ($LASTEXITCODE -ne 0) {
        throw 'psql rejected the backup; SQL output was suppressed to avoid exposing restored data.'
    }

    $stage = 'verify_counts'
    $countQuery = @"
SELECT json_build_object(
  'users', (SELECT COUNT(*) FROM users),
  'groups', (SELECT COUNT(*) FROM groups),
  'accounts', (SELECT COUNT(*) FROM accounts),
  'api_keys', (SELECT COUNT(*) FROM api_keys),
  'usage_logs', (SELECT COUNT(*) FROM usage_logs)
)::text;
"@
    $countOutput = & docker exec $ContainerName psql `
        --username $ExerciseUser `
        --dbname $ExerciseDatabase `
        --tuples-only `
        --no-align `
        --set ON_ERROR_STOP=1 `
        --command $countQuery 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw 'Core table count verification failed.'
    }

    $countJson = (($countOutput | Out-String).Trim())
    try {
        $counts = ConvertFrom-Json -InputObject $countJson -ErrorAction Stop
    } catch {
        throw 'Core table count verification returned an invalid result.'
    }
    foreach ($table in $CoreTables) {
        $countValue = $counts.$table
        if ($null -eq $countValue -or [int64]$countValue -lt 0) {
            throw "Core table count verification is missing table $table."
        }
    }

    Write-Output "backup_sha256=$backupHash"
    Write-Output "backup_size_bytes=$backupSize"
    Write-Output 'restore_status=passed'
    Write-Output 'loopback_status=verified'
    Write-Output "table_counts=$countJson"
    Write-Output "isolation_container=$ContainerName"
    Write-Output "isolation_volume=$VolumeName"
    Write-Output "isolation_database=$ExerciseDatabase"
    Write-PreservationActions
} catch {
    Write-Output "backup_sha256=$backupHash"
    Write-Output "backup_size_bytes=$backupSize"
    Write-Output 'restore_status=failed'
    Write-Output "failed_stage=$stage"
    Write-PreservationActions
    Write-Error "Isolated DEV restore drill failed at stage '$stage': $($_.Exception.Message)"
    exit 1
}
