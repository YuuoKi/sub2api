# Push branch and open PR for Wei-Shaw/sub2api
# Run once after: gh auth login
# Usage: powershell -File tools/push-and-pr.ps1
#
# If origin push is denied (wrong GitHub account), this script pushes to fork
# remote `fork` (YuuoKi/sub2api) and opens PR with --head YuuoKi:<branch>.

$ErrorActionPreference = 'Stop'
$Branch = 'wujie/video-capture-moat-20260702'

Set-Location (Join-Path $PSScriptRoot '..')

Write-Host "Checking gh auth..."
gh auth status

$remote = 'origin'
$prHead = $Branch
Write-Host "Pushing $Branch to $remote..."
try {
    git push -u $remote $Branch
} catch {
    Write-Host "origin push failed; using fork remote..."
    if (-not (git remote get-url fork 2>$null)) {
        git remote add fork https://github.com/YuuoKi/sub2api.git
    }
    $remote = 'fork'
    $prHead = "YuuoKi:$Branch"
    git push -u $remote $Branch
}

$body = @'
## Summary
- Boss console v2: overview, key vault, members/cards, task records with unified spend
- P0-4 channel health alerts on overview linking to key vault
- Backend: Seedance CNY billing with balance deduction, video content[] contract, R1 reconciliation (migrations 149-154)
- API-key Seedance production path without trial_mode when production_authorized
- API contracts and Codex handoff (V-1 through R2); R2-A smoke scripts + blocked review (needs SEEDANCE_API_KEY)

## Test plan
- [x] Browser walkthrough on http://127.0.0.1:18081 (mock path)
- [x] go test + golangci-lint, vue-tsc + eslint + critical vitest
- [ ] R2-A: one real Seedance production smoke after SEEDANCE_API_KEY in dev env

## Notes
Branch includes prior moat/video-gateway work. Focus review on commits from console v2 + R1 if diff is large.
'@

$existing = gh pr view --repo Wei-Shaw/sub2api --head $prHead --json number -q .number 2>$null
if ($existing) {
    Write-Host "PR already exists: https://github.com/Wei-Shaw/sub2api/pull/$existing"
} else {
    gh pr create --repo Wei-Shaw/sub2api --base main --head $prHead `
        --title "feat: console v2, billing reconciliation, and video API contract (R1)" `
        --body $body
}

Write-Host "Done."
