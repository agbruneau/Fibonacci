# AUDIT_STATE — boucle d'audit FibGo (démarrée 2026-06-10)
Dernière itération : 8 — 2026-06-10 07:25 — dernier commit : 7999c39
Branche de travail : `refactor/audit-loop-2026-06` (créée depuis `main` propre)

| Phase | Intitulé                          | Statut      | Commit(s) | Preuves / métriques |
|-------|-----------------------------------|-------------|-----------|---------------------|
| 1     | Perf cœur Fibonacci               | DONE        | 4e34b82, fa13bfd, 7999c39, doc audits | geomean sec/op −12,0 % ; FastDoubling/10M −15,3 % ; B/op 10M ~−70 % cumulés ; 0 régression ; -race WSL PASS ; lint 0 nouvelle ; baseline `docs/audits/bench-audit-loop-2026-06.md` |
| 2     | Couverture ≥ 90 %                 | DONE        | 9cad06e, a2e4eee, c306344…d549da4 (9 commits par pkg) | **Total 95,0 %** (88,9 → 95,0) ; -race ./... complet WSL PASS ; data race réelle corrigée (threshold) ; lint 0 nouvelle ; exception unique : generate-golden 88,6 % (main/os.Exit) |
| 3     | Documentation fidèle              | DONE        | e790f17, 7675a7f, 3170baf, 7297c8b, 111d085, 648798c, ed6c334, 7b97d48 | 95/95 corrections traitées (68 appliquées + 26 adaptées + 1 obsolète) ; liens OK ; CHANGELOG Unreleased complet ; doc.go 21/21 ; commandes exécutées (check.ps1 Overall PASS, cov 95,1 %) ; bug Makefile PGO corrigé ; GMP marqué non-vérifié (libgmp-dev absent) |
| 4     | Claude.md                         | IN_PROGRESS |           | 3 dérives cataloguées (lignes 69/84/103) + nouveaux invariants |
| 5     | README.md                         | PENDING     |           |                     |
| 6     | /understand + dashboard           | PENDING     |           |                     |
| 7     | Épuration                         | PENDING     |           |                     |
| 8     | Validation finale + push main     | PENDING     |           |                     |

## Sous-tâches de la phase en cours (Phase 3 — documentation fidèle)
- [ ] Appliquer les 95 corrections préparées par le workflow (`audit_phase3_claims.md` : 77 périmées, 15 douteuses, 2 liens cassés) — fan-out par document
- [ ] Exécuter réellement les commandes documentées — fait pour : §4.1 -race (passes complètes archivées) ; §4.4 cache FFT (mesuré 2026-06-10, le cache n'aide pas le chemin par défaut) ; §4.6 dry-runs des commandes corrigées (bench → 6 sous-benchs matchés ; progress → ok). §4.5 GMP : **impossible** — libgmp-dev absent de WSL (gmp.h manquant), sudo interactif requis ⇒ chiffres GMP.md marqués « non vérifiés » avec commande prête. §4.2 knowledge graph = Phase 6. §4.3 : la baseline fraîche du jour est bench-audit-loop-2026-06.md (source unique). Reste : scripts check.ps1/sh + make targets clés à exécuter en fin de Phase 3.
- [ ] Diagrammes Mermaid C4 vs arborescence réelle (+ registres ADR à étendre à 0008)
- [ ] CHANGELOG.md : section Unreleased avec les changements Phases 1–2
- [ ] ADR : aucune réécriture ; notes de statut datées si nécessaire
- [ ] `doc.go` complets (`Glob internal/*/doc.go`)
- [ ] Liens relatifs valides ; commits `docs(scope):`

## Phase 2 (DONE — résumé)
- **Total : 88,9 % → 95,0 %** (preuve : `total: (statements) 95.0%`). Tous packages ≥ 88,6 % ; seuls < 90 : generate-golden 88,6 % (exception : main/os.Exit, justifiée dans le rapport du workflow).
- 9 commits `test(pkg)` atomiques (c306344…d549da4) + `9cad06e` (réparation test cassé masqué par pollution) + `a2e4eee` (fix **data race réelle** du DynamicThresholdManager découverte par la 1re passe `-race ./...` complète, WSL).
- Aucune assertion affaiblie (consigne agents + spot-check) ; états globaux systématiquement sérialisés/restaurés ; golden intouché ; `-race ./...` complet PASS post-fix ; lint : zéro nouvelle issue vs baseline (le seul ajout réel — un misspell dans un commentaire — corrigé ; flip-flops `uniq-by-line` revive↔gocritic sur prod intacte documentés comme bruit d'outil).

## Sous-tâches Phase 2 (closes — archive)
- [x] Mesure initiale : **total 88,9 %** (`go tool cover -func`, preuve : `total: (statements) 88.9%`). Packages < 90 % (croissant) : cmd/generate-golden 28,6 ; cmd/fibcalc 75,0 ; bigfft 85,3 ; tui 87,8 ; fibonacci/memory 87,9 ; orchestration 88,0 ; app 88,6 ; config 88,6 ; errors 89,2. ≥ 90 % : cli 90,1 ; fibonacci 91,2 ; calibration 91,6 ; progress 93,9 ; ui 94,9 ; metrics 95,7 ; completion 96,1 ; threshold 96,9 ; format 98,6 ; parallel/testutil/system/fibonaccitest 100.
- [x] Fan-out multi-agents terminé (workflow wy9d9ouyx, 9 agents, ~32 min) : rapports — bigfft 85,3→96,0 % (+40 tests, états globaux sérialisés+restaurés, exceptions documentées ligne par ligne) ; tui 87,8→97,3 % (+19, TTY interactif en exception) ; memory 87,9→99,1 % (+8, GOGC sérialisé) ; + orchestration/app/config/errors/fibcalc/generate-golden (rapports complets dans la sortie du workflow).
- [x] Re-mesure totale : **95,0 %** (preuve log : `total: (statements) 95.0%`) — objectif ≥ 90 % dépassé.
- [x] **Data race RÉELLE détectée et corrigée (commit a2e4eee, Directive 7)** : première passe `-race` complète du module (WSL) → race `MetricsBuffer.Record` vs `Count` dans le DynamicThresholdManager (commentaire mensonger « internally safe », aucun verrou). Fix : tous les accès buffer sous le `mu` existant ; bench DTM ciblé avant/après : 10M neutres p>0.05, zéro régression.
- [ ] Validation finale Phase 2 en cours (tâche bhv37xolw) : `wsl go test -race ./...` complet post-fix + lint diff. Ensuite : commits atomiques par package du fan-out.
- [x] **Bug préexistant corrigé en passant (Directive 7, commit 9cad06e)** : `TestSetTransformCacheConfig` était structurellement cassé — mock non conforme à la garde de forme A-05 (N=10 vs coefficients 101 mots ⇒ Put toujours rejeté silencieusement) + sous-tests `t.Parallel` mutant le singleton global ; il ne passait en suite que par pollution du cache par les tests voisins et échouait en isolation. Révélé par le 1er run de couverture (exit 1).
- [ ] Traiter les packages du moins couvert au plus couvert (chemins d'erreur d'abord ; table-driven ; t.Parallel ; testutil/fibonaccitest ; pas de tests tautologiques)
- [ ] Workflow multi-agents pour le fan-out par package (directive U2) une fois le tableau établi
- [ ] Justifier les exceptions <90 % (plateforme, TUI interactif, tag gmp) et tenir le total ≥ 90 %
- [ ] Preuve : ligne `total:` copiée ici ; commits `test(scope):` atomiques

## Phase 1 (DONE — résumé)
- Détails et tableaux : `docs/audits/bench-audit-loop-2026-06.md` + messages des commits 4e34b82 / fa13bfd / 7999c39.
- DoD vérifiée : suites fibonacci+bigfft vertes (-count=1 et -race via WSL) ; 6 tests gardiens historiques + 8 nouveaux verts ; benchstat archivé, aucune régression > 5 % (cumul : tout ~ ou négatif) ; lint 0 nouvelle issue vs baseline D1 ; refactor lisibilité : non retenu (aucun candidat neutre profitable, discipline chirurgicale) ; arrêt des cycles : plus de candidat crédible désigné par le profil (C1 réfuté par lecture, C2/C3 retenus, parallélisation des forward transforms écartée — gain borné, complexité).

## Archive des sous-tâches Phase 1 (closes)
- [x] Lire ADR-0004 (backlog : B1/B2 WONT-FIX — FFTContext et calibration hors périmètre) et ADR-0008 (R1–R7 rejetés, ne pas re-tenter)
- [x] Lire baseline de référence `docs/audits/bench-parallel-pointwise-2026-06.md` (FastDoubling/10M ≈ 36,25 ms ; 1M ≈ 5,35 ms ±40 % ; F(100M) ≈ 0,204 s)
- [x] Gate de validation initial — `go build` ✓, `go vet` ✓, lint = ADVISORY (décision D1, baseline 151 issues), `go test ./... -count=1` ✓ **24/24 packages ok** (golden inclus ; log `audit_iter0_gate_bench.log` 05:07:07)
- [x] Baseline bench `./internal/fibonacci/` → `bench_before.txt` (benchtime=2s, count=6, 314.9 s, terminée 05:12:23)
- [x] Baseline bench `./internal/bigfft/` → `bench_before_bigfft.txt` ✓ (benchstat parse 0 erreur)
- [x] Baseline bench fibonacci PROPRE (`bench_before.txt`, 310 s, benchstat 0 erreur de parse). Cause racine de la 1re capture inutilisable : logs JSON zerolog trace interfoliés (logger global au niveau trace hors `app.New`, `calculator.go:227-232`). Correctif test-only commité : `internal/fibonacci/testmain_test.go` (TestMain aligne le niveau global sur Info comme la prod). Revalidation complète AVANT commit : build ✓ vet ✓ test 24/24 ✓ lint 0 nouvelle issue (log `audit_iter3_validate_bench_profile.log`).
- [x] Comparer baseline vs référence 2026-06-09 : SAINE (FastDoubling/10M 33,30 ms vs 36,25 ; FFT/10M 34,63 vs 39,22 ; MatrixExp/10M 38,40 vs 42,38 ; CI ±2-3 % sur 10M)
- [x] Profilage cpu/mem analysé. CPU 10M : 76,5 % flat dans les primitives asm math/big (addMulVVWW 37,8 %, subVV 14,1 %, lshVU 13,3 %, addVV 11,3 %) ; couche FibGo mince. CPU 1M : >97 % math/big (Karatsuba), rien à optimiser côté FibGo. MEM 10M : NewCalculationArena 45,8 % des allocs, AcquireBumpAllocator+Alloc ~25 %, Poly.IntTo 9,9 % (=F-004, ne pas toucher).
- [x] **Candidat 1 (rejeté par vérification, sans bench)** : garde `kb != 0` autour du `shlVU` de `fermat.Shift` (2,17 s/24,5 s au profil). RÉFUTÉ : `math/big.shlVU` (linkname) court-circuite déjà s==0 par `copy(z,x)` → memmove fast-path src==dst O(1) (lu dans `$GOROOT/src/math/big/arith_decl.go:48-54`). Les lshVU mesurés sont des shifts réels nécessaires.
- [x] **Candidat 2 — RETENU, commit fa13bfd** : cache mono-slot GC-immune state+arena par instance (`FastDoublingCalculator.cachedState`). benchstat n=6 p=0.002 : FastDoubling/10M −12,3 % (33,30→29,22 ms), FFTBased/10M −10,2 %, MatrixExp/10M −25,3 %, DTM 10M −16/−17 %, geomean sec/op −7,96 % ; B/op −45 à −61 % sur les chemins fast doubling. Aucune régression significative (pire : FFTBased/1M +2,9 % p=0.065 NS). Gate : build/vet/test ./... verts, lint 0 nouvelle issue (un shadow govet dans le test corrigé avant commit), 5 tests gardiens neufs + gardien historique vert. bigfft non modifié (re-bench non requis).
- [ ] **Candidat 3 (cycle en cours, tâche b95c3p8ky)** : F-012 implémenté — bump allocator acquis une fois par calcul (dimensionné pour l'étape finale via `s.fftBumpCapWords`, posé par `prepareStateForN`), porté par le `CalculationState`, `Reset()` entre étapes (Alloc zéroe chaque slice ⇒ réutilisation sûre), jamais visible d'`executeParallel3` (ses ops utilisent le pool allocator). Politique de rétention : suit l'arène (drop anti-bloat + drop overLimit → `ReleaseBumpAllocator`). 3 tests gardiens bump ajoutés. Verdict au benchstat vs base C2.
- [x] **Revue adversariale C2 (workflow wjp0s4mtb, 20 agents)** : aliasing=APPROVE, concurrence=APPROVE, conformité=CONCERNS (0 critical/major, 7 minor). Traités : 2 tests durcis (SequentialResultsIndependent → n2=80k<n1 pour réfuter réellement la perte du deep-copy ; ReusesArena → état construit hors statePool pour éliminer un flake sous t.Parallel) ; **-race exécutable sur cet hôte via WSL (go1.26.0 linux/amd64 vérifié)** — inclus au cycle C3 (fibonacci+bigfft) ; dérive Claude.md (3 lignes) → consignée pour la Phase 4 dans `audit_phase3_claims.md`.
- [x] **Préparation Phase 3 (même workflow)** : 1019 affirmations documentaires vérifiées, 95 problèmes (77 périmées, 15 douteuses, 2 liens cassés, 1 non-vérifiable). Artefact : `audit_phase3_claims.md` (top-5 : identifiants fantômes transverses ; commandes bench mortes ; cycle de vie state/arena post-fa13bfd à documenter ; références/registres ADR périmés ; MATRIX.md décrit Strassen classique alors que le code implémente Winograd + off-by-one PROGRESS_BAR).
- [ ] Cycles candidats : hypothèse → diff minimal → golden + tests gardiens → benchstat (p<0.05) → garder ou revert ; arrêt après 2 candidats secs consécutifs
- [ ] Refactor lisibilité neutre (optionnel, bench ± bruit)
- [ ] Rédiger `docs/audits/bench-audit-loop-2026-06.md` (nouvelle baseline datée + candidats rejetés)
- [ ] `golangci-lint run ./internal/fibonacci/... ./internal/bigfft/...` propre
- [ ] Commits `perf(...)`/`refactor(...)` + mise à jour état

## Mesures
- Couverture totale initiale : (sera mesurée en Phase 2) / courante : —
- Baseline bench : `bench_before.txt` + `bench_before_bigfft.txt` (2026-06-10, en cours de capture)
- Référence comparée : `docs/audits/bench-parallel-pointwise-2026-06.md` (2026-06-09, même hôte)

## Environnement (vérifié it. 0)
- Windows 11, Go 1.26.4 windows/amd64, golangci-lint v1.64.8, benchstat présent.
- gcc ABSENT ⇒ la validation `-race` n'est PAS exécutée sur cet hôte ; déléguée à WSL/Linux (non exécutée à ce stade — à mentionner dans le commit final, cf. Phase 8).
- Quirks hôte appliqués : `-bench=Benchmark` (jamais `-bench=.` en PowerShell) ; `git add Claude.md` (casse) ; comptage de lignes via `wc -l`/`(Get-Content f).Count`.

## Directives utilisateur (reçues en cours de boucle)
- **U1 (2026-06-10, pour la Phase 5 — README.md)** : ajouter AU DÉBUT du README une mention
  indiquant que le code a été audité en totalité, refactorisé et optimisé à l'aide du modèle
  **Claude Fable 5** en mode effort **Max**, avec un hyperlien vers la page Anthropic présentant
  Claude Fable 5. URLs vérifiées (WebSearch 2026-06-10) :
  annonce <https://www.anthropic.com/news/claude-fable-5-mythos-5> (cible recommandée — présente
  spécifiquement Fable 5, publiée le 2026-06-09) ; page produit <https://www.anthropic.com/claude/fable>.

- **U2 (2026-06-10)** : l'utilisateur demande l'usage des **workflows dynamiques** (orchestration
  multi-agents) dans la boucle. Appliqué : workflow `wjp0s4mtb` (revue adversariale 3 lentilles du
  commit fa13bfd + vérification statique des claims de toute la documentation → artefact
  `audit_phase3_claims.md` pour la Phase 3). À réutiliser quand une phase s'y prête
  (Phase 2 : fan-out par package ; Phase 3 : application des corrections par document),
  en gardant les agents read-only pendant les benchmarks.

## Décisions de boucle
- **D1 (it.1) — Lint = advisory, critère « aucune nouvelle issue »** : `golangci-lint run ./...`
  retourne 151 issues PRÉEXISTANTES (capturées dans `lint_baseline.txt`, format line-number).
  C'est la politique écrite du dépôt, pas une dérogation de la boucle :
  `scripts/check.ps1:15-20` (« golangci-lint is ADVISORY ... the hard gate is
  build/vet/test/coverage »), `docs/BUILD.md:285`, et finding F-009 de
  `docs/audits/audit-2026-05-29.md` (burn-down jugé non chirurgical, reporté à une PR dédiée).
  Gate de la boucle adapté : hard gate = build/vet/test (+ couverture en Phases 2/8) ;
  lint = comparaison à `lint_baseline.txt` avant chaque commit (zéro NOUVELLE issue).
  La DoD Phase 1 « lint propre sur fibonacci/bigfft » s'interprète de même (politique
  « aucune nouvelle alerte », cf. audit-2026-05-29 §5).

## Blocages / questions pour l'utilisateur
- (aucun)

## Candidats d'épuration non tranchés
- (aucun)
