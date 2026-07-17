[CmdletBinding()]
param(
    [Parameter(Mandatory, Position = 0)]
    [ValidateSet('Check', 'Build')]
    [string]$Action
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ExpectedWorktree = 'D:\sub2api-trunk\.worktrees\console-unification'
$ExpectedBranch = 'codex/wujie-console-unification-20260717'
$RequiredProductCommit = '4c6111502eb59e83e2c5d750a2a724aaf1f70b55'
$CanonicalImage = 'wujie-sub2api:local'
$RepoRoot = Split-Path -Parent $PSScriptRoot
$DockerIgnorePath = Join-Path $RepoRoot '.dockerignore'
$ReviewPackagePath = Join-Path $RepoRoot 'docs/reviews/LATEST_REVIEW_PACKAGE.html'
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
    $topLevel = Invoke-GitText -Arguments @('rev-parse', '--show-toplevel')
    $resolvedTopLevel = (Resolve-Path -LiteralPath $topLevel).Path
    $resolvedRepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
    $resolvedExpectedWorktree = (Resolve-Path -LiteralPath $ExpectedWorktree).Path
    if ($resolvedRepoRoot -ne $resolvedExpectedWorktree) {
        throw "Wrong delivery worktree: expected $resolvedExpectedWorktree, got $resolvedRepoRoot."
    }
    if ($resolvedTopLevel -ne $resolvedRepoRoot) {
        throw "Wrong worktree: expected $resolvedRepoRoot, got $resolvedTopLevel."
    }

    $branch = Invoke-GitText -Arguments @('branch', '--show-current')
    if ($branch -ne $ExpectedBranch) {
        throw "Wrong branch: expected $ExpectedBranch, got $branch."
    }

    & git merge-base --is-ancestor $RequiredProductCommit HEAD
    if ($LASTEXITCODE -ne 0) {
        throw "Required product commit $RequiredProductCommit is not an ancestor of HEAD."
    }

    $trackedStatus = & git status --short --untracked-files=no
    if ($LASTEXITCODE -ne 0) {
        throw 'Unable to inspect tracked worktree state.'
    }
    if (($trackedStatus | Out-String).Trim()) {
        throw 'Tracked worktree is not clean. Commit or revert intended changes before delivery build.'
    }

    return @{
        Branch = $branch
        Head = Invoke-GitText -Arguments @('rev-parse', 'HEAD')
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
    Push-Location -LiteralPath $RepoRoot
    try {
        $repository = Assert-RepositoryState
        Assert-BuildContextPolicy
        Assert-ReviewPackage
        Assert-DeliveryPortsFree
        Assert-DockerEngine

        Write-Host "PASS action=Check branch=$($repository.Branch) head=$($repository.Head) product_commit=$RequiredProductCommit review_status=verified ports=free docker=available"
    } finally {
        Pop-Location
    }
}

function Invoke-DeliveryBuild {
    Invoke-DeliveryCheck

    Push-Location -LiteralPath $RepoRoot
    try {
        & docker build --progress=plain -t $CanonicalImage .
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

        Write-Host "PASS action=Build image=$CanonicalImage image_id=$imageId"
    } finally {
        Pop-Location
    }
}

switch ($Action) {
    'Check' {
        Invoke-DeliveryCheck
    }
    'Build' {
        Invoke-DeliveryBuild
    }
}
