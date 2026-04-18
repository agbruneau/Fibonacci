# TEAM D — Documentation

Audit read-only de tout le Markdown + `doc.go` du repo FibGo. Aucune modification appliquée. Chemins absolus pour références.

## Résumé exécutif

- **Fichiers Markdown audités** : 19 (root + `docs/**`) ; **fichiers `.mermaid`** : 9.
- **Packages Go (`go list ./...`)** : 22. **`doc.go` présents** : 13. **Packages sans `doc.go`** : 6 (`internal/format`, `internal/metrics`, `internal/progress`, `internal/sysmon` partiellement, `internal/fibonacci/memory`, `internal/fibonacci/threshold`). +2 `cmd/*` ont leur `// Package main` directement dans `main.go` (acceptable).
- **Références fichiers obsolètes (P0/P1)** : 4 fichiers `.md` cités mais inexistants (`docs/INNOVATION.md`, `docs/INNOVEPLAN.md`, `docs/architecture/patterns/design-patterns.md`, et 4 sous-fichiers `flows/*.md` annoncés "créés" par le rapport de validation mais matérialisés en `.mermaid` seulement).
- **Incohérences env vars** : 3 variables documentées sans implémentation (`FIBCALC_GC_CONTROL`, `FIBCALC_LAST_DIGITS`), 1 implémentée non documentée (`FIBCALC_TUI_THEME`).
- **Stats périmées** : `Claude.md` annonce 22 linters / "103 fichiers source, 89 tests" ; mesuré aujourd'hui : 22 linters (OK), 108 sources (.go non-test), 97 tests, ~35.5k lignes.
- **CHANGELOG** : Keep-a-Changelog respecté ; section `[Unreleased]` capture la restructuration des sous-packages mais n'inclut PAS les commits perf récents (FFT cache key, NTransform, ExecutionConfig refactor, file permissions security fix, manual sem release, PowerShell completion perf).
- **CONTRIBUTING** : workflow OK ; recense `mockgen` mais TESTING.md note (correctement) que mocks ne sont PAS branchés. Incohérence à résoudre.
- **README condensation** : 585 lignes / ~38 Ko ; 4 sections (Usage Guide / Performance Benchmarks / Configuration / Development) extractibles vers `docs/`.

Compte de findings : **P0 = 4**, **P1 = 9**, **P2 = 7**.

---

## Checklist par fichier

| Fichier | Statut | Notes principales |
|---|---|---|
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\README.md` | A corriger (P0+P1+P2) | Refs `docs/INNOVATION.md` & `docs/INNOVEPLAN.md` (lignes 193, 543-544) inexistantes. Coverage badge `80%` non vérifié. Très long (585 lignes) — condenser. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\CHANGELOG.md` | A corriger (P1) | Format Keep-a-Changelog OK ; section `[Unreleased]` sous-représente les commits perf 2026 récents. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\CONTRIBUTING.md` | A corriger (P2) | Recommande `mockgen` alors que TESTING.md déclare "pas branché". Incohérence à arbitrer. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\Claude.md` | A corriger (P2) | Stats "103 fichiers source / 89 tests" → réalité 108 / 97. Manque `internal/fibonacci/memory/` & `threshold/` dans l'arbre. Liste pas `internal/sysmon` correctement (juste "Monitoring mémoire" alors que c'est CPU+mem). |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\PRD.md` | OK (read-only) | Stats reprises de Claude.md (donc identiques périmées). Pas à modifier dans cet audit. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\ARCH.md` | A corriger (P1) | Refs INNOVATION.md (lignes 3, 766) inexistantes. Stats ligne 12 ("19 packages, 105 sources, 93 tests") périmées. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\BUILD.md` | A corriger (P1) | Manque `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_MEMORY_LIMIT` dans tableaux env. Mentionne "24 linters" alors que .golangci.yml en active 22. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\TESTING.md` | A corriger (P1) | Référence test files relocalisés : `arena_test.go`, `gc_control_test.go`, `memory_budget_test.go` sont dans `internal/fibonacci/memory/`, pas `internal/fibonacci/`. `progress_eta_test.go` est dans `internal/format/`, pas `internal/cli/`. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\PERFORMANCE.md` | A corriger (P1) | Documente `FIBCALC_GC_CONTROL` (ligne 159) qui n'existe pas dans `internal/config/env.go`. Tableau benchmarks de référence non daté ni source-tracked. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\CALIBRATION.md` | OK | Cohérent avec `internal/calibration/`. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\TUI_GUIDE.md` | A corriger (P0) | Liens `architecture/patterns/design-patterns.md` (lignes 7, 391) — fichier inexistant. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\README.md` | A corriger (P1) | Annonce "Patterns/" avec `Strategy/Observer/Decorator…` mais le dossier ne contient que `interface-hierarchy.mermaid`. Renvoie vers `ARCH.md#14-architectural-decision-records-adr` qui existe (OK). |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\validation\validation-report.md` | A corriger (P0) | Annonce 4 documents `flows/*.md` créés (cli-flow.md, tui-flow.md, config-flow.md, algorithm-flows.md) — seuls les `.mermaid` existent. Référence `docs/calibration/CALIBRATION.md` (ligne 170) — fichier réel : `docs/CALIBRATION.md`. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\FAST_DOUBLING.md` | OK | Identités cohérentes avec `internal/fibonacci/fastdoubling.go`. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\MATRIX.md` | OK (à vérifier) | Non audité en détail (effort hors-scope), structure cohérente avec `internal/fibonacci/matrix*.go`. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\FFT.md` | OK | Cohérent avec `internal/bigfft/`. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\BIGFFT.md` | A corriger (P0) | Référence ligne 694 : `architecture/patterns/design-patterns.md` (inexistant). |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\COMPARISON.md` | OK | Tableau d'opérations cohérent. Bench bench config "Ryzen 9 5900X" — différente du README ("Intel Core Ultra 9 275HX"). Choisir une source unique. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\GMP.md` | OK | Léger (106 lignes), cohérent avec build tag. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\PROGRESS_BAR_ALGORITHM.md` | OK | Modèle géométrique correct vs implémentation. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\system-context.mermaid` | OK | Diagramme C4 L1 correct. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\container-diagram.mermaid` | OK | Diagramme C4 L2 correct (corrections passées appliquées). |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\component-diagram.mermaid` | OK | Conforme aux interfaces réelles. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\dependency-graph.mermaid` | OK | Cohérent avec `go list -deps`. Manque `internal/progress` et `internal/fibonacci/{memory,threshold}` comme nœuds explicites. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\flows\*.mermaid` | OK | 6 fichiers présents, syntaxe Mermaid valide. |
| `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\patterns\interface-hierarchy.mermaid` | OK | Seul fichier du dossier (autres patterns évoqués mais absents). |

---

## Matrice doc.go

Sources de vérité : `internal/**/doc.go` + recherche `^// Package` dans chaque package.

| Package (chemin) | doc.go présent | Qualité | Notes |
|---|---|---|---|
| `cmd/fibcalc` | N/A | bon | `// Package main` dans `main.go` (3 lignes idiomatiques). |
| `cmd/generate-golden` | N/A | bon | `// Package main` dans `main.go`. |
| `internal/app` | OUI (`internal/app/doc.go`) | trivial | 1 phrase ; sous-spécifie le DI/`WithFactory`/dispatch. |
| `internal/bigfft` | OUI | TRIVIAL (P1) | "implements multiplication of big.Int using FFT." — 1 ligne. Package complexe (~17 fichiers, Fermat, bump, cache, SIMD) mérite ≥ 1 paragraphe. |
| `internal/calibration` | OUI | TRIVIAL (P1) | 1 phrase ; pas mention 3-tier (cached/quick/full), profile JSON, etc. |
| `internal/cli` | OUI | bon | 3 lignes, couvre output/progress/completion/presenter. |
| `internal/config` | OUI | bon | Mentionne `ApplyAdaptiveThresholds`, `DetectHardwareHeuristic`. |
| `internal/errors` | OUI (déclaré `apperrors`) | bon | Décrit wrapping et `CalculationError`. |
| `internal/fibonacci` | OUI | bon | Décrit `Calculator`, stratégies Fast Doubling/Matrix/FFT, pooling. |
| `internal/fibonacci/fibonaccitest` | OUI | bon | Doc du `CoreStub` claire. |
| `internal/fibonacci/memory` | NON (P1) | manquant | Sous-package critique (arena, GC controller, budget). Aucun fichier `doc.go` ni `// Package memory` doc-string en tête d'un fichier. |
| `internal/fibonacci/threshold` | NON (P1) | manquant | Sous-package `DynamicThresholdManager`. Aucun `doc.go`. |
| `internal/format` | NON (P1) | manquant | Aucun `doc.go` ni package comment. |
| `internal/metrics` | NON (P1) | manquant | Aucun `doc.go` ni package comment. |
| `internal/orchestration` | OUI | bon | Mentionne `ProgressReporter`, `ResultPresenter`. |
| `internal/parallel` | OUI | TRIVIAL (P2) | "Package parallel provides utilities for concurrent operations." — devrait mentionner `ErrorCollector`. |
| `internal/progress` | NON (P1) | manquant | Pattern observer extrait — pas de `doc.go`. |
| `internal/sysmon` | OUI (en tête de `sysmon.go`, pas de `doc.go`) | bon | 1 ligne dans `sysmon.go` ; suffisant pour package utilitaire. |
| `internal/testutil` | OUI | TRIVIAL (P2) | 1 phrase ; pourrait lister utilitaires (`StripAnsiCodes`, etc.). |
| `internal/tui` | OUI | bon | Multi-paragraphes décrivant Elm, intégration orchestration, activation. |
| `internal/ui` | OUI | bon | Décrit thèmes/`NO_COLOR`. |
| `internal/fibonacci/mocks` | (n'existe pas) | N/A | Référencé dans CONTRIBUTING.md & TESTING.md mais aucun fichier généré. |

**Synthèse** : 6 packages sans `doc.go` → 5 réellement problématiques (memory/threshold/format/metrics/progress) + bigfft/calibration triviaux (à enrichir).

---

## Findings

### F-D1 : `docs/INNOVATION.md` et `docs/INNOVEPLAN.md` référencés mais inexistants
- **Sévérité** : P0
- **Fichier:ligne** :
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\README.md:193` (les deux liens)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\README.md:543-544` (arbre projet)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\ARCH.md:3` et `:766` (INNOVATION uniquement)
- **Diagnostic** : 4 références à `docs/INNOVATION.md` + 1 à `docs/INNOVEPLAN.md`. Aucun de ces fichiers n'existe (`ls docs/` confirmé). L'ADR-010 dans `docs/ARCH.md` argumente sur la base d'un document fantôme.
- **Diff proposé** :
  ```diff
  -> **Innovation & delivery plan**: product and technical tracks are listed in [docs/INNOVATION.md](docs/INNOVATION.md); the operational checklist and task status table (including completed P3 CPU heuristics and the documented P4 decision on research backends) live in [docs/INNOVEPLAN.md](docs/INNOVEPLAN.md).
  +> **Innovation & delivery plan**: voir l'historique git et les ADR consolidés dans [docs/ARCH.md §14](docs/ARCH.md#14-architectural-decision-records-adr). (TODO: créer docs/INNOVATION.md ou retirer ces références.)
  ```
  Idem pour `docs/ARCH.md:3` : retirer ou créer le fichier cible. Pour `docs/ARCH.md:766` : retirer la référence INNOVATION.md ou la transformer en mention historique. Idem dans le bloc arbre `Project Structure` (lignes 543-544 README).
- **Effort** : S (3 chemins à corriger / ou créer 1 fichier squelette).

### F-D2 : `docs/architecture/patterns/design-patterns.md` référencé mais inexistant
- **Sévérité** : P0
- **Fichier:ligne** :
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\TUI_GUIDE.md:7`
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\TUI_GUIDE.md:391`
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\BIGFFT.md:694`
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\validation\validation-report.md:75`
- **Diagnostic** : Le dossier `docs/architecture/patterns/` ne contient que `interface-hierarchy.mermaid`. Le rapport de validation B2 (validation-report.md ligne 75) prétend l'avoir audité mais le fichier est absent. Le contenu attendu existe en réalité dans `docs/ARCH.md §5 (Design Patterns)`.
- **Diff proposé** : Remplacer chaque lien par `docs/ARCH.md#5-design-patterns-14-patterns` :
  ```diff
  -For the bridge pattern and interface-based decoupling, see [Design Patterns](architecture/patterns/design-patterns.md).
  +For the bridge pattern and interface-based decoupling, see [Design Patterns (ARCH.md §5)](ARCH.md#5-design-patterns-14-patterns).
  ```
- **Effort** : S.

### F-D3 : `validation-report.md` annonce 4 docs flows `.md` jamais créés
- **Sévérité** : P0
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\validation\validation-report.md:148-156`
- **Diagnostic** : Le rapport déclare avoir créé `flows/cli-flow.md`, `flows/tui-flow.md`, `flows/config-flow.md`, `flows/algorithm-flows.md`. Le dossier `docs/architecture/flows/` ne contient que des `.mermaid` (cli-flow.mermaid, tui-flow.mermaid, etc.). Le rapport est trompeur quant à l'état du repo.
- **Diff proposé** :
  ```diff
  -| `flows/cli-flow.md` | Complete CLI mode execution path: ... |
  -| `flows/tui-flow.md` | TUI mode execution path: ... |
  -| `flows/config-flow.md` | Configuration resolution: ... |
  -| `flows/algorithm-flows.md` | Per-algorithm execution: ... |
  +| `flows/cli-flow.mermaid` | Diagramme Mermaid du flux CLI |
  +| `flows/tui-flow.mermaid` | Diagramme Mermaid du flux TUI |
  +| `flows/config-flow.mermaid` | Diagramme Mermaid de résolution config |
  +| `flows/fastdoubling.mermaid`, `matrix.mermaid`, `fft-pipeline.mermaid` | Diagrammes Mermaid des flux algos |
  ```
  Idem ligne 170 : `docs/calibration/CALIBRATION.md` → `docs/CALIBRATION.md`.
- **Effort** : S.

### F-D4 : `validation-report.md` cite chemin `docs/calibration/CALIBRATION.md` inexistant
- **Sévérité** : P0
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\validation\validation-report.md:170`
- **Diagnostic** : Le fichier réel est `docs/CALIBRATION.md`.
- **Diff proposé** :
  ```diff
  -- `docs/calibration/CALIBRATION.md`
  +- `docs/CALIBRATION.md`
  ```
- **Effort** : S.

### F-D5 : `FIBCALC_GC_CONTROL` documenté sans implémentation
- **Sévérité** : P1
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\PERFORMANCE.md:159`
- **Diagnostic** : Documenté comme override de `--gc-control`, mais `internal/config/env.go` (table `envOverrides`) n'a aucune entrée `GC_CONTROL`. L'utilisateur croit pouvoir piloter le GC via env, sans effet.
- **Diff proposé** : Soit ajouter l'override dans env.go, soit corriger la doc :
  ```diff
  -Configure via `--gc-control` or `FIBCALC_GC_CONTROL`.
  +Configure via `--gc-control`. (Pas d'override env supporté à ce jour ; cf. `internal/config/env.go`.)
  ```
- **Effort** : S (doc) ou M (impl).

### F-D6 : `FIBCALC_LAST_DIGITS` & `FIBCALC_TUI_THEME` non documentés
- **Sévérité** : P1
- **Fichier:ligne** :
  - README.md:438-457 (tableau env)
  - `.env.example` (15 vars listées)
  - `docs/BUILD.md:251-289` (sections env)
- **Diagnostic** :
  - `--last-digits` n'a pas de variable env (intentionnel ?). Si oui, ajouter une note explicite.
  - `FIBCALC_TUI_THEME` est documenté dans le commentaire de `internal/config/env.go:203` mais absent de README/BUILD.md/.env.example. Lu par `internal/ui/themes.go`.
- **Diff proposé** : Ajouter `FIBCALC_TUI_THEME` au tableau Configuration du README, à BUILD.md, et au `.env.example` :
  ```diff
  +| `FIBCALC_TUI_THEME`           | TUI palette (e.g. `high-contrast`)             |             |
  ```
- **Effort** : S.

### F-D7 : `.env.example` incomplet vs README/BUILD.md
- **Sévérité** : P1
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\.env.example` (15 vars vs 17 documentées dans README ligne 440-457)
- **Diagnostic** : Manquent `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_MEMORY_LIMIT`, `FIBCALC_TUI_THEME`. README/BUILD se réfèrent à `.env.example` comme source de vérité.
- **Diff proposé** : ajouter les 3 entrées manquantes :
  ```diff
  +# Machine-readable CLI output (no ANSI colors); for scripts/CI
  +# Type: bool
  +# Default value: false
  +FIBCALC_MACHINE_OUTPUT=false
  +
  +# Memory budget (e.g., 8G, 512M). Warns if estimate exceeds limit.
  +# Type: string
  +# Default value: (none)
  +FIBCALC_MEMORY_LIMIT=
  +
  +# TUI palette (e.g., high-contrast)
  +# Type: string
  +# Default value: (none)
  +FIBCALC_TUI_THEME=
  ```
- **Effort** : S.

### F-D8 : Chemins de tests dans `TESTING.md` obsolètes après extraction sous-packages
- **Sévérité** : P1
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\TESTING.md:281, 283`
- **Diagnostic** : Ligne 281 cite `arena_test.go`, `gc_control_test.go`, `memory_budget_test.go` dans `internal/fibonacci` ; ils sont en réalité dans `internal/fibonacci/memory/`. Ligne 283 cite `progress_eta_test.go` dans `internal/cli` ; il est dans `internal/format/`. Ces déplacements correspondent au CHANGELOG `[Unreleased]`.
- **Diff proposé** :
  ```diff
  -| `internal/fibonacci` | `fibonacci_test.go`, `fibonacci_golden_test.go`, ..., `arena_test.go`, `gc_control_test.go`, `memory_budget_test.go`, `modular_test.go` | ... |
  +| `internal/fibonacci` | `fibonacci_test.go`, `fibonacci_golden_test.go`, `fibonacci_fuzz_test.go`, `fibonacci_property_test.go`, `fibonacci_strassen_test.go`, `modular_test.go` | ... |
  +| `internal/fibonacci/memory` | `arena_test.go`, `gc_control_test.go`, `budget_test.go` | Arena allocation, GC control, memory budget |
  +| `internal/fibonacci/threshold` | `manager_test.go` | Dynamic threshold manager |
  +| `internal/format` | `progress_eta_test.go` | Duration/ETA formatting |
  -| `internal/cli` | `output_test.go`, `ui_test.go`, `goldens_test.go`, `progress_eta_test.go` | ... |
  +| `internal/cli` | `output_test.go`, `ui_test.go`, `goldens_test.go`, `provider_test.go`, `completion_test.go` | ... |
  ```
- **Effort** : S.

### F-D9 : `BUILD.md` annonce "24 linters" — `.golangci.yml` en active 22
- **Sévérité** : P1
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\BUILD.md:207`
- **Diagnostic** : Compté manuellement dans `.golangci.yml` (lignes 14-45) : 22 linters explicitement listés. Le rapport B4 prétendait corriger 22→24 (validation-report.md:179, 194). En l'état, c'est 22 (cohérent avec `Claude.md:64` et `PRD.md:39`). Donc BUILD.md est faux.
- **Diff proposé** :
  ```diff
  -The project uses `golangci-lint` with 24 linters configured in `.golangci.yml`.
  +The project uses `golangci-lint` with 22 linters configured in `.golangci.yml`.
  ```
- **Effort** : S.

### F-D10 : `Claude.md` stats périmées
- **Sévérité** : P2
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\Claude.md:10`
- **Diagnostic** : Annonce "~31 900 lignes Go, 103 fichiers source, 89 fichiers de test". Mesure réelle (find + wc) : **~35 560 lignes**, **108 sources non-test**, **97 tests**.
- **Diff proposé** :
  ```diff
  -- **Taille** : ~31 900 lignes Go, 103 fichiers source, 89 fichiers de test
  +- **Taille** : ~35 560 lignes Go, 108 fichiers source, 97 fichiers de test (mesure 2026-04)
  ```
  Note : PRD.md:13 reprend les anciennes stats (synchroniser).
- **Effort** : S.

### F-D11 : `Claude.md` arbre `internal/` ne reflète pas les sous-packages
- **Sévérité** : P2
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\Claude.md:18-37`
- **Diagnostic** : N'inclut pas `internal/fibonacci/memory/` ni `internal/fibonacci/threshold/` ni `internal/fibonacci/fibonaccitest/`, qui sont pourtant des packages distincts. `internal/sysmon` décrit comme "Monitoring mémoire système" alors que `sysmon.go` couvre CPU+memory.
- **Diff proposé** :
  ```diff
       fibonacci/         # CŒUR : Fast Doubling, Matrix Exp., FFT, Strassen, GMP
  +      memory/          # Arena, GC controller, memory budget
  +      threshold/       # Dynamic threshold manager
  +      fibonaccitest/   # CoreStub pour tests externes
  ...
  -    sysmon/            # Monitoring mémoire système
  +    sysmon/            # Monitoring CPU + mémoire système (gopsutil)
  ```
- **Effort** : S.

### F-D12 : `docs/ARCH.md` stats divergentes
- **Sévérité** : P2
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\ARCH.md:12`
- **Diagnostic** : Annonce "19 Go packages | 105 source files | 93 test files | 19 Markdown files". Mesuré : **22 packages** (`go list ./...`), **108 sources**, **97 tests**, **19 .md** (OK uniquement pour le compte Markdown).
- **Diff proposé** :
  ```diff
  -- **Codebase stats:** 19 Go packages (`go list ./...`) | 105 source (non-`*_test.go`) files | 93 test files | 19 Markdown files at repo root and under `docs/`
  +- **Codebase stats:** 22 Go packages (`go list ./...`) | 108 source files | 97 test files | 19 Markdown files (mesure 2026-04)
  ```
- **Effort** : S.

### F-D13 : `CHANGELOG.md` `[Unreleased]` sous-représente les commits perf récents
- **Sévérité** : P1
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\CHANGELOG.md:8-28`
- **Diagnostic** : Les commits `git log --oneline -20` mentionnent : `f499f66 perf: explicit lock release` ; `3035984 perf: optimize PowerShell completion` ; `fe8073e perf: optimize slice allocations in NTransform` ; `117e24e refactor: ExecutionConfig` ; `f09bae4 fix(security): file permissions` ; `3c7bff4 perf: FFT cache key FNV-1a` ; `6a5e0d7 feat(fibonacci): dynamic FFT caching (O-02)` ; `6ae2739 feat(calibration): FFT threshold profiling (O-01)`. Aucun n'est dans `[Unreleased]`.
- **Diff proposé** : ajouter sous `[Unreleased]` :
  ```diff
  ### Added
  + - **Dynamic FFT caching** (O-02): runtime adaptive FFT transform cache
  + - **Fine-grained FFT threshold profiling** (O-01) in calibration

  ### Changed
  + - perf: explicit lock release avoids defer overhead (`#15`)
  + - perf: PowerShell completion script generation optimised (`#14`)
  + - perf: NTransform slice allocations reduced (`#13`)
  + - perf: FFT cache key FNV-1a allocation-free (`#10`)
  + - refactor: `ExecuteCalculations` parameters consolidated into `ExecutionConfig` (`#12`)

  ### Fixed
  + - security: restrict file/dir permissions to owner-only (`#11`)
  ```
- **Effort** : S.

### F-D14 : `CONTRIBUTING.md` recommande `mockgen` alors que `TESTING.md` indique non branché
- **Sévérité** : P2
- **Fichier:ligne** :
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\CONTRIBUTING.md:291-339` (section Mock Generation détaillée + tableau de mocks)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\TESTING.md:208` ("Mock generation infrastructure ... is configured in the Makefile but is not currently wired into the codebase — no `//go:generate` directives or `mocks/` directories exist.")
- **Diagnostic** : Contradiction directe. `internal/fibonacci/mocks/`, `internal/cli/mocks/` n'existent pas (`ls` confirmé). Le tableau CONTRIBUTING (lignes 316-321) est trompeur.
- **Diff proposé** : aligner CONTRIBUTING sur TESTING — soit retirer la section Mock, soit la marquer "Planned":
  ```diff
  -## Mock Generation
  -
  -This project uses [mockgen](https://github.com/uber-go/mock) for generating test mocks automatically.
  +## Mock Generation (planifié)
  +
  +Le projet a des cibles `make generate-mocks` / `make install-mockgen` mais aucun `//go:generate`
  +ni dossier `mocks/` n'existe à ce jour. Voir `docs/TESTING.md` (section Mock Generation).
  ```
- **Effort** : S.

### F-D15 : `doc.go` manquants pour 5 packages
- **Sévérité** : P1
- **Fichier:ligne** :
  - `internal/fibonacci/memory/` (aucun fichier `doc.go`, aucun commentaire `// Package memory`)
  - `internal/fibonacci/threshold/` (idem)
  - `internal/format/` (idem)
  - `internal/metrics/` (idem)
  - `internal/progress/` (idem)
- **Diagnostic** : Viole `CLAUDE.md` directive #5 ("Chaque package a un `doc.go`") et règle `revive: package-comments`. `golangci-lint` ne hurle pas car `revive` ne signale qu'à l'export niveau, mais `package-comments` est listé.
- **Diff proposé** : créer 5 fichiers `doc.go`. Exemple pour `internal/format/doc.go` :
  ```go
  // Package format provides shared formatting helpers for durations, large
  // numbers, and progress ETA, used by both the CLI and TUI presentation layers.
  package format
  ```
  (Idem pour memory : arena/GC/budget ; threshold : DynamicThresholdManager ; metrics : MemoryCollector + indicators ; progress : observer pattern.)
- **Effort** : S.

### F-D16 : `doc.go` triviaux (1 phrase) à enrichir
- **Sévérité** : P2
- **Fichier:ligne** :
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\internal\bigfft\doc.go` (1 ligne pour ~17 fichiers)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\internal\calibration\doc.go` (1 ligne)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\internal\app\doc.go` (1 phrase)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\internal\parallel\doc.go`
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\internal\testutil\doc.go`
- **Diagnostic** : Pour les packages les plus complexes (bigfft, calibration), 1 phrase est insuffisant. Comparer à la qualité de `internal/tui/doc.go` (multi-paragraphes, activation, intégration).
- **Diff proposé** (ex. bigfft) :
  ```go
  // Package bigfft implements multiplication and squaring of math/big big.Int
  // values via a Schönhage-Strassen FFT operating on the Fermat ring Z/(2^k+1).
  //
  // The package exposes Mul, MulTo, Sqr, SqrTo and is used by internal/fibonacci
  // when operands exceed the FFTThreshold (default 500k bits). Internals include:
  //   - Configurable parallel FFT recursion (FFTParallelismConfig)
  //   - Thread-safe LRU transform cache for reuse across mul/sqr in one step
  //   - Size-class object pools with adaptive pre-warming
  //   - Bump allocator for batch temporary allocations (O(1) reset)
  //   - go:linkname into math/big for vector arithmetic on amd64 (AVX2 detection)
  package bigfft
  ```
- **Effort** : S par package.

### F-D17 : Benchmarks README vs PERFORMANCE/COMPARISON contradictoires (matériel + chiffres)
- **Sévérité** : P2
- **Fichier:ligne** :
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\README.md:351-359` ("Intel Core Ultra 9 275HX, 24 cores" ; F(10M) Fast=60ms)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\PERFORMANCE.md:11-26` ("AMD Ryzen 9 5900X" ; F(10M) Fast=2.1s)
  - `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\algorithms\COMPARISON.md:78-80` ("AMD Ryzen 9 5900X")
- **Diagnostic** : Trois sources, deux machines, chiffres incohérents (~30× écart sur F(10M)). Aucun n'est daté ni reproductible. Le README cite des chiffres très optimistes sans source-tracking.
- **Diff proposé** : choisir 1 source (PERFORMANCE.md) et faire pointer README + COMPARISON dessus, en indiquant la machine de référence et `make bench-versioned` comme procédure de mise à jour.
- **Effort** : M (vérifier chiffres en exécutant `make bench-versioned` sur une machine repère).

### F-D18 : README badge coverage `80%` non vérifié
- **Sévérité** : P2
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\README.md:5`
- **Diagnostic** : Hardcodé. La cible `coverage` produit `coverage.html` ; le pourcentage réel n'est pas extrait (ni archivé dans `bench/baseline/coverage.txt` au format brut). À mettre à jour sur la base de la baseline générée par les autres équipes.
- **Diff proposé** : remplacer par un badge dynamique ou retirer la précision exacte.
- **Effort** : S.

### F-D19 : `docs/architecture/README.md` annonce `Patterns/` riche
- **Sévérité** : P2
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\README.md:30-37`
- **Diagnostic** : Énumère "14 design patterns documentés ici" et liste Strategy/Observer/Decorator. Le dossier `patterns/` ne contient qu'`interface-hierarchy.mermaid`. Le contenu existe ailleurs (`docs/ARCH.md §5`).
- **Diff proposé** : remplacer la section par un renvoi explicite vers `docs/ARCH.md#5-design-patterns-14-patterns`.
- **Effort** : S.

### F-D20 : `dependency-graph.mermaid` ne représente pas `internal/progress` ni sous-packages fibonacci
- **Sévérité** : P2
- **Fichier:ligne** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\docs\architecture\dependency-graph.mermaid`
- **Diagnostic** : `internal/progress`, `internal/fibonacci/memory`, `internal/fibonacci/threshold`, `internal/fibonacci/fibonaccitest` n'apparaissent pas comme nœuds (extractions récentes — voir CHANGELOG `[Unreleased]`).
- **Diff proposé** : ajouter ces 4 nœuds + arêtes correctes.
- **Effort** : S.

---

## Annexe : plan de condensation README

### Diagnostic

`README.md` = 585 lignes / ~38 Ko. Sections > 80 lignes :

| Section | Lignes ~ | Contenu |
|---|---|---|
| Key Features | 47-83 (~36) | Long inventaire ; déjà résumé dans CLAUDE.md/ARCH.md |
| Usage Guide | 213-346 (~133) | Tableaux flags + TUI ASCII + 7 exemples |
| Performance Benchmarks | 349-368 (~19) | Court mais redondant avec PERFORMANCE.md |
| Configuration | 434-458 (~24) | Tableau env vars (chevauche BUILD.md) |
| Development | 461-564 (~103) | Liste exhaustive des targets Make + arbre projet |

### Plan proposé (sans appliquer)

1. **Garder dans README** (≤ 250 lignes cible) :
   - Overview, Quick Start, Mathematical Background résumé, lien vers `docs/algorithms/`.
   - Architecture : un seul diagramme Mermaid + lien vers `docs/ARCH.md` et `docs/architecture/README.md`.
   - Installation + 1 exemple usage canonique.
   - Section "Documentation" pointant vers les guides spécialisés.

2. **Extraire vers `docs/USAGE.md`** : Usage Guide complet (flags, env, exemples, TUI ASCII).

3. **Renvoyer vers `docs/PERFORMANCE.md`** : tableaux de benchmarks (et résoudre F-D17 simultanément).

4. **Renvoyer vers `docs/BUILD.md` + `docs/TESTING.md`** : sections Development / Project Structure / Testing.

5. **Section Configuration** : factoriser dans `.env.example` + `docs/BUILD.md` (déjà bien fait), README ne garde qu'un mini-extrait + lien.

### Bénéfices attendus

- README ≤ 250 lignes (lisible en une scroll) ; ~33 % du volume actuel.
- Élimine doublons entre README ↔ BUILD.md (env vars), README ↔ COMPARISON/PERFORMANCE (benchmarks), README ↔ ARCH.md (architecture).
- Facilite mise à jour : un seul lieu de vérité par sujet.

### Effort

- Restructuration : M (1 PR ciblée doc).
- Aucune modification de code.
