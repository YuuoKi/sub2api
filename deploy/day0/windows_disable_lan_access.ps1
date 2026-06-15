$ErrorActionPreference = "Stop"

$ruleName = "Wujie Internal Trial 8080"
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
  Write-Host "Run PowerShell as Administrator, then execute:"
  Write-Host "Remove-NetFirewallRule -DisplayName `"$ruleName`""
  Write-Host "netsh interface portproxy delete v4tov4 listenaddress=0.0.0.0 listenport=8080"
  exit 1
}

Remove-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
netsh interface portproxy delete v4tov4 listenaddress=0.0.0.0 listenport=8080 | Out-Null

Write-Host "LAN access rule and optional port forwarding were removed."
