####
## branding.nsh — the ONE file a fork overrides to re-brand the installer.
##
## FORK OVERRIDE POINT: the PH fork (and any future fork) replaces this file
## wholesale with its own slug/key/basename (and swaps ..\icon.ico beside it).
## Nothing else in project.nsi differs between substrate and fork builds —
## every brand-specific value used by the installer lives here.
####

# INSTALL_SLUG is the deployment slug: it names the per-user code-plane
# directory ($LOCALAPPDATA\Programs\${INSTALL_SLUG}) AND is written into
# deployment.json, which is the AUTHORITATIVE source deploy.DeploymentSlug()
# reads at runtime. It also roots the identity plane
# (%APPDATA%\Asymmetrica\${INSTALL_SLUG}\identity) referenced in the
# if-absent seed section below and in the uninstall dialog text.
!define INSTALL_SLUG "AsymmFlow-Dev"

# UNINST_KEY_NAME must be DISTINCT per brand so a dev install and a fork
# install (e.g. AsymmFlow-PH) coexist on the same machine as two independent
# HKCU uninstall entries and two independent code-plane directories.
!define UNINST_KEY_NAME "AsymmetricaAsymmFlowDev"

# INSTALLER_BASENAME names the emitted Setup.exe: ${INSTALLER_BASENAME}-${INFO_PRODUCTVERSION}-setup.exe
!define INSTALLER_BASENAME "AsymmFlow-Setup"
