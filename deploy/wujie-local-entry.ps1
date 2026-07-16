[CmdletBinding()]
param(
    [Parameter(Mandatory, Position = 0)]
    [ValidateSet('Start', 'Status', 'Stop')]
    [string]$Action
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$CanonicalImage = 'wujie-sub2api:local'
$CanonicalContainer = 'wujie-single-entry-sub2api'
$CanonicalService = 'sub2api'
$CanonicalEndpoint = 'http://127.0.0.1:8080/'
$CanonicalTitle = ConvertFrom-Json -InputObject '"\u65e0\u754c \u00b7 \u4f01\u4e1a AI \u7ba1\u7406\u4e2d\u53f0"'
$RepoRoot = Split-Path -Parent $PSScriptRoot
$ComposePath = 'deploy/docker-compose.yml'

function Invoke-CanonicalCompose {
    param(
        [Parameter(Mandatory)]
        [string[]]$Arguments
    )

    Push-Location -LiteralPath $RepoRoot
    try {
        & docker compose -f $ComposePath @Arguments
        return $LASTEXITCODE
    } finally {
        Pop-Location
    }
}

function Assert-DockerAvailable {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        throw 'Docker CLI is unavailable.'
    }
}

function Assert-CanonicalImageAvailable {
    & docker image inspect --format '{{.Id}}' $CanonicalImage *> $null
    if ($LASTEXITCODE -ne 0) {
        throw "Canonical image $CanonicalImage is unavailable. Build it explicitly before Start."
    }
}

function Get-CanonicalImageId {
    $imageId = & docker image inspect --format '{{.Id}}' $CanonicalImage 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Canonical image $CanonicalImage is unavailable."
    }
    return (($imageId | Out-String).Trim())
}

function Get-ContainerField {
    param(
        [Parameter(Mandatory)]
        [string]$Format,

        [switch]$AllowMissing
    )

    # Keep Go-template string literals intact when Docker is reached through docker.cmd.
    $formatArgument = "--format=$Format"
    $value = & docker inspect $formatArgument $CanonicalContainer 2>$null
    if ($LASTEXITCODE -ne 0) {
        if ($AllowMissing) {
            return $null
        }
        throw "Canonical container $CanonicalContainer is unavailable."
    }

    return (($value | Out-String).Trim())
}

function Assert-Port3000Free {
    $listeners = @(
        [System.Net.NetworkInformation.IPGlobalProperties]::GetIPGlobalProperties().GetActiveTcpListeners() |
            Where-Object { $_.Port -eq 3000 }
    )
    if ($listeners) {
        throw 'Host port 3000 has an active listener. Stop the confirmed development service before using the canonical entry.'
    }
}

function Assert-CanonicalRuntime {
    Assert-Port3000Free

    $running = Get-ContainerField -Format '{{.State.Running}}'
    if ($running -ne 'true') {
        throw "Canonical container $CanonicalContainer is not running."
    }

    $image = Get-ContainerField -Format '{{.Config.Image}}'
    if ($image -ne $CanonicalImage) {
        throw "Canonical container image mismatch: expected $CanonicalImage, got $image."
    }

    $expectedImageId = Get-CanonicalImageId
    $containerImageId = Get-ContainerField -Format '{{.Image}}'
    if ($containerImageId -ne $expectedImageId) {
        throw "Canonical container image ID mismatch: expected $expectedImageId, got $containerImageId."
    }

    $bindingsJson = Get-ContainerField -Format '{{json .HostConfig.PortBindings}}'
    $bindings = ConvertFrom-Json -InputObject $bindingsJson
    $portBindings = @($bindings.'8080/tcp')
    $binding = if ($portBindings.Count -gt 0) {
        "$($portBindings[0].HostIp):$($portBindings[0].HostPort)"
    } else {
        ''
    }
    if ($binding -ne '127.0.0.1:8080') {
        throw "Canonical binding mismatch: expected 127.0.0.1:8080, got $binding."
    }

    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $CanonicalEndpoint -TimeoutSec 5
    } catch {
        throw "Canonical HTTP check failed: $($_.Exception.Message)"
    }

    if ([int]$response.StatusCode -ne 200) {
        throw "Canonical HTTP check returned status $($response.StatusCode), expected 200."
    }
    if ($response.Content -notmatch [regex]::Escape($CanonicalTitle)) {
        throw 'Canonical HTTP response does not contain the Wujie title.'
    }

    Write-Host "PASS image=$CanonicalImage image_id=$expectedImageId container=$CanonicalContainer endpoint=$CanonicalEndpoint title=verified port3000=free"
}

function Start-CanonicalRuntime {
    Assert-Port3000Free
    Assert-CanonicalImageAvailable

    $composeExitCode = Invoke-CanonicalCompose -Arguments @('up', '-d', '--no-build', '--no-deps', $CanonicalService)
    if ($composeExitCode -ne 0) {
        throw 'Canonical compose start failed.'
    }

    $lastFailure = 'runtime verification did not run'
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        try {
            Assert-CanonicalRuntime
            return
        } catch {
            $lastFailure = $_.Exception.Message
            if ($attempt -lt 30) {
                Start-Sleep -Seconds 1
            }
        }
    }

    throw "Canonical runtime did not become ready: $lastFailure"
}

function Stop-CanonicalRuntime {
    $composeExitCode = Invoke-CanonicalCompose -Arguments @('stop', $CanonicalService)
    if ($composeExitCode -ne 0) {
        throw 'Canonical compose stop failed.'
    }

    $running = Get-ContainerField -Format '{{.State.Running}}' -AllowMissing
    if ($running -eq 'true') {
        throw "Canonical container $CanonicalContainer is still running after Stop."
    }

    Write-Host "PASS container=$CanonicalContainer state=stopped volumes=untouched"
}

Assert-DockerAvailable

switch ($Action) {
    'Start' {
        Start-CanonicalRuntime
    }
    'Status' {
        Assert-CanonicalRuntime
    }
    'Stop' {
        Stop-CanonicalRuntime
    }
}
