# Matrix-ClaudeAI

![Diagramme de Séquence](SequenceDiagram-1.jpeg)

## Description

Ce projet est une implémentation efficace et concurrente pour le calcul de la suite de Fibonacci, écrite en Go. Il utilise l'algorithme d'exponentiation rapide des matrices pour calculer les valeurs de la suite de manière performante. Le programme exploite la puissance de calcul parallèle en utilisant un pool de workers et des structures de synchronisation adaptées. Le calcul est géré de manière itérative, en utilisant la structure `big.Int` de la bibliothèque standard de Go, pour permettre de manipuler des nombres de très grande taille.

## Fonctionnalités

- **Exponentiation rapide des matrices** : Permet de calculer les valeurs de la suite de Fibonacci en temps logarithmique.
- **Calcul parallèle** : Utilise un pool de workers pour paralléliser le calcul des valeurs, améliorant la performance sur les systèmes multi-cœurs.
- **Cache** : Stocke les résultats précédemment calculés pour éviter les recalculs inutiles.
- **Gestion du contexte** : Intègre l'annulation et la gestion des délais pour arrêter les calculs si nécessaire.

## Structure du Code

1. **Matrix2x2** : Représentation d'une matrice 2x2 utilisée pour les calculs de Fibonacci par exponentiation rapide.
2. **FibCalculator** : Objet responsable du calcul des valeurs de Fibonacci en utilisant les matrices.
3. **WorkerPool** : Gère un pool de workers pour distribuer le calcul des segments de la suite de Fibonacci.
4. **Fonction Main** : Initialise le pool de workers, répartit les segments de la suite à calculer et collecte les résultats.

## Prérequis

Pour exécuter ce programme, vous aurez besoin de :

- **Go 1.16 ou supérieur** : Le programme est écrit en Go et utilise des fonctionnalités modernes de la bibliothèque standard.
- **Multithreading support** : Un système capable de gérer plusieurs threads de manière efficace afin de tirer parti de la parallélisation.

## Installation

1. **Cloner le dépôt** :
   ```bash
   git clone <URL_du_dépôt>
   ```

2. **Naviguer dans le répertoire** :
   ```bash
   cd <nom_du_dépôt>
   ```

3. **Compiler le programme** :
   ```bash
   go build -o fibonacci_calculator
   ```

4. **Exécuter le programme** :
   ```bash
   ./fibonacci_calculator
   ```

## Utilisation

Le programme calcule la somme des `n` premiers termes de la suite de Fibonacci en utilisant un pool de workers afin de répartir les calculs sur plusieurs cœurs. L'utilisateur peut ajuster la valeur `n` et la durée limite du calcul en modifiant les paramètres dans la fonction `main`.

Le programme est conçu pour gérer de grands calculs en utilisant la structure `big.Int`, garantissant ainsi que même les valeurs de Fibonacci très élevées peuvent être traitées sans limitation d'entier.

## Détails Techniques

- **Exponentiation Rapide des Matrices** : L'utilisation de la multiplication de matrices pour calculer la suite de Fibonacci permet d'obtenir une complexité logarithmique, comparée à la méthode naïve itérative ou récursive.
- **Pool de Workers** : Un `WorkerPool` est utilisé pour paralléliser le calcul. Chaque worker reçoit une portion du travail à accomplir, ce qui réduit significativement le temps de calcul sur les machines multi-cœurs.
- **Gestion des Ressources** : Le programme utilise des sémaphores pour contrôler l'accès aux workers, et des primitives de synchronisation comme `sync.Mutex` et `sync.WaitGroup` pour assurer la sécurité des threads.

## Exemples

Pour ajuster le nombre de termes de Fibonacci à calculer, vous pouvez modifier la valeur de `n` dans la fonction `main()` :

```go
n := 100000 // Limite de la suite de Fibonacci
```

Le programme utilise également un contexte (`context.WithTimeout`) pour s'assurer que l'exécution ne dépasse pas une durée limite spécifiée.

## Contributions

Les contributions sont les bienvenues. Vous pouvez créer une pull request ou ouvrir une issue pour discuter des améliorations potentielles.

## Licence

Ce projet est sous licence MIT. Vous êtes libre de l'utiliser, de le modifier et de le distribuer selon les termes de la licence.

## Remerciements

Ce projet a été réalisé pour démontrer l'utilisation combinée de la parallélisation et des algorithmes efficaces pour le calcul de nombres de Fibonacci, et pour approfondir la compréhension de la gestion des threads en Go.

