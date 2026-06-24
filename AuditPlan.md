# AuditPlan.md — Audit Go exhaustif FibGo (2026-06-24)

> Document de travail pilotant l'exécution des révisions issues de l'audit Go exhaustif
> du dépôt `github.com/agbruneau/FibGo`. Auteur de l'audit : **Claude Opus 4.8** (effort `xhigh`,
> orchestration multi-agents : 19 agents, vérification adversariale de chaque trouvaille moyenne/haute).

---

## 1. Loop retenu (signals.forwardfuture.ai/loop-library)

**The architecture satisfaction loop** — *« Refactors code architecture through tested,
independently reviewed checkpoints. »*

**Justification.** Sur les 64 loops de la librairie, c'est le seul qui correspond à une base
**stabilisée et perf-critique** où chaque modification doit franchir un *checkpoint testé* avant la
suivante. Le dépôt fournit déjà l'appareillage de checkpoint exigé par ce loop :

- tests gardiens nommés par invariant (`arch_test.go`, `TestReleaseState_*`, `TestStateBump_*`, …) ;
- oracle de non-régression (`fibonacci_golden.json`) ;
- garde perf `benchstat` vs `make bench-baseline` (Directive #1, régression > 5 % = blocage) ;
- gate local `scripts/check.{sh,ps1}` (build → vet → test → lint → couverture ≥ 80 %).

Sous-loops adaptés :
- **The housekeeper loop** (#21) → phase de purge du code mort (« purger les fichiers inutiles »).
- **The docs sweep** (#1) → resynchronisation `CLAUDE.md` / `README.md` / `docs/`.

**Adaptation.** Chaque item ci-dessous = un checkpoint : *test rouge (si bug) → correctif minimal →
build+test verts → (benchstat si hot path)*. Aucun item ne passe au suivant tant que son gate n'est pas vert.

---

## 2. État de référence (baseline avant exécution)

| Gate | Résultat |
|------|----------|
| `go build ./...` | ✓ |
| `go vet ./...` | ✓ |
| `go test ./... -count=1` (Windows, sans `-race`) | ✓ 0 échec, 24/24 paquets |
| `internal/tui/component` | seul paquet sans tests (48 LOC, stub documenté — KEEP) |

**Constat global.** Base **saine** : aucun bug haute sévérité, aucune data race, aucune fuite,
invariants `fibonacci`/`bigfft` respectés. La consigne « réviser chacun des fichiers » est exécutée
comme **ensemble ciblé chirurgical** : la plupart des 271 fichiers n'ont aucun changement.

**Périmètre validé :** *agressif / complet* (inclut les items différés risqués). Livraison : commits
atomiques conventionnels directs sur `main` + push `origin/main`.

---

## 3. Tâches — Tier 1 : Correctness (sévérité moyenne)

- [x] **T1.1 — Panic dans la récursion FFT parallèle échappe au `recover`** *(hot path → benchstat)*
  - `internal/bigfft/fft_recursion.go:130-143`, `internal/bigfft/fft_recursion_ctx.go:55-67`
  - Pb : la goroutine async forward l'erreur (`errAsync`) mais **pas** les panics → crash process,
    contourne `fermatPanicToError` (viole ADR-0002).
  - Fix : miroir du pattern frère (`executeReconstruction`, `runPointwise`) — `panicCh` bufferisé (cap 1),
    `recover()` → envoi non bloquant, drain + re-panic après `wg.Wait()`, avant le check `errAsync`.
  - Vérif : nouveau test plantant un `dst[1]` malformé dans la moitié async ; `benchstat` vs baseline (< 5 %).

- [x] **T1.2 — `--algo all --quiet` saute la vérif de divergence**
  - `internal/app/calculate.go:197-200`
  - Pb : `present()` court-circuite vers `DisplayQuietResult`/`ExitSuccess` sans appeler la comparaison
    de cohérence → résultat faux silencieux, exit 0.
  - Fix : en mode quiet avec `len(results) > 1`, exécuter le check de cohérence (→ `ExitErrorMismatch`)
    avant de court-circuiter ; chemin quiet mono-calculateur inchangé.
  - Vérif : test quiet+all, 2 mocks divergents → `ExitErrorMismatch`.

- [x] **T1.3 — Garde de génération absente sur les messages de données (TUI)**
  - `internal/tui/handlers.go:17-48`, `internal/tui/model.go:100-125`, `messages.go`, `commands.go:44`
  - Pb : `ProgressMsg`/`ComparisonResultsMsg`/`FinalResultMsg`/`ErrorMsg` ne portent pas de génération ;
    après Restart (`r`), la goroutine précédente pollue l'UID réinitialisé (corruption d'état).
  - Fix : ajouter `Generation` à ces 4 messages, l'estampiller depuis `startCalculationCmd`, garde
    `if msg.Generation != m.generation { return }` en tête de chaque handler (router `model.go` reste pur).
  - Vérif : tests de génération stale (étendre les `TestModel_Update_*_StaleGeneration` existants).

## 4. Tâches — Tier 2 : Correctness (sévérité basse, vérifiés)

- [x] **T2.1 — `End()` écrase un GOMEMLIMIT préexistant** — `internal/fibonacci/memory/gc_control.go:122-127,153-154`
  - Fix : ajouter `gcSavedMemLimit` (package-level) ; capturer via `debug.SetMemoryLimit(-1)` (lecture seule)
    sous `gcGlobalMu`/`gcActiveDepth==0` ; restaurer la valeur sauvée au lieu de `MaxInt64`. Pas de champ par-contrôleur.
  - Vérif : étendre `TestGCController_ConcurrentBeginEnd_RestoresOriginal` (assert restauration du memlimit).
- [x] **T2.2 — Troncature au milieu d'une rune UTF-8** — `internal/errors/errors.go:113-121`
  - Fix : tronquer sur une frontière de rune (`utf8`). Vérif : test avec multibyte.
- [x] **T2.3 — `flagRegistry` désynchronisé + pas de garde** — `internal/cli/completion/registry.go`
  - Fix : ajouter les 6 flags manquants (`verbose`, `calculate`/`c`, `tui`, `last-digits`, `memory-limit`,
    `gc-control`) ; test de sync registre ↔ `flag.FlagSet` (allowlist `help`/`version`).
- [x] **T2.4 — Escaper zsh `_arguments` (`:` `[` `]`)** — `internal/cli/completion/zsh.go`, `escape.go`
  - Fix : escaper **dédié** au contexte `_arguments` (NE PAS replier dans `escapeZshSingleQuoted` partagé) ;
    tests adversariaux avec `:[]`.
- [x] **T2.5 — Échec total de construction → exit 0** — `internal/orchestration/orchestrator.go:145-148`
  - Fix : si 0 calculateur construit, retourner un code d'échec.
- [x] **T2.6 — `Close()` ignoré (golden generator)** — `cmd/generate-golden/main.go:44` ; vérifier l'erreur.
- [x] **T2.7 — Code exit config 4 contourné** — `cmd/fibcalc/main.go:33-39` ; router erreur config → `ExitErrorConfig`.
- [x] **T2.8 — Doc `ComputeLastDigits` fausse (ctx non transmis)** — `internal/orchestration/lastdigits.go:28-30`
  - Fix : corriger le commentaire (ou transmettre `ctx` si le calculateur le supporte) ; choix chirurgical = doc.

## 5. Tâches — Tier 3 : Purge code mort (housekeeper)

- [x] **T3.1** — branche négative inatteignable `FastDoublingMod` — `internal/fibonacci/modular.go:39-41`.
- [x] **T3.2** — 5 helpers `getEnv*` morts + sous-tests dédiés — `internal/config/env.go:22-78`.
- [x] **T3.3** — `TimeoutError`/`ValidationError` jamais construits — `internal/errors/errors.go:176-205`.
- [x] **T3.4** — `NTransform`/`InvNTransform` (+ `MulCached`/`SqrCached` non-bump) sans appelant — `internal/bigfft/fft_poly.go` *(vérifier l'absence d'appelant de test avant suppression)*.
- [x] **T3.5** — linkname `mulAddVWW` inutilisé — `internal/bigfft/arith_decl.go:57-60`.
- [x] **T3.6** — `NewMatrixFrameworkWithSquareFunc` inutilisé — `internal/fibonacci/matrix_framework.go:37-46`.
- [x] **T3.7** — `SetTaskLogger` non câblé — `internal/fibonacci/common.go:99-102` (suppression).
- [x] **T3.8** — `DynamicThresholdManager.SetLogger` non câblé — `internal/fibonacci/threshold/manager.go:172-175` *(fichier sensible ; re-tester threshold)*.
- [x] **T3.9** — suppression du sous-système `cpu_amd64.go` inutilisé (API locale sans appelant ; l'AVX2 réel vient de `x/sys/cpu`) — `internal/bigfft/cpu_amd64.go` + corriger `internal/bigfft/doc.go` *(vérifier l'absence de sibling non-amd64)*.

## 6. Tâches — Tier 4 : gmp + micro-optims (différés, périmètre agressif)

- [x] **T4.1 — `findHighestBit` → `bits.Len64`** — `internal/fibonacci/calculator_gmp.go:56-65,124`
  - ⚠ **Non vérifiable par compilation** (build tag `gmp`, libgmp absente). Changement mécanique + signalé non testé.
- [x] **T4.2 — Double snapshot dans `ShouldAdjust`** — `internal/fibonacci/threshold/manager.go:243-244,279-311`
  - Fix : un seul snapshot du buffer (lock + alloc uniques). *(Fichier sensible ; re-tester + benchstat si pertinent.)*
- [x] **T4.3 — Check `ctx` avant warm-up** — `internal/calibration/microbench.go:188-189` ; déplacer le premier check avant le warm-up multiply.
- [x] **T4.4 — Sous-warming `EnsurePoolsWarmed`** — `internal/bigfft/pool_warming.go:117-121`
  - Évaluer ; **ne corriger que si** benchstat ne montre pas de régression et le bénéfice est net, sinon documenter (KEEP).

## 7. Tâches — Tier 5 : Docs (docs sweep)

- [x] **T5.1** — `internal/bigfft/doc.go:11,19-21,28` : retirer les fausses claims SIMD/AVX2 + l'exemple `Mul` à 3 args.
- [x] **T5.2** — `internal/config/doc.go:26-29` : ajouter l'import réel `internal/ui` à la liste de dépendances.
- [x] **T5.3** — `internal/tui/model_test.go:771-781` : corriger le commentaire de formule obsolète.
- [x] **T5.4** — `docs/algorithms/BIGFFT.md:15` : `~4 100` → `~4 350` LOC.
- [x] **T5.5** — `docs/algorithms/MATRIX.md:3` : réconcilier le hash de commit dashboard (`2fca040`).
- [x] **T5.6** — `README.md` : **mettre à jour la bannière d'audit** pour refléter l'audit Opus 4.8 (2026-06-24) ;
  délier le plancher de couverture figé (95 %) vs directive A5-04 (pointer vers `make coverage-check`, plancher 80 %) ;
  vérifier/corriger l'URL du modèle.
- [x] **T5.7** — `CLAUDE.md` : refléter les suppressions de code mort et les nouveaux tests gardiens ; cohérence générale.

## 8. Tâches — Tier 6 : Renommage

- [x] **T6.1** — `Claude.md` → `CLAUDE.md` (casse canonique ; discoverabilité Claude Code sous FS sensible à la casse).
  - `git mv Claude.md tmp_claude && git mv tmp_claude CLAUDE.md`.

## 9. Items explicitement NON traités (KEEP / won't-fix justifiés)

- **SA6002 value-Put** (`pool.go`/`pool_warming.go`) — décision alloc-neutre mesurée (ADR-0007).
- **`internal/tui/component`** — stub documenté intentionnel (HK-06).
- **`RecoveredObserverCount`** (`progress/observer.go`) — compteur public diagnostique documenté (E2-R3).
- **`fibonacci_golden.json`** — oracle immuable (aucun `-update`).
- **Bannière README factuelle « data race corrigée / 1019 affirmations »** — historique daté (sauf maj demandée).

---

## 10. Gate final & livraison

1. `go build ./... && go vet ./... && go test ./... -count=1` verts.
2. `benchstat` pour T1.1 (et T4.2/T4.4 si retenus) vs `docs/audits/bench-baseline.txt` (< 5 %).
3. `golangci-lint run ./...` (advisory) ; `make coverage-check` (≥ 80 %).
4. `CHANGELOG.md` : entrée audit 2026-06-24.
5. Commits atomiques conventionnels (`fix`/`perf`/`refactor`/`docs`/`test`) → push `origin/main`.

**Note purge fichiers.** Aucun *fichier* parasite à supprimer (le `plan.md` historique l'a déjà été ;
`cpu_amd64.go` est traité en T3.9). La « purge » = retrait du **code mort** (Tier 3). `AuditPlan.md`
est conservé comme registre d'exécution (les commits + CHANGELOG sont la trace canonique).

---

## 11. Statut final d'exécution (2026-06-24)

**Exécuté et vérifié.** Gate final : `go build ./...` ✓ · `go vet ./...` ✓ · `go test ./... -count=1` **0 échec** ·
`gofmt -l` propre · couverture **95,0 %** (plancher 80 %) · `golangci-lint` : **aucune** nouvelle alerte
introduite par cet audit (les alertes restantes préexistent dans des fichiers non touchés). T1.1 (hot path)
validé par `benchstat` A/B sur `BenchmarkFFTMulWithBump` : cas 1M-mots **inchangé** (p=0,72), surcoût
alloc borné (~+15 allocs à 10K-mots, dû au `panicCh`) — **aucune régression** (< 5 %).

**Déviations assumées (« ose la contradiction » — supprimer une API publique testée et cohérente n'est pas
le sens d'« agressif ») :**

- **T3.3 — CONSERVÉ** : `TimeoutError`/`ValidationError` forment une taxonomie d'erreurs publique cohérente
  (aux côtés de `MemoryError`/`CalculationError`), avec tests `errors.As` dédiés. Supprimer 2 des 4 types
  testés briserait la symétrie et de la couverture réelle de la machinerie d'unwrap, pour un gain marginal.
- **T3.4 — CONSERVÉ** : `NTransform`/`InvNTransform`/`MulCached`/`SqrCached` ont des tests de round-trip
  dédiés (`fft_extra_test.go`, `misc_extra_test.go`) — API publique du *layer* polynômes FFT, pas du cruft.
- **T3.8 — CONSERVÉ** : `DynamicThresholdManager.SetLogger` — `manager.go` est explicitement **sensible**
  (table des invariants) ; retirer un setter non câblé pour un gain nul n'y justifie pas un diff. Cohérent
  avec la famille `SetXLogger`. (`SetTaskLogger` en `common.go`, hors fichier sensible, **a** été retiré — T3.7.)
- **T4.4 — DOCUMENTÉ (KEEP)** : `EnsurePoolsWarmed` reste one-shot ; re-warm en cours de process risquerait
  du thrashing de pool pour un bénéfice nul. Plafond désormais explicitement documenté.
- **T4.1** : changement mécanique appliqué ; **équivalence prouvée** (`findHighestBit` ≡ `bits.Len64`,
  contrat documenté du stdlib) mais **non compilable ici** (tag `gmp`, libgmp absente) — à valider sous
  `-tags gmp` en environnement libgmp.

**Codes de sortie modifiés (contrat documenté, désormais respecté) :** erreurs de configuration →
`ExitErrorConfig` (4) au lieu de 1 (T2.7) ; `--algo all --quiet` divergent → `ExitErrorMismatch` (3) au
lieu de 0 (T1.2). Tests unitaires et e2e du binaire mis à jour en conséquence.

**Nouveaux tests gardiens** : `TestFourierRecursiveAsyncPanicPropagates`/`...CtxAsyncPanicPropagates`,
`TestPresentQuietAllMismatchReturnsMismatch`/`...ConsistentPrintsValue`, `TestModel_Update_ErrorMsg_StaleGeneration`/
`...StaleDataMessagesDropped`, `TestGCController_RestoresMemoryLimit`, `TestSanitizeConfigExcerpt_RuneBoundary`,
`TestFlagRegistryInSyncWithConfig`, `TestEscapeZshArgSpec_NeutralisesArgMetachars`/`TestZshArgEntry_EscapesArgMetachars`,
`TestAnalyzeComparisonResults_EmptyResultsNeverSucceeds`, `TestRunSingleTestRespectsCancellationBeforeWarmup`.
