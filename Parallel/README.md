# Fibonacci Concurrent Calculator

![Diagramme de l'algorithme de Fibonacci]([https://github.com/agbruneau/Fibonacci/blob/main/Function/Fibonacci%20Golang%20Sequence%20Diagram.jpeg](https://github.com/agbruneau/Fibonacci/blob/main/Parallel/Fibonacci%20Golang%20Sequence%20Diagram.jpeg))

## Description

Ce programme en Go calcule la somme d'une partie de la suite de Fibonacci jusqu'à un indice donné (`n`) de manière concurrente à l'aide de goroutines, de mémoïsation et de synchronisation avec `sync.WaitGroup`. Il est conçu pour diviser le calcul de Fibonacci en plusieurs segments, chaque segment étant traité par une goroutine séparée, ce qui permet de répartir la charge de travail et d'accélérer le calcul sur des machines multi-cœurs.

Une fois que toutes les parties de la suite de Fibonacci sont calculées, elles sont combinées pour produire un résultat final qui est ensuite enregistré dans un fichier nommé `fibonacci_result.txt`.

## Fonctionnalités

- **Mémoïsation** : Utilisation de `sync.Map` pour stocker et réutiliser les valeurs précédemment calculées de la suite de Fibonacci.
- **Calcul concurrent** : Le calcul est réparti entre plusieurs goroutines (par défaut, 4) pour optimiser les performances.
- **Synchronisation** : Utilisation de `sync.WaitGroup` pour synchroniser l'exécution des goroutines.
- **Résultat enregistré** : Le résultat final est enregistré dans un fichier texte.

## Structure du Code

1. **Mémoïsation avec `sync.Map`** : Le programme utilise une map sécurisée pour stocker les résultats déjà calculés et éviter de recalculer les mêmes valeurs de Fibonacci.
   
2. **Fonction `calcFibonacci`** : Cette fonction calcule une portion de la suite de Fibonacci entre deux indices donnés (de `start` à `end`). Elle est exécutée par des goroutines pour chaque segment.

3. **Canal `partialResult`** : Les résultats partiels de chaque calcul sont envoyés via un canal, puis combinés dans le `main`.

4. **Fonction `main`** :
    - Définit les paramètres (longueur de la suite `n` et nombre de workers `numWorkers`).
    - Lance plusieurs goroutines pour paralléliser le calcul de la suite.
    - Attend la fin des goroutines et combine les résultats partiels.
    - Écrit le résultat final dans un fichier et affiche le temps d'exécution.

## Prérequis

Pour exécuter ce programme, vous aurez besoin de :

- Go 1.16 ou supérieur (recommandé)
- Un environnement compatible avec Go installé

## Installation

1. Installez Go depuis [le site officiel](https://golang.org/dl/).
2. Clonez ou téléchargez ce dépôt sur votre machine.
3. Naviguez vers le répertoire du projet.

```bash
git clone <repo-url>
cd <repo-directory>
```

## Usage

Pour exécuter le programme, vous pouvez utiliser la commande suivante dans votre terminal :

```bash
go run main.go
```

Par défaut, le programme calcule la somme des nombres de Fibonacci jusqu'à `n = 1 000 000` en utilisant 4 goroutines. Le résultat sera sauvegardé dans un fichier `fibonacci_result.txt` à la racine du répertoire où le programme est exécuté.

### Paramètres Modifiables

- **`n`** : Le nombre total de termes Fibonacci à calculer. Il est défini dans la fonction `main()` et peut être modifié.
- **`numWorkers`** : Le nombre de goroutines à utiliser pour le calcul. Plus ce nombre est élevé, plus le travail est distribué de manière concurrente.

### Exemple de modification

Si vous voulez changer le nombre de termes de Fibonacci calculés et le nombre de goroutines, vous pouvez modifier les variables suivantes dans `main()` :

```go
n := 500000     // Exemple : Calculer jusqu'au 500 000e nombre de Fibonacci
numWorkers := 8 // Utiliser 8 goroutines au lieu de 4
```

## Résultats

Après l'exécution du programme, vous verrez un fichier texte nommé `fibonacci_result.txt` contenant la somme des termes Fibonacci calculés. Le programme affichera également le temps d'exécution total dans la console.

Exemple de sortie :

```
Temps d'exécution: 2.34567s
Résultat et temps d'exécution écrits dans 'fibonacci_result.txt'.
```

## Détails Techniques

1. **Mémoïsation** : Le programme utilise une table de hachage sécurisée (avec `sync.Map`) pour stocker les résultats des calculs de Fibonacci. Cela évite de recalculer les mêmes valeurs et améliore les performances.

2. **Concurrent Computing** : Le calcul de Fibonacci étant coûteux pour des grands nombres, le programme divise le travail en segments. Chaque goroutine calcule un segment de la suite. Cela permet de paralléliser le calcul et d'accélérer l'exécution sur des machines multi-cœurs.

3. **Synchronisation avec `sync.WaitGroup`** : `sync.WaitGroup` est utilisé pour s'assurer que toutes les goroutines ont terminé leur travail avant de combiner les résultats.

4. **Canaux Go** : Le canal `partialResult` est utilisé pour transmettre les résultats partiels des goroutines au processus principal, qui les combine ensuite pour obtenir le résultat final.

## Améliorations Futures

- **Gestion des erreurs améliorée** : Bien que des vérifications d'erreurs existent, des améliorations peuvent être apportées pour gérer des cas d'erreur plus complexes.
- **Optimisation de la performance** : Des algorithmes de Fibonacci plus avancés, comme l'exponentiation de matrices, pourraient être utilisés pour des calculs encore plus rapides.

## License

Ce projet est sous licence MIT. Vous êtes libre de l'utiliser, le modifier et le distribuer comme vous le souhaitez.

---

Cela résume bien le fonctionnement et les aspects clés de votre programme en Go. Vous pouvez adapter ce fichier `README.md` en fonction de vos besoins spécifiques.
