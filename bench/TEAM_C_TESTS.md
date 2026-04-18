# TEAM C — Tests & Qualité

Audit read-only basé sur la baseline `bench/baseline/` (test.txt, coverage.out, coverage.txt). Aucune modification de code.

## Résumé exécutif

- **Couverture pondérée totale** : **87.5 %** (`go tool cover -func` sur `coverage.out`).
- **Suite verte** : 22 packages OK, 0 FAIL, 1 553 sous-tests `=== RUN`, 639 lignes `--- PASS`, **0 `DATA RACE`** détecté (rappel baseline : `-race` désactivé sur Windows par absence de gcc — voir `test.txt`).
- **Cible 80 % atteinte** sur 19/22 packages. **2 packages < 80 %** : `cmd/fibcalc` 75.0 %, `cmd/generate-golden` 29.4 % (P1). `internal/format` 79.8 % (proche-cible).
- **`t.Parallel()`** : 664 occurrences pour 610 `func Test` (ratio 1.09 grâce aux sous-tests). Mais **0 `t.Parallel()` dans tout `internal/tui/` hors `cli_flags_test.go`** (10 fichiers, 122 tests séquentiels).
- **Fuzz targets** : 5 (`FuzzFastDoublingConsistency`, `FuzzFFTBasedConsistency`, `FuzzFibonacciIdentities`, `FuzzFastDoublingMod`, `FuzzProgressMonotonicity`) — tous concentrés dans `internal/fibonacci/`. **Manques** : `internal/config` (parser `ParseMemoryLimit`, env vars), `internal/bigfft/scan.go` (`FromDecimalString`), `internal/cli` (parsing flags), `internal/calibration/profile` (loader JSON).
- **Aucun corpus seed persistant** sous `testdata/fuzz/` — uniquement `f.Add()` en code (P2).
- **Mocks** : pas de gomock généré (`generate-mocks` Makefile cible existe mais aucun `//go:generate mockgen`, aucun import `go.uber.org/mock`). Mocks faits main (cohérents) dans 14 fichiers.
- **Findings** : 2 P1, 5 P2, 3 P3.

## Matrice couverture (par package)

| Package | Couverture | Cible 80% | Notes |
|---|---|---|---|
| `cmd/fibcalc` | 75.0 % | ❌ | seul `main()` non couvert (entrypoint), reste à 100 %. Score gonflé à la baisse par 1 fonction. |
| `cmd/generate-golden` | 29.4 % | ❌ | `main()` à 0 %, seule `fibBig` testée. **P1**. |
| `internal/app` | 91.7 % | ✅ | seul `runTUI` 0 % (intégration Bubble Tea). |
| `internal/bigfft` | 88.7 % | ✅ | quelques getters/setters de config à 0 % (`SetCacheLogger`, `SetFFTParallelismConfig`). |
| `internal/calibration` | 90.2 % | ✅ | `AutoCalibrateWithProfile` 58.1 %, `GenerateParallelThresholds` 55.6 %. |
| `internal/cli` | 84.7 % | ✅ | adaptateur `presenter.go` (7 fonctions) à 0 %. **P2**. |
| `internal/config` | 89.4 % | ✅ | `internal/config/thresholds.go` entièrement à 0 % (4 fonctions). |
| `internal/errors` | 91.3 % | ✅ | `HasDiagnostic` 40 %. |
| `internal/fibonacci` | 84.4 % | ✅ | `internal/fibonacci/testing.go` (helpers `TestFactory`) à 0 % — non utilisé hors-package. |
| `internal/fibonacci/fibonaccitest` | 100.0 % | ✅ | — |
| `internal/fibonacci/memory` | 82.2 % | ✅ | `FormatMemoryEstimate`, `formatBytesInternal`, `preSizeBigInt` à 0 %. |
| `internal/fibonacci/threshold` | 93.0 % | ✅ | seul `SetLogger` à 0 %. |
| `internal/format` | **79.8 %** | ⚠️ | 0.2 pt sous cible. `UpdateWithETA` 32 %, `FormatBytes` 0 %. **P2**. |
| `internal/metrics` | 96.1 % | ✅ | `lastNDigits` 80 %. |
| `internal/orchestration` | 85.7 % | ✅ | `ExecuteCalculations` 57.1 % (chemins d'erreur peu testés). |
| `internal/parallel` | 100.0 % | ✅ | — |
| `internal/progress` | 93.3 % | ✅ | `PrecomputePowers4` 44.4 %, `Update` (NoOp) 0 %. |
| `internal/sysmon` | 100.0 % | ✅ | — |
| `internal/testutil` | 100.0 % | ✅ | — |
| `internal/tui` | 87.5 % | ✅ | bonne couverture statements MAIS quasi tout sans `t.Parallel()`. **P1 parallélisme**. |
| `internal/ui` | 94.9 % | ✅ | — |
| `test/e2e` | n/a | n/a | `[no statements]` (binaire compilé). |

## Drill-down couverture (fonctions à 0 % dans packages clés)

### `internal/fibonacci/` (84.4 %)
- `testing.go:46-96` — `NewTestFactory`, `Create`, `Get`, `List`, `Register`, `GetAll`, `(*UnknownCalculatorError).Error` → toutes 0 %. Helpers de test exportés mais jamais consommés en interne (utilisés peut-être par cmd/, voir P2 F-C5).
- `common.go:77` — `SetTaskLogger` 0 % (façade logger).
- `matrix_framework.go:37` — `NewMatrixFrameworkWithSquareFunc` 0 % (variante non testée).
- `matrix_types.go:38` — `(*matrix).Set` 0 % (utilitaire).
- `registry.go:38` `SetRegistryLogger` 0 %, `registry.go:281` `ResetGlobalFactory` 0 %.

### `internal/bigfft/` (88.7 %)
- `fft_cache.go:70` `Config` 0 %, `fft_cache.go:87` `SetCacheLogger` 0 % (config getters/setters).
- `fft_cache.go:366` `TransformCached` **27.3 %** — seul ~1/3 des branches couvert. **P2**.
- `fft_poly.go:405` `Clone` 0 %.
- `fft_recursion.go:45/55` `SetFFTParallelismConfig`/`GetFFTParallelismConfig` 0 % (toggles globaux).
- `pool_warming.go:116` `EnsurePoolsWarmed` 0 % (fonction publique de warm-up).
- `pool.go:419` `acquireFFTState` 69.2 % (chemins de fallback).

### `internal/cli/` (84.7 %)
- `presenter.go:25,47,92,101,107,112,121` — **7 méthodes adapter à 0 %** (`DisplayProgress`, `PresentComparisonTable`, `padRight`, `PresentResult`, `FormatDuration`, `HandleError`, `DisplayMemoryStats`). Adaptateur thin pour `orchestration` ; les fonctions sous-jacentes sont testées séparément, mais l'adapter lui-même n'est jamais couvert. **P2 F-C2**.

### `internal/app/` (91.7 %)
- `app.go:121` `runTUI` 0 % — entrypoint TUI, justifié (lance Bubble Tea), mais aucun harnais d'intégration headless.

### `internal/config/` (89.4 %)
- `thresholds.go:17,33,74,99` — **4 fonctions à 0 %** : `ApplyAdaptiveThresholds`, `EstimateOptimalParallelThreshold`, `EstimateOptimalFFTThreshold`, `EstimateOptimalStrassenThreshold`. Si ces fonctions sont obsolètes/duplicates de `internal/calibration/adaptive.go`, suspicion code mort. **P1 F-C3**.

## `t.Parallel()` — analyse par package

| Package | `func Test` | `t.Parallel()` | Ratio | Verdict |
|---|---|---|---|---|
| `cmd/fibcalc` | 6 | 19 | 3.17 | OK (sub-tests) |
| `cmd/generate-golden` | 3 | 0 | 0.00 | ⚠ pas de Parallel |
| `test/e2e` | 12 | 4 | 0.33 | ⚠ |
| `internal/app` | 27 | 61 | 2.26 | OK |
| `internal/calibration` | 36 | 38 | 1.06 | OK |
| `internal/bigfft` | 113 | 170 | 1.50 | OK |
| `internal/cli` | 20 | 20 | 1.00 | acceptable |
| `internal/config` | ~26 | ~57 | 2.19 | OK |
| `internal/fibonacci` | 94 | 143 | 1.52 | OK |
| `internal/orchestration` | 16 | 12 | 0.75 | ⚠ |
| `internal/progress` | 23 | 27 | 1.17 | OK |
| `internal/tui` | **145** | **23** (1 fichier) | **0.16** | **❌ P1 F-C1** |
| `internal/ui` | 6 | n/c | n/c | small |

**Total** : 610 `func Test`, 664 `t.Parallel()` → ratio 1.09 (sous-tests inclus).

`internal/tui` : 9 fichiers sur 10 sans aucun `t.Parallel()` (`bridge_test.go`, `chart_test.go`, `footer_test.go`, `header_test.go`, `keymap_test.go`, `logs_test.go`, `metrics_test.go`, `model_test.go`, `sparkline_test.go`). Seul `cli_flags_test.go` (23 occurrences) le fait. À justifier ou corriger.

## `t.Skip` / `testing.Short()` — recensement

11 occurrences au total. Toutes **acceptables** (skip large/stress/SIMD-spécifique) :

| Fichier | Ligne | Motif | Verdict |
|---|---|---|---|
| `cmd/generate-golden/main_test.go` | 72 | large values en short mode | OK |
| `internal/config/hardware_test.go` | 55 | défauts SIMD 64-bit only | OK |
| `internal/tui/logs_test.go` | 225, 292 | stress test en short | OK |
| `internal/tui/logs_test.go` | 283 | viewport > content (défensif) | OK, mais `t.Skip` masque potentiellement un bug dimensionnel — voir F-C7 |
| `internal/bigfft/fft_precision_test.go` | 80 | very large en short | OK |
| `internal/bigfft/sqr_test.go` | 76, 240 | very large en short | OK |
| `internal/fibonacci/fft_squaring_test.go` | 105 | very large en short | OK |
| `internal/fibonacci/fibonacci_fuzz_test.go` | 286, 289 | early-skip dans fuzz (modVal/n hors plage) | OK (idiomatique fuzz) |

Aucun `t.Skip` non justifié. Pas de `t.Skip("flaky")`, pas de skip masquant un bug.

## Golden tests

- **Fichier** : `internal/fibonacci/testdata/fibonacci_golden.json` (94 lignes, 7 414 octets).
- **Cas couverts** (23) : `n ∈ {0, 1, 2, 3, 4, 5, 10, 20, 50, 92, 93, 94, 100, 128, 256, 512, 1000, 1024, 2000, 2048, 5000, 8192, 10000}`.
  - **Couvre** : base case n=0, n=1, frontière uint64 (92, 93, 94), puissances de 2 jusqu'à 8192, n=10000.
  - **Manque** : valeurs négatives (impossible, type `uint64` côté générateur — donc N/A), valeurs très grandes (n > 10 000) où FFT est dominant. La clause "max" évoquée dans la mission est implicitement `n=10000` (limite générateur). **Suggestion P3** : ajouter quelques cas FFT-bound (`n=100000`, `n=500000`) pour valider golden au-delà du seuil FFT (~500k bits ≈ n≈700k).
- **Générateur** : `cmd/generate-golden/main.go` — Oracle naïf itératif `math/big`, déterministe. Cible `targets[]` codée en dur (lignes 46-50). Couvert à 29.4 % uniquement (`fibBig` 100 %, `main()` 0 %). **F-C4**.
- **Consommation** : `internal/fibonacci/fibonacci_golden_test.go` exécute 3 calculateurs (`FastDoubling`, `MatrixExp`, `FFTBased`) × 23 cas, tous avec `t.Parallel()` (lignes 43, 48). Excellent.
- **Intégrité** : aucun checksum/hash dans le repo. Le générateur ne signe pas le fichier. Risque faible (Oracle naïf vérifiable indépendamment) mais possible d'ajouter `sha256` du fichier en commentaire ou test (P3).

## Fuzzing

5 cibles, toutes dans `internal/fibonacci/fibonacci_fuzz_test.go` :

| Fuzz | Plage | Identités vérifiées |
|---|---|---|
| `FuzzFastDoublingConsistency` | n ≤ 50 000 | FastDoubling vs Matrix |
| `FuzzFFTBasedConsistency` | n ≤ 20 000 | FFT vs FastDoubling |
| `FuzzFibonacciIdentities` | n ≤ 10 000 | Doubling, d'Ocagne, Cassini, Addition |
| `FuzzFastDoublingMod` | n ≤ 100 000, mod ≤ 1e9 | range output |
| `FuzzProgressMonotonicity` | 10 ≤ n ≤ 20 000 | progression croissante |

**Aucun corpus seed persistant** (`internal/fibonacci/testdata/fuzz/` absent — seul `fibonacci_golden.json` est dans `testdata/`). Tout repose sur `f.Add()`. Pas de regression corpus, donc les crashes trouvés en CI ne seraient pas rejoués automatiquement. **F-C8**.

**Manques évidents** (cibles fuzz à ajouter — voir Annexe) :
1. `internal/bigfft/scan.go::FromDecimalString` — parser de chaîne décimale, candidat parfait pour fuzz (entrées arbitraires).
2. `internal/config/...` — `ParseMemoryLimit` (`memory/budget.go`), parsing flags CLI.
3. `internal/calibration/profile.go::loadProfile` — décodeur JSON.
4. `internal/cli/completion` — génération de complétions shell, parsing arguments.

## Tests fragiles / flaky

Sleeps fixes recensés (5 occurrences) :

| Fichier | Ligne | Sleep | Risque |
|---|---|---|---|
| `internal/calibration/calibration_advanced_test.go` | 100 | 10ms | Faible — test d'annulation contexte, sleep avant `cancel()` |
| `internal/cli/ui_test.go` | 156 | 10ms | Faible — coordination spinner/channel |
| `internal/cli/ui_advanced_test.go` | 43 | 50ms | **Moyen** — commentaire `enough to trigger ticker potentially` ; flaky potentiel sur runner lent |
| `internal/fibonacci/generator_test.go` | 327 | 5ms | Faible — laisse expirer `WithTimeout(1ms)` |

Tests utilisant `time.Now()` non mocké (10 occurrences dans `internal/tui/metrics_test.go`, `model_test.go`, `app_test.go`, `calibration/profile_test.go`) :
- Usage **passé** (e.g. `Add(-1 * time.Second)`) pour simuler horodatage : robuste.
- Usage **présent** (`time.Now()`) dans `app_test.go:588` (`CalibratedAt: time.Now()`) : non flaky.
- Pas d'horloge injectée (`Clock` interface) → si la précision sub-seconde devient critique, refactor `internal/format/progress_eta.go::UpdateWithETA` (32 % cov, dépend de `time`).

**Verdict** : pas de test ostensiblement flaky. `ui_advanced_test.go:43` à surveiller (P3).

## Mocks

- **Pas de gomock généré** : aucun `//go:generate mockgen` dans le code, aucun import `go.uber.org/mock` (vérifié via `grep -rn`).
- **Cible Makefile** `generate-mocks: go generate ./...` (Makefile L271) et `install-mockgen` (L274) **présents mais inertes** — aucun `go:generate` ne consomme mockgen. **F-C6**.
- **Mocks à la main** dans 14 fichiers, principalement :
  - `internal/fibonacci/testing.go` (`MockCalculator`, `TestFactory`)
  - `internal/calibration/calibration_test.go`, `calibration_advanced_test.go` (`MockBlockingCalculator`)
  - `internal/orchestration/orchestrator_test.go`, `internal/cli/ui_test.go` (`MockSpinner`)
  - `internal/progress/observer_test.go` (49 occurrences `mock`)
- Cohérents et lisibles. La cible Makefile est trompeuse (P2 F-C6).

## Bench reproductibility

- **Cible Makefile `bench-versioned`** (L168-182) : excellente — capture date UTC, SHA Git, `git describe`, version Go, commande exacte (`-count=3 -benchtime=2s`), output dans `build/bench/snapshot-<timestamp>.txt`. Conformité bonne pratique `benchstat`.
- **Doc `docs/PERFORMANCE.md`** (331 lignes) : présente baseline machine (Ryzen 9 5900X, 32 GB DDR4, Linux 6.1, Go 1.25.0) et table de référence. Décrit `make bench-versioned`. **OK**.
- **Limites** :
  - `bench-versioned` ne cible que `BenchmarkFastDoubling` (L181) — couverture restreinte. Étendre à `Benchmark...` global ou ajouter `-run=^$` pour skip tests (déjà implicite).
  - Pas de hook `benchstat` automatique pour comparaison `before/after` (juste écriture brute).
  - **Note baseline actuelle** : `bench/baseline/benchmark.txt` fait 25 MB — issu d'un run `make benchmark` complet (sans `-count`/snapshot versionné). Indiquer dans le rapport quand utiliser l'un vs l'autre.

## Findings

### F-C1 : `internal/tui` — quasi-aucun `t.Parallel()`
- **Sévérité** : P1
- **Fichier(s)** : `internal/tui/{bridge,chart,footer,header,keymap,logs,metrics,model,sparkline}_test.go` (9 fichiers, 122 `func Test`)
- **Diagnostic** : 23 `t.Parallel()` au total, tous dans `cli_flags_test.go`. Les autres tests ne paralléllisent pas. La couverture (87.5 %) est bonne, mais la durée (10.6 s, plus long du repo) est largement amortissable. Probable cause : usage d'état global (`initTUIStyles`, `GetCurrentTheme`, `lastUpdate`) qui rend le parallélisme dangereux.
- **Recommandation** :
  1. Auditer chaque test pour détecter les vrais conflits (probables : `styles.go`, `themes.go`).
  2. Marquer parallélisables ceux qui n'écrivent que `Model` local (la majorité de `header_test.go`, `footer_test.go`, `sparkline_test.go`, `keymap_test.go`).
  3. Documenter les exclusions explicitement (`// non-parallèle: muta state global X`).
- **Effort** : M

### F-C2 : `internal/cli/presenter.go` — 0 % sur 7 méthodes adapter
- **Sévérité** : P2
- **Fichier(s)** : `internal/cli/presenter.go:25,47,92,101,107,112,121`
- **Diagnostic** : adaptateur entre `cli` et `orchestration` (implémente `orchestration.ProgressReporter`/`ResultPresenter`). 7 méthodes à 0 % alors que les fonctions sous-jacentes (`DisplayProgress`, `DisplayResult`, …) sont testées via `ui_test.go`. Les adapters wrappent simplement des appels — risque faible mais couverture trompeuse.
- **Recommandation** : ajouter un seul `presenter_test.go` qui instancie `CLIProgressReporter{}` / `CLIResultPresenter{}` et appelle chaque méthode avec `io.Discard`. ROI : passer de 0 → 100 % sur 7 fonctions, ~30 LOC.
- **Effort** : S

### F-C3 : `internal/config/thresholds.go` — 4 fonctions à 0 % (suspicion code mort)
- **Sévérité** : P1
- **Fichier(s)** : `internal/config/thresholds.go:17,33,74,99`
- **Diagnostic** : `ApplyAdaptiveThresholds`, `EstimateOptimalParallelThreshold`, `EstimateOptimalFFTThreshold`, `EstimateOptimalStrassenThreshold` sont **toutes à 0 %**. Or, `internal/calibration/adaptive.go` expose `EstimateOptimalParallelThreshold/FFTThreshold/StrassenThreshold` (toutes à 100 %). Forte suspicion de duplication/code mort.
- **Recommandation** :
  1. Croiser via `gopls references` ou `staticcheck -unused` pour confirmer.
  2. Si dupliqué : supprimer `thresholds.go` et router via `internal/calibration`.
  3. Sinon : ajouter tests ou justifier l'API publique non testée.
- **Effort** : S (vérification) → M (suppression/refactor si confirmé).

### F-C4 : `cmd/generate-golden` — couverture 29.4 %, cas figés en dur
- **Sévérité** : P2
- **Fichier(s)** : `cmd/generate-golden/main.go:22,46-50`
- **Diagnostic** : `main()` à 0 % (entrypoint, acceptable), mais la liste `targets[]` est codée en dur sans test paramétré. Pas de validation cross-check (e.g. relancer `fibBig` vs Fast Doubling pour détecter dérive Oracle).
- **Recommandation** :
  1. Ajouter un test qui invoque `main()` via `os.Args` factice + dossier temporaire + comparaison déterministe.
  2. Inclure `n=100000` / `n=500000` (FFT-dominant) dans `targets[]` pour étendre golden au-delà du seuil FFT (voir manque section Golden).
- **Effort** : S

### F-C5 : `internal/fibonacci/testing.go` — `TestFactory` exporté mais 0 % de couverture
- **Sévérité** : P2
- **Fichier(s)** : `internal/fibonacci/testing.go:46-96` (NewTestFactory, Create, Get, List, Register, GetAll, UnknownCalculatorError.Error)
- **Diagnostic** : ces helpers sont conçus pour les tests externes (commentaire ligne 33 : *"intended for use in tests where mock calculators are needed"*). Le seul `testing_test.go` (2 Tests, 14 mocks) ne les exerce pas exhaustivement. L'API est exportée pour usage externe mais aucun consommateur n'a été détecté.
- **Recommandation** : soit ajouter un test interne complet (`testing_test.go`), soit déplacer dans `fibonaccitest` (déjà 100 % couvert) — alignement avec convention "package_test" externe.
- **Effort** : S

### F-C6 : Cibles Makefile `generate-mocks` / `install-mockgen` inertes
- **Sévérité** : P2
- **Fichier(s)** : `Makefile:271-276`
- **Diagnostic** : aucun `//go:generate mockgen ...` dans le repo, aucun import `go.uber.org/mock`. La cible `make generate-mocks` est silencieuse (aucun fichier généré). Confusion possible pour contributeurs.
- **Recommandation** : (au choix) soit supprimer les deux cibles, soit annoter les interfaces critiques (`Calculator`, `Multiplier`, `DoublingStepExecutor`) avec des directives `go:generate`. Documenter la décision dans `CONTRIBUTING.md`.
- **Effort** : S

### F-C7 : `internal/tui/logs_test.go:283` — `t.Skip("viewport is larger than content")`
- **Sévérité** : P3
- **Fichier(s)** : `internal/tui/logs_test.go:280-287`
- **Diagnostic** : le test bypass silencieusement quand `viewport.AtBottom()` est vrai. Si la dimension viewport change à l'avenir (refactor), le test deviendra **un skip permanent invisible**.
- **Recommandation** : forcer la taille via `SetSize(60, 5)` (déjà fait L266) **et** valider que le contenu dépasse (`require len(entries) > height`). Le `t.Skip` devient une `t.Fatal` si la précondition est violée.
- **Effort** : S

### F-C8 : Aucun corpus seed persistant pour les fuzz tests
- **Sévérité** : P2
- **Fichier(s)** : `internal/fibonacci/fibonacci_fuzz_test.go` (5 fonctions) ; `internal/fibonacci/testdata/fuzz/` absent
- **Diagnostic** : seuls les `f.Add(...)` en code servent de seed. Tout crash trouvé localement (en exécutant `go test -fuzz`) écrirait dans `testdata/fuzz/FuzzXxx/<hash>` — actuellement non versionné/git-ignoré. Pas de regression corpus partagé.
- **Recommandation** :
  1. Définir une politique CI : `go test -fuzz=Fuzz... -fuzztime=30s` en nightly, commit automatique du corpus.
  2. Versionner `testdata/fuzz/` (ou seulement les seeds curatés).
- **Effort** : M

### F-C9 : Sleep fixe `50ms` dans `ui_advanced_test.go:43`
- **Sévérité** : P3
- **Fichier(s)** : `internal/cli/ui_advanced_test.go:43`
- **Diagnostic** : commentaire du code lui-même (`enough to trigger ticker potentially`) admet l'incertitude. Sur runner CI lent, le ticker peut ne pas se déclencher → assertion `mockS.started` reste valide mais la branche "ticker fired" n'est pas couverte.
- **Recommandation** : injecter une horloge mockable ou exposer un canal `tickFired` pour synchronisation déterministe.
- **Effort** : M

### F-C10 : `bench-versioned` ne benchmark qu'une famille (`BenchmarkFastDoubling`)
- **Sévérité** : P3
- **Fichier(s)** : `Makefile:181`
- **Diagnostic** : la cible reproductible ne couvre pas Matrix Exp., FFT, Strassen, ni `internal/bigfft/`. Pour suivre la régression complète, il faudrait `-bench=.`.
- **Recommandation** : ajouter `bench-versioned-all` (ou paramétrer via `BENCH=...`). Documenter dans `docs/PERFORMANCE.md`.
- **Effort** : S

## Annexe : recommandations fuzz

Cibles candidates absentes (priorité décroissante) :

1. **`FuzzFromDecimalString`** — `internal/bigfft/scan.go:12`
   - Seed : `"0"`, `"1"`, `"123"`, longues chaînes décimales, caractères invalides, vide.
   - Vérifier : pas de panic, round-trip `String() → FromDecimalString → String()` stable, comparaison vs `big.Int.SetString(s, 10)`.

2. **`FuzzParseMemoryLimit`** — `internal/fibonacci/memory/budget.go:40` (couverture 100 % mais pas de fuzz)
   - Seed : `"1KB"`, `"1.5GB"`, `"0"`, `""`, `"abc"`, `"-1MB"`, gros nombres.
   - Vérifier : pas de panic, monotonie (KB < MB < GB), erreurs structurées.

3. **`FuzzLoadProfile`** — `internal/calibration/profile.go:90`
   - Seed : profils JSON valides, JSON malformé, champs inattendus, dates aux limites (Unix epoch, futur).
   - Vérifier : pas de panic, erreurs propres, idempotence `Save → Load`.

4. **`FuzzCompletion`** — `internal/cli/completion*.go`
   - Seed : `"bash"`, `"zsh"`, `""`, `"unknown"`, caractères ANSI.
   - Vérifier : pas de panic sur shells inconnus, sortie déterministe pour shells valides.

5. **`FuzzConfigFromFlags`** — `internal/config/config.go`
   - Seed : combinaisons de flags valides, contradictoires, valeurs aberrantes.
   - Vérifier : `ConfigError` structurée, jamais de `panic`.

**Politique recommandée** : `go test -fuzz=. -fuzztime=30s ./...` en CI nightly, commit auto du corpus si nouveaux crashes (workflow GitHub Actions dédié).

---

*Rapport généré sans modification de code, sur la base de `bench/baseline/coverage.out` (170 KB), `coverage.txt`, `test.txt` (320 KB) et inspection statique du repo.*
