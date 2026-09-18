<#
.SYNOPSIS
Packs the Windows build into an MSIX package for the Microsoft Store.

.DESCRIPTION
Builds the release executable, lays it out with the manifest and the tile
images of this folder, indexes the images with makepri and packs everything
with makeappx. Both tools come with the Windows SDK. The package is not signed:
the Store signs what it publishes.

The version comes from internal/app/version.go, which the release workflow
writes, so running this on a release commit packs that release.

Needs Go and a MinGW gcc on PATH, like any other Windows build.

.EXAMPLE
powershell -ExecutionPolicy Bypass -File packaging\msix\build-msix.ps1
#>
param(
    # The version to pack, such as 2.0.0. Read from the code when left out.
    [string] $Version,

    # Where the package is written, relative to the repository.
    [string] $Output = "bin\msix"
)

$ErrorActionPreference = "Stop"

function Invoke-Tool([string] $Tool, [string[]] $Arguments) {
    & $Tool @Arguments | Out-Host

    if ($LASTEXITCODE -ne 0) {
        throw "$(Split-Path -Leaf $Tool) failed with exit code $LASTEXITCODE"
    }
}

$root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
Push-Location $root

try {
    if (-not $Version) {
        $line = Select-String -Path "internal\app\version.go" -Pattern '^const applicationVersion = "([^"]+)"'
        $Version = $line.Matches[0].Groups[1].Value
    }

    if ($Version -notmatch '^\d+\.\d+\.\d+$') {
        throw "'$Version' is not a release version such as 2.0.0. Pack a release commit, or pass -Version."
    }

    # The Store wants four numbers, the last of them zero.
    $packageVersion = "$Version.0"

    # The newest Windows SDK that has the tools.
    $makeappx = Get-ChildItem "${env:ProgramFiles(x86)}\Windows Kits\10\bin\*\x64\makeappx.exe" |
        Sort-Object { [version] $_.Directory.Parent.Name } |
        Select-Object -Last 1

    if (-not $makeappx) {
        throw "makeappx.exe was not found. Install the Windows SDK."
    }

    $makepri = Join-Path $makeappx.DirectoryName "makepri.exe"

    $output = Join-Path $root $Output
    $layout = Join-Path $output "layout"

    if (Test-Path $layout) {
        Remove-Item -Recurse -Force $layout
    }

    New-Item -ItemType Directory -Force $layout | Out-Null

    Write-Host "Building pdx-flag-builder $Version"
    Invoke-Tool "go" @("build", "-trimpath", "-ldflags", "-s -w -H windowsgui",
        "-o", (Join-Path $layout "pdx-flag-builder.exe"), ".\cmd\pdx-flag-builder")

    Copy-Item -Recurse "packaging\msix\Assets" (Join-Path $layout "Assets")

    # The font is built into the application, and its licence asks to travel
    # with it.
    Copy-Item "LICENSE" (Join-Path $layout "LICENSE.txt")
    Copy-Item "assets\fonts\roboto\OFL.txt" (Join-Path $layout "Roboto-OFL.txt")

    $manifest = (Get-Content -Raw "packaging\msix\AppxManifest.xml").Replace('$VERSION$', $packageVersion)
    [IO.File]::WriteAllText((Join-Path $layout "AppxManifest.xml"), $manifest)

    # The tiles come in several scales and sizes. The resource index is what
    # lets Windows pick the right file for Assets\...Logo.png.
    $config = Join-Path $output "priconfig.xml"
    Invoke-Tool $makepri @("createconfig", "/cf", $config, "/dq", "en-US", "/pv", "10.0.0", "/o")

    # The default configuration splits the index by scale and language, into
    # resource packs for an app bundle. A single package needs all of it in
    # one resources.pri, so the splitting is taken out.
    $xml = [xml] (Get-Content -Raw $config)
    $packaging = $xml.resources.SelectSingleNode("packaging")
    if ($packaging) {
        [void] $xml.resources.RemoveChild($packaging)
    }
    $xml.Save($config)

    Invoke-Tool $makepri @("new", "/pr", $layout, "/cf", $config,
        "/mn", (Join-Path $layout "AppxManifest.xml"), "/of", (Join-Path $layout "resources.pri"), "/o")

    $package = Join-Path $output "pdx-flag-builder_${Version}_x64.msix"
    Invoke-Tool $makeappx.FullName @("pack", "/d", $layout, "/p", $package, "/o")

    Write-Host "Packed $package"
}
finally {
    Pop-Location
}
