# 22 — Conformité conventions FibGo

Date : 2026-04-28 — WD : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo`

## 1. t.Parallel()

| Package | # Tests | # avec t.Parallel() | Couverture |
|---|---:|---:|---:|
| cmd/fibcalc | 6 | 19* | 100 %+ |
| cmd/generate-golden | 5 | 0 | **0 %** |
| internal/app | 27 | 61* | 100 %+ |
| internal/bigfft | 113 | 170* | 100 %+ |
| internal/calibration | 36 | 38* | 100 %+ |
| internal/cli | 27 | 29* | 100 %+ |
| internal/config | 37 | 57* | 100 %+ |
| internal/errors | 14 | 25* | 100 %+ |
| internal/fibonacci | 86 | 127* | 100 %+ |
| internal/fibonacci/fibonaccitest | 3 | 3 | 100 % |
| internal/fibonacci/memory | 15 | 18* | 100 %+ |
| internal/fibonacci/threshold | 14 | 25* | 100 %+ |
| internal/format | 13 | 18* | 100 %+ |
| internal/metrics | 13 | 2 | **15 %** |
| internal/orchestration | 19 | 15 | **79 %** |
| internal/parallel | 3 | 3 | 100 % |
| internal/progress | 23 | 27* | 100 %+ |
| internal/sysmon | 2 | 0 | **0 %** |
| internal/testutil | 1 | 2* | 100 %+ |
| internal/tui | 145 | 159* | 100 %+ |
| internal/ui | 6 | 0 | **0 %** |
| test/e2e | 12 | 4 | **33 %** |

*100 %+ : davantage de `t.Parallel()` que de fonctions `Test*` (sous-tests parallélisés via `t.Run`).

- **Total** : 620 fonctions `Test*`, 802 appels à `t.Parallel()`.
- **Synthèse** : couverture globale ≈ **94 %** au niveau top-level. Cinq packages en retrait : `cmd/generate-golden`, `internal/sysmon`, `internal/ui`, `internal/metrics` (15 %), `test/e2e` (33 %).

## 2. Sentinelles & types d'erreur

- **Types `*Error`** (5) : `ConfigError`, `CalculationError`, `TimeoutError`, `ValidationError`, `MemoryError` dans `internal/errors/errors.go`. Convention respectée.
- **Sentinelles `Err...`** : **aucune** (`grep "^var Err" → 0`). Le projet privilégie systématiquement les types structurés et `fmt.Errorf("%w", err)`. Choix cohérent, pas un écart.
- 50 occurrences `errors.New/Is/As` réparties sur 18 fichiers — usage normal.

## 3. Structure tests

- **Tests** : 620
- **Benchmarks** : 47
- **Examples** : 5
- **Fuzz** : 5 (centralisés dans `internal/fibonacci/fibonacci_fuzz_test.go`)

Bonne couverture de la palette Go ; Examples sous-représentés (5 seulement) — opportunité d'amélioration documentaire.

## 4. Layout par package

- `doc.go` présent dans **20/21** packages.
- **Anomalie unique** : `cmd/fibcalc/` ne contient pas de `doc.go` (déjà signalé dans audit 1.1).
- Séparation `xxx.go` / `xxx_test.go` respectée partout (aucun fichier de test orphelin détecté).

## 5. Usage de context.Context

Sondage sur les trois packages cœur :
- `internal/fibonacci` : 11 fonctions exportées/internes longues prennent `ctx` (Calculate, CalculateCore, ExecuteDoublingLoop, ExecuteMatrixLoop, multiplyMatrices…). Conforme.
- `internal/orchestration` : `ExecuteCalculations(ctx, …)` — conforme.
- `internal/calibration` : `RunCalibration`, `RunCalibrationWithOptions`, `AutoCalibrate`, `QuickCalibrate`, `MicroBenchmark.RunQuick` — toutes acceptent `ctx`. Conforme.
- **Aucune fonction longue exportée détectée sans ctx.**

## 6. Logging zerolog

- `github.com/rs/zerolog v1.35.0` déclaré dans `go.mod`.
- Usage cohérent : 8 fichiers de production utilisent zerolog (`internal/app`, `internal/bigfft/fft_cache`, `internal/fibonacci/{calculator,common,registry,memory/gc_control,threshold/manager}`, `internal/progress/observers`).
- **Aucun import** de `"log"` (stdlib) ou `log/slog` détecté en production.
- `cmd/generate-golden/main.go` utilise zerolog également.
- Conforme.

## 7. Pattern observer (progress)

`internal/progress/observer.go` implémente le pattern Observer canonique :
- Interface `ProgressObserver { Update(int, float64) }` (Observer).
- `ProgressSubject` thread-safe (sync.RWMutex) avec `Register`/`Unregister`/`Notify`/`ObserverCount`.
- Variantes performance : `AsProgressCallback`, `Freeze` (snapshot lock-free + recover panic).
- Pleinement conforme à la directive CLAUDE.md.

## Synthèse

- **Score conformité : 6,5 / 7** — toutes les conventions sont respectées ; seul accroc : couverture `t.Parallel()` inégale dans 5 packages.

### Top 5 écarts
1. `internal/sysmon`, `internal/ui`, `cmd/generate-golden` : 0 % de tests parallèles.
2. `internal/metrics` : 15 % seulement (2/13).
3. `test/e2e` : 33 % (4/12) — acceptable si tests partagent ressources globales.
4. `cmd/fibcalc/` toujours sans `doc.go` (audit 1.1).
5. Examples godoc sous-représentés (5 au total) — opportunité documentaire.
