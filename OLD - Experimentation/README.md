# Calculateur de Nombres de Fibonacci en Parallèle

![Diagramme de Séquence](SequenceDiagram.jpeg)

## Description du Projet

Ce projet est une implémentation en Go (Golang) d'un calculateur de nombres de Fibonacci, optimisé pour des calculs à grande échelle et utilisant le parallélisme afin d'améliorer les performances. Le programme utilise des techniques avancées telles que la décomposition binaire pour réduire la complexité temporelle du calcul, et gère un pool de travailleurs ("worker pool") pour tirer parti des multiples cœurs de CPU disponibles.

Le projet est conçu pour être utilisé dans des environnements où des calculs intensifs sont requis, tout en évitant les problèmes liés à la concurrence à travers l'utilisation d'outils de synchronisation fournis par la librairie standard de Go.

## Fonctionnalités

1. **Calcul Optimisé des Nombres de Fibonacci**  
   - Utilisation de la méthode de "doublage" pour calculer efficacement les nombres de Fibonacci, en réduisant la complexité du calcul par rapport aux méthodes itératives ou récursives traditionnelles.

2. **Parallélisation des Calculs**  
   - Utilisation de goroutines pour paralléliser le calcul sur plusieurs cœurs de CPU, maximisant ainsi l'efficacité du programme, particulièrement sur des machines multicœurs.

3. **Gestion de la Concurrence avec un Pool de Travailleurs**  
   - Gestion des ressources à travers un "worker pool" pré-configuré en fonction du nombre de cœurs CPU, afin d'équilibrer la charge et d'assurer une utilisation optimale des ressources.

4. **Prévention des Débordements et Gestion de la Précision**  
   - Utilisation de `big.Int` pour gérer les grands nombres de Fibonacci sans risque de débordement.

5. **Sauvegarde et Lecture des Résultats**  
   - Les résultats des calculs, y compris la somme des nombres de Fibonacci calculés, sont écrits dans un fichier texte. Le contenu du fichier est ensuite affiché pour fournir un récapitulatif des calculs effectués.

## Structure du Code

### 1. `FibCalculator`
La structure `FibCalculator` est la pierre angulaire du programme. Elle encapsule les variables nécessaires au calcul des nombres de Fibonacci (instances de `big.Int`) et assure une réutilisation optimale de celles-ci. La méthode `Calculate` implémente un algorithme efficace basé sur la décomposition binaire pour calculer les nombres de Fibonacci.

### 2. `WorkerPool`
La structure `WorkerPool` gère un ensemble d'instances de `FibCalculator`, assurant la parallélisation des calculs. Elle permet de répartir la charge de travail de manière équilibrée entre plusieurs goroutines, réduisant ainsi le temps total de calcul.

### 3. Calcul Parallèle avec `calcFibonacci`
La fonction `calcFibonacci` est responsable de calculer une portion de la séquence de Fibonacci et d'envoyer le résultat partiel à travers un canal. Cette fonction est exécutée par plusieurs goroutines, permettant ainsi de paralléliser l'exécution.

### 4. Fonction `main`
La fonction principale coordonne l'ensemble du processus de calcul en divisant le travail entre les travailleurs, en collectant les résultats partiels, et en écrivant les résultats finaux dans un fichier texte. Elle fournit également des informations sur le temps d'exécution et la performance du calcul.

## Prérequis

- **Go (Golang)** : Ce programme nécessite la présence de Go sur votre système. Vous pouvez télécharger Go à partir du site officiel : [https://golang.org/](https://golang.org/).
- **Ressources Matérielles** : Pour maximiser les bénéfices de cette implémentation, il est recommandé d'utiliser une machine multi-cœurs.

## Instructions d'Utilisation

1. **Clôner le Projet**  
   ```bash
   git clone <url_du_répertoire>
   cd <nom_du_répertoire>
   ```

2. **Compiler et Exécuter le Programme**  
   Utilisez la commande suivante pour compiler et exécuter le programme :
   ```bash
   go run main.go
   ```

3. **Modifier le Nombre de Calculs**  
   Vous pouvez modifier la valeur de `n` dans la fonction `main()` pour ajuster le nombre de nombres de Fibonacci à calculer. Notez que pour des valeurs très grandes, les calculs peuvent être extrêmement longs.

## Sortie du Programme

- **Fichier de Résultat** : Les résultats sont écrits dans le fichier `fibonacci_result.txt`. Ce fichier contient la somme des nombres de Fibonacci calculés, le temps total d'exécution, ainsi que le nombre de calculs effectués.
- **Affichage des Résultats** : Le contenu du fichier de résultat est affiché à la fin de l'exécution du programme.

## Remarques Importantes

- **Complexité du Calcul** : La méthode de calcul utilisée est très efficace, mais les valeurs de `n` très grandes peuvent encore demander un temps de calcul significatif et nécessitent beaucoup de mémoire.
- **Gestion des Ressources** : Pour garantir l'efficacité des calculs parallèles, des verrous (`mutex`) sont utilisés afin de s'assurer que plusieurs threads ne modifient pas les mêmes variables simultanément.

## Contributions

Les contributions sont les bienvenues pour améliorer ce projet. Vous pouvez soumettre des "pull requests" pour suggérer des améliorations ou signaler des problèmes en utilisant l'élément "issues" de GitHub.

## Licence

Ce projet est sous licence MIT. Vous êtes libre de l'utiliser, le modifier et le distribuer selon les termes de cette licence.