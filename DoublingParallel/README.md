# Calcul de Fibonacci par la Méthode de Calcul Parallèle avec Mémoïsation et Benchmark

![Diagramme de contexte du code golang de calcul de la liste de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/DoublingParallel/ContextDiagram.jpeg)
![Diagramme de séquence du code golang du calcul de la liste de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/DoublingParallel/SequenceDiagram.jpeg)


# README : Programme de Calcul Parallèle de la Somme des Nombres de Fibonacci

## Introduction

Ce projet est un programme écrit en Go permettant de calculer la somme des nombres de Fibonacci jusqu'à un certain terme. L'objectif principal est de démontrer l'utilisation de la programmation parallèle et de la mémoïsation en Go afin de réaliser des calculs intensifs de manière optimale. Le programme exploite la méthode de doublage (doubling) pour le calcul des termes de Fibonacci et utilise des goroutines pour la parallélisation du calcul, ce qui réduit de manière significative le temps d'exécution.

## Fonctionnalités

- **Calcul Parallèle** : Division du calcul de la somme des nombres de Fibonacci en plusieurs segments, chacun étant traité par une goroutine différente afin d'améliorer les performances sur les systèmes multi-cœurs.
- **Mémoïsation Thread-Safe** : Utilisation de la mémoïsation à l'aide de la structure `sync.Map` pour stocker les valeurs intermédiaires déjà calculées et éviter les recalculs inutiles.
- **Gestion de la Synchronisation** : Gestion des goroutines via `sync.WaitGroup` afin d'assurer la synchronisation et la bonne gestion des ressources.
- **Résultats** : Le résultat final, soit la somme des termes de la suite de Fibonacci, est écrit dans un fichier texte nommé `fibonacci_result.txt`. Le temps d'exécution est également affiché dans le terminal.

## Dépendances

Le programme utilise plusieurs bibliothèques standards de Go pour assurer le bon fonctionnement :

- **`math/big`** : Pour manipuler des entiers de grande taille (arbitraire) car les termes de la suite de Fibonacci peuvent atteindre des valeurs très élevées.
- **`sync`** : Pour synchroniser les goroutines via `sync.Map` et `sync.WaitGroup`.
- **`time`** : Pour mesurer et afficher le temps d'exécution.
- **`os`** : Pour la création et l'écriture dans un fichier de sortie.

## Prérequis

Pour exécuter ce programme, vous devez disposer de Go (à partir de la version 1.16). Vous pouvez vérifier votre version de Go en utilisant la commande suivante :

```sh
$ go version
```

Si Go n'est pas installé sur votre système, vous pouvez le télécharger depuis [le site officiel de Go](https://golang.org/dl/).

## Installation et Exécution

1. **Cloner le dépôt**

   Pour télécharger le code source, vous pouvez cloner ce dépôt en utilisant Git :

   ```sh
   $ git clone <URL_DU_DEPOT>
   $ cd <NOM_DU_REPERTOIRE>
   ```

2. **Exécuter le Programme**

   Pour compiler et exécuter le programme, utilisez les commandes suivantes :

   ```sh
   $ go build -o fibonacci_sum
   $ ./fibonacci_sum
   ```

   Par défaut, le programme calculera la somme des termes de la suite de Fibonacci jusqu'à 100 millions.

## Fonctionnement du Programme

Le programme calcule la somme des nombres de Fibonacci jusqu'à un terme donné en divisant le calcul en plusieurs segments, chacun traité par une goroutine différente. Chaque goroutine calcule la somme partielle de son segment et envoie le résultat via un canal. Le programme principal récupère ces résultats partiels et les agrège pour obtenir la somme finale.

Les calculs sont basés sur la méthode de **doublage** (“doubling”), qui est une méthode efficace pour le calcul de la suite de Fibonacci, permettant de diviser la complexité et ainsi d'améliorer la rapidité des calculs.

## Organisation du Code

- **`fibDoubling(n int)`** : Fonction principale qui calcule le nième terme de Fibonacci en utilisant la méthode de doublage.
- **`fibDoublingHelperIterative(n int)`** : Fonction auxiliaire itérative utilisant des opérations de bits pour optimiser le calcul.
- **`calcFibonacci(start, end int, partialResult chan<- *big.Int, wg *sync.WaitGroup)`** : Fonction qui calcule la somme des termes de Fibonacci sur un segment défini et transmet le résultat partiel via un canal.
- **`main()`** : Fonction principale qui divise le travail entre les goroutines, synchronise les calculs et gère l'écriture du résultat dans un fichier.

## Limitations et Améliorations Possibles

- **Performance** : Bien que la parallélisation améliore significativement les performances, le calcul des très grands termes de Fibonacci peut être coûteux en ressources mémoire et temps d'exécution. Une future optimisation pourrait consister à utiliser une approche adaptative pour la répartition des segments.
- **Tolérance aux Pannes** : Il pourrait être utile d'ajouter des mécanismes de redondance ou de récupération en cas d'échec d'une goroutine, afin d'améliorer la robustesse globale.
- **Cache Amélioré** : L'intégration d'une cache plus sophistiquée, comme une cache LRU (Least Recently Used), pourrait aider à éviter les recalculs répétitifs tout en évitant une surcharge mémoire.

## Contribution

Les contributions sont les bienvenues ! Pour contribuer, veuillez cloner le dépôt, créer une branche, et soumettre une pull request avec vos modifications. Assurez-vous d'inclure des tests et des descriptions claires de vos changements.

## Licence

Ce projet est sous licence MIT. Vous êtes libre de l'utiliser, de le modifier et de le distribuer sous les conditions de cette licence.

## Contact

Pour toute question ou suggestion, veuillez contacter l'auteur à l'adresse suivante : [votre@email.com].

