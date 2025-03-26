# Calcul de Fibonacci(n) via la Formule de Binet en Go

Ce programme calcule le n-ième nombre de Fibonacci en utilisant la formule mathématique de Binet, implémentée en Go avec le package `math/big` pour gérer la précision arbitraire requise pour les grands nombres.

**Auteur Original du Code:** Adapté par l'IA Gemini 2.5 Pro Experimental (Mars 2025)
**Version:** 1.0 (Binet)

**ATTENTION:** Cette implémentation est principalement à **but démonstratif et éducatif**. Elle illustre l'utilisation de la formule de Binet avec des nombres à virgule flottante de haute précision. Pour des calculs de Fibonacci performants et l'obtention de résultats entiers exacts, en particulier pour de grandes valeurs de `n`, l'algorithme **Fast Doubling (exponentiation matricielle)** est **fortement recommandé** car il est significativement plus rapide et moins gourmand en mémoire.

## Table des Matières

1.  [Fonctionnalités](#fonctionnalités)
2.  [Analyse du Code](#analyse-du-code)
    *   [Vulgarisation : Comment ça marche ?](#vulgarisation--comment-ça-marche-)
    *   [Points Clés et Limitations](#points-clés-et-limitations)
    *   [Pourquoi Binet est-il moins adapté ici ?](#pourquoi-binet-est-il-moins-adapté-ici-)
3.  [Prérequis](#prérequis)
4.  [Installation](#installation)
5.  [Utilisation](#utilisation)
    *   [Configuration](#configuration)
    *   [Exécution](#exécution)
    *   [Profiling](#profiling)
6.  [Exemple de Sortie](#exemple-de-sortie)

## Fonctionnalités

*   Calcule Fibonacci(n) pour un `n` non-négatif donné.
*   Utilise la **formule de Binet** avec `math/big.Float` pour les calculs en virgule flottante à haute précision.
*   Estimation automatique de la précision nécessaire pour les calculs `big.Float`.
*   Configuration via une structure `Config` dans le code (N, Timeout, Précision d'affichage, Workers, Profiling).
*   Gestion d'un **timeout** global pour éviter les exécutions trop longues.
*   Possibilité d'activer le **profiling CPU et mémoire** via `pprof`.
*   Affichage formaté du résultat (notation scientifique, nombre de chiffres, premiers/derniers chiffres pour les très grands nombres).
*   Journalisation (logging) détaillée des étapes et des timings.

## Analyse du Code

### Vulgarisation : Comment ça marche ?

Imaginez que vous vouliez connaître la taille d'une plante (le nombre de Fibonacci) après un certain nombre de jours (`n`). La méthode "normale" (itérative ou récursive simple) serait de regarder sa taille la veille et l'avant-veille et de les additionner, jour après jour. C'est simple mais peut être long si le nombre de jours est grand.

La **formule de Binet** est comme une formule magique qui prétend donner directement la taille de la plante au jour `n`, sans avoir besoin de calculer toutes les tailles précédentes. Elle utilise deux nombres spéciaux :
1.  Le **nombre d'or** (souvent appelé `phi`, environ 1.618...).
2.  Un autre nombre lié (`psi`, environ -0.618...).

La formule ressemble à peu près à ça :
`Fibonacci(n) ≈ (phi^n) / √5`

(En réalité, la formule exacte est `Fibonacci(n) = (phi^n - psi^n) / √5`).

Le problème est que `phi` et `psi` ne sont pas des nombres "ronds" (ils ont une infinité de chiffres après la virgule). Pour calculer `phi^n` (phi multiplié par lui-même `n` fois), surtout si `n` est grand, on a besoin d'une calculatrice *extrêmement* précise.

Ce code Go utilise donc :
1.  Le package `math/big` : C'est une "super calculatrice" capable de manier des nombres avec des centaines, voire des milliers de chiffres après la virgule (`big.Float`).
2.  **Estimation de Précision** : Le code calcule d'abord combien de chiffres après la virgule il lui faut pour que le calcul reste assez juste, basé sur la valeur de `n`.
3.  **Calculs avec `big.Float`** : Il calcule `√5`, `phi`, `psi`, puis `phi^n` et `psi^n` avec cette haute précision.
4.  **Application de la Formule** : Il applique la formule `(phi^n - psi^n) / √5`.
5.  **Arrondi Malin** : Comme même la super calculatrice a ses limites, le résultat obtenu n'est pas *exactement* un nombre entier, mais il en est très très proche. Le code ajoute `0.5` au résultat final et prend ensuite la partie entière, ce qui revient à arrondir au nombre entier le plus proche.

En résumé, le code utilise une formule mathématique directe (Binet) mais doit employer des outils très sophistiqués (`big.Float`) pour gérer la précision extrême nécessaire, ce qui le rend complexe et souvent plus lent que la méthode "jour après jour" optimisée (comme Fast Doubling).

### Points Clés et Limitations

*   **`math/big.Float`** : Le cœur de l'implémentation. Permet de réaliser des calculs flottants avec une précision arbitraire, essentielle pour Binet avec de grands `n`.
*   **Calcul de Précision (`prec`)** : Une étape cruciale. Si la précision est insuffisante, le résultat final sera incorrect après l'arrondi. La formule `uint(math.Ceil(float64(n)*0.69424191) + 128)` estime le nombre de *bits* de précision nécessaires (basé sur `n * log2(phi)` plus une marge).
*   **Exponentiation par Carré (`pow`)** : La fonction `pow` calcule `base^exp` efficacement pour les `big.Float`, même si cela reste coûteux en termes de calcul.
*   **Arrondi Final** : L'ajout de `0.5` avant la conversion en `big.Int` (`resultFloat.Add(resultFloat, half); resultInt, _ := resultFloat.Int(nil)`) est la méthode standard pour arrondir un `big.Float` positif à l'entier le plus proche.
*   **Performance** : C'est le **point faible majeur**.
    *   Les opérations sur `big.Float` sont intrinsèquement beaucoup plus lentes que les opérations sur les entiers natifs ou même `big.Int`.
    *   La complexité temporelle dépend fortement de la précision requise, qui augmente linéairement avec `n`. L'exponentiation ajoute un autre facteur logarithmique. Grossièrement, on peut s'attendre à une complexité autour de `O(n * log(n) * M(p))` où `p` est la précision (qui dépend de `n`) et `M(p)` est le coût de la multiplication de nombres de `p` bits. C'est **beaucoup plus lent** que `O(log n)` pour Fast Doubling.
*   **Mémoire** : La consommation mémoire est **très élevée** car les `big.Float` doivent stocker un grand nombre de bits pour maintenir la précision. Elle croît avec `n`.
*   **Complexité du Code** : Gérer la précision, les conversions et les opérations `big.Float` rend le code plus complexe à écrire et à maintenir qu'une solution basée sur des entiers.
*   **Pas d'Annulation Interne** : La fonction `FibBinet` elle-même n'écoute pas le `context.Done()`. Si le timeout global est atteint, le programme s'arrêtera, mais le calcul Binet ne sera pas interrompu proprement en cours de route (par exemple, au milieu de la fonction `pow`).
*   **`GOMAXPROCS` Peu Pertinent Ici** : Définir `runtime.GOMAXPROCS` a peu d'impact sur la performance d'un *unique* calcul Binet, car celui-ci est essentiellement séquentiel et gourmand en CPU sur un seul thread logique (même si `big` peut utiliser plusieurs cœurs pour certaines opérations internes complexes, le goulot d'étranglement reste le calcul lui-même).

### Pourquoi Binet est-il moins adapté ici ?

La suite de Fibonacci produit des nombres **entiers**. Les méthodes comme l'itération simple, la récursion avec mémoïsation, ou l'exponentiation matricielle (Fast Doubling) fonctionnent entièrement avec des **entiers** (en utilisant `big.Int` pour les grands nombres). Elles garantissent un résultat exact sans problème d'arrondi et sont beaucoup plus efficaces.

La formule de Binet, bien qu'élégante mathématiquement, introduit des **nombres irrationnels** (√5, phi, psi). Pour obtenir le résultat entier exact via cette formule, il faut effectuer des calculs intermédiaires avec une précision flottante *extrêmement* élevée, puis arrondir. Ce détour par les flottants de haute précision est coûteux en temps de calcul et en mémoire, et introduit une complexité liée à la gestion de cette précision.

En pratique, pour calculer F(n) où F(n) est un entier :
*   **Fast Doubling (Exponentiation Matricielle)** : `O(log n)` opérations sur `big.Int`. Très rapide et efficace en mémoire (relativement à la taille du résultat). **C'est la méthode de choix.**
*   **Itératif simple** : `O(n)` opérations sur `big.Int`. Plus lent que Fast Doubling pour grand `n`, mais simple et efficace en mémoire.
*   **Formule de Binet (cette implémentation)** : Complexité supérieure (dépendant de `n` et de la précision), très gourmand en mémoire, nécessite des arrondis. Principalement utile pour comprendre la formule ou si l'on travaillait *naturellement* avec des flottants.

## Prérequis

*   **Go** : Une version récente de Go installée (par exemple, 1.18 ou supérieure).

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

2.  **Compiler le programme** :
    Ouvrez un terminal dans le répertoire contenant `main.go` et exécutez :

    ```bash
    go build -o fibonacci_binet main.go
    ```

    Cela créera un exécutable nommé `fibonacci_binet` (ou `fibonacci_binet.exe` sous Windows).

## Utilisation

### Configuration

Les paramètres principaux sont définis directement dans la fonction `DefaultConfig()` au début du fichier `main.go`. Modifiez ces valeurs avant de compiler si nécessaire :

*   `N`: L'index du nombre de Fibonacci à calculer. **Attention :** commencez avec des valeurs petites (ex: 1000, 10000). Des valeurs comme 1 000 000 prendront déjà beaucoup de temps et de mémoire. 10 000 000 est très long.
*   `Timeout`: Durée maximale allouée à l'exécution (ex: `"5m"` pour 5 minutes).
*   `Precision`: Nombre de chiffres après la virgule pour l'**affichage** en notation scientifique (n'affecte pas la précision du calcul).
*   `Workers`: Nombre de threads CPU que Go peut utiliser (`runtime.GOMAXPROCS`). Peu d'impact sur ce calcul spécifique.
*   `EnableProfiling`: Mettre à `true` pour activer le profiling CPU et mémoire.

**Recompilez le programme (`go build ...`) après toute modification de la configuration.**

### Exécution

Exécutez simplement le binaire compilé depuis votre terminal :

```bash
./fibonacci_binet

Si EnableProfiling est mis à true dans la configuration (et que le programme est recompilé) :
Deux fichiers seront créés à la fin de l'exécution (ou en cas de timeout/erreur, une tentative sera faite) :
cpu_binet.pprof: Profil d'utilisation CPU.
mem_binet.pprof: Profil d'utilisation mémoire (tas).
Vous pouvez analyser ces fichiers avec l'outil go tool pprof :
# Pour analyser le profil CPU (ouvre une interface interactive)
go tool pprof cpu_binet.pprof

# Pour analyser le profil mémoire
go tool pprof mem_binet.pprof

# Pour visualiser le graphe d'appel en PDF (nécessite graphviz)
go tool pprof -pdf cpu_binet.pprof > cpu_graph.pdf
go tool pprof -pdf mem_binet.pprof > mem_graph.pdf

# Pour voir le profil via une interface web
go tool pprof -http=:8080 cpu_binet.pprof
go tool pprof -http=:8081 mem_binet.pprof


Exemple de Sortie
Pour une petite valeur de N, par exemple N=1000 (après avoir modifié DefaultConfig et recompilé) :
INFO: Configuration (Binet): N=1000, Timeout=5m0s, Workers=8, Profiling=false, Précision Affichage=10
INFO: ATTENTION: La méthode de Binet est utilisée. Elle est lente et gourmande en mémoire pour N > ~10000.
INFO: Démarrage du calcul Binet de Fibonacci(1000)... (Timeout global: 5m0s)
INFO: En attente du résultat Binet ou du timeout...
INFO (Binet): Démarrage calcul pour n=1000
INFO (Binet): Utilisation de précision big.Float = 823 bits
INFO (Binet): Calcul de phi^1000...
INFO (Binet): Calcul de psi^1000...
INFO (Binet): Calcul flottant terminé en 15.5ms
INFO (Binet): Conversion Float -> Int Accuracy: Below
INFO (Binet): Calcul total (avec arrondi) terminé en 16.2ms
INFO: Calcul Binet terminé avec succès. Durée calcul pur: 16ms

=== Résultats Binet pour Fibonacci(1000) ===
Temps total d'exécution                     : 17ms
Temps de calcul pur (FibBinet)              : 16ms

Résultat F(1000) :
  Notation scientifique (~10 chiffres)      : 4.3466557687e+208
  Nombre total de chiffres décimaux         : 209
  Valeur exacte                             : 43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875
INFO: Programme (Binet) terminé.