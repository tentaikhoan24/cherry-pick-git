# dev.ps1 — start the Lazy Cherry Pick Tauri dev server
# Usage: from any PowerShell window, run:   d:\project\lazy-cherry-pick\dev.ps1
#   or, from inside the project root:        .\dev.ps1

$env:Path = "$env:USERPROFILE\.cargo\bin;C:\Program Files\Go\bin;" + $env:Path

$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location (Join-Path $projectRoot "app")

Write-Host "Starting Tauri dev server..." -ForegroundColor Cyan
Write-Host "Press Ctrl+C in this window to stop." -ForegroundColor DarkGray
Write-Host ""

npm run tauri dev
