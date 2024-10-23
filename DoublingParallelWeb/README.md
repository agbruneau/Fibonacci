# Calcul de Fibonacci Web par la Méthode du Doublement avec Mémoïsation et Benchmark

![Diagramme de l'algorithme de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/DoublingParallelWeb/Sequence%20Diagram.jpeg)

## Table des Matières
- [Introduction](#introduction)
- [Fonctionnalités](#fonctionnalités)
- [Prérequis](#prérequis)
- [Installation et Utilisation](#installation-et-utilisation)
  - [Installation](#installation)
  - [Utilisation](#utilisation)
- [Détails de l'Implémentation](#détails-de-limplémentation)
  - [Configuration](#configuration)
  - [Cache LRU](#cache-lru)
  - [Algorithme de Doublage Rapide](#algorithme-de-doublage-rapide)
  - [Gestion des Erreurs](#gestion-des-erreurs)
- [API HTTP](#api-http)
  - [Endpoint `/compute`](#endpoint-compute)
- [Optimisations](#optimisations)
- [Métriques et Monitoring](#métriques-et-monitoring)
- [Contribuer](#contribuer)
- [Licence](#licence)

## Introduction
Ce projet implémente un **service web en Go** permettant de calculer des nombres de la suite de **Fibonacci** à l'aide d'une **approche optimisée en parallèle**. Le service a été conçu pour être très performant, capable de gérer de nombreuses requêtes simultanées, grâce à des techniques avancées telles que l'utilisation d'un **cache LRU**, des **pools de mémoire**, et un **algorithme de doublage rapide**.

## Fonctionnalités
- Calcul de nombres de la suite de Fibonacci jusqu'à une valeur maximale configurable.
- Utilisation d'un **cache LRU** (Least Recently Used) pour optimiser les performances des requêtes répétitives.
- Calcul concurrent en utilisant plusieurs **workers** et **pools de big.Int** pour minimiser les allocations mémoire.
- Serveur **HTTP** permettant de faire des requêtes JSON pour calculer les valeurs de Fibonacci.
- **Métriques** pour surveiller les performances, notamment les hits/misses du cache et le temps de calcul.

## Prérequis
- **Go** version 1.18 ou supérieure.
- **Git** pour cloner le dépôt.

## Installation et Utilisation
### Installation
Clonez ce dépôt à l'aide de Git :
```bash
$ git clone <URL_DU_DEPOT>
$ cd <NOM_DU_DEPOT>
```

Ensuite, compilez le projet :
```bash
$ go build DoublingParallel.go
```

### Utilisation
Pour lancer le serveur HTTP :
```bash
$ ./DoublingParallel
```
Le serveur sera par défaut lancé sur le port **8080**. Vous pouvez modifier ce paramètre dans la configuration.

## Détails de l'Implémentation
### Configuration
Le fichier de configuration vous permet de définir les paramètres suivants :
- **MaxValue** : La valeur maximale de `n` pour le calcul de Fibonacci.
- **MaxCacheSize** : La taille maximale du cache LRU.
- **WorkerCount** : Le nombre de **workers** utilisés pour calculer les valeurs de Fibonacci.
- **Timeout** : Le **délai d'expiration** du calcul.
- **HTTPPort** : Le port HTTP sur lequel le serveur écoute.

Ces paramètres peuvent être modifiés selon vos besoins afin d'ajuster la charge et la performance du service.

### Cache LRU
Le **cache LRU** est implémenté à l'aide de la bibliothèque `hashicorp/golang-lru`. Il permet de conserver en mémoire les résultats des calculs récents afin d'éviter des recalculs inutiles, améliorant ainsi la **latence** des réponses.

- Les éléments du cache sont régulièrement mis à jour pour maintenir les valeurs les plus utilisées en mémoire.
- Utilisation de verrous (`RWMutex`) pour assurer une **concurrence sûre** lors de la lecture et l'écriture dans le cache.

### Algorithme de Doublage Rapide
L'algorithme utilisé pour calculer les valeurs de **Fibonacci** est basé sur la méthode de **doublage rapide** (Fast Doubling). Cette technique est particulièrement efficace pour calculer des valeurs de Fibonacci car elle a une **complexité logarithmique**, permettant ainsi des calculs plus rapides.

### Gestion des Erreurs
Le service gère plusieurs types d'erreurs :
- **Entrées négatives** : Une erreur (`ErrNegativeInput`) est renvoyée si `n` est inférieur à 0.
- **Entrées trop grandes** : Si `n` dépasse `MaxValue`, une erreur (`ErrInputTooLarge`) est renvoyée.
- **Annulation de Contexte** : Le service vérifie si un **contexte** a été annulé (éventuellement suite à un **timeout**) pour éviter des calculs inutiles.

## API HTTP
### Endpoint `/compute`
- **URL** : `/compute`
- **Méthode** : `POST`
- **Corps de la Requête** :
  ```json
  {
    "n": <valeur_entier>
  }
  ```
- **Réponse** :
  ```json
  {
    "result": "<valeur_de_fibonacci>"
  }
  ```
- **Code de Statut** : `200 OK` en cas de succès, `400 Bad Request` si la requête est invalide ou si `n` est hors des limites acceptées.

Exemple d'utilisation avec `curl` :
```bash
$ curl -X POST -H "Content-Type: application/json" -d '{"n": 10}' http://localhost:8080/compute
```

## Optimisations
Le service a été optimisé de plusieurs façons pour garantir des performances élevées :
- **Parallélisme** : Utilisation de `runtime.GOMAXPROCS` pour maximiser l'utilisation des processeurs disponibles.
- **Pools de `big.Int`** : Utilisation de `sync.Pool` pour réutiliser les objets `big.Int` afin de minimiser l'overhead des allocations dynamiques.
- **Cache LRU** : Minimise le temps de calcul pour des valeurs répétées.
- **Verrous** : Protection de l'accès au cache via des verrous (`RWMutex`), assurant une lecture concurrente rapide tout en évitant les problèmes de course.

## Métriques et Monitoring
Les **métriques** sont collectées pour suivre les performances et sont accessibles via la structure `Metrics` :
- **Cache Hits/Misses** : Permet de savoir combien de fois une valeur a été trouvée dans le cache par rapport au nombre de calculs requis.
- **Temps de Calcul Total** : Temps total passé à calculer des valeurs de Fibonacci.
- **Utilisation de la Mémoire** : Mise à jour régulière pour surveiller l'utilisation des ressources.

## Contribuer
Les contributions sont les bienvenues ! Veuillez suivre les étapes suivantes pour contribuer :
1. Forkez le dépôt.
2. Créez une branche pour vos modifications : `git checkout -b ma-nouvelle-fonctionnalite`.
3. Soumettez un **pull request** avec une description de vos modifications.

## Licence
Ce projet est sous licence **MIT**. Pour plus de détails, veuillez consulter le fichier `LICENSE`.
