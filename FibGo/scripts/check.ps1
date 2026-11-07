#!/usr/bin/env pwsh
# check.ps1 - Local pre-commit gate for FibGo (Windows hosts).
#
# Runs a REDUCED version of check.sh, adapted to PowerShell 7 and a Windows
# environment. One deliberate difference from check.sh remains:
#   - there is no step 3b: the `-tags gmp` build/vet/test that check.sh runs
#     when the libgmp headers are present has no counterpart here.
# The race detector is no longer a difference: step 3 probes for CGO and a C
# compiler and enables -race when both are present (verified 2026-09-03: all 21
# packages pass -race on a Windows host with a toolchain installed). The header
# used to assert flatly that Windows could not run it, which was a statement
# about one machine's setup dressed up as a platform limitation.
#
# Steps (stop at first hard failure):
#   1. go build ./...
#   2. go vet ./...
#   3. go test -shuffle=on -count=1 -coverprofile ./...  (with -race when the
#      host has CGO and a C compiler; coverage floor derived from this run)
#   4. golangci-lint run ./...  (HARD — see below)
#   5. coverage floor (>= 90% on the module total)
#   6. govulncheck ./...  (HARD — audit DEP-01 / SEC-01)
#
# -shuffle=on randomizes test order within each package (audit TST-03). A
# failure that appears only under shuffling is a real finding: the test depends
# on hidden shared state. -count=1 disables the result cache.
#
# Lint behaviour (changed by audit GATE-01, 2026-09-03): golangci-lint is now
# part of the HARD gate. It previously ran "soft" — findings and even outright
# execution failures were printed, then the script wrote "Overall: PASS" and
# exited 0. That masked a total loss of static analysis: the pinned v1.64.8
# binary could not analyze the module under a go1.27 toolchain (export data
# version 4), so for an unknown period only `go vet` was actually running on
# this host. A missing binary is ALSO a hard failure now, for the same reason:
# a gate that silently checks nothing is worse than no gate.
#
# Tool versions are NOT installed by hand any more (audit PRO-02): they live in
# scripts/tools.env and are invoked with `go run <pkg>@<version>`, which rebuilds
# them with the current Go toolchain. That closes the GATE-01 failure class — an
# installed binary compiled against an older Go silently stops working, which is
# what happened to golangci-lint in 2026-09 and to govulncheck, gosec and
# staticcheck all at once on 2026-09-07.
#
# The expected state is zero findings. Tolerated cases live in TWO places:
#   1. .golangci.yml, in two distinct sections:
#      - linters.exclusions.rules (revive stutter, staticcheck SA1019, SA6002 in
#        bigfft pools, three named gocyclo overages, and the _test.go
#        relaxations — which since the v2 migration also cover noctx, whose
#        os/exec check the e2e suites trip deliberately);
#      - linters.settings.gosec.excludes, which is where G104 and G115 live —
#        NOT in exclusions.rules;
#   2. in-source annotations, which .golangci.yml does not list at all —
#      inline //nolint directives (gocognit in bigfft/fft_recursion.go,
#      gocritic in cli/completion/bash.go, fibonacci/common.go and
#      fibonacci/fibonacci_property_test.go, unparam in fibonacci/fft.go) plus
#      the #nosec G115 / G304 annotations across internal/bigfft,
#      internal/calibration/profile.go and cmd/generate-golden/main.go.
# Anything golangci-lint still reports outside those two sets is a real finding
# to fix, not a pre-approved exception. Note: `.gitattributes` pins *.go to LF
# in the working tree, so the old Windows-CRLF gofmt false positives no longer
# occur.

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

# Run from repo root regardless of the caller's cwd.
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

$CoverageFloor = 90.0

# Pinned tool versions — same file check.sh, the Makefile and the CI read.
# Format is KEY=value, one per line, '#' comments; parse it rather than
# duplicating the versions here.
$ToolsEnv = @{}
foreach ($line in Get-Content (Join-Path $PSScriptRoot 'tools.env')) {
    if ($line -match '^\s*([A-Z_][A-Z0-9_]*)=(.+?)\s*$') {
        $ToolsEnv[$Matches[1]] = $Matches[2]
    }
}
foreach ($required in @('GOLANGCI_LINT', 'GOVULNCHECK')) {
    if (-not $ToolsEnv.ContainsKey($required)) {
        Write-Host "FAIL: $required missing from scripts/tools.env" -ForegroundColor Red
        exit 1
    }
}

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Invoke-HardStep {
    param(
        [string]$Name,
        [scriptblock]$Action
    )
    Write-Step $Name
    & $Action
    if ($LASTEXITCODE -ne 0) {
        Write-Host "FAIL: $Name (exit $LASTEXITCODE)" -ForegroundColor Red
        exit 1
    }
    Write-Host "OK: $Name" -ForegroundColor Green
}

# 1. Build
Invoke-HardStep -Name "go build ./..." -Action { go build ./... }

# 2. Vet
Invoke-HardStep -Name "go vet ./..." -Action { go vet ./... }

# 3. Tests + coverage — single run, one profile
# Note: pass the flag value as a separate argument. PowerShell 7 mis-parses the
# "-coverprofile=coverage.out" form (it splits on '=' and treats ".out" as a
# package), so the equals form is intentionally avoided here.
#
# -race when the host can: the header used to state flatly that Windows hosts
# cannot run the race detector, which was true only while no C toolchain was
# installed. `go test -race` needs CGO, so probe for both rather than assume.
# A host with a compiler now gets the same concurrency coverage as check.sh.
$cgoEnabled = (go env CGO_ENABLED) -eq '1'
$cc = Get-Command gcc -ErrorAction SilentlyContinue
if ($null -eq $cc) { $cc = Get-Command clang -ErrorAction SilentlyContinue }
$raceArgs = @()
if ($cgoEnabled -and $null -ne $cc) {
    $raceArgs = @('-race')
    Write-Host "race detector: ENABLED (CGO_ENABLED=1, $($cc.Name) on PATH)" -ForegroundColor Green
} else {
    Write-Host "race detector: SKIPPED (needs CGO_ENABLED=1 and a C compiler on PATH)" -ForegroundColor Yellow
}
Invoke-HardStep -Name "go test $raceArgs -shuffle=on -count=1 -coverprofile coverage.out ./..." -Action { go test @raceArgs -shuffle=on -count=1 -coverprofile coverage.out ./... }

# 4. Lint (HARD — GATE-01). Built from source at the pinned version, so there is
# no PATH binary that can go stale against the toolchain.
Write-Step "golangci-lint run ./... ($($ToolsEnv['GOLANGCI_LINT']))"
go run $ToolsEnv['GOLANGCI_LINT'] run ./...
# Any non-zero exit fails the gate: 1 means findings, higher codes mean the
# linter could not analyze the module at all (the GATE-01 failure mode).
if ($LASTEXITCODE -ne 0) {
    Write-Host "FAIL: golangci-lint exited $LASTEXITCODE." -ForegroundColor Red
    if ($LASTEXITCODE -ne 1) {
        Write-Host "      Exit != 1 means the linter could not run (config mismatch, or" -ForegroundColor Red
        Write-Host "      the pinned version cannot build), not that your code has" -ForegroundColor Red
        Write-Host "      findings. Check GOLANGCI_LINT in scripts/tools.env." -ForegroundColor Red
    }
    exit 1
}
Write-Host "OK: golangci-lint" -ForegroundColor Green

# 5. Coverage floor (>= 90% on the module total) — derived from the profile above
Write-Step "coverage floor (>= $CoverageFloor%)"
$coverFunc = go tool cover -func coverage.out
if ($LASTEXITCODE -ne 0) {
    Write-Host "FAIL: go tool cover -func exit $LASTEXITCODE" -ForegroundColor Red
    exit 1
}

$totalLine = $coverFunc | Select-String -Pattern '^total:' | Select-Object -First 1
if ($null -eq $totalLine) {
    Write-Host "FAIL: could not locate 'total:' line in coverage output." -ForegroundColor Red
    exit 1
}

$match = [regex]::Match($totalLine.ToString(), '([0-9]+(?:\.[0-9]+)?)%')
if (-not $match.Success) {
    Write-Host "FAIL: could not parse coverage percentage from: $($totalLine.ToString())" -ForegroundColor Red
    exit 1
}

# Parse with InvariantCulture: `go tool cover` always emits a '.' decimal
# separator, so do not let a comma-decimal locale (e.g. fr-CA) skew the compare.
$total = [double]::Parse($match.Groups[1].Value, [System.Globalization.CultureInfo]::InvariantCulture)
Write-Host ("Total coverage: {0}%" -f $match.Groups[1].Value)
if ($total -lt $CoverageFloor) {
    Write-Host ("FAIL: coverage {0}% < {1}%" -f $match.Groups[1].Value, $CoverageFloor) -ForegroundColor Red
    exit 1
}
Write-Host ("OK: coverage {0}% >= {1}%" -f $match.Groups[1].Value, $CoverageFloor) -ForegroundColor Green

# 6. Vulnerability scan (HARD — audit DEP-01 / SEC-01). govulncheck reports only
# vulnerabilities the code can actually reach, so a finding here is actionable.
Write-Step "govulncheck ./... ($($ToolsEnv['GOVULNCHECK']))"
go run $ToolsEnv['GOVULNCHECK'] ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "FAIL: govulncheck reported reachable vulnerabilities (or could not run)." -ForegroundColor Red
    exit 1
}
Write-Host "OK: govulncheck" -ForegroundColor Green

# Summary
Write-Host ""
Write-Host "================ summary ================"
Write-Host "build/vet/test/coverage: PASS"
Write-Host "lint:                    PASS"
Write-Host "govulncheck:             PASS"
Write-Host "========================================"

# Every step above is hard (GATE-01): reaching here means all of them passed.
Write-Host "Overall: PASS" -ForegroundColor Green
exit 0
