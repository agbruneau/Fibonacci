# Guide de Performance

> **Version** : 1.0.0  
> **Dernière mise à jour** : Novembre 2025

## Vue d'ensemble

Ce document décrit les techniques d'optimisation utilisées dans le Calculateur Fibonacci et fournit des conseils pour obtenir les meilleures performances sur votre matériel.

## Benchmarks de Référence

### Configuration de test

- **CPU** : AMD Ryzen 9 5900X (12 cores, 24 threads)
- **RAM** : 32 GB DDR4-3600
- **OS** : Linux 6.1
- **Go** : 1.25

### Résultats

| N | Fast Doubling | Matrix Exp. | FFT-Based | Résultat (chiffres) |
|---|---------------|-------------|-----------|---------------------|
| 1,000 | 15µs | 18µs | 45µs | 209 |
| 10,000 | 180µs | 220µs | 350µs | 2,090 |
| 100,000 | 3.2ms | 4.1ms | 5.8ms | 20,899 |
| 1,000,000 | 85ms | 110ms | 95ms | 208,988 |
| 10,000,000 | 2.1s | 2.8s | 2.3s | 2,089,877 |
| 100,000,000 | 45s | 62s | 48s | 20,898,764 |
| 250,000,000 | 3m12s | 4m25s | 3m28s | 52,246,909 |

> **Note** : Les temps varient selon le matériel. Utilisez `--calibrate` pour des mesures précises sur votre système.

## Optimisations Implémentées

### 1. Stratégie Zero-Allocation

#### Problème
Les calculs de Fibonacci pour de grands N créent des millions d'objets `big.Int` temporaires, causant une pression excessive sur le garbage collector.

#### Solution
Utilisation de `sync.Pool` pour recycler les états de calcul :

```go
var statePool = sync.Pool{
    New: func() interface{} {
        return &calculationState{
            f_k:  new(big.Int),
            f_k1: new(big.Int),
            t1:   new(big.Int),
            // ...
        }
    },
}

func acquireState() *calculationState {
    s := statePool.Get().(*calculationState)
    s.Reset()
    return s
}

func releaseState(s *calculationState) {
    statePool.Put(s)
}
```

#### Impact
- Réduction des allocations de 95%+
- Amélioration des performances de 20-30%
- Temps de pause GC réduits

### 2. Multiplication Adaptative (Karatsuba vs FFT)

#### Complexité comparative

| Algorithme | Complexité | Meilleur pour |
|------------|------------|---------------|
| Standard | O(n²) | Petits nombres |
| Karatsuba | O(n^1.585) | Nombres moyens |
| FFT | O(n log n) | Très grands nombres |

#### Seuil de basculement

Le paramètre `--fft-threshold` (défaut: 1,000,000 bits) contrôle quand la multiplication FFT est utilisée :

```go
func smartMultiply(z, x, y *big.Int, threshold int) *big.Int {
    if threshold > 0 {
        bx := x.BitLen()
        by := y.BitLen()
        if bx > threshold && by > threshold {
            return bigfft.MulTo(z, x, y)
        }
    }
    return z.Mul(x, y)
}
```

### 3. Parallélisme Multi-cœurs

#### Stratégie

Les trois multiplications principales de l'algorithme Fast Doubling sont parallélisées :

```go
func parallelMultiply3Optimized(s *calculationState, fftThreshold int) {
    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        s.t3 = smartMultiply(s.t3, s.f_k, s.t2, fftThreshold)
    }()
    go func() {
        defer wg.Done()
        s.t1 = smartMultiply(s.t1, s.f_k1, s.f_k1, fftThreshold)
    }()
    s.t4 = smartMultiply(s.t4, s.f_k, s.f_k, fftThreshold)
    wg.Wait()
}
```

#### Considérations

- **Seuil d'activation** : `--threshold` (défaut: 4096 bits)
- **Désactivation avec FFT** : Le parallélisme est désactivé quand FFT est utilisé car FFT sature déjà le CPU
- **Seuil FFT parallèle** : Réactivé au-dessus de 10 millions de bits

### 4. Algorithme de Strassen

Pour l'exponentiation matricielle, l'algorithme de Strassen réduit le nombre de multiplications de 8 à 7 :

```
Multiplication classique 2x2 : 8 multiplications
Strassen 2x2 : 7 multiplications + 18 additions
```

Activé via `--strassen-threshold` (défaut: 3072 bits) quand les éléments de la matrice sont suffisamment grands pour que l'économie de multiplications compense les additions supplémentaires.

### 5. Mise au Carré de Matrices Symétriques

Optimisation spécifique pour l'élévation au carré de matrices symétriques (où b = c) :

```go
// Carré classique : 8 multiplications
// Carré symétrique : 4 multiplications
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, inParallel bool, fftThreshold int) {
    a2 = smartMultiply(a2, mat.a, mat.a, fftThreshold)  // a²
    b2 = smartMultiply(b2, mat.b, mat.b, fftThreshold)  // b²
    d2 = smartMultiply(d2, mat.d, mat.d, fftThreshold)  // d²
    b_ad = smartMultiply(b_ad, mat.b, ad, fftThreshold) // b(a+d)
    
    dest.a.Add(a2, b2)    // a² + b²
    dest.b.Set(b_ad)      // b(a+d)
    dest.c.Set(b_ad)      // = dest.b (symétrie)
    dest.d.Add(b2, d2)    // b² + d²
}
```

## Guide de Tuning

### Calibration Automatique

```bash
# Calibration complète (recommandé pour production)
./fibcalc --calibrate

# Calibration rapide au démarrage
./fibcalc --auto-calibrate -n 100000000
```

La calibration teste différents seuils et détermine les valeurs optimales pour votre matériel.

### Paramètres de Configuration

| Paramètre | Défaut | Description | Ajustement |
|-----------|--------|-------------|------------|
| `--threshold` | 4096 | Seuil parallélisme (bits) | ↑ sur CPU lent, ↓ sur many-core |
| `--fft-threshold` | 1000000 | Seuil FFT (bits) | ↓ sur CPU avec cache L3 large |
| `--strassen-threshold` | 3072 | Seuil Strassen (bits) | ↑ si overhead d'additions visible |

### Recommandations par Type de Charge

#### Petits calculs (N < 10,000)

```bash
./fibcalc -n 5000 --threshold 0  # Désactiver le parallélisme
```

#### Calculs moyens (10,000 < N < 1,000,000)

```bash
./fibcalc -n 500000 --threshold 2048
```

#### Grands calculs (N > 1,000,000)

```bash
./fibcalc -n 10000000 --auto-calibrate
```

#### Très grands calculs (N > 100,000,000)

```bash
./fibcalc -n 250000000 --fft-threshold 500000 --timeout 30m
```

## Monitoring des Performances

### Mode Serveur

Le serveur expose des métriques sur `/metrics` :

```bash
curl http://localhost:8080/metrics
```

Métriques disponibles :
- `total_requests` : Nombre total de requêtes
- `total_calculations` : Nombre de calculs effectués
- `calculation_duration_*` : Distribution des durées par algorithme
- `errors_*` : Compteurs d'erreurs

### Profiling Go

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/

# Memory profiling
go test -memprofile=mem.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/

# Analyse
go tool pprof cpu.prof
```

## Comparaison des Algorithmes

### Fast Doubling

✅ **Avantages** :
- Le plus rapide pour la majorité des cas
- Parallélisation efficace
- Moins de multiplications que Matrix

⚠️ **Inconvénients** :
- Code plus complexe

### Matrix Exponentiation

✅ **Avantages** :
- Implémentation élégante et mathématiquement claire
- Optimisation Strassen efficace pour grands nombres

⚠️ **Inconvénients** :
- 8 multiplications par itération vs 3 pour Fast Doubling
- Plus lent en pratique

### FFT-Based

✅ **Avantages** :
- Force l'utilisation de FFT pour toutes les multiplications
- Utile pour benchmarking de FFT

⚠️ **Inconvénients** :
- Overhead significatif pour petits nombres
- Principalement utilisé pour tests

## Conseils d'Optimisation Avancée

### 1. Affinité CPU (Linux)

```bash
# Forcer l'utilisation de cœurs spécifiques
taskset -c 0-7 ./fibcalc -n 100000000
```

### 2. Désactiver le scaling de fréquence

```bash
# Mode performance
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### 3. GOMAXPROCS

```bash
# Limiter le nombre de threads Go
GOMAXPROCS=8 ./fibcalc -n 100000000
```

### 4. Compilation optimisée

```bash
# Build avec optimisations agressives
go build -ldflags="-s -w" -gcflags="-B" ./cmd/fibcalc
```

## Limites Connues

1. **Mémoire** : F(1 milliard) nécessite ~25 GB de RAM pour le résultat seul
2. **Temps** : Les calculs pour N > 500M peuvent prendre des heures
3. **FFT Contention** : L'algorithme FFT sature les cœurs, limitant le parallélisme externe

