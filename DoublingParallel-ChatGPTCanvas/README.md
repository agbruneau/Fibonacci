# Programme de calcul de la somme des nombres de Fibonacci en parallèle

## Description
Ce programme, écrit en Go, permet de calculer la somme des nombres de Fibonacci en parallèle en utilisant plusieurs workers, chacun exploitant un calculateur de Fibonacci dédié. Le calcul est effectué de manière optimisée grâce à la décomposition binaire des calculs et au parallélisme fourni par les goroutines de Go. Ce programme est particulièrement adapté à des calculs intensifs en raison de son approche orientée sur l'optimisation des ressources disponibles, notamment par l'utilisation de plusieurs cœurs de processeur.

## Objectif
L'objectif principal de ce programme est de démontrer l'efficacité des techniques de parallélisme pour résoudre des problèmes computationnels exigeants. En particulier, il s'agit de calculer la série de Fibonacci jusqu'à une valeur `n` donnée, tout en utilisant efficacement les ressources CPU pour minimiser le temps d'exécution total. Ce programme utilise des concepts avancés tels que la synchronisation des goroutines, le verrouillage des ressources partagées, et l'utilisation de structures de données thread-safe (`sync.Mutex`).

## Composants principaux
Le programme est structuré autour de plusieurs composants majeurs :

### 1. `FibCalculator`
`FibCalculator` est une structure qui encapsule les variables nécessaires au calcul des nombres de Fibonacci en utilisant de grandes valeurs entières (via le package `math/big`). Cette structure permet de rendre le calcul thread-safe grâce à l'utilisation d'un `sync.Mutex`, garantissant ainsi que plusieurs goroutines ne peuvent pas modifier simultanément les mêmes données.

### 2. `WorkerPool`
`WorkerPool` est une structure qui gère un ensemble de calculateurs de Fibonacci. Cette structure permet de distribuer le travail de manière efficace entre les différents calculateurs, en garantissant une allocation optimale des ressources. Elle utilise également un verrou (`sync.Mutex`) pour éviter les conflits d'accès concurrent.

### 3. `calcFibonacci`
La fonction `calcFibonacci` est responsable du calcul d'une portion de la liste des nombres de Fibonacci entre deux bornes spécifiées (`start` et `end`). Chaque portion est calculée par un worker et le résultat partiel est envoyé à un canal (`channel`) pour être additionné ultérieurement.

### 4. Fonction `main`
La fonction `main` gère l'initialisation des différents composants, la division du travail entre les workers, et la collecte des résultats partiels. Elle initialise le nombre de workers à utiliser, crée un pool de calculateurs, et lance des goroutines pour le calcul parallèle. Elle mesure également le temps d'exécution total et le temps moyen par calcul pour fournir des statistiques de performance.

## Fonctionnement
1. **Initialisation des paramètres** : Le programme commence par déterminer le nombre de CPU disponibles pour établir le nombre de workers à créer.
2. **Division du travail** : Le travail est divisé en segments, chacun traité par une goroutine indépendante.
3. **Synchronisation** : Des `WaitGroup` et `Mutex` sont utilisés pour gérer la synchronisation et l'accès aux ressources partagées afin de s'assurer qu'il n'y a pas de conflits lors des calculs.
4. **Collecte des résultats** : Les résultats partiels sont collectés et additionnés pour obtenir la somme totale des nombres de Fibonacci jusqu'à `n`.

## Exécution
Pour exécuter le programme, vous devez avoir Go installé sur votre machine. Ensuite, compilez et exécutez le programme comme suit :

```sh
$ go run main.go
```

Le programme affichera des statistiques sur le calcul, y compris le nombre de workers utilisés, le temps moyen par calcul, et le temps d'exécution total.

## Prérequis
- **Go** : Version 1.16 ou supérieure.
- **Processeur multi-cœurs** : Le programme est conçu pour exploiter les cœurs multiples pour des performances optimales.

## Notes sur la performance
Ce programme est conçu pour être performant sur des machines disposant de plusieurs cœurs CPU. L'utilisation du parallélisme via les goroutines permet de réduire significativement le temps nécessaire pour calculer de grands nombres de Fibonacci. Toutefois, la taille de `n` est limitée à 250 millions pour éviter des calculs trop coûteux en temps et en mémoire.

## Limites
- Le programme impose une limite à la valeur de `n` (250 millions) en raison de la nature exponentielle de la croissance des valeurs de Fibonacci, ce qui pourrait entraîner des calculs extrêmement coûteux en temps et en mémoire.
- Les nombres de Fibonacci calculés sont de très grande taille, ce qui nécessite l'utilisation de `math/big` pour les représenter correctement.

## Licence
Ce projet est sous licence MIT. Vous êtes libre de l'utiliser, de le modifier et de le distribuer conformément aux termes de cette licence.