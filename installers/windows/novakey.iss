#define AppName "NovaKey"
#define AppPublisher "OsbornePro"

#define AppVersion "1.0.0"
#ifdef MyAppVersion
  #undef AppVersion
  #define AppVersion MyAppVersion
#endif

#define AppExeName "novakey.exe"

#define AppRoot "{localappdata}\NovaKey"
#define DataDir "{localappdata}\NovaKey"

[Setup]
AppId={{A5B6D7E8-1234-4A8B-9C9D-111111111111}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
; Install payload into ...\NovaKey
DefaultDirName={#AppRoot}
DisableDirPage=yes
DisableProgramGroupPage=yes
OutputDir=.\out
OutputBaseFilename=NovaKey-Setup
Compression=lzma
SolidCompression=yes
PrivilegesRequired=lowest

[Dirs]
; Data directory is under LocalAppData; user already owns it so no special permissions needed
Name: "{#DataDir}"

[Files]
; App payload
Source: "..\..\dist\windows\{#AppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: ".\helper\out\novakey-installer-helper.exe"; DestDir: "{app}"; Flags: ignoreversion

; Helper also extracted to temp during install for the install-time Run entry
Source: ".\helper\out\novakey-installer-helper.exe"; DestDir: "{tmp}"; Flags: deleteafterinstall

; Config goes to DATA (runtime). onlyifdoesntexist so user edits survive upgrades/reinstalls.
Source: "..\..\server_config.yaml"; DestDir: "{#DataDir}"; DestName: "server_config.yaml"; Flags: onlyifdoesntexist

; Optional devices.json if you ever ship it. Put it in DATA too.
Source: "..\..\devices.json"; DestDir: "{#DataDir}"; DestName: "devices.json"; Flags: skipifsourcedoesntexist onlyifdoesntexist

[Run]
; Create Scheduled Task: exe from {app}, working dir + config in DATA
Filename: "{tmp}\novakey-installer-helper.exe"; Parameters: "install ""{app}"" ""{#DataDir}"""; Flags: runhidden

[UninstallRun]
; Prefer helper uninstall if it exists
Filename: "{app}\novakey-installer-helper.exe"; Parameters: "uninstall"; Flags: runhidden; RunOnceId: "NovaKeyUninstallHelper"; Check: HelperExists

[UninstallDelete]
; Remove app payload
Type: files; Name: "{app}\novakey.exe"
Type: files; Name: "{app}\novakey-installer-helper.exe"

; Remove generated runtime artifacts in DATA (except server_keys.json which is user-choice)
Type: files; Name: "{#DataDir}\novakey-pair.png"
Type: files; Name: "{#DataDir}\devices.json"
Type: filesandordirs; Name: "{#DataDir}\logs"
Type: filesandordirs; Name: "{#DataDir}\log"
Type: filesandordirs; Name: "{#DataDir}\tmp"

[Code]
var
  KeepServerKeys: Boolean;

function HelperExists: Boolean;
begin
  Result := FileExists(ExpandConstant('{app}\novakey-installer-helper.exe'));
end;

procedure DeleteTaskFallback;
var
  ResultCode: Integer;
begin
  { Best-effort: end + delete task. Ignore errors. }
  Exec('schtasks', '/End /TN "NovaKey"', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec('schtasks', '/Delete /TN "NovaKey" /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
end;

function InitializeUninstall(): Boolean;
var
  ResultCode: Integer;
begin
  ResultCode := MsgBox(
    'Keep NovaKey pairing keys for future reinstall?' + #13#10 + #13#10 +
    'Yes = keep server_keys.json (recommended if you plan to reinstall).' + #13#10 +
    'No  = delete keys (reinstall will require re-pairing).',
    mbConfirmation, MB_YESNO);

  KeepServerKeys := (ResultCode = IDYES);
  Result := True;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  KeysPath: String;
begin
  if CurUninstallStep = usUninstall then
  begin
    { If helper is missing for any reason, clean up the scheduled task directly. }
    if not HelperExists then
      DeleteTaskFallback;

    { Delete keys only if user chose not to keep them. }
    KeysPath := ExpandConstant('{#DataDir}\server_keys.json');
    if not KeepServerKeys then
      DeleteFile(KeysPath);
  end;
end;
