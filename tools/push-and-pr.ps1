# Push branch and open PR for Wei-Shaw/sub2api
# Run once after: gh auth login
# Usage: powershell -File tools/push-and-pr.ps1

$ErrorActionPreference = 'Stop'
$Branch = 'wujie/video-capture-moat-20260702'

Set-Location (Join-Path $PSScriptRoot '..')

Write-Host "Checking gh auth..."
gh auth status

Write-Host "Pushing $Branch..."
git push -u origin $Branch

$body = @'
## Summary
- Boss console v2: overview, key vault, members/cards, task records with unified spend
- P0-4 channel health alerts on overview linking to key vault
- Backend: Seedance CNY billing with balance deduction, video content[] contract, R1 reconciliation (migrations 149-154)
- API-key Seedance production path without trial_mode when production_authorized
- API contracts and Codex handoff (V-1 through R2 production verify task)

## Test plan
- [x] Browser walkthrough on http://127.0.0.1:18081 (mock path)
- [x] go test + golangci-lint, vue-tsc + eslint + critical vitest
- [ ] Codex R2-A: one real Seedance production smoke (post-merge)

## Notes
Branch includes prior moat/video-gateway work. Focus review on commits from console v2 + R1 if diff is large.
'@

gh pr create --base main --head $Branch `
  --title "feat: console v2, billing reconciliation, and video API contract (R1)" `
  --body $body

Write-Host "Done."
