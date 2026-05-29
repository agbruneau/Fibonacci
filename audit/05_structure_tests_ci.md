# Audit — Axe 5 : Structure projet, Tests & CI

> Commit audité : `866b8cdcdde5256bd78db260ed5434e1837d86ec` (2026-05-24) — Audit : 2026-05-28
> Go `1.26.3` (module cible `go 1.26.0`). Hôte : Windows 11, `CGO_ENABLED=0` (pas de `-race`, pas de backend `gmp`).

## Verdict d'axe

La structure du dépôt est saine et fidèle à la Clean Architecture annoncée : layout `cmd/`+`internal/`+`test/` cohérent, **gate d'architecture exécutable** (`internal/arch_test.go`), couverture **totale réelle de 88,6 %** (au-dessus du seuil 80 %), adoption `t.Parallel()` quasi systématique (940 appels / 113 fichiers de test), et tous les types de tests revendiqués sont présents (unit, table-driven, 7 fuzz, property gopter, golden, e2e). Aucune CI distante (`.github/` absent) — **confirmé** ; toute la rigueur repose sur la discipline locale. Deux problèmes méritent action : (1) un **test flaky sous Windows** (`TestSaveProfile_NeverObservablyTruncated`) qui échoue de façon non déterministe sous charge, et (2) plusieurs **dérives documentaires de version** (badge « Go 1.25+ », « toolchain 1.26.2 », couverture « 87,5 % », path module `agbru/fibcalc` ≠ dépôt `agbruneau/FibGo`). Aucun défaut CRITIQUE de structure.

## Tableau récapitulatif

| ID | Sévérité | Titre | Marqueur |
|---|---|---|---|
| A5-01 | MAJEUR | Test flaky sous Windows : `TestSaveProfile_NeverObservablyTruncated` échoue sous charge | [confirmé] |
| A5-02 | MAJEUR | Absence de CI distante : aucun garde-fou automatisé (vet/lint/race/coverage/build) | [confirmé] |
| A5-03 | MINEUR | Dérive de version documentaire (README « Go 1.25+ », « toolchain 1.26.2 ») vs `go.mod` 1.26.0 / réel 1.26.3 | [confirmé] |
| A5-04 | MINEUR | Couverture documentée « 87,5 % » obsolète vs mesure réelle 88,6 % | [confirmé] |
| A5-05 | MINEUR | `make test`/README revendiquent `-race`, indisponible sur l'hôte Windows documenté (CGO requis) | [confirmé] |
| A5-06 | MINEUR | Incohérence path module `github.com/agbru/fibcalc` vs dépôt GitHub `agbruneau/FibGo` | [confirmé] |
| A5-07 | INFORMATIF | `.golangci.yml` en schéma v1 (legacy) ; `gosimple` fusionné dans `staticcheck` en v2 | [confirmé] |
| A5-08 | INFORMATIF | Angles morts de couverture : e2e (`[no statements]`) et backend `gmp` non bâtis par défaut | [confirmé] |
| A5-09 | INFORMATIF | `cmd/generate-golden` à 29,4 % de couverture (outil oracle de dév) | [confirmé] |
| A5-10 | INFORMATIF | Aucune assertion automatisée du plancher de couverture (le seuil 80 % n'est que documentaire) | [confirmé] |

---

## Détail des constats

### [A5-01] Test flaky sous Windows : `TestSaveProfile_NeverObservablyTruncated`
- **Sévérité** : MAJEUR
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `internal/calibration/profile_test.go:219` (test) ; `internal/calibration/profile.go:175` (`renameAtomic`)
- **Preuve** : Sortie d'un `go test -count=1 ./...` (premier passage), le test échoue ; un second passage immédiat réussit :

  ```
  === RUN 1 EXIT: 1 ===
  --- FAIL: TestSaveProfile_NeverObservablyTruncated (0.89s)
      profile_test.go:274: rewrite SaveProfile(40) failed: failed to finalize profile:
      rename C:\...\profile.json.tmp-3120000235 C:\...\profile.json: Access is denied.
  FAIL    github.com/agbru/fibcalc/internal/calibration   3.524s
  === RUN 2 EXIT: 0 ===
  ```

  Le retry de `renameAtomic` est **borné** (10 tentatives, sleeps cumulés ~225 ms max) :

  ```go
  func renameAtomic(src, dst string) error {
  	const maxAttempts = 10
  	var err error
  	for attempt := 0; attempt < maxAttempts; attempt++ {
  		if err = os.Rename(src, dst); err == nil {
  			return nil
  		}
  		if attempt < maxAttempts-1 {
  			time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
  		}
  	}
  	return err
  }
  ```

  Le test lance un lecteur en boucle serrée sur la destination pendant 80 réécritures concurrentes ; le commentaire du test (`profile_test.go:214-217`) ne reconnaît la fenêtre de partage Windows que **côté lecteur** (« the reader treats such transient OS access errors as retryable »), pas l'épuisement du retry **côté écrivain**.
- **Impact** : Échec non déterministe de `make test` / `go test ./...` sur l'hôte Windows officiellement supporté. En l'absence de CI, ce flake détériore directement le seul garde-fou existant (la validation locale) : un contributeur peut voir un FAIL rouge sans régression réelle, ou pire, ignorer un vrai échec par habitude du flake. L'invariant testé (atomicité A-11) reste correct ; c'est la robustesse du test sous contention Windows qui est en cause.
- **Recommandation** : Stabiliser le test sous Windows sans toucher l'invariant : soit augmenter `maxAttempts`/le backoff de `renameAtomic` (proposition d'ADR si cela touche le contrat de l'écrivain), soit — plus chirurgical et hors invariant — tolérer côté **test** l'erreur de partage Windows de `SaveProfile` (la traiter comme retryable au même titre que l'erreur de lecture, conformément à la note de portabilité déjà présente), ou réduire l'agressivité du lecteur (petit `time.Sleep` dans la boucle de lecture) pour fermer la fenêtre. Reconfirmer le taux de flake sous charge sur runner Windows CI.
- **Marqueur** : [confirmé]

---

### [A5-02] Absence de CI distante : aucun garde-fou automatisé
- **Sévérité** : MAJEUR
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : racine du dépôt (`.github/` inexistant) ; `README.md:234`, `CHANGELOG.md:82`
- **Preuve** :

  ```
  $ ls -la .github
  ls: cannot access '.../.github': No such file or directory
  ```

  Le retrait est documenté (`CHANGELOG.md:82`) : « **GitHub Actions workflows deleted** : `.github/workflows/ci.yml` (vet + golangci-lint + 3-OS race matrix + cross-compile + bench gate) and `.github/workflows/coverage.yml` (… `MIN_COVERAGE=80%` floor) … are gone as well. » README.md:234 : « **No remote CI/CD** … verification … is **the contributor's responsibility, run locally** ».
- **Impact** : Aucune barrière automatisée n'empêche la fusion d'un code non formaté, d'une régression de couverture, d'une violation d'architecture, d'une data race (le seul moyen de détecter une race ici est `-race`, qui exige un runner Linux/CGO) ou d'une régression perf > 5 %. La discipline « tout en local » est fragile : `-race` est indisponible sur l'hôte Windows du contributeur principal (cf. A5-05), donc en pratique la matrice de race n'est jamais rejouée. Combiné à A5-01 (flake local), le risque de régression silencieuse est réel.
- **Recommandation** : Réintroduire une CI GitHub Actions minimale et reproductible :
  - **matrice Go** `1.25.x` + `1.26.x` (le module cible `go 1.26.0`, mais valider la limite basse documentée) ;
  - job **Linux** avec `CGO_ENABLED=1` exécutant `go test -race -cover ./...` (seul environnement où `-race` est disponible) ;
  - job **lint** : `golangci-lint` épinglé à une version reproductible (cf. historique A-12 dans `CHANGELOG.md:117` : `v1.64.8` était déjà épinglé) ;
  - **plancher de couverture** réintroduit (`MIN_COVERAGE=80%`, cf. A5-10) ;
  - **build-all** (cross-compile linux/windows/darwin × amd64/arm64) pour garder les fallbacks `PORTABILITY.md` verts ;
  - job optionnel **`-tags=gmp`** sur runner Linux avec `libgmp-dev` (le seul moyen de compiler/tester ce backend, cf. A5-08).
- **Marqueur** : [confirmé]

---

### [A5-03] Dérive de version documentaire
- **Sévérité** : MINEUR
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `README.md:3`, `README.md:35`, `README.md:233` ; `CLAUDE.md` (« Go : 1.25.0+ (toolchain 1.26.2) ») ; `go.mod:3`
- **Preuve** :

  ```
  README.md:3  [![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?...)]
  README.md:35 Requires **Go 1.25** or later.
  go.mod:3     go 1.26.0
  ```

  Toolchain réelle mesurée au bootstrap : `go1.26.3`. CLAUDE.md annonce « toolchain 1.26.2 ».
- **Impact** : Un contributeur installant Go 1.25 (comme l'invite le badge et le « Quick Start ») ne pourra pas compiler le module (`go.mod` exige `go 1.26.0`). La documentation sous-déclare la version minimale et fige une toolchain (1.26.2) déjà dépassée.
- **Recommandation** : Aligner le badge et le texte « Requires Go 1.25 » sur `go 1.26.0` (badge `Go-1.26+`), et mettre à jour la mention « toolchain 1.26.2 » de CLAUDE.md. Si la rétro-compat 1.25 est réellement voulue, abaisser la directive `go.mod` ; sinon corriger la doc.
- **Marqueur** : [confirmé]

---

### [A5-04] Couverture documentée « 87,5 % » obsolète
- **Sévérité** : MINEUR
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `CHANGELOG.md:113` (« badge coverage updated to 87.5% »)
- **Preuve** : Mesure réelle agrégée sur tout le module :

  ```
  $ go test -coverprofile=audit/cover.out ./... && go tool cover -func=audit/cover.out | tail -1
  total:    (statements)    88.6%
  ```

  Détail des extrêmes par package : `fibonaccitest`/`metrics/system`/`parallel`/`testutil` = 100 % ; `internal/fibonacci` = 88,6 % ; `internal/bigfft` = 86,4 % ; `cmd/generate-golden` = 29,4 % (cf. A5-09) ; `internal/tui/component` = `[no test files]`.
- **Impact** : Dérive mineure : la doc sous-évalue la couverture réelle (88,6 % > 87,5 %). Aucun risque de correction, mais la valeur affichée n'est plus exacte ; sans plancher automatisé (A5-10), rien ne resynchronise ce chiffre.
- **Recommandation** : Mettre à jour la valeur (ou, mieux, supprimer le chiffre figé et pointer vers `make coverage`, déjà la source canonique mentionnée dans `docs/TESTING.md:277`).
- **Marqueur** : [confirmé]

---

### [A5-05] `-race` revendiqué mais indisponible sur l'hôte Windows documenté
- **Sévérité** : MINEUR
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `Makefile:162` (`test: $(GO) test -v -race -cover ./...`) ; `README.md:255`, `README.md:339`
- **Preuve** :

  ```
  Makefile:162   $(GO) test -v -race -cover ./...
  README.md:339  go test -v -race -cover ./...                # all tests
  ```

  Bootstrap §2 : `go env CGO_ENABLED` = `0`, `gcc`/`clang` introuvables → `go test -race` échoue immédiatement sur l'hôte. CLAUDE.md le reconnaît (« sous Windows la validation `-race` se fait via WSL ou un autre poste »), mais `make test` reste la commande de référence pré-commit (directive #8).
- **Impact** : La commande de validation canonique (`make test`) ne peut pas s'exécuter telle quelle sur l'hôte Windows principal. Combiné à l'absence de CI (A5-02), la matrice de détection de races n'est jamais exécutée en pratique — exactement le scénario où une race comme celles parquées (cf. mémoire `bigfft-concurrency-archived-tag`) reste indétectée.
- **Recommandation** : Documenter explicitement dans le README/Makefile que `make test` (race) requiert WSL/Linux ou un runner CGO, et fournir une cible `make test-norace` (ou `make test-win`) pour la validation locale Windows sans `-race`. Idéalement, déléguer `-race` à la CI Linux (A5-02).
- **Marqueur** : [confirmé]

---

### [A5-06] Incohérence path module vs dépôt GitHub
- **Sévérité** : MINEUR
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `go.mod:1` ; `git remote` ; `README.md:38`
- **Preuve** :

  ```
  go.mod:1     module github.com/agbru/fibcalc
  $ git remote -v
  origin    https://github.com/agbruneau/FibGo.git (fetch/push)
  README.md:38 git clone https://github.com/agbruneau/FibGo.git
  ```

  *Nuance vs énoncé* : l'énoncé évoquait un « badge CI pointant vers `agbru/fibcalc` ». **Infirmé** : il n'existe **aucun badge CI** dans le README (les 4 badges sont Go-Version, License, Status, Dashboard). Le seul écart `agbru/fibcalc` ↔ `agbruneau/FibGo` réel est le **path du module** vs le dépôt GitHub.
- **Impact** : Le path du module (`github.com/agbru/fibcalc`) ne résout pas vers le dépôt réel (`github.com/agbruneau/FibGo`). Pour un prototype académique « internal-only » (aucun package exporté hors `internal/`), l'impact pratique est nul : le module n'est pas destiné à être `go get`. Mais c'est un piège pour quiconque tenterait d'importer ou de vendoriser le module via son path déclaré.
- **Recommandation** : Soit aligner le path module sur `github.com/agbruneau/FibGo` (changement transversal, à justifier en commit), soit documenter explicitement que le path module est un identifiant interne sans correspondance dépôt. À traiter comme cosmétique tant que le module reste privé.
- **Marqueur** : [confirmé]

---

### [A5-07] `.golangci.yml` en schéma v1 (legacy)
- **Sévérité** : INFORMATIF
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `.golangci.yml:11`, `.golangci.yml:18`
- **Preuve** :

  ```yaml
  linters:
    disable-all: true     # clé de schéma v1 ; en v2 c'est `default: none`
    enable:
      - gofmt
      - gosimple          # fusionné dans staticcheck depuis golangci-lint v2
  ```

  Le décompte des `enable:` donne bien **24 linters** (claim CLAUDE.md/README confirmé). Bootstrap §5 : les binaires `staticcheck`/`golangci-lint` préinstallés (build go1.25) refusent go1.26 et doivent être recompilés ; `govulncheck` en PATH présente le même refus (`package requires newer Go version go1.26 (application built with go1.25)`).
- **Impact** : La config reste fonctionnelle avec `golangci-lint v1.64.8` (recompilé localement), mais elle est incompatible avec la branche v2 courante (`disable-all`→`default: none`, `gosimple` retiré). À terme, toute migration v2 exigera une réécriture de la config. C'est une dette de modernisation outillage, pas un défaut de structure.
- **Recommandation** : Planifier une migration `.golangci.yml` vers le schéma v2 (et figer la version de `golangci-lint` dans la CI A5-02 pour reproductibilité), ou documenter explicitement le verrouillage sur la branche v1.
- **Marqueur** : [confirmé]

---

### [A5-08] Angles morts de couverture : e2e et backend `gmp`
- **Sévérité** : INFORMATIF
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `test/e2e/` ; `internal/fibonacci/calculator_gmp.go:1` (`//go:build gmp`)
- **Preuve** : Les tests e2e exécutent le binaire en sous-processus (boîte noire) → non instrumentés :

  ```
  ok  github.com/agbru/fibcalc/test/e2e   2.714s   coverage: [no statements]
  ```

  Le backend `gmp` ne compile pas sur l'hôte (CGO requis) :

  ```
  $ go build -tags=gmp ./internal/fibonacci/
  internal\fibonacci\calculator_gmp.go:74:40: undefined: gmp.Int
  ...
  BUILD EXIT: 1
  ```
- **Impact** : Les 88,6 % de couverture statement **n'incluent ni** les chemins exercés par les e2e (le binaire CLI complet est validé en boîte noire, mais sa couverture n'est pas comptée) **ni** le backend `gmp` (jamais bâti par défaut, 0 % sur l'hôte). La couverture réelle des chemins CLI est donc sous-évaluée d'un côté et le backend `gmp` est entièrement non testé sur l'environnement courant.
- **Recommandation** : (1) Pour mesurer les e2e, utiliser `go build -cover` (Go 1.20+) afin d'instrumenter le binaire sous-processus et agréger via `GOCOVERDIR`. (2) Tester `gmp` sur un runner Linux avec `libgmp-dev` (cf. A5-02). Documenter ces deux angles morts dans `docs/TESTING.md`.
- **Marqueur** : [confirmé]

---

### [A5-09] `cmd/generate-golden` à 29,4 % de couverture
- **Sévérité** : INFORMATIF
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `cmd/generate-golden/main.go:22` (`main` à 0,0 %)
- **Preuve** :

  ```
  ok  github.com/agbru/fibcalc/cmd/generate-golden   4.643s   coverage: 29.4% of statements
  cmd/generate-golden/main.go:22:  main   0.0%
  ```
- **Impact** : Faible et attendu : il s'agit de l'oracle indépendant servant à régénérer le golden file (outil de développement, pas de chemin de production). Son `main` n'est pas couvert, ce qui est normal pour un point d'entrée CLI. Le risque est qu'un bug dans cet oracle invaliderait silencieusement le golden — mais le golden est immuable sans ADR, donc le risque est gelé.
- **Recommandation** : Aucune action requise ; au plus, documenter dans `docs/TESTING.md` que `generate-golden` est un outil de dév délibérément peu couvert (ne pas le faire entrer dans un éventuel plancher de couverture par package).
- **Marqueur** : [confirmé]

---

### [A5-10] Aucune assertion automatisée du plancher de couverture
- **Sévérité** : INFORMATIF
- **Axe** : 5 Structure/Tests/CI
- **Emplacement** : `Makefile:170` (cible `coverage`) ; `README.md:313`, `docs/TESTING.md:277`
- **Preuve** : La cible `make coverage` génère seulement un rapport HTML, sans contrôle de seuil :

  ```
  coverage:
  	$(GO) test -coverprofile=coverage.out ./...
  	$(GO) tool cover -html=coverage.out -o coverage.html
  ```

  Le seuil « ≥ 80 % » (`README.md:313`, `docs/TESTING.md:277`) n'est qu'une **cible documentaire**. L'historique (`CHANGELOG.md:82,136`) confirme que le plancher `MIN_COVERAGE=80%` vivait dans `coverage.yml`, désormais supprimé.
- **Impact** : Rien n'empêche la couverture de descendre sous 80 % : le « plancher » est purement déclaratif depuis le retrait de la CI. Conjugué à A5-04 (chiffre figé obsolète), la métrique de couverture n'est plus gouvernée.
- **Recommandation** : Réintroduire l'assertion de plancher, soit dans la CI (A5-02), soit dans une cible `make coverage-check` qui parse `go tool cover -func` et échoue si `total < 80%`. Cela referme le garde-fou perdu avec `coverage.yml`.
- **Marqueur** : [confirmé]

---

## Annexe — Inventaire des types de tests (revendications vérifiées)

| Type | Présence | Vérification |
|---|---|---|
| Unit / table-driven | Oui, dominant | 765 fonctions `Test*/Benchmark*/Fuzz*/Example*` sur 113 fichiers (`go test ./...` vert) |
| `t.Parallel()` | Adoption très élevée | 940 appels sur 97 fichiers (sous-tests inclus) — quasi systématique, conforme à la cible 100 % |
| Fuzz | 7 cibles | 5 dans `fibonacci/fibonacci_fuzz_test.go` + 2 dans `bigfft/fft_fuzz_test.go` (`FuzzMul`, `FuzzSqr`) — conforme |
| Property (gopter) | Oui | `internal/fibonacci/fibonacci_property_test.go` |
| Golden | Oui (immuable) | `internal/fibonacci/testdata/fibonacci_golden.json` |
| E2E | Oui (gated `-short`) | `test/e2e/cli_e2e_test.go`, `extended_e2e_test.go` (boîte noire, `[no statements]`) |
| Gate d'architecture | Oui (exécutable) | `internal/arch_test.go` — `TestArchitectureLayering`, 3 arrows remontants gardés |

**Layout vs Clean Architecture 4 couches** : conforme. `cmd/` (2 entrypoints) → `internal/app` → `internal/orchestration` → `internal/fibonacci`+`internal/bigfft` → `internal/config`/`internal/errors`. Les leaves (`format`, `metrics`, `parallel`, `ui`, `testutil`) sont bien des feuilles. Le gate `arch_test.go` valide les 3 inversions interdites au runtime via `go list`. 25 packages ; 17 655 LOC de production / 24 996 LOC de test (ratio test:prod ≈ 1,42:1, sain).

**État `go.mod`/`go.sum`** : `go mod verify` → `all modules verified`. Dépendances directes cohérentes (errgroup, gopter, bubbletea, zerolog, gopsutil, ncw/gmp). Aucune incohérence détectée hormis A5-03 (directive `go 1.26.0` vs doc 1.25).
