# promptAudit.md — Audit complet FibGo piloté par /loop

> **Comment lancer** — Dans Claude Code, à la racine du dépôt :
>
> ```
> /loop Lis promptAudit.md à la racine du dépôt et exécute-le. Reprends à la première phase incomplète selon AUDIT_STATE.md.
> ```
>
> Omettre l'intervalle : le mode auto-cadencé (`ScheduleWakeup`) laisse l'agent choisir le rythme
> entre les itérations. Chaque itération de la boucle traite **une phase (ou une sous-tâche de
> phase) jusqu'à un point vérifiable**, persiste l'état, puis re-planifie la prochaine itération.
> La boucle se termine d'elle-même quand la Phase 8 est complétée avec succès (ne plus appeler
> `ScheduleWakeup` à ce moment-là).

---

## 0. Contrat de la boucle (à respecter à CHAQUE itération)

1. **Lire l'état** : ouvrir `AUDIT_STATE.md` à la racine. S'il n'existe pas, le créer avec le
   gabarit de la section 9 (toutes les phases `PENDING`) — c'est l'itération 0.
2. **Choisir le travail** : prendre la première phase non-`DONE` dans l'ordre 1 → 8. Les phases
   sont **strictement séquentielles** : ne jamais entamer la phase N+1 si la phase N n'est pas
   `DONE`. Une phase trop grosse pour une itération se découpe en sous-tâches listées dans
   `AUDIT_STATE.md` (cases à cocher).
3. **Travailler** : exécuter la phase selon sa spécification (sections 1 à 8 ci-dessous), dans le
   respect absolu des garde-fous globaux (section G).
4. **Vérifier** : exécuter les critères de sortie de la phase (« Definition of Done »). Aucune
   phase ne passe `DONE` sans preuve d'exécution réelle des commandes de vérification (sortie de
   commande observée, pas supposée).
5. **Commiter la phase** : un ou plusieurs commits conventionnels **atomiques par phase**
   (`perf(...)`, `test(...)`, `docs(...)`, `chore(...)`). Ne pas attendre la Phase 8 pour
   commiter le travail intermédiaire — la Phase 8 est le commit/push **final** de clôture.
6. **Persister** : mettre à jour `AUDIT_STATE.md` (statut, sous-tâches cochées, métriques
   mesurées, blocages rencontrés, hash du dernier commit de la phase).
7. **Re-planifier ou terminer** : si des phases restent, appeler `ScheduleWakeup` avec le même
   prompt /loop ; choisir un délai court (60–270 s) si le travail continue immédiatement, long
   (1200 s+) seulement si on attend quelque chose d'externe. Si la Phase 8 est `DONE` :
   **ne pas re-planifier**, produire le rapport final (section 10) et s'arrêter.

**Blocage** : si une phase est bloquée par une décision que seul l'utilisateur peut prendre
(ex. push refusé par un garde-fou, suppression d'un fichier ambigu), marquer la phase `BLOCKED`
dans `AUDIT_STATE.md` avec la question précise, passer ce qui est passable **sans** violer la
séquentialité (c.-à-d. s'arrêter là), et terminer la boucle proprement avec un rapport de blocage.

---

## G. Garde-fous globaux (non négociables, toutes phases)

- **Golden tests immuables** : `internal/fibonacci/testdata/fibonacci_golden.json` ne doit
  JAMAIS être modifié (aucun `-update`). Tout changement algorithmique doit le faire passer
  tel quel. C'est l'oracle de non-régression — sa modification requiert un ADR explicite,
  hors périmètre de cet audit.
- **Invariants documentés** : avant TOUTE modification dans `internal/fibonacci/` ou
  `internal/bigfft/`, relire les deux tableaux « Invariants à préserver » et « Modules
  sensibles » de `Claude.md`. Chaque invariant cité (clearStateAliases inconditionnel, WithGC
  panic-safe + refcount GOGC, IsStale → CompleteStrategy, routage pool par `cap`, backing frais
  dans `putByKey`, atomics privés de `fft.go`, re-propagation des sentinels Fermat, knobs
  single-writer-before-use A2-04, étanchéité `errors↛format` / `tui↛fibonacci`) reste vrai
  après modification. Les tests gardiens nommés dans ces tableaux doivent passer.
- **Régression perf > 5 % = blocage** : tout diff touchant `internal/fibonacci/` ou
  `internal/bigfft/` exige benchmark avant/après comparé aux baselines `docs/audits/`.
  Régression > 5 % sur un benchmark significatif ⇒ revert du changement, pas de négociation.
- **Étanchéité des couches** : hiérarchie `cmd → app → orchestration → fibonacci/bigfft →
  config/errors` ; `internal/arch_test.go` doit passer. Pas de nouveaux globals dans `bigfft/`.
- **Validation locale obligatoire** (pas de CI distante) avant chaque commit :
  `go build ./... && go vet ./... && golangci-lint run ./... && go test ./... -count=1`.
  Le race detector (`-race`) exige CGO/gcc — indisponible sur cet hôte Windows sans gcc :
  noter dans `AUDIT_STATE.md` que la validation `-race` est déléguée à WSL/Linux, ne pas
  prétendre l'avoir exécutée si elle ne l'a pas été.
- **Particularités de cet hôte Windows** (mémoire de session, vérifiées) :
  - `go test -bench=.` est mal parsé en PowerShell ⇒ utiliser `-bench=Benchmark` (ou préfixe
    explicite) et un `-benchtime` constant entre avant/après.
  - Le dépôt suit le fichier sous le nom `Claude.md` (casse) : `git add CLAUDE.md` ne stage
    rien silencieusement ⇒ utiliser `git add Claude.md` ou `git add -A`.
  - `Get-Content | Measure-Object -Line` compte mal les lignes ⇒ utiliser `wc -l` (Bash) ou
    `(Get-Content f).Count`.
- **Modifications chirurgicales** : diff minimal, pas d'« améliorations » opportunistes hors
  périmètre de la phase courante. Bug actif découvert en passant ⇒ commit isolé `fix(scope):`
  AVANT de poursuivre (Directive 7 du projet).
- **Conventions** : pas d'emoji dans le code ; commentaires seulement pour le « pourquoi » ;
  complexité cyclomatique ≤ 15 ; fonctions ≤ 100 lignes ; tests `t.Parallel()` ; documentation
  du dépôt en **français** (cohérence avec l'existant).

---

## Phase 1 — Refactorisation et optimisation perf du cœur de calcul Fibonacci

**Périmètre** : `internal/fibonacci/` (fastdoubling, doubling_framework, fft, matrix, memory,
threshold), `internal/bigfft/` (fft, fft_recursion, fft_cache, pool, fermat),
`internal/calibration/`, `internal/orchestration/`.

**Démarche obligatoire** :
1. **Baseline d'abord** : capturer
   `go test -bench=Benchmark -benchmem -run='^$' -benchtime=2s -count=6 ./internal/fibonacci/ | tee bench_before.txt`
   (idem `./internal/bigfft/`). Comparer à la baseline la plus récente de `docs/audits/`
   (référence : `bench-parallel-pointwise-2026-06.md`) pour valider que l'environnement est sain.
2. **Lire le backlog avant de chercher** : `docs/adr/0004-*.md` (backlog assumé, dont migration
   `FFTContext` ADR-0007 et annulation FFT ADR-0006) et `docs/adr/0008-*.md` (**candidats déjà
   rejetés — ne pas les re-tenter**). Tout candidat d'optimisation doit être inédit ou
   explicitement listé comme « à faire » dans le backlog.
3. **Profiler avant d'optimiser** : `go test -bench=<cible> -cpuprofile/-memprofile` +
   `go tool pprof -top` sur F(1M)/F(10M). Ne modifier que ce que le profil désigne comme coûteux.
4. **Un candidat = un cycle complet** : hypothèse → modification minimale → golden + tests
   gardiens verts → bench après (`benchstat bench_before.txt bench_after.txt`, installer
   benchstat si absent) → garder seulement si gain mesurable et significatif (p < 0.05 sous
   benchstat), sinon revert. Itérer tant qu'il reste des candidats crédibles, s'arrêter quand
   deux candidats consécutifs ne donnent rien (loop-until-dry).
5. **Refactor lisibilité** (second temps, optionnel) : simplifications neutres en perf dans le
   périmètre (dédoublonnage, extraction de fonctions trop longues) — chaque refactor validé
   par bench neutre (± bruit) et suite verte.

**Definition of Done (Phase 1)** :
- [ ] `go test ./internal/fibonacci/... ./internal/bigfft/... -count=1` vert (golden inclus).
- [ ] Tests gardiens des invariants verts (`TestReleaseState_OverLimit_AliasesCleared`,
      `TestPointwiseWorkerPanicPropagates`, `TestPointwiseParallelMatchesSequential`,
      `TestFermatPostConditionPanicClassifier`, `TestWireThresholdTuning`,
      `TestArchitectureLayering`).
- [ ] `benchstat` avant/après archivé dans `docs/audits/bench-audit-loop-2026-06.md` (nouvelle
      baseline datée, gains et candidats rejetés documentés). Aucune régression > 5 %.
- [ ] `golangci-lint run ./internal/fibonacci/... ./internal/bigfft/...` propre.
- [ ] Commits `perf(scope):` / `refactor(scope):` poussés sur la branche de travail locale.

---

## Phase 2 — Couverture de tests ≥ 90 % sur la totalité du projet

**Mesure de référence** :
```
go test ./... -coverprofile=coverage.out -covermode=atomic -count=1
go tool cover -func=coverage.out | tail -1        # total
go tool cover -func=coverage.out | sort -k3 -n    # tri par fonction la moins couverte
```

**Démarche** :
1. Mesurer la couverture **par package**, lister dans `AUDIT_STATE.md` chaque package < 90 %
   avec son pourcentage (tableau trié croissant).
2. Traiter les packages du moins couvert au plus couvert. Pour chacun :
   - Couvrir d'abord les **chemins d'erreur** et branches non testées (sortie
     `go tool cover -html=coverage.out` pour voir les lignes rouges).
   - Tests table-driven, `t.Parallel()`, helpers de `internal/testutil/` réutilisés —
     **pas de tests vides ou tautologiques** écrits uniquement pour gonfler le chiffre :
     chaque test doit pouvoir échouer sur une vraie régression.
   - Utiliser `internal/fibonacci/fibonaccitest/` (doubles de CoreCalculator) plutôt que de
     nouveaux mocks ad hoc ; suivre `docs/TESTING.md` pour la génération de mocks.
3. **Refactoriser les tests existants** au passage dans les packages touchés : dédoublonner les
   fixtures, extraire les helpers répétés vers `internal/testutil/`, convertir en table-driven
   ce qui gagne en lisibilité — sans affaiblir aucune assertion existante.
4. Cas spéciaux à documenter (pas à forcer) : code dépendant de plateforme
   (`docs/PORTABILITY.md`), chemins TUI interactifs (Bubble Tea), build tag `gmp`. Si un package
   ne peut raisonnablement pas atteindre 90 %, le justifier dans `AUDIT_STATE.md` et compenser
   ailleurs pour tenir le **total ≥ 90 %**.

**Definition of Done (Phase 2)** :
- [ ] `go tool cover -func=coverage.out | tail -1` affiche **total ≥ 90.0 %** (copier la ligne
      de sortie dans `AUDIT_STATE.md` comme preuve).
- [ ] `go test ./... -count=1` vert ; lint propre ; aucune assertion existante affaiblie.
- [ ] Exceptions < 90 % par package justifiées par écrit.
- [ ] Commits `test(scope):` atomiques par package ou groupe cohérent.

---

## Phase 3 — Révision et restructuration de TOUTE la documentation

**Périmètre** : `docs/ARCH.md`, `docs/BUILD.md`, `docs/CALIBRATION.md`, `docs/PERFORMANCE.md`,
`docs/PORTABILITY.md`, `docs/TESTING.md`, `docs/TUI_GUIDE.md`, `docs/adr/`, `docs/algorithms/`,
`docs/architecture/`, `docs/audits/`, `docs/external-reviews/`, `CHANGELOG.md`,
`CONTRIBUTING.md`. (`README.md` et `Claude.md` ont leurs propres phases 5 et 4 ;
`docs/dashboard/` est généré — ne pas l'éditer à la main, c'est la Phase 6.)

**Démarche — fidélité au code avant style** :
1. Pour chaque document : extraire ses **affirmations vérifiables** (noms de fichiers/fonctions,
   commandes, seuils, chiffres de perf, diagrammes) et les confronter au code actuel
   (post-Phases 1–2, qui ont pu changer des choses). Corriger toute affirmation périmée ;
   marquer les chiffres de perf avec leur date de mesure.
2. **Exécuter réellement** chaque commande documentée (make targets, équivalents `go`, scripts
   `scripts/check.ps1` / `check.sh`) ; corriger ou signaler celles qui échouent.
3. Vérifier la cohérence des diagrammes Mermaid C4 de `docs/architecture/` avec l'arborescence
   réelle des packages ; régénérer/corriger si l'audit a déplacé des responsabilités.
4. ADR : ne **jamais réécrire** un ADR accepté (ils sont historiques) ; si l'audit a invalidé
   ou réalisé un point de backlog, ajouter une note de statut datée ou un nouvel ADR.
5. Mettre à jour `CHANGELOG.md` (format Keep-a-Changelog, section Unreleased) avec l'ensemble
   des changements des Phases 1–2.
6. Restructurer là où c'est utile : supprimer les redondances inter-documents (une seule source
   de vérité par sujet, les autres pointent dessus), tables des matières pour les documents
   > 150 lignes, liens relatifs valides (les vérifier).
7. Chaque package public doit avoir son `doc.go` à jour (convention projet) — vérifier par
   `Glob internal/*/doc.go` et combler les manques.

**Definition of Done (Phase 3)** :
- [ ] Zéro affirmation contredite par le code (revue document par document, consignée dans
      `AUDIT_STATE.md` : fichier → corrections apportées).
- [ ] Toutes les commandes documentées exécutées avec succès ou corrigées.
- [ ] Liens internes valides ; `CHANGELOG.md` à jour ; `doc.go` complets.
- [ ] Commits `docs(scope):`.

---

## Phase 4 — Refactorisation et optimisation de CLAUDE.md (fichier `Claude.md`)

**Attention casse** : le fichier suivi par git est `Claude.md` — utiliser ce nom pour `git add`.

**Démarche** :
1. Vérifier chaque ligne factuelle contre l'état post-Phases 1–3 : tableau des invariants
   (toujours vrais ? tests gardiens toujours nommés correctement ?), tableau des modules
   sensibles, chiffres (LOC, gains perf, seuils), commandes `make`, liste des ADR
   (0001 → dernier), arborescence des packages.
2. Intégrer les **nouveaux** invariants ou modules sensibles introduits par les Phases 1–2
   (toute correction subtile de l'audit qui casserait silencieusement sous un refactor naïf
   mérite sa ligne dans le tableau).
3. Optimiser pour l'usage réel d'un agent : supprimer ce qui est dérivable du code ou redondant
   avec `docs/` (pointer au lieu de dupliquer), garder les invariants, pièges et commandes ;
   viser un fichier plus court ou égal, jamais plus long sans justification.
4. Conserver la structure existante (sections, tableaux, français) — refactor, pas réécriture
   de style.

**Definition of Done (Phase 4)** :
- [ ] Chaque invariant listé pointe vers du code et des tests qui existent réellement
      (vérification grep/lecture, consignée).
- [ ] Aucune commande citée qui échoue ; aucune référence à un fichier supprimé.
- [ ] Commit `docs(claude-md):` (stagé via `git add Claude.md` ou `-A`).

---

## Phase 5 — Révision et restructuration complète de README.md

**Démarche** :
1. Restructurer pour un lecteur découvrant le dépôt : pitch (1 paragraphe), fonctionnalités,
   installation/prérequis (Go 1.26+, note CGO/race, build tag `gmp`), démarrage rapide
   (commandes réellement testées : build, premier calcul, TUI), algorithmes (tableau des 4 +
   complexités), architecture (résumé 4 couches + lien `docs/architecture/` et dashboard
   GitHub Pages <https://agbruneau.github.io/FibGo/dashboard/>), performance (chiffres datés
   sourcés de `docs/audits/`), tests/contribution (liens `docs/TESTING.md`, `CONTRIBUTING.md`),
   licence Apache 2.0.
2. **Tester chaque commande du README** sur cet hôte avant de l'inscrire (avec les équivalents
   sans-make pour Windows, comme le fait déjà `Claude.md`).
3. Aligner avec le contenu réel post-Phases 1–4 ; aucun chiffre de perf sans date ni source.
4. Français, cohérent avec le reste de la documentation ; pas de section marketing creuse.

**Definition of Done (Phase 5)** :
- [ ] Toutes les commandes du README exécutées avec succès sur cet hôte (ou marquées
      explicitement Linux/WSL).
- [ ] Tous les liens (relatifs + dashboard) valides.
- [ ] Commit `docs(readme):`.

---

## Phase 6 — Exécuter /understand : mise à jour et publication du Dashboard

**Démarche** :
1. Invoquer la commande `/understand` (skill/commande du projet — c'est elle qui régénère le
   knowledge-graph ; cf. commits antérieurs « understand purge », « docs: regenerate knowledge
   graph + dashboard »). Si `/understand` n'est pas disponible dans la session, suivre la
   procédure documentée `docs/BUILD.md#dashboard-statique-github-pages`
   (`pnpm --filter @understand-anything/dashboard build:demo` puis recopie dans
   `docs/dashboard/`) et le noter dans `AUDIT_STATE.md`.
2. Ne **jamais** éditer `docs/dashboard/` à la main (artefact généré).
3. Vérifier que le build généré reflète l'état post-audit (spot-check : nouveaux/renommés
   packages visibles dans le graph).
4. Commiter le dashboard régénéré : `docs(dashboard): regenerate knowledge graph post-audit`.
   La publication est effective au push sur `main` (GitHub Pages sert `docs/dashboard/`) —
   le push lui-même appartient à la Phase 8 ; noter ici que la publication sera vérifiée
   en ligne après le push final.

**Definition of Done (Phase 6)** :
- [ ] Dashboard régénéré par l'outil (pas d'édition manuelle), diff cohérent.
- [ ] Commit dédié créé.

---

## Phase 7 — Épuration des fichiers inutiles

**Démarche — prudence avant suppression** :
1. Construire la liste des candidats, chacun avec une **preuve d'inutilité** (aucune référence
   dans le code, les docs, le Makefile, les scripts, `.golangci.yml`, le devcontainer —
   vérifié par grep) :
   - `ruvector.db` à la racine (artefact d'outil probable — vérifier s'il est gitignoré/suivi) ;
   - `bench_before.txt` / `bench_after.txt` / `coverage.out` et autres artefacts produits par
     cet audit (à supprimer ou gitignorer, jamais à commiter) ;
   - fichiers orphelins dans `scripts/`, `test/e2e/`, `docs/external-reviews/` non référencés ;
   - code mort signalé par `golangci-lint` (unused) **uniquement** s'il est devenu mort du fait
     de cet audit — le code mort préexistant se signale dans le rapport, ne se supprime pas
     sans accord (règle « Surgical Changes »).
2. **Ne pas toucher** : `docs/audits/` (baselines de non-régression), `docs/adr/` (historique),
   golden files, `docs/dashboard/` (généré), `LICENSE`, fichiers de config outillage.
3. Pour chaque suppression : `git rm` (traçable), motif dans le message de commit. En cas de
   doute sur un fichier → ne pas supprimer, le lister dans le rapport final comme « candidat
   non tranché ».
4. Compléter `.gitignore` pour les artefacts récurrents constatés.

**Definition of Done (Phase 7)** :
- [ ] Chaque suppression justifiée par une preuve grep consignée dans `AUDIT_STATE.md`.
- [ ] `go build ./... && go test ./... -count=1` vert après suppressions.
- [ ] Commit(s) `chore(cleanup):`.

---

## Phase 8 — Validation finale, commit et push sur main

**Préconditions** : Phases 1–7 toutes `DONE` dans `AUDIT_STATE.md`.

**Démarche** :
1. **Gate de validation complet** (toutes les sorties observées, pas supposées) :
   ```
   go build ./...
   go vet ./...
   golangci-lint run ./...
   go test ./... -count=1
   go test ./... -coverprofile=coverage.out -count=1 && go tool cover -func=coverage.out | tail -1   # re-confirmer ≥ 90 %
   go test -bench=Benchmark -benchmem -run='^$' ./internal/fibonacci/                                 # sanité perf
   ```
   (`-race` : déléguée à WSL/Linux si gcc absent — l'indiquer dans le message de commit final.)
2. Supprimer/gitignorer les artefacts de travail (`coverage.out`, `bench_*.txt`,
   `AUDIT_STATE.md` : le supprimer ou le déplacer en `docs/audits/audit-loop-2026-06-state.md`
   selon sa valeur d'archive — préférer l'archive datée dans `docs/audits/`).
3. `git status` doit être propre après un dernier commit de clôture
   `chore(audit): close 2026-06 audit loop — perf, coverage ≥90%, docs, dashboard, cleanup`.
4. **Push sur main** : `git push origin main`. L'utilisateur autorise le git direct ; si un
   garde-fou de l'environnement refuse le push agent vers main, ne PAS le contourner :
   marquer la phase `BLOCKED`, laisser tous les commits locaux prêts, et demander à
   l'utilisateur d'exécuter `git push origin main` lui-même.
5. Après push réussi : vérifier que GitHub Pages republie le dashboard
   (<https://agbruneau.github.io/FibGo/dashboard/> — la propagation peut prendre quelques
   minutes ; un échec de propagation ne rouvre pas la phase, le noter simplement).

**Definition of Done (Phase 8)** :
- [ ] Gate complet vert, couverture totale ≥ 90 % re-confirmée, benchmarks sains.
- [ ] Arbre de travail propre, tout commité en messages conventionnels.
- [ ] `git push origin main` réussi (ou blocage documenté + instruction à l'utilisateur).
- [ ] Rapport final produit (section 10). **Fin de la boucle — ne pas re-planifier.**

---

## 9. Gabarit AUDIT_STATE.md (créé à l'itération 0)

```markdown
# AUDIT_STATE — boucle d'audit FibGo (démarrée AAAA-MM-JJ)
Dernière itération : <n> — <horodatage> — dernier commit : <hash>

| Phase | Intitulé                          | Statut  | Commit(s) | Preuves / métriques |
|-------|-----------------------------------|---------|-----------|---------------------|
| 1     | Perf cœur Fibonacci               | PENDING |           |                     |
| 2     | Couverture ≥ 90 %                 | PENDING |           |                     |
| 3     | Documentation fidèle              | PENDING |           |                     |
| 4     | Claude.md                         | PENDING |           |                     |
| 5     | README.md                         | PENDING |           |                     |
| 6     | /understand + dashboard           | PENDING |           |                     |
| 7     | Épuration                         | PENDING |           |                     |
| 8     | Validation finale + push main     | PENDING |           |                     |

## Sous-tâches de la phase en cours
- [ ] ...

## Mesures
- Couverture totale initiale : <x %> / courante : <y %>
- Baseline bench : bench_before.txt (<date>) ; benchstat : <résumé>

## Blocages / questions pour l'utilisateur
- (aucun)

## Candidats d'épuration non tranchés
- (aucun)
```

Statuts admis : `PENDING` → `IN_PROGRESS` → `DONE` (ou `BLOCKED`). Ne jamais rétrograder un
`DONE` ; si une phase ultérieure invalide une phase `DONE` (ex. la Phase 3 révèle un bug de
code), corriger via un commit `fix(scope):` ciblé sans rouvrir la phase.

---

## 10. Rapport final (au terme de la Phase 8)

Produire dans le dernier message de la boucle :
- Tableau des 8 phases avec commits associés et métriques clés (gains perf benchstat,
  couverture avant → après, nombre de documents corrigés, fichiers supprimés).
- Liste des décisions notables (candidats perf rejetés, exceptions de couverture justifiées,
  candidats d'épuration non tranchés).
- Confirmation du push sur main (hash) et statut de la publication du dashboard.
- Ce qui reste volontairement hors périmètre (backlog ADR-0004, migration FFTContext, etc.).
