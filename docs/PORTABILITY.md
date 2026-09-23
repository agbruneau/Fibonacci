# Portability — FibCalc

Reference document for the supported OS/arch matrix, the platform
*fallbacks*, and the per-target build chain. Audit-PRD P1-09 /
E10-R5 / Sprint S4-T5.

## 1. Supported matrix

| OS | Architecture | CGO | Race detector | Local verification |
|---|---|---|---|---|
| Linux | amd64 | ✅ required for `-race` and `gmp` | ✅ via gcc | `make test` (native race) |
| Linux | arm64 | ❌ disabled | ❌ (cross-compile only) | `make build-all` (build only) |
| Windows | amd64 | ✅ required for `-race` (via MinGW) | ✅ via MinGW gcc if installed | `make test` or `go test -race` |
| Windows | arm64 | ❌ disabled when cross-compiling | ❌ (cross-compile only) | `make build-windows-arm64` (build only) |
| macOS | amd64 | ✅ for `-race` | ✅ via clang | `make test` (native race) |
| macOS | arm64 (Apple Silicon) | ❌ disabled when cross-compiling | ❌ (cross-compile only) | `make build-all` (build only) |
| WASI / js | — | — | — | not supported — `go build ./...` fails in the third-party TUI stack (see §6) |

**32-bit: compiles, but is neither tested nor distributed.** Since 2026-09-07
(audit TYP-01), `GOOS=linux GOARCH=386 go build ./...`, `GOARCH=arm` and
`GOARCH=arm64` all exit 0, and CI guards these three targets
(`.github/workflows/ci.yml`, job `cross-build`).

What blocked it was not a design limit but a **compile
error**: `maxReasonableWords = 1 << 60`, declared twice
(`internal/fibonacci/memory/arena.go` and `internal/fibonacci/fastdoubling.go`),
does not fit in a 32-bit `int`. The constant is now single and
relative to the word size:
`memory.MaxReasonableWords = 1 << (bits.UintSize - 4)`, i.e. the same value
`1 << 60` on 64 bits. The `float64 → int` conversion guard in
`acquireSizingForN` now compares against `math.MaxInt` rather than `math.MaxInt64`,
which overflows at 2³¹ on these targets.

**What "compiles" does not mean.** No 32-bit target is built
by `make build-all`, none is tested, and the useful range there is bounded by
`int`: the arena caps at `1 << 28` words. The CI job checks compilation, not
behavior. The mention of `386` in §2.2 describes only the
`runtime.GOARCH` branch of `DetectHardwareHeuristic()`; it is not a
statement of support.

## 2. Platform fallbacks

### 2.1 `internal/bigfft/arith.go`

- `arith.go` (portable, no build tag — merge of the former
  `arith_amd64.go`/`arith_generic.go` split, audit FFT-06): exported wrappers
  `AddVV`, `SubVV`, `AddMulVVW`, which delegate to the internal routines of
  `math/big` via `go:linkname`. The `go:linkname` declarations live in
  `arith_decl.go` (common to all architectures), which also covers
  `addVW`, `subVW`, `shlVU`. No original assembly in this repository:
  the optimized assembly in use is `math/big`'s, for all
  architectures (amd64, arm64, riscv64, ppc64le, etc.).
- **Pure-Go fallback** (EVAL-21): `internal/bigfft/arith_purego.go`
  (`//go:build purego`) implements in pure Go, with `math/bits`, the six
  functions that `arith_decl.go` (now `//go:build !purego`) links through
  `go:linkname` — `addVV`, `subVV`, `addVW`, `subVW`, `shlVU`, `addMulVVW`.
  It is the way out when a Go release renames or removes one of these
  `math/big` internals. `go test -tags purego ./internal/bigfft/` passes
  (153 tests), and the CI `cross-build` job runs it. The default path
  remains `go:linkname`.

**Consequence**: the binary built for `linux/arm64` or `darwin/arm64`
is functionally equivalent; raw arithmetic performance is
slightly lower (5-10% gap expected on very large `big.Int`s,
not formally profiled to date).

### 2.2 `internal/config/hardware.go`

- No `//go:build`: `DetectHardwareHeuristic()` branches at run time on
  `runtime.GOARCH` and consults `golang.org/x/sys/cpu` (`HasAVX512F`,
  `HasAVX2`) only for `amd64`/`386`; every other architecture stays at
  `SIMDNone`. **No code path in `internal/bigfft` depends on it**: the
  FFT dispatch never consults `SIMDKind`. The result has two
  consumers, both in `internal/config`:
  - `HeuristicKey()` (`hardware.go:HardwareHeuristic.HeuristicKey`) /
    `CurrentHardwareHeuristicKey()` (`hardware.go:CurrentHardwareHeuristicKey`) —
    calibration profile invalidation (a profile calibrated on a different SIMD
    class is rejected);
  - **the three adaptive threshold estimators**, which branch directly
    on `h.SIMD`: `thresholds.go:estimateParallelThresholdForHeuristic`
    (parallel: −512 / −256 bits if NumCPU ≥ 8),
    `thresholds.go:estimateFFTThresholdForHeuristic` (FFT: 460,000 / 480,000 /
    500,000 bits for AVX512 / AVX2 / other),
    `thresholds.go:estimateStrassenThresholdForHeuristic` (Strassen: 224 / 240 /
    256 bits if NumCPU ≥ 4).
- On non-amd64/386 architectures, no advanced SIMD detection;
  only the `NumCPU`/`GOARCH` classification applies.

### 2.3 GMP backend (`build tag gmp`)

- Enabled via `go build -tags gmp`; requires `libgmp-dev` at build
  and run time.
- Disabled by default. The `Dockerfile` (production image) is
  `CGO_ENABLED=0` with no `apt` package: it cannot build this backend.
  Only `.devcontainer/devcontainer.json` installs `libgmp-dev` (development
  workstation); the `gmp` tag must then be added explicitly to the
  local build command.

## 3. Race detector

The Go race detector requires CGO. The canonical `make test` target runs
`go test -race`, which therefore needs a C compiler:

- **Linux + macOS**: CGO via native gcc/clang. `make test` (with `-race`)
  works out of the box.
- **Windows**: CGO via MinGW if the contributor installs it locally.
  Otherwise, `make test` fails for lack of gcc. On a plain Windows workstation
  without gcc, use the target without `-race`, **`make test-win`** (equivalent to
  `go test -v -cover ./...`). `-race` remains **recommended**: run it
  via WSL or on a Linux/macOS workstation.
- **`scripts/check.ps1` is no longer a fallback without `-race`** (2026-09-03,
  [ADR-0010 D4](adr/0010-audit-2026-09-decisions.md)): it probes `CGO_ENABLED`
  and the presence of a C compiler, and enables `-race` when both are
  present — as checked on this Windows host: **22 packages green**, re-run on
  2026-09-07 (`go1.27.0 windows/amd64`, `CGO_ENABLED=1`, gcc MinGW-W64 16.1.0;
  `go test -race -count=1 ./...` exits 0, no *data race*). Without a C toolchain,
  it falls back to the same suite without `-race`. The former wording described
  an installation, not a platform limit.

> Summary: `make test` = full suite with `-race` (CGO and a C compiler
> required); `make test-win` = explicit fallback without `-race`;
> `scripts/check.ps1` = `-race` if the host can, otherwise without.

For cross-compiled builds (`linux/arm64`, `darwin/arm64`), the race
detector is not run — only CGO-free **compilability** is
checked by `make build-all`.

## 4. Per-target build procedure

### Linux/amd64 (primary target)

```bash
make build              # standard
make build-pgo          # with PGO profile
```

### Linux/arm64 (cross)

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/fibcalc-linux-arm64 ./cmd/fibcalc
```

### macOS/arm64 (Apple Silicon, cross)

```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o build/fibcalc-darwin-arm64 ./cmd/fibcalc
```

### Windows/amd64

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/fibcalc.exe ./cmd/fibcalc
```

### Reproducible container

```bash
docker build -t fibcalc:local .
docker run --rm fibcalc:local --help
```

The distroless image ships only the statically linked binary (no libc
runtime, no shell). To debug: replace the final stage with
`gcr.io/distroless/base-debian12:debug`.

## 5. Local verification

`make build-all` runs `go build` for the following targets and must
pass without error before any commit touching `internal/bigfft/` or
`internal/fibonacci/`:

- `linux/amd64`
- `linux/arm64`
- `windows/amd64`
- `windows/arm64`
- `darwin/amd64`
- `darwin/arm64`

A regression introducing an amd64-only dependency not guarded by
`//go:build` will immediately fail one of the targets above.

Verified on 2026-09-04 from a Windows host, by replaying the six `go build`
commands that `build-all` issues (`GOOS=<os> GOARCH=<arch> go build -trimpath ./cmd/fibcalc`):
all six targets exit 0.

## 6. Known limitations

- **No cross-arch bench**: the figures in
  `docs/audits/bench-baseline.txt` come from an amd64 host. An
  arm64 benchmark needs an Apple Silicon or ARM Linux workstation.
- **GMP not tested when cross-compiling**: the `gmp` tag requires CGO, so no
  cross target exercises it. The only **automated** check is step 3b of
  `scripts/check.sh`, which triggers only if `/usr/include/gmp.h` or
  `/usr/include/x86_64-linux-gnu/gmp.h` exists. The **first** guard is specific
  neither to a distribution nor to an architecture — `/usr/include/gmp.h` is
  the default header path of a `libgmp` installed by the package
  manager, amd64 and arm64 alike; the second covers only the Debian/Ubuntu amd64
  multiarch path. What the repository supports claiming is the shape of the
  guard (`scripts/check.sh`, step 3b), not the list of hosts where it passes:
  no host is exercised here. `check.ps1` has no equivalent of this step.
  A manual build
  `go build -tags gmp` remains possible on any CGO host with libgmp, macOS included
  (`brew install gmp`), but no script in the repository checks it.
- **WebAssembly**: not supported, but **not** because of the compute core.
  Verified on 2026-08-09, re-verified on 2026-09-04 (`go1.27.0`, same results):
  the compute core compiles for both WASM targets —
  `GOOS=js GOARCH=wasm go build ./internal/bigfft/ ./internal/fibonacci/... ./internal/progress/`
  **passes**, and so does the same command under `GOOS=wasip1 GOARCH=wasm` — even
  though these packages are precisely
  the ones that import `runtime/debug` (`import` blocks of `internal/bigfft/fft.go`
  and of `internal/fibonacci/memory/gc_control.go`) and that carry the
  post-condition panic assertions. What breaks is the third-party TUI stack:
  `GOOS=js GOARCH=wasm go build ./...` fails in
  `github.com/charmbracelet/bubbletea` (`p.listenForResize undefined`,
  `undefined: openInputTTY`, …) and `GOOS=wasip1 GOARCH=wasm go build ./...`
  fails in `github.com/muesli/termenv` (`output.ColorProfile undefined`, …).
  A WASM port would therefore go through excluding `internal/tui` and its
  dependencies, not through rewriting the core.
- **`unsafe`**: production code does **not** use `unsafe.Pointer`; the
  only use of `unsafe` is `unsafe.Sizeof` (`internal/bigfft/fft.go:_W`), plus
  the blank import required by `go:linkname` (`import` block of `internal/bigfft/arith_decl.go`);
  the repository's only `unsafe.Pointer` is in a test
  (`internal/fibonacci/memory/arena_test.go`, `TestCalculationArena_MultipleAllocs_NoAliasing`).
- **32-bit**: compiles since 2026-09-07, not tested and not distributed; see §1.
