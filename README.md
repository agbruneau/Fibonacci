# Calculateur de Suite de Fibonacci de Haute Performance

![Go version](https://img.shields.io/badge/Go-1.18+-blue.svg)
![Licence](https://img.shields.io/badge/Licence-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

## 1. Résumé

Ce projet n'est pas un simple calculateur pour la suite de Fibonacci ; il s'agit d'une **étude de cas** et d'une implémentation de référence démontrant des techniques avancées d'ingénierie logicielle en langage Go. L'objectif principal est d'explorer et de mettre en œuvre des algorithmes de calcul pour de très grands entiers, tout en appliquant des optimisations de bas niveau et des patrons de conception de haut niveau pour atteindre des performances maximales.

Le code est intégralement commenté en français, avec une perspective académique, pour servir de support pédagogique.

## 2. Caractéristiques Principales

*   **Calcul sur de Très Grands Nombres** : Utilisation de `math/big` pour une précision arithmétique arbitraire.
*   **Algorithmes Multiples** : Implémentation de plusieurs algorithmes de complexité logarithmique :
    *   **Fast Doubling** (`fast`)
    *   **Exponentiation Matricielle** (`matrix`)
*   **Optimisations de Performance Avancées** :
    *   **Stratégie "Zéro-Allocation"** : Utilisation intensive de `sync.Pool` pour la réutilisation d'objets (`big.Int`, états de calcul), minimisant la pression sur le ramasse-miettes (Garbage Collector).
    *   **Parallélisme de Tâches** : Exploitation des processeurs multi-cœurs pour paralléliser les multiplications d'entiers au-delà d'un seuil configurable.
    *   **Multiplication par FFT** : Utilisation adaptative de la Transformée de Fourier Rapide pour la multiplication de nombres dépassant un seuil de plusieurs dizaines de milliers de bits.
    *   **Table de Consultation (LUT)** : Résolution en O(1) pour les petits nombres de Fibonacci via une table pré-calculée.
*   **Architecture Modulaire et Robuste** :
    *   **Séparation des Responsabilités (SoC)** : Découplage strict entre la logique métier, l'interface utilisateur et l'orchestration.
    *   **Gestion du Cycle de Vie** : Utilisation avancée de `context` pour une gestion propre des timeouts et des signaux d'interruption (graceful shutdown).
    *   **Concurrence Structurée** : Orchestration des calculs parallèles avec `golang.org/x/sync/errgroup`.
*   **Interface en Ligne de Commande (CLI) Riche** :
    *   Barre de progression dynamique et non-bloquante.
    *   Modes de comparaison, de calibration et d'affichage détaillé.
    *   Validation robuste de la configuration.

## 3. Principes et Patrons de Conception

Ce projet sert de démonstration pratique pour plusieurs principes et patrons de conception fondamentaux :

*   **SOLID** :
    *   **Principe de Responsabilité Unique** : Chaque module (`cmd/fibcalc`, `internal/fibonacci`, `internal/cli`) a une responsabilité unique et bien définie.
    *   **Principe Ouvert/Fermé** : Le `calculatorRegistry` permet d'ajouter de nouveaux algorithmes sans modifier le code d'orchestration existant.
    *   **Principe d'Inversion de Dépendances** : Les modules de haut niveau dépendent d'abstractions (`Calculator`) plutôt que d'implémentations concrètes.
    *   **Principe de Ségrégation des Interfaces** : La séparation entre `Calculator` (interface publique) et `coreCalculator` (interface interne) évite de surcharger les implémentations avec des dépendances inutiles.
*   **Patron Décorateur** : La structure `FibCalculator` encapsule un `coreCalculator` pour y ajouter des fonctionnalités transversales (comme l'optimisation par LUT) de manière transparente.
*   **Patron Adaptateur** : `FibCalculator` adapte également l'interface de communication basée sur les canaux (`chan`) en une interface de rappel (`ProgressReporter`) plus simple pour les algorithmes.
*   **Patron Producteur/Consommateur** : Les algorithmes (Producteurs) génèrent des mises à jour de progression qui sont traitées de manière asynchrone par l'UI (Consommateur) via des canaux Go.
*   **Patron Registre (Registry)** : Le `calculatorRegistry` centralise les implémentations d'algorithmes disponibles, favorisant un couplage faible.
*   **Pooling d'Objets** : L'utilisation de `sync.Pool` pour gérer les états de calcul (`calculationState`, `matrixState`) est une optimisation mémoire cruciale pour atteindre la "zéro-allocation".

## 4. Architecture Logicielle

Le projet est structuré en trois modules principaux :

*   `cmd/fibcalc`: **La Racine de Composition (Composition Root)**. Point d'entrée de l'application, responsable de l'analyse des arguments, de la configuration, de l'injection des dépendances et de l'orchestration du cycle de vie.
*   `internal/fibonacci`: **Le Domaine Métier**. Contient toute la logique de calcul, les implémentations des algorithmes et les optimisations de bas niveau.
*   `internal/cli`: **La Couche de Présentation**. Gère toutes les interactions avec l'utilisateur (barre de progression, affichage des résultats).

## 5. Installation et Compilation

Le projet utilise les modules Go standards. Pour compiler l'exécutable :

```bash
go build -o fibcalc ./cmd/fibcalc
```

Un binaire nommé `fibcalc` (ou `fibcalc.exe` sur Windows) sera créé dans le répertoire courant.

## 6. Guide d'Utilisation

L'exécutable s'utilise de la manière suivante :

```bash
./fibcalc [options]
```

### Options de la Ligne de Commande

| Flag             | Alias       | Description                                                              | Défaut      |
| ---------------- | ----------- | ------------------------------------------------------------------------ | ----------- |
| `-n`             |             | L'indice 'n' de la suite de Fibonacci à calculer.                        | `100000000` |
| `-algo`          |             | Algorithme : `fast`, `matrix`, ou `all` pour comparer.                   | `all`       |
| `-timeout`       |             | Délai d'exécution maximal (ex: `10s`, `1m30s`).                          | `5m0s`      |
| `-threshold`     |             | Seuil (en bits) pour paralléliser les multiplications.                   | `2048`      |
| `-fft-threshold` |             | Seuil (en bits) pour utiliser la multiplication FFT (0=désactivé).        | `20000`     |
| `-d`             | `--details` | Afficher les détails de performance et les métadonnées du résultat.      | `false`     |
| `-v`             | `--verbose` | Afficher la valeur complète du résultat (peut être extrêmement long).    | `false`     |
| `--calibrate`    |             | Lancer la calibration pour trouver le seuil de parallélisme optimal.     | `false`     |

### Exemples

**1. Calcul simple de F(1 000 000) avec l'algorithme "Fast Doubling" et affichage des détails :**
```bash
./fibcalc -n 1000000 -algo fast -d
```

**2. Comparaison des performances de tous les algorithmes pour F(10 000 000) :**
```bash
./fibcalc -n 10000000 -algo all
```

**3. Calcul de F(250 000 000) avec affichage du résultat complet et un timeout de 10 minutes :**
```bash
./fibcalc -n 250000000 -algo fast -v -d --timeout 10m
```

## 7. Modes Spéciaux

### Mode Calibration (`--calibrate`)

Ce mode exécute une série de benchmarks sur la machine hôte pour déterminer empiriquement le seuil de parallélisme (`-threshold`) le plus performant. Il ignore l'argument `-n`.

```bash
./fibcalc --calibrate
```
Le programme testera différentes valeurs pour le seuil et recommandera la meilleure pour les calculs intensifs.

### Mode Comparaison (`-algo all`)

Ce mode exécute tous les algorithmes enregistrés en parallèle et affiche un tableau comparatif de leurs performances. Il effectue également une **validation croisée** : si tous les calculs réussissent, il vérifie que leurs résultats sont identiques. En cas de divergence, une erreur critique est signalée.

## 8. Validation et Tests

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
    Cette commande exécute les benchmarks pour mesurer la latence et les allocations mémoire des algorithmes.

*   **Tests Basés sur les Propriétés (Property-Based Testing)** :
    Le projet emploie des tests basés sur les propriétés (avec la bibliothèque `gopter`) pour valider des invariants mathématiques, comme l'**Identité de Cassini**, offrant un niveau de confiance supérieur quant à l'exactitude des algorithmes.

## 9. Licence

Ce projet est distribué sous la licence MIT. Voir le fichier `LICENSE` pour plus de détails.
