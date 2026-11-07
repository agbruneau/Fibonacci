# Audit exhaustif du répertoire `rust/`

**Date de l'audit** : 2026-10-14  
**Répertoire audité** : `rust/` (identifiant du contrat : `rust`)  
**Environnement d'exécution vérifié** : Windows 11 (`x86_64-pc-windows-gnu`), `rustc 1.98.0 (88d9e12ae 2026-08-18)`, `cargo 1.98.0`, édition Rust 2024  
**Statut global** : **Conforme (Vert)** — `cargo clippy --manifest-path rust/Cargo.toml --all-targets -- -D warnings`, `cargo test --manifest-path rust/Cargo.toml` (26 tests répartis en 4 cibles) et le contrat CLI (`node contracts/cli.test.ts rust`, 114 passés, 2 sautés sous Windows pour `CLI-V04`) passent sans aucun écart.

---

## 1. Vue d'ensemble et métriques du répertoire

### 1.1 Inventaire des fichiers et volumétrie

Le crate `fib` (édition 2024) sépare la bibliothèque (`src/lib.rs`) du binaire CLI (`src/main.rs`), et répartit ses tests entre des modules unitaires internes (`#[cfg(test)]` dans `src/lib.rs` et `src/main.rs`) et deux crates de tests d'intégration (`tests/bib.rs` et `tests/alloc.rs`). Aucun fichier ne contient de caractère `\r` (0 octet CR).

| Fichier | Rôle | Octets | Lignes (`wc -l`) | Détail (Production / Tests) |
|---|---|---:|---:|---|
| `rust/Cargo.toml` | Manifeste du crate (`edition = "2024"`, `num-bigint = "0.5"`, profil `release` avec `lto = true`, `codegen-units = 1`) | 303 | 13 | Configuration |
| `rust/Cargo.lock` | Verrouillage exact des dépendances (`num-bigint` 0.5.1, `num-integer` 0.1.47, `num-traits` 0.2.19, `autocfg` 1.5.1) | 1 072 | 44 | Configuration |
| `rust/rustfmt.toml` | Formatage (`max_width = 120`, `use_small_heuristics = "Max"`) | 45 | 2 | Configuration |
| `rust/src/lib.rs` | Bibliothèque (`Algo`, `UnknownAlgo`, `fib`, `fib_list`, `iterative`, `matrix`, `fast`, additionneur décimal) + 5 gardes unitaires | 10 121 | 250 | 190 prod + 60 tests (`PERF-RS-4` à `7`, `13`) |
| `rust/src/main.rs` | Adaptateur CLI (`parse`, `timed`, `compute`, `run`, `write_lines`, `main`) + 3 gardes unitaires | 11 873 | 251 | 149 prod + 102 tests (`PERF-RS-8`, `11`, `12`) |
| `rust/tests/bib.rs` | Tests d'intégration de la seam 1 (`BIB-01` à `BIB-13`) et gardes algorithmiques (`PERF-RS-1` à `3`) | 4 946 | 155 | 155 tests |
| `rust/tests/alloc.rs` | Crate de test isolé avec allocateur global de comptage (`PERF-RS-9`, `PERF-RS-10`) | 3 185 | 80 | 80 tests |
| **Total sources (`src/`)** | `src/lib.rs` + `src/main.rs` | **21 994** | **501** | **339 lignes de prod + 162 lignes de tests** |
| **Total `tests/`** | `tests/bib.rs` + `tests/alloc.rs` | **8 131** | **235** | **235 lignes de tests d'intégration** |

- **Conformité A2 (Dépendances)** : `Cargo.lock` ne contient que le crate autorisé `num-bigint 0.5.1` et ses trois dépendances transitives autorisées au PRD § 5 (`num-integer 0.1.47`, `num-traits 0.2.19`, `autocfg 1.5.1`).
- **Conformité A3 (Taille)** : le décompte de 501 lignes (`wc -l`) pour `src/lib.rs` (250) et `src/main.rs` (251), dont 60 et 102 lignes de tests unitaires, correspond à l'unité près aux chiffres inscrits dans `PRD.md` (§ 7, A3) et `README.md` (§ Coût).

### 1.2 Chaîne de compilation, profils et environnement Windows

- **Analyse statique** : `cargo clippy --manifest-path rust/Cargo.toml --all-targets -- -D warnings` — 0 avertissement sur l'ensemble des cibles (bibliothèque, binaire, tests unitaires et tests d'intégration).
- **Optimisations du profil `release` (`Cargo.toml:11-13`)** : `lto = true` et `codegen-units = 1` permettent l'optimisation inter-crates avec `num-bigint` (gain mesuré de 15 % au démarrage et sur `matrix`, ADR-0004).
- **Particularité de la toolchain sous Windows** : sur une machine Windows dépourvue des Build Tools C++ de Visual Studio (`link.exe`), la cible par défaut `x86_64-pc-windows-msvc` échoue à l'édition de liens. Conformément au PRD § 5 et § 10, l'override locale `rustup override set stable-x86_64-pc-windows-gnu` lie le binaire sans prérequis externe. *Note opérationnelle* : comme `rustup` associe son override au chemin absolu du dossier sur le disque, tout déplacement ou renommage du répertoire racine du dépôt (ex. depuis `OneDrive\Documents\GitHub\Fib` vers `Github\Fib`) nécessite de réappliquer `rustup override set stable-x86_64-pc-windows-gnu` dans le nouveau chemin.

---

## 2. Architecture et seam de bibliothèque (`rust/src/lib.rs`)

### 2.1 Typage fort et contrat de la seam 1

Contrairement aux trois autres langages qui passent l'algorithme sous forme de chaîne et l'indice sous forme d'entier signé ou dynamique, l'implémentation Rust exploite son système de types pour rendre deux catégories d'erreurs irreprésentables dans `fib` et `fib_list` :

1. **Indice `u64` non signé** : `pub fn fib(n: u64, algo: Algo) -> BigUint` et `pub fn fib_list(n: u64) -> Vec<String>` rendent impossible le passage d'un indice négatif, flottant ou `NaN`. Le cas `BIB-14` est donc sans objet par construction (PRD § 7, `docs/cas-de-test.md` § 4).
2. **Énumération `Algo` et `FromStr`** (`lib.rs:5-34`) :
   - `pub enum Algo { Iterative, Matrix, Fast }` avec `Algo::ALL` et `Algo::name(self) -> &'static str`.
   - La conversion depuis une chaîne est exposée via `impl FromStr for Algo` (`Err = UnknownAlgo`), qui vérifie l'égalité exacte `a.name() == s` et rejette `"all"`, `"bogus"`, `""` et `"FAST"` (`BIB-13`).
   - En conséquence, `fib` et `fib_list` sont infaillibles et retournent directement `BigUint` et `Vec<String>` sans `Result`.

### 2.2 Séparation de l'aiguillage (`algorithm`, `lib.rs:37-49`)

```rust
pub fn fib(n: u64, algo: Algo) -> BigUint {
    algorithm(algo)(n)
}

fn algorithm(algo: Algo) -> fn(u64) -> BigUint {
    match algo {
        Algo::Iterative => iterative,
        Algo::Matrix => matrix,
        Algo::Fast => fast,
    }
}
```

- Cette séparation permet au test unitaire `PERF-RS-13` de vérifier par égalité d'adresses de fonctions (`std::ptr::fn_addr_eq(algorithm(algo), own)`) que chaque variante de `Algo` pointe vers sa propre fonction privée (`iterative`, `matrix`, `fast`), et que ces trois fonctions ont des adresses distinctes deux à deux.
- Comme documenté dans `docs/exploration.md` (#40), ce retour de pointeur de fonction `fn(u64) -> BigUint` remplace un appel direct par un appel indirect dans `fib`, dont l'impact sur le temps de réponse a été mesuré nul au bruit près par l'A/B apparié (200 paires).

---

## 3. Audit algorithmique et arithmétique en précision arbitraire

### 3.1 Algorithme itératif (`iterative`, `lib.rs:121-128`)

```rust
fn iterative(n: u64) -> BigUint {
    let (mut a, mut b) = (BigUint::ZERO, BigUint::ONE);
    for _ in 0..n {
        a += &b;
        std::mem::swap(&mut a, &mut b);
    }
    a
}
```

- **Addition sur place sans clonage** : `a += &b` réutilise le vecteur de limbes internes du `BigUint` `a`, suivi d'un échange de pointeurs `std::mem::swap(&mut a, &mut b)` en $O(1)$.
- **Comptage exact des allocations (`PERF-RS-9`)** : dans `tests/alloc.rs`, un allocateur global `Counting` (déléguant à `std::alloc::System` avec un compteur `thread_local!`) vérifie que `fib(20_000, Algo::Iterative)` effectue au plus 32 allocations (16 observées, dues uniquement aux doublements de capacité des deux `BigUint`, contre $20\,123$ allocations si l'on écrivait `a = &a + &b`).

### 3.2 Algorithme matriciel (`matrix`, `lib.rs:131-153`)

- **Structure `QPower { p, q, r }`** : représente la matrice symétrique $Q^k = \begin{pmatrix} F(k+1) & F(k) \\ F(k) & F(k-1) \end{pmatrix}$ initialisée à $Q^0 = \begin{pmatrix} 1 & 0 \\ 0 & 1 \end{pmatrix}$ (`bits = u64::BITS - n.leading_zeros()`, qui vaut `0` pour `n = 0`).
- **Boucle sur les bits `for i in (1..bits).rev()`** :
  - Calcule trois carrés `(&m.p * &m.p, &m.q * &m.q, &m.r * &m.r)`, puis `p = p2 + &q2`, `r = q2 + r2` et `q = &p - &r`.
  - En arithmétique non signée (`BigUint`), la soustraction `&p - &r` est garantie sans sous-dépassement car $p' - r' = (p^2 + q^2) - (q^2 + r^2) = p^2 - r^2 = F(k+1)^2 - F(k-1)^2 = F(2k) \ge 0$ pour tout $k \ge 0$.
- **Dernier bit (`lib.rs:152`)** :
  - Si `n & 1 == 0`, retourne `&m.q * (&m.p + &m.r)` (pour $n = 0$, $q = 0$ donne $0$).
  - Si `n & 1 == 1`, retourne `&m.p * &m.p + &m.q * &m.q` (pour $n = 1$, la boucle `1..1` est vide et $1^2 + 0^2 = 1$).
- **Garde `PERF-RS-2`** : vérifie que `matrix` concorde avec `iterative` sur toutes les puissances de 2 jusqu'à $2^{14}$ et leurs voisins ($2^j - 1, 2^j, 2^j + 1$).

### 3.3 Algorithme de doublement rapide (`fast`, `lib.rs:156-189`)

- **Gestion de l'arithmétique non signée (`BigUint`)** : dans la formule de GMP $F(2k+1) = 4F(k)^2 - F(k-1)^2 + 2(-1)^k$, l'application de $+2(-1)^k$ est isolée dans `add_alternating_two(&mut odd, k_odd)` après le calcul de `(a2 << 2u32) - &b2`. Comme $k \ge 1$, on a toujours $4F(k)^2 \ge 4 > F(k-1)^2 + 2$, donc ni `(a2 << 2u32) - &b2` ni `*x -= 2u8` ne peuvent sous-dépasser dans `BigUint`.
- **Factorisation du dernier bit en une seule multiplication (`lib.rs:180-189`)** :
  - Pour $n$ pair : $F(2k) = F(k)(F(k) + 2F(k-1))$ via `&a * (&a + (b << 1u32))`.
  - Pour $n$ impair : $F(2k+1) = 4F(k)^2 - F(k-1)^2 + 2(-1)^k = (2F(k) - F(k-1))(2F(k) + F(k-1)) + 2(-1)^k$ via `(&a2 - &b) * (a2 + &b)` suivi de `add_alternating_two`. Cette identité économise une multiplication de grands entiers au dernier pas impair (1 produit au lieu de 2 carrés).
- **Garde `PERF-RS-3`** : éprouve `fast` autour des puissances de 2 jusqu'à $2^{14}$, couvrant les deux parités de $k$ et de $n$.

### 3.4 Additionneur décimal en base $10^{18}$ (`fib_list`, `add`, `put_limb`, `decimal`, `lib.rs:51-118`)

- **Table `QUADS` évaluée à la compilation (`static QUADS: [[u8; 4]; 10_000]`)** : construite dans un bloc `const` par une boucle `while`, elle est intégrée directement dans le segment `.rodata` du binaire compilé, sans aucun coût d'initialisation à l'exécution.
- **Addition sans branche (`add`, `lib.rs:69-80`)** :
  ```rust
  let sum = *x + y + carry;
  carry = u64::from(sum >= BASE);
  *x = sum - carry * BASE;
  ```
  Évite tout saut conditionnel sur la retenue interne (imprévisible par le prédicteur de branche) ; gardé par les 4 tests unitaires `PERF-RS-4` à `PERF-RS-7`.
- **Formatage décimal sans réallocation (`decimal`, `lib.rs:105-118`)** : calcule la longueur exacte du limbe de tête par `top.checked_ilog10().map_or(1, |d| d as usize + 1)`, alloue `out = vec![0; top_len + rest.len() * LIMB_DIGITS]`, remplit les tranches par `put_limb` et convertit en `String::from_utf8(out)`.
- **Garde d'allocation `PERF-RS-10`** : vérifie que `fib_list(10_000)` n'alloue que les $10\,001$ chaînes et la croissance logarithmique des vecteurs `terms`, `a` et `b` (28 allocations supplémentaires observées, plafond `LIST_SLACK = 64`).

---

## 4. Audit de l'adaptateur CLI (`rust/src/main.rs`) et conformité au contrat

### 4.1 Conformité à la grammaire et aux exigences (PRD § 4)

- **Lecture robuste des arguments (`main.rs:135`)** : utilise `env::args_os().skip(1).map(|a| a.to_string_lossy().into_owned()).collect()` au lieu de `env::args()`, qui paniquerait sur un argument non UTF-8 (piège vérifié du PRD § 5).
- **Règle 1 (`--help` / `-h`)** : vérifiée en tête de `run` (`main.rs:109-112`). `USAGE.trim_end()` retire le `\n` final parce que `write_lines` ajoute `b"\n"` après chaque élément de `lines`.
- **Règles 2 à 6 (`parse`, `main.rs:34-81`)** :
  - Utilise un unique itérateur `let mut it = args.iter()` partagé entre `while let Some(a) = it.next()` et la lecture de la valeur séparée de `--algo` (`it.next()`).
  - Chaque occurrence de `--algo` / `--algo=` est validée immédiatement via `"all"` ou `value.parse::<Algo>()` ; en cas d'erreur, l'argument fautif est cité en ASCII grâce à `value.escape_default()`.
  - Les options inconnues sont citées de même avec `a.escape_default()` (`CLI-A18`, `CLI-A20`, `CLI-U04`, `CLI-U05`).
  - L'indice est validé par `!index.is_empty() && index.bytes().all(|b| b.is_ascii_digit())` avant `index.parse().unwrap_or(u64::MAX)`, ce qui transforme tout débordement de `u64` (`CLI-I14`) en dépassement de borne (`n > bound`, code 2) sans panique.
- **Comparaison, discordance et durées (`timed`, `compute`, `main.rs:84-104`)** :
  - `COMPARISON = [Algo::Fast, Algo::Iterative, Algo::Matrix]`.
  - `timed` mesure `Instant::now()` et `start.elapsed().as_secs_f64() * 1000.0` uniquement autour du calcul, puis formate `{name} {ms:.3} ms\n` (en Rust, `format!` utilise toujours le point décimal, quelle que soit la locale POSIX/Windows, `CLI-V01`).
  - Si `terms.iter().any(|t| *t != terms[0])`, `compute` renvoie `Err(CliError(3, ...))` : comme `run` propage l'erreur par `?`, les durées accumulées sont abandonnées et `main` n'écrit que `fib: {msg}` sur `stderr` et sort avec `ExitCode::from(3)` (`F3`).
- **Écriture tamponnée et gestion des flux (`write_lines`, `main`, `main.rs:120-148`)** :
  - `BufWriter::with_capacity(1 << 16, std::io::stdout().lock())` tamponne les écritures par blocs de 64 Kio (`PERF-RS-8`).
  - `let _ = out.write_all(...)` et `let _ = writeln!(std::io::stderr(), ...)` ignorent les erreurs d'E/S au lieu d'utiliser `println!`/`eprintln!` (qui paniqueraient sur un flux fermé), garantissant le respect de `CLI-V04` (`stderr` fermé) et l'absence de panique sur un tube fermé.
  - `std::mem::forget(lines)` (`main.rs:140`) évite de désallouer une à une les $10\,001$ `String` juste avant la terminaison du processus, économisant ~2 ms sur `--list 10000` (ADR-0004).

---

## 5. Audit de la suite de tests et couverture des gardes

### 5.1 Organisation des 26 tests Rust

1. **`tests/bib.rs` (16 tests)** :
   - `BIB-01` à `BIB-13` (`BIB-14` exclu par le type `u64`), avec lecture et validation stricte de `../contracts/b000045.txt` (lignes vides filtrées par `.filter(|l| !l.is_empty())`, indice et 2 001 termes vérifiés).
   - `PERF-RS-1` (`fib_list(5000)` comparé terme à terme à `fib(k, Algo::Fast).to_string()`), `PERF-RS-2` (`matrix` autour de $2^1 \dots 2^{14}$) et `PERF-RS-3` (`fast` autour de $2^1 \dots 2^{14}$).
2. **`tests/alloc.rs` (2 tests)** :
   - `PERF-RS-9` (`fib(20_000, Algo::Iterative)` $\le 32$ allocations) et `PERF-RS-10` (`fib_list(10_000)` $\le 10\,065$ allocations), isolés dans leur propre binaire de test pour que `#[global_allocator]` n'interfère pas avec les autres tests.
3. **`src/lib.rs::tests` (5 tests)** :
   - `PERF-RS-4` à `PERF-RS-7` : tests unitaires de `add(&mut a, &b)` aux frontières exactes de $10^{18}$ (somme égale à `BASE`, propagation à travers des limbes pleins `BASE - 1`, création d'un nouveau limbe, conservation du reste `BASE - 2`).
   - `PERF-RS-13` : vérification par `std::ptr::fn_addr_eq` de l'aiguillage de `algorithm(algo)`.
4. **`src/main.rs::tests` (3 tests)** :
   - `PERF-RS-8` : vérification par un `Spy` implémentant `io::Write` que `write_lines` sur `fib_list(10_000)` effectue au plus $\lceil \text{total} / 32\text{ Kio} \rceil$ appels à `write` sans perdre un octet.
   - `PERF-RS-11` : vérification du code 3 et de l'absence de `\n` et de ` ms` dans le message d'erreur lorsque l'un des trois algorithmes discorde dans `run`.
   - `PERF-RS-12` : vérification par espion (`RefCell<Vec<Algo>>`) que les algorithmes exécutés correspondent exactement aux lignes de durée retournées par `run`.

---

## 6. Constats d'audit : Forces, Risques et Lacunes connues

### 6.1 Forces majeures
1. **Temps de réponse au plancher matériel** : sur la machine d'audit, Rust affiche le temps total le plus bas des quatre langages sur les scénarios courts (`petit (10)` en 7,8 ms, `terme 100000` en 8,0 ms, `matrix 100000` en 8,2 ms).
2. **Sûreté par le typage et `clippy -D warnings`** : l'utilisation de `u64`, de l'énumération `Algo`, de `args_os` et d'un `Result<(Vec<String>, String), CliError>` pur dans `run` élimine à la compilation des classes entières de défauts (accès hors bornes, algorithme invalide dans la bibliothèque, écriture partielle sur `stdout` avant détection d'une discordance).
3. **Couverture chirurgicale de l'additionneur décimal** : là où Go n'a qu'un test de retenue interne (`PERF-GO-5`), Rust teste quatre configurations limites de retenue sur les limbes $10^{18}$ (`PERF-RS-4` à `PERF-RS-7`).

### 6.2 Observations, pistes d'optimisation et lacunes connues

| Sévérité | Catégorie | Description et localisation | Statut |
|---|---|---|---|
| **Observation** | Allocations dans `fib_list` (`src/lib.rs:57-66`) | Dans `fib_list`, `a`, `b` et `terms` sont initialisés sans `Vec::with_capacity(...)` (`vec![0]`, `vec![1]`, `vec![decimal(&a)]`), ce qui provoque 28 réallocations de croissance (observées dans `alloc.rs:55`) et 1 allocation de `String` par terme ($10\,001$ allocations). Le temps reste excellent (6,3 ms de calcul pour `--list 10000`), mais `Vec::with_capacity(n as usize + 1)` supprimerait les réallocations du vecteur de sortie. | Choix de simplicité ; couvert par `PERF-RS-10` |
| **Observation** | Environnement Windows (`rustup override`) | La toolchain par défaut sous Windows (`msvc`) échoue sans `link.exe` (Visual Studio C++ Build Tools). L'override `rustup override set stable-x86_64-pc-windows-gnu` dépend du chemin absolu du répertoire et doit être réappliquée si le dépôt est déplacé. Un fichier `rust-toolchain.toml` n'est pas utilisé car il imposerait la cible `windows-gnu` sur Linux/macOS. | Documenté au PRD § 5 et au `README.md` |
| **Lacune connue** | `std::mem::forget(lines)` (`src/main.rs:140`) | L'optimisation `std::mem::forget(lines)` dans `main()` n'est couverte par aucun test de garde (`main` ne peut pas être appelée par un test unitaire, et sa suppression laisse `cargo test` et le contrat verts). | Documenté au catalogue (`docs/cas-de-test.md` § 6) ; gardé par la revue de code |
| **Lacune connue** | Mesure réelle de la durée (`F6`) et câblage de `main()` | Comme en Go, `PERF-RS-12` vérifie les noms d'algorithmes mais pas la valeur numérique issue de `Instant::now()`. De même, dans `main()` (`src/main.rs:136-147`), le passage de `fib` à `run(&args, fib)`, l'absence d'`eprint!` direct dans `compute` et le passage de `code` sans plafonnement à `ExitCode::from(code)` reposent sur l'inspection de `main.rs`. | Documenté dans `src/main.rs:205-206` et au catalogue § 6 ; vérifié par lecture directe |

---

## 7. Suivi du 2026-10-14

Les §§ 1 à 6 décrivent l'état du commit a92a808, tel que l'audit l'a lu; ils ne sont pas réécrits. Ce qui a changé depuis, et ce qu'ils affirmaient à tort, est ici.

Cet audit a été vérifié constat par constat, sous Linux, avec rustc 1.97. Trois agents adverses (exactitude, tri, critique) ont lu le code, compté les allocations, lu le code machine et joué des mutants dans des copies. Les constats retenus ont été implémentés par `fib-dev`, puis revus par un vérificateur par branche et par `/code-review` ([agbruneau/Fib#45](https://github.com/agbruneau/Fib/pull/45)). Le code de production de Rust ne change pas : seules des gardes s'ajoutent, dans le module de tests de `src/main.rs`.

### 7.1 Suite donnée aux constats

| Constat | Suite |
|---|---|
| `Vec::with_capacity` dans `fib_list` (§ 6.2) | Non fait. Il retire 13 réallocations sur 10 029 allocations. Un A/B apparié dans le processus (1 000 paires, trois passes) donne environ −0,01 ms sur 6,3 à 7 ms, soit −0,15 %, et le gain n'est plus rapide que dans 51 à 52 % des paires : c'est du bruit. La seule garde possible resserrerait la marge de PERF-RS-10, que les changements de croissance de `Vec` de la chaîne d'outils rendraient fragile. |
| Override `rustup` liée au chemin (§ 6.2) | Documenté au PRD § 5 et au README § Prérequis. L'override se refait après un déplacement du dépôt ou dans un nouveau clone, et `rustup override unset --nonexistent` retire l'ancienne. C'est vérifié sous Linux. |
| `std::mem::forget(lines)` sans garde (§ 6.2) | Laissé à la revue de code, comme le prévoit le catalogue § 6. Une garde exigerait d'extraire de `main` une fonction de production, et un allocateur compteur dans les tests de `main.rs`, pour un gain mesuré ici de 0,4 à 0,9 ms. |
| Mesure réelle de la durée, F6, et câblage de `main` (§ 6.2) | La durée est gardée par PERF-RS-14. Un espion dort 20 ms, tour à tour pour chaque algorithme de `--algo all`, et chronomètre son sommeil par `Instant`. La ligne de l'algorithme lent vaut au moins ce sommeil, et les trois lignes au plus le temps mural de `run`. La durée de `--list` va du vingtième au tout du temps mural de `run` : le plafond est exact, le plancher dépend de la charge. Restent à la revue : l'étendue exacte de la fenêtre, une durée qui perdrait ses secondes entières (`subsec_nanos`), et le câblage de `main` (le calcul passé à `run`, le code de sortie). |

Ajout de parité, hors audit : PERF-RS-12 joue aussi n = 1 000 000, avec un espion qui ne calcule pas, comme PERF-GO-10, PERF-TS-12 et PERF-PY-27. Un algorithme qui change selon n dans `compute` rougit.

### 7.2 Errata

Affirmations de cet audit que la vérification a trouvées inexactes :

- § 3.4 : PERF-RS-4 à 7 ne gardent pas l'addition sans branche de `add`. Ils en vérifient les valeurs, non la forme : une forme à branche les laisse verts. Aucune garde n'est utile, car `rustc -O` compile les deux formes en `cmov`. Le catalogue § 6 le note désormais.
- § 3.3 : l'inégalité « 4F(k)² ≥ 4 > F(k−1)² + 2 » est fausse dès k = 4. La conclusion tient pour une autre raison : F(k) ≥ F(k−1), donc 4F(k)² − F(k−1)² ≥ 3F(k)² ≥ 3 > 2.
- § 4.1 :
  - CLI-V04 ne dépend pas de `let _ =` : sous POSIX, le runtime de Rust rouvre `/dev/null` sur un fd 0 à 2 fermé au lancement ;
  - CLI-A18 et CLI-A20 citent une valeur de `--algo` (`value.escape_default()`), non une option inconnue ;
  - l'expression `BufWriter::with_capacity(1 << 16, std::io::stdout().lock())` n'existe pas telle quelle (`STDOUT_BUFFER`, et le verrou est pris dans `main`).
- § 6.1, force 2 : rien n'empêche à la compilation une écriture partielle depuis `compute`. Le catalogue § 6 la laisse à la revue de code.
- § 6.2, ligne 1 : les 28 allocations en sus des 10 001 chaînes sont 3 allocations initiales et 25 réallocations, et non « 28 réallocations ».
- § 6.2, ligne 3 : `main` peut être appelée par un test. Elle y lirait les arguments du harnais, ce qui la rend inutile comme garde.
- § 1.2 : le gain de 15 % de `lto` et `codegen-units` vient du commentaire de `Cargo.toml`, et non de l'ADR-0004.
- § 3.4 : `QUADS` est l'initialiseur d'un `static`, évalué à la compilation, et non un bloc `const`.

Les temps de la machine Windows (§ 6.1) n'ont pas pu être revérifiés.
