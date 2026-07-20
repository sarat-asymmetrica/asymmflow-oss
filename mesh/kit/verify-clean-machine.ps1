# verify-clean-machine.ps1 -- sealed-kit field verification (Phase 4, clean-VM gate).
#
# Runs on ANY Windows machine with NOTHING installed: Windows PowerShell 5.1
# ships with Windows itself, so this verifier adds no dependency the sealed
# kit doesn't already satisfy. Double-click verify_clean_machine.cmd to run.
#
# WHAT IT PROVES, in order (campaign method laws applied):
#   A. Machine cleanliness EVIDENCE -- records whether node/npm/npx/bare are
#      resolvable on this machine. Without this, a ceremony pass cannot
#      distinguish "kit is self-contained" from "machine secretly had Node"
#      (verify the probe: a run that can't report the opposite proves nothing).
#      A dev machine will report NOT CLEAN -- that is the script working.
#   B. Path hazards -- '#' anywhere in the kit path is a KNOWN Bare defect
#      (URL fragment truncation in addon resolution, merge-gate finding
#      2026-07-20): warned loudly before it wastes anyone's time.
#   C. Probe self-test -- one control run that posts nothing; the content
#      checker MUST report it as not-posted, proving the checker can go red.
#   D. The real ceremony, N times (default 16), through the REAL launcher
#      (run_bare_mesh.cmd -- the layer a client double-clicks, never
#      bare.exe directly), asserting on CONTENT ("(posted, seq N)" +
#      "Goodbye"), NEVER exit codes (they lie at three layers here --
#      CAMPAIGN_REPORT.md section 5). N>=16 because the campaign's one wrong
#      verdict was taken at N<=5 against a 1-in-4 defect (method rule 5).
#
# Everything is appended to VERIFY_EVIDENCE.txt next to the kit, with the
# full stdout of every run under verify-logs\ -- carry that whole folder
# back as the gate artifact.

param([int]$Runs = 16, [int]$TimeoutMs = 180000)

$ErrorActionPreference = 'Continue'
$kit = $PSScriptRoot
$logDir = Join-Path $kit 'verify-logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$evidence = Join-Path $kit 'VERIFY_EVIDENCE.txt'

function Log([string]$msg) {
  $line = ('[{0}] {1}' -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $msg)
  Write-Host $line
  Add-Content -Path $evidence -Value $line
}

Log '================================================================'
Log '=== AsymmFlow sealed-kit clean-machine verification -- START ==='
Log ('kit dir : ' + $kit)
Log ('windows : ' + [Environment]::OSVersion.VersionString + ' / 64-bit OS: ' + [Environment]::Is64BitOperatingSystem)
Log ('user    : ' + $env:USERNAME + ' on ' + $env:COMPUTERNAME)

# -- Phase A: machine cleanliness evidence ---------------------------------
$machineClean = $true
foreach ($tool in @('node', 'npm', 'npx', 'bare')) {
  $cmd = Get-Command $tool -ErrorAction SilentlyContinue
  if ($cmd) {
    $machineClean = $false
    Log ('A: FOUND on PATH: ' + $tool + ' -> ' + $cmd.Source)
  } else {
    Log ('A: ok, not resolvable: ' + $tool)
  }
}
if (Test-Path (Join-Path $env:ProgramFiles 'nodejs')) {
  $machineClean = $false
  Log ('A: FOUND: ' + (Join-Path $env:ProgramFiles 'nodejs') + ' exists')
}
if ($machineClean) {
  Log 'A: MACHINE IS CLEAN for the Node-free claim.'
} else {
  Log 'A: MACHINE IS NOT CLEAN -- a ceremony pass here does NOT close the clean-machine gate (expected on a dev machine; this is the script working, not failing).'
}
# Informational only: the classic "works on the dev machine" DLL. bare.exe
# may or may not need it -- recorded so a failure on a truly fresh install
# can be correlated instantly.
Log ('A: informational: vcruntime140.dll in System32 = ' + (Test-Path (Join-Path $env:WINDIR 'System32\vcruntime140.dll')))

# -- Phase B: path hazards --------------------------------------------------
if ($kit -match '#') {
  Log 'B: FATAL: the kit path contains "#" -- a KNOWN Bare runtime defect truncates module paths at "#" (URL fragment parsing). Move this folder to a path without "#" and re-run.'
  Log '=== VERDICT: NOT RUN (hazardous path) ==='
  exit 2
}
Log 'B: ok, no "#" in the kit path.'

$manifestOk = $true
foreach ($f in @('bare.exe', 'app.bundle', 'run_bare_mesh.cmd', 'dist\reducer.wasm')) {
  $p = Join-Path $kit $f
  if (Test-Path $p) {
    Log ('B: manifest ok: ' + $f + ' (' + (Get-Item $p).Length + ' bytes)')
  } else {
    $manifestOk = $false
    Log ('B: MANIFEST MISSING: ' + $f)
  }
}
if (-not $manifestOk) {
  Log '=== VERDICT: FAIL (incomplete kit -- re-copy the sealed folder, compare sizes against the build manifest) ==='
  exit 1
}
if (Test-Path (Join-Path $kit 'data')) {
  Log 'B: note: a data\ directory already exists (previous runs on this machine) -- ceremony still creates its own room per run; recorded for honesty.'
}

# -- ceremony runner --------------------------------------------------------
$launcher = Join-Path $kit 'run_bare_mesh.cmd'

function Invoke-Ceremony([string]$label, [string]$stdinText, [int]$timeout) {
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = 'cmd.exe'
  $psi.Arguments = '/d /c "' + $launcher + '"'
  $psi.WorkingDirectory = $kit
  $psi.UseShellExecute = $false
  $psi.RedirectStandardInput = $true
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.EnvironmentVariables['ASYMMFLOW_KIT_NONINTERACTIVE'] = '1'
  $proc = [System.Diagnostics.Process]::Start($psi)
  $outTask = $proc.StandardOutput.ReadToEndAsync()
  $errTask = $proc.StandardError.ReadToEndAsync()
  $proc.StandardInput.Write($stdinText)
  $proc.StandardInput.Close()
  $exited = $proc.WaitForExit($timeout)
  if (-not $exited) {
    # kill the whole tree -- killing only cmd.exe would orphan bare.exe
    & taskkill /T /F /PID $proc.Id 2>$null | Out-Null
    $result = @{ verdict = 'HANG'; out = ''; err = '' }
  } else {
    $out = $outTask.Result
    $err = $errTask.Result
    $posted = $out -match '\(posted, seq \d+\)'
    $goodbye = $out -match 'Goodbye'
    if ($posted -and $goodbye) { $v = 'OK' } else { $v = 'CONTENT_FAIL' }
    $result = @{ verdict = $v; out = $out; err = $err; posted = $posted; goodbye = $goodbye }
  }
  $logFile = Join-Path $logDir ($label + '.log')
  Set-Content -Path $logFile -Value ("--- stdout ---`r`n" + $result.out + "`r`n--- stderr ---`r`n" + $result.err)
  return $result
}

# CRLF line endings on stdin, matching what a real console sends.
$CEREMONY_STDIN = "2`r`n`r`nhello from the clean-machine verification`r`n/exit`r`n5`r`n"
$CONTROL_STDIN  = "5`r`n"

# -- Phase C: probe self-test (the checker must be able to go red) ----------
Log 'C: probe control -- a run that only closes the menu MUST be reported as not-posted...'
$ctl = Invoke-Ceremony 'control' $CONTROL_STDIN $TimeoutMs
if ($ctl.verdict -eq 'OK') {
  Log 'C: PROBE CONTROL FAILED -- the checker reported a no-post run as OK; NOTHING below can be trusted. Stop and report.'
  Log '=== VERDICT: INVALID (probe cannot go red) ==='
  exit 3
}
Log ('C: probe control correctly NOT ok (verdict=' + $ctl.verdict + ') -- the checker can go red; proceeding.')

# -- Phase D: the real ceremony, N times ------------------------------------
Log ('D: running the full ceremony ' + $Runs + 'x through the real launcher (first run may be slow: Defender scans a 45 MB unsigned exe once)...')
$tally = @{ OK = 0; HANG = 0; CONTENT_FAIL = 0 }
for ($i = 1; $i -le $Runs; $i++) {
  $r = Invoke-Ceremony ('run-' + $i.ToString('00')) $CEREMONY_STDIN $TimeoutMs
  $tally[$r.verdict] = $tally[$r.verdict] + 1
  Log ('D: run ' + $i.ToString('00') + '/' + $Runs + ': ' + $r.verdict)
}

Log ('D: TALLY  OK=' + $tally.OK + '/' + $Runs + '  HANG=' + $tally.HANG + '/' + $Runs + '  CONTENT_FAIL=' + $tally.CONTENT_FAIL + '/' + $Runs)

# -- Phase E: OPTIONAL corridor section (Sealed Corridor, SC-4) --------------
#
# OFF BY DEFAULT, DELIBERATELY. The receptionist's Round-2 protocol is the
# single-machine ceremony above and its shape is unchanged by this campaign —
# that matters, because that protocol is already queued with a field contact
# and re-teaching it would be a real cost for no gain.
#
# The genuinely two-machine corridor checks CANNOT run here at all: they need
# a second machine and a second human. Rather than fake them, this phase runs
# only what one machine can honestly answer, and it is opt-in:
#
#     set ASYMMFLOW_VERIFY_CORRIDOR=1
#     verify_clean_machine.cmd
#
# E1 records whether the kit CONTAINS the corridor's native addons at all. A
# kit missing udx-native/bare-tcp boots, renders its whole ceremony, creates
# its room, and silently never reaches the other machine — the campaign's
# signature broken-but-green shape. Cheap to check here, and this is the last
# place it can be caught before a ceremony.
#
# E2 runs menu [1] and records the corridor verdict word. READ E2's OWN
# CAVEAT BELOW BEFORE ACTING ON IT: a single connection check was measured
# false-red 1 time in 7 on the development machine, with a verified negative
# control. It is recorded as evidence for a support call, NEVER as a pass/fail
# gate, and it deliberately does not affect this script's exit code.
if ($env:ASYMMFLOW_VERIFY_CORRIDOR -eq '1') {
  Log 'E: OPTIONAL corridor section enabled (ASYMMFLOW_VERIFY_CORRIDOR=1).'

  $addonRoot = Join-Path $kit 'node_modules'
  foreach ($a in @('udx-native', 'bare-tcp', 'sodium-native', 'bare-dns')) {
    $p = Join-Path $addonRoot ($a + '\prebuilds\win32-x64\' + $a + '.bare')
    if (Test-Path $p) {
      Log ('E1: corridor addon present: ' + $a + '.bare (' + (Get-Item $p).Length + ' bytes)')
    } else {
      Log ('E1: WARNING -- corridor addon MISSING: ' + $a + '.bare. This kit can render its whole ceremony and still never reach the other machine. Do NOT run a corridor ceremony on this copy; report it.')
    }
  }

  Log 'E2: running menu [1] "Check the connection" once, for the record...'
  $conn = Invoke-Ceremony 'corridor-connection-check' "1`r`nskip`r`n5`r`n" $TimeoutMs
  $verdictWord = 'NOT CAPTURED'
  if ($conn.out -match 'CORRIDOR (GREEN|AMBER|RED)') { $verdictWord = $Matches[0] }
  Log ('E2: connection check verdict = ' + $verdictWord)
  Log 'E2: CAVEAT -- one check is NOT a verdict on the network. A single run was'
  Log 'E2:   measured false-RED 1 time in 7 on the dev machine (with a verified'
  Log 'E2:   negative control). Run it again before escalating. This line is'
  Log 'E2:   evidence for a support call and does NOT affect the exit code.'
  Log 'E2: the real end-to-end proof is a message crossing BOTH ways between two machines.'
} else {
  Log 'E: corridor section skipped (set ASYMMFLOW_VERIFY_CORRIDOR=1 to enable). Two-machine checks cannot run single-machine and are not simulated here.'
}

# -- verdict ----------------------------------------------------------------
if ($tally.OK -eq $Runs) {
  if ($machineClean) {
    Log ('=== VERDICT: PASS -- CLEAN machine, ' + $Runs + '/' + $Runs + ' content-verified ceremonies. The Node-free claim held HERE. ===')
  } else {
    Log ('=== VERDICT: KIT PASS (' + $Runs + '/' + $Runs + ') but machine NOT clean -- valid as a kit check, NOT as the clean-machine gate. ===')
  }
  exit 0
} else {
  Log '=== VERDICT: FAIL -- see verify-logs\ for the failing runs'' full output; carry VERIFY_EVIDENCE.txt + verify-logs\ back for diagnosis. ==='
  exit 1
}
