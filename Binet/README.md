# Calculateur Fibonacci(n) en Go (Méthode de Binet)

Ce programme Go calcule le n-ième nombre de Fibonacci, F(n), en utilisant la **formule de Binet** implémentée avec la bibliothèque `math/big.Float` pour la gestion de l'arithmétique en virgule flottante à précision arbitraire.

**Adapté par :** IA Gemini 2.5 Pro Experimental (03-2025)

**Version :** 1.0 (Binet)

---

## ATTENTION : Performance et Utilisation

🛑 **Cette implémentation est principalement à but démonstratif.** 🛑

La méthode de Binet, bien qu'élégante mathématiquement, présente des inconvénients majeurs pour le calcul *exact* de nombres de Fibonacci entiers, surtout pour de grandes valeurs de `n` :

1.  **Performance Médiocre :** Elle est **significativement plus lente** que les algorithmes basés sur l'arithmétique entière, comme l'exponentiation matricielle (Fast Doubling). Les calculs avec `big.Float` à très haute précision sont très coûteux en temps CPU.
2.  **Consommation Mémoire Élevée :** La précision requise pour `big.Float` augmente linéairement avec `n`. Pour de grands `n`, cela entraîne une consommation mémoire très importante.
3.  **Complexité de la Précision :** La gestion de la précision nécessaire pour garantir un résultat entier exact après arrondi est complexe et une source potentielle d'erreurs subtiles.

**Pour des calculs performants et exacts de Fibonacci(n), en particulier pour `n` élevé, l'utilisation d'un algorithme comme [Fast Doubling (exponentiation matricielle)](https://github.com/example/fibonacci-fast-doubling-go) (lien hypothétique vers une version précédente/alternative) est fortement recommandée.**

Utilisez cette version Binet principalement pour comprendre comment la formule peut être implémentée avec `big.Float` ou pour des `n` très petits où la performance n'est pas critique.

---

## Fonctionnalités

*   Calcule Fibonacci(n) en utilisant la formule mathématique de Binet.
*   Utilise `math/big.Float` pour les calculs nécessitant une haute précision.
*   Inclut un timeout global configurable pour l'exécution.
*   Support optionnel pour le profiling CPU et mémoire via `pprof`.

---

## Prérequis

*   Compilateur Go (testé avec Go 1.18+, mais devrait fonctionner avec des versions antérieures supportant `big.Float`).

---

## Utilisation

1.  **Compilation :**
    ```bash
    go build -o fibonacci_binet main.go
    ```

2.  **Configuration :**
    *   Modifiez la fonction `DefaultConfig()` dans `main.go` pour changer les paramètres :
        *   `N`: L'indice `n` pour lequel calculer Fibonacci(n). **Attention : gardez une valeur faible (ex: < 10000) pour éviter des temps d'exécution et une consommation mémoire excessifs.**
        *   `Timeout`: La durée maximale d'exécution du programme (ex: `"5m"`).
        *   `Precision`: Le nombre de chiffres significatifs pour l'affichage scientifique *final* du résultat.
        *   `Workers`: Nombre de threads CPU (moins pertinent ici car un seul calcul Binet est peu parallélisable).
        *   `EnableProfiling`: Mettre à `true` pour activer le profiling `pprof`.

3.  **Exécution :**
    ```bash
    ./fibonacci_binet
    ```
    Le programme affichera les logs de configuration, les informations de calcul Binet, et le résultat final (si le timeout n'est pas atteint).

---

## Profiling (Optionnel)

Si `EnableProfiling` est mis à `true` dans la configuration :

1.  Le programme générera les fichiers suivants à la fin de l'exécution (ou lors d'un timeout/erreur) :
    *   `cpu_binet.pprof`: Profil d'utilisation CPU.
    *   `mem_binet.pprof`: Profil d'utilisation de la mémoire (heap).
2.  Vous pouvez analyser ces fichiers avec l'outil `go tool pprof` :
    *   **CPU :** `go tool pprof cpu_binet.pprof`
    *   **Mémoire :** `go tool pprof mem_binet.pprof`
    *   Dans l'interface `pprof`, utilisez des commandes comme `top`, `web` (nécessite Graphviz), `list <nom_fonction>` pour explorer les données de performance.

---

## Limitations

*   **Performance :** Très lent pour `n` élevé (exponentiation de `big.Float` à haute précision).
*   **Mémoire :** Très gourmand en mémoire pour `n` élevé.
*   **Gestion du Timeout :** Le timeout est global. La fonction `FibBinet` elle-même ne vérifie pas le contexte et ne peut pas être interrompue "proprement" en cours de calcul.

---

