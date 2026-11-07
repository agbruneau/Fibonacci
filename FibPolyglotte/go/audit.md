# Audit exhaustif du répertoire `go/`

**Date de l'audit** : 2026-10-14  
**Répertoire audité** : `go/` (identifiant du contrat : `go`)  
**Environnement d'exécution vérifié** : Windows 11 (`windows/amd64`), `go version go1.27.0 windows/amd64` (minimum exigé par le PRD § 5 : Go 1.26)  
**Statut global** : **Conforme (Vert)** — `go vet ./...`, `go test ./...` (18 fonctions de test, 6 342 sous-tests) et le contrat CLI (`node contracts/cli.test.ts go`, 114 passés, 2 sautés sous Windows pour `CLI-V04`) passent sans aucun écart.

---

## 1. Vue d'ensemble et métriques du répertoire

### 1.1 Inventaire des fichiers et volumétrie

L'implémentation Go est structurée en un module racine `fib` exposant la bibliothèque (seam 1) et un sous-paquet exécutable `fib/cmd/fib` constituant l'adaptateur CLI (seam 2). Toutes les fins de ligne respectent la norme LF (`0` octet CR), conformément à `.gitattributes` et à l'exigence F10.

| Fichier | Rôle | Octets | Lignes (`wc -l`) | Caractères CR |
|---|---|---:|---:|---:|
| `go/go.mod` | Déclaration du module `fib` (`go 1.26`, aucune dépendance externe) | 20 | 3 | 0 |
| `go/fib.go` | Bibliothèque : `Fib`, `FibList`, `iterative`, `matrix`, `fast`, additionneur décimal `addLimbs` / `appendDecimal` | 7 372 | 199 | 0 |
| `go/cmd/fib/main.go` | Adaptateur CLI : `parse`, `isDigits`, `lap`, `compute`, `run`, `main` | 5 081 | 186 | 0 |
| `go/fib_test.go` | Tests de la seam 1 (`BIB-01` à `BIB-14`) et gardes publiques (`PERF-GO-1` à `PERF-GO-4`) | 8 975 | 252 | 0 |
| `go/fib_internal_test.go` | Gardes internes du paquet `fib` (`PERF-GO-5`, `PERF-GO-11`) | 4 241 | 84 | 0 |
| `go/cmd/fib/main_test.go` | Gardes internes du paquet `main` (`PERF-GO-6` à `PERF-GO-10`) | 8 376 | 194 | 0 |
| **Total production** | `fib.go` + `cmd/fib/main.go` | **12 453** | **385** | **0** |
| **Total tests** | `fib_test.go` + `fib_internal_test.go` + `cmd/fib/main_test.go` | **21 592** | **530** | **0** |

- **Conformité A2 (Dépendances)** : `go/go.mod` ne contient aucune directive `require`, et aucun fichier `go.sum` n'est nécessaire. Seule la bibliothèque standard Go est utilisée (`math/big`, `math/bits`, `encoding/binary`, `bufio`, `slices`, `strconv`, `strings`, `sync`, `time`, `errors`, `fmt`, `io`, `os`).
- **Conformité A3 (Taille)** : le total de 385 lignes (`wc -l`) de code de production correspond exactement au chiffre consigné dans `PRD.md` (§ 7, A3) et `README.md` (§ Coût).

### 1.2 Chaîne de compilation et commandes de vérification

- **Analyse statique** : `go -C go vet ./...` — aucune alerte.
- **Tests unitaires et de garde** : `go -C go test ./...` — 18 fonctions `Test*` de premier niveau et 6 342 sous-tests `t.Run` exécutés en ~1,3 s.
- **Compilation pour le contrat et le banc d'essai (`contracts/lancement.ts`)** :
  ```bash
  go -C go build -o ../.build/fib.exe ./cmd/fib
  ```
  Ce choix pare le piège documenté au PRD § 5 : `go run` intercepte tout code de sortie non nul, écrit `exit status N` sur `stderr` et force le code de sortie à `1`, ce qui violerait F3 (code 3) et F9 (code 2). La commande de lancement est protégée contre toute régression par le contrôle du harnais `HAR-7`.

---

## 2. Architecture et seam de bibliothèque (`go/fib.go`)

### 2.1 Signatures exportées et contrat de la seam 1

Le paquet `fib` n'expose que deux fonctions publiques, conformément au PRD § 5 :

1. `func Fib(n int, algo string) (*big.Int, error)` (`fib.go:16-29`)
   - **Validation de l'indice** : rejette tout `n < 0` avec `fmt.Errorf("indice negatif : %d", n)` avant l'aiguillage. Le typage statique `int` exclut à la compilation les nombres flottants (`1.5`, `NaN`) et les booléens (`BIB-14`).
   - **Validation et aiguillage de l'algorithme** : `switch algo` aiguille `"iterative"`, `"matrix"` et `"fast"` vers leurs fonctions privées respectives, et rejette tout autre nom (dont `"all"`, `""`, `"FAST"`, `"bogus"`) par `fmt.Errorf("algorithme inconnu : %+q", algo)` (`BIB-13`).
2. `func FibList(n int) ([]string, error)` (`fib.go:42-73`)
   - **Validation de l'indice** : rejette `n < 0` avec la même erreur que `Fib`.
   - **Sortie** : retourne une tranche de `n + 1` chaînes décimales représentant $F(0) \dots F(n)$ calculées directement en base $10^{18}$.

### 2.2 Décisions d'initialisation et impact sur le démarrage

Conformément à l'ADR-0004, aucune initialisation coûteuse n'est exécutée au chargement du paquet `fib` :
- La table de conversion ASCII `quads [10_000]uint32` est déclarée en mémoire statique non initialisée (`var quads [10_000]uint32`), et son remplissage par `fillQuads()` est différé au premier appel de `FibList` via `quadsOnce.Do(fillQuads)` (`fib.go:46, 93-102`).
- Ainsi, un appel de la CLI pour un terme seul (`fib 10`, `fib 1000000`) ne paie ni l'initialisation de la table de $10\,000$ entrées ni aucune fonction `init()`. Cette propriété est vérifiée dynamiquement par `PERF-GO-7` sous `GODEBUG=inittrace=1`.

---

## 3. Audit algorithmique et arithmétique en précision arbitraire

### 3.1 Algorithme itératif (`iterative`, `fib.go:129-139`)

```go
words := n*7/10/bits.UintSize + 3
a := new(big.Int).SetBits(make([]big.Word, 0, words))
b := new(big.Int).SetBits(make([]big.Word, 0, words)).SetInt64(1)
for range n {
    a.Add(a, b)
    a, b = b, a
}
return a
```

- **Exactitude mathématique** : à l'entrée de l'itération $k \in \{0, \dots, n-1\}$, `a` vaut $F(k)$ et `b` vaut $F(k+1)$. `a.Add(a, b)` place $F(k+2)$ dans le tampon de `a`, puis `a, b = b, a` place $F(k+1)$ dans `a` et $F(k+2)$ dans `b`. Pour $n = 0$, la boucle `for range 0` ne s'exécute pas et `a` (de longueur 0 dans `math/big`, soit l'entier `0`) est retourné.
- **Pré-allocation sans réallocation** : $\log_2(\varphi) \approx 0{,}69424 < 0{,}7$. Le nombre de bits de $F(n)$ est majoré par $0{,}7n + 1$. En réservant `n*7/10/bits.UintSize + 3` mots machine (`big.Word`), les tranches sous-jacentes de `a` et `b` ne subissent aucune réallocation pendant les $n$ additions sur place.
- **Empreinte mémoire vérifiée** : `testing.AllocsPerRun` mesure exactement **4 allocations** pour `Fib(20000, "iterative")` (les deux `big.Int` et leurs deux tranches `[]big.Word`), gardées sans marge par `PERF-GO-3` (`maxAllocsIterative = 4`).

### 3.2 Algorithme matriciel (`matrix`, `fib.go:143-165`)

- **Représentation** : exploite la symétrie de $Q^k = \begin{pmatrix} 1 & 1 \\ 1 & 0 \end{pmatrix}^k = \begin{pmatrix} F(k+1) & F(k) \\ F(k) & F(k-1) \end{pmatrix}$ en ne maintenant que trois entiers $(a, b, c) = (F(k+1), F(k), F(k-1))$, initialisés à $Q^1 = (1, 1, 0)$ pour consommer d'emblée le bit de poids fort de $n \ge 2$.
- **Élévation au carré en trois multiplications** : au lieu de calculer $b' = b(a+c)$ par une multiplication générale, le code calcule les trois carrés `aa = a*a`, `bb = b*b`, `cc = c*c`, puis pose $a' = a^2 + b^2$, $c' = b^2 + c^2$ et déduit $b' = a' - c' = a^2 - c^2 = (a-c)(a+c) = b(a+c)$ par une simple soustraction, en réutilisant l'allocation de `bb` (`b = bb.Sub(a, c)`).
- **Produit par $Q$** : lorsque le bit courant vaut `1`, la multiplication par $Q$ se réduit à une seule addition `(a+b, a, b)`.
- **Optimisation de la dernière étape (`i == 0`)** : la boucle s'arrête à `i >= 1`. Pour le bit de poids faible (`n&1`), seul le terme $F(n)$ est calculé :
  - Si $n$ est pair (`n&1 == 0`), $F(n) = b(a+c)$ (1 addition et 1 multiplication).
  - Si $n$ est impair (`n&1 == 1`), $F(n) = a^2 + b^2$ (2 carrés et 1 addition).

### 3.3 Algorithme de doublement rapide (`fast`, `fib.go:169-199`)

- **Formules de GMP en deux carrés par bit** : maintient le couple $(a, c) = (F(k), F(k-1))$ depuis $k = 1$. À chaque bit intérieur `i` (de `bits.Len(uint(n)) - 2` jusqu'à `1`) :
  - Calcule `sa = a²` et `sc = c²`.
  - $F(2k-1) = a^2 + c^2$ (`odd`).
  - $F(2k+1) = 4a^2 - c^2 + 2(-1)^k$ (`mid`), où `4a²` est obtenu sur place par décalage `sa.Lsh(sa, 2)` et le signe $(-1)^k$ est déterminé sans variable d'état par le bit précédent de $n$ (`n>>(i+1)&1 == 0` indique que $k$ est pair).
  - $F(2k) = F(2k+1) - F(2k-1)$ (`even = sc.Sub(mid, odd)`, réutilisant `sc`).
- **Optimisation du dernier bit** : hors de la boucle, évite le calcul du voisin inutile :
  - Si $n$ est pair : $F(2k) = F(k)(F(k) + 2F(k-1))$ en 1 seule multiplication et 1 décalage (`Lsh(c, 1)`).
  - Si $n$ est impair : $F(2k+1) = F(k)^2 + (F(k) + F(k-1))^2$ en 2 carrés.
- **Note comparative inter-langages** : sur un dernier bit impair, Go calcule $F(k)^2 + (F(k)+F(k-1))^2$ (deux carrés), tandis que Rust et Python factorisent $(2F(k)-F(k-1))(2F(k)+F(k-1)) + 2(-1)^k$ (un seul produit). À $N = 1\,000\,000$ (qui est pair), la branche impaire finale n'est pas sollicitée ; pour un grand indice impair, la différence entre deux carrés et un produit dans `math/big` est inférieure à la milliseconde.

### 3.4 Additionneur décimal en base $10^{18}$ (`FibList`, `addLimbs`, `appendDecimal`, `fib.go:31-125`)

- **Représentation** : chaque terme est stocké en limbes `uint64` de poids faible d'abord en base $10^{18}$ (`baseDigits = 18`). Comme $2 \times (10^{18} - 1) + 1 = 2 \cdot 10^{18} - 1 < 2^{64} - 1 \approx 1{,}844 \cdot 10^{19}$, l'addition dans `addLimbs` ne peut jamais déborder d'un `uint64`.
- **Inlining vérifié** : l'extraction de `addLimbs` (pour permettre le test unitaire `PERF-GO-5` sur une retenue exacte à $10^{18}$, inatteignable par les seuls termes de Fibonacci sous $n \le 10\,000$) ne coûte aucun appel de fonction : `go -C go build -gcflags=-m .` confirme `./fib.go:77:6: can inline addLimbs` et `./fib.go:64:20: inlining call to addLimbs`.
- **Encodage ASCII par groupes de 4 chiffres (`appendDecimal`)** :
  - Le limbe de poids fort est formaté sans zéro de tête par `strconv.AppendUint`.
  - Les limbes suivants (exactement 18 chiffres chacun) sont écrits directement dans une tranche pré-agrandie d'un coup via `slices.Grow`, en convertissant la sous-tranche en pointeur de tableau de taille fixe `(*[18]byte)(dst[start:])` (ce qui élimine les contrôles de bornes à chaque accès) et en déposant 4 octets d'un coup par `binary.BigEndian.PutUint32(t[j:], quads[x-q*10_000])`.
- **Allocation unique de chaîne (`strings.Builder`)** : tous les termes $F(0) \dots F(n)$ sont concaténés dans un unique `strings.Builder` pré-alloué (`text.Grow(...)` pour ~2,1 Mo à $n = 10\,000$), puis découpés en sous-chaînes `all[start:ends[k]]` qui partagent la même mémoire sous-jacente sans copier ni allouer une chaîne par terme. `PERF-GO-4` vérifie que `FibList(5000)` n'effectue au total que **6 allocations** (`text`, `a`, `b`, `ends`, `digits`, `list`).

---

## 4. Audit de l'adaptateur CLI (`go/cmd/fib/main.go`) et conformité au contrat

### 4.1 Conformité à la grammaire (PRD § 4, règles 1 à 6)

| Règle | Implémentation dans `go/cmd/fib/main.go` | Lignes | Conformité |
|---|---|---|---|
| **Règle 1** (`--help` / `-h` prioritaire) | `if slices.Contains(args, "--help") \|\| slices.Contains(args, "-h")` exécuté dans `run` avant `parse(args)` | `155-158` | Exacte (`CLI-H01` à `H11`) |
| **Règle 2** (`--algo X` / `--algo=X`) | `strings.CutPrefix(a, "--algo=")` ; si absent, consomme `args[i+1]`. Vérifie chaque occurrence contre `"all"` ou `slices.Contains(comparison, value)` ; écrase `o.algos` (la dernière gagne) | `52-70` | Exacte (`CLI-A01` à `A20`, `D04`, `D05`) |
| **Règle 3** (`--list`, `--time`) | `o.list = true` et `o.time = true` idempotents | `71-74` | Exacte (`CLI-L08`, `D08`) |
| **Règle 4** (Options inconnues `-*`) | `case strings.HasPrefix(a, "-"):` placé après `--algo`, `--list`, `--time`, rejette `-`, `--`, `-1`, `--list=5`, `--time=1` | `75-76` | Exacte (`CLI-I11`, `L14`, `D10`, `U01` à `U06`) |
| **Règle 5** (Indice `N` unique `^[0-9]+$`) | `len(positional) != 1`, puis `!isDigits(positional[0])` par boucle sur les octets `'0'..'9'`, puis `strconv.Atoi` et comparaison à `bound` | `87-103` | Exacte (`CLI-T01` à `T14`, `I01` à `I17`) |
| **Règle 6** (`--list` exclut `--algo`) | `if o.list && o.algos != nil` vérifié avant l'affectation par défaut `o.algos = []string{"fast"}` | `81-86` | Exacte (`CLI-L10` à `L12`) |

### 4.2 Conformité aux exigences F1 à F10

- **F1, F2, F4, F7** : sorties et algorithme par défaut (`fast`) conformes à l'octet près.
- **F3 (Comparaison et discordance)** : `comparison = []string{"fast", "iterative", "matrix"}` respecte l'ordre alphabétique. `compute` accumule les termes et les durées en mémoire sans rien écrire sur les flux ; si `t.Cmp(terms[0]) != 0`, retourne une erreur enveloppant `errDiscordance`. Dans `run`, `errors.Is(err, errDiscordance)` retourne le code `3` après avoir écrit la seule ligne `fib: ...` sur `stderr`, laissant `stdout` vide et omettant les durées même avec `--time`.
- **F6 (Durées `--time`)** : `lap` mesure `time.Since(start)` autour du seul appel `term(o.n, algo)` ou `fib.FibList(o.n)` (avant `terms[0].String()` et avant l'écriture tamponnée). Le format `fmt.Sprintf("%s %.3f ms\n", ...)` utilise toujours le point décimal en Go, indépendamment de `LANG` / `LC_NUMERIC` (`CLI-V01`).
- **F8 (Bornes)** : `maxTerm = 1_000_000` et `maxList = 10_000`. Sur un indice géant (`99999999999999999999`, `CLI-I14`), `isDigits` passe mais `strconv.Atoi` renvoie `strconv.ErrRange`, qui est traitée comme dépassement de borne (`if err != nil || n > bound`), produisant le code `2` sans panique. Sur `0` répété 5 000 fois suivi de `5` (`CLI-T14`), `strconv.Atoi` ignore les zéros de tête et retourne `5` sans limite de longueur de chaîne.
- **F9, F10 (Erreurs et ASCII pur)** : les arguments invalides sont cités avec le verbe `%+q` (`fmt.Errorf("option inconnue : %+q", a)`), qui échappe en séquences ASCII (`\n`, `\x..`, `\u....`) tout caractère non imprimable ou non ASCII (`CLI-A18`, `CLI-A20`, `CLI-U04`, `CLI-U05`).
- **Écriture tamponnée** : `bufio.NewWriterSize(stdout, 64<<10)` regroupe les 10 001 lignes de `--list 10000` (~10,5 Mo) en blocs de 64 Kio, réduisant les appels système `Write` d'environ $20\,002$ à ~165 (`PERF-GO-6`).

---

## 5. Audit de la suite de tests et couverture des gardes

### 5.1 Cas de bibliothèque (`BIB-01` à `BIB-14`, `go/fib_test.go`)

- L'oracle `../contracts/b000045.txt` est lu une seule fois par `oracleText(t)` et `oracle(t)` avec vérification stricte de la présence de 2 001 lignes consécutives (`0` à `2000`) et décimales : un oracle absent ou tronqué fait échouer immédiatement le test (`t.Fatalf`).
- Les balayages `BIB-05`, `BIB-06` et `BIB-07` créent chacun 2 001 sous-tests paramétrés (`t.Run`), soit 6 003 sous-tests qui franchissent les frontières `int64` ($F(92)$) et `uint64` ($F(93)$, $F(94)$).
- `BIB-08` (concordance et 20 899 chiffres à $n = 100\,000$), `BIB-09` (identité de Cassini à $n = 100\,000$), `BIB-10` à `BIB-12` (`FibList`), `BIB-13` (`all`, `bogus`, `""`, `FAST`) et `BIB-14` (`-1` sur `Fib` et `FibList`) sont tous présents et conformes au catalogue.

### 5.2 Inventaire des 11 tests de garde (`PERF-GO-1` à `PERF-GO-11`)

| ID | Fichier | Cible protégée | Mécanisme de vérification |
|---|---|---|---|
| `PERF-GO-1` | `fib_test.go:177` | Exactitude de `FibList(10000)` au-delà de l'oracle | Compare chaque terme `FibList(10000)[k]` à `Fib(k, "fast").String()` pour $k \in [0, 10\,000]$ |
| `PERF-GO-2` | `fib_test.go:188` | Dimensionnement des limbes de `FibList` aux petits indices | Compare `FibList(n)[n]` à l'oracle pour chaque $n \in [0, 300]$ (301 sous-tests) |
| `PERF-GO-3` | `fib_test.go:209` | Addition sur place dans `iterative` | `testing.AllocsPerRun(20, ...)` sur `Fib(20000, "iterative")` $\le 4$ allocations |
| `PERF-GO-4` | `fib_test.go:218` | Addition sur place et tampon unique dans `FibList` | `testing.AllocsPerRun(20, ...)` sur `FibList(5000)` $\le 6$ allocations |
| `PERF-GO-5` | `fib_internal_test.go:12` | Retenue exacte à `s == base` ($10^{18}$) dans `addLimbs` | Teste `addLimbs([base/2, 3, 0], [base/2, 4]) == [0, 8, 0]` avec `carry == 0` (tue le mutant `s > base`) |
| `PERF-GO-6` | `cmd/fib/main_test.go:40` | Tampon `bufio.Writer` de 64 Kio dans `run` | `writeCounter` sur `run(["--list", "10000"])` : nombre d'appels `Write` $\le \lceil \text{size} / 32\text{ Kio} \rceil$ |
| `PERF-GO-7` | `cmd/fib/main_test.go:61` | Absence d'`init` au démarrage dans `fib` et `fib/cmd/fib` | Relance le binaire de test avec `-test.run=^$` et `GODEBUG=inittrace=1` ; vérifie son témoin (`init `) et l'absence de `init fib ` / `init fib/cmd/fib ` (saute proprement sous `-cover`) |
| `PERF-GO-8` | `cmd/fib/main_test.go:89` | Absence d'import de `regexp` dans la CLI | Exécute `go list -deps .`, vérifie le témoin `"fib"` et l'absence de `"regexp"` ou `"regexp/*"` |
| `PERF-GO-9` | `cmd/fib/main_test.go:120` | Exigence F3 : code 3 sur discordance, stdout vide, 1 seule ligne stderr sans durée | Injecte un `termFunc` faussant tour à tour `fast`, `iterative`, `matrix` de $+1$ sur `--time --algo all 10` |
| `PERF-GO-10` | `cmd/fib/main_test.go:158` | Exigences F2 et F6 : l'algorithme nommé par `--time` est celui appelé | Injecte un `termFunc` espion sur `10`, `iterative`, `matrix` et `all` |
| `PERF-GO-11` | `fib_internal_test.go:46` | Exigences F2 et F6 : aiguillage de `Fib` vers la bonne fonction privée | Compare l'empreinte d'allocations `AllocsPerRun` à $n = 1023$ de `Fib(1023, nom)` (4, 67 et 52 allocations) à celle de l'appel direct `iterative(1023)`, `matrix(1023)`, `fast(1023)`, après avoir vérifié que les trois empreintes sont deux à deux distinctes |

---

## 6. Constats d'audit : Forces, Risques et Lacunes connues

### 6.1 Forces majeures
1. **Performance et frugalité mémoire exemplaires** : Go est l'implémentation la plus rapide sur les calculs lourds (`terme 1000000` en 24,0 ms dont 3,2 ms de calcul sur cette machine ; `iterative 100000` en 25,5 ms dont 15,9 ms de calcul ; `liste 10000` en 25,7 ms dont 4,4 ms de calcul). L'allocation unique du texte de `FibList` via `strings.Builder.String()` + découpage de sous-chaînes est particulièrement élégante : contrairement à Rust qui alloue une `String` par terme ($10\,001$ allocations), Go n'alloue que 6 objets au total pour toute la liste.
2. **Zéro coût d'initialisation** : l'usage de `sync.Once` pour `fillQuads` et le passage de `termFunc` en paramètre de `run`/`compute` (au lieu d'une `map` globale) évitent tout code d'`init` au démarrage, prouvé par `PERF-GO-7`.
3. **Garde d'identité non intrusive (`PERF-GO-11`)** : contrairement à Rust (qui sépare `algorithm(algo)` et introduit un pointeur de fonction) et à TypeScript (qui exporte `algos`), Go garde son `switch algo` de production intact et distingue les trois algorithmes par leur signature exacte d'allocations à $n = 1023$ ($2^{10}-1$, tous bits à 1).

### 6.2 Observations et lacunes connues (survivants de mutation documentés)

| Sévérité | Catégorie | Description et localisation | Statut |
|---|---|---|---|
| **Observation** | Hors contrat (PRD § 4) | Dans `go/cmd/fib/main.go:156, 174-180`, les valeurs de retour d'erreur de `io.WriteString`, `out.WriteString`, `out.WriteByte` et `out.Flush()` sont ignorées. C'est intentionnel et conforme au PRD § 4 : un `stdout` ou `stderr` fermé ou en échec d'écriture est hors contrat, et ignorer l'erreur permet en outre de passer `CLI-V04` (`stderr` fermé au lancement) sans changer le code de sortie. | Conforme par conception |
| **Lacune connue** | Garde d'aiguillage (`PERF-GO-11`) | `PERF-GO-11` (`fib_internal_test.go:46`) n'échantillonne l'empreinte d'allocations qu'à $n = 1023$. Un aiguillage conditionnel dépendant de $n$ (ex. `if n > 10_000 { return fast(n), nil }` dans la branche `"matrix"`) ou un corps de `matrix` appelant `fast` tout en simulant 15 allocations factices lui échapperait. | Documenté au catalogue § 6 ; couvert par la revue de code |
| **Lacune connue** | Mesure réelle de la durée (`F6`) | Contrairement à `PERF-TS-8` et `PERF-TS-9` en TypeScript, `PERF-GO-10` vérifie le nom de l'algorithme chronométré mais ne vérifie pas que la durée affichée dans `<algo> <durée> ms` provient réellement de `time.Since(start)` (une constante `"0.000 ms"` passerait `PERF-GO-10` et le contrat). | Documenté au catalogue § 6 ; vérifié par lecture de `main.go:120-122` |
| **Lacune connue** | Point d'entrée `main()` | `main()` (`main.go:184-186`) contient `os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, fib.Fib))`. Comme `PERF-GO-9` et `PERF-GO-10` testent `run(...)` directement, un mutant dans `main()` qui bornerait le code de sortie à 2 ou passerait une fonction câblée sur `fast` à la place de `fib.Fib` survivrait aux tests unitaires (le second serait toutefois détecté sur les allocations ou durées si testé de bout en bout, mais pas par le contrat sur la valeur). | Documenté dans `main_test.go:117-119` ; vérifié par lecture directe de `main.go:185` |

---

## 7. Suivi du 2026-10-14

Les §§ 1 à 6 décrivent l'état du commit a92a808, tel que l'audit l'a lu; ils ne sont pas réécrits. Ce qui a changé depuis, et ce qu'ils affirmaient à tort, est ici.

Cet audit a été vérifié constat par constat, sous Linux, avec Go 1.26.0. Trois agents adverses (exactitude, tri, critique) ont lu le code, recompté et joué des mutants dans des copies. Les constats retenus ont été implémentés par `fib-dev`, puis revus par un vérificateur par branche et par `/code-review` ([agbruneau/Fib#45](https://github.com/agbruneau/Fib/pull/45)). Le code de production de Go ne change que par le dernier bit impair de `fast`, décidé ensuite par le propriétaire; le reste ne fait qu'ajouter des gardes.

### 7.1 Suite donnée aux constats

| Constat | Suite |
|---|---|
| Erreurs d'écriture ignorées (§ 6.2) | Conforme par conception, rien à faire. La justification de l'audit est fausse (§ 7.2). |
| PERF-GO-11 n'échantillonne que n = 1023 (§ 6.2) | Fermé pour le mutant de l'audit et pour la délégation de corps, par un volet statique de PERF-GO-11 qui lit `fib.go` par `go/parser`. Dans `Fib`, chaque nom d'algorithme n'apparaît que dans le retour de son propre cas, et `algo` et `n` ne sont ni réécrits, ni redéclarés, ni pris par adresse. Les corps de `iterative`, `matrix` et `fast` ne nomment ni les deux autres, ni `Fib`, ni `FibList`. Restent à la revue de code : une délégation par une variable de fonction ou par une fonction auxiliaire, et la copie en ligne du code de `fast`, à tout n si elles imitent l'empreinte d'allocations. |
| Mesure réelle de la durée, F6 (§ 6.2) | Fermé par PERF-GO-12. Un calcul espion dort 20 ms, tour à tour pour chaque algorithme de `--algo all`, et chronomètre son sommeil par `time.Since`. La ligne de l'algorithme lent vaut au moins ce sommeil, et les trois lignes au plus le temps mural de `run`. Restent à la revue : la fenêtre de `--list` (`FibList` n'est pas injectable) et l'étendue exacte de la fenêtre. |
| Point d'entrée `main()` (§ 6.2) | Laissé à la revue de code, comme le prévoit le catalogue § 6. Le code 3 est inatteignable avec le vrai `fib.Fib`, si bien que le borner à 2 est un mutant équivalent. Seule une garde du texte de `main` verrait le calcul passé à `run`. |
| Dernier bit impair de `fast` en deux carrés (§ 3.3) | Fait à la demande du propriétaire, comme en Rust et en Python : F(2k+1) = (2F(k) − F(k−1))(2F(k) + F(k−1)) + 2(−1)^k, en un produit. `go/fib.go` passe de 199 à 208 lignes. PERF-GO-13 garde la forme en lisant `fib.go` (un appel de `Mul` dans le bloc `if n&1 == 0`, un dans la branche impaire qui le suit, et aucun avant le test de parité : un total de deux laissait passer une `Mul` déplacée dans une aide, compensée par les deux carrés, ou un produit impair calculé avant le test), puisque l'empreinte d'allocations ne la distingue pas sans fragilité (51 contre 52 allocations à n = 1023). Les deux facteurs de la branche impaire se calculent sur place dans `c` et dans un entier temporaire : 9, 33, 51 et 111 allocations de `fast` à n = 5, 101, 1023 et 999 999 contre 10, 34, 52 et 112, et 428 319 octets contre 478 506 à n = 999 999 (non chronométré). La justesse reste gardée par BIB-05 à 07, BIB-09 et PERF-GO-1. Mesures, sous Linux, par le banc, qui gagne un scénario impair, terme `999999` (`contracts/bench.ts`). Banc ordinaire, la mesure d'un travail de calcul (ADR-0004), avant (7d126ea) et après, 31 exécutions, colonne calcul : 6,9 → 6,0 ms à 999 999, mais 6,5 → 5,9 ms aussi à 1 000 000, dont le code n'a pas changé ; l'écart impair − pair passe de +0,4 à +0,1 ms. La dérive de la machine entre deux passes masque donc la plus grande part du gain. L'A/B du banc (`--contre 7d126ea`, 100 paires) n'apparie que le temps total, et n'y distingue pas l'effet du bruit (−1,7 % à 999 999, −1,8 % au pair inchangé). En complément, hors banc, un A/B apparié des deux binaires sur `--time` donne −1,10 ms (−15,6 %) à 999 999, −9,6 % à 100 001 et −2,5 % à 1 000 000. Apparier aussi la durée de calcul dans l'A/B du banc rendrait cette mesure reproductible (suite proposée au propriétaire). |

Ajout de parité, hors audit : PERF-GO-10 joue aussi n = 1 000 000, avec un calcul espion qui ne calcule pas, comme PERF-TS-12, PERF-RS-12 et PERF-PY-27. Un nom qui change selon n dans `compute` rougit.

### 7.2 Errata

Affirmations de cet audit que la vérification a trouvées inexactes :

- § 6.2, PERF-GO-11 : le second échappement, un corps de `matrix` qui appelle `fast` et imite son empreinte par 15 allocations factices, ne figurait pas au catalogue § 6, malgré le statut « Documenté ». Il est désormais fermé par le volet statique.
- § 6.2, erreurs d'écriture :
  - un stderr fermé au lancement est dans le contrat (PRD § 4, CLI-V04), non hors contrat ;
  - sous POSIX, ce n'est pas l'erreur ignorée qui fait passer CLI-V04, mais le runtime de Go, qui rouvre `/dev/null` sur les fd 0 à 2 fermés au démarrage ;
  - la liste oublie `fmt.Fprintf(stderr, …)` (`main.go:165`).
- § 3.4 : à n = 10 000, `text.Grow` réserve 10 471 047 octets (environ 10,5 Mo), et non « ~2,1 Mo ». C'est F(10 000) qui compte 2 090 chiffres.
- § 4.2 :
  - `--list 10000` fait exactement 160 appels à `Write` (⌈10 461 936 / 65 536⌉), et non « ~165 » ;
  - `%+q` ne cite que la valeur de `--algo` et le nom d'une option inconnue, et non tous les arguments invalides.
- § 1.2 : HAR-7 garde la commande de lancement (le binaire), mais pas la commande de build.
- § 3.3 : « inférieure à la milliseconde » dépend de la machine. Sous Linux, la dernière étape coûte, à n = 999 999, de 4,6 à 5,2 ms en deux carrés, contre 3,2 à 3,7 ms en un produit.

Les temps de la machine Windows (§ 6.1) n'ont pas pu être revérifiés.
