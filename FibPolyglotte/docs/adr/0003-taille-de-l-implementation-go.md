---
status: superseded by ADR-0004
---

# Go dépasse la taille visée d'environ 200 lignes

L'implémentation Go compte 255 lignes hors tests : 94 pour la bibliothèque et 161 pour la CLI. Le PRD (A3) vise environ 200 lignes, et les trois autres langages en comptent de 151 à 205.

L'écart tient surtout au langage :

- `math/big` n'a pas d'opérateurs, si bien qu'on écrit `new(big.Int).Mul(a, b)` là où les autres écrivent `a * b`;
- chaque erreur se propage explicitement.

On accepte l'écart plutôt que de compacter le code au détriment de la lecture, qui est l'objectif O3.

## Consequences

La revue de code (`/code-review`) peut encore retirer du code superflu; le seuil ne justifie pas, à lui seul, une réécriture. En Rust, un `rustfmt.toml` à 120 colonnes garde la CLI près de la cible; il ne change que la mise en forme.

## Amendement du 2026-10-11 : après la refactorisation

La refactorisation qui a suivi l'audit du 2026-10-11 aligne les quatre CLI sur une même forme : une constante de comparaison, un `compute` qui rend les termes et les durées, une déclaration par ligne et les mêmes commentaires. Go passe à 270 lignes (bibliothèque 97, CLI 173) et Rust à 229 (bibliothèque 101, CLI 128). Chez Go, l'écart vient surtout de la CLI : le bloc d'import, la struct `options`, `bufio` et la propagation des erreurs. Rust dépasse lui aussi la cible. On l'accepte pour la même raison qu'en Go : un étudiant qui compare les quatre langages doit les lire côte à côte (O3), et cela compte plus que le nombre de lignes.

## Amendement du 2026-10-11 : remplacé par l'ADR-0004

L'ADR-0004 fait du temps de réponse un objectif et remplace A3 et O3 : la cible d'environ 200 lignes ne s'applique plus, à aucun langage. Cet ADR ne reste que pour l'historique; la taille actuelle est au PRD (A3).
