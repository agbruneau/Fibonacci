# Comparaison des Algorithmes

> **Dernière mise à jour** : Novembre 2025

## Vue d'ensemble

Ce document compare les trois algorithmes de calcul de Fibonacci implémentés dans FibCalc.

## Algorithmes Disponibles

| Algorithme | Flag | Description |
|------------|------|-------------|
| Fast Doubling | `-algo fast` | Algorithme principal, le plus performant |
| Matrix Exponentiation | `-algo matrix` | Approche matricielle avec Strassen |
| FFT-Based | `-algo fft` | Force la multiplication FFT |

## Comparaison Théorique

### Complexité

Tous les algorithmes ont la même complexité asymptotique :

```
O(log n × M(n))
```

Où M(n) est le coût de multiplication de nombres de n bits.

Cependant, les constantes multiplicatives diffèrent :

| Algorithme | Multiplications par itération | Notes |
|------------|-------------------------------|-------|
| Fast Doubling | 3 | Minimum théorique |
| Matrix Exp. (classique) | 8 | Sans optimisation |
| Matrix Exp. (symétrique) | 4 | Élévation au carré optimisée |
| Matrix Exp. (Strassen) | 7 | Multiplication générale |

### Mémoire

| Algorithme | Variables temporaires | Pool objects |
|------------|----------------------|--------------|
| Fast Doubling | 6 big.Int | calculationState |
| Matrix Exp. | 3 matrices + ~22 big.Int | matrixState |

## Benchmarks

### Configuration de Test

```
CPU: AMD Ryzen 9 5900X (12 cores)
RAM: 32 GB DDR4-3600
Go: 1.25
OS: Linux 6.1
```

### Résultats (temps moyens sur 10 exécutions)

#### Petits N (N ≤ 10,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 100 | 1.2µs | 1.5µs | 8.5µs |
| 1,000 | 15µs | 18µs | 45µs |
| 10,000 | 180µs | 220µs | 350µs |

**Gagnant** : Fast Doubling (3-4× plus rapide que FFT-Based)

#### N Moyens (10,000 < N ≤ 1,000,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 100,000 | 3.2ms | 4.1ms | 5.8ms |
| 500,000 | 35ms | 48ms | 42ms |
| 1,000,000 | 85ms | 110ms | 95ms |

**Gagnant** : Fast Doubling, mais écart réduit avec FFT-Based

#### Grands N (N > 1,000,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 5,000,000 | 850ms | 1.15s | 920ms |
| 10,000,000 | 2.1s | 2.8s | 2.3s |
| 50,000,000 | 18s | 25s | 19s |
| 100,000,000 | 45s | 62s | 48s |

**Gagnant** : Fast Doubling de justesse (FFT-Based très proche)

#### Très Grands N (N > 100,000,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 250,000,000 | 3m12s | 4m25s | 3m28s |
| 500,000,000 | 8m45s | 12m10s | 9m15s |

**Gagnant** : Fast Doubling (toujours ~10% plus rapide)

## Graphique de Performance

```
Temps (log)
    │
  1h├                                    /
    │                                   / ← Matrix
    │                                  /
 10m├                              /  /
    │                             / /
    │                            /╱  ← FFT-Based
  1m├                         /╱╱
    │                       ╱╱╱
    │                     ╱╱╱ ← Fast Doubling
 10s├                  ╱╱╱
    │               ╱╱╱
    │            ╱╱╱
  1s├         ╱╱╱
    │      ╱╱╱
    │   ╱╱╱
100ms├╱╱╱
    └─────┬─────┬─────┬─────┬─────┬─────
        10K   100K    1M   10M  100M    N
```

## Quand Utiliser Chaque Algorithme

### Fast Doubling (`-algo fast`)

✅ **Recommandé pour** :
- Usage général (défaut)
- Performance maximale
- Tous les ordres de grandeur de N

```bash
./fibcalc -n 10000000 -algo fast
```

### Matrix Exponentiation (`-algo matrix`)

✅ **Recommandé pour** :
- Compréhension pédagogique
- Vérification croisée des résultats
- Quand vous voulez tester l'algorithme de Strassen

```bash
./fibcalc -n 10000000 -algo matrix --strassen-threshold 2048
```

### FFT-Based (`-algo fft`)

✅ **Recommandé pour** :
- Benchmarking de la multiplication FFT
- Tests de très grands nombres (N > 100M)
- Comparaison des performances FFT vs Karatsuba

```bash
./fibcalc -n 100000000 -algo fft
```

## Comparaison Complète

```bash
# Comparer tous les algorithmes sur un même N
./fibcalc -n 10000000 -algo all -d
```

Sortie typique :

```
=== Execution Configuration ===
Calculating F(10000000) with a timeout of 5m0s.
Environment: 24 logical processors, Go go1.25.
Optimization thresholds: Parallelism=4096 bits, FFT=1000000 bits.
Execution mode: Parallel comparison of all algorithms.

=== Comparison Summary ===
Algorithm                                    Duration    Status
Fast Doubling (O(log n), Parallel, Zero-Alloc)   2.1s       ✅ Success
FFT-Based Doubling (O(log n), FFT Mul)           2.3s       ✅ Success
Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)   2.8s       ✅ Success

=== All algorithms succeeded ===
Result binary size: 6,942,420 bits.
```

## Recommandations de Configuration

### Pour Petits Calculs (N < 100,000)

```bash
./fibcalc -n 50000 -algo fast --threshold 0 --fft-threshold 0
```

- Désactiver le parallélisme (overhead > gains)
- Désactiver FFT (trop petit)

### Pour Calculs Moyens (100,000 < N < 10,000,000)

```bash
./fibcalc -n 5000000 -algo fast --threshold 4096
```

- Parallélisme activé
- FFT pour les opérations les plus grandes

### Pour Grands Calculs (N > 10,000,000)

```bash
./fibcalc -n 100000000 -algo fast --auto-calibrate --timeout 30m
```

- Auto-calibration pour seuils optimaux
- Timeout étendu

## Conclusion

| Critère | Fast Doubling | Matrix Exp. | FFT-Based |
|---------|---------------|-------------|-----------|
| **Performance** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Mémoire** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Simplicité** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Pédagogie** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ |

**Recommandation générale** : Utilisez **Fast Doubling** (`-algo fast`) pour tous les cas d'usage, sauf besoins spécifiques de tests ou comparaisons.

