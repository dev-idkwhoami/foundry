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

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\Foundry"
  CreateShortcut "$SMPROGRAMS\Foundry\Foundry.lnk" "$INSTDIR\foundry.exe"
  CreateShortcut "$SMPROGRAMS\Foundry\Uninstall.lnk" "$INSTDIR\uninstall.exe"

  ; Desktop shortcut
  CreateShortcut "$DESKTOP\Foundry.lnk" "$INSTDIR\foundry.exe"

  ; Uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\foundry.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\Foundry\Foundry.lnk"
  Delete "$SMPROGRAMS\Foundry\Uninstall.lnk"
  RMDir "$SMPROGRAMS\Foundry"
  Delete "$DESKTOP\Foundry.lnk"
SectionEnd
