# Calcul de Fibonacci par la Méthode du Doublement et Parallélisation avec Mémoïsation et Benchmark

![Diagramme de l'algorithme de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/DoublingParallel/Sequence%20Diagram.jpeg)

## Table des Matières
- [Introduction](#introduction)
- [Prérequis](#prérequis)
- [Installation](#installation)
- [Fonctionnalités](#fonctionnalités)
- [Architecture du Code](#architecture-du-code)
- [Usage](#usage)
- [Détails Techniques](#détails-techniques)
  - [Méthode de Doublage](#méthode-de-doublage)
  - [Cache LRU](#cache-lru)
  - [Parallélisation](#parallélisation)
- [Benchmark et Performance](#benchmark-et-performance)
- [Améliorations Futures](#améliorations-futures)
- [Auteurs](#auteurs)

## Introduction
Ce programme en Go est conçu pour calculer les nombres de Fibonacci en utilisant des techniques avancées d'optimisation telles que la **méthode de doublage**, la **mémoïsation via un cache LRU** (Least Recently Used), et la **parallélisation avec goroutines** et un **pool de workers**. Ces techniques sont combinées pour réduire le temps de calcul des valeurs élevées de la suite de Fibonacci, tout en optimisant l'utilisation des ressources du système.

L'objectif principal est de présenter une approche efficace pour des calculs intensifs, tout en démontrant la puissance de Go pour la gestion de la concurrence et la parallélisation.

## Prérequis
Avant d'exécuter le programme, assurez-vous que votre système répond aux exigences suivantes :
- **Go 1.16** ou version supérieure est installée sur votre système.
- **Accès à internet** pour télécharger les dépendances du cache LRU.
- Connaissance de base du terminal pour exécuter des commandes.

## Installation
Pour installer et lancer ce projet, suivez les étapes ci-dessous :

1. Clonez ce dépôt :
   ```bash
   git clone https://github.com/votre-utilisateur/fibonacci-go-optimise.git
   cd fibonacci-go-optimise
   ```

2. Installez les dépendances requises :
   ```bash
   go mod tidy
   ```

3. Compilez le programme :
   ```bash
   go build -o fibonacci
   ```

4. Exécutez le programme :
   ```bash
   ./fibonacci
   ```

## Fonctionnalités
- **Calcul optimisé de Fibonacci** utilisant la méthode de doublage.
- **Mémoïsation** avec un **cache LRU** pour améliorer les performances des calculs redondants.
- **Parallélisation** avec **goroutines** et **pool de workers** pour utiliser pleinement les systèmes multi-cœurs.
- **Benchmark** des performances avec des valeurs prédéfinies et une mesure du temps moyen d'exécution.

## Architecture du Code
Le programme est composé des modules suivants :

1. **fibDoubling** : Fonction principale pour calculer les nombres de Fibonacci en utilisant la méthode de doublage.
2. **Cache LRU** : Gère la mémoïsation pour éviter les recalculs inutiles.
3. **Benchmarking** : Mesure le temps de calcul pour différentes valeurs d'indices.
4. **Parallélisation** : Utilisation de goroutines et `sync.WaitGroup` pour paralléliser les calculs.

## Usage
Le programme est conçu pour être exécuté depuis la ligne de commande et calcule les nombres de Fibonacci pour de grandes valeurs d'indices. Voici comment exécuter un benchmark des performances :

1. Lancez le programme en utilisant le fichier compilé :
   ```bash
   ./fibonacci
   ```

2. Le programme commence par initialiser le cache, puis lance plusieurs **workers** qui calculent les nombres de Fibonacci pour des valeurs prédéfinies.
3. Les résultats sont affichés, comprenant le **temps d'exécution moyen** pour chaque valeur de Fibonacci calculée.

## Détails Techniques
### Méthode de Doublage
La méthode de doublage permet de calculer les nombres de Fibonacci à partir des bits de l'indice `n`. Elle utilise des relations mathématiques basées sur les propriétés suivantes :
- `F(2k) = F(k) * [2 * F(k+1) - F(k)]`
- `F(2k + 1) = F(k)^2 + F(k+1)^2`

Ces relations permettent de calculer efficacement des valeurs élevées sans avoir recours à la récursion traditionnelle.

### Cache LRU
Le programme utilise un **cache LRU** (à l'aide de la bibliothèque `hashicorp/golang-lru`) pour mémoriser les valeurs calculées et éviter les recalculs. Le cache est géré de manière **thread-safe** avec des verrous (`sync.RWMutex`) qui permettent de :
- Lire les valeurs en parallèle sans conflit.
- Protéger les écritures pour garantir l'intégrité des données.

### Parallélisation
Pour exploiter les processeurs **multi-cœurs**, le programme crée plusieurs **goroutines** qui agissent comme des **workers**. Ces workers effectuent les calculs de Fibonacci en parallèle, réduisant ainsi le temps total d'exécution. La synchronisation est assurée par `sync.WaitGroup`, et un contexte avec **timeout** est utilisé pour éviter que le programme ne s'exécute indéfiniment.

## Benchmark et Performance
Le programme inclut un **benchmark** qui permet de mesurer les performances des techniques d'optimisation mises en œuvre. Voici les étapes principales :

1. **Contexte avec Timeout** : Le benchmark est limité à 10 minutes pour garantir qu'il ne s'exécute pas trop longtemps.
2. **Répétitions** : Pour chaque valeur de Fibonacci calculée, 10 répétitions sont effectuées afin de calculer un **temps moyen** d'exécution.
3. **Rapports** : Chaque worker rapporte le temps d'exécution moyen, permettant d'évaluer les gains de performance apportés par l'utilisation du cache et de la parallélisation.

## Améliorations Futures
Voici quelques suggestions d'améliorations possibles :
- **Gestion Dynamique du Cache** : Au lieu d'avoir une taille fixe, le cache pourrait s'ajuster dynamiquement en fonction des besoins en mémoire.
- **Amélioration du Benchmark** : Ajouter plus de valeurs de `n` pour une meilleure évaluation des performances sur différentes échelles.
- **Interface Utilisateur** : Fournir une interface utilisateur plus conviviale (par exemple, une interface web) pour faciliter les tests.
- **Journalisation** : Ajouter un système de **logs** pour suivre plus précisément les événements, erreurs, et performances.



