; claude-use installer (Inno Setup 6.3+ / 7).
; Build from the repository root, after producing dist\claude-use.exe:
;   iscc /DAppVersion=1.0.0 installer\claude-use.iss
;
; Install modes (chosen in a dialog when Setup starts):
;   - "Install for all users" (default): Program Files, needs administrator rights (UAC).
;   - "Install for me only": %LOCALAPPDATA%\Programs, no administrator rights needed.
; The destination folder can be changed in both modes. Silent installs pick the mode
; with /ALLUSERS or /CURRENTUSER.

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

#define AppName "claude-use"
#define AppExe "claude-use.exe"
#define AppGUID "6D3F0C4E-8B1A-4E4B-9C55-2B7F3C1D9A10"
#define RunKey "Software\Microsoft\Windows\CurrentVersion\Run"
#define UninstallKey "Software\Microsoft\Windows\CurrentVersion\Uninstall\{" + AppGUID + "}_is1"

[Setup]
AppId={{{#AppGUID}}
AppName={#AppName}
AppVersion={#AppVersion}
AppVerName={#AppName} {#AppVersion}
AppPublisher=Pedro Júnior
AppPublisherURL=https://github.com/pedrojr/claude-use
AppSupportURL=https://github.com/pedrojr/claude-use/issues
AppUpdatesURL=https://github.com/pedrojr/claude-use/releases
; Program Files by default; users without administrator rights can choose
; "Install for me only" in the dialog shown at startup.
PrivilegesRequired=admin
PrivilegesRequiredOverridesAllowed=dialog commandline
; Always ask (a previous per-user install is migrated, see [Code]).
UsePreviousPrivileges=no
; {autopf} = Program Files (all users) or %LOCALAPPDATA%\Programs (me only).
DefaultDirName={autopf}\{#AppName}
DisableDirPage=no
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
; Start with Windows is per user (HKCU) and is registered as the user who ran Setup.
UsedUserAreasWarning=no
LicenseFile=..\LICENSE
OutputDir=..\dist
OutputBaseFilename=claude-use-setup-{#AppVersion}
SetupIconFile=..\assets\claude-use.ico
UninstallDisplayIcon={app}\{#AppExe}
UninstallDisplayName={#AppName}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
SetupLogging=yes
; Same mutex the app creates (native_windows.go), used to detect that it is running.
AppMutex=Local\com.github.claude-use
CloseApplications=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"

[CustomMessages]
english.Options=Options:
english.AutoStart=Start with Windows
english.DirNotWritable=You do not have permission to write to this folder:%n%n%1%n%nChoose a folder inside your user profile (for example %2), or restart Setup and choose to install for all users.
brazilianportuguese.Options=Opções:
brazilianportuguese.AutoStart=Iniciar com o Windows
brazilianportuguese.DirNotWritable=Você não tem permissão para gravar nesta pasta:%n%n%1%n%nEscolha uma pasta dentro do seu perfil de usuário (por exemplo %2) ou reinicie o instalador e escolha instalar para todos os usuários.

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "autostart"; Description: "{cm:AutoStart}"; GroupDescription: "{cm:Options}"

[Files]
Source: "..\dist\{#AppExe}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README_PTBR.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExe}"
Name: "{group}\{cm:UninstallProgram,{#AppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Run]
; The app writes the Run value itself (same as the tray menu item). runasoriginaluser makes it
; land in the HKCU of the user who started Setup, even when elevation used another account.
Filename: "{app}\{#AppExe}"; Parameters: "-autostart=on"; Flags: runasoriginaluser waituntilterminated; Tasks: autostart
Filename: "{app}\{#AppExe}"; Parameters: "-autostart=off"; Flags: runasoriginaluser waituntilterminated; Tasks: not autostart
Filename: "{app}\{#AppExe}"; Description: "{cm:LaunchProgram,{#AppName}}"; Flags: nowait postinstall skipifsilent

[Code]
// A per-user install (older versions only installed that way) is removed when installing
// for all users, so the user does not end up with two copies.
procedure RemovePerUserInstall();
var
  Uninstaller: String;
  ResultCode: Integer;
begin
  if not IsAdminInstallMode then
    exit;
  if not RegQueryStringValue(HKCU, '{#UninstallKey}', 'UninstallString', Uninstaller) then
    exit;
  Uninstaller := RemoveQuotes(Uninstaller);
  if not FileExists(Uninstaller) then
    exit;
  Log('Removing per-user install: ' + Uninstaller);
  ExecAsOriginalUser(Uninstaller, '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART', '', SW_HIDE,
    ewWaitUntilTerminated, ResultCode);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssInstall then
    RemovePerUserInstall();
end;

// Without administrator rights, check that the chosen folder is writable before copying files.
function NextButtonClick(CurPageID: Integer): Boolean;
var
  Dir, Parent, TestFile: String;
begin
  Result := True;
  if (CurPageID <> wpSelectDir) or IsAdminInstallMode then
    exit;
  Dir := WizardDirValue;
  while not DirExists(Dir) do
  begin
    Parent := ExtractFileDir(Dir);
    if (Parent = '') or (Parent = Dir) then
      break;
    Dir := Parent;
  end;
  TestFile := AddBackslash(Dir) + 'claude-use-write-test.tmp';
  if SaveStringToFile(TestFile, '', False) then
    DeleteFile(TestFile)
  else
  begin
    // (a line in [Code] must not start with "[", Inno would read it as a section tag)
    MsgBox(FmtMessage(CustomMessage('DirNotWritable'), [WizardDirValue,
      ExpandConstant('{localappdata}\Programs')]), mbError, MB_OK);
    Result := False;
  end;
end;

// Remove start with Windows (even when enabled from the tray menu), but only if it
// points to this copy of the app.
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Value: String;
begin
  if CurUninstallStep <> usPostUninstall then
    exit;
  if RegQueryStringValue(HKCU, '{#RunKey}', '{#AppName}', Value) and
     (Pos(Lowercase(ExpandConstant('{app}\{#AppExe}')), Lowercase(Value)) > 0) then
    RegDeleteValue(HKCU, '{#RunKey}', '{#AppName}');
end;
