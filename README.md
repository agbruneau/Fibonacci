# Calcul de Fibonacci en Notation Scientifique Optimisé

Ce projet implémente un programme en Go permettant de calculer le n‑ième nombre de Fibonacci de manière optimisée. Le calcul repose sur l’algorithme du doublement (doubling method), qui réduit la complexité à **O(log n)**. Afin de maximiser la puissance de calcul disponible, le programme exploite la parallélisation des opérations lourdes sur de grands entiers (via le package `math/big`) en utilisant des **goroutines** et des **canaux**. De plus, il gère la durée d’exécution à l’aide d’un contexte avec timeout et collecte des métriques de performance.

---

## Sommaire

- [Description](#description)
- [Caractéristiques](#caractéristiques)
- [Installation et Exécution](#installation-et-exécution)
- [Structure du Code](#structure-du-code)
- [Utilisation et Personnalisation](#utilisation-et-personnalisation)
- [Licence](#licence)

---

## Description

Ce programme calcule le n‑ième nombre de Fibonacci en combinant plusieurs techniques avancées :

- **Algorithme du doublement** : Permet de calculer Fibonacci(n) en temps logarithmique en parcourant les bits de n et en mettant à jour deux variables intermédiaires.
- **Parallélisation** : Les opérations de multiplication sur de grands entiers sont effectuées en parallèle grâce à des goroutines, permettant ainsi d’exploiter pleinement les architectures multi‑cœurs.
- **Gestion du Timeout** : Un contexte avec timeout est mis en place pour s’assurer que l’exécution du calcul ne dépasse pas une durée maximale définie.
- **Collecte de métriques** : Le programme mesure le temps total d’exécution et le nombre de calculs réalisés, offrant une vision claire des performances.
- **Formatage en notation scientifique** : Le résultat est affiché en notation scientifique avec un exposant converti en caractères Unicode superscript pour une lecture facilitée des nombres volumineux.

---

## Caractéristiques

- **Calcul Optimisé** : Utilisation de l’algorithme du doublement pour réduire la complexité.
- **Exploitation du Parallélisme** : Parallélisation des multiplications coûteuses via des goroutines et des canaux.
- **Gestion de Contexte** : Implémentation d’un timeout pour éviter une exécution prolongée.
- **Support des Grands Entiers** : Utilisation du package `math/big` pour gérer les très grands nombres.
- **Affichage Lisible** : Formatage du résultat en notation scientifique avec conversion des exposants en caractères superscript.
- **Mesure des Performances** : Collecte et affichage des métriques de performance (temps d’exécution, nombre de calculs).

---

## Installation et Exécution

### Prérequis

- [Go](https://golang.org/dl/) (version 1.13 ou ultérieure recommandée)

### Compilation

Clonez ce dépôt puis compilez le programme en utilisant la commande suivante dans un terminal :

```bash
git clone https://votre-repository-url.git
cd votre-repertoire
go build -o fibonacci
