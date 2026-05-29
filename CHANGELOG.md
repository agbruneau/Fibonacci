# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Audit remediation (May 2026)

Remédiation des 45 constats de l'audit multi-agents (rapports d'audit archivés
en historique git ; plan et décisions consolidés dans les ADR 0005-0007 + ce
changelog). Chaque constat est soit corrigé, soit
acté comme décision documentée. Nouveaux ADR : 0005 (GC concurrent), 0006
(annulation FFT), 0007 (SA6002 pool).

#### Fixed

- **A2-01 (CRITIQUE)** — Contrôle GC *concurrency-safe* : refcount package-level
  (`gc_control.go`). En mode `--algo all`, seul le 1ᵉʳ `Begin` actif désactive le
  GC / mémorise le vrai `GOGC`, seul le dernier `End` restaure et lève la limite
  mémoire. Invariant `WithGC` panic-safe préservé (ADR-0005).
- **A2-02 (MAJEUR)** — `TransformCache.logger` → `atomic.Pointer[zerolog.Logger]`
  (écriture `SetCacheLogger` / lecture hot path `logPeriodicStats`).
- **A5-01 (MAJEUR)** — Flaky `TestSaveProfile` sous Windows : `renameAtomic`
  backoff exponentiel borné (10→40 tentatives) + tolérance de l'erreur de partage
  côté écrivain dans le test.
- **A1-04 / A1-05 (MINEUR)** — Arithmétique saturante (`EstimateMemoryUsage`) et
  clamp float→int + ×15 (`AcquireStateForN` / `NewCalculationArena`) contre les
  débordements à `n` non borné.
- **A2-05 (MINEUR)** — `releaseFFTState` relâche les buffers `tmp/tmp2` au-delà de
  `maxPooledFFTTmpCap` (anti-rétention de pic).
- **A3-03 (MINEUR)** — `FFTOnlyStrategy.Multiply/Square` écrivent dans `z` via
  `bigfft.MulTo/SqrTo` (suppression alloc neuve + copie O(n)).
- **Correctness/tests** — A1-01 (cross-val régime FFT F(1M) + oracle
  `FastDoublingMod`), A1-02 (cross-val GMP, build tag `gmp`), A1-03 (oracle dans
  `FuzzFastDoublingMod`), A1-08 (assertions T1/T2/T3), A1-06 (métrique `usedFFT`
  alignée sur FK1).
- **Idiomatique** — A4-02 (dead code `formatAlgoList`), A4-03 (`RenderBrailleChart`
  cyclo 17→9), A4-04 (dead store `fermat.norm`), A4-05 (récepteur `fermat`),
  A4-06 (`run() error` generate-golden), A4-08 (octal `0o`, `if`→`switch`,
  `paramTypeCombine` ; `bash.go` `%q` *non* appliqué — annotation), A4-13
  (`#nosec G304`).
- **Structure/doc** — A5-03 (versions Go 1.26.0/toolchain 1.26.3), A5-06 (path
  module `github.com/agbruneau/FibGo`), A5-08/A5-09 (angles morts de couverture
  documentés).

#### Changed

- **A5-02 (MAJEUR) — décision assumée** : pas de CI distante réintroduite.
  Garde-fous **locaux** ajoutés à la place : cibles `make test-win` /
  `make coverage-check` (plancher 80 %) et `scripts/check.ps1` / `scripts/check.sh`.
- **A5-05 / A5-10** — `-race` documenté comme requérant CGO/WSL ; plancher de
  couverture ré-outillé localement.
- **A4-07 / A4-09 / A4-10 / A5-07** — `.golangci.yml` : exclusion *stutter* revive
  ciblée ; `misspell.locale` maintenu **US** (la mesure a infirmé la prémisse
  « britannique cohérent » : ~339 US vs ~50 UK) avec normalisation des ~50
  graphies UK résiduelles ; `.gitattributes` `*.go text eol=lf` (faux positifs
  gofmt CRLF) ; verrou schéma golangci-lint v1 documenté.
- **A3-01 / A3-04 / A3-06** — Documentation perf corrigée : le cache de
  transformées FFT ne bénéficie qu'aux chemins `bigfft.Mul/Sqr` / `FFTOnly` (pas
  au calculateur Fast Doubling par défaut) ; prudence sur l'ordre des algos à
  très grand N ; domination Karatsuba sous le seuil FFT.

#### Decisions (no-fix documenté)

- **A2-03 (MAJEUR)** — Annulation fine *intra*-multiplication FFT **reportée** au
  token par-appel (`FFTContext`, ADR-0004 §B1) ; le drapeau atomic global est
  **rejeté** (clear-race sous FFT concurrentes, ADR-0006). L'annulation grossière
  existante (entre pas + entre les 3 produits) reste fonctionnelle.
- **A4-01 (MAJEUR)** — SA6002 : micro-benchmark (ADR-0007) prouve que le fix
  préservant la signature est **alloc-neutre** ; le vrai zéro-alloc exige le
  refactor `FFTContext`. Exclusion SA6002 ciblée + ADR.
- **A2-04** (knobs `threshold` single-writer-before-use), **A2-06** (LRU correct
  par construction), **A2-07** (deux sémaphores `NumCPU`), **A3-02** (clé cache
  O(n) gatée par `MinBitLen`), **A3-05** (sous-goroutines FFT sur pool — compromis
  sécurité concurrence), **A3-07** (baseline conservatrice), **A4-11**
  (annotations mathématiques conservées), **A4-12** (shadow `err` hot path vérifié
  bénin) — commentés ou documentés sans changement fonctionnel.

#### Deferred verification (indisponible sur l'hôte Windows `CGO_ENABLED=0`)

- `-race` : A2-01/A2-02/A2-06/A4-04/A4-12 — rejouer `CGO_ENABLED=1 go test -race`
  sous Linux/WSL.
- `-tags gmp` : A1-02 — `CGO_ENABLED=1 go test -tags gmp ./internal/fibonacci/`
  avec `libgmp-dev`.
- `ns/op` fiables : A3-04/A3-07 — machine de référence Ryzen/Linux idle.

### Hardening sprint (May 2026)

Architecture / concurrency / security hardening sweep documented in
`docs/adr/0001`..`0004`. Net effect : closed every data race the dual-
audit (Claude + Gemini) had flagged, broke three upward import arrows
gated by `internal/arch_test.go`, restored panic propagation for FFT
post-condition violations, lifted `internal/cli/completion/` coverage
from 0 % to 95.7 %, and added the multi-stage `Dockerfile` /
`.devcontainer/` for reproducible builds. Working audit documents have
been purged ; the ADR series is the surviving source of truth.

### Added (hardening)

- **Concurrence safe** : `DynamicThresholdManager` fields converted to
  `atomic.Int64` / `atomic.Pointer[time.Time]` ; `bigfft` globals
  (`fftThreshold`, `parallelFFTRecursionThreshold`,
  `maxParallelFFTDepth`) routed through atomic accessors (ADR-0003) ;
  `TransformCache.cacheGate()` snapshot under RLock at 5 hot-path sites.
- **Cache FFT** : `putByKey` allocates fresh backing on eviction — no
  recycle — eliminating the use-after-free aliasing window for
  concurrent `Get()` holders.
- **FFT panic policy** : sentinel `isFermatPostConditionPanic` re-
  propagates internal invariant violations instead of masking them as
  opaque errors (ADR-0002).
- **CLI completion security** : per-shell escape helpers
  (`escapeBashDoubleQuoted`, `escapeFishSingleQuoted`,
  `escapeZshSingleQuoted`, `escapePowerShellSingleQuoted`) plus
  9 adversarial tests per shell covering `$(...)`, backticks, `;`,
  spaces, quotes, backslashes, newlines.
- **Reproducible build** : multi-stage `Dockerfile`
  (`golang:1.26-bookworm` builder + distroless runtime) and
  `.devcontainer/devcontainer.json` (VS Code, CGO + libgmp +
  benchstat pre-installed).
- **Architecture gate** : `internal/arch_test.go` blocks PRs that
  reintroduce `threshold → config`, `errors → format`, or
  `tui → fibonacci`.
- **Perf regression gate** : `.github/workflows/ci.yml` `bench` job is
  blocking ; `benchstat` + `.github/scripts/bench_gate.py` compare each
  PR to `docs/audits/bench-baseline.txt` at a 5 % threshold.
  `make bench-baseline` refreshes the file.
- **Cross-compile job** : CI builds `linux/arm64`, `darwin/arm64`,
  `darwin/amd64` on every PR.
- **Fuzz coverage extended** : new `bigfft.FuzzMul` / `FuzzSqr` targets
  validate against `math/big` ; Fibonacci fuzz bounds raised
  50 000 → 200 000 to exercise the FFT regime.
- **Golden corpus extended** : `internal/fibonacci/testdata/fibonacci_golden.json`
  gained F(50 000), F(100 000), F(200 000) under ADR-0004 §B5.
- **Observer counter** : `progress.RecoveredObserverCount()` exposes
  swallowed panics from frozen `ProgressCallback`s.
- **Documentation** : `docs/PORTABILITY.md` (matrix + fallbacks),
  `docs/adr/0001..0004`, enriched `doc.go` for `cli`, `config`,
  `orchestration`, `fibonaccitest`, and a `make stats` target as the
  canonical source for package & LOC counts.

### Removed (hardening)

- Working audit drafts at repo root (`Audit - Claude - FibGo.md`,
  `Audit - Gemini - FibGo.md`, `Audit - Global - FibGo.md`,
  `Audit - Global - FibGo - v2.md`, `Audit - PRD - FibGo.md`,
  `Audit - PRDPLan - FibGo.md`) ; the durable record now lives in
  `docs/adr/` and this CHANGELOG.
- Pre-hardening bench baseline `docs/audits/bench-baseline-pre-hardening.txt`
  (superseded by `docs/audits/bench-baseline.txt`) and the legacy
  `bench-A10-*.txt` artifacts from the previous audit cycle.

### Changed

- **Post-audit remediation period closed.** The audit-driven remediation workflow (A-NN finding tracking, "Vague A" review freeze, post-audit branch gating) is wound down. Project documentation is purged of audit scaffolding: contributor-facing docs no longer route through `audit.md` / `AuditPlanning.md` / `audit-prompt.md`, and the standard `CONTRIBUTING.md` workflow is the single source of process truth going forward. Historical CHANGELOG entries below are retained verbatim as the accurate record of what was done during remediation.
- **Project status repositioned** from `Production-Ready` to `Academic Prototype` (README status badge). Aligns the marketing claim with the actual self-description in the README intro and removes the implicit promise of automated quality gates that the CI removal (below) makes untenable.

### Removed (CI/CD retirement)

- **GitHub Actions workflows deleted** : `.github/workflows/ci.yml` (vet + golangci-lint + 3-OS race matrix + cross-compile + bench gate) and `.github/workflows/coverage.yml` (PR/push coverage with `MIN_COVERAGE=80%` floor). The accompanying regression-gate engine `.github/scripts/bench_gate.py` and the now-empty `.github/` tree are gone as well.
- **Consequence for contributors** : every gate that used to run on push/PR is now the contributor's local responsibility — `make test` (race, requires CGO/gcc), `make lint`, `make coverage`, `make build-all` for cross-compile, and `benchstat` against `docs/audits/bench-baseline.txt` for perf-sensitive changes. CLAUDE.md directive #8 is the canonical checklist.
- **Documentation refresh** : README/CLAUDE/PORTABILITY/PERFORMANCE/TESTING/BUILD/architecture docs reworded to drop references to `ci.yml`/`coverage.yml` and to reframe gates as local-discipline conventions (no behavioural change to the Go code).

### Added

- **Interactive knowledge-graph dashboard** published on GitHub Pages: <https://agbruneau.github.io/FibGo/dashboard/>. 744 nodes / 3 526 edges / 8 architectural layers / 13-step guided tour, generated from `.understand-anything/knowledge-graph.json` via the `understand-anything` plugin and bundled into `docs/dashboard/` as a static Vite build. Republish steps documented in [docs/BUILD.md — Dashboard statique (GitHub Pages)](docs/BUILD.md#dashboard-statique-github-pages).
- **Interactive TUI mode**: btop-style dashboard built with Bubble Tea (Elm architecture), featuring real-time progress charts, algorithm comparison, and keyboard navigation
- Portable arithmetic fallback for non-amd64 architectures (`arith_generic.go`)
- Godoc example functions for `Calculator`, `DefaultFactory`, and `CalculateWithObservers`
- `doc.go` for every internal package; enriched `internal/bigfft`, `internal/calibration`, `internal/app`, `internal/parallel`, `internal/testutil` with role/invariants/example comments (audit P2-13, P2-18)
- Cross-compilation targets for Linux and Windows **arm64** in the Makefile
- `.env.example` coverage for `FIBCALC_TUI_THEME`, `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_MEMORY_LIMIT` (audit P1-13, P1-14)
- `docs/architecture/patterns/design-patterns.md` inventory of concrete design patterns in use (audit P2-21)
- **Arena pooling**: `AcquireStateForN(n)` / `ReleaseStateWithResult(s, src)` API in `internal/fibonacci/` — `CalculationState` now owns its `CalculationArena`, the two pools share a single lifecycle, and all big.Int slot aliases are detached before `sync.Pool.Put` to keep the arena race-free for reuse (audit P1-04)
- New regression tests: `TestArenaPoolingNoAliasing`, `TestArenaStateConcurrent` (16 goroutines × 8 iters × 3 sizes), `TestStateReuseAcrossSizes` in `internal/fibonacci/state_pool_arena_test.go`

### Changed

- **Go toolchain**: bumped `go.mod` to Go 1.25 (toolchain go1.26.2) — audit P0-02
- **Dependencies**: minor/patch upgrades for `golang.org/x/sync`, `x/sys`, `x/term`, `x/text`, `github.com/rs/zerolog`, and `gopsutil` (audit P1-24)
- **Dependencies (major)**: bumped `github.com/charmbracelet/bubbles` from `v0.21.1` to `v1.0.0`. The bubbles v1.0 release preserved the `key` and `viewport` sub-package surfaces actually used by the TUI; zero source changes were required (audit P0-03)
- **Package restructuring**: Extracted `internal/progress/` package from `internal/fibonacci/` (observer pattern, progress types); backward-compatible type aliases in `progress_aliases.go`
- **Package restructuring**: Extracted `internal/fibonacci/memory/` sub-package (arena, GC control, memory budget)
- **Package restructuring**: Extracted `internal/fibonacci/threshold/` sub-package (dynamic threshold manager)
- Extracted `internal/app/calculate.go` — calculation dispatch logic from `app.go`
- Extracted `internal/config/thresholds.go` — adaptive threshold estimation (canonical implementation)
- Added `internal/orchestration/progress.go` — `ProgressAggregator` for multi-calculator progress
- Dependency injection: `app.New()` accepts `WithFactory()` option for custom `CalculatorFactory`
- Removed `MultiplicationStrategy` deprecated type alias from `strategy.go`
- Removed server, REPL, and observability layers to simplify the codebase
- Documentation restructure: badge coverage updated to 87.5%; README condensed (~38 KB → ≤ 22 KB) with deep-links to `docs/PERFORMANCE.md`, `docs/BUILD.md`, `docs/TUI_GUIDE.md`
- Benchmark reporting unified on AMD Ryzen 9 5900X reference; Intel Core Ultra 9 numbers retained as comparison annex (audit P2-19)
- Dependency graph (`docs/architecture/dependency-graph.mermaid`) now includes `progress`, `memory`, `threshold` nodes (audit P2-22)
- Makefile hygiene: POSIX-only targets, `go mod tidy`, removed dead cross-compile targets (audit P2-23 to P2-26)
- **CI hardening (audit A-12/A-13/A-14)**: pinned `golangci-lint` to `v1.64.8` (was `latest`, non-reproducible); dropped `check-latest: true`; `-race` now runs on Windows too; `coverage.yml` runs on PRs with a minimum-coverage floor; added an informational benchmark job
- **test/e2e (A-15)**: heavy build+subprocess e2e tests are skipped under `-short`; timeout exit-code assertions are now discriminant (`ExitErrorTimeout=2`) instead of accepting any non-zero; `TestCLI_E2E` reuses the shared one-shot build
- **docs (A-16/A-18/A-19)**: clarified `MaxPooledBitLen` (bits) vs `maxArenaPoolWords` (words); documented the `threshold.ShouldAdjust` single-writer invariant; documented that `progress` production path is `Freeze` and annotated the TUI `ProgressDoneMsg` drain sink

### Fixed

- **orchestration**: propagate `errgroup` error instead of silently ignoring (audit P0-08)
- **fibonacci**: check `ctx.Err()` after semaphore acquisition (audit P1-19)
- **bigfft / io**: handle `Flush()` and `fourier()` errors explicitly (audit P2-11, P2-12)
- Documentation links: fixed 20+ broken cross-references across `README.md`, `docs/ARCH.md`, `docs/TESTING.md`, architecture/ hub (audit P0-04 through P0-07, P1-16)
- `docs/TESTING.md` Test Organization table: paths updated for extracted sub-packages (audit P1-15)
- `docs/PERFORMANCE.md`: removed phantom `FIBCALC_GC_CONTROL` reference; GC control is automatic (audit P1-12)
- `CONTRIBUTING.md` vs `docs/TESTING.md` mockgen divergence: `TESTING.md` is now the single source of truth (audit P2-17)
- Formatting: applied `gofmt -s` and `goimports` across the tree (audit P1-09)
- **Documentation realignment (audit A-04/A-21/A-23)**: `Claude.md` rewritten to reflect post-refactoring reality (CI exists; R1.1–R1.5 resolved; removed dead `ultrareview.md`/`ultrareviewplan.md` references; `parallel` is alive; pointers now to `audit.md`/`AuditPlanning.md`). `README.md`: linter count 22→24, `sysmon`→`metrics/system`, added `FIBCALC_PROFILE_MAX_AGE`. `docs/CALIBRATION.md`: added `Confidence` field, Strategy-pattern note, `MicroBenchTimeout` var correction, env var documented. `.env.example`: added `FIBCALC_PROFILE_MAX_AGE`
- **config (A-08)**: `Validate()` now rejects `LastDigits < 0` (was silently ignored, falling back to O(memory) full computation), negative `StrassenThreshold`, out-of-set `GCControl`/`Completion`
- **config (A-09)**: malformed explicitly-set env overrides (e.g. `FIBCALC_N=abc`) now return a structured `ConfigError` instead of being silently swallowed (phantom default)
- **calibration (A-11)**: `SaveProfile` is now atomic (temp file + `os.Rename`) — no truncated/corrupt profile under concurrent writers or crash
- **progress (A-10)**: `CalcTotalWork` no longer overflows to `+Inf` (`math.Pow(4,numBits)` past ~512 bits) which froze the progress bar on exactly the large-N calculations; reworked in log-space / closed form
- **build (A-13)**: `.gitignore` `coverage.*` glob was unanchored and also hid `.github/workflows/coverage.yml` — the coverage workflow was never tracked and never ran on GitHub; now kept under version control
- Audit reference: `audit.md` (23 findings), remediation plan `AuditPlanning.md`. Concurrency hardening of `internal/bigfft` (A-01 Critical use-after-free, A-02/A-03 data races, A-05/A-06/A-07) is implemented on the frozen branch `review/vague-A-bigfft-concurrency`, **pending human review before merge** (hot-path concurrency; `-race` validated only in CI)

### Security

- **gosec G115**: explicit whitelist with justification in `.golangci.yml` (audit P1-23, P2-09)
- **gosec G304**: documented file-path inclusion exceptions (audit P2-10)

### Performance

- **FFT pool leak**: release `PolValues` / `Poly` buffers in `internal/bigfft` (audit P0-01, P0-09)
- **Arena reuse**: `CalculationArena` is now retained across calls inside the pooled `CalculationState` (sized once, `Reset()` between uses; rebuilt only when `n` outgrows the previous tenancy). `B/op` improves ~7 % on `BenchmarkFibonacci/FastDoubling/1M`; `ns/op` is unchanged within run-to-run variance. The "steal `s.FK`" zero-copy trick was removed in `ExecuteDoublingLoop` (incompatible with arena reuse) and replaced by a single deep-copy in `ReleaseStateWithResult` (~850 KB memcpy for F(10M), <0.01 % of runtime). Audit P1-04 — previously skipped because `s.FK` aliasing pooled memory raced with the next tenant's `Reset()`; now resolved by detaching the result before pool return.

- **TUI (A-22)**: lazy log-pane content flush — `updateContent()` no longer rebuilds the whole snapshot on every push (O(N²) over a long session). `BenchmarkLogsUpdateContent` ~8× faster, −76 % allocations; observable TUI output unchanged
- Added regression guards: `TestCalculateSmall_OracleEquivalence` (A-20, cross-checks `calculateSmall` vs an independent oracle, no golden tautology) and parallel-FFT arena-aliasing race tests (A-17)

### Removed

- **Audit scaffolding documents**: deleted `audit.md`, `AuditPlanning.md`, and `audit-prompt.md` from the repository. These tracked the now-closed post-audit remediation effort and are intentionally not restored; outstanding follow-ups are carried as normal issues, not audit findings.
- Phantom `FIBCALC_GC_CONTROL` environment variable reference from `docs/PERFORMANCE.md` (never implemented)
- Tracked development log artifacts and stale files (audit P1-20)

---

## [1.0.0] - 2025-12-22

### Added

#### Core Features

- **Fast Doubling Algorithm**: O(log n) Fibonacci calculation with parallel multiplication
- **Matrix Exponentiation**: O(log n) with Strassen's algorithm for large matrices
- **FFT-Based Calculator**: Optimized for extremely large numbers using FFT multiplication
- **GMP Support**: Optional GNU Multiple Precision library integration via build tag

#### Performance Optimizations

- Zero-allocation strategy using `sync.Pool` for 95%+ reduction in GC pressure
- Adaptive parallelism based on input size and hardware capabilities
- Smart multiplication switching (Karatsuba vs FFT) based on operand size
- Symmetric matrix squaring optimization (50% reduction in multiplications)
- Auto-calibration system for hardware-specific threshold optimization

#### User Interface

- Modern CLI with progress spinners, ETA calculation, and colour themes
- Shell autocompletion generation (bash, zsh, fish, PowerShell)
- JSON output format support
- Hexadecimal result display option

#### Documentation

- Comprehensive README with production deployment guide
- Architecture documentation with ADRs
- Performance tuning guide
- Security policy with vulnerability disclosure process
- Algorithm-specific documentation (Fast Doubling, Matrix, FFT, GMP)

#### Development

- Comprehensive test suite with 80%+ coverage
- Benchmark suite for performance validation
- Mock generation with mockgen
- golangci-lint configuration

### Security

- Input validation for all parameters
- Maximum N value limit (1 billion) to prevent resource exhaustion
- Configurable request timeouts
- Rate limiting protection against DoS

---

## [0.1.0] - 2025-11-01

### Added

- Initial project structure
- Basic Fast Doubling implementation
- Command-line interface
- Unit tests for core algorithms

---

[Unreleased]: https://github.com/agbru/fibcalc/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/agbru/fibcalc/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/agbru/fibcalc/releases/tag/v0.1.0
