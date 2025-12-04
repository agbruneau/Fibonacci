# Étude d'opportunité : Migration vers Rust

Ce document présente une analyse détaillée des impacts, efforts, bénéfices et inconvénients liés à la migration du projet `fibcalc` (actuellement en Go) vers le langage Rust.

## 1. Synthèse

La migration vers Rust est un projet d'envergure (**estimé à 3-6 semaines-homme**) qui transformerait fondamentalement l'architecture du projet.
*   **Performance :** Gains potentiels **significatifs (+20% à +50%)** uniquement si l'implémentation repose sur **GMP** (via la crate `rug`). Une implémentation "Rust pur" (`num-bigint`) serait probablement **plus lente** que le code Go actuel optimisé en assembleur.
*   **Complexité :** Élevée. Le code Go actuel utilise des techniques avancées (`unsafe`, `go:linkname`) pour accéder aux primitives bas niveau de `math/big`. Le portage nécessiterait une réécriture complète de la couche mathématique (FFT).
*   **Recommandation :** Non recommandée sauf si la latence déterministe (sans GC) est un critère absolu, ou si l'objectif est purement académique.

---

## 2. Analyse des Efforts (Coût de migration)

### 2.1 Volume de Code
Le codebase actuel représente environ **18 000 lignes de code Go**.
*   **Coeur algorithmique (`internal/fibonacci`, `internal/bigfft`)** : ~40% du code. C'est la partie la plus critique et la plus difficile à porter.
*   **Infrastructure (CLI, Config, API)** : ~60% du code. Facile à porter mais chronophage.

### 2.2 Complexité Technique (Le défi `bigfft`)
Le module `internal/bigfft` actuel est hautement optimisé. Il utilise `go:linkname` pour appeler directement des fonctions assembleur internes de la librairie standard Go (`addVV`, `mulAddVWW`).
*   **En Rust :** Il n'existe pas d'équivalent direct "plug-and-play" pour un FFT sur grands entiers qui s'interface nativement avec les structures internes d'une librairie BigInt standard.
*   **Effort :** Il faudrait soit réécrire ces primitives en assembleur/Rust unsafe, soit trouver une librairie mathématique spécialisée (rare et complexe).

### 2.3 Écosystème et Dépendances
| Composant Go | Équivalent Rust suggéré | Complexité d'adaptation |
| :--- | :--- | :--- |
| `math/big` | `rug` (bindings GMP) ou `num-bigint` | **Élevée** (voir section Performance) |
| `cobra` (CLI) | `clap` | Faible |
| `gopter` (Property Testing) | `proptest` | Moyenne |
| `sync.Pool` | Réutilisation manuelle / `arena` | Moyenne (Gestion mémoire manuelle) |
| `errgroup` / Goroutines | `rayon` (parallélisme de données) | Moyenne |

---

## 3. Analyse de Performance (Gains potentiels)

C'est le point central de la demande. La performance dépendra entièrement du choix de la librairie de grands entiers (BigInt).

### Scénario A : Rust pur (`num-bigint`)
*   **Performance :** Probablement **inférieure** à Go.
*   **Raison :** La librairie standard Go `math/big` est écrite en grande partie en assembleur optimisé par architecture. La crate Rust `num-bigint` est écrite en Rust pur. Bien que rapide, elle peine souvent à battre l'assembleur manuel sur les opérations arithmétiques brutes.

### Scénario B : Rust avec GMP (`rug`)
*   **Performance :** Gains attendus de **20% à 50%**.
*   **Raison :** La librairie GMP (GNU Multiple Precision Arithmetic Library), écrite en C/Assembleur, est l'état de l'art mondial. Le code Go est rapide, mais GMP est généralement plus rapide.
*   **Inconvénient majeur :** Cela introduit une dépendance externe C. Le binaire ne sera plus "statique" par défaut, et la compilation deviendra plus complexe (nécessite d'installer GMP sur la machine de build).

### Gestion de la Mémoire et GC
*   **Go (Actuel) :** Utilise `sync.Pool` pour éviter les allocations. Le Garbage Collector (GC) peut introduire des micro-pauses, mais elles sont minimisées par le pooling.
*   **Rust :** Absence de GC. La latence sera parfaitement déterministe.
*   **Gain :** Sur des calculs de plusieurs minutes/heures, le gain lié à l'absence de GC est négligeable (le CPU passe 99% du temps dans les multiplications). Le gain se ferait surtout sur la *stabilité* de l'utilisation mémoire.

---

## 4. Impacts et Risques

### 4.1 Maintenance et Compétences
*   Rust a une courbe d'apprentissage plus raide que Go.
*   Le code deviendra plus verbeux pour gérer explicitement la durée de vie des objets (Lifetimes), surtout avec les pools de mémoire complexes utilisés dans `matrix.go`.

### 4.2 Portabilité
*   **Go :** Compilation croisée triviale (`GOOS=windows go build`).
*   **Rust (avec GMP) :** Compilation croisée très difficile (nécessite de cross-compiler la librairie C GMP pour chaque architecture cible).

---

## 5. Bilan : Avantages et Inconvénients

### Avantages (Pourquoi migrer ?)
1.  **Performance Maximale (via GMP) :** Accès à l'arithmétique la plus rapide du monde.
2.  **Sûreté Mémoire :** Garantie absolue contre les *data races* à la compilation (Go les détecte seulement au runtime avec `-race`).
3.  **Robustesse :** Le système de types de Rust (Result/Option) oblige à gérer tous les cas d'erreur, réduisant les bugs en production.

### Inconvénients (Pourquoi ne pas migrer ?)
1.  **Coût de développement :** Réécriture totale nécessaire.
2.  **Complexité du Build :** Perte de la simplicité du binaire statique unique si GMP est utilisé.
3.  **Régressions possibles :** Le code Go actuel est mature et fortement optimisé (FFT custom). Une réécriture risque d'introduire des bugs subtils ou d'être moins performante au début.

## Conclusion

Si l'objectif unique est la **performance brute à tout prix**, une migration vers Rust utilisant **GMP (`rug`)** est la voie à suivre. Cependant, cela complexifiera considérablement la chaîne de compilation et la maintenance.

Si l'objectif est de maintenir un outil performant, portable et facile à maintenir, **rester en Go est préférable**, car l'implémentation actuelle est déjà très proche des limites théoriques du langage.
