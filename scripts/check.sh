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
# Lint behaviour: golangci-lint is ADVISORY. Residual findings are documented
# intentional exceptions (A4-11 math annotations, A4-12 benign shadow/prealloc,
# errcheck in tests). Lint output is shown for review but does NOT fail this
# script; the hard gate is build/vet/test/coverage.

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
