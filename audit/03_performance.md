# Audit — Axe 3 : Performance & Benchmarks

> Commit audité : `866b8cd` (2026-05-24). Exécution : 2026-05-28, hôte Windows 11 / Intel Core Ultra 9 275HX (24 threads), Go 1.26.3, `CGO_ENABLED=0`. `-benchtime` borné (2x–10x) selon le mandat.

## 1. Verdict d'axe

**Aucune régression de performance ni d'allocation détectée par rapport aux baselines `docs/audits/`** — au contraire, le hot path Fast Doubling à N=1M alloue *moins* qu'avant (102–110 allocs/op mesurés contre 132 en baseline). Les invariants de complexité tiennent : Fast Doubling est bien O(log n · M(n)) et la multiplication FFT évolue empiriquement en O(n log n). Les stratégies `sync.Pool`, bump allocator et arena sont efficaces : les chemins de réutilisation atteignent 0 allocs/op. Les constats relevés sont **de nature documentaire et structurelle**, pas des défauts de performance : le cache de transformées FFT est **contourné sur le chemin de production par défaut** (gap doc/réalité MAJEUR), et le calcul de clé de cache impose un coût O(n) sur l'entrée même sans bénéfice (MINEUR). Aucun blocage au sens de la directive « régression > 5 % ».

## 2. Tableau récapitulatif

| ID | Sévérité | Titre | Marqueur |
|----|----------|-------|----------|
| A3-01 | MAJEUR | Cache de transformées FFT contourné sur le chemin de production par défaut (gap doc/réalité « 15-30 % speedup ») | [confirmé] |
| A3-02 | MINEUR | Coût O(n) du hachage de clé de cache FFT (~5 % CPU) imposé même sans gain possible | [confirmé] |
| A3-03 | MINEUR | `FFTOnlyStrategy.Multiply/Square` : copie redondante `z.Set(result)` via `setOrReturn` après allocation neuve | [confirmé] |
| A3-04 | INFORMATIF | Ordre des algorithmes (Matrix plus rapide que Fast Doubling à 10M) contredit `docs/PERFORMANCE.md` sur cet hôte | [à vérifier] |
| A3-05 | INFORMATIF | Sous-goroutines FFT parallèles retombent sur le pool (bump non thread-safe) : bénéfice zéro-alloc non étendu | [confirmé] |
| A3-06 | INFORMATIF | `math/big.getStack` = 27 % du *nombre* d'allocations (récursion Karatsuba profonde) | [confirmé] |
| A3-07 | INFORMATIF | Confirmation : aucune régression vs baselines ; allocations en amélioration | [confirmé] |

---

## 3. Détail des constats

### [A3-01] Cache de transformées FFT contourné sur le chemin de production par défaut

- **Sévérité** : MAJEUR
- **Axe** : 3 Performance
- **Emplacement** : `internal/fibonacci/fft.go:128-140` (`executeDoublingStepFFT`) vs `internal/bigfft/fft_cache.go:444-504` (`TransformCached*`) ; `internal/bigfft/fft_core.go:65,100`
- **Preuve** :

Le pas de doublement FFT de production (`AdaptiveStrategy.ExecuteStep` → `executeDoublingStepFFT`) transforme FK/FK1 via `TransformWithBump`, **pas** via `TransformCachedWithBump` :

```go
// internal/fibonacci/fft.go:129
fkPoly, err := pFk.TransformWithBump(n, ba)
...
// internal/fibonacci/fft.go:136
fk1Poly, err := pFk1.TransformWithBump(n, ba)
```

Recherche des appelants du chemin caché (`grep`) :

```
internal\bigfft\fft_core.go:65:	rp, err := xp.MulCachedWithBump(&yp, ba)
internal\bigfft\fft_core.go:100:	rp, err := xp.SqrCachedWithBump(ba)
```

Le cache n'est consulté que par `mulFFT`/`sqrFFT` (package `bigfft`), eux-mêmes utilisés par `FFTOnlyStrategy.Multiply/Square` et `smartMultiply` — pas par le pas de doublement adaptatif par défaut. La sonde `BenchmarkCacheHitRate` (N=1M, Fast Doubling) le confirme :

```
--- BENCH: BenchmarkCacheHitRate-24
    cache_bench_test.go:133: Cache stats - Hits: 0, Misses: 0, Hit Rate: 0.00%, Size: 0
```

- **Impact** : `docs/PERFORMANCE.md` et `CLAUDE.md` annoncent « Cache FFT LRU thread-safe — 15-30 % speedup ». Sur le chemin de production réel (calculateur Fast Doubling, défaut), ce cache n'est **jamais alimenté ni interrogé** : zéro hit, zéro miss. Le gain annoncé ne s'applique pas au mode par défaut. Le cache, son LRU, son `mutex`, son hachage de clé et son maintien LRU constituent du code mort sur le hot path par défaut (coût net nul mais bénéfice nul). Note : `executeDoublingStepFFT` applique tout de même la bonne optimisation locale (transformer FK/FK1 une fois, réutiliser pour les 3 produits du pas) — la correction est seulement que le bénéfice inter-itérations annoncé n'existe pas au défaut.
- **Recommandation** : (1) Corriger la documentation pour préciser que le cache de transformées ne bénéficie qu'aux chemins `FFTOnly` / appels directs `bigfft.Mul/Sqr`, pas au calculateur Fast Doubling par défaut. (2) Si le gain inter-itérations est souhaité au défaut, mesurer un branchement de `executeDoublingStepFFT` sur `TransformCachedWithBump` pour FK1 (réutilisé d'une itération à l'autre via la rotation de pointeurs) — bénéfice à valider avant/après par `benchstat`, car la clé O(n) (cf. A3-02) peut annuler le gain. Toute bascule touchant `bigfft/fft_cache.go` doit respecter l'invariant anti-aliasing `putByKey` (proposition d'ADR si modification de comportement).
- **Marqueur** : [confirmé]

---

### [A3-02] Coût O(n) du hachage de clé de cache FFT imposé même sans gain possible

- **Sévérité** : MINEUR
- **Axe** : 3 Performance
- **Emplacement** : `internal/bigfft/fft_cache.go:152-204` (`writeUint64`, `writeNat`, `computePolyKey`)
- **Preuve** : profil CPU de `BenchmarkFFTMulWithBump/fftmulTo_1M_words` (chemin `MulCachedWithBump`) :

```
     0.08s  4.52% 87.01%      0.08s  4.52%  ...bigfft.(*cacheKeyBuilder).writeUint64
     0.01s  0.56% 95.48%      0.09s  5.08%  ...bigfft.(*cacheKeyBuilder).writeNat (inline)
```

`computePolyKey` hache **chaque mot** des coefficients polynomiaux (FNV-1a octet par octet, 8 multiplications/mot) :

```go
// fft_cache.go:174
func (b *cacheKeyBuilder) writeNat(data nat) {
	for _, word := range data {
		b.writeUint64(uint64(word))   // 8 ops par mot
	}
}
// fft_cache.go:196
func computePolyKey(p *Poly, k uint, n int) uint64 {
	...
	for _, a := range p.A {
		b.writeNat(a)                 // hache tout l'opérande -> O(n)
	}
}
```

- **Impact** : la clé est recalculée en O(taille de l'entrée) à **chaque** `TransformCached*`, y compris sur un calcul à multiplication unique où aucun hit n'est possible (la première transformée est forcément un miss). Pour 1M mots, ~5 % du CPU de la multiplication FFT part dans le hachage de clé. Sur le chemin par défaut (cf. A3-01) ce coût est nul puisque le cache n'est pas consulté ; il ne pèse que sur `FFTOnly` / appels directs `bigfft.Mul`.
- **Recommandation** : (1) Sur `FFTOnlyStrategy`, comme la clé hache tout l'opérande, valider si le `MinBitLen` (100 000 bits) suffit à éviter le hachage sur petits opérandes — vérifier que le gate `len(data)*_W < minBitLen` court-circuite *avant* `computePolyKey` (c'est le cas dans `TransformCached*`, ligne 452/482 ; le coût ne frappe donc que les gros opérandes, ce qui est correct). (2) Optionnel : hacher un échantillon (premier/dernier mot + longueur + k + n) plutôt que l'intégralité, au prix d'un risque de collision accru — à peser via proposition d'ADR car cela change la sémantique de clé documentée (« bit-identical hashes »). Mesure avant/après obligatoire.
- **Marqueur** : [confirmé]

---

### [A3-03] Copie redondante dans `FFTOnlyStrategy.Multiply/Square`

- **Sévérité** : MINEUR
- **Axe** : 3 Performance
- **Emplacement** : `internal/fibonacci/strategy.go:15-21,128-143`
- **Preuve** :

```go
// strategy.go:128
func (s *FFTOnlyStrategy) Multiply(z, x, y *big.Int, opts Options) (*big.Int, error) {
	res, err := mulFFT(x, y)        // alloue déjà un *big.Int neuf (new(big.Int) dans mulFFT)
	...
	return setOrReturn(z, res), nil // si z != nil : z.Set(res) -> recopie de tous les mots
}
// strategy.go:15
func setOrReturn(z, result *big.Int) *big.Int {
	if z != nil {
		z.Set(result)               // copie O(n) supplémentaire
		return z
	}
	return result
}
```

- **Impact** : sur le chemin `FFTOnlyStrategy.Multiply/Square` (calculateur FFTBased, outil de benchmark), `mulFFT` alloue déjà un résultat neuf, puis `setOrReturn` recopie ces mots dans `z` quand `z != nil`. C'est une copie O(n) inutile : on pourrait écrire directement dans `z` via `MulTo`/`SqrTo` (déjà présents dans `bigfft/fft.go`). Confirmé par les allocs élevés du calculateur FFTBased : 3821 allocs/op à 10M contre 1111 pour Fast Doubling. Impact limité car (a) `FFTOnlyStrategy.ExecuteStep` utilise en réalité `executeDoublingStepFFT` (chemin sans cette copie), donc `Multiply/Square` ne sont sur le hot path du calculateur FFTBased que pour les appels hors `ExecuteStep` ; (b) FFTBased est documenté comme outil de benchmark.
- **Recommandation** : faire pointer `FFTOnlyStrategy.Multiply/Square` sur `bigfft.MulTo(z, x, y)` / `bigfft.SqrTo(z, x)` quand `z != nil`, éliminant l'allocation neuve + la recopie. Vérifier l'équivalence via les golden tests et `benchstat` sur `BenchmarkFibonacci/FFTBased`.
- **Marqueur** : [confirmé]

---

### [A3-04] Ordre des algorithmes contredit `docs/PERFORMANCE.md` sur cet hôte

- **Sévérité** : INFORMATIF
- **Axe** : 3 Performance
- **Emplacement** : `docs/PERFORMANCE.md:20-42` (table de référence Ryzen) vs mesure locale
- **Preuve** : `go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)/10M' -benchtime=5x` :

```
FastDoubling/10M   42863660 ns/op   27991955 B/op   1111 allocs/op
MatrixExp/10M      34594340 ns/op   56908790 B/op   1976 allocs/op   <-- plus rapide en temps
FFTBased/10M       44471160 ns/op   30640382 B/op   3821 allocs/op
```

`docs/PERFORMANCE.md` affirme « Fast Doubling : Fastest for the majority of cases » et liste Fast Doubling < Matrix partout. Ici, à 10M, **Matrix est plus rapide en temps** (34,6 ms vs 42,9 ms) mais consomme **2× la mémoire** (57 MB vs 28 MB).

- **Impact** : observation, pas un défaut. L'écart de temps (~19 %) dépasse le seuil de 5 %, mais il s'agit (a) d'une comparaison inter-algorithmes (pas une régression d'une même cible), (b) sur hôte Windows partagé à `-benchtime=5x` (faible précision, cf. mandat — `ns/op` = [probable]), (c) sur un CPU (Core Ultra 9) différent de la référence (Ryzen 9 5900X). Le coût mémoire 2× de Matrix reste réel et stable. La doc reste cohérente sur l'ordre *mémoire* (Fast Doubling plus léger).
- **Recommandation** : reconfirmer l'ordre temps Fast Doubling vs Matrix à 10M+ sur la machine de référence Ryzen sous Linux avec `-count>=10` + `benchstat` avant d'ajuster la doc. Si l'inversion se confirme sur le matériel cible, nuancer la formule « Fastest for the majority of cases » dans `docs/PERFORMANCE.md`.
- **Marqueur** : [à vérifier] (machine et OS différents de la baseline ; précision `-benchtime=5x` faible)

---

### [A3-05] Sous-goroutines FFT parallèles retombent sur le pool (bump non thread-safe)

- **Sévérité** : INFORMATIF
- **Axe** : 3 Performance
- **Emplacement** : `internal/bigfft/fft_recursion.go:130-143`
- **Preuve** :

```go
// fft_recursion.go:137
t1, cleanup1 := GetPoolAllocator().AllocFermatTemp(n)
t2, cleanup2 := GetPoolAllocator().AllocFermatTemp(n)
```

Et le profil d'allocations (par *nombre* d'objets) du `BenchmarkFibonacci` :

```
     49153 20.53% 75.35%      49281 20.59%  ...bigfft.(*PoolAllocator).AllocFermatTemp
```

- **Impact** : la récursion FFT parallèle, quand elle parallélise une branche, n'utilise pas le bump allocator de la goroutine appelante (non thread-safe, commentaire ligne 134-136) mais le `sync.Pool` global. Le bénéfice « zéro fragmentation / O(1) » du bump ne s'étend donc pas aux sous-arbres parallèles : ~20 % du nombre d'allocations vient de cette retombée. C'est un compromis correct (sécurité concurrence > zéro-alloc), pas un défaut, mais cela explique pourquoi le hot path FFT ne descend pas à 0 allocs malgré la présence du bump allocator.
- **Recommandation** : aucune action requise. Optionnel et à valider par benchmark : doter chaque goroutine parallèle de son propre `BumpAllocator` acquis via `AcquireBumpAllocator` (un par branche), au lieu du pool de fermats — réduirait le nombre d'allocations mais ajoute une acquisition de pool par branche. Gain incertain ; ne pas entreprendre sans mesure avant/après.
- **Marqueur** : [confirmé]

---

### [A3-06] `math/big.getStack` = 27 % du nombre d'allocations (récursion Karatsuba)

- **Sévérité** : INFORMATIF
- **Axe** : 3 Performance
- **Emplacement** : interne `math/big` (atteint via `smartMultiply` → `z.Mul`), profil sur `BenchmarkFibonacci`
- **Preuve** : profil mémoire `-sample_index=alloc_objects` :

```
     65700 27.44% 27.44%      65700 27.44%  sync.(*Pool).pinSlow
     65536 27.38% 54.82%      65536 27.38%  math/big.getStack
     49153 20.53% 75.35%      49281 20.59%  ...bigfft.(*PoolAllocator).AllocFermatTemp
```

Profil CPU (mêmes benchmarks) : `math/big.basicMul` cum 41,96 %, `math/big.karatsuba` cum 52,68 % — la multiplication standard domine le CPU pour N=1M (sous le seuil FFT de 500 000 bits sur la majorité des itérations).

- **Impact** : observation neutre. `math/big.getStack` (pile de récursion interne de Karatsuba) et `sync.Pool.pinSlow` (épinglage P du pool) représentent chacun ~27 % du *nombre* d'allocations mais un volume d'octets faible (objets petits). Ce ne sont pas des allocations contrôlables par le projet — elles sont internes à `math/big` et à `sync.Pool`. Cela confirme que pour N ≤ ~720 K bits (F(1M)), la quasi-totalité des multiplications passe par `math/big` (Karatsuba), pas par FFT, ce qui est conforme au seuil documenté (`DefaultFFTThreshold = 500_000`).
- **Recommandation** : aucune. Documenter éventuellement dans `docs/PERFORMANCE.md` que pour 500 K < bits < quelques M, le coût est dominé par Karatsuba (`math/big`) et non par le code projet — c'est attendu.
- **Marqueur** : [confirmé]

---

### [A3-07] Confirmation : aucune régression vs baselines ; allocations en amélioration

- **Sévérité** : INFORMATIF
- **Axe** : 3 Performance
- **Emplacement** : `docs/audits/bench-baseline.txt` vs mesures locales ; `internal/fibonacci/fastdoubling.go` (arena), `internal/bigfft/{bump.go,pool.go}`
- **Preuve** :

Baseline (`docs/audits/bench-baseline.txt`, même CPU Core Ultra 9) :
```
BenchmarkFibonacci/FastDoubling/1M-24   1   3051500 ns/op   2646032 B/op   132 allocs/op
BenchmarkFibonacci/FastDoubling/10M-24  1  41295200 ns/op  35915568 B/op  1163 allocs/op
```

Mesure locale `-benchtime=10x` (1M) et `-benchtime=5x` (10M) :
```
FastDoubling/1M    3074410 ns/op   1779842 B/op   102 allocs/op   <-- moins d'allocs et d'octets
FastDoubling/10M  42863660 ns/op  27991955 B/op  1111 allocs/op   <-- moins d'allocs et d'octets
```

Efficacité des pools / bump / arena confirmée (chemins de réutilisation à 0 alloc) :
```
BenchmarkMulToReuse-24      250.0 ns/op    40 B/op    0 allocs/op
BenchmarkSqrToReuse-24      200.0 ns/op    40 B/op    0 allocs/op
BenchmarkMulToWithReuse-24  12450 ns/op     0 B/op    0 allocs/op   (vs WithoutReuse: 51136 B/op, 2 allocs)
BenchmarkAllocVsMake/Bump_1K-24  650.0 ns/op  0 B/op  0 allocs/op   (vs Make: 8192 B/op, 1 alloc)
```

Complexité FFT O(n log n) confirmée empiriquement (`BenchmarkFFTMulWithBump`, par décade de taille) :
```
fftmulTo_10K_words     1332100 ns/op
fftmulTo_100K_words   17789500 ns/op   (x13,4 pour x10 taille)
fftmulTo_1M_words    229117750 ns/op   (x12,9 pour x10 taille)   -> surlinéaire léger ~ n log n
```

- **Impact** : positif. Les allocations par opération sur le hot path Fast Doubling sont **inférieures** à la baseline (102 vs 132 à 1M ; 1111 vs 1163 à 10M), et les octets/op sont en baisse (1,78 MB vs 2,65 MB à 1M). Aucune cible ne dépasse le seuil de régression de 5 %. Le comportement de l'arena (réutilisation du `[]big.Word` entre appels) et du bump allocator est conforme aux invariants documentés.
- **Recommandation** : envisager de rafraîchir `docs/audits/bench-baseline.txt` (sur machine au repos, `make bench-baseline`) pour figer les nouveaux chiffres d'allocation améliorés comme référence de non-régression. Les `ns/op` absolus restant bruités sous Windows partagé + faible `-benchtime`, ne capturer la baseline temps que sur Linux idle avec `-count>=10`.
- **Marqueur** : [confirmé]

---

## 4. Notes de méthode et limites

- **Précision temps** : hôte Windows partagé + `-benchtime` borné (2x–10x) ⇒ tous les `ns/op` absolus sont **[probable]**, non [confirmé]. Les conclusions reposent prioritairement sur `allocs/op` et `B/op` (stables), les ratios relatifs, et le raisonnement de complexité.
- **`-race` indisponible** (`CGO_ENABLED=0`) : aucun constat de performance de cet axe n'en dépend ; les aspects concurrence sont du ressort de l'axe 2.
- **Comparaisons cross-environnement** vs `docs/audits/` : la baseline et les baselines DTM sont des snapshots **Windows / Core Ultra 9** ; la table de référence `docs/PERFORMANCE.md` est **Linux / Ryzen 9 5900X**. Les écarts inter-machines sont marqués **[à vérifier]**.
- **Artefacts produits** (autorisés par le mandat) : `audit/cpu.prof`, `audit/mem.prof` (profil `BenchmarkFibonacci`, benchtime=3x), `audit/bigfft_cpu.prof` (profil `fftmulTo_1M_words`, benchtime=5x).
- **Aucun invariant du dépôt n'a été contredit** : les constats A3-01/A3-02 touchant `bigfft/fft_cache.go` sont formulés en respect de l'invariant anti-aliasing `putByKey` (toute modification proposée passerait par une ADR).
