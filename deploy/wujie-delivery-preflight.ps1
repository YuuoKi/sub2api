[CmdletBinding()]
param(
    [Parameter(Mandatory, Position = 0)]
    [ValidateSet('Check', 'Build')]
    [string]$Action,

    [Parameter(Mandatory)]
    [string]$RepoRoot,

    [Parameter(Mandatory)]
    [string]$ReleaseRoot,

    [Parameter(Mandatory)]
    [string]$ExpectedBranch,

    [Parameter(Mandatory)]
    [ValidatePattern('^[0-9a-fA-F]{40}$')]
    [string]$RequiredProductCommit,

    [string]$Version = '',
    [string]$Commit = '',
    [string]$Date = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$CanonicalImage = 'wujie-sub2api:local'
$ResolvedRepoRoot = $null
$ResolvedReleaseRoot = $null
$DockerIgnorePath = $null
$ReviewPackagePath = $null

function Resolve-RequiredDirectory {
    param(
        [Parameter(Mandatory)]
        [string]$Path,

        [Parameter(Mandatory)]
        [string]$Label
    )

    if ([string]::IsNullOrWhiteSpace($Path)) {
        throw "$Label is required."
    }
    if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
        throw "$Label does not exist: $Path"
    }
    return (Resolve-Path -LiteralPath $Path).Path
}
$ExpectedReviewStatus = [string](ConvertFrom-Json -InputObject '"\u5f85\u590d\u6838 / \u90e8\u5206\u95e8\u7981\u901a\u8fc7"')

function Invoke-GitText {
    param(
        [Parameter(Mandatory)]
        [string[]]$Arguments
    )

    $value = & git @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Git command failed: git $($Arguments -join ' ')"
    }
    return (($value | Out-String).Trim())
}

function Assert-RepositoryState {
    $topLevel = Invoke-GitText -Arguments @('-C', $ResolvedRepoRoot, 'rev-parse', '--show-toplevel')
    $resolvedTopLevel = (Resolve-Path -LiteralPath $topLevel).Path
    if ($resolvedTopLevel -ne $ResolvedRepoRoot) {
        throw "Wrong repository root: expected $ResolvedRepoRoot, got $resolvedTopLevel."
    }

    $branch = Invoke-GitText -Arguments @('-C', $ResolvedRepoRoot, 'branch', '--show-current')
    if ($branch -ne $ExpectedBranch) {
        throw "Wrong branch: expected $ExpectedBranch, got $branch."
    }

    & git -C $ResolvedRepoRoot merge-base --is-ancestor $RequiredProductCommit HEAD
    if ($LASTEXITCODE -ne 0) {
        throw "Required product commit $RequiredProductCommit is not an ancestor of HEAD."
    }

    $trackedStatus = & git -C $ResolvedRepoRoot status --short --untracked-files=no
    if ($LASTEXITCODE -ne 0) {
        throw 'Unable to inspect tracked worktree state.'
    }
    if (($trackedStatus | Out-String).Trim()) {
        throw 'Tracked worktree is not clean. Commit or revert intended changes before delivery build.'
    }

    return @{
        Branch = $branch
        Head = Invoke-GitText -Arguments @('-C', $ResolvedRepoRoot, 'rev-parse', 'HEAD')
    }
}

function Assert-BuildContextPolicy {
    if (-not (Test-Path -LiteralPath $DockerIgnorePath -PathType Leaf)) {
        throw 'Root .dockerignore is missing.'
    }

    $dockerIgnore = Get-Content -Raw -Encoding UTF8 -LiteralPath $DockerIgnorePath
    foreach ($requiredPattern in @('.cache/', 'backend/.cache/')) {
        if ($dockerIgnore -notmatch "(?m)^$([regex]::Escape($requiredPattern))\s*$") {
            throw "Root .dockerignore is missing required pattern: $requiredPattern"
        }
    }
}

function Assert-ReviewPackage {
    if (-not (Test-Path -LiteralPath $ReviewPackagePath -PathType Leaf)) {
        throw 'Latest review package is missing.'
    }

    $reviewPackage = Get-Content -Raw -Encoding UTF8 -LiteralPath $ReviewPackagePath
    if (-not $reviewPackage.Contains($RequiredProductCommit)) {
        throw 'Latest review package does not reference the required product commit.'
    }
    if (-not $reviewPackage.Contains($ExpectedReviewStatus)) {
        throw 'Latest review package does not retain the required pending-review status.'
    }
}

function Assert-DeliveryPortsFree {
    $listeners = @(
        [System.Net.NetworkInformation.IPGlobalProperties]::GetIPGlobalProperties().GetActiveTcpListeners() |
            Where-Object { $_.Port -in 3000, 8080 }
    )
    if ($listeners) {
        $summary = ($listeners | ForEach-Object { "$($_.Address):$($_.Port)" }) -join ', '
        throw "Delivery ports must be free before the build/runtime handoff: $summary"
    }
}

function Assert-DockerEngine {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        throw 'Docker CLI is unavailable.'
    }

    & docker version *> $null
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker engine is unavailable. Restore WSL/Docker first; no image was built.'
    }
}

function Invoke-DeliveryCheck {
    $repository = Assert-RepositoryState
    Assert-BuildContextPolicy
    Assert-ReviewPackage
    Assert-DeliveryPortsFree
    Assert-DockerEngine

    Write-Host "PASS action=Check branch=$($repository.Branch) head=$($repository.Head) product_commit=$RequiredProductCommit release_root=$ResolvedReleaseRoot review_status=verified ports=free docker=available"
}

function Invoke-DeliveryBuild {
    Invoke-DeliveryCheck

    $head = Invoke-GitText -Arguments @('-C', $ResolvedRepoRoot, 'rev-parse', 'HEAD')
    $buildCommit = if ($Commit) { $Commit.Trim() } else { $head }
    if ($buildCommit -ne $head) {
        throw "Build COMMIT must equal the checked repository HEAD."
    }
    $buildDate = if ($Date) { $Date.Trim() } else { [DateTime]::UtcNow.ToString('o') }
    $buildVersion = if ($Version) { $Version.Trim() } else { 'release-' + $buildCommit.Substring(0, 12) }

    & docker build --progress=plain -t $CanonicalImage `
        --build-arg "VERSION=$buildVersion" `
        --build-arg "COMMIT=$buildCommit" `
        --build-arg "DATE=$buildDate" `
        $ResolvedRepoRoot
    if ($LASTEXITCODE -ne 0) {
        throw "Docker build failed for $CanonicalImage."
    }

    $imageId = & docker image inspect --format '{{.Id}}' $CanonicalImage
    if ($LASTEXITCODE -ne 0) {
        throw "Built image $CanonicalImage cannot be inspected."
    }
    $imageId = (($imageId | Out-String).Trim())
    if (-not $imageId) {
        throw "Built image $CanonicalImage has an empty image ID."
    }

    $manifestPath = Join-Path $ResolvedReleaseRoot 'wujie-sub2api-build-manifest.json'
    [ordered]@{
        image = $CanonicalImage
        imageId = $imageId
        version = $buildVersion
        commit = $buildCommit
        date = $buildDate
        repoRoot = $ResolvedRepoRoot
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $manifestPath -Encoding UTF8
    Write-Host "PASS action=Build image=$CanonicalImage image_id=$imageId commit=$buildCommit release_root=$ResolvedReleaseRoot manifest=$manifestPath"
}

$ResolvedRepoRoot = Resolve-RequiredDirectory -Path $RepoRoot -Label 'RepoRoot'
$ResolvedReleaseRoot = Resolve-RequiredDirectory -Path $ReleaseRoot -Label 'ReleaseRoot'
$DockerIgnorePath = Join-Path $ResolvedRepoRoot '.dockerignore'
$ReviewPackagePath = Join-Path $ResolvedRepoRoot 'docs/reviews/LATEST_REVIEW_PACKAGE.html'

switch ($Action) {
    'Check' {
        Invoke-DeliveryCheck
    }
    'Build' {
        Invoke-DeliveryBuild
    }
}
