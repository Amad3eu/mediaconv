param(
    [string]$Version = $env:MEDIACONV_VERSION,
    [string]$InstallDir = $env:MEDIACONV_INSTALL_DIR
)

$ErrorActionPreference = "Stop"

$Repo = "Amad3eu/mediaconv"
$Project = "mediaconv"
$GitHubBase = "https://github.com/$Repo"

function Write-Info {
    param([string]$Message)
    Write-Host $Message
}

function Stop-Install {
    param([string]$Message)
    Write-Error "error: $Message"
    exit 1
}

function Resolve-LatestTag {
    $release = Invoke-RestMethod `
        -Uri "https://api.github.com/repos/$Repo/releases/latest" `
        -Headers @{ "User-Agent" = "$Project-installer" }

    if (-not $release.tag_name) {
        Stop-Install "could not resolve latest release tag"
    }

    return $release.tag_name
}

function Resolve-Arch {
    $arch = $env:PROCESSOR_ARCHITEW6432

    if (-not $arch) {
        $arch = $env:PROCESSOR_ARCHITECTURE
    }

    if (-not $arch) {
        $runtimeInformation = "System.Runtime.InteropServices.RuntimeInformation" -as [type]
        if ($runtimeInformation) {
            $arch = $runtimeInformation::OSArchitecture.ToString()
        }
    }

    if (-not $arch) {
        Stop-Install "could not detect CPU architecture"
    }

    switch -Regex ($arch.ToLowerInvariant()) {
        "x64|amd64" { return "amd64" }
        "arm64" { return "arm64" }
        default { Stop-Install "unsupported architecture: $arch" }
    }
}

function Invoke-Download {
    param(
        [string]$Url,
        [string]$OutFile
    )

    Invoke-WebRequest `
        -Uri $Url `
        -OutFile $OutFile `
        -UseBasicParsing `
        -Headers @{ "User-Agent" = "$Project-installer" }
}

function Confirm-Checksum {
    param(
        [string]$File,
        [string]$ChecksumsFile
    )

    $name = Split-Path -Leaf $File
    $line = Get-Content $ChecksumsFile | Where-Object { $_ -match "\s+$([regex]::Escape($name))$" } | Select-Object -First 1

    if (-not $line) {
        Stop-Install "checksum for $name was not found"
    }

    $expectedHash = ($line -split "\s+")[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -Algorithm SHA256 $File).Hash.ToLowerInvariant()

    if ($expectedHash -ne $actualHash) {
        Stop-Install "checksum verification failed"
    }

    Write-Info "${name}: OK"
}

function Resolve-InstallDir {
    if ($InstallDir) {
        return $InstallDir
    }

    if ($env:LOCALAPPDATA) {
        return Join-Path $env:LOCALAPPDATA "Programs\$Project\bin"
    }

    if ($env:USERPROFILE) {
        return Join-Path $env:USERPROFILE ".mediaconv\bin"
    }

    Stop-Install "set MEDIACONV_INSTALL_DIR to choose an install directory"
}

function Add-ToUserPath {
    param([string]$Directory)

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @()

    if ($currentPath) {
        $entries = $currentPath -split ";"
    }

    $alreadyConfigured = $entries | Where-Object {
        $_.TrimEnd("\") -ieq $Directory.TrimEnd("\")
    }

    if (-not $alreadyConfigured) {
        $newPath = if ($currentPath) { "$currentPath;$Directory" } else { $Directory }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Info "Added $Directory to your user PATH."
        Write-Info "Open a new terminal before running $Project from anywhere."
    }

    if (($env:Path -split ";") -notcontains $Directory) {
        $env:Path = "$env:Path;$Directory"
    }
}

if ([System.Environment]::OSVersion.Platform -ne [System.PlatformID]::Win32NT) {
    Stop-Install "install.ps1 supports Windows only"
}

if (-not $Version -or $Version -eq "latest") {
    $Tag = Resolve-LatestTag
} else {
    $Tag = $Version
}

if ($Tag -notmatch "^v\d+\.\d+\.\d+$") {
    Stop-Install "MEDIACONV_VERSION must look like v1.2.3"
}

$PlainVersion = $Tag.TrimStart("v")
$Arch = Resolve-Arch
$Archive = "${Project}_${PlainVersion}_windows_${Arch}.zip"
$Checksums = "${Project}_${PlainVersion}_checksums.txt"
$DownloadBase = "$GitHubBase/releases/download/$Tag"
$TargetDir = Resolve-InstallDir
$Target = Join-Path $TargetDir "$Project.exe"
$TempDir = Join-Path ([IO.Path]::GetTempPath()) "$Project-install-$([guid]::NewGuid())"

New-Item -ItemType Directory -Force -Path $TempDir | Out-Null

try {
    Write-Info "Installing $Project $Tag for windows/$Arch"

    $ArchivePath = Join-Path $TempDir $Archive
    $ChecksumsPath = Join-Path $TempDir $Checksums

    Invoke-Download "$DownloadBase/$Archive" $ArchivePath
    Invoke-Download "$DownloadBase/$Checksums" $ChecksumsPath
    Confirm-Checksum $ArchivePath $ChecksumsPath

    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force

    $Binary = Join-Path $TempDir "$Project.exe"
    if (-not (Test-Path $Binary)) {
        Stop-Install "archive did not contain $Project.exe"
    }

    New-Item -ItemType Directory -Force -Path $TargetDir | Out-Null
    Copy-Item -Path $Binary -Destination $Target -Force
    Add-ToUserPath $TargetDir

    Write-Info "Installed: $Target"
    & $Target version

    if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue) -or -not (Get-Command ffprobe -ErrorAction SilentlyContinue)) {
        Write-Info "warning: ffmpeg and ffprobe are required for conversions"
        Write-Info "run '$Project doctor' after installing FFmpeg"
    }
}
finally {
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
