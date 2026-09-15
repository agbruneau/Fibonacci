# Évaluation académique — FibCalc (dépôt `agbruneau/Fibonacci`)

- **Objet** : dépôt `github.com/agbruneau/Fibonacci` (module `github.com/agbruneau/FibGo`),
  branche `main`, commit `dbac1a2` (`v4.1.1`, 2026-09-07), arbre propre.
- **Date de l'évaluation** : 2026-09-15.
- **Évaluateur** : Claude Fable 5.1, à la demande du mainteneur.
- **Hôte** : Windows 11, `go1.27.0 windows/amd64`, `CGO_ENABLED=1` (MinGW),
  `golangci-lint v2.13.2` et `govulncheck v1.7.0` lancés par `go run` depuis
  `scripts/tools.env`.
- **Gabarit** : [`gabarit-evaluation-academique.md`](gabarit-evaluation-academique.md).
- **Plan d'exécution** : [`plan-evaluation-2026-09-15.md`](plan-evaluation-2026-09-15.md)
  (tâches `EVAL-00` à `EVAL-25`, décisions D1–D8 à trancher par le mainteneur).
- **Régime de preuve** : `[E]` exécuté ici, `[L]` lu, `[D]` déduit.
- **Évaluation antérieure** : `EVALUATION.md` (2026-02-08, 98/100, retirée de
  l'arbre le 2026-05-21, commit `0f79a6c`). Elle n'a pas été reprise : elle ne
  cite aucune commande exécutée et ne distingue pas ce qui a été vérifié de ce
  qui a été lu.

---

## 1. Verdict

**Note globale : 82 / 100 — A-.**

| # | Critère | Poids | Score | Pondéré |
|---|---|---:|---:|---:|
| C1 | Problème, objectifs, positionnement | 5 | 80 | 4,00 |
| C2 | Fondements théoriques et algorithmiques | 10 | 82 | 8,20 |
| C3 | Conception et architecture | 15 | 85 | 12,75 |
| C4 | Implémentation | 15 | 86 | 12,90 |
| C5 | Vérification et validation | 15 | 90 | 13,50 |
| C6 | Performance et méthode expérimentale | 10 | 80 | 8,00 |
| C7 | Documentation et communication | 10 | 80 | 8,00 |
| C8 | Processus d'ingénierie et outillage | 10 | 82 | 8,20 |
| C9 | Intégrité, attribution, licences | 5 | 55 | 2,75 |
| C10 | Maintenabilité, dette, perspectives | 5 | 78 | 3,90 |
| | **Total** | **100** | | **82,2** |

**Recevabilité : recevable sous réserve.** R1, R2 et R4 sont satisfaites. R3 ne
l'est pas : `internal/bigfft` dérive de la bibliothèque `remyoudompheng/bigfft`
(BSD-3-Clause) sans qu'aucune notice, aucun nom d'auteur ni aucune URL amont ne
figure dans l'arbre (§ 4, C9 ; preuve en annexe A). La correction tient en un
fichier de licence et deux paragraphes ; elle est obligatoire avant tout dépôt
académique.

**Synthèse.** Le dépôt est un travail d'ingénierie logicielle nettement au-dessus
de ce qu'exige un projet universitaire sur la vérification (oracle indépendant,
propriétés, fuzzing, race detector, 96,1 % de couverture réexécutée), sur la
discipline de mesure (benchstat, comparaison A/A, candidats rejetés sur preuve) et
sur la traçabilité des décisions (douze ADR, index des identifiants d'audit). Ses
faiblesses sont d'un autre ordre : une attribution manquante, une bibliographie
absente, une complexité asymptotique mal énoncée à cinq endroits, un CHANGELOG
dont la préhistoire liste des fonctions qui n'existent pas, et une documentation
si tournée vers l'historique de ses propres audits qu'un lecteur neuf lit huit
campagnes avant l'algorithme. Le score est celui d'un travail solide dont la
partie « communication scientifique » est en retard sur la partie « ingénierie ».

---

## 2. Méthode

### 2.1 Lu `[L]`

`README.md`, `CONTRIBUTING.md`, `CHANGELOG.md` (têtes 4.1.1 / 4.1.0 / 4.0.0,
sections 1.0.0 et 0.1.0), `docs/ARCH.md`, `docs/TESTING.md`, `docs/PERFORMANCE.md`,
`docs/PORTABILITY.md`, `docs/CALIBRATION.md`, `docs/algorithms/{FAST_DOUBLING,
COMPARISON, FFT, BIGFFT}.md`, `docs/architecture/{dependency-graph,
validation/validation-report}.md`, `docs/audits/{HISTORY, INDEX,
audit-2026-09-livre}.md`, les douze ADR, `.golangci.yml`, `Dockerfile`, `Makefile`,
`scripts/{check.ps1, tools.env}`, `.github/workflows/ci.yml`, `.env.example`.
Code : `cmd/fibcalc`, `internal/fibonacci/{calculator, fastdoubling,
doubling_framework, strategy, fft, fft_based, matrix, matrix_ops, modular}.go`,
`internal/fibonacci/fibmath`, `internal/fibonacci/memory/budget.go`,
`internal/bigfft/{fft, fermat, fft_core, fft_recursion, doc}.go`,
`internal/orchestration/orchestrator.go`, `internal/app/{app, calculate}.go`,
`internal/calibration/{calibration, microbench}.go` (têtes), `internal/tui/model.go`
(tête), `internal/arch_test.go`, les tests golden et de propriétés,
`test/e2e/cli_e2e_test.go` (tête).

### 2.2 Exécuté `[E]`

| Commande | Résultat |
|---|---|
| `go build ./...`, `go vet ./...` | zéro erreur |
| `gofmt -l .`, `go mod tidy -diff`, `go mod verify` | zéro écart, modules vérifiés |
| `go test -race -shuffle=on -count=1 -coverprofile ./...` | 22 paquets verts, **96,1 %** des instructions (annexe B) |
| `go run golangci-lint@v2.13.2 run ./...` | **0 issue** |
| `go run govulncheck@v1.7.0 ./...` | aucune vulnérabilité |
| `fibcalc -n 100 -c -q` | `354224848179261915075` (conforme au README) |
| `fibcalc -n 100000000 -last-digits 10 -q -machine` | `7760546875` (conforme au README) |
| `fibcalc -n 10000000 -algo all -machine` | trois résultats cohérents ; fast 90 ms, fft 102 ms, matrix 459 ms |
| Décomptes : LOC, fonctions de test, `nolint`, `nosec`, `panic(`, `go:linkname`, langue des documents | annexe B |
| Comparaison de `internal/bigfft/{fft,fermat}.go` avec l'amont `remyoudompheng/bigfft` | annexe A |

### 2.3 Non vérifié

- Le backend `-tags gmp` (pas d'en-têtes libgmp sur l'hôte ; couvert par le job CI `gmp`, non rejoué ici).
- La construction de l'image Docker et le job `cross-build` (pas de démon Docker ; CI non rejouée).
- Le comportement 32 bits à l'exécution (la doc dit elle-même : compile, non testé).
- Le fuzzing par mutation (seule la relecture des graines tourne dans `go test`).
- La TUI interactive.
- La baseline de débit `docs/audits/bench-baseline.txt` (linux/amd64) : je n'ai pas
  de machine Linux comparable ; la seule mesure de cette évaluation est le
  relevé mono-exécution du tableau ci-dessus, qui reproduit l'**ordre** de la
  baseline, pas ses valeurs.
- L'exactitude de chaque figure des 1 400 lignes de `FFT.md` / `BIGFFT.md` : les
  sections lues sont cohérentes avec le code ; le reste est pris tel quel.

---

## 3. Portrait chiffré `[E]`

| Mesure | Valeur |
|---|---|
| Paquets Go | 22 |
| Fichiers Go suivis / fichiers de test | 268 / 142 |
| Lignes Go : production / tests / total | 17 738 / 31 169 / 48 907 (ratio tests ÷ production : 1,76) |
| Production par composant | `bigfft` 3 553 · `fibonacci` (+ `memory`, `fibmath`) 4 111 · `calibration` 1 993 · `tui` 1 939 · `config` 1 052 · `cli` + `completion` 1 468 · `app` 766 · `orchestration` 641 · `threshold` 646 · reste 1 569 |
| Fonctions `Test` / `Fuzz` / `Benchmark` / `Example` | 1 672 / 14 / 102 / 10 |
| Couverture réexécutée | 96,1 % (plancher gardé : 80 %) ; toutes les bibliothèques ≥ 85,9 % sauf `testutil` (20 %) |
| `TODO`/`FIXME` · `//nolint` · `#nosec` · `panic(` en production | 0 · 5 · 19 · 18 |
| Lignes de commentaire en production | 9 855 (≈ 35 % des lignes non-test, chiffre repris de l'audit 2026-09-07) |
| Documents Markdown suivis / lignes | 46 / 13 787 (dont `docs/` 11 712, `CHANGELOG.md` ≈ 1 200) |
| ADR | 12 (0001–0012) + gabarit |
| Dépendances directes / modules du graphe | 8 / 198 |
| Commits (2026-01-07 → 2026-09-07) | 711 ; 517 portent un `Co-Authored-By: Claude …` ; 693 signés par le mainteneur |
| Préhistoire | le code Go entre en un commit (`5bfc50e`, 2026-01-23, « added FibGo », 238 fichiers) dans un dépôt qui contenait auparavant un projet Python ; les entrées CHANGELOG 0.1.0 (2025-11) et 1.0.0 (2025-12) n'ont donc pas d'historique git ici |

---

## 4. Évaluation par critère

### C1 — Problème, objectifs, positionnement — 80 / 100 (Solide)

**Constats forts.**
- Le cadrage est explicite et honnête : le README se présente comme un
  « laboratoire d'expérimentation algorithmique et d'ingénierie logicielle », dit
  que « le nombre de Fibonacci n'est pas la finalité ; c'est le banc d'essai », et
  porte le badge *Prototype académique* `[L]`.
- Les objectifs sont mesurables et la règle de preuve est énoncée dès le README :
  « toute affirmation chiffrée doit venir d'un artefact de mesure du dépôt ; ce qui
  n'a pas été réexécuté est signalé comme tel » `[L]`.
- Le choix du problème est justifié (réponse exactement vérifiable, plusieurs
  algorithmes comparables, coût arbitrairement croissant).

**Réserves.**
- Aucun positionnement par rapport à l'existant : ni `mpz_fib_ui` de GMP, ni
  PARI/GP, ni une implémentation Go antérieure ne servent de point de comparaison
  externe `[L]`. Le backend GMP, qui aurait pu jouer ce rôle, n'est pas atteignable
  depuis le binaire même avec le tag de build (`COMPARISON.md`, `ARCH.md` § 5) `[L]`.
- Pas de question de recherche ni d'hypothèses formulées avant les mesures ; les
  hypothèses apparaissent après coup (« l'idée que `fft` devienne compétitif
  au-delà est une hypothèse que le dépôt ne teste pas ») `[L]`.
- Aucune section « travaux connexes ».

### C2 — Fondements théoriques et algorithmiques — 82 / 100 (Solide)

**Constats forts.**
- Les identités de doublement sont dérivées correctement par élévation au carré
  de la matrice Q, avec la substitution F(k−1) = F(k+1) − F(k) ; la preuve de
  `FAST_DOUBLING.md` et le commentaire de `fastdoubling.go` concordent avec le
  code (`T3 = F(k)·F(k+1)`, `F(2k) = 2·T3 − F(k)²`, `F(2k+1) = F(k+1)² + F(k)²`) `[L]`.
- L'analyse de complexité est juste **et** nuancée : O(log n) multiplications de
  coût M(n), et le README dit explicitement qu'« aucune ligne du tableau ne domine
  les autres asymptotiquement », ce qui est exact et rarement dit `[L]`.
- Strassen-Winograd à 7 multiplications et 15 additions/soustractions
  (`matrix_ops.go`) : les S1–S8, P1–P7 et l'assemblage correspondent à la variante
  de Winograd `[L]`. La mise au carré symétrique (3 carrés + 1 produit) est
  correcte et son invariant (la puissance de Q est symétrique) est nommé.
- Le mode `--last-digits` s'appuie sur `big.Int.Mod` euclidien et le commente `[L]`.
- La documentation de la FFT sur anneaux de Fermat est substantielle
  (`FFT.md` 540 lignes, `BIGFFT.md` 879) et va jusqu'à calculer l'écart entre la
  règle « K ≈ 2√N » citée en commentaire et la table `fftSizeThreshold` réellement
  utilisée (rapport 0,26 à 1,04 selon N) — un exercice de lucidité peu commun `[L]`.

**Réserves.**
- **Complexité de Schönhage-Strassen mal énoncée, cinq fois.** README (ligne 100),
  `FFT.md` (lignes 3, 8, 471, 481), `BIGFFT.md` (ligne 4) et `fastdoubling.go`
  (ligne 73) donnent M(n) ≈ O(n log n). L'algorithme implémenté (FFT modulo
  2^n+1, récursion de Fermat) est Schönhage-Strassen, dont la borne est
  O(n log n log log n). O(n log n) est le résultat de Harvey et van der Hoeven
  (2021), qui n'est pas ce que fait `internal/bigfft`. Aucune occurrence de
  « log log » dans l'arbre `[E]`. L'évaluation de février 2026 donnait la bonne
  borne ; elle s'est perdue.
- **Aucune bibliographie.** Ni Schönhage et Strassen (1971), ni Winograd, ni la
  littérature du *fast doubling*, ni la bibliothèque amont ne sont cités ; le
  README invoque « la littérature classique » sans une référence `[E]`.
- Les trois seuils (parallélisme 4 096 bits, FFT 500 000, Strassen 3 072) sont
  des constantes sans justification théorique ni mesure ; le dépôt le dit
  honnêtement (« une table de constantes, pas une mesure »), mais un travail
  académique attend au moins l'argument d'ordre de grandeur qui les place là `[L]`.

### C3 — Conception et architecture — 85 / 100 (Solide)

**Constats forts.**
- Hiérarchie `cmd → app → orchestration → fibonacci → bigfft` gardée par un test
  d'architecture à sept règles qui interroge `go list` `[L: arch_test.go]`.
- Le graphe de dépendances documenté (48 arêtes) est présenté comme vérifié
  arête par arête contre `go list` le 2026-09-07, avec le pipeline shell qui
  permet de le refaire `[L: validation-report.md]`. Je n'ai pas rejoué ce
  pipeline ; la règle « `bigfft` n'importe aucun paquet interne » est cohérente
  avec les imports lus dans `fft.go`, `fermat.go`, `fft_core.go`,
  `fft_recursion.go` `[L]`.
- Interfaces définies côté consommateur (`app.CalculatorRegistry`,
  `orchestration.CalculatorSource`) ; port `calibration.Reporter` pour sortir la
  présentation de la couche application ; `apperrors.ExitCodeFor` pur, texte
  côté `cli` `[L]`.
- Le décorateur `FibCalculator` concentre les préoccupations transverses (chemin
  rapide n ≤ 93, contrôle GC, cache FFT, préchauffage) ; les cœurs d'algorithme
  n'en savent rien `[L: calculator.go]`.
- La discipline mémoire est argumentée à la ligne : l'invariant « aucun alias de
  `big.Int` ne survit à son arène » est énoncé, l'ordre `checkLimit →
  clearStateAliases → put` justifié, et un test de régression nommé
  (`TestReleaseState_OverLimit_AliasesCleared`) `[L: fastdoubling.go]`.

**Réserves.**
- **Pile d'abstractions lourde pour trois calculateurs qui partagent une boucle.**
  `DoublingFramework` + `DoublingStepExecutor` + `Multiplier` + `AdaptiveStrategy`
  / `FFTOnlyStrategy` + `CacheStrategy` + décorateur + fabrique/registre, pour
  deux stratégies dont l'une (`fft`) est, de l'aveu du README, « pas un troisième
  algorithme » mais la même boucle avec le test de seuil retiré — exposée
  pourtant à l'utilisateur comme `-algo fft` `[L]`.
- **État mutable de paquet dans le noyau.** `bigfft` garde en variables globales
  le seuil FFT (`atomic.Int64`), les deux seuils de récursion parallèle, le
  sémaphore (`sync.Once`), les pools et la cache de transformées ;
  `fibonacci` garde `defaultStrassenThresholdBits`. La migration vers un
  `FFTContext` par appel a été abandonnée (ADR-0004 §B1, WONT-FIX) et ADR-0006
  reporte encore vers ce type supprimé `[L]`.
- **Deux sémaphores indépendants** (`GOMAXPROCS(0)` côté `fibonacci`, `NumCPU()`
  côté `bigfft`) : `ARCH.md` reconnaît « jusqu'à GOMAXPROCS + NumCPU goroutines
  simultanées », soit un sur-abonnement potentiel de 2× `[L]`.
- Imports latéraux `config → fibonacci/memory` et `config → ui` tolérés et
  « gelés » par une règle plutôt que résolus `[L: arch_test.go]`.
- Le calculateur GMP est enregistré dans une fabrique privée que `app.New`
  ne consulte pas : fonctionnalité à moitié câblée, documentée comme telle `[L]`.

### C4 — Implémentation — 86 / 100 (Solide)

**Constats forts.**
- Outillage à zéro sur l'hôte : `gofmt`, `go vet` (tous les analyseurs sauf
  `fieldalignment`), `golangci-lint` v2 avec 21 linters dont `gosec`, `gocritic`
  (tous les tags), `revive`, `gocognit`, `errcheck` ; `govulncheck` propre `[E]`.
- `-race` vert sur 22 paquets avec ordre mélangé et cache désactivée `[E]`.
- Erreurs enveloppées avec `%w` et typées (`ConfigError`, `CalculationError`,
  `MemoryError`) ; classification annulation/expiration par `errors.Is` ; codes
  de sortie 0/1/2/3/4/130 documentés et testés `[L]`.
- Contexte vérifié entre les trois multiplications d'un pas et entre les
  itérations ; `Validate` rapporte toutes les erreurs par `errors.Join` `[L]`.
- Politique de `panic` explicite (ADR-0002) : pré-conditions converties en
  erreurs, post-conditions re-propagées par sentinelle, un seul point
  d'implémentation (`fermatPanicToError`) `[L]`.
- Contrôle GC par compteur de références sous mutex, `panic`-safe (ADR-0005) `[L]`.
- Zéro `TODO`/`FIXME` `[E]`.

**Réserves.**
- **`go:linkname` vers six routines internes de `math/big`** (`addVV`, `subVV`,
  `addVW`, `subVW`, `shlVU`, `addMulVVW`), sans repli Go pur `[E: grep ; L: arith_decl.go]`.
  Cela compile sous go1.27 parce que la chaîne Go tolère encore ces symboles ;
  c'est une dépendance à une tolérance, pas à une API. `PORTABILITY.md` le
  documente sans proposer de repli.
- **Densité de commentaires ≈ 35 %**, en grande partie narrative : « used to
  be », « audit X », « the previous “20M” was computed against… ». Le code
  raconte son histoire plus qu'il n'énonce son intention ; `INDEX.md` rend les
  ~350 identifiants résolubles, ce qui corrige la traçabilité, pas la
  lisibilité `[L]`.
- 18 sites `panic(` en production `[E]` — pour l'essentiel des invariants
  internes, ce qui est défendable, mais le mécanisme « panic puis recover puis
  erreur » des quatre points d'entrée `bigfft` reste un compromis que l'ADR
  lui-même qualifie ainsi.
- Surface exportée sans appelant de production : `SetFFTThreshold`,
  `SetFFTParallelismConfig`, `GetFFTParallelismConfig`, `SetDefaultStrassenThreshold`
  — commentées « test-only in practice » et conservées `[L]`.
- 5 `//nolint`, 19 `#nosec`, trois fonctions exemptées de `gocyclo` par nom `[E]`.

### C5 — Vérification et validation — 90 / 100 (Exemplaire)

**Constats forts** (tous `[E]` sauf mention).
- **Oracle indépendant** : `cmd/generate-golden` n'importe aucun paquet interne et
  produit 26 entrées jusqu'à F(200 000) ; le test épingle la taille du corpus et
  la présence des trois grandes entrées, et aucun drapeau `-update` n'existe.
- **Tests de propriétés** (gopter, 100 cas par propriété et par calculateur) :
  Cassini, récurrence, identité de doublement sur les trois calculateurs ;
  GCD(F(m), F(n)) = F(GCD(m, n)) sur `fast` `[L]`.
- **Fuzzing** : sept cibles, identités de d'Ocagne et d'addition, validation
  croisée `bigfft` contre `math/big` avec des graines placées de part et d'autre
  du seuil ; relecture des graines à chaque `go test`, mutation hebdomadaire en
  CI `[L]`.
- **Vérification N-versions à l'exécution** : `-algo all` compare les trois
  résultats et sort en code 3 sur divergence ; vérifié sur F(10 000 000).
- `-race -shuffle=on -count=1` verts ; couverture 96,1 % au total, ≥ 85,9 % dans
  chaque bibliothèque hors `testutil`.
- Test d'architecture, tests de contrat de `panic`, tests e2e sur le binaire
  construit, tests de golden de sortie CLI, spies d'orchestration `[L]`.
- Un test *flaky* documenté a été corrigé à la cause (CHANGELOG 4.1.0) plutôt
  que rejoué `[L]`.

**Réserves.**
- **Plancher 80 % contre 96,1 % mesuré** : 16 points de mou non gardés, admis
  dans `TESTING.md` `[L]`.
- **Les propriétés ne franchissent pas le palier FFT sur `fast` et `matrix`.**
  Elles plafonnent n à 25 000 avec `FFTThreshold: 20000` ; or F(25 000) ≈ 17 400
  bits (25 000 × 0,694) `[E: arithmétique]`. Le palier FFT de ces deux calculateurs
  est couvert par des tests dédiés à seuil abaissé (`matrix_fft_path_test.go`,
  seuil 64 bits ; `fft_crossval_test.go`) et par le calculateur `fft` ; la
  bascule au seuil **par défaut** (n ≈ 1,38 M) n'est exercée que par les
  benchmarks et la vérification croisée à l'exécution `[L]`.
- Corpus de fuzz : 12 des 15 fichiers de graines répètent une valeur déjà dans
  `f.Add` ; aucun *crasher* n'a jamais été trouvé — le fuzzing par mutation n'a
  encore aucune preuve de valeur `[L: TESTING.md]`.
- 31 169 lignes de tests pour 17 738 de production : c'est une force pour la
  confiance et un coût pour toute refonte ; plusieurs fichiers de test portent
  encore des noms d'audit (`fft_race_test.go`, `state_pool_arena_test.go`) `[E]`.

### C6 — Performance et méthode expérimentale — 80 / 100 (Solide)

**Constats forts.**
- Protocole écrit et suivi : `benchstat`, `-count` de 5 à 10, comparaison A/A à
  code identique pour calibrer le bruit, protocole en double ordre pour le
  balayage du multiplicateur d'arène (ADR-0009 R4), seuil de régression 5 % `[L]`.
- **Candidats rejetés sur mesure et consignés** : FIB-05 (+18 à +34 % sur Ryzen),
  plafond de cache ×4 (+22 % sur `MatrixExp/10M`), gain DTM non reproduit à
  `-count=8` — la pratique de garder la mesure qui rejette est ce qu'un
  évaluateur demande le plus souvent et obtient le moins `[L: ADR-0001, 0009, 0010]`.
- Baseline archivée (`bench-baseline.txt`) et cinq A/B ciblés `[L]`.
- Empreinte mémoire mesurée **par processus** (delta `MemStats.Sys`) et
  l'estimateur `--memory-limit` refondu après une sous-estimation de 5 à 12×,
  désormais borne haute ≤ 2,47× `[L: budget.go, README]`.
- Ma mesure sur cet hôte reproduit l'ordre de la baseline (fast < fft < matrix) `[E]`.

**Réserves.**
- **Deux tailles chronométrées seulement** (F(1M), F(10M)). Aucune courbe
  d'échelle ; la promesse « plusieurs centaines de millions » du README repose sur
  un seul chiffre (0,204 s à F(100M)) sans artefact `[L]`.
- **`PERFORMANCE.md` conserve deux tableaux faux de 20× à 88×** sous des encadrés
  d'avertissement, plutôt que de les retirer. Un lecteur qui lit le tableau et
  pas l'encadré emporte la mauvaise grandeur `[L]`.
- La baseline vient d'un seul hôte linux/amd64, sans estampille de version Go ;
  aucune reproduction sur un second hôte n'est archivée `[L]`.
- Le gestionnaire de seuils dynamiques a été mesuré neutre et reste dans l'arbre
  (646 lignes de production, 1 491 avec tests) derrière un drapeau `[L: ADR-0001 ; E: LOC]`.
- Le calculateur `fft` est plus lent que `fast` aux deux tailles ; il n'existe
  pas de mesure au-delà pour trancher l'hypothèse de croisement `[L]`.
- Le seuil de régression de 5 % est une discipline locale, non un gate CI `[L]`.

### C7 — Documentation et communication — 80 / 100 (Solide)

**Constats forts.**
- Volume et traçabilité hors normes : 13 787 lignes de Markdown, douze ADR avec
  candidats rejetés, diagrammes C4 présentés comme vérifiés contre `go list`,
  `INDEX.md` qui rend résolubles les identifiants d'audit, `CHANGELOG` au format
  Keep a Changelog `[L]`.
- Chaque chiffre du README est daté, sourcé et, quand il n'a pas été rejoué, dit
  comme tel ; deux commandes du démarrage rapide reproduisent leur sortie
  annotée `[E]`.
- Les limites sont déclarées en tête de README (gmp non compilable ici, 32 bits
  non testé) `[L]`.

**Réserves.**
- **Règle de langue énoncée, non appliquée.** `CONTRIBUTING.md` § 4 : la
  documentation narrative — README, CHANGELOG, ADR, `docs/*.md` — est en français.
  Or `TESTING.md`, `PERFORMANCE.md`, `CALIBRATION.md`, `BUILD.md`, `TUI_GUIDE.md`
  et les sept `docs/algorithms/*.md` sont en anglais, et `ARCH.md` alterne
  (≈ 207 lignes à marqueurs français, 256 à marqueurs anglais) `[E: décompte]`.
- **Contradictions internes.** `docs/audits/HISTORY.md`, paragraphe « Limites
  déclarées », affirme que `GOARCH=386 go build ./...` « échoue toujours » et que
  `TestStateBump_PinnedAcrossCachedCalls` est *flaky* ; la ligne 2026-09-07 du
  même tableau, `CHANGELOG.md` 4.1.0 et `PORTABILITY.md` § 1 disent les deux
  corrigés le 2026-09-07 `[L]`.
- **`CHANGELOG.md` 1.0.0 (2025-12-22) liste des fonctions absentes de l'arbre** :
  « JSON output format support », « Hexadecimal result display option »,
  « Security policy with vulnerability disclosure process », « Rate limiting
  protection against DoS » — pour un calculateur en ligne de commande sans
  réseau. Cette préhistoire n'a pas d'historique git ici (le code arrive le
  2026-01-23). Un journal qui affirme des fonctions inexistantes affaiblit la
  crédibilité du reste, qui est pourtant méticuleux `[L ; E: grep `JSON`/`hex` sans
  résultat dans la table des drapeaux]`.
- **Documentation d'archéologie.** Le lecteur neuf rencontre huit campagnes
  d'audit, ~350 identifiants et 400 lignes de README à mises en garde avant
  l'algorithme. La règle « le commentaire doit rester compréhensible si on efface
  l'identifiant » est écrite mais pas encore appliquée à l'existant `[L]`.
- Pas de bibliographie (C2) ; complexité SS mal énoncée (C2).

### C8 — Processus d'ingénierie et outillage — 82 / 100 (Solide)

**Constats forts.**
- Gate local en deux implémentations (`check.sh`, `check.ps1`) à sept étapes,
  lint et `govulncheck` bloquants ; outils épinglés dans `scripts/tools.env` et
  lancés par `go run pkg@version`, ce qui ferme la classe de panne « binaire
  compilé pour une ancienne chaîne Go » qui avait rendu le lint inerte `[L ; E:
  les deux outils ont tourné par ce mécanisme]`.
- CI à cinq jobs : gate Ubuntu + Windows (`-race -shuffle -count=1`, lint,
  `govulncheck`, plancher, `tidy -diff`), `gmp` avec libgmp, `cross-build`
  386/arm64, image Docker avec assertion de version et fumée, fuzz hebdomadaire `[L]`.
- Image Docker : multi-étapes, `distroless`, `nonroot`, bases épinglées par
  digest d'index multi-arch avec provenance consignée, rapport de dérive non
  bloquant `[L]`.
- `go.mod` sans directive `toolchain`, `go mod tidy -diff` et `go mod verify`
  propres `[E]` ; 8 dépendances directes.
- Commits conventionnels (43 des 60 derniers selon l'audit du 2026-09-07),
  étiquettes SemVer `v3.0.0`, `v4.0.0`, `v4.1.0`, `v4.1.1` `[E]`.

**Réserves.**
- **La CI n'existe que depuis le 2026-09-07**, après huit mois de développement
  et **deux défaillances silencieuses** du gate local (lint qui écrivait
  `Overall: PASS` sans tourner ; `govulncheck`, `gosec`, `staticcheck` incapables
  de démarrer). La décision « pas de CI » (ADR-0004 §B3, ADR-0010 D4) a été
  prise deux fois avant d'être renversée par ADR-0012 `[L]`.
- **Pas de revue humaine indépendante** : un seul mainteneur, PR auto-fusionnées,
  517 commits sur 711 co-signés par des modèles d'IA ; les « vérifications
  adversariales » mentionnées dans `HISTORY.md` sont des panels d'agents, pas des
  pairs `[E: shortlog ; L]`.
- Journal de versions lacunaire : `v2.x` et `v3.0.0` n'ont jamais eu d'entrée
  (note en tête de 4.0.0) ; quinze étiquettes `rewrite/*` encombrent l'espace de
  noms `[E: git tag]`.
- Un *worktree* périmé (`.claude/worktrees/lucid-nightingale-3e810f`, branche
  `claude/determined-carson-0df1c3`) est présent dans le répertoire de travail,
  exclu par `.git/info/exclude` ; `TESTING.md` mentionne le risque de double
  décompte `[E]`.
- Le plancher de couverture n'a pas été relevé depuis la campagne de mai 2026.

### C9 — Intégrité, attribution, licences — 55 / 100 (Insuffisant)

**Constats forts.**
- **L'aide de l'IA est divulguée** : le README nomme les modèles et les
  campagnes ; 517 commits portent une ligne `Co-Authored-By` `[E]`. C'est
  au-dessus de la pratique courante.
- `arith_decl.go` conserve l'en-tête « Copyright 2010 The Go Authors … BSD-style »
  pour les déclarations `go:linkname` `[E]`.
- Licence Apache 2.0 à la racine `[E]`.

**Réserves — bloquante.**
- **`internal/bigfft` est un dérivé non attribué de
  `github.com/remyoudompheng/bigfft`.** Comparaison avec l'amont (annexe A) `[E]` :
  la table `fftSizeThreshold` (16 valeurs) est identique ; la constante 1 800 mots
  et son commentaire « TestCalibrate seems to indicate a threshold of 60kbits on
  32-bit arches and 110kbits on 64-bit arches » sont identiques ; `ShiftHalf`,
  `norm`, le début de `Shift` et le commentaire « copied from math/big » de
  `basicMul` sont identiques ; l'en-tête « based on the Schönhage-Strassen method
  using integer FFT modulo 2^n+1 » est celui du paquet amont.
- Le nom de l'auteur et l'URL amont n'apparaissent **nulle part** dans l'arbre
  (`grep -ri oudompheng` : zéro résultat) `[E]`. La seule trace est ADR-0009 R1 :
  « API héritée de bigfft upstream jamais branchée », sans nommer l'amont `[L]`.
- La licence amont est BSD-3-Clause (« Copyright (c) 2012 The Go Authors »)
  `[E: LICENSE amont]`. Sa première clause exige la conservation de la notice dans
  toute redistribution du code source. Relicencier un dérivé sous Apache 2.0 est
  permis ; supprimer la notice ne l'est pas.
- Les « Remerciements » du README créditent « la littérature classique » et les
  outils (Go, Bubble Tea, benchstat, golangci-lint, gosec), pas la bibliothèque
  dont descend le noyau `[L]`.
- Lecture : la mention « upstream » dans un ADR indique un oubli, pas une
  dissimulation. Dans un cadre académique, la distinction entre réutilisation et
  appropriation tient précisément à la notice ; la correction est petite et
  obligatoire (§ 6, R-1).
- Absence de références bibliographiques (C2).

### C10 — Maintenabilité, dette, perspectives — 78 / 100 (Adéquat)

**Constats forts.**
- La dette est consignée avec ses clauses de réouverture (« À revoir si : … »)
  dans les ADR ; la passe de sur-ingénierie du 2026-09-03 a retiré ~25 éléments
  et l'audit de juillet ~500 lignes de code mort `[L]`.
- 0 `TODO`, 8 dépendances directes, graphe de 198 modules, deux dépendances
  inertes retirées en septembre `[E ; L]`.

**Réserves.**
- **Surface périphérique.** Noyau et algorithmes (`bigfft` + `fibonacci` +
  `memory` + `fibmath`) font 7 664 lignes de production sur 17 738 (43 %).
  Calibration (1 993), TUI (1 939) et seuils dynamiques (646) en font 4 578 (26 %),
  dont une composante mesurée neutre `[E]`. Pour un banc d'essai, c'est une
  masse à maintenir qui ne produit pas de résultat.
- **Facteur bus égal à un** ; la mémoire du projet est dans les ADR et le
  CHANGELOG, ce qui atténue sans résoudre.
- **Fragilités connues et acceptées** : `go:linkname` sans repli ; 32 bits
  compilé mais non testé ; `gmp` non compilable sur l'hôte de référence ;
  ADR-0006 reporte vers un type supprimé ; pile Bubble Tea = une vingtaine de
  modules indirects pour la seule TUI `[L ; E: go.mod]`.
- Aucune feuille de route au-delà des clauses « à revoir si ».

---

## 5. Constats prioritaires transversaux

| # | Sévérité | Constat | Critères | Preuve |
|---|---|---|---|---|
| P1 | **Bloquant** | Dérivé de `remyoudompheng/bigfft` (BSD-3) sans notice ni attribution | C9, R3 | annexe A `[E]` |
| P2 | Haute | Complexité de Schönhage-Strassen donnée comme O(n log n) à cinq endroits ; la borne est O(n log n log log n) | C2, C7 | grep `[E]` |
| P3 | Haute | `CHANGELOG.md` 1.0.0 liste quatre fonctions inexistantes (JSON, hexadécimal, politique de sécurité, limitation de débit) | C7 | lecture `[L]`, grep `[E]` |
| P4 | Moyenne | `HISTORY.md` contredit `CHANGELOG` 4.1.0 et `PORTABILITY.md` sur 386 et sur le test *flaky* | C7 | lecture `[L]` |
| P5 | Moyenne | Règle de langue (docs en français) non appliquée à 13 documents ; `ARCH.md` mixte | C7 | décompte `[E]` |
| P6 | Moyenne | `PERFORMANCE.md` conserve des tableaux faux de 20× à 88× | C6, C7 | lecture `[L]` |
| P7 | Moyenne | Gestionnaire de seuils dynamiques conservé après mesure neutre | C6, C10 | ADR-0001 `[L]`, LOC `[E]` |
| P8 | Moyenne | Aucune bibliographie ; aucun point de comparaison externe | C1, C2 | grep `[E]` |
| P9 | Basse | Propriétés bornées sous le palier FFT de `fast`/`matrix` ; plancher de couverture 16 points sous la mesure | C5 | arithmétique `[E]` |
| P10 | Basse | Étiquettes `rewrite/*` (15), *worktree* périmé dans l'arbre, entrées CHANGELOG `v2`/`v3` absentes | C8 | git `[E]` |
| P11 | Info | `go:linkname` vers `math/big` sans repli ; deux sémaphores non coordonnés ; `-algo gmp` inatteignable | C3, C4, C10 | lecture `[L]` |

---

## 6. Recommandations

Ordonnées par rapport valeur/effort. S : moins de deux heures ; M : une
demi-journée ; L : plusieurs jours.

1. **R-1 (S, obligatoire)** — Ajouter `internal/bigfft/LICENSE` avec la notice
   BSD-3 amont, un en-tête « Derived from github.com/remyoudompheng/bigfft
   (BSD-3-Clause); modifications © 2026 André-Guy Bruneau, Apache-2.0 » dans les
   fichiers concernés, un `NOTICE` à la racine, et une ligne dans les
   « Remerciements » du README. Corrige P1.
2. **R-2 (S)** — Remplacer O(n log n) par O(n log n log log n) aux six
   emplacements et ajouter une phrase qui distingue Schönhage-Strassen de
   Harvey–van der Hoeven. Corrige P2.
3. **R-3 (S)** — Réécrire l'entrée 1.0.0 du CHANGELOG en ne gardant que ce qui
   est attesté (ou la marquer « préhistoire, non vérifiable dans ce dépôt »),
   et corriger le paragraphe « Limites » de `HISTORY.md`. Corrige P3, P4.
4. **R-4 (M)** — Ajouter une section « Références » (Schönhage & Strassen 1971 ;
   Winograd ; une source pour le *fast doubling* ; la bibliothèque amont ;
   Harvey & van der Hoeven 2021) et un court « Travaux connexes » avec une
   mesure contre `mpz_fib_ui`. Corrige P8 et relève C1/C2.
5. **R-5 (M)** — Retirer les deux tableaux invalidés de `PERFORMANCE.md` et
   produire une courbe d'échelle (n ∈ {10⁵, 10⁶, 10⁷, 10⁸}, `-count=10`,
   `benchstat`, artefact archivé avec version Go et hôte). Corrige P6 et
   fonde la promesse « centaines de millions ».
6. **R-6 (M)** — Trancher le sort du gestionnaire de seuils dynamiques : soit
   une mesure qui le justifie, soit sa suppression (le CHANGELOG et ADR-0001
   gardent l'histoire). Corrige P7.
7. **R-7 (L)** — Appliquer la règle de langue : traduire ou déplacer les treize
   documents anglais, ou amender la règle pour dire ce qui est vraiment
   pratiqué. Corrige P5.
8. **R-8 (S)** — Relever le plancher de couverture à 92 % ; étendre une
   propriété au-delà du seuil FFT par défaut (un cas à n ≈ 1,5 M sur `fast`,
   marqué `-short`). Corrige P9.
9. **R-9 (S)** — Supprimer les étiquettes `rewrite/*` (ou les documenter) et le
   *worktree* périmé. Corrige P10.
10. **R-10 (L, perspective)** — Un repli Go pur derrière `go:linkname` protégé
    par un test de build, et une passe qui retire des commentaires la narration
    d'audit devenue redondante avec `INDEX.md`.

---

## Annexe A — Filiation de `internal/bigfft` `[E]`

Source amont consultée le 2026-09-15 :
`https://raw.githubusercontent.com/remyoudompheng/bigfft/master/{fft.go,fermat.go,LICENSE}`.

| Élément | Amont | Dépôt évalué |
|---|---|---|
| En-tête de paquet | « The implementation is based on the Schönhage-Strassen method using integer FFT modulo 2^n+1. » | identique, `fft.go` lignes 1–2 |
| Seuil FFT | `var fftThreshold = 1800` ; commentaire « TestCalibrate seems to indicate a threshold of 60kbits on 32-bit arches and 110kbits on 64-bit arches. » | `defaultFFTThresholdWords = 1800`, commentaire identique ; stockage `atomic.Int64` (ADR-0003) |
| `fftSizeThreshold` | `{0, 0, 0, 4<<10, 8<<10, 16<<10, 32<<10, 64<<10, 1<<18, 1<<20, 3<<20, 8<<20, 30<<20, 100<<20, 300<<20, 600<<20}` | identique, 16 valeurs |
| Commentaire de dimensionnement | « A FFT size of K=1<<k is adequate when K is about 2*sqrt(N) where N = x.Bitlen() + y.Bitlen(). » | identique |
| `fermat.norm`, `fermat.Shift` (15 premières lignes), `fermat.ShiftHalf` | — | identiques au caractère près (`ShiftHalf` : `a := u + (3*_W/4)*n`, `b := u + (_W/4)*n`) |
| `basicMul` | « // copied from math/big » | commentaire identique ; corps modernisé (`clear(z)`) |
| Licence amont | BSD-3-Clause, « Copyright (c) 2012 The Go Authors. All rights reserved. » | aucune notice dans `internal/bigfft` ; `arith_decl.go` porte l'en-tête Go Authors 2010 pour les seules déclarations `linkname` |
| Occurrences de « oudompheng » / URL amont dans l'arbre | — | **0** (`grep -ri` sur `docs`, `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `internal`, `cmd`) |
| Mention indirecte | — | ADR-0009 R1 : « API héritée de bigfft upstream jamais branchée » |

Les apports propres au dépôt sont réels et substantiels (allocateur bump, pools par
classe de taille, cache LRU de transformées, récursion parallèle avec sémaphore,
politique de `panic`, wrappers `*Safe`, `MulTo`/`SqrTo`, tests, 879 lignes de
documentation). Ils ne dispensent pas de la notice.

## Annexe B — Sorties de commandes `[E]`

```text
$ go test -race -shuffle=on -count=1 -coverprofile cov.out ./...
ok  cmd/fibcalc                       3.433s  coverage: 90.0%
ok  cmd/generate-golden               6.500s  coverage: 87.9%
ok  internal                          1.953s  [no statements]
ok  internal/app                      1.425s  coverage: 92.4%
ok  internal/apperrors                1.274s  coverage: 100.0%
ok  internal/bigfft                  11.318s  coverage: 96.8%
ok  internal/calibration              7.392s  coverage: 93.8%
ok  internal/cli                      1.328s  coverage: 85.9%
ok  internal/cli/completion           1.302s  coverage: 98.7%
ok  internal/config                   1.331s  coverage: 96.1%
ok  internal/fibonacci                2.778s  coverage: 94.8%
ok  internal/fibonacci/fibmath        1.261s  coverage: 100.0%
ok  internal/fibonacci/memory         1.358s  coverage: 99.3%
ok  internal/fibonacci/threshold      1.300s  coverage: 98.3%
ok  internal/format                   1.259s  coverage: 100.0%
ok  internal/metrics                  1.248s  coverage: 98.7%
ok  internal/orchestration            1.328s  coverage: 99.4%
ok  internal/progress                 1.372s  coverage: 94.4%
ok  internal/testutil                 1.258s  coverage: 20.0%
ok  internal/tui                     19.925s  coverage: 99.3%
ok  internal/ui                       1.300s  coverage: 100.0%
ok  test/e2e                          3.726s  [no statements]
total:                                        (statements)  96.1%

$ go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./...
0 issues.

$ go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
No vulnerabilities found.

$ gofmt -l . ; go mod tidy -diff ; go mod verify
(vide) ; (vide) ; all modules verified

$ NO_COLOR=1 fibcalc -n 10000000 -algo all -machine
Fast Doubling (O(log n), Parallel, Zero-Alloc)            90ms   Success
FFT-Based Doubling                                       102ms   Success
Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)   459ms   Success
Global Status: Success. All valid results are consistent.
```

Décomptes (fichiers suivis, hors `.claude/`) : `grep -rh '^func Test'` 1 672 ;
`^func Fuzz` 14 ; `^func Benchmark` 102 ; `^func Example` 10 ; `nolint` 5 ;
`nosec` 19 ; `panic(` hors tests 18 ; `TODO|FIXME|HACK|XXX` 0 ; `go list -m all`
198 modules ; `git ls-files '*_test.go'` 142 ; `git ls-files '*.md'` 46 fichiers,
13 787 lignes ; `git shortlog -sn --all` : 693 André-Guy Bruneau, 13
google-labs-jules[bot], 11 agbruneau, 1 Andre-Guy Bruneau ; trailers
`Co-Authored-By: Claude …` sur 517 commits.
