# catude installer for Windows (no administrator rights needed).
# Downloads the latest release, verifies its checksum and runs `catude install`, which puts catude and ct
# on your user PATH (pre-selected) so they work in cmd, PowerShell and Windows Terminal right away.
#   Review it first, then:  powershell -ExecutionPolicy Bypass -File install.ps1
param([string]$Repo = "catude/catude", [switch]$Yes)
$ErrorActionPreference = "Stop"
$arch = if ([Environment]::Is64BitOperatingSystem) { if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" } } else { throw "32-bit Windows is not supported" }
$rel = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "catude-installer" }
$asset = $rel.assets | Where-Object { $_.name -like "*windows_$arch.zip" } | Select-Object -First 1
$sums = $rel.assets | Where-Object { $_.name -eq "checksums.txt" } | Select-Object -First 1
if (-not $asset -or -not $sums) { throw "no Windows $arch build in $($rel.tag_name)" }
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("catude-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
$zip = Join-Path $tmp $asset.name
Invoke-WebRequest $asset.browser_download_url -OutFile $zip -UseBasicParsing
$expected = ((Invoke-WebRequest $sums.browser_download_url -UseBasicParsing).Content -split "`n" | Where-Object { $_ -match [regex]::Escape($asset.name) }) -split '\s+' | Select-Object -First 1
$actual = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected.ToLower()) { throw "checksum mismatch: refusing to install" }
Expand-Archive $zip -DestinationPath $tmp
$exe = Get-ChildItem $tmp -Recurse -Filter catude.exe | Select-Object -First 1
if ($Yes) { & $exe.FullName install --yes } else { & $exe.FullName install }
Remove-Item $tmp -Recurse -Force
