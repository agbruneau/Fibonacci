# Benchmark — boucle d'audit 2026-06-10 (cache state/arène + bump F-012)

Hôte : Windows 11, Intel Core Ultra 9 275HX (24 threads), Go 1.26.4 windows/amd64.
Méthodologie : `go test -bench=Benchmark -benchmem -run='^$' -benchtime=2s -count=6
./internal/fibonacci/`, comparaisons `benchstat` (n=6, significatif si p<0.05).
Baseline capturée le 2026-06-10 après le commit `4e34b82` (TestMain : sans lui, les
logs zerolog trace interfoliés rendaient la sortie inanalysable par benchstat).
Sanité environnement validée contre `bench-parallel-pointwise-2026-06.md`
(FastDoubling/10M : 33,30 ms mesuré vs 36,25 ms référence du 2026-06-09).

## Changements mesurés

1. **`fa13bfd` — cache state+arène par calculateur.** Le pattern GC-disable/re-enable
   (`memory.GCController`) force une collection après chaque calcul ≥ 1M ; cette
   collection purge les `sync.Pool`, donc `statePool` ne retenait jamais l'arène
   entre deux appels : ~46 % des allocations à F(10M) étaient la recréation
   d'arène (profil mem : 3,89 GB / ~155 ops). Correctif : slot mono-état GC-immune
   par instance (`FastDoublingCalculator.cachedState`, `atomic.Pointer`), borné par
   `maxCachedArenaWords` (4M mots ≈ 32 Mo), repli `sync.Pool`, chemin de teardown
   unique préservé (`finalizeStateReleaseTo`, ordre checkLimit → clearStateAliases
   → sink).
2. **`7999c39` — bump FFT acquis une fois par calcul (F-012, audit 2026-05-29).**
   Le `BumpAllocator` des transforms forward était acquis/relâché à chaque pas de
   doubling et regrossissait presque à chaque itération. Il est désormais
   dimensionné pour le pas final (`s.fftBumpCapWords`), porté par le
   `CalculationState` et `Reset()` entre les pas ; rétention alignée sur la
   politique anti-bloat de l'arène.

## benchstat cumulé (baseline → après les deux correctifs)

| Benchmark (sec/op) | avant | après | delta (p=0.002) |
|---|---:|---:|---:|
| Fibonacci/FastDoubling/10M | 33,30 ms | 28,20 ms | **−15,3 %** |
| Fibonacci/MatrixExp/10M | 38,40 ms | 27,86 ms | −27,5 % |
| Fibonacci/FFTBased/10M | 34,63 ms | 30,78 ms | −11,1 % |
| FibonacciDTM/Off/10M | 33,99 ms | 26,33 ms | −22,6 % |
| FibonacciDTM/On/10M | 33,94 ms | 26,52 ms | −21,9 % |
| CacheImpact/* (3 benchs) | 32,6–34,6 ms | 21,2–23,5 ms | −32 à −35 % |
| Fibonacci/*/1M, DTM */1M, SmartSquare* | — | — | ~ (neutres, p>0.05) |
| **geomean sec/op** | 1,680 ms | 1,478 ms | **−12,0 %** |

Mémoire (B/op, chemins fast doubling) : 24,6→13,5 Mi après `fa13bfd` (−45 %), puis
13,6→7,25 Mi après `7999c39` (−46 % supplémentaires) — soit ~−70 % cumulés à 10M ;
−61 % à 1M. Aucune régression sec/op significative au cumul (les +1-3,5 %
intermédiaires sur les benchs 1M/SmartSquare relevaient du bruit : dispersion
historique ±40 % documentée pour les benchs 1M sub-5 ms sur cet hôte, code de
smartSquare inchangé).

## Candidats examinés et rejetés

- **Garde `kb != 0` autour du `shlVU` de `fermat.Shift`** (2,17 s/24,5 s au profil
  CPU 10M) : réfuté par lecture avant tout cycle — `math/big.shlVU` court-circuite
  déjà s==0 par un `copy(z, x)` qui, in-place, touche le fast-path memmove
  src==dst (O(1)). Les `lshVU` mesurés sont des décalages réels nécessaires.
- **Parallélisation des deux transforms forward de `executeDoublingStepFFT`** :
  non retenu — gain borné (les transforms forward ne dominent pas le profil :
  la reconstruction et le pointwise, déjà parallélisés en 2026-06, pèsent
  davantage), et le bump par état est mono-goroutine (il faudrait deux scratchs).
- Rappels de périmètre : `Poly.IntTo` (= F-004, prémisse réfutée, ne pas toucher),
  candidats R1–R7 d'ADR-0008 (fermés), backlog ADR-0004 (B1/B2 wont-fix).

## Validation

- Golden tests : verts (aucune modification de valeur algorithmique).
- Tests gardiens verts : `TestReleaseState_OverLimit_AliasesCleared`,
  `TestPointwiseWorkerPanicPropagates`, `TestPointwiseParallelMatchesSequential`,
  `TestFermatPostConditionPanicClassifier`, `TestWireThresholdTuning`,
  `TestArchitectureLayering`, + 8 nouveaux gardiens du cache d'état/bump
  (`state_cache_test.go`).
- **`-race` exécuté sur cet hôte via WSL** (go1.26.0 linux/amd64) sur
  `./internal/fibonacci/ ./internal/bigfft/` : PASS (protocole Swap/CAS du slot
  inclus). Revue adversariale multi-agents (3 lentilles) : aliasing approve,
  concurrence approve, conformité-repo concerns (0 critical/major ; findings
  traités par durcissement de 2 tests, le reste routé vers les Phases 3–4).
- `golangci-lint` : zéro nouvelle issue vs la baseline de 151 issues
  préexistantes (lint advisory — politique du dépôt, cf. `scripts/check.ps1` et
  F-009 de `audit-2026-05-29.md`).

Cette page constitue la **baseline de non-régression** courante pour
`internal/fibonacci/` : comparer les benchs futurs aux colonnes « après »
ci-dessus (mêmes flags, même hôte de référence).
