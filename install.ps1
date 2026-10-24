# PhoneDrop installer. Put this next to phonedrop.exe and run it in PowerShell:
#   powershell -ExecutionPolicy Bypass -File .\install.ps1
# Installs to %LOCALAPPDATA%\PhoneDrop (no admin needed), starts it at sign-in and adds a Start-menu
# shortcut and a Send to entry. Run it from an *admin* PowerShell to also add the firewall rule (Private and
# Domain networks only); otherwise Windows asks the first time PhoneDrop starts listening.
# Rerun to update. Uninstall: .\install.ps1 -Uninstall
param([switch]$Uninstall)
$ErrorActionPreference = 'Stop'
$name = 'PhoneDrop'
$dir = Join-Path $env:LOCALAPPDATA $name
$exe = Join-Path $dir 'phonedrop.exe'
$lnk = Join-Path ([Environment]::GetFolderPath('Programs')) 'PhoneDrop.lnk'
$sendTo = Join-Path ([Environment]::GetFolderPath('SendTo')) 'PhoneDrop (to phone).lnk'
$run = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$admin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)

Get-Process phonedrop -ErrorAction SilentlyContinue | Stop-Process -Force -PassThru |
    Wait-Process -Timeout 10 -ErrorAction SilentlyContinue

if ($Uninstall) {
    Remove-ItemProperty -Path $run -Name $name -ErrorAction SilentlyContinue
    Remove-Item $lnk, $sendTo -ErrorAction SilentlyContinue
    if ($admin) { Remove-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue }
    Remove-Item $exe -ErrorAction SilentlyContinue
    Write-Host "Removed PhoneDrop. Settings and paired phones are still in $dir; delete that folder to forget them."
    return
}

$src = Join-Path $PSScriptRoot 'phonedrop.exe'
if (-not (Test-Path $src)) { throw "phonedrop.exe isn't next to this script ($PSScriptRoot)." }
New-Item -ItemType Directory -Force -Path $dir | Out-Null
# Copy beside it, then swap in; Windows can hold the old exe for a moment after the process exits.
Copy-Item -Force $src "$exe.new"
for ($i = 0; $i -lt 20; $i++) {
    try { Move-Item -Force "$exe.new" $exe -ErrorAction Stop; break } catch { Start-Sleep -Milliseconds 500 }
}
if (Test-Path "$exe.new") { throw "Couldn't replace $exe (still in use). Quit PhoneDrop from its tray icon and run this again." }
Unblock-File $exe

Set-ItemProperty -Path $run -Name $name -Value "`"$exe`""
$sc = (New-Object -ComObject WScript.Shell).CreateShortcut($lnk)
$sc.TargetPath = $exe
$sc.WorkingDirectory = $dir
$sc.IconLocation = "$exe,0"
$sc.Description = 'Receive files from your iPhone'
$sc.Save()
# Explorer: right-click a file > Send to > PhoneDrop queues it for the phone.
$sc = (New-Object -ComObject WScript.Shell).CreateShortcut($sendTo)
$sc.TargetPath = $exe
$sc.IconLocation = "$exe,0"
$sc.Description = 'Send to your iPhone with PhoneDrop'
$sc.Save()

if ($admin) {
    Remove-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue
    New-NetFirewallRule -DisplayName $name -Direction Inbound -Program $exe -Action Allow -Protocol TCP `
        -Profile Private,Domain -Description 'PhoneDrop: paired phones on Private and Domain networks' | Out-Null
    $fw = 'Firewall rule added (Private and Domain networks only).'
} else {
    $fw = 'Windows will ask about the firewall once: allow Private networks only.'
}
Start-Process $exe
Write-Host "Installed PhoneDrop (starts at sign-in; Start menu: PhoneDrop). $fw"
