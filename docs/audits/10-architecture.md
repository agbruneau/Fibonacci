# 10 — Audit Architecture

Audit FibGo (`github.com/agbru/fibcalc`) — Tâche 2.1 (Clean Architecture).
Date : 2026-04-28. Branche : `main`. HEAD : `4c8f0c1`.
Méthode : Read sur `docs/architecture/*` + `internal/**/doc.go`, `go list -f` pour
extraire les imports réels, comparaison documentation vs code.

## Synthèse

- **Conformité Clean Architecture : OK** (4 couches respectées, aucun cycle, sens
  des dépendances domaine-vers-infrastructure correct).
- **Points forts**
  - Aucun cycle d'import détecté (cf. § Cycles).
  - Couche domaine (`fibonacci`, `bigfft`, `calibration`) ne dépend d'aucun
    package de présentation (`cli`, `tui`) ni d'orchestration. `bigfft` est
    parfaitement isolé (zéro import interne).
  - Sous-packages techniques (`fibonacci/memory`, `fibonacci/threshold`,
    `fibonaccitest`) bien factorisés et sans dépendances internes parasites.
  - Interfaces étroites au sens ISP (`Calculator`, `Multiplier`,
    `DoublingStepExecutor`, `ProgressReporter`, `ResultPresenter`,
    `TempAllocator`).
  - `testutil` bien isolé : importé uniquement par les tests, jamais par du
    code de production (cf. invariant doc.go).
  - 20 / 21 packages possèdent un `doc.go` (manque `cmd/fibcalc`).
- **Écarts identifiés**
  1. `internal/errors` importe `internal/format` : c'est cohérent (formatage de
     diagnostics) mais l'inverse du positionnement « leaf » montré dans le
     diagramme de dépendances. Sans gravité, mais à documenter.
  2. Trois packages dépassent 1 000 LOC (`bigfft` 3 316, `fibonacci` 3 179,
     `tui` 1 668, `calibration` 1 303, `cli` 1 160) — voir § god packages.
  3. La documentation `docs/architecture/*` contient plusieurs arêtes
     incorrectes (cf. § Cohérence) malgré le rapport B1 de
     `validation-report.md` qui les déclarait « PASS ».

## Dépendances inter-paquets (par couche)

Données extraites via `go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./...`
puis filtrées sur `github.com/agbru/fibcalc/...`.

```
[Entrée]
cmd/fibcalc                  -> app
cmd/generate-golden          -> (stdlib uniquement)

[Application / Orchestration]
internal/app                 -> calibration, cli, config, errors, fibonacci,
                                fibonacci/memory, orchestration, tui, ui
internal/orchestration       -> errors, fibonacci, fibonacci/memory, format,
                                progress
internal/cli                 -> config, errors, format, metrics, orchestration,
                                progress, ui
internal/tui                 -> config, errors, fibonacci, format, metrics,
                                orchestration, progress, sysmon, ui
internal/config              -> errors, ui

[Domaine / Algorithmes]
internal/fibonacci           -> bigfft, fibonacci/memory, fibonacci/threshold,
                                parallel, progress
internal/fibonacci/memory    -> (rien)
internal/fibonacci/threshold -> (rien)
internal/fibonacci/fibonaccitest -> fibonacci   (test-only)
internal/bigfft              -> (rien)
internal/calibration         -> bigfft, config, errors, fibonacci, format,
                                progress, ui

[Infrastructure / Utilitaires]
internal/errors              -> format
internal/format              -> (rien)
internal/metrics             -> (rien)
internal/parallel            -> (rien)
internal/progress            -> (rien)
internal/sysmon              -> (rien)
internal/ui                  -> (rien)
internal/testutil            -> (rien)   (test-only)
```

Diagramme ASCII simplifié (sens des flèches = « importe ») :

```
                cmd/fibcalc
                     |
                     v
                  internal/app
   +------+----------+----+--------+------+----+
   |      |          |    |        |      |    |
   v      v          v    v        v      v    v
 cli    tui    orchestration   calibration  config  ...
   \    / \         / \           / | \       \
    \  /   \       /   \         /  |  \       \-> errors -> format
     \/     \     /     \       /   |   \
   format   metrics    fibonacci    |   ui
     |        |         /  |  \     |
     v        v        /   |   \    v
   (leaf)  (leaf) bigfft mem thr  (leaf)
                  (leaf)(leaf)(leaf)
                       \  |  /
                        v v v
                     parallel, progress (leaves)
```

Lecture clé : aucune flèche ne remonte d'une couche basse vers une couche
haute. La règle de dépendance Clean Architecture (« inward-only ») est
respectée.

## Cycles détectés

**Aucun cycle.** Le tri topologique des 21 packages est valide :

```
sysmon, parallel, format, metrics, ui, testutil, fibonacci/memory,
fibonacci/threshold, bigfft, progress
  -> errors
  -> config
  -> fibonacci
  -> orchestration, calibration
  -> cli, tui
  -> app
  -> cmd/fibcalc
```

`go list -deps ./...` complète sans erreur ; `go build ./...` ne signale aucune
référence circulaire (sinon la compilation échouerait). `fibonaccitest` et
`testutil` étant test-only, ils ne participent pas au graphe de production.

## Packages à risque (god packages)

Seuils retenus : > 1 000 LOC production **et** plusieurs responsabilités
distinctes.

| Package | LOC | Responsabilités identifiées | Risque |
|---|---|---|---|
| `internal/bigfft` | 3 316 | FFT Schönhage-Strassen, arithmétique Fermat, pool, bump allocator, cache LRU, intrinsics CPU amd64 | **Modéré** — cohésion thématique forte (« multiplication grand entier »), mais 5 sous-domaines dans un même package. Découpe possible (`fft/`, `pool/`, `cache/`, `arith/`). |
| `internal/fibonacci` | 3 179 | Calculator factory + registry, Fast Doubling, Matrix, FFT-based, GMP, framework Doubling/Matrix, stratégies Multiplier, options. 19 fichiers Go. | **Modéré** — déjà partiellement décomposé (`memory/`, `threshold/`). Les sous-thèmes `matrix_*` (3 fichiers, 597 LOC) et `*_framework` pourraient former un sous-package `fibonacci/framework`. |
| `internal/tui` | 1 668 | Modèle Bubble Tea, viewport logs, sparkline, chart, métriques, styles, header/footer/keymap, bridge orchestration. | **Faible** — découpe par composant UI déjà fine (12 fichiers), `model.go` (425 LOC) reste le seul point chaud. |
| `internal/calibration` | 1 303 | Auto-calibration, micro-benchmarks, profile JSON I/O, runner, adaptatif. | **Faible** — découpage interne propre (5 fichiers thématiques). |
| `internal/cli` | 1 160 | Spinner, completion shell, présentation résultats, output, provider. `completion.go` à lui seul fait 520 LOC. | **Faible-modéré** — la complétion shell mériterait son propre sous-package (`internal/cli/completion/`). |

Aucun god package au sens strict (responsabilité unique violée). Tous
respectent l'invariant « package = responsabilité » de `CLAUDE.md`.

## Cohérence avec docs/architecture/

- **Diagrammes à jour** : `validation-report.md` (2026-02-08) déclare 14
  corrections appliquées sur `dependency-graph.mermaid` et marque le check
  « PASS ». Cependant, après vérification croisée avec `go list` au commit
  `4c8f0c1`, **plusieurs arêtes restent incorrectes**.
- **Écarts code/doc**
  1. `dependency-graph.mermaid` montre `cli --> fib` ; le code actuel n'a
     **pas** d'import `internal/cli -> internal/fibonacci`
     (`grep -r "internal/fibonacci" internal/cli` est vide).
  2. `dependency-graph.mermaid` et `container-diagram.mermaid` montrent
     `calib --> cli` ; le code n'a **pas** d'import
     `internal/calibration -> internal/cli`. La calibration utilise
     `format`, `ui`, `progress` mais pas `cli`.
  3. `dependency-graph.mermaid` place `errors` parmi les feuilles
     (« Support Packages — Leaf Nodes ») alors que `errors --> format`.
     Soit déplacer `errors` hors du groupe leaves, soit dupliquer `format`
     en dépendance explicite dans le diagramme.
  4. `container-diagram.mermaid` ne montre pas la dépendance
     `tui --> sysmon`, pourtant présente dans le code (et correctement
     présente dans `dependency-graph.mermaid`).
  5. `component-diagram.mermaid` est cohérent avec les interfaces réelles
     (cf. `Calculator`, `Multiplier`, `DoublingStepExecutor`,
     `TempAllocator`, `ProgressSubject`).
- **`README.md` de `docs/architecture/`** : lien vers `../ARCH.md` ; la
  cartographie y est cohérente avec ce qui est observé dans le code.

## Recommandations (priorisées, non implémentées)

1. **Mettre à jour `docs/architecture/dependency-graph.mermaid`** (P1) :
   retirer les arêtes `cli --> fib` et `calib --> cli` ; sortir `errors`
   du groupe « leaf » ou ajouter explicitement `errors --> format`. Mettre
   à jour `validation-report.md` en conséquence.
2. **Synchroniser `container-diagram.mermaid`** (P2) : retirer
   `Rel(calib, cli, ...)` ; ajouter `Rel(tui, sysmon, ...)`.
3. **Ajouter un `doc.go` à `cmd/fibcalc`** (P2) pour atteindre 21/21 et
   fermer le seul écart relevé en `01-inventory.md`.
4. **Découper `internal/cli/completion.go`** (P3) en sous-package
   `internal/cli/completion/` (520 LOC, registre + 4 générateurs shell) — la
   responsabilité est suffisamment isolée pour gagner en lisibilité sans
   toucher au reste.
5. **Évaluer un sous-package `internal/fibonacci/framework`** (P3) pour
   regrouper `doubling_framework.go`, `matrix_framework.go`, `strategy.go`
   (~539 LOC) ; même rationale que `memory/` et `threshold/` déjà sortis.
6. **Évaluer un découpage `internal/bigfft/{fft,arith,pool,cache}`** (P3) :
   3 316 LOC pour un seul package est défendable (toute l'API est `Mul`),
   mais une décomposition interne aiderait la maintenance et le test
   ciblé. Aucune urgence — pas de cycle ni de fuite de couche.
7. **Documenter explicitement la convention « `errors` peut importer
   `format` »** (P3) dans `docs/ARCH.md` (sinon l'arête restera perçue
   comme une régression Clean Architecture lors d'audits futurs).
