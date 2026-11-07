# Fib

Une même CLI de calcul de la suite de Fibonacci, écrite quatre fois, en Go, Rust, TypeScript et Python, sous un contrat commun vérifié à l'octet près.

Fib est un terrain d'essai académique. Il sert à explorer, avec Claude Code, les [skills d'ingénierie de Matt Pocock](https://github.com/mattpocock/skills), du PRD jusqu'à la revue de code. Le calcul de Fibonacci n'est qu'un prétexte. Depuis l'[ADR-0004](docs/adr/0004-performance-du-calcul.md), le temps de réponse prime sur la brièveté du code, et chaque optimisation porte un commentaire d'une ligne qui dit pourquoi. Fib reprend un sous-ensemble de [FibCalc](https://github.com/agbruneau/Fibonacci).

## État

| Étape | Date | État |
|---|---|---|
| PRD, glossaire, décisions (ADR-0001 à 0003) | 2026-10-10 | fait |
| Spec ([agbruneau/Fib#1](https://github.com/agbruneau/Fib/issues/1)) et tickets | 2026-10-10 | fait |
| Catalogue de cas de test | 2026-10-10 | fait |
| Contrat commun et implémentation TypeScript ([agbruneau/Fib#2](https://github.com/agbruneau/Fib/issues/2)) | 2026-10-10 | fait |
| Go ([agbruneau/Fib#3](https://github.com/agbruneau/Fib/issues/3)), Rust ([agbruneau/Fib#4](https://github.com/agbruneau/Fib/issues/4)), Python ([agbruneau/Fib#5](https://github.com/agbruneau/Fib/issues/5)) | 2026-10-10 | fait |
| Audit en 8 dimensions, puis contrat durci | 2026-10-11 | fait |
| Refactorisation vers une forme commune aux quatre CLI, puis `/code-review` | 2026-10-11 | fait |
| Optimisation du temps de réponse ([ADR-0004](docs/adr/0004-performance-du-calcul.md)) et banc d'essai, puis `/code-review` | 2026-10-11 | fait |
| Borne d'un terme portée de 100 000 à 1 000 000, puis `/code-review` | 2026-10-11 | fait |
| Suites de l'épreuve des cas de test : huit cas CLI, gardes de performance, décisions de [agbruneau/Fib#11](https://github.com/agbruneau/Fib/issues/11) (tickets #6 à #10, #12 à #14), puis `/code-review` | 2026-10-12 | fait |
| Étape 9 du PRD § 9, `diagnosing-bugs` : écart du stderr fermé en Python ([agbruneau/Fib#16](https://github.com/agbruneau/Fib/issues/16)), puis `/code-review` | 2026-10-12 | fait |
| Étape 10 du PRD § 9, `/improve-codebase-architecture` et `codebase-design` : rapport, puis la CLI TypeScript appelable dans le processus ([agbruneau/Fib#20](https://github.com/agbruneau/Fib/issues/20)) et `/code-review` | 2026-10-13 | fait |
| Étape 11 du PRD § 9 : `/handoff`, document de reprise hors dépôt | 2026-10-13 | fait |
| Horloge `process.hrtime` en TypeScript ([agbruneau/Fib#21](https://github.com/agbruneau/Fib/issues/21)) et garde du préchargement de `decimal` en Python ([agbruneau/Fib#25](https://github.com/agbruneau/Fib/issues/25)), puis `/code-review` | 2026-10-13 | fait |
| Pistes A, C, D et E de l'étape 10 : conception en workflow, une série de questions, puis tickets #27 à #36 livrés en trois vagues (lancement commun et contrôles du harnais, A/B apparié, boucle différentielle, discordance et algorithme exécuté gardés dans les quatre langages, gardes Python), puis `/code-review` | 2026-10-13 | fait |
| Revue finale de cohérence en quatre angles, gardes d'identité de l'aiguillage ([agbruneau/Fib#39](https://github.com/agbruneau/Fib/issues/39) à [#42](https://github.com/agbruneau/Fib/issues/42)), PERF-TS-11 et HAR-7 ([#43](https://github.com/agbruneau/Fib/issues/43)), puis `/code-review` | 2026-10-13 | fait |
| Audits par langage (`go/`, `rust/`, `ts/` et `python/audit.md`) : constats vérifiés en workflow, gardes ajoutées ou étendues dans les quatre langages, témoin Windows de PERF-PY-23 corrigé, puis `/code-review` | 2026-10-14 | fait |

Les quatre implémentations passent `npm run check` sous Windows 11, avec Python 3.14 (2026-10-12), et sous Linux (2026-10-14), avec Go 1.26, Rust 1.97, Node 24.20 et Python 3.13. Sous Windows, `check:py` rougissait depuis #35 (2026-10-13) : un témoin de PERF-PY-23 attendait `\n` là où CPython écrit `\r\n`. Corrigé le 2026-10-14, ce point n'est pas encore revérifié sous Windows. La suite Python et le contrat Python passent aussi sous Python 3.12 (3.12.3, Linux, 2026-10-13).

| Implémentation | Grand entier |
|---|---|
| TypeScript | `bigint` natif |
| Python | `int` natif |
| Rust | `num-bigint` 0.5 |
| Go | `math/big` |

## Résultats

Temps total du processus, en millisecondes, médianes de 15 exécutions (`npm run bench -- --runs 15`), avant toute optimisation → final :

| Scénario | Go | Rust | TypeScript | Python |
|---|---|---|---|---|
| petit (`10`) | 6,1 → 7,2 | 5,8 → 5,5 | 63,8 → 38,1 | 29,3 → 18,7 |
| terme `100000` (`fast`) | 6,7 → 7,3 | 7,0 → 5,9 | 69,2 → 39,6 | 33,9 → 21,3 |
| terme `1000000` (`fast`) | refusé → 21,8 | refusé → 29,1 | refusé → 59,7 | refusé → 38,1 |
| `--algo matrix 100000` | 7,9 → 7,3 | 8,5 → 6,2 | 67,6 → 40,1 | 40,9 → 21,9 |
| `--algo iterative 100000` | 88,1 → 23,0 | 42,2 → 31,7 | 109,6 → 82,9 | 88,2 → 77,2 |
| `--algo all 100000` | 73,5 → 23,8 | 45,4 → 32,5 | 110,1 → 83,1 | 95,3 → 77,0 |
| `--list 10000` | 78,6 → 23,2 | 107,2 → 25,2 | 206,3 → 71,8 | 229,2 → 65,3 |

Le scénario impair, terme `999999`, s'ajoute au banc le 2026-10-14 : il n'a pas encore de mesure sous Windows.

- Go et Rust sont au plancher du démarrage, environ 5 à 7 ms, sur les scénarios petit, terme `100000` et `matrix` : les différences y sont du bruit. Le plancher est d'environ 36 ms pour Node et 15 ms pour Python.
- Le 2026-10-13, l'horloge `process.hrtime` ([agbruneau/Fib#21](https://github.com/agbruneau/Fib/issues/21)) a retiré de 1,8 à 2,6 ms au démarrage de TypeScript : le premier accès à `performance` chargeait jusqu'à 11 modules internes. C'est un A/B apparié sous Linux et Node 22.22, postérieur au tableau.
- La colonne « avant » de Rust lance déjà le binaire. L'ancienne commande, `cargo run`, ajoutait environ 50 ms par lancement.
- « Refusé » : l'ancienne borne d'un terme était 100 000. F(1 000 000) compte 208 988 chiffres; avec la seule borne relevée, Python mettait 288 ms.

**Qualité.**

- L'audit, en 8 dimensions (les quatre langages, contrat, cohérence, test différentiel, documentation), a rendu 233 constats, dont 36 réfutés par un vérificateur sceptique. Il n'a trouvé aucune erreur de calcul et un seul écart au contrat : Python laissait `PYTHONIOENCODING` changer l'encodage de sa sortie. L'écart est corrigé et gardé par PERF-PY-20 : CLI-V03 ne peut pas le voir, car `python -I` ignore la variable. Un test différentiel de 121 vecteurs hors catalogue n'a trouvé aucun écart entre les quatre implémentations. Le 2026-10-12, une boucle différentielle d'environ 170 000 entrées en a trouvé un : avec stderr fermé, Python sortait en code 1, stdout vide. Il est corrigé et gardé par CLI-V04 et PERF-PY-23. Elle a aussi relevé des comportements que le contrat ne tranchait pas, comme un stdout qui refuse l'écriture : ils sont désormais hors contrat (PRD § 4, [agbruneau/Fib#18](https://github.com/agbruneau/Fib/issues/18)).
- Chaque langage passe les 99 cas CLI du contrat (CLI-V04 se saute sous Windows), ORA-01 à ORA-06 sur l'oracle et les contrôles du harnais HAR-1 à HAR-7 (HAR-7 en un test par commande de lancement) et la sonde du fd 2 de CLI-V04 (116 tests par langage, 413 en tout), BIB-01 à BIB-14 sur sa bibliothèque (BIB-14 est sans objet en Rust), et des tests de garde de ses optimisations : 13 en Go, 14 en Rust, 12 en TypeScript, 31 en Python, dont PERF-PY-20, PERF-PY-23 et PERF-PY-29, qui gardent des correctifs (F10 et CLI-V04). D'autres gardes tiennent des exigences que le contrat ne sait pas provoquer. La discordance (F3) est gardée dans les quatre langages (PERF-TS-7, PERF-GO-9, PERF-RS-11, PERF-PY-26), ainsi que l'algorithme réellement exécuté (F2, F6 : PERF-TS-8 et 12, PERF-GO-10, PERF-RS-12, PERF-PY-27) et l'aiguillage de chaque nom vers sa propre fonction (PERF-TS-10, PERF-GO-11, PERF-RS-13, PERF-PY-30), dont, sauf en Rust, un corps d'algorithme qui n'en appelle aucun autre. La durée réellement mesurée l'est aussi : par une horloge factice en TypeScript (PERF-TS-8, et PERF-TS-9 pour l'unité réelle), par un calcul espion qui dort en Go, Rust et Python (PERF-GO-12, PERF-RS-14, PERF-PY-31).
- Une revue `/code-review` a suivi chaque chantier du tableau d'état depuis le 2026-10-11, sauf l'étape 11; les tickets #2 à #5 du 2026-10-10 ont été fermés sans elle. Ces revues ont trouvé, entre autres, une violation du PRD § 7 en Rust (lignes vides de l'oracle) et un `matrix` TypeScript qui refaisait l'ancien `fast`. Les constats sont corrigés.

**Coût.**

| Lignes de code (`wc -l`, hors fichiers de test) | Go | Rust | TypeScript | Python |
|---|---|---|---|---|
| avant → final | 255 → 394 | 205 → 568 | 151 → 270 | 158 → 270 |

- Rust compte aussi ses tests unitaires : 60 lignes dans `lib.rs` (dont PERF-RS-13) et 169 dans `main.rs` (PERF-RS-8, 11, 12 et 14). Sans eux, Rust compte 339 lignes. La cible d'environ 200 lignes (PRD, A3) est remplacée par l'ADR-0004.
- `iterative` reste en O(n²) opérations sur les mots : à 1 000 000, il prend environ 1,6 s en Go et 5 s en Python, ce que F8 accepte.

Le détail des décisions et des mesures est dans l'[ADR-0004](docs/adr/0004-performance-du-calcul.md).

## Utilisation

```
usage: fib [--algo iterative|matrix|fast|all] [--time] N
       fib --list [--time] N
       fib --help
```

```console
$ fib 10
55
$ fib --list 5
0
1
1
2
3
5
$ fib --time --algo all 50
12586269025
fast 0.012 ms
iterative 0.004 ms
matrix 0.015 ms
```

Les durées affichées sont illustratives. Ici, `fib` désigne l'une des quatre commandes de la section suivante.

| Algorithme | Principe | Coût |
|---|---|---|
| `iterative` | la définition, indice par indice : deux termes consécutifs, additionnés sur place en Go et en Rust | O(n) additions de grands entiers |
| `matrix` | [[1,1],[1,0]]ⁿ par carrés successifs, du poids fort au poids faible de n, sur les trois coefficients d'une matrice symétrique; un bit à 1 multiplie par [[1,1],[1,0]] en additions | trois carrés par bit de n, soit O(log n) multiplications |
| `fast` (par défaut) | doublement rapide : de F(k) à F(2k), puis F(2k+1) si le bit vaut 1, par les formules de GMP (nombres de Lucas en TypeScript) | deux carrés par bit (un produit et un carré en TypeScript), soit O(log n) multiplications |

`--algo all` compare les trois algorithmes et vérifie qu'ils concordent. `--list` additionne directement en décimal, sans conversion binaire → décimal. `--time` écrit la durée du calcul sur stderr.

- **Bornes** : N ≤ 1 000 000 pour un terme, N ≤ 10 000 pour une liste.
- **Codes de sortie** : 0 succès, 2 erreur d'usage, 3 discordance entre algorithmes. Hors contrat (PRD § 4 : flux en échec d'écriture, limites de ressources), le code est libre.

Le contrat complet est au § 4 du [PRD](PRD.md) : grammaire, formats, erreurs.

## Lancer une implémentation

Depuis la racine du dépôt :

| Langage | Commande |
|---|---|
| TypeScript | `npx tsc -p tsconfig.build.json`, puis `node .build/ts/cli.js 10` |
| Go | `go -C go build -o ../.build/fib.exe ./cmd/fib`, puis `.build/fib.exe 10` (sans `.exe` hors de Windows) |
| Rust | `cargo build -q --release --manifest-path rust/Cargo.toml`, puis `rust/target/release/fib.exe 10` (sans `.exe` hors de Windows) |
| Python | `python -I -S python/fib.py 10` |

Go, Rust et TypeScript se lancent compilés : la compilation ne se paie qu'une fois, et `go run` remplace en plus le code de sortie du programme. Python prend `-I -S` pour sauter l'import de `site` et les variables `PYTHON*`.

## Vérifier

```bash
npm install
```

```bash
npm run check:ts
```

```bash
npm run contract
```

```bash
npm run check
```

```bash
npm run bench -- --runs 15
npm run bench -- py --contre main
npm run differentiel
```

- `check:<id>` vérifie un langage (`go`, `rust`, `ts` ou `py`) : analyse statique (aucune en Python), tests du langage, puis contrat.
- `contract` compile Go, Rust et TypeScript, puis joue le contrat sur les implémentations présentes.
- `check` vérifie les quatre langages.
- `bench` compile, puis mesure le temps total et le temps de calcul de chaque langage sur huit scénarios, dont un terme à 1 000 000 et un terme impair, 999 999 (la branche impaire du dernier bit de `fast`). Il prend aussi des identifiants de langage; sans `--runs`, il fait 11 exécutions.
- `differentiel` lance la boucle différentielle, un outil de recherche hors de `check` et de `contract` (ADR-0001) : elle joue les quatre CLI sur les mêmes entrées et rend les écarts, sans attendu ni référence, en code 1 s'il y en a.
- `bench -- <id> --contre REF` mesure un travail de démarrage par un A/B apparié : la révision REF, extraite sous `.build/revisions/` (à vider par `npm run bench -- --nettoyer`, jamais par `Remove-Item -Recurse` : chaque révision porte un lien vers `node_modules`), contre l'arbre de travail, lancées en alternance, avec la médiane des différences par paire (ADR-0004). Sans `--runs`, l'A/B fait 100 paires ; un `--runs` impair est arrondi au pair supérieur, pour que chaque côté parte le premier aussi souvent.

## Prérequis

- Go 1.26 ou plus
- Rust stable, édition 2024
- Node 24.2 ou plus, ou 22.18 et plus dans la ligne 22, avec npm : la CLI TypeScript lance son point d'entrée par `import.meta.main`
- Python 3.12 ou plus

Sous Windows, la toolchain Rust MSVC exige le composant C++ de Visual Studio Build Tools. Sans lui, utilisez la toolchain GNU pour ce dossier seulement, depuis la racine du dépôt :

```bash
rustup override set stable-x86_64-pc-windows-gnu
```

rustup range cette override sous le chemin absolu du dossier : elle couvre ses sous-dossiers, donc les worktrees, mais se refait après un déplacement du dépôt ou dans un nouveau clone. L'ancienne apparaît alors en « not a directory » dans `rustup override list`, et `rustup override unset --nonexistent` la retire.

Le fichier `.gitattributes` force les fins de ligne LF, dont le contrat a besoin.

## Organisation

```
Fib/
├── README.md, PRD.md, CLAUDE.md, CONTEXT.md
├── docs/
│   ├── adr/             décisions durables
│   ├── agents/          configuration des skills de Matt Pocock
│   ├── cas-de-test.md   catalogue des cas de test (TDD)
│   └── exploration.md   journal des skills utilisées
├── contracts/           oracle, lancement commun des CLI (lancement.ts), test de contrat, banc d'essai (bench.ts) et boucle différentielle (differentiel.ts)
├── go/  rust/  ts/  python/
├── .build/              Go et TypeScript compilés, et révisions extraites par `bench --contre` et `differentiel --revision` (ignoré par git)
├── package.json         point d'entrée des vérifications
├── tsconfig.json        typage de ts/ et contracts/
└── tsconfig.build.json  compile ts/src en JavaScript dans .build/ts
```

Les quatre implémentations ne partagent aucun code ([ADR-0001](docs/adr/0001-implementations-independantes.md)). Elles ne se rejoignent que par le contrat de la CLI et par l'oracle.

## Documentation

- [PRD.md](PRD.md) : exigences, contrat de la CLI, pièges vérifiés par langage, démarche.
- [CONTEXT.md](CONTEXT.md) : glossaire du domaine (indice, terme, liste, comparaison, discordance, écart, oracle…).
- [docs/cas-de-test.md](docs/cas-de-test.md) : les cas de test, leurs attendus, leurs sources et l'ordre des cycles TDD.
- [docs/adr/](docs/adr/) : [implémentations indépendantes](docs/adr/0001-implementations-independantes.md), [analyse manuelle des arguments](docs/adr/0002-analyse-manuelle-des-arguments.md), [taille de l'implémentation Go](docs/adr/0003-taille-de-l-implementation-go.md) (caduque depuis l'ADR-0004) et [performance du calcul](docs/adr/0004-performance-du-calcul.md).
- [docs/exploration.md](docs/exploration.md) : ce que chaque skill a apporté et ce qu'elle a coûté.

## Démarche

Le travail suit le flux des skills de Matt Pocock :

`/setup-matt-pocock-skills` → `/grill-with-docs` → `/to-spec` → `/to-tickets` → `fib-dev` avec `tdd` (à la place de `/implement`) → `/code-review`

- **Orchestration** : une session Claude Code (Opus 5.5, en mode `ultracode`) découpe le travail, intègre et vérifie.
- **Implémentation** : chaque ticket part chez un sous-agent `fib-dev` (Sonnet 5.5), dans son propre worktree, test d'abord.

La répartition des rôles est décrite dans [CLAUDE.md](CLAUDE.md).

## Licence et attributions

Aucune licence n'est encore déclarée pour le code de ce dépôt.

L'oracle des tests (`contracts/b000045.txt`) est généré par la formule de Binet et vérifié par la définition de la suite. Il coïncide avec la suite [A000045 de l'OEIS](https://oeis.org/A000045), qui reste la référence publique.
