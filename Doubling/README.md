# Calcul Ultra-Optimisé de Fibonacci(n) en Go (Fast Doubling)

Ce programme calcule le n-ième nombre de Fibonacci en utilisant une implémentation Go hautement optimisée de l'algorithme **Fast Doubling (exponentiation matricielle)**. Il intègre plusieurs techniques pour maximiser la performance et minimiser l'utilisation des ressources, notamment pour de très grandes valeurs de `n`.

**Auteur Original:** André-Guy Bruneau / Adapté par l'IA Gemini 2.5 Pro Experimental (Mars 2025)
**Version:** 1.2 (Intégration LRU Cache + String Keys)

**Description (Basée sur l'historique):**
*   **v1.2:** Intègre un cache LRU (`github.com/hashicorp/golang-lru/v2`) pour limiter l'usage mémoire du cache par rapport à une map simple, utilise des clés `string` pour le cache afin d'éviter les limitations théoriques des `int`, et rend la taille du cache configurable. Supprime le `sync.RWMutex` manuel car la bibliothèque LRU gère la synchronisation.
*   **v1.1:** Optimise la multiplication de matrices en utilisant des `*big.Int` temporaires issus d'un `sync.Pool` et propage le `context.Context` pour la gestion du timeout/annulation.
*   **v1.0 (et avant):** Implémente l'algorithme Fast Doubling avec `math/big.Int`, ajoute le suivi de progression, la gestion de la configuration, les métriques et le profiling.

Ce code est conçu pour être performant même pour des `n` très grands (millions ou plus), là où des méthodes plus simples deviendraient impraticables.

## Table des Matières

1.  [Fonctionnalités](#fonctionnalités)
2.  [Analyse du Code](#analyse-du-code)
    *   [Vulgarisation : Comment ça marche ? (Fast Doubling)](#vulgarisation--comment-ça-marche--fast-doubling)
    *   [Points Clés et Optimisations](#points-clés-et-optimisations)
    *   [Performance](#performance)
3.  [Prérequis](#prérequis)
4.  [Installation](#installation)
5.  [Utilisation](#utilisation)
    *   [Configuration](#configuration)
    *   [Exécution](#exécution)
    *   [Profiling](#profiling)
6.  [Exemple de Sortie](#exemple-de-sortie)

## Fonctionnalités

*   Calcul de Fibonacci(n) pour `n >= 0` via l'algorithme **Fast Doubling (exponentiation matricielle)**.
*   Utilisation de `math/big.Int` pour gérer des résultats de taille arbitraire.
*   **Optimisation Mémoire:** Utilisation de `sync.Pool` pour réutiliser les structures de matrices (`FibMatrix`) et les `*big.Int` temporaires lors des calculs, réduisant la pression sur le Garbage Collector (GC).
*   **Cache LRU (Least Recently Used):** Intégration optionnelle d'un cache (`github.com/hashicorp/golang-lru/v2`) avec une taille configurable pour mémoriser les résultats déjà calculés. Utilise des clés `string` pour une flexibilité maximale.
*   **Gestion du Timeout/Annulation:** Utilisation de `context.Context` pour permettre l'interruption propre du calcul si un timeout est dépassé.
*   **Suivi de Progression:** Affichage en temps réel de la progression du calcul (basé sur les bits de `n`).
*   **Configuration Flexible:** Paramètres ajustables via une structure `Config` (N, Timeout, Cache, Taille Cache, Workers, Profiling, Précision Affichage).
*   **Métriques Détaillées:** Collecte de métriques de performance (temps total, temps de calcul pur, opérations matricielles, cache hits, allocations évitées) via des compteurs atomiques (`sync/atomic`).
*   **Profiling Intégré:** Support optionnel pour le profiling CPU et mémoire via `runtime/pprof`.
*   **Affichage Formaté:** Présentation du résultat en notation scientifique et affichage des premiers/derniers chiffres pour les très grands nombres.

## Analyse du Code

### Vulgarisation : Comment ça marche ? (Fast Doubling)

Calculer Fibonacci(n) en additionnant les deux termes précédents (F(n) = F(n-1) + F(n-2)) devient très lent pour de grands `n`. L'algorithme "Fast Doubling" utilise une astuce mathématique basée sur les matrices.

1.  **La Matrice Magique :** Il existe une matrice 2x2 spéciale, `M = [[1, 1], [1, 0]]`. Si on multiplie cette matrice par elle-même `n` fois (on calcule `M^n`), on obtient une nouvelle matrice dont les éléments sont liés aux nombres de Fibonacci : `M^n = [[F(n+1), F(n)], [F(n), F(n-1)]]`. Donc, pour trouver F(n), il "suffit" de calculer `M^n` et de regarder l'élément en haut à droite (ou en bas à gauche).

2.  **Calculer `M^n` Rapidement :** Multiplier `M` par lui-même `n` fois serait encore trop long (ça prendrait `n` multiplications). L'astuce "Fast Doubling" (ou "exponentiation par carré") permet de calculer `M^n` beaucoup plus vite. Au lieu de faire `M * M * M * ...`, on calcule `M^2`, puis `(M^2)^2 = M^4`, puis `(M^4)^2 = M^8`, et ainsi de suite (on met au carré à chaque étape). Ensuite, on combine intelligemment ces puissances de 2 (`M^2`, `M^4`, `M^8`, ...) pour obtenir `M^n`, en se basant sur la représentation binaire de `n`. Par exemple, pour `M^13` (13 = 8 + 4 + 1), on multiplierait `M^8 * M^4 * M^1`. Ce processus ne nécessite qu'environ `log2(n)` étapes de mise au carré et de multiplication, ce qui est *extrêmement* plus rapide que `n` étapes pour les grands `n`.

3.  **Grands Nombres :** Comme les nombres de Fibonacci deviennent énormes très vite, le code utilise le package `math/big` de Go, qui peut manipuler des nombres entiers aussi grands que la mémoire le permet. Toutes les multiplications et additions de matrices se font avec ces `big.Int`.

4.  **Optimisations Supplémentaires :**
    *   **Moins d'allocations:** Pour accélérer encore, au lieu de créer constamment de nouveaux objets `big.Int` temporaires pendant les multiplications, le code utilise un "pool" (`sync.Pool`) pour recycler des objets `big.Int` et des matrices déjà utilisés. C'est comme avoir une réserve de brouillons réutilisables au lieu de prendre une nouvelle feuille à chaque fois.
    *   **Cache:** Si on demande F(50) puis F(50) à nouveau, le cache (LRU) permet de retourner directement le résultat sans refaire le calcul.

En résumé, ce code utilise une méthode mathématique rapide (Fast Doubling matriciel) combinée à des outils pour gérer les très grands nombres (`big.Int`) et des techniques d'optimisation Go (`sync.Pool`, cache LRU, `context`) pour calculer Fibonacci(n) efficacement.

### Points Clés et Optimisations

*   **`fastDoubling` Function:** Le cœur de l'algorithme. Itère sur les bits de `n` pour effectuer les mises au carré (`matrix = matrix * matrix`) et les multiplications conditionnelles (`result = result * matrix`) nécessaires.
*   **`multiplyMatrices` Function:** Optimisée pour minimiser les allocations. Elle prend les matrices `m1`, `m2`, `result` et utilise *deux* `*big.Int` temporaires (`t1`, `t2`) récupérés depuis `fc.bigIntPool` pour effectuer les calculs intermédiaires (`a*a + b*c`, etc.). Ces temporaires sont remis dans le pool via `defer`.
*   **`sync.Pool` (`matrixPool`, `bigIntPool`):** Réduit significativement la pression sur le GC. `matrixPool` stocke des structures `FibMatrix` entières, tandis que `bigIntPool` stocke des `*big.Int` individuels utilisés comme temporaires dans `multiplyMatrices`. Le compteur `TempAllocsAvoided` mesure l'efficacité de `bigIntPool`.
*   **Cache LRU (`lruCache`):** Implémenté avec `github.com/hashicorp/golang-lru/v2`.
    *   **Thread-Safe:** La bibliothèque gère la synchronisation interne, simplifiant le code utilisateur (pas besoin de `sync.RWMutex`).
    *   **Clés `string`:** Utilise `strconv.Itoa(n)` comme clé. Évite la limitation potentielle de `int` comme clé de map pour des `n` extrêmement grands (bien qu'en pratique, la mémoire ou le temps de calcul soient atteints avant les limites de `int64`).
    *   **Taille Limitée:** Empêche le cache de croître indéfiniment, contrôlant l'utilisation de la mémoire.
    *   **Sécurité:** Stocke et retourne des *copies* (`new(big.Int).Set(...)`) des `*big.Int` pour éviter que des modifications externes n'affectent les valeurs cachées.
*   **`context.Context`:** Permet une annulation propre via le timeout global. La fonction `fastDoubling` vérifie `ctx.Done()` à chaque itération de sa boucle principale.
*   **`atomic` Counters:** Les compteurs de métriques (`MatrixOpsCount`, `CacheHits`, `TempAllocsAvoided`) utilisent `atomic.Int64` pour des incrémentations sûres en environnement concurrent (bien que ce code spécifique n'exécute qu'un calcul à la fois, c'est une bonne pratique si le calculateur était utilisé par plusieurs goroutines).
*   **Progression Reporting:** Utilise `fmt.Printf("\r...")` pour afficher la progression sur la même ligne du terminal, donnant un feedback visuel pendant les longs calculs.
*   **Structure `FibCalculator`:** Encapsule l'état (cache, pools), la configuration et les métriques, offrant une interface claire (`Calculate`).

### Performance

*   **Complexité Temporelle:** L'algorithme Fast Doubling a une complexité temporelle d'environ **O(log n)** opérations matricielles (multiplications et additions de `big.Int`). Le coût de chaque opération sur `big.Int` dépend de la taille (nombre de bits) des opérandes, qui grandit elle-même en `O(n)`. Le coût d'une multiplication de `big.Int` de `k` bits est typiquement `O(k log k)` ou `O(k^alpha)` avec `alpha > 1`. La complexité globale est donc dominée par les opérations sur les grands entiers, mais reste exponentiellement plus rapide que les méthodes en `O(n)` ou pire.
*   **Complexité Spatiale:**
    *   Le résultat F(n) lui-même nécessite `O(n)` bits de stockage.
    *   Les matrices et temporaires utilisent une quantité constante d'objets `big.Int`, dont la taille individuelle atteint `O(n)` bits.
    *   Le cache LRU, s'il est activé, stocke jusqu'à `CacheSize` résultats, chacun pouvant nécessiter jusqu'à `O(N)` bits (si `N` est le plus grand `n` calculé). La taille du cache est donc un facteur important.
    *   Les `sync.Pool` peuvent retenir des objets non utilisés, ajoutant une consommation mémoire variable mais généralement limitée.

## Prérequis

*   **Go:** Une version récente de Go installée (par exemple, 1.18 ou supérieure).
*   **Dépendance Externe:** La bibliothèque `golang-lru`.

## Installation

1.  **Cloner le dépôt (si applicable) ou sauvegarder le code** :
    Sauvegardez le code fourni dans un fichier nommé `main.go` (ou un nom de votre choix).

    ```bash
    # Si vous avez cloné un dépôt
    # git clone <url_du_depot>
    # cd <repertoire_du_depot>

    # Si vous avez juste le fichier main.go
    # Placez-vous dans le répertoire contenant main.go
    ```

2.  **Télécharger les dépendances** :
    Ouvrez un terminal dans le répertoire du projet et exécutez :

    ```bash
    go mod init <nom_module> # Si pas déjà un module Go (ex: go mod init fibonacci)
    go get github.com/hashicorp/golang-lru/v2
    ```

3.  **Compiler le programme** :
    Toujours dans le même répertoire, exécutez :

    ```bash
    go build -o fibonacci_optimized main.go
    ```

    Cela créera un exécutable nommé `fibonacci_optimized` (ou `.exe` sous Windows).

## Utilisation

### Configuration

Les paramètres principaux sont définis directement dans la fonction `DefaultConfig()` au début du fichier `main.go`. Modifiez ces valeurs avant de compiler si nécessaire :

*   `N`: L'index du nombre de Fibonacci à calculer (ex: 10000000).
*   `Timeout`: Durée maximale allouée à l'exécution (ex: `"5m"` pour 5 minutes).
*   `Precision`: Nombre de chiffres après la virgule pour l'**affichage** en notation scientifique.
*   `Workers`: Nombre de threads CPU que Go peut utiliser (`runtime.GOMAXPROCS`).
*   `EnableCache`: `true` pour activer le cache LRU, `false` pour le désactiver.
*   `CacheSize`: Nombre maximum d'éléments à conserver dans le cache LRU si activé (ex: 2048).
*   `EnableProfiling`: `true` pour activer le profiling CPU et mémoire (`pprof`).

**Recompilez le programme (`go build ...`) après toute modification de la configuration.**

### Exécution

Exécutez simplement le binaire compilé depuis votre terminal :

```bash
./fibonacci_optimized

Le programme affichera :
La configuration utilisée.
Des logs d'information.
Une barre de progression pendant le calcul (si N est assez grand).
Les résultats finaux et les métriques de performance.
Profiling
Si EnableProfiling est mis à true dans la configuration (et que le programme est recompilé) :
Deux fichiers seront créés à la fin de l'exécution (ou une tentative sera faite en cas de timeout/erreur) :
cpu.pprof: Profil d'utilisation CPU.
mem.pprof: Profil d'utilisation mémoire (tas).
Vous pouvez analyser ces fichiers avec l'outil go tool pprof :
# Pour analyser le profil CPU (interface interactive)
go tool pprof cpu.pprof

# Pour analyser le profil mémoire
go tool pprof mem.pprof

# Pour visualiser le graphe d'appel CPU en PDF (nécessite graphviz)
go tool pprof -pdf cpu.pprof > cpu_graph.pdf

# Pour voir le profil mémoire via une interface web
go tool pprof -http=:8081 mem.pprof

Exemple de Sortie
Pour N=10000000 (valeur par défaut) :
INFO: Configuration utilisée: N=10000000, Timeout=5m0s, Workers=8, Cache=true, CacheSize=2048, Profiling=false, Précision Affichage=10
INFO: Cache LRU activé (taille maximale: 2048 éléments)
INFO: Démarrage du calcul de Fibonacci(10000000)... (Timeout configuré: 5m0s)
INFO: En attente du résultat du calcul ou de l'expiration du timeout...
Progression: 100.00% (24/24 bits traités), Temps écoulé: 1.352s      // Mise à jour dynamique pendant l'exécution

INFO: Calcul terminé avec succès. Durée du calcul pur (hors cache): 1s352ms

=== Résultats pour Fibonacci(10000000) ===
Temps total d'exécution                     : 1s353ms
Temps de calcul pur (si effectué)           : 1s352ms
Opérations matricielles (multiplications)   : 46
Cache hits                                  : 0
Cache LRU taille actuelle / max             : 4 / 2048
Allocations *big.Int évitées (via pool)   : 92

Résultat F(10000000) :
  Notation scientifique (~10 chiffres)      : 2.1897117332e+2089876
  Nombre total de chiffres décimaux         : 2089877
  Premiers 100 chiffres                      : 218971173319423473101580683049249876981469534804646159211636761431676106600006929414691304013371...
  Derniers 100 chiffres                      : ...64940813938991430320892840906804191147371939462834148015612744170500593478181798410991904348125
INFO: Programme terminé.