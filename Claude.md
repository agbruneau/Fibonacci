# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique démontrant Clean Architecture, pooling mémoire, parallélisme adaptatif et optimisation PGO.

## Projet

- **Module** : `github.com/agbru/fibcalc`
- **Go** : 1.25.0+ (toolchain 1.26.2)
- **Licence** : Apache 2.0
- **Taille** : ~37 000 lignes Go, 21 packages

## Architecture (Clean Architecture, 4 couches)

```
cmd/
  fibcalc/           # Point d'entrée CLI principal
  generate-golden/   # Générateur de données de test
internal/
  app/               # Cycle de vie applicatif, dispatch, version
  bigfft/            # Multiplication FFT (Schönhage-Strassen), allocateur bump
  calibration/       # Auto-calibration adaptative, micro-benchmarks
  cli/               # Interface CLI, formatage, complétion shell
  config/            # Parsing config, flags, variables d'environnement
  errors/            # Types d'erreurs structurées (ConfigError, CalcError)
  fibonacci/         # CŒUR : Fast Doubling, Matrix Exp., FFT, Strassen, GMP
    fibonaccitest/   # Doubles de test pour CoreCalculator
    memory/          # Arena, GCController, budget mémoire
    threshold/       # Gestionnaire dynamique de seuils (FFT/parallèle)
  format/            # Formatage durées, nombres, ETA
  metrics/           # Indicateurs de performance, monitoring mémoire
  orchestration/     # Exécution concurrente, agrégation résultats
  parallel/          # Utilitaires d'exécution parallèle
  progress/          # Rapports de progression (pattern observer)
  sysmon/            # Monitoring mémoire système
  testutil/          # Utilitaires de test partagés
  tui/               # Dashboard TUI interactif (Bubble Tea)
  ui/                # Thèmes couleur et styling
docs/
  architecture/      # Diagrammes C4 (Mermaid)
  algorithms/        # Documentation mathématique
  audits/            # Rapports d'audits archivés
```

## Algorithmes

1. **Fast Doubling** (défaut) — O(log n), identité F(2k) = F(k)(2F(k+1) − F(k))
2. **Matrix Exponentiation** — O(log n), Strassen pour grandes matrices
3. **FFT (Schönhage-Strassen)** — Seuil adaptatif (~500k bits par défaut)
4. **GMP** (build tag `gmp`) — Backend GNU Multiple Precision

## Patterns de performance critiques

- **sync.Pool** pour `big.Int` — réduction GC >95 %
- **Allocateur bump** pour FFT — O(1), zéro fragmentation
- **GC désactivé** pendant calculs N ≥ 1M
- **Parallélisme adaptatif** via sémaphore (`NumCPU()*2`)
- **Cache FFT** LRU thread-safe — 15-30 % speedup
- **PGO** supporté via `make build-pgo`

## Commandes essentielles

```bash
make all             # clean + build + test
make test            # Tests avec race detector
make test-short      # Tests rapides
make coverage        # Rapport couverture HTML
make benchmark       # Benchmarks
make bench-versioned # Benchmarks avec versionnage
make lint            # golangci-lint (24 linters)
make build-pgo       # Build avec PGO
make build-all       # Cross-compilation (linux, windows, macOS)
```

## Conventions de code

- Packages par responsabilité (pas par feature)
- Interfaces étroites (ISP) : `Multiplier`, `DoublingStepExecutor`
- Erreurs structurées : `fmt.Errorf("%w", err)`
- Tests parallèles (`t.Parallel()`) systématiques, race detector en CI
- Complexité cyclomatique max 15, cognitive max 30 (cf. `.golangci.yml`)
- Longueur fonction max 100 lignes / 50 statements
- `doc.go` pour chaque package public

## Directives projet

> Les lignes directrices comportementales générales (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution) sont dans `~/.claude/CLAUDE.md` et s'appliquent ici. Ci-dessous : spécificités FibGo.

1. **Performance critique** — Pas d'allocations inutiles. Toute modification dans `internal/fibonacci/` ou `internal/bigfft/` doit être vérifiée avec `make benchmark`.
2. **Golden tests obligatoires** — Tout changement algorithmique doit passer `internal/fibonacci/testdata/fibonacci_golden.json`.
3. **Étanchéité des couches** — `internal/` ne doit pas fuiter vers `cmd/` directement. Respecter la hiérarchie Clean Architecture.
4. **Concurrence contrôlée** — `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutines sans contrôle de cycle de vie.
5. **Codebase mature** — Modifications chirurgicales uniquement. Pas de refactor non demandé.
