# Calculateur de Suite de Fibonacci de Haute Performance

## 1. Résumé

Ce projet n'est pas simplement un calculateur pour la suite de Fibonacci ; il s'agit d'une **étude de cas** et d'une implémentation de référence démontrant des techniques avancées d'ingénierie logicielle en langage Go. L'objectif principal est d'explorer et de mettre en œuvre des algorithmes de calcul pour de très grands entiers, tout en appliquant des optimisations de bas niveau et des patrons de conception de haut niveau pour atteindre des performances maximales.

Le code est intégralement commenté en français, avec une perspective académique, pour servir de support pédagogique.

## 2. Caractéristiques Principales

*   **Calcul sur de Très Grands Nombres** : Utilisation de `math/big` pour une précision arithmétique arbitraire.
*   **Algorithmes Multiples** : Implémentation de plusieurs algorithmes de complexité logarithmique, notamment :
    *   **Fast Doubling** (`fast`)
    *   **Exponentiation Matricielle** (`matrix`)
*   **Optimisations de Performance** :
    *   **Stratégie "Zéro-Allocation"** : Utilisation intensive de `sync.Pool` pour la réutilisation d'objets (`big.Int`, états de calcul), minimisant la pression sur le ramasse-miettes (Garbage Collector).
    *   **Parallélisme de Tâches** : Exploitation des processeurs multi-cœurs pour paralléliser les multiplications d'entiers au-delà d'un seuil configurable.
    *   **Multiplication par FFT** : Utilisation adaptative de la Transformée de Fourier Rapide pour la multiplication de nombres dépassant un seuil de plusieurs dizaines de milliers de bits.
    *   **Table de Consultation (LUT)** : Résolution en O(1) pour les petits nombres de Fibonacci.
*   **Architecture Modulaire et Robuste** :
    *   **Séparation des Responsabilités (SoC)** : Découplage strict entre la logique métier (`internal/fibonacci`), l'interface utilisateur (`internal/cli`) et l'orchestration (`cmd/fibcalc`).
    *   **Gestion du Cycle de Vie** : Utilisation avancée de `context` pour une gestion propre des timeouts et des signaux d'interruption (graceful shutdown).
    *   **Concurrence Structurée** : Orchestration des calculs parallèles avec `golang.org/x/sync/errgroup`.
*   **Interface en Ligne de Commande (CLI) Riche** :
    *   Barre de progression dynamique et non-bloquante.
    *   Modes de comparaison, de calibration et d'affichage détaillé.
    *   Validation robuste de la configuration.

## 3. Architecture Logicielle

Le projet est structuré en trois modules principaux, respectant le principe de séparation des responsabilités :

*   `cmd/fibcalc`: **La Racine de Composition (Composition Root)**. Ce module est le point d'entrée de l'application. Il est responsable de :
    1.  L'analyse des arguments de la ligne de commande.
    2.  La validation de la configuration.
    3.  L'injection des dépendances.
    4.  L'orchestration de haut niveau du cycle de vie de l'application.

*   `internal/fibonacci`: **Le Domaine Métier**. Ce module contient toute la logique de calcul de Fibonacci.
    *   Il définit les interfaces (`Calculator`, `coreCalculator`) qui découplent les algorithmes de leur orchestration, en application des principes SOLID (Inversion de dépendances, Ségrégation des interfaces).
    *   Il implémente les algorithmes de calcul (`OptimizedFastDoubling`, `MatrixExponentiation`).
    *   Il gère les optimisations de bas niveau (pools d'objets, parallélisme).
    *   Le patron **Décorateur** est utilisé (`FibCalculator`) pour ajouter des fonctionnalités transversales (comme la LUT) de manière transparente aux algorithmes de base.

*   `internal/cli`: **La Couche de Présentation**. Ce module gère toutes les interactions avec l'utilisateur.
    *   Il est responsable de l'affichage de la progression, des résultats et des métadonnées.
    *   Il est conçu pour être non-bloquant et communique avec le reste de l'application via des canaux Go, suivant le patron **Producteur/Consommateur**.

## 4. Installation et Compilation

Le projet utilise les modules Go standards. Pour compiler l'exécutable :

```bash
go build -o fibcalc ./cmd/fibcalc
```

Un binaire nommé `fibcalc` (ou `fibcalc.exe` sur Windows) sera créé dans le répertoire courant.

## 5. Guide d'Utilisation

L'exécutable s'utilise de la manière suivante :

```bash
./fibcalc [options]
```

### Options de la Ligne de Commande

| Flag             | Alias | Description                                                                         | Défaut                  |
| ---------------- | ----- | ----------------------------------------------------------------------------------- | ----------------------- |
| `-n`             |       | L'indice 'n' de la suite de Fibonacci à calculer.                                   | `250000000`             |
| `-algo`          |       | L'algorithme à utiliser. Valeurs : `fast`, `matrix`, ou `all` pour comparer.        | `all`                   |
| `-timeout`       |       | Délai d'exécution maximal (ex: `10s`, `1m30s`).                                     | `5m0s`                  |
| `-threshold`     |       | Seuil (en bits) pour activer la parallélisation des multiplications.                | `2048`                  |
| `-fft-threshold` |       | Seuil (en bits) pour utiliser la multiplication FFT (0 pour désactiver).            | `20000`                 |
| `-d`             | `--details` | Afficher les détails de performance et les métadonnées du résultat.                 | `false`                 |
| `-v`             | `--verbose` | Afficher la valeur complète du résultat (peut être extrêmement long).               | `false`                 |
| `--calibrate`    |       | Exécuter le mode de calibration pour trouver le seuil de parallélisme optimal.      | `false`                 |

### Exemples

**1. Calcul simple de F(1,000,000) avec l'algorithme "Fast Doubling" et affichage des détails :**
```bash
./fibcalc -n 1000000 -algo fast -d
```

**2. Comparaison des performances de tous les algorithmes pour F(10,000,000) :**
```bash
./fibcalc -n 10000000 -algo all
```

**3. Calcul de F(250,000,000) avec affichage du résultat complet et un timeout de 10 minutes :**
```bash
./fibcalc -n 250000000 -algo fast -v -d --timeout 10m
```

## 6. Modes Spéciaux

### Mode Calibration (`--calibrate`)

Ce mode exécute une série de benchmarks sur la machine hôte pour déterminer empiriquement le seuil de parallélisme (`-threshold`) le plus performant. Il ignore l'argument `-n` et utilise une valeur interne optimisée pour la calibration.

```bash
./fibcalc --calibrate
```
Le programme testera différentes valeurs pour le seuil et recommandera la meilleure, qui pourra ensuite être utilisée pour les calculs intensifs.

### Mode Comparaison (`--algo all`)

Ce mode exécute tous les algorithmes enregistrés en parallèle. À la fin, il affiche un tableau comparatif de leurs temps d'exécution et de leur statut. Il effectue également une **validation croisée** : si tous les calculs réussissent, il vérifie que leurs résultats sont identiques. En cas de divergence, une erreur critique est signalée.

## 7. Validation et Tests

Le projet est doté d'une suite de tests complète pour garantir son exactitude et sa robustesse.

*   **Tests Unitaires et d'Intégration** :
    ```bash
    go test ./... -v
    ```
    Cette commande exécute tous les tests du projet, y compris la validation de la logique de parsing, des cas limites des algorithmes, et du comportement de l'UI.

*   **Tests de Performance (Benchmarks)** :
    ```bash
    go test -bench . ./...
    ```
    Cette commande exécute les benchmarks définis pour mesurer la latence et les allocations mémoire des algorithmes sur de grandes entrées.

*   **Tests Basés sur les Propriétés (Property-Based Testing)** :
    Au-delà des tests unitaires classiques, le projet emploie des tests basés sur les propriétés (avec la bibliothèque `gopter`) pour valider des invariants mathématiques. Par exemple, il vérifie que l'**Identité de Cassini** est respectée pour un grand nombre d'entrées générées aléatoirement, offrant un niveau de confiance supérieur quant à l'exactitude des algorithmes.

## 8. Licence

Ce projet est distribué sous la licence MIT. Voir le fichier `LICENSE` pour plus de détails.