#!/usr/bin/env bash
# check.sh - Local pre-commit gate for FibGo (CGO / Linux / macOS hosts).
#
# Runs the canonical pre-commit checks in sequence, stopping at the first hard
# failure. Tests run WITH the race detector (-race), which requires a C
# toolchain (CGO). On Windows / no-CGO hosts use scripts/check.ps1 instead.
#
# Steps:
#   1. go build ./...
#   2. go vet ./...
#   3. go test -race -coverprofile=coverage.out ./...  (coverage floor derived from this run)
#   3b. gmp build tag: build+vet+test -tags gmp (hard when libgmp present, else skipped)
#   4. golangci-lint run ./...  (HARD — see below)
#   5. coverage floor (>= 80% on the module total)
#
# --coverage-only: skip to a no-race test run + the coverage floor (used by
# `make coverage-check` so the floor has a single source of truth).
#
# Lint behaviour (changed by audit GATE-01, 2026-09-03): golangci-lint is now
# part of the HARD gate. It previously ran "soft" — findings and even outright
# execution failures were printed, then the script wrote "Overall: PASS" and
# exited 0. That masked a total loss of static analysis: the pinned v1.64.8
# binary could not analyze the module under a go1.27 toolchain (export data
# version 4), so for an unknown period only `go vet` was actually running.
# A missing binary is ALSO a hard failure now, for the same reason: a gate that
# silently checks nothing is worse than no gate.
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
# to fix, not a pre-approved exception.

set -euo pipefail

COVERAGE_FLOOR=80.0
GOLANGCI_INSTALL='go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest'

# Run from repo root regardless of the caller's cwd.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

step() {
    printf '\n==> %s\n' "$1"
}

check_coverage_floor() {
    step "coverage floor (>= ${COVERAGE_FLOOR}%)"
    total="$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$3); print $3}')"
    if [ -z "${total}" ]; then
        echo "FAIL: could not parse total coverage from coverage.out"
        exit 1
    fi
    printf 'Total coverage: %s%%\n' "${total}"
    if awk -v t="${total}" -v f="${COVERAGE_FLOOR}" 'BEGIN { exit (t+0 < f+0) }'; then
        printf 'OK: coverage %s%% >= %s%%\n' "${total}" "${COVERAGE_FLOOR}"
    else
        printf 'FAIL: coverage %s%% < %s%%\n' "${total}" "${COVERAGE_FLOOR}"
        exit 1
    fi
}

# --coverage-only: single-source entry point for `make coverage-check`
# (BUILD-03) — regenerate the profile (no -race: works on no-CGO hosts too)
# and apply the same floor as the full gate, nothing else.
if [ "${1:-}" = "--coverage-only" ]; then
    step "go test -coverprofile=coverage.out ./... (coverage-only mode)"
    go test -coverprofile=coverage.out ./... >/dev/null
    check_coverage_floor
    exit 0
fi

# 1. Build
step "go build ./..."
go build ./...
echo "OK: go build"

# 2. Vet
step "go vet ./..."
go vet ./...
echo "OK: go vet"

# 3. Tests + coverage (with race detector; requires CGO) — single run, one profile
step "go test -race -coverprofile=coverage.out ./..."
go test -race -coverprofile=coverage.out ./...
echo "OK: go test -race"

# 3b. GMP build tag (hard when libgmp is available, skipped otherwise).
# The gmp backend (CGO + libgmp) is not compiled by the default steps above;
# it has silently broken before (globalFactory, fixed in the 2026-07 audit).
# Header search covers the Debian/Ubuntu multiarch path as well as /usr/include.
step "gmp build tag (-tags gmp)"
if [ -f /usr/include/gmp.h ] || [ -f /usr/include/x86_64-linux-gnu/gmp.h ]; then
    go build -tags gmp ./...
    go vet -tags gmp ./internal/fibonacci/
    go test -tags gmp -race -count=1 ./internal/fibonacci/
    echo "OK: gmp build tag"
else
    echo "SKIP: gmp (libgmp headers not found; apt-get install libgmp-dev)"
fi

# 4. Lint (HARD — GATE-01)
step "golangci-lint run ./..."
if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "FAIL: golangci-lint not found on PATH."
    echo "      Static analysis is part of the hard gate; install it with:"
    echo "        ${GOLANGCI_INSTALL}"
    exit 1
fi
# Any non-zero exit fails the gate: 1 means findings, higher codes mean the
# linter could not analyze the module at all (the GATE-01 failure mode).
if golangci-lint run ./...; then
    echo "OK: golangci-lint"
else
    lint_status=$?
    echo "FAIL: golangci-lint exited ${lint_status}."
    if [ "${lint_status}" -ne 1 ]; then
        echo "      Exit != 1 means the linter could not run (config or toolchain"
        echo "      mismatch), not that your code has findings. Reinstall with:"
        echo "        ${GOLANGCI_INSTALL}"
    fi
    exit 1
fi

# 5. Coverage floor (>= 80% on the module total) — derived from the profile above
check_coverage_floor

# Summary
echo ""
echo "================ summary ================"
echo "build/vet/test/coverage: PASS"
echo "lint:                    PASS"
echo "========================================"

# Every step above is hard (GATE-01): reaching here means all of them passed.
echo "Overall: PASS"
exit 0
