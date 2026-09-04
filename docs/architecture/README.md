# FibGo Architecture — Référence Détaillée

Ce répertoire contient la documentation architecturale détaillée du projet FibCalc : diagrammes techniques, flux de données et le relevé de validation des invariants (§5). Les ADR (Architectural Decision Records) ne vivent pas ici mais dans [`docs/adr/`](../adr/) ; la §4 ci-dessous n'en donne que l'index.

> **Partage du travail avec [`docs/ARCH.md`](../ARCH.md).** Ce répertoire **dessine** ;
> `ARCH.md` **narre** — à une exception près : la figure du flux CLI est dessinée dans
> [`ARCH.md` §6](../ARCH.md#6-data-flow-cli-input-to-final-result), en tête de la section
> qui la commente, parce que sa légende est cette section et rien d'autre.
> Les dix figures ci-dessous sont la vue faisant foi sur la *forme*
> du système — arêtes d'import, sous-graphes, ordre des branches — et les sections
> d'`ARCH.md` en sont la légende : le pourquoi, les constantes, les défauts, ce qui n'est
> pas garanti. Il n'existe **pas** deux vues concurrentes de l'architecture : `ARCH.md`
> cite ces figures au lieu d'en redessiner de secondes, et chaque figure indique en pied
> de page la section qui la commente. La correspondance complète est la
> [carte des figures](../ARCH.md#0-carte-des-figures). Un changement de forme se corrige
> **dans la figure d'abord**, puis dans la légende.

> **Format des diagrammes.** Les dix diagrammes sont des fichiers `.md` dont le corps est un
> bloc clôturé `mermaid`. C'est le seul format que GitHub rend graphiquement : un fichier
> `.mermaid` ou `.mmd` autonome s'affiche en texte brut. Ils portaient l'extension `.mermaid`
> jusqu'au 2026-09-04 et ont été convertis pour cette raison ; le repo utilisait déjà cette
> convention ailleurs (`docs/algorithms/*.md`, `docs/TUI_GUIDE.md`). Les onze blocs du corpus
> — les dix d'ici plus celui de [`ARCH.md` §6](../ARCH.md#6-data-flow-cli-input-to-final-result) —
> ont été passés au parseur `mermaid` v11.17.2 le 2026-09-04 : zéro erreur de syntaxe.

## 1) Diagrammes d'Architecture (C4 Model)

Nous utilisons le modèle C4 pour documenter l'architecture à différents niveaux d'abstraction :

- **[System Context](system-context.md) :** Vue de haut niveau de FibCalc et de ses interactions avec l'utilisateur et le système d'exploitation. — *légende : [`ARCH.md` §1](../ARCH.md#1-project-overview)*
- **[Container Diagram](container-diagram.md) :** Décomposition de l'application en conteneurs logiques (CLI, TUI, Core Engine). Chaque `Rel` entre deux `Container` est un import Go réel — le relevé §5 les compte un à un. — *légende : [`ARCH.md` §2](../ARCH.md#2-high-level-architecture-clean-architecture)*
- **[Component Diagram](component-diagram.md) :** Détail des composants internes du moteur de calcul et de l'orchestration. C'est un `classDiagram` : ses flèches sont des relations de classes, **pas** des imports de packages. — *légende : [`ARCH.md` §4](../ARCH.md#4-core-packages-responsibilities-key-types-interfaces)*

## 2) Graphe des Dépendances

Le projet suit rigoureusement les principes de la **Clean Architecture**. Le graphe suivant illustre les relations entre les packages :

- **[Dependency Graph](dependency-graph.md)** — les 46 imports internes directs du module, un par arête. Reproductible avec la commande `go list` donnée dans le [relevé de validation](./validation/validation-report.md#layer-tightness--dependency-direction). — *légende : [`ARCH.md` §2](../ARCH.md#2-high-level-architecture-clean-architecture) (la règle de superposition) et [§3](../ARCH.md#3-directory-structure) (les répertoires derrière les nœuds)*

## 3) Flux de Données et Chemins d'Exécution

Six `flowchart` retracent les chemins d'exécution critiques, du point d'entrée au résultat :

- **Exécution CLI** — la figure et sa légende sont ensemble dans
  [`ARCH.md` §6](../ARCH.md#6-data-flow-cli-input-to-final-result) : le `flowchart` en tête
  de section, puis dix étapes numérotées qui commentent chacune un sous-graphe.
- **[Flows/](./flows/) :**
  - Exécution [TUI](./flows/tui-flow.md) — *légende : [`ARCH.md` §6, mode TUI](../ARCH.md#tui-mode-figure)*.
  - [Résolution de configuration](./flows/config-flow.md) — *légende : [`ARCH.md` §8](../ARCH.md#configuration-cascade) et [§9](../ARCH.md#9-configuration-and-environment)*.
  - Pipelines algorithmiques : [Fast Doubling](./flows/fastdoubling.md) (*[§7A](../ARCH.md#a-fast-doubling-fastdoublingcalculator)*), [FFT](./flows/fft-pipeline.md) (*[§7C](../ARCH.md#c-fft-based-doubling-fftbasedcalculator)*), [Matrix](./flows/matrix.md) (*[§7B](../ARCH.md#b-matrix-exponentiation-matrixexponentiationcalculator)*).

## 4) Design Patterns et ADR

L'architecture repose sur les design patterns documentés ici :

- **[Patterns/](./patterns/) :**
  - **[Design Patterns inventory](./patterns/design-patterns.md)** — l'**inventaire faisant foi** : 17 patterns (Decorator, Strategy, ISP, Factory/Registry, Observer, Template Method, Facade, Adapter, Object Pool, Arena Allocator, Bump Allocator, LRU Cache, Circuit Breaker, Dynamic Threshold Adjustment, Zero-Copy Result Return, Generics with Pointer Constraints, GC Controller) et 5 mécanismes d'ingénierie, avec la raison d'être et le site d'implémentation de chacun. Fusion (2026-09-04) de cette table et de celle que `ARCH.md` §5 tenait en parallèle ; il n'y a plus qu'une liste, et [`ARCH.md` §5](../ARCH.md#5-design-patterns) y renvoie.
  - **[Hiérarchie des interfaces](./patterns/interface-hierarchy.md)** — les interfaces clés et leurs implémentations. Commentée par [`ARCH.md` §5](../ARCH.md#5-design-patterns) et [§8](../ARCH.md#presentation-layer-integration).

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
| [0006](../adr/0006-fft-recursion-cancellation.md) | Annulation récursion FFT — report au token par-appel (FFTContext) | Accepted ⚠ *objet retiré du code* |
| [0007](../adr/0007-pool-pointer-vs-value.md) | SA6002 (`sync.Pool.Put` de slice) — décision mesurée | Accepted |
| [0008](../adr/0008-audit-2026-06-rejected-candidates.md) | Audit de refactorisation 2026-06 — candidats rejetés après vérification | Accepted |
| [0009](../adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) | Audit 2026-07 — purge bigfft, rétention oracle, rejet puis adoption ×10 (addendum R4) | Accepted |
| [0010](../adr/0010-audit-2026-09-decisions.md) | Audit 2026-09 — précédence des seuils explicites, DTM opt-in, lint bloquant, candidats rejetés sur mesure | Accepted |

⚠ **ADR-0006 porte « Accepted » et son objet n'est plus dans l'arbre.** L'API opt-in `FFTContext`
(`NewFFTContext`, `*WithContext`, `fourierRecursiveCtx`) a été **retirée** — zéro occurrence dans
`internal/bigfft/` au relevé du 2026-08-08 —, la migration qu'elle préparait ayant été classée
WONT-FIX par [ADR-0004 §B1](../adr/0004-backlog-decisions.md) ; le retrait est consigné à
[`CHANGELOG.md`](../../CHANGELOG.md) et le code se relit à l'historique git. ⚠ *Un ADR décrit une
décision datée, non l'état du code : celui-ci reste exact comme décision et cesse d'être vérifiable
à la source.* **Changer son statut est une décision de mainteneur, pas une resynchronisation de
documentation — elle n'est pas prise ici.**

L'historique granulaire des décisions héritées (heuristique CPU, backends
de recherche) reste résumé dans **[docs/ARCH.md](../ARCH.md#14-architectural-decision-records-adr)**.

⚠ **ADR-0001 a changé de sens pratique sans changer de statut.** Le
`DynamicThresholdManager` avait été conservé (KEEP) sur la foi d'un gain de
5-6 % à F(10M) ; l'audit 2026-09 (M-04) a constaté qu'aucun chemin de
production ne l'activait, l'a câblé derrière `--dynamic-thresholds`, et la
mesure faite à travers ce flag (`-count=8`) **ne reproduit pas** le gain —
d'où un défaut à `false`. Voir la note datée en fin d'[ADR-0001](../adr/0001-dtm-decision.md).

### Gate d'architecture

`internal/arch_test.go` enforce cinq invariants Clean Architecture :
`threshold → config`, `errors → format`, `tui → fibonacci`,
`orchestration → format` (APP-10) et `config → fibonacci`/`config → bigfft`
(ARCH-02) sont interdits. Tout PR réintroduisant
un de ces imports remontants fait échouer `make test` (ou
`go test ./internal/`). Détail : [`docs/TESTING.md` §Architecture-Layering Gate](../TESTING.md#architecture-layering-gate).

## 5) Validation des invariants

- **[Validation/](./validation/) :**
  - **[validation-report.md](./validation/validation-report.md)** — relevé des invariants que la
    documentation affirme et qui ont été confrontés à la source : étanchéité des couches et sens des
    dépendances, affirmations d'interfaces et de patterns, flux d'exécution, note de maintenance.
    Référence vivante, à re-vérifier et mettre à jour sur place quand la structure change.

---
[← Retour à la vue d'ensemble (ARCH.md)](../ARCH.md)
