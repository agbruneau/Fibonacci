# Analyse manuelle des arguments, même grammaire dans les quatre langages

Chaque implémentation lit ses arguments à la main, selon la grammaire du PRD § 4, plutôt qu'avec `flag` (Go), `parseArgs` (Node), `argparse` (Python) ou `clap` (Rust). Les sondes du 2026-10-10 montrent que ces bibliothèques divergent sur des points du contrat :

- `flag` s'arrête au premier argument positionnel et écrit l'usage sur stderr à chaque erreur;
- `argparse` prend `-1` pour un positionnel, ajoute une ligne d'usage à ses erreurs, peut les colorer selon l'environnement et écrit `\r\n` sous Windows;
- `parseArgs` refuse `-1` comme option inconnue.

Plier chacune au contrat (une seule ligne `fib: `, aide sur stdout) coûterait plus de code qu'une boucle d'une vingtaine de lignes, identique par construction dans les quatre langages.

## Considered Options

- **La bibliothèque standard de chaque langage, enveloppée pour respecter le contrat** : idiomatique, mais quatre enveloppes différentes à garder alignées.
- **`clap` pour Rust** : une dépendance de plus, que le PRD § 5 exclut.

## Consequences

Comme aucune bibliothèque ne garantit la grammaire, le test de contrat doit la couvrir en entier : ordre libre, `--algo=X`, répétition, `-h`, arguments inconnus.

## Amendement du 2026-10-11 : l'exemple de `parseArgs`

Refuser `-1` n'est pas un écart : le contrat le refuse aussi (règle 4). L'écart réel, vérifié ce jour, est que `parseArgs` traite `--` comme la fin des options : `-- 5` y donne l'indice 5, alors que le contrat exige une erreur (cas CLI-U03).
