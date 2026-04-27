param(
    [string]$PortableExe = (Join-Path $PSScriptRoot "..\dist\portable.exe"),
    [string]$PortableStageRoot = (Join-Path $PSScriptRoot "..\dist\portable_stage"),
    [string]$FirefoxExecutable = "C:\Program Files\Mozilla Firefox\firefox.exe"
)

$ErrorActionPreference = 'Stop'

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Force -Path $Path | Out-Null
    }
}

function Copy-File([string]$Source, [string]$Destination) {
    Ensure-Dir (Split-Path -Parent $Destination)
    Copy-Item -LiteralPath $Source -Destination $Destination -Force
}

function Copy-IfExists([string]$Source, [string]$Destination) {
    if (Test-Path -LiteralPath $Source) {
        Copy-File $Source $Destination
    }
}

function New-AppIconFromImage([string]$SourceImage, [string]$DestinationIco) {
    if (-not (Test-Path -LiteralPath $SourceImage)) {
        throw "icon source image not found: $SourceImage"
    }
    Add-Type -AssemblyName System.Drawing
    $img = [System.Drawing.Image]::FromFile($SourceImage)
    try {
        $cropW = [Math]::Min($img.Width, [int]($img.Height / 2))
        if ($cropW -le 0) {
            throw "invalid crop size for icon source image: $SourceImage"
        }
        $cropH = $cropW
        $x = [int](($img.Width - $cropW) / 2)
        $y = 0
        $crop = New-Object System.Drawing.Bitmap $cropW, $cropH
        $g = [System.Drawing.Graphics]::FromImage($crop)
        try {
            $g.DrawImage(
                $img,
                [System.Drawing.Rectangle]::new(0, 0, $cropW, $cropH),
                $x, $y, $cropW, $cropH,
                [System.Drawing.GraphicsUnit]::Pixel
            )
        } finally {
            $g.Dispose()
        }
        $thumb = New-Object System.Drawing.Bitmap 256, 256
        $g2 = [System.Drawing.Graphics]::FromImage($thumb)
        try {
            $g2.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
            $g2.DrawImage($crop, 0, 0, 256, 256)
        } finally {
            $g2.Dispose()
        }
        try {
            $icon = [System.Drawing.Icon]::FromHandle($thumb.GetHicon())
            $fs = [System.IO.File]::Open($DestinationIco, [System.IO.FileMode]::Create)
            try {
                $icon.Save($fs)
            } finally {
                $fs.Dispose()
            }
        } finally {
            $crop.Dispose()
            $thumb.Dispose()
        }
    } finally {
        $img.Dispose()
    }
}

function Resolve-FullPath([string]$Path, [string]$BasePath) {
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path $BasePath $Path))
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$PortableExe = Resolve-FullPath $PortableExe $repoRoot
$PortableStageRoot = Resolve-FullPath $PortableStageRoot $repoRoot
$PortableDataRoot = Join-Path (Split-Path -Parent $PortableExe) "portable-data"
$packageRuntimeRoot = Join-Path $PortableStageRoot "runtime"
$packageAdblockRoot = Join-Path $PortableStageRoot "adblock"
$launcherSourceDir = Join-Path $repoRoot "cmd\portable-launcher"
$frontendSourceDir = Join-Path $repoRoot "cmd\win32-frontend"
$launcherPayloadZip = Join-Path $launcherSourceDir "payload.zip"
$launcherSysoPath = Join-Path $launcherSourceDir "portable_icon.syso"
$frontendSysoPath = Join-Path $frontendSourceDir "comic_icon.syso"
$launcherPayloadPlaceholder = "placeholder payload archive; replaced by scripts/build_portable.ps1 before building the portable exe"

Set-Location $repoRoot

$env:GOCACHE = Join-Path $repoRoot ".gocache"
$env:GOTMPDIR = Join-Path $repoRoot ".gotmp"
Ensure-Dir $env:GOCACHE
Ensure-Dir $env:GOTMPDIR
Ensure-Dir (Split-Path -Parent $PortableExe)
Ensure-Dir (Split-Path -Parent $PortableStageRoot)

Write-Host "Portable exe: $PortableExe"
Write-Host "Portable stage root: $PortableStageRoot"
Write-Host "Firefox executable: $FirefoxExecutable"

if (Test-Path -LiteralPath $PortableStageRoot) {
    Remove-Item -LiteralPath $PortableStageRoot -Recurse -Force
}
if (Test-Path -LiteralPath $PortableExe) {
    Remove-Item -LiteralPath $PortableExe -Force
}
if (Test-Path -LiteralPath $PortableDataRoot) {
    Remove-Item -LiteralPath $PortableDataRoot -Recurse -Force
}
if (Test-Path -LiteralPath $launcherSysoPath) {
    Remove-Item -LiteralPath $launcherSysoPath -Force
}
if (Test-Path -LiteralPath $frontendSysoPath) {
    Remove-Item -LiteralPath $frontendSysoPath -Force
}
if (-not (Test-Path -LiteralPath $launcherPayloadZip)) {
    Set-Content -LiteralPath $launcherPayloadZip -Value $launcherPayloadPlaceholder -Encoding ASCII
}

Ensure-Dir $PortableStageRoot
Ensure-Dir $packageRuntimeRoot
Ensure-Dir (Join-Path $packageRuntimeRoot "browser-profiles")
Ensure-Dir (Join-Path $packageRuntimeRoot "browser-profiles\baseline-userdata")
Ensure-Dir (Join-Path $packageRuntimeRoot "browser-profiles\tasks")
Ensure-Dir (Join-Path $packageRuntimeRoot "browser-profiles\verification")
Ensure-Dir (Join-Path $packageRuntimeRoot "logs")
Ensure-Dir (Join-Path $packageRuntimeRoot "output")
Ensure-Dir (Join-Path $packageRuntimeRoot "tasks")
Ensure-Dir (Join-Path $packageRuntimeRoot "thumbnails")
Ensure-Dir $packageAdblockRoot

$iconSource = Join-Path $repoRoot "dist\01.jpg"
$iconRuntimePath = Join-Path $repoRoot "runtime\app.ico"
$iconSysoPath = Join-Path $repoRoot "comic_icon.syso"
if (Test-Path -LiteralPath $iconSource) {
    New-AppIconFromImage $iconSource $iconRuntimePath
    Copy-File $iconRuntimePath (Join-Path $packageRuntimeRoot "app.ico")
    & rsrc -ico $iconRuntimePath -o $iconSysoPath
    if ($LASTEXITCODE -ne 0) {
        throw "rsrc failed with exit code $LASTEXITCODE"
    }
} elseif (Test-Path -LiteralPath $iconSysoPath) {
    Write-Host "icon source image not found, reusing existing icon syso: $iconSysoPath"
    Copy-IfExists $iconRuntimePath (Join-Path $packageRuntimeRoot "app.ico")
} else {
    throw "icon source image not found and existing icon syso missing: $iconSource"
}

& go test ./...
if ($LASTEXITCODE -ne 0) {
    throw "go test failed"
}

& go build -tags playwright -ldflags "-H windowsgui" -o (Join-Path $PortableStageRoot "comic_downloader.exe") .\cmd\win32-frontend
if ($LASTEXITCODE -ne 0) {
    throw "go build failed"
}

Copy-IfExists (Join-Path $repoRoot "runtime\firefox_stealth.js") (Join-Path $packageRuntimeRoot "firefox_stealth.js")
Copy-IfExists (Join-Path $repoRoot "adblock\AWAvenue-Ads-Rule.txt") (Join-Path $packageAdblockRoot "AWAvenue-Ads-Rule.txt")
Copy-File $iconSysoPath $frontendSysoPath

$packageReadme = @"
Comic Downloader Portable Payload

This directory is packed into the single-file portable launcher.

- Firefox executable: $FirefoxExecutable

The launcher extracts this payload to a temp directory at runtime.
"@
Set-Content -LiteralPath (Join-Path $PortableStageRoot "README.txt") -Value $packageReadme -Encoding ASCII

Copy-File $iconSysoPath $launcherSysoPath
Compress-Archive -Path (Join-Path $PortableStageRoot '*') -DestinationPath $launcherPayloadZip -Force

& go build -tags playwright -ldflags "-H windowsgui" -o $PortableExe .\cmd\portable-launcher
if ($LASTEXITCODE -ne 0) {
    throw "go build portable launcher failed"
}

Set-Content -LiteralPath $launcherPayloadZip -Value $launcherPayloadPlaceholder -Encoding ASCII
Remove-Item -LiteralPath $launcherSysoPath -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $frontendSysoPath -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $PortableStageRoot -Recurse -Force -ErrorAction SilentlyContinue

Ensure-Dir $PortableDataRoot
Ensure-Dir (Join-Path $PortableDataRoot "adblock")
Ensure-Dir (Join-Path $PortableDataRoot "browser-profiles")
Ensure-Dir (Join-Path $PortableDataRoot "browser-profiles\baseline-userdata")
Ensure-Dir (Join-Path $PortableDataRoot "browser-profiles\tasks")
Ensure-Dir (Join-Path $PortableDataRoot "browser-profiles\verification")
Ensure-Dir (Join-Path $PortableDataRoot "logs")
Ensure-Dir (Join-Path $PortableDataRoot "output")
Ensure-Dir (Join-Path $PortableDataRoot "tasks")
Ensure-Dir (Join-Path $PortableDataRoot "thumbnails")

Write-Host "Portable exe created at: $PortableExe"
