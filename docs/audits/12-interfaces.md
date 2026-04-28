# 12 — Interfaces (ISP)

Audit de conformité au principe de ségrégation d'interface (ISP) revendiqué dans `CLAUDE.md`.
Périmètre : tous les `*.go` non-`_test.go`, hors vendored. Date : 2026-04-28.

## Inventaire

| Interface | Package | # méthodes | # implémentations (prod / test) | Statut |
|---|---|---|---|---|
| `Multiplier` | `internal/fibonacci` | 3 (`Multiply`, `Square`, `Name`) | 2 / 0 | Saine |
| `DoublingStepExecutor` | `internal/fibonacci` | 4 (embed `Multiplier` + `ExecuteStep`) | 2 / 0 | Saine |
| `CoreCalculator` | `internal/fibonacci` | 2 (`CalculateCore`, `Name`) | 4 / 2 | Saine, point d'extension |
| `Calculator` | `internal/fibonacci` | 2 (`Calculate`, `Name`) | 1 / 5 | Frontière publique |
| `CalculatorFactory` | `internal/fibonacci` | 5 (`Create`, `Get`, `List`, `Register`, `GetAll`) | 2 (Default + Test) / 0 | Limite haute ISP |
| `task` | `internal/fibonacci` (non exporté) | 1 (`execute`) | 2 / 0 | Saine (généricité interne) |
| `TempAllocator` | `internal/bigfft` | 2 (`AllocFermatTemp`, `AllocFermatSlice`) | 2 / 0 | Saine |
| `ProgressObserver` | `internal/progress` | 1 (`Update`) | 3 / 2 | Excellente |
| `ProgressReporter` | `internal/orchestration` | 1 (`DisplayProgress`) | 3 / 0 | Excellente |
| `ResultPresenter` | `internal/orchestration` | 2 (`PresentComparisonTable`, `PresentResult`) | 2 / 1 | Saine |
| `ErrorHandler` | `internal/orchestration` | 1 (`HandleError`) | 2 / 1 | Excellente |
| `ColorProvider` | `internal/errors` (apperrors) | 2 (`Yellow`, `Reset`) | 2 / 2 | Saine, anti-cycle |
| `Spinner` | `internal/cli` | 3 (`Start`, `Stop`, `UpdateSuffix`) | 1 / 1 | Frontière + mock |

Légende production : implémentations dans des fichiers non-`_test.go`. Test : doubles dans `_test.go` ou packages `*test`.

## Interfaces saines (≤3 méthodes, ≥2 impls)

- `Multiplier` (3 méth., 2 impls : `AdaptiveStrategy`, `FFTOnlyStrategy`) — exemple canonique d'ISP, cité dans `CLAUDE.md`.
- `DoublingStepExecutor` (4 méth. composées, 2 impls) — extension explicite de `Multiplier`, séparation propre des deux niveaux d'abstraction.
- `CoreCalculator` (2 méth., 4 impls prod : `OptimizedFastDoubling`, `MatrixExponentiation`, `FFTBasedCalculator`, `GMPCalculator` + 2 mocks) — point d'extension Decorator parfait.
- `TempAllocator` (2 méth., 2 impls : `PoolAllocator`, `BumpAllocatorAdapter`) — abstraction strict du minimum nécessaire.
- `ProgressObserver` (1 méth., 3 impls prod : `ChannelObserver`, `LoggingObserver`, `NoOpObserver`) — quasi-Observer.
- `ProgressReporter` (1 méth., 3 impls : `ProgressReporterFunc`, `NullProgressReporter`, `TUIProgressReporter`) — single-method interface idéale.
- `ErrorHandler` (1 méth., 2 impls prod : `CLIResultPresenter`, `TUIResultPresenter`) — single-method.
- `ColorProvider` (2 méth., 2 impls prod : `DefaultColorProvider`, `CLIColorProvider`) — sert également à briser un cycle d'import (cli ↔ apperrors).
- `task` (1 méth.) — astuce interne pour factoriser via génériques, parfaitement justifiée.

## Interfaces volumineuses (>5 méthodes)

Aucune. Le seuil est respecté partout.

`CalculatorFactory` à 5 méthodes est la limite haute. Évaluation : les méthodes forment une cohésion fonctionnelle (CRUD du registre). Pas de découpage immédiat requis, mais à surveiller. Une éventuelle ségrégation pourrait isoler un `CalculatorRegistry` (lecture : `Get`/`List`/`GetAll`) d'un `CalculatorRegistrar` (écriture : `Register`/`Create`) si l'usage diverge.

## Interfaces à 1 seule implémentation prod (sur-abstraction ?)

- `Calculator` (1 prod : `FibCalculator`) — JUSTIFIÉE : c'est la frontière publique de la couche fibonacci consommée par `orchestration`. Cinq mocks de test l'implémentent (calibration ×2, orchestration ×2, tui ×2), ce qui valide son rôle de seam de test.
- `Spinner` (1 prod : `realSpinner`) — JUSTIFIÉE : adaptateur de `briandowns/spinner` + mock test (`MockSpinner`). Découple le code CLI de la dépendance externe pour les tests.

Aucune sur-abstraction nette détectée.

## Doubles de test

Le package `internal/fibonacci/fibonaccitest/` contient :

- `CoreStub` (`stub.go`) — implémente `fibonacci.CoreCalculator` avec champs configurables `NameVal` et `CoreFunc`. Stub pragmatique, paramétrable, suffisant pour la majorité des tests algorithmiques.
- `doc.go` — documentation du package.
- `stub_test.go` — tests internes du stub.

D'autres mocks subsistent dispersés dans des `_test.go` (`MockCalculator` dans `calibration`, `orchestration`, `tui`, `fibonacci/testing.go` ; `mockCoreCalculator` dans `registry_test.go`). Cette duplication trahit un manque de centralisation : `fibonaccitest` ne couvre que `CoreCalculator`, pas `Calculator`.

## Recommandations

1. **Centraliser les doubles `Calculator`** : ajouter un `CalculatorStub` ou `CalculatorSpy` dans `internal/fibonacci/fibonaccitest/` pour remplacer les ~5 `MockCalculator` répliqués dans `calibration/`, `orchestration/`, `tui/`. Réduira la duplication de test.
2. **Surveiller `CalculatorFactory`** : à 5 méthodes, c'est la seule interface proche du seuil. Documenter clairement ses responsabilités ou envisager une ségrégation lecture/écriture si une nouvelle méthode est ajoutée.
3. **Confirmer la suppression de `internal/fibonacci/testing.go`** : ce fichier (hors `_test.go`) expose un `MockCalculator` et `TestFactory` dans le package de prod, ce qui pollue l'API publique du package. À déplacer dans `fibonaccitest/` ou à passer sous build tag `test`.
4. **Aucune action sur les autres interfaces** : la conformité ISP est globalement excellente (médiane de 2 méthodes par interface, 0 interface volumineuse).
