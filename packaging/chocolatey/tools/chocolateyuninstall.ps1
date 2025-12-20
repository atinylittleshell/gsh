$ErrorActionPreference = 'Stop'

$packageName = $env:ChocolateyPackageName
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"

# Remove the gsh executable
$gshPath = Join-Path $toolsDir 'gsh.exe'
if (Test-Path $gshPath) {
  Remove-Item $gshPath -Force
  Write-Host "gsh uninstalled successfully"
}
