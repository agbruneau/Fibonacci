# Contributing to Fibonacci Calculator

Thank you for your interest in contributing to the Fibonacci Calculator project! This document provides guidelines and information for contributors.

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 971 nodes / 8 layers / 13-step tour). Useful when scoping a PR — verify which packages your change touches and what depends on them.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Reporting Issues](#reporting-issues)

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
   git clone https://github.com/YOUR-USERNAME/fibcalc.git
   cd fibcalc
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/agbru/fibcalc.git
   ```

## Development Setup

### Prerequisites

- Go 1.25 or later
- Make (optional but recommended)
### Setup

```bash
# Download dependencies
make deps
# or
go mod download

# Verify the setup
make test
# or
go test ./...

# Build the project
make build
# or
go build -o build/fibcalc ./cmd/fibcalc
```

### Useful Commands

| Command           | Description              |
| ----------------- | ------------------------ |
| `make build`      | Build the binary         |
| `make test`       | Run all tests            |
| `make test-short` | Run quick tests          |
| `make coverage`   | Generate coverage report |
| `make benchmark`  | Run benchmarks           |
| `make bench-versioned` | Fixed-flag benchmark snapshot + Git/Go metadata (`build/bench/`, see [docs/PERFORMANCE.md](docs/PERFORMANCE.md)) |
| `make lint`       | Run linter               |
| `make format`     | Format code              |
| `make check`      | Run all checks           |

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

4. **Run checks locally**:

   ```bash
   make check
   ```

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
2. Register your algorithm in the global factory inside an `init` function (if internal) or explicitly in your main application:
   ```go
   fibonacci.RegisterCalculator("myalgo", func() fibonacci.CoreCalculator { return &MyAlgorithm{} })
   ```

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
   go test -fuzz=FuzzFastDoubling ./internal/fibonacci/
   ```

### Testing `CoreCalculator` in isolation

For tests that need a tiny algorithm implementation without registering a global factory type, use [`internal/fibonacci/fibonaccitest`](internal/fibonacci/fibonaccitest) (`CoreStub`). Wrap with [`fibonacci.NewCalculator`](internal/fibonacci/calculator.go) to obtain a [`fibonacci.Calculator`](internal/fibonacci/calculator.go) for orchestration or integration tests.

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

Aim for at least 75% code coverage:

```bash
make coverage
# Open coverage.html in your browser
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

Documentation files:

| File                       | Purpose                         |
| -------------------------- | ------------------------------- |
| `README.md`                | Main project documentation      |
| `docs/architecture/README.md` | Architecture hub (source of truth) |
| `docs/PERFORMANCE.md`        | Performance tuning                 |
| `docs/BUILD.md`              | Build config + dashboard republish steps |

### Generated artifacts — do not edit by hand

The following directories are produced by tooling; manual edits will be overwritten on the next regeneration.

| Path                          | Regenerated by                                           |
| ----------------------------- | -------------------------------------------------------- |
| `.understand-anything/*.json` | `/understand` (Anthropic `understand-anything` plugin)   |
| `docs/dashboard/`             | Vite build of the dashboard — see [`docs/BUILD.md#dashboard-statique-github-pages`](docs/BUILD.md#dashboard-statique-github-pages) |

If your PR changes code under `cmd/`, `internal/`, or `docs/architecture/`, consider rerunning `/understand` and rebuilding the dashboard so the published knowledge graph stays in sync.

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
