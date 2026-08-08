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
#   4. golangci-lint run ./...  (only if the binary is present; soft-reported)
#   5. coverage floor (>= 80% on the module total)
#
# Lint behaviour: golangci-lint is reported separately from the hard gate. Its
# output is shown and its status appears in the summary, but it does NOT fail
# this script; the hard gate is build/vet/test/coverage. The expected state is
# zero findings. Tolerated cases live in TWO places, not one:
#   1. .golangci.yml, in two distinct sections:
#      - issues.exclude-rules (revive stutter, staticcheck SA1019, SA6002 in
#        bigfft pools, three named gocyclo overages, and the _test.go
#        relaxations);
#      - linters-settings.gosec.excludes, which is where G104 and G115 live —
#        NOT in exclude-rules;
#   2. in-source annotations, which .golangci.yml does not list at all —
#      four inline //nolint directives (gocognit in bigfft/fft_recursion.go,
#      gocritic in cli/completion/bash.go and fibonacci/common.go, unparam in
#      fibonacci/fft.go) plus the #nosec G115 / G304 annotations across
#      internal/bigfft, internal/calibration/profile.go and
#      cmd/generate-golden/main.go.
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
$lintFailed = $false

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

# 4. Lint (soft: reported but distinct from the hard gate)
Write-Step "golangci-lint run ./..."
$golangci = Get-Command golangci-lint -ErrorAction SilentlyContinue
if ($null -eq $golangci) {
    Write-Host "WARN: golangci-lint not found on PATH; skipping lint." -ForegroundColor Yellow
} else {
    golangci-lint run ./...
    if ($LASTEXITCODE -ne 0) {
        $lintFailed = $true
        Write-Host "LINT FAIL: golangci-lint reported issues (exit $LASTEXITCODE)." -ForegroundColor Yellow
        Write-Host "       (reported separately from the build/test/coverage gate)" -ForegroundColor Yellow
    } else {
        Write-Host "OK: golangci-lint" -ForegroundColor Green
    }
}

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
if ($null -eq $golangci) {
    Write-Host "lint:                    SKIPPED (golangci-lint absent)"
} elseif ($lintFailed) {
    Write-Host "lint:                    ADVISORY (review findings above)"
} else {
    Write-Host "lint:                    PASS"
}
Write-Host "========================================"

# Lint is advisory (see header): it never gates this script. Reaching here means
# the hard gate (build/vet/test/coverage) passed.
Write-Host "Overall: PASS" -ForegroundColor Green
if ($lintFailed) {
    Write-Host "(golangci-lint reported advisory findings above; review them.)" -ForegroundColor Yellow
}
exit 0
