# Contributing to Fibonacci Calculator

Thank you for your interest in contributing to the Fibonacci Calculator project! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Mock Generation](#mock-generation)
- [Documentation](#documentation)
- [Reporting Issues](#reporting-issues)
- [Questions?](#questions)

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to:

- Be respectful and inclusive
- Accept constructive criticism gracefully
- Focus on what is best for the community
- Show empathy towards other community members

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/Fibonacci.git
   cd Fibonacci
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/agbruneau/Fibonacci.git
   ```

   The repository is `agbruneau/Fibonacci`; the Go module path is still
   `github.com/agbruneau/FibGo`, the repository's original name. That is
   deliberate — changing a module path breaks every existing import — and
   GitHub's redirect keeps both working, for `git clone` and for the module
   proxy alike.

## Development Setup

### Prerequisites

- Go 1.26.1 or later (`go.mod` declares `go 1.26.1`, no `toolchain` directive)
- Make (optional but recommended) — POSIX/WSL only, see the note under Useful Commands
- A C toolchain, if you want `-race` locally (both gates probe for it and skip the flag without it)

**No tool installation step.** Since 2026-09-07 (audit PRO-02) `golangci-lint`,
`govulncheck`, `gosec` and `benchstat` are pinned in [`scripts/tools.env`](scripts/tools.env)
and invoked as `go run <pkg>@<version>`, which rebuilds them with your current Go
toolchain. Nothing has to be on your `PATH`, and there is no `make install-tools`
target any more.

That is not a convenience choice. An installed tool binary is compiled once and
silently stops working when the Go toolchain moves: `golangci-lint` failed that
way in 2026-09 (GATE-01 — the gate printed `Overall: PASS` while running no
static analysis at all), and on 2026-09-07 `govulncheck`, `gosec` and
`staticcheck` were all found dead on the maintainer's host for the same reason.
`go run pkg@version` cannot fail that way.

### Setup

```bash
# Download dependencies
make deps
# or
go mod download

# Verify the setup
make test            # runs: go test -v -race -cover ./...
# or, without make (omits -race):
go test -v -cover ./...

# Build the project
make build
# or (the make target additionally injects version/commit/date via -ldflags and
#     builds with cmd/fibcalc/default.pgo when that profile is present)
go build -o build/fibcalc ./cmd/fibcalc
```

> Note: `-race` requires CGO and a C compiler (gcc/clang). On Windows
> without gcc, use `make test-win` (same tests without `-race`) or run the
> race-enabled suite under WSL.

### Useful Commands

| Command           | Description              |
| ----------------- | ------------------------ |
| `make build`      | Build the binary         |
| `make test`       | Run all tests            |
| `make test-short` | Run quick tests          |
| `make coverage`   | Generate the HTML coverage report (asserts nothing) |
| `make coverage-check` | Enforce the 90% floor — delegates to `bash scripts/check.sh --coverage-only` |
| `make benchmark`  | Run benchmarks           |
| `make bench-versioned` | Fixed-flag benchmark snapshot + Git/Go metadata (`build/bench/`, see [docs/PERFORMANCE.md](docs/PERFORMANCE.md)) |
| `make lint`       | Run linter               |
| `make format`     | Format code              |
| `make check`      | Pre-commit gate — delegates to `bash scripts/check.sh` |

> **Every `make` target here is POSIX/WSL-only.** The Makefile says so in its own
> header: "every recipe uses a POSIX shell ([ -f ], mkdir -p, ...). On Windows, run
> via WSL (`wsl make ...`); the native gate is `scripts/check.ps1`."

## Making Changes

### Branch Naming

Use descriptive branch names:

- `feature/add-new-algorithm` - New features
- `fix/memory-leak-in-fft` - Bug fixes
- `docs/update-readme` - Documentation updates
- `refactor/simplify-matrix-ops` - Code refactoring
- `perf/optimize-parallel-mult` - Performance improvements

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Types:**

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Code style (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(fibonacci): add Schönhage-Strassen multiplication

docs(readme): update installation instructions

perf(bigfft): optimize FFT butterfly operations
```

## Pull Request Process

1. **Update your fork** with the latest upstream changes:

   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Create a feature branch**:

   ```bash
   git checkout -b feature/your-feature
   ```

3. **Make your changes** and commit them

4. **Run checks locally.** GitHub Actions runs the same sequence on every push
   and pull request (`.github/workflows/ci.yml`), but the local gate is faster
   and should still be green before you push.

   ```bash
   # Linux / macOS / WSL (make check is just a wrapper around this)
   bash scripts/check.sh

   # Windows without a POSIX shell — the Makefile does not run there
   pwsh ./scripts/check.ps1
   ```

   The two scripts are not equivalent: `check.sh` adds a step 3b that builds/vets/tests
   under `-tags gmp` when the libgmp headers are present, which `check.ps1` has no
   counterpart for. The race detector is no longer a difference — since 2026-09-03
   `check.ps1` probes for CGO and a C compiler and runs `-race` when both are there.
   Prefer `check.sh` (via WSL on Windows) before anything touching
   `internal/fibonacci` or `internal/bigfft`. In both scripts lint is a **hard** step:
   a `golangci-lint` that is absent or failing fails the gate.

5. **Push to your fork**:

   ```bash
   git push origin feature/your-feature
   ```

6. **Create a Pull Request** on GitHub

### PR Requirements

- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make format`)
- [ ] Linter passes (`make lint`)
- [ ] New code has tests (total coverage must stay >= 90%)
- [ ] Documentation is updated if needed
- [ ] Commit messages follow conventions

### Adding New Algorithms

The project uses the Decorator pattern. To add a new algorithm, you only need to implement the core logic; cross-cutting concerns (GC, caching, thresholds) are handled for you.

1. Create a type that implements the `fibonacci.CoreCalculator` interface:
   ```go
   type MyAlgorithm struct{}
   
   func (a *MyAlgorithm) CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts fibonacci.Options) (*big.Int, error) {
       // Your core algorithm logic here...
       // Report progress via reporter(float64) between 0.0 and 1.0
       return result, nil
   }
   
   func (a *MyAlgorithm) Name() string {
       return "My Algorithm Name"
   }
   ```
2. Register your algorithm on the factory your application builds (there is no global registry; `app.New` creates one via `fibonacci.NewDefaultFactory()`):
   ```go
   factory := fibonacci.NewDefaultFactory()
   if err := factory.Register("myalgo", func() fibonacci.CoreCalculator { return &MyAlgorithm{} }); err != nil {
       return fmt.Errorf("register myalgo: %w", err)
   }
   ```

   `Register` returns an `error` (`internal/fibonacci/registry.go:DefaultFactory.Register`). Do not drop
   it: `errcheck` is enabled and unexcluded (`.golangci.yml`, `linters.enable`), so `make lint`
   rejects the bare call.

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Project-Specific Guidelines

1. **Package Comments**: Every package should have a doc comment

2. **Function Documentation**: Public functions must have doc comments:

   ```go
   // Calculate computes the nth Fibonacci number using the configured algorithm.
   // It returns an error if the context is canceled or times out.
   //
   // Parameters:
   //   - ctx: Context for cancellation
   //   - n: Index of the Fibonacci number
   //
   // Returns:
   //   - *big.Int: The calculated Fibonacci number
   //   - error: Any error that occurred
   func (c *Calculator) Calculate(ctx context.Context, n uint64) (*big.Int, error) {
       // ...
   }
   ```

3. **Comments carry the reason, not a reference.** A comment explains why the
   code is the way it is, in words. An audit identifier (`FIB-02`, `OVR-10`,
   `M-01`, `A2-04`) is a pointer to where the decision was recorded — never the
   explanation itself. Erase the identifier and the comment must still make
   sense.

   ```go
   // Bad — the reader has to run `git log -S` to learn anything:
   //   // FIB-02
   //   thresholds := []int{config.ThresholdDisabled}

   // Good — the reason is here, the identifier only says where to read more:
   //   // ThresholdDisabled (-1, not 0) is the genuine sequential baseline:
   //   // normalizeOptions replaces ==0 with the package default, so a 0
   //   // candidate silently re-measured the default (FIB-02).
   //   thresholds := []int{config.ThresholdDisabled}
   ```

   Only prefixes listed in [`docs/audits/INDEX.md`](docs/audits/INDEX.md) may be
   cited, and an audit report is archived under `docs/audits/` when its work is
   done rather than deleted. Roughly 350 identifiers in this code base pointed
   at reports that no longer existed anywhere in the tree; the index and that
   rule are what make them resolvable (audit DOC-01).

4. **Language.** The language follows the genre of the document:
   - **French** for narrative: `README.md`, `CHANGELOG.md`, the ADRs
     (`docs/adr/`) and the audit and evaluation reports (`docs/audits/`);
   - **English** for technical reference: `docs/*.md`, `docs/algorithms/`,
     `docs/architecture/`, and for code, comments, identifiers, error
     messages, linter configuration and commit subjects.

   A document stays in one language; it does not alternate inside itself.
   Amended on 2026-09-23 (ADR-0013 D2, EVAL-19): the 2026-09-07 rule put all of
   `docs/*.md` in French (ADR-0012 D4), and thirteen reference documents never
   followed it. The amended rule describes the corpus instead of being ignored
   by it.

5. **Error Handling**: Use the `internal/apperrors` package for custom errors

6. **Configuration**: Use functional options pattern for configurable components

7. **Concurrency**: Use `sync.Pool` for frequently allocated objects

### File Organization

```
internal/
├── fibonacci/          # Core algorithms
│   ├── calculator.go   # Public interface
│   ├── strategy.go     # Strategy pattern
│   └── *_test.go       # Tests alongside code
├── cli/                # Command-line interface
└── config/             # Configuration
```

## Testing Guidelines

### Test Types

1. **Unit Tests**: Test individual functions

   ```bash
   go test -v ./internal/fibonacci/
   ```

2. **Integration Tests**: Test component interaction

   ```bash
   go test -v ./cmd/fibcalc/
   ```

3. **Benchmarks**: Measure performance

   ```bash
   go test -bench=. -benchmem ./internal/fibonacci/
   ```

4. **Fuzzing**: Find edge cases
   ```bash
   go test -fuzz=FuzzFastDoublingConsistency ./internal/fibonacci/
   ```

### Testing `CoreCalculator` in isolation

For tests that need a tiny algorithm implementation, implement [`fibonacci.CoreCalculator`](internal/fibonacci/calculator.go) directly (a configurable stub is ~30 lines — see `coreStub` in `internal/orchestration/contract_test.go`). Wrap with [`fibonacci.NewCalculator`](internal/fibonacci/calculator.go) to obtain a [`fibonacci.Calculator`](internal/fibonacci/calculator.go) for orchestration or integration tests.

### Writing Tests

- Use table-driven tests when possible
- Include edge cases (n=0, n=1, very large n)
- Test error conditions
- Use subtests for better organization:
  ```go
  func TestCalculator(t *testing.T) {
      t.Run("small values", func(t *testing.T) {
          // ...
      })
      t.Run("large values", func(t *testing.T) {
          // ...
      })
  }
  ```

### Test Coverage

Keep total coverage at or above 90% — this is the floor enforced by the
`make coverage-check` gate (it fails if total coverage drops below 90%):

```bash
make coverage        # generate the HTML report (open coverage.html in your browser)
make coverage-check  # verify total coverage is >= 90%
```

## Mock Generation

The test suite currently uses hand-written mocks; `mockgen` is not wired in
(no `//go:generate` directives, no `mocks/` directories, and no `mockgen`
Makefile targets). A future migration to generated mocks is documented but
not yet implemented. See [docs/TESTING.md — Mock Generation](docs/TESTING.md#mock-generation)
for the authoritative reference and migration plan.

## Documentation

### Code Documentation

- All exported types, functions, and methods must have doc comments
- Use examples where helpful (see `ExampleCalculator_Calculate`)

### Project Documentation

Update documentation when:

- Adding new features
- Changing public APIs
- Modifying configuration options
- Updating deployment procedures

Where each kind of change lands (not an exhaustive list of `docs/` — see the README's own links):

| File | What belongs there |
| ---- | ------------------ |
| `README.md` | Entry point: quick start, flag table, the end-to-end walkthrough of one `fibcalc` run, audit history, headline numbers |
| `CHANGELOG.md` | Every observable change, Keep-a-Changelog format |
| `docs/ARCH.md` | Architecture *prose*: the why, the constants, the defaults, what is not guaranteed. It cites the figures below instead of redrawing them |
| `docs/architecture/` | Architecture *figures* (C4, dependency graph, flows, patterns) — the authoritative view of the system's shape. `README.md` there is the hub; `patterns/design-patterns.md` is the single pattern inventory. Fix the figure first, then its legend in `ARCH.md` |
| `docs/adr/NNNN-*.md` | A decision *and* the candidates you rejected, with the measurement or prior ADR that rejects them |
| `docs/TESTING.md` | Test strategy, golden files, mock policy |
| `docs/PERFORMANCE.md` | Tuning method and the non-regression protocol |
| `docs/BUILD.md` | Build config, PGO, cross-compilation, Docker |
| `docs/audits/*.txt` | The raw output behind any number you publish |

Numbers stated in prose must be traceable to a command, a source symbol, or a file under
`docs/audits/`. A figure that has not been re-run is marked as such rather than restated.

### Generated artifacts — do not edit by hand

The following directories are produced by tooling; manual edits will be overwritten on the next regeneration.

| Path                          | Regenerated by                                           |
| ----------------------------- | -------------------------------------------------------- |
| `.understand-anything/*.json` | `/understand` (Anthropic `understand-anything` plugin)   |

`.understand-anything/` is not tracked in git.

## Reporting Issues

### Bug Reports

Include:

1. **Go version**: `go version`
2. **Operating system**
3. **Steps to reproduce**
4. **Expected behaviour**
5. **Actual behaviour**
6. **Relevant logs or output**

### Feature Requests

Describe:

1. **Use case**: What problem does this solve?
2. **Proposed solution**: How should it work?
3. **Alternatives considered**: Other approaches you've thought of

### Security Issues

For security vulnerabilities, please open a private issue or contact the maintainers directly.

---

## Questions?

Feel free to open an issue for any questions about contributing. We're happy to help!

Thank you for contributing to Fibonacci Calculator!
