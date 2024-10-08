# Calcul de Fibonacci par la Méthode de Calcul Parallèle avec Mémoïsation et Benchmark

![Diagramme de l'algorithme de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/Parallel/Fibonacci%20Golang%20Sequence%20Diagram.jpeg)

## Description

Ce projet, développé en Go, implémente le calcul des sommes partielles de la suite de Fibonacci jusqu'à un indice donné (`n`) en exploitant des techniques avancées de concurrence, notamment l'utilisation de goroutines, la mémoïsation, et la synchronisation via `sync.WaitGroup`. L'objectif est de décomposer le calcul de la suite de Fibonacci en plusieurs segments parallèles, chaque segment étant traité indépendamment par une goroutine distincte, afin de mieux exploiter les capacités des systèmes multi-cœurs et d'optimiser la performance.

Après le calcul des différentes parties de la suite, les résultats partiels sont combinés pour former le résultat final, qui est ensuite écrit dans un fichier intitulé `fibonacci_result.txt`.

## Fonctionnalités

- **Mémoïsation** : Utilisation de `sync.Map` pour stocker les valeurs calculées antérieurement, permettant leur réutilisation et réduisant ainsi la redondance de calcul.
- **Calcul parallèle** : Répartition du calcul sur plusieurs goroutines (par défaut, 4) afin d'accélérer le processus sur des systèmes multi-cœurs.
- **Synchronisation** : Utilisation de `sync.WaitGroup` pour garantir une coordination correcte entre les différentes goroutines.
- **Enregistrement des résultats** : Le résultat final est sauvegardé dans un fichier texte afin de permettre une consultation ultérieure.

## Structure du Code

1. **Mémoïsation avec `sync.Map`** : Le programme emploie une structure de données sécurisée pour stocker les valeurs de Fibonacci déjà calculées, permettant d'éviter des recalculs inutiles et d'améliorer l'efficacité.

2. **Fonction `calcFibonacci`** : Cette fonction est responsable du calcul d'une portion de la suite de Fibonacci entre deux indices (`start` et `end`). Elle est conçue pour être exécutée par des goroutines, facilitant la distribution du calcul.

3. **Canal `partialResult`** : Les résultats partiels générés par chaque goroutine sont transmis via un canal, puis agrégés dans la fonction principale (`main`).

4. **Fonction `main`** :
    - Définit les paramètres tels que la longueur de la suite (`n`) et le nombre de workers (`numWorkers`).
    - Lance plusieurs goroutines pour effectuer le calcul de manière concurrente.
    - Attend la fin de toutes les goroutines et agrège les résultats partiels.
    - Écrit le résultat final dans un fichier et affiche le temps total d'exécution.

## Prérequis

Pour exécuter ce programme, les éléments suivants sont requis :

- **Go version 1.16 ou supérieure**
- **Un environnement compatible pour l'installation et l'exécution de Go**

## Installation

1. Installez Go depuis [le site officiel](https://golang.org/dl/).
2. Clonez ou téléchargez ce dépôt sur votre machine.
3. Naviguez jusqu'au répertoire du projet :

```bash
git clone <repo-url>
cd <repo-directory>
```

## Utilisation

Pour exécuter le programme, lancez la commande suivante dans votre terminal :

```bash
go run main.go
```

Par défaut, le programme calcule la somme des nombres de Fibonacci jusqu'à `n = 1 000 000`, en utilisant 4 goroutines. Le résultat est écrit dans un fichier intitulé `fibonacci_result.txt`, situé à la racine du répertoire d'exécution.

### Paramètres Modifiables

- **`n`** : Le nombre total de termes de la suite de Fibonacci à calculer. Ce paramètre est défini dans la fonction `main()` et peut être ajusté selon vos besoins.
- **`numWorkers`** : Le nombre de goroutines à utiliser pour le calcul parallèle. Un nombre plus élevé de goroutines peut améliorer l'efficacité du calcul en répartissant davantage la charge de travail.

### Exemple de Modification

Pour modifier le nombre de termes de Fibonacci calculés ou le nombre de goroutines, ajustez les variables dans la fonction `main()` comme suit :

```go
n := 500000     // Par exemple, calculer jusqu'au 500 000e terme de Fibonacci
numWorkers := 8 // Utiliser 8 goroutines au lieu de 4
```

## Résultats

Après l'exécution du programme, un fichier texte nommé `fibonacci_result.txt` contiendra la somme des termes de la suite de Fibonacci calculés. Le programme affichera également le temps d'exécution total dans la console.

### Exemple de Sortie

```
Temps d'exécution: 2.34567s
Résultat et temps d'exécution écrits dans 'fibonacci_result.txt'.
```

## Détails Techniques

1. **Mémoïsation** : Le programme utilise `sync.Map`, une table de hachage sécurisée et concurrente, pour stocker les résultats de la suite de Fibonacci déjà calculés. Cela permet de minimiser la redondance des calculs et d'améliorer l'efficacité globale.

2. **Calcul Concurrent** : Le calcul de Fibonacci étant particulièrement coûteux pour des indices élevés, le programme segmente le travail en plusieurs parties. Chaque segment est traité indépendamment par une goroutine, permettant une parallélisation du calcul et une réduction significative du temps d'exécution sur des systèmes multi-cœurs.

3. **Synchronisation via `sync.WaitGroup`** : La structure `sync.WaitGroup` est utilisée pour garantir que toutes les goroutines ont terminé leur exécution avant que les résultats ne soient combinés. Cela permet une coordination précise entre les différentes parties du calcul.

4. **Canaux Go** : Le canal `partialResult` est employé pour transmettre les résultats partiels de chaque goroutine au processus principal, qui les combine ensuite pour obtenir le résultat final.

## Améliorations Futures

- **Gestion Améliorée des Erreurs** : Bien que le programme comprenne des vérifications d'erreurs, des améliorations peuvent être apportées pour couvrir des cas d'erreur plus variés et complexes.
- **Optimisation de la Performance** : L'intégration d'algorithmes plus avancés pour le calcul de Fibonacci, comme l'exponentiation par matrices, pourrait permettre d'accélérer davantage le calcul, surtout pour des valeurs de `n` extrêmement élevées.

## Licence

Ce projet est sous licence MIT. Vous êtes libre de l'utiliser, le modifier et le distribuer selon les termes de cette licence.