#!/usr/bin/env pwsh
# check.ps1 - Local pre-commit gate for FibGo (Windows / no-CGO hosts).
#
# Runs a REDUCED version of check.sh, adapted to PowerShell 7 and a
# Windows/no-CGO environment. Two deliberate differences from check.sh:
#   - tests run WITHOUT the race detector, which requires a C toolchain (CGO)
#     that is typically unavailable here;
#   - there is no step 3b: the `-tags gmp` build/vet/test that check.sh runs
#     when the libgmp headers are present has no counterpart here.
# The two scripts are therefore NOT equivalent gates.
#
# Steps (stop at first hard failure):
#   1. go build ./...
#   2. go vet ./...
#   3. go test -coverprofile ./...  (no -race; Windows/no-CGO; coverage floor derived from this run)
#   4. golangci-lint run ./...  (HARD — see below)
#   5. coverage floor (>= 80% on the module total)
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
# Requires golangci-lint v2 (config schema v2). Install:
#   go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
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

$CoverageFloor = 80.0
$GolangciInstall = 'go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest'

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

# 3. Tests + coverage (no -race on Windows/no-CGO) — single run, one profile
# Note: pass the flag value as a separate argument. PowerShell 7 mis-parses the
# "-coverprofile=coverage.out" form (it splits on '=' and treats ".out" as a
# package), so the equals form is intentionally avoided here.
Invoke-HardStep -Name "go test -coverprofile coverage.out ./... (no -race)" -Action { go test -coverprofile coverage.out ./... }

# 4. Lint (HARD — GATE-01)
Write-Step "golangci-lint run ./..."
$golangci = Get-Command golangci-lint -ErrorAction SilentlyContinue
if ($null -eq $golangci) {
    Write-Host "FAIL: golangci-lint not found on PATH." -ForegroundColor Red
    Write-Host "      Static analysis is part of the hard gate; install it with:" -ForegroundColor Red
    Write-Host "        $GolangciInstall" -ForegroundColor Red
    exit 1
}
golangci-lint run ./...
# Any non-zero exit fails the gate: 1 means findings, higher codes mean the
# linter could not analyze the module at all (the GATE-01 failure mode).
if ($LASTEXITCODE -ne 0) {
    Write-Host "FAIL: golangci-lint exited $LASTEXITCODE." -ForegroundColor Red
    if ($LASTEXITCODE -ne 1) {
        Write-Host "      Exit != 1 means the linter could not run (config or toolchain" -ForegroundColor Red
        Write-Host "      mismatch), not that your code has findings. Reinstall with:" -ForegroundColor Red
        Write-Host "        $GolangciInstall" -ForegroundColor Red
    }
    exit 1
}
Write-Host "OK: golangci-lint" -ForegroundColor Green

# 5. Coverage floor (>= 80% on the module total) — derived from the profile above
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

# Summary
Write-Host ""
Write-Host "================ summary ================"
Write-Host "build/vet/test/coverage: PASS"
Write-Host "lint:                    PASS"
Write-Host "========================================"

# Every step above is hard (GATE-01): reaching here means all of them passed.
Write-Host "Overall: PASS" -ForegroundColor Green
exit 0
