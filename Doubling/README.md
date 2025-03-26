# Calcul Optimisé de Fibonacci en Go (v1.2)

Ce programme Go implémente un calcul ultra-optimisé du N-ième nombre de Fibonacci, F(N), en utilisant l'algorithme d'exponentiation matricielle par doublement rapide ("Fast Doubling"). Il est conçu pour calculer des termes très élevés de la suite de Fibonacci de manière performante, même pour des N de l'ordre de centaines de millions ou plus.

La version actuelle (1.2) intègre des optimisations avancées telles que l'utilisation de pools d'objets (`sync.Pool`), un cache LRU (Least Recently Used) pour les résultats intermédiaires, la propagation de contexte pour la gestion des timeouts, et un suivi de progression détaillé.

**Auteurs :** André-Guy Bruneau (Concept original), IA Gemini 2.5 Pro Experimental (Adaptations et raffinements V1.1, V1.2 - 03/2025)

## Fonctionnalités Principales

*   **Algorithme Efficace :** Utilise l'exponentiation matricielle (méthode "Fast Doubling") avec une complexité temporelle O(log N) multiplications de grands entiers.
*   **Grands Nombres :** Utilise le package `math/big` pour gérer des résultats potentiellement gigantesques sans dépassement de capacité.
*   **Optimisation Mémoire :**
    *   Utilisation intensive de `sync.Pool` pour réutiliser les objets `*big.Int` temporaires et les structures de matrices (`FibMatrix`), réduisant drastiquement la pression sur le garbage collector.
    *   Multiplication matricielle optimisée nécessitant seulement 2 `*big.Int` temporaires par opération.
*   **Cache LRU :** Implémente un cache LRU thread-safe (via `github.com/hashicorp/golang-lru/v2`) pour mémoriser les résultats déjà calculés, utile si le programme est appelé plusieurs fois ou intégré dans une application plus large. Taille du cache configurable.
*   **Gestion de Timeout :** Utilise `context.Context` pour permettre l'annulation propre du calcul si un délai maximum (`Timeout`) est dépassé.
*   **Suivi de Progression :** Affiche en temps réel (mise à jour environ chaque seconde) le pourcentage de progression du calcul (basé sur les bits de N traités) et le temps écoulé.
*   **Métriques de Performance :** Collecte et affiche des métriques à la fin de l'exécution (temps total, temps de calcul pur, nombre d'opérations matricielles, hits du cache, allocations évitées grâce aux pools).
*   **Profiling Intégré :** Support optionnel pour le profiling CPU et mémoire via le package standard `runtime/pprof`.
*   **Configuration Flexible :** Paramètres principaux (N, Timeout, Workers, Cache, etc.) facilement modifiables dans le code (`DefaultConfig`).
*   **Sortie Lisible :** Affiche le résultat en notation scientifique et, si la taille le permet, les premiers et derniers chiffres du nombre exact.

## Prérequis

*   **Go:** Version 1.18 ou supérieure (pour les génériques utilisés par la dépendance `golang-lru/v2`).
*   **Modules Go:** Le projet doit être configuré pour utiliser les modules Go.

## Installation et Mise en Place

1.  **Cloner ou Télécharger :** Obtenez le code source (fichier `main.go`).
2.  **Accéder au Répertoire :** Ouvrez un terminal et naviguez jusqu'au dossier contenant `main.go`.
3.  **Initialiser les Modules (si non existant) :** Si vous n'avez pas de fichier `go.mod`, exécutez :
    ```bash
    go mod init <nom_du_module>
    # Exemple: go mod init fibonacci_calculator
    ```
    (Remplacez `<nom_du_module>` par un nom pertinent pour votre projet).
4.  **Télécharger les Dépendances :** Exécutez la commande suivante pour télécharger la bibliothèque de cache LRU et l'ajouter à votre `go.mod` / `go.sum` :
    ```bash
    go get github.com/hashicorp/golang-lru/v2
    ```

## Utilisation

*   **Exécution directe :**
    ```bash
    go run main.go
    ```
*   **Compilation puis exécution :**
    1.  Compilez le programme :
        ```bash
        go build -o fibonacci_calculator .
        # Vous pouvez omettre '-o fibonacci_calculator', le nom par défaut sera 'main' ou le nom du répertoire.
        ```
    2.  Exécutez le binaire compilé :
        ```bash
        ./fibonacci_calculator
        ```

Le programme utilisera les paramètres définis dans la fonction `DefaultConfig()` du fichier `main.go`.

## Configuration

Les paramètres principaux peuvent être ajustés en modifiant les valeurs retournées par la fonction `DefaultConfig()` dans `main.go`:

*   `N`: L'index du nombre de Fibonacci à calculer (par défaut : 100 000 000).
*   `Timeout`: La durée maximale allouée pour le calcul (par défaut : 5 minutes).
*   `Precision`: Le nombre de chiffres significatifs pour l'affichage en notation scientifique (par défaut : 10).
*   `Workers`: Le nombre de cœurs CPU à utiliser (`GOMAXPROCS`, par défaut : tous les cœurs disponibles).
*   `EnableCache`: Activer (`true`) ou désactiver (`false`) le cache LRU (par défaut : `true`).
*   `CacheSize`: Le nombre maximum d'éléments à conserver dans le cache LRU si activé (par défaut : 2048).
*   `EnableProfiling`: Activer (`true`) ou désactiver (`false`) la génération des fichiers de profiling `pprof` (par défaut : `false`).

## Profiling

Si `EnableProfiling` est mis à `true` dans la configuration :

1.  Deux fichiers seront générés à la racine du projet lors de l'exécution :
    *   `cpu.pprof`: Profil d'utilisation CPU.
    *   `mem.pprof`: Profil d'allocation mémoire (heap).
2.  Vous pouvez analyser ces fichiers avec l'outil `go tool pprof` :
    ```bash
    # Pour le CPU
    go tool pprof cpu.pprof

    # Pour la mémoire
    go tool pprof mem.pprof
    ```
    Consultez la documentation de `pprof` pour les commandes d'analyse (`top`, `web`, `list`, etc.).

## Exemple de Sortie (pour un N plus petit)
INFO: Configuration: N=1000000, Timeout=5m0s, Workers=8, Cache=true, CacheSize=2048, Profiling=false, Précision Affichage=10
INFO: Cache LRU activé (taille: 2048)
INFO: Démarrage du calcul de Fibonacci(1000000)... (Timeout: 5m0s)
INFO: En attente du résultat ou du timeout...
Progress: 100.00% (20/20 bits), Elapsed: 152ms
INFO: Calcul terminé avec succès. Durée calcul pur: 152ms
=== Résultats Fibonacci(1000000) ===
Temps total d'exécution : 153ms
Temps de calcul pur (fastDoubling) : 152ms
Opérations matricielles (multiplications) : 39
Cache hits : 0
Cache LRU taille actuelle/max : 3/2048
Allocations *big.Int évitées (via pool) : 78
Résultat F(1000000) :
Notation scientifique (~10 chiffres) : 1.95328217...e+208987 <-- (tronqué pour l'exemple)
Nombre total de chiffres décimaux : 208988
Premiers 100 chiffres : 195328217....
Derniers 100 chiffres : ...834375
INFO: Programme terminé.

## Détails Techniques
*   **Algorithme:** L'implémentation utilise la relation matricielle suivante :
    ```
    [ F(n+1) F(n)   ] = [ 1 1 ]^n
    [ F(n)   F(n-1) ]   [ 1 0 ]
    ```
    La puissance N de la matrice est calculée efficacement en O(log N) via la méthode d'exponentiation rapide (ici, "Fast Doubling", qui est une variante optimisée). F(N) se trouve alors dans l'élément (0, 1) ou (1, 0) de la matrice résultante.

*   **Pools:** La réutilisation d'objets via `sync.Pool` est cruciale car la multiplication de `big.Int` alloue de nombreux objets temporaires. Éviter ces allocations réduit la charge GC et améliore significativement les performances pour les grands N.

*   **Cache LRU:** Le cache `golang-lru` gère automatiquement l'éviction des éléments les moins récemment utilisés lorsque la taille maximale est atteinte, prévenant une croissance mémoire incontrôlée dans des scénarios d'utilisation prolongée.

---