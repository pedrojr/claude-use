; Instalador do claude-use (Inno Setup 6).
; Compilar a partir da raiz do repositório, depois de gerar dist\claude-use.exe:
;   iscc /DAppVersion=1.0.0 installer\claude-use.iss

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

#define AppName "claude-use"
#define AppExe "claude-use.exe"
#define RunKey "Software\Microsoft\Windows\CurrentVersion\Run"

[Setup]
AppId={{6D3F0C4E-8B1A-4E4B-9C55-2B7F3C1D9A10}
AppName={#AppName}
AppVersion={#AppVersion}
AppVerName={#AppName} {#AppVersion}
AppPublisher=Pedro Júnior
AppPublisherURL=https://github.com/pedrojr/claude-use
AppSupportURL=https://github.com/pedrojr/claude-use/issues
AppUpdatesURL=https://github.com/pedrojr/claude-use/releases
; Instalação por usuário (sem UAC): {autopf} vira %LOCALAPPDATA%\Programs.
PrivilegesRequired=lowest
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
DisableDirPage=auto
LicenseFile=..\LICENSE
OutputDir=..\dist
OutputBaseFilename=claude-use-setup-{#AppVersion}
UninstallDisplayIcon={app}\{#AppExe}
UninstallDisplayName={#AppName}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
; Mesmo mutex usado pelo app (native_windows.go) para detectar que ele está aberto.
AppMutex=Local\com.github.claude-use
CloseApplications=yes

[Languages]
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "autostart"; Description: "Iniciar com o Windows"; GroupDescription: "Opções:"

[Files]
Source: "..\dist\{#AppExe}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExe}"
Name: "{group}\{cm:UninstallProgram,{#AppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Registry]
; Mesmo valor que o item "Iniciar com o Windows" da bandeja grava (setAutostart).
Root: HKCU; Subkey: "{#RunKey}"; ValueType: string; ValueName: "{#AppName}"; ValueData: """{app}\{#AppExe}"""; Tasks: autostart

[Run]
Filename: "{app}\{#AppExe}"; Description: "{cm:LaunchProgram,{#AppName}}"; Flags: nowait postinstall skipifsilent

[Code]
// Remove o início automático mesmo quando ele foi ligado pela bandeja, não pelo instalador.
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    RegDeleteValue(HKCU, '{#RunKey}', '{#AppName}');
end;
