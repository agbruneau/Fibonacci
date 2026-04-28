# 01 — Inventaire du dépôt

Audit FibGo (`github.com/agbru/fibcalc`) — Tâche 1.1 (Cartographie).
Date : 2026-04-28. Branche : `main`. HEAD : `4c8f0c1`.

## Synthèse

- LOC production (`*.go` hors `*_test.go`) : **14 949**
- LOC test (`*_test.go`, incl. `test/e2e`) : **21 242**
- LOC test (packages `cmd/` + `internal/` uniquement) : **20 636**
- LOC test (`test/e2e`) : **606**
- Total LOC Go : **36 191**
- Packages (`cmd/` + `internal/`) : **21**
- Packages avec `doc.go` : **20 / 21** (manque : `cmd/fibcalc`)
- Module Go : `github.com/agbru/fibcalc` (Go 1.25.0+, toolchain 1.26.2)

## Packages (cmd/ + internal/)

LOC mesurés via `wc -l` (lignes physiques, blank/comments inclus).

| Chemin                              | LOC prod | LOC test | Fichiers `.go` (prod) | Fichiers `_test.go` | `doc.go` |
| ----------------------------------- | -------: | -------: | --------------------: | ------------------: | :------: |
| `cmd/fibcalc`                       |       43 |      250 |                     1 |                   1 |    no    |
| `cmd/generate-golden`               |      131 |      164 |                     2 |                   1 |   yes    |
| `internal/app`                      |      462 |    1 369 |                     4 |                   2 |   yes    |
| `internal/bigfft`                   |    3 316 |    4 295 |                    17 |                  15 |   yes    |
| `internal/calibration`              |    1 303 |    1 197 |                     7 |                   6 |   yes    |
| `internal/cli`                      |    1 160 |      974 |                     8 |                   9 |   yes    |
| `internal/config`                   |      642 |    1 480 |                     6 |                   6 |   yes    |
| `internal/errors`                   |      327 |      559 |                     3 |                   2 |   yes    |
| `internal/fibonacci`                |    3 179 |    4 017 |                    19 |                  25 |   yes    |
| `internal/fibonacci/fibonaccitest`  |       36 |       53 |                     2 |                   1 |   yes    |
| `internal/fibonacci/memory`         |      325 |      280 |                     4 |                   3 |   yes    |
| `internal/fibonacci/threshold`      |      476 |      579 |                     3 |                   1 |   yes    |
| `internal/format`                   |      358 |      338 |                     4 |                   1 |   yes    |
| `internal/metrics`                  |      221 |      272 |                     3 |                   2 |   yes    |
| `internal/orchestration`            |      388 |      642 |                     5 |                   5 |   yes    |
| `internal/parallel`                 |       96 |       84 |                     2 |                   1 |   yes    |
| `internal/progress`                 |      443 |      697 |                     4 |                   2 |   yes    |
| `internal/sysmon`                   |       49 |       20 |                     2 |                   1 |   yes    |
| `internal/testutil`                 |       46 |       43 |                     2 |                   1 |   yes    |
| `internal/tui`                      |    1 668 |    3 074 |                    12 |                  10 |   yes    |
| `internal/ui`                       |      280 |      249 |                     3 |                   1 |   yes    |
| **Total**                           | **14 949** | **20 636** |               **117** |             **96**  |   20/21  |

Notes :
- `internal/fibonacci` contient également `testdata/fibonacci_golden.json` et `testdata/fuzz/`.
- Trois sous-packages techniques sous `internal/fibonacci/` (`fibonaccitest`, `memory`, `threshold`) sont comptés séparément.

## docs/

Arborescence et contenu :

- `docs/ARCH.md`
- `docs/BUILD.md`
- `docs/CALIBRATION.md`
- `docs/PERFORMANCE.md`
- `docs/TESTING.md`
- `docs/TUI_GUIDE.md`
- `docs/algorithms/`
  - `BIGFFT.md`, `COMPARISON.md`, `FAST_DOUBLING.md`, `FFT.md`, `GMP.md`, `MATRIX.md`, `PROGRESS_BAR_ALGORITHM.md`
- `docs/architecture/`
  - `README.md`
  - `system-context.mermaid`, `container-diagram.mermaid`, `component-diagram.mermaid`, `dependency-graph.mermaid`
  - `flows/` : `cli-flow.mermaid`, `config-flow.mermaid`, `fastdoubling.mermaid`, `fft-pipeline.mermaid`, `matrix.mermaid`, `tui-flow.mermaid`
  - `patterns/` : `design-patterns.md`, `interface-hierarchy.mermaid`
  - `validation/` : `validation-report.md`
- `docs/audits/`
  - `_bench_raw.log`
  - `2026-04/`
    - `AUDIT_REPORT.md`, `EXECUTION_PLAN.md`, `PDRTask.md`, `PRD.md`
    - `bench/exec-baseline/` : `benchmark.txt`, `coverage.out`, `coverage.txt`, `lint.txt`, `test.txt`
    - `bench/perf-results/` : `P1-04-SKIPPED.md`, `P0-01-P0-09/{after.txt,before.txt}`

## test/

- `test/e2e/`
  - `cli_e2e_test.go`
  - `extended_e2e_test.go`
  - LOC test : 606 ; pas de fichiers prod.

## Fichiers racine

| Fichier             | Taille (octets) | Présence |
| ------------------- | --------------: | :------: |
| `go.mod`            |           2 033 |    oui   |
| `go.sum`            |          68 225 |    oui   |
| `Makefile`          |           9 717 |    oui   |
| `README.md`         |          13 500 |    oui   |
| `CHANGELOG.md`      |           6 401 |    oui   |
| `CONTRIBUTING.md`   |           9 652 |    oui   |
| `CLAUDE.md`         |           4 408 |    oui   |
| `PLAN.md`           |           9 213 |    oui   |
| `LICENSE`           |          11 558 |    oui   |
| `.golangci.yml`     |           5 659 |    oui   |
| `.gitignore`        |           1 375 |    oui   |
| `.env.example`      |           5 066 |    oui   |
| `coverage.out`      |         173 470 |    oui   |

Absents (notables) :
- `.github/` (aucun workflow CI versionné dans le dépôt)
- `Dockerfile` / `docker-compose.yml`
- `goreleaser.yml`
- `CODEOWNERS`, `SECURITY.md`, `CODE_OF_CONDUCT.md`
