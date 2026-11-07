# Audit — FibCalc à l'aune de *Building Enterprise Projects with Go*

- **Référence** : Saeed Shahsavan, *Building Enterprise Projects with Go — Clarity at Scale in Production-Grade Go Systems*, Apress, 2026 (20 chapitres, lus intégralement).
- **Objet audité** : dépôt `github.com/agbruneau/FibGo`, branche `main`, commit `c6ce7fb` (2026-09-05), arbre propre.
- **Date de l'audit** : 2026-09-07. Hôte : Windows 11, `go1.27.0 windows/amd64`, `CGO_ENABLED=1`, `golangci-lint v2.13.2`.
- **Auteur** : Claude Fable 5.1, à la demande du mainteneur.
- **Régime** : production — chaque constat porte une preuve rejouable et chaque tâche du plan un critère d'acceptation.

---

## 1. Verdict

FibCalc est **au-dessus du livre** sur ce que le livre traite le plus longuement — modèle d'erreurs, discipline de concurrence, profondeur des tests, mesure avant affirmation, ADR avec candidats rejetés — et **en dessous** sur ce que le livre présente comme non négociable pour un projet « enterprise » : une chaîne d'intégration continue qui exécute le gate à la place de la discipline individuelle, des outils épinglés, et des frontières de couches sans fuite de présentation ni de journalisation tierce dans le domaine.

Le dépôt a déjà payé une fois le prix du premier écart : le lint est resté silencieusement inactif pendant une période inconnue (GATE-01, ADR-0010 D3). Le même mécanisme est de nouveau à l'œuvre aujourd'hui sur cet hôte : `govulncheck`, `gosec` (autonome) et `staticcheck` sont tous trois **incapables de s'exécuter** sous `go1.27` (binaires compilés pour `go1.26`), et rien ne le signale.

### Cinq constats prioritaires

| ID | Sévérité | Constat | Chapitre du livre |
|---|---|---|---|
| PRO-01 | Haute | Aucune CI distante ; le gate ne tourne que si quelqu'un y pense. Le déclencheur de révision inscrit dans ADR-0010 D4 (« outillage local non supposable ») est déjà atteint : trois outils sont cassés sur l'hôte de référence. | ch. 1 « Agreed Standards », ch. 11, ch. 17 « CI Integration and Performance » |
| TST-01 | Haute | Un test *flaky* connu et documenté (`TestStateBump_PinnedAcrossCachedCalls`) est laissé en place. Le livre est catégorique : un test instable est pire qu'un test absent. | ch. 7 « Flaky Test Prevention » |
| CON-01 | Moyenne | Les modes `--calibrate` et `--auto-calibrate` s'exécutent **sans** `signal.NotifyContext` ni `-timeout` : Ctrl-C n'est pas géré, le chemin « Calibration interrupted » est inatteignable depuis le binaire. | ch. 13 « Graceful Shutdown », ch. 20 « Context Pattern » |
| OBS-01 | Moyenne | Journalisation inversée : `zerolog` est importé par cinq paquets du domaine, mais **aucun** événement n'est émis en production ; un `TestMain` existe uniquement pour museler le logger global. La couche d'application, elle, ne journalise rien. | ch. 6 « Don't Log and Return », ch. 14 « Rules to Keep Your Architecture Clean » |
| ARC-01 / API-04 | Moyenne | Présentation dans les mauvaises couches : `internal/calibration` imprime 31 chaînes colorées ; `internal/errors` (feuille) écrit des statuts formatés et définit une interface `ColorProvider` pour contourner un cycle d'import. | ch. 14 « Hexagonal Architecture », ch. 6 « Interfaces » |

### Chiffres clés (relevés le 2026-09-07)

| Mesure | Valeur | Source |
|---|---|---|
| Paquets Go | 21 | `go list ./...` |
| Lignes de production / de test | 16 945 / 29 948 | `find … -name '*.go'` |
| Fonctions `Test` / `Benchmark` / `Fuzz` / `Example` | 820 / 51 / 7 / 5 | grep |
| Appels `t.Parallel()` | 1 086 | grep |
| Couverture (`-short`, 21 paquets) | 96,7 % ; plancher gardé : 80 % | `go test -coverprofile` |
| `go build`, `go vet`, `golangci-lint`, `gofmt -l`, `go mod tidy -diff` | tous à zéro | rejoués |
| Dépendances directes / modules du graphe | 10 / 201 | `go list -m all` |
| Lignes de commentaires en production | 5 992 (35 % des lignes) | grep |
| Références à des identifiants d'audit dans les commentaires de production | 164 | grep (annexe B) |
| Documents Markdown / lignes | 42 / 12 129 | `git ls-files '*.md'` |
| Commits conventionnels sur les 60 derniers | 43 | `git log` |
| Cibles cassées sur cet hôte | `GOARCH=386`, `-tags gmp` (attendu), `govulncheck`, `gosec`, `staticcheck` | rejoués |

---

## 2. Méthode et périmètre

### 2.1 Ce qui a été lu

- **Le livre en entier** (texte extrait par `pdftotext`, 22 922 lignes) : les 20 chapitres, y compris ceux qui ne s'appliquent pas à un CLI (HTTP, hexagonal en microservices, MySQL, Testcontainers, Pulsar, gRPC), pour ne rien écarter sans motif.
- **Tout le code de production** : `cmd/`, `internal/` (fichiers non `_test.go`), `test/e2e` en tête de fichier.
- **La documentation** : `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md` (tête), `docs/ARCH.md`, `TESTING.md`, `BUILD.md`, `PORTABILITY.md`, `PERFORMANCE.md`, `CALIBRATION.md`, ADR-0002/0003/0010/0011, le graphe de dépendances, `.env.example`.
- **L'outillage** : `Makefile`, `.golangci.yml`, `scripts/check.sh`, `scripts/check.ps1`, `Dockerfile`, `.devcontainer/devcontainer.json`, `.gitignore`, `.gitattributes`, `go.mod`.

### 2.2 Ce qui a été exécuté

Toutes les sorties sont en annexe B.

- `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `gofmt -l .`, `go mod tidy -diff`, `go mod verify` : zéro écart.
- `go test -short -coverprofile ./...` : 21 paquets verts, 96,7 %.
- `GOOS=linux GOARCH=386 go build ./...` : échec (attendu, documenté).
- `go build -tags gmp ./internal/fibonacci/` : échec faute d'en-têtes libgmp (attendu).
- `govulncheck ./...`, `gosec ./...`, `staticcheck ./...` : **les trois binaires échouent à démarrer** sous `go1.27`.
- Greps ciblés : contextes, signaux, journalisation, `os.Setenv`, `t.Parallel`, identifiants d'audit, `init()`, variables de paquet, panics, `%w`.

### 2.3 Ce qui n'a pas pu être vérifié

- Le backend `gmp` (pas de libgmp sur l'hôte).
- La construction de l'image Docker (pas de démon Docker).
- Le comportement réel sous Ctrl-C en mode calibration (déduit du code, non exécuté).
- Les temps de fuzzing et les figures de performance : non rejoués, hors du périmètre de cet audit (les artefacts de `docs/audits/` sont pris tels quels).

### 2.4 Échelles

- **Sévérité** — *Haute* : contredit un principe central du livre et a (ou a déjà eu) un effet observable. *Moyenne* : écart structurel sans incident connu. *Basse* : hygiène, lisibilité, dette. *Info* : constat ou choix documenté, sans action requise.
- **Effort** — *S* : moins de deux heures. *M* : une demi-journée à une journée. *L* : un à trois jours.

### 2.5 Ce que cet audit ne refait pas

Les audits 2026-06, 2026-07, 2026-08 et 2026-09 (ADR-0008 à 0011) ont couvert la correction, la sur-ingénierie et la synchronisation documentaire. Leurs candidats rejetés sur mesure ne sont pas re-proposés ici sans élément nouveau. Le présent audit apporte un élément nouveau : la grille du livre.

---

## 3. Grille de correspondance

| Chapitre du livre | Dimension auditée | Section |
|---|---|---|
| 1 — Teamwork ; 11 — Branching, Makefile/Dockerfile | Processus, pipeline, contrat de build | 4.1, 4.2 |
| 2 — Choosing a Language ; 11 — MVS | Dépendances, chaîne d'approvisionnement, sécurité | 4.10, 4.13 |
| 4 — Structuring Go Projects | Structure, modules, imports | 4.2 |
| 5 — Core Language Elements ; 8 — Memory | Types, constantes, portée, mémoire | 4.3, 4.6 |
| 6 — Functions, Methods, Interfaces, Errors | API, interfaces, erreurs, panics | 4.4 |
| 7 — Testing | Tests | 4.5 |
| 9 — Concurrency ; 20 — Concurrency patterns & anti-patterns | Concurrence, contexte, arrêt propre | 4.7 |
| 12 — Configuration | Configuration | 4.8 |
| 10, 14 — Blueprint, Hexagonal | Frontières de couches | 4.9 |
| 10, 13, 18, 19 — Observability | Observabilité | 4.11 |
| 1, 6, 20 — Clarity, documentation vivante | Documentation, lisibilité | 4.12 |
| 13, 15, 16, 17, 18, 19 — HTTP, REST, DB, Testcontainers, Pulsar, gRPC | Non applicables, avec motif | 5 |

---

## 4. Constats

Chaque constat : **principe** du livre (chapitre, section, page), **preuve** dans le dépôt, **recommandation**. Les identifiants sont repris dans le plan (§ 7) et l'index (annexe C).

### 4.1 Processus, équipe et pipeline (ch. 1, 11, 17)

#### PRO-01 · Haute · Effort M — Aucune intégration continue

**Principe.** Ch. 1 « How to Build Shared Ownership » (p. 13-14) : *« Agreed Standards: Style guides, continuous integration, automated testing, and code review rules ensure that the system behaves predictably regardless of who last touched it. Strong guardrails enable freedom. »* Ch. 7 (p. 211) : *« Always include -race and -shuffle=on in your CI pipeline. »* Ch. 11 (p. 326) : *« A typical CI pipeline simply executes make tidy / make test / make build. The CI system does not need to understand the project. »* Ch. 17 (p. 443-445) : pratiques CI.

**Preuve.**
- Aucun répertoire `.github/` dans l'arbre. `.gitignore` conserve un vestige : `!.github/workflows/coverage.yml` (ligne 30) pour un fichier qui n'existe plus.
- `README.md` § Développement : « **Pas de CI distante — décision assumée** ». ADR-0004 §B3 et ADR-0010 D4 actent ce choix, avec la clause « À revoir si : le projet accueille des contributeurs dont on ne peut pas supposer l'outillage local ».
- Cette clause est atteinte **sur l'hôte de référence lui-même** : `govulncheck ./...` → *« This application uses version go1.26 of the source-processing packages but runs version go1.27 »* ; `gosec ./...` → *« internal error: package "os" without types »* ; `staticcheck ./...` → *« export data version 4 is greater than maximum supported version 2 »*. C'est exactement le mode de défaillance de GATE-01, reproduit sur trois outils.
- Conséquence déjà matérialisée : ADR-0010 D3 — le lint a écrit `Overall: PASS` sans s'exécuter pendant une durée inconnue.

**Recommandation.** Un workflow GitHub Actions minimal, qui n'est pas un remplacement du gate local mais son exécution garantie : matrice `ubuntu-latest` + `windows-latest`, Go lu depuis `go.mod` (`go-version-file`), étapes `gofmt -l`, `go vet`, `go build`, `go test -race -shuffle=on -count=1 -coverprofile`, plancher 80 % (réutiliser `scripts/check.sh --coverage-only` sous Ubuntu), `golangci-lint` épinglé (PRO-02), `govulncheck`. Sous Ubuntu, un job supplémentaire `apt-get install libgmp-dev` + `go test -tags gmp -race ./internal/fibonacci/` — la seule façon aujourd'hui d'exécuter l'étape 3b sur chaque commit. Un job planifié hebdomadaire pour le fuzzing (TST-04). Consigner le renversement de la décision dans un ADR-0012.

#### PRO-02 · Moyenne · Effort S — Outils non épinglés

**Principe.** Ch. 17 « Key Practices for CI » (p. 444) : *« Use Fixed Image Versions. Avoid image tags like latest. »* Ch. 2 (p. 46) : le déterminisme des dépendances est une exigence de conformité, pas une commodité.

**Preuve.**
- `Makefile` cible `install-tools` : `go install …/golangci-lint@latest` et `…/gosec@latest`. Le commentaire explique que l'épinglage `v1.64.8` a été retiré parce qu'il a cassé ; la réponse a été de ne plus épingler du tout.
- `.devcontainer/devcontainer.json` installe `staticcheck@latest` et `benchstat@latest` ; `staticcheck` n'est appelé par aucun gate.
- Les trois outils cassés (PRO-01) le sont précisément parce qu'ils ont été installés à une date et jamais reconstruits.

**Recommandation.** Épingler la version et reconstruire l'outil avec la chaîne courante à chaque appel, ce qui supprime la classe de défaillance GATE-01. Deux formes existent ; **retenu : `go run <pkg>@<version>`**, avec les versions dans un fichier unique (`scripts/tools.env`) lu par les deux gates, le `Makefile` et la CI.

La directive `tool` de `go.mod` (Go ≥ 1.24) a été essayée d'abord et **rejetée sur mesure** : `go get -tool …/golangci-lint@v2.13.2` tire l'arbre de dépendances du linter dans le module principal, fait passer le graphe de **201 à 450 modules** et, par MVS, **remonte une dépendance de production** (`github.com/shirou/gopsutil/v4` v4.26.3 → v4.26.7). Un épinglage d'outil ne doit pas déplacer une dépendance de production. Retirer `install-tools` et l'installation `@latest` du devcontainer.

#### PRO-03 · Basse · Effort S — Messages de commit hors convention

**Principe.** Ch. 1 (p. 14) : standards convenus, appliqués automatiquement. `CONTRIBUTING.md` § Commit Messages impose *Conventional Commits*.

**Preuve.** Sur les 60 derniers commits, 43 respectent le format. Les 17 autres : « supression évaluation », « évaluaiton », « Update README.md » (×3), « readme » (×2), « audit », « Audit Gemini », « Docs à jour ». Les deux plus récents (`c6ce7fb`, `81e852a`) sont hors format et comportent des fautes.

**Recommandation.** Un hook `commit-msg` local (regex sur le sujet) ou une étape CI `commitlint`-équivalente en 10 lignes de shell. Effort nul pour le mainteneur une fois posé.

#### PRO-04 · Info — Étiquettes et branches

**Principe.** Ch. 11 « Branching Strategies » (p. 327-330).

**Preuve.** Une seule branche de travail (`main`), 22 étiquettes dont huit `rewrite/v3/*-green` et `rewrite/vP/done` — des jalons d'échafaudage, non des versions. `CHANGELOG.md` porte un `[Unreleased]` volumineux depuis `v4.0.0` (2026-07-07).

**Recommandation.** Le modèle `main/staging/prod` du livre est dimensionné pour des équipes ; pour un laboratoire mono-mainteneur il n'apporte rien. Deux gestes suffisent : supprimer les étiquettes d'échafaudage et couper une version (`v4.1.0`) pour vider `[Unreleased]`.

#### PRO-05 · Info — Leçons apprises

**Principe.** Ch. 1 « Lessons Learned: The Architecture of Continuous Improvement » (p. 20-22) : *« A lesson is not learned until it changes behavior. »*

**Preuve.** Les ADR-0008 à 0011 consignent décisions **et** candidats rejetés avec la mesure qui les rejette ; le CHANGELOG décrit chaque audit avec ses corrections et ses limites déclarées. C'est la forme la plus aboutie de post-mortem sans blâme que le livre décrit. Aucune action, sauf DOC-01 : les rapports d'audit sont supprimés de l'arbre après exécution alors que le code y renvoie encore (164 fois).

### 4.2 Structure du projet et contrat de build (ch. 4, 11)

**Conformités.** Disposition `cmd/` + `internal/` + `test/` (ch. 4 p. 68-69) ; `main.go` de 38 lignes qui ne fait que câbler (p. 70) ; module unique (p. 67 : *« Keep it monolithic until it hurts »*) ; aucun `pkg/` fourre-tout (p. 70) ; revue des imports par `go list` (p. 71) matérialisée dans `internal/arch_test.go` et le relevé de validation ; DAG vérifié à 46 arêtes.

#### STR-01 · Moyenne · Effort M — Deux gates divergents, un contrat non portable

**Principe.** Ch. 11 « Build Automation As an Executable Contract » (p. 324) : *« This contract must work in all environments: on a developer's laptop, inside a CI pipeline, and in production build systems. If the process changes depending on where it runs, the system becomes fragile. »*

**Preuve.**
- `Makefile`, en-tête : « POSIX/WSL only ». Sous Windows natif, le contrat n'est pas exécutable.
- `scripts/check.ps1` reproduit `scripts/check.sh` mais **diverge** : pas d'étape 3b (`-tags gmp`), et les deux fichiers portent chacun ~50 lignes de commentaires expliquant l'autre.
- Aucun test ne vérifie que les deux scripts exécutent les mêmes étapes.

**Recommandation.** Avec PRO-01, la CI devient le contrat faisant foi et les deux scripts des commodités locales ; la divergence cesse d'être dangereuse. Si la CI est refusée, l'alternative est un gate écrit en Go (`cmd/gate`, ~120 lignes, un seul fichier) appelé par `make`, par PowerShell, par le `Dockerfile` et par la CI — moins de lignes que les deux scripts réunis.

#### STR-02 · Basse · Effort S — Le `Dockerfile` ne délègue pas au `Makefile`

**Principe.** Ch. 11 « The Dockerfile » (p. 326) : *« the Dockerfile delegates all build responsibility to the Makefile … there is a single source of truth for how the application is compiled. »*

**Preuve.**
- `Dockerfile` : `RUN go build -ldflags="-s -w" -o /out/fibcalc ./cmd/fibcalc` — ligne de build dupliquée, **sans** les `-X` de version. L'image répond `fibcalc dev / Commit: unknown / Built: unknown` à `--version`.
- Deux `TODO(SEC-04)` ouverts : images non épinglées par digest.
- Pas de `.dockerignore` : `COPY . .` embarque `coverage.out` (220 Ko), `docs/`, `.claude/`.

**Recommandation.** `RUN apk add --no-cache make && VERSION=… make build` (ou `ARG VERSION COMMIT BUILD_DATE` passés à la même ligne `LDFLAGS`), `.dockerignore` (`coverage.*`, `docs/`, `.claude/`, `*.md`), et résoudre les deux digests sur un hôte disposant de `docker buildx imagetools inspect`.

#### STR-03 · Basse · Effort M — `internal/errors` masque le nom d'un paquet standard

**Principe.** Ch. 4 (p. 70) : « Explicit Dependencies » ; ch. 6 : clarté des noms.

**Preuve.** Le répertoire s'appelle `errors` mais le paquet se déclare `apperrors` ; chaque importateur doit écrire `apperrors "github.com/agbruneau/FibGo/internal/errors"`. `go doc ./internal/errors` et la lecture des imports sont plus lents qu'il ne faut, et un `import "errors"` oublié à côté est une confusion classique.

**Recommandation.** `git mv internal/errors internal/apperrors` + `sed` sur les imports + mise à jour de la règle `arch_test.go`. Mécanique, sans risque, mais large diff : à décider par le mainteneur (D3 du plan).

#### STR-04 · Basse · Effort S — Règle d'architecture documentée mais non gardée

**Principe.** Ch. 4 (p. 71) : détecter les dépendances rampantes avant qu'elles ne prennent racine.

**Preuve.** `internal/cli/doc.go` : « This package MUST NOT import internal/fibonacci directly … the cli production code follows the same rule **by convention**. » `internal/arch_test.go` ne l'impose que pour `tui`.

**Recommandation.** Ajouter la règle `cli → fibonacci` à `architectureRules` (une ligne), et après ARC-01 la règle `calibration → ui`.

### 4.3 Types, constantes et portée (ch. 5, 8)

**Conformités.** `*bool` pour l'état tri-valué de `Options.FFTCacheEnabled` (ch. 5 p. 87) ; `govet` avec `shadow` activé (p. 99) ; constantes nommées pour les seuils et codes de sortie (p. 97) ; `uint64` pour l'indice `n` (p. 79 : modéliser l'impossibilité d'une valeur négative dans le type).

#### TYP-01 · Moyenne · Effort S — Constante qui déborde `int` sur 32 bits, dupliquée

**Principe.** Ch. 5 « Integers » (p. 76-79) : *« The plain int adapts to your machine's word size … you must know the environment. »* et « Types are guardrails ».

**Preuve.**
- `internal/fibonacci/memory/arena.go:14` : `const maxReasonableWords = 1 << 60`.
- `internal/fibonacci/fastdoubling.go:296` : **même constante, redéclarée** (`const maxReasonableWords = 1 << 60`).
- `GOOS=linux GOARCH=386 go build ./...` échoue sur `arena.go:26,29,30` (rejoué le 2026-09-07). `PORTABILITY.md` § 1 en fait une limite déclarée (« 64 bits uniquement »).

**Recommandation.** Une seule définition, exportée du paquet `memory` (ou de TYP-04), en `1 << (bits.UintSize - 4)` : identique sur 64 bits, valide sur 32 bits, et la duplication disparaît. Ajouter `GOARCH=386 go build ./...` (build seulement) à la CI ; si une dépendance TUI ne compile pas en 32 bits, remplacer par une contrainte de build explicite avec message plutôt que par un débordement de constante.

#### TYP-02 · Moyenne · Effort M — État mutable au niveau paquet

**Principe.** Ch. 5 « Scope » (p. 98-99) : *« Defining variables in the smallest possible scope is … a technical optimization. »* Ch. 20 « When to Avoid Singletons » (p. 527) : *« harder to replace in tests, hidden coupling between packages, remove flexibility. »* Ch. 9 (p. 279-283) : tout état partagé a un propriétaire clair et une protection.

**Preuve.**
- `internal/fibonacci/threshold/manager.go:40-61` : `FFTSpeedupThreshold`, `ParallelSpeedupThreshold`, `HysteresisMargin`, `minFFTThresholdFloor`, `minParallelThresholdFloor` sont des `var` de paquet **non synchronisées**, écrites par `SetTuning`. Le commentaire A2-04 documente un protocole « single-writer-before-use » et admet : *« Calling SetTuning concurrently with an active calculation would be a data race. »* `internal/app/app.go:wireThresholdTuning` ajoute un `sync.Once` pour respecter ce protocole.
- `internal/calibration/microbench.go:38,42` : `MicroBenchTimeout` et `MicroBenchTestSizes` (tranche mutable) en `var`, alors que `MicroBenchmark` porte déjà les mêmes champs.
- `internal/fibonacci/options.go:configureFFTCache` : chaque `Calculate` réinstalle la configuration **globale** du cache `bigfft` ; sous `--algo all`, trois goroutines le font en concurrence (protégé par mutex, valeurs identiques, mais l'état est processus-global et mutable depuis le domaine).
- `internal/fibonacci/matrix_ops.go:17` : `defaultStrassenThresholdBits` atomique + `init()`, « test-only safety net » de son propre aveu.
- `internal/cli/ui.go:newSpinner`, `internal/tui/commands.go:runProgram` : coutures de test en variables de paquet.

**Recommandation.** Passer `threshold.Tuning` **par valeur** dans `NewDynamicThresholdManagerFromConfig` (le zéro → constantes du paquet) et supprimer `SetTuning`, les cinq `var`, `wireThresholdTuning` et son `sync.Once` : le protocole A2-04 n'a plus rien à protéger. Faire des deux `var` de `microbench.go` des constantes et laisser les champs de `MicroBenchmark` seuls configurables. Les atomiques de `bigfft` restent (ADR-0003) ; installer la configuration du cache **une fois** par `ExecuteCalculations` plutôt qu'une fois par calculateur.

#### TYP-03 · Info — Énumérations en chaînes

**Principe.** Ch. 5 « Constants and iota » (p. 93-97).

**Preuve.** `memory.GCMode` est un type chaîne (`"auto"`, `"aggressive"`, `"disabled"`) validé par `switch` ; `AppConfig.GCControl` est un `string` brut jusqu'à la conversion.

**Recommandation.** Aucune : les valeurs sont celles du drapeau CLI et du JSON ; un `iota` ajouterait une table de conversion sans supprimer la validation.

#### TYP-04 · Basse · Effort S — Savoir dupliqué : le facteur de croissance

**Principe.** Ch. 6 « DRY in Enterprise Go Project » (p. 144) : *« DRY doesn't mean abstracting everything; it means capturing true duplication of knowledge. »*

**Preuve.** `0.69424` (log₂ φ) apparaît en littéral dans `internal/fibonacci/constants.go` (`FibonacciGrowthFactor`), `internal/fibonacci/memory/budget.go:96` et `internal/bigfft/memory_est.go:19` ; `baselineMinN = 93` dans `budget.go` duplique `MaxFibUint64` avec un commentaire qui explique pourquoi l'import est impossible ; trois `formatBytes` (défendus par ADR-0011 R3 comme imposés par le gate d'architecture).

**Recommandation.** Un paquet feuille sans aucun import, `internal/fibmath` (constantes + `BitsFor(n)`, `WordsFor(n)`), importable par `fibonacci`, `memory` et `config` sans toucher au gate. `bigfft` peut le rejoindre ou garder son littéral avec renvoi — le choix est documentaire (ARCH.md affirme que `bigfft` n'importe aucun paquet interne).

#### TYP-05 · Info — `AppConfig` par valeur

**Principe.** Ch. 8 « Choosing Between Values and Pointers » (p. 252-253).

**Preuve.** `config.AppConfig` (24 champs) est passée par valeur à une dizaine d'endroits et `Validate` a un récepteur valeur ; `gocritic.hugeParam` est désactivé pour cela.

**Recommandation.** Aucune : quelques centaines d'octets copiés une poignée de fois par exécution ; la sémantique valeur protège contre la mutation partagée entre `app`, `tui` et `calibration`.

### 4.4 Fonctions, interfaces et erreurs (ch. 6)

**Conformités.** Constructeurs `NewX` qui valident (p. 138) ; `errors.Is`/`errors.As` aux bords (p. 183-184), 49 sites `%w` (p. 182) ; sentinelles (`context.*`, `ErrMissingFastCalculator`) **et** erreurs typées (`ConfigError`, `MemoryError`, `CalculationError` avec `Unwrap`) — exactement la partition du livre (p. 183-187) ; `NewConfigError` retourne `error` et non le type concret, ce qui ferme le piège du *typed nil* (p. 142) ; `WriteResultToFile` capture l'erreur de `Close` par retour nommé (p. 190-191, mot pour mot le modèle du livre) ; clauses de garde partout (p. 181) ; panics réservés aux invariants (`bigfft`, ADR-0002) et convertis à la frontière (p. 192-193) ; `MustNewCalculator` suit la convention `template.Must` (p. 192).

#### API-01 · Moyenne · Effort M — Interface de fabrique définie côté fournisseur, à cinq méthodes

**Principe.** Ch. 6 « Consumer-Defined Interfaces » (p. 154) : *« interfaces belong to consumers, not providers … the team using a behavior decides how much abstraction is necessary. »* Ch. 6 (p. 160) et ch. 20 (p. 514-517) : une à trois méthodes ; *« Introduce the interface only when a second implementation or test requires it. »*

**Preuve.**
- `internal/fibonacci/registry.go:CalculatorFactory` : `Create`, `Get`, `List`, `Register`, `GetAll` — définie dans le paquet fournisseur.
- Consommateurs réels : `orchestration.GetCalculatorsToRun` n'utilise que `List` et `Get` ; `app` n'utilise que `List` et `GetAll` (puis passe une `map` à `calibration`).
- Le prix de la largeur : `internal/fibonacci/testing.go:TestFactory` doit implémenter les cinq méthodes, dont un `Register` qui est un **no-op retournant nil** (une doublure qui ment) et un `List` **non trié** alors que le contrat de l'interface dit « sorted list ».

**Recommandation.** `orchestration` définit `type CalculatorSource interface { List() []string; Get(string) (Calculator, error) }` ; `app` garde le type concret `*fibonacci.DefaultFactory` (l'option `WithFactory` l'accepte) ; `CalculatorFactory` et `TestFactory` disparaissent, les tests de `orchestration` portent une doublure de deux méthodes.

#### API-02 · Basse · Effort S — Contrats de registre non tenus

**Principe.** Ch. 6 (p. 138-140) : le constructeur *« enforces invariants … shields callers from invalid or incomplete state »* ; ch. 6 (p. 180-183) : les erreurs sont des valeurs qu'on regarde.

**Preuve.**
- `registry.go:Register` retourne toujours `nil` et remplace silencieusement un nom déjà enregistré (« If a calculator with the same name already exists, it will be replaced ») ; `NewDefaultFactory` jette le résultat (`_ = f.Register(…)`) ; `CONTRIBUTING.md` exige pourtant de le vérifier parce qu'`errcheck` le demande.
- `registry.go:176` (`GetAll`) : `if calc, err := NewCalculator(creator()); err == nil` — l'erreur est avalée.
- Un créateur `nil` n'est rejeté nulle part.

**Recommandation.** `Register` valide (nom vide, créateur `nil`, doublon → erreur) ; `NewDefaultFactory` panique sur sa propre erreur d'enregistrement (bogue de programmeur, convention `Must`) ; `GetAll` peut alors documenter que `NewCalculator` ne peut plus échouer. Tests : doublon et créateur nil.

#### API-03 · Moyenne · Effort M — Journalisation dans le domaine, silence à la périphérie

**Principe.** Ch. 6 « Don't Log and Return » (p. 189) : *« library code returns errors; application edges log them. »* Ch. 14 « Rules to Keep Your Architecture Clean » (p. 370) : *« If a logger requires a third-party dependency, keep it confined to the adapter layer. »*

**Preuve.**
- `zerolog` est importé par `internal/fibonacci/calculator.go:13`, `internal/bigfft/fft_cache.go`, `internal/fibonacci/memory/gc_control.go`, `internal/fibonacci/threshold/manager.go`, `internal/progress/observers.go` — cinq paquets du domaine.
- Six émetteurs existent (`gc disabled`/`gc re-enabled` avec statistiques de tas, `fft cache stats`, `thresholds adjusted`, `calculation progress`, `calculation completed`). **Aucun n'est atteignable en production** : cinq loggers sont figés à `zerolog.Nop()` avec des *setters* réservés aux tests (`setLogger`, `setCacheLogger`) ; le sixième, `log.Trace()` sur le logger global (`calculator.go:227`), est filtré par `zerolog.SetGlobalLevel(zerolog.InfoLevel)` posé dans `app.Run` (`app.go:111`).
- `internal/fibonacci/testmain_test.go` n'existe que pour reposer ce même niveau global afin que `-bench` reste lisible par `benchstat` — un `TestMain` (que le livre déconseille, p. 212) pour neutraliser une journalisation qui ne sert jamais.
- `internal/tui/bridge.go:27` : logger `log.New(io.Discard, …)` de la bibliothèque standard, même motif.
- La couche d'application (`app`, `cli`) ne journalise rien ; les diagnostics utiles (GC coupé, seuils ajustés, cache) restent invisibles à l'utilisateur d'un dépôt dont la devise est de tout mesurer.

**Recommandation.** Deux options, la seconde recommandée (voir OBS-01) :
1. *Minimale* — supprimer `zerolog`, les six émetteurs, les setters de test et `testmain_test.go` (moins une dépendance directe et sa branche transitive).
2. *Alignée sur le livre* — injecter un `*slog.Logger` (bibliothèque standard, pas de dépendance) depuis `app` via `fibonacci.Options.Logger` (nil → `slog.DiscardHandler`), exposer `--log-level` / `FIBCALC_LOG_LEVEL` (stderr, texte ; JSON sous `-machine`), convertir les six émetteurs. Le domaine ne dépend plus d'un tiers, les événements deviennent observables, `TestMain` disparaît.

#### API-04 · Moyenne · Effort M — Présentation dans le paquet d'erreurs

**Principe.** Ch. 6 (p. 180) : *« error messages … are part of your program's API, not prose for humans »* ; ch. 14 (p. 366-368) : le cœur ne parle ni HTTP ni console.

**Preuve.**
- `internal/errors/handler.go:HandleCalculationError` écrit « Status: Failure (Timeout)… », « Status: Canceled… » sur un `io.Writer` avec des codes couleur.
- `internal/errors/handler.go:ColorProvider` existe, de son propre commentaire, pour « break the import cycle with cli » ; `internal/cli/provider.go` (`CLIColorProvider`) et `apperrors.DefaultColorProvider` ne servent qu'à cela.
- `internal/tui/bridge.go:HandleError` appelle `HandleCalculationError(err, duration, io.Discard, nil)` : la TUI ne veut que le **code de sortie** et jette le texte.

**Recommandation.** Scinder : `apperrors.ExitCodeFor(err) int` (pur, testable en une table) et `cli.WriteCalculationStatus(w, err, d)` qui utilise `ui.Color*` directement. Supprimer `ColorProvider`, `DefaultColorProvider`, `CLIColorProvider`, `provider.go`, et le paramètre `colorProvider` de `calibration.RunCalibration`.

#### API-05 · Basse · Effort S — Style des messages d'erreur

**Principe.** Ch. 6 « Styling Error Messages » (p. 180-181) : minuscule initiale, pas de ponctuation finale, symptôme plutôt qu'intention.

**Preuve.** L'immense majorité respecte la règle. Écarts : `fmt.Errorf("panic in bigfft.%s: %v\nStack: %s", …, debug.Stack())` (`internal/bigfft/fft.go:fermatPanicToError`) embarque une pile multi-lignes dans une **valeur** d'erreur ; « Strassen threshold cannot be negative » et « FFT threshold must be … » débutent par un nom propre ou un sigle (toléré par la convention Go).

**Recommandation.** Garder la pile hors de la valeur (la journaliser via API-03) et ne conserver que `panic in bigfft.%s: %v`. Le reste est conforme.

#### API-06 · Basse · Effort S — Panics d'observateurs avalés sur le chemin chaud

**Principe.** Ch. 6 (p. 192-193) : *« App edges (servers, workers) should recover from panics to protect uptime, log the failure, and continue serving. »* — récupérer **et** signaler.

**Preuve.** `internal/progress/observer.go:Freeze` : `recover()` par observateur, compteur `recoveredObservers` incrémenté ; `RecoveredObserverCount()` n'est lu que par `observer_test.go`. Un observateur TUI ou CLI qui panique est donc masqué sans aucune trace.

**Recommandation.** Soit journaliser au moment de la récupération (via API-03), soit afficher le compteur en fin d'exécution sous `-d`. Deux lignes.

#### API-07 · Basse · Effort S — Validation qui s'arrête à la première erreur

**Principe.** Ch. 12 « Validate Configuration Before Returning » (p. 340-342) : *« the Validate method calls internal validate methods and at the end joins all errors to return all incorrect configurations. »*

**Preuve.** `internal/config/config.go:AppConfig.Validate` : neuf vérifications, retour au premier échec. L'utilisateur qui se trompe sur deux drapeaux fait deux allers-retours.

**Recommandation.** Accumuler dans `[]error` et retourner `errors.Join`. Adapter les tests qui comparent la chaîne exacte de la première erreur.

#### API-08 · Basse · Effort S — Commentaire de paquet périmé

**Principe.** Ch. 7 (p. 223-224) : les exemples sont de la documentation exécutée ; ch. 15 (p. 387) : clarté pour les nouveaux venus.

**Preuve.** `internal/app/doc.go:29-31` : l'exemple appelle `action.ShouldExit()` et `action.Code()` sur le type `ExitAction`, supprimé par ADR-0011 D1 ; `Run` retourne aujourd'hui un `int`.

**Recommandation.** Mettre l'exemple au niveau de `cmd/fibcalc/main.go`.

#### API-09 · Info — Longueur des fonctions

**Principe.** Ch. 6 (p. 135) : *« If a function grows beyond ~20 lines, consider extracting helpers. »*

**Preuve.** Le dépôt fixe `funlen` à 100 lignes / 50 instructions et `gocyclo` à 15 avec trois exemptions nommées (`fourierRecursiveUnified`, `Model.Update`, `AppConfig.Validate`) plus un `nolint:gocognit`. Cinq fichiers de production dépassent 550 lignes (`fft_cache.go` 703, `fft_poly.go` 650, `microbench.go` 570, `fastdoubling.go` 556, `calibration.go` 550).

**Recommandation.** Aucune : les seuils sont explicites, mesurés et justifiés fonction par fonction dans `.golangci.yml` ; le livre donne un ordre de grandeur, pas une limite.

### 4.5 Tests (ch. 7)

**Conformités.** Tests pilotés par table avec sous-tests nommés (p. 213-216) ; `t.Parallel()` massif (1 086) ; `t.Helper()` (23), `t.Cleanup()` (27), `t.TempDir()` (38) (p. 200, 206-208) ; oracles *golden* avec générateur indépendant (`cmd/generate-golden`, p. 203) ; propriétés mathématiques (`gopter`) ; 51 benchmarks avec `ReportAllocs` (p. 220) et un protocole `benchstat` propre au dépôt (ADR-0009, hors livre) ; 5 exemples exécutables (p. 224) ; tests de bout en bout par sous-processus ; gate d'architecture ; couverture 96,7 % avec plancher appliqué (80 %) ; `-race` local sur 21 paquets ; attentes de canaux bornées par `time.After` (10 sites, p. 228) ; tests de contrat de panic (`TestFermatPanicSites`).

#### TST-01 · Haute · Effort M — Test *flaky* connu et conservé

**Principe.** Ch. 7 « Speed and Determinism » (p. 199-200) : *« A flaky test is worse than no test at all. It slows teams down, creates doubt, and makes developers stop trusting the suite. »* et *« Avoid using shared or global state between tests. »*

**Preuve.**
- `README.md` § Limites déclarées : « `TestStateBump_PinnedAcrossCachedCalls` est *flaky* au même taux sur le commit parent (un test antérieur laisse une arène surdimensionnée dans le pool d'état global). »
- `internal/fibonacci/state_cache_test.go:126`. La cause est nommée : `statePool` (`sync.Pool` de paquet) partagé entre tests parallèles.

**Recommandation.** Rendre le test **séquentiel** (retirer `t.Parallel`) et repartir d'un état connu : un *helper* de test `resetStatePoolForTest()` qui réassigne `statePool` et vide le slot `cachedState` du calculateur. En Go, un test sans `t.Parallel` ne chevauche jamais un test parallèle du même paquet (ceux-ci sont suspendus jusqu'à la fin de la passe séquentielle), donc la réinitialisation est sûre. Critère : 50 exécutions consécutives vertes.

#### TST-02 · Moyenne · Effort S — `os.Setenv` dans des tests parallèles

**Principe.** Ch. 7 (p. 200, 217) : pas d'état global partagé ; *« Write each parallel test as if it runs alone. »*

**Preuve.**
- 22 appels `os.Setenv` dans les tests contre 18 `t.Setenv`.
- `internal/config/config_test.go` : `TestTUIFlag` et `TestParseConfig` contiennent **à la fois** `os.Setenv` et `t.Parallel()` (relevé par analyse de fonction). `t.Setenv` refuse un test parallèle précisément pour empêcher cela ; l'usage de `os.Setenv` contourne la protection, et les tests de `config`, `ui` et `cli` lisent l'environnement.
- Aucun `-shuffle` ne tourne (TST-03), ce qui laisse la dépendance à l'ordre invisible.

**Recommandation.** Remplacer chaque `os.Setenv` par `t.Setenv` et retirer `t.Parallel()` des tests concernés. Vérifier par `go test -race -shuffle=on -count=3` sur les trois paquets.

#### TST-03 · Moyenne · Effort S — Pas de `-shuffle=on`

**Principe.** Ch. 7 (p. 210-211, 215, 226) : *« Always include -race and -shuffle=on in your CI pipeline »* ; *« If your test fails when the order changes, that is a design warning. »*

**Preuve.** `grep -rn shuffle scripts Makefile` : aucune occurrence.

**Recommandation.** `-shuffle=on -count=1` dans `check.sh`, `check.ps1`, `make test` et la CI. Coût nul, révèle TST-02.

#### TST-04 · Moyenne · Effort S — Le fuzzing ne fuzze pas, et ses graines n'atteignent pas la FFT

**Principe.** Ch. 7 « Fuzzing » (p. 218-223) : le fuzzing est un pilier de la suite ; les graines représentent *« typical or boundary inputs »*.

**Preuve.**
- `docs/TESTING.md` : aucun gate ne passe `-fuzz` ; seules 63 graines sont rejouées.
- `internal/bigfft/fft_fuzz_test.go` : les graines de `FuzzMul`/`FuzzSqr` plafonnent à 512 mots alors que `Mul` ne prend le chemin FFT qu'au-delà de 1 800 mots (`defaultFFTThresholdWords`). À chaque `go test`, ces deux cibles comparent donc `math/big` **à lui-même**.

**Recommandation.** Ajouter des graines à 2 048 et 4 096 mots (le rejeu couvre alors `fftmul`/`fftsqr` de façon déterministe — vérifiable par `go tool cover -func | grep fftmul` après `go test -run Fuzz -coverprofile`) ; cible `make fuzz-smoke` (30 s par cible) ; job CI hebdomadaire.

#### TST-05 · Basse · Effort S — `TestMain` sans nécessité

**Principe.** Ch. 7 « When to Use and When to Avoid » (p. 212) : *« use TestMain() only when you truly need shared setup … It breaks test isolation. »*

**Preuve.** `internal/fibonacci/testmain_test.go` fixe le niveau global de `zerolog` ; il disparaît avec API-03.

#### TST-06 · Basse · Effort S — Binaire e2e dans un chemin fixe

**Principe.** Ch. 7 (p. 226) : *« Do not write to /tmp or fixed locations. Use t.TempDir(). »*

**Preuve.** `test/e2e/cli_e2e_test.go:buildBinary` : `os.TempDir()` + nom fixe `fibcalc_e2e_test[.exe]`, documenté comme choix. Deux exécutions simultanées de `go test` (deux terminaux, deux arbres de travail) écrasent le même fichier.

**Recommandation.** `os.MkdirTemp` dans un `TestMain` du paquet e2e (avec nettoyage) — la seule utilisation légitime de `TestMain` ici — ou `t.TempDir()` du premier appelant.

#### TST-07 · Info — Séparation des tests lents

**Principe.** Ch. 7 « Structuring Large Code Bases » (p. 228) : étiquette de build `//go:build integration`.

**Preuve.** Les e2e sont sautés sous `-short` (`skipShortE2E`) et tournent par défaut. Équivalent fonctionnel ; le livre préfère l'étiquette, mais ici il est souhaitable que `go test ./...` compile le binaire à chaque fois.

#### TST-08 · Basse · Effort S — Noms de fichiers de test vagues

**Principe.** Ch. 7 (p. 229) : *« Consistent Naming: Use clear and predictable file names. »*

**Preuve.** Dix fichiers : `fft_extra_test.go`, `globals_extra_test.go`, `misc_extra_test.go`, `pool_extra_test.go`, `recursion_extra_test.go`, `calibration_advanced_test.go`, `ui_advanced_test.go`, `config_exhaustive_test.go`, `config_extra_test.go`, `env_more_test.go`. « extra », « more », « misc » décrivent l'historique de leur création, pas leur sujet.

**Recommandation.** Renommer par thème après lecture (par exemple `env_more_test.go` → `env_precedence_test.go`).

#### TST-09 · Info — `testing/synctest`

**Principe.** Ch. 7 (p. 197-200) et ch. 9 « Deterministic Concurrency Tests with testing/synctest » (p. 289-292).

**Preuve.** Aucun usage ; un seul `time.Sleep` de test (`internal/cli/ui_advanced_test.go`) ; les 10 `time.After` sont des bornes de sécurité, pas des synchronisations. `go.mod` déclare `go 1.26.0`, le paquet est disponible.

**Recommandation.** Candidat unique : le test de `ui_advanced_test.go` qui dort. Sans urgence.

#### TST-10 · Info — Angle mort de couverture e2e

**Preuve.** Documenté (`TESTING.md` A5-08) : `go build -cover` + `GOCOVERDIR` non câblés. Faible valeur : les chemins CLI sont déjà couverts en unitaire (`internal/app` 95,1 %, `internal/cli` 94,7 %).

### 4.6 Mémoire et performance (ch. 8)

**Conformités.** Le dépôt applique le chapitre 8 au-delà de ce qu'il demande : `sync.Pool` par classe de taille, arène, allocateur bump, contrôle du GC avec `SetMemoryLimit` comme filet, cache borné en octets, décisions prises sur `benchstat` en double ordre (ADR-0009 R4), profil PGO régénéré, commandes `pprof` documentées (p. 255-257). Les commentaires distinguent systématiquement ce qui est mesuré de ce qui est supposé.

#### MEM-01 · Basse · Effort S — Garde mémoire du calculateur jamais armée

**Principe.** Ch. 6 (p. 138) : les invariants qu'on documente doivent être ceux qu'on applique.

**Preuve.** `fibonacci.Options.MemoryLimitBytes` et `CanCalculate` sont présentés comme « defense in depth » de `config.ValidateMemoryBudget`, mais **aucun code de production ne renseigne le champ** (grep hors `options.go`/`calculator.go` : vide). La garde n'existe que pour les appelants programmatiques et les tests. ADR-0011 R2 l'a laissée au mainteneur.

**Recommandation.** `validateMemoryBudget` expose `report.LimitBytes` ; `executeCalculations` et `startCalculationCmd` (TUI) le copient dans `Options`. Le test-espion de `orchestration` vérifie la propagation. Sinon, supprimer le champ et `CanCalculate`.

#### MEM-02 · Info — Coût des goroutines par pas de doublement

**Principe.** Ch. 9 « Goroutine Overhead » (p. 284-285).

**Preuve.** `executeParallel3` lance trois goroutines par pas puis acquiert un jeton ; borné par `bits.Len64(n) ≤ 64` pas. Négligeable et mesuré (`parallel3Result` regroupé en une allocation).

#### MEM-03 · Info — `-gcflags=-m`

**Principe.** Ch. 8 « Seeing It Yourself » (p. 241).

**Recommandation.** Une ligne dans `PERFORMANCE.md` § Profiling : `go build -gcflags='-m' ./internal/fibonacci/ 2>&1 | grep 'moved to heap'`. Le dépôt en parle dans des commentaires, pas dans le guide.

### 4.7 Concurrence (ch. 9, 20)

**Conformités.** `context.Context` en premier paramètre partout (26 fichiers de production, p. 542) ; `context.Background()` uniquement dans `main` (p. 543) ; `errgroup.WithContext` pour le fan-out des calculateurs avec annulation croisée (p. 287-288) ; sémaphores bornés à `GOMAXPROCS(0)` (p. 278, 555) ; propriétaire unique des canaux — `ExecuteCalculations` crée et ferme `progressChan` après `Wait` (p. 269-271) ; `select` + `ticker` pour l'affichage (p. 277) ; atomiques pour les seuils chauds (ADR-0003, p. 283) ; `sync.Once` pour l'initialisation paresseuse (p. 286) ; code 130 sur SIGINT, gardes de génération dans la TUI ; acquisition non bloquante dans la récursion FFT (pas d'interblocage imbriqué) ; `-race` propre sur 21 paquets.

#### CON-01 · Moyenne · Effort S — Calibration sans signaux ni délai

**Principe.** Ch. 9 (p. 259) : *« every goroutine has a lifecycle, every external call has a timeout »* ; ch. 13 « Listening for Shutdown Signals » (p. 355-356) : `signal.NotifyContext` comme racine unique ; ch. 20 « Context Pattern » (p. 539-544) : *« The edge of your system creates the root context. »*

**Preuve.**
- `signal.NotifyContext` apparaît à trois endroits seulement : `internal/app/app.go:169` (`runTUI`), `internal/app/calculate.go:38` (`runCalculate`), `calculate.go:139` (`runLastDigits`).
- `Application.Run` transmet le `context.Background()` de `main` **tel quel** à `runCalibration` et à `runAutoCalibrationIfEnabled`. Ni signal, ni délai.
- Conséquences : sous `--calibrate`, Ctrl-C est traité par le gestionnaire par défaut du runtime (arrêt immédiat, sans « Calibration interrupted », sans passage par `HandleCalculationError`) ; le chemin `ctx.Err() != nil` de `runPassSequence` et le test qui le couvre décrivent un comportement inatteignable depuis le binaire ; `-timeout` est ignoré par `--calibrate` (seul `CompleteStrategy` en dérive un délai par essai). Pendant la phase `--auto-calibrate` d'un calcul normal, même trou, avant que `runCalculate` n'installe ses gardes.
- Trois copies du même bloc « timeout + signaux » dans trois fonctions.

**Recommandation.** Installer `signal.NotifyContext` **une fois** dans `Run` (avant l'aiguillage) et un `context.WithTimeout` par mode qui calcule, calibration comprise ; retirer les trois blocs dupliqués (la TUI garde son délai par génération). Un test : `--calibrate` sur un contexte déjà annulé retourne 130 et imprime le message d'interruption.

#### CON-02 · Moyenne — Variables de paquet non synchronisées

Voir TYP-02 (`threshold`). Le livre (p. 279-283) refuse le compromis « pas de course tant que le protocole d'appel est respecté » : l'état partagé se protège ou s'élimine. La recommandation de TYP-02 l'élimine.

#### CON-03 · Info — Lancement puis étranglement

**Preuve.** `executeTasks`/`executeMixedTasks` lancent chaque tâche par `g.Go` puis acquièrent le jeton à l'intérieur ; le nombre de goroutines en attente est borné par le nombre de tâches (≤ 8 par pas). Conforme à l'esprit du *worker pool* (p. 555) sans en payer la structure.

#### CON-04 · Info — Goroutine du *spinner*

**Preuve.** `internal/cli/ui.go:realSpinner.UpdateSuffix` arrête et relance la goroutine de la bibliothèque `briandowns/spinner` à chaque rafraîchissement (toutes les 200 ms), pour contourner une course dans la bibliothèque (CONC-01). Fonctionnel, mais c'est une dépendance qui impose sa propre solution de contournement. Voir DEP-02.

### 4.8 Configuration (ch. 12)

**Conformités.** Priorité drapeaux > environnement > défauts, documentée et testée (p. 339-340) ; `.env.example` exhaustif ; validation avant retour (p. 340) ; marqueurs d'explicitation des seuils pour que le profil ne l'emporte pas sur l'utilisateur (ADR-0010 D1) ; profil JSON écrit atomiquement (`CreateTemp` + `Rename`) ; re-validation des bornes d'un profil forgé (SEC-01) ; aucun secret à masquer (p. 343).

#### CFG-01 · Basse — `errors.Join`

Voir API-07.

#### CFG-02 · Basse · Effort S — Variables d'environnement lues hors de la table

**Principe.** Ch. 12 (p. 338-340) : un chargeur, une source de vérité ; ch. 14 (p. 369) : la configuration se résout au point de composition.

**Preuve.** `FIBCALC_PROFILE_MAX_AGE` est lue dans `internal/calibration/calibration.go:profileMaxAgeFromEnv` et `FIBCALC_TUI_THEME` dans `internal/ui/themes.go:GetCurrentTUITheme` ; `NO_COLOR` est lue à deux endroits (`config/usage.go`, `ui/themes.go`). Le test de synchronisation `FlagNames` ne peut pas les voir.

**Recommandation.** Deux champs sur `AppConfig` (`ProfileMaxAge time.Duration`, `TUITheme string`) renseignés par `envOverrides`, transmis explicitement à `calibration` et `ui`. Après quoi `grep os.Getenv` hors `config/env.go` ne doit rien retourner en production.

#### CFG-03 · Info — Décodage permissif du profil

**Principe.** Ch. 15 « Defensive Decoding » (p. 396-397) : `DisallowUnknownFields`.

**Preuve.** `profile.go:loadProfile` utilise `json.Unmarshal`. Le fichier est possédé par l'utilisateur et re-validé ; un champ inconnu signalerait surtout un changement de version. Faible valeur.

### 4.9 Frontières et architecture (ch. 10, 14)

**Conformités.** Le dépôt est une transposition fidèle de l'hexagone du livre à un CLI : les *ports* (`ProgressReporter`, `ResultPresenter`, `ErrorHandler`) sont définis côté consommateur dans `orchestration` (p. 367), les *adapters* sont `cli` et `tui`, le domaine (`fibonacci`, `bigfft`) n'importe aucune présentation, le point de composition est `app` (p. 369-370), et un test fait échouer la suite si une flèche remonte (p. 369-370). Le piège de l'« Edge API module » (ch. 10 p. 307) est évité : la logique n'est pas derrière une couche de transport.

#### ARC-01 · Moyenne · Effort M — La calibration fait de la présentation

**Principe.** Ch. 14 « Service » (p. 367) : *« This layer does not talk to HTTP, SQL, Kafka, or file systems directly. »* ; « Adapters » (p. 367-368) : la présentation est un adapter remplaçable.

**Preuve.** `internal/calibration` importe `internal/ui` et écrit directement des chaînes colorées : 19 occurrences de `ui.Color*` dans `calibration.go`, 9 dans `io.go`, 3 dans `strategy_fast.go`. `RunCalibration` reçoit un `io.Writer`, une fonction d'affichage de progression **et** un `ColorProvider`. Les tests du paquet doivent dépouiller l'ANSI pour vérifier des résultats. `internal/config` importe aussi `ui` pour l'aide colorée (toléré par le gate, ARCH-02).

**Recommandation.** Un port défini par `calibration` et implémenté par `cli` : `type Reporter interface { PassResult(threshold int, d time.Duration, err error); Summary(results []Result, best int); Notice(format string, args ...any); Warning(format string, args ...any) }` — quatre méthodes sans couleur ni écriture — avec un `NopReporter` pour les tests. `calibration` cesse d'importer `ui` ; règle ajoutée au gate (STR-04).

#### ARC-02 · Moyenne — Présentation dans `errors`

Voir API-04.

#### ARC-03 · Basse · Effort S — Texte utilisateur émis par le cas d'usage

**Preuve.** `internal/orchestration/orchestrator.go:AnalyzeComparisonResults` écrit lui-même « Global Status: Failure… », « Global Status: Success… » et `MismatchMessage` sur `out`/`errOut`, à côté de l'interface `ResultPresenter` qu'il définit pour déléguer exactement cela.

**Recommandation.** Deux méthodes de plus sur `ResultPresenter` (`PresentGlobalStatus`, `PresentMismatch`) ; `orchestration` ne touche plus aux `io.Writer`.

### 4.10 Dépendances et chaîne d'approvisionnement (ch. 2, 11)

**Conformités.** `go.mod`/`go.sum` versionnés, `go mod verify` et `go mod tidy -diff` propres, sélection minimale de version (p. 322-323), aucune directive `replace`, aucun `go.work` (ch. 11, p. 317-320).

#### DEP-01 · Moyenne · Effort S — Aucune analyse de vulnérabilités dans le gate

**Principe.** Ch. 2 « Security and Compliance » (p. 35), « Dependency Management » (p. 46-47) : provenance et audit des artefacts externes ; ch. 17 (p. 444).

**Preuve.** Ni `check.sh`, ni `check.ps1`, ni le `Makefile` n'appellent `govulncheck` ; le binaire présent sur l'hôte ne démarre plus (PRO-01). 201 modules dans le graphe pour 10 dépendances directes.

**Recommandation.** `go tool govulncheck ./...` (épinglé, PRO-02) dans les deux gates et la CI.

#### DEP-02 · Basse · Effort M — Dépendances dont la valeur est discutable

**Principe.** Ch. 2 « Rich Ecosystem to Avoid Reinventing the Wheel » (p. 34) : *« Every custom library adds long-term maintenance, testing, and security overhead »* — et symétriquement chaque dépendance.

**Preuve.**
- `github.com/rs/zerolog` : inerte en production (API-03).
- `github.com/briandowns/spinner` (+ `fatih/color`, `mattn/*`) : rend une ligne de progression ; a imposé la course CONC-01 et sa solution de contournement (arrêt/relance de goroutine 5 fois par seconde), une interface `Spinner`, un adaptateur `realSpinner`, une variable de couture `newSpinner` et un test dédié (`ui_suffix_race_test.go`).
- `github.com/shirou/gopsutil/v4` (+ `go-ole`, `purego`, `wmi`, `tklauser/*`) : deux appels dans `tui/commands.go` (CPU %, mémoire %). Portable, difficile à remplacer honnêtement ; à garder.
- La pile `charmbracelet` (3 directes, ~15 indirectes) porte une fonctionnalité réelle ; à garder.

**Recommandation.** Retirer `zerolog` (API-03, option 1 ou 2). Remplacer `spinner` par un rafraîchissement `\r` piloté par le `ticker` déjà présent dans `DisplayProgress` (une tranche de 8 caractères de cadre ; ~30 lignes) : moins une dépendance, moins une course, moins une interface et une couture. Décision de produit (la ligne de progression change de forme) : D5 du plan.

#### DEP-03 · Basse · Effort S — Mise à niveau en bloc

**Preuve.** `Makefile:315-318` : `upgrade: go get -u ./... && go mod tidy`. Le livre (p. 323) : *« dependency upgrades should be planned deliberately »*.

**Recommandation.** `go get -u=patch ./...` par défaut, ou supprimer la cible et laisser Dependabot (avec PRO-01) proposer une dépendance à la fois.

#### DEP-04 · Info — Dépendances de test et sous étiquette

`leanovate/gopter` (tests) et `ncw/gmp` (étiquette `gmp`) sont directes ; c'est correct en Go modules et le coût est nul pour le binaire par défaut.

### 4.11 Observabilité (ch. 10, 13, 18, 19)

**Conformités.** `-machine` pour une sortie stable (p. 310) ; indicateurs post-calcul sous `-d` ; la TUI expose `NumGoroutine`, `NumGC`, pauses et tas (p. 292-293) ; codes de sortie stables et documentés (p. 401-402).

#### OBS-01 · Moyenne · Effort M — Pas de canal de diagnostic en production

**Principe.** Ch. 10 « Observability and Metrics Where They Matter » (p. 310) : rendre le système compréhensible ; ch. 19 (p. 498-503) : journaux structurés, clé-valeur ; ch. 19 (p. 509-510) : *« if a gRPC service is important enough to run in production, it is a must to be observable. »*

**Preuve.** Les événements qui expliquent le comportement (GC coupé/rétabli avec octets alloués et cycles, seuils dynamiques ajustés, statistiques du cache FFT) sont **déjà écrits** dans le code, mais vers des loggers `Nop` (API-03). L'utilisateur qui veut savoir si son calcul est passé en FFT, ou combien de cycles GC ont eu lieu malgré `GOGC=off`, n'a aucun moyen de le voir sans recompiler.

**Recommandation.** API-03 option 2 : `*slog.Logger` injecté, `--log-level`. Les six événements existants deviennent immédiatement utiles.

#### OBS-02 · Basse · Effort S — Profilage du binaire

**Principe.** Ch. 8 « Profiling a Running Service » (p. 256-257) ; ch. 9 (p. 293).

**Preuve.** `pprof` n'est atteignable que par `go test -bench`. Un utilisateur qui observe une lenteur à son `n` ne peut pas la profiler.

**Recommandation.** Deux drapeaux `-cpuprofile`/`-memprofile` dans `config`, `runtime/pprof` autour de `Run` dans `app` (une dizaine de lignes).

### 4.12 Documentation et lisibilité (ch. 1, 6, 20)

**Conformités.** Documentation vivante et confrontée à la source (ch. 1 p. 16) : onze figures Mermaid vérifiées par `go list`, relevé de validation daté, chaque chiffre publié rattaché à un artefact ou signalé comme non rejoué ; ADR au format fixe ; `.env.example` ; `CONTRIBUTING.md` complet.

#### DOC-01 · Moyenne · Effort M — Archéologie dans les commentaires

**Principe.** Ch. 6 (p. 179) : *« code is read far more often than it is written »* ; ch. 15 (p. 387) : *« Keep it obvious for newcomers »* ; ch. 1 (p. 13) : *« Explain not just what was chosen but why. »*

**Preuve.**
- 164 références à des identifiants d'audit dans les commentaires de production (`audit OVR-10` ×11, `ADR-0009` ×10, `R4.2` ×8, `M-01` ×14, `H-02` ×6, `P1-04`, `FIB-03`, `SEC-01`, `A2-04`, `APP-05`, `DEAD-09`…). ADR-0010 § References confirme que les documents sources (`audit.md`, `audit Gemini.md`) ont été **purgés de l'arbre** ; `audit.md` de l'audit 2026-09 a connu le même sort (commit « Delete audit.md »).
- 35 % des lignes de production sont des commentaires. Beaucoup portent un *pourquoi* précieux (les meilleurs du dépôt : `finalizeStateReleaseTo`, `putByKey`, `loadProfile`), mais un lecteur nouveau ne peut résoudre `OVR-10` ni `R4.2` sans `git log -S`.

**Recommandation.** Règle dans `CONTRIBUTING.md` : un commentaire porte le *pourquoi* en clair ; un identifiant n'est admis que s'il résout vers un fichier de l'arbre (`ADR-00NN`, `CHANGELOG 2026-09-03`). Passe mécanique : conserver la phrase de justification, retirer l'identifiant nu ou le remplacer par sa cible. `docs/audits/INDEX.md` pour les identifiants qu'on tient à garder. Un grep dans le gate empêche les nouveaux identifiants nus. Ne pas supprimer les rapports d'audit à l'avenir : les archiver sous `docs/audits/` comme les sorties de benchmark (le présent fichier inclus).

#### DOC-02 · Moyenne · Effort S — Corpus bilingue

**Principe.** Ch. 1 « Building a Shared Language » (p. 16).

**Preuve.** `README.md`, `CHANGELOG.md`, `PORTABILITY.md`, ADR-0010, ADR-0011 en français ; `ARCH.md`, `TESTING.md`, `BUILD.md`, `PERFORMANCE.md`, `CALIBRATION.md`, `CONTRIBUTING.md` en anglais ; `ARCH.md` alterne les deux **dans le même document** (légendes de figures en français, sections en anglais) ; code et commentaires en anglais avec un îlot français dans `.golangci.yml`.

**Recommandation.** Décision du mainteneur (D4) : par exemple **français** pour la documentation narrative, **anglais** pour le code, les commentaires, les messages d'erreur et les sujets de commit. L'inscrire dans `CONTRIBUTING.md` et rendre `ARCH.md` monolingue.

#### DOC-03 · Basse · Effort M — Volume et structure

**Principe.** Ch. 1 « A Product Mindset for Every Stage » (p. 19) : *« Would you "buy" this document? »* ; ch. 15 (p. 398) : YAGNI s'applique aussi aux contrats.

**Preuve.** 12 129 lignes de Markdown pour 16 945 lignes de production (0,72 : 1). `README.md` ouvre sur un tableau d'historique d'audits de 25 lignes **avant** « Démarrage rapide ». `ARCH.md` fait 1 193 lignes. Trois passes de resynchronisation documentaire en un mois (CHANGELOG 2026-09-04) mesurent le coût d'entretien de ce volume.

**Recommandation.** Déplacer l'historique d'audits vers `docs/audits/HISTORY.md` ; ramener le README à l'essentiel (objet, démarrage, drapeaux, configuration, développement, liens) ; élaguer dans `ARCH.md` ce qui redit une figure (le document énonce lui-même cette règle).

#### DOC-04 · Basse · Effort S — Renvois périmés

**Preuve.**
- `internal/app/doc.go` (API-08).
- `git remote -v` → `https://github.com/agbruneau/Fibonacci.git` ; `README.md:69` et `CONTRIBUTING.md:33` clonent `…/FibGo.git` ; le module s'appelle `github.com/agbruneau/FibGo`. Le livre (ch. 4 p. 67) : *« Use realistic paths from day one … to avoid painful renames. »* Si le dépôt a été renommé, GitHub redirige, mais les documents doivent le dire.

**Recommandation.** Vérifier la redirection ; aligner les URL de clonage ; ne pas renommer le module (coût disproportionné), le documenter.

#### DOC-05 · Info — Style `Parameters:/Returns:`

`CONTRIBUTING.md` impose des blocs *Parameters/Returns* sur chaque fonction exportée, y compris `Name() string`. Le livre préfère des commentaires godoc concis et des exemples exécutables (ch. 7 p. 223-224). Choix de convention ; aucune action.

### 4.13 Sécurité (ch. 2)

**Conformités.** `gosec` dans le lint ; fichiers `0600`, répertoires `0700` ; écriture atomique du profil ; 12 `#nosec` en production, chacun motivé (celui de `loadProfile` est un modèle) ; image `distroless` en `nonroot` ; échappement des scripts de complétion ; aucun secret, aucune entrée réseau.

#### SEC-01 · Moyenne — Vulnérabilités non analysées

Voir DEP-01.

#### SEC-02 · Basse — Image non épinglée, contexte de build non filtré

Voir STR-02.

---

## 5. Ce que le livre prescrit et qui ne s'applique pas ici

| Prescription du livre | Pourquoi elle ne s'applique pas | Décision |
|---|---|---|
| Fichier YAML de configuration + `gopkg.in/yaml.v3` (ch. 12) | Un CLI sans secret ni environnement multiple ; drapeaux + variables couvrent le besoin, et le profil de calibration est déjà un fichier | Ne pas ajouter |
| Espace de travail multi-modules `go.work` (ch. 11) | Un binaire, une équipe ; le livre lui-même dit de rester monolithique tant que ça ne fait pas mal | Ne pas scinder |
| Branches `main/staging/prod` (ch. 11) | Mono-mainteneur, pas de déploiement continu | Étiquettes de version suffisent (PRO-04) |
| Répertoires `models/ports/service/adapters` (ch. 14) | La superposition actuelle est équivalente et gardée par test ; renommer coûterait sans ajouter de contrainte | Conserver |
| Serveur HTTP, timeouts, `/healthz`, headers de corrélation (ch. 13) | Pas de réseau | Sans objet ; l'arrêt propre par signaux est repris (CON-01) |
| OpenAPI, versionnage d'API, RFC 9457 (ch. 15) | Pas d'API ; les codes de sortie POSIX tiennent lieu de contrat et sont documentés | Sans objet |
| Base de données, migrations, ORM (ch. 16) | Pas de persistance hors le profil JSON | Sans objet |
| Testcontainers (ch. 17) | Pas de dépendance externe à conteneuriser ; les e2e lancent le binaire | Sans objet ; la partie CI du chapitre est reprise (PRO-01) |
| Pulsar, Avro, DLQ (ch. 18) ; gRPC, interceptors (ch. 19) | Pas de messagerie ni de RPC | Sans objet ; la journalisation structurée est reprise (OBS-01) |
| Étiquettes `//go:build integration` (ch. 7) | Les e2e doivent tourner par défaut ; `-short` fait le tri | Conserver |
| `iota` pour `GCMode` (ch. 5) | Les valeurs sont des chaînes de drapeau | Conserver |
| Scinder les fonctions au-delà de ~20 lignes (ch. 6) | Seuils explicites, mesurés, justifiés dans `.golangci.yml` | Conserver |

---

## 6. Conformités notables

Pour que le tableau soit juste, ce que le dépôt fait mieux que la plupart des projets que le livre décrit :

1. **Modèle d'erreurs** (ch. 6) : typage, enveloppement, sentinelles, `Unwrap`, hygiène de `Close`, codes de sortie — complet.
2. **Concurrence** (ch. 9, 20) : `errgroup`, sémaphores, propriété des canaux, contexte en premier argument, arrêt propre à 130, atomiques, `-race` propre.
3. **Tests** (ch. 7) : oracles indépendants, propriétés, benchmarks avec protocole `benchstat`, exemples, e2e, gate d'architecture, 96,7 %.
4. **Mesure avant affirmation** (ch. 8) : chaque chiffre publié a un artefact ou est marqué non rejoué ; les décisions de performance sont prises en double ordre.
5. **Décisions tracées** (ch. 1) : ADR avec candidats rejetés et clauses « à revoir si ».
6. **Frontières** (ch. 14) : ports côté consommateur, adapters CLI/TUI, domaine sans UI, gate exécutable.

---

## 7. Plan d'exécution

### 7.1 Principes

- **Une branche** : `audit/2026-09-livre`. Un commit par tâche, message *Conventional Commits* portant l'identifiant (`fix(app): wrap every mode in signal.NotifyContext (CON-01)`).
- **Le gate après chaque tâche** : `bash scripts/check.sh` (WSL) ou `pwsh scripts/check.ps1`, plus `go test -race -shuffle=on -count=1 ./...` dès T03.
- **Performance** : toute tâche qui touche `internal/fibonacci`, `internal/bigfft` ou `internal/progress` est mesurée par `benchstat` en **double ordre** (protocole ADR-0009 R4, `-count=8`, `BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)`), seuil 5 %.
- **Décisions** : un `ADR-0012` consigne les décisions préalables (§ 7.2), les tâches rejetées et les mesures. Ce fichier `audit.md` est archivé sous `docs/audits/audit-2026-09-livre.md` en fin de plan (DOC-01) plutôt que supprimé.
- **Ordre** : filet de sécurité d'abord (phase 1), correctifs de comportement ensuite (phase 2), refactorisations de frontières quand le filet tient (phase 3), lisibilité en dernier (phase 4). Deux tâches restent optionnelles, marquées *(opt.)* : T20 et T30.

### 7.2 Décisions préalables du mainteneur

Tranchées par le mainteneur le 2026-09-07, en suivant les recommandations de l'audit. Aucune tâche du plan n'est plus bloquée par une décision ; T21 et T22 passent d'optionnelles à retenues.

| # | Question | Décision | Motif | Tâches débloquées |
|---|---|---|---|---|
| D1 | Réintroduire une CI GitHub Actions ? | **Oui** | La clause de révision d'ADR-0010 D4 est atteinte : trois outils cassés sur l'hôte de référence, non détectés. À consigner dans ADR-0012 comme renversement d'ADR-0004 §B3 / ADR-0010 D4 | T01, T04, T08 (build 386), T11 (job planifié) |
| D2 | Journalisation : `slog` injecté ou suppression de `zerolog` ? | **`slog` injecté** (`--log-level`, `FIBCALC_LOG_LEVEL`) | Aligne le domaine sur le livre (aucun tiers hors adapters) et rend observables six événements déjà écrits ; `zerolog` et `testmain_test.go` disparaissent | T14 |
| D3 | Renommer `internal/errors` → `internal/apperrors` ? | **Oui, maintenant** | Diff purement mécanique (un commit `git mv`, un commit `sed`), aucun risque fonctionnel ; le faire pendant que T15 touche déjà le paquet évite un second passage sur les importateurs | T22 (à enchaîner juste après T15) |
| D4 | Langue des documents ? | **Français** pour la documentation narrative (README, CHANGELOG, ADR, `docs/*.md`) ; **anglais** pour le code, les commentaires, les messages d'erreur, les sujets de commit et `.golangci.yml` | Le mainteneur écrit en français ; le code est déjà en anglais ; règle inscrite dans `CONTRIBUTING.md` | T24, T25 |
| D5 | Remplacer `briandowns/spinner` par un rendu maison ? | **Oui** | Moins une dépendance directe (et quatre transitives), moins une course (CONC-01), moins une interface et une couture de test ; la forme de la ligne de progression change, goldens régénérés et relus | T21 |

### 7.3 Phases et tâches

Effort : S < 2 h, M ½–1 j, L 1–3 j. « Vérif. » = commande dont le résultat conclut la tâche.

#### Phase 0 — Cadrage (½ j)

| ID | Tâche | Constats | Critère d'acceptation | Vérif. | Effort |
|---|---|---|---|---|---|
| T00 | Branche `audit/2026-09-livre` ; base de référence `benchstat` (`-count=8`) et couverture conservées dans le scratchpad ; `docs/adr/0012-audit-2026-09-livre.md` créé depuis `0000-template.md` avec D1–D5 tranchées | — | ADR-0012 en statut *Proposed* avec les cinq décisions | `git log -1`, `ls docs/adr/0012*` | S |

#### Phase 1 — Filet de sécurité outillage (1 j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| T01 | Workflow CI : matrice `ubuntu-latest`/`windows-latest`, `go-version-file: go.mod`, étapes `gofmt -l`, `go vet`, `go build`, `go test -race -shuffle=on -count=1 -coverprofile`, plancher 80 % (`scripts/check.sh --coverage-only` sous Ubuntu), lint (T02), `govulncheck` (T04) ; job Ubuntu `libgmp-dev` + `-tags gmp` ; job planifié hebdomadaire fuzz (T11) | PRO-01, STR-01 | `.github/workflows/ci.yml` ; retirer la ligne `!.github/workflows/coverage.yml` de `.gitignore` | Vert sur `push` et `pull_request` ; l'étape `gmp` compile et teste `calculator_gmp.go` pour la première fois depuis un hôte sans libgmp | Exécution GitHub ; `gh run list` | M | — (D1 tranchée) |
| T02 | Versions d'outils dans `scripts/tools.env` (source unique : deux gates + Makefile + CI), invoquées par `go run <pkg>@<version>` ; retirer `install-tools` ; devcontainer sans `@latest` | PRO-02 | `scripts/tools.env`, `Makefile`, `scripts/check.*`, `.devcontainer/devcontainer.json`, `CONTRIBUTING.md`, `docs/BUILD.md` | lint et `govulncheck` s'exécutent à la version épinglée sur tout hôte ; aucun binaire de `PATH` requis ; graphe de modules inchangé (201) | `go mod tidy -diff` vide ; `bash scripts/check.sh` | S | — |
| T03 | `-shuffle=on -count=1` dans `check.sh`, `check.ps1`, `make test`, `make test-win` | TST-03 | `scripts/check.*`, `Makefile` | Les deux scripts passent ; toute défaillance d'ordre est traitée par T10 | `grep -c shuffle scripts/*` = 2 | S | — |
| T04 | `go tool govulncheck ./...` comme étape dure des deux gates | DEP-01, SEC-01 | `scripts/check.*`, `Makefile` | Étape présente, exit 0 | `bash scripts/check.sh` | S | T02 |
| T05 | `.dockerignore` ; `Dockerfile` : `ARG VERSION COMMIT BUILD_DATE` injectés par la même `LDFLAGS` que le Makefile (ou `make build`) ; digests résolus sur un hôte Docker | STR-02, SEC-02 | `Dockerfile`, `.dockerignore`, `docs/BUILD.md` | `docker run fibcalc:local --version` affiche le `git describe` ; les deux `TODO(SEC-04)` fermés | Sur hôte Docker : `docker build` puis `--version` | S | Hôte Docker |

#### Phase 2 — Correctifs de comportement (1½ j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| T06 | `signal.NotifyContext` **une fois** dans `Application.Run` ; `context.WithTimeout(cfg.Timeout)` pour `runCalibration` et la phase `--auto-calibrate` ; retirer les trois blocs dupliqués (la TUI garde son délai par génération) | CON-01 | `internal/app/app.go`, `internal/app/calculate.go`, `internal/app/app_test.go` | Un seul `NotifyContext` dans `internal/app` ; test : `--calibrate` sur contexte annulé → 130 + « Calibration interrupted » ; test : `--calibrate -timeout 1ms` → 2 | `grep -c NotifyContext internal/app/*.go` = 1 ; `go test ./internal/app/` | S | — |
| T07 | Propager `--memory-limit` dans `Options.MemoryLimitBytes` (CLI et TUI) | MEM-01 | `internal/app/calculate.go`, `internal/app/app.go`, `internal/tui/commands.go`, `internal/orchestration/orchestration_spy_test.go` | Le test-espion voit `MemoryLimitBytes == limite analysée` ; `CanCalculate` devient atteignable depuis le binaire | `go test ./internal/orchestration/ ./internal/app/` | S | — |
| T08 | Une seule définition de `maxReasonableWords` (`1 << (bits.UintSize - 4)`), exportée de `memory` (ou T20) ; build `GOARCH=386` en CI (build seulement) ; si une dépendance bloque, contrainte de build explicite avec message | TYP-01 | `internal/fibonacci/memory/arena.go`, `internal/fibonacci/fastdoubling.go`, `docs/PORTABILITY.md`, CI | `GOOS=linux GOARCH=386 go build ./...` exit 0, ou message explicite ; une seule déclaration | `grep -rn 'maxReasonableWords =' internal` = 1 | S | — |
| T09 | Test *flaky* : retirer `t.Parallel`, ajouter `resetStatePoolForTest()` (réassigne `statePool`, vide `cachedState`) appelé en tête | TST-01 | `internal/fibonacci/state_cache_test.go` (+ helper dans un `_test.go`) | 50 passes vertes consécutives ; paquet vert sous `-shuffle=on -count=5` | `go test -run TestStateBump_PinnedAcrossCachedCalls -count=50 ./internal/fibonacci/` | M | T03 |
| T10 | `os.Setenv` → `t.Setenv` partout ; retirer `t.Parallel` des tests qui touchent l'environnement | TST-02 | `internal/config/config_test.go`, `env_test.go`, `config_extra_test.go`, `internal/cli/provider_test.go`, `internal/ui/themes_test.go` | Aucun `os.Setenv` dans les tests ; vert sous `-race -shuffle=on -count=3` | `grep -r os.Setenv --include=*_test.go internal` vide | S | T03 |
| T11 | Graines fuzz ≥ 1 801 mots (2 048, 4 096) pour `FuzzMul`/`FuzzSqr` ; `make fuzz-smoke` (30 s/cible) ; job CI hebdomadaire | TST-04 | `internal/bigfft/fft_fuzz_test.go`, `Makefile`, CI, `docs/TESTING.md` | Le rejeu des graines couvre `fftmul`/`fftsqr` | `go test -run 'Fuzz(Mul\|Sqr)' -coverprofile=c.out ./internal/bigfft/ && go tool cover -func=c.out \| grep -E 'fftmul\|fftsqr'` > 0 % | S | T01 (job) |
| T12 | `Register` valide (nom vide, créateur nil, doublon → erreur) ; `NewDefaultFactory` panique sur sa propre erreur ; `TestFactory.List` trie ; `TestFactory.Register` honnête (stocke ou refuse) — ou supprimé par T17 | API-02, API-01 | `internal/fibonacci/registry.go`, `testing.go`, `registry_test.go` | Tests doublon/nil verts ; `CONTRIBUTING.md` reste vrai | `go test ./internal/fibonacci/ -run Registry` | S | — |
| T13 | Renvois périmés : exemple de `app/doc.go` ; URL de clonage (vérifier la redirection `FibGo` → `Fibonacci`) ; commentaire `install-tools` | API-08, DOC-04 | `internal/app/doc.go`, `README.md`, `CONTRIBUTING.md` | `go vet` ne signale rien ; les URL clonent | `git clone <URL> /tmp/x` | S | — |

#### Phase 3 — Frontières et dépendances (3½–4 j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| T14 | Journalisation `slog` : `Options.Logger *slog.Logger` (nil → `slog.DiscardHandler`) ; convertir les six émetteurs ; `bigfft` garde un `atomic.Pointer[slog.Logger]` + `SetTransformCacheLogger` ; `--log-level`/`FIBCALC_LOG_LEVEL` dans `config`, handler texte sur stderr (JSON sous `-machine`) construit dans `app` ; supprimer `zerolog`, `testmain_test.go`, les setters de test devenus inutiles ; journaliser la récupération d'observateur | API-03, OBS-01, API-06, TST-05, DEP-02 | `internal/fibonacci/{calculator,options}.go`, `memory/gc_control.go`, `threshold/manager.go`, `progress/observers.go`, `bigfft/fft_cache.go`, `internal/app/*`, `internal/config/*`, `go.mod` | `go list -deps ./... \| grep zerolog` vide ; `fibcalc -n 2000000 -algo fast --log-level debug` affiche `gc disabled`/`gc re-enabled` sur stderr ; benchstat neutre en double ordre | `go test -race ./...` ; benchstat | L | — (D2 tranchée) |
| T15 | Scinder erreurs/présentation : `apperrors.ExitCodeFor(err) int` pur ; `cli.WriteCalculationStatus` ; supprimer `ColorProvider`, `DefaultColorProvider`, `CLIColorProvider`, `provider.go` ; `RunCalibration` perd `colorProvider` | API-04, ARC-02 | `internal/errors/handler.go`, `internal/cli/*`, `internal/tui/bridge.go`, `internal/calibration/calibration.go`, `internal/app/calculate.go` | `internal/errors` n'importe ni `io` ni couleur ; goldens CLI inchangés ; codes de sortie e2e inchangés | `go test ./... ; go test ./test/e2e/` | M | — |
| T16 | Port `calibration.Reporter` (4 méthodes) implémenté dans `cli` ; `NopReporter` pour les tests ; `calibration` cesse d'importer `ui` ; règle `calibration → ui` dans `arch_test.go` | ARC-01, STR-04 | `internal/calibration/{calibration,io,strategy_fast}.go`, `internal/cli/calibration_reporter.go`, `internal/app/app.go`, `internal/arch_test.go`, tests | `go list -f '{{.Imports}}' ./internal/calibration` sans `internal/ui` ; les tests de calibration n'appellent plus `StripAnsiCodes` | `go test ./internal/... -run Architecture` | M | T15 |
| T17 | Interface côté consommateur : `orchestration.CalculatorSource {List; Get}` ; `app.Application.Factory` en `*fibonacci.DefaultFactory` ; supprimer `CalculatorFactory` et `TestFactory` ; doublure de deux méthodes dans les tests d'orchestration | API-01 | `internal/orchestration/{interfaces,calculator_selection}.go`, `internal/fibonacci/{registry,testing}.go`, `internal/app/app.go`, tests | `grep -rn CalculatorFactory internal cmd` vide ; app/orchestration tests verts | `go build ./... && go test ./...` | M | T12 |
| T18 | `threshold.Tuning` par valeur : `NewDynamicThresholdManagerFromConfig(cfg, tuning)` ; supprimer `SetTuning`, les cinq `var`, `wireThresholdTuning` et `thresholdTuningOnce` ; `app` traduit `config.DefaultThresholdTuning` en `Options.ThresholdTuning` | TYP-02, CON-02 | `internal/fibonacci/threshold/manager.go`, `internal/fibonacci/{fastdoubling,options}.go`, `internal/app/app.go`, `app_tuning_test.go`, `threshold/tuning_test.go` | Aucune `var` mutable dans `threshold` ; le test A2-04 devient un test de propagation d'`Options` | `grep -n '^var' internal/fibonacci/threshold/manager.go` vide ; benchstat neutre | M | — |
| T19 | `MicroBenchTimeout`/`MicroBenchTestSizes` → constantes ; les champs de `MicroBenchmark` seuls configurables ; cache FFT configuré une fois par `ExecuteCalculations` | TYP-02 | `internal/calibration/microbench.go`, `internal/fibonacci/{options,calculator}.go`, `internal/orchestration/orchestrator.go` | Aucune `var` de paquet mutable dans `microbench.go` ; un seul `SetTransformCacheConfig` par exécution (test-espion) | `go test ./internal/calibration/ ./internal/fibonacci/` | S | — |
| T20 | *(opt.)* Paquet feuille `internal/fibmath` (log₂ φ, `MaxFibUint64`, `MaxReasonableWords`, `BitsFor`, `WordsFor`) utilisé par `fibonacci`, `memory`, `config` ; `bigfft` au choix (mettre `ARCH.md` à jour) | TYP-04 | nouveau paquet, `constants.go`, `budget.go`, `arena.go`, `fastdoubling.go` | Un seul `0.69424` hors `bigfft` ; gate d'architecture vert | `grep -rn 0.69424 internal --include=*.go` | S | T08 |
| T21 | Retirer `briandowns/spinner` : rendu `\r` par le `ticker` de `DisplayProgress` ; supprimer `Spinner`, `realSpinner`, `newSpinner`, `ui_suffix_race_test.go` ; goldens de progression mis à jour | DEP-02, CON-04 | `internal/cli/{ui,display}.go`, tests, `go.mod` | `go list -deps \| grep spinner` vide ; `NumGoroutine` stable pendant la progression (test) | `go test -race ./internal/cli/` | M | — (D5 tranchée) |
| T22 | `git mv internal/errors internal/apperrors` + imports + règle `arch_test.go` (un commit `git mv`, un commit `sed`) | STR-03 | tout importateur | Build vert ; `find internal -type d -name errors` vide | `go build ./...` | M | T15 |

#### Phase 4 — Lisibilité, documentation, hygiène (1½ j)

| ID | Tâche | Constats | Fichiers | Critère d'acceptation | Vérif. | Effort | Dép. |
|---|---|---|---|---|---|---|---|
| T23 | Politique de commentaires dans `CONTRIBUTING.md` ; passe mécanique sur les 164 identifiants (garder le *pourquoi*, retirer ou résoudre l'identifiant) ; `docs/audits/INDEX.md` ; grep de garde dans le gate ; archiver ce fichier sous `docs/audits/` | DOC-01, PRO-05 | `internal/**/*.go` (commentaires), `CONTRIBUTING.md`, `docs/audits/INDEX.md`, `scripts/check.*` | 0 identifiant nu hors `docs/audits/` | `grep -rhoE '\b(OVR\|DEAD\|FIB\|SEC\|CONC\|ERR\|APP\|ARCH\|CAL\|FFT\|GATE)-[0-9]+' internal cmd --include=*.go \| wc -l` = 0 | M | — |
| T24 | Règle de langue dans `CONTRIBUTING.md` ; `ARCH.md` monolingue ; `.golangci.yml` en anglais | DOC-02 | `CONTRIBUTING.md`, `docs/ARCH.md`, `.golangci.yml` | Revue humaine | — | S | — (D4 tranchée) |
| T25 | README recentré (≤ 200 lignes) ; historique d'audits → `docs/audits/HISTORY.md` ; `ARCH.md` élagué là où une figure suffit | DOC-03 | `README.md`, `docs/audits/HISTORY.md`, `docs/ARCH.md` | Le démarrage rapide est visible sans défilement ; aucune information perdue (déplacée) | `wc -l README.md` ≤ 200 | M | T24 |
| T26 | Renommer les dix fichiers de test vagues par thème | TST-08 | `internal/**/*_test.go` | Aucun `_extra_`/`_more_`/`misc_`/`_advanced_` | `git ls-files '*_test.go' \| grep -E 'extra\|more\|misc\|advanced'` vide | S | — |
| T27 | `errors.Join` dans `AppConfig.Validate` ; tests ajustés | API-07, CFG-01 | `internal/config/config.go`, `config_test.go` | Deux drapeaux invalides → deux messages en une sortie | `go test ./internal/config/` | S | — |
| T28 | `ProfileMaxAge` et `TUITheme` dans `AppConfig` via `envOverrides` ; `calibration` et `ui` les reçoivent en paramètre ; `NO_COLOR` lu une fois | CFG-02 | `internal/config/{config,env}.go`, `internal/calibration/calibration.go`, `internal/ui/themes.go`, `internal/tui/styles.go`, `.env.example` | `os.Getenv` hors `config/env.go` = 0 en production | `grep -rn 'os.Getenv\|os.LookupEnv' internal --include=*.go --exclude=*_test.go` | S | — |
| T29 | Règle `cli → fibonacci` dans `arch_test.go` (et `calibration → ui`, T16) | STR-04 | `internal/arch_test.go`, `internal/cli/doc.go` | Sept règles, test vert | `go test ./internal/ -run Architecture` | S | T16 |
| T30 | *(opt.)* `-cpuprofile`/`-memprofile` sur le binaire | OBS-02 | `internal/config/config.go`, `internal/app/app.go`, `docs/PERFORMANCE.md` | `fibcalc -n 10000000 -cpuprofile cpu.prof` produit un profil lisible par `go tool pprof` | manuel | S | — |
| T31 | e2e : répertoire temporaire créé par `TestMain` du paquet, nettoyé | TST-06 | `test/e2e/cli_e2e_test.go` | Deux `go test ./test/e2e/` simultanés n'interfèrent pas | manuel (deux terminaux) | S | — |
| T32 | Clôture : ADR-0012 en *Accepted* (décisions, rejets, mesures benchstat), entrée `CHANGELOG.md`, `docs/BUILD.md`/`TESTING.md`/`PORTABILITY.md` resynchronisés (CI, `go tool`, shuffle, 386, slog), version `v4.1.0` | PRO-04 | `docs/adr/0012-*.md`, `CHANGELOG.md`, `docs/*.md` | Chaque affirmation nouvelle rattachée à une commande rejouée, selon la règle du dépôt | relecture | S | tout |

### 7.4 Protocole de vérification

À la fin de chaque phase :

```bash
go build ./... && go vet ./... && gofmt -l . && go mod tidy -diff
go test -race -shuffle=on -count=1 ./...
go tool golangci-lint run ./...          # après T02
go tool govulncheck ./...                # après T02/T04
bash scripts/check.sh                    # WSL ; ou pwsh scripts/check.ps1
```

À la fin de la phase 3, et pour toute tâche marquée « benchstat » :

```bash
go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -run='^$' -count=8 ./internal/fibonacci/ > new.txt
benchstat base.txt new.txt               # puis l'inverse, même session, par ADR-0009 R4
```

Critère : aucune régression > 5 % confirmée dans les deux ordres.

### 7.5 Risques et retours arrière

| Tâche | Risque | Mitigation / retour arrière |
|---|---|---|
| T06 | Changer la sémantique de Ctrl-C en calibration (voulu) ; oublier un mode | Test par mode ; revert du commit isolé |
| T14 | Chemin chaud : le `defer` de journalisation dans `CalculateWithObservers` | `slog.DiscardHandler` court-circuite avant formatage ; benchstat double ordre ; revert isolé |
| T17 | Signature de `WithFactory` et tests d'`app` | Le type concret est déjà ce que tout le monde construit ; diff localisé |
| T18 | Le test A2-04 disparaît avec le mécanisme | Le remplacer par un test de propagation d'`Options` (valeur) — plus fort |
| T21 | Sortie visible modifiée (goldens) | Décision D5 explicite ; goldens régénérés et relus |
| T22 | Diff très large (imports) | Un commit pur `git mv` + un commit `sed`, sans autre changement |
| T23 | Perdre un *pourquoi* en retirant un identifiant | Règle : on retire l'identifiant, jamais la phrase ; relecture par diff |

### 7.6 Définition de terminé

- Toutes les tâches non optionnelles fermées, chaque commit vert au gate et en CI.
- ADR-0012 en *Accepted*, avec la liste des tâches optionnelles non retenues et leur motif.
- Aucune régression `benchstat` > 5 % en double ordre.
- `audit.md` archivé sous `docs/audits/` ; ce fichier n'est plus à la racine.
- Les trois outils cassés sur l'hôte (`govulncheck`, `gosec`, `staticcheck`) sont soit épinglés par `go tool`, soit retirés de la documentation.

### 7.7 Effort total estimé

| Phase | Effort |
|---|---|
| 0 — Cadrage | ½ j |
| 1 — Filet outillage | 1 j |
| 2 — Correctifs de comportement | 1½ j |
| 3 — Frontières et dépendances (T21 et T22 incluses) | 3½ j |
| 3 — Optionnelle T20 | + ½ j |
| 4 — Lisibilité et documentation | 1½ j |
| **Total** | **8 à 9 jours-personne** |

### 7.8 Tableau de suivi d'exécution

Mis à jour à chaque tâche. Statuts : ☐ à faire · ⟳ en cours · ☑ terminée et
vérifiée · ⊘ écartée (motif consigné).

Exécution menée directement sur `main` (le mainteneur pousse sur `main` à la
fin de chaque phase), et non sur la branche `audit/2026-09-livre` prévue en
T00.

**Résultat : 33 tâches sur 33 exécutées et vérifiées** (T00 à T32, les deux
optionnelles T20 et T30 incluses), en cinq commits poussés sur `main`. Un seul
écart au plan, dans T25, consigné dans sa ligne.

La CI a trouvé cinq défauts qu'aucun gate local ne pouvait produire, sur ses
trois premiers passages : un test attendant une valeur que le code ne produit
plus depuis FIB-02 (branche jamais prise sur un hôte à 24 cœurs), la coupure
PowerShell de `-coverprofile=`, le build 32 bits, la collision du nom `GOFLAGS`
dans le `Makefile`, et une vulnérabilité de la bibliothèque standard atteignable
dans la version de Go déclarée. C'est la justification de PRO-01, obtenue en
trois minutes.

| Phase | Tâche | Statut | Commit | Vérification |
|---|---|---|---|---|
| 0 | T00 Cadrage, base de référence, ADR-0012 | ☑ | — | base `benchstat` `-count=8` et couverture 96,7 % capturées ; ADR-0012 en *Proposed* |
| 1 | T01 Workflow CI | ☑ |  | `.github/workflows/ci.yml` : 5 jobs (gate ubuntu+windows, gmp, cross-build 386/arm64, docker, fuzz hebdo). YAML validé ; exécution GitHub à confirmer au premier push |
| 1 | T02 Épinglage des outils (`go run pkg@version`) | ☑ |  | `scripts/tools.env` lu par les 2 gates, le Makefile et la CI ; `go run pkg@version`. Directive `tool` mesurée puis rejetée (montait `gopsutil` v4.26.3→v4.26.7, graphe 201→450) |
| 1 | T03 `-shuffle=on -count=1` dans les gates | ☑ |  | gate PowerShell vert avec ordre aléatoire ; ajouté aussi à `make test`/`test-win` et à la CI |
| 1 | T04 `govulncheck` dans les gates | ☑ |  | étape 6 dure dans les deux gates ; `v1.7.0` tourne sous go1.27 et sort 0 (1 vuln non appelée dans `x/text` indirect) |
| 1 | T05 `.dockerignore`, version et digests | ☑ |  | `.dockerignore` ajouté ; le `Dockerfile` délègue à `make build` avec `VERSION/COMMIT/BUILD_DATE`. Digests SEC-04 fermés après l'étiquette : résolus par le job CI `docker` (run 34135480664, digests d'index OCI multi-arch), épinglés au commit `24cfd38` ; le job rapporte la dérive tag/digest à chaque exécution |
| 2 | T06 Signaux et délai pour tous les modes | ☑ |  | `signal.NotifyContext` unique dans `Run` (test de garde) ; `--timeout` appliqué à `--calibrate` et à `--auto-calibrate` ; `findBest` consulte son contexte |
| 2 | T07 Propagation de `MemoryLimitBytes` | ☑ |  | `AppConfig.MemoryLimitBytes` résolu par `validateMemoryBudget`, lu par le CLI et la TUI ; 4 tests dont un espion d'options |
| 2 | T08 `maxReasonableWords` unique, build 386 | ☑ |  | constante unique `memory.MaxReasonableWords = 1 << (bits.UintSize-4)` ; `GOARCH=386`, `arm` et `arm64` compilent (exit 0) ; garde `math.MaxInt` |
| 2 | T09 Test *flaky* isolé | ☑ |  | `resetStatePoolForTest` + test séquentiel ; 50 passes isolées et 10 passes `-race -shuffle=on` du paquet, toutes vertes |
| 2 | T10 `t.Setenv` au lieu d'`os.Setenv` | ☑ |  | 0 `os.Setenv` restant dans les tests ; `TestTUIFlag` n'est plus parallèle ; `testutil.Unsetenv` ajouté (pas de `t.Unsetenv` en stdlib) |
| 2 | T11 Graines fuzz au-delà du seuil FFT | ☑ |  | graines dimensionnées en mots relatifs au seuil ; le rejeu couvre `fftmul` et `fftsqr` à 100 % ; `make fuzz-smoke` + job CI ; 5,8 M exécutions sans échec |
| 2 | T12 Contrats du registre | ☑ |  | `Register` refuse nom vide, créateur nil et doublon ; `NewDefaultFactory` panique sur son propre bogue ; `TestFactory.List` trié ; 6 tests |
| 2 | T13 Renvois périmés | ☑ |  | exemple `app/doc.go` aligné sur `main.go` ; URL de clonage vers `Fibonacci.git` ; écart de chemin de module documenté et vérifié |
| 3 | T14 Journalisation `slog` | ☑ |  | `Options.Logger *slog.Logger` injecté ; 6 émetteurs convertis ; `--log-level` + `FIBCALC_LOG_LEVEL` ; `zerolog` et `testmain_test.go` supprimés ; vérifié au binaire (silencieux par défaut, `gc disabled`/`gc re-enabled` en debug) |
| 3 | T15 Scission erreurs / présentation | ☑ |  | `apperrors.ExitCodeFor` pur + `cli.WriteCalculationStatus` ; `ColorProvider`, `DefaultColorProvider`, `CLIColorProvider` et `provider.go` supprimés (indirection doublement redondante avec `ui.InitTheme`) |
| 3 | T16 Port `calibration.Reporter` | ☑ |  | port à 4 méthodes + `NopReporter` ; `CalibrationReporter` dans `cli` ; `calibration` n'importe plus `ui` ni `format` ; règle d'architecture ajoutée |
| 3 | T17 Interface côté consommateur | ☑ |  | `orchestration.CalculatorSource` (2 méthodes) + `app.CalculatorRegistry` ; `fibonacci.CalculatorFactory` supprimée ; `TestFactory` perd son `Register` menteur |
| 3 | T18 `threshold.Tuning` par valeur | ☑ |  | 5 `var` de paquet, `SetTuning`, `wireThresholdTuning` et son `sync.Once` supprimés ; `Tuning` copiée dans chaque manager ; le protocole A2-04 n'a plus rien à protéger |
| 3 | T19 Constantes de micro-bench, cache FFT | ☑ |  | `MicroBenchTimeout()`/`MicroBenchTestSizes()` (copie) ; `SetTransformCacheConfig` et `SetTransformCacheLogger` ignorent une écriture identique |
| 3 | T20 *(opt.)* Paquet `internal/fibmath` | ☑ |  | `GrowthFactor`, `MaxUint64Index`, `BitsFor` ; 2 des 3 duplications de log₂ φ et celle de 93 supprimées ; `bigfft` garde son littéral (kernel sans import interne) avec renvoi ; 4 tests contre la définition |
| 3 | T21 Retrait de `briandowns/spinner` | ☑ |  | `progressLine` (dessin sur le ticker existant) ; `Spinner`, `realSpinner`, `newSpinner` et le test de course supprimés ; 2 dépendances directes en moins (198 modules) |
| 3 | T22 Renommage `internal/apperrors` | ☑ |  | `git mv` + réécriture des imports et des renvois de documentation ; 21 paquets verts |
| 4 | T23 Politique de commentaires | ☑ |  | `docs/audits/INDEX.md` (toutes les familles de préfixes résolubles), politique dans `CONTRIBUTING.md`, et `TestAuditIdentifiersResolve` comme garde-fou — un test plutôt qu'un grep dupliqué dans deux shells. 8 familles non listées trouvées et ajoutées |
| 4 | T24 Règle de langue | ☑ |  | inscrite dans `CONTRIBUTING.md` (français pour le narratif, anglais pour le code) ; conforme à ADR-0012 D4 |
| 4 | T25 README et ARCH recentrés | ☑ |  | README 494 → 400 lignes ; historique → `docs/audits/HISTORY.md`, chemin de calcul → `ARCH.md`, dépannage → `BUILD.md`. **Écart au plan** : la cible ≤ 200 lignes n'est pas atteinte et est abandonnée — le reste est de la prose technique mesurée, la couper aurait supprimé des affirmations vérifiées. Le critère réel (« démarrage rapide sans défilement ») est tenu |
| 4 | T26 Renommage des fichiers de test | ☑ |  | 10 fichiers renommés par thème ; plus aucun `_extra_`, `_more_`, `misc_`, `_advanced_`, `_exhaustive_` |
| 4 | T27 `errors.Join` dans `Validate` | ☑ |  | toutes les vérifications s'exécutent, `errors.Join` les assemble ; `errors.As` trouve toujours un `ConfigError` ; 3 tests |
| 4 | T28 Variables d'environnement centralisées | ☑ |  | `ProfileMaxAge` et `TUITheme` dans `AppConfig` avec drapeaux `--profile-max-age` / `--tui-theme` ; `os.Getenv` hors `config/env.go` réduit au seul `NO_COLOR` (convention externe, un seul lecteur) |
| 4 | T29 Règles d'architecture complétées | ☑ |  | règles `cli → fibonacci` et `calibration → ui|format` ajoutées ; alias `orchestration.ThresholdTuning` pour que la TUI ne perce pas la façade ; 7 règles vertes |
| 4 | T30 *(opt.)* Drapeaux de profilage | ☑ |  | `--cpuprofile` / `--memprofile` ; profils produits et relus par `go tool pprof` (29 nœuds) |
| 4 | T31 Répertoire temporaire e2e | ☑ |  | `TestMain` crée un répertoire `os.MkdirTemp` propre et le supprime ; plus de chemin fixe partagé entre deux exécutions |
| 4 | T32 Clôture, ADR-0012 *Accepted*, `v4.1.0` | ☑ |  | ADR-0012 accepté avec la section « Exécution » et les mesures ; entrée `[4.1.0]` au CHANGELOG ; documents resynchronisés (CI, `-shuffle=on`, 386, `slog`, épinglage) ; ce rapport archivé ici même. La ligne MEM-03 (`-gcflags=-m` dans `PERFORMANCE.md`) avait été oubliée : ajoutée en 7.9 |

### 7.9 Après le plan (commits postérieurs à `v4.1.0`, étiquette non déplacée)

| Objet | Commits | Vérification |
|---|---|---|
| SEC-04 : digests des deux images de base lus sur le runner CI (index OCI multi-arch, `MediaType` vérifié à l'œil), épinglés ; rapport de dérive non bloquant dans le job `docker` | `26f5e3c`, `24cfd38` | run 34136872359 vert, dérive nulle au premier passage |
| Actions GitHub v4/v5 → v7 (dépréciation de Node 20) | `cc87692` | plus d'annotation sur les jobs |
| Clôture : documentation resynchronisée avec l'état final (48 arêtes vérifiées par `go list`, 7 règles, symboles renommés, 5 drapeaux dans le README, profilage du binaire et MEM-03 dans `PERFORMANCE.md`, ligne 2026-09-07 dans `HISTORY.md`) ; `progressLine` sans allocation `[]rune` | — | 22 paquets `-race` verts, couverture 96,1 %, gate `check.ps1` vert |

Une évaluation externe reçue après `v4.1.0` attribuait à Cgo et à GMP les échecs
initiaux des jobs `cross-build` et `docker`. Les journaux disent autre chose :
`cross-build` échouait sur TYP-01 (`1 << 60` en 32 bits), `docker` sur la
collision `GOFLAGS` du `Makefile` ; `CGO_ENABLED=0` est le réglage voulu de
l'image, et le backend GMP est un *build tag* optionnel que le job `gmp` couvre.
Aucun changement de code n'a été fait sur la base de cette évaluation.

---

## Annexe A — Mesures relevées

### A.1 Lignes par paquet (production / test)

| Paquet | Prod | Test |
|---|---|---|
| `cmd/fibcalc` | 38 | 328 |
| `cmd/generate-golden` | 164 | 331 |
| `internal` (gate d'architecture) | 0 | 133 |
| `internal/app` | 542 | 2 146 |
| `internal/bigfft` | 3 523 | 6 369 |
| `internal/calibration` | 1 952 | 2 139 |
| `internal/cli` | 673 | 675 |
| `internal/cli/completion` | 681 | 638 |
| `internal/config` | 958 | 2 220 |
| `internal/errors` | 254 | 498 |
| `internal/fibonacci` | 3 430 | 5 612 |
| `internal/fibonacci/memory` | 523 | 818 |
| `internal/fibonacci/threshold` | 611 | 824 |
| `internal/format` | 178 | 135 |
| `internal/metrics` | 174 | 260 |
| `internal/orchestration` | 608 | 1 312 |
| `internal/progress` | 449 | 703 |
| `internal/testutil` | 24 | 43 |
| `internal/tui` | 1 912 | 3 825 |
| `internal/ui` | 251 | 204 |
| `test/e2e` | 0 | 735 |

### A.2 Couverture par paquet (`go test -short`, 2026-09-07)

`cmd/fibcalc` 90,0 · `cmd/generate-golden` 87,9 · `app` 95,1 · `bigfft` 97,0 · `calibration` 92,7 · `cli` 94,7 · `cli/completion` 98,7 · `config` 97,4 · `errors` 100 · `fibonacci` 94,7 · `memory` 99,3 · `threshold` 98,2 · `format` 100 · `metrics` 98,7 · `orchestration` 99,4 · `progress` 95,2 · `testutil` 100 · `tui` 99,3 · `ui` 97,3 · **total 96,7 %**.

### A.3 Inventaire structurel

| Élément | Compte | Détail |
|---|---|---|
| Interfaces déclarées (prod) | 15 | `tempAllocator`, `CalibrationStrategy`, `Spinner`, `ColorProvider`, `CacheStrategy`, `Calculator`, `CoreCalculator`, `task`, `CalculatorFactory`, `Multiplier`, `DoublingStepExecutor`, `ProgressReporter`, `ResultPresenter`, `ErrorHandler`, `ProgressObserver` |
| Fonctions `init()` | 5 | `bigfft/fft.go`, `bigfft/fft_recursion.go`, `fibonacci/calculator_gmp.go` (étiquette), `fibonacci/matrix_ops.go`, `tui/styles.go` |
| Déclarations `var` de niveau paquet (prod) | 56 | dont pools `sync.Pool`, atomiques `bigfft`, seuils `threshold` non synchronisés, coutures de test |
| `panic(` en production | 16 | 15 dans `bigfft` (invariants, ADR-0002), 1 `MustNewCalculator` |
| `recover()` en production | 14 | 13 dans `bigfft` (`fft.go` 6, `fft_recursion.go` 4, `fft_poly.go` 2, `fermat.go` 1), 1 `progress.Freeze` |
| Sites `%w` | 49 | — |
| `errors.Is`/`errors.As` | 15 | — |
| Lancements `go` (prod) | 8 | tous bornés par sémaphore ou `WaitGroup` |
| Primitives `sync`/`atomic` (prod) | 78 | — |
| `//nolint` (prod / avec tests) | 4 / 5 | tous motivés |
| `#nosec` (prod / avec tests) | 12 / 13 | tous motivés |
| Fichiers de production > 500 lignes | 5 | `fft_cache.go` 703, `fft_poly.go` 650, `microbench.go` 570, `fastdoubling.go` 556, `calibration.go` 550 |

### A.4 Dépendances directes et empreinte

| Module | Usage | Fichiers qui l'importent |
|---|---|---|
| `charmbracelet/bubbletea`, `bubbles`, `lipgloss` | TUI | 10, 4, 7 |
| `rs/zerolog` | journalisation (inerte en production) | 10 (6 en production) |
| `briandowns/spinner` | ligne de progression CLI | 4 |
| `golang.org/x/sync` | `errgroup` | 3 |
| `golang.org/x/sys` | détection SIMD | 2 |
| `shirou/gopsutil/v4` | CPU %/mémoire % (TUI) | 1 |
| `leanovate/gopter` | tests de propriétés | 1 |
| `ncw/gmp` | backend sous étiquette `gmp` | 1 |

Graphe complet : 201 modules.

## Annexe B — Commandes rejouées (2026-09-07, `c6ce7fb`)

```text
go build ./...                              exit 0
go vet ./...                                exit 0
golangci-lint run ./...                     0 issues.            (v2.13.2, go1.27.0)
gofmt -l .                                  (vide)
go mod tidy -diff                           exit 0 (vide)
go mod verify                               all modules verified
go test -short -coverprofile ./...          21 ok, total 96.7%
GOOS=linux GOARCH=386 go build ./...        arena.go:26,29,30 overflows int
go build -tags gmp ./internal/fibonacci/    gmp.h: No such file or directory
govulncheck ./...                           "uses version go1.26 ... runs go1.27" (échec)
gosec ./...                                 internal error: package "os" without types (échec)
staticcheck ./...                           export data version 4 > 2 (échec)
grep -rn NotifyContext internal cmd         3 sites (runTUI, runCalculate, runLastDigits)
grep -rn 'os.Setenv' --include=*_test.go    22 ; t.Setenv : 18
grep -rn shuffle scripts Makefile           0
grep -rn 'log.Trace\|Debug()' (prod)        6 émetteurs, tous Nop ou filtrés
grep -c 'ui.Color' calibration.go io.go strategy_fast.go   19 / 9 / 3
git log --format=%s -60 | grep -cE conventionnel           43 / 60
git ls-files '*.md' | wc -l                 42 (12 129 lignes)
```

## Annexe C — Index des constats

| ID | Sévérité | Titre | Tâches |
|---|---|---|---|
| PRO-01 | Haute | Aucune CI | T01 |
| PRO-02 | Moyenne | Outils non épinglés | T02 |
| PRO-03 | Basse | Commits hors convention | (hook, hors plan) |
| PRO-04 | Info | Étiquettes et version | T32 |
| PRO-05 | Info | Leçons apprises : rapports supprimés | T23 |
| STR-01 | Moyenne | Deux gates divergents | T01 |
| STR-02 | Basse | Dockerfile sans délégation ni version | T05 |
| STR-03 | Basse | `internal/errors` masque `errors` | T22 |
| STR-04 | Basse | Règle d'architecture non gardée | T29 |
| TYP-01 | Moyenne | Constante 32 bits dupliquée | T08 |
| TYP-02 | Moyenne | État mutable de paquet | T18, T19 |
| TYP-03 | Info | Énumérations en chaînes | — |
| TYP-04 | Basse | Facteur de croissance dupliqué | T20 (opt.) |
| TYP-05 | Info | `AppConfig` par valeur | — |
| API-01 | Moyenne | Fabrique : interface fournisseur à 5 méthodes | T17 |
| API-02 | Basse | Contrats de registre | T12 |
| API-03 | Moyenne | Journalisation dans le domaine | T14 |
| API-04 | Moyenne | Présentation dans `errors` | T15 |
| API-05 | Basse | Pile dans une valeur d'erreur | T14 |
| API-06 | Basse | Panics d'observateurs avalés | T14 |
| API-07 | Basse | Validation au premier échec | T27 |
| API-08 | Basse | Doc de paquet périmée | T13 |
| API-09 | Info | Longueur des fonctions | — |
| TST-01 | Haute | Test *flaky* conservé | T09 |
| TST-02 | Moyenne | `os.Setenv` + `t.Parallel` | T10 |
| TST-03 | Moyenne | Pas de `-shuffle` | T03 |
| TST-04 | Moyenne | Fuzz sans mutation, graines sous le seuil FFT | T11 |
| TST-05 | Basse | `TestMain` inutile | T14 |
| TST-06 | Basse | Binaire e2e en chemin fixe | T31 |
| TST-07 | Info | Séparation par `-short` | — |
| TST-08 | Basse | Noms de fichiers de test vagues | T26 |
| TST-09 | Info | `synctest` | — |
| TST-10 | Info | Angle mort e2e | — |
| MEM-01 | Basse | Garde mémoire non armée | T07 |
| MEM-02 | Info | Goroutines par pas | — |
| MEM-03 | Info | `-gcflags=-m` non documenté | T32 |
| CON-01 | Moyenne | Calibration sans signaux ni délai | T06 |
| CON-02 | Moyenne | Variables non synchronisées | T18 |
| CON-03 | Info | Lancement puis étranglement | — |
| CON-04 | Info | Goroutine du spinner | T21 |
| CFG-01 | Basse | `errors.Join` | T27 |
| CFG-02 | Basse | Variables lues hors table | T28 |
| CFG-03 | Info | Décodage permissif | — |
| ARC-01 | Moyenne | Calibration fait de la présentation | T16 |
| ARC-02 | Moyenne | Présentation dans `errors` | T15 |
| ARC-03 | Basse | Texte utilisateur dans le cas d'usage | T15/T16 |
| DEP-01 | Moyenne | Pas de `govulncheck` | T04 |
| DEP-02 | Basse | Dépendances discutables | T14, T21 |
| DEP-03 | Basse | `go get -u` en bloc | (Makefile, hors plan) |
| DEP-04 | Info | Deps de test | — |
| OBS-01 | Moyenne | Pas de diagnostic en production | T14 |
| OBS-02 | Basse | Profilage du binaire | T30 (opt.) |
| DOC-01 | Moyenne | Archéologie des commentaires | T23 |
| DOC-02 | Moyenne | Corpus bilingue | T24 |
| DOC-03 | Basse | Volume et structure | T25 |
| DOC-04 | Basse | Renvois périmés | T13 |
| DOC-05 | Info | Style godoc | — |
| SEC-01 | Moyenne | Vulnérabilités non analysées | T04 |
| SEC-02 | Basse | Image non épinglée | T05 |
