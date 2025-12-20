$ErrorActionPreference = 'Stop'

$packageName = $env:ChocolateyPackageName
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
$url64 = 'https://github.com/robottwo/gsh_prime/releases/download/v0.26.0/gsh_Windows_x86_64.zip'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = $toolsDir
  url64bit      = $url64
  checksum64    = 'PLACEHOLDER_CHECKSUM'
  checksumType64= 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Create shim for gsh.exe
$gshPath = Join-Path $toolsDir 'gsh.exe'
if (Test-Path $gshPath) {
  Write-Host "gsh installed successfully to $gshPath"
}
