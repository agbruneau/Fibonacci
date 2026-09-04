# Container Diagram — C4 niveau 2

Découpage de l'application en conteneurs logiques. Chaque `Rel` entre deux `Container` est un import Go direct ; `Rel(user, entry, …)` est la seule exception (relation `Person` → `Container`).

```mermaid
C4Container
    title Container Diagram — FibCalc (C4 Level 2)

    Person(user, "User")

    Container_Boundary(fibcalc, "FibCalc Application") {
        Container(entry, "Entry Point", "cmd/fibcalc", "main.go — delegates to app.New() and app.Run()")
        Container(app, "Application Core", "internal/app", "Lifecycle management, mode dispatching")
        Container(config, "Configuration", "internal/config", "Flag parsing, env var overrides, validation")
        Container(orch, "Orchestration", "internal/orchestration", "Parallel execution via errgroup, result analysis")
        Container(fib, "Fibonacci Algorithms", "internal/fibonacci", "Fast Doubling, Matrix, FFT-based, GMP calculators")
        Container(bigfft, "FFT Multiplication", "internal/bigfft", "Schonhage-Strassen FFT, Fermat arithmetic, caching")
        Container(cli, "CLI Presentation", "internal/cli", "Spinner, progress bar, ETA, result formatting")
        Container(tui, "TUI Presentation", "internal/tui", "Bubble Tea Elm architecture, dashboard panels")
        Container(calib, "Calibration", "internal/calibration", "Benchmarking, threshold estimation, profile persistence")
        Container(support, "Leaf Packages", "internal/*", "errors, format, metrics, progress, ui, testutil, fibonacci/memory, fibonacci/threshold, cli/completion — every one has zero internal imports")
    }

    Rel(user, entry, "Invokes")
    Rel(entry, app, "Delegates to")
    Rel(entry, support, "errors — Exit* codes for os.Exit")
    Rel(app, config, "Parses configuration")
    Rel(app, calib, "Loads/runs calibration")
    Rel(app, orch, "Dispatches calculation")
    Rel(app, fib, "Creates calculators via factory")
    Rel(app, cli, "CLI mode presentation")
    Rel(app, tui, "TUI mode presentation")
    Rel(orch, fib, "Runs calculators")
    Rel(fib, bigfft, "FFT multiplication")
    Rel(cli, orch, "Implements ProgressReporter/ResultPresenter")
    Rel(cli, config, "Reads AppConfig")
    Rel(tui, orch, "Implements ProgressReporter/ResultPresenter")
    Rel(tui, config, "Reads AppConfig")
    Rel(calib, config, "Reads AppConfig, adaptive estimates")
    Rel(calib, bigfft, "Benchmarks bigfft.Mul")
    Rel(calib, fib, "Benchmarks algorithms")
    Rel(app, support, "errors, ui, fibonacci/memory, fibonacci/threshold, cli/completion")
    Rel(config, support, "errors, ui, fibonacci/memory")
    Rel(orch, support, "errors, progress, fibonacci/memory")
    Rel(fib, support, "errors, progress, fibonacci/memory, fibonacci/threshold")
    Rel(cli, support, "errors, format, metrics, progress, ui")
    Rel(tui, support, "errors, format, metrics, progress, ui")
    Rel(calib, support, "errors, format, progress, ui")
```

---
[← Retour au hub architecture](./README.md)
Légende narrative de cette figure : [§2 High-Level Architecture](../ARCH.md#2-high-level-architecture-clean-architecture).
