# 21 — Code mort & duplication

Date : 2026-04-28. Audit read-only, sondage par grep.

## Symboles exportés non utilisés (hors leur package + ses tests)

| Symbole | Défini dans | Utilisé hors package ? |
|---|---|---|
| `calibration.EstimateOptimalParallelThreshold` | `internal/calibration/adaptive.go:106` | NON — délègue à `config.X`, appelé seulement dans `adaptive_test.go` |
| `calibration.EstimateOptimalFFTThreshold` | `internal/calibration/adaptive.go:109` | NON — idem |
| `calibration.EstimateOptimalStrassenThreshold` | `internal/calibration/adaptive.go:112` | NON — idem |
| `fibonacci.UnknownCalculatorError` | `internal/fibonacci/testing.go:92` | NON — utilisé par `TestFactory.Get`, lui-même non référencé hors tests |
| `(*TestFactory).GetAll` | `internal/fibonacci/testing.go:83` | NON — appelé seulement dans `testing_test.go` |
| `(*TestFactory).Register` | `internal/fibonacci/testing.go:77` | NON — no-op, jamais appelé |
| `(*TestFactory).List` | `internal/fibonacci/testing.go:68` | NON |

## Fichiers d'alias / rétrocompat

`internal/fibonacci/progress_aliases.go` (56 LOC) — re-export de 7 types + 7 vars (le commentaire dit 8, en réalité 7) vers `internal/progress`.

| Symbole | Référencé hors fichier ? |
|---|---|
| `ProgressUpdate` (type) | OUI — `internal/tui/{cli_flags_test.go, model_test.go}`, doc |
| `ProgressCallback` (type) | OUI — `internal/fibonacci/calculator.go`, `fibonaccitest/stub.go`, `orchestration/contract_test.go` |
| `ProgressObserver` (type) | NON |
| `ProgressSubject` (type) | NON |
| `ChannelObserver` (type) | NON |
| `LoggingObserver` (type) | NON |
| `NoOpObserver` (type) | NON |
| `NewProgressSubject` (var) | NON |
| `NewChannelObserver` (var) | NON |
| `NewLoggingObserver` (var) | NON |
| `NewNoOpObserver` (var) | NON |
| `CalcTotalWork` (var) | NON |
| `PrecomputePowers4` (var) | NON |
| `ReportStepProgress` (var) | NON |

Bilan : 5 types + 7 vars (sur 14) jamais référencés. Seuls `ProgressUpdate` et `ProgressCallback` justifient encore le fichier — et les consommateurs pourraient importer `internal/progress` directement (sauf TUI qui ne devrait pas selon ISP, à arbitrer).

## Duplications confirmées

### MockCalculator / mocks de `fibonacci.Calculator` — 6 occurrences

| Fichier | Type | LOC env. | Notes |
|---|---|---|---|
| `internal/fibonacci/testing.go:10` | `MockCalculator` (production !) | ~30 | Exporté, exposé hors `_test.go`. Anti-pattern. Couplé à `TestFactory`. |
| `internal/orchestration/orchestrator_test.go:28` | `MockCalculator` | ~30 | Le plus complet (NameFunc + CalculateFunc + adaptateur reporter→channel) |
| `internal/calibration/calibration_test.go:37` | `MockCalculator` | ~45 | Logique métier (durée variable selon thresholds) |
| `internal/tui/cli_flags_test.go:20` | `capturingCalculator` | ~25 | Capture les paramètres |
| `internal/tui/cli_flags_test.go:47` | `blockingCalculator` | ~10 | Bloque sur ctx |
| `internal/tui/model_test.go:19` | `mockCalculator` | ~15 | Minimal |

L'audit 2.3 annonçait 4 occurrences ; il y en a en fait **6** distinctes (deux variantes spécialisées dans `tui`).

### `formatBytes` / `FormatBytes` — duplication exacte

- `internal/format/numbers.go:54` `func FormatBytes(uint64) string` — public
- `internal/fibonacci/memory/budget.go:80` `func formatBytesInternal(uint64) string` — privé, **logique identique** (mêmes seuils, mêmes formats `%.1f`). Doit appeler `format.FormatBytes`.

## Constantes / variables orphelines

Aucune constante exportée orpheline détectée. `MaxPooledBitLen`, `FFTSafetyMarginWords`, `ProgressBufferMultiplier`, `ProgressReportThreshold` sont toutes utilisées (parfois uniquement intra-package, ce qui est normal). Pas de `var X = ...` non-référencé hors tests détecté lors du sondage.

## Synthèse

- **Symboles candidats à suppression** : ~17
  - 12 sur 14 dans `progress_aliases.go` (5 types + 7 vars non utilisés)
  - 3 wrappers `EstimateOptimal*` dans `calibration/adaptive.go`
  - 2 méthodes `TestFactory` (`Register`, `List`) jamais appelées
- **LOC potentiellement supprimables** : **~80 LOC** (≈ 50 LOC dans `progress_aliases.go`, ≈ 15 LOC `adaptive.go` wrappers, ≈ 15 LOC `testing.go` méthodes mortes), plus la fusion mock (~50 LOC après consolidation des 6 mocks).
- **Top 5 actions priorisées** :
  1. **Réduire `progress_aliases.go`** à ses 2 types réellement utilisés (`ProgressUpdate`, `ProgressCallback`) ou idéalement supprimer le fichier en migrant TUI/orchestration vers un import direct de `internal/progress`. Gain : ~40 LOC + dépendance circulaire latente éliminée.
  2. **Déplacer `internal/fibonacci/testing.go` vers `_test.go` ou `fibonaccitest/`** — un mock exporté en code de production pollue l'API publique du package cœur (anti-pattern audit 2.3 confirmé).
  3. **Fusionner les 6 mocks `Calculator`** dans `internal/fibonacci/fibonaccitest/` (à côté de `CoreStub`). Le mock de `orchestration_test.go` est le plus complet — base de la consolidation.
  4. **Supprimer `formatBytesInternal`** dans `memory/budget.go` ; remplacer par `format.FormatBytes`. Gain : 12 LOC + suppression d'une dépendance latente sur `1024^N` vs `1<<N0`.
  5. **Supprimer les 3 wrappers `Estimate*` de `calibration/adaptive.go`** (et leurs tests) ; les appelants utilisent déjà `config.X` directement. Gain : ~15 LOC + 50 LOC de tests redondants.
