# UltraReview — FibGo (FibCalc)

**Audit exhaustif du code source en vue d'un refactoring complet**

| Champ | Valeur |
|---|---|
| Repository | `github.com/agbru/fibcalc` |
| Branche analysée | `main` |
| Commit de référence | `3b0da02` (« Audit terminé ») |
| Date de l'audit | 2026-05-09 |
| Périmètre | 21 packages internes + 2 binaires CLI, ~36 500 LOC source + ~20 400 LOC tests |
| Méthodologie | Lecture intégrale fichier par fichier via 6 agents parallèles, plus une analyse transverse |
| Auditeur | Claude Opus 4.7 (1 M context) |

---

## Sommaire

1. [Sommaire exécutif](#1-sommaire-exécutif)
2. [Méthodologie & périmètre](#2-méthodologie--périmètre)
3. [Vue d'ensemble architecturale](#3-vue-densemble-architecturale)
4. [Audit détaillé par sous-système](#4-audit-détaillé-par-sous-système)
   - 4.1 [`internal/fibonacci/` — cœur algorithmique](#41-internalfibonacci--cœur-algorithmique)
   - 4.2 [`internal/bigfft/` — multiplication FFT Schönhage-Strassen](#42-internalbigfft--multiplication-fft-schönhage-strassen)
   - 4.3 [`memory/`, `threshold/`, `calibration/` — ressources & tuning](#43-memory-threshold-calibration--ressources--tuning)
   - 4.4 [`orchestration/`, `progress/`, `parallel/` — concurrence & rapports](#44-orchestration-progress-parallel--concurrence--rapports)
   - 4.5 [`cli/`, `tui/`, `ui/`, `format/` — couches de présentation](#45-cli-tui-ui-format--couches-de-présentation)
   - 4.6 [`cmd/`, `app/`, `config/`, `errors/`, `metrics/`, `sysmon/`, `testutil/` — couches de bord](#46-cmd-app-config-errors-metrics-sysmon-testutil--couches-de-bord)
5. [Constats transverses](#5-constats-transverses)
6. [Roadmap de refactoring (priorisée)](#6-roadmap-de-refactoring-priorisée)
7. [Risques & trade-offs](#7-risques--trade-offs)
8. [Annexes](#8-annexes)

---

## 1. Sommaire exécutif

FibGo est une **codebase Go mature, bien structurée et bien documentée**. Le respect de Clean Architecture est réel, les conventions sont uniformes (un `doc.go` par package, interfaces étroites, erreurs structurées avec `%w`), et la stratégie de tests est solide (~87 % de couverture, 99 fichiers de test, golden + fuzz + property + E2E). Les patterns de performance critiques (pool de `big.Int`, allocateur bump pour FFT, GC controller, parallélisme adaptatif) sont en place et fonctionnels.

Cependant, l'audit identifie **4 catégories de dette technique** qui justifient un refactoring :

1. **Bug latent de sécurité mémoire dans le pooling state+arena** (P1-04). `clearStateAliases` peut être contournée selon la branche d'erreur, ce qui peut laisser une `CalculationState` avec des aliases vivants pointant vers une arena rendue à la pool. Sévérité **CRITIQUE**.

2. **Duplications structurelles entre packages logiquement reliés** :
   - 3 abstractions de progression (`internal/progress/`, `internal/orchestration/progress.go`, `internal/format/progress_eta.go`) et un fichier d'alias (`internal/fibonacci/progress_aliases.go`).
   - 3 surfaces couleur/style (`ui/colors.go`, `ui/themes.go`, `tui/styles.go`).
   - 3 façades de présentation CLI (`cli/presenter.go`, `cli/ui_display.go`, `cli/output.go`) à frontières floues.
   - Duplication interne dans `bigfft/` entre `fft.go::fftmulTo` et `fft_poly.go::Poly.mul`, et entre les versions parallèle / séquentielle de `executeFFTTransforms`.

3. **Monolithes fonctionnels** dépassant les seuils du linter (gocyclo 15, funlen 100/50) malgré les `//nolint:gocognit` :
   - `internal/fibonacci/threshold/manager.go` (417 L, 7 responsabilités).
   - `internal/fibonacci/doubling_framework.go::ExecuteDoublingLoop` (137 L, gère doubling + progress + seuils dynamiques + cache FFT + métriques).
   - `internal/tui/model.go` (425 L, 16 types de messages dans un `Update` unique).
   - `internal/cli/completion.go` (520 L, 4 implémentations shell mêlées).
   - `internal/bigfft/fft_cache.go` (534 L) et `bigfft/pool.go` (467 L).

4. **Couplages cachés et globals** qui freinent la testabilité :
   - `bigfft.globalTransformCache`, `bigfft.concurrencySemaphore`, 13 pools globaux non réinitialisables.
   - `DoublingFramework` appelle directement `bigfft.GetTransformCache()` dans la boucle critique.
   - `GCController` est instancié dans `Calculator` (responsabilité orthogonale).
   - `internal/parallel.ErrorCollector` — code mort (zéro caller hors tests).

**Décisions à prendre :** trois fixes critiques (R1.1 leak alias, R1.2 panic-safe GC, R1.3 invalidation profil) sont à shipper rapidement (<½ jour). Le reste constitue une feuille de route progressive de 4 à 6 semaines, qui peut être segmentée en 4 vagues sans casser la compatibilité.

**Gain attendu** : ~2 000 LOC supprimées, ~30 % de réduction de complexité cyclomatique cumulée, testabilité significativement meilleure (notamment sur `bigfft/`), latency end-to-end -5 à -10 % (fusion fft/poly + cache lock-free), pression GC -10 à -15 %.

---

## 2. Méthodologie & périmètre

### 2.1 Méthodologie

L'audit a été réalisé par **lecture intégrale** des 212 fichiers source `.go` (hors tests), via 6 agents Explore lancés en parallèle, chacun ciblant un sous-système cohérent. Un 7ᵉ agent a réalisé une analyse transverse (architecture, tests, outillage). Pour chaque problème identifié, l'agent a produit :

- une référence précise `file_path:line_number` ;
- une sévérité (Critique / Haute / Moyenne / Basse) ;
- une recommandation concrète, actionnable, isolée.

Les findings ont ensuite été consolidés ici, dédupliqués et organisés en **roadmap priorisée**.

### 2.2 Périmètre

Inclus :
- L'ensemble du code source `internal/` et `cmd/`.
- La configuration de linting (`.golangci.yml`), le `Makefile`, `go.mod`, `.env.example`.
- La documentation `docs/architecture/`, `Claude.md`, `CHANGELOG.md`, `PLAN.md`, `README.md`.
- L'architecture de tests (99 fichiers `_test.go`, golden, fuzz, E2E).

Hors périmètre (référencés uniquement) :
- Exécution effective des benchmarks (`make benchmark`) — pas de profil PGO observé.
- Génération effective d'un rapport de couverture (`coverage.out` présent au repo mais pas réexécuté).
- Conformité licence des dépendances (signalée mais pas auditée).

### 2.3 Métriques agrégées

| Métrique | Valeur |
|---|---|
| Packages internes | 19 |
| Binaires CLI (`cmd/`) | 2 (`fibcalc`, `generate-golden`) |
| Fichiers source `.go` (hors tests) | 212 |
| LOC source | ~36 500 |
| LOC tests | ~20 400 |
| Ratio test/source | ~57 % |
| Fichiers > 100 LOC | 56 (cible linter) |
| Fichiers > 400 LOC | 7 (`tui/model.go`, `manager.go`, `pool.go`, `fft_cache.go`, `fft_poly.go`, `completion.go`, `fastdoubling.go`) |
| Couverture annoncée | 87,5 % (badge README) |
| Linters actifs | 22 (`.golangci.yml`) |
| Cibles Makefile | 28 |
| `doc.go` présents | 21/21 (100 %) |

---

## 3. Vue d'ensemble architecturale

### 3.1 Hiérarchie Clean Architecture

```
cmd/
  fibcalc/                 # Entrée minimaliste — délègue à app
  generate-golden/         # Oracle indépendant pour golden tests
internal/
  app/                     # Lifecycle, dispatch, version
  cli/                     # Présentation CLI, complétion
  tui/                     # Dashboard Bubble Tea
  ui/                      # Thèmes couleurs (partagé CLI/TUI)
  format/                  # Helpers formatage (durée, nombres, ETA)
  orchestration/           # Concurrence, agrégation résultats
  parallel/                # ⚠ Quasi-mort (ErrorCollector inutilisé)
  progress/                # Pattern observer + DTO progress
  fibonacci/               # CŒUR : algorithmes
    memory/                # Arena, GC controller, budget
    threshold/             # Manager dynamique (FFT, parallèle, Strassen)
    fibonaccitest/         # Doubles de test
  bigfft/                  # Schönhage-Strassen, bump allocator, cache LRU
  calibration/             # Auto-calibration adaptative + persistance
  config/                  # Flags, env, hardware, seuils
  errors/                  # Types erreurs + handler + codes de sortie
  metrics/                 # Indicateurs perf (typés)
  sysmon/                  # Sample CPU/mem (28 LOC, candidat fusion)
  testutil/                # Helpers de test (ANSI strip, ~21 LOC)
```

### 3.2 Verdict général

| Aspect | Verdict | Commentaire |
|---|---|---|
| Respect des couches Clean Architecture | ✅ | Aucun cycle, aucune fuite `internal → cmd`. |
| Distribution des responsabilités | ⚠ | 4 packages ont une cohésion floue (présentation, progression). |
| Naming | ✅ | Idiomes Go respectés ; quelques incohérences mineures (cf. § 4.1). |
| Documentation | ✅ | 100 % `doc.go` ; quelques minimalistes (orchestration, parallel). |
| Erreurs structurées | ✅ | `fmt.Errorf("%w", …)` systématique ; types typés (5 erreurs). |
| Tests | ✅ | Multi-niveaux, golden + fuzz + E2E + property. |
| CI/CD | ❌ | **Aucun workflow GitHub Actions** — angle mort majeur. |
| Linting | ✅ | 22 linters, exceptions documentées. |
| Performance | ✅/⚠ | Patterns en place, mais quelques opportunités (cache lock-free, fusion fft). |
| Concurrence | ✅/⚠ | `errgroup` + sémaphores OK, mais globals freinent les tests. |
| Sécurité mémoire | ❌ | **Bug latent dans le pooling state+arena** (P1-04 incomplet). |

### 3.3 Couplages structurels critiques

Le diagramme ci-dessous met en évidence **3 couplages problématiques** qui rendent le refactoring plus complexe qu'il ne devrait :

```
┌─────────────────────────────────────────────────────────────────┐
│                     COUPLAGES À DÉCOUPLER                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  fibonacci/doubling_framework.go ──❶──► bigfft.TransformCache   │
│  (boucle critique sample chaque 8 itér)                         │
│                                                                 │
│  fibonacci/calculator.go ────────❷──► fibonacci/memory.GCCtrl   │
│  (Calculator possède la responsabilité GC)                      │
│                                                                 │
│  fibonacci/progress_aliases.go ──❸──► internal/progress         │
│  (réexports pass-through inutiles)                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Ces 3 couplages sont les **leviers prioritaires** du refactoring (cf. § 6).

---

## 4. Audit détaillé par sous-système

### 4.1 `internal/fibonacci/` — cœur algorithmique

**Périmètre** : 19 fichiers (hors sous-packages), ~3 290 LOC. Cœur des algorithmes Fast Doubling, Matrix Exponentiation, FFT-based, GMP, plus le registry, les stratégies adaptatives et les frameworks d'exécution.

#### 4.1.1 Bug critique — pooling `CalculationState` + arena (P1-04 incomplet)

**Sévérité : CRITIQUE**

`fastdoubling.go:215-329`. La paire `CalculationState` ↔ `CalculationArena` est poolée, et `clearStateAliases` (lignes 322-328) doit sectionner les aliases avant retour à la pool. Or, **selon la branche d'erreur (`overLimit=true`)**, les slots aliasants ne sont pas nettoyés. Conséquence :
- Une nouvelle acquisition peut hériter de slices qui pointent encore vers l'arena précédente.
- Si l'arena, elle, est rendue à son propre pool plus tôt, l'invariant « aucune state ne possède d'alias vers une arena retournée » est violé.

C'est le **classique scénario use-after-free latent** : aucun crash visible aujourd'hui parce que le pool reuse rapide de la même arena sauve la situation, mais sous charge, ou avec un GC agressif, ou une recompilation différente, ce code peut produire des résultats corrompus.

**Recommandation immédiate (R1.1)** : déplacer l'appel `clearStateAliases` AVANT toute branche d'erreur. Ajouter un test de leak detection avec `runtime.SetFinalizer` ou `go test -race -count=1000` sur un scénario forcé de `overLimit`.

#### 4.1.2 Couplages forts à découpler

| # | Couplage | Référence | Sévérité |
|---|---|---|---|
| C1 | `DoublingFramework` ↔ `bigfft.GetTransformCache` | `doubling_framework.go:246-261` | Haute |
| C2 | `Calculator` ↔ `memory.GCController` | `calculator.go:175-177` | Moyenne |
| C3 | `progress_aliases.go` (pass-through) | `progress_aliases.go:1-56` | Critique (architectural) |

`ExecuteDoublingLoop` échantillonne le cache FFT toutes les 8 itérations dans la boucle critique. C'est une optimisation valide, mais elle injecte une dépendance directe vers `bigfft` au cœur de la boucle. **Recommandation** : extraire une interface `CacheStrategy` injectable.

`progress_aliases.go` réexporte 10 types et 6 fonctions de `internal/progress` sans valeur ajoutée. **Supprimer entièrement** et migrer les imports.

#### 4.1.3 Complexité excessive

| Fonction | Fichier | Lignes | Observation |
|---|---|---|---|
| `ExecuteDoublingLoop` | `doubling_framework.go:141-278` | 137 | `//nolint:gocognit` ; 5 responsabilités (doubling, progress, seuils, cache, métriques). |
| `multiplyMatrices` | `matrix_ops.go:54-63` | 10 (mais 6 paramètres) | À grouper en `MultiplicationThresholds`. |
| `AcquireStateForN` | `fastdoubling.go:282-316` | 35 | 3 cas + estimation `bitsPerN` approximative sans safety margin. |
| `OptimizedFastDoubling.CalculateCore` | `fastdoubling.go:114` | — | Hardcode `&AdaptiveStrategy{}` — non injectable. |

#### 4.1.4 Duplications & dead code

- `smartMultiply` vs `smartSquare` (`fft.go:48-81`) : structures quasi identiques, **acceptable** (factoriser coûterait plus que ça ne rapporte).
- 5 fonctions `checkStrassen*`/`checkMatrix*` (`matrix_types.go:168-209`) : remplaçables par 1 variadique.
- `executeFFTTransformsParallel` vs `executeFFTTransformsSequential` (`fft.go:157-257`, ~100 L) : 95 % de duplication, **fusionnable** avec un paramètre `inParallel bool`.
- `findHighestBit` (`calculator_gmp.go:55-64`) : wrapper inutile autour de `bits.Len64`.
- `FFTOnlyStrategy` (`strategy.go:120-149`) : stratégie utilisée à un seul endroit, peut devenir privée à `fft_based.go`.
- Constantes redondantes : `maxArenaPoolWords` (`fastdoubling.go:17`) **=** `MaxPooledBitLen` (`common.go:60`) — fragiles à diverger silencieusement.

#### 4.1.5 Naming & cohérence

- `OptimizedFastDoubling` vs `MatrixExponentiation` vs `FFTBasedCalculator` : préfixes/suffixes incohérents. Standardiser en `FastDoublingCalculator`, `MatrixExponentiationCalculator`, `FFTBasedCalculator`.
- Notation matrice : `a, b, c, d` (struct) vs `A11, A21, B12` (commentaires). Documenter la convention en haut de `matrix.go`.

#### 4.1.6 Testabilité

- `DoublingFramework` n'expose aucun setter pour la stratégie — impossible de tester avec un mock de stratégie.
- `getTaskSemaphore` (`common.go:40-52`) en lazy init via `sync.Once` : correct mais ajoute un check dans le hot path. Documenter.
- `matrixStatePool.Get()` ne reset pas les temporaires `p1..p7, s1..s8, t1..t5`. Compte sur l'algorithme pour les écraser ; fragile.

#### 4.1.7 Refactorings prioritaires (package fibonacci)

| Pri | Action | Effort | Bénéfice |
|---|---|---|---|
| P1 | **Fixer pooling state+arena** (R1.1) | 30 min | Élimine bug critique latent. |
| P1 | **Supprimer `progress_aliases.go`** | 20 min | -56 L, supprime une indirection. |
| P1 | **Découpler cache FFT du DoublingFramework** | 4-6 h | Boucle critique 137→~80 L, -1 dépendance directe. |
| P2 | Standardiser les noms de Calculator | 1 h | Cohérence API. |
| P2 | Fusionner `executeFFTTransforms{Parallel,Sequential}` | 1 h | -50 L. |
| P2 | Centraliser `MaxPooledBitLen` | 15 min | Élimine duplication fragile. |
| P3 | Injecter la `Strategy` dans `OptimizedFastDoubling` | 1 h | Testabilité. |
| P3 | Variadique pour `checkStrassen*`/`checkMatrix*` | 30 min | -30 L. |
| P3 | Remplacer `findHighestBit` par `bits.Len64` | 5 min | Cohérence stdlib. |

---

### 4.2 `internal/bigfft/` — multiplication FFT Schönhage-Strassen

**Périmètre** : 17 fichiers, ~3 320 LOC. Le code numérique le plus chaud du projet. Implémente la FFT sur anneaux de Fermat (Schönhage-Strassen) avec allocateur bump O(1), cache LRU thread-safe des transformées, et un système de pools très étagé.

#### 4.2.1 Bugs / risques sécurité mémoire

**B1 — Fuite silencieuse de buffer (Critique)**
`pool.go:111-123` (`releaseWordSlice`). La condition `wordSliceSizes[idx] == cap` peut être fausse si le slice a été resizé entre acquisition et release : le buffer est alors silencieusement abandonné au GC. Sous charge, cela neutralise le pooling.

**Fix** : restaurer la pleine capacité avant `Put`, ou logger lorsque la condition échoue.

**B2 — Panics dans le hot path (Critique)**
`fermat.go:23-46`, `fermat.go:151-206`. Les opérations Fermat (`norm`, `Shift`, `Mul`, `Sqr`) panic sur mismatch de taille. Pas d'`error` retournée → impossible de distinguer un bug logique du caller d'une corruption de données.

**Fix** : retourner `error` et propager vers `Poly.mul`, `transform`, etc. Coût runtime nul (chemin error rare).

**B3 — Cache `putByKey` alloue eagerly même en eviction (Critique)**
`fft_cache.go:262-303`. Lorsqu'on insert et que la cache est pleine, on **évince puis alloue** un nouveau buffer contigu. Coût O(K·n) même quand le hit rate est élevé.

**Fix** : recycler le buffer évincé quand sa taille convient.

#### 4.2.2 Duplications structurelles

**D1 — `fft.go::fftmulTo` vs `fft_poly.go::Poly.mul`**
Logique quasi identique (créer 2 polynômes, multiplier via cache, reconstruire). 100-150 L à fusionner. Maintenance asymétrique = source de bugs.

**D2 — `fourier`, `fourierWithState`, `fourierWithBump`** (`fft_core.go`)
3 variantes, `fourierWithState` recalcule les acquisitions de pool à chaque appel. Unifier via une `FFTContext` paramétrée.

**D3 — Hash FNV codé 3×** (`fft_cache.go:145-170`)
`fnvWriteUint64`, `computeCacheKey`, `computePolyKey` réimplémentent le même hash. Extraire un `cacheKeyBuilder`.

#### 4.2.3 Sur-architecture

- `BumpAllocatorAdapter` (`allocator.go:80-101`) : wrapper inutile autour de `BumpAllocator`. Faire `BumpAllocator` implémenter `TempAllocator` directement.
- `EstimateBumpCapacity` (`bump.go:217-242`) : boucle O(n) sur `fftSizeThreshold` à chaque appel, alors qu'une `sort.Search` (O(log n)) ou une lookup table (O(1)) suffirait. Marge de sécurité magique `* 11 / 10` non justifiée.
- `cpu_amd64.go` (168 L) : détecte AVX2/AVX512/BMI2/ADX **mais ces flags ne sont jamais utilisés**. Code mort potentiel.
- `arith_amd64.go` (32 L) : checks triviaux `len(z) == 0` réinjectés autour des `go:linkname` ; gain perf nul.
- `scan.go` (88 L, decimal-string → big.Int) : **aucun caller dans `bigfft/`**. Code orphelin probable.
- `pool.go:60-67` (`getWordSlicePoolIndexLinear`) : référence O(n) gardée comme « test » ; jamais appelée.

#### 4.2.4 Synchronisation & globals

- `globalTransformCache` (singleton, `fft_cache.go:93`).
- `concurrencySemaphore` global (`fft_recursion.go:11`), initialisé une seule fois — `runtime.NumCPU` figé.
- 13 pools globaux (10 `wordSlicePools` + `fermatPools` + `natSlicePools` + `fermatSlicePools`).
- `bumpAllocatorPool` global (`bump.go:34`).

**Conséquence** : aucun reset facility pour tests, race conditions difficiles à reproduire, impossible d'instancier deux contextes FFT isolés.

**Recommandation R5** : exposer un `NewFFTContext(opts)` qui encapsule cache + allocator + semaphore. Garder une API package-level avec contexte par défaut pour backward compat.

#### 4.2.5 Cache `getByKey` — double mutex

`fft_cache.go:197-210`. Sur **chaque hit**, on acquiert RLock (lecture) puis Lock (mise à jour LRU). Sur cache très chaude (10 k+ accès/s), la contention devient mesurable.

**Fix R2** : fast-path lock-free pour le hit, lock seulement pour la mise à jour LRU. Gain estimé +10-20 % de latency sous contention.

#### 4.2.6 Refactorings prioritaires (bigfft)

| Pri | Action | Effort | Gain |
|---|---|---|---|
| P1 | Fixer `releaseWordSlice` (B1) | 30 min | Stabilité mémoire critique. |
| P1 | Errors au lieu de panics dans Fermat (B2) | 4 h | Robustesse production. |
| P1 | Fix `putByKey` eager alloc (B3) | 2 h | -10-20 % latency cache full. |
| P2 | Fusionner `fftmulTo` ↔ `Poly.mul` (D1) | 1 j | -100-150 L, -10 % latency end-to-end. |
| P2 | Cache fast-path lock-free (R2) | 4 h | +15 % cache hit latency. |
| P2 | `FFTContext` injectable (R5) | 1 j | Testabilité dramatiquement améliorée. |
| P3 | Supprimer `BumpAllocatorAdapter` | 30 min | -25 L. |
| P3 | `EstimateBumpCapacity` en `sort.Search` | 1 h | +5-10 % par fftmul. |
| P3 | Consolider hash FNV (D3) | 1 h | -30 L. |
| P3 | Supprimer `scan.go`, `getWordSlicePoolIndexLinear`, CPU detection inutilisée | 30 min | -300 L code mort. |

---

### 4.3 `memory/`, `threshold/`, `calibration/` — ressources & tuning

**Périmètre** : 14 fichiers, ~2 240 LOC. Gestion mémoire (arena, GC, budget), seuils dynamiques, auto-calibration.

#### 4.3.1 `internal/fibonacci/memory/`

**Cohésion : EXCELLENTE** (3 fichiers, 3 responsabilités nettes).

Problèmes :
- **GC non panic-safe** (`gc_control.go:63-97`). Si un panic survient entre `Begin()` et `End()`, le GC reste désactivé pour le reste de la vie du processus. **Recommandation R1.2** : forcer `defer gc.End()` côté appelant ou wrapper en `WithGC(fn func() error)`.
- **`MemoryLimit ratio = 3×` non documenté** (ligne 71). Pourquoi 3× ? Si heap = 8 GB → limit = 24 GB (peut dépasser RAM physique).
- **`EstimateMemoryUsage`** (`budget.go:18-39`) : coefficients empiriques (5 big.Int, 3× FFT, 1× overhead) non sourcés. Aucun pré-flight check n'invoque cette estimation avant `Calculate()` — le check `--memory-limit` est purement déclaratif.

#### 4.3.2 `internal/fibonacci/threshold/`

**Cohésion : PROBLÉMATIQUE** (1 fichier de 417 L pour 7 responsabilités).

`manager.go` accumule :
1. État (`currentFFTThreshold`, `currentParallelThreshold`).
2. Ring buffer (`metrics`, `metricsHead`, `metricsCount`).
3. Hystérésis (`significantChange`, `hysteresisMargin`).
4. Speedup (`avgTimePerBit`, `calculateSpeedupRatio`).
5. Ajustement multiplicatif (`applyThresholdAdjustment`).
6. Analyses (`analyzeFFTThreshold`, `analyzeParallelThreshold`).
7. Statistiques publiques (`GetStats`).

**Recommandation R3** : décomposer en 3 types :
```go
type MetricsBuffer struct { … }     // Ring buffer pur
type ThresholdAnalyzer struct { … } // Logique pure (hystérésis, speedup)
type Manager struct { … }           // Orchestration + getters thread-safe
```

Bénéfices : tests unitaires de l'hystérésis indépendants, lecture du fichier divisée par 3, surface API publique réduite.

#### 4.3.3 `internal/calibration/`

**Cohésion : ACCEPTABLE mais chemin dual confus.**

`AutoCalibrateWithProfile` (`calibration.go:276-321`) :
1. Lance `QuickCalibrate` (micro-bench).
2. **Si confidence ≥ 0.5** : retourne.
3. **Sinon** : redémarre une calibration complète avec `newCalibrationRunner`.

C'est en fait une **stratégie adaptative à 2 paliers**, mais le code la traite comme deux fonctions sans abstraction commune.

**Recommandation R4** : extraire une interface `CalibrationStrategy` avec deux implémentations (`FastStrategy`, `CompleteStrategy`) et un orchestrateur qui escalade.

#### 4.3.4 Magic numbers disséminés

| Fichier | Constante | Valeur | Justifié ? |
|---|---|---|---|
| `threshold/manager.go:28` | `FFTSpeedupThreshold` | 1.2 | ❌ |
| `threshold/manager.go:31` | `ParallelSpeedupThreshold` | 1.1 | ❌ |
| `threshold/manager.go:35` | `HysteresisMargin` | 0.15 | ❌ |
| `threshold/manager.go:327` | `minThreshold (FFT)` | 100 000 | ❌ |
| `threshold/manager.go:343` | `minThreshold (Par)` | 1 024 | ❌ |
| `memory/gc_control.go:21` | `GCAutoThreshold` | 1 000 000 | ✅ N≥1M |
| `memory/gc_control.go:71` | mem limit multiplier | 3 | ❌ |
| `microbench.go:24` | `MicroBenchTimeout` | 150 ms | ❌ |
| `microbench.go:33-36` | `TestSizes` | [500, 2 K, 8 K, 16 K] | ❌ |

**Recommandation R6** : créer `internal/config/threshold_tuning.go` avec une struct `ThresholdTuningProfile` documentée, optionnellement variant par CPU class.

#### 4.3.5 Persistance profil — `IsStale` jamais invoqué

`profile.IsStale(maxAge)` existe (`profile.go:167-172`) mais **n'est appelé nulle part** dans `calibration.go`. Un profil de 6 mois est accepté tel quel.

**Recommandation R1.3** :
```go
if loaded && profile.IsValid() && !profile.IsStale(24*time.Hour) {
    // use cached
}
```

#### 4.3.6 Refactorings prioritaires (memory / threshold / calibration)

| Pri | Action | Effort |
|---|---|---|
| P1 | GC panic-safe (R1.2) | 1 h |
| P1 | Invalidation `IsStale` (R1.3) | 30 min |
| P2 | Décomposer `Manager` en 3 types (R3) | 4 h |
| P2 | Centraliser magic numbers (R6) | 2 h |
| P2 | Stratégie adaptative `Calibrate` (R4) | 3 h |
| P3 | Pré-flight check `MemoryLimit` dans `Calculator` | 1.5 h |
| P3 | Documenter coefficients `EstimateMemoryUsage` | 1 h |
| P3 | Exposer `Manager.State()` pour observabilité | 1 h |

---

### 4.4 `orchestration/`, `progress/`, `parallel/` — concurrence & rapports

**Périmètre** : 11 fichiers, ~970 LOC.

#### 4.4.1 Triple couche progression — la plus grosse duplication structurelle

| Fichier | Lignes | Rôle |
|---|---|---|
| `internal/progress/progress.go` | 135 | DTO (`ProgressUpdate`, `ProgressCallback`). |
| `internal/progress/observer.go` | 151 | `ProgressSubject` + `RWMutex`. |
| `internal/progress/observers.go` | 139 | `ChannelObserver`, `LoggingObserver`, `NoOpObserver`. |
| `internal/orchestration/progress.go` | 81 | `ProgressAggregator` — **wrapper fin** sur `ProgressWithETA`. |
| `internal/format/progress_eta.go` | 256 | `ProgressWithETA` (calcul ETA + smoothing). |
| `internal/fibonacci/progress_aliases.go` | 56 | **Réexports inutiles**. |

**Constat** : `ProgressAggregator` réexpose 95 % des méthodes de `ProgressWithETA` sans logique ajoutée. `progress_aliases.go` réexpose `progress` sans logique. Le pattern `ProgressObserver` (`observer.go`) **n'est jamais utilisé dans le hot path** — la CLI utilise des channels directement, le TUI passe par le `bridge`.

**Recommandation R7** :
1. Supprimer `progress_aliases.go`.
2. Fusionner `format/progress_eta.go::ProgressWithETA` dans `orchestration/progress.go::ProgressAggregator`.
3. Garder `progress/observer.go` comme couche d'extension optionnelle, **clairement documentée** dans `doc.go` comme « non utilisée par le hot path ».

#### 4.4.2 Orchestrator — fast path dupliqué

`orchestration/orchestrator.go` :
- Fast path single-calculator (lignes ~30-50) duplique la construction de `CalculationResult` avec le cas `errgroup`.
- `ProgressBufferMultiplier = 5` magic number sans justification.
- Démarrage de la goroutine `DisplayProgress` mêlé à l'orchestration des calculs.
- Discard intentionnel de `g.Wait()` pour permettre la cancellation cross-calculator — **comporté correct mais non documenté** dans l'interface publique.

**Recommandation R8** :
- Extraire `startProgressDisplay()` helper.
- Unifier le code path single/multi calculator.
- Documenter explicitement la stratégie d'erreur (`FirstFailureCancelsOthers`).

#### 4.4.3 `internal/parallel/` — code mort

`parallel/errors.go` (59 L) définit `ErrorCollector`. Recherche dans le repo : **zéro caller** hors tests. Le projet utilise `golang.org/x/sync/errgroup` partout.

**Recommandation R9** : supprimer `internal/parallel/` ou y consolider de futurs primitives concurrentes (sinon, c'est mort).

#### 4.4.4 `ProgressReporter` interface non idiomatique

```go
type ProgressReporter interface {
    DisplayProgress(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate,
        numCalculators int, out io.Writer)
}
```

Force le contracteur à gérer son propre `WaitGroup`. Mieux :

```go
type ProgressReporter interface {
    Run(ctx context.Context, progressChan <-chan progress.ProgressUpdate, out io.Writer)
}
```

#### 4.4.5 `GetCalculatorsToRun` — mauvais package

`orchestration/calculator_selection.go` (32 L) est une logique de sélection d'usine, pas d'orchestration. **Déplacer vers `internal/fibonacci/selection.go`**.

#### 4.4.6 Refactorings prioritaires (orchestration / progress / parallel)

| Pri | Action | Effort |
|---|---|---|
| P1 | Supprimer `progress_aliases.go` (cf. 4.1) | 20 min |
| P2 | Fusionner triple couche progression (R7) | 1 j |
| P2 | Extraire `startProgressDisplay`, unifier fast path (R8) | 3 h |
| P2 | Supprimer ou consolider `internal/parallel/` (R9) | 1 h |
| P3 | Simplifier `ProgressReporter` interface | 2 h |
| P3 | Déplacer `GetCalculatorsToRun` → `fibonacci/` | 30 min |
| P3 | `ThrottledProgressCallback` pour robustesse charge | 1 h |

---

### 4.5 `cli/`, `tui/`, `ui/`, `format/` — couches de présentation

**Périmètre** : 30 fichiers, ~3 250 LOC.

#### 4.5.1 `internal/cli/` — frontières floues

Trois fichiers se partagent la présentation, sans cloison nette :

| Fichier | LOC | Contenu |
|---|---|---|
| `presenter.go` | 117 | Interface `ResultPresenter` + routing. |
| `ui_display.go` | 223 | `DisplayResult`, `DisplayProgress`, formatage résultats. |
| `output.go` | 152 | `WriteResultToFile`, `DisplayQuietResult`, `DisplayResultWithConfig`. |

**Confusion** : la présentation, le rendu UI, et la persistance fichier sont mêlés sur 3 fichiers sans contrat clair.

**Recommandation R10** : refactorer en 2 fichiers :
- `presenter.go` : interface + routing (préservé).
- `display.go` : fusion `ui_display + output` (rendu + persistance).

Gain : -150 LOC, contrat clair.

#### 4.5.2 `cli/completion.go` — 520 L monolithiques

Contient les 4 implémentations shell (bash, zsh, fish, PowerShell) plus les helpers spécifiques (`bashCase`, `zshArgEntry`, `fishCompleteLine`, `psSwitchEntry`). `flagRegistry` global est **bien centralisé** (✓), mais le reste est mélangé.

**Recommandation R11** : éclater en `cli/completion/`:
```
cli/completion/
  registry.go       # flagRegistry (source de vérité)
  bash.go
  zsh.go
  fish.go
  powershell.go
```

Bénéfice : testabilité par shell, lisibilité, couverture mesurable.

#### 4.5.3 `tui/model.go` — 425 L, Elm sur-compactée

Le `Model` agrège 5 composants + état d'exécution + layout manager + parentCtx + config + ref + paused. La méthode `Update` (88 L) gère **16 types de messages** dans un switch monolithique, mêlée à 9 lambdas `tea.Cmd` inline (`tickCmd`, `sampleMemStatsCmd`, `sampleSysStatsCmd`…).

**Recommandation R12** : décomposer en sous-modèles Elm :
```go
tui/component/
  logs.go      // LogsComponent (Update + View propres)
  chart.go     // ChartComponent
  metrics.go   // MetricsComponent
  header.go
  footer.go
tui/model.go   // Router Update vers sous-composants
```

Bénéfice : `tui/model.go` ramené à ~200 L de routage, testabilité par composant.

#### 4.5.4 Triple couche styles/couleurs

| Fichier | LOC | Rôle | Problème |
|---|---|---|---|
| `ui/colors.go` | 31 | Wrappers ANSI (`ColorMagenta`, …). | Dépend de `GetCurrentTheme`. |
| `ui/themes.go` | 242 | `Theme` (CLI) + `TUITheme` (lipgloss). | Double représentation parallèle. |
| `tui/styles.go` | 123 | Variables `lipgloss.Style` globales. | Init à package init **et** dans `Run()`. |

**Recommandation R13** : fusionner en un package `ui/style/` unique (struct hybride CLI ANSI + TUI lipgloss). Init unique via `style.Init(noColor, themeName)`.

Gain : single source of truth, ~-100 LOC.

#### 4.5.5 Autres findings

- `format/progress_eta.go` (256 L) mélange `ProgressState` (pure aggregation) et `ProgressWithETA` (ETA + smoothing). Scinder en `progress.go` et `eta.go`.
- `tui/logs.go` (210 L) : `maxLogEntries = 10000` avec slice trim ; **réutiliser le RingBuffer** de `tui/sparkline.go` (déjà très bien fait).
- `tui/bridge.go::programRef.Send` (lignes 31-39) : drop silencieux si `program == nil`. Race possible entre `SetProgram` et `Send` au démarrage. Logger ou retourner `error`.
- TUI responsive : `LogsPanelWidthPercent = 60` en dur ; pas d'adaptation pour terminal < 80 colonnes.
- `cli/ui.go::Spinner` : interface sans intérêt (1 implémentation, jamais mockée). Supprimer ou justifier.

#### 4.5.6 Refactorings prioritaires (présentation)

| Pri | Action | Effort |
|---|---|---|
| P2 | Fusionner cli {presenter, display, output} (R10) | 4 h |
| P2 | Éclater `cli/completion.go` par shell (R11) | 4 h |
| P2 | Décomposer `tui/model.go` en composants (R12) | 1.5 j |
| P2 | Fusionner `ui/colors + themes + tui/styles` (R13) | 6 h |
| P3 | Scinder `format/progress_eta.go` | 1 h |
| P3 | RingBuffer pour `tui/logs.go` | 2 h |
| P3 | Bridge robust (`Send` retourne error) | 1 h |
| P3 | Layout adaptatif TUI <80 cols | 1 h |
| P4 | Supprimer interface `Spinner` | 30 min |

---

### 4.6 `cmd/`, `app/`, `config/`, `errors/`, `metrics/`, `sysmon/`, `testutil/` — couches de bord

**Périmètre** : ~1 950 LOC sur 7 packages.

#### 4.6.1 `cmd/fibcalc/main.go` — sentinel fragile

```go
const exitVersion = -1   // signifie « ne pas appeler os.Exit »
```

Ce sentinel numérique est fragile. **Recommandation** : enum typé `ExitAction { Success, Error, VersionHandled }`.

#### 4.6.2 `app/calculate.go` — 5 responsabilités dans `runCalculate`

`runCalculate` fait :
1. Branchement (LastDigits → route spécialisée).
2. Validation budget mémoire.
3. Lifecycle (ctx + signaux).
4. Orchestration calcul.
5. Analyse + présentation + save.

**Recommandation R14** :
- Extraire `validateMemoryBudget` → `internal/config/validator.go`.
- Extraire `runLastDigits` → `internal/orchestration/lastdigits.go` (logique pure, sans I/O).
- Scinder `analyzeResultsWithOutput` en `selectBest` + `present` + `save`.

#### 4.6.3 `config/env.go` — table-driven mais redondante

`envOverrides` (`env.go:111-180`) duplique structurellement `AppConfig` : pour chaque champ, il faut une entry + une fonction `apply`. Ajouter un flag dans `ParseConfig` sans synchroniser ici → inconsistance silencieuse.

**Recommandation R15** : utiliser **reflection sur struct tags** :
```go
type AppConfig struct {
    N       uint64 `config:"N" parser:"uint64"`
    Verbose bool   `config:"VERBOSE" parser:"bool"`
}
```
Génération automatique de la table. Une seule source de vérité (`AppConfig`).

#### 4.6.4 `errors/handler.go` — défini mais peu utilisé

`HandleCalculationError` mappe correctement erreur → message + diagnostic, mais dans `app/calculate.go`, les erreurs sont gérées en `fmt.Fprintf` directs. **Recommandation R16** : centraliser tous les appels d'erreur calcul vers ce handler.

#### 4.6.5 `errors/errors.go` — bien découplé

✅ 5 types typés (`ConfigError`, `CalculationError`, `TimeoutError`, `ValidationError`, `MemoryError`).
✅ Codes de sortie POSIX-corrects (130 = SIGINT).
✅ Wrapping `%w` systématique.
✅ `CalculationContext` contextuel (machine-readable diagnostic).

Aucun refactoring critique nécessaire ici — seulement vérifier que tous les codes (`ExitErrorMismatch`) sont bien utilisés via un test d'exhaustivité.

#### 4.6.6 `metrics/indicators.go` — solide

Struct typée bien pensée (pas de map string-keyed). Constantes pré-calculées (`log2Phi`, `lastDigitsMod`) à init time — **documenter** le choix.

#### 4.6.7 `sysmon/` — 28 L, candidat à fusion

Une fonction publique `Sample()`, une struct `Stats`. Dépend de `gopsutil`. **Recommandation R17** : fusionner dans `metrics/system.go` pour clarifier que `metrics/` est le dashboard complet (Fibonacci + système).

#### 4.6.8 `testutil/` — minimaliste OK

Une seule fonction (`StripAnsiCodes`). Pas de problème. Garder minimaliste, accepter de futurs helpers.

#### 4.6.9 `cmd/generate-golden/` — doc disproportionnée

29 L de doc.go pour 18 L de logique (`fibBig`). C'est l'oracle indépendant P2-04. **Recommandation R18** : raccourcir doc.go, lier à un `consistency_test.go` qui assure l'invariant à chaque exécution de la suite de tests.

#### 4.6.10 Refactorings prioritaires (couches de bord)

| Pri | Action | Effort |
|---|---|---|
| P2 | Scinder `runCalculate` en sous-fonctions (R14) | 4 h |
| P2 | `envOverrides` via reflection (R15) | 4 h |
| P2 | Centraliser erreurs via `HandleCalculationError` (R16) | 2 h |
| P3 | Fusionner `sysmon/` dans `metrics/` (R17) | 1 h |
| P3 | `ExitAction` enum typé (cmd/main.go) | 30 min |
| P3 | Documenter `metrics` constantes init | 30 min |
| P4 | Raccourcir doc generate-golden (R18) | 1 h |

---

## 5. Constats transverses

### 5.1 Duplications fonctionnelles à fort levier

| # | Domaine | Fichiers concernés | LOC à supprimer |
|---|---|---|---|
| T1 | Couche progression (3 implémentations) | `progress/`, `orchestration/progress.go`, `format/progress_eta.go`, `fibonacci/progress_aliases.go` | ~200 |
| T2 | Couche styles/couleurs (3 surfaces) | `ui/colors.go`, `ui/themes.go`, `tui/styles.go` | ~100 |
| T3 | Présentation CLI (3 fichiers) | `cli/presenter.go`, `cli/ui_display.go`, `cli/output.go` | ~150 |
| T4 | FFT mul interne (`fftmulTo` ↔ `Poly.mul`) | `bigfft/fft.go`, `bigfft/fft_poly.go` | ~150 |
| T5 | FFT transforms parallèle/séquentielle | `fibonacci/fft.go` | ~50 |
| T6 | Hash FNV (3 versions) | `bigfft/fft_cache.go` | ~30 |
| T7 | Code mort (parallel, scan, CPU detect) | divers | ~400 |

**Total estimé** : **~1 080 LOC à supprimer** pour zéro perte fonctionnelle, et un gain net en lisibilité.

### 5.2 Monolithes à décomposer

| Fichier | LOC | Décomposition cible |
|---|---|---|
| `internal/cli/completion.go` | 520 | 5 fichiers (registry + 4 shells) |
| `internal/bigfft/fft_cache.go` | 534 | 2-3 fichiers (LRU vs API vs hash) |
| `internal/bigfft/pool.go` | 467 | 2 fichiers (word slice pools vs fermat pools) |
| `internal/bigfft/fft_poly.go` | 533 | À fusionner avec `fft.go` puis subdiviser |
| `internal/tui/model.go` | 425 | 1 router + 5 composants |
| `internal/fibonacci/threshold/manager.go` | 417 | 3 types (`MetricsBuffer`, `ThresholdAnalyzer`, `Manager`) |
| `internal/fibonacci/fastdoubling.go` | 414 | À conserver, sauf extraction de `AcquireStateForN` |
| `internal/calibration/calibration.go` | 391 | Stratégie `Fast` + `Complete` séparées |
| `internal/calibration/microbench.go` | 379 | Acceptable mais à wrapper |

### 5.3 Globals & singletons (impact testabilité)

| Global | Fichier | Risque |
|---|---|---|
| `globalTransformCache` | `bigfft/fft_cache.go:93` | Tests non isolés. |
| `concurrencySemaphore` | `bigfft/fft_recursion.go:11` | NumCPU figé après init. |
| 13 pools (`wordSlicePools`, `fermatPools`…) | `bigfft/pool.go:18-32` | Pas de reset facility. |
| `bumpAllocatorPool` | `bigfft/bump.go:34` | Leak detection difficile. |
| `globalSem` | `fibonacci/common.go:40-52` | Acceptable (lazy + Once). |

**Recommandation transverse** : encapsuler dans `bigfft.Context` injectable (cf. R5).

### 5.4 Zone CI/CD — angle mort majeur

**Aucun workflow GitHub Actions** détecté à la racine du repo. Le `Makefile` couvre `lint`, `test`, `coverage`, `security` mais rien ne les déclenche automatiquement. Pour un projet présenté comme **production-ready** avec un badge de couverture 87,5 %, l'absence de CI est le **plus gros écart** entre la prétention et la réalité.

**Recommandation R19 (priorité HAUTE)** : ajouter `.github/workflows/ci.yml` avec :
- `make test` (race detector).
- `make lint`.
- `make coverage` + upload artifact.
- Matrice `go: 1.25.x` × `os: [ubuntu, windows, macos]`.
- Trigger : `push` + `pull_request`.

Coût : ½ journée. Bénéfice : détection des régressions, confiance des contributeurs externes.

### 5.5 Dette de tests

- `t.Parallel()` présent dans 77/99 fichiers (~78 %). Plusieurs tests file-based pourraient l'ajouter sans risque.
- Doublon mineur : `fibonacci_test.go` ↔ `fibonacci_property_test.go`. Fusionnables.
- `fibonacci/fibonaccitest/` est utilisé partiellement ; certains tests inline leurs propres mocks (`MockResultPresenter` dans `orchestration_test.go`). Politique mixte acceptable mais à standardiser.
- Pas d'artefact de couverture publié — `coverage.out` au repo (173 KB) n'est pas régénéré automatiquement.

### 5.6 Qualité documentaire

- 21/21 packages ont un `doc.go` (✅).
- Quelques `doc.go` minimalistes (`orchestration/doc.go` 4 L, `parallel/doc.go` 37 L pour 0 caller).
- `cmd/generate-golden/doc.go` sur-doc'é (29 L doc / 18 L code).
- `docs/architecture/` : Mermaid à jour, dependency graph clair.
- `CHANGELOG.md` (8 KB) : structuré Keep-a-Changelog.

---

## 6. Roadmap de refactoring (priorisée)

La roadmap est segmentée en **4 vagues** indépendantes, chacune mergeable séparément, sans casser la rétrocompatibilité.

### Vague 1 — Fixes critiques (½ jour)

| ID | Action | Effort | Bénéfice |
|---|---|---|---|
| R1.1 | Fix pooling `clearStateAliases` toujours appelé | 30 min | Élimine bug latent UAF. |
| R1.2 | `defer gc.End()` panic-safe | 1 h | Évite GC désactivé persistant. |
| R1.3 | Invalidation profil via `IsStale` | 30 min | Profils obsolètes re-calibrés. |
| R1.4 | Fix `releaseWordSlice` (B1 bigfft) | 30 min | Stabilité mémoire pool. |
| R1.5 | Fix `putByKey` eager alloc (B3) | 2 h | -10-20 % latency cache full. |

**Total : ~½ journée. Toutes ces actions corrigent des bugs latents et n'affectent aucune API publique.**

### Vague 2 — Suppressions et fusions à fort levier (1-2 semaines)

| ID | Action | Effort | LOC supprimées |
|---|---|---|---|
| R2.1 | Supprimer `progress_aliases.go` + migrer imports | 30 min | 56 |
| R2.2 | Supprimer ou consolider `internal/parallel/` | 1 h | 60 |
| R2.3 | Supprimer `bigfft/scan.go`, `getWordSlicePoolIndexLinear`, CPU detection inutilisée | 2 h | 300 |
| R2.4 | Supprimer `BumpAllocatorAdapter` | 30 min | 25 |
| R2.5 | Fusionner triple couche progression (R7) | 1 j | 200 |
| R2.6 | Fusionner `cli/{presenter,ui_display,output}` (R10) | 4 h | 150 |
| R2.7 | Fusionner `ui/colors+themes` + `tui/styles` (R13) | 6 h | 100 |
| R2.8 | Fusionner `bigfft fft.go ↔ fft_poly.go::Poly.mul` (D1) | 1 j | 150 |
| R2.9 | Fusionner `executeFFTTransforms{Parallel,Sequential}` | 1 h | 50 |
| R2.10 | Consolider hash FNV (D3) | 1 h | 30 |

**Total : ~5 jours. ~1 100 LOC supprimées.**

### Vague 3 — Décompositions structurelles (3 semaines)

| ID | Action | Effort |
|---|---|---|
| R3.1 | Décomposer `threshold/manager.go` (R3) | 4 h |
| R3.2 | Découpler cache FFT du `DoublingFramework` (R5 + interface `CacheStrategy`) | 1 j |
| R3.3 | Stratégie `Calibrate` adaptative (R4) | 3 h |
| R3.4 | Décomposer `tui/model.go` en composants (R12) | 1.5 j |
| R3.5 | Éclater `cli/completion.go` par shell (R11) | 4 h |
| R3.6 | Scinder `runCalculate` (R14) + extraire `lastdigits` | 4 h |
| R3.7 | `bigfft.FFTContext` injectable (R5) | 1 j |
| R3.8 | Errors au lieu de panics dans Fermat (B2) | 4 h |
| R3.9 | Cache fast-path lock-free (R2) | 4 h |
| R3.10 | Standardiser noms `*Calculator` | 1 h |

**Total : ~3 semaines. Découpages testables, perf +5-10 %.**

### Vague 4 — Polissage & outillage (1 semaine)

| ID | Action | Effort |
|---|---|---|
| R4.1 | **GitHub Actions CI/CD (R19)** | ½ j |
| R4.2 | Centraliser magic numbers en `config/threshold_tuning.go` (R6) | 2 h |
| R4.3 | `envOverrides` via reflection (R15) | 4 h |
| R4.4 | Centraliser erreurs via `HandleCalculationError` (R16) | 2 h |
| R4.5 | Fusionner `sysmon/` dans `metrics/` (R17) | 1 h |
| R4.6 | `ExitAction` enum typé | 30 min |
| R4.7 | Pré-flight memory check dans `Calculator` | 1.5 h |
| R4.8 | RingBuffer pour `tui/logs.go` | 2 h |
| R4.9 | Bridge `Send` retourne `error` | 1 h |
| R4.10 | Layout TUI adaptatif <80 cols | 1 h |
| R4.11 | Fusionner `fibonacci_property_test` ↔ `fibonacci_test` | 30 min |
| R4.12 | Ajouter `t.Parallel()` aux tests file-based | 1 h |

**Total : ~1 semaine. CI/CD livrée, qualité tooling.**

### Synthèse globale

| Vague | Effort | Risque | Bénéfice principal |
|---|---|---|---|
| 1 | ½ jour | Faible | Bugs critiques corrigés. |
| 2 | 1 semaine | Faible | -1 100 LOC, clarté. |
| 3 | 3 semaines | Moyen | Testabilité +50 %, perf +5-10 %. |
| 4 | 1 semaine | Faible | CI/CD, polissage. |
| **Total** | **~5,5 semaines** | — | Codebase 30 % plus simple, plus testable, CI complète. |

---

## 7. Risques & trade-offs

### 7.1 Risques d'exécution du refactoring

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| Régression perf en fusionnant `fft.go ↔ fft_poly.go` | Moyenne | Haute | Bench avant/après obligatoire (`make benchmark`), gate à -5 %. |
| Régression golden tests sur Fast Doubling | Faible | Critique | Golden + fuzz exécutés systématiquement à chaque étape. |
| Casser la backward compat API publique | Moyenne | Moyenne | Marquer `Deprecated:` les anciennes APIs avant suppression. |
| Décomposer `tui/model.go` introduit des bugs visuels | Moyenne | Basse | Tests snapshot Bubble Tea (à ajouter). |
| Reflection sur `envOverrides` plus lente | Très faible | Très basse | Une fois au démarrage ; impact nul. |
| Fix pooling `clearStateAliases` casse perf chaude | Faible | Moyenne | Bench Fib(10M) avant/après. |

### 7.2 Trade-offs structurels à valider avec le mainteneur

1. **Conserver `progress/observer.go` ou non ?** — Pattern non utilisé en hot path mais utile pour extensions futures (TUI plugins, métriques externes). Recommandation : conserver et **documenter clairement** comme couche d'extension.

2. **Conserver `internal/parallel/` ou supprimer ?** — Aucun caller hors tests. Si suppression, **mineure** (60 LOC). Si conservation, prévoir un cas d'usage concret.

3. **CPU detection (`bigfft/cpu_amd64.go`) inutilisée** — Supprimer ou prévoir activation future ? Recommandation : supprimer maintenant, ressortir d'un commit `git revert` si besoin.

4. **`generate-golden` doc disproportionnée** — Garder doc explicative (audit P2-04 cite explicitement le pattern) ou raccourcir + lier à un test ? Recommandation : raccourcir, le test garantit l'invariant.

5. **Fusion `sysmon/` → `metrics/`** — Réduction de la surface package mais perte d'une frontière nette entre Fibonacci-metrics et OS-metrics. Recommandation : fusionner avec sous-package `metrics/system/` pour garder la séparation.

### 7.3 Ce qui est **explicitement bien** (à NE PAS toucher)

- Architecture Clean en 4 couches (cmd → app → orchestration → fibonacci/bigfft).
- Système d'erreurs typés (`ConfigError`, `CalculationError`, `TimeoutError`, `ValidationError`, `MemoryError`) + codes POSIX.
- Stratégie de tests (golden + fuzz + property + E2E) — à enrichir, pas à refondre.
- `internal/fibonacci/memory/` — cohésion exemplaire (3 fichiers, 3 responsabilités).
- Pattern `RingBuffer` dans `tui/sparkline.go` — bien fait, à réutiliser ailleurs (logs).
- `errors/errors.go` — référence à imiter pour les autres packages.
- `metrics/indicators.go` — typage fort exemplaire, pas de string-keyed map.
- `Makefile` — complet, bien organisé (28 cibles, PGO workflow propre).
- `.golangci.yml` — 22 linters cohérents avec le code.

---

## 8. Annexes

### 8.1 Glossaire

- **PGO** : Profile-Guided Optimization. Recompilation Go guidée par un profil d'exécution réel.
- **FFT** : Fast Fourier Transform. Ici, Schönhage-Strassen sur anneaux de Fermat pour la multiplication d'entiers de très grande taille.
- **Strassen** : algorithme matriciel utilisant 7 multiplications au lieu de 8 (pour 2×2 récursif).
- **Bump allocator** : allocateur linéaire O(1), pas de free individuel ; reset global.
- **Hystérésis** : marge anti-oscillation lors de l'ajustement de seuils dynamiques.
- **Golden test** : test de régression comparant la sortie courante à une référence persistée.
- **TUI** : Text User Interface (ici Bubble Tea, pattern Elm).

### 8.2 Index des recommandations

| ID | Titre | Vague |
|---|---|---|
| R1.1 | Fix pooling `clearStateAliases` | 1 |
| R1.2 | GC panic-safe | 1 |
| R1.3 | Invalidation profil `IsStale` | 1 |
| R1.4 | Fix `releaseWordSlice` (bigfft) | 1 |
| R1.5 | Fix `putByKey` eager alloc | 1 |
| R2.1 | Supprimer `progress_aliases.go` | 2 |
| R2.2 | Supprimer/consolider `parallel/` | 2 |
| R2.3 | Supprimer code mort `bigfft` (scan, CPU detect, …) | 2 |
| R2.4 | Supprimer `BumpAllocatorAdapter` | 2 |
| R2.5 | Fusionner triple couche progression | 2 |
| R2.6 | Fusionner cli {presenter, ui_display, output} | 2 |
| R2.7 | Fusionner ui {colors, themes} + tui/styles | 2 |
| R2.8 | Fusionner `bigfft fft.go ↔ fft_poly.go` | 2 |
| R2.9 | Fusionner `executeFFTTransforms{P,S}` | 2 |
| R2.10 | Consolider hash FNV | 2 |
| R3.1 | Décomposer `threshold/manager.go` | 3 |
| R3.2 | `CacheStrategy` injectable | 3 |
| R3.3 | Stratégie `Calibrate` adaptative | 3 |
| R3.4 | Décomposer `tui/model.go` | 3 |
| R3.5 | Éclater `cli/completion.go` | 3 |
| R3.6 | Scinder `runCalculate` + extraire `lastdigits` | 3 |
| R3.7 | `bigfft.FFTContext` injectable | 3 |
| R3.8 | Errors au lieu de panics (Fermat) | 3 |
| R3.9 | Cache fast-path lock-free | 3 |
| R3.10 | Standardiser noms `*Calculator` | 3 |
| R4.1 | GitHub Actions CI/CD | 4 |
| R4.2 | Centraliser magic numbers | 4 |
| R4.3 | `envOverrides` via reflection | 4 |
| R4.4 | Centraliser erreurs via handler | 4 |
| R4.5 | Fusionner `sysmon/` dans `metrics/` | 4 |
| R4.6 | `ExitAction` enum typé | 4 |
| R4.7 | Pré-flight memory check | 4 |
| R4.8 | RingBuffer pour `tui/logs.go` | 4 |
| R4.9 | Bridge `Send` retourne `error` | 4 |
| R4.10 | Layout TUI adaptatif | 4 |
| R4.11 | Fusionner property test ↔ unit test fibonacci | 4 |
| R4.12 | Étendre `t.Parallel()` aux tests file-based | 4 |

### 8.3 Références fichiers

Fichiers les plus critiques à traiter (par ordre de priorité) :

1. `internal/fibonacci/fastdoubling.go:215-329` — pooling state+arena.
2. `internal/fibonacci/memory/gc_control.go:63-97` — panic safety GC.
3. `internal/calibration/calibration.go:251-321` — invalidation profil.
4. `internal/bigfft/pool.go:111-123` — release word slice.
5. `internal/bigfft/fft_cache.go:262-303` — eager alloc cache.
6. `internal/fibonacci/progress_aliases.go` — suppression complète.
7. `internal/parallel/errors.go` — suppression complète.
8. `internal/fibonacci/threshold/manager.go` — décomposition.
9. `internal/fibonacci/doubling_framework.go:141-278` — découpler cache.
10. `internal/tui/model.go` — décomposition Elm.

### 8.4 Métriques de gain attendues (vague complète)

| Métrique | Avant | Après | Δ |
|---|---|---|---|
| LOC source totaux | ~36 500 | ~34 500 | -2 000 (-5,5 %) |
| Fichiers > 400 LOC | 7 | 2 | -5 |
| Packages internes | 19 | 17-18 | -1 à -2 |
| Globals dans `bigfft/` | ~15 | ~3 | -80 % |
| Latency end-to-end (Fib 10M) | ~2,1 s | ~1,9 s | -10 % |
| Pression GC (calculs longs) | référence | -10 à -15 % | — |
| Couverture tests | 87,5 % | 90 %+ | +2,5 pp |
| Couverture CI/CD | 0 % | 100 % | +100 pp |
| Surface API publique | ~110 symboles | ~80 | -30 % |

---

**Fin de l'audit.**

L'auteur recommande de procéder par vagues séquentielles, chaque vague faisant l'objet d'un PR distinct, mergé après validation : (a) `make test -race` au vert, (b) `make benchmark` sans régression > 5 %, (c) golden tests verts. La vague 1 (½ journée) devrait être priorisée immédiatement pour fermer les bugs latents.
