# FibGo Architecture — Référence Détaillée

Ce répertoire contient la documentation architecturale détaillée du projet FibCalc, incluant les diagrammes techniques, les ADR (Architectural Decision Records) et les flux de données.

## 0) Vue interactive — Dashboard knowledge-graph

Pour une exploration interactive complémentaire aux diagrammes statiques :

**[https://agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)**

- **906 nœuds** (fichiers, fonctions, classes, configs, docs) · **3 809 arêtes** (imports, calls, contains, tested_by, documents…)
- **8 couches** architecturales (entry-point, application, presentation, domain, math-kernel, cross-cutting, e2e-tests, project-support)
- **Tour guidé 11 étapes** depuis `cmd/fibcalc/main.go` jusqu'aux tests/CI
- Recherche, filtres par couche/type, layout dynamique (force, ELK, dagre)

Source : [`.understand-anything/knowledge-graph.json`](../../.understand-anything/knowledge-graph.json). Build statique dans [`../dashboard/`](../dashboard/). Régénération : voir [`../BUILD.md#dashboard-statique-github-pages`](../BUILD.md#dashboard-statique-github-pages).

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

L'historique granulaire des décisions héritées (heuristique CPU, backends
de recherche) reste résumé dans **[docs/ARCH.md](../ARCH.md#14-architectural-decision-records-adr)**.

### Gate d'architecture

`internal/arch_test.go` enforce trois invariants Clean Architecture :
`threshold → config`, `errors → format`, `tui → fibonacci` sont
interdits. Tout PR réintroduisant un de ces imports remontants fait
échouer la CI. Détail : [`docs/TESTING.md` §Architecture-Layering Gate](../TESTING.md#architecture-layering-gate).

---
[← Retour à la vue d'ensemble (ARCH.md)](../ARCH.md)
