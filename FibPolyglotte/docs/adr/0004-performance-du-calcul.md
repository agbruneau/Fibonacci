# La performance du calcul devient un objectif

Le 2026-10-11, le propriétaire demande de minimiser le temps de réponse de la CLI dans les quatre langages. La décision renverse O3 et la phrase du PRD § 1 « La performance n'est pas un objectif ». Désormais, le temps de réponse prime : le temps total du processus, et le temps de calcul que rapporte `--time`. Le code reste lisible : chaque optimisation porte un commentaire d'une ligne qui dit pourquoi.

Un banc d'essai (`npm run bench`, sept scénarios depuis la borne à 1 000 000, huit depuis le 2026-10-14 avec un terme impair, 999 999; médianes) montre où va le temps. Le démarrage du processus domine le terme seul, la conversion binaire → décimal domine `--list`, et le calcul ne domine que `iterative` et `all`. D'où les décisions :

- **Commandes de lancement** :
  - TypeScript est compilé en JavaScript par tsc dans `.build/ts/`, car le retrait des types d'un `.ts` se payait à chaque lancement (mesure au PRD § 5).
  - Python est lancé avec `-I -S`, donc sans `site`.
  - Rust est lancé par son binaire, et non par `cargo run`.
- **La liste en décimal** : `fibList` rend des chaînes décimales calculées directement en base 10, par un additionneur écrit à la main en Go, Rust et TypeScript, et par `decimal.Decimal` en Python. La liste ne passe plus par le grand entier du langage.
- **Les algorithmes** gardent chacun leur identité, pour que `--algo all` compare toujours trois algorithmes :
  - `iterative` additionne sur place, sans allouer, en Go et en Rust;
  - `matrix` élève au carré, de gauche à droite, une matrice symétrique à trois coefficients, et multiplie par Q en additions;
  - `fast` fait deux carrés par bit (formules de GMP), ou un produit et un carré (nombres de Lucas, en TypeScript).

  La dernière étape de `matrix` et de `fast` ne calcule que F(n).
- **Démarrage** :
  - aucun import ni initialisation coûteuse : ni `re` en Python, ni regex compilée au chargement en Go;
  - en Python, le terme s'écrit par tranches, pour éviter l'import de `_pylong` au-delà de 9 000 chiffres (voir l'amendement pour les grands termes).
- **Aucune dépendance nouvelle** : GMP reste exclu (PRD § 8).

## Considered Options

- **Garder O3** : refusé par le propriétaire.
- **GMP (gmpy2, rug)** : à N = 100 000, `fast` calcule déjà en moins d'une milliseconde. Le temps vient du démarrage et de la conversion, que GMP ne supprime pas, et il faudrait une dépendance native de plus.
- **`NODE_COMPILE_CACHE`** : il supprime aussi le coût du `.ts`, mais il exige une variable d'environnement à chaque lancement.

## Consequences

- A3 et l'ADR-0003 ne s'appliquent plus. À la décision : TypeScript 232, Python 228, Rust 359, Go 376 lignes; la taille actuelle est au PRD (A3).
- F6 : pour `--list`, la durée couvre la production du texte décimal, qui est désormais le calcul lui-même.
- BIB-10 à BIB-12 comparent `fibList` aux chaînes de l'oracle. Des tests de garde `PERF-<ID>-<n>` protègent les optimisations, au besoin par des fonctions privées. Ils sont hors seams (catalogue § 6).
- Temps total au moment de la décision, en millisecondes, médianes sur le même banc. « Avant » est mesuré le 2026-10-11 avec les anciennes commandes, mais Rust par son binaire, sans `cargo run`. L'état final est au [README](../../README.md), § Résultats.

| Scénario | Go | Rust | TypeScript | Python |
|---|---|---|---|---|
| petit (`10`) | 6,1 → 6,8 | 5,8 → 5,7 | 63,8 → 40,7 | 29,3 → 19,9 |
| terme `100000` (`fast`) | 6,7 → 6,7 | 7,0 → 5,8 | 69,2 → 38,1 | 33,9 → 22,5 |
| `--algo matrix 100000` | 7,9 → 6,6 | 8,5 → 5,6 | 67,6 → 38,3 | 40,9 → 21,7 |
| `--algo iterative 100000` | 88,1 → 22,9 | 42,2 → 31,6 | 109,6 → 82,6 | 88,2 → 75,8 |
| `--algo all 100000` | 73,5 → 23,4 | 45,4 → 32,3 | 110,1 → 84,0 | 95,3 → 76,8 |
| `--list 10000` | 78,6 → 23,4 | 107,2 → 24,7 | 206,3 → 70,2 | 229,2 → 66,9 |

## Amendement du 2026-10-11 : borne d'un terme à 1 000 000

Le propriétaire relève la borne d'un terme de 100 000 à 1 000 000 (F8). Celle d'une liste reste à 10 000. À 1 000 000, F(n) compte 208 988 chiffres, et sa conversion en décimal devient la plus grosse part du temps.

- **Go, Rust, TypeScript** : seule la borne change. `math/big`, `num-bigint` 0.5.1 et V8 convertissent déjà par diviser pour régner. Aucune conversion écrite à la main ne les bat de plus de 3 % de façon répétable. Seul le parallélisme gagnerait vraiment, et le PRD § 8 l'exclut.
- **Python** : l'écriture par tranches est quadratique et prenait 240 ms à 1 000 000. Deux seuils, mesurés sous Python 3.14 (3.12 et 3.13 non vérifiés), la remplacent :
  - dès `DECIMAL_FROM` (170 000), `fast` et `matrix` calculent en `decimal.Decimal` exact, dont `str()` est linéaire. La CLI passe pour cela par une aide privée de la bibliothèque, et la seam `fib()` rend toujours un `int`;
  - au-delà de `STR_FROM_BITS` (200 000 bits), un `int` passe par `str()`, sous-quadratique par `_pylong`. Ce seuil ne sert qu'avec `iterative` et `all`.
- **`--time`** : sur le chemin `Decimal`, le calcul de `fast` est un peu plus lent qu'en `int` (environ 3 ms contre 1 ms à 170 000), mais le temps total baisse. C'est le temps de réponse qui prime.
- **GMP** reste exclu. À 1 000 000, le temps de réponse va de 22 à 60 ms selon le langage, dont environ 5 à 36 ms de démarrage ([README](../../README.md), § Résultats). GMP gagnerait quelques millisecondes de calcul et de conversion, au prix d'une dépendance native.

## Mesures finales du 2026-10-11

Après la borne à 1 000 000 et les trois revues de code (`npm run bench -- --runs 15`, Windows 11) : voir le [README](../../README.md), § Résultats.

## Amendement du 2026-10-13 : l'A/B apparié, mesure d'un travail de démarrage

Le banc seul ne tranche pas un gain de quelques pour cent. Pour l'horloge `process.hrtime` (#21), les médianes séparées se recouvraient : de 46,3 à 55,8 ms pour `main`, de 44,9 à 47,8 ms pour la branche. Seul un A/B apparié a tranché, de −1,8 à −2,6 ms (−4,0 à −5,7 %), sur 6 tours de 201 lancements. Il avait été refait à la main au moins quatre fois, chaque fois par un script jetable propre à un langage, sans les commandes du contrat. La série Python (D, puis la partie Python de C, puis E) devait refondre le texte décimal et le point d'entrée de `python/fib.py`. Réduite à des gardes (#33 à #35), elle n'a touché `python/fib.py` que par un commentaire.

Le propriétaire décide que l'A/B devient la mesure d'un travail de démarrage (imports, chargement, point d'entrée) :

- **Forme** : `npm run bench -- <langage> --contre REF`. Le banc extrait la révision REF sous `.build/revisions/<sha>`, puis construit les deux côtés avec les commandes de lancement d'aujourd'hui (`contracts/lancement.ts`). Par scénario, il joue des paires alternées (A puis B, B puis A), écarte le tour 0, et écrit A, B, B − A, le pourcentage et le nombre de paires où B est plus rapide. Le résumé est la médiane des différences par paire, non la différence des médianes : la dérive de la machine touche A et B d'un même tour. Pour un nombre pair de valeurs, toute médiane du banc (A, B et B − A) est la moyenne des deux éléments centraux, non l'élément haut, qui biaiserait chaque résumé vers le haut.
- **Quand** : un travail de démarrage se mesure par l'A/B, REF étant la révision d'avant le travail. Un travail de calcul garde le banc ordinaire, en médianes (`npm run bench -- <langage>`, avant et après).
- **Limite** : l'A/B ne vaut que si les commandes de lancement n'ont pas changé entre REF et l'arbre de travail. Elles n'ont pas changé depuis la décision du 2026-10-11. Un ticket qui change une commande de lancement ne se mesure pas ainsi.
- **Ce qu'il ne fait pas** : il ne compare que des temps, jamais des sorties, car REF n'est pas un attendu (ADR-0001). Comme le banc, il lève si une CLI sort avec un code non nul. Le contrat reste seul juge de la sortie.
- **Mesure** : en série, jamais en parallèle; une seule mesure après la fusion fait foi (PRD § 10).
  - **Une centaine de paires, en nombre pair** : `--runs` vaut 100 par défaut sous `--contre`, et 11 pour le banc ordinaire. L'A/B de #21 a tranché sur 6 tours de 201 lancements; 100 paires font 200 lancements par scénario, du même ordre. L'erreur de la médiane des différences décroît en 1/√n : 100 paires la divisent par 3 environ sur 11 paires, ce qui sépare un écart de quelques pour cent du bruit. L'essai coûtait 1 414 lancements par langage (sept scénarios, un tour de chauffe), et 1 616 depuis le huitième scénario, impair (2026-10-14).
  - **Pourquoi pair** : l'ordre alterne, A part le premier aux tours pairs et B aux impairs, et les tours gardés vont de 1 à `runs`. Avec 101 paires, B part le premier dans 51 d'entre elles et A dans 50 : la médiane des différences retombe dans le groupe majoritaire et garde l'effet d'ordre que l'alternance devait annuler. En simulation, avec un écart réel nul, un effet d'ordre de 0,5 ms et un bruit de 0,1 ms, B − A vaut +0,27 ms avec 101 paires, et 0,00 avec 100 ou 102. Le défaut est donc 100, le pair juste au-dessous de 101 (102 coûterait 28 lancements de plus par langage, pour 1 % d’erreur en moins). Un `--runs` explicite l’emporte, mais un nombre impair donné à la main sous `--contre` est arrondi au pair supérieur plutôt que refusé, pour que `--runs 1` serve encore à un essai de fumée (2 paires) : le banc l'écrit sur stderr, `bench: --runs 3 est impair sous --contre, arrondi à 4 paires : …`, et l'en-tête du tableau donne le nombre de paires jouées. Sans `--contre`, `--runs` reste libre.
  - **Nettoyage** : les révisions extraites s'accumulent sous `.build/revisions/`, ignoré par git, et se suppriment par `npm run bench -- --nettoyer`. Chaque révision porte un lien (une jonction sous Windows) vers le `node_modules` du dépôt, que `Remove-Item -Recurse` de Windows PowerShell 5.1 peut traverser pour vider sa cible : la commande détache le lien de chaque révision (`unlink` retire le lien, jamais sa cible), puis supprime le dossier, et ne touche rien d'autre sous `.build/`. `fs.rmSync` ne suit pas les liens sous Linux (vérifié); la jonction de Windows n'a pas été éprouvée sur une machine Windows, d'où le détachement préalable.
- **Erreurs** : le banc valide avant tout build et toute extraction. Un identifiant inconnu ou une option mal écrite est une erreur d'usage (code 2). Une révision introuvable, ou privée du dossier d'un langage demandé, écrit `bench: …` en une ligne, sans pile (code 1) : un langage absent de REF ne se découvrirait sinon qu'au milieu de l'A/B, après les langages qui le précèdent. Un échec de mesure dit son côté : `côté A (la révision REF, …)` ou `côté B (l'arbre de travail)`.
- **Gardes** : HAR-5 garde l'alternance, le résumé (la médiane d'un nombre pair comprise), la politique de mesure partagée avec le banc (`rounds`), l'indice de tour et les racines que l'A/B passe à `abRound`, les messages d'échec, la validation avant toute extraction, le défaut de `--runs`, l’arrondi d’un `--runs` impair, le refus d’un nombre impair de paires par `abPairs` (là où vit l’alternance) et le nettoyage (`docs/cas-de-test.md` § 6). Aucune garde `PERF` n'est créée, puisque aucune implémentation n'est touchée.
