# Plan d'exécution — évaluation académique du 2026-09-15

- **Source** : [`evaluation-academique-2026-09-15.md`](evaluation-academique-2026-09-15.md)
  (constats P1–P11, recommandations R-1–R-10, réserves par critère C1–C10).
- **Base** : `main` à `dbac1a2` (`v4.1.1`), arbre propre, gate vert (build, vet,
  `-race -shuffle=on -count=1` sur 22 paquets, lint 0, `govulncheck` 0,
  couverture 96,1 %).
- **Régime** : production — chaque tâche porte un critère d'acceptation et une
  commande de vérification ; rien n'est déclaré fait sans elle.
- **Identifiants** : les tâches s'appellent `EVAL-nn`. Le préfixe est ajouté à
  [`INDEX.md`](INDEX.md) en tâche EVAL-00, sinon `TestAuditIdentifiersResolve`
  refuse toute citation dans un commentaire Go.

---

## 1. Principes

- **Une branche** : `eval/2026-09-15`. Un commit par tâche, sujet *Conventional
  Commits* portant l'identifiant (`docs(bigfft): retain upstream BSD notice (EVAL-01)`).
- **Le gate après chaque tâche** : `pwsh scripts/check.ps1` (ou
  `bash scripts/check.sh` sous WSL), CI verte avant de passer à la suivante.
- **Performance** : toute tâche qui touche `internal/fibonacci` ou `internal/bigfft`
  est mesurée par `benchstat` en **double ordre** (protocole ADR-0009 R4,
  `-count=8`, `BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)`), seuil 5 %.
- **Décisions** : un `ADR-0013` consigne les décisions préalables (§ 2), les
  tâches écartées et les mesures. Il est créé en *Proposed* (EVAL-00) et passe
  en *Accepted* à la clôture (EVAL-25).
- **Ordre** : recevabilité d'abord (phase 1, bloquante), exactitude des
  documents ensuite (phase 2), filet de tests avant toute suppression de code
  (phase 4 avant phase 3), mesures sur le code final (phase 3), hygiène (phase 5),
  perspectives optionnelles (phase 6), clôture (phase 7).
- **Ce que le plan ne refait pas** : les candidats rejetés sur mesure par les
  ADR 0008 à 0012 ne sont pas rouverts ; les constats `[E]` de l'évaluation sont
  pris comme établis, les `[L]`/`[D]` sont revérifiés au moment de la tâche.

---

## 2. Décisions préalables du mainteneur

À trancher avant EVAL-00. Chaque ligne porte la recommandation de l'évaluation ;
la colonne « Décision » reste vide jusqu'à l'arbitrage.

| # | Question | Options | Recommandation | Décision | Tâches |
|---|---|---|---|---|---|
| D1 | Sort du gestionnaire de seuils dynamiques (DTM), mesuré neutre (ADR-0001, `-count=8`) | **(a)** supprimer le paquet `threshold`, le drapeau et la plomberie ; **(b)** remesurer à `-count=10` sur deux hôtes et garder si ≥ 5 % reproductible | **(a)** — la mesure qui le justifiait ne se reproduit pas ; 646 lignes de production et 845 de tests sans résultat ; l'histoire reste dans git et ADR-0001 | | EVAL-10 |
| D2 | Règle de langue (CONTRIBUTING § 4, ADR-0012 D4) non appliquée à treize documents | **(a)** traduire les treize documents (≈ 8 000 lignes) ; **(b)** amender la règle : narratif (README, CHANGELOG, ADR, `docs/audits/`) en français, référence technique (`docs/*.md`, `docs/algorithms/`, `docs/architecture/`) en anglais, `ARCH.md` rendu monolingue | **(b)** — la traduction coûte plusieurs jours pour un lectorat qui lit déjà le code en anglais ; une règle qui décrit la pratique vaut mieux qu'une règle ignorée | | EVAL-19 |
| D3 | Entrée `CHANGELOG.md` 1.0.0 (2025-12-22) citant JSON, hexadécimal, politique de sécurité, limitation de débit | **(a)** réécrire en ne gardant que l'attesté ; **(b)** retirer les quatre lignes démontrées fausses et marquer l'entrée « préhistoire, hors historique git de ce dépôt » | **(b)** — les quatre lignes sont réfutables par `--help` ; le reste n'est ni prouvable ni réfutable ici, on le dit | | EVAL-03 |
| D4 | Plancher de couverture (80 % gardé, 96,1 % mesuré) | 90 % ou 92 % | **92 %** — laisse 4 points pour la variance entre Ubuntu et Windows (fichiers spécifiques à une plateforme) ; à confirmer sur le relevé CI Ubuntu avant de fixer | | EVAL-12 |
| D5 | Quinze étiquettes `rewrite/*` (juillet 2026) | **(a)** supprimer localement et sur `origin` ; **(b)** garder et les expliquer dans `HISTORY.md` | **(a)** — points de sauvegarde d'une réécriture close ; la suppression distante est irréversible, elle exige l'accord explicite du mainteneur et une sauvegarde préalable | | EVAL-16 |
| D6 | *Worktree* `.claude/worktrees/lucid-nightingale-3e810f` (branche `claude/determined-carson-0df1c3`) | **(a)** vérifier son état, puis `git worktree remove` et suppression de la branche ; **(b)** conserver | **(a)**, sous réserve que `git status` y soit propre et la branche fusionnée ou abandonnée — un *worktree* peut être du travail en cours, le mainteneur tranche | | EVAL-17 |
| D7 | Numéro de version de clôture | `v4.2.0` ou `v5.0.0` | **`v5.0.0` si D1 = (a)** (retrait du drapeau `-dynamic-thresholds` et de `FIBCALC_DYNAMIC_THRESHOLDS`, rupture d'interface, même précédent que les codes de sortie en 4.0.0) ; **`v4.2.0`** sinon | | EVAL-25 |
| D8 | Mesure externe de référence (GMP) | **(a)** brancher `-algo gmp` sur la fabrique par défaut sous le tag et mesurer dans le job CI `gmp` ; **(b)** mesure ponctuelle hors code (PARI/GP ou `gmpy2`) archivée | **(a)** — le calculateur existe déjà, le job CI a libgmp, et la mesure se rejoue à chaque poussée | | EVAL-23, EVAL-07 |

---

## 3. Phases et tâches

Effort : S < 2 h, M ½–1 j, L 1–3 j. « Vérif. » = commande dont le résultat conclut
la tâche. « Constats » renvoie à l'évaluation.

### Phase 0 — Cadrage (½ j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-00 | Branche `eval/2026-09-15` ; `docs/adr/0013-evaluation-2026-09-decisions.md` depuis `0000-template.md`, statut *Proposed*, D1–D8 recopiées avec leur arbitrage ; ligne `EVAL-` dans `docs/audits/INDEX.md` (famille, source, plan) ; base `benchstat` (`-count=8`) et profil de couverture conservés dans le scratchpad | — | `docs/adr/0013-*.md`, `docs/audits/INDEX.md` | ADR-0013 existe ; `TestAuditIdentifiersResolve` vert avec un commentaire d'essai citant `EVAL-00` | `go test ./internal/ -run AuditIdentifiers` ; `ls docs/adr/0013*` | S | D1–D8 |

### Phase 1 — Recevabilité (bloquante, ½ j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-01 | Attribution de `internal/bigfft` : (1) `internal/bigfft/LICENSE` = texte BSD-3-Clause amont **verbatim** (« Copyright (c) 2012 The Go Authors ») ; (2) `NOTICE` à la racine (Apache § 4(d)) nommant `github.com/remyoudompheng/bigfft`, sa licence, la liste des fichiers dérivés et la mention « modifications © 2026 André-Guy Bruneau, Apache-2.0 » ; (3) en-tête « Derived from github.com/remyoudompheng/bigfft (BSD-3-Clause; see LICENSE in this directory) » dans chaque fichier dérivé — déterminer la liste par comparaison avec l'amont (`fft.go`, `fermat.go`, `arith_decl.go` sûrs ; `fft_core.go`, `fft_poly.go`, `fft_recursion.go` sont des scissions de l'ancien `fft.go` amont, à confirmer par `diff`) ; (4) paragraphe « Provenance » dans `internal/bigfft/doc.go` et dans `docs/algorithms/BIGFFT.md` ; (5) ligne dans « Remerciements » du `README.md` ; (6) entrée `CHANGELOG.md` sous *Unreleased*, section *Fixed* | P1, R-1, R3 | `internal/bigfft/LICENSE`, `NOTICE`, `internal/bigfft/*.go` (en-têtes), `internal/bigfft/doc.go`, `docs/algorithms/BIGFFT.md`, `README.md`, `CHANGELOG.md` | La notice amont est présente sans modification ; chaque fichier dérivé la référence ; le nom de la bibliothèque apparaît dans README, doc du paquet et CHANGELOG ; aucune ligne de code ne change | `test -f internal/bigfft/LICENSE && test -f NOTICE` ; `grep -rl remyoudompheng internal/bigfft README.md CHANGELOG.md docs/algorithms/BIGFFT.md \| wc -l` ≥ 8 ; `git diff --stat -- '*.go'` ne montre que des lignes de commentaire | S | EVAL-00 |

### Phase 2 — Exactitude des documents (1 j, tâches parallélisables)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-02 | Complexité de Schönhage-Strassen : remplacer O(n log n) par O(n log n log log n) aux six sites (`README.md` § Algorithmes, `docs/algorithms/FFT.md` en-tête, introduction, tableau et figure ASCII, `docs/algorithms/BIGFFT.md` en-tête, `internal/fibonacci/fastdoubling.go` commentaire de type) ; ajouter dans `FFT.md` un paragraphe qui distingue Schönhage-Strassen (implémenté) de Harvey–van der Hoeven 2021 (borne O(n log n), non implémentée) | P2, R-2 | `README.md`, `docs/algorithms/FFT.md`, `docs/algorithms/BIGFFT.md`, `internal/fibonacci/fastdoubling.go` | Zéro occurrence de « O(n log n) » qui désigne la multiplication implémentée ; le paragraphe de distinction cite la référence ajoutée par EVAL-06 | `grep -rn 'O(n log n)' README.md docs internal --include=*.md --include=*.go \| grep -v 'log log'` vide | S | — |
| EVAL-03 | `CHANGELOG.md` 1.0.0 selon D3 : retirer « JSON output format support », « Hexadecimal result display option », « Security policy with vulnerability disclosure process », « Rate limiting protection against DoS » ; note en tête d'entrée : « Préhistoire : le code Go entre dans ce dépôt le 2026-01-23 (`5bfc50e`) ; les entrées 0.1.0 et 1.0.0 ne sont pas vérifiables ici. » ; même note pour 0.1.0 | P3, D3 | `CHANGELOG.md` | Les quatre fonctions absentes ne sont plus affirmées ; la note est présente | `grep -n 'JSON output\|Hexadecimal\|Rate limiting\|Security policy' CHANGELOG.md` vide | S | — |
| EVAL-04 | `docs/audits/HISTORY.md`, paragraphe « Limites déclarées par la passe du 2026-08-07 » : réécrire à l'état courant — `GOARCH=386` compile depuis le 2026-09-07 (TYP-01, job CI `cross-build`), `TestStateBump_PinnedAcrossCachedCalls` corrigé à la cause (4.1.0) ; garder la trace « était vrai jusqu'au … » plutôt que d'effacer | P4 | `docs/audits/HISTORY.md` | Le fichier ne contredit plus `CHANGELOG.md` 4.1.0 ni `PORTABILITY.md` § 1 | `grep -n 'échoue toujours\|est \*flaky\*' docs/audits/HISTORY.md` vide | S | — |
| EVAL-05 | `docs/PERFORMANCE.md` : retirer les deux tableaux invalidés (Ryzen « historique » et Intel « snapshot ») et leurs encadrés ; conserver une phrase sur l'ordre observé (fast ≤ fft < matrix) avec renvoi au relevé 2026-09-04 ; le tableau des médianes de `bench-baseline.txt` reste seul jusqu'à EVAL-09 | P6, R-5 | `docs/PERFORMANCE.md` | Aucune valeur de durée dans le document ne provient d'un artefact absent de `docs/audits/` | `grep -n '2\.1s\|45s\|62s\|3m12s\|2m10s' docs/PERFORMANCE.md` vide | S | — |
| EVAL-06 | `docs/REFERENCES.md` : liste numérotée, chaque entrée avec DOI ou URL vérifiés à la rédaction — Schönhage & Strassen 1971 (*Computing* 7) ; Karatsuba & Ofman 1962 ; Strassen 1969 (*Numer. Math.* 13) ; Winograd 1971 (*Linear Algebra Appl.* 4) pour la variante à 15 additions ; Takahashi 2000 (*Inf. Process. Lett.* 75) pour le *fast doubling* de grands Fibonacci ; Knuth, *TAOCP* vol. 1 § 1.2.8 (identités, Cassini) ; Harvey & van der Hoeven 2021 (*Ann. of Math.* 193) ; `remyoudompheng/bigfft` ; Go `math/big` ; Shahsavan 2026 (déjà cité par l'audit précédent). Renvois `[n]` depuis `README.md` § Algorithmes, `FAST_DOUBLING.md`, `MATRIX.md`, `FFT.md`, `BIGFFT.md`, `fibonacci_property_test.go` (Cassini) | P8, R-4, C2 | `docs/REFERENCES.md`, `README.md`, `docs/algorithms/*.md`, un commentaire de test | Chaque algorithme documenté cite au moins une référence ; chaque DOI résout | relecture ; `curl -sI https://doi.org/<doi>` → 302 pour chaque entrée | M | — |
| EVAL-08 | Argument d'ordre de grandeur pour les trois seuils, en une phrase chacun, dans `internal/fibonacci/constants.go` et `docs/CALIBRATION.md` § « Calibrated Thresholds » : 4 096 bits ≈ taille où trois produits Karatsuba de 64 mots couvrent le coût de lancement de trois goroutines ; 500 000 bits ≈ 4,5× le seuil interne de `bigfft` (1 800 mots = 115 200 bits) hérité de la calibration amont, marge délibérée ; 3 072 bits ≈ taille où un produit `big.Int` coûte plus que les huit additions/soustractions supplémentaires de Winograd. Étiqueter « argument, pas mesure » | C2 réserve (c) | `internal/fibonacci/constants.go`, `docs/CALIBRATION.md` | Un lecteur sait d'où vient chaque nombre et que rien ne le mesure | relecture | S | — |

### Phase 4 — Filet de tests (½ j, avant la phase 3)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-13 | Test `TestCalculators_AboveDefaultFFTThreshold` : n = 1 500 000 (opérande `FK1` > 500 000 bits sur `fast`, donc `AdaptiveStrategy.ExecuteStep` prend la branche FFT au seuil **par défaut**) ; les trois calculateurs avec `Options{}` normalisées par défaut ; égalité deux à deux ; pas de golden (le corpus est immuable sans ADR) ; `testing.Short()` → `t.Skip` ; commentaire citant EVAL-13 | P9, R-8, C5 réserve (b) | `internal/fibonacci/fft_crossval_test.go` | La branche FFT d'`AdaptiveStrategy.ExecuteStep` est couverte par ce seul test ; durée < 1 s | `go test -run AboveDefaultFFTThreshold -coverprofile=c.out ./internal/fibonacci/ && go tool cover -func=c.out \| grep 'strategy.go.*ExecuteStep'` = 100 % | S | — |
| EVAL-14 | Corpus de fuzz : supprimer les douze fichiers de graines qui répètent un `f.Add` (`FuzzFastDoublingConsistency/*` ×3, `FuzzFFTBasedConsistency/*` ×3, `FuzzProgressMonotonicity/*` ×3, `FuzzFastDoublingMod/seed-boundary`, `FuzzFibonacciIdentities/{seed-edge-m-one,seed-small-pair}`) ; garder les trois qui apportent une valeur ; mettre à jour la colonne « Seeds » et le paragraphe « 12 duplicates » de `docs/TESTING.md` | C5 réserve (c) | `internal/fibonacci/testdata/fuzz/**`, `docs/TESTING.md` | 3 fichiers restants ; relecture des graines verte | `find internal/fibonacci/testdata/fuzz -type f \| wc -l` = 3 ; `go test -run Fuzz ./internal/fibonacci/` | S | — |
| EVAL-12 | Plancher de couverture à la valeur D4 dans les **trois** sites (`scripts/check.sh` `COVERAGE_FLOOR`, `scripts/check.ps1` `$CoverageFloor` + commentaire d'en-tête, `.github/workflows/ci.yml` étape « coverage floor ») ; `docs/TESTING.md` § Coverage et `Makefile` (commentaire `coverage-check`) | P9, R-8, D4 | `scripts/check.sh`, `scripts/check.ps1`, `.github/workflows/ci.yml`, `docs/TESTING.md`, `Makefile` | Une seule valeur dans les trois sites ; gate et CI verts avec la marge annoncée (lire la couverture du job Ubuntu avant de fixer) | `grep -n '80' scripts/check.* .github/workflows/ci.yml \| grep -i cover` vide ; CI verte | S | D4 |
| EVAL-15 | *(écartée)* Couverture de `internal/testutil` (20 %) — un aide de test de 51 lignes ; tester le testeur n'apporte rien | C5 | — | ⊘ motif consigné dans ADR-0013 | — | — | — |

### Phase 3 — Code et mesures (2–3 j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-23 | `-algo gmp` atteignable sous le tag : `calculator_gmp.go` enregistre `"gmp"` dans un point d'extension consulté par `NewDefaultFactory` (une variable de paquet `extraRegistrations []func(*DefaultFactory)` remplie par `init()` sous `//go:build gmp`, vide sinon) ; supprimer la fabrique privée `globalFactory` ; `COMPARISON.md`, `GMP.md`, `ARCH.md` § 5 et § 12 mis à jour ; le job CI `gmp` exécute `go run -tags gmp ./cmd/fibcalc -n 100000 -algo all` et vérifie que quatre lignes sortent | P11, C3 réserve (e), D8 | `internal/fibonacci/{registry,calculator_gmp}.go`, `calculator_gmp_test.go`, `.github/workflows/ci.yml`, `docs/algorithms/{COMPARISON,GMP}.md`, `docs/ARCH.md` | Sans le tag : `List()` inchangée, tests verts ; avec le tag (CI) : `-algo all` compare quatre calculateurs et sort 0 | `go test ./internal/fibonacci/` ; job CI `gmp` vert avec la nouvelle étape | S | D8 |
| EVAL-10 | Selon D1 = (a) : supprimer `internal/fibonacci/threshold/`, `Options.{EnableDynamicThresholds, DynamicAdjustmentInterval, ThresholdTuning}`, `NewDoublingFrameworkWithDynamicThresholds`, le crochet `CacheStrategy` et `cache_strategy_bigfft.go` (seul consommateur : la boucle sous DTM), le drapeau `-dynamic-thresholds` et `FIBCALC_DYNAMIC_THRESHOLDS` (`config`, `env`, `usage`, `completion/registry`, `.env.example`), `config.ThresholdTuning` / `DefaultThresholdTuning` (déplacer `MicroBenchTimeout` en constante de `calibration`), `orchestration.ThresholdTuning`, la plomberie `tuning` de `app` et `tui`, les tests `dtm_*`, `app_tuning_test.go`, `threshold/*` ; la règle 1 de `arch_test.go` (`threshold → config`) disparaît → six règles (mettre à jour `README.md`, `TESTING.md`, `ARCH.md`, `validation-report.md`, `dependency-graph.md` : 48 → 47 arêtes moins celles du nœud `fibthr`) ; ADR-0001 passe en *Superseded by ADR-0013* ; `bench-dtm-2026-09.txt` reste archivé. Selon D1 = (b) : remesurer (`BenchmarkFibonacciDTM`, `-count=10`, deux hôtes, double ordre), archiver, et ne rien supprimer | P7, R-6, C6 réserve (d), C10 réserve (a), D1 | voir tâche | `go list ./... \| grep threshold` vide ; `fibcalc -dynamic-thresholds` → « flag provided but not defined », exit 4 ; `registry_sync_test` vert ; gate et CI verts ; **benchstat double ordre neutre** (le chemin chaud perd un `if dtm != nil` par itération) | `grep -rn 'DynamicThreshold\|ThresholdTuning\|dynamic-thresholds' cmd internal docs README.md .env.example` vide ; benchstat | L | EVAL-13, EVAL-00 |
| EVAL-09 | Courbe d'échelle : sous-benchmarks `100K` et `100M` ajoutés à `BenchmarkFibonacci` pour les trois calculateurs (`-benchtime=1x`, `b.ReportAllocs`) ; artefact `docs/audits/bench-scale-2026-09.txt` produit par `go test -bench='BenchmarkFibonacci/(FastDoubling\|MatrixExp\|FFTBased)' -benchmem -run='^$' -count=10 -benchtime=1x`, précédé de `go version`, `GOMAXPROCS`, modèle de CPU, `git rev-parse HEAD` ; `README.md` § Performance passe à quatre lignes de N ; `PERFORMANCE.md` gagne une section « Échelle » (durée et B/op en fonction de N, pente observée entre 10M et 100M) ; `bench-baseline.txt` n'est **pas** régénéré (ses deux tailles restent comparables) ; `INDEX.md` § artefacts | R-5, C6 réserves (a) et (c) | `internal/fibonacci/fibonacci_test.go` (ou le fichier qui porte `BenchmarkFibonacci`), `docs/audits/bench-scale-2026-09.txt`, `README.md`, `docs/PERFORMANCE.md`, `docs/audits/INDEX.md` | 4 tailles × 3 calculateurs × 10 échantillons dans l'artefact ; chaque chiffre du README renvoie à l'artefact ; `benchstat docs/audits/bench-baseline.txt` reste lisible sur les sous-tests communs | `grep -c 'BenchmarkFibonacci/' docs/audits/bench-scale-2026-09.txt` = 120 ; `head -5` montre version et hôte | M | EVAL-10 (mesurer le code final) |
| EVAL-07 | Mesure externe de référence (D8 = a) : le job CI `gmp` exécute `go test -tags gmp -bench='BenchmarkFibonacci/FastDoubling\|BenchmarkGMPCalculator' -benchmem -run='^$' -count=5 -benchtime=1x ./internal/fibonacci/` et téléverse la sortie en artefact ; le premier relevé est copié dans `docs/audits/bench-gmp-2026-09.txt` avec l'identifiant du *run* ; `README.md` gagne un paragraphe « Positionnement » (ratio fast ÷ gmp à F(1M) et F(10M), en précisant que `GMPCalculator` est la même boucle de doublement sur `mpz`, pas `mpz_fib_ui`) ; `docs/algorithms/COMPARISON.md` § Travaux connexes cite GMP `mpz_fib_ui` et PARI/GP comme références non mesurées | P8, R-4, C1 réserve (a) | `.github/workflows/ci.yml`, `docs/audits/bench-gmp-2026-09.txt`, `README.md`, `docs/algorithms/COMPARISON.md`, `docs/audits/INDEX.md` | Un ratio mesuré sur un runner identifié ; l'artefact est rejouable par le job | job CI `gmp` vert avec artefact ; `test -f docs/audits/bench-gmp-2026-09.txt` | M | EVAL-23 |
| EVAL-11 | *(écartée, sauf demande)* Job CI de benchmark de non-régression — les runners partagés produisent un bruit supérieur au seuil de 5 % ; l'ancien job informatif (A-14) avait été retiré pour cette raison. Garder la discipline locale de `PERFORMANCE.md` § Regression baseline | C6 réserve (f) | — | ⊘ motif dans ADR-0013 ; à revoir si un runner dédié devient disponible | — | — | — |

### Phase 5 — Hygiène et processus (½ j, + L si D2 = a)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-19 | Selon D2 = (b) : amender `CONTRIBUTING.md` § 4 et ADR-0012 D4 (addendum daté, décision d'origine conservée) ; rendre `docs/ARCH.md` monolingue (anglais, puisque les figures qu'il commente et 60 % de son texte le sont ; ses encadrés et § 0 français traduits) ; `README.md` § Développement renvoie à la règle. Selon D2 = (a) : traduire les treize documents, un commit par document, relecture croisée avec le code pour chaque affirmation traduite | P5, R-7 | `CONTRIBUTING.md`, `docs/adr/0012-*.md`, `docs/ARCH.md` (b) ; `docs/*.md`, `docs/algorithms/*.md`, `docs/architecture/**` (a) | La règle écrite décrit le corpus ; `ARCH.md` ne mélange plus | (b) : décompte de marqueurs de langue par document, un seul groupe dominant ; (a) : relecture humaine | S + M (b) / L (a) | D2 |
| EVAL-16 | Étiquettes `rewrite/*` selon D5 = (a) : sauvegarde `git for-each-ref refs/tags/rewrite > <scratchpad>/tags-rewrite.txt` ; `git tag -d` ×15 ; **avec l'accord explicite du mainteneur**, `git push origin --delete` ×15 ; ligne dans `HISTORY.md` (2026-07, « points de sauvegarde de la réécriture, retirés le … ») | P10, D5 | `docs/audits/HISTORY.md` | Aucune étiquette `rewrite/*` locale ni distante ; la sauvegarde existe | `git tag \| grep -c rewrite` = 0 ; `git ls-remote --tags origin \| grep -c rewrite` = 0 | S | D5, accord explicite |
| EVAL-17 | *Worktree* selon D6 = (a) : `git -C .claude/worktrees/lucid-nightingale-3e810f status` et `git log main..claude/determined-carson-0df1c3` ; si propre et sans commit non fusionné utile → `git worktree remove` puis `git branch -D` ; sinon rapporter et laisser | P10, D6 | — | `git worktree list` n'affiche que `main` ; rien de non commité perdu | `git worktree list \| wc -l` = 1 | S | D6 |
| EVAL-18 | *(écartée)* Entrées `CHANGELOG.md` pour `v2.x` / `v3.0.0` — la note en tête de 4.0.0 dit déjà qu'elles n'existent pas et pourquoi ; reconstruire depuis `git log` coûte une journée pour un lecteur qui a `git log` | P10 | — | ⊘ motif dans ADR-0013 | — | — | — |
| EVAL-20 | *(opt.)* Règle de revue dans `CONTRIBUTING.md` § Pull Request Process : tant que le projet n'a qu'un mainteneur, toute PR reste ouverte au moins jusqu'à la CI verte des cinq jobs et à une relecture du diff hors de l'outil qui l'a produit ; dès un second contributeur, une approbation humaine par PR | C8 réserve (b) | `CONTRIBUTING.md` | La règle est écrite et datée | relecture | S | — |

### Phase 6 — Perspectives (optionnelles, 2–3 j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-21 | *(opt.)* Repli Go pur pour `arith_decl.go` : `arith_purego.go` sous `//go:build purego` implémentant `addVV`, `subVV`, `addVW`, `subVW`, `shlVU`, `addMulVVW` avec `math/bits` ; `arith_decl.go` sous `//go:build !purego` ; job CI `cross-build` ajoute `go test -tags purego ./internal/bigfft/` ; `PORTABILITY.md` § 2.1 ; `benchstat` du tag par défaut inchangé (le repli ne touche pas le chemin normal) | P11, C4 réserve (a), C10 réserve (c) | `internal/bigfft/arith_{decl,purego}.go`, `.github/workflows/ci.yml`, `docs/PORTABILITY.md` | `go test -tags purego ./internal/bigfft/` vert ; `FuzzMul`/`FuzzSqr` verts sous le tag | CI ; `go vet -tags purego ./...` | M | — |
| EVAL-22 | *(opt.)* Passe de commentaires : par paquet, retirer la narration d'historique (« used to be », « the previous … was ») quand `CHANGELOG.md` ou `INDEX.md` la porte déjà ; garder le *pourquoi* et l'identifiant ; règle « on retire la phrase d'histoire, jamais la phrase de raison » ; relecture par diff | P11, C4 réserve (b), C7 réserve (d) | `internal/**/*.go` | Densité de commentaires en baisse mesurée ; aucun `TestAuditIdentifiersResolve` rouge ; aucune justification perdue (relecture) | `grep -rc 'used to\|previously\|the previous' --include=*.go internal \| awk -F: '{s+=$2} END {print s}'` en baisse d'au moins moitié | L | à faire paquet par paquet, quand on y touche |
| EVAL-24 | *(écartée, sauf mesure)* Unifier les deux sémaphores (`GOMAXPROCS(0)` côté `fibonacci`, `NumCPU()` côté `bigfft`) — aucune mesure de contention n'existe ; sans elle, c'est un changement de chemin chaud sans motif | P11, C3 réserve (c) | — | ⊘ à revoir si un profil montre du sur-abonnement | — | — | — |

### Phase 7 — Clôture (½ j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| EVAL-25 | ADR-0013 en *Accepted* (décisions, tâches écartées et leur motif, résultats `benchstat` d'EVAL-10) ; `CHANGELOG.md` : section de version selon D7, avec « Vérification » (gate rejoué, couverture, benchstat) ; `docs/audits/HISTORY.md` ligne 2026-09-15 ; `README.md` « État vérifié le … » re-daté ; `docs/audits/evaluation-academique-2026-09-15.md` gagne un § « Suivi » renvoyant au § 8 ci-dessous ; étiquette de version créée localement — **la poussée de la branche, de l'étiquette et la fusion dans `main` sont demandées au mainteneur** | — | `docs/adr/0013-*.md`, `CHANGELOG.md`, `docs/audits/{HISTORY,evaluation-academique-2026-09-15}.md`, `README.md` | Chaque affirmation nouvelle rattachée à une commande rejouée ; CI verte sur la branche | relecture ; `gh run list --branch eval/2026-09-15` | S | tout |

---

## 4. Protocole de vérification

Après chaque tâche :

```bash
pwsh scripts/check.ps1        # ou : bash scripts/check.sh (WSL)
```

Ce que le script enchaîne : `go build`, `go vet`, `go test -race -shuffle=on
-count=1 -coverprofile`, `golangci-lint` (épinglé), plancher de couverture,
`govulncheck`. Puis la CI sur la branche (cinq jobs).

Pour EVAL-10, EVAL-13, EVAL-21, EVAL-23 (chemin chaud ou paquet `bigfft`) :

```bash
go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -run='^$' -count=8 ./internal/fibonacci/ > new.txt
benchstat base.txt new.txt     # puis l'ordre inverse, même session (ADR-0009 R4)
```

Critère : aucune régression > 5 % confirmée dans les deux ordres. Un écart qui
s'inverse avec l'ordre est du bruit et se consigne comme tel.

Pour EVAL-01 (aucune ligne de code ne doit bouger) :

```bash
git diff --stat main -- '*.go'        # seuls des commentaires
go build ./... && go test ./internal/bigfft/
```

---

## 5. Risques et retours arrière

| Tâche | Risque | Mitigation / retour arrière |
|---|---|---|
| EVAL-01 | Formulation de licence incorrecte (attribuer aux Go Authors du code nouveau, ou l'inverse) | Notice amont recopiée sans modification ; `NOTICE` distingue « dérivé de » et « modifications » ; relecture du mainteneur avant commit |
| EVAL-06 | Référence bibliographique fausse (mauvais volume, DOI invalide) | Chaque DOI résolu par `curl -sI` à la rédaction ; pas de référence sans identifiant vérifié |
| EVAL-10 | Diff large et chemin chaud (la boucle de doublement perd une branche) ; oubli d'un consommateur (`tui`, `completion`, `.env.example`) | Filet EVAL-13 posé avant ; `registry_sync_test` et `env` sync tests attrapent le drapeau oublié ; benchstat double ordre ; un commit par sous-ensemble (paquet, drapeau, docs) pour un revert isolé |
| EVAL-09 | Bruit thermique sur `100M` (une itération de 0,6 à 2 s) ; artefact non comparable | `-count=10`, machine au repos, version Go et hôte dans l'en-tête ; l'artefact est une courbe, pas une baseline de gate |
| EVAL-12 | Plancher trop près de la mesure sur une plateforme | Lire la couverture Ubuntu de la CI avant de fixer ; garder 4 points de marge |
| EVAL-16 | Suppression distante irréversible | Sauvegarde `for-each-ref` ; accord explicite ; les commits restent joignables tant que la sauvegarde existe (`git tag` recréable depuis le SHA) |
| EVAL-17 | Effacer du travail en cours | `status` et `log main..branche` lus d'abord ; en cas de doute, ne rien supprimer et rapporter |
| EVAL-19 (a) | Traduction qui altère une affirmation technique | Un commit par document ; relecture croisée avec la source pour chaque chiffre et chaque nom de symbole |
| EVAL-22 | Perdre une justification en retirant une phrase d'histoire | Règle « jamais la phrase de raison » ; relecture par diff ; paquet par paquet |
| EVAL-23 | Enregistrement sous tag qui casse `List()` sans tag | Test sans tag inchangé ; job CI `gmp` exerce l'autre branche |

---

## 6. Définition de terminé

- EVAL-01 fusionnée : c'est la condition de recevabilité R3 ; aucune autre
  tâche ne compte tant qu'elle n'est pas faite.
- Toutes les tâches non optionnelles fermées, chaque commit vert au gate et
  en CI ; les tâches écartées (EVAL-11, 15, 18, 24) et optionnelles non prises
  (EVAL-20, 21, 22) listées dans ADR-0013 avec leur motif.
- ADR-0013 en *Accepted* ; ADR-0001 en *Superseded* si D1 = (a).
- Aucune régression `benchstat` > 5 % en double ordre.
- Les critères de l'évaluation qui portaient ces constats sont re-notés dans le
  § « Suivi » de l'évaluation, avec la preuve rejouée — pas de score
  réattribué sans commande.

---

## 7. Effort total estimé

| Phase | Effort |
|---|---|
| 0 — Cadrage | ½ j |
| 1 — Recevabilité | ½ j |
| 2 — Exactitude des documents | 1 j |
| 4 — Filet de tests | ½ j |
| 3 — Code et mesures (EVAL-23 S, EVAL-10 L, EVAL-09 M, EVAL-07 M) | 2–3 j |
| 5 — Hygiène (D2 = b) | ½ j (+ 3 j si D2 = a) |
| 6 — Optionnelles (EVAL-21, EVAL-22) | + 2–3 j |
| 7 — Clôture | ½ j |
| **Total hors optionnelles** | **5½ à 6½ jours-personne** |

Chemin critique : EVAL-00 → EVAL-01 → EVAL-13 → EVAL-10 → EVAL-09 → EVAL-25.
La phase 2 et EVAL-23 / EVAL-07 se mènent en parallèle du chemin critique.

---

## 8. Tableau de suivi d'exécution

Mis à jour à chaque tâche. Statuts : ☐ à faire · ⟳ en cours · ☑ terminée et
vérifiée · ⊘ écartée (motif consigné).

| ID | Statut | Commit | Vérification rejouée | Note |
|---|---|---|---|---|
| EVAL-00 | ☑ | `aba5202` | `TestAuditIdentifiersResolve` vert avec `EVAL-00`, rouge avec un préfixe inconnu | D1–D8 : recommandations retenues (mainteneur, 2026-09-23) |
| EVAL-01 | ☑ | `498bc2c` | `cmp` LICENSE = module amont ; AST sans commentaires identique (7 fichiers) ; `go test ./internal/bigfft/` | + `LICENSE`/`NOTICE` dans l'image Docker |
| EVAL-02 | ☑ | `5a6552e` | `grep 'O(n log n)'` : ne reste que Harvey–van der Hoeven | la borne prévue était fausse : FFT à un niveau, Θ(n^1,585) |
| EVAL-03 | ☑ | `d81edd7` | `grep` des quatre lignes vide | |
| EVAL-04 | ☑ | `8527827` | `grep 'échoue toujours\|est \*flaky\*'` vide | |
| EVAL-05 | ☑ | `35a1073` | `grep` des durées historiques vide | |
| EVAL-06 | ☑ | `5a6552e` | 15 DOI/URL résolus (302) et contrôlés sur Crossref | sections de livres non revérifiées, dit |
| EVAL-07 | ☑ | `4c78cbe`, `2f487d9` | `bench-gmp-2026-09.txt` (WSL2) ; étape CI `gmp` ajoutée | relevé local, pas de run CI (branche non poussée) |
| EVAL-08 | ☑ | `93f2a16` | critique à l'aveugle gagné ; chiffres rapportés au code | deux déductions du plan réfutées ; la mesure contredit deux défauts |
| EVAL-09 | ☑ | `751c1cc`, `4c78cbe`, `2f487d9` | `grep -c '^BenchmarkFibonacci/'` = 120 | `grep -c 'BenchmarkFibonacci/'` donne 121 : la ligne `# command:` compte |
| EVAL-10 | ☑ | `8ce097d` | `go list \| grep threshold` vide ; `-dynamic-thresholds` → exit 4 ; benchstat neutre une fois l'ordre équilibré | 48 → 45 arêtes (le plan disait 47) ; logger TUI réparé |
| EVAL-11 | ⊘ | | | runners bruyants ; à revoir si runner dédié |
| EVAL-12 | ☑ | `4656ff2` | trois sites à 90 ; gate à 95,9 % | 90 %, non 92 % : CI Ubuntu à 94,0 % |
| EVAL-13 | ☑ | `54cc9d9`, `827b390` | `ExecuteStep` (strategy.go:99) 66,7 % à n = 1M → 100 % à 1,5M | |
| EVAL-14 | ☑ | `10a18bf` | 3 fichiers restants ; 51 sous-tests, 51 entrées | |
| EVAL-15 | ⊘ | | | aide de test de 51 lignes |
| EVAL-16 | ⟳ | | `git tag \| grep -c rewrite` = 0 (local) | 18 étiquettes, sauvegardées ; `origin` : accord du mainteneur attendu |
| EVAL-17 | ⊘ | | `git -C … status` : une modification non commitée | worktree non propre : laissé, décision au mainteneur |
| EVAL-18 | ⊘ | | | déjà dit en tête de 4.0.0 |
| EVAL-19 | ☑ | `fda4362`, `30ef357` | marqueurs de langue : `docs/` à 0 marqueur français | 15 documents traduits en plus d'`ARCH.md` |
| EVAL-20 | ☑ | `fb84a2a` | relecture | |
| EVAL-21 | ☑ | `8a9ff79` | `go test -tags purego ./internal/bigfft/` : 153 tests ; 386/arm64 compilent | |
| EVAL-22 | ☑ | `f1dd229` | décompte 93 → 29 ; AST sans commentaires identique (44 fichiers) | relecture indépendante : aucune raison perdue |
| EVAL-23 | ☑ | `c4b4c99` | WSL2 : `-algo all` sort quatre `Success` | |
| EVAL-24 | ⊘ | | | sans mesure de contention |
| EVAL-25 | ☑ | (commit de clôture) | gate final ; comparaison à l'aveugle `main` / branche | poussée, fusion et étiquette distante : mainteneur |
