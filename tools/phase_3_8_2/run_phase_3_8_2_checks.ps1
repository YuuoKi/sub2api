param(
  [string]$CustomerZip = "",
  [string]$InternalZip = ""
)

$ErrorActionPreference = "Continue"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$ProjectRoot = Resolve-Path (Join-Path $RepoRoot "..\..")
$ReviewDirName = "03_" + [string][char]0x5ba1 + [string][char]0x67e5 + [string][char]0x5305
$ReviewRoot = Join-Path $ProjectRoot $ReviewDirName
$LogPath = Join-Path $ReviewRoot "PHASE_3_8_2_AUTOMATED_CHECK_LOG.md"
$ScriptRoot = $PSScriptRoot
$results = @()

function Add-Log {
  param([string]$Text)
  Add-Content -LiteralPath $LogPath -Value $Text -Encoding UTF8
}

function Run-Step {
  param(
    [string]$Name,
    [scriptblock]$Command
  )
  Add-Log "## $Name"
  $output = & $Command 2>&1
  $code = $LASTEXITCODE
  if ($null -eq $code) { $code = 0 }
  $text = ($output | Out-String).TrimEnd()
  if ($text) {
    Add-Log '```text'
    Add-Log $text
    Add-Log '```'
  }
  if ($code -eq 0) {
    Add-Log ""
    Add-Log "Result: PASS"
    $script:results += [pscustomobject]@{ Name = $Name; Result = "PASS" }
  } else {
    Add-Log ""
    Add-Log "Result: FAIL ($code)"
    $script:results += [pscustomobject]@{ Name = $Name; Result = "FAIL" }
  }
  Add-Log ""
}

New-Item -ItemType Directory -Force -Path $ReviewRoot | Out-Null
Set-Content -LiteralPath $LogPath -Encoding UTF8 -Value "# Phase 3.8.2 Automated Check Log`n`nRun time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss zzz')`n"

Push-Location $RepoRoot
try {
  Run-Step "customer package scrub" { node (Join-Path $ScriptRoot "check_customer_package_scrub.js") --review $ReviewRoot }
  Run-Step "review link check" { node (Join-Path $ScriptRoot "check_review_links.js") --review $ReviewRoot }
  Run-Step "manifest integrity" { node (Join-Path $ScriptRoot "check_manifest_integrity.js") --review $ReviewRoot }
  Run-Step "git diff --check" { git diff --check }
  Run-Step "frontend build" { corepack pnpm --dir frontend run build }
  Run-Step "demo build" {
    $old = $env:VITE_PRODUCT_MODE
    $env:VITE_PRODUCT_MODE = "video_gateway_demo"
    corepack pnpm --dir frontend run build
    $env:VITE_PRODUCT_MODE = $old
  }
  if ($CustomerZip) {
    Run-Step "customer zip forbidden files" { node (Join-Path $ScriptRoot "check_forbidden_files_in_zip.js") --customer $CustomerZip }
  }
  if ($InternalZip) {
    Run-Step "internal zip forbidden files" { node (Join-Path $ScriptRoot "check_forbidden_files_in_zip.js") $InternalZip }
  }
} finally {
  Pop-Location
}

Add-Log "## Summary"
foreach ($item in $results) {
  Add-Log "- $($item.Name): $($item.Result)"
}

if ($results.Result -contains "FAIL") {
  Add-Log ""
  Add-Log "Overall: FAIL"
  exit 1
}

Add-Log ""
Add-Log "Overall: PASS"
exit 0
