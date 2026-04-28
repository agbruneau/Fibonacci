# 40 — Audit performance internal/fibonacci/

Date : 2026-04-28 — Tâche 5.1.
Périmètre : 19 fichiers `.go` non-test (~3 110 LOC) sous `internal/fibonacci/` + sous-packages `memory/` et `threshold/`. Aucun benchmark exécuté (réservé à 5.4) ; lecture exhaustive des sources.

## Cartographie

| Fichier | Rôle | LOC |
|---|---|---|
| `calculator.go` | `Calculator`/`CoreCalculator` ; décorateur `FibCalculator` (GC ctrl, FFT cache, fast path n≤93) | 247 |
| `calculator_gmp.go` | Backend GMP (build tag `gmp`) — Fast Doubling sur `gmp.Int` | 153 |
| `common.go` | Sémaphore global, `MaxPooledBitLen`, helpers `executeParallel3`/`executeTasks`/`executeMixedTasks`, arena pre-sizing | 369 |
| `constants.go` | `DefaultFFTThreshold=500k`, `DefaultParallelThreshold=4096`, `ParallelFFTThreshold=5M`, `DefaultStrassenThreshold=3072`, `FibonacciGrowthFactor=0.69424` | 69 |
| `doc.go` | Doc package | 22 |
| `doubling_framework.go` | Boucle Fast Doubling + dispatch parallèle/séquentiel + tuning cache FFT | 276 |
| `fastdoubling.go` | `OptimizedFastDoubling`, `CalculationState`, `statePool` | 271 |
| `fft.go` | `mulFFT`/`sqrFFT`/`smartMultiply`/`smartSquare` ; `executeDoublingStepFFT` (réutilise transforms) | 258 |
| `fft_based.go` | `FFTBasedCalculator` (FFT-only) | 62 |
| `matrix.go` | `MatrixExponentiation` calculator | 78 |
| `matrix_framework.go` | Boucle d'exponentiation binaire 2x2 | 114 |
| `matrix_ops.go` | Multiplication 2x2, Strassen-Winograd, squaring symétrique | 310 |
| `matrix_types.go` | `matrix`, `matrixState`, `matrixStatePool` (23 big.Int) | 209 |
| `modular.go` | `FastDoublingMod` (pas de pool, mod m) | 66 |
| `options.go` | `Options`, `normalizeOptions`, `configureFFTCache` | 110 |
| `progress_aliases.go` | Aliases vers `internal/progress` | 56 |
| `registry.go` | `DefaultFactory` (registre) | 262 |
| `strategy.go` | `Multiplier`/`DoublingStepExecutor` ; `AdaptiveStrategy`, `FFTOnlyStrategy` | 149 |
| `testing.go` | Helpers test (export `_test`) | 98 |
| `memory/arena.go` | `CalculationArena` bump-pointer | 107 |
| `memory/budget.go` | Estimation mémoire | 91 |
| `memory/gc_control.go` | `GCController` (auto/aggressive/disabled) | 107 |
| `threshold/manager.go` | `DynamicThresholdManager` (ring buffer 20 métriques) | 417 |
| `threshold/types.go` | Types métriques | 43 |

## Allocations dans hot paths

| Localisation | Pattern | Justification |
|---|---|---|
| `fastdoubling.go:223-227` | `new(big.Int)` ×5 dans `statePool.New` | Pool, alloué 1× puis recyclé |
| `matrix_types.go:101-125` | `new(big.Int)` ×20 + 3×`newMatrix()` dans `matrixStatePool.New` | Pool, alloué 1× puis recyclé |
| `doubling_framework.go:274` | `s.FK = new(big.Int)` après "stealing" du résultat | Header 24B unique en sortie de `ExecuteDoublingLoop` (commentaire explicite ligne 268-272) |
| `matrix_framework.go:112` | `state.res.a = new(big.Int)` | Idem, header de remplacement |
| `fft.go:50,69` | `z = new(big.Int)` dans `smartMultiply`/`smartSquare` | Uniquement si `z==nil` (caller passe destination) |
| `common.go:83` | `make([]big.Word, 0, words)` dans `preSizeBigInt` | Pre-sizing one-shot avant boucle |
| `arena.go:46-48,89` | `make([]big.Word, 0, words)` fallback | Hors-arena uniquement |
| `calculator_gmp.go:117-120` | `gmp.NewInt(0)` ×4 | Hors pool (pas de pool GMP), alloué une fois par calcul |
| `calculator_gmp.go:103` | `new(big.Int).SetBytes(g.Bytes())` | Conversion finale GMP→big.Int (1×) |
| `modular.go:26-29` | `big.NewInt(0)/(1)`, `new(big.Int)` ×2 | Pas de pool — chemin secondaire (mod m, mémoire O(log m)) |
| `threshold/manager.go:218,258-259` | `make([]IterationMetric, …)` dans `getActiveMetrics`/`filterMetricsByMode` | Allouée à chaque évaluation ; `MaxMetricsHistory=20` donc ≤480B, mais déclenchée tous les 5 iters dans la boucle chaude |

Aucune allocation `big.Int` dans les corps des trois multiplications ou la boucle de bits — tout passe par les états poolés et la rotation de pointeurs (`fastdoubling.go:210, 224`).

## sync.Pool — usage

| Pool | Get/Put bien appariés ? |
|---|---|
| `statePool` (`fastdoubling.go:220`) | Oui. `AcquireState()`/`ReleaseState()` ; tous les call sites (`fastdoubling.go:96-97`, `fft_based.go:48-49`) utilisent `defer ReleaseState`. `ReleaseState(nil)` safe. Garde-fou oversize via `checkLimit` ≥ 50M bits = `MaxPooledBitLen`. |
| `matrixStatePool` (`matrix_types.go:98`) | Oui. `acquireMatrixState`/`releaseMatrixState` avec `defer` (`matrix.go:72-73`). Oversize check découpé en 4 helpers (P1-08). |
| `globalSem` (`common.go:42`) | Sémaphore (channel), pas un Pool. `runParallel3Op` : `sem<-` puis `defer <-sem`. `executeTasks`/`executeMixedTasks` : idem avec `defer func(){<-sem}()`. Toutes les voies acquisition/relâche correctes. |

Note : la fonction `FastDoublingMod` n'utilise pas le pool — acceptable car son footprint est borné par O(log m).

## GC controller

- `memory/gc_control.go` — `GCController.Begin()` capture `originalGCPercent` via `debug.SetGCPercent(-1)` et pose un `SetMemoryLimit` à 3× heap initial (filet OOM). `End()` restaure `originalGCPercent` et remet `SetMemoryLimit(MaxInt64)` puis force un `runtime.GC()`.
- **Restauration garantie** : `calculator.go:175-177` — `gcCtrl := memory.NewGCController(...) ; gcCtrl.Begin() ; defer gcCtrl.End()`. Le `defer` couvre panics et early returns. `Begin/End` sont no-op si `!gc.active` (mode `disabled` ou `auto` sous `GCAutoThreshold=1M`).
- **Risques** : (1) en `aggressive` pour n petit, GC reste désactivé pendant un calcul rapide — mineur. (2) `runtime.GC()` synchrone dans `End()` peut allonger la latence post-calcul. (3) `SetGCPercent`/`SetMemoryLimit` sont **process-globaux** : deux calculs concurrents avec GC actif corrompent l'état (le second `Begin` lit le `-1` posé par le premier comme `originalGCPercent`). Pas de lock observé — voir Findings.

## Cache FFT LRU

- **Localisation** : `internal/bigfft/fft_cache.go` (hors `internal/fibonacci/`). Singleton `globalTransformCache` via `sync.Once`. Configuré côté Fibonacci par `configureFFTCache` (`options.go:78`) et ajusté dynamiquement dans `doubling_framework.go:245-262` (sample tous les 8 iters, P1-02).
- **Thread-safety** : `sync.RWMutex` pour `entries`/`lru` ; compteurs `atomic.Uint64` (hits/misses/evictions/accesses). `Get` prend RLock puis Lock pour `MoveToFront` — fenêtre de race bénigne (LRU advisory). Lock complet dans `Put`/`Clear`/`SetTransformCacheConfig`.
- **Stratégie d'éviction** : LRU classique via `container/list` ; éviction tant que `len ≥ MaxEntries` (`fft_cache.go:273`). Filtres : `!Enabled` ou `len(data)*_W < MinBitLen` (default 100k bits) court-circuitent Get/Put. **Deep copy** à l'insertion via buffer contigu unique (`fft_cache.go:286-292`) pour éviter de retenir les pools.

## Sous-packages

### memory/

- **Arena** (`arena.go`) : bump-pointer one-shot. `NewCalculationArena(n)` alloue `15 × wordsPerInt` `big.Word` si `n≥1000`. `AllocBigInt`/`PreSizeFromArena` carvent par triple-slice (`a.buf[off:off+w:off+w]` avec cap=length empêche tout overlap). Fallback heap si épuisé. `Reset()` est documenté comme invalidant tous les `*big.Int` issus de l'arena. Invariant tenu : aucun appel à `Reset()` dans le code prod (l'arena est jetée à la fin du calcul, le GC libère le bloc une fois `s` collecté).
- **GCController** (`gc_control.go`) : voir section dédiée. Mode `auto` à partir de `GCAutoThreshold=1_000_000`. `Stats()` calcule des deltas `endStats - startStats` (cohérent uniquement si End a été appelé).
- **Budget** (`budget.go`) : estimateur statique `~5 + 3 + 2 + 5 = 15× bytesPerFib` (state + FFT + cache + overhead). Utilisé par CLI et tests, pas dans la boucle. `ParseMemoryLimit` accepte K/M/G.

### threshold/

- **Calibration** : `DynamicThresholdManager` collecte des `IterationMetric{BitLen, Duration, UsedFFT, UsedParallel}` dans un ring buffer fixe de 20 entrées (`manager.go:53`, P1-03 : remplace une slice qui croissait). Adjustment toutes les 5 itérations (`DynamicAdjustmentInterval=5`), minimum 3 métriques (`MinMetricsForAdjustment`). Calcul `avgTimePerBit` partitionné FFT vs non-FFT et parallèle vs séquentiel ; ratios comparés à `FFTSpeedupThreshold=1.2` / `ParallelSpeedupThreshold=1.1`. Hystérésis `HysteresisMargin=0.15` empêche les oscillations. Bornes : floor 100k bits (FFT) / 1024 bits (parallèle), ceiling `2×` / `4×` la valeur initiale.
- **Race** : `RecordIteration` et `ShouldAdjust` documentés comme appelés depuis un seul goroutine (la boucle de doubling) — pas de mutex sur les writes (`manager.go:127`). `GetThresholds`/`GetStats` prennent RLock — thread-safe en lecture externe. **Note** : `analyzeFFTThreshold`/`analyzeParallelThreshold` allouent deux slices à chaque appel (5.1 hot path, voir Findings).

## Lisibilité hot paths

- `OptimizedFastDoubling.CalculateCore` (`fastdoubling.go:95-133`) : 38 lignes, doc ~50 lignes ci-dessus listant les 3 optimisations clés (zero-alloc, parallèle, FFT adaptatif) et la dérivation des identités F(2k)/F(2k+1). Lecture facile.
- `DoublingFramework.ExecuteDoublingLoop` (`doubling_framework.go:141-276`) : 135 lignes, marqué `nolint:gocognit` avec justification. Commentaires denses (P1-02, swap pattern, "stealing" résultat). Le pattern `s.FK, s.FK1, s.T2, s.T3, s.T1 = s.T3, s.T1, s.FK, s.FK1, s.T2` (l. 210) est expliqué juste au-dessus mais reste subtil — recommander un schéma ASCII en commentaire.
- `executeDoublingStepFFT` (`fft.go:94`) : bien commenté pour P1-01 (bump allocator). Les `defer Release()` sont correctement placés sur fkPoly/fk1Poly, puis dans chaque sous-goroutine sur v et p.
- `multiplyMatrixStrassen` (`matrix_ops.go:79`) : structure claire (8 add → 7 mul → 7 add), nommage S1..S8/P1..P7 conforme à Winograd. Pas de magic numbers résiduels.
- `gmpDoublingStep` (`calculator_gmp.go:73`) : 14 lignes, commentaires utiles. Réutilise judicieusement `a` comme temporaire après que `t1` contient F(2k).

## Findings perf potentiels

| # | Localisation | Description | Impact estimé |
|---|---|---|---|
| F1 | `gc_control.go:68-74` | `SetGCPercent`/`SetMemoryLimit` sont process-globaux mais aucun lock ne sérialise les `Begin`/`End` concurrents. Deux calculs simultanés (mode `auto` ou `aggressive`) avec n≥1M corrompent `originalGCPercent` et restaurent une valeur fausse. | Correctness>>perf : restauration GC permanente possible |
| F2 | `threshold/manager.go:218, 258-259` | `getActiveMetrics` + `filterMetricsByMode` allouent 3 slices par appel à `ShouldAdjust` (toutes les 5 itérations). Pour F(10M) = ~24 itérations donc ~5 appels — négligeable. Mais sous `DynamicAdjustmentInterval=1` ou pour des futurs n>>10M cela escalade. | Faible aujourd'hui, scalability tax |
| F3 | `doubling_framework.go:191` | `shouldParallelizeMultiplicationCached` appelé à chaque iter même quand `useParallel==false` à l'entrée — rien à optimiser puisque le test est court-circuité par `useParallel &&`. RAS, vérifié. | (faux positif) |
| F4 | `matrix_ops.go:96-104, 245-254` | Création d'un slice `[]multiplicationTask` à chaque `multiplyMatrixStrassen` / `multiplyMatrix2x2` (7 ou 8 entrées × 32B). Pour Matrix Exp avec n≥1M, ~24 itérations × ~2 mul/sqr = 50 allocs de ~256B. Marginal vs coût des big.Int mais pourrait être stack-allouable via `[8]multiplicationTask` + slice. | <1% probable |
| F5 | `common.go:281, 320, 337` | Goroutines `go func(t PT)` capturent un pointeur — escape analysis force `t` sur le heap à chaque batch parallèle (Strassen). Sur 7 mul × 24 itérations = 168 escapes/calc. Comparable à F4. | <1% |
| F6 | `doubling_framework.go:246` | `bigfft.GetTransformCache()` + `cache.Stats()` + `cache.Config()` sous RLock, échantillonné 1/8 itérations — mais déjà optimisé par P1-02. RAS sauf à passer à 1/16. | (déjà traité) |
| F7 | `calculator.go:240-246` | `calculateSmall` réalloue `big.NewInt(0)` et `big.NewInt(1)` à chaque appel pour n≤93. Hot path TUI/CLI rapide (n petits). Pas critique car n≤93 rarissime côté calcul long, mais visible en stress test. | <0.1% global |
| F8 | `calculator_gmp.go:117-120` | Pas de pool pour `gmp.Int` (contrairement à `*big.Int`). Chaque calcul GMP alloue 4 `gmp.Int` puis les libère. CGO + malloc système. | Mesurable seulement en boucle externe (orchestration multi-runs) |
| F9 | `arena.go:31` | `CalculationArena` est dimensionnée pour 15 big.Int. La doc parle de "12 temporaires FFT" — vérifier si 15 suffit toujours : `CalculationState` = 5, mais le chemin FFT crée des polynômes hors-arena via `bigfft`. Si le path Adaptive ne consomme que 5 big.Int côté Fibonacci, 15 est très généreux (gaspillage mémoire 3× sur grands n). | Mémoire seulement, pas de CPU |
| F10 | `modular.go` | `FastDoublingMod` n'utilise ni pool ni arena ; alloue 4 `big.Int` par appel. Acceptable pour usage one-shot, problématique en service web. | Dépend usage |
| F11 | `doubling_framework.go:170-173, 228-230` | `time.Now()` + `time.Since` à chaque iter quand `dtm != nil`. Sur >1 GHz CPUs `time.Now` ~25ns, soit ~600ns sur 24 iters — négligeable. RAS. | (faux positif) |

## Synthèse

- **Score perf : 9/10**. Le package est extrêmement soigné. Le cœur algorithmique respecte rigoureusement la directive « Pas d'allocations inutiles » : pools `statePool` et `matrixStatePool` cycle complet, arena bump-pointer pour pre-sizing, swaps de pointeurs au lieu de `Set` dans la boucle de doubling, "stealing" du résultat pour éviter une copie O(n) finale (commentaires P1-06, P1-08 et autres traces de refactor explicites). Les `defer Release/Put/End` sont systématiques. Le cache FFT est correctement isolé dans `bigfft/`, thread-safe, avec deep-copy à l'insertion pour découpler du pool. Le sous-package `threshold/` utilise un ring buffer fixe (P1-03) plutôt qu'une slice dynamique. Les commentaires de hot paths citent les benchmarks empiriques pour chaque constante.
- **Top 5 optimisations possibles (non implémentées)** :
  1. **F1 — Lock sur `GCController` global** : sérialiser `Begin/End` via un mutex package-level dans `memory/`, ou refuser explicitement les calculs concurrents à GC contrôlé. Risque correctness élevé en service multi-tenant.
  2. **F2 — Allocations dans `threshold.getActiveMetrics`** : retourner directement le `[20]IterationMetric` (array, pas slice copié) ou pré-allouer un buffer membre du manager. Gain mineur mais aligné avec la philosophie zéro-alloc du reste.
  3. **F4/F5 — Tasks Strassen sur la stack** : remplacer `[]multiplicationTask` par `[8]multiplicationTask` + slice + lambdas non-capturantes ; éviterait 7-8 escapes/iter en Matrix Exp.
  4. **F8 — Pool `gmp.Int`** : `sync.Pool` autour de `gmp.Int` pour les 4 temporaires (a, b, t1, t2) ; bénéfice surtout en orchestration multi-runs ou serveur.
  5. **F9 — Right-sizing de `CalculationArena`** : compter exactement combien de buffers sont pré-sized (5 pour Adaptive, davantage pour FFT-only ?) au lieu du `× 15` constant ; économise plusieurs MB par calcul à n=10M+.
- **Aucun finding bloquant** côté correctness perf à part F1 (qui dépend du modèle d'usage : single-process CLI = OK, service web concurrent = à corriger).
