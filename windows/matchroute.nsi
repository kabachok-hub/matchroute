Unicode true
RequestExecutionLevel admin
SetCompressor /SOLID lzma

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinVer.nsh"
!include "x64.nsh"

!define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\MatchRoute"

Name "MatchRoute"
OutFile "${SETUP_OUTPUT}"
InstallDir "$PROGRAMFILES64\MatchRoute"
InstallDirRegKey HKLM "${UNINSTALL_KEY}" "InstallLocation"
VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey /LANG=1033 "ProductName" "MatchRoute"
VIAddVersionKey /LANG=1033 "FileDescription" "MatchRoute Setup"
VIAddVersionKey /LANG=1033 "FileVersion" "${APP_VERSION}"
VIAddVersionKey /LANG=1033 "ProductVersion" "${APP_VERSION}"
VIAddVersionKey /LANG=1033 "LegalCopyright" "Based on fnport by YUNGSLXRD (MIT)"

!define MUI_ABORTWARNING
!define MUI_ICON "${APP_ICON}"
!define MUI_UNICON "${APP_ICON}"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "Russian"
!insertmacro MUI_LANGUAGE "English"

!macro RequireClosedApp
    System::Call 'kernel32::OpenMutexW(i 0x00100000, i 0, w "Local\matchroute-windows") p.r0'
    ${If} $0 != 0
        System::Call 'kernel32::CloseHandle(p r0)'
        MessageBox MB_OK|MB_ICONEXCLAMATION "Закройте MatchRoute через меню значка в области уведомлений и повторите попытку." /SD IDOK
        SetErrorLevel 2
        Abort
    ${EndIf}
!macroend

Function .onInit
    ${IfNot} ${AtLeastWin10}
    ${OrIfNot} ${IsNativeAMD64}
        MessageBox MB_OK|MB_ICONSTOP "Для MatchRoute нужна Windows 10 или 11 на ПК с процессором x64." /SD IDOK
        SetErrorLevel 1
        Abort
    ${EndIf}
FunctionEnd

Function un.onInit
    !insertmacro RequireClosedApp
FunctionEnd

Section "MatchRoute"
    ${If} ${FileExists} "$INSTDIR\Uninstall.exe"
        !insertmacro RequireClosedApp
    ${EndIf}
    SetShellVarContext all
    SetOutPath "$INSTDIR"
    File "${PAYLOAD}\MatchRoute.exe"
    File "${PAYLOAD}\sing-box.exe"
    File "${PAYLOAD}\wintun.dll"
    File "${PAYLOAD}\LICENSES.txt"
    SetOutPath "$INSTDIR\fakes"
    File "${PAYLOAD}\fakes\quic_initial_vk_com.bin"
    File "${PAYLOAD}\fakes\quic_initial_gosuslugi_ru.bin"
    File "${PAYLOAD}\fakes\quic_initial_ozon_ru.bin"
    SetOutPath "$INSTDIR"
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    CreateShortCut "$SMPROGRAMS\MatchRoute.lnk" "$INSTDIR\MatchRoute.exe"
    WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayName" "MatchRoute"
    WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayVersion" "${APP_VERSION}"
    WriteRegStr HKLM "${UNINSTALL_KEY}" "Publisher" "kabachok-hub"
    WriteRegStr HKLM "${UNINSTALL_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayIcon" "$INSTDIR\MatchRoute.exe"
    WriteRegStr HKLM "${UNINSTALL_KEY}" "URLInfoAbout" "https://github.com/kabachok-hub/matchroute"
    WriteRegStr HKLM "${UNINSTALL_KEY}" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
    WriteRegStr HKLM "${UNINSTALL_KEY}" "QuietUninstallString" '$\"$INSTDIR\Uninstall.exe$\" /S'
    WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoModify" 1
    WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoRepair" 1
SectionEnd

Section "Uninstall"
    nsExec::Exec '"$SYSDIR\schtasks.exe" /Delete /TN "\MatchRoute AutoStart" /F /HRESULT'
    Pop $0
    ${If} $0 != 0
    ${AndIf} $0 != -2147024894
    ${AndIf} $0 != 2147942402
        MessageBox MB_OK|MB_ICONSTOP "Не удалось удалить задачу автозапуска MatchRoute." /SD IDOK
        SetErrorLevel 3
        Abort
    ${EndIf}
    SetShellVarContext all
    Delete "$SMPROGRAMS\MatchRoute.lnk"
    Delete "$INSTDIR\MatchRoute.exe"
    Delete "$INSTDIR\sing-box.exe"
    Delete "$INSTDIR\wintun.dll"
    Delete "$INSTDIR\LICENSES.txt"
    Delete "$INSTDIR\fakes\quic_initial_vk_com.bin"
    Delete "$INSTDIR\fakes\quic_initial_gosuslugi_ru.bin"
    Delete "$INSTDIR\fakes\quic_initial_ozon_ru.bin"
    RMDir "$INSTDIR\fakes"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"
    DeleteRegKey HKLM "${UNINSTALL_KEY}"
SectionEnd
