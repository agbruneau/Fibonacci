# FibGo Architecture — Référence Détaillée

Ce répertoire contient la documentation architecturale détaillée du projet FibCalc, incluant les diagrammes techniques, les ADR (Architectural Decision Records) et les flux de données.

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
  - Initialisation et injection de dépendances.
  - Boucle de calcul principale (Doubling Loop).
  - Processus de calibration adaptatif.

## 4) Design Patterns et ADR

L'architecture repose sur 14 design patterns documentés ici :

- **[Patterns/](./patterns/) :**
  - **Strategy :** Sélection de l'algorithme de multiplication.
  - **Observer :** Propagation du progrès vers l'Interface.
  - **Decorator :** Enrichissement des calculateurs.
  - *Etc.*

Pour les décisions historiques majeures, consultez les ADR indexés dans **[docs/ARCH.md](../ARCH.md#L714)**.

---
[← Retour à la vue d'ensemble (ARCH.md)](../ARCH.md)
