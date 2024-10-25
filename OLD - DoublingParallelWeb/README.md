# Calcul de Fibonacci Web par la Méthode du Doublement avec Mémoïsation et Benchmark

![Diagramme de Séquence](SequenceDiagram.jpeg)

Ce projet est un programme en Go (Golang) qui propose un service Web pour calculer la somme des termes de la suite de Fibonacci jusqu'à un terme donné. Ce service est conçu pour les utilisateurs ayant des connaissances en programmation et en calculs mathématiques avancés. Voici une description complète du fonctionnement du programme, des technologies utilisées, ainsi que des instructions pour son utilisation.

## Description

Ce programme expose une API REST qui permet de calculer la somme des termes de la suite de Fibonacci jusqu'au terme `n`. Pour garantir une performance optimale, le calcul repose sur la **méthode du doublage** (doubling method) combinée à une **approche concurrente**, exploitant les cœurs du processeur afin d'accélérer le traitement.

Le service est conçu pour répondre à une requête HTTP POST contenant un JSON avec la valeur de `n`, et renvoie la somme des termes de Fibonacci, le nombre de calculs effectués, le temps moyen par calcul, ainsi que le temps d'exécution total.

## Fonctionnalités

- **API REST** : Expose une interface pour calculer la somme de la suite de Fibonacci jusqu'à `n`.
- **Calcul Parallèle** : Division du calcul en segments, exécutés en parallèle à l'aide de **goroutines**.
- **Méthode de Doublage** : Utilisation de la méthode du doublage pour améliorer l'efficacité du calcul des termes de la suite.
- **Support des Grands Nombres** : Utilisation du package `math/big` pour manipuler les entiers de grande taille et garantir la précision.

## Prérequis

- **Go (Golang)** : Version 1.16 ou ultérieure.
- **Machine avec plusieurs cœurs de CPU** : Recommandé pour exploiter pleinement les performances parallèles du programme.

## Installation et Lancement

1. Clonez le dépôt :
   ```sh
   git clone https://github.com/votre-utilisateur/service-fibonacci.git
   cd service-fibonacci
   ```

2. Compilez et lancez le serveur :
   ```sh
   go run main.go
   ```

3. Le serveur sera démarré sur le port **8080** par défaut. Vous pouvez maintenant envoyer des requêtes HTTP POST à l'endpoint `/fibonacci`.

## Utilisation de l'API

- **Endpoint** : `/fibonacci`
- **Méthode** : POST
- **Corps de la requête** : JSON avec la structure suivante :
  ```json
  {
    "n": 10
  }
  ```

### Exemple de Requête avec cURL

```sh
curl -X POST -H "Content-Type: application/json" -d '{"n": 10}' http://localhost:8080/fibonacci
```

### Exemple de Réponse

```json
{
  "sum": "143",
  "num_calculations": 10,
  "avg_time_per_calculation_in_second": 0.002,
  "execution_time_in_second": 0.02
}
```

## Détails Techniques

### 1. **Fonctionnalités de Calcul**
- La fonction `fibDoubling(n int)` est utilisée pour calculer le nième terme de la suite en utilisant une approche à la fois efficace et précise, grâce au package `math/big` pour les grands nombres.
- La fonction auxiliaire `fibDoublingHelperIterative` utilise une approche itérative en combinant les valeurs via des opérations sur les bits de `n`, ce qui optimise le nombre de multiplications nécessaires.

### 2. **Calcul Concurrent avec Goroutines**
- Le calcul de la somme des nombres de Fibonacci est divisé en **segments**. Chaque segment est attribué à une goroutine distincte pour être traité en parallèle, permettant ainsi d'accélérer le traitement.
- Le programme détermine automatiquement le nombre de cœurs de CPU disponibles (à l'aide de `runtime.NumCPU()`) et crée autant de travailleurs que de cœurs.

### 3. **Méthode du Doublage**
- La **méthode du doublage** réduit le nombre d'opérations arithmétiques nécessaires en exploitant la structure binaire de `n`. Cela permet de diviser pour mieux régner, en évitant le recalcul de termes déjà connus.

- La méthode de doublage est une technique avancée utilisée pour calculer efficacement les termes de la série de Fibonacci. Vulgarisons cette méthode afin de la rendre accessible à un public plus large, tout en préservant sa rigueur mathématique.

- Les nombres de Fibonacci sont une séquence de nombres où chaque terme est la somme des deux termes précédents, commençant par 0 et 1. Bien que cela semble simple, la croissance exponentielle de ces nombres rend leur calcul très exigeant en termes de ressources, surtout pour des valeurs très élevées de n (le nième terme que l'on cherche à calculer).

- La méthode de doublage repose sur la représentation binaire de l'indice n et permet de calculer les termes de manière plus efficace que les approches itératives classiques. En utilisant cette méthode, on peut diviser le problème en plusieurs petites étapes, exploitant la structure binaire de n pour réduire le nombre total d'opérations nécessaires. Plutôt que de calculer les termes de manière linéaire, l'algorithme utilise des techniques de **décalage de bits** et des **multiplications parallèles** afin d'obtenir un résultat en beaucoup moins d'étapes.

- Pour illustrer cela, imaginons que l'on veut atteindre un certain terme F(n). La méthode de doublage décompose cette opération en utilisant une série de doubles et de multiplications conditionnelles basées sur la valeur binaire de n. Concrètement, l'algorithme se sert des deux éléments F(k) et F(k+1) pour calculer F(2k) et F(2k+1) à chaque étape, ce qui évite une accumulation linéaire lente.

- L'utilisation de la méthode de doublage est particulièrement avantageuse lorsqu'elle est combinée avec le **parallélisme**, c'est-à-dire en effectuant simultanément plusieurs calculs grâce à la puissance de traitement de plusieurs cœurs de CPU. Cela permet de diviser la charge de travail et d'accélérer considérablement le calcul de termes très élevés de la série.

- En résumé, la méthode de doublage est une approche efficace qui permet de calculer rapidement de grands nombres de Fibonacci. Grâce à sa capacité à exploiter la représentation binaire des indices et à diviser la charge de travail, elle surpasse largement les méthodes plus traditionnelles lorsqu'il s'agit de calculs à grande échelle.

## Améliorations Potentielles
- **Optimisation de la mémoire** : Ajouter un système de mise en cache (via `sync.Map`) pourrait réduire la charge de calcul, en évitant le recalcul de valeurs de Fibonacci déjà obtenues.
- **Limitation dynamique** : Adapter dynamiquement la limite de `n` en fonction des ressources disponibles (mémoire et puissance du CPU).

## Mise en Garde
- Ce programme peut être très **gourmand en ressources** (CPU et mémoire) pour des valeurs élevées de `n`. Il est fortement conseillé de l'utiliser sur une machine disposant de plusieurs cœurs et d'une bonne quantité de mémoire vive.
- **Temps de Calcul** : Le temps de calcul augmente exponentiellement avec la taille de `n`. Soyez vigilant avant de lancer des calculs très longs.

## Contributions
Les contributions sont les bienvenues. Pour contribuer :
- Forkez le projet.
- Créez une branche de fonctionnalité (`git checkout -b feature/NouvelleFonctionnalite`).
- Faites vos modifications et soumettez une **Pull Request**.

## Licence
Ce projet est distribué sous la licence MIT. Voir le fichier `LICENSE` pour plus de détails.