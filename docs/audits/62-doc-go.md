# 62 — Conformité doc.go par package

Date : 2026-04-28 — Cible : 21 packages production (cf. audit 1.1).

## Inventaire

| Package | doc.go présent | LOC doc | Qualité | Notes |
|---|---|---|---|---|
| `cmd/fibcalc` | non (doc en tête de `main.go`) | 2 | Faible | 2 lignes, pas de sections, pas d'exemple |
| `cmd/generate-golden` | oui | 28 | Excellente | Section « Independent oracle (P2-04) », rationale claire |
| `internal/app` | oui | 30 | Excellente | Role, Invariants, Example godoc |
| `internal/bigfft` | oui | 30 | Excellente | Role, Invariants, Example, renvoi `docs/algorithms/FFT.md` |
| `internal/calibration` | oui | 35 | Excellente | Role, Invariants, Entry points, Example, renvoi `docs/CALIBRATION.md` |
| `internal/cli` | oui | 4 | Faible | 1 phrase, pas de section, pas d'exemple |
| `internal/config` | oui | 5 | Moyenne | Renvois godoc `[ApplyAdaptiveThresholds]`, mais pas de structure |
| `internal/errors` (pkg `apperrors`) | oui | 12 | Bonne | Wrapping guidelines documentés, renvois vers `CalculationError` |
| `internal/fibonacci` | oui | 22 | Excellente | Section « Must* convention », renvoi audit P2-13 |
| `internal/fibonacci/fibonaccitest` | oui | 4 | Moyenne | Concis mais purpose clair, renvois godoc |
| `internal/fibonacci/memory` | oui | 20 | Excellente | 3 concerns détaillés (Arena/Budget/GC) |
| `internal/fibonacci/threshold` | oui | 16 | Excellente | Décrit constantes adaptatives + concurrency |
| `internal/format` | oui | 13 | Bonne | 3 concerns, déclare la pureté du package |
| `internal/metrics` | oui | 17 | Bonne | 2 familles (Indicators/Memory), thread-safety |
| `internal/orchestration` | oui | 4 | Faible | Role OK, manque invariants/exemple |
| `internal/parallel` | oui | 37 | Excellente | Role, Invariants, Example complet |
| `internal/progress` | oui | 18 | Excellente | DTO, callback, Observer, seuil 1 % |
| `internal/sysmon` | oui | 21 | Excellente | Role, Invariants, Example |
| `internal/testutil` | oui | 25 | Excellente | Avertissement « production MUST NOT import » |
| `internal/tui` | oui | 14 | Bonne | Activation flag/env documenté, renvois godoc |
| `internal/ui` | oui | 7 | Bonne | Purpose + role partagé |

Total : 20 `doc.go` présents + 1 doc-on-`main.go` = 21/21 packages avec doc package-level.

## Packages sans doc.go

- `cmd/fibcalc` : la doc package-level vit dans `main.go` (lignes 1-2). Valide pour godoc, mais incohérent avec le standard du projet et trop succinct.
- Aucun autre package ne manque.

## Qualité du contenu

Top 3 exemplaires :
1. `internal/parallel` — Role, Invariants, Example complet (37 LOC).
2. `internal/calibration` — Role, Invariants, Entry points, Example, renvoi `docs/CALIBRATION.md`.
3. `internal/bigfft` — Invariants critiques (sync.Pool, SIMD guard, LRU), renvoi `docs/algorithms/FFT.md`.

Bottom 3 (à enrichir) :
1. `cmd/fibcalc` (2 LOC) — promouvoir vers `doc.go` dédié, ajouter sections Role/Usage.
2. `internal/cli` (4 LOC) — pas d'invariants ni de carte des sous-fichiers.
3. `internal/orchestration` (4 LOC) — manque Role détaillé, Invariants, exemple d'usage du couple ProgressReporter/ResultPresenter.

Renvois `docs/algorithms/` : seul `bigfft` cite `FFT.md`. `internal/fibonacci` ne référence pas `docs/algorithms/FAST_DOUBLING.md` ni `MATRIX.md` — manque de traçabilité algo-doc.

## Commentaires d'API exportée

Sondage (ratio commentaires `// Name ...` / fonctions exportées) :
- `internal/fibonacci/calculator.go` : 22 commentaires / 7 fonctions et méthodes exportées — couverture complète, style godoc respecté (commencent par identifiant, terminent par point).
- `internal/bigfft/allocator.go` : 19 commentaires / 6 fonctions — bonne couverture.
- `internal/config/config.go` : 5 commentaires / 2 fonctions — couverture suffisante.

Forme godoc globalement conforme : phrases commençant par le nom du symbole, points finaux, paragraphes séparés par lignes vides.

## Examples godoc

| Example | Package | Pertinence |
|---|---|---|
| `ExampleMustNewCalculator` | `internal/fibonacci` | Élevée — montre les 3 algos |
| `ExampleDefaultFactory` | `internal/fibonacci` | Élevée — pattern factory + List/Get |
| `ExampleFibCalculator_CalculateWithObservers` | `internal/fibonacci` | Élevée — observer pattern non trivial |
| `Example_smallValues` | `internal/fibonacci` | Moyenne — couvre fast-path n ≤ 93 |
| `ExampleCalculator_Calculate` | `internal/fibonacci` (`fibonacci_test.go:214`) | Moyenne — usage minimaliste |

Limite : tous les Examples sont concentrés dans `internal/fibonacci`. Aucun pour `bigfft`, `calibration`, `parallel`, `progress` alors que ces packages exposent une API non triviale ; les `doc.go` contiennent déjà du pseudo-code sans `// Output:` exécutable.

## Synthèse

- Score conformité : 21/21 packages avec doc package-level (100 %), mais qualité hétérogène (8 excellents, 6 bons, 4 moyens, 3 faibles).
- Top 3 actions :
  1. Créer `cmd/fibcalc/doc.go` (extraire les 2 lignes de `main.go` et étoffer Role/Usage/Exit codes).
  2. Étoffer `internal/cli/doc.go` et `internal/orchestration/doc.go` au modèle `internal/parallel` (Role + Invariants + Example).
  3. Convertir les blocs `# Example` pseudo-code de `bigfft`, `calibration`, `parallel` en `Example*` exécutables (bénéfice : vérifiés par `go test`).
- Action mineure : ajouter renvois `docs/algorithms/FAST_DOUBLING.md` et `MATRIX.md` dans `internal/fibonacci/doc.go`.
