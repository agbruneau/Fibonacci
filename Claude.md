# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique démontrant Clean Architecture, pooling mémoire, parallélisme adaptatif et optimisation PGO.

> **🔎 ÉTAT POST-AUDIT** — Les vagues de refactoring (R1–R4) sont mergées. Les 5 bugs latents historiques (R1.1–R1.5) sont **corrigés ou infirmés**.
> - Audit de référence : [`audit.md`](audit.md) — 23 constats (A-01…A-23), vérifiés `fichier:ligne`.
> - Plan de remédiation : [`AuditPlanning.md`](AuditPlanning.md) — tableau de suivi à jour (statut + SHA).
> - **Avant tout changement non trivial dans `internal/fibonacci/` ou `internal/bigfft/`, consulter `audit.md` §4–§5 et `AuditPlanning.md`.**

---

## Projet

- **Module** : `github.com/agbru/fibcalc`
- **Go** : 1.25.0+ (toolchain 1.26.2)
- **Licence** : Apache 2.0
- **Taille (mesurée)** : ~35 500 LOC `.go` (source + tests), 24 packages (22 internes + 2 sous `cmd/`)
- **CI/CD** : ✅ GitHub Actions — `ci.yml` (vet + golangci-lint épinglé `v1.64.8` + build, `go test -race -short` sur matrice **3 OS dont Windows** via `CGO_ENABLED=1`, + job `bench` informatif) et `coverage.yml` (sur push+PR, seuil `MIN_COVERAGE=80%`). A-12/A-13/A-14 résolus ; `coverage.yml` désormais versionné (était masqué par un glob `.gitignore`).

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
  cli/               # Interface CLI, formatage
    completion/      # Complétion shell (bash/zsh/fish/powershell, registry unique)
  config/            # Parsing config, flags, variables d'environnement
  errors/            # Types d'erreurs structurées (ConfigError, CalcError)
  fibonacci/         # CŒUR : Fast Doubling, Matrix Exp., FFT, Strassen, GMP
    fibonaccitest/   # Doubles de test pour CoreCalculator
    memory/          # Arena, GCController, budget mémoire
    threshold/       # Gestionnaire dynamique de seuils (FFT/parallèle/Strassen)
  format/            # Formatage durées, nombres, ETA
  metrics/           # Indicateurs de performance, monitoring mémoire
    system/          # Échantillonnage CPU/mém (ex-`sysmon`, fusionné R4.5)
  orchestration/     # Exécution concurrente (errgroup), agrégation
  parallel/          # ErrorCollector — VIVANT : utilisé par fibonacci/common.go
  progress/          # Pattern observer + DTO progression (chemin prod : Freeze)
  testutil/          # Helpers de test partagés
  tui/               # Dashboard TUI interactif (Bubble Tea)
    component/       # Composant TUI réutilisable
  ui/                # Thèmes couleur (source unique, tui/styles en dérive)
docs/
  architecture/      # Diagrammes C4 (Mermaid), validation
  algorithms/        # Documentation mathématique par algorithme
  audits/            # Baselines benchmark (créé en Vague A)
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
- **State + arena unifiés** — `CalculationState` owne sa `CalculationArena` ; mêmes `[]big.Word` réutilisés entre appels via `AcquireStateForN`/`ReleaseStateWithResult`. Aliases détachés via `finalizeStateRelease` (chemin unique, ordre `checkLimit → clearStateAliases → Put`, gardé par `TestReleaseState_OverLimit_AliasesCleared`).
- **Allocateur bump** pour FFT — O(1), zéro fragmentation
- **GC désactivé** pendant calculs N ≥ 1M — **panic-safe** via `gcCtrl.WithGC(fn)` (`defer End()`). `Begin`/`End` directs `Deprecated`.
- **Parallélisme adaptatif** via sémaphore (`NumCPU()`)
- **Cache FFT** LRU thread-safe — 15-30 % speedup. ⚠ Risque résiduel A-01 (recyclage `backing` en éviction vs `pv` aliasé) — voir `audit.md`.
- **PGO** supporté via `make build-pgo`

---

## ⚠ Bugs historiques — RÉSOLUS (référence : `audit.md` §4)

Les 5 bugs latents R1.1–R1.5 ont été corrigés ou infirmés avant cet audit. **Ne pas les « re-corriger ».** Détail et preuves `fichier:ligne` dans `audit.md` §4.

| ID | Statut | Preuve (cf. `audit.md` §4) |
|----|--------|-----------------------------|
| R1.1 | CORRIGÉ | `fastdoubling.go` `finalizeStateRelease` — `clearStateAliases` inconditionnel + test de régression |
| R1.2 | CORRIGÉ | `memory/gc_control.go` `WithGC` + `defer End()` panic-safe |
| R1.3 | CORRIGÉ | `calibration.go` `IsStale` invoqué + branche stale → `CompleteStrategy` |
| R1.4 | CORRIGÉ | `bigfft/pool.go` `releaseWordSlice` route sur `cap` + miss counter |
| R1.5 | INFIRMÉ | `bigfft/fft_cache.go` `putByKey` n'alloue que si aucun backing salvageable |

**Risque résiduel actif** : voir `audit.md` Matrice de risques. Critique unique = A-01 (cache FFT). Hautes = A-02/A-03 (globaux FFT non synchronisés), A-04 (dérive doc — ce fichier).

---

## ⚠ Modules sensibles (changements à risque)

Ces fichiers concentrent la complexité ou des couplages cachés. **Toute modification doit citer le constat `A-NN` correspondant de `audit.md`** dans le message de commit.

| Fichier | Risque |
|---------|--------|
| `internal/fibonacci/fastdoubling.go` | Hot path + pooling state+arena (R1.1 résolu, A-16/A-17). |
| `internal/fibonacci/doubling_framework.go` | Boucle critique couplée à `bigfft` (R3.2). |
| `internal/fibonacci/threshold/manager.go` | ~277 L ; invariant single-writer non documenté (A-18). |
| `internal/bigfft/fft_cache.go` | Globals + cache LRU (A-01 Critique, A-02, A-05). |
| `internal/bigfft/pool.go` | 13 pools globaux (R1.4 résolu). |
| `internal/bigfft/fermat.go` | Panics d'invariants ; `recover()` global masque les post-conditions (A-06). |
| `internal/bigfft/fft.go`, `fft_recursion.go` | Globaux mutables non synchronisés sur hot path (A-03). |
| `internal/tui/model.go` | ~184 L, routeur Update pur (déjà refactoré R3.4). |
| `internal/cli/completion/` | Registry unique, 4 générateurs shell ; échappement latent (A-19/sécurité). |
| `internal/fibonacci/testdata/fibonacci_golden.json` | **Immuable** sans accord explicite. |

---

## Commandes essentielles

```bash
make all             # clean + build + test  (équiv. sans make : go build ./... && go test ./...)
make test            # go test -v -race -cover ./...   (race nécessite CGO/gcc)
make test-short      # go test -v -short ./...
make coverage        # rapport couverture HTML
make benchmark       # go test -bench=. -benchmem ./internal/fibonacci/
make lint            # golangci-lint run ./...  (24 linters)
make build-pgo       # build avec PGO
make build-all       # cross-compilation (linux, windows, macOS)
```

> Sans `make` (ex. Windows sans GNU make) : utiliser les équivalents `go` ci-dessus. Le race detector exige un compilateur C (CGO) — indisponible sous Windows sans gcc ; la validation `-race` est assurée par la CI Linux/macOS.

---

## Conventions de code

- Packages par responsabilité (pas par feature).
- Interfaces étroites (ISP) : `Multiplier`, `DoublingStepExecutor`, `Calculator`, `ProgressReporter`.
- Erreurs structurées : `fmt.Errorf("%w", err)`. **Pas de panic** sauf pour invariants internes (cf. A-06 pour bigfft/fermat).
- Tests parallèles (`t.Parallel()`) systématiques (adoption élevée ; cible 100 %).
- Race detector en CI (Linux/macOS ; Windows à activer cf. A-14).
- Complexité cyclomatique max 15, cognitive max 30 (cf. `.golangci.yml`).
- Longueur fonction max 100 lignes / 50 statements.
- `doc.go` pour chaque package public.
- **Pas d'emoji** dans le code.
- **Commentaires uniquement quand le « pourquoi » n'est pas évident** (pas de description du « quoi »).

---

## Directives projet (période de remédiation post-audit)

> Les lignes directrices comportementales générales (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution) sont dans `~/.claude/CLAUDE.md` et s'appliquent ici.

1. **Performance critique** — Toute modification dans `internal/fibonacci/` ou `internal/bigfft/` doit être vérifiée avec `make benchmark` (ou `go test -bench=BenchmarkFibonacci -benchmem -run=^$ ./internal/fibonacci/`) avant + après. Régression > 5 % = blocage.

2. **Golden tests obligatoires** — Tout changement algorithmique doit passer `internal/fibonacci/testdata/fibonacci_golden.json`. Le fichier golden est **immuable** sans approbation explicite (aucun `-update`).

3. **Étanchéité des couches** — `internal/` ne doit pas fuiter vers `cmd/`. Hiérarchie : `cmd → app → orchestration → fibonacci/bigfft → config/errors`. (Vérifiée conforme en audit.)

4. **Concurrence contrôlée** — `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutines sans contrôle de cycle de vie. **Pas de nouveaux globals dans `bigfft/`** — A-03 va dans le sens inverse (convertir les globaux existants en atomiques en place, zéro ajout).

5. **Modifications chirurgicales** — Tout refactoring d'envergure (> 50 LOC sur > 2 fichiers) doit être tracé dans `AuditPlanning.md` (entrée `A-NN` ou nouvelle entrée).

6. **Tableau de suivi à maintenir** — À chaque transition de statut, mettre à jour le tableau de [`AuditPlanning.md`](AuditPlanning.md) avec le SHA du commit.

7. **Pas de nouveaux fichiers `progress*` sans consultation** — Couche progression : le chemin de production est `Freeze` (cf. A-19). Ajouter à un endroit existant après lecture.

8. **Pas de nouveaux globals dans `bigfft/`** — Direction inverse (trajectoire `FFTContext` injectable, R3.7 / A-03).

9. **Bug fix avant refactor** — Si un défaut actif de `audit.md` est touché par hasard, le corriger en priorité (commit isolé `fix(A-NN):`) avant le refactor planifié.

10. **CI active** — `make test -race && make lint` (ou équivalents `go`) en local avant chaque PR reste recommandé ; la CI (`ci.yml`) le rejoue sur 3 OS. Ne PAS recréer de workflow concurrent.

---

## Workflow recommandé pour une nouvelle modification

```
1. Identifier le constat dans AuditPlanning.md (ID A-NN ; sinon ajouter une entrée).
2. Marquer 🟡 InProgress dans le tableau de suivi.
3. Branche dédiée : git checkout -b fix/A-NN-description-courte
4. Test rouge → fix minimal → vert + golden + (benchmark si perf-sensitive).
5. go test ./<pkg>/... -count=1 && go vet ./... && golangci-lint run ./<pkg>/...
6. Comparer aux baselines docs/audits/.
7. Commit « fix(A-NN): description » (ou docs/ci/test/perf).
8. PR, review, merge (Vague A : gel pour revue humaine).
9. AuditPlanning.md : ✅ Done + SHA.
10. Commit doc : « docs(plan): A-NN marked Done <sha> ».
```

---

## Références

- [`audit.md`](audit.md) — audit exhaustif (23 constats, 10 sections).
- [`AuditPlanning.md`](AuditPlanning.md) — plan de remédiation + tableau de suivi.
- [`audit-prompt.md`](audit-prompt.md) — prompt d'audit réutilisable.
- [`docs/architecture/`](docs/architecture/) — diagrammes C4, dependency graph.
- [`docs/algorithms/`](docs/algorithms/) — Fast Doubling, Matrix, FFT, GMP, comparaison.
- [`CHANGELOG.md`](CHANGELOG.md) — Keep-a-Changelog format, SemVer.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — workflow contribution.
- `.golangci.yml` — 24 linters configurés, exceptions documentées.
