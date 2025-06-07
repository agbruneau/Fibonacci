Calculateur de Fibonacci Concurrent en Go

Ce projet est un outil en ligne de commande écrit en Go pour calculer le n-ième nombre de Fibonacci. Sa particularité est d'implémenter plusieurs algorithmes distincts et de les exécuter en parallèle pour comparer leur performance, leur consommation mémoire et valider leurs résultats.

L'application est conçue pour être à la fois un outil pratique et une démonstration de plusieurs concepts avancés de Go, tels que la concurrence, la gestion des contextes, l'optimisation de la mémoire et les tests de performance (benchmarking).

✨ Caractéristiques

Calcul de Très Grands Nombres : Utilise le package math/big pour calculer des nombres de Fibonacci bien au-delà des limites des types entiers standards.

Exécution Concurrente : Lance plusieurs algorithmes en parallèle grâce aux goroutines pour une comparaison directe des performances.

Multi-Algorithmes : Implémente trois méthodes de calcul différentes avec des caractéristiques de performance distinctes.

Affichage de la Progression : Affiche en temps réel la progression de chaque algorithme sur une seule ligne.

Gestion des Timeouts : Utilise context.WithTimeout pour garantir que le programme se termine proprement si les calculs sont trop longs.

Optimisation de la Mémoire : Emploie un sync.Pool pour recycler les objets *big.Int et réduire la pression sur le ramasse-miettes (Garbage Collector).

Suite de Tests Complète : Inclut des tests unitaires pour valider l'exactitude des algorithmes et des benchmarks pour mesurer leurs performances de manière formelle.

🛠️ Prérequis

Go (version 1.16+ recommandée)

Git (pour cloner le projet)

🚀 Installation

Clonez le dépôt sur votre machine locale :

git clone https://github.com/votre-utilisateur/votre-repo.git


(Remplacez l'URL par celle de votre dépôt)

Accédez au répertoire du projet :

cd votre-repo
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Sh
IGNORE_WHEN_COPYING_END
Usage

L'outil se lance directement depuis la ligne de commande.

Exécution simple

Pour lancer le calcul avec les valeurs par défaut (n=100000, timeout=1m) :

go run .
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Sh
IGNORE_WHEN_COPYING_END
Options de la ligne de commande

Vous pouvez personnaliser l'exécution avec les options suivantes :

-n <nombre> : Spécifie l'index n du nombre de Fibonacci à calculer.

-timeout <durée> : Spécifie le délai d'attente global pour l'exécution (ex: 30s, 2m, 1h).

Exemples

Calculer F(500 000) avec un timeout de 30 secondes :

go run . -n 500000 -timeout 30s
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Sh
IGNORE_WHEN_COPYING_END

Calculer F(1 000 000) avec un timeout de 5 minutes :

go run . -n 1000000 -timeout 5m
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Sh
IGNORE_WHEN_COPYING_END
Exemple de Sortie
2023/10/27 10:30:00 Calcul de F(200000) avec un timeout de 1m...
2023/10/27 10:30:00 Algorithmes à exécuter: Doublage Rapide, Matrice 2x2, Binet
2023/10/27 10:30:00 Lancement des calculs concurrents...
Doublage Rapide:  100.00%   Matrice 2x2:     100.00%   Binet:           100.00%                 
2023/10/27 10:30:01 Calculs terminés.

--------------------------- RÉSULTATS ORDONNÉS ---------------------------
Doublage Rapide  : 8.8475ms     [OK              ] Résultat: 25974...03125
Matrice 2x2      : 18.0673ms    [OK              ] Résultat: 25974...03125
Binet            : 43.1258ms    [OK              ] Résultat: 25974...03125
------------------------------------------------------------------------

🏆 Algorithme le plus rapide (ayant réussi) : Doublage Rapide (8.848ms)
Nombre de chiffres de F(200000) : 41798
Valeur (notation scientifique) ≈ 2.59740692e+41797
✅ Tous les résultats valides produits sont identiques.
2023/10/27 10:30:01 Programme terminé.
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
IGNORE_WHEN_COPYING_END
🧠 Algorithmes Implémentés

Doublage Rapide (Fast Doubling)
Un des algorithmes les plus rapides pour les grands entiers. Il utilise les identités F(2k) = F(k) * [2*F(k+1) – F(k)] et F(2k+1) = F(k)² + F(k+1)² pour réduire considérablement le nombre d'opérations.

Exponentiation de Matrice 2x2
Une approche classique qui repose sur le fait que [[1,1],[1,0]]^n = [[F(n+1), F(n)], [F(n), F(n-1)]]. Le calcul est optimisé via l'exponentiation par la carré.

Formule de Binet
Une solution analytique basée sur le nombre d'or. Elle est calculée en utilisant des nombres à virgule flottante de haute précision (big.Float). Bien qu'élégante, elle est généralement moins performante et peut souffrir d'erreurs de précision pour de très grands n.

🏗️ Architecture du Code

code.go (ou main.go) : Contient l'ensemble du code, y compris la logique principale, les implémentations des algorithmes, la gestion de la concurrence et l'affichage de la progression.

main_test.go : Contient les tests unitaires et les benchmarks.

La concurrence est gérée par un sync.WaitGroup pour attendre la fin de tous les calculs. Un canal unique (progressAggregatorCh) centralise les mises à jour de progression de toutes les goroutines, qui sont ensuite affichées par une goroutine dédiée.

✅ Tests

Le projet est fourni avec une suite de tests pour garantir son bon fonctionnement.

Lancer les Tests Unitaires

Pour vérifier que les algorithmes produisent des résultats corrects pour des valeurs connues :

go test -v
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Sh
IGNORE_WHEN_COPYING_END
Lancer les Benchmarks

Pour mesurer et comparer les performances (temps d'exécution et allocations mémoire) de chaque algorithme :

go test -bench .
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Sh
IGNORE_WHEN_COPYING_END
📜 Licence

Ce projet est distribué sous la licence MIT. Voir le fichier LICENSE pour plus de détails.