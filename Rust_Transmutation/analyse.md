# Analyse de la Transmutation : Go vers Rust - Projet Fibonacci

Cette analyse détaille la stratégie, les implications techniques et les défis de la migration (transmutation) du calculateur de Fibonacci haute performance de Go vers Rust.

## 1. Contexte et Vision Stratégique

L'objectif principal est de créer une implémentation de référence en Rust qui **dépasse les performances** de la version Go existante tout en garantissant une **parité fonctionnelle stricte (1:1)**. Ce n'est pas une simple réécriture, mais une "transmutation" visant à exploiter les avantages intrinsèques de Rust (sécurité mémoire sans GC, abstractions à coût nul) pour atteindre de nouveaux sommets d'efficacité.

### Pourquoi Rust ?

- **Performance Déterministe** : Absence de Garbage Collector (GC), éliminant les pauses imprévisibles lors de calculs intensifs sur de grands nombres.
- **Sécurité Mémoire** : Garantie à la compilation, réduisant les risques de fuites mémoire ou de data races dans les algorithmes parallélisés.
- **Écosystème** : Utilisation de bibliothèques modernes (`tokio`, `axum`, `clap`) offrant une ergonomie et des performances de premier plan.

## 2. Architecture et Mapping Technique

La transition implique un changement de paradigme architectural, passant d'un module Go monolithique à un Workspace Rust modulaire.

| Concept Go            | Concept Rust               | Avantage Rust                                                             |
| :-------------------- | :------------------------- | :------------------------------------------------------------------------ |
| **Module (`go.mod`)** | **Workspace Cargo**        | Isolation plus stricte des dépendances et compilation incrémentale.       |
| **Lib (`internal/`)** | **Crate (`fibcalc-core`)** | Réutilisabilité claire et visibilité contrôlée (`pub(crate)`).            |
| **Goroutines**        | **Tokio Tasks**            | Modèle asynchrone ultra-léger, idéal pour le serveur API.                 |
| **`sync.WaitGroup`**  | **`tokio::task::JoinSet`** | Gestion plus robuste des cycles de vie des tâches concurrentes.           |
| **Interfaces**        | **Traits**                 | Polymorphisme statique (Monomorphisation) = inlining et meilleures perfs. |

### Dépendances Clés (La "Holy Trinity" Rust)

1.  **`num-bigint`** : Choix stratégique d'une implémentation "Pure Rust" pour éviter les complexités via FFI (GMP). C'est un pari sur la portabilité (Windows/Linux) au prix potentiel d'une optimisation manuelle nécessaire pour égaler GMP.
2.  **`tokio` + `axum`** : Le standard industriel pour l'asynchrone et le web, garantissant robustesse et scalabilité.
3.  **`clap`** : Pour une CLI riche et typée, supérieure à l'approche standard `flag` de Go.

## 3. Analyse des Algorithmes de Transmutation

Le cœur du projet réside dans le portage fidèle des algorithmes mathématiques.

### Algorithmes Cibles

1.  **Fast Doubling** : L'algorithme roi. En Rust, l'optimisation des clones (`Cow` ou références) sera cruciale pour battre Go, car les `BigInt` sont alloués sur le tas.
2.  **Calcul Matriciel & Strassen** : L'implémentation récursive de Strassen bénéficiera grandement du système de types de Rust pour éviter les erreurs de dimension.
3.  **FFT (Fast Fourier Transform)** : Le défi majeur. L'implémentation de la multiplication de grands nombres via FFT en Rust pur demande une attention particulière à la gestion mémoire pour être compétitive.

### Gestion de la Mémoire

- **Go** : `sync.Pool` est utilisé pour réduire la pression sur le GC.
- **Rust** : `crossbeam::queue::SegQueue` servira de pool d'objets. Cependant, l'absence de GC signifie que la stratégie de propriété ("Ownership") doit être pensée pour minimiser les allocations/désallocations fréquentes de `BigInt`.

## 4. Risques et Défis (Risk Assessment)

### 🔴 Risque Critique : Performance de `num-bigint` vs GMP

L'implémentation Go utilise souvent des bindings vers GMP (via `math/big` optimisé en assembleur) ou une implémentation native très optimisée. `num-bigint` en Rust pur est performant mais peut être plus lent que GMP sur des nombres astronomiques.

- _Mitigation_ : Profiling agressif (`criterion`), et contribution potentielle à `num-bigint` ou implémentation ad-hoc de routines critiques (Karatsuba/Toom-Cook) si nécessaire.

### 🟠 Risque Moyen : Complexité de l'Asynchrone

Mélanger calcul intensif (CPU-bound) et serveur API (IO-bound) nécessite une gestion fine. Bloquer l'executor Tokio avec un calcul Fibonacci long est une erreur classique.

- _Solution_ : Utiliser `tokio::task::spawn_blocking` ou un thread pool dédié (`rayon`) pour les calculs lourds afin de ne pas affamer le serveur HTTP.

### 🟡 Risque Faible : Courbe d'Apprentissage

La rigueur du Borrow Checker peut ralentir le développement initial ("fighting the borrow checker"), notamment sur les structures de données récursives ou partagées.

## 5. Stratégie de Validation et Qualité

Pour garantir le succès de la transmutation, une approche rigoureuse de validation est définie :

1.  **Parité API (Vertébral)** : Les contrats JSON (Request/Response) sont strictement identiques. Un client ne doit pas pouvoir distinguer le serveur Go du serveur Rust.
2.  **Tests de Propriétés (`proptest`)** : Vérification de l'identité de Cassini ($F_{n-1}F_{n+1} - F_n^2 = (-1)^n$) sur des milliers d'itérations aléatoires pour garantir l'exactitude mathématique.
3.  **Benchmarking Comparatif** : Utilisation de "Golden Files" générés par Go pour valider les sorties Rust, et benchmarks croisés pour mesurer les gains de performance (latence, mémoire).

## 6. Conclusion

La transmutation vers Rust n'est pas une simple traduction syntaxique mais une élévation architecturale. Elle promet une application plus robuste, plus portable (Windows/Linux sans douleur GMP) et potentiellement plus rapide. Le succès reposera sur une maîtrise fine des allocations mémoire et une utilisation judicieuse du parallélisme offert par Rust.
