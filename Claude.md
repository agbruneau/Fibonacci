# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique démontrant Clean Architecture, pooling mémoire, parallélisme adaptatif et optimisation PGO.

> **État** — Base de code stabilisée. Les modules listés en « Modules sensibles » concentrent la complexité et les couplages cachés : toute modification non triviale dans `internal/fibonacci/` ou `internal/bigfft/` doit d'abord lire l'invariant documenté du fichier concerné ci-dessous.

---

## Projet

- **Module** : `github.com/agbruneau/FibGo`
- **Go** : 1.26.0+ (toolchain 1.26.3)
- **Licence** : Apache 2.0
- **Taille** : exécuter `make stats` pour le décompte de packages et LOC à jour. Référence historique : ~35 500 LOC `.go` au commit de l'audit v1.
- **CI/CD** : aucun (workflows GitHub Actions retirés). Validation locale uniquement — `make test` (race, requiert CGO/gcc), `make lint`, `make benchmark` avant chaque commit perf-sensitive.

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
                     #   échappement des identifiants à valider sur tout ajout
  config/            # Parsing config, flags, variables d'environnement
  errors/            # Types d'erreurs structurées (ConfigError, CalcError)
  fibonacci/         # CŒUR : Fast Doubling, Matrix Exp., FFT, Strassen, GMP
    fibonaccitest/   # Doubles de test pour CoreCalculator
    memory/          # Arena, GCController, budget mémoire
    threshold/       # Gestionnaire dynamique de seuils (FFT/parallèle/Strassen)
  format/            # Formatage durées, nombres, ETA
  metrics/           # Indicateurs de performance, monitoring mémoire
    system/          # Échantillonnage CPU/mém
  orchestration/     # Exécution concurrente (errgroup), agrégation
  parallel/          # ErrorCollector — utilisé par fibonacci/common.go
  progress/          # Pattern observer + DTO progression (chemin prod : Freeze)
  testutil/          # Helpers de test partagés
  tui/               # Dashboard TUI interactif (Bubble Tea)
    component/       # Composant TUI réutilisable
  ui/                # Thèmes couleur (source unique, tui/styles en dérive)
docs/
  architecture/      # Diagrammes C4 (Mermaid), validation
  algorithms/        # Documentation mathématique par algorithme
  audits/            # Baselines benchmark (référence de non-régression perf)
  dashboard/         # Build statique du knowledge-graph (GitHub Pages, généré)
.understand-anything/ # Knowledge-graph + fingerprints (généré, source du dashboard)
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
- **GC désactivé** pendant calculs N ≥ 1M — **panic-safe** via `gcCtrl.WithGC(fn)` (`defer End()`). `Begin`/`End` directs `Deprecated`. Contrôle GC concurrency-safe via refcount package-level (`gcGlobalMu`/`gcActiveDepth`/`gcSavedPercent`) : en mode comparaison (plusieurs `GCController` concurrents), seul le premier `Begin` actif capture le vrai GOGC d'origine et seul le dernier `End` le restaure (A2-01, ADR-0005).
- **Parallélisme adaptatif** via sémaphore (`NumCPU()`)
- **Cache FFT** LRU thread-safe — 15-30 % speedup **uniquement sur les chemins qui consultent le cache** : appels directs `bigfft.Mul/Sqr` et `FFTOnlyStrategy` (via `TransformCached*`/`MulCachedWithBump`/`SqrCachedWithBump`). Le calculateur **Fast Doubling par défaut** (`executeDoublingStepFFT`, `internal/fibonacci/fft.go`) transforme FK/FK1 via `TransformWithBump` et **ne consulte pas** le cache : le gain inter-itérations ne s'applique donc pas au mode par défaut (zéro hit/miss). Le recyclage de `backing` à l'éviction a été **retiré** (Audit-PRD E1-R4) : `putByKey` alloue toujours un buffer frais pour éliminer l'aliasing avec un `PolValues` issu d'un `Get()` concurrent. Net cost négligeable sur le hot path.
- **PGO** supporté via `make build-pgo`

---

## ⚠ Invariants à préserver (corrections subtiles déjà en place)

Les correctifs ci-dessous sont **en place et testés**. Ils encodent des invariants non évidents : une régression naïve les casserait sans faire échouer un test trivial. **Ne pas les réécrire sans comprendre l'invariant.**

| Fichier | Invariant à préserver |
|---------|-----------------------|
| `fibonacci/fastdoubling.go` | `finalizeStateRelease` appelle `clearStateAliases` **inconditionnellement** avant tout retour ; gardé par `TestReleaseState_OverLimit_AliasesCleared`. |
| `fibonacci/memory/gc_control.go` | `WithGC(fn)` est panic-safe (`defer End()`). `Begin`/`End` directs sont `Deprecated` — ne pas les réintroduire. Le save/restore GOGC est concurrency-safe via refcount package-level (`gcGlobalMu`/`gcActiveDepth`/`gcSavedPercent`) ; le champ `originalGCPercent` par contrôleur a été **supprimé** (un save/restore par contrôleur corromprait le GOGC d'origine en mode comparaison) — A2-01, ADR-0005. |
| `calibration/calibration.go` | `IsStale` doit rester invoqué ; la branche stale doit router vers `CompleteStrategy`. |
| `bigfft/pool.go` | `releaseWordSlice` route sur `cap` (pas `len`) et incrémente un compteur de miss. `releaseFFTState` relâche les buffers `tmp`/`tmp2` dont `cap` dépasse `maxPooledFFTTmpCap` (anti-bloat, A2-05). SA6002 (Put de slice valeur) est une **décision documentée alloc-neutre** (mesurée) avec exclusion golangci ciblée `pool.go`/`pool_warming.go` ; le vrai fix = migration `FFTContext` — ADR-0007 (cf. ADR-0004 §B1). |
| `bigfft/fft_cache.go` | `putByKey` alloue **toujours** un backing frais ; ne pas réintroduire de recyclage à l'éviction — cf. Audit-PRD E1-R4 (aliasing avec un `PolValues` vivant). `TransformCache.logger` est un `atomic.Pointer[zerolog.Logger]` : `SetCacheLogger` ne doit pas racer avec la lecture hot-path de `logPeriodicStats` (A2-02). |
| `bigfft/fft.go`, `fft_recursion.go` | `fftThreshold`/`parallelFFTRecursionThreshold`/`maxParallelFFTDepth` sont **`atomic.Int64/Uint64` privés** ; lectures via `getFFTThreshold()`/`GetParallelFFTRecursionThreshold()`/`GetMaxParallelFFTDepth()`. Ne pas réintroduire de globaux mutables non synchronisés. |
| `bigfft/fft.go` (`Mul`/`MulTo`/`Sqr`/`SqrTo`) | Le `recover()` re-propage les sentinels `isFermatPostConditionPanic` ; les panics post-condition de `fermat.go` ne doivent pas être masquées en `error`. Gardé par `TestFermatPostConditionPanicClassifier`. |
| `fibonacci/threshold/manager.go` | Champs `currentFFTThreshold`/`currentParallelThreshold`/`iterationCount` en `atomic.Int64`, `lastAdjustment` en `atomic.Pointer[time.Time]`. L'invariant A-18 single-writer est **obsolète**. Le package n'importe **pas** `internal/config` ; passer par `threshold.SetTuning` depuis la couche supérieure. |
| `errors/errors.go` | N'importe **pas** `internal/format` ; un helper local `formatBytesLocal` couvre le besoin. Gardé par `TestArchitectureLayering`. |
| `tui/` (production) | N'importe **pas** `internal/fibonacci` directement ; passer par les aliases `orchestration.Calculator`/`Options`/`Default*Threshold`. Gardé par `TestArchitectureLayering`. |

---

## ⚠ Modules sensibles (changements à risque)

Ces fichiers concentrent la complexité ou des couplages cachés. Avant toute modification, lire l'invariant ci-dessous et la section « Invariants à préserver ».

| Fichier | Pourquoi sensible |
|---------|-------------------|
| `internal/fibonacci/fastdoubling.go` | Hot path ; pooling state+arena partagé. Tout chemin de release doit détacher les aliases avant `statePool.Put`. |
| `internal/fibonacci/doubling_framework.go` | Boucle critique étroitement couplée à `bigfft` ; toute régression perf y est amplifiée. |
| `internal/fibonacci/threshold/manager.go` | ~283 L ; invariant single-writer non explicité dans le code — ne pas introduire d'écrivain concurrent. |
| `internal/bigfft/fft_cache.go` | Globaux + cache LRU. Le risque d'aliasing backing/`pv` en éviction est **fermé** : `putByKey` alloue toujours un buffer frais (cf. invariants ci-dessus + ADR-0004 §B1). `logger` est en `atomic.Pointer[zerolog.Logger]` (A2-02) — ne pas le remettre en champ nu. |
| `internal/bigfft/pool.go` | Pools globaux par classe de taille (`wordSlicePools`, `fermatPools`, `natSlicePools`, `fermatSlicePools`) + `fftStatePool` ; routage par capacité critique. `releaseFFTState` borne les buffers réutilisés à `maxPooledFFTTmpCap` (A2-05). SA6002 = décision assumée alloc-neutre, exclusion golangci ciblée (ADR-0007). |
| `internal/bigfft/fermat.go` | Panics d'invariants. Récepteur uniformisé sur `z`. Les panics post-condition de `Mul`/`Sqr` doivent propager ; le `recover()` global de `fft.go` re-route via le classifier sentinel — cf. ADR-0002. |
| `internal/bigfft/fft.go`, `fft_recursion.go` | Globaux ramenés à `atomic.Int64`/`atomic.Uint64` privés (ADR-0003). Lectures hot path via les accesseurs. Ne pas réintroduire de globaux non synchronisés. |
| `internal/tui/model.go` | ~188 L, routeur `Update` pur ; garder la pureté (pas d'effets de bord dans le routage). |
| `internal/cli/completion/` | Registry unique, 4 générateurs shell ; échappement des identifiants vers le shell — risque de sécurité latent. |
| `internal/fibonacci/testdata/fibonacci_golden.json` | **Immuable** sans accord explicite ADR (oracle de non-régression algorithmique). Étendu à F(50k/100k/200k) en mai 2026 sous accord ADR-0004 §B5 ; toute extension future requiert le même protocole. |

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

> Sans `make` (ex. Windows sans GNU make) : utiliser les équivalents `go` ci-dessus. Le race detector exige un compilateur C (CGO) — indisponible sous Windows sans gcc ; sous Windows la validation `-race` se fait via WSL ou un autre poste Linux/macOS.

---

## Conventions de code

- Packages par responsabilité (pas par feature).
- Interfaces étroites (ISP) : `Multiplier`, `DoublingStepExecutor`, `Calculator`, `ProgressReporter`.
- Erreurs structurées : `fmt.Errorf("%w", err)`. **Pas de panic** sauf pour invariants internes (cas assumé : `bigfft/fermat.go`).
- Tests parallèles (`t.Parallel()`) systématiques (adoption élevée ; cible 100 %).
- Race detector recommandé en local (`CGO_ENABLED=1`, requiert gcc/clang).
- Complexité cyclomatique max 15, cognitive max 30 (cf. `.golangci.yml`).
- Longueur fonction max 100 lignes / 50 statements.
- `doc.go` pour chaque package public.
- **Pas d'emoji** dans le code.
- **Commentaires uniquement quand le « pourquoi » n'est pas évident** (pas de description du « quoi »).

---

## Directives projet

> Les lignes directrices comportementales générales (Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution) sont dans `~/.claude/CLAUDE.md` et s'appliquent ici.

1. **Performance critique** — Toute modification dans `internal/fibonacci/` ou `internal/bigfft/` doit être vérifiée avec `make benchmark` (ou `go test -bench=BenchmarkFibonacci -benchmem -run=^$ ./internal/fibonacci/`) avant + après. Régression > 5 % = blocage. Comparer aux baselines `docs/audits/`.

2. **Golden tests obligatoires** — Tout changement algorithmique doit passer `internal/fibonacci/testdata/fibonacci_golden.json`. Le fichier golden est **immuable** sans approbation ADR explicite (aucun `-update`). Le corpus a été étendu à F(50k/100k/200k) sous ADR-0004 §B5.

3. **Étanchéité des couches** — `internal/` ne doit pas fuiter vers `cmd/`. Hiérarchie : `cmd → app → orchestration → fibonacci/bigfft → config/errors`. Trois arrows remontants spécifiquement gardés par `internal/arch_test.go` : `threshold → config`, `errors → format`, `tui → fibonacci`.

4. **Concurrence contrôlée** — `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutines sans contrôle de cycle de vie. **Pas de nouveaux globals dans `bigfft/`** : les trois existants (`fftThreshold`, `parallelFFTRecursionThreshold`, `maxParallelFFTDepth`) sont en `atomic.*` privés avec accesseurs (ADR-0003). Une migration future vers `FFTContext` exclusif est tracée en backlog (ADR-0004 §B1, won't-fix pour la release courante).

5. **Modifications chirurgicales** — Préférer le diff minimal. Un refactoring d'envergure (> 50 LOC sur > 2 fichiers) se justifie dans le message de commit (raison technique, compromis, alternative écartée).

6. **Pas de nouveaux fichiers `progress*` sans consultation** — Couche progression : le chemin de production est `Freeze`. Étendre un point existant après lecture du package `internal/progress`.

7. **Bug fix avant refactor** — Si un défaut actif est touché par hasard pendant un refactor, le corriger en priorité dans un commit isolé (`fix(scope):`) avant de poursuivre le refactor.

8. **Validation locale** — `make test` (race) `&& make lint` (ou équivalents `go`) avant chaque commit/PR. Aucun garde-fou CI distant : la rigueur tient à la discipline locale du contributeur.

---

## Workflow recommandé pour une nouvelle modification

```
1. Branche dédiée : git checkout -b <type>/<description-courte>
2. Test rouge → fix minimal → vert + golden + (benchmark si perf-sensitive).
3. go test ./<pkg>/... -count=1 && go vet ./... && golangci-lint run ./<pkg>/...
4. Si perf-sensitive : comparer aux baselines docs/audits/.
5. Commit conventionnel « <type>(scope): description » (feat/fix/docs/perf/test/refactor).
6. PR, review, merge.
```

---

## Artefacts générés (NE PAS éditer à la main)

Ces fichiers sont produits par des outils ; les modifier directement sera écrasé au prochain build. Utiliser la commande de régénération indiquée.

| Chemin | Outil de régénération | Description |
|---|---|---|
| `.understand-anything/knowledge-graph.json` | `/understand` (plugin `understand-anything`) | Graphe : 744 nœuds, 3 526 arêtes, 8 couches, tour guidé 13 étapes. |
| `.understand-anything/fingerprints.json` | Idem (Phase 7 de `/understand`) | Baseline structurelle pour mises à jour incrémentales. |
| `.understand-anything/meta.json` | Idem | Commit hash + horodatage de la dernière analyse. |
| `docs/dashboard/` | `pnpm --filter @understand-anything/dashboard build:demo` puis recopie (voir [`docs/BUILD.md`](docs/BUILD.md#dashboard-statique-github-pages)) | Build React/Vite statique, déployé sur <https://agbruneau.github.io/FibGo/dashboard/>. |

Cycle typique : modifier le code → `/understand` (régénère le graphe) → rebuild dashboard → commit `docs/dashboard/` + `.understand-anything/`.

---

## Références

- **[Dashboard interactif](https://agbruneau.github.io/FibGo/dashboard/)** — knowledge-graph navigable (GitHub Pages, built from `docs/dashboard/`).
- [`docs/adr/`](docs/adr/) — décisions architecturales (0001 DTM, 0002 recover, 0003 globaux atomic, 0004 backlog, 0005 contrôle GC concurrent par refcount, 0006 annulation récursion FFT reportée au token par-appel/FFTContext, 0007 pool SA6002 pointeur vs valeur).
- [`docs/architecture/`](docs/architecture/) — diagrammes C4, dependency graph.
- [`docs/algorithms/`](docs/algorithms/) — Fast Doubling, Matrix, FFT, GMP, comparaison.
- [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md) — tuning et méthodologie de benchmark.
- [`docs/TESTING.md`](docs/TESTING.md) — stratégie de test, génération de mocks.
- [`docs/PORTABILITY.md`](docs/PORTABILITY.md) — matrice OS/arch, fallbacks, race detector.
- [`docs/BUILD.md`](docs/BUILD.md) — cross-compilation, PGO, signing, Docker/devcontainer.
- [`CHANGELOG.md`](CHANGELOG.md) — Keep-a-Changelog format, SemVer.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — workflow contribution.
- `.golangci.yml` — 24 linters configurés (incl. `govet shadow`), exceptions documentées.
