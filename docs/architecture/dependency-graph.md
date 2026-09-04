# Graphe des dépendances internes

Les 46 imports internes directs du module, un par arête — ni plus, ni moins. Le pipeline
`go list` qui établit cette égalité d'ensembles est dans le
[relevé de validation](./validation/validation-report.md#layer-tightness--dependency-direction)
(exécuté le 2026-09-04, `diff` vide).

```mermaid
flowchart LR
    subgraph Entry["Entry Point"]
        main["cmd/fibcalc<br/>main.go"]
    end

    subgraph Tooling["Dev Tooling"]
        gen["cmd/generate-golden<br/>golden oracle — zéro import interne"]
    end

    subgraph Core["Application Core"]
        app["internal/app<br/>Lifecycle & Dispatch"]
        config["internal/config<br/>Flag Parsing & Env"]
    end

    subgraph Orchestration["Orchestration Layer"]
        orch["internal/orchestration<br/>Parallel Execution"]
    end

    subgraph Business["Business Logic"]
        fib["internal/fibonacci<br/>Algorithms & Frameworks"]
        bigfft["internal/bigfft<br/>FFT Multiplication"]
        calib["internal/calibration<br/>Benchmarking & Tuning"]
        fibmem["internal/fibonacci/memory<br/>Arena, GC, Budget"]
        fibthr["internal/fibonacci/threshold<br/>Parallel/FFT Thresholds"]
    end

    subgraph Presentation["Presentation Layer"]
        cli["internal/cli<br/>CLI Output"]
        completion["internal/cli/completion<br/>Shell Completion"]
        tui["internal/tui<br/>TUI Dashboard"]
    end

    subgraph Support["Support Packages (Leaf Nodes)"]
        errors["internal/errors"]
        format["internal/format"]
        metrics["internal/metrics"]
        progress["internal/progress"]
        ui["internal/ui"]
        testutil["internal/testutil"]
    end

    main --> app
    main --> errors
    app --> config
    app --> orch
    app --> cli
    app --> tui
    app --> calib
    app --> fib
    app --> errors
    app --> ui

    orch --> fib
    orch --> errors
    orch --> progress
    orch --> fibmem

    config --> errors
    config --> fibmem
    config --> ui

    calib --> fib
    calib --> bigfft
    calib --> config
    calib --> errors
    calib --> format
    calib --> ui
    calib --> progress

    fib --> bigfft
    fib --> errors
    fib --> progress
    fib --> fibmem
    fib --> fibthr

    cli --> format
    cli --> errors
    cli --> metrics
    cli --> ui
    cli --> orch
    cli --> config
    cli --> progress

    tui --> format
    tui --> metrics
    tui --> ui
    tui --> errors
    tui --> config
    tui --> orch
    tui --> progress

    app --> fibmem
    app --> fibthr
    app --> completion

    style Entry fill:#e1f5fe
    style Tooling fill:#ede7f6
    style Core fill:#f3e5f5
    style Orchestration fill:#fff3e0
    style Business fill:#e8f5e9
    style Presentation fill:#fce4ec
    style Support fill:#f5f5f5
```

---
[← Retour au hub architecture](./README.md)
Légende narrative de cette figure : [§2 High-Level Architecture](../ARCH.md#2-high-level-architecture-clean-architecture) et [§3 Directory Structure](../ARCH.md#3-directory-structure).
