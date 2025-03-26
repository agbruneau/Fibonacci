# Calcul Ultra-Optimisé de Fibonacci(n) en Go (Version 5.2)

Ce programme calcule le n-ième nombre de Fibonacci, F(n), en utilisant une approche hautement optimisée en langage Go. Il implémente l'algorithme de "fast doubling" (basé sur l'exponentiation matricielle) avec `math/big` pour gérer des nombres arbitrairement grands, tout en intégrant des optimisations de mémoire et de performance.

## Fonctionnalités Clés

*   **Algorithme Rapide :** Utilise l'algorithme "Fast Doubling" (complexité O(log n)) pour un calcul très rapide, même pour des valeurs de `n` extrêmement élevées.
*   **Grands Nombres :** Support complet des très grands nombres grâce au package `math/big`.
*   **Optimisation Mémoire :** Utilisation intensive de `sync.Pool` pour la réutilisation des objets `big.Int` et des structures de matrices temporaires, réduisant significativement la pression sur le garbage collector et le nombre d'allocations.
*   **Suivi de Progression :** Affiche la progression du calcul en temps réel (pourcentage des bits de `n` traités et temps écoulé), particulièrement utile pour les grandes valeurs de `n`.
*   **Cache :** Implémente un cache simple en mémoire (activable/désactivable) pour stocker les résultats déjà calculés, accélérant les appels répétés pour le même `n`.
*   **Métriques de Performance :** Collecte et affiche des métriques détaillées :
    *   Temps total d'exécution.
    *   Temps de calcul pur (hors préparation, cache, etc.).
    *   Nombre d'opérations matricielles effectuées.
    *   Nombre de "cache hits".
    *   Nombre estimé d'allocations `big.Int` évitées grâce aux pools.
*   **Configuration Facile :** Paramètres principaux (N, timeout, nombre de workers, cache, profiling) configurables via la structure `Config`.
*   **Gestion du Timeout :** Intègre un mécanisme de timeout pour arrêter le calcul s'il dépasse une durée définie.
*   **Profiling Intégré :** Support optionnel pour le profiling CPU et mémoire via le package standard `runtime/pprof`, générant des fichiers `cpu.pprof` et `mem.pprof` pour analyse.
*   **Gestion de Concurrence :** Utilise `sync.RWMutex` pour protéger l'accès concurrentiel au cache.

## Prérequis

*   **Go :** Version 1.13 ou ultérieure (pour `math/bits`). Version 1.18+ recommandée.

## Installation / Compilation

1.  Sauvegardez le code source ci-dessous dans un fichier nommé `main.go`.
2.  Compilez le programme via le terminal :
    ```bash
    go build -o fibonacci_calculator main.go
    ```
    Cela créera un exécutable nommé `fibonacci_calculator` (ou `fibonacci_calculator.exe` sous Windows).

## Utilisation

Exécutez simplement le binaire compilé :

```bash
./fibonacci_calculator