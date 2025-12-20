; Inno Setup Script for gsh
; This script creates a Windows installer (.exe) for gsh

#define MyAppName "gsh"
#define MyAppVersion GetEnv('GSH_VERSION')
#define MyAppPublisher "robottwo"
#define MyAppURL "https://github.com/robottwo/gsh_prime"
#define MyAppExeName "gsh.exe"

[Setup]
; Unique identifier for this application
AppId={{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
; No Start Menu group needed for CLI tool
DisableProgramGroupPage=yes
; License file
LicenseFile=LICENSE
; Output settings
OutputDir=.
OutputBaseFilename=gsh_{#MyAppVersion}_windows_x86_64_setup
; Compression
Compression=lzma2
SolidCompression=yes
; Modern look
WizardStyle=modern
; Require admin for Program Files, but allow per-user install
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
; Architecture
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "gsh.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "README.md"; DestDir: "{app}"; Flags: ignoreversion; Check: SourceFileExists('README.md')

[Icons]
; Optional: Add to Start Menu if user wants
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Check: not WizardNoIcons

[Registry]
; Add to PATH for current user or all users depending on install mode
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Check: NeedsAddPath('{app}') and not IsAdminInstallMode
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Check: NeedsAddPath('{app}') and IsAdminInstallMode

[Code]
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
  ParamExpanded: string;
begin
  // Expand the path parameter
  ParamExpanded := ExpandConstant(Param);

  // Get current PATH
  if IsAdminInstallMode then
    RegQueryStringValue(HKLM, 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment', 'Path', OrigPath)
  else
    RegQueryStringValue(HKCU, 'Environment', 'Path', OrigPath);

  // Check if already in PATH
  Result := Pos(';' + Uppercase(ParamExpanded) + ';', ';' + Uppercase(OrigPath) + ';') = 0;
end;

function SourceFileExists(FileName: string): boolean;
begin
  Result := FileExists(ExpandConstant('{src}\' + FileName));
end;

[UninstallDelete]
Type: filesandordirs; Name: "{app}"
