# Installs the latest ghswitch release binary on Windows.
# Usage: irm https://raw.githubusercontent.com/MadhuS-1605/ghswitch/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "MadhuS-1605/ghswitch"
$InstallDir = if ($env:GHSWITCH_INSTALL_DIR) { $env:GHSWITCH_INSTALL_DIR } else { "$env:LOCALAPPDATA\ghswitch" }

$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    throw "Unsupported architecture"
}

$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$version = $release.tag_name
if (-not $version) { throw "Could not determine latest version." }

$archive = "ghswitch_windows_${arch}.zip"
$url = "https://github.com/$Repo/releases/download/$version/$archive"

$tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([System.Guid]::NewGuid()))
try {
    $zipPath = Join-Path $tmp $archive
    Write-Host "Downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $zipPath
    Expand-Archive -Path $zipPath -DestinationPath $tmp -Force

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Move-Item -Path (Join-Path $tmp "ghswitch.exe") -Destination (Join-Path $InstallDir "ghswitch.exe") -Force

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        Write-Host "Added $InstallDir to your user PATH. Restart your terminal to pick it up."
    }

    Write-Host "Installed ghswitch $version to $InstallDir\ghswitch.exe"
    Write-Host 'Add: Invoke-Expression (& ghswitch shellinit) to your $PROFILE for per-terminal identity isolation.'
} finally {
    Remove-Item -Recurse -Force $tmp
}
