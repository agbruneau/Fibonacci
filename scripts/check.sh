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
#   4. golangci-lint run ./...  (only if the binary is present; soft-reported)
#   5. coverage floor (>= 80% on the module total)
#
# --coverage-only: skip to a no-race test run + the coverage floor (used by
# `make coverage-check` so the floor has a single source of truth).
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
# to fix, not a pre-approved exception.

set -euo pipefail

COVERAGE_FLOOR=80.0
lint_failed=0
golangci_present=1

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

# 4. Lint (soft: reported but distinct from the hard gate)
step "golangci-lint run ./..."
if command -v golangci-lint >/dev/null 2>&1; then
    # Disable -e locally so a lint failure does not abort the script.
    if golangci-lint run ./...; then
        echo "OK: golangci-lint"
    else
        lint_failed=1
        echo "LINT FAIL: golangci-lint reported issues."
        echo "       (reported separately from the build/test/coverage gate)"
    fi
else
    golangci_present=0
    echo "WARN: golangci-lint not found on PATH; skipping lint."
fi

# 5. Coverage floor (>= 80% on the module total) — derived from the profile above
check_coverage_floor

# Summary
echo ""
echo "================ summary ================"
echo "build/vet/test/coverage: PASS"
if [ "${golangci_present}" -eq 0 ]; then
    echo "lint:                    SKIPPED (golangci-lint absent)"
elif [ "${lint_failed}" -eq 1 ]; then
    echo "lint:                    ADVISORY (review findings above)"
else
    echo "lint:                    PASS"
fi
echo "========================================"

# Lint is advisory (see header): it never gates this script. Reaching here means
# the hard gate (build/vet/test/coverage) passed.
echo "Overall: PASS"
if [ "${lint_failed}" -eq 1 ]; then
    echo "(golangci-lint reported advisory findings above; review them.)"
fi
exit 0
