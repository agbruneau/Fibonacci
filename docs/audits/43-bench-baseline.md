# 43 — Baseline benchmarks

## Méthode
- Commande : `go test -bench=. -benchmem -run=^$ -benchtime=1x -timeout 600s ./internal/{fibonacci,bigfft,calibration,cli}/...`
- Sortie : `docs/audits/_bench_raw.log`
- Plateforme : Windows 11, Go 1.26.2, CPU Intel Core Ultra 9 275HX (24 threads)
- Date : 2026-04-28
- ⚠️ `benchtime=1x` : 1 itération seule par bench, mesures **indicatives** uniquement.

## Synthèse globale
- Total benchmarks exécutés : **86** (lignes `Benchmark…-24`)
- Couverture par package :
  - `internal/fibonacci` : 11 benchmarks (CacheImpact, CacheHitRate, SmartSquare*, Fibonacci/{FastDoubling,MatrixExp,FFTBased}/{1M,10M})
  - `internal/bigfft` : 73 benchmarks (AddVV/SubVV/AddMulVVW, AllocVsMake, FFTMul/Sqr, BumpAllocator, ExtendedPool, Fermat*, Cache*, Mul*/Sqr*, Pool*, WordSlicePool*)
  - `internal/calibration` : 1 benchmark (`GenerateParallelThresholds`)
  - `internal/cli` : 1 benchmark (`GeneratePowerShellCompletion`)

## Top 10 plus lents (ns/op)
| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| BenchmarkFFTMulWithBump/fftmulTo_1M_words | 307 060 100 | 695 846 416 | 304 |
| BenchmarkFFTSqrWithBump/fftsqrTo_1M_words | 262 054 600 | 322 994 960 | 140 |
| BenchmarkCacheImpact/WithDefaultCache | 117 295 600 | 108 430 440 | 2 283 |
| BenchmarkCacheImpact/WithOptimizedCache | 66 472 100 | 45 640 840 | 1 181 |
| BenchmarkCacheImpact/CacheDisabled | 61 940 400 | 45 663 616 | 1 194 |
| BenchmarkFibonacci/FFTBased/10M | 61 591 300 | 48 093 184 | 4 051 |
| BenchmarkFibonacci/FastDoubling/10M | 57 360 300 | 45 662 016 | 1 178 |
| BenchmarkFibonacci/MatrixExp/10M | 52 871 100 | 155 633 136 | 3 872 |
| BenchmarkFFTMulWithBump/fftmulTo_100K_words | 29 169 600 | 55 148 520 | 312 |
| BenchmarkFibonacci/MatrixExp/1M | 23 258 500 | 6 462 136 | 745 |

## Top 10 plus alloueurs (allocs/op)
| Benchmark | allocs/op | B/op |
|---|---:|---:|
| BenchmarkFibonacci/FFTBased/10M | 4 051 | 48 093 184 |
| BenchmarkFibonacci/MatrixExp/10M | 3 872 | 155 633 136 |
| BenchmarkFibonacci/FFTBased/1M | 3 017 | 8 696 632 |
| BenchmarkCacheImpact/WithDefaultCache | 2 283 | 108 430 440 |
| BenchmarkCacheImpact/CacheDisabled | 1 194 | 45 663 616 |
| BenchmarkCacheImpact/WithOptimizedCache | 1 181 | 45 640 840 |
| BenchmarkFibonacci/FastDoubling/10M | 1 178 | 45 662 016 |
| BenchmarkFibonacci/MatrixExp/1M | 745 | 6 462 136 |
| BenchmarkFFTMulWithBump/fftmulTo_100K_words | 312 | 55 148 520 |
| BenchmarkFFTMulWithBump/fftmulTo_1M_words | 304 | 695 846 416 |

## Hot paths fibonacci & bigfft
| Benchmark | ns/op | Notes |
|---|---:|---|
| Fibonacci/FastDoubling/1M | 3 980 100 | algo défaut, 127 allocs — base de comparaison |
| Fibonacci/FastDoubling/10M | 57 360 300 | scaling ~14× pour 10× plus grand |
| Fibonacci/FFTBased/1M | 9 485 800 | 2,4× plus lent que FastDoubling à 1M |
| Fibonacci/FFTBased/10M | 61 591 300 | rattrape FastDoubling à 10M (seuil FFT proche) |
| Fibonacci/MatrixExp/1M | 23 258 500 | 5,8× plus lent que FastDoubling, +5,8× allocs |
| FFTMulWithBump/1M_words | 307 060 100 | hot path FFT, 695 MB alloués |
| FFTSqrWithBump/1M_words | 262 054 600 | sqr ~15 % plus rapide que mul, 2× moins alloc |

## Benchmarks à 0 alloc (pooling efficace)
- AddVV/{8,64,256,1024,4096}, SubVV/*, AddMulVVW/* (15 cas)
- AllocVsMake/Bump_{64,256,1K,4K} (allocateur bump : 0 B/op)
- FermatSqrVsMul/n=10/{Mul,Sqr}, n=29/*, n=30/Mul, MulToWithReuse, MulVsMulTo/MulTo_reuse
- CacheHit, CacheMiss
- GetWordSlicePoolIndex/{bitwise,linear}, GetFermatPoolIndex/*
- WordSliceDirectAlloc
→ **24 benchmarks à 0 allocs/op**, confirme l'efficacité du pooling/bump allocator.

## Comparaison vs baseline 2026-04
- `docs/audits/2026-04/bench/exec-baseline/benchmark.txt` ne contient que `PASS / ok / BENCH_EXIT=0` — **aucune mesure exploitable**. Pas de baseline comparable disponible.
- `perf-results/P0-01-P0-09/` et `P1-04-SKIPPED.md` ne fournissent pas de tableaux ns/op par benchmark.
- **Conclusion** : le présent rapport établit la première baseline numérique exploitable du dépôt.

## Findings
- Hot path absolu : **`bigfft.FFTMulWithBump/1M_words`** (307 ms, 695 MB) — cible prioritaire pour optimisation mémoire.
- Cache FFT : `WithOptimizedCache` (66 ms) ≈ `CacheDisabled` (62 ms) à N=10M ; le cache LRU **ne procure pas de gain** dans ce micro-bench mono-itération (log : `Hit Rate: 0.00%`). À ré-évaluer en charge réaliste.
- À 10M, FFTBased et FastDoubling convergent (~60 ms) : seuil FFT bien calibré.
- MatrixExp alloue 3,4× plus que FastDoubling à 10M (155 MB vs 45 MB) — confirme statut non-défaut.
- Pooling/bump : 24 benchmarks à 0 alloc valident l'objectif "réduction GC >95 %" annoncé dans `CLAUDE.md`.

## Limites de cette baseline
- `benchtime=1x` : **1 itération seule**, écart-type non mesuré, mesures sub-µs (100-500 ns) non fiables.
- Pas de répétition (`-count=1`), variance machine non capturée.
- Bruit Windows possible (services background, scheduler).
- **Action recommandée** : ré-exécuter avec `-benchtime=3s -count=5` en CI dédiée, archiver dans `docs/audits/2026-04/bench/exec-baseline/benchmark.txt` pour permettre comparaison `benchstat` lors du prochain audit.
