# FibGo Architecture — Référence Détaillée

Ce répertoire contient la documentation architecturale détaillée du projet FibCalc : diagrammes techniques et flux de données. Les ADR (Architectural Decision Records) ne vivent pas ici mais dans [`docs/adr/`](../adr/) ; la §4 ci-dessous n'en donne que l'index.

## 0) Vue interactive — Dashboard knowledge-graph

Pour une exploration interactive complémentaire aux diagrammes statiques :

**[https://agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)**

- **1 128 nœuds** (fichiers, fonctions, classes, configs, docs) · **4 782 arêtes** (imports, calls, contains, tested_by, exports, documents…) — comptes mesurés sur le graphe régénéré le 2026-07-06 (commit 6e3ec29) ; à re-vérifier à la prochaine régénération
- **9 couches** architecturales (entry-points, application, core-fibonacci, bigfft, calibration, presentation, support, infra-config, documentation)
- **Tour guidé 12 étapes** depuis `README.md` jusqu'à l'infrastructure et au build
- Recherche, filtres par couche/type, layout dynamique (force, ELK, dagre)

Source : [`docs/dashboard/knowledge-graph.json`](../dashboard/knowledge-graph.json). Build statique dans [`../dashboard/`](../dashboard/). Régénération : voir [`../BUILD.md#dashboard-statique-github-pages`](../BUILD.md#dashboard-statique-github-pages).

## 1) Diagrammes d'Architecture (C4 Model)

Nous utilisons le modèle C4 pour documenter l'architecture à différents niveaux d'abstraction :

- **[System Context](system-context.mermaid) :** Vue de haut niveau de FibCalc et de ses interactions avec l'utilisateur et le système d'exploitation.
- **[Container Diagram](container-diagram.mermaid) :** Décomposition de l'application en conteneurs logiques (CLI, TUI, Core Engine).
- **[Component Diagram](component-diagram.mermaid) :** Détail des composants internes du moteur de calcul et de l'orchestration.

## 2) Graphe des Dépendances

Le projet suit rigoureusement les principes de la **Clean Architecture**. Le graphe suivant illustre les relations entre les packages :

- **[Dependency Graph](dependency-graph.mermaid)**

## 3) Flux de Données et Séquences

Les répertoires suivants contiennent des diagrammes de séquence illustrant les processus critiques :

- **[Flows/](./flows/) :**
  - Exécution CLI (`cli-flow.mermaid`) et TUI (`tui-flow.mermaid`).
  - Résolution de configuration (`config-flow.mermaid`).
  - Pipelines algorithmiques : Fast Doubling (`fastdoubling.mermaid`), FFT (`fft-pipeline.mermaid`), Matrix (`matrix.mermaid`).

## 4) Design Patterns et ADR

L'architecture repose sur les design patterns documentés ici :

- **[Patterns/](./patterns/) :**
  - **[Design Patterns inventory](./patterns/design-patterns.md)** — table des patterns concrets utilisés (Strategy, Factory/Registry, Observer, Object Pool, Bump Allocator, Decorator, Facade, Template Method, LRU Cache, Circuit Breaker, Adapter) avec liens vers les sites d'implémentation.
  - **[interface-hierarchy.mermaid](./patterns/interface-hierarchy.mermaid)** — hiérarchie des interfaces clés.

### ADR — Décisions architecturales courantes

Les Architectural Decision Records vivent dans [`docs/adr/`](../adr/) :

| ADR | Titre | Statut |
|---|---|---|
| [0000](../adr/0000-template.md) | Template | — |
| [0001](../adr/0001-dtm-decision.md) | Sort de `DynamicThresholdManager` vs `internal/calibration/` | Accepted (KEEP) |
| [0002](../adr/0002-recover-strategy.md) | Stratégie `recover()` dans `bigfft` (sentinel post-condition) | Accepted |
| [0003](../adr/0003-globals-vs-context.md) | Globaux `bigfft` mutables → `atomic.Int64` | Accepted |
| [0004](../adr/0004-backlog-decisions.md) | Décisions de backlog formelles post-hardening | Accepted |
| [0005](../adr/0005-gc-control-concurrent.md) | Contrôle GC concurrency-safe (refcount package-level) | Accepted |
| [0006](../adr/0006-fft-recursion-cancellation.md) | Annulation récursion FFT — report au token par-appel (FFTContext) | Accepted |
| [0007](../adr/0007-pool-pointer-vs-value.md) | SA6002 (`sync.Pool.Put` de slice) — décision mesurée | Accepted |
| [0008](../adr/0008-audit-2026-06-rejected-candidates.md) | Audit de refactorisation 2026-06 — candidats rejetés après vérification | Accepted |
| [0009](../adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) | Audit 2026-07 — purge bigfft, rétention oracle, rejet puis adoption ×10 (addendum R4) | Accepted |

L'historique granulaire des décisions héritées (heuristique CPU, backends
de recherche) reste résumé dans **[docs/ARCH.md](../ARCH.md#14-architectural-decision-records-adr)**.

### Gate d'architecture

`internal/arch_test.go` enforce cinq invariants Clean Architecture :
`threshold → config`, `errors → format`, `tui → fibonacci`,
`orchestration → format` (APP-10) et `config → fibonacci`/`config → bigfft`
(ARCH-02) sont interdits. Tout PR réintroduisant
un de ces imports remontants fait échouer `make test` (ou
`go test ./internal/`). Détail : [`docs/TESTING.md` §Architecture-Layering Gate](../TESTING.md#architecture-layering-gate).

---
[← Retour à la vue d'ensemble (ARCH.md)](../ARCH.md)
