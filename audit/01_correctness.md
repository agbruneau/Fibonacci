# 01 — Correctness & exactitude algorithmique

## Verdict d'axe

**Les implémentations Fibonacci sont algorithmiquement saines : aucun écart de résultat détecté.** Les golden tests (3 calculateurs × N jusqu'à 200 000), les 5 cibles fuzz `fibonacci` + 2 cibles `bigfft` (toutes PASS, ~30 s/cible, zéro crasher), les tests de propriété (Cassini, récurrence, doublement, GCD) et la cross-validation Fast Doubling ↔ Matrix ↔ FFT couvrent solidement le cœur. **Le risque résiduel est de couverture, pas de logique** : le chemin FFT à l'échelle de production (opérandes de plusieurs millions de bits, régime du `n=100M` par défaut) n'est jamais recoupé contre un oracle indépendant ; le backend GMP n'est validé que jusqu'à F(100) et reste non vérifiable dans ce sandbox (CGO désactivé). Quelques débordements entiers silencieux existent dans l'estimation mémoire mais ne sont atteignables qu'à des `n` physiquement incalculables.

## Tableau récapitulatif

| ID | Sévérité | Titre | Marqueur |
|---|---|---|---|
| A1-01 | MAJEUR | Aucun oracle de non-régression sur le chemin FFT à l'échelle production (>500k bits) | [confirmé] |
| A1-02 | MAJEUR | Backend GMP : exactitude non vérifiable + couverture limitée à F(100), jamais recoupée | [à vérifier] |
| A1-03 | MINEUR | `FuzzFastDoublingMod` ne recoupe pas le résultat modulaire contre un oracle | [confirmé] |
| A1-04 | MINEUR | Débordement uint64 silencieux dans `EstimateMemoryUsage` (garde-fou mémoire) | [confirmé] |
| A1-05 | MINEUR | Débordement int potentiel dans le dimensionnement d'arène (`AcquireStateForN`) | [probable] |
| A1-06 | INFORMATIF | `usedFFT`/`bitLen` (métriques DTM) calculés sur FK, décision réelle sur FK1 | [confirmé] |
| A1-07 | INFORMATIF | `smartMultiply` n'active FFT que si les DEUX opérandes dépassent le seuil | [confirmé] |
| A1-08 | INFORMATIF | `TestExecuteDoublingStepFFT` n'assertit que l'absence d'erreur, pas la valeur | [confirmé] |

---

## Détail des constats

### [A1-01] Aucun oracle de non-régression sur le chemin FFT à l'échelle production (>500k bits)
- **Sévérité** : MAJEUR
- **Axe** : 1 Correctness
- **Emplacement** : `internal/fibonacci/fibonacci_golden_test.go:44-60` ; `internal/fibonacci/fibonacci_fuzz_test.go:84-120` ; `internal/bigfft/fft_precision_test.go:84-100,501-502`
- **Preuve** :
  Golden file : N maximal = 200 000 (≈ 138 700 bits, sous le `DefaultFFTThreshold = 500_000`). Aucun cas golden n'atteint le régime FFT par défaut.
  ```
  --- PASS: TestCalculatorsAgainstGoldenFile/MatrixExp/N=200000 (0.00s)
  ```
  La cross-validation FFT du fuzz force le seuil à 0 (régime différent du défaut) et plafonne à F(200 000) :
  ```go
  // fibonacci_fuzz_test.go:87,94
  if n > 200_000 { return }
  opts := Options{ ParallelThreshold: DefaultParallelThreshold, FFTThreshold: 0 } // Force FFT usage for testing
  ```
  L'oracle bas-niveau `bigfft.Mul` vs `math/big.Mul` plafonne à 10 000 octets (80 000 bits) :
  ```go
  // fft_precision_test.go:501-502
  aBytes := make([]byte, 10000)
  bBytes := make([]byte, 10000)
  ```
- **Impact** : Le chemin le plus utilisé en production (CLI `-n` défaut = 100 000 000, soit ≈ 69,4 M bits, intégralement en régime FFT Schönhage-Strassen) n'a aucun garde-fou numérique recoupant un résultat de référence. Une régression dans le dimensionnement FFT (`GetFFTParams`, `ValueSize`), l'anneau de Fermat ou la transformée inverse qui ne se manifesterait qu'au-delà de ~138k bits passerait inaperçue. La validité repose sur l'hypothèse d'invariance en taille de l'algorithme — non garantie pour une FFT (le nombre d'étages, le choix de `k`/`m` et les marges anti-débordement varient avec la taille).
- **Recommandation** : Ajouter un test (hors golden immuable, donc pas une extension du JSON) qui recoupe un calculateur contre un autre à au moins un N franchissant le seuil FFT par défaut (p. ex. F(1 000 000) ≈ 694k bits : Fast Doubling défaut vs Matrix défaut, ou vs `FastDoublingMod` sur les K derniers chiffres). Coût ~quelques centaines de ms, acceptable hors `-short`. Alternativement, étendre l'oracle `bigfft` (`fft_precision_test.go`) à un opérande > 600k bits.
- **Marqueur** : [confirmé]

---

### [A1-02] Backend GMP : exactitude non vérifiable + couverture limitée à F(100), jamais recoupée
- **Sévérité** : MAJEUR
- **Axe** : 1 Correctness
- **Emplacement** : `internal/fibonacci/calculator_gmp.go:74-153` ; `internal/fibonacci/calculator_gmp_test.go:19-48`
- **Preuve** :
  Build GMP impossible dans le sandbox (CGO_ENABLED=0, libgmp absente) — le package `ncw/gmp` se réduit à un stub vide :
  ```
  $ go build -tags gmp ./internal/fibonacci/
  internal\fibonacci\calculator_gmp.go:74:40: undefined: gmp.Int
  internal\fibonacci\calculator_gmp.go:118:11: undefined: gmp.NewInt
  ...
  ```
  Le seul test GMP plafonne à F(100) et n'utilise que des valeurs codées en dur, sans recoupement avec les autres calculateurs ni le golden file :
  ```go
  // calculator_gmp_test.go:30-32
  {50, "12586269025"},
  {100, "354224848179261915075"},
  ```
  Le calculateur GMP est pourtant enregistré dans la factory globale (`init()` → `RegisterGMPCalculator(globalFactory)`, `calculator_gmp.go:35-37`), donc exposé comme algorithme « gmp » exécutable.
- **Impact** : L'exactitude de GMP aux grandes tailles (sa raison d'être : N > 10^8) n'est garantie par aucun test in-tree. La revue statique de `gmpDoublingStep` est correcte (l'aliasing `a = b*b` après mise à l'abri de F(2k) dans `t1` est valide, l'identité F(2k)=a·(2b−a), F(2k+1)=a²+b² est respectée), mais aucune exécution ne le confirme et aucune cross-validation GMP↔FastDoubling n'existe à grande échelle.
- **Recommandation** : (1) Sous Linux/WSL avec `libgmp-dev` + `CGO_ENABLED=1`, exécuter `go test -tags gmp ./internal/fibonacci/` pour lever le doute d'exactitude. (2) Ajouter dans `calculator_gmp_test.go` une cross-validation GMP vs FastDoubling à au moins un N franchissant le seuil FFT (p. ex. F(500 000)), sous build tag `gmp`. La revue statique seule ne suffit pas pour un backend arithmétique.
- **Marqueur** : [à vérifier] — exactitude GMP à reconfirmer sous Linux/WSL avec `-tags gmp` et libgmp installée.

---

### [A1-03] `FuzzFastDoublingMod` ne recoupe pas le résultat modulaire contre un oracle
- **Sévérité** : MINEUR
- **Axe** : 1 Correctness
- **Emplacement** : `internal/fibonacci/fibonacci_fuzz_test.go:291-306`
- **Preuve** :
  ```go
  result, err := FastDoublingMod(n, m)
  if err != nil { t.Fatalf("error: %v", err) }
  if result.Sign() < 0 || result.Cmp(m) >= 0 {
      t.Errorf("result %s out of range [0, %s)", result, m)
  }
  ```
  Le fuzz ne vérifie QUE l'appartenance à `[0, m)`. Il ne compare jamais `FastDoublingMod(n, m)` à une valeur de référence (p. ex. `FastDoubling(n) mod m`). Un résultat dans la bonne plage mais numériquement faux passerait.
- **Impact** : Limité — `TestFastDoublingMod_ConsistentWithFull` (`modular_test.go:42-65`) effectue ce recoupement pour le cas fixe N=500/mod=10^100, et `TestFastDoublingMod_KnownValues` ancre 5 valeurs. Le trou est donc seulement la généralisation aléatoire du recoupement, pas une absence totale de validation.
- **Recommandation** : Dans la cible fuzz, lorsque `n <= 10000`, calculer `FastDoubling(n) mod m` et asserter l'égalité avec `FastDoublingMod(n, m)`. Recoupement à coût négligeable qui transforme un simple test de plage en test d'exactitude.
- **Marqueur** : [confirmé]

---

### [A1-04] Débordement uint64 silencieux dans `EstimateMemoryUsage` (garde-fou mémoire)
- **Sévérité** : MINEUR
- **Axe** : 1 Correctness
- **Emplacement** : `internal/fibonacci/memory/budget.go:19-38`
- **Preuve** :
  ```go
  func EstimateMemoryUsage(n uint64) MemoryEstimate {
      bitsPerFib := float64(n) * 0.69424
      wordsPerFib := int(bitsPerFib/64) + 1
      bytesPerFib := uint64(wordsPerFib) * 8
      stateBytes := bytesPerFib * 5
      fftBytes := bytesPerFib * 3
      cacheBytes := bytesPerFib * 2
      overheadBytes := stateBytes
      total := stateBytes + fftBytes + cacheBytes + overheadBytes // facteur 15×
  ```
  `total` ≈ `bytesPerFib × 15`. Pour `n` proche de `uint64` max (~1,8·10^19), `bytesPerFib` ≈ 1,6·10^18 et `total` ≈ 2,4·10^19, ce qui **dépasse uint64 max (≈ 1,8·10^19)** et wrappe silencieusement vers une petite valeur.
- **Impact** : La vérification de budget mémoire (`CanCalculate` → `est.TotalBytes <= memLimitBytes`, `calculator.go:171-176`) pourrait accepter une requête au `n` gigantesque dont l'estimation a wrappé à une valeur minuscule, contournant la protection. En pratique non exploitable : un tel `n` est physiquement incalculable (et `int(bitsPerFib/64)` est lui-même proche de la limite int64). Risque théorique de defence-in-depth, pas un bug atteignable sur charge réelle.
- **Recommandation** : Saturer l'arithmétique (détecter le débordement et retourner `math.MaxUint64` pour `TotalBytes`), ou borner explicitement `n` en amont dans la validation de config avec un message clair. Documenter la borne pratique de `n`.
- **Marqueur** : [confirmé]

---

### [A1-05] Débordement int potentiel dans le dimensionnement d'arène (`AcquireStateForN`)
- **Sévérité** : MINEUR
- **Axe** : 1 Correctness
- **Emplacement** : `internal/fibonacci/fastdoubling.go:287-291` ; `internal/fibonacci/memory/arena.go:25-36`
- **Preuve** :
  ```go
  // fastdoubling.go:288-291
  estimatedBits := int(float64(n) * FibonacciGrowthFactor)
  wordsNeeded := estimatedBits/64 + 1
  totalWords := wordsNeeded * 15
  ```
  `int(float64(n) * 0.69424)` : pour `n` > ~1,3·10^19, le produit `float64` dépasse `math.MaxInt64`, rendant la conversion `int(...)` indéfinie (Go spec : conversion float→int hors plage = comportement d'implémentation). `totalWords := wordsNeeded * 15` peut aussi déborder int pour des `n` extrêmes.
- **Impact** : Atteignable uniquement à des `n` physiquement incalculables (le défaut CLI est 10^8, et même 10^12 reste très en-deçà du seuil de débordement). Aucune charge réaliste ne le déclenche. Cohérent avec A1-04 : même classe de fragilité aux `n` non bornés.
- **Recommandation** : Borner `n` en amont (validation de config) avec un plafond documenté, OU clamper `estimatedBits`/`totalWords`. Comme pour A1-04, le correctif naturel est une borne supérieure unique sur `n` partagée par tous les estimateurs.
- **Marqueur** : [probable] — débordement non exécuté (n non atteignable dans les tests), déduit de l'arithmétique.

---

### [A1-06] `usedFFT`/`bitLen` (métriques DTM) calculés sur FK, décision FFT réelle sur FK1
- **Sévérité** : INFORMATIF
- **Axe** : 1 Correctness
- **Emplacement** : `internal/fibonacci/doubling_framework.go:180-183,209` ; `internal/fibonacci/strategy.go:108-115`
- **Preuve** :
  ```go
  // doubling_framework.go:180-183 — métriques
  fkBitLen := s.FK.BitLen()
  bitLen := fkBitLen
  usedFFT := bitLen > currentOpts.FFTThreshold
  // ... dtm.RecordIteration(bitLen, iterDuration, usedFFT, shouldParallel)
  ```
  ```go
  // strategy.go:110 — décision FFT RÉELLE
  if opts.FFTThreshold > 0 && state.FK1.BitLen() > opts.FFTThreshold {
      return executeDoublingStepFFT(ctx, state, opts, inParallel)
  }
  ```
  La métrique `usedFFT` se base sur `FK` (= F(k)), tandis que le routage effectif vers FFT se base sur `FK1` (= F(k+1)). Comme F(k+1) > F(k) (d'environ 1 bit), à l'itération de bascule la métrique peut signaler `usedFFT=false` alors que l'exécution a réellement emprunté la voie FFT (ou inversement), avec un décalage d'au plus une itération.
- **Impact** : Aucun sur l'exactitude du résultat (la métrique n'alimente que le `DynamicThresholdManager`). Effet de second ordre possible sur l'auto-ajustement des seuils dynamiques (feature opt-in, `EnableDynamicThresholds`), négligeable.
- **Recommandation** : Aligner la métrique sur l'opérande réellement testé (`FK1`) pour cohérence, ou documenter explicitement que `usedFFT` est une approximation. Pas de correctif fonctionnel requis.
- **Marqueur** : [confirmé]

---

### [A1-07] `smartMultiply` n'active FFT que si les DEUX opérandes dépassent le seuil
- **Sévérité** : INFORMATIF
- **Axe** : 1 Correctness (impact réel : axe 3 Performance)
- **Emplacement** : `internal/fibonacci/fft.go:48-63`
- **Preuve** :
  ```go
  bx := x.BitLen()
  by := y.BitLen()
  // Tier 1: FFT Multiplication for very large operands
  if fftThreshold > 0 && bx > fftThreshold && by > fftThreshold {
      return bigfft.MulTo(z, x, y)
  }
  // Tier 2: math/big Multiplication
  return z.Mul(x, y), nil
  ```
  Le `&&` impose que les deux opérandes franchissent le seuil. `smartSquare` (un seul opérande) n'a pas ce problème. Dans le doublement Fibonacci, FK et FK1 diffèrent d'~1 bit, donc sans effet. Mais `smartMultiply` sert aussi la multiplication matricielle Strassen (`common.go:181-185`, `matrix_ops.go`), où des opérandes asymétriques sont possibles (p. ex. produit d'un grand par un petit dans les scommes intermédiaires).
- **Impact** : Aucun sur l'exactitude — `math/big.Mul` est toujours correct. Conséquence purement performance : un produit où un seul opérande est très grand n'emprunte pas FFT et reste en Karatsuba. Marginal en pratique (les opérandes matriciels Fibonacci restent de tailles comparables).
- **Recommandation** : À évaluer côté axe Performance — un critère `max(bx,by) > seuil` (comme `shouldParallelizeMultiplicationCached` qui utilise déjà `maxBitLen`) serait plus cohérent. Aucune action de correctness.
- **Marqueur** : [confirmé]

---

### [A1-08] `TestExecuteDoublingStepFFT` n'assertit que l'absence d'erreur, pas la valeur
- **Sévérité** : INFORMATIF
- **Axe** : 1 Correctness
- **Emplacement** : `internal/fibonacci/fft_test.go:9-78`
- **Preuve** :
  ```go
  err := executeDoublingStepFFT(context.Background(), state, opts, false)
  if err != nil {
      t.Errorf("executeDoublingStepFFT returned unexpected error: %v", err)
  }
  // aucune assertion sur state.T1/T2/T3
  ```
  Les trois sous-tests vérifient uniquement `err == nil` ; aucune comparaison des produits FFT (T1=FK1², T2=FK², T3=FK·FK1) contre une référence `math/big`.
- **Impact** : Faible — la valeur du pas FFT est indirectement validée bout-en-bout par `FuzzFFTBasedConsistency` et les property tests. Mais ce test unitaire ciblé donne une fausse impression de couverture numérique du pas FFT.
- **Recommandation** : Renforcer les assertions en comparant `state.T1/T2/T3` à `new(big.Int).Mul(...)` de référence après l'appel. Améliore la localisation d'une régression future au niveau du pas FFT plutôt qu'au niveau bout-en-bout.
- **Marqueur** : [confirmé]

---

## Synthèse des exécutions

- `go test ./internal/fibonacci/... -run . -count=1` → **PASS** (4 packages).
- `TestCalculatorsAgainstGoldenFile` → **PASS** (FastDoubling/MatrixExp/FFTBased × N ∈ {0..200000}).
- Property tests (Cassini, récurrence, doublement, GCD(F(m),F(n))=F(GCD(m,n))) → **PASS** sur les 3 calculateurs.
- Fuzz (30 s/cible, zéro crasher) : `FuzzFastDoublingConsistency`, `FuzzFFTBasedConsistency`, `FuzzFibonacciIdentities`, `FuzzFastDoublingMod`, `FuzzProgressMonotonicity` → **PASS**.
- Bonus fuzz `bigfft` (20 s/cible) : `FuzzMul`, `FuzzSqr` → **PASS**.
- `go build -tags gmp ./internal/fibonacci/` → **ÉCHEC** (CGO_ENABLED=0, libgmp absente) — exactitude GMP marquée [à vérifier].
- `go vet ./internal/fibonacci/...` → **propre** (exit 0).
- Couverture `internal/fibonacci` : **87,7 %** (cœur algorithmique : `AcquireStateForN` 100 %, `FastDoublingMod` 96,6 %, `squareSymmetricMatrix` 100 %, `multiplyMatrixStrassen` 90 %, `executeDoublingStepFFT` 85,7 %).
- Artefact : `audit/cover_fib.out`.
