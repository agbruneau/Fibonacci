# Audit — Axe 4 : Idiomatique Go & qualité de code

> Commit audité : `866b8cd` (2026-05-24) · Date : 2026-05-28 · Module `github.com/agbru/fibcalc` (go 1.26.0, toolchain go1.26.3)

## 1. Verdict d'axe

Le code est **globalement idiomatique et propre** : `go vet ./...` est **vert** (exit 0, zéro diagnostic) et aucune erreur n'est ignorée sur un chemin de production (`errcheck` ne signale **que** des fichiers `_test.go`). Les garde-fous de complexité de `.golangci.yml` (cyclomatique 15, cognitive 30, longueur 100 L / 50 stmts) sont **respectés** : `gocognit` et `funlen` ne lèvent **aucun** constat, `gocyclo` en lève **un seul** (`RenderBrailleChart`, complexité 17). Le bruit `golangci-lint` (315 lignes) est dominé par des **faux positifs** (86 `gofmt` = CRLF, 50 `misspell` = orthographe britannique volontaire) et des **choix intentionnels documentés** (stutter `format.*`, panics d'invariant `bigfft`, `//nosec` G115/G304). Le seul constat de qualité à réelle portée est **SA6002** (8 sites) : les pools `bigfft` font `Put()` d'une **valeur slice non pointer-like**, ce qui provoque une allocation de boxing à chaque libération — contre-productif dans une couche de pooling. Reste un petit lot d'hygiène : code mort (`formatAlgoList`, `defaultContext`), une affectation morte dans `fermat.go`, et de la cohérence de nommage de récepteur.

## 2. Tableau récapitulatif

| ID | Sévérité | Titre | Marqueur |
|---|---|---|---|
| A4-01 | MAJEUR | SA6002 × 8 : `sync.Pool.Put` d'une valeur slice (boxing alloc) dans la couche de pooling `bigfft` | [confirmé] |
| A4-02 | MINEUR | Code mort : `formatAlgoList` (completion) et `defaultContext` (bigfft) non utilisés | [confirmé] |
| A4-03 | MINEUR | `gocyclo` : `RenderBrailleChart` dépasse le seuil (17 > 15) | [confirmé] |
| A4-04 | MINEUR | Affectation morte `c = 1` dans `fermat.norm()` (fichier sensible) | [confirmé] |
| A4-05 | MINEUR | `revive` receiver-naming : récepteur `fermat` incohérent (`z` vs `n`) sur 19 méthodes | [confirmé] |
| A4-06 | MINEUR | `exitAfterDefer` : `os.Exit` court-circuite `defer file.Close()` (generate-golden) | [confirmé] |
| A4-07 | MINEUR | `revive` exported stutter : `format.FormatBytes`, `ThresholdAnalyzer`, etc. (20 sites) | [confirmé] |
| A4-08 | MINEUR | `gocritic` octalLiteral / sprintfQuotedString / ifElseChain (hygiène de style) | [confirmé] |
| A4-09 | INFORMATIF | `misspell` × 50 : orthographe britannique cohérente vs locale US configurée | [confirmé] |
| A4-10 | INFORMATIF | `gofmt` × 86 = faux positifs CRLF (`core.autocrlf=true`) — code commité gofmt-propre | [confirmé] |
| A4-11 | INFORMATIF | `gocritic` commentedOutCode × 22 majoritairement faux positifs (commentaires d'annotation) | [confirmé] |
| A4-12 | INFORMATIF | `unparam` × 2, `ineffassign` test × 1, `whitespace` × 2, `prealloc` × 6, `govet shadow` × 5 (tests) | [confirmé] |
| A4-13 | INFORMATIF | gosec : G115/G304 tous couverts par exclusions `.golangci.yml` + `//nosec` documentés | [confirmé] |

---

## 3. Détail des constats

### [A4-01] SA6002 × 8 : `sync.Pool.Put` d'une valeur slice provoque une allocation de boxing dans la couche de pooling `bigfft`

- **Sévérité** : MAJEUR
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/bigfft/pool.go:148`, `:245`, `:333`, `:421` ; `internal/bigfft/pool_warming.go:70`, `:79`, `:88`, `:97`
- **Preuve** :

```
internal\bigfft\pool.go:148:27: SA6002: argument should be pointer-like to avoid allocations (staticcheck)
internal\bigfft\pool.go:245:24: SA6002: argument should be pointer-like to avoid allocations (staticcheck)
internal\bigfft\pool.go:333:26: SA6002: argument should be pointer-like to avoid allocations (staticcheck)
internal\bigfft\pool.go:421:29: SA6002: argument should be pointer-like to avoid allocations (staticcheck)
internal\bigfft\pool_warming.go:70:32: SA6002: argument should be pointer-like to avoid allocations (staticcheck)
```

Les sites concernés font tous `Put` d'une **valeur** de slice (en-tête de slice) — exemples confirmés par lecture directe :

```go
// pool.go:148  (releaseWordSlice — []big.Word)
wordSlicePools[idx].Put(slice[:c])
// pool.go:245  (releaseFermat — fermat == []big.Word)
fermatPools[idx].Put(f[:c])
// pool.go:333  (releaseNatSlice — []nat)
natSlicePools[idx].Put(slice[:c])
```

- **Impact** : `sync.Pool.Put(x interface{})` boxe son argument. Pour un argument **pointer-like** (pointeur, map, chan…), Go évite l'allocation ; pour une **valeur slice** (en-tête de 3 mots), l'en-tête est copié dans une allocation tas à chaque `Put`. La couche `pool.go` est précisément le mécanisme censé **éliminer** les allocations sur le hot path FFT (`releaseWordSlice`/`releaseFermat`/`releaseNatSlice` sont appelés à chaque libération de buffer). On paie donc une alloc par libération — l'inverse de l'objectif. C'est le constat de qualité le plus matériel de l'axe (impact perf mesurable, à confirmer au benchmark par le sous-agent 3).
- **Recommandation** : pooler des **pointeurs de slice** (`*[]big.Word`, `*[]nat`, `*fermat`) : `Put(&s)` puis `*(p.Get().(*[]big.Word))` au `Get`. **Attention** : l'invariant `CLAUDE.md` / `bigfft/pool.go` (« `releaseWordSlice` route sur `cap`, pas `len` ») est **orthogonal** à ce changement et doit être conservé — le routage par capacité reste identique, seul le type stocké change. Comme ce module est sous gel perf (régression > 5 % bloquante) et touche 4 pools globaux, formuler la modification comme **proposition d'ADR** avec benchmark avant/après (`make benchmark` sur `internal/fibonacci/` + `internal/bigfft/`). Alternative à coût nul si la migration est jugée trop risquée : annoter chaque site `//nosec`-style via `//lint:ignore SA6002` avec justification, pour distinguer le choix assumé du défaut latent.
- **Marqueur** : [confirmé] (diagnostic staticcheck + lecture directe des 8 sites ; gain perf réel **[à vérifier]** au benchmark)

---

### [A4-02] Code mort : `formatAlgoList` et `defaultContext` non référencés

- **Sévérité** : MINEUR
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/cli/completion/registry.go:46` ; `internal/bigfft/context.go:104`
- **Preuve** :

```
internal\cli\completion\registry.go:46:6: func formatAlgoList is unused (U1000)
internal\bigfft\context.go:104:5: var defaultContext is unused (U1000)
```

```go
// registry.go:45
// formatAlgoList joins algorithm names with space separators.
func formatAlgoList(algorithms []string) string {
	return strings.Join(algorithms, " ")
}
```

- **Impact** : surface morte qui alourdit la lecture et masque l'intention. `formatAlgoList` est un helper jamais appelé. `defaultContext` (un `*FFTContext`) est lié au chantier de migration `FFTContext` tracé en backlog (ADR-0004 §B1, won't-fix release courante) : ce n'est pas un oubli mais un échafaudage anticipé.
- **Recommandation** : supprimer `formatAlgoList` (dead code franc). Pour `defaultContext`, **ne pas supprimer** sans arbitrage : soit le rattacher explicitement à l'ADR-0004 §B1 via un commentaire `//nolint:unused // réservé migration FFTContext (ADR-0004 §B1)`, soit le retirer si la migration est abandonnée. À trancher avec le mainteneur, pas en aveugle.
- **Marqueur** : [confirmé]

---

### [A4-03] `RenderBrailleChart` dépasse le seuil de complexité cyclomatique

- **Sévérité** : MINEUR
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/tui/sparkline.go:46`
- **Preuve** :

```
internal\tui\sparkline.go:46:1: cyclomatic complexity 17 of func `RenderBrailleChart` is high (> 15) (gocyclo)
```

- **Impact** : seul dépassement de complexité du dépôt (le reste du code respecte les seuils 15/30/100/50 fixés par `.golangci.yml` — `gocognit` et `funlen` sont à zéro). Léger : 17 contre une limite de 15, dans du code de rendu TUI non critique.
- **Recommandation** : extraire la conversion des points en quadrants Braille dans un helper (réduit la cyclomatique sous 15) **ou** documenter l'exception par `//nolint:gocyclo // logique de rendu Braille intrinsèquement branchue` comme le projet le fait ailleurs.
- **Marqueur** : [confirmé]

---

### [A4-04] Affectation morte `c = 1` dans `fermat.norm()` (fichier sensible)

- **Sévérité** : MINEUR
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/bigfft/fermat.go:61`
- **Preuve** :

```
internal\bigfft\fermat.go:61:3: ineffectual assignment to c (ineffassign)
```

```go
// fermat.go:58-68
subVW(z, z, c) // Subtract c
if c > 1 {
	z[n] -= c - 1
	c = 1            // <- jamais relue : la suite lit z[n], pas c
}
// Add back c.
if z[n] == 1 {
	z[n] = 0
	return
}
addVW(z, z, 1)
```

- **Impact** : `c` n'est plus lue après la ligne 61 (la suite raisonne sur `z[n]`). L'affectation est inerte — aucun effet de correction, mais elle peut induire en erreur un lecteur sur l'état logique de `c`. `fermat.go` est listé comme **module sensible** (panics d'invariant) ; toute modification y est à risque.
- **Recommandation** : retirer la ligne `c = 1` **uniquement** après confirmation par lecture qu'aucun chemin ne relit `c` (confirmé ici : la portée s'arrête en fin de `norm()`). Étant donné la sensibilité du fichier et le gel perf, traiter en commit isolé `fix(bigfft):` avec golden + benchmark, **pas** dans un refactor groupé. Alternative conservatrice : conserver en l'état (l'affectation est inoffensive) et documenter l'intention par un commentaire si `c` était censée refléter un invariant.
- **Marqueur** : [confirmé]

---

### [A4-05] Nommage de récepteur incohérent sur le type `fermat` (19 méthodes)

- **Sévérité** : MINEUR
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/bigfft/fermat.go` (`norm` :46, `Shift` :72, `ShiftHalf` :131, `Add` :146, `Sub` :156, `Mul` :174, … 19 sites)
- **Preuve** :

```
internal\bigfft\fermat.go:46:1: receiver-naming: receiver name z should be consistent with previous receiver name n for fermat (revive)
```

- **Impact** : Go Code Review Comments demande un nom de récepteur **unique et cohérent** par type. Le type `fermat` alterne `z` (méthodes qui écrivent dans le récepteur) et `n` (ailleurs), ce qui brouille la convention. C'est un patron **hérité** de l'implémentation FFT d'origine (style `math/big` interne, où `z` = destination) ; le mélange `z`/`n` est probablement intentionnel dans l'esprit « z écrit, n lit » mais revive le signale car non uniforme.
- **Recommandation** : uniformiser sur `z` (sémantique « destination », alignée sur `math/big`). Comme c'est un fichier sensible sous gel perf et que le changement est purement cosmétique, l'envergure (19 sites, 1 fichier) reste sous le seuil de justification ADR mais mérite un commit `refactor(bigfft):` isolé et benchmark de non-régression. Faible priorité.
- **Marqueur** : [confirmé]

---

### [A4-06] `os.Exit` court-circuite `defer file.Close()` dans generate-golden

- **Sévérité** : MINEUR
- **Axe** : 4 Idiomatique
- **Emplacement** : `cmd/generate-golden/main.go:76` (defer ligne 37)
- **Preuve** :

```
cmd\generate-golden\main.go:76:3: exitAfterDefer: os.Exit will exit, and `defer file.Close()` will not run (gocritic)
```

```go
defer file.Close()              // ligne 37
// ...
if err := encoder.Encode(data); err != nil {
	fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	os.Exit(1)                  // ligne 76 : Close() sauté
}
```

- **Impact** : sur l'échec d'encodage JSON, `os.Exit(1)` saute le `defer Close()`, laissant un descripteur ouvert et un fichier golden potentiellement partiel. Portée limitée : c'est un **outil de build hors ligne** (oracle golden), mono-exécution, le process meurt de toute façon. Le risque réel est un fichier golden tronqué silencieusement écrit sur disque avant l'exit — mais l'exit code 1 signale l'échec.
- **Recommandation** : remplacer le couple `defer Close()` + `os.Exit` par un pattern `run() error` retournant l'erreur à un `main()` minimal qui appelle `os.Exit`, **ou** fermer/supprimer explicitement le fichier partiel avant `os.Exit` dans la branche d'erreur. Faible priorité (outil dev).
- **Marqueur** : [confirmé]

---

### [A4-07] Stutter de nommage sur exports (`format.FormatBytes`, etc.) — choix partiellement assumé

- **Sévérité** : MINEUR
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/format/{duration.go:18, numbers.go:18/54, eta.go:17/85}` ; `internal/fibonacci/threshold/{analyzer.go:13, types.go:18}` ; `internal/progress/{progress.go:17/32, observer.go:27/46}` ; `internal/cli/{presenter.go:17, provider.go:19}` ; `internal/calibration/{profile.go:19, strategy.go:72, calibration.go:53}` ; `internal/tui/bridge.go:69/100` ; `internal/fibonacci/memory/budget.go:10` (20 sites)
- **Preuve** :

```
internal\format\numbers.go:54:6: exported: func name will be used as format.FormatBytes by other packages,
  and that is repetitive; consider calling this Bytes (revive)
internal\progress\progress.go:17:6: exported: type name will be used as progress.ProgressUpdate ...
  consider calling this Update (revive)
```

- **Impact** : le préfixe répète le nom de package (`format.FormatBytes`, `progress.ProgressUpdate`, `calibration.CalibrationOptions`). Effective Go décourage ce stutter (le package est déjà un namespace). Lisibilité côté appelant uniquement ; zéro impact fonctionnel. À noter : `CLAUDE.md` mentionne explicitement le helper `formatBytesLocal` dans `errors.go` (pour éviter d'importer `internal/format`), ce qui suggère que l'API `format.Format*` est un point de couplage **connu et toléré**.
- **Recommandation** : renommer vers les formes courtes (`format.Bytes`, `format.Duration`, `progress.Update`, `calibration.Options`) lors d'un futur passage — c'est un changement d'API interne (`internal/`, pas d'impact public). Si le projet préfère préserver la stabilité des call-sites, documenter le choix : ajouter `format`/`progress` à une exclusion `revive` ciblée plutôt que de laisser 20 avertissements récurrents noyer le signal. **Pas** un bug ; arbitrage de convention.
- **Marqueur** : [confirmé]

---

### [A4-08] Hygiène de style `gocritic` : octalLiteral, sprintfQuotedString, ifElseChain, paramTypeCombine

- **Sévérité** : MINEUR
- **Axe** : 4 Idiomatique
- **Emplacement** (non-test, sélection) : `cmd/generate-golden/main.go:26/32` & `internal/cli/display.go:283/288` & `internal/calibration/profile.go:151` (octalLiteral) ; `internal/cli/completion/bash.go:23/100` (sprintfQuotedString) ; `internal/{cli/completion/fish.go:94, zsh.go:66, bigfft/fermat.go:115, bigfft/memory_est.go:39, bigfft/pool_warming.go:57}` (ifElseChain) ; ~10 sites paramTypeCombine
- **Preuve** :

```
cmd\generate-golden\main.go:26:36: octalLiteral: use new octal literal style, 0o700 (gocritic)
internal\cli\completion\bash.go:23:13: sprintfQuotedString: use %q instead of "%s" for quoted strings (gocritic)
internal\bigfft\fermat.go:115:2: ifElseChain: rewrite if-else to switch statement (gocritic)
internal\fibonacci\matrix_ops.go:54:1: paramTypeCombine: func(... fftThreshold int, strassenThreshold int)
  could be replaced with func(... fftThreshold, strassenThreshold int) (gocritic)
```

- **Impact** : purement stylistique. `0700` → `0o700` (style octal moderne Go 1.13+), `"%s"` autour d'une chaîne → `%q`, chaînes if/else → `switch`, et regroupement de paramètres de même type. Aucun effet sur le comportement. Le `sprintfQuotedString` en completion `bash.go` mérite une attention particulière : `CLAUDE.md` signale l'échappement shell comme « risque de sécurité latent » — `%q` (qui échappe) est plus sûr que `"%s"` selon le contexte d'usage (à vérifier par le sous-agent sécurité que `%q` Go produit bien un échappement shell-safe, ce qui n'est **pas** garanti — `%q` est l'échappement Go, pas POSIX shell).
- **Recommandation** : appliquer les correctifs triviaux (octal, paramTypeCombine, ifElseChain) lors d'un passage de nettoyage groupé. Pour `bash.go`, **ne pas** substituer `"%s"`→`%q` à l'aveugle : valider que l'échappement attendu est bien Go-quoting et non shell-quoting (escape `$`, backticks, `"` côté shell). Faible priorité hors le point completion.
- **Marqueur** : [confirmé]

---

### [A4-09] `misspell` × 50 : orthographe britannique cohérente vs locale US configurée

- **Sévérité** : INFORMATIF
- **Axe** : 4 Idiomatique
- **Emplacement** : 50 sites en commentaires/strings (`app/`, `calibration/`, `tui/`, `config/`, `fibonacci/`, …)
- **Preuve** :

```
     16 `behaviour` is a misspelling of `behavior`
      9 `cancelled` is a misspelling of `canceled`
      3 `materialises` is a misspelling of `materializes`
      3 `honoured` is a misspelling of `honored`
internal\orchestration\progress.go:74:33: `behaviour` is a misspelling of `behavior` (misspell)
```

- **Impact** : **aucun défaut réel**. Ce sont des graphies **britanniques cohérentes** (`behaviour`, `cancelled`, `honour`, `optimised`, `synchronisation`, `defence`) que `misspell` signale parce que `.golangci.yml` fixe `locale: US`. Le commentaire de config dit lui-même « or UK/Canadian if preferred ». Le code est cohérent dans son choix britannique ; c'est la **locale du linter** qui diverge de l'intention, pas l'inverse.
- **Recommandation** : décider d'une politique unique. Soit basculer `misspell.locale` sur `UK` pour aligner le linter sur la graphie réellement employée (et faire disparaître les 50 avertissements), soit normaliser le code en US et garder `locale: US`. La situation actuelle génère 50 faux signaux qui noient le vrai signal. Choix éditorial, pas technique.
- **Marqueur** : [confirmé]

---

### [A4-10] `gofmt` × 86 : faux positifs CRLF — le code commité est gofmt-propre

- **Sévérité** : INFORMATIF
- **Axe** : 4 Idiomatique
- **Emplacement** : 86 fichiers, tous signalés en `1:1`
- **Preuve** :

```
internal\app\app.go:1:1: File is not properly formatted (gofmt)
```

Confirmation que c'est un artefact CRLF (working tree) et non un défaut de contenu :

```
$ git config core.autocrlf            -> true
$ file internal/app/app.go            -> ... with CRLF line terminators (CRLF=146, LF-only=0)
$ git show :internal/app/app.go | gofmt -l   -> (vide)   # la version indexée (LF) est gofmt-propre
$ golangci-lint --no-config -E gofmt ./internal/format/...  -> (vide)
```

- **Impact** : **nul** sur le code. Le working tree est en CRLF (`core.autocrlf=true`), l'index en LF ; `gofmt` voit les `\r` et signale chaque fichier en `1:1` sans aucune différence de contenu. La version **commitée** (index, LF) est parfaitement formatée. Ne **pas** rapporter « le code n'est pas formaté ».
- **Recommandation** : aucune action sur le code. Optionnellement, ajouter un `.gitattributes` (`*.go text eol=lf`) pour normaliser et faire taire le faux signal côté contributeurs Windows — mais c'est cosmétique et hors périmètre idiomatique strict.
- **Marqueur** : [confirmé]

---

### [A4-11] `gocritic` commentedOutCode × 22 : majoritairement des commentaires d'annotation, pas du code mort

- **Sévérité** : INFORMATIF
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/fibonacci/matrix_ops.go:128/130/132/134/158`, `internal/bigfft/fermat.go:110/111`, `internal/fibonacci/fft.go:95`, etc.
- **Preuve** :

```
internal\fibonacci\matrix_ops.go:128:21: commentedOutCode: may want to remove commented-out code (gocritic)
```

```go
// matrix_ops.go:128 — commentaire d'ANNOTATION mathématique, pas du code mort
s1.Add(m1.c, m1.d) // S1 = A21 + A22
s2.Sub(s1, m1.a)   // S2 = S1 - A11
```

- **Impact** : la majorité de ces 22 occurrences sont des **commentaires d'annotation algébrique** (`// S1 = A21 + A22`, `// C12 = T1 + P5 + P6`) que gocritic confond avec du code commenté à cause de leur forme `identifiant = expression`. Ce sont des commentaires utiles documentant l'algorithme de Strassen — leur suppression **dégraderait** la lisibilité. Quelques cas peuvent être du vrai code mort (à inspecter au cas par cas, ex. `fft.go:95`).
- **Recommandation** : ne **pas** supprimer en masse. Inspecter les rares sites qui seraient du code réellement commenté (vs annotation) et les retirer ; laisser les annotations mathématiques. Faible priorité.
- **Marqueur** : [confirmé]

---

### [A4-12] Lot d'hygiène mineure : unparam, ineffassign, whitespace, prealloc, govet shadow

- **Sévérité** : INFORMATIF
- **Axe** : 4 Idiomatique
- **Emplacement** : voir preuve
- **Preuve** :

```
internal\calibration\calibration.go:140:73: tryUseCachedCalibrationProfile - result 0 (int) is always 0 (unparam)
internal\config\env.go:22:24: getEnvString - defaultVal always receives "default" (unparam)
internal\orchestration\orchestrator.go:54:6: shadow: declaration of "ctx" shadows declaration at line 37 (govet)
internal\fibonacci\doubling_framework.go:112:5: shadow: declaration of "err" shadows declaration at line 107 (govet)
internal\cli\output_test.go:191:1: unnecessary trailing newline (whitespace)
internal\cli\completion\bash.go:84:2: Consider pre-allocating cases (prealloc)
```

- **Impact** : marginal. `unparam` (2) signale un retour `int` toujours nul et un paramètre toujours appelé avec la même valeur — pistes de simplification d'API interne. Les `govet shadow` (5) sont du shadowing d'`err`/`ctx` : `.golangci.yml` active **délibérément** `shadow` (commentaire « new occurrences will surface as warnings on PRs ») — ces 5 sites sont à examiner mais le shadowing d'`err` dans des blocs imbriqués est souvent bénin (à confirmer côté correctness, sous-agent 1, pour `doubling_framework.go` qui est un hot path). `prealloc` (6, completion/cmd) : micro-optim de capacité de slice. `whitespace`/`ineffassign` restants sont en `_test.go`.
- **Recommandation** : traiter au fil de l'eau lors de modifications des fichiers concernés (surgical). Prioriser la vérification du shadow `err` dans `doubling_framework.go:112/119` (hot path) auprès du sous-agent correctness — un shadow d'`err` peut masquer une erreur non propagée. Les autres sont du nettoyage opportuniste.
- **Marqueur** : [confirmé]

---

### [A4-13] gosec : G115/G304 entièrement couverts par exclusions documentées

- **Sévérité** : INFORMATIF
- **Axe** : 4 Idiomatique
- **Emplacement** : `internal/bigfft/{pool.go, fermat.go}`, `internal/fibonacci/matrix_ops.go` (G115) ; `internal/calibration/profile.go`, `cmd/generate-golden/main.go` (G304)
- **Preuve** :

```
$ gosec -quiet ./...   -> Issues : 22   (toutes G115 + G304)
$ golangci-lint run ./... | grep -iE '\(gosec\)|G304|G115'   -> (rien, hors un commentaire //nosec)
```

`.golangci.yml` exclut explicitement G115 (« integer overflow casts validated … annotated with `//nosec G115` ») et G304 pour `profile.go`/`output.go` (« mono-user CLI ; every path originates from a flag/env var … non-exploitable »).

- **Impact** : sous le run `golangci-lint` configuré du projet, **zéro** alerte gosec survit — toutes les 22 issues du run standalone sont des G115 (casts `int→uint`/`int→int32` bornés par contexte, annotés `//nosec G115`) et G304 (lecture de fichier depuis un flag CLI dans un outil mono-utilisateur). Choix **assumés et documentés** dans `.golangci.yml` et `CLAUDE.md`, cohérents avec le modèle de menace. Le cas `cmd/generate-golden/main.go:32` (G304) n'est pas dans la liste d'exclusion mais relève du même modèle (outil de build lisant un flag `-out`).
- **Recommandation** : aucune action côté correctness/idiomatique. Pour cohérence, le sous-agent 5 (sécurité/CI) pourra noter que `generate-golden/main.go` mériterait d'être ajouté à l'exclusion G304 ou annoté `//nosec G304` pour aligner sur le reste. Hors périmètre axe 4.
- **Marqueur** : [confirmé]

---

## Annexe — Synthèse outillage (comptages bruts)

| Outil | Résultat global | Lecture |
|---|---|---|
| `go vet ./...` | **exit 0, zéro diagnostic** | Propre |
| `staticcheck ./...` | 10 (8× SA6002, 1× U1000 `defaultContext`, 1× U1000 `formatAlgoList`) + 3 SA9003 en tests | A4-01, A4-02 |
| `golangci-lint run ./...` | 315 lignes | dont 86 gofmt (CRLF, faux+), 74 gocritic, 50 misspell (UK, faux+), 47 revive, 27 errcheck (tests only), 11 staticcheck, 6 prealloc, 5 govet shadow, 2 unparam, 2 unused, 2 ineffassign, 2 whitespace, 1 gocyclo |
| `gocognit` / `funlen` | **0 / 0** | Seuils 30 / (100 L,50 stmts) **respectés** |
| `gocyclo` | **1** (`RenderBrailleChart` = 17) | A4-03 |
| `errcheck` (production) | **0** | Aucune erreur ignorée hors tests |
| `gosec -quiet ./...` | 22 (G115 + G304) | tous exclus/annotés (A4-13) |
| `gofmt` (index/LF) | **propre** | faux positifs CRLF (A4-10) |
