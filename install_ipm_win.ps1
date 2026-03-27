$ErrorActionPreference = "Stop"

$APPNAME = "ipm"
$DIST_DIR = "ipm_dist"
$INSTALL_DIR = "$HOME\.local\bin"

# Find the exe regardless of arch suffix (amd64 or arm64)
$BuiltFile = Get-ChildItem -Path "$DIST_DIR" -Filter "$APPNAME-windows-*.exe" | Select-Object -First 1

if (-not $BuiltFile) {
    Write-Error "Could not find a built .exe in $DIST_DIR. Did you run the build script in WSL first?"
    exit
}

$INSTALL_PATH = "$INSTALL_DIR\$APPNAME.exe"
$TARGET_FOLDER = Split-Path -Parent $INSTALL_PATH

Write-Host "==> Preparing install directory..." -ForegroundColor Cyan
if (!(Test-Path $TARGET_FOLDER)) {
    New-Item -ItemType Directory -Force -Path $TARGET_FOLDER | Out-Null
}

Write-Host "==> Installing $APPNAME to $INSTALL_PATH..." -ForegroundColor Cyan
Copy-Item -Path $BuiltFile.FullName -Destination $INSTALL_PATH -Force

Write-Host ""
Write-Host "==> Installation complete!" -ForegroundColor Green
Write-Host "You can now run '$APPNAME' (ensure $INSTALL_DIR is in your PATH)."