# Turns the recorded frames into the README images (Windows, with the Edge that ships with it).
#   1. demo svg  -in FRAMES            (SVG terminal windows)
#   2. Edge headless: SVG -> PNG
#   3. demo gif: PNG sets -> assets/demo.gif and assets/live.gif; screenshots -> assets/screenshots
# usage: pwsh scripts/demo/render.ps1 -Frames DIR -Demo path\to\demo.exe
param([Parameter(Mandatory)] [string]$Frames, [Parameter(Mandatory)] [string]$Demo)
$ErrorActionPreference = "Stop"
$repo = Resolve-Path "$PSScriptRoot\..\.."
& $Demo svg -in $Frames
$edge = "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
$profileDir = Join-Path ([IO.Path]::GetTempPath()) "catude-demo-edge"
foreach ($f in Get-ChildItem $Frames -Filter *.svg) {
  $out = [IO.Path]::ChangeExtension($f.FullName, ".png")
  $uri = ([Uri]$f.FullName).AbsoluteUri
  Start-Process -FilePath $edge -Wait -WindowStyle Hidden -ArgumentList "--headless=new", "--disable-gpu", "--hide-scrollbars",
    "--user-data-dir=$profileDir", "--force-device-scale-factor=1", "--window-size=1224,784", "--screenshot=$out", $uri
}
& $Demo gif -in $Frames -set hero -out "$repo\assets\demo.gif"
& $Demo gif -in $Frames -set live -out "$repo\assets\live.gif"
& $Demo gif -in $Frames -set tour -out "$repo\assets\tour.gif"
foreach ($s in 'quota', 'agents', 'switch') { & $Demo gif -in $Frames -set $s -out "$repo\assets\$s.gif" }
New-Item -ItemType Directory -Force "$repo\assets\screenshots" | Out-Null
foreach ($f in Get-ChildItem $Frames -Filter "cli-*.png") { Copy-Item $f.FullName "$repo\assets\screenshots\$($f.Name)" }
foreach ($f in Get-ChildItem $Frames -Filter "shots-*.png") {
  Copy-Item $f.FullName "$repo\assets\screenshots\$($f.Name -replace '^shots-', '')"
}
# the Mermaid diagram of the demo machine, as `w` in the Graph tab opens it (Mermaid loads from jsDelivr)
$graph = Join-Path $Frames "graph.html"
if (Test-Path $graph) {
  Start-Process -FilePath $edge -Wait -WindowStyle Hidden -ArgumentList "--headless=new", "--disable-gpu", "--hide-scrollbars",
    "--user-data-dir=$profileDir", "--force-device-scale-factor=1", "--virtual-time-budget=8000", "--window-size=1400,1700",
    "--screenshot=$repo\assets\screenshots\mermaid.png", (([Uri]$graph).AbsoluteUri + "#fit")
}
"README images updated in $repo\assets"
