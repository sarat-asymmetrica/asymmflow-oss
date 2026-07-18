Unicode true

####
## project.nsi — AsymmFlow Windows installer (per-user, no UAC).
##
## THREE-PLANE DOCTRINE (see pkg/infra/deploy):
##   CODE PLANE     %LOCALAPPDATA%\Programs\<slug>\        this script owns it fully;
##                   replaced wholesale on every install/update.
##   IDENTITY PLANE %APPDATA%\Asymmetrica\<slug>\identity\  this script seeds it ONCE,
##                   IF-ABSENT ONLY (overlay.json + ssot\). A present identity plane
##                   is a human's edits and is NEVER touched again.
##   DATA PLANE     %APPDATA%\Asymmetrica\<slug>\data\      this script NEVER references
##                   it, writes to it, or deletes it — anywhere, install or uninstall.
##                   The app's own update contract (EnsureDatabase, pkg/infra/deploy)
##                   owns seeding/migrating the data plane on first boot. The installer
##                   has no opinion about the data plane's existence. This is enforced
##                   by an automated gate (G1) that greps the generated script for any
##                   `Asymmetrica\...\data` reference — there must be none.
##
## Brand values (slug, uninstall-registry key, installer basename) come from
## branding.nsh, included below BEFORE wails_tools.nsh so our !define wins over
## wails_tools.nsh's !ifndef defaults. branding.nsh is the ONE file a fork
## replaces; nothing else in this template differs between substrate and fork
## builds.
####

!define REQUEST_EXECUTION_LEVEL "user"

!include "branding.nsh"
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

####
## Pages — deliberately minimal (§9.3): Welcome -> Install -> Finish. NO
## directory picker (the code plane is doctrine, not preference — portable
## mode is a separate, non-installer concern via portable.flag) and NO
## component picker (one artifact, one version).
####

!define MUI_WELCOMEPAGE_TITLE "Welcome to the ${INFO_PRODUCTNAME} Setup Wizard"
!define MUI_WELCOMEPAGE_TEXT "This will install ${INFO_PRODUCTNAME} ${INFO_PRODUCTVERSION} for this Windows user only. No administrator password is needed.$\r$\n$\r$\nThe application goes to your local user profile ($LOCALAPPDATA\Programs\${INSTALL_SLUG}). Your business data is stored separately under your Windows user's AppData and is never touched by this installer.$\r$\n$\r$\nClick Install to continue."
!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.

!insertmacro MUI_PAGE_INSTFILES # Installing page.

!define MUI_FINISHPAGE_TEXT "Your data is stored separately and is never modified by installation or updates."
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch ${INFO_PRODUCTNAME}"
# MUI_FINISHPAGE_RUN_NOTCHECKED intentionally NOT defined: the run checkbox
# defaults to CHECKED (§9.3 "Launch AsymmFlow" default ON).
!insertmacro MUI_PAGE_FINISH # Finished installation page.

####
## Uninstall pages — MUI_UNPAGE_CONFIRM carries the §9.3 verbatim uninstall
## text (brand-slotted via ${INSTALL_SLUG}) so the receptionist sees it before
## confirming.
####

!define MUI_UNCONFIRMPAGE_TEXT_TOP "This removes the application only. Your business data, documents and backups remain at %APPDATA%\Asymmetrica\${INSTALL_SLUG} and will be picked up by any future reinstall."
!insertmacro MUI_UNPAGE_CONFIRM # Uninstall confirmation page (carries the text above).
!insertmacro MUI_UNPAGE_INSTFILES # Uninstalling page

!insertmacro MUI_LANGUAGE "English" # Set the Language of the installer

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INSTALLER_BASENAME}-${INFO_PRODUCTVERSION}.exe"
InstallDir "$LOCALAPPDATA\Programs\${INSTALL_SLUG}" # Per-user code plane. NOT $PROGRAMFILES.
ShowInstDetails show # This will always show the installation details.

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext # per-user (REQUEST_EXECUTION_LEVEL "user" -> SetShellVarContext current)

    !insertmacro wails.webview2runtime

    # --- Code plane: the executable ---
    SetOutPath $INSTDIR
    !insertmacro wails.files

    # --- Code plane: packaged synthetic-canon seed DB. This lives inside the
    #     CODE plane ($INSTDIR\data\ph_holdings.db, where deploy.PackagedSeedPath()
    #     already looks) — it is NOT the data plane. The app's own update
    #     contract copies it into the (separate, %APPDATA%-rooted) data plane on
    #     first boot if that data plane is absent. This installer does not touch
    #     %APPDATA%\Asymmetrica\...\data anywhere.
    SetOutPath "$INSTDIR\data"
    File "payload\ph_holdings.db"

    # --- Code plane: First-Run Checklist (smoke checklist v2), shortcut below ---
    SetOutPath $INSTDIR
    File "first_run_checklist.txt"

    # --- Code plane: deployment.json is the AUTHORITATIVE slug source
    #     deploy.DeploymentSlug() reads at runtime. Written on every install.
    FileOpen $0 "$INSTDIR\deployment.json" w
    FileWrite $0 '{$\"slug$\": $\"${INSTALL_SLUG}$\"}'
    FileClose $0

    # --- Identity plane: seed IF-ABSENT ONLY. A present identity plane may
    #     carry a human's hand-edited overlay.json and is never overwritten.
    #     (%APPDATA%\Asymmetrica\<slug>\identity — NOT the data plane.)
    IfFileExists "$APPDATA\Asymmetrica\${INSTALL_SLUG}\identity\overlay.json" identity_present identity_absent
    identity_absent:
        CreateDirectory "$APPDATA\Asymmetrica\${INSTALL_SLUG}\identity"
        CreateDirectory "$APPDATA\Asymmetrica\${INSTALL_SLUG}\identity\ssot"
        SetOutPath "$APPDATA\Asymmetrica\${INSTALL_SLUG}\identity"
        File "payload\overlay.json"
        SetOutPath "$APPDATA\Asymmetrica\${INSTALL_SLUG}\identity\ssot"
        # /nonfatal: the substrate ships no letterheads (empty payload\ssot); the
        # fork ships PH letterheads. Empty glob -> warning, not a build error.
        File /nonfatal "payload\ssot\*.*"
        DetailPrint "Identity plane seeded (first install)."
        Goto identity_done
    identity_present:
        DetailPrint "Identity plane already present — left untouched."
    identity_done:

    SetOutPath $INSTDIR

    # --- Shortcuts ---
    CreateDirectory "$SMPROGRAMS\${INFO_PRODUCTNAME}"
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}\First-Run Checklist.lnk" "$INSTDIR\first_run_checklist.txt"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" # default ON, no checkbox (§9.7#1)

    # --- Uninstall registry entry: HKCU (per-user install -> per-user uninstall
    #     entry). wails.writeUninstaller hardcodes HKLM, so we do not call it —
    #     we reuse the ${UNINST_KEY} subkey path (from wails_tools.nsh) under
    #     HKCU instead, and write WriteUninstaller ourselves.
    SetRegView 64
    WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "${UNINST_KEY}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"
    WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
    WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKCU "${UNINST_KEY}" "EstimatedSize" "$0"

    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    # Code plane ONLY. No %APPDATA%\Asymmetrica\... reference of any kind here —
    # neither identity nor data plane is touched by uninstall (§9.3 "NOTHING
    # else"). Deliberately NOT the default wails template's
    # `RMDir /r "$AppData\<exe>"` WebView2-datapath line, which would reach
    # into %APPDATA% and is forbidden by doctrine.
    SetOutPath "$TEMP" # leave $INSTDIR so it is not the CWD (else RMDir /r cannot remove it)
    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}\${INFO_PRODUCTNAME}.lnk"
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}\First-Run Checklist.lnk"
    RMDir "$SMPROGRAMS\${INFO_PRODUCTNAME}"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    SetRegView 64
    DeleteRegKey HKCU "${UNINST_KEY}"

    Delete "$INSTDIR\uninstall.exe" # already gone via RMDir /r above; harmless no-op if so
SectionEnd
