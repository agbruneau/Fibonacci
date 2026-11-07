# PRD — Fib : calculateur Fibonacci académique en quatre langages

Version 1.15 · 2026-10-13 · André-Guy Bruneau. Révisée par `/grill-with-docs`, par un audit, par le catalogue de cas de test, par l'implémentation des tickets #2 à #5, par l'audit et la refactorisation du 2026-10-11, par l'optimisation des performances (ADR-0004), par la borne d'un terme à 1 000 000 et le bilan des résultats, puis par l'étape 9 (`diagnosing-bugs`, #16), par les cas hors contrat (#18), par l'étape 10 (#20), par l'étape 11 (`/handoff`) et par les pistes A, C, D et E de l'étape 10 (#27 à #36 : lancement commun des CLI, A/B apparié, boucle différentielle, gardes du code 3, de l'algorithme exécuté et du texte décimal), puis par la revue finale de cohérence (#39 à #43 : gardes d'identité de l'aiguillage, PERF-TS-11, HAR-7) : le vocabulaire est dans `CONTEXT.md`, les décisions durables dans `docs/adr/`.

## 1. Contexte et objectifs

Fib sert de terrain d'essai pour explorer les skills de Matt Pocock (plugin `mattpocock-skills` v1.2.3) en Go, Rust, TypeScript et Python. Le calcul de Fibonacci est un prétexte : chaque implémentation reste académique et lisible. La référence fonctionnelle est [agbruneau/Fibonacci](https://github.com/agbruneau/Fibonacci) (FibCalc, en Go); Fib n'en garde qu'un sous-ensemble.

- **O1** — Une même CLI dans les quatre langages, avec un contrat respecté à l'octet près.
- **O2** — Chaque skill du flux de Matt Pocock (§ 9) sert au moins une fois sur du vrai travail, et son constat est consigné.
- **O3** — Un code qu'un étudiant lit d'un trait : l'algorithme, sans optimisation au-delà. Remplacé le 2026-10-11 par l'ADR-0004 : le temps de réponse prime (temps total et `--time`, mesurés par `npm run bench`), et chaque optimisation porte son pourquoi en commentaire.

## 2. Ce que Fib retient de FibCalc

| FibCalc | Fib | Raison |
|---|---|---|
| `-algo fast` (fast doubling) | `fast` (doublement rapide) | cœur académique en O(log n) |
| `-algo matrix` (avec Strassen) | `matrix`, sans Strassen | exponentiation matricielle simple |
| — | `iterative` | base de comparaison en O(n) |
| `-algo all`, par défaut, en concurrence | `all` (comparaison), séquentielle, sur demande | des durées mesurées sans concurrence; un seul calcul par défaut |
| `-n`, qui accepte `0x…` et `_` | indice `N` positionnel, en décimal ASCII seulement | la même règle dans les quatre langages |
| code 3 sur discordance | code 3 | repris tel quel |
| code 4 sur erreur de configuration | code 2 | convention Unix des erreurs d'usage |
| usage sur stderr | `--help` sur stdout | l'aide demandée est une sortie normale |
| `-details` : durée du calcul seul | `--time` : même périmètre | mesures comparables |
| golden généré par le code | oracle indépendant du code (Binet, vérifié par la définition) | ADR-0001 |
| `fft`, `gmp`, TUI, calibration, seuils réglables, `-memory-limit`, `-gc-control`, `-timeout`, `-last-digits`, `-output`, variables `FIBCALC_*`, profilage, complétion, Docker, CI | exclus | voir § 8 |

Fib ajoute `--list`, qui produit la liste F(0) … F(N).

## 3. Domaine

Le vocabulaire (indice, terme, liste, borne, algorithme, comparaison, discordance, oracle, contrat, test de garde, banc d'essai…) est défini dans `CONTEXT.md`.

- F(0) = 0, F(1) = 1, F(n) = F(n−1) + F(n−2) pour n ≥ 2. Les termes sont des entiers exacts, jamais en virgule flottante ni tronqués.
- `iterative` : deux termes consécutifs en mémoire, O(n) additions, faites sur place en Go et en Rust.
- `matrix` : [[1,1],[1,0]]ⁿ = [[F(n+1), F(n)], [F(n), F(n−1)]], par carrés successifs de gauche à droite. Une puissance de Q est symétrique, donc trois coefficients suffisent, et multiplier par Q ne coûte que des additions.
- `fast` : doublement rapide, de F(k) à F(2k) ou F(2k+1) à chaque bit de n, en deux multiplications de grands entiers : deux carrés par les formules de GMP (F(2k−1) = F(k)² + F(k−1)², F(2k+1) = 4F(k)² − F(k−1)² + 2(−1)ᵏ), ou un produit et un carré par les nombres de Lucas en TypeScript (F(2k) = F(k)·L(k), L(2k) = L(k)² − 2(−1)ᵏ). La dernière étape ne calcule que F(n), comme celle de `matrix`.

## 4. Contrat de la CLI

### Grammaire

Chaque implémentation analyse ses arguments à la main selon cette grammaire (ADR-0002). Options et indice peuvent venir dans n'importe quel ordre.

1. Avant toute autre analyse : si un argument vaut exactement `--help` ou `-h`, y compris à la place de la valeur de `--algo`, le texte d'usage va sur stdout, code 0, quels que soient les autres arguments.
2. `--algo X` ou `--algo=X`, avec X exactement l'un de `iterative`, `matrix`, `fast` ou `all`, casse comprise. Dans la forme `--algo X`, X est l'argument suivant, quel qu'il soit : `--algo --list 5` est une erreur. Chaque occurrence doit être valide : `--algo bogus --algo fast 5` est une erreur. Parmi des occurrences valides, la dernière gagne. Sans valeur, ou avec une valeur vide, c'est une erreur d'usage.
3. `--list` et `--time` sont des drapeaux; les répéter ne change rien.
4. Tout autre argument qui commence par `-` est une option inconnue, `-1` compris.
5. Il reste exactement un argument, l'indice `N`. Il doit correspondre à `^[0-9]+$` (chiffres ASCII) **avant toute conversion**. Les zéros de tête sont permis : `007` vaut 7.
6. `--list` combiné à `--algo`, même `--algo fast`, est une erreur d'usage.

Texte d'usage, identique à l'octet près dans les quatre langages :

```
usage: fib [--algo iterative|matrix|fast|all] [--time] N
       fib --list [--time] N
       fib --help
```

### Exigences

| ID | Exigence |
|---|---|
| F1 | `fib N` écrit le terme F(N) en décimal, suivi de `\n`, sur stdout. Code 0. Algorithme par défaut : `fast`. |
| F2 | `--algo X` choisit l'algorithme. La sortie est la même pour tout X. |
| F3 | `--algo all` lance la comparaison : `fast`, `iterative` puis `matrix`, l'un après l'autre (ordre alphabétique, comme FibCalc). S'ils concordent, le terme est écrit une seule fois. En cas de discordance, stdout reste vide, une seule ligne qui commence par `fib: ` va sur stderr, sans durée même avec `--time`, et le code est 3. |
| F4 | `--list N` écrit la liste F(0) … F(N), un terme par ligne (N + 1 lignes, chacune terminée par `\n`, la dernière comprise), générée par l'algorithme `iterative`. |
| F5 | `--list` et `--algo` s'excluent (grammaire, règle 6). |
| F6 | `--time` écrit sur stderr une ligne par algorithme exécuté, dans l'ordre d'exécution : `<algo> <durée> ms`, la durée à 3 décimales avec un point décimal, quelle que soit la locale. La durée ne couvre que le calcul : ni l'analyse des arguments, ni la conversion décimale d'un terme, ni l'écriture. `--list` compte comme `iterative`; sa durée couvre la production du texte décimal, puisque la liste est calculée en décimal (ADR-0004). stdout ne change pas. |
| F7 | `--help` et `-h` écrivent le texte d'usage ci-dessus sur stdout. Code 0. |
| F8 | Bornes : N ≤ 1 000 000 pour un terme (208 988 chiffres); N ≤ 10 000 pour une liste, soit environ 10,5 Mo de sortie : une liste jusqu'à 1 000 000 en écrirait environ 100 Go. `iterative` reste en O(n) additions de grands entiers, donc en O(n²) opérations sur les mots, et prend environ 1,6 s en Go et 5 s en Python à N = 1 000 000 (de même pour `all`), ce qui est accepté. |
| F9 | Erreur d'usage (tout écart à la grammaire, ou N hors borne, y compris trop grand pour un entier natif) : stdout vide, exactement une ligne sur stderr qui commence par `fib: `, code 2. Le texte qui suit le préfixe est libre. Pas de panique, de trace de pile ni de texte d'usage ajouté. |
| F10 | stdout et stderr sont en ASCII, sans séquence ANSI, avec des fins de ligne `\n` sur toutes les plateformes, Windows compris. |

Dans le contrat, les codes de sortie sont 0 succès, 2 erreur d'usage, 3 discordance; tout autre code signale une défaillance. Hors contrat, le code est libre (voir plus bas).

**Hors contrat** (comportement non spécifié, non testé) :

- un stdout fermé, au lancement ou en cours d'écriture.
- un argument qui n'est pas de l'Unicode valide.
- un stdout ou un stderr **ouvert** dont l'écriture échoue : tube non bloquant plein (`EAGAIN`), `/dev/full`, tube sans lecteur (`fib --list 10000 | head`).
- toute limite de ressources (`ulimit` ou l'équivalent du système), par exemple mémoire, espace d'adressage, nombre de descripteurs ouverts, taille de fichier.
- l'ordre relatif des octets de stdout et de stderr quand ils partagent une même destination : tube (`2>&1`), fichier ou terminal. Le contrat juge chaque flux séparément.

Dans ces cas, ni F9 ni F10 ne s'appliquent : le code, le texte et les traces sont libres.

En revanche, un stderr fermé au lancement (`2>&-`) est dans le contrat : il ne change ni stdout ni le code (cas CLI-V04). C'est la seule exception : un stdout fermé, au lancement ou en cours d'écriture, et un flux ouvert dont l'écriture échoue, stderr compris, sont hors contrat.

## 5. Exigences par langage

Chaque langage expose une seule seam de bibliothèque, nommée selon ses conventions :

- `fib(n, algo)` → le terme F(n), pour l'un des trois algorithmes. Elle rejette un algorithme inconnu, et n < 0 quand le type l'admet.
- `fibList(n)` → la liste F(0) … F(n), en chaînes décimales calculées directement en base 10 (ADR-0004).

La CLI est un adaptateur mince : grammaire, bornes, comparaison, durées, formatage, codes de sortie.

| | Go | Rust | TypeScript | Python |
|---|---|---|---|---|
| Dossier | `go/` | `rust/` | `ts/` | `python/` |
| Identifiant (scripts, test de contrat) | `go` | `rust` | `ts` | `py` |
| Version minimale | Go 1.26 | stable, édition 2024 | Node `^22.18` ou `>=24.2`, TypeScript 7 | Python 3.12 |
| Grand entier | `math/big` | crate `num-bigint` 0.5 | `bigint` natif | `int` natif |
| Liste en décimal | additionneur en base 10^18 | additionneur en base 10^18 | additionneur en base 10^4 | `decimal.Decimal` |
| Grand terme en décimal | conversion de `math/big` | conversion de `num-bigint` | conversion de V8 | `int` écrit par tranches, ou par `str()` au-delà de 200 000 bits; `fast` ou `matrix` seul calculé en `decimal.Decimal` dès l'indice 170 000 |
| Arguments | analyse manuelle (ADR-0002) | idem, depuis `std::env::args_os` | idem | idem |
| Tests | `go test`, tests table-driven | `cargo test` | `node --test` | `unittest` |
| Analyse statique | `go vet` | `cargo clippy --all-targets -- -D warnings` | `tsc` (sans émission) | — |
| Exécutable du contrat, depuis la racine | `go -C go build -o ../.build/fib ./cmd/fib`, puis `.build/fib` (`fib.exe` sous Windows, suffixe donné à `-o`) | `cargo build -q --release --manifest-path rust/Cargo.toml`, puis `rust/target/release/fib` (`fib.exe` sous Windows) | `npx tsc -p tsconfig.build.json` (émet dans `.build/ts/`), puis `node .build/ts/cli.js` | `python -I -S python/fib.py` |

Dépendances autorisées : `num-bigint`, avec ses dépendances transitives `num-integer`, `num-traits` et `autocfg`, pour Rust; `typescript` et `@types/node` en `devDependencies` pour TypeScript, avec leurs dépendances transitives (`undici-types`, et le binaire natif `@typescript/typescript-<plateforme>` de tsc 7). Aucune autre.

Durée indicative du calcul itératif de F(100 000) sur la machine de développement, mesurée le 2026-10-11 après optimisation (ADR-0004) : Go 16 ms, Rust 26 ms, Node 42 ms, Python 54 ms. Le temps de réponse complet, démarrage compris, se mesure par `npm run bench`.

### Pièges vérifiés

- **Tous les langages** : valider `N` comme `^[0-9]+$`, par expression régulière ou par une boucle sur les caractères, avant de le convertir. `int()` de Python accepte `+5`, `1_000`, ` 5 ` et des chiffres non ASCII; `Number` et `BigInt` acceptent `+5`, ` 5` et `0x10`, et valent 0 pour la chaîne vide; `parse::<u64>()` de Rust accepte `+5`, tout comme `strconv.Atoi` de Go (mais pas `ParseUint`). Vérifié le 2026-10-10. En Python, `s.isascii() and s.isdigit()` évite l'import de `re`, qui coûte 8 ms au démarrage. Avec une regex, il faut `re.fullmatch`, parce que le `$` de `^[0-9]+$` accepte un saut de ligne final (cas CLI-I16).
- **Python** : sous Windows, stdout et stderr écrivent `\r\n`, et `PYTHONIOENCODING` change leur encodage (cas CLI-V03). Au point d'entrée, appeler `reconfigure(encoding="ascii", newline="\n")` sur stdout et sur stderr. Un flux standard fermé au lancement (`2>&-`) vaut `None`, et `reconfigure` y lève `AttributeError` avant `main` : la CLI sort en code 1, stdout vide (cas CLI-V04). Lui donner un puits, `open(os.devnull, "w")`, avant `reconfigure`, avec `import os` dans cette seule branche, car il coûte 2,5 ms au démarrage sous `-S` (médiane de 200 lancements, 2026-10-12). `str()` d'un terme de plus de 4 300 chiffres lève `ValueError` : appeler `sys.set_int_max_str_digits(0)` au chargement du module. Au-delà d'environ 9 000 chiffres, `str()` importe `_pylong` (environ 10 ms) : écrire le terme par tranches, mais seulement jusqu'à 200 000 bits, au-delà desquels l'écriture par tranches, quadratique, perd. Dès l'indice 170 000, la CLI calcule `fast` ou `matrix` seul (pas `all`) en `decimal.Decimal`, dont `str()` est linéaire (ADR-0004).
- **TypeScript** :
  - Le `package.json` racine doit déclarer `"type": "module"`, sans quoi tsc 7 refuse les imports ES.
  - N'utiliser que de la syntaxe effaçable (`erasableSyntaxOnly` : ni `enum` ni `namespace`) et des imports relatifs avec l'extension `.ts` (`allowImportingTsExtensions`).
  - Terminer en posant `process.exitCode`, jamais par `process.exit()` : sur un pipe POSIX, les écritures sont asynchrones et `exit()` peut tronquer la sortie. Ce n'est pas reproduit sous Windows, où elles sont synchrones.
  - Le point d'entrée est `if (import.meta.main) process.exitCode = run(process.argv.slice(2));`, ce qui laisse `cli.test.ts` importer `run` sans rien lancer. `import.meta.main` existe depuis Node 24.2 et, rétroporté, depuis 22.18, mais pas en 23.x : la contrainte est `^22.18 || >=24.2`, et la vérification sous Linux tourne sous Node 22.22. Sous un Node plus ancien, `cli.ts` lève à l'import une erreur qui nomme cette version, au lieu de laisser la CLI sortir en 0 sans rien faire.
  - `node --test` prend un glob (`"ts/**/*.test.ts"`), pas un dossier.
- **Go** : `go run` remplace le code de sortie du programme par 1 et ajoute `exit status N` sur stderr (vérifié le 2026-10-10) : le contrat s'exécute donc sur un binaire compilé par `go build`, dont le premier build à froid prend environ 17 s et les suivants moins d'une seconde.
- **Commandes de lancement** : Rust et TypeScript passent aussi par un build préalable, non par `cargo run -q` (qui conserve pourtant le code et le stderr) ni par `node ts/src/cli.ts` : le contrat ne paie plus ces surcoûts à chaque cas, soit environ 50 ms pour `cargo run` et 25 ms pour le retrait des types d'un `.ts` (médiane de 9 lancements de `10`, 2026-10-11). Python prend `-I -S`, qui saute l'import de `site` (environ 3 ms). Sous Windows, l'horloge peut afficher `0.000 ms` pour un petit N, ce qui est accepté.
- **Rust** : `std::env::args` panique sur un argument qui n'est pas de l'Unicode valide, d'où `args_os`. Sous Windows, la toolchain MSVC exige le composant C++ de Visual Studio Build Tools, absent de la machine de développement. Le dossier du dépôt y utilise donc une override rustup locale (`rustup override set stable-x86_64-pc-windows-gnu`), qui ne fait pas partie du dépôt mais couvre ses sous-dossiers, donc les worktrees de `fib-dev` sous `.claude/worktrees/` (vérifié). Rangée sous le chemin absolu du dossier, elle se refait après un déplacement du dépôt ou dans un nouveau clone (vérifié sous Linux le 2026-10-14). Un `rust-toolchain.toml` ne la remplace pas : son canal nommerait le triplet `x86_64-pc-windows-gnu` partout, Linux et macOS compris (non éprouvé). Ailleurs, la toolchain par défaut convient.

## 6. Dépôt

```
Fib/
├── README.md, PRD.md, CLAUDE.md, CONTEXT.md, .gitignore, .gitattributes
├── .claude/            agents/fib-dev.md (sous-agent) et settings.json (orchestrateur, worktrees), voir CLAUDE.md
├── docs/adr/ (0001 à 0004), docs/agents/, docs/cas-de-test.md, docs/exploration.md
├── contracts/
│   ├── b000045.txt      oracle : F(0) … F(2000), au format du b-file OEIS A000045
│   ├── generer_oracle.py  génère l'oracle par la formule de Binet
│   ├── README.md        provenance de l'oracle
│   ├── oracle.ts        lecture et vérification de l'oracle (ORA), en TypeScript
│   ├── lancement.ts     commandes de build et de lancement des CLI (§ 5) et extraction d'une révision, partagées par le contrat, le banc et la boucle
│   ├── bench.ts         banc d'essai des temps de réponse, et son A/B apparié (`npm run bench`)
│   ├── differentiel.ts  boucle différentielle, outil de recherche hors du contrat (`npm run differentiel`)
│   └── cli.test.ts      test de contrat, et contrôles du harnais (HAR)
├── go/  rust/  ts/  python/
├── package.json         "type": "module", point d'entrée unique (avec package-lock.json)
├── tsconfig.json        couvre ts/ et contracts/
└── tsconfig.build.json  compile ts/src en JavaScript dans .build/ts (ADR-0004)
```

Les quatre implémentations sont indépendantes (ADR-0001) : la seule frontière est la CLI, et la seule source de vérité partagée est `contracts/`. Le banc d'essai et la boucle différentielle y vivent aussi, sans en être des sources : le banc mesure des temps, et la boucle compare des sorties entre elles, sans attendu ni référence (amendement de l'ADR-0001, gardé par HAR-6).

| Script de `package.json` | Rôle |
|---|---|
| `check:go`, `check:rust`, `check:ts`, `check:py` | analyse statique (aucune en Python, § 5), tests du langage, puis test de contrat pour ce seul langage |
| `contract` | test de contrat pour les implémentations présentes |
| `bench` | banc d'essai : médianes du temps total et du temps de calcul, par langage et par scénario, après le même build que le contrat et par le même lancement (`--runs N`, 11 par défaut; ADR-0004). `--contre REF` : A/B apparié de la révision REF, extraite sous `.build/revisions/<sha>`, contre l'arbre de travail, en temps total (médiane des différences par paire; `--runs` vaut alors 100 par défaut, et un nombre impair y est arrondi au pair supérieur, que le banc écrit sur stderr : l'ordre alterne, et seul un nombre pair de paires le répartit à égalité, ADR-0004), pour un travail de démarrage. `--nettoyer` vide `.build/revisions/` |
| `differentiel` | boucle différentielle, un outil de recherche : lance les implémentations présentes sur les mêmes entrées (grammaire, valeurs, environnement, descripteurs) et regroupe celles dont les sorties diffèrent, sans référence ni attendu. Code 1 s'il y a un écart. `--dimension`, `--jetons K`, `--revision REF` (les implémentations de REF entre elles, extraites sous `.build/revisions/<sha>` comme celles de l'A/B). Hors de `check` et de `contract` (ADR-0001, HAR-6) |
| `check` | les quatre `check:*` |

`node contracts/cli.test.ts [id…]` vérifie les implémentations nommées par leur identifiant. Un identifiant dont le dossier manque fait échouer le test. Il joue toujours l'oracle (ORA-01 à ORA-06) et le groupe `harnais`, qui suppose les quatre dossiers (sans eux, HAR-5 et HAR-6 rougissent), puis les implémentations demandées. Sans argument, il vérifie les implémentations dont le dossier existe. Pour Go, Rust et TypeScript, il compile d'abord (§ 5). Chaque lancement d'une CLI, par le test de contrat comme par le banc d'essai et la boucle différentielle, a un délai de 300 s.

## 7. Tests et critères d'acceptation

**Seams convenues** (celles que `tdd` utilisera, aucune autre) :

1. la bibliothèque `fib` / `fibList` de chaque langage;
2. la CLI, par le test de contrat commun.

Les tests de garde `PERF-<ID>-<n>` (ADR-0004) peuvent viser une fonction privée ou l'interface dans le processus de la CLI. Ce ne sont pas des seams.

**Oracle**

- `contracts/b000045.txt` est la table F(0) … F(2000) au format du b-file OEIS A000045 : une ligne `n F(n)`, avec une espace, F(n) sans zéro de tête, fins de ligne LF, sans en-tête. Les lecteurs ignorent les lignes vides.
- Le téléchargement du b-file ayant été bloqué par les permissions de la session, la table est générée par `contracts/generer_oracle.py`. Ce script applique la formule de Binet en arithmétique décimale à 600 chiffres, une méthode indépendante des trois algorithmes testés (amendement de l'ADR-0001).
- Son exactitude ne dépend pas du générateur : ORA-04 et ORA-05 la vérifient par la définition seule, et toute table qui les passe est la suite de Fibonacci. Elle coïncide avec la page OEIS pour F(2000).
- Les termes attendus viennent de l'oracle ou de faits mathématiques indépendants du code. Ils ne sont jamais recalculés comme le fait le code (anti-pattern « tautologique » de `tdd`).

**Cas de test** : `docs/cas-de-test.md` est la source unique des cas. Il donne, pour chacun, l'attendu et sa source, ainsi que l'ordre des cycles TDD.

- ORA-01 à ORA-06 : vérification de l'oracle par le test de contrat, avant tout cas CLI.
- BIB-01 à BIB-14 : la bibliothèque, dans chaque langage (seam 1); BIB-14 est sans objet en Rust.
- les cas CLI : le contrat de la CLI, joué sur chaque implémentation (seam 2). Des assertions communes s'appliquent à chaque cas : code de sortie, stdout octet pour octet, stderr, format ASCII.

Le chemin de la discordance (code 3) dans la CLI n'a pas de cas de test, puisque aucun cas ne peut la provoquer. BIB-08 vérifie que les algorithmes concordent. En TypeScript, PERF-TS-7 éprouve ce chemin par l'interface dans le processus (`run(args, io)`, avec un `fib` discordant); PERF-GO-9 et PERF-RS-11 l'éprouvent de même par `run`, avec un calcul discordant passé en paramètre; PERF-PY-26 par `main`, avec un algorithme discordant substitué dans la table `_ALGOS`.

**Acceptation globale**

- **A1** — Chaque ticket de langage : les cas ORA, les cas BIB de son langage et tous les cas CLI de son identifiant passent, donc `npm run check:<id>` et `npm run contract` sont verts sur `main`. Le projet : `npm run check` passe sous Windows 11 une fois les quatre implémentations livrées.
- **A2** — Aucune dépendance hors de celles du § 5.
- **A3** — Remplacé par l'ADR-0004 : la taille suit la performance. Mesuré le 2026-10-14 (`wc -l` des sources, hors fichiers de test) : TypeScript 270, Python 270, Rust 568 (dont 60 lignes de tests unitaires dans `lib.rs` et 169 dans `main.rs`), Go 394. Le 2026-10-13 : 268, 270, 501 (dont 60 et 102) et 385; le 268 de TypeScript a précédé adac033 (#41), qui a ajouté deux lignes de commentaire à `fib.ts` le même jour. Les gardes des audits du 2026-10-14 n'ajoutent que des tests : les 67 lignes de plus en Rust sont celles de PERF-RS-14 et de PERF-RS-12 dans `main.rs`. Les 9 lignes de plus en Go viennent du dernier bit impair de `fast` en un produit, décidé par le propriétaire le même jour. Le 2026-10-11, après optimisation et borne à 1 000 000 : 232, 261, 359 et 376. Avant les travaux du 2026-10-11 : 151, 158, 205 et 255.
- **A4** — `docs/exploration.md` contient une entrée pour chaque skill du § 9 réellement utilisée.

## 8. Hors périmètre

FFT, GMP, Strassen, parallélisme, TUI, mode serveur, calibration et seuils réglables (ceux de Python, ADR-0004, sont fixes), limites de mémoire, de GC et de temps, `-last-digits`, `-output`, variables d'environnement, profilage, complétion de shell, Docker, négafibonacci (n < 0), Mojo, ponts entre langages (FFI, WASM). La CI GitHub Actions est candidate pour une phase 2.

## 9. Démarche d'exploration des skills

L'orchestrateur est la session principale (Opus 5.5 en `ultracode`); le code revient au sous-agent `fib-dev` (Sonnet 5.5). Voir `CLAUDE.md`. Les étapes 0, 1 et 4 à 11 sont faites : les quatre implémentations passent `npm run check`. L'étape 8 a pris la forme d'un audit, puis de quatre chantiers suivis chacun de `/code-review` : la refactorisation, l'optimisation des performances (ADR-0004) et la borne d'un terme à 1 000 000 (2026-10-11), puis les suites de l'épreuve des cas de test (tickets #6 à #14, 2026-10-12). L'étape 9 a corrigé l'écart du stderr fermé en Python (#16, 2026-10-12), que sa boucle différentielle avait trouvé : `npm run differentiel` la rejoue depuis, comme outil de recherche (ADR-0001). L'étape 10 a rendu la CLI TypeScript appelable dans le processus (#20, 2026-10-13). Les étapes 2 et 3 n'ont pas servi. L'étape 11 a produit un document de reprise, hors dépôt comme la skill l'exige (2026-10-13). Les pistes A, C, D et E du rapport de l'étape 10 ont suivi le même jour, conçues en workflow puis livrées en trois vagues (#27 à #36). Une revue finale de cohérence en quatre angles a précédé la version v1.0.0 (#39 à #43). Le 2026-10-14, les audits de chaque dossier de langage (`audit.md`) ont été vérifiés en workflow, puis implémentés par des `fib-dev` et revus par `/code-review` (#45). Le bilan par skill, dont `/implement` remplacée par `fib-dev`, est dans `docs/exploration.md`.

| # | Étape | Skill | Livrable |
|---|---|---|---|
| 0 | Configurer le dépôt | `/setup-matt-pocock-skills` | `docs/agents/`, bloc `## Agent skills` dans `CLAUDE.md`, étiquettes de triage sur GitHub |
| 1 | Éprouver ce PRD | `/grill-with-docs` | `CONTEXT.md`, ADR-0001 et ADR-0002, PRD 1.1 |
| 2 | Lever un doute technique (au besoin) | `research` | note citée dans le dépôt |
| 3 | Trancher une question de forme (au besoin) | `prototype` | prototype jetable |
| 4 | Publier la spec | `/to-spec` | issue GitHub `ready-for-agent` |
| 5 | Découper | `/to-tickets` | tickets tracer bullet, avec leurs bloqueurs |
| 6 | Implémenter | `/implement`, qui applique `tdd` | une branche de worktree par ticket |
| 7 | Intégrer | `resolving-merge-conflicts` (au besoin) | `main` vert |
| 8 | Réviser | `/code-review` | rapports Standards et Spec |
| 9 | Corriger un écart entre langages | `diagnosing-bugs` | test de non-régression; `npm run differentiel` pour chercher l'écart |
| 10 | Revoir l'architecture | `/improve-codebase-architecture`, `codebase-design` | rapport |
| 11 | Passer le relais entre sessions | `/handoff` | document de reprise |

Découpage retenu par `/to-tickets` (spec #1) : le ticket #2 livre le contrat (l'oracle et son `README.md`, le test de contrat avec la vérification de l'oracle, `package.json` et `tsconfig.json`) avec l'implémentation TypeScript, pour qu'une implémentation exerce le harnais dès le départ. Il bloque #3 (Go), #4 (Rust) et #5 (Python), qui livrent chacun tout le contrat d'un coup et avancent en parallèle.

Chaque entrée de `docs/exploration.md` donne : la skill, l'étape ou le ticket, ce qui a aidé, la friction, ce qu'on changerait.

## 10. Risques

| Risque | Parade |
|---|---|
| Fins de ligne CRLF et encodage imposé par l'environnement (Python, checkout Git) | F10, `reconfigure(encoding="ascii", newline="\n")`, `.gitattributes` en `eol=lf`, contrat comparé à l'octet |
| Conversions trop permissives de `N` | validation de `^[0-9]+$` avant conversion (§ 5); cas `+5`, `1_000`, `0x10`, `٣`, `""` |
| Limite de conversion entier → texte de Python | `set_int_max_str_digits(0)`, gardé par CLI-T14 et PERF-PY-14 : les cas `100000` et `1000000` passent désormais par les tranches et par `Decimal` |
| Débordement à l'analyse de `N` | cas `99999999999999999999` |
| Sortie tronquée par `process.exit()` (Node, POSIX) | `process.exitCode`; vérifié sous Linux le 2026-10-13 : `process.exit()` y tronque la sortie dans un tube, et le contrat le voit (CLI-L05, L06 et L16 rougissent) |
| Rust ne lie pas sous Windows sans le composant C++ MSVC | override rustup GNU locale au dossier |
| Termes attendus recopiés dans chaque test, qui finissent par différer | un seul oracle, vérifié par le test lui-même et lu par tous les tests |
| `go run` remplace le code de sortie et pollue stderr | binaire compilé par `go build` (§ 5) |
| Le worktree d'un `fib-dev` part de `origin/main` et ignore le travail fusionné localement | `worktree.baseRef: "head"` dans `.claude/settings.json` |
| Coût en jetons d'`ultracode` | `/effort ultracode off` pour les petites tâches |
| Une optimisation casse un cas limite que les seams ne visent pas : retenue de l'additionneur décimal, dernière étape de `matrix` et de `fast`, chemin `Decimal` de Python | balayages de 0 à 2000 (BIB-05 à BIB-07, BIB-12), tests de garde `PERF-<ID>-<n>`, test différentiel entre les quatre CLI (`npm run differentiel`), sondes de `/code-review` jusqu'à F(1 000 000) |
| Le contrat ou le banc d'essai lance un binaire ou un `.js` périmé | les deux compilent Go, Rust et TypeScript avant de lancer (§ 6), par le même module, `contracts/lancement.ts` |
| Seuils de Python (`DECIMAL_FROM`, `STR_FROM_BITS`) mesurés sur Python 3.14 seulement, alors que le minimum est 3.12 | ils ne changent que la vitesse : chaque chemin écrit le même terme exact, gardé par les tests de garde Python et CLI-T13; à remesurer par `npm run bench` sous 3.12 et 3.13 |
| Mesures bruitées : des agents qui mesurent en même temps se gênent, et les médianes séparées ne tranchent pas un gain de quelques pour cent (démarrage) | médianes; mesures en série, jamais en parallèle; une seule mesure après la fusion fait foi; l'A/B apparié (`bench --contre REF`), qui lance les deux côtés dans le même tour et résume par la médiane des différences, pour un travail de démarrage |
| La boucle différentielle devient un oracle : son verdict, sans attendu, finirait par trancher à la place du PRD, ou par rougir le contrat | hors de `check*` et de `contract`, sans référence ni majorité, sans lecture de l'oracle (amendement de l'ADR-0001); HAR-6 garde son lancement et son import, et son témoin sa projection et son regroupement; un écart se tranche par le PRD § 4, puis par un cas du catalogue |
