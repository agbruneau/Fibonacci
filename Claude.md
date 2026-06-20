# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique : Clean Architecture, pooling mémoire, parallélisme adaptatif, optimisation PGO.

> **Règle d'or** — Base de code stabilisée. Toute modification non triviale dans `internal/fibonacci/` ou `internal/bigfft/` doit d'abord lire l'invariant documenté du fichier concerné (section « Invariants à préserver »). Une régression naïve y casse un invariant sans faire échouer un test trivial.

---

## Projet

- **Module** : `github.com/agbruneau/FibGo`
- **Go** : `go.mod` requiert `go 1.26.0` (toolchain non épinglée).
- **Licence** : Apache 2.0
- **CI/CD** : aucune (`.github/workflows/` absent). Validation locale uniquement, gardée par `scripts/check.{sh,ps1}`.
- **Taille / décompte packages + LOC** : `make stats` (source canonique ; ne pas coder en dur de chiffres datés).

---

## Architecture

Clean Architecture, 4 couches : `cmd → app → orchestration → fibonacci/bigfft → config/errors`.

- **Structure des packages** : source de vérité dans [`docs/architecture/`](docs/architecture/) (diagrammes C4, [`dependency-graph.mermaid`](docs/architecture/dependency-graph.mermaid)) et la table des packages du [README](README.md#architecture). Ne pas redupliquer l'arbre ici (il dérive).
- **Étanchéité** : `internal/arch_test.go` (`TestArchitectureLayering`) interdit trois arrows remontants — `threshold → config`, `errors → format`, `tui → fibonacci`. `internal/` ne doit pas fuiter vers `cmd/`.

---

## Algorithmes

1. **Fast Doubling** (défaut) — O(log n), identité F(2k) = F(k)(2F(k+1) − F(k)).
2. **Matrix Exponentiation** — O(log n), Strassen-Winograd pour grandes matrices.
3. **FFT (Schönhage-Strassen)** — seuil adaptatif (~500k bits par défaut).
4. **GMP** (build tag `gmp`) — backend GNU MP (CGO + libgmp requis).

Détails math : [`docs/algorithms/`](docs/algorithms/).

---

## Invariants à préserver (corrections subtiles déjà en place et testées)

Ne pas réécrire sans comprendre l'invariant. Chaque entrée nomme son test gardien.

| Fichier | Invariant à préserver |
|---------|-----------------------|
| `fibonacci/fastdoubling.go` | `finalizeStateReleaseTo` est le **chemin unique** de teardown (`finalizeStateRelease` = simple wrapper vers le sink pool) : `clearStateAliases` **inconditionnel** avant tout sink (`statePool.Put` **ou** slot `cachedState`), état overLimit jamais publié, bump relâché sur drop anti-bloat et overLimit. Ne jamais publier un état dans le slot sans passer par ce chemin. Gardé par `TestReleaseState_OverLimit_AliasesCleared`, `TestCalculatorStateCache_OverLimitNotCached`, `TestStateBump_*`. |
| `fibonacci/memory/gc_control.go` | `WithGC(fn)` est panic-safe (`defer End()`). `Begin`/`End` directs sont `Deprecated` — ne pas les réintroduire. Le save/restore GOGC est concurrency-safe via refcount package-level (`gcGlobalMu`/`gcActiveDepth`/`gcSavedPercent`) ; le champ `originalGCPercent` par contrôleur a été **supprimé** (un save/restore par contrôleur corromprait le GOGC d'origine en mode comparaison) — A2-01, ADR-0005. |
| `calibration/calibration.go` | `IsStale` doit rester invoqué ; la branche stale doit router vers `CompleteStrategy`. |
| `bigfft/pool.go` | `releaseWordSlice` route sur `cap` (pas `len`) et incrémente un compteur de miss. `releaseFFTState` relâche les buffers `tmp`/`tmp2` dont `cap` dépasse `maxPooledFFTTmpCap` (anti-bloat, A2-05). SA6002 (Put de slice valeur) est une **décision documentée alloc-neutre** (mesurée) avec exclusion golangci ciblée `pool.go`/`pool_warming.go` ; le vrai fix = migration `FFTContext` — ADR-0007 (cf. ADR-0004 §B1). |
| `bigfft/fft_cache.go` | `putByKey` alloue **toujours** un backing frais ; ne pas réintroduire de recyclage à l'éviction — cf. Audit-PRD E1-R4 (aliasing avec un `PolValues` vivant). `TransformCache.logger` est un `atomic.Pointer[zerolog.Logger]` : `SetCacheLogger` ne doit pas racer avec la lecture hot-path de `logPeriodicStats` (A2-02). |
| `bigfft/fft.go`, `fft_recursion.go` | `fftThreshold`/`parallelFFTRecursionThreshold`/`maxParallelFFTDepth` sont **`atomic.Int64`/`atomic.Uint64` privés** ; lectures via `getFFTThreshold()`/`GetParallelFFTRecursionThreshold()`/`GetMaxParallelFFTDepth()`. Ne pas réintroduire de globaux mutables non synchronisés. |
| `bigfft/fft.go` (`Mul`/`MulTo`/`Sqr`/`SqrTo`) | Le `recover()` re-propage les sentinels `isFermatPostConditionPanic` ; les panics post-condition de `fermat.go` ne doivent pas être masquées en `error`. Gardé par `TestFermatPostConditionPanicClassifier`. |
| `fibonacci/threshold/manager.go` | Champs **par-instance** `currentFFTThreshold`/`currentParallelThreshold`/`iterationCount` en `atomic.Int64`, `lastAdjustment` en `atomic.Pointer[time.Time]` : l'invariant A-18 single-writer **sur ces champs par-instance** est donc **obsolète** (migrés en atomic). Distinct de l'invariant A2-04 single-writer-before-use, lui **toujours actif**, qui porte sur les **knobs de tuning package-level** (`FFTSpeedupThreshold`, etc. ; cf. Modules sensibles + `manager.go:33-39`). Le package n'importe **pas** `internal/config` ; le câblage `config.DefaultThresholdTuning → threshold.SetTuning` est exécuté une fois par `app.New` (`wireThresholdTuning`, `sync.Once`) — gardé par `TestWireThresholdTuning`. `MetricsBuffer` n'est **pas** goroutine-safe par contrat : tout accès (`Record`/`Count`/`RecentMetrics`) passe sous `mu` — gardé par `TestConcurrentAccess` sous `-race`. |
| `errors/errors.go` | N'importe **pas** `internal/format` ; un helper local `formatBytesLocal` couvre le besoin. Gardé par `TestArchitectureLayering`. |
| `tui/` (production) | N'importe **pas** `internal/fibonacci` directement ; passer par les aliases `orchestration.Calculator`/`Options`/`Default*Threshold`. Gardé par `TestArchitectureLayering`. |

---

## Modules sensibles (changements à risque)

Ces fichiers concentrent la complexité ou des couplages cachés. Avant toute modification, lire l'invariant ci-dessus.

| Fichier | Pourquoi sensible |
|---------|-------------------|
| `internal/fibonacci/fastdoubling.go` | Hot path ; pooling state+arena+bump partagé et slot GC-immune par calculateur (`cachedState`, borné par `maxCachedArenaWords` = 4M mots ≈ 32 Mo). Tout chemin de release doit détacher les aliases avant **tout sink** via `finalizeStateReleaseTo`. |
| `internal/fibonacci/doubling_framework.go` | Boucle critique étroitement couplée à `bigfft` ; toute régression perf y est amplifiée. |
| `internal/fibonacci/threshold/manager.go` | Invariant A2-04 single-writer-before-use des tuning knobs **documenté en tête de fichier** (`manager.go:33-39`) — ne pas introduire d'écrivain concurrent sur `ShouldAdjust`/`Reset`. Les accès `MetricsBuffer` doivent rester sous `mu` (cf. Invariants). |
| `internal/bigfft/fft_cache.go` | Globaux + cache LRU. Risque d'aliasing backing/`pv` en éviction **fermé** : `putByKey` alloue toujours un buffer frais (ADR-0004 §B1). `logger` en `atomic.Pointer[zerolog.Logger]` (A2-02) — ne pas le remettre en champ nu. |
| `internal/bigfft/pool.go` | Pools globaux par classe de taille (`wordSlicePools`, `fermatPools`, `natSlicePools`, `fermatSlicePools`) + `fftStatePool` ; routage par capacité critique. `releaseFFTState` borne les buffers réutilisés à `maxPooledFFTTmpCap` (A2-05). SA6002 = décision assumée alloc-neutre, exclusion golangci ciblée (ADR-0007). |
| `internal/bigfft/fermat.go` | Panics d'invariants. Récepteur uniformisé sur `z`. Les panics post-condition de `Mul`/`Sqr` doivent propager ; le `recover()` global de `fft.go` re-route via le classifier sentinel — ADR-0002. |
| `internal/bigfft/fft.go`, `fft_recursion.go` | Globaux ramenés à `atomic.Int64`/`atomic.Uint64` privés (ADR-0003). Lectures hot path via les accesseurs. Ne pas réintroduire de globaux non synchronisés. |
| `internal/tui/model.go` | Routeur `Update` pur ; garder la pureté (pas d'effets de bord dans le routage). |
| `internal/cli/completion/` | Registry unique, 4 générateurs shell ; échappement des identifiants vers le shell — risque de sécurité latent. |
| `internal/fibonacci/testdata/fibonacci_golden.json` | **Immuable** sans accord explicite ADR (oracle de non-régression algorithmique). Étendu à F(50k/100k/200k) sous ADR-0004 §B5 ; toute extension future requiert le même protocole. |

---

## Patterns de performance critiques

Contexte « pourquoi » du hot path (détail mesuré : [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md)). La table « Invariants » porte les contrats à ne pas casser.

- **`sync.Pool` pour `big.Int`** + **état/arène/bump unifié** dans `CalculationState`, réutilisé inter-appels via le slot GC-immune par calculateur (le GC forcé post-calcul purge les `sync.Pool`).
- **Allocateur bump** O(1), zéro fragmentation, mono-goroutine, pour les tampons FFT.
- **GC désactivé** pendant les grands calculs (N ≥ 1M) via `gcCtrl.WithGC(fn)` (panic-safe).
- **Parallélisme adaptatif** : produits pointwise et butterflies FFT répartis sur les cœurs via un sémaphore FFT global à **acquisition non bloquante** (pas de jeton ⇒ exécution sur la goroutine appelante : aucun interblocage avec la récursion). Panics de worker re-propagées.
- **Cache LRU de transformées FFT** : bénéficie **uniquement** aux chemins qui le consultent (`bigfft.Mul/Sqr` directs, `FFTOnlyStrategy`). Le **Fast Doubling par défaut ne consulte pas** le cache (zéro hit, mesuré).
- **PGO** via `make build-pgo`.

> Toute modif dans `internal/fibonacci/` ou `internal/bigfft/` : `benchstat` avant/après contre une baseline (`make bench-baseline`). Régression > 5 % = blocage.

---

## Commandes essentielles

```bash
make all            # clean + build + test
make test           # go test -v -race -cover ./...   (race: CGO/gcc requis — Linux/macOS/WSL)
make test-win       # go test -v -cover ./...          (Windows sans gcc : pas de -race)
make test-short     # go test -v -short ./...
make lint           # golangci-lint run ./...          (24 linters)
make coverage       # rapport HTML
make coverage-check # plancher de couverture 80 %
make benchmark      # go test -bench=BenchmarkFibonacci -benchmem -run=^$ ./internal/fibonacci/
make bench-baseline # régénère docs/audits/bench-baseline.txt
make stats          # décompte packages / LOC (canonique)
make build-pgo      # build avec PGO
make build-all      # cross-compilation (linux, windows, macOS ; amd64 + arm64)
```

- **Gate de pré-commit** : `scripts/check.sh` (POSIX) / `scripts/check.ps1` (Windows) — enchaîne build → vet → test -race → lint (**advisory**, ne bloque pas) → plancher couverture 80 %. Le gate dur est build/vet/test/couverture.
- **Sans `make`** (Windows) : utiliser les équivalents `go`. Sous PowerShell, `go test -bench=.` est mal parsé — préfixer (`-bench=BenchmarkFibonacci`).
- **`-race`** : exige CGO/gcc, indisponible sous Windows sans gcc. Sur cet hôte, passes `-race` complètes via WSL (`wsl go test -race ./...`). `libgmp-dev` absent de WSL ⇒ tag `gmp` non testable là.

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

1. **Performance critique** — Modif dans `internal/fibonacci/` ou `internal/bigfft/` : benchmark `benchstat` avant + après (`make bench-baseline`). Régression > 5 % = blocage.
2. **Golden tests obligatoires** — Tout changement algorithmique passe `fibonacci_golden.json`. Fichier **immuable** sans approbation ADR (aucun `-update`). Corpus étendu à F(50k/100k/200k) sous ADR-0004 §B5.
3. **Étanchéité des couches** — Hiérarchie `cmd → app → orchestration → fibonacci/bigfft → config/errors`. Trois arrows remontants gardés par `arch_test.go` : `threshold → config`, `errors → format`, `tui → fibonacci`.
4. **Concurrence contrôlée** — `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutine sans contrôle de cycle de vie. **Pas de nouveaux globals dans `bigfft/`** : les trois existants sont en `atomic.*` privés avec accesseurs (ADR-0003). Migration `FFTContext` exclusive tracée en backlog (ADR-0004 §B1, won't-fix release courante).
5. **Modifications chirurgicales** — Diff minimal. Un refactoring d'envergure (> 50 LOC sur > 2 fichiers) se justifie dans le message de commit (raison, compromis, alternative écartée).
6. **Pas de nouveaux fichiers `progress*` sans consultation** — Chemin de production : `Freeze`. Étendre un point existant après lecture de `internal/progress`.
7. **Bug fix avant refactor** — Un défaut actif touché par hasard pendant un refactor se corrige d'abord dans un commit isolé (`fix(scope):`).
8. **Validation locale** — Avant chaque commit/PR : `scripts/check.sh` / `scripts/check.ps1`, ou à défaut `make test` (ou `make test-win` sans gcc) `&& make lint`. Aucun garde-fou CI distant : la rigueur tient à la discipline locale.

---

## Workflow recommandé

```
1. Branche dédiée : git checkout -b <type>/<description-courte>
2. Test rouge → fix minimal → vert + golden + (benchmark si perf-sensitive).
3. go test ./<pkg>/... -count=1 && go vet ./... && golangci-lint run ./<pkg>/...
4. Si perf-sensitive : comparer via benchstat (baseline make bench-baseline).
5. Commit conventionnel « <type>(scope): description » (feat/fix/docs/perf/test/refactor).
6. PR, review, merge.
```

---

## Artefacts générés (NE PAS éditer à la main)

| Chemin | Régénération | Description |
|---|---|---|
| `docs/dashboard/` | `pnpm --filter @understand-anything/dashboard build:demo` puis recopie ([`docs/BUILD.md`](docs/BUILD.md#dashboard-statique-github-pages)) | Build React/Vite statique, déployé sur GitHub Pages. |

---

## Références

- [`docs/adr/`](docs/adr/) — décisions (0001 DTM, 0002 recover, 0003 globaux atomic, 0004 backlog, 0005 contrôle GC concurrent par refcount, 0006 annulation récursion FFT, 0007 pool SA6002 pointeur vs valeur, 0008 candidats rejetés audit 2026-06).
- [`docs/architecture/`](docs/architecture/) — diagrammes C4, dependency graph. [Dashboard interactif](https://agbruneau.github.io/FibGo/dashboard/).
- [`docs/algorithms/`](docs/algorithms/), [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md), [`docs/TESTING.md`](docs/TESTING.md), [`docs/PORTABILITY.md`](docs/PORTABILITY.md), [`docs/BUILD.md`](docs/BUILD.md).
- [`CHANGELOG.md`](CHANGELOG.md), [`CONTRIBUTING.md`](CONTRIBUTING.md), `.golangci.yml` (24 linters, `govet shadow`, exceptions documentées).
