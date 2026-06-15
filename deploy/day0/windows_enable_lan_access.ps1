$ErrorActionPreference = "Stop"

$ruleName = "Wujie Internal Trial 8080"
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
  Write-Host "Run PowerShell as Administrator, then execute:"
  Write-Host "New-NetFirewallRule -DisplayName `"$ruleName`" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8080 -Profile Private,Domain"
  Write-Host ""
  Write-Host "Rollback command:"
  Write-Host "Remove-NetFirewallRule -DisplayName `"$ruleName`""
  exit 1
}

$existingRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
if (-not $existingRule) {
  New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8080 -Profile Private,Domain | Out-Null
}

$listening = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue |
  Where-Object { $_.LocalAddress -eq "0.0.0.0" -or $_.LocalAddress -eq "::" -or $_.LocalAddress -eq "127.0.0.1" }

if ($listening) {
  Write-Host "Firewall rule is enabled. Port 8080 is already listening on this Windows host."
  exit 0
}

$wslIp = $null
try {
  $wslOutput = (wsl.exe hostname -I 2>$null)
  if ($wslOutput) {
    $wslIp = ($wslOutput.Trim() -split "\s+")[0]
  }
} catch {
  $wslIp = $null
}

if ($wslIp) {
  netsh interface portproxy delete v4tov4 listenaddress=0.0.0.0 listenport=8080 | Out-Null
  netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=8080 connectaddress=$wslIp connectport=8080 | Out-Null
  Write-Host "Firewall rule is enabled and port forwarding points to WSL ${wslIp}:8080."
} else {
  Write-Host "Firewall rule is enabled. No WSL IP was detected, so port forwarding was skipped."
}
