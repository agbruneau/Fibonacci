# FibGo Architecture — Detailed Reference

This directory holds the detailed architecture documentation of the FibCalc project: technical diagrams, data flows, and the invariant validation record (§5). The ADRs (Architectural Decision Records) do not live here but in [`docs/adr/`](../adr/); §4 below gives only their index.

> **Division of labor with [`docs/ARCH.md`](../ARCH.md).** This directory **draws**;
> `ARCH.md` **narrates** — with one exception: the CLI flow figure is drawn in
> [`ARCH.md` §6](../ARCH.md#6-data-flow-cli-input-to-final-result), at the head of the section
> that explains it, because its legend is that section and nothing else.
> The ten figures below are the authoritative view of the system's *shape*
> — import edges, subgraphs, branch order — and the sections
> of `ARCH.md` are their legend: the why, the constants, the defaults, what is
> not guaranteed. There are **not** two competing views of the architecture: `ARCH.md`
> cites these figures instead of redrawing second ones, and each figure names in its
> footer the section that explains it. The full mapping is the
> [figure map](../ARCH.md#0-figure-map). A change of shape is corrected
> **in the figure first**, then in the legend.

> **Diagram format.** The ten diagrams are `.md` files whose body is a
> fenced `mermaid` block. It is the only format GitHub renders graphically: a standalone
> `.mermaid` or `.mmd` file displays as plain text. They carried the `.mermaid` extension
> until 2026-09-04 and were converted for that reason; the repo already used this
> convention elsewhere (`docs/algorithms/*.md`, `docs/TUI_GUIDE.md`). The corpus's eleven blocks
> — the ten here plus the one in [`ARCH.md` §6](../ARCH.md#6-data-flow-cli-input-to-final-result) —
> were run through the `mermaid` v11.17.2 parser on 2026-09-04: zero syntax errors.

## 1) Architecture Diagrams (C4 Model)

We use the C4 model to document the architecture at different levels of abstraction:

- **[System Context](system-context.md):** High-level view of FibCalc and its interactions with the user and the operating system. — *legend: [`ARCH.md` §1](../ARCH.md#1-project-overview)*
- **[Container Diagram](container-diagram.md):** Breakdown of the application into logical containers (CLI, TUI, Core Engine). Every `Rel` between two `Container`s is a real Go import — the §5 record counts them one by one. — *legend: [`ARCH.md` §2](../ARCH.md#2-high-level-architecture-clean-architecture)*
- **[Component Diagram](component-diagram.md):** Detail of the internal components of the compute engine and of orchestration. It is a `classDiagram`: its arrows are class relations, **not** package imports. — *legend: [`ARCH.md` §4](../ARCH.md#4-core-packages-responsibilities-key-types-interfaces)*

## 2) Dependency Graph

The project strictly follows the principles of **Clean Architecture**. The following graph shows the relations between packages:

- **[Dependency Graph](dependency-graph.md)** — the module's 45 direct internal imports, one per edge. Reproducible with the `go list` command given in the [validation record](./validation/validation-report.md#layer-tightness--dependency-direction). — *legend: [`ARCH.md` §2](../ARCH.md#2-high-level-architecture-clean-architecture) (the layering rule) and [§3](../ARCH.md#3-directory-structure) (the directories behind the nodes)*

## 3) Data Flows and Execution Paths

Six `flowchart`s trace the critical execution paths, from entry point to result:

- **CLI execution** — the figure and its legend sit together in
  [`ARCH.md` §6](../ARCH.md#6-data-flow-cli-input-to-final-result): the `flowchart` at the head
  of the section, then ten numbered steps that each comment one subgraph.
- **[Flows/](./flows/):**
  - [TUI](./flows/tui-flow.md) execution — *legend: [`ARCH.md` §6, TUI mode](../ARCH.md#tui-mode-figure)*.
  - [Configuration resolution](./flows/config-flow.md) — *legend: [`ARCH.md` §8](../ARCH.md#configuration-cascade) and [§9](../ARCH.md#9-configuration-and-environment)*.
  - Algorithm pipelines: [Fast Doubling](./flows/fastdoubling.md) (*[§7A](../ARCH.md#a-fast-doubling-fastdoublingcalculator)*), [FFT](./flows/fft-pipeline.md) (*[§7C](../ARCH.md#c-fft-based-doubling-fftbasedcalculator)*), [Matrix](./flows/matrix.md) (*[§7B](../ARCH.md#b-matrix-exponentiation-matrixexponentiationcalculator)*).

## 4) Design Patterns and ADRs

The architecture rests on the design patterns documented here:

- **[Patterns/](./patterns/):**
  - **[Design Patterns inventory](./patterns/design-patterns.md)** — the **authoritative inventory**: 16 patterns (Decorator, Strategy, ISP, Factory/Registry, Observer, Template Method, Facade, Adapter, Object Pool, Arena Allocator, Bump Allocator, LRU Cache, Circuit Breaker, Zero-Copy Result Return, Generics with Pointer Constraints, GC Controller) and 5 engineering mechanisms, with the rationale and implementation site of each. Merged (2026-09-04) from this table and the one `ARCH.md` §5 kept in parallel; there is now only one list, and [`ARCH.md` §5](../ARCH.md#5-design-patterns) points to it.
  - **[Interface hierarchy](./patterns/interface-hierarchy.md)** — the key interfaces and their implementations. Explained by [`ARCH.md` §5](../ARCH.md#5-design-patterns) and [§8](../ARCH.md#presentation-layer-integration).

### ADRs — Current architectural decisions

The Architectural Decision Records live in [`docs/adr/`](../adr/):

| ADR | Title | Status |
|---|---|---|
| [0000](../adr/0000-template.md) | Template | — |
| [0001](../adr/0001-dtm-decision.md) | Fate of `DynamicThresholdManager` vs `internal/calibration/` | Superseded by 0013 |
| [0002](../adr/0002-recover-strategy.md) | `recover()` strategy in `bigfft` (post-condition sentinel) | Accepted |
| [0003](../adr/0003-globals-vs-context.md) | Mutable `bigfft` globals → `atomic.Int64` | Accepted |
| [0004](../adr/0004-backlog-decisions.md) | Formal post-hardening backlog decisions | Accepted |
| [0005](../adr/0005-gc-control-concurrent.md) | Concurrency-safe GC control (package-level refcount) | Accepted |
| [0006](../adr/0006-fft-recursion-cancellation.md) | FFT recursion cancellation — deferred to the per-call token (FFTContext) | Accepted ⚠ *subject removed from the code* |
| [0007](../adr/0007-pool-pointer-vs-value.md) | SA6002 (`sync.Pool.Put` of a slice) — measured decision | Accepted |
| [0008](../adr/0008-audit-2026-06-rejected-candidates.md) | 2026-06 refactoring audit — candidates rejected after verification | Accepted |
| [0009](../adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) | 2026-07 audit — bigfft purge, oracle retention, rejection then adoption of ×10 (addendum R4) | Accepted |
| [0010](../adr/0010-audit-2026-09-decisions.md) | 2026-09 audit — precedence of explicit thresholds, opt-in DTM, blocking lint, candidates rejected on measurement | Accepted |
| [0011](../adr/0011-audit-2026-09-ponytail.md) | 2026-09-03 over-engineering audit — removals and fallbacks, candidates set aside with their reason | Accepted |
| [0012](../adr/0012-audit-2026-09-livre-decisions.md) | 2026-09-07 audit against *Building Enterprise Projects with Go* — CI, pinned tools, language rule (amended by 0013) | Accepted |
| [0013](../adr/0013-evaluation-2026-09-decisions.md) | 2026-09-15 academic evaluation — DTM removed, language rule amended, coverage floor, GMP reachable | Accepted |

⚠ **ADR-0006 carries "Accepted" and its subject is no longer in the tree.** The opt-in `FFTContext` API
(`NewFFTContext`, `*WithContext`, `fourierRecursiveCtx`) was **removed** — zero occurrences in
`internal/bigfft/` as of the 2026-08-08 check —, the migration it prepared having been classified
WONT-FIX by [ADR-0004 §B1](../adr/0004-backlog-decisions.md); the removal is recorded in
[`CHANGELOG.md`](../../CHANGELOG.md) and the code can be reread in the git history. ⚠ *An ADR describes a
dated decision, not the state of the code: this one stays accurate as a decision and stops being verifiable
against the source.* **Changing its status is a maintainer decision, not a documentation
resync — it is not taken here.**

The granular history of inherited decisions (CPU heuristic, search
backends) remains summarized in **[docs/ARCH.md](../ARCH.md#14-architectural-decision-records-adr)**.

**ADR-0001 is superseded by ADR-0013.** The `DynamicThresholdManager` had
been kept (KEEP) on the strength of a 5-6% gain at F(10M); the 2026-09 audit
(M-04) wired it behind `--dynamic-thresholds`, and the measurement taken through that
flag (`-count=8`) did not reproduce the gain. The 2026-09-15 evaluation drew
the consequence: the package, the flag and the variable have been removed since
2026-09-23 ([ADR-0013](../adr/0013-evaluation-2026-09-decisions.md) D1).

### Architecture gate

`internal/arch_test.go` enforces six Clean Architecture rules, eight forbidden arrows:
`errors → format`, `tui → fibonacci`, `orchestration → format` (APP-10),
`cli → fibonacci` (STR-04), `calibration → ui`/`calibration → format` (ARC-01) and
`config → fibonacci`/`config → bigfft` (ARCH-02). Any PR reintroducing
one of these upward imports fails `make test` (or
`go test ./internal/`). Detail: [`docs/TESTING.md` §Architecture-Layering Gate](../TESTING.md#architecture-layering-gate).

## 5) Invariant Validation

- **[Validation/](./validation/):**
  - **[validation-report.md](./validation/validation-report.md)** — record of the invariants the
    documentation asserts and that have been checked against the source: layer tightness and dependency
    direction, interface and pattern claims, execution flows, maintenance note.
    Living reference, to re-verify and update in place when the structure changes.

---
[← Back to the overview (ARCH.md)](../ARCH.md)
