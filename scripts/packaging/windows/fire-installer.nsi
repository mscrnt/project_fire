; F.I.R.E. NSIS Installer Script

!include "MUI2.nsh"

; General
Name "F.I.R.E. - Full Intensity Rigorous Evaluation"
OutFile "..\..\..\..\dist\windows-amd64\fire-installer-${VERSION}.exe"
InstallDir "$PROGRAMFILES64\FIRE"
InstallDirRegKey HKLM "Software\FIRE" "Install_Dir"
RequestExecutionLevel admin

; Version info
!define VERSION "1.0.0"
!define PRODUCT_NAME "F.I.R.E."
!define PRODUCT_PUBLISHER "F.I.R.E. Team"
!define PRODUCT_WEB_SITE "https://github.com/mscrnt/project_fire"

; MUI Settings
!define MUI_ABORTWARNING
!define MUI_ICON "..\..\..\assets\logos\fire.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Welcome page
!insertmacro MUI_PAGE_WELCOME
; License page
!insertmacro MUI_PAGE_LICENSE "..\..\..\LICENSE"
; Directory page
!insertmacro MUI_PAGE_DIRECTORY
; Instfiles page
!insertmacro MUI_PAGE_INSTFILES
; Finish page
!define MUI_FINISHPAGE_RUN "$INSTDIR\fire-gui.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Launch F.I.R.E. GUI"
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_INSTFILES

; Language
!insertmacro MUI_LANGUAGE "English"

; Version Information
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey "CompanyName" "${PRODUCT_PUBLISHER}"
VIAddVersionKey "LegalCopyright" "Copyright (c) 2025 ${PRODUCT_PUBLISHER}"
VIAddVersionKey "FileDescription" "F.I.R.E. Installer"
VIAddVersionKey "FileVersion" "${VERSION}"

; Installer Section
Section "MainSection" SEC01
    SetOutPath "$INSTDIR"
    SetOverwrite ifnewer
    
    ; Copy executables
    File "..\..\..\bench.exe"
    File "..\..\..\fire-gui.exe"
    
    ; Copy documentation
    File /nonfatal "..\..\..\README.md"
    File /nonfatal "..\..\..\LICENSE"
    
    ; Create shortcuts
    CreateDirectory "$SMPROGRAMS\F.I.R.E."
    CreateShortcut "$SMPROGRAMS\F.I.R.E.\F.I.R.E. GUI.lnk" "$INSTDIR\fire-gui.exe"
    CreateShortcut "$SMPROGRAMS\F.I.R.E.\F.I.R.E. CLI.lnk" "$INSTDIR\bench.exe"
    CreateShortcut "$SMPROGRAMS\F.I.R.E.\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    ; Desktop shortcut (optional)
    CreateShortcut "$DESKTOP\F.I.R.E..lnk" "$INSTDIR\fire-gui.exe"
    
    ; Add to PATH
    nsExec::ExecToLog 'setx PATH "$INSTDIR;%PATH%" /M'
    
    ; Write registry keys
    WriteRegStr HKLM "SOFTWARE\FIRE" "Install_Dir" "$INSTDIR"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FIRE" "DisplayName" "F.I.R.E."
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FIRE" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FIRE" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FIRE" "NoRepair" 1
    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

; Uninstaller Section
Section "Uninstall"
    ; Remove files
    Delete "$INSTDIR\bench.exe"
    Delete "$INSTDIR\fire-gui.exe"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\LICENSE"
    Delete "$INSTDIR\uninstall.exe"
    
    ; Remove shortcuts
    Delete "$SMPROGRAMS\F.I.R.E.\*.*"
    Delete "$DESKTOP\F.I.R.E..lnk"
    
    ; Remove directories
    RMDir "$SMPROGRAMS\F.I.R.E."
    RMDir "$INSTDIR"
    
    ; Remove registry keys
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FIRE"
    DeleteRegKey HKLM "SOFTWARE\FIRE"
SectionEnd