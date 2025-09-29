# FibCalc - Calculateur de Fibonacci Haute Performance

`fibcalc` est un outil en ligne de commande (CLI) écrit en Go, conçu pour le calcul ultra-rapide de très grands nombres de la suite de Fibonacci. Ce projet est autant un outil fonctionnel qu'une démonstration académique de concepts avancés d'ingénierie logicielle, d'algorithmique et d'optimisation de la performance en Go.

Il a été spécifiquement développé pour illustrer :
- L'implémentation d'algorithmes à complexité logarithmique (O(log n)).
- Les techniques de gestion mémoire "zéro-allocation" pour les calculs intensifs.
- Le parallélisme de tâches pour exploiter les architectures CPU multi-cœurs.
- La conception d'une architecture logicielle robuste, modulaire et testable.

## Fonctionnalités

*   **Calculs Extrêmement Rapides** : Utilise des algorithmes de pointe comme le **Fast Doubling** et l'**Exponentiation Matricielle** pour calculer des termes de Fibonacci avec des millions de chiffres en quelques secondes.
*   **Optimisation Zéro-Allocation** : Met en œuvre un système de pool d'objets (`sync.Pool`) pour recycler les structures de données et les grands nombres, minimisant ainsi la charge sur le Garbage Collector et évitant les allocations mémoire dans les boucles de calcul critiques.
*   **Parallélisme Intelligent** : Exploite les CPU multi-cœurs en parallélisant les multiplications de grands nombres au-delà d'un seuil de complexité configurable.
*   **Mode Benchmark** : Compare les performances des différents algorithmes implémentés sur votre machine.
*   **Mode Calibration** : Exécute une série de tests pour déterminer le seuil de parallélisme optimal pour votre configuration matérielle spécifique, vous permettant d'obtenir les meilleures performances possibles.
*   **Interface Utilisateur Soignée** : Affiche une barre de progression dynamique, même lors des calculs parallèles, et formate les résultats pour une lisibilité maximale.
*   **Gestion Robuste des Erreurs** : Gère proprement les timeouts, les signaux d'interruption (Ctrl+C) et les erreurs internes avec des codes de sortie standardisés.

## Installation

Pour utiliser `fibcalc`, vous devez disposer d'une installation Go (version 1.18 ou supérieure est recommandée pour le support des génériques).

1.  **Clonez le dépôt** (ou téléchargez les fichiers) :
    ```bash
    git clone <url-du-repo>
    cd <repertoire-du-repo>
    ```

2.  **Compilez le binaire** :
    La commande `go build` va compiler le code source et créer un exécutable `fibcalc` dans le répertoire `cmd/fibcalc`.
    ```bash
    go build ./cmd/fibcalc
    ```
    Pour une commodité d'utilisation, vous pouvez déplacer ce binaire dans un répertoire de votre `PATH` (ex: `/usr/local/bin`).

## Utilisation

L'outil se contrôle via des flags en ligne de commande.

### Syntaxe de base

```bash
./fibcalc [options]
```

### Options (Flags)

| Flag                | Type      | Défaut                 | Description                                                                                             |
| ------------------- | --------- | ---------------------- | ------------------------------------------------------------------------------------------------------- |
| `-n`                | `uint64`  | `250000000`            | L'indice 'n' du nombre de Fibonacci F(n) à calculer.                                                    |
| `-algo`             | `string`  | `"all"`                | L'algorithme à utiliser. Options : `fast`, `matrix`, ou `all` pour comparer les deux.                   |
| `-timeout`          | `duration`| `5m`                   | Délai maximum pour l'exécution (ex: `10s`, `2m30s`).                                                    |
| `-threshold`        | `int`     | `2048`                 | Seuil (en nombre de bits) au-delà duquel la multiplication parallèle est activée.                       |
| `-calibrate`        | `bool`    | `false`                | Si `true`, lance le mode de calibration pour trouver le meilleur `threshold` et ignore les autres calculs.|
| `-v`, `-verbose`    | `bool`    | `false`                | Si `true`, affiche le résultat complet du nombre de Fibonacci, même s'il est très long.                 |

### Exemples

1.  **Calculer F(1,000,000) avec l'algorithme "Fast Doubling"** :
    ```bash
    ./fibcalc -n 1000000 -algo fast
    ```

2.  **Comparer les performances des algorithmes pour F(10,000,000)** :
    ```bash
    ./fibcalc -n 10000000 -algo all
    ```

3.  **Lancer le mode de calibration** pour trouver le meilleur seuil de parallélisme pour votre machine :
    ```bash
    ./fibcalc -calibrate
    ```
    Le résultat vous recommandera une valeur pour le flag `-threshold` à utiliser pour des performances optimales.

4.  **Calculer un très grand nombre et afficher le résultat complet** :
    ```bash
    ./fibcalc -n 50000000 -v
    ```

## Concepts Architecturaux et Pédagogiques

Ce projet met en œuvre plusieurs patrons de conception et techniques d'ingénierie logicielle avancées :

*   **Séparation des préoccupations (SoC)** : La logique est clairement découpée en couches :
    *   `cmd/fibcalc` : Point d'entrée, parsing des arguments, orchestration.
    *   `internal/fibonacci` : Cœur de la logique métier, implémentation des algorithmes.
    *   `internal/cli` : Couche de présentation (UI), gestion de l'affichage.
*   **Patron Décorateur** : Le `FibCalculator` enveloppe un `coreCalculator` pour ajouter des fonctionnalités transversales (comme la consultation d'une table pré-calculée) sans modifier les algorithmes de base.
*   **Patron Adaptateur** : Le même `FibCalculator` adapte l'interface de communication par canaux (utilisée par l'orchestrateur) à une simple fonction de callback (utilisée par les algorithmes), simplifiant ainsi leur implémentation.
*   **Concurrence Structurée** : Utilisation de `errgroup` pour gérer le cycle de vie de groupes de goroutines, assurant une propagation correcte des erreurs et une annulation propre.
*   **Injection de Dépendances** : Les dépendances (comme les writers de sortie ou les configurations) sont passées explicitement, rendant le code modulaire et facile à tester unitairement.
*   **Code Orienté Performance** : L'accent est mis sur la minimisation des allocations mémoire (`sync.Pool`), l'optimisation des calculs matriciels (matrices symétriques) et l'utilisation efficace du parallélisme.
*   **Immuabilité** : Des précautions sont prises pour garantir l'immuabilité des données partagées (comme la table de consultation pré-calculée) en retournant des copies, évitant ainsi des effets de bord complexes.