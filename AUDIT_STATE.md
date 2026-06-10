# AUDIT_STATE — boucle d'audit FibGo (démarrée 2026-06-10)
Dernière itération : 6 — 2026-06-10 06:20 — dernier commit : fa13bfd
Branche de travail : `refactor/audit-loop-2026-06` (créée depuis `main` propre)

| Phase | Intitulé                          | Statut      | Commit(s) | Preuves / métriques |
|-------|-----------------------------------|-------------|-----------|---------------------|
| 1     | Perf cœur Fibonacci               | IN_PROGRESS |           | Gate + baselines lancés en arrière-plan (it. 0) |
| 2     | Couverture ≥ 90 %                 | PENDING     |           |                     |
| 3     | Documentation fidèle              | PENDING     |           |                     |
| 4     | Claude.md                         | PENDING     |           |                     |
| 5     | README.md                         | PENDING     |           |                     |
| 6     | /understand + dashboard           | PENDING     |           |                     |
| 7     | Épuration                         | PENDING     |           |                     |
| 8     | Validation finale + push main     | PENDING     |           |                     |

## Sous-tâches de la phase en cours (Phase 1)
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
