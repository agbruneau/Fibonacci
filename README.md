# Calculateur Haute Performance pour la Suite de Fibonacci

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage Status](https://img.shields.io/badge/coverage-75.2%25-brightgreen)

## 1. Objectif

Ce projet est un calculateur de Fibonacci haute performance et une étude de cas en ingénierie logicielle avancée avec Go. Il est conçu pour explorer et implémenter des algorithmes efficaces pour la manipulation de très grands entiers, en appliquant des optimisations de bas niveau et des patrons de conception de haut niveau pour maximiser la performance.

Les objectifs principaux sont :
- Servir de référence pour l'implémentation d'algorithmes sophistiqués en Go.
- Démontrer les meilleures pratiques en architecture logicielle, y compris la modularité et la testabilité.
- Fournir un exemple pratique de techniques d'optimisation de la performance.

## 2. Démarrage

Suivez ces étapes pour mettre en service le calculateur de Fibonacci sur votre machine locale.

### Prérequis

- Go 1.25 ou une version ultérieure

### Installation

1. Clonez le dépôt :
   ```bash
   git clone https://github.com/votre-nom-utilisateur/fibcalc.git
   cd fibcalc
   ```

2. Compilez l'exécutable :
   ```bash
   go build -o fibcalc ./cmd/fibcalc
   ```
   Cela créera un binaire `fibcalc` (ou `fibcalc.exe` sur Windows) à la racine du projet.

### Démarrage Rapide

- **Calibrez pour votre machine :**
  ```bash
  ./fibcalc --calibrate
  ```
  Cela déterminera les paramètres de performance optimaux pour votre système.

- **Lancez une comparaison de tous les algorithmes :**
  ```bash
  ./fibcalc -n 10000000 -algo all
  ```

## 3. Fonctionnalités

*   **Support des Grands Nombres** : Utilise `math/big` pour une arithmétique de précision arbitraire.
*   **Algorithmes Multiples** : Implémente plusieurs algorithmes en O(log n) :
    *   **Fast Doubling** (`fast`)
    *   **Exponentiation Matricielle** (`matrix`)
    *   **Fast Doubling Basé sur la FFT** (`fft`)
*   **Optimisations de Performance** :
    *   **Stratégie Zéro-Allocation** : Emploie `sync.Pool` pour minimiser la surcharge du ramasse-miettes.
    *   **Parallélisme** : Tire parti des cœurs multiples pour des performances améliorées.
    *   **Multiplication FFT Adaptative** : Bascule vers la multiplication basée sur la FFT pour les très grands nombres.
*   **Architecture Modulaire** :
    *   **Séparation des Préoccupations** : Découple la logique, la présentation et l'orchestration.
    *   **Arrêt Propre** : Gère le cycle de vie de l'application avec `context`.
    *   **Concurrence Structurée** : Utilise `golang.org/x/sync/errgroup` pour l'orchestration.
*   **CLI Conviviale** :
    *   Spinner et barre de progression pour un retour visuel.
    *   Modes pour la comparaison, la calibration et les résultats détaillés.
    *   Configuration et validation robustes.

## 4. Utilisation

Le calculateur est contrôlé via des drapeaux de ligne de commande :

```bash
./fibcalc [options]
```

### Options

| Drapeau              | Alias       | Description                                                 | Défaut       |
| -------------------- | ----------- | ----------------------------------------------------------- | ------------ |
| `-n`                 |             | Index du nombre de Fibonacci à calculer.                    | `250000000`  |
| `-algo`              |             | Algorithme à utiliser : `fast`, `matrix`, `fft`, ou `all`.  | `all`        |
| `-timeout`           |             | Temps d'exécution maximum (ex: `10s`, `1m30s`).             | `5m`         |
| `-threshold`         |             | Seuil en bits pour paralléliser les multiplications.        | `4096`       |
| `-fft-threshold`     |             | Seuil en bits pour activer la multiplication FFT.           | `20000`      |
| `--strassen-threshold` |           | Seuil en bits pour basculer vers l'algorithme de Strassen.  | `256`        |
| `-d`                 | `--details` | Afficher les détails de performance et les métadonnées.     | `false`      |
| `-v`                 |             | Afficher le résultat complet (peut être très long).         | `false`      |
| `--calibrate`        |             | Calibrer le seuil de parallélisme optimal.                  | `false`      |
| `--auto-calibrate`   |             | Lancer une calibration rapide au démarrage.                 | `false`      |
| `--lang`             |             | Langue pour l'i18n (ex: `fr`, `en`).                        | `en`         |
| `--i18n-dir`         |             | Répertoire des fichiers de traduction (ex: `./locales`).    | `""`         |

### Exemples

- **Calibrez pour votre machine :**
  ```bash
  ./fibcalc --calibrate
  ```

- **Comparez les algorithmes pour F(10,000,000) :**
  ```bash
  ./fibcalc -n 10000000 -algo all -d
  ```

- **Calculez F(250,000,000) avec un timeout de 10 minutes :**
  ```bash
  ./fibcalc -n 250000000 -algo fast -d --timeout 10m
  ```

## 5. Architecture Logicielle

Ce projet est structuré selon les meilleures pratiques de l'ingénierie logicielle Go, en mettant l'accent sur la **modularité** et la **séparation des préoccupations**. Cette approche garantit que la base de code est non seulement performante, mais aussi maintenable, testable et facile à faire évoluer.

L'architecture est organisée comme suit :

-   **`cmd/fibcalc`**: C'est le **point d'entrée** de l'application. Son rôle unique est d'analyser les arguments de la ligne de commande, de construire les dépendances nécessaires (comme les calculateurs et la configuration) et d'orchestrer le déroulement de l'exécution. Il agit comme la "racine de composition" (*composition root*) de l'application.

-   **`internal/config`**: Ce paquet est exclusivement dédié à la **gestion de la configuration**. Il définit la structure `AppConfig`, gère l'analyse des drapeaux de la ligne de commande et assure la validation des entrées utilisateur. En isolant la configuration, nous pouvons facilement modifier ou étendre les options de l'application sans impacter la logique métier.

-   **`internal/fibonacci`**: Le **cœur de la logique métier** réside ici. Ce paquet contient les implémentations des algorithmes de calcul de Fibonacci. Il expose une interface `Calculator` claire, ce qui permet au reste de l'application d'interagir avec les algorithmes de manière agnostique. C'est également là que se trouvent les optimisations de bas niveau (gestion de la mémoire, parallélisme).

-   **`internal/cli`**: La **couche de présentation** est entièrement gérée par ce paquet. Il est responsable de l'affichage du spinner, de la barre de progression et des résultats finaux. En séparant l'interface utilisateur de la logique de calcul, nous pourrions, par exemple, exposer la même logique via une API web sans modifier le paquet `fibonacci`.

-   **`internal/i18n`**: Ce paquet gère l'**internationalisation (i18n)** de l'application. Il permet de charger et d'utiliser des traductions pour les messages affichés à l'utilisateur, rendant l'application accessible à un public plus large.

Cette conception en couches assure un **faible couplage** entre les différents composants du système, ce qui est la clé d'un logiciel robuste et de haute qualité.

## 6. Analyse Algorithmique et Complexité

La complexité `O(log n)` souvent citée pour les algorithmes de Fibonacci rapides se réfère au nombre d'opérations arithmétiques. Cependant, lors de l'utilisation de l'arithmétique de précision arbitraire (`math/big`), le coût de la multiplication `M(k)` pour des nombres de `k` bits devient le facteur dominant. Le nombre de bits dans F(n) est proportionnel à `n`.

La complexité réelle est donc `O(log n * M(n))`.

-   Avec la multiplication de Karatsuba (utilisée par `math/big`), `M(n) ≈ O(n^1.585)`.
-   Avec la multiplication basée sur la FFT, `M(n) ≈ O(n log n)`.

### Dérivation des Formules de Fast Doubling

Les identités de *Fast Doubling* sont dérivées de la forme matricielle :
```
[ F(2k+1) F(2k)   ] = [ F(k+1)²+F(k)²     F(k)(2F(k+1)-F(k)) ]
[ F(2k)   F(2k-1) ]   [ F(k)(2F(k+1)-F(k)) F(k)²+F(k-1)²     ]
```
De là, nous extrayons :
-   `F(2k) = F(k) * (2*F(k+1) - F(k))`
-   `F(2k+1) = F(k+1)² + F(k)²`

## 7. Optimisations de Performance

### Stratégie Zéro-Allocation

-   **Pools d'Objets (`sync.Pool`)**: Pour minimiser la pression sur le ramasse-miettes, les états de calcul (`calculationState`, `matrixState`) sont recyclés. Cela élimine presque toutes les allocations de mémoire dans les boucles de calcul critiques.

### Parallélisme et Seuils

-   **Parallélisme Multi-cœur**: Les multiplications de grands nombres sont exécutées en parallèle sur plusieurs goroutines.
-   **Seuils Empiriques**:
    -   `--threshold` (défaut `4096` bits) : Active le parallélisme. Ce seuil est un compromis entre le coût de la création de goroutines et le gain de la parallélisation.
    -   `--fft-threshold` (défaut `20000` bits) : Active la multiplication FFT. Ce seuil conservateur garantit que la FFT n'est utilisée que lorsque sa complexité asymptotique est avantageuse.
    -   `--strassen-threshold` (défaut `256` bits) : Utilise l'algorithme de Strassen pour la multiplication de matrices, réduisant les multiplications de 8 à 7 au prix d'additions supplémentaires.

## 8. Tests

Le projet inclut une suite de tests robuste pour garantir la correction et la stabilité.

-   **Tests Unitaires**: Valident les cas limites et les petites valeurs de `n`.
-   **Tests de Propriétés**: Utilisent `gopter` pour effectuer des tests basés sur les propriétés, qui vérifient des identités mathématiques (comme l'identité de Cassini : `F(n-1) * F(n+1) - F(n)² = (-1)ⁿ`) pour un grand nombre d'entrées aléatoires.
-   **Benchmarks**: Mesurent la performance des différents algorithmes.

**Exécuter les tests :**
```bash
# Exécuter tous les tests (unitaires et propriétés)
go test ./... -v

# Exécuter les benchmarks
go test -bench . ./...
```

## 9. Licence

Ce projet est sous licence MIT. Voir le fichier `LICENSE` pour plus de détails.
