# Audit exhaustif du répertoire `ts/`

**Date de l'audit** : 2026-10-14  
**Répertoire audité** : `ts/` (identifiant du contrat : `ts`)  
**Environnement d'exécution vérifié** : Windows 11 (`win32/x64`), Node.js `v24.20.0`, TypeScript `Version 7.0.2` (`tsgo` / `@typescript/native-preview`, minimum exigé par le PRD § 5 : Node.js 24 LTS et TypeScript 6)  
**Statut global** : **Conforme (Vert)** — `tsc --noEmit` (0 erreur), `node --test "ts/**/*.test.ts"` (41 tests de premier niveau, 6 064 sous-tests et assertions passés en ~1,30 s) et le contrat CLI (`node contracts/cli.test.ts ts`, 114 passés, 2 sautés sous Windows pour `CLI-V04`) passent sans aucun écart fonctionnel. Un écart documentaire mineur de 2 lignes de commentaire est relevé entre `ts/src/fib.ts` (270 lignes de production au total) et le tableau du PRD § 7 A3 / `README.md` (qui indique `268`).

---

## 1. Vue d'ensemble et métriques du répertoire

### 1.1 Inventaire des fichiers et volumétrie

L'implémentation TypeScript répartit son code de production sous `ts/src/` en deux modules ES : la bibliothèque (`fib.ts`, seam 1) et l'adaptateur CLI (`cli.ts`, seam 2), accompagnés de leurs fichiers de tests respectifs (`fib.test.ts` et `cli.test.ts`). Toutes les fins de ligne respectent la norme LF (`0` octet CR), conformément à `.gitattributes` et à l'exigence F10.

| Fichier | Rôle | Octets | Lignes (`wc -l`) | Caractères CR |
|---|---|---:|---:|---:|
| `ts/src/fib.ts` | Bibliothèque : `Algo`, `algos`, `fib`, `fibList`, `iterative`, `matrix`, `fast`, `asciiTable`, additionneur décimal base $10^4$ | 6 130 | 129 | 0 |
| `ts/src/cli.ts` | Adaptateur CLI : `Io`, `processIo`, `parse`, `timed`, `compute`, `run`, garde `import.meta.main` | 6 688 | 141 | 0 |
| `ts/src/fib.test.ts` | Tests de la seam 1 (`BIB-01` à `BIB-14`) et gardes `PERF-TS-1` à `PERF-TS-5`, `PERF-TS-10` | 8 411 | 169 | 0 |
| `ts/src/cli.test.ts` | Gardes de la CLI en mémoire et en sous-processus (`PERF-TS-6` à `PERF-TS-9`, `PERF-TS-11`) | 13 757 | 231 | 0 |
| `tsconfig.json` | Configuration de vérification de types à la racine (`strict: true`, `noEmit: true`, `erasableSyntaxOnly: true`) | 309 | 14 | 0 |
| `tsconfig.build.json` | Configuration d'émission JavaScript vers `.build/ts/` pour le contrat et le banc d'essai | 159 | 8 | 0 |
| **Total production (`ts/src/`)** | `fib.ts` + `cli.ts` | **12 818** | **270** | **0** |
| **Total tests (`ts/src/`)** | `fib.test.ts` + `cli.test.ts` | **22 168** | **400** | **0** |

- **Conformité A2 (Dépendances)** : `package.json` ne déclare aucune `dependencies` d'exécution (uniquement `@types/node` et `typescript` dans `devDependencies`). Les fichiers de `ts/src/` n'importent que des modules internes `node:*` ou des chemins relatifs `./*.ts`, ce que vérifie statiquement le contrôle du harnais `HAR-6`. En particulier, le code de production (`fib.ts` et `cli.ts`) ne contient **aucun `import` de module `node:*`** : `fib.ts` utilise uniquement le global `Buffer` et les tableaux typés standard (`Int32Array`, `Uint32Array`, `Uint8Array`), et `cli.ts` n'importe que `./fib.ts`.
- **Constat sur A3 (Taille)** : `wc -l` mesure **270 lignes** de code de production (`129` dans `fib.ts` + `141` dans `cli.ts`), tandis que `PRD.md` (§ 7, A3, ligne 192) et `README.md` (§ Coût, ligne 66) mentionnent `268` lignes. Cet écart de 2 lignes provient du commit final `adac0337` (*« ts: algos en lecture seule par le type, les espions de PERF-TS-10 le convertissent (#41) »*), qui a ajouté deux lignes de commentaires explicatifs dans `ts/src/fib.ts` (lignes 117–120) lors de la revue finale sans mettre à jour le compteur dans `PRD.md` et `README.md`.

### 1.2 Chaîne de compilation et commandes de vérification

- **Vérification de typage** : `tsc --noEmit` — 0 erreur avec `strict: true`, `erasableSyntaxOnly: true`, `verbatimModuleSyntax: true` et `allowImportingTsExtensions: true`.
- **Tests unitaires et de garde** : `node --test "ts/**/*.test.ts"` — exécute directement les fichiers `.ts` grâce au dépouillement de types natif de Node.js 24, sans étape de compilation préalable.
- **Compilation pour le contrat et le banc d'essai (`contracts/lancement.ts`)** :
  ```bash
  node node_modules/typescript/bin/tsc -p tsconfig.build.json
  node .build/ts/cli.js
  ```
  Conformément à l'ADR-0002 et au contrôle du harnais `HAR-7`, le contrat et le banc d'essai compilent une seule fois `ts/src/` vers `.build/ts/cli.js` et `.build/ts/fib.js` (via `rewriteRelativeImportExtensions: true` dans `tsconfig.build.json`) afin de mesurer l'exécution du programme et non le coût de transformation des types au démarrage.

---

## 2. Architecture et seam de bibliothèque (`ts/src/fib.ts`)

### 2.1 Signatures exportées et contrat de la seam 1

Le module `ts/src/fib.ts` expose les éléments suivants :

1. `export type Algo = 'iterative' | 'matrix' | 'fast'` (`fib.ts:2`)
2. `export function fib(n: number, algo: Algo): bigint` (`fib.ts:125-129`)
   - **Validation de l'algorithme contre la chaîne de prototype** :
     ```ts
     if (!Object.hasOwn(algos, algo)) throw new TypeError(`algorithme inconnu : ${algo}`);
     ```
     L'utilisation de `Object.hasOwn(algos, algo)` (plutôt que `algo in algos` ou `if (!algos[algo])`) est cruciale en JavaScript : elle rejette non seulement `'all'`, `'bogus'`, `''` et `'FAST'`, mais aussi toute propriété héritée de `Object.prototype` telle que `'toString'` ou `'valueOf'`. Ce cas est explicitement verrouillé par `BIB-13` (`fib.test.ts:105`).
   - **Validation de l'indice (`checkIndex`, `fib.ts:46-48`)** :
     ```ts
     if (!Number.isInteger(n) || n < 0) throw new RangeError(`indice invalide : ${n}`);
     ```
     Le type `number` de JavaScript étant un flottant IEEE 754 double précision, `Number.isInteger(n)` rejette simultanément `NaN`, `±Infinity` et les nombres décimaux (`1.5`), tandis que `n < 0` rejette les entiers négatifs (`-1`), en levant `RangeError` (`BIB-14`).
3. `export function fibList(n: number): string[]` (`fib.ts:71-115`)
   - Valide `n` via `checkIndex(n)` et retourne un tableau de `n + 1` chaînes décimales représentant $F(0) \dots F(n)$.
4. `export const algos: Readonly<Record<Algo, (n: number) => bigint>> = { iterative, matrix, fast }` (`fib.ts:121`)
   - Table d'aiguillage exposée uniquement pour la garde d'identité `PERF-TS-10`. Elle est protégée en lecture seule au niveau du système de types (`Readonly<...>`), sans `Object.freeze` à l'exécution afin de n'introduire aucun surcoût et de permettre aux trois sous-tests d'espionnage de `PERF-TS-10` d'y substituer temporairement leurs sondes.

---

## 3. Audit algorithmique et arithmétique en précision arbitraire

### 3.1 Algorithme itératif (`iterative`, `fib.ts:5-10`)

```ts
function iterative(n: number): bigint {
  let [a, b] = [0n, 1n];
  for (let i = 0; i < n; i++) [a, b] = [b, a + b];
  return a;
}
```

- **Exactitude et compilation JIT** : partant de $(F(0), F(1)) = (0, 1)$, effectue $n$ pas de récurrence. Le compilateur JIT de V8 élimine l'allocation du tuple de déstructuration `[a, b] = [b, a + b]` (comme documenté au commentaire ligne 7, une variable temporaire explicite ou un déroulage par 2 n'apportent aucun gain mesurable sur V8).

### 3.2 Algorithme matriciel (`matrix`, `fib.ts:14-26`)

- **Parcours des bits du poids fort au poids faible** : maintient les trois coefficients $(a, b, c)$ de la matrice symétrique $Q^k = \begin{pmatrix} F(k+1) & F(k) \\ F(k) & F(k-1) \end{pmatrix}$ initialisée à l'identité $Q^0 = (1, 0, 1)$.
- **Élévation au carré en trois carrés sans produit croisé** : pour chaque bit de `bits.slice(0, -1)` (tous les bits sauf le bit de poids faible), calcule `const [aa, bb, cc] = [a * a, b * b, c * c]`, puis `[a, c] = [aa + bb, bb + cc]` et déduit $b' = b(a + c)$ par la simple soustraction `b = a - c` (car $a' - c' = a^2 - c^2 = (a - c)(a + c) = b(a + c)$). Si le bit vaut `'1'`, multiplie par $Q$ via deux additions `[a, b, c] = [a + b, a, b]`.
- **Optimisation du dernier bit (`fib.ts:25`)** : hors de la boucle, calcule uniquement l'entrée $F(n)$ :
  ```ts
  return bits.endsWith('0') ? b * (a + c) : a * a + b * b;
  ```
  Pour $n = 0$, `bits` vaut `"0"`, `bits.slice(0, -1)` est vide et `bits.endsWith('0')` retourne `0n * 2n = 0n`.

### 3.3 Algorithme de doublement rapide par la suite de Lucas (`fast`, `fib.ts:30-44`)

Conformément à l'ADR-0003 (`docs/adr/0003-formules-arithmetiques-par-langage.md`), l'implémentation TypeScript est la seule des quatre à utiliser la **suite de Lucas** $L(k) = F(k-1) + F(k+1)$ au lieu du couple $(F(k), F(k-1))$ de GMP :
- **Justification liée au moteur V8** : dans V8, les `BigInt` sont immuables (toute addition ou soustraction alloue un nouveau `BigInt`), mais le décalage binaire d'un bit (`>> 1n`) est très rapide et les identités de Lucas réduisent l'étape de doublement à **1 produit et 1 carré** :
  $$F(2k) = F(k) \cdot L(k), \qquad L(2k) = L(k)^2 - 2(-1)^k$$
- **Pas impair sans multiplication de grands entiers** : lorsque le bit courant vaut `'1'`, le passage de $2k$ à $2k+1$ utilise deux divisions par 2 exactes :
  $$F(2k+1) = \frac{F(2k) + L(2k)}{2}, \qquad L(2k+1) = \frac{5F(2k) + L(2k)}{2}$$
  codées par `[(even + lucas) >> 1n, (5n * even + lucas) >> 1n, -1n]`.
- **Optimisation du dernier bit (`fib.ts:42-43`)** : au bit de poids faible, $L(n)$ n'est pas nécessaire. Si le dernier bit est `'0'`, seul le produit `even = f * l` est calculé (**une seule multiplication**, sans élever `l` au carré) ; s'il est `'1'`, retourne `(even + l * l - 2n * sign) >> 1n`.
- **Performance obtenue** : sur la machine d'audit, ce choix permet à TypeScript/V8 de calculer $F(1\,000\,000)$ en **2,9 ms** de temps interne (`internal_ms`), soit le calcul pur le plus rapide des quatre langages devant Go (3,2 ms), Rust (5,3 ms) et Python (12,9 ms).

### 3.4 Additionneur décimal en base $10^4$ et encodage ASCII simultané (`fibList`, `fib.ts:50-115`)

Pour éviter $10\,001$ conversions binaires $\to$ décimales de `BigInt` lors de `--list 10000` (qui seraient quadratiques par terme, donc cubiques sur la liste), `fibList` implémente un additionneur décimal sur mesure en base $10^4$ (`BASE = 10_000`) :
1. **Dimensionnement et limbe d'assurance (`fib.ts:75`)** :
   ```ts
   const size = Math.ceil((n * 0.209) / 4) + 1;
   ```
   Comme $\log_{10}(\varphi) \approx 0{,}2089876 < 0{,}209$, $\lceil 0{,}209 \cdot n / 4 \rceil$ limbes de 4 chiffres suffisent à contenir $F(n)$. Le `+ 1` final est une sécurité indispensable en JavaScript : contrairement aux tableaux ordinaires, un `Int32Array` ignore silencieusement toute écriture hors bornes (`x[len] = 1` avec `len >= size` ne lève aucune exception et perd le chiffre en silence). Cette formule est protégée aux petits indices par `PERF-TS-5` ($n \in [0, 300]$) et à $n = 10\,000$ par `PERF-TS-2`.
2. **Addition en place sans branchement et encodage ASCII dans la même boucle (`addInPlaceAndEncode`, `fib.ts:85-98`)** :
   - Les limbes numériques dans `a` et `b` (`Int32Array(size)`) sont rangés du poids faible (`k = 0`) au poids fort (`k = len - 1`), ce qui permet de ne parcourir que les `len` limbes actifs à chaque terme ($\mathcal{O}(\text{len})$ et non $\mathcal{O}(\text{size})$).
   - La retenue `m` est représentée par `-1` (pas de retenue) ou `0` (retenue de $+1$), et mise à jour sans branchement conditionnel grâce au décalage arithmétique signé sur 32 bits :
     ```ts
     const u = x[k] + y[k] - (BASE - 1) + m; // somme + retenue - BASE
     m = u >> 31;                           // -1 si u < 0, 0 si u >= 0
     x[k] = u + (BASE & m);
     words[size - 1 - k] = ascii[x[k]];
     ```
   - Dans la même instruction, `words[size - 1 - k] = ascii[x[k]]` écrit les 4 octets ASCII du limbe (alignés à droite dans le tampon `words: Uint32Array(size)` partagé avec `bytes: Buffer`).
3. **Extraction `latin1` sans copie inutile (`lastTermText`, `fib.ts:100-104`)** :
   - Avance l'indice `start` au-delà des octets `'0'` (`0x30`) de tête du limbe le plus significatif et appelle `bytes.toString('latin1', start, size * 4)`.
4. **Déroulage par paires (`fib.ts:106-113`)** :
   - La boucle `for (let i = 2; i <= n; i += 2)` alterne `addInPlaceAndEncode(a, b)` pour $F(i)$ et `addInPlaceAndEncode(b, a)` pour $F(i+1)$, évitant tout échange de tampons.

---

## 4. Audit de l'adaptateur CLI (`ts/src/cli.ts`) et conformité au contrat

### 4.1 Conformité à la grammaire (PRD § 4, règles 1 à 6)

| Règle | Implémentation dans `ts/src/cli.ts` | Lignes | Conformité |
|---|---|---|---|
| **Règle 1** (`--help` / `-h` prioritaire) | `if (args.includes('--help') \|\| args.includes('-h'))` en tête de `run`, avant `parse(args)` | `118-121` | Exacte (`CLI-H01` à `H11`) |
| **Règle 2** (`--algo X` / `--algo=X`) | `arg === '--algo' ? args[++i] : arg.slice('--algo='.length)` ; valide chaque occurrence contre `'all'` ou `COMPARISON.find((a) => a === value)` ; la dernière occurrence écrase `algos` | `45-55` | Exacte (`CLI-A01` à `A20`, `D04`, `D05`) |
| **Règle 3** (`--list`, `--time`) | `list = true` et `time = true` idempotents | `56-59` | Exacte (`CLI-L08`, `D08`) |
| **Règle 4** (Options inconnues `-*`) | `else if (arg.startsWith('-'))` rejette `-`, `--`, `-1`, `--list=5`, `--time=1` avec `CliError` (code 2) | `60-61` | Exacte (`CLI-I11`, `L14`, `D10`, `U01` à `U06`) |
| **Règle 5** (Indice `N` unique `^[0-9]+$`) | `positional.length !== 1`, puis `!/^[0-9]+$/.test(positional[0])`, puis `const n = Number(positional[0])` et `if (n > bound)` | `67-72` | Exacte (`CLI-T01` à `T14`, `I01` à `I17`) |
| **Règle 6** (`--list` exclut `--algo`) | `if (list && algos.length > 0)` vérifié avant le repli `algos.length > 0 ? algos : ['fast']` | `66, 73` | Exacte (`CLI-L10` à `L12`) |

### 4.2 Conformité aux exigences F1 à F10 et optimisations du démarrage (ADR-0004)

- **F3 (Comparaison et discordance)** : `COMPARISON = ['fast', 'iterative', 'matrix']`. Dans `compute`, `if (terms.some((term) => term !== terms[0]))` lève `new CliError(..., 3)` avant toute écriture sur `io.stdout` ou `io.stderr`. Le bloc `catch` de `run` écrit uniquement `fib: ${e.message}\n` sur `io.stderr` et retourne `3` (`PERF-TS-7`).
- **F6 (Durées `--time`) et évitement de `performance` (`ADR-0004`, `PERF-TS-9`)** :
  - En Node.js, le premier accès à l'objet global `performance` déclenche le chargement paresseux de 11 modules internes (`perf_hooks`, neuf `internal/perf/*` et `internal/histogram`), ce qui alourdit le démarrage de 1,8 à 2,6 ms.
  - `cli.ts:99, 105` utilise à la place `const ORIGIN = process.hrtime.bigint()` et `clock: () => Number(process.hrtime.bigint() - ORIGIN) / 1e6`, puis `.toFixed(3)` (qui produit toujours un point décimal en JavaScript, indépendamment de la locale, `CLI-V01`).
- **F8 (Bornes et indices géants)** : après validation par `/^[0-9]+$/`, `Number(positional[0])` convertit tout entier dépassant $2^{53}-1$ en flottant (ou `Infinity` au-delà de $10^{308}$), qui est dans tous les cas strictement supérieur à `bound` (`1_000_000` ou `10_000`) et rejeté proprement avec le code `2` (`CLI-I14`). À l'inverse, 5 000 zéros de tête suivis de `5` donnent `Number("00...05") === 5` (`CLI-T14`).
- **F9, F10 (Erreurs en ASCII pur)** : la fonction `ascii = (s: string) => s.replace(/[^\x20-\x7e]/gu, '?')` (`cli.ts:33`) remplace tout octet de contrôle (dont `\n` et `\r`) ou caractère non ASCII par `?` avant de citer l'argument fautif entre guillemets (`CLI-A18`, `CLI-A20`, `CLI-U04`, `CLI-U05`).
- **Accesseurs paresseux sur `processIo` (`cli.ts:106-111`, `PERF-TS-11`)** : `get stdout()` et `get stderr()` retardent l'évaluation de `process.stdout` et `process.stderr` au moment de l'écriture effective, évitant de charger `internal/fs/sync_write_stream` sur un chemin d'erreur lorsque `stdout` est redirigé vers un fichier.
- **Écriture unique en `latin1` (`cli.ts:127`, `PERF-TS-6`)** : `io.stdout.write([...lines, ''].join('\n'), 'latin1')` émet toute la sortie en un seul appel et sans encodage UTF-8.
- **Point d'entrée sécurisé (`cli.ts:139-141`)** : vérifie `if (import.meta.main === undefined)` pour échouer explicitement sur une ancienne version de Node dépourvue de `import.meta.main`, et affecte `process.exitCode = run(...)` sans appeler `process.exit()` pour ne jamais tronquer un tube plein.

---

## 5. Audit de la suite de tests et couverture des gardes

### 5.1 Cas de bibliothèque (`BIB-01` à `BIB-14`, `ts/src/fib.test.ts`)

- Les 2 001 termes de l'oracle sont chargés via `terms()` de `../../contracts/oracle.ts`.
- `BIB-05`, `BIB-06` et `BIB-07` créent chacun 2 001 sous-tests `await t.test(...)` pour `n = 0..2000`.
- `BIB-08` (concordance et 20 899 chiffres à $n = 100\,000$), `BIB-09` (identité de Cassini à $n = 100\,000$ sur les 3 algorithmes), `BIB-10` à `BIB-12` (`fibList`), `BIB-13` (`'all'`, `'bogus'`, `''`, `'FAST'`, `'toString'`) et `BIB-14` (`-1`, `1.5`, `NaN` sur `fib` pour les 3 algorithmes et sur `fibList`) sont tous présents et conformes au catalogue.

### 5.2 Inventaire des 11 tests de garde (`PERF-TS-1` à `PERF-TS-11`)

| ID | Fichier | Cible protégée | Mécanisme de vérification |
|---|---|---|---|
| `PERF-TS-1` | `fib.test.ts:70` | Boucle à pas de 2 et arrêt pair de `fibList` | Vérifie `fibList(10000).length === 10001` (tue `i <= n` et `i += 3`) |
| `PERF-TS-2` | `fib.test.ts:78` | Retenue sortante du limbe de poids fort et retrait des zéros de tête | Vérifie `fibList(10000)[10000].length === 2090` (tue `if (m === 1)` et la suppression du `while (bytes[start] === 0x30)`) |
| `PERF-TS-3` | `fib.test.ts:83` | Format décimal sans zéro de tête dans `fibList(10000)` | Vérifie `/^(0\|[1-9][0-9]*)$/` sur chacun des $10\,001$ termes |
| `PERF-TS-4` | `fib.test.ts:88` | Propagation de retenue entre limbes de 4 chiffres jusqu'à $n = 10\,000$ | Vérifie `BigInt(list[i]) === BigInt(list[i-1]) + BigInt(list[i-2])` pour tout $i \in [2, 10\,000]$ |
| `PERF-TS-5` | `fib.test.ts:100` | Formule de taille `size` aux petits indices et arrêt impair de `fibList` | Vérifie `fibList(n)` contre l'oracle pour chaque $n \in [0, 300]$ (tue `Math.round((n * 0.209) / 4)` sans `+ 1` et `if (i < n - 1)`) |
| `PERF-TS-6` | `cli.test.ts:35` | Écriture unique de `stdout` en `'latin1'` | `harness()` sur `['--list', '300']` et `['--time', '--algo', 'all', '300']` : 1 seul appel `stdout.write` avec `'latin1'` |
| `PERF-TS-7` | `cli.test.ts:45` | Exigence F3 : code 3 sur discordance, `stdout` vide, 1 seule ligne `stderr` sans durée | Injecte un `fib` faussant tour à tour `fast`, `iterative`, `matrix` de `+ 1n` sur `--time --algo all 10` |
| `PERF-TS-8` | `cli.test.ts:75, 94, 103` | Exigence F6 : algorithmes exécutés dans l'ordre, calcul de durée par différence d'horloge, appel de `io.clock()` avec `this` préservé | Horloge simulée avançant de coûts distincts (`0.5`, `1.25`, `2.125` ms), test de `--list` et test d'une horloge à état lisant `this.now` |
| `PERF-TS-9` | `cli.test.ts:166, 174, 186` | Absence de chargement de `perf_hooks` / `internal/perf/performance` et unité réelle en millisecondes de `processIo.clock` | Sous-processus avec crochet `--import` inspectant `process.moduleLoadList` + témoin positif `performance.now()` + vérification que la durée mesurée sur `iterative 100000` est comprise entre `wall / 20` et `wall` |
| `PERF-TS-10` | `fib.test.ts:142, 151` | Identité des fonctions dans `algos` et aiguillage de `fib(n, algo)` | Vérifie `algos[algo].name === algo`, `algos[algo].toString().startsWith('function ' + algo + '(')`, et vérifie par espions que `fib(3, algo)` n'appelle que `algos[algo]` |
| `PERF-TS-11` | `cli.test.ts:220, 227` | Paresse de l'accesseur `get stdout()` sur `processIo` | Sous-processus avec `stdout` redirigé vers un fichier temporaire sur `--bogus 5` (code 2) : vérifie que `internal/fs/sync_write_stream` n'est pas chargé + témoin positif |

---

## 6. Constats d'audit : Forces, Risques et Lacunes connues

### 6.1 Forces majeures
1. **Calcul de grand terme le plus rapide du banc d'essai (`internal_ms = 2,9 ms` à $n = 1\,000\,000$)** : la combinaison des formules de Lucas dans `fast` (1 produit et 1 carré par bit, pas impair par simple décalage `>> 1n`, court-circuit du carré final sur un bit pair) et des multiplications Karatsuba/Toom-Cook de V8 place TypeScript en tête sur le temps de calcul pur à $N = 1\,000\,000$.
2. **Additionneur décimal base $10^4$ avec encodage ASCII intégré (`fibList`)** : maintient `--list 10000` en 10,4 ms de temps interne sans aucune conversion binaire $\to$ décimale.
3. **Couverture d'horloge et d'introspection Node.js exemplaire** : contrairement à Go et Rust où la valeur numérique de la durée `--time` n'est pas vérifiée par les tests de garde, TypeScript vérifie à la fois la soustraction exacte des lectures d'horloge en mémoire (`PERF-TS-8`), l'encadrement de la durée réelle en millisecondes par rapport au temps mural (`PERF-TS-9`), et la liste exacte des modules internes C++/JS chargés par Node.js (`process.moduleLoadList` dans `PERF-TS-9` et `PERF-TS-11`).

### 6.2 Observations, écarts documentaires et lacunes connues

| Sévérité | Catégorie | Description et localisation | Statut |
|---|---|---|---|
| **Mineure (Documentation)** | Décompte de lignes A3 (`PRD.md` / `README.md`) | `ts/src/fib.ts` (129 lignes) et `ts/src/cli.ts` (141 lignes) totalisent **270 lignes** (`wc -l`), alors que `PRD.md` (§ 7 A3, ligne 192) et `README.md` (§ Coût, ligne 66) affichent `268`. Cet écart de 2 lignes provient de l'ajout de deux lignes de commentaire dans `ts/src/fib.ts:117-120` lors du commit `adac0337` (#41). | Écart documentaire de 2 lignes ; mettre à jour `268` $\to$ `270` dans `PRD.md` et `README.md` si souhaité |
| **Lacune connue** | Câblage de `processIo.fib` (`cli.ts:103`) | `PERF-TS-9` n'exécute `cli.ts` en sous-processus qu'à $n = 10$ (ou avec `iterative` à $n = 100\,000$), où les trois algorithmes rendent `55`. Un mutant `processIo = { fib: (n) => fib(n, 'fast'), ... }` dans `cli.ts:103` laisserait `PERF-TS-8` (qui passe son propre `io.fib`) et `PERF-TS-9` verts. | Documenté au catalogue (`docs/cas-de-test.md` § 6) ; vérifié par lecture directe de `cli.ts:103` |
| **Lacune connue** | Délégation interne dans un algorithme (`fib.ts:14-44`) | `PERF-TS-10` vérifie que `algos[algo]` commence par `function <algo>(` et est appelée par `fib`, mais ne peut pas détecter si le corps de `function matrix(n: number)` déléguait en interne à `return fast(n)`. | Documenté au catalogue § 6 ; vérifié par lecture directe de `fib.ts:14-44` |
| **Lacune connue** | Point d'entrée `cli.ts:139-141` et étendue de la fenêtre `timed` | La garde `if (import.meta.main === undefined)` (pour Node < 22.18) et l'exclusion de `String(terms[0])` hors de la fenêtre `timed` (`cli.ts:91-95`) ne sont pas sollicitées par un mutant d'horloge dans `String(...)`. | Documenté dans `cli.test.ts:57-58` et au catalogue § 6 ; vérifié par lecture de `cli.ts:91-95, 139-141` |

---

## 7. Suivi du 2026-10-14

Les §§ 1 à 6 décrivent l'état du commit a92a808, tel que l'audit l'a lu; ils ne sont pas réécrits. Ce qui a changé depuis, et ce qu'ils affirmaient à tort, est ici.

Cet audit a été vérifié constat par constat, sous Linux, avec Node 24.20.0 et TypeScript 7.0.2. Trois agents adverses (exactitude, tri, critique) ont lu le code, recompté et joué des mutants dans des copies. Les constats retenus ont été implémentés par `fib-dev`, puis revus par un vérificateur par branche et par `/code-review` ([agbruneau/Fib#45](https://github.com/agbruneau/Fib/pull/45)). En production, seul le commentaire de l'export `algos` change, à nombre de lignes constant.

### 7.1 Suite donnée aux constats

| Constat | Suite |
|---|---|
| Décompte A3 : 270 lignes, et non 268 (§ 6.2) | Corrigé au PRD § 7 et au README § Coût, avec le recomptage des quatre langages. |
| Câblage de `processIo.fib` (§ 6.2) | Fermé par PERF-TS-12. `run(args)` sans Io, donc sous `processIo`, est joué à n = 10 et à n = 1 000 000. Des espions qui ne calculent pas sont posés dans la table `algos`, et les flux du processus sont captés le temps de l'appel. Chaque ligne de durée doit avoir appelé l'entrée qu'elle nomme. Le mutant du catalogue, `fib: (n, algo) => fib(n, algo === 'matrix' ? 'fast' : algo)`, rougit. Le mutant que cite l'audit rougissait déjà PERF-TS-9 (§ 7.2). Reste à la revue de code : un aiguillage sur des valeurs de n que la garde ne joue pas. |
| Délégation dans le corps d'un algorithme (§ 6.2) | Fermé par PERF-TS-10. Le texte brut de chaque entrée de `algos`, lu par `toString()`, commentaires compris, ne nomme aucun autre algorithme, ni `fib`, `fibList`, `algos` ou `this`. Un retrait des commentaires par regex, essayé d'abord, laissait passer des littéraux et des fins de ligne que chaque revue trouvait : un commentaire qui nommerait un autre algorithme rougit désormais, bruyamment, et se récrit autrement. Les espions de PERF-TS-10 jouent aussi n = 1 000 000. Restent à la revue : une délégation par une fonction auxiliaire ou par un nom calculé. |
| Point d'entrée `import.meta.main` et fenêtre de `timed` (§ 6.2) | La fenêtre est fermée par PERF-TS-8 : sous une horloge factice, la conversion décimale du terme et l'écriture de stdout n'entrent pas dans la durée. La garde `import.meta.main === undefined` reste sans test, aucun Node pris en charge ne l'atteignant. Elle est désormais nommée au catalogue § 6. |

### 7.2 Errata

Affirmations de cet audit que la vérification a trouvées inexactes :

- § 6.2, câblage de `processIo.fib` : le mutant cité, `fib: (n) => fib(n, 'fast')`, fait rougir PERF-TS-9 (vérifié trois fois) : sous lui, `--time --algo iterative 100000` rapporte moins du vingtième du temps mural. Cette borne basse est lâche : la détection est habituelle, non garantie. Les trois algorithmes concordent à tout n, et non seulement à 10. La lacune du catalogue était `matrix` seul, remplacé par `fast`.
- § 1.1 : HAR-6 ne garde pas A2. Il garde que la boucle différentielle reste hors du contrat. Aucun test ne garde A2, que seule la lecture des imports assure.
- § 3.3 : le fichier `docs/adr/0003-formules-arithmetiques-par-langage.md` n'existe pas. L'ADR-0003 porte sur la taille de Go. Le choix des nombres de Lucas est écrit à l'ADR-0004, au PRD § 3 et au README.
- § 1.2 : la compilation par `tsc` vient de l'ADR-0004 et du PRD § 5, et non de l'ADR-0002.
- § 3.4 :
  - le « + 1 » de `size` n'est pas « indispensable » jusqu'à n = 10 000, et PERF-TS-2 ne le garde pas : seul PERF-TS-5 rougit, sous `Math.round` ;
  - V8 convertit par diviser pour régner, en temps sous-quadratique, et non « cubique sur la liste ».
- § 3.2 : un bit à 1 coûte une seule addition (`a + b`). Les deux autres coefficients sont des copies.
- En-tête :
  - `node --test` compte 61 tests de premier niveau et 6 003 sous-tests, et non « 41 tests et 6 064 sous-tests et assertions » ;
  - le PRD § 5 exige Node `^22.18` ou `>=24.2` et TypeScript 7, et non Node 24 LTS et TypeScript 6 ;
  - le paquet installé est `typescript` 7.0.2, sans `@typescript/native-preview`.
- § 1.1 : `tsconfig.build.json` compte 11 lignes et 245 octets.
- § 4.2 : `Number` ne rend `Infinity` qu'au-delà d'environ 1,8 × 10^308, et non dès 10^308.
- § 2.1 : les espions de PERF-TS-10 sont des tests de premier niveau, et non des sous-tests.
- § 6.1 : « le calcul pur le plus rapide des quatre » n'est pas reproduit sous Linux. Sur 7 lancements de `--time 1000000`, les médianes donnent Go 5,7 ms, Rust 8,1 ms, TypeScript 8,6 ms et Python 31,6 ms.
