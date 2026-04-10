!include "MUI2.nsh"

Name "Foundry"
OutFile "build\bin\foundry-setup.exe"
InstallDir "$LOCALAPPDATA\Foundry"
RequestExecutionLevel user

!define MUI_ICON "build\windows\icon.ico"
!define MUI_UNICON "build\windows\icon.ico"

!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "build\bin\foundry.exe"
  File "build\bin\foundry-cli.exe"

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\Foundry"
  CreateShortcut "$SMPROGRAMS\Foundry\Foundry.lnk" "$INSTDIR\foundry.exe"
  CreateShortcut "$SMPROGRAMS\Foundry\Uninstall.lnk" "$INSTDIR\uninstall.exe"

  ; Desktop shortcut
  CreateShortcut "$DESKTOP\Foundry.lnk" "$INSTDIR\foundry.exe"

  ; Add to user PATH so foundry (GUI) and foundry-cli work from any terminal.
  nsExec::ExecToLog 'powershell -NoProfile -Command "\
    $$p = [Environment]::GetEnvironmentVariable(\"Path\", \"User\"); \
    if ($$p -notlike \"*$INSTDIR*\") { \
      [Environment]::SetEnvironmentVariable(\"Path\", \"$$p;$INSTDIR\", \"User\") \
    }"'

  ; Notify running processes of the env change.
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  ; Uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\foundry.exe"
  Delete "$INSTDIR\foundry-cli.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\Foundry\Foundry.lnk"
  Delete "$SMPROGRAMS\Foundry\Uninstall.lnk"
  RMDir "$SMPROGRAMS\Foundry"
  Delete "$DESKTOP\Foundry.lnk"

  ; Remove from user PATH.
  nsExec::ExecToLog 'powershell -NoProfile -Command "\
    $$p = [Environment]::GetEnvironmentVariable(\"Path\", \"User\"); \
    $$p = ($$p.Split(\";\") | Where-Object { $$_ -ne \"$INSTDIR\" }) -join \";\"; \
    [Environment]::SetEnvironmentVariable(\"Path\", $$p, \"User\")"'

  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
SectionEnd
