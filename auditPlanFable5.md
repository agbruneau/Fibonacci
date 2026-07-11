# auditPlanFable5 — Plan d'exécution de l'audit auditFable5 (2026-07-11)

> **Statut : EN EXÉCUTION** — les 12 findings majeurs sont en cours de traitement (mandat mainteneur du 2026-07-11). Avancement détaillé : [§8](#8-état-davancement-exécution-2026-07-11). Les vagues mineures/infos restent à exécuter.

## 8. État d'avancement (exécution 2026-07-11)

Mandat : corriger les 12 majeurs. Arbitrages mainteneur obtenus en séance : **DEAD-01 → option A (couper)** ; **DEAD-02 → conserver (ADR-0009 R3)**.

| ID | Statut | Commit | Vérification |
|---|---|---|---|
| CONC-01 | ✅ corrigé | `9a0d538` | `wsl -race -run TestStateBump -count=10` vert |
| CONC-02 | ✅ corrigé | `9267977` | `wsl -race -run TestReleaseState -count=10` vert |
| ERR-01 | ✅ corrigé | `c32e85c` | test writer-en-échec rouge→vert ; suite cli verte |
| ERR-03 | ✅ corrigé | `df5bf24` | repro CLI : exit 0 → **exit 2** (timeout honoré) |
| ERR-02 | ✅ corrigé | `f22544b` | repro CLI : stderr peuplé, stdout propre, codes inchangés ; suites app/orch/tui/cli/e2e vertes |
| DOCS-01 | ✅ corrigé | `b7190c0` | section Unreleased remplie (×10, gate GMP, LF, PGO/baseline + fixes de séance) |
| ARCH-01/DOCS-02 | ✅ corrigé | `2f59ecb` | bijection mermaid ↔ `go list` vérifiée arête par arête |
| DOCS-03 | ✅ corrigé | `7df46dc` | grep `GlobalFactory()\|RegisterCalculator` vide hors historiques |
| DOCS-04 | ✅ corrigé | `333f15d` | grep artefacts fantômes vide dans BIGFFT.md |
| TEST-01 | ✅ corrigé | `fb1ba06` | **mutation-validé** : salvage réintroduit → test rouge ; réel → vert |
| DEAD-01 | ✅ corrigé (option A) | `23ab593` | −4 fichiers (~572 LOC prod + tests dédiés), tests miroirs excisés, CLAUDE.md + ADR-0004 §B1 + TESTING.md + CHANGELOG synchronisés. Gates : suite Windows verte, `wsl -race` bigfft+fibonacci vert, golden intact, **benchstat geomean sec/op −0,33 % / B/op −0,21 % / allocs +0,37 %** (bruit — Directive #1 satisfaite) |
| DEAD-02 | ✅ clos par décision | — (aucun code) | conserver (R3) ; la clause de révision n'est pas déclenchée — journalisé §7 |

**Bilan majeurs : 12/12 traités** (11 corrigés par commit, 1 clos par décision de conservation). Gate complet `check.ps1` passé post-exécution.

### Vagues mineures/infos (exécution 2026-07-11, 2e passe)

| ID | Vague | Statut | Commit | Note |
|---|---|---|---|---|
| CAL-01 | 1 | ✅ corrigé (option A) | `47c3b91` | `-calibrate` honore `--calibration-profile` ; test câblage + CALIBRATION.md alignée |
| ERR-04 | 1 | ✅ corrigé | `9475395` | erreur `p.Run` sur errOut + seam de test `runProgram` |
| ERR-05 | 1 | ✅ corrigé | `ab4c6e3` | warning jaune avec cause sur échec du repli terminal |
| ERR-06 | 1 (rattaché) | ✅ corrigé | `3abf15e` | `-version`/`-h` listés dans l'aide (footer statique) |
| ERR-07 | 1 (rattaché) | ✅ corrigé | `39ccc68` | cause racine préférée aux annulations de frères |
| DOCS-05+12 | 2 | ✅ corrigés | `6b96638` | arith.go portable + cp config.json dashboard |
| DOCS-06+07+10 | 2 | ✅ corrigés | `7b5792e` | 4 règles partout, ADR-0009 dans les tables, types morts purgés, rapport re-signé |
| DOCS-08 | 2 | ✅ corrigé | `d596108` | ancres CHANGELOG v4.0.0 |
| DOCS-09 | 2 | ✅ corrigé | `ac6bc87` | référence INNOVEPLAN retirée |
| DOCS-11 | 2 | ✅ corrigé | `4f4fbb0` | convention : prochain ADR = 0011 |
| CONC-03 (=ARCH-03) | 2 | ✅ corrigé | `4da9873` | commentaire ×15 → ×10 |
| SEC-01 | 2 | ✅ corrigé | `fe334b9` | limite compgen -W documentée (bash.go + contrat E5) |
| BUILD-05+06 | 2 | ✅ corrigés | `ce9a7d9` | décompte linters précisé, en-tête Makefile POSIX/WSL |
| ERR-08 | 2 | ✅ corrigé | `277231e` | 6 commentaires traduits |
| BUILD-02 | 3 | ✅ corrigé | `1er de la vague` (`chore(git)`) | `* text=auto eol=lf` + binaires ; renormalisation prouvée no-op ; WSL et Windows voient le même arbre |
| BUILD-03+04+01 | 3 | ✅ corrigés | `chore(build)` | plancher 80 % single-source (`check.sh --coverage-only`, vérifié WSL 95,3 %) ; golangci épinglé v1.64.8 ; protocole bench noté (exécution différée à la prochaine régénération justifiée) |
| SEC-02 | 3 | ✅ corrigé | `7ea55d3` | x/sys v0.43.0→v0.47.0, govulncheck : zéro vuln |
| TEST-03 | 4 | ✅ corrigé | `6ee8708` | assertion arithmétique insensible au stall, `-count=20` vert |
| TEST-04 | 4 | ✅ corrigé | `18967fe` | corpus golden épinglé (≥26 + F(50k/100k/200k)) |
| TEST-02 | 4 | ✅ corrigé | `80e9a80` | branche `!ok` assertée (config inchangée + warning ERR-05) |
| ARCH-02 | 4 | ✅ corrigé | `7170af5` | 5e règle arch_test : config ↛ {fibonacci racine, bigfft} |
| TEST-05 | 4 | ✅ corrigé | `73e8491` | commentaires « Not parallel » sur les clusters non documentés |
| TEST-06 | 4 | ✅ accepté | — (aucun code) | recommandation d'audit : acceptable en l'état (500 ms uniques) |
| DEAD-03 | 5 | ✅ corrigé | `6573888` | re-coupe getVersionInfo/VersionData (diff 760064a rejoué) |
| DEAD-05 | 5 | ✅ corrigé | `46010ce` | metrics/system inliné dans tui, mermaid+ARCH.md à jour |
| DEAD-06 | 5 | ✅ corrigé | `849eed8` + suivi docs | coreStub absorbé dans contract_test.go, package supprimé |
| DEAD-13 | 5 | ✅ corrigé | `13178f1` | setCurrentTheme dé-exporté |
| DEAD-16 | 5 | ✅ corrigé | `619a0c9` | doc.go testutil réduit à 3 lignes |
| DEAD-07 | 5 | ✅ corrigé (a minima) | `9683ace` | AsProgressCallback coupé ; **pattern Observer complet = décision mainteneur toujours ouverte (Directive #6/D-02)** |
| ARCH-04 | 5 | ✅ corrigé | `4a84ebe` | 3 alias Default*Threshold supprimés, CLAUDE.md aligné |
| DEAD-12 | 5 | ✅ accepté | — (aucun code) | indirection options conservée (branche LoadProfile testée) ; le vrai bug de câblage était CAL-01, corrigé |
| DEAD-04 | 6 | ✅ corrigé | `aa8ef4d` | internal/parallel supprimé (errgroup + `parallel3Result` du commit annulé da80099) ; benchstat 2 runs encadrant zéro (+7,1 %/−5,8 % CPU = bruit 1x documenté BUILD-01), allocs −1,0/−1,25 % stables ; `-race` + golden verts |
| DEAD-08 | 6 | ✅ corrigé | `bc9f5a6` | 6 méthodes + thresholdStats dé-exportés, CLAUDE.md A2-04 synchronisé, `-race` threshold vert |
| DEAD-09 | 6 | ✅ corrigé | `18b4b32` | trio FFTParallelismConfig documenté test-only (précédent SetFFTThreshold) |
| DEAD-10 | 6 | ✅ corrigé | `762f7b7` | allocUnsafe dé-exporté |
| DEAD-11 | 6 | ✅ corrigé (partiel par décision) | `63468c9` | GCStats/stats() coupés ; AllocBigInt/UsedWords **conservés** (précédent d89eb5c, aucun élément nouveau) |
| ARCH-05 | 6 | ✅ assumé | `337af0f` | 3 dépassements gocyclo (1-2 pts) en exclusions documentées ; le 4e (fourierRecursiveCtx) a disparu avec DEAD-01 |
| ARCH-06/07, DEAD-14/15 | 6 (constats) | ✅ acceptés | — (aucun code) | recommandation d'audit : aucune action en bloc (internal/ borne la visibilité ; découpage de fichiers = localité des invariants ; WithFactory = seam ADR-0008 R6 ; dé-exports cosmétiques à porter par un autre chantier) |
| Vague 7 (clôture) | 7 | ✅ close | dernier commit de la série | CHANGELOG consolidé (Removed/Fixed) ; CLAUDE.md synchronisé au fil des vagues ; **G7 PASS** (`check.sh` WSL : build, vet, test `-race`, GMP 3b OK, couverture 95,2 %) ; arbres git Windows et WSL propres |

### Bilan final (2026-07-11)

**59/59 findings traités** : 12 majeurs (passe 1) + 30 mineurs + 17 infos (passe 2, vagues 1-7).
- Corrigés par commit : 44 findings (~35 commits conventionnels, un par finding ou lot homogène).
- Clos par décision/acceptation documentée : DEAD-02 (oracles R3), DEAD-12, DEAD-14, DEAD-15, TEST-06, ARCH-06, ARCH-07, DEAD-11 partiel (AllocBigInt/UsedWords), BUILD-01 (différé à la prochaine régénération, protocole noté dans le Makefile).
- Décision restée ouverte (volontairement) : le sort du pattern Observer complet de `internal/progress` (Directive #6/D-02) — `AsProgressCallback` mort coupé a minima.
- Signalement hors périmètre en suivi séparé : câblage GMP dans la fabrique `app.New` (chip de tâche créé).

Critères de sortie §6 : (1) ✅ tous les findings non-décision traités ; (2) ✅ décisions présentées et arbitrées ou explicitement laissées ouvertes ; (3) ✅ G7 vert, benchstat sans régression reproductible, golden intact, arbres propres ; (4) ✅ CHANGELOG/CLAUDE.md synchronisés, journal des déviations complet (§7).

Sort des fichiers d'audit (étape 4 de la vague 7) : **conservés à la racine** — la purge (précédent `d10299b`) reste au choix du mainteneur ; auditFable5.md documente l'état des findings au moment de l'audit, ce tableau §8 documente leur exécution.

Signalement hors périmètre (non corrigé, chip de suivi créé) : sous `-tags gmp`, l'algo « gmp » n'est enregistré que dans la fabrique privée du package fibonacci — probablement insélectionnable depuis le binaire (`app.go:75` n'appelle pas `RegisterGMPCalculator`). À vérifier dans une session dédiée.

- **Exécutant prévu** : Claude Opus 4.8, mode **ultramode** (`ultracode` — orchestration Workflow multi-agents par défaut sur chaque vague substantielle).
- **Source** : findings d'[`auditFable5.md`](auditFable5.md) exclusivement (IDs cités ci-dessous). Aucun élargissement de périmètre sans journalisation (§7).
- **Références obligatoires avant exécution** : `CLAUDE.md` (Règle d'or + Invariants), `docs/adr/0008` et `0009` (candidats fermés), ce plan.
- **Pré-état vérifié (2026-07-11)** : `scripts/check.ps1` **vert** sur HEAD `6da3f3b` ; baseline benchstat et profil PGO **confirmés représentatifs** de HEAD (preuve : `git diff 3f9e7be HEAD -- internal/fibonacci internal/bigfft` vide — voir auditFable5 §5.7).

---

## 1. Principes d'exécution

1. **Un finding = un commit conventionnel** (`fix|refactor|test|docs|chore|perf(scope): …`). Groupement autorisé uniquement pour les lots homogènes du même fichier/thème dans la même vague (ex. DOCS-05+06 dans ARCH.md).
2. **Bug avant refactor** (Directive #7) : tout défaut actif se corrige dans un commit `fix(...)` isolé avant toute coupe/refactor du même fichier.
3. **Chirurgie** (Directive #5) : diff minimal, aucune « amélioration » opportuniste hors finding.
4. **Ordre des vagues** = risque croissant ; chaque vague fermée par son gate avant d'ouvrir la suivante.
5. **Items `DÉCISION MAINTENEUR`** : ne s'exécutent **pas** sans arbitrage explicite. L'exécutant présente les options en fin de vague 5 (AskUserQuestion ou équivalent) et poursuit le reste ; sans réponse, les items restent ouverts et sont journalisés.
6. **Ultramode** : vague à ≥ 3 findings → fan-out Workflow (1 agent par finding ou par fichier cible), puis 1 agent réfutateur par diff avant commit (« casse ce correctif : invariant CLAUDE.md violé ? appelant oublié ? gate requis ? »). Vague triviale → exécution directe.
7. **Git côté Windows uniquement** (`git status/add/commit` via Bash tool) — jamais depuis `wsl bash -lc` (faux modifiés CRLF/LF, précédent 2026-07-10). WSL réservé à `go test -race`/`make`/benchstat.

## 2. Gates (seuils de vérification)

| Gate | Commande | Quand | Seuil |
|---|---|---|---|
| G1 build+vet | `go build ./... && go vet ./...` | chaque commit | zéro erreur |
| G2 tests | `go test ./... -count=1` (Windows, sans -race) | chaque commit | 100 % vert |
| G3 race | `wsl go test -race ./... -count=1` | fin de chaque vague code | 100 % vert (échec intermittent → 5 passes en isolation avant verdict ; précédent connu : `TestModel_HandleReset_FreshTimeoutBudget`) |
| G4 couverture | plancher 80 % (`scripts/check.ps1` étape 4) | fin de chaque vague | ≥ 80 % |
| G5 golden | `go test ./internal/fibonacci/ -run Golden -count=1` | tout changement algorithmique | vert, `fibonacci_golden.json` intact (immuable) |
| G6 benchstat | `wsl` : bench post vs `docs/audits/bench-baseline.txt` | tout diff prod `internal/fibonacci/` ou `internal/bigfft/` | régression > 5 % = blocage (Directive #1). Diff test-only : G6 sans objet, à noter au commit. Comparaison **toujours sous WSL** (baseline en goos: linux) |
| G7 gate complet | `scripts/check.sh` (WSL, inclut 3b GMP) | fin de plan, avant push | 100 % vert |
| G8 gardiens | exécution nominative des tests gardiens CLAUDE.md touchés par la vague | chaque vague | 100 % vert |

## 3. Vagues d'exécution

### Vague 0 — pré-vol (aucun commit)
Arbre propre ; `check.ps1` vert (déjà constaté 2026-07-11) ; relire auditFable5 §3 (exclusions) et §7 (synthèse). Baseline/PGO : **rien à faire** (cohérence prouvée, auditFable5 §5.7).

### Vague 1 — correctifs actifs (`fix`/`test`) — risque faible, valeur haute
| Ordre | ID | Action | Gate spécifique |
|---|---|---|---|
| 1.1 | CONC-01 | `state_cache_test.go` : `ReleaseState(s)` → `finalizeStateReleaseTo(s, func(*CalculationState) {})` | `wsl go test -race ./internal/fibonacci/ -run 'TestStateBump' -count=10` ; G6 sans objet (test-only) |
| 1.2 | CONC-02 | idem, sous-cas nominal de `state_pool_arena_test.go` | `wsl … -run 'TestReleaseState' -count=10` |
| 1.3 | ERR-01 | `WriteResultToFile` : Close vérifié via named return (patron generate-golden) | test writer-en-échec rouge→vert ; errcheck |
| 1.4 | ERR-03 | `ctx` dans `FastDoublingMod` + check par itération ; wrap `DeadlineExceeded` | test deadline mi-boucle ; repro CLI → exit 2. ⚠ prod `fibonacci/` → G6 requis (attendu : nul, chemin --last-digits hors benchmarks) |
| 1.5 | ERR-02 | router échecs de calcul vers `ErrWriter` (validateMemoryBudget + chemin comparaison) | repro des 3 invocations : stderr peuplé, codes inchangés ; goldens CLI ajustés |
| 1.6 | CAL-01 *(confirmé, mineur — auditFable5 §5.9)* | option A (recommandée) : transmettre `cfg.CalibrationProfile` au flux `-calibrate` + aligner CALIBRATION.md ; option B : restreindre le texte d'aide du flag | test unitaire chemin profil personnalisé ; aide CLI ↔ CALIBRATION.md cohérentes |
| 1.7 | ERR-04 | erreur de `p.Run()` écrite sur stderr | test injecté |
| 1.8 | ERR-05 | warning si auto-calibration échoue | test strategy-en-échec |

### Vague 2 — documentation & cohérence (`docs`) — zéro risque runtime
| ID | Action |
|---|---|
| DOCS-01 + DOCS-08 | CHANGELOG : section Unreleased (×10, gate GMP 3b, LF, PGO/baseline) + pied de page (ancres, `[4.0.0]`, commentaire tags) |
| ARCH-01/DOCS-02 | mermaid : retirer `orch --> format`, ajouter les 4 arêtes réelles ; puis DOCS-10 (re-signer validation-report) |
| DOCS-03 | exemples `GlobalFactory`/`RegisterCalculator` → `NewDefaultFactory()` (7 sites + CONTRIBUTING + ARCH) |
| DOCS-04 + DOCS-05 + DOCS-06 | BIGFFT.md (sections mortes, table fichiers), BUILD/PORTABILITY (split arith), ARCH.md (scan.go, types d'erreurs) |
| DOCS-07 | « trois règles » → quatre (TESTING.md, architecture/README ×2, ARCH.md:731) + ligne ADR-0009 dans les tables |
| DOCS-09 | retirer la référence INNOVEPLAN.md |
| DOCS-11 | acter « prochain ADR = 0011 » (template ou CLAUDE.md) |
| DOCS-12 | procédure dashboard : `cp config.json` (ou documenter que dist/ le contient, après vérif) |
| CONC-03 (=ARCH-03) | commentaire `×15` → `×10` dans `memory/arena.go:12` |
| SEC-01 | commentaire de contrat compgen dans `bash.go` + amendement contrat E5 d'`escape.go` |
| BUILD-05, BUILD-06 | notes doc (« 24 entrées = 23 linters + typecheck », « Makefile = POSIX/WSL ») — facultatif, grouper |

### Vague 3 — outillage (`chore`/`fix`) 
| ID | Action | Gate spécifique |
|---|---|---|
| BUILD-02 | `.gitattributes` : `* text=auto eol=lf` + binaires explicites | `git add --renormalize . && git status --porcelain` → vide ; `wsl git status` → vide |
| BUILD-03 | `coverage-check` (Makefile) délègue à `check.sh` | `wsl make coverage-check` |
| BUILD-04 | épingler golangci-lint `@v1.64.8` dans `install-tools` | lecture |
| SEC-02 | bump `golang.org/x/sys` | `govulncheck ./...` → 0 (import inclus) ; `go mod tidy -diff` propre ; G1-G2 |
| BUILD-01 | **conditionnel** : ne s'exécute qu'à la prochaine régénération justifiée de la baseline — noter le protocole (`-benchtime=3x` ou écarter le 1er échantillon) en commentaire de la cible `bench-baseline` | lecture |

### Vague 4 — tests (`test`)
| ID | Action | Gate spécifique |
|---|---|---|
| TEST-01 | assertion d'épinglage E1-R4 (`&entry.backing[0]` diffère) + corriger commentaires (test + `fft_cache.go:47`) | mutation locale salvage → rouge ; réel → vert. Commentaire prod touché dans bigfft : G6 sans objet (commentaire seul) |
| TEST-02 | assertion du chemin `!ok` (config inchangée) | sous-test forcé `!ok` |
| TEST-03 | assertion budget frais insensible au stall (`deadline.Sub(before) >= Timeout`) | `-count=20` sous charge |
| TEST-04 | épinglage du corpus golden (`len >= 26` + présence 50k/100k/200k) | copie tronquée → rouge |
| ARCH-02 | règle arch_test : config ↛ {fibonacci racine, bigfft} | `TestArchitectureLayering` |
| TEST-05 | commentaires « Not parallel: … » manquants (lot) | re-scan |

### Vague 5 — coupes hors cœur (`refactor`) + points de décision
| ID | Action | Gate spécifique |
|---|---|---|
| DEAD-03 | re-couper `getVersionInfo`/`VersionData` (+ test) | `go test ./internal/app/` |
| DEAD-05 | inliner `metrics/system.Sample` dans tui, supprimer le package | `go test ./internal/tui/` ; mermaid + docs à jour dans le même commit |
| DEAD-06 | `CoreStub` → `orchestration/contract_test.go`, supprimer `fibonaccitest` | `go test ./internal/orchestration/` ; doc.go refs |
| DEAD-13 | dé-exporter `ui.SetCurrentTheme` | `go test ./internal/ui/` |
| DEAD-16 | réduire `testutil/doc.go` | build |
| DEAD-07 | couper `AsProgressCallback` seul ; **pattern complet = DÉCISION MAINTENEUR (Directive #6)** | `go test ./internal/progress/ ./internal/fibonacci/` |
| ARCH-04 | **DÉCISION légère** : supprimer les 3 aliases `Default*Threshold` OU corriger la phrase CLAUDE.md | build + grep vide |
| DEAD-12 | accepter (défaut) ou aplatir l'indirection options calibration — décision légère | `go test ./internal/calibration/` |
| ⚠ Point d'arrêt | présenter au mainteneur : DEAD-01 (FFTContext, options A/B), DEAD-02 (oracles vs ADR-0009 R3), DEAD-07 pattern, DEAD-11 (`AllocBigInt`/`UsedWords` — précédent de conservation d89eb5c), ARCH-05 (refactor vs exclusion lint) | — |

### Vague 6 — cœur sensible (`refactor`/`perf`) — G6 par commit, WSL `-race` par commit
| ID | Action | Gate spécifique |
|---|---|---|
| DEAD-04 | `internal/parallel` → intégration dans `common.go`. ⚠ NE PAS utiliser errgroup naïf pour `executeParallel3` (précédent mesuré : +15-19 % allocs/op) — reprendre l'approche struct `parallel3Result` du commit annulé (`git show` des commits de vague 1 restaurés) | benchstat complet + golden + `-race` |
| DEAD-08 | dé-exporter les 6 méthodes test-only du threshold manager + **sync CLAUDE.md (A2-04 cite `Reset`)** | `wsl go test -race ./internal/fibonacci/threshold/` |
| DEAD-09 | knobs bigfft : documenter test-only ou dé-exporter (cohérent avec la décision DEAD-01) | suite bigfft ; G6 sans objet si symboles jamais appelés |
| DEAD-10 | couper/dé-exporter `AllocUnsafe` | `go test ./internal/bigfft/` |
| DEAD-11 | couper `GCStats`/`stats()` ; `AllocBigInt`/`UsedWords` selon décision vague 5 | `-race` memory + gardiens |
| DEAD-01 / DEAD-02 | **uniquement si arbitrés** en vague 5 ; DEAD-02 exige un ADR amendant R3 **avant** le code ; DEAD-01 option A inclut : gardien CLAUDE.md retiré, commentaires fft.go réécrits, ADR-0004 §B1 amendé | build + fuzz + golden + benchstat + `-race` complet |

### Vague 7 — clôture
1. G7 (`scripts/check.sh` sous WSL, gate GMP inclus) + G3 complet.
2. CHANGELOG : consolider les entrées des vagues sous Unreleased.
3. CLAUDE.md : synchroniser (gardiens si DEAD-01A, A2-04 si DEAD-08, ARCH-04, note gitattributes, chiffre linters si retouché).
4. Sort des deux fichiers d'audit : **purge** (précédent : commit `d10299b`) ou archivage dans `docs/audits/` — décision mainteneur, défaut = purge avec renvoi CHANGELOG/ADR.
5. Commit(s) + push `main` (Windows).

## 4. Protocole ultramode par vague (gabarit Workflow)

```
phase('Scan')    : relire les findings de la vague dans auditFable5.md + fichiers cibles (1 agent).
phase('Exécute') : pipeline(findings) — 1 agent par finding : test rouge d'abord si fix,
                   diff minimal, gardiens locaux verts, commit conventionnel.
phase('Réfute')  : 1 agent réfutateur par diff : invariants CLAUDE.md, appelants oubliés,
                   exclusions ADR-0008/0009 respectées, gate benchstat si cœur.
phase('Gate')    : G1-G8 applicables ; échec → revert du commit fautif, re-tentative unique,
                   sinon journaliser (§7) et reporter.
```
Les agents d'exécution reçoivent : le texte du finding, les exclusions (auditFable5 §3), la règle git-Windows (§1.7) et le gate applicable.

## 5. Rollback

- Chaque commit est atomique → `git revert <sha>`. Jamais de `reset --hard` sur du poussé.
- Vague dont le gate de sortie échoue après re-tentative : commits sains conservés, finding fautif reporté avec cause au journal.

## 6. Critères de sortie du plan

1. Tous les findings non-`DÉCISION MAINTENEUR` traités (exécutés ou reportés avec cause).
2. Items décision : options présentées, décision obtenue ou explicitement laissée ouverte au journal.
3. G7 vert ; G6 sans régression > 5 % ; golden intact ; `wsl git status` et `git status` propres tous deux.
4. CHANGELOG et CLAUDE.md synchronisés ; journal des déviations complet.

## 7. Journal des déviations (à remplir pendant l'exécution)

Format, une entrée par déviation :

```
[AAAA-MM-JJ] vague N, package P
Écart au plan : …
Raison : …
Alternative écartée : …
```

```
[2026-07-11] exécution des 12 majeurs, internal/bigfft
Écart au plan : DEAD-01 et DEAD-02 (items « DÉCISION MAINTENEUR », prévus
vague 6) ont été arbitrés et traités dès la passe des majeurs, hors
séquence de vagues.
Raison : mandat mainteneur explicite « corriger les 12 majeurs » ;
arbitrage obtenu en séance (DEAD-01 → option A couper ; DEAD-02 →
conserver, position ADR-0009 R3 maintenue, clause de révision non
déclenchée). L'ordre des vagues reste applicable aux mineurs/infos.
Alternative écartée : attendre la vague 6 — rejeté, l'arbitrage était
la seule dépendance et il était acquis.
```

```
[2026-07-11] exécution des 12 majeurs, docs/algorithms/BIGFFT.md
Écart au plan : l'agent DOCS-04 a retiré context.go/fft_recursion_ctx.go
de la table Package Structure AVANT leur suppression effective (DEAD-01).
Raison : les conserver aurait rendu la table fausse dès la fin de la
session ; la suppression DEAD-01 était déjà arbitrée.
Alternative écartée : mise à jour en deux temps — rejeté, churn inutile.
```
