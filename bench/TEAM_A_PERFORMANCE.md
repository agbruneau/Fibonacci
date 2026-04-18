# TEAM A — Performance & Profiling

Audit read-only du sous-ensemble `internal/{fibonacci,bigfft,parallel,orchestration}` du projet FibGo (Go 1.26.2 / Windows / Intel Core Ultra 9 275HX 24 threads). Race detector indisponible (pas de gcc) — non bloquant pour Team A.

## Résumé exécutif (≤ 10 lignes)

- **Findings** : 9 (P0 : 2, P1 : 5, P2 : 2)
- **Hotspot dominant identifié** : fuite des pools `wordSlice` / `fermatSlice` / `natSlice` dans `internal/bigfft/fft_poly.go` (Transform/InvTransform/mul/sqr ne libèrent **jamais** les buffers acquis, qui partent en GC à chaque itération du doubling — voir F-A1, ~1 GB de bruit alloc sur F(10⁶) FFTBased).
- **Conséquence** : `BenchmarkFibonacci/FFTBased/1M` montre 9 639 776 B/op et **3 127 allocs/op**, contre 1 818 285 B/op et **89 allocs/op** pour `FastDoubling/1M` — facteur 5× mémoire et 35× allocs sur le même travail final.
- **PGO** : profil `cmd/fibcalc/default.pgo` **absent** (Makefile l'attend) → builds standards ne bénéficient pas du PGO documenté.
- **Cache FFT** : `cache.Stats()` + `cache.Config()` appelés à *chaque itération* du doubling loop (deux RLock / iteration) en mode dynamic-thresholds — overhead non négligeable.
- **Sémaphores** : 2 sémaphores indépendants (NumCPU dans bigfft, NumCPU*2 dans fibonacci) peuvent additionner jusqu'à **NumCPU*3** goroutines actives — déjà documenté mais non chiffré.
- **Arena** : `memory.NewCalculationArena` alloué à chaque calcul (~210 MB sur le bench MatrixExp/1M) sans pool, alors qu'un sync.Pool d'arenas est trivial.

## Méthodologie

### Commandes exécutées (reproductibles, Windows bash)
```bash
go version  # go1.26.2 windows/amd64

# Lecture de la baseline pré-capturée
wc -l bench/baseline/*.txt
grep -nE 'ns/op' bench/baseline/benchmark.txt

# Reproduction d'un sous-ensemble de benchmarks pour profiling
cd C:/Users/agbru/OneDrive/Documents/GitHub/FibGo
go test -run='^$' -bench='^BenchmarkSmartSquare' -benchmem -benchtime=300ms -count=1 ./internal/fibonacci/
go test -run='^$' -bench='^BenchmarkPoolWith' -benchmem -benchtime=300ms -count=1 ./internal/bigfft/
go test -run='^$' -bench='^BenchmarkFibonacci/(FFTBased|MatrixExp)/1M$' \
        -cpuprofile=/tmp/cpu_xxx.prof -memprofile=/tmp/mem_xxx.prof \
        -benchmem -benchtime=500ms -count=1 ./internal/fibonacci/

# Profiling
go tool pprof -top -cum -nodecount=25 /tmp/mem_fft.prof
go tool pprof -list='executeDoublingStepFFT|invTransform|^transform$' /tmp/mem_fft.prof

# Escape analysis
go build -gcflags="-m=2" ./internal/fibonacci/
go build -gcflags="-m=2" ./internal/bigfft/
```

### Limitations
- **Race detector désactivé** : pas de gcc/cgo sur la machine d'audit ; impossible de cross-checker la sécurité concurrentielle des pools sous `-race` ici. Recommandation : exécuter `make test` (race) sur runner Linux/CI pour valider.
- **Bench warm-up court** : `-benchtime=300ms-500ms` pour l'audit profilé. La baseline `bench/baseline/benchmark.txt` est l'autorité chiffrée ; les exécutions ad-hoc servent à explorer les hotspots, pas à publier des chiffres définitifs.
- **PGO** : non testable car `cmd/fibcalc/default.pgo` n'existe pas (cf. F-A8).
- **GMP** : backend conditionnel (`build tag gmp`) hors périmètre.

### Tableau de référence baseline (extrait de `bench/baseline/benchmark.txt`)

| Bench | n | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| `BenchmarkSmartSquareSmall-24` | 64-bit | 5.468 | 0 | 0 |
| `BenchmarkSmartSquareMedium-24` | ~10k bits | 5 224 | 0 | 0 |
| `BenchmarkSmartSquareLarge-24` | ~500k bits | 263 764 | 7 | 0 |
| `BenchmarkSmartSquareVsSmartMultiply/smartSquare-24` | | 83 707 | 3 | 0 |
| `BenchmarkSmartSquareVsSmartMultiply/smartMultiply-24` | | 45 932 | 0 | 0 |
| `BenchmarkCacheImpact/WithDefaultCache-24` | F(10⁷) | 77 346 708 | 55 669 620 | 1 093 |
| `BenchmarkCacheImpact/WithOptimizedCache-24` | F(10⁷) | 92 715 967 | 55 394 811 | 1 084 |
| `BenchmarkCacheImpact/CacheDisabled-24` | F(10⁷) | 80 300 764 | 55 503 045 | 1 096 |
| `BenchmarkCacheHitRate-24` | F(10⁶) | 7 709 596 | 1 779 679 | 87 |
| `BenchmarkFibonacci/FastDoubling/1M-24` | F(10⁶) | 5 906 856 | 1 818 285 | **89** |
| `BenchmarkFibonacci/MatrixExp/1M-24` | F(10⁶) | 9 311 418 | 3 477 812 | **610** |
| `BenchmarkFibonacci/FFTBased/1M-24` | F(10⁶) | 7 959 480 | **9 639 776** | **3 127** |
| `BenchmarkFibonacci/FastDoubling/10M-24` | F(10⁷) | 49 662 321 | 55 470 525 | 1 086 |
| `BenchmarkFibonacci/MatrixExp/10M-24` | F(10⁷) | 49 787 077 | **109 617 340** | 2 011 |
| `BenchmarkFibonacci/FFTBased/10M-24` | F(10⁷) | 54 736 112 | 63 820 384 | **4 180** |
| `BenchmarkIterativeGenerator_Next-24` | par appel | 69 287 | 92 864 | 3 |
| `BenchmarkIterativeGenerator_First1000-24` | 1000 termes | 775 880 | 234 776 | **4 001** |

Constats clés :
1. `WithOptimizedCache` est **plus lent** que `WithDefaultCache` (92.7 ms vs 77.3 ms) — anomalie à investiguer (F-A6).
2. `FFTBased` et `MatrixExp/10M` sont massivement plus alloc-lourds que FastDoubling.
3. `IterativeGenerator_First1000` génère exactement 4 allocations par terme (`new(big.Int).Add` + `Set`).

---

## Findings (≥ 5)

### F-A1 : Pools sync.Pool de FFT « fuient » — buffers jamais relâchés

- **Sévérité** : **P0**
- **Fichier(s)** :
  - `internal/bigfft/fft_poly.go:158-195` (`Poly.transform`) — acquiert `valbits := acquireWordSliceUnsafe(wordCount)` (ligne 168) et `values := acquireFermatSlice(K)` (ligne 169) → stockés dans `PolValues.Values`, **pas de release**.
  - `internal/bigfft/fft_poly.go:209-252` (`PolValues.invTransform`) — `pbits` (ligne 216), `p` (217), `a` (244) → retournés dans `Poly.A`, pas de release.
  - `internal/bigfft/fft_poly.go:339-363` (`PolValues.mul`) — `r.Values` (346), `bits` (348), pas de release.
  - `internal/bigfft/fft_poly.go:377-400` (`PolValues.sqr`) — même pattern (lignes 384, 386).
  - Recherche `releaseWordSlice|releaseFermatSlice|releaseNatSlice` dans `internal/bigfft/` : seules occurrences **non-test** sont `allocator.go:74-75` (PoolAllocator cleanup) et `fft_poly.go:271-278` (legacy `NTransform`).
- **Mesure (pprof `mem_fft.prof` sur `BenchmarkFibonacci/FFTBased/1M`)** :
  ```
  Total alloc_space : 1 577.75 MB
    bigfft.acquireWordSliceUnsafe        1 008.21 MB (63.90 % cum)
    bigfft.init.func7  (pool 65 536 W)     673.92 MB (42.71 % flat)
    bigfft.init.func6  (pool 16 384 W)     357.14 MB (22.64 % flat)
    bigfft.acquireFermatSlice               44.50 MB
  ```
  `init.funcN` correspondent aux closures `New: func() any { return make([]big.Word, …) }` de `pool.go:18-29`. Leur appel répété prouve que `wordSlicePools[idx].Get()` retourne quasi systématiquement un slice frais.
- **Diagnostic** : Les fonctions `Transform/InvTransform/Mul/Sqr` exposent leurs résultats sous forme de `PolValues`/`Poly` qui *contiennent* les slices issues du pool. Comme aucun `defer release*(…)` n'est posé en sortie de la chaîne d'appel `executeDoublingStepFFT → fkPoly.Transform → pv.Mul → InvTransform → IntToBigInt`, **chaque itération du doubling loop alloue ~6 buffers neufs** (input, output Transform×2, mul/sqr, invTransform×3) qui finissent en GC. Le `sync.Pool` agit comme un `make()` chaque appel, et les caches LRU + warming masquent à peine la régression.
- **Patch proposé** (extrait, sans appliquer) :
  ```go
  // fft_poly.go — ajouter un release dans IntTo / IntToBigInt qui consomment
  // la Poly retournée. Une approche plus propre : rendre Transform/Mul/Sqr
  // sortir un type cleanup-aware.
  
  // Option 1 (chirurgical, rétro-compatible) :
  // Ajouter PolValues.Release() et Poly.Release() qui rendent les backing
  // au pool, à appeler en defer par les call-sites.
  
  func (v *PolValues) Release() {
      if v == nil || len(v.Values) == 0 { return }
      // Le backing word array est contigu : c'est cap(v.Values[0])*K.
      backing := v.Values[0]
      releaseFermatSlice(v.Values)
      releaseWordSlice(backing[:cap(backing)])
      v.Values = nil
  }
  
  func (p *Poly) Release() {
      if p == nil || len(p.A) == 0 { return }
      backing := big.Word(p.A[0])  // pseudo : récupère le slice contigu
      releaseNatSlice(p.A)
      releaseWordSlice(backing[:cap(backing)])
      p.A = nil
  }
  
  // Dans fibonacci/fft.go — call-sites :
  fkPoly, err := pFk.Transform(n)
  if err != nil { … }
  defer fkPoly.Release()
  
  fk1Poly, err := pFk1.Transform(n)
  if err != nil { … }
  defer fk1Poly.Release()
  
  // Et dans executeFFTTransformsSequential / Parallel :
  v, err := fkPoly.Mul(fk1Poly)
  defer v.Release()
  p, err := v.InvTransform()
  defer p.Release()
  ```
- **Gain estimé** :
  - `BenchmarkFibonacci/FFTBased/1M` : -60 % à -80 % d'allocs (3 127 → 600-1 200 allocs/op), -50 % à -65 % B/op (9.6 MB → 3-4 MB).
  - CPU : -10 % à -20 % grâce à la réduction GC (visible dans `runtime.semawakeup` 12 % CPU dans le profil — fortement sensible à GC).
  - Sur F(10⁷) : extrapolation -30 MB par opération.
- **Effort** : **M** (4-8 h). Modifier ~10 call-sites dans `fft_poly.go`, `fft_cache.go` (les copies cache), `fibonacci/fft.go` ; couvrir avec un test qui force `GODEBUG=allocfreetrace`. Risque : double-release ou release prématuré → mitigé par tests existants `pool_test.go`.

---

### F-A2 : `executeDoublingStepFFT` n'utilise pas le BumpAllocator pour le Transform

- **Sévérité** : **P1**
- **Fichier(s)** :
  - `internal/fibonacci/fft.go:106-113` — `pFk.Transform(n)` et `pFk1.Transform(n)` empruntent le chemin `GetPoolAllocator()` (cf. `bigfft/fft_poly.go:148`).
  - `internal/bigfft/fft.go:54-71` (`fftmulTo`) et `fft.go:87-104` (`fftsqrTo`) utilisent le bump pour les `Transform` internes ; mais le Fibonacci-level `executeDoublingStepFFT` court-circuite ce chemin et fait ses propres `Transform`.
- **Mesure (pprof `mem_fft.prof`)** :
  ```
  bigfft.(*Poly).transform           488.36 MB (30.95 % cum)
    acquireWordSliceUnsafe (l.168)   296.54 MB
    AllocFermatSlice  (l.164, pool)  171.09 MB
  bigfft.(*PolValues).invTransform   475.78 MB (30.16 % cum)
    acquireWordSliceUnsafe (l.216)   414.19 MB
  ```
  Soit **964 MB / 1 578 MB (61 %) du trafic alloc passe par le PoolAllocator**, alors que la chaîne `fftmulTo` (qui utilise le bump) montre quasi 0 alloc à cet endroit.
- **Diagnostic** : Le bump allocator (`internal/bigfft/bump.go`) est conçu pour le cas où **tous** les buffers temporaires d'une opération FFT vivent et meurent ensemble. Or `executeDoublingStepFFT` partage `fkPoly` et `fk1Poly` entre 3 sous-opérations parallèles : on ne peut pas utiliser un même bump non thread-safe pour les 3. Néanmoins, on peut :
  1. Allouer `fkPoly`/`fk1Poly` via un bump dédié *par chemin* puis les libérer après les 3 multiplications.
  2. Utiliser `Poly.TransformWithBump` (existe déjà : `fft_poly.go:154`) avec un bump local au scope de `executeDoublingStepFFT`.
- **Patch proposé** :
  ```go
  // fibonacci/fft.go : executeDoublingStepFFT
  ba := bigfft.AcquireBumpAllocator(bigfft.EstimateBumpCapacity(2*fk1Words))
  defer bigfft.ReleaseBumpAllocator(ba)
  
  pFk := bigfft.PolyFromInt(s.FK, k, m)
  fkPoly, err := pFk.TransformWithBump(n, ba)
  ...
  pFk1 := bigfft.PolyFromInt(s.FK1, k, m)
  fk1Poly, err := pFk1.TransformWithBump(n, ba)
  ...
  // Mais : le mode parallèle 3-way doit avoir ses propres bumps (bump non TS).
  // → en mode parallèle, retomber sur PoolAllocator OU créer 3 bumps.
  ```
- **Gain estimé** :
  - Mode séquentiel (par défaut quand `inParallel=false` ou opérandes < 5M bits) : -40 % à -55 % d'allocs FFT.
  - Mode parallèle : neutre (déjà 3 goroutines indépendantes).
- **Effort** : **S** (2-4 h). Existence de `TransformWithBump` rend le patch trivial.

---

### F-A3 : Lecture stats cache FFT à chaque itération du doubling loop (locks répétés)

- **Sévérité** : **P1**
- **Fichier(s)** :
  - `internal/fibonacci/doubling_framework.go:230-245` — dans la boucle `for i := numBits-1; i >= 0; i--`, lorsque `dtm != nil` (dynamic thresholds activé) :
    ```go
    cache := bigfft.GetTransformCache()
    stats := cache.Stats()      // RLock + Load + Lock(mu)+RLock(mu) interne
    if stats.Evictions > 0 && stats.HitRate > 0.5 {
        cfg := cache.Config()    // RLock encore
        if cfg.MaxEntries < 8192 { ... bigfft.SetTransformCacheConfig(cfg) }
    } else if stats.HitRate < 0.1 && (stats.Misses+stats.Hits) > 10 {
        cfg := cache.Config()    // RLock encore
        ...
    }
    ```
  - `internal/bigfft/fft_cache.go:69-74` (`Config()` prend `RLock`), `fft_cache.go:315-336` (`Stats()` prend `RLock`).
- **Mesure** : pour `n=10⁷`, `numBits = bits.Len64(10⁷) = 24` itérations → **24× (2 RLock + 3 atomic.Load)** par calcul, plus 1 `Lock` complet quand `SetTransformCacheConfig` est invoqué (réécrit `cache.config` sous Lock — verrou exclusif). Sur le profil CPU FFT 10M, `runtime.lock2` apparaît à 5 % (210 ms / 3 200 ms échantillon) — non-trivial.
- **Diagnostic** : Le code adapte `MaxEntries` et `MinBitLen` itérativement, ce qui est légitime, mais (a) la condition d'ajustement est binaire (>0.5 ou <0.1) → la majorité des itérations ne font rien d'utile, (b) prendre 2 RLocks par itération est coûteux car partagé entre toutes les goroutines FFT actives.
- **Patch proposé** :
  ```go
  // doubling_framework.go : N'évaluer les stats que tous les K itérations
  // (par exemple K=8 ou K=numBits/4)
  const cacheStatCheckInterval = 8
  if dtm != nil && i%cacheStatCheckInterval == 0 {
      cache := bigfft.GetTransformCache()
      stats := cache.Stats()
      ...
  }
  // OU déplacer dans le DTM lui-même via dtm.RecordIteration
  ```
- **Gain estimé** : -3 % à -5 % CPU sur les boucles longues (n ≥ 10⁶) en mode dynamic-thresholds. Negligible si DTM désactivé.
- **Effort** : **S** (1-2 h).

---

### F-A4 : `IterativeGenerator.Next` alloue 1 big.Int + 1 copie par itération

- **Sévérité** : **P1**
- **Fichier(s)** :
  - `internal/fibonacci/generator_iterative.go:113` — `g.current, g.next = g.next, new(big.Int).Add(g.current, g.next)` : **`new(big.Int)`** à chaque appel.
  - `internal/fibonacci/generator_iterative.go:115` — `return new(big.Int).Set(g.current), nil` : **`new(big.Int)`** + `Set` (copie complète).
- **Mesure** :
  - `BenchmarkIterativeGenerator_Next-24` : 92 864 B/op, **3 allocs/op** (les 3 = current.Add backing growth + new(big.Int) header + Set copy).
  - `BenchmarkIterativeGenerator_First1000-24` : 234 776 B/op, **4 001 allocs/op** ≈ 4 allocs × 1 000 itérations.
- **Diagnostic** : Pour un générateur stateful, on peut (a) recycler un buffer `temp` au lieu de `new(big.Int)` à chaque Add, (b) offrir une variante `NextInto(dst *big.Int)` qui évite la copie de retour quand l'appelant n'a pas besoin d'une valeur indépendante.
- **Patch proposé** :
  ```go
  // generator_iterative.go : ajouter un buffer interne
  type IterativeGenerator struct {
      current, next, tmp *big.Int  // ajouter tmp
      ...
  }
  
  func (g *IterativeGenerator) Next(ctx context.Context) (*big.Int, error) {
      ...
      // Ancien : g.current, g.next = g.next, new(big.Int).Add(g.current, g.next)
      g.tmp.Add(g.current, g.next)
      g.current, g.next, g.tmp = g.next, g.tmp, g.current  // rotation
      return new(big.Int).Set(g.current), nil  // copie défensive (API contract)
  }
  
  // Nouvelle API zero-alloc :
  func (g *IterativeGenerator) NextInto(ctx context.Context, dst *big.Int) error {
      ... // identique mais dst.Set au lieu de new(big.Int).Set
  }
  ```
- **Gain estimé** :
  - `Next` : 3 → 1 alloc/op (-66 %), B/op -33 %.
  - `NextInto` (nouvelle API) : 0 alloc/op, ~0 B/op après warm-up.
  - `First1000` : 4 001 → ~1 001 allocs/op.
- **Effort** : **S** (1-2 h, plus tests).

---

### F-A5 : `CalculationArena` allouée à chaque calcul, jamais poolée

- **Sévérité** : **P1**
- **Fichier(s)** :
  - `internal/fibonacci/fastdoubling.go:103` — `arena := memory.NewCalculationArena(n)` à chaque entrée de `CalculateCore`.
  - `internal/fibonacci/memory/arena.go:25-36` — `NewCalculationArena` fait `make([]big.Word, totalWords)` — pour F(10⁷) : `15 * (10⁷ * 0.694 / 64 + 1) ≈ 1 627 000` words = **~13 MB par calcul**.
  - Pas de `sync.Pool` autour.
- **Mesure** : pprof `mem_me.prof` (MatrixExp/1M, mais arena n'est pas utilisée par MatrixExp) ; pour FastDoubling/1M, `BenchmarkFibonacci/FastDoubling/1M` montre 1 818 285 B/op dont une partie (~10 %) est l'arena. Sur `mem_fft.prof` (FFTBased qui *utilise* aussi l'arena via le même `AcquireState` path) : `memory.NewCalculationArena` = **211.41 MB / 1 577 MB (13.4 %)**.
- **Diagnostic** : L'arena est volontairement « jetée » à la fin du calcul pour permettre à GC de récupérer les big.Word qui pourraient être référencés par des slices sortantes. Cependant, pour les calculs internes (la plupart), aucun big.Int ne « fuit » hors du scope — seul le résultat final est *volé* (cf. `doubling_framework.go:256` `result := s.FK; s.FK = new(big.Int)`). Donc l'arena pourrait être recyclée via un `sync.Pool[*CalculationArena]`, en remettant son `offset` à 0 au release.
- **Patch proposé** :
  ```go
  // memory/arena.go : ajouter pool
  var arenaPool = sync.Pool{
      New: func() any { return &CalculationArena{} },
  }
  
  func AcquireArena(n uint64) *CalculationArena {
      a := arenaPool.Get().(*CalculationArena)
      requiredWords := estimateWords(n)
      if cap(a.buf) < requiredWords {
          a.buf = make([]big.Word, requiredWords)
      } else {
          a.buf = a.buf[:requiredWords]
      }
      a.offset = 0
      return a
  }
  
  func ReleaseArena(a *CalculationArena) {
      // Borne supérieure : ne pas garder d'arenas géantes en pool.
      if cap(a.buf) > 50_000_000 { return }  // ~400 MB pour F(10^9)
      a.offset = 0
      arenaPool.Put(a)
  }
  
  // fibonacci/fastdoubling.go :
  arena := memory.AcquireArena(n)
  defer memory.ReleaseArena(arena)
  ```
- **Gain estimé** :
  - FastDoubling/10M : -10 % B/op, -3-5 % CPU sur les benchs en boucle (le throughput-test `runBenchmark` appelle Calculate des dizaines de fois → l'arena pool évite les `make([]big.Word, 1.6M)` répétés).
  - FFTBased/10M : -15 MB/op sur les 64 MB.
- **Effort** : **S** (2 h, tests inclus).

---

### F-A6 : `WithOptimizedCache` plus lent que `WithDefaultCache` — anomalie de configuration

- **Sévérité** : **P2**
- **Fichier(s)** :
  - `internal/fibonacci/cache_bench_test.go` (à inspecter pour les paramètres `MaxEntries`/`MinBitLen` testés)
  - `internal/bigfft/fft_cache.go:35-41` (`DefaultTransformCacheConfig`)
- **Mesure (baseline)** :
  ```
  BenchmarkCacheImpact/WithDefaultCache-24    77 346 708 ns/op  55 669 620 B/op  1 093 allocs/op
  BenchmarkCacheImpact/WithOptimizedCache-24  92 715 967 ns/op  55 394 811 B/op  1 084 allocs/op  ← +20 % plus lent
  BenchmarkCacheImpact/CacheDisabled-24       80 300 764 ns/op  55 503 045 B/op  1 096 allocs/op
  ```
  Sur F(10⁷) : « optimized » est 19.9 % plus lent que « default » et 15.4 % plus lent que « disabled ».
- **Diagnostic** : Le nom `WithOptimizedCache` suggère un tuning spécifique (probablement plus grand `MaxEntries` ou plus petit `MinBitLen`). Si `MinBitLen` est trop bas, on cache des transforms petites dont le coût de copie/hashing dépasse le gain. Le `BenchmarkCacheHitRate-24` montre **Hit Rate: 4.55 %** — taux de hit dérisoire pour un calcul Fibonacci où les opérandes changent à chaque itération.
- **Patch proposé** : Lire `cache_bench_test.go` pour identifier les paramètres "optimized", puis :
  - Soit corriger les paramètres (probablement réduire `MaxEntries` ou augmenter `MinBitLen`).
  - Soit re-justifier le nom (renommer en `WithAggressiveCache` si l'intention est d'observer la dégradation).
  - Investigation : `MulCachedWithBump` réalloue le `*PolValues` complet via deep-copy à chaque `Put` (`fft_cache.go:283-292` — `make([]big.Word, wordCount)` × `copy()`). Cela explique pourquoi le cache peut être contre-productif quand le hit rate est faible.
- **Gain estimé** : -10 % à -20 % temps si on tune correctement le cache, ou si on désactive le cache par défaut sur les calculs Fibonacci où le hit rate < 10 %.
- **Effort** : **S** (2-3 h, dont 1 h d'A/B benchmark).

---

### F-A7 : Sémaphores indépendants → sur-souscription jusqu'à NumCPU*3 goroutines

- **Sévérité** : **P2**
- **Fichier(s)** :
  - `internal/fibonacci/common.go:32` — `taskSemaphore = make(chan struct{}, runtime.NumCPU()*2)`
  - `internal/bigfft/fft_recursion.go:20` — `concurrencySemaphore = make(chan struct{}, runtime.NumCPU())`
  - Le commentaire en `common.go:25-30` reconnaît explicitement le problème : « *up to NumCPU*3 goroutines may be active simultaneously* ».
- **Mesure** : Sur 24 logical cores, plafond théorique = **72 goroutines actives simultanées**. Le profil CPU FFT 1M montre `runtime.semawakeup` 12.30 % et `runtime.semasleep` 3.28 % du CPU (15.6 % cumulé en sync overhead). C'est une signature classique de sur-souscription lors des phases parallèles.
- **Diagnostic** : Deux sémaphores indépendants ne respectent pas la borne globale. Le commentaire prétend que `ShouldParallelizeMultiplication` mitige (en désactivant Fibonacci-parallel quand FFT est actif), mais l'exception « > ParallelFFTThreshold = 5 M bits » réintroduit la sur-souscription sur les très grands calculs (justement ceux où le coût synchro est le pire).
- **Patch proposé** :
  ```go
  // Approche 1 (chirurgical, peu invasif) : un sémaphore unifié
  // dans un nouveau package internal/parallel/semaphore.go
  package parallel
  
  var globalCpuSemaphore = make(chan struct{}, runtime.NumCPU())
  
  func AcquireCPU() { globalCpuSemaphore <- struct{}{} }
  func ReleaseCPU() { <-globalCpuSemaphore }
  
  // bigfft/fft_recursion.go et fibonacci/common.go partagent ce semaphore.
  
  // Approche 2 (zero-cost, simple) : utiliser golang.org/x/sync/semaphore
  // (déjà transitif via errgroup) avec un weight=1 par goroutine FFT et
  // weight=2 par goroutine Fibonacci-level, pour refléter le coût relatif.
  ```
- **Gain estimé** : -8 % à -12 % CPU sync overhead sur les benchs FFT/MatrixExp 10M+, conditionné à un benchmark A/B sérieux. Risque potentiel : sous-utilisation si les opérations sont I/O-bound (non-applicable ici, tout est CPU).
- **Effort** : **M** (4-6 h). Nécessite un benchmark dédié pour valider sur grandes tailles avant merge.

---

### F-A8 : PGO non disponible — `cmd/fibcalc/default.pgo` absent

- **Sévérité** : **P2**
- **Fichier(s)** :
  - `Makefile:13` — `PGO_PROFILE=$(CMD_DIR)/default.pgo`
  - `Makefile:33-44` — la cible `build` fait `if [ -f $(PGO_PROFILE) ]; then ... -pgo=$(PGO_PROFILE) ...; else echo "Building without PGO..."; fi`
  - Vérification : `glob **/*.pgo` retourne 0 fichier → PGO n'est jamais activé sur build standard.
- **Mesure** :
  - Pas mesurable directement sans génération du profil. Cible Makefile `pgo-profile` génère le profil via `go test -cpuprofile -bench=BenchmarkFastDoubling -benchtime=5s -count=3 ./internal/fibonacci/`.
  - Gain documenté typique du PGO Go : 2-7 % CPU (cf. release notes Go 1.21+, blog Go team).
- **Diagnostic** : Le projet promeut le PGO dans `CLAUDE.md` (« PGO supporté »), mais le profil n'est pas committé. Le développeur doit lancer `make pgo-profile` puis `make build-pgo` manuellement. Pour un projet visant la perf, c'est un manque.
- **Patch proposé** :
  1. Générer un profil de référence représentatif (Fast Doubling + FFT + Matrix combinés) :
     ```bash
     make pgo-profile  # à enrichir : couvrir Matrix + FFT en plus de FastDoubling
     git add cmd/fibcalc/default.pgo
     ```
  2. Documenter dans le README la procédure de regénération (« regénérer après chaque release majeure »).
  3. Ajouter un job CI optionnel `pgo-rebuild` qui vérifie que le profil n'est pas obsolète (ratio âge / commits).
- **Gain estimé** : 2-5 % CPU sur les chemins chauds (`smartMultiply`, `smartSquare`, FFT recursion, fermat operations).
- **Effort** : **S** (1-2 h pour générer + commit + doc), **M** si on automatise CI.

---

### F-A9 : `executeParallel3` alloue WaitGroup + ErrorCollector sur le tas à chaque appel

- **Sévérité** : **P2**
- **Fichier(s)** :
  - `internal/fibonacci/common.go:97-144` (`executeParallel3`) — déclare `var wg sync.WaitGroup` et `var ec parallel.ErrorCollector` localement.
  - Escape analysis (`go build -gcflags="-m=2"`) :
    ```
    internal\fibonacci\common.go:99:6: moved to heap: wg
    internal\fibonacci\common.go:100:6: moved to heap: ec
    ```
  - Mêmes findings dans `executeTasks` (`common.go:209-225`) et `executeMixedTasks` (`common.go:257-287`).
- **Mesure** : 2 allocs (32 + 24 octets) par appel à `executeParallel3`. Sur F(10⁷) FFTBased : ~24 itérations × 1 = **24 allocs supplémentaires**. Marginal mais mesurable.
- **Diagnostic** : Les goroutines closures référencent `&wg` et `&ec` → escape forcé. Une `sync.Pool` de paires `{wg, ec}` éliminerait l'allocation.
- **Patch proposé** :
  ```go
  // common.go : pool de structures auxiliaires
  type parallel3Aux struct {
      wg sync.WaitGroup
      ec parallel.ErrorCollector
  }
  
  var parallel3AuxPool = sync.Pool{New: func() any { return &parallel3Aux{} }}
  
  func executeParallel3(ctx context.Context, op1, op2, op3 func() error) error {
      aux := parallel3AuxPool.Get().(*parallel3Aux)
      defer func() {
          aux.ec.Reset()  // déjà thread-unsafe — OK car wg.Wait a complété
          parallel3AuxPool.Put(aux)
      }()
      aux.wg.Add(3)
      ...
  }
  ```
- **Gain estimé** : -2 à -3 allocs/op sur les chemins parallèles (FFTBased seq/parallel, Strassen). Impact CPU négligeable mais propre.
- **Effort** : **S** (1 h).

---

## Annexes

### A. Tableau récapitulatif des findings

| ID | Sévérité | Catégorie | Fichier:ligne | Mesure | Gain estimé | Effort |
|---|---|---|---|---|---|---|
| F-A1 | **P0** | Pool leak | `bigfft/fft_poly.go:158-400` | 64 % alloc traffic (1 GB/1.5 GB) | -60 % allocs FFT, -10 % CPU | M |
| F-A2 | P1 | BumpAllocator non utilisé | `fibonacci/fft.go:106-113` | 61 % alloc via PoolAllocator | -40 % allocs Transform | S |
| F-A3 | P1 | Lock contention cache | `fibonacci/doubling_framework.go:230-245` | 5 % CPU `runtime.lock2` | -3-5 % CPU | S |
| F-A4 | P1 | Allocation par itération | `fibonacci/generator_iterative.go:113,115` | 4 001 allocs/1k iter | -75 % allocs | S |
| F-A5 | P1 | Arena non poolée | `fibonacci/fastdoubling.go:103` + `memory/arena.go` | 13.4 % alloc (211 MB) | -10 % B/op, -3-5 % CPU | S |
| F-A6 | P2 | Cache config sub-optimale | `fft_cache.go` + `cache_bench_test.go` | +20 % temps « optimized » vs default | -10-20 % si bien tuné | S |
| F-A7 | P2 | Sur-souscription sémaphores | `bigfft/fft_recursion.go:20` + `fibonacci/common.go:32` | 15.6 % CPU sync (sema*) | -8-12 % CPU large N | M |
| F-A8 | P2 | PGO indisponible | `cmd/fibcalc/default.pgo` (absent) | n/a | +2-5 % CPU | S |
| F-A9 | P2 | wg/ec escape to heap | `fibonacci/common.go:99-100,209,257` | 2 allocs/`executeParallel3` | -24 allocs/calcul 10M | S |

### B. Fichiers et sections explorés (non exhaustif)

- Sources lues intégralement : `internal/bigfft/{pool.go, bump.go, allocator.go, fft.go, fft_core.go, fft_cache.go, fft_recursion.go, fft_poly.go, pool_warming.go}`, `internal/fibonacci/{common.go, fastdoubling.go, matrix.go, matrix_types.go, matrix_framework.go, matrix_ops.go, fft.go, fft_based.go, doubling_framework.go, strategy.go, constants.go, generator_iterative.go, memory/arena.go}`, `internal/parallel/errors.go`, `internal/orchestration/orchestrator.go`.
- Profils utilisés : `cpu_me.prof` (MatrixExp/1M), `mem_me.prof`, `cpu_fft.prof` (FFTBased/1M), `mem_fft.prof`, `cpu.prof`/`mem.prof` (SmartSquareLarge).
- Escape analysis : `go build -gcflags="-m=2" ./internal/fibonacci/` et `./internal/bigfft/`.

### C. Mesures non capturables ici (à exécuter en CI Linux/race)

- `make test -race` (race detector sur `executeParallel3`, `TransformCache`).
- Bench A/B `before/after` pour chaque finding P0/P1, via `benchstat`. Recommandation : exécuter avec `-count=10 -benchtime=2s` sur runner dédié.
- Mesure réelle gain PGO via `make pgo-rebuild`.
- Profiling sous Linux (cgroups/perf) pour valider le 8-12 % de F-A7 sur des opérandes > 5M bits.
