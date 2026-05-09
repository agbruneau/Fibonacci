# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique démontrant Clean Architecture, pooling mémoire, parallélisme adaptatif et optimisation PGO.

> **🔄 REFACTORING EN COURS** — Le repo est dans une période de refactoring planifié.
> - Audit exhaustif : [`ultrareview.md`](ultrareview.md) — 37 recommandations identifiées.
> - Plan d'exécution parallèle : [`ultrareviewplan.md`](ultrareviewplan.md) — tableau de suivi à jour.
> - **Avant tout changement non trivial dans `internal/fibonacci/` ou `internal/bigfft/`, consulter ces deux documents.**

---

## Projet

- **Module** : `github.com/agbru/fibcalc`
- **Go** : 1.25.0+ (toolchain 1.26.2)
- **Licence** : Apache 2.0
- **Taille (avant refactoring)** : ~36 500 LOC source + ~20 400 LOC tests, 21 packages (19 internes + 2 sous `cmd/`)
- **Cible post-refactoring (vague 4 complète)** : ~34 500 LOC, 17-18 packages
- **CI/CD** : ❌ aucun workflow GitHub Actions à ce jour (cf. R4.1, priorité absolue)

---

## Architecture (Clean Architecture, 4 couches)

```
cmd/
  fibcalc/           # Point d'entrée CLI principal (délègue à app)
  generate-golden/   # Oracle indépendant pour golden tests (P2-04)
internal/
  app/               # Lifecycle, dispatch, version
  bigfft/            # Multiplication FFT (Schönhage-Strassen), allocateur bump
  calibration/       # Auto-calibration adaptative, micro-benchmarks
  cli/               # Interface CLI, formatage, complétion shell
  config/            # Parsing config, flags, variables d'environnement
  errors/            # Types d'erreurs structurées (ConfigError, CalcError)
  fibonacci/         # CŒUR : Fast Doubling, Matrix Exp., FFT, Strassen, GMP
    fibonaccitest/   # Doubles de test pour CoreCalculator
    memory/          # Arena, GCController, budget mémoire
    threshold/       # Gestionnaire dynamique de seuils (FFT/parallèle/Strassen)
  format/            # Formatage durées, nombres, ETA
  metrics/           # Indicateurs de performance, monitoring mémoire
  orchestration/     # Exécution concurrente (errgroup), agrégation
  parallel/          # ⚠ Quasi-mort (ErrorCollector inutilisé) — voir R2.2
  progress/          # Pattern observer + DTO progression
  sysmon/            # Sample CPU/mem (28 LOC) — voir R4.5 (fusion vers metrics)
  testutil/          # Helpers de test partagés
  tui/               # Dashboard TUI interactif (Bubble Tea)
  ui/                # Thèmes couleur — voir R2.7 (fusion ui+tui/styles)
docs/
  architecture/      # Diagrammes C4 (Mermaid), validation
  algorithms/        # Documentation mathématique par algorithme
  audits/            # Rapports d'audits archivés + baselines benchmark
```

---

## Algorithmes

1. **Fast Doubling** (défaut) — O(log n), identité F(2k) = F(k)(2F(k+1) − F(k))
2. **Matrix Exponentiation** — O(log n), Strassen pour grandes matrices
3. **FFT (Schönhage-Strassen)** — Seuil adaptatif (~500k bits par défaut)
4. **GMP** (build tag `gmp`) — Backend GNU Multiple Precision

---

## Patterns de performance critiques

- **sync.Pool** pour `big.Int` — réduction GC >95 %
- **State + arena unifiés** — `CalculationState` owne sa `CalculationArena` ; mêmes `[]big.Word` réutilisés entre appels via `AcquireStateForN`/`ReleaseStateWithResult`.
  - ⚠ **Bug latent connu (P1-04 incomplet)** : `clearStateAliases` peut être contournée selon la branche `overLimit=true`. Cf. R1.1 dans le plan.
- **Allocateur bump** pour FFT — O(1), zéro fragmentation
- **GC désactivé** pendant calculs N ≥ 1M
  - ⚠ **Pas panic-safe** : si un panic survient entre `Begin()` et `End()`, GC reste off. Cf. R1.2.
- **Parallélisme adaptatif** via sémaphore (`NumCPU()`)
- **Cache FFT** LRU thread-safe — 15-30 % speedup
- **PGO** supporté via `make build-pgo`

---

## ⚠ Bugs latents connus (à corriger en Vague 1)

Ces bugs n'ont **pas encore été corrigés** au moment de la rédaction. Si on touche au code concerné, **lire d'abord la recommandation correspondante dans `ultrareview.md`**.

| ID | Fichier | Problème | Sévérité |
|----|---------|----------|----------|
| R1.1 | `internal/fibonacci/fastdoubling.go:215-329` | `clearStateAliases` pas toujours appelée → use-after-free latent | Critique |
| R1.2 | `internal/fibonacci/memory/gc_control.go:63-97` | GC désactivé persistant en cas de panic | Haute |
| R1.3 | `internal/calibration/calibration.go:251-321` | `IsStale` jamais invoqué → profils obsolètes acceptés | Haute |
| R1.4 | `internal/bigfft/pool.go:111-123` | `releaseWordSlice` perd silencieusement les buffers resizés | Critique |
| R1.5 | `internal/bigfft/fft_cache.go:262-303` | `putByKey` alloue eagerly même en eviction | Critique |

---

## ⚠ Modules sensibles (changements à risque)

Ces fichiers concentrent la complexité ou des couplages cachés. **Toute modification doit citer la section correspondante de `ultrareview.md`** dans le message de commit.

| Fichier | Risque |
|---------|--------|
| `internal/fibonacci/fastdoubling.go` | Hot path + pooling state+arena (R1.1, R3.10). |
| `internal/fibonacci/doubling_framework.go` | Boucle critique 137 L couplée à `bigfft` (R3.2). |
| `internal/fibonacci/threshold/manager.go` | 417 L, 7 responsabilités (R3.1). |
| `internal/bigfft/fft_cache.go` | 534 L + globals (R1.5, R2.10, R3.9). |
| `internal/bigfft/pool.go` | 467 L + 13 pools globaux (R1.4, R2.3). |
| `internal/bigfft/fermat.go` | Panics au lieu d'errors (R3.8). |
| `internal/tui/model.go` | 425 L, Update à 16 messages (R3.4). |
| `internal/cli/completion.go` | 520 L, 4 implémentations shell (R3.5). |
| `internal/fibonacci/testdata/fibonacci_golden.json` | **Immuable** sans accord explicite. |

---

## Commandes essentielles

```bash
make all             # clean + build + test
make test            # Tests avec race detector
make test-short      # Tests rapides
make coverage        # Rapport couverture HTML
make benchmark       # Benchmarks (à exécuter avant/après tout changement perf-sensitive)
make bench-versioned # Benchmarks avec versionnage
make lint            # golangci-lint (22 linters)
make build-pgo       # Build avec PGO
make build-all       # Cross-compilation (linux, windows, macOS)
```

---

## Conventions de code

- Packages par responsabilité (pas par feature).
- Interfaces étroites (ISP) : `Multiplier`, `DoublingStepExecutor`, `Calculator`, `ProgressReporter`.
- Erreurs structurées : `fmt.Errorf("%w", err)`. **Pas de panic** sauf pour invariants internes (cf. R3.8 pour bigfft/fermat).
- Tests parallèles (`t.Parallel()`) systématiques (78 % aujourd'hui, cible 100 % cf. R4.12).
- Race detector en CI (à activer cf. R4.1).
- Complexité cyclomatique max 15, cognitive max 30 (cf. `.golangci.yml`).
- Longueur fonction max 100 lignes / 50 statements.
- `doc.go` pour chaque package public (100 % aujourd'hui).
- **Pas d'emoji** dans le code.
- **Commentaires uniquement quand le « pourquoi » n'est pas évident** (pas de description du « quoi »).

---

## Directives projet (période de refactoring)

> Les lignes directrices comportementales générales (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution) sont dans `~/.claude/CLAUDE.md` et s'appliquent ici. Ci-dessous : spécificités FibGo en période de refactoring actif.

1. **Performance critique** — Pas d'allocations inutiles. Toute modification dans `internal/fibonacci/` ou `internal/bigfft/` doit être vérifiée avec `make benchmark` (avant + après). Régression > 5 % = blocage.

2. **Golden tests obligatoires** — Tout changement algorithmique doit passer `internal/fibonacci/testdata/fibonacci_golden.json`. Le fichier golden est **immuable** sans approbation explicite.

3. **Étanchéité des couches** — `internal/` ne doit pas fuiter vers `cmd/` directement. Respecter la hiérarchie Clean Architecture (`cmd → app → orchestration → fibonacci/bigfft → config/errors`).

4. **Concurrence contrôlée** — `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutines sans contrôle de cycle de vie. Pas de nouveaux globals dans `bigfft/` (cf. R3.7 vise à les éliminer).

5. **Codebase mature mais en refactoring** — Modifications chirurgicales uniquement. **Tout refactoring d'envergure (> 50 LOC modifiées sur > 2 fichiers) doit être tracé dans `ultrareviewplan.md`** ou faire l'objet d'une nouvelle entrée Rx.y.

6. **Tableau de suivi à maintenir** — À chaque transition de statut d'une tâche du plan (ex : `🟡 InProgress` → `✅ Done`), mettre à jour le tableau de [`ultrareviewplan.md`](ultrareviewplan.md) avec le SHA du commit correspondant.

7. **Pas de nouveaux fichiers `progress*` sans consultation** — La couche progression est en cours de consolidation (cf. R2.5). Ajouter à un endroit existant après lecture.

8. **Pas de nouveaux globals dans `bigfft/`** — La direction est inverse (cf. R3.7 introduira `FFTContext` injectable).

9. **Bug fix avant refactor** — Si un bug latent listé ci-dessus est touché par hasard, le corriger en priorité (PR isolée) avant le refactor planifié.

10. **CI absente — vérifier manuellement** — En attendant R4.1, exécuter `make test -race && make lint` localement avant chaque PR. Aucun workflow ne le fera pour vous.

---

## Workflow recommandé pour une nouvelle modification

```
1. Identifier la tâche dans ultrareviewplan.md (ID Rx.y).
   - Si la tâche n'existe pas, l'ajouter au tableau (statut ⬜ Pending).
2. Marquer la tâche 🟡 InProgress dans le tableau.
3. Branche dédiée : git checkout -b refactor/Rx.y-description-courte
4. Modification + tests.
5. make test -race && make lint && (make benchmark si perf-sensitive)
6. Comparer aux baselines dans docs/audits/.
7. Commit avec message « refactor(Rx.y): description ».
8. PR, review, merge.
9. Mettre à jour ultrareviewplan.md : ✅ Done + SHA.
10. Commit doc : « docs(plan): Rx.y marked Done a1b2c3d ».
```

---

## Références

- [`ultrareview.md`](ultrareview.md) — audit exhaustif (952 lignes, 8 sections).
- [`ultrareviewplan.md`](ultrareviewplan.md) — plan d'exécution + tableau de suivi.
- [`docs/architecture/`](docs/architecture/) — diagrammes C4, dependency graph.
- [`docs/algorithms/`](docs/algorithms/) — Fast Doubling, Matrix, FFT, GMP, comparaison.
- [`CHANGELOG.md`](CHANGELOG.md) — Keep-a-Changelog format, SemVer.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — workflow contribution.
- `.golangci.yml` — 22 linters configurés, exceptions documentées.
