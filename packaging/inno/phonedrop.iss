; PhoneDrop installer (Inno Setup 6). CI builds it on Windows:
;   iscc /DAppVersion=0.9.2 packaging\inno\phonedrop.iss
; with phonedrop.exe in the repo root. Output: dist\PhoneDrop-Setup.exe.
;
; Per-user install, no admin: the exe goes to %LOCALAPPDATA%\Programs\PhoneDrop, settings and paired
; phones stay in %LOCALAPPDATA%\PhoneDrop (the app's data folder, never touched by the uninstaller).
; Only the optional firewall rule needs elevation, so Windows asks for it once, separately.

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

[Setup]
AppId={{DE644CFA-4D8F-4A22-B4B8-50E7AFC16701}
AppName=PhoneDrop
AppVersion={#AppVersion}
AppVerName=PhoneDrop {#AppVersion}
AppPublisher=PhoneDrop contributors
AppPublisherURL=https://github.com/mynameishaheer/phonedrop
AppSupportURL=https://github.com/mynameishaheer/phonedrop/issues
VersionInfoVersion={#AppVersion}
DefaultDirName={localappdata}\Programs\PhoneDrop
DisableDirPage=yes
DisableProgramGroupPage=yes
DisableReadyPage=yes
PrivilegesRequired=lowest
MinVersion=10.0
ArchitecturesAllowed=x64compatible
CloseApplications=no
SetupIconFile=phonedrop.ico
UninstallDisplayIcon={app}\phonedrop.exe
UninstallDisplayName=PhoneDrop
WizardStyle=modern
Compression=lzma2/max
SolidCompression=yes
OutputDir=..\..\dist
OutputBaseFilename=PhoneDrop-Setup

[Messages]
FinishedLabel=PhoneDrop is installed.%n%nLook for its icon in the notification area of the taskbar (click ^ if you don't see it), then click Pair and scan the code with your iPhone.

[Tasks]
Name: startup; Description: "Start PhoneDrop when I sign in"
Name: firewall; Description: "Let my iPhone reach PhoneDrop through Windows Firewall (Private networks only; Windows asks for permission)"

[Files]
Source: "..\..\phonedrop.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{userprograms}\PhoneDrop"; Filename: "{app}\phonedrop.exe"; Comment: "Receive files from your iPhone"
; Explorer: right-click a file > Send to > PhoneDrop queues it for the phone.
Name: "{usersendto}\PhoneDrop (to phone)"; Filename: "{app}\phonedrop.exe"; Comment: "Send to your iPhone with PhoneDrop"

[Registry]
; The Run value is how PhoneDrop itself reads "Start at sign-in", so the panel toggle stays in step.
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "PhoneDrop"; ValueData: """{app}\phonedrop.exe"""; Tasks: startup

[Run]
Filename: "{app}\phonedrop.exe"; Description: "Start PhoneDrop now"; Flags: nowait postinstall

[Code]
const
  RunKey = 'Software\Microsoft\Windows\CurrentVersion\Run';

procedure StopPhoneDrop();
var
  rc: Integer;
begin
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/F /IM phonedrop.exe', '', SW_HIDE, ewWaitUntilTerminated, rc);
  Sleep(700); // Windows can hold the exe for a moment after the process exits
end;

// Runs cmd /c <cmd> elevated. Cancelling the UAC prompt is fine: PhoneDrop then relies on the
// prompt Windows Firewall shows the first time it listens.
procedure RunElevated(const Cmd: String);
var
  rc: Integer;
begin
  ShellExec('runas', ExpandConstant('{cmd}'), '/c ' + Cmd, '', SW_HIDE, ewWaitUntilTerminated, rc);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
begin
  StopPhoneDrop();
  // install.ps1 used to put the exe in the data folder; the new one supersedes it.
  DeleteFile(ExpandConstant('{localappdata}\PhoneDrop\phonedrop.exe'));
  Result := '';
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if (CurStep = ssPostInstall) and WizardIsTaskSelected('firewall') then
    RunElevated('netsh advfirewall firewall delete rule name="PhoneDrop" >nul 2>&1 & ' +
      'netsh advfirewall firewall add rule name="PhoneDrop" dir=in action=allow protocol=TCP ' +
      'profile=private,domain program="' + ExpandConstant('{app}\phonedrop.exe') + '" ' +
      'description="PhoneDrop: paired phones on Private and Domain networks"');
end;

function InitializeUninstall(): Boolean;
begin
  StopPhoneDrop();
  Result := True;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
  begin
    // The panel's "Start at sign-in" toggle may have written this even if the task was unticked.
    RegDeleteValue(HKCU, RunKey, 'PhoneDrop');
    RunElevated('netsh advfirewall firewall delete rule name="PhoneDrop" >nul 2>&1 & ' +
      'netsh advfirewall firewall delete rule name="PhoneDrop (trusted Public networks)" >nul 2>&1');
  end;
end;
