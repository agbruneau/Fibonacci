# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique démontrant les patterns d'ingénierie logicielle avancés : Clean Architecture, pooling mémoire, parallélisme adaptatif et optimisation PGO.

## Projet

- **Module** : `github.com/agbru/fibcalc`
- **Go** : 1.25.0+
- **Licence** : Apache 2.0
- **Taille** : ~31 900 lignes Go, 103 fichiers source, 89 fichiers de test

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
  algorithms/        # Documentation mathématique des algorithmes
```

## Algorithmes implémentés

1. **Fast Doubling** (défaut) — O(log n), identité F(2k) = F(k)(2F(k+1) - F(k))
2. **Matrix Exponentiation** — O(log n), algorithme de Strassen pour grandes matrices
3. **FFT (Schönhage-Strassen)** — Seuil adaptatif (~500k bits par défaut)
4. **GMP** (optionnel, build tag) — Backend GNU Multiple Precision

## Patterns de performance

- **sync.Pool** pour recyclage `big.Int` (réduction GC >95%)
- **Allocateur bump** pour FFT (O(1), zéro fragmentation)
- **GC désactivé** pendant calculs N ≥ 1M
- **Parallélisme adaptatif** via sémaphore (`NumCPU()*2`)
- **Cache FFT** (LRU thread-safe, 15-30% speedup)
- **PGO** (Profile-Guided Optimization) supporté

## Commandes essentielles

```bash
make all             # clean + build + test
make test            # Tests avec race detector
make test-short      # Tests rapides
make coverage        # Rapport couverture HTML
make benchmark       # Benchmarks
make lint            # golangci-lint (22 linters)
make build-pgo       # Build avec PGO
make build-all       # Cross-compilation (linux, windows, macOS)
```

## Conventions de code

- **Packages par responsabilité** (pas par feature)
- **Interfaces étroites** (ISP) : `Multiplier`, `DoublingStepExecutor`
- **Erreurs structurées** : `fmt.Errorf("%w", err)`
- **Tests parallèles** : `t.Parallel()` systématique
- **Race detector** activé par défaut dans CI
- **Complexité cyclomatique** max 15, cognitive max 30
- **Longueur fonction** max 100 lignes / 50 statements

## Directives pour Claude

1. **Performance critique** : Ce projet optimise au niveau mémoire/GC. Ne pas introduire d'allocations inutiles.
2. **Tests obligatoires** : Tout changement algorithmique doit passer les golden tests (`testdata/fibonacci_golden.json`).
3. **Cohérence architecturale** : Respecter la séparation en couches. Les packages `internal/` ne doivent pas fuiter vers `cmd/` directement.
4. **Linting** : `make lint` doit passer. Respecter les seuils de complexité configurés dans `.golangci.yml`.
5. **Documentation** : Chaque package a un `doc.go`. Maintenir les commentaires de package.
6. **Concurrence** : Utiliser `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutines sans contrôle de cycle de vie.
7. **Modifications chirurgicales** : Ce codebase est mûr — ne pas refactorer sans demande explicite.
