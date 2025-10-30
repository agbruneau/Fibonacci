# Calculateur haute performance de la suite de Fibonacci

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

## 1. Résumé

Ce projet n’est pas qu’un simple calculateur de Fibonacci ; c’est une **étude de cas** et une implémentation de référence démontrant des techniques avancées d’ingénierie logicielle en Go. L’objectif principal est d’explorer et d’implémenter des algorithmes de calcul pour des entiers très grands, tout en appliquant des optimisations bas niveau et des patterns de conception haut niveau pour atteindre des performances maximales.

## 2. Fonctionnalités principales

*   **Calcul sur très grands entiers** : `math/big` pour la précision arbitraire.
*   **Plusieurs algorithmes** (complexité logarithmique) :
    *   **Fast Doubling** (`fast`)
    *   **Exponentiation matricielle** (`matrix`)
    *   **Fast Doubling basé FFT** (`fft`)
*   **Optimisations de performance avancées** :
    *   **Stratégie "zéro allocation"** : usage intensif de `sync.Pool` (réutilisation d’objets), pression GC minimisée.
    *   **Parallélisme des tâches** : exploitation multi-cœur au-delà d’un seuil configurable.
    *   **Multiplication FFT** : activation adaptative (seuil en bits configurable).
*   **Architecture modulaire et robuste** :
    *   **Séparation des responsabilités (SoC)** : découplage strict logique/présentation/orchestration.
    *   **Gestion du cycle de vie** : `context` pour délais et signaux (arrêt propre).
    *   **Concurrence structurée** : orchestration via `golang.org/x/sync/errgroup`.
*   **Interface CLI riche** :
    *   Animation (spinner) et barre de progression.
    *   Modes comparaison, calibration, affichage détaillé.
    *   Validation de configuration robuste.

## 3. Principes de conception et patterns

Ce projet illustre concrètement plusieurs principes et patterns de conception :

*   **SOLID** :
    *   **Responsabilité unique** : chaque module (`cmd/fibcalc`, `internal/fibonacci`, `internal/config`, `internal/cli`) a un rôle clair.
    *   **Ouvert/Fermé** : `calculatorRegistry` permet d’ajouter des algorithmes sans modifier l’orchestration.
    *   **Inversion de dépendances** : dépendance vers l’interface `Calculator`.
    *   **Séparation d’interface** : `Calculator` (public) vs `coreCalculator` (interne).
*   **Decorator** : `FibCalculator` encapsule un `coreCalculator` pour ajouter des préoccupations transversales (LUT).
*   **Adapter** : adaptation du canal d’UI en callback `ProgressReporter` pour les algorithmes.
*   **Producteur/Consommateur** : envoi asynchrone des progrès via channels.
*   **Registry** : centralisation des implémentations disponibles.
*   **Object Pooling** : `sync.Pool` pour les états de calcul afin d’approcher le "zéro allocation".

## 4. Architecture logicielle

Le projet est structuré en quatre modules principaux :

*   `cmd/fibcalc` : **composition root**. Point d’entrée : parsing des arguments, injection des dépendances, orchestration.
*   `internal/config` : **couche configuration**. Flags CLI et validation.
*   `internal/fibonacci` : **domaine métier**. Algorithmes et optimisations.
*   `internal/cli` : **couche présentation**. Affichage, progression, rendu du résultat.

## 5. Installation et compilation

Le projet utilise les modules Go. Pour compiler l’exécutable :

```bash
go build -o fibcalc ./cmd/fibcalc
```

Un binaire `fibcalc` (ou `fibcalc.exe` sous Windows) sera créé dans le répertoire courant.

## 6. Guide d’utilisation et optimisation des performances

Utilisation de l’exécutable :

```bash
./fibcalc [options]
```

### Options en ligne de commande

| Flag             | Alias       | Description                                                              | Default      |
| ---------------- | ----------- | ------------------------------------------------------------------------ | ----------- |
| `-n`             |             | Index `n` du nombre de Fibonacci à calculer.                         | `250000000` |
| `-algo`          |             | Algorithme : `fast`, `matrix`, `fft`, ou `all` pour comparer.        | `all`       |
| `-timeout`       |             | Durée maximale d’exécution (ex. `10s`, `1m30s`).                      | `5m0s`      |
| `-threshold`     |             | Seuil (en bits) de parallélisation des multiplications.               | `4096`      |
| `-fft-threshold` |             | Seuil (en bits) pour activer la multiplication FFT (0 pour désact.). | `20000`     |
| `-d`             | `--details` | Afficher les détails de performance et les métadonnées du résultat.   | `false`     |
| `-v`             | `--verbose` | Afficher la valeur complète du résultat (très long).                  | `false`     |
| `--calibrate`    |             | Lancer la calibration du seuil de parallélisation optimal.            | `false`     |

### Optimisation des performances

To achieve the best possible performance, a methodical approach is recommended:

#### Étape 1 : Calibration du seuil de parallélisation

Les performances sur de très grands nombres dépendent fortement de l’architecture du processeur. Le projet inclut un mode calibration pour déterminer empiriquement le meilleur seuil de parallélisation (`--threshold`) pour votre machine.

Exécutez la commande suivante :
```bash
./fibcalc --calibrate
```
Le programme testera plusieurs valeurs et proposera une recommandation, par exemple : `✅ Recommandation pour cette machine : --threshold 4096`.

#### Étape 2 : Utiliser des paramètres optimaux

Une fois le seuil optimal déterminé, utilisez-le dans vos calculs.

*   `--threshold` : seuil de parallélisation (calibré), crucial sur machines multi-cœurs.
*   `--fft-threshold` : seuil d’activation de la multiplication FFT, efficace pour des nombres immenses (millions de bits).

#### Étape 3 : Comparaison des algorithmes

Le programme propose trois algorithmes de pointe. Leurs performances peuvent varier. Utilisez le mode comparaison pour identifier le plus rapide selon votre cas.

```bash
./fibcalc -n <a_large_number> -algo all --threshold <calibrated_value>
```
Le programme exécute tous les algorithmes en parallèle et affiche un tableau comparatif. Il effectue aussi une **validation croisée** : en cas de succès, il vérifie l’égalité des résultats pour garantir l’exactitude.

### Exemples d’utilisation

**1. Trouver le paramètre de performance optimal pour votre machine :**
```bash
./fibcalc --calibrate
```

**2. Comparer les algorithmes pour F(10 000 000) avec un seuil de parallélisation calibré à 4096 :**
```bash
./fibcalc -n 10000000 -algo all --threshold 4096 -d
```

**3. Calculer F(250 000 000) avec l’algorithme le plus rapide, affichage détaillé, et un timeout de 10 minutes :**
```bash
# Après avoir déterminé à l’étape 2 que `fast` est le plus rapide
./fibcalc -n 250000000 -algo fast --threshold 4096 -d --timeout 10m
```

## 7. Validation et tests

Le projet dispose d’une suite de tests couvrante pour garantir exactitude et robustesse.

*   **Tests unitaires et d’intégration** :
    ```bash
    go test ./... -v
    ```
    Cette commande exécute tous les tests du projet : parsing de la configuration, cas limites des algorithmes, comportement UI, etc.

*   **Tests de performance (benchmarks)** :
    ```bash
    go test -bench . ./...
    ```
    Cette commande lance les benchmarks pour mesurer latence et allocations mémoire des algorithmes.

*   **Property-Based Testing** :
    Le projet utilise des tests basés sur les propriétés (librairie `gopter`) pour valider des invariants mathématiques, comme **l’identité de Cassini**, offrant un haut niveau de confiance.

## 8. Licence

Ce projet est distribué sous licence MIT. Voir le fichier `LICENSE` pour plus de détails.
