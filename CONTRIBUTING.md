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
   git clone https://github.com/YOUR-USERNAME/FibGo.git
   cd FibGo
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/agbruneau/FibGo.git
   ```

## Development Setup

### Prerequisites

- Go 1.26.0 or later (`go.mod` declares `go 1.26.0`, no `toolchain` directive)
- Make (optional but recommended) — POSIX/WSL only, see the note under Useful Commands
- `golangci-lint` **v2**, required by the pre-commit gate:

  ```bash
  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
  # or, with gosec as well:
  make install-tools
  ```

  A v1 binary will not do. `.golangci.yml` uses the v2 schema, and a v1 binary cannot analyze this
  module under a go1.27 toolchain at all — every package fails with `export data version 4 is greater
  than maximum supported version 2`. Both gate scripts treat that (and a missing binary) as a hard
  failure since 2026-09-03; the version pin was dropped for the same reason.

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
| `make coverage-check` | Enforce the 80% floor — delegates to `bash scripts/check.sh --coverage-only` |
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

4. **Run checks locally.** There is no remote CI, so this is the only gate.

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
- [ ] New code has tests (aim for >80% coverage)
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

3. **Error Handling**: Use the `internal/errors` package for custom errors

4. **Configuration**: Use functional options pattern for configurable components

5. **Concurrency**: Use `sync.Pool` for frequently allocated objects

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

Aim for at least 80% code coverage — this is the floor enforced by the
`make coverage-check` gate (it fails if total coverage drops below 80%):

```bash
make coverage        # generate the HTML report (open coverage.html in your browser)
make coverage-check  # verify total coverage is >= 80%
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
| `README.md` | Entry point: quick start, flag table, audit history, headline numbers |
| `CHANGELOG.md` | Every observable change, Keep-a-Changelog format |
| `docs/architecture/README.md` | Architecture hub and C4 diagrams (source of truth) |
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
