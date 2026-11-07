# Audit exhaustif du répertoire `python/`

**Date de l'audit** : 2026-10-14  
**Répertoire audité** : `python/` (identifiant du contrat : `py`)  
**Environnement d'exécution vérifié** : Windows 11 (`win32/amd64`), CPython `3.14.7` (`MSC v.1944 64 bit (AMD64)`, minimum exigé par le PRD § 5 : Python 3.12), invoqué avec `python -I -S`  
**Statut global** : **Production 100 % Conforme au Contrat CLI / 1 Échec de portabilité Windows dans un sous-test témoin de `test_fib.py`** — Le contrat CLI (`node contracts/cli.test.ts py`, 114 passés, 2 sautés sous Windows pour `CLI-V04`) passe à 100 %. Dans la suite unitaire `python -m unittest discover -s python` (44 méthodes de test : `BIB-01` à `BIB-14` et `PERF-PY-1` à `PERF-PY-30`), **43 méthodes passent intégralement**, mais le sous-test témoin `absent='les deux flux, après HOSTILE'` de `test_PERF_PY_23` (`python/test_fib.py:523-528`) échoue sous Windows à cause de la traduction CRLF (`\r\n`) par défaut de `sys.__stdout__` et `sys.__stderr__` lorsque `fib.py` n'est pas exécuté.

---

## 1. Vue d'ensemble et métriques du répertoire

### 1.1 Inventaire des fichiers et volumétrie

L'implémentation Python regroupe la bibliothèque (seam 1) et l'adaptateur CLI (seam 2) dans un unique module `python/fib.py`, accompagné de sa suite de tests `python/test_fib.py`. Aucun fichier ne contient de caractère `\r` (0 octet CR).

| Fichier | Rôle | Octets | Lignes (`wc -l`) | Caractères CR |
|---|---|---:|---:|---:|
| `python/fib.py` | Bibliothèque (`fib`, `fib_list`, `_iterative`, `_matrix`, `_fast`, `_exact_decimal`, `_fib_decimal`) et CLI (`parse`, `_text`, `compute`, `main`, bloc `__main__`) | 11 189 | 270 | 0 |
| `python/test_fib.py` | Suite `unittest` : cas `BIB-01` à `BIB-14` et 30 gardes `PERF-PY-1` à `PERF-PY-30` | 45 544 | 771 | 0 |
| **Total production** | `python/fib.py` | **11 189** | **270** | **0** |
| **Total tests** | `python/test_fib.py` | **45 544** | **771** | **0** |

- **Conformité A2 (Dépendances)** : aucune dépendance tierce (`pip`, `pyproject.toml` ou `requirements.txt` absents). Seuls `sys` et `time` (modules natifs intégrés au binaire CPython) sont importés au chargement de `python/fib.py` ; `decimal` et `os` sont importés paresseusement uniquement sur les chemins qui en ont besoin.
- **Conformité A3 (Taille)** : le total de **270 lignes** (`wc -l`) pour `python/fib.py` correspond exactement au chiffre consigné dans `PRD.md` (§ 7, A3) et `README.md` (§ Coût). Avec 771 lignes de tests pour 270 lignes de production (ratio 2,86×), `python/` possède la densité de tests de garde la plus élevée du dépôt.

### 1.2 Options de lancement et maîtrise du travail de démarrage (`python -I -S`)

Conformément à l'ADR-0002 et au contrôle du harnais `HAR-7`, la commande de lancement enregistrée dans `contracts/lancement.ts` est :
```bash
python -I -S python/fib.py
```
- **`-I` (mode isolé)** : ignore toutes les variables d'environnement `PYTHON*` (`PYTHONPATH`, `PYTHONIOENCODING`, etc.) et n'ajoute pas le répertoire utilisateur à `sys.path`.
- **`-S` (sans `import site`)** : supprime l'import automatique du module `site` au démarrage de l'interpréteur, économisant ~10 à 15 ms de lectures disque sur `site-packages`. Sur la machine d'audit Windows 11, cela ramène le temps de démarrage (`petit (10)`) à **21,2 ms**, soit près de deux fois moins que Node.js (41,4 ms).
- **Débridage de la conversion décimale (`fib.py:7`)** : `sys.set_int_max_str_digits(0)` est appelé en tête de module afin de désactiver le plafond de 4 300 chiffres de CPython 3.11+ tant pour `int(positional[0])` (ex. `CLI-T14` avec 5 000 zéros de tête) que pour `str(x)` au-delà de `STR_FROM_BITS`.

---

## 2. Architecture et seam de bibliothèque (`python/fib.py`)

### 2.1 Signatures exposées et contrat de la seam 1

Le module `fib` expose deux fonctions publiques de bibliothèque conformément au PRD § 5 :

1. `def fib(n, algo)` (`fib.py:70-76`)
   - **Validation de l'algorithme** : vérifie `if algo not in _ALGOS: raise ValueError(f"algorithme inconnu : {algo!r}")`, où `_ALGOS = {"iterative": _iterative, "matrix": _matrix, "fast": _fast}` (`fib.py:58`). Rejette `"all"`, `"bogus"`, `""` et `"FAST"` (`BIB-13`).
   - **Validation stricte de l'indice (`_check_index`, `fib.py:61-67`)** :
     ```python
     def _check_index(n):
         if type(n) is not int:
             raise TypeError(f"indice invalide : {n!r}")
         if n < 0:
             raise ValueError(f"indice invalide : {n}")
     ```
     En Python, `bool` est une sous-classe de `int` (`isinstance(True, int) == True`). L'emploi de `type(n) is not int` rejette donc simultanément `True`, `False`, les flottants (`0.0`, `1.5`, `nan`) avec `TypeError`, et les entiers négatifs (`-1`) avec `ValueError` (`BIB-14`, `PERF-PY-16`). Sans ce contrôle, `_matrix(0.0)` retournerait `0` au lieu de lever une exception.
2. `def fib_list(n)` (`fib.py:99-112`)
   - Valide `n` par `_check_index(n)`, ouvre un contexte `Decimal` exact et local via `_exact_decimal()`, et accumule les termes $F(0) \dots F(n)$ directement en `Decimal` pour retourner la liste des `n + 1` chaînes décimales.

---

## 3. Audit algorithmique, arithmétique polymorphe et seuils de conversion

### 3.1 Algorithme itératif (`_iterative`, `fib.py:10-16`)

- Exécute `for _ in range(n): a, b = b, a + b` sur des `int` Python natifs en partant de `a, b = 0, 1`.

### 3.2 Algorithme matriciel polymorphe (`_matrix(n, kind=int)`, `fib.py:19-34`)

- **Polymorphisme `int` / `Decimal`** : accepte le constructeur `kind` (`int` par défaut, ou `decimal.Decimal` lorsqu'il est appelé depuis `_fib_decimal`).
- **Parcours du poids fort au poids faible** : initialise la matrice symétrique à l'identité `a, b, c = kind(1), kind(0), kind(1)` et parcourt `bits = format(n, "b")` sur `bits[:-1]`.
- **Élévation au carré en trois carrés** : calcule `square_a, square_b, square_c = a * a, b * b, c * c`, puis `a, c = square_a + square_b, square_b + square_c` et `b = a - c`. Si le bit vaut `"1"`, multiplie par $Q$ via `a, b, c = a + b, a, b`.
- **Optimisation du dernier bit (`fib.py:34`)** : ne calcule que $F(n)$ en une seule multiplication pour un bit pair (`b * (a + c)`) ou deux carrés pour un bit impair (`a * a + b * b`).

### 3.3 Algorithme de doublement rapide polymorphe (`_fast(n, kind=int)`, `fib.py:37-53`)

- **Formules de GMP en deux carrés par bit** : pour $n \ge 2$, démarre à $k = 1$ (`f, g = kind(1), kind(0)`, `sign = -2` représentant $2(-1)^k$) et décompose `format(n, "b")[1:]` en `*middle, last`.
- Sur chaque bit intermédiaire `bit in middle`, calcule `square_f, square_g = f * f, g * g`, puis $F(2k-1) = f^2 + g^2$ (`before`), $F(2k+1) = 4f^2 - g^2 + 2(-1)^k$ (`odd`), et $F(2k) = F(2k+1) - F(2k-1)$ (`odd - before`).
- **Factorisation du dernier bit en un seul produit (`fib.py:53`)** :
  ```python
  return (2 * f - g) * (2 * f + g) + sign if last == "1" else f * (f + 2 * g)
  ```
  Grâce à l'identité $(2f - g)(2f + g) = 4f^2 - g^2$, le dernier pas impair ne requiert **qu'une seule multiplication** au lieu de deux carrés, tout comme le dernier pas pair `f * (f + 2 * g)`.

### 3.4 Architecture à double seuil : `DECIMAL_FROM = 170_000` et `STR_FROM_BITS = 200_000`

L'implémentation Python contourne deux goulots d'étranglement de CPython 3.12+ :
1. La conversion `str(int)` d'un entier binaire Python de plus de ~4 300 chiffres est quadratique jusqu'au seuil où CPython importe paresseusement le module Python `_pylong` (qui tire avec lui `re` et `decimal`, coûtant ~10 ms au premier appel), et prend encore ~54 ms pour convertir $F(1\,000\,000)$.
2. Le module standard `decimal` (adossé à la bibliothèque C `libmpdec`) stocke les nombres directement en base $10^{19}$ et utilise la transformée théorique des nombres (NTT) pour multiplier les grands nombres : son `str(d)` ne coûte que **0,5 ms** à $n = 1\,000\,000$, mais `import decimal` coûte ~5 ms au premier chargement.

Pour optimiser chaque régime, `python/fib.py` combine deux mécanismes :

#### A. Bascule vers `Decimal` au-delà de `DECIMAL_FROM = 170_000` (`_exact_decimal`, `_fib_decimal`, `compute`, `fib.py:79-96, 204-232`)
- Dans `compute(algos, as_list, n)`, le booléen `use_decimal = n >= DECIMAL_FROM and "iterative" not in algos` détermine si le calcul s'effectue en `Decimal`.
- **Préchargement hors chronomètre (`fib.py:223-226`)** : si `as_list` ou `use_decimal` est vrai, `import decimal` est exécuté **avant** l'entrée dans `timed(...)`, afin que les ~5 ms d'import du module ne faussent pas la durée `elapsed` rapportée par `--time` (`PERF-PY-25`).
- **Contexte exact et local (`_exact_decimal`, `fib.py:79-85`)** :
  ```python
  from decimal import MAX_EMAX, MAX_PREC, Context, Decimal, localcontext
  return Decimal, localcontext(Context(prec=MAX_PREC, Emax=MAX_EMAX))
  ```
  Chaque calcul dans `_fib_decimal` et dans `fib_list` s'exécute dans un bloc `with exact:` doté de la précision maximale de `libmpdec` (`MAX_PREC`, aucun arrondi possible) sans jamais modifier le contexte `decimal.getcontext()` de l'appelant (`PERF-PY-6`, `PERF-PY-7`, `PERF-PY-21`).
- En revanche, sous $n < 170\,000$ (tous les scénarios $n = 10$ et $n = 100\,000$ du banc d'essai hors `--list`), ou lorsque `"iterative"` figure dans `algos`, `decimal` n'est pas importé et `compute` appelle `fib(n, a)` en `int` (`PERF-PY-8`, `PERF-PY-13`, `PERF-PY-17`).

#### B. Découpage par tranches de 1 000 chiffres dans `_text(x)` sous `STR_FROM_BITS = 200_000` (`fib.py:182-199`)
- Lorsque `compute` produit un `int` (ex. $F(100\,000)$, qui compte $69\,424\text{ bits} \le 200\,000$), appeler `str(x)` déclencherait l'import de `_pylong` (10 ms) pour un entier de 20 899 chiffres.
- `_text(x)` vérifie donc `if type(x) is not int or x.bit_length() > STR_FROM_BITS: return str(x)`.
- En deçà de `200_000` bits, `_text(x)` découpe l'entier par divisions successives `x, rest = divmod(x, 10**1000)` (`CHUNK_DIGITS = 1000`) et formate chaque reste de 1 000 chiffres via `str(rest).zfill(1000)` (sous le seuil de `_pylong`). À $n = 100\,000$, ces 21 divisions C prennent moins d'une milliseconde et évitent de charger `_pylong`, `re` et `decimal` (`PERF-PY-1` à `PERF-PY-5`, `PERF-PY-8`, `PERF-PY-14`, `PERF-PY-17` à `PERF-PY-19`).

---

## 4. Audit de l'adaptateur CLI (`python/fib.py:115-271`) et conformité au contrat

### 4.1 Conformité à la grammaire (PRD § 4, règles 1 à 6)

| Règle | Implémentation dans `python/fib.py` | Lignes | Conformité |
|---|---|---|---|
| **Règle 1** (`--help` / `-h` prioritaire) | `if "--help" in args or "-h" in args:` en tête de `main(args)`, avant `parse(args)` | `241-243` | Exacte (`CLI-H01` à `H11`) |
| **Règle 2** (`--algo X` / `--algo=X`) | Itérateur unique `rest = iter(args)` partagé entre `for a in rest:` et `next(rest, None)` ; valide chaque occurrence contre `"all"` ou `value in COMPARISON` ; cite en ASCII via `ascii(value)` | `141-157` | Exacte (`CLI-A01` à `A20`, `D04`, `D05`) |
| **Règle 3** (`--list`, `--time`) | `as_list = True` et `show_time = True` idempotents | `158-161` | Exacte (`CLI-L08`, `D08`) |
| **Règle 4** (Options inconnues `-*`) | `elif a.startswith("-"): raise CliError(f"option inconnue : {ascii(a)}")` | `162-163` | Exacte (`CLI-I11`, `L14`, `D10`, `U01` à `U06`) |
| **Règle 5** (Indice `N` unique `^[0-9]+$`) | `len(positional) != 1`, puis `not (positional[0].isascii() and positional[0].isdigit())` (sans importer `re`, 8 ms économisés), puis `int(positional[0])` et `if n > bound:` | `168-177` | Exacte (`CLI-T01` à `T14`, `I01` à `I17`) |
| **Règle 6** (`--list` exclut `--algo`) | `if as_list and algos:` vérifié avant le repli `algos or ["fast"]` | `166-167, 178` | Exacte (`CLI-L10` à `L12`) |

### 4.2 Écriture par tranches, purge et durcissement du bloc `__main__`

- **Écriture par tranches de `LINES_PER_WRITE = 50` lignes (`fib.py:236, 250-251`)** : évite de matérialiser une chaîne unique de 10,5 Mo pour `--list 10000` (qui provoquerait des fautes de page et coûterait ~7 ms de plus) tout en divisant par 50 le nombre d'appels à `sys.stdout.write` (`PERF-PY-22`).
- **Purge `sys.stdout.flush()` (`fib.py:253`)** : garantit que les termes sont vidés avant l'écriture des durées sur `sys.stderr`, même lorsque les deux flux partagent un même tube.
- **Bloc `if __name__ == "__main__":` (`fib.py:259-270`)** :
  - Si `sys.stdout` ou `sys.stderr` vaut `None` (descripteur 1 ou 2 fermé au lancement, `CLI-V04`), importe `os` paresseusement (`fib.py:265`, économisant 2,5 ms sur le chemin normal, `PERF-PY-24`) et ouvre `os.devnull` en écriture (`PERF-PY-23`).
  - Reconfigure systématiquement `sys.stdout` et `sys.stderr` avec `stream.reconfigure(encoding="ascii", newline="\n")`, garantissant une sortie en ASCII pur et des fins de ligne `\n` même sous Windows ou en présence de `PYTHONIOENCODING=utf-16` (`PERF-PY-20`, `PERF-PY-29`).

---

## 5. Audit de la suite de tests (`python/test_fib.py`) et analyse de l'échec sous Windows

### 5.1 Inventaire des 44 méthodes de test (`BIB-01` à `BIB-14` et `PERF-PY-1` à `PERF-PY-30`)

- **Cas de bibliothèque (`BIB-01` à `BIB-14`, lignes 248–323)** : `oracle_text()` et `oracle()` lisent et valident `../contracts/b000045.txt` (2 001 termes consécutifs). `BIB-05` à `BIB-07` utilisent `self.subTest(n=n)` sur `0..2000`. `BIB-14` teste `REFUSED = ((-1, ValueError), (1.5, TypeError), (float("nan"), TypeError), (0.0, TypeError), (True, TypeError))` sur les 3 algorithmes et sur `fib_list`.
- **Gardes de `_text(x)` et `STR_FROM_BITS` (`PERF-PY-1` à `5`, `14`, `18`, `19`)** : éprouvent les frontières de tranches de 1 000 chiffres ($10^k$, $10^k - 1$, $10^{2000} + 1$, $10^{2000} + 10^{1000}$), la concordance avec `str(fib(100000, "fast"))`, et vérifient par espion sur `fib.divmod` le basculement exact entre $k = 60\,205$ ($199\,997$ bits, tranches) et $k = 60\,206$ ($200\,001$ bits, `str()`).
- **Gardes du chemin `Decimal` et de `DECIMAL_FROM` (`PERF-PY-6`, `7`, `9` à `13`, `15`, `16`, `21`, `25`, `28`)** :
  - `PERF-PY-6`, `7`, `21` vérifient que l'objet contexte, `prec` et tous les champs (`repr(decimal.getcontext())`, y compris `Emax`) de l'appelant restent intacts.
  - `PERF-PY-10` à `12` vérifient aux abords de `DECIMAL_FROM` ($169\,999$, $170\,000$) et à $1\,000\,000$ le nombre exact de chiffres ($\lfloor n \log_{10}\varphi - \log_{10}\sqrt{5}\rfloor + 1$) et les congruences modulo 2048 (période de Pisano 3072) et modulo 1000 (période de Pisano 1500).
  - `PERF-PY-25` vérifie que `decimal` est déjà présent dans `sys.modules` au démarrage du chronomètre `time.perf_counter` et qu'aucun module ne s'ajoute pendant la fenêtre chronométrée.
  - `PERF-PY-28` combine quatre espions dans un sous-processus neuf (`fib.divmod`, absence de `_pylong`, sous-classe `Spy(Decimal)` refusant tout `int` de plus de 64 bits, et vérification de `issubclass(kind, Decimal)` dans `_ALGOS`) sur toute la plage `[170_000, 1_000_000]` pour prouver qu'aucun `int` n'est calculé ni converti sur le chemin `Decimal`.
- **Gardes d'imports légers (`PERF-PY-8`, `17`, `24`)** : vérifient que `100000` et `--algo all 100000` n'importent ni `re`, ni `decimal`, ni `_pylong`, et que `python -I -S python/fib.py 10` n'ajoute pas `os` par rapport à `python -I -S -c pass` (`-X importtime`, qui passe et n'est pas sauté sous Windows 11 / Python 3.14.7).
- **Gardes des exigences F2, F3, F6, F10 et `CLI-V04` (`PERF-PY-20`, `22`, `23`, `26`, `27`, `29`, `30`)** : vérifient `PYTHONIOENCODING=utf-16` hors `-I`, l'écriture par lots de 50 lignes (`out.calls <= 401`), les flux absents, la discordance (code 3, `stdout` vide, `stderr` sans durée), l'ordre et le nom des algorithmes chronométrés, le redressement de flux UTF-16 CRLF en ASCII LF, et l'identité `_ALGOS[name] is function` + `function.__name__ == f"_{name}"`.

### 5.2 Diagnostic de l'échec du sous-test témoin de `PERF-PY-23` sous Windows (`python/test_fib.py:523-528`)

Lors de l'exécution de `python -m unittest discover -v -s python` sous Windows 11, un unique sous-test échoue :
```text
FAIL: test_PERF_PY_23_un_flux_standard_absent_au_lancement_ne_change_pas_le_code (test_fib.TestFib.test_PERF_PY_23_un_flux_standard_absent_au_lancement_ne_change_pas_le_code) (absent='les deux flux, après HOSTILE')
AssertionError: Tuples differ: (b'stdout\r\n', b'stderr\r\n', 0) != (b'stdout\n', b'stderr\n', 0)
```

**Analyse de la cause racine** :
1. Aux lignes 515–522 de `python/test_fib.py`, les trois premiers sous-tests de `PERF-PY-23` exécutent le bloc `__main__` de `python/fib.py` via `run_main_block(...)` lorsque `stderr` ou `stdout` vaut `None`. **Ces trois sous-tests passent à 100 % sous Windows**, car `fib.py:269` exécute `stream.reconfigure(encoding="ascii", newline="\n")` sur les flux ouverts.
2. En revanche, le 4ᵉ sous-test aux lignes 523–528 est un **témoin du harnais de test (`HOSTILE`)** qui **n'exécute pas `fib.py`** :
   ```python
   with self.subTest(absent="les deux flux, après HOSTILE"):
       probe = HOSTILE + "sys.stdout = sys.stderr = None\n" + "".join(
           f"sys.__{name}__.write('{name}\\n'); sys.__{name}__.flush()\n" for name in ("stdout", "stderr")
       )
       run = subprocess.run([sys.executable, "-I", "-S", "-c", probe], capture_output=True)
       self.assertEqual((run.stdout, run.stderr, run.returncode), (b"stdout\n", b"stderr\n", 0))
   ```
3. Son but (documenté lignes 512–514) est de vérifier que `kept = sys.stdout, sys.stderr` dans `HOSTILE` empêche le ramasse-miettes de fermer les tampons sous-jacents de `sys.__stdout__` et `sys.__stderr__` lorsque `sys.stdout` et `sys.stderr` sont mis à `None`.
4. Or, comme `fib.py` n'est pas lancé dans `probe`, `sys.__stdout__` et `sys.__stderr__` conservent leur configuration native de démarrage de CPython. Sous Linux, leur fin de ligne native est `\n` (`b"stdout\n"`), mais **sous Windows, CPython initialise `sys.__stdout__` et `sys.__stderr__` avec `newline=None` (traduction automatique de `\n` en `\r\n`)**, produisant `(b"stdout\r\n", b"stderr\r\n", 0)`.

---

## 6. Constats d'audit : Forces, Risques et Recommandations

### 6.1 Forces majeures
1. **Plus rapide des langages non compilés en AOT sur l'ensemble du banc d'essai** : grâce à `python -I -S` (21,2 ms au démarrage), aux tranches de `_text` à $n = 100\,000$ (23,6 ms au total) et à l'arithmétique `Decimal` à $n = 1\,000\,000$ (40,3 ms au total) et sur `--list 10000` (67,6 ms au total), Python devance systématiquement TypeScript/Node.js en temps de réponse mural (`wall_ms`) sur chacun des 7 scénarios du banc d'essai.
2. **Isolation irréprochable du contexte `Decimal` et du chargement des modules** : ni le contexte `decimal` de l'appelant, ni la fenêtre du chronomètre `--time`, ni le démarrage à froid ne subissent d'effet de bord.
3. **Suite de tests de garde d'une rigueur exceptionnelle** : les 30 gardes `PERF-PY-1` à `PERF-PY-30` combinent des tests de propriétés mathématiques (périodes de Pisano modulo 1000 et 2048, formule de Binet pour le nombre de chiffres) et des sondes d'exécution (`-X importtime`, `sys.modules`, espions sur `divmod`, `Decimal.__new__` et `_ALGOS`).

### 6.2 Constats, lacunes connues et recommandations

| ID | Sévérité | Catégorie | Description et localisation | Recommandation / Statut |
|---|---|---|---|---|
| **PY-AUD-01** | **Majeure (Test sous Windows)** | Portabilité Windows du sous-test témoin de `PERF-PY-23` (`python/test_fib.py:523-528`) | Le témoin écrit directement dans `sys.__stdout__` et `sys.__stderr__` sans lancer `fib.py`. Sous Windows, CPython traduit `\n` en `\r\n` sur ces flux bruts, ce qui fait échouer `self.assertEqual((run.stdout, run.stderr, run.returncode), (b"stdout\n", b"stderr\n", 0))` et bloque `npm run check:py` sous Windows alors que `python/fib.py` est 100 % conforme. | Dans `python/test_fib.py:528`, normaliser `run.stdout.replace(b"\r\n", b"\n")` et `run.stderr.replace(b"\r\n", b"\n")` (ou comparer à `os.linesep.encode("ascii")`), puisque ce témoin vérifie uniquement que `sys.__stdout__` et `sys.__stderr__` restent ouverts après `HOSTILE`. |
| **PY-AUD-02** | **Lacune connue** | Délégation interne dans un algorithme (`python/fib.py:19-53`) | Comme documenté dans `test_fib.py:761-763` et au catalogue § 6, `PERF-PY-30` vérifie l'identité et `__name__` des fonctions dans `_ALGOS`, mais ne peut pas voir si le corps de `_matrix` appelait `return _fast(n, kind)`. | Documenté au catalogue § 6 ; vérifié par lecture directe de `fib.py:19-53`. |
| **PY-AUD-03** | **Lacune connue** | Mesure réelle de la durée (`F6`) et frontière exacte `bit_length() == 200_000` | `PERF-PY-27` vérifie l'ordre et le nom des algorithmes dans les lignes `--time` mais pas la valeur numérique issue de `time.perf_counter()`. Par ailleurs, dans `PERF-PY-19`, $10^{60\,205} + 1$ compte $199\,997$ bits et $10^{60\,206} + 1$ en compte $200\,001$, laissant le mutant `< STR_FROM_BITS` vs `<= STR_FROM_BITS` (`fib.py:191`) indifférent sur les puissances de 10 et les termes de Fibonacci. | Documenté au catalogue § 6 ; un sous-test `_text(1 << (STR_FROM_BITS - 1))` (exactement $200\,000$ bits) dans `PERF-PY-19` couvrirait la frontière exacte si souhaité. |

---

## 7. Suivi du 2026-10-14

Les §§ 1 à 6 décrivent l'état du commit a92a808, tel que l'audit l'a lu; ils ne sont pas réécrits. Ce qui a changé depuis, et ce qu'ils affirmaient à tort, est ici.

Cet audit a été vérifié constat par constat, sous Linux, avec CPython 3.13.16. Trois agents adverses (exactitude, tri, critique) ont lu le code et les sources de CPython, émulé Windows et joué des mutants dans des copies. Les constats retenus ont été implémentés par `fib-dev`, puis revus par un vérificateur par branche et par `/code-review` ([agbruneau/Fib#45](https://github.com/agbruneau/Fib/pull/45)). En production, seul le commentaire de `_check_index` change, à nombre de lignes constant : `python/fib.py` reste à 270 lignes.

### 7.1 Suite donnée aux constats

| ID | Suite |
|---|---|
| PY-AUD-01 | Corrigé. Le témoin « les deux flux, après HOSTILE » de PERF-PY-23 attend `os.linesep`, la fin de ligne native que gardent `sys.__stdout__` et `sys.__stderr__` (`\r\n` sous Windows), et non `\n`. Sous Windows émulé (flux reconfigurés en `\r\n`), l'ancien test rougit avec le message de l'audit, et le nouveau passe. Le mutant « HOSTILE sans `kept` » rougit encore. Ce sous-test, ajouté en #35 (2026-10-13), rendait `npm run check:py` rouge sous Windows. Le correctif reste à revérifier sur la machine Windows. |
| PY-AUD-02 | Fermé par PERF-PY-30. Le code de `_iterative`, `_matrix` et `_fast` (`co_names`, récursivement) ne nomme aucun autre algorithme, ni `fib`, `fib_list`, `_ALGOS` ou `_fib_decimal`. Des espions qui ne calculent pas sont joués par `fib()` et `_fib_decimal` à huit indices : 0, 1, 11, `DECIMAL_FROM - 1`, `DECIMAL_FROM`, 500 000, `MAX_TERM - 1` et `MAX_TERM`. Restent à la revue : une délégation par un alias capturé ou par une fonction auxiliaire, et un aiguillage sur des valeurs de n que la garde ne joue pas. |
| PY-AUD-03, durée | Fermé par PERF-PY-31. Un algorithme espion dort 20 ms, tour à tour pour chacun de `--algo all`, et chronomètre son sommeil par `time.perf_counter`. La ligne de l'algorithme lent vaut au moins ce sommeil, et les trois lignes au plus le temps mural de `main`. `--list` est éprouvé par un `fib_list` qui dort de même. Reste à la revue : l'étendue exacte de la fenêtre. |
| PY-AUD-03, frontière de 200 000 bits | Fermé par PERF-PY-19, qui joue `1 << (STR_FROM_BITS - 1)`, soit 200 000 bits tout juste : le mutant `>=` rougit. F(288 085), qui compte aussi 200 000 bits, est le seul terme à cette frontière, et seuls `iterative` et `all` l'atteignent. |

Ajouts hors audit :
- PERF-PY-27 joue aussi 0, `DECIMAL_FROM - 1` et la borne F8, avec des espions qui ne calculent pas : un nom qui change selon n dans `compute` rougit.
- Le commentaire de `_check_index`, faux, est corrigé. `_matrix(0.0)` lève `ValueError`. C'est `_fast` qui rendrait 0 pour 0.0 et 1 pour 1.5.

### 7.2 Errata

Affirmations de cet audit que la vérification a trouvées inexactes :

- PY-AUD-03 :
  - la frontière exacte de `STR_FROM_BITS` n'était pas au catalogue § 6, malgré le statut « Documenté » ;
  - le mutant littéral est `x.bit_length() >= STR_FROM_BITS`, la ligne disant `>` ;
  - le mutant n'est pas « indifférent sur les termes de Fibonacci », puisque F(288 085) compte exactement 200 000 bits.
- § 2.1 : sans le contrôle de type, `_matrix(0.0)` lève `ValueError`, car `format(n, "b")` refuse un float ; il ne « retournerait » pas 0. C'est `_fast(0.0)` qui rend 0 et `_fast(1.5)` qui rend 1. L'audit recopiait le commentaire de `fib.py`, faux lui aussi, que ce suivi corrige.
- § 1.2 :
  - `-S` épargne environ 3 à 5 ms (PRD § 5 ; mesuré ici, 4,5 ms), et non « ~10 à 15 ms » ;
  - la commande de lancement vient de l'ADR-0004 et du PRD § 5, et non de l'ADR-0002.
- § 3.4 B : à n = 100 000, `_text` fait 20 divisions, et non 21, pour 21 tranches. Elles prennent environ 2 à 5 ms, et non « moins d'une milliseconde ».
- § 5.1 :
  - PERF-PY-28 échantillonne cinq indices de la plage, et non « toute la plage » ;
  - PERF-PY-22 garde une optimisation, et non une exigence.
- § 4 : `fib.py` compte 270 lignes, donc la plage va de 115 à 270.

Les temps de la machine Windows (§ 1.2, § 6.1) n'ont pas pu être revérifiés.
