# Benchmark — parallélisation pointwise + butterflies (2026-06-09)

Hôte : Windows 11, 24 threads logiques, Go 1.26.4. Méthodologie : runs
appariés (même session, machine au repos), médianes ; `-count=5` pour les
benchs Go, 4 runs `Measure-Command` pour F(100M) (binaire `--algo fast`,
sans conversion décimale).

Changement mesuré : `runPointwise` (produits de coefficients fermat
parallélisés dans `PolValues.mul/sqr`) + `executeReconstruction`
parallélisé (butterflies), bornés par le sémaphore FFT global,
acquisition non bloquante, scratch pool par worker.

## `go test -bench` (médianes ns/op → ms)

| Benchmark | avant | après (étapes 1+2) | delta |
|---|---:|---:|---:|
| Fibonacci/FastDoubling/10M | 50,06 | 36,25 | −27,6 % |
| Fibonacci/FFTBased/10M | 50,85 | 39,22 | −22,9 % |
| Fibonacci/MatrixExp/10M | 49,29 | 42,38 | −14,0 % |
| FibonacciDTM/Off/10M | 49,67 | 32,36 | −34,8 % |
| FibonacciDTM/On/10M | 50,62 | 38,14 | −24,6 % |
| Fibonacci/FastDoubling/1M | 5,36 | 5,35 | −0,2 % |
| Fibonacci/FFTBased/1M | 6,58 | 6,15 | −6,6 % |
| Fibonacci/MatrixExp/1M | 9,13 | 9,29 | +1,8 % (bruit) |

Les benchs 1M sub-5ms ont une dispersion run-à-run de ±40 % sur cet hôte ;
les minima appariés DTM/Off/1M (3,18 vs 3,21) confirment l'absence de
régression réelle.

## F(100M) calcul seul (binaire, médiane de 4 runs)

| Variante | temps | delta vs HEAD |
|---|---:|---:|
| HEAD (séquentiel) | 0,379 s | — |
| + pointwise parallèle | 0,220 s | −42 % |
| + butterflies parallèles | 0,204 s | **−46 %** |

Utilisation CPU mesurée au profil (FastDoubling/10M) : 145 % → 224 % après
l'étape 1 ; `addMulVVWW` (noyau des produits de coefficients) reste le
poste dominant mais est désormais réparti sur les cœurs.

## Validation

- Golden tests : verts (aucune modification algorithmique de valeur).
- Suite complète + e2e : vertes.
- `-race` : exécuté via WSL (CGO indisponible sous Windows sans gcc).
- Gates : `pointwiseMinParallelWords = 1<<16`,
  `reconstructionMinParallelWords = 1<<16` — sous le gate, le chemin
  séquentiel historique est strictement préservé (mêmes allocations).
