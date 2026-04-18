# PRD — Audit Exhaustif & Optimisation FibGo (Mode Agent Teams)

**Projet** : FibGo / FibCalc — Calculateur Fibonacci haute performance en Go
**Date** : 2026-04-18
**Mode d'exécution** : Claude Code — Agent Teams (délégation parallèle via sous-agents)
**Branche cible** : `claude/audit-exhaustif-YYYYMMDD`
**Propriétaire** : agbruneau@gmail.com

---

## 1. Contexte & Objectifs

FibGo est un prototype académique mature (~31 900 lignes Go, 103 fichiers source, 89 fichiers de test) démontrant Clean Architecture, pooling mémoire, parallélisme adaptatif et PGO. Le codebase a déjà fait l'objet de multiples passes d'optimisation (cf. `git log`). **L'objectif de cette mission n'est pas de refactorer agressivement, mais d'identifier de façon rigoureuse et priorisée les gains résiduels**, de combler les lacunes documentaires et de valider la cohérence architecturale de bout en bout.

### Objectifs mesurables

1. **Performance** — Identifier ≥ 5 optimisations chiffrées (allocation/ns/op, réduction GC, throughput) avec benchmark `before/after`.
2. **Refactorisation** — Proposer ≤ 10 refactorisations chirurgicales à ROI élevé (complexité cyclomatique, duplications, cohésion). Aucune réécriture large.
3. **Documentation** — Auditer 100 % du Markdown (`README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `docs/**`, `doc.go` de chaque package).
4. **Qualité** — Rapport `make lint`, `make test -race`, `make coverage` → identifier tout `FIXME`, `TODO`, code mort, fuite de couche `internal/`.
5. **Livrable final** — Un rapport unifié `AUDIT_REPORT.md` priorisé (P0/P1/P2) avec estimation effort et impact.

### Hors-scope

- Réécriture d'algorithmes existants (Fast Doubling, Strassen, FFT Schönhage-Strassen) sans demande explicite.
- Ajout de nouvelles fonctionnalités utilisateur.
- Changement de licence, de module path ou de compatibilité Go (reste sur Go 1.25.0+).
- Modifications chirurgicales uniquement — directive `CLAUDE.md` #7.

---

## 2. Contraintes & Invariants

Tirés de `CLAUDE.md` (à respecter strictement) :

- **Performance critique** : aucune allocation inutile, respecter `sync.Pool`, allocateur bump FFT.
- **Golden tests** : tout changement algorithmique doit passer `testdata/fibonacci_golden.json`.
- **Architecture** : aucune fuite `internal/` → `cmd/`.
- **Lint** : `make lint` (22 linters), complexité cyclomatique ≤ 15, cognitive ≤ 30, fonction ≤ 100 lignes / 50 statements.
- **Concurrence** : `errgroup`, sémaphores bornés, pas de goroutine sans cycle de vie contrôlé.
- **Race detector** : activé systématiquement.
- **doc.go** : un par package, à jour.

---

## 3. Organisation en Agent Teams

L'audit est scindé en **6 équipes d'agents parallèles**, chacune produisant un rapport partiel. Un agent **Coordinateur** consolide en `AUDIT_REPORT.md`.

### Team A — Performance & Profiling
**Sous-agent** : `general-purpose` (ou `Explore` en medium/thorough)
**Périmètre** : `internal/fibonacci/`, `internal/bigfft/`, `internal/parallel/`, `internal/orchestration/`
**Missions** :
- Exécuter `make benchmark` et archiver la baseline dans `bench/baseline.txt`.
- Identifier les hotspots via `go test -bench=. -benchmem -cpuprofile -memprofile`.
- Détecter les allocations cachées (escape analysis : `go build -gcflags="-m=2"`).
- Vérifier l'usage correct de `sync.Pool` (fuites, resets manqués).
- Valider l'allocateur bump FFT (fragmentation, seuils).
- Examiner le parallélisme adaptatif (`NumCPU()*2`) — contention, false sharing.
- Vérifier l'impact PGO actuel (`make build-pgo` vs build standard).
**Livrable** : `bench/TEAM_A_PERFORMANCE.md` avec ≥ 5 findings chiffrés + patchs proposés (diff).

### Team B — Architecture & Refactorisation
**Sous-agent** : `Plan` ou `general-purpose`
**Périmètre** : toute la structure `internal/` + `cmd/`
**Missions** :
- Cartographier les dépendances inter-packages (`go list -deps`, `go mod graph`).
- Détecter les violations Clean Architecture (fuites de couches).
- Mesurer cohésion (LCOM) et couplage afférent/efférent par package.
- Repérer duplications (`dupl`, `gocognit`) et code mort (`deadcode`, `unused`).
- Valider ISP sur `Multiplier`, `DoublingStepExecutor` et autres interfaces.
- Proposer ≤ 10 refactorisations chirurgicales (pas de big-bang).
**Livrable** : `bench/TEAM_B_ARCHITECTURE.md` avec schéma de dépendances + liste priorisée.

### Team C — Tests & Qualité
**Sous-agent** : `general-purpose`
**Périmètre** : `*_test.go`, `test/e2e/`, `testdata/`, `internal/testutil/`
**Missions** :
- Exécuter `make test`, `make coverage`, `make test-short`.
- Couverture par package : cible ≥ 80 %, signaler les packages < 70 %.
- Vérifier `t.Parallel()` systématique.
- Détecter tests fragiles (flaky), `t.Skip` non justifiés, assertions faibles.
- Valider golden tests (`testdata/fibonacci_golden.json`) — intégrité, exhaustivité.
- Fuzzing : recenser les cibles `FuzzXxx`, proposer ajouts si absent sur parsers/configs.
**Livrable** : `bench/TEAM_C_TESTS.md` avec matrice couverture + recommandations.

### Team D — Documentation
**Sous-agent** : `general-purpose`
**Périmètre** : `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `Claude.md`, `docs/**`, tous `doc.go`.
**Missions** :
- Vérifier que `README.md` reflète l'état actuel (commandes, flags, algorithmes, benchmarks).
- Détecter références obsolètes (commandes renommées, flags supprimés, chemins disparus).
- Valider cohérence entre `docs/architecture/*.md` (diagrammes C4) et le code.
- Valider `docs/algorithms/*.md` vs implémentations réelles.
- Auditer chaque `doc.go` : présent, à jour, idiomatique.
- Vérifier `CHANGELOG.md` Keep-a-Changelog, `CONTRIBUTING.md` à jour sur le workflow.
- Repérer les sections du README à condenser (38 Ko — possiblement trop long).
**Livrable** : `bench/TEAM_D_DOCS.md` avec checklist par fichier + diffs proposés.

### Team E — Sécurité & Robustesse
**Sous-agent** : `general-purpose`
**Périmètre** : parsing config, entrées CLI, I/O fichier, goroutines, erreurs.
**Missions** :
- Exécuter `govulncheck ./...`, `gosec` (déjà dans `.golangci.yml` ?).
- Valider la gestion d'erreurs : `errors.Is`/`As`, pas de `err == nil` incorrect, wrapping correct.
- Vérifier les types d'erreurs structurés (`ConfigError`, `CalcError`).
- Auditer les points d'entrée : parsing CLI, lecture fichiers, interactions shell.
- Recenser les `panic()` — justifiés ou à remplacer par erreur.
- Vérifier cycle de vie goroutines (context.Context propagation, leaks).
- Analyser les fichiers résiduels (`build_err.txt`, `e2e_rich_out.txt`, `test_err.txt`, `test_out.txt`) — à nettoyer/gitignore ?
**Livrable** : `bench/TEAM_E_SECURITY.md` avec findings et patchs.

### Team F — Build, CI/CD & Outillage
**Sous-agent** : `general-purpose`
**Périmètre** : `Makefile`, `.github/workflows/**`, `.golangci.yml`, `.env.example`, `go.mod`/`go.sum`.
**Missions** :
- Auditer chaque cible Makefile — doublons, cibles obsolètes, portabilité Windows/Linux/macOS.
- Vérifier workflows GitHub Actions : versions actions, cache Go modules, matrix builds.
- Valider `.golangci.yml` : linters activés pertinents, seuils cohérents avec CLAUDE.md.
- Scanner `go.mod` : dépendances obsolètes (`go list -u -m all`), version Go, directives `toolchain`.
- Vérifier build PGO, cross-compilation (`build-all`), reproductibilité.
- Artefacts résiduels (`build_err.txt`, `test_err.txt`, `test_out.txt`, `e2e_rich_out.txt`) : à supprimer ou `.gitignore`.
**Livrable** : `bench/TEAM_F_TOOLING.md`.

### Coordinateur — Consolidation
**Sous-agent** : exécution dans la conversation principale (pas de délégation).
**Mission** : Lire les 6 rapports, dédupliquer, prioriser P0/P1/P2, produire **`AUDIT_REPORT.md`** final en racine avec :
1. Résumé exécutif (≤ 1 page).
2. Tableau consolidé des findings (catégorie, sévérité, effort, impact).
3. Roadmap proposée (3 sprints ou 3 phases).
4. Annexes : liens vers les 6 rapports partiels.

---

## 4. Méthodologie d'exécution

### Phase 1 — Préparation (Coordinateur, séquentiel)
1. Créer branche `claude/audit-exhaustif-YYYYMMDD`.
2. Créer dossier `bench/` pour les rapports partiels.
3. Capturer baseline : `make test`, `make coverage`, `make benchmark`, `make lint` → archiver sorties dans `bench/baseline/`.

### Phase 2 — Audit parallèle (6 teams, parallèle)
Lancer les 6 agents **en un seul message** avec plusieurs invocations `Agent` pour exploitation parallèle. Chaque agent reçoit :
- Son périmètre précis (liste de packages/fichiers).
- Le lien vers `bench/baseline/` pour comparaison.
- Instruction de produire **uniquement** son rapport `bench/TEAM_X_*.md` — **ne pas modifier le code** à ce stade.
- Limite de longueur : rapport ≤ 800 lignes, findings structurés.

### Phase 3 — Consolidation (Coordinateur, séquentiel)
1. Lire les 6 rapports.
2. Produire `AUDIT_REPORT.md` priorisé.
3. Présenter le plan à l'utilisateur pour validation avant toute modification de code.

### Phase 4 — Exécution (sur validation explicite)
- Créer un sous-agent par finding P0, en isolation `worktree` pour les changements risqués.
- Chaque modification → PR séparée, revue par l'agent `review` avant merge.
- Benchmarks `before/after` obligatoires pour les findings perf.

---

## 5. Critères d'acceptation

- [ ] `bench/baseline/` contient sorties `test`, `coverage`, `benchmark`, `lint` horodatées.
- [ ] 6 rapports partiels `bench/TEAM_*.md` présents et complets.
- [ ] `AUDIT_REPORT.md` en racine, ≤ 500 lignes, priorisé P0/P1/P2.
- [ ] Aucune modification de code source à ce stade — audit read-only.
- [ ] Toutes les commandes exécutées sont reproductibles (notées dans le rapport).
- [ ] Zéro régression sur `make test`, `make lint` après d'éventuels changements cosmétiques (formatage, typos doc).

---

## 6. Prompt de lancement (à coller dans Claude Code)

```
Exécute le PRD.md en mode Agent Teams.

1. Lis PRD.md intégralement.
2. Phase 1 : capture la baseline dans bench/baseline/ (test, coverage, benchmark, lint).
3. Phase 2 : lance les 6 teams EN PARALLÈLE dans un seul message multi-Agent :
   - Team A (Performance) → general-purpose
   - Team B (Architecture) → Plan
   - Team C (Tests) → general-purpose
   - Team D (Documentation) → general-purpose
   - Team E (Sécurité) → general-purpose
   - Team F (Tooling) → general-purpose
   Chaque agent doit produire UNIQUEMENT son rapport bench/TEAM_X_*.md, sans modifier le code source.
4. Phase 3 : consolide les 6 rapports dans AUDIT_REPORT.md (racine), priorisé P0/P1/P2.
5. Présente-moi AUDIT_REPORT.md et attends validation avant toute modification de code.

Respecte strictement CLAUDE.md (modifications chirurgicales, pas de refactor sans demande).
```

---

## 7. Risques & Mitigations

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| Agents produisent du bruit (findings triviaux) | Moyenne | Faible | Prompts exigent chiffres, benchmarks, preuves |
| Divergence entre rapports (doublons) | Élevée | Moyen | Coordinateur dédoublonne explicitement en Phase 3 |
| Modification code pendant l'audit | Faible | Élevé | Audit read-only en Phase 2, modifications uniquement Phase 4 |
| Dépassement contexte agent | Moyenne | Moyen | Périmètre restreint par team, rapport ≤ 800 lignes |
| Benchmarks non reproductibles | Moyenne | Moyen | Commandes notées, warm-up, `benchstat` obligatoire |

---

## 8. Annexes

- [CLAUDE.md](CLAUDE.md) — directives projet
- [Makefile](Makefile) — cibles de build/test
- [.golangci.yml](.golangci.yml) — config lint
- [docs/architecture/](docs/architecture/) — diagrammes C4
- [docs/algorithms/](docs/algorithms/) — doc mathématique
