# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique : Clean Architecture, pooling mémoire, parallélisme adaptatif, optimisation PGO.

> **Règle d'or** — Base de code **stabilisée**. Avant toute modification non triviale dans `internal/fibonacci/` ou `internal/bigfft/`, lire **d'abord** l'entrée correspondante de la section [Invariants à préserver](#invariants-à-préserver--modules-sensibles). Une régression naïve y casse un invariant **sans faire échouer un test trivial**.

---

## Projet

| | |
|---|---|
| **Module** | `github.com/agbruneau/FibGo` |
| **Go** | `go.mod` requiert `go 1.26.0` (toolchain non épinglée) |
| **Licence** | Apache 2.0 |
| **CI/CD** | Aucune (`.github/workflows/` absent). Validation **locale** uniquement, gardée par `scripts/check.{sh,ps1}` — la rigueur tient à la discipline locale |
| **Chiffres** | `make stats` est la **source canonique** (packages, LOC) — ne jamais coder en dur de décompte daté |
| **Audit** | Dernier audit exhaustif : 2026-07 — rapport [`audit.md`](audit.md), plan d'exécution [`auditPlan.md`](auditPlan.md) |

---

## Architecture

Clean Architecture, 4 couches : `cmd → app → orchestration → fibonacci/bigfft → config/errors`.

- **Structure des packages** : source de vérité dans [`docs/architecture/`](docs/architecture/) (diagrammes C4, [`dependency-graph.mermaid`](docs/architecture/dependency-graph.mermaid)) et la table des packages du [README](README.md#architecture). Ne pas redupliquer l'arbre ici — il dérive.
- **Étanchéité** (`internal/arch_test.go` / `TestArchitectureLayering`) : trois arrows remontants interdits — `threshold → config`, `errors → format`, `tui → fibonacci`. Le non-débordement de `internal/` vers `cmd/` est garanti par le langage, pas par ce test.

## Algorithmes

1. **Fast Doubling** (défaut) — O(log n), identité F(2k) = F(k)(2F(k+1) − F(k)).
2. **Matrix Exponentiation** — O(log n), Strassen-Winograd pour grandes matrices.
3. **FFT (Schönhage-Strassen)** — seuil adaptatif (~500k bits par défaut).
4. **GMP** (build tag `gmp`) — backend GNU MP (CGO + libgmp requis).

Détails mathématiques : [`docs/algorithms/`](docs/algorithms/).

---

## Invariants à préserver & modules sensibles

Ces fichiers concentrent la complexité et des couplages cachés ; chacun porte une correction subtile **déjà en place et testée**. Chaque entrée nomme son **test gardien**.

### `fibonacci/fastdoubling.go` — hot path

- `finalizeStateReleaseTo` est le **chemin unique** de teardown (`finalizeStateRelease` = wrapper vers le sink pool).
- `clearStateAliases` **inconditionnel** avant **tout** sink : `statePool.Put` **ou** slot GC-immune `cachedState` (borné à `maxCachedArenaWords` = 4M mots ≈ 32 Mo).
- État overLimit **jamais publié** ; bump relâché sur drop anti-bloat et overLimit. Jamais publier un état dans le slot hors de ce chemin.
- **Gardiens** : `TestReleaseState_OverLimit_AliasesCleared`, `TestCalculatorStateCache_OverLimitNotCached`, `TestStateBump_*`.

### `fibonacci/doubling_framework.go`

- Boucle critique étroitement couplée à `bigfft` ; toute régression perf y est **amplifiée**. Gate benchstat impératif (Directive #1).

### `fibonacci/threshold/manager.go`

- Champs **par-instance** (`currentFFTThreshold`/`currentParallelThreshold`/`iterationCount` en `atomic.Int64`, `lastAdjustment` en `atomic.Pointer[time.Time]`) : l'ancien invariant single-writer par-instance (A-18) est **obsolète** — migrés atomic.
- **A2-04 toujours actif** : single-writer-before-use sur les knobs **package-level** (`FFTSpeedupThreshold`, etc. — documenté en tête `manager.go:33-39`). Pas d'écrivain concurrent sur `ShouldAdjust`/`Reset`.
- Le package n'importe **pas** `internal/config` : câblage `config.DefaultThresholdTuning → threshold.SetTuning` exécuté une fois par `app.New` (`wireThresholdTuning`, `sync.Once`). **Gardien** : `TestWireThresholdTuning`.
- `MetricsBuffer` n'est **pas** goroutine-safe : tout accès (`Record`/`Count`/`RecentMetrics`) sous `mu`. **Gardien** : `TestConcurrentAccess` sous `-race`.

### `fibonacci/memory/gc_control.go`

- `WithGC(fn)` est panic-safe (`defer End()`). `Begin`/`End` directs sont **`Deprecated`** — ne pas les réintroduire.
- Save/restore GOGC concurrency-safe via refcount package-level (`gcGlobalMu`/`gcActiveDepth`/`gcSavedPercent`). Le champ `originalGCPercent` par contrôleur a été **supprimé** (sinon corruption du GOGC d'origine en mode comparaison) — A2-01, ADR-0005.

### `calibration/calibration.go`

- `IsStale` doit rester invoqué ; la branche stale route vers `CompleteStrategy`.

### `bigfft/pool.go`

- Pools globaux par classe de taille (`wordSlicePools`, `fermatPools`, `natSlicePools`, `fermatSlicePools`) + `fftStatePool`. **Routage par capacité** critique : `releaseWordSlice` route sur `cap` (**pas** `len`) + compteur de miss ; `releaseFFTState` relâche les buffers `tmp`/`tmp2` dont `cap` dépasse `maxPooledFFTTmpCap` (anti-bloat, A2-05).
- SA6002 (Put de slice valeur) = **décision alloc-neutre mesurée**, exclusion golangci ciblée `pool.go`/`pool_warming.go` ; le vrai fix = migration `FFTContext` — ADR-0007 (cf. ADR-0004 §B1).
- `TestReleaseWordSliceResizedReturnsToBucket` valide le routage `cap`→bucket via le **compteur de miss uniquement** : ne **pas** y réintroduire d'assertion d'identité `sync.Pool` (canary / réapparition du backing) — non contractuelle et **flaky sous `-race`** (victim cache + timing GC), retirée 2026-06-21. **Gardien** du routage exhaustif : `TestReleaseWordSliceAllExactBuckets`.

### `bigfft/fft_cache.go`

- Globaux + cache LRU. `putByKey` alloue **toujours** un backing frais — pas de recyclage à l'éviction (aliasing avec un `PolValues` vivant, Audit-PRD E1-R4).
- `logger` = `atomic.Pointer[zerolog.Logger]` (ne pas remettre en champ nu) : `SetCacheLogger` ne doit pas racer avec la lecture hot-path de `logPeriodicStats` (A2-02).

### `bigfft/fft.go`, `fft_recursion.go`

- `fftThreshold`/`parallelFFTRecursionThreshold`/`maxParallelFFTDepth` sont des **`atomic.Int64`/`atomic.Uint64` privés** ; lectures via `getFFTThreshold()`/`GetParallelFFTRecursionThreshold()`/`GetMaxParallelFFTDepth()`. Ne pas réintroduire de globaux mutables non synchronisés (ADR-0003).

### `bigfft/fft.go` (`Mul`/`MulTo`/`Sqr`/`SqrTo`), `fermat.go`

- Récepteur `fermat` uniformisé sur `z`. Le `recover()` global re-propage les sentinels `isFermatPostConditionPanic` ; les panics post-condition de `fermat.go` ne doivent **pas** être masquées en `error` (re-route via classifier sentinel — ADR-0002). **Gardien** : `TestFermatPostConditionPanicClassifier`.
- **Tous** les chemins parallèles capturent les panics worker via un `panicCh` bufferisé et les re-panic sur l'appelant après `wg.Wait()` — un panic sur goroutine nue crasherait le process en contournant ADR-0002. **Gardiens** : récursion async `fourierRecursiveUnified`/`fourierRecursiveCtx` (`TestFourierRecursiveAsyncPanicPropagates`/`...CtxAsyncPanicPropagates`), `executeReconstruction` (`TestExecuteReconstructionPanicPropagates`), `runPointwise` (`TestPointwiseWorkerPanicPropagates`).

### `errors/errors.go`

- N'importe **pas** `internal/format` ; le helper local `formatBytesLocal` couvre le besoin. **Gardien** : `TestArchitectureLayering`.

### `tui/` (production), `tui/model.go`

- N'importe **pas** `internal/fibonacci` directement → passer par les aliases `orchestration.Calculator`/`Options`/`Default*Threshold`. **Gardien** : `TestArchitectureLayering`.
- `model.go` : routeur `Update` **pur** — aucun effet de bord dans le routage.

### `cli/completion/`

- Registry unique, 4 générateurs shell. Échappement des identifiants vers le shell **par dialecte** (`escape.go`) — **risque sécu latent**, vecteur d'injection à garder fermé.

### `fibonacci/testdata/fibonacci_golden.json`

- **Immuable** sans accord ADR explicite (oracle de non-régression algorithmique ; aucun `-update`). Étendu à F(50k/100k/200k) sous ADR-0004 §B5 ; toute extension future requiert le même protocole.

---

## Patterns de performance critiques

Contexte « pourquoi » du hot path (détail mesuré : [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md)). Les **contrats** à ne pas casser sont dans la section précédente.

- **`sync.Pool` pour `big.Int`** + état/arène/bump unifié dans `CalculationState`, réutilisé inter-appels via le slot GC-immune par calculateur (le GC forcé post-calcul purge les `sync.Pool`).
- **Allocateur bump** O(1), zéro fragmentation, mono-goroutine, pour les tampons FFT.
- **GC désactivé** pendant les grands calculs (N ≥ 1M) via `gcCtrl.WithGC(fn)` (panic-safe).
- **Parallélisme adaptatif** : produits pointwise et butterflies FFT répartis sur les cœurs via un sémaphore FFT global à **acquisition non bloquante** (pas de jeton ⇒ exécution sur la goroutine appelante : aucun interblocage avec la récursion). Panics de worker re-propagées.
- **Cache LRU de transformées FFT** : bénéficie **uniquement** aux chemins qui le consultent (`bigfft.Mul/Sqr/MulTo/SqrTo` directs ; `FFTOnlyStrategy.Multiply/Square` les délèguent). **Aucune boucle de doublement ne le consulte** : le Fast Doubling par défaut *et* le calculateur FFT-only passent par `executeDoublingStepFFT` (`TransformWithBump`, non caché) — zéro hit, mesuré.
- **PGO** via `make build-pgo`.

> **Gate perf-sensitive** : toute modif `fibonacci/`|`bigfft/` se compare via `benchstat` à la baseline (`make bench-baseline` → `docs/audits/bench-baseline.txt`) — seuil et procédure en **Directive #1**.

---

## Commandes essentielles

`make help` liste tout. Les plus utilisées :

```bash
make all            # clean + build + test
make test           # go test -v -race -cover ./...   (race : CGO/gcc requis — Linux/macOS/WSL)
make test-win       # go test -v -cover ./...          (Windows sans gcc : pas de -race)
make lint           # golangci-lint run ./...          (24 linters, advisory)
make coverage-check # plancher de couverture 80 %
make benchmark      # go test -bench=. -benchmem ./internal/fibonacci/
make bench-baseline # régénère docs/audits/bench-baseline.txt
make stats          # décompte packages / LOC (canonique)
make build-pgo      # build avec PGO
```

Points d'attention :

- **Gate de pré-commit** : `scripts/check.{sh,ps1}` — build → vet → test → lint (**advisory**) → couverture ≥ 80 %. Le gate **dur** est build/vet/test/couverture.
- **`-race`** : exige CGO/gcc, indisponible sous Windows sans gcc ⇒ passes complètes via **WSL** (`wsl go test -race ./...`). `libgmp-dev` absent de WSL ⇒ tag `gmp` non testable là.
- **PowerShell** : `go test -bench=.` est mal parsé — préfixer le pattern (`-bench=BenchmarkFibonacci`).

---

## Conventions de code

- Packages par responsabilité (pas par feature). Interfaces étroites (ISP) : `Multiplier`, `DoublingStepExecutor`, `Calculator`, `ProgressReporter`.
- Erreurs structurées : `fmt.Errorf("%w", err)`. **Pas de panic** sauf invariants internes (cas assumé : `bigfft/fermat.go`).
- `t.Parallel()` systématique (cible 100 %). `doc.go` pour chaque package public.
- Complexité cyclomatique max 15, cognitive max 30 ; fonction max 100 lignes / 50 statements (`.golangci.yml`).
- **Pas d'emoji** dans le code. Commentaires **uniquement quand le « pourquoi » n'est pas évident**.

---

## Directives projet

> Les directives comportementales générales (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution) sont dans `~/.claude/CLAUDE.md`.

1. **Performance critique** — Modif dans `internal/fibonacci/` ou `internal/bigfft/` : benchmark `benchstat` avant + après (baseline `make bench-baseline`). **Régression > 5 % = blocage.**
2. **Golden tests obligatoires** — Tout changement algorithmique passe `fibonacci_golden.json`. Fichier **immuable** sans approbation ADR (aucun `-update`).
3. **Étanchéité des couches** — Hiérarchie `cmd → app → orchestration → fibonacci/bigfft → config/errors`. Trois arrows remontants gardés par `arch_test.go` : `threshold → config`, `errors → format`, `tui → fibonacci`.
4. **Concurrence contrôlée** — `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutine sans contrôle de cycle de vie. **Pas de nouveaux globals dans `bigfft/`** : les trois existants sont en `atomic.*` privés avec accesseurs (ADR-0003). Migration `FFTContext` exclusive tracée en backlog (ADR-0004 §B1, won't-fix release courante).
5. **Modifications chirurgicales** — Diff minimal. Un refactoring d'envergure (> 50 LOC sur > 2 fichiers) se justifie dans le message de commit (raison, compromis, alternative écartée).
6. **Pas de nouveaux fichiers `progress*` sans consultation** — Chemin de production : `Freeze`. Étendre un point existant après lecture de `internal/progress`.
7. **Bug fix avant refactor** — Un défaut actif touché par hasard pendant un refactor se corrige d'abord dans un commit isolé (`fix(scope):`).
8. **Validation locale** — Avant chaque commit : `scripts/check.sh` / `scripts/check.ps1`, ou à défaut `make test` (ou `make test-win` sans gcc) `&& make lint`. Aucun garde-fou CI distant.

---

## Workflow

Pratique effective : **trunk-based** (mainteneur solo). Une branche dédiée + PR reste pertinente pour les travaux d'envergure ou expérimentaux.

```
1. Test rouge → fix minimal → vert + golden + (benchmark si perf-sensitive).
2. go test ./<pkg>/... -count=1 && go vet ./... && golangci-lint run ./<pkg>/...
3. Si perf-sensitive : comparer via benchstat (baseline make bench-baseline).
4. Gate complet : scripts/check.{sh,ps1}.
5. Commit conventionnel « <type>(scope): description » (feat/fix/docs/perf/test/refactor) → push main.
```

---

## Artefacts générés (NE PAS éditer à la main)

| Chemin | Régénération | Description |
|---|---|---|
| `docs/dashboard/` | Procédure complète : [`docs/BUILD.md`](docs/BUILD.md#dashboard-statique-github-pages) | Build React/Vite statique (knowledge graph), déployé sur GitHub Pages. |

---

## Références

- [`audit.md`](audit.md) / [`auditPlan.md`](auditPlan.md) — audit exhaustif 2026-07 et plan d'exécution des correctifs.
- [`docs/adr/`](docs/adr/) — décisions : 0001 DTM, 0002 recover, 0003 globaux atomic, 0004 backlog, 0005 contrôle GC concurrent par refcount, 0006 annulation récursion FFT, 0007 pool SA6002 pointeur vs valeur, 0008 candidats rejetés audit 2026-06.
- [`docs/architecture/`](docs/architecture/) — diagrammes C4, dependency graph. [Dashboard interactif](https://agbruneau.github.io/FibGo/dashboard/).
- [`docs/algorithms/`](docs/algorithms/), [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md), [`docs/TESTING.md`](docs/TESTING.md), [`docs/PORTABILITY.md`](docs/PORTABILITY.md), [`docs/BUILD.md`](docs/BUILD.md), [`docs/CALIBRATION.md`](docs/CALIBRATION.md).
- [`CHANGELOG.md`](CHANGELOG.md), [`CONTRIBUTING.md`](CONTRIBUTING.md), `.golangci.yml` (24 linters, `govet shadow`, exceptions documentées).
