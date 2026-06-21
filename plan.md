# plan.md — Boucle d'audit globale FibGo

> **Nature** — Spécification *exécutable*. Conçue pour être lancée comme une **boucle
> globale** qui répète des passes sur les activités ci-dessous **jusqu'à ce que toutes
> soient `DONE` ou `BLOCKED`**. Chaque activité est elle-même une *loop* (Loop Library)
> avec condition d'arrêt vérifiable. Dérivé de l'analyse de priorité des loops sur ce repo.
>
> **Règle d'or du repo** — Toute modif dans `internal/fibonacci/` ou `internal/bigfft/`
> lit d'abord l'invariant documenté (`CLAUDE.md` § « Invariants à préserver »). Diff
> chirurgical. Pas de mutation du golden sans ADR. Pas de nouveaux globaux dans `bigfft/`.

---

## 0. Protocole de la boucle globale (`GLOBAL UNTIL DONE`)

```
état initial : toutes les activités = PENDING (voir Ledger § 2)

répéter pour passe p = 1, 2, 3, … :
    progrès = faux
    pour chaque activité A (ordre de priorité A1 → A8), si A.statut ∉ {DONE, BLOCKED} :
        1. lire les invariants en tête de A (si présents)
        2. exécuter UNE itération du corps de boucle de A
        3. exécuter A.vérification
        4. décision :
             - critère de succès atteint ........... A.statut = DONE ;        progrès = vrai
             - changements produits cette itération . A.statut = IN_PROGRESS ; progrès = vrai ; A.dry = 0
             - aucun changement & A ouverte & A.dry ≥ K(A) . A.statut = DONE (convergée à sec)
             - aucun changement & A.dry < K(A) ..... A.dry += 1
             - condition BLOCKED rencontrée ......... A.statut = BLOCKED ; consigner la raison
        5. mettre le Ledger (§ 2) à jour
    arrêt global SI : toutes les activités ∈ {DONE, BLOCKED}
    arrêt global SI : progrès = faux sur 2 passes consécutives  → escalader au mainteneur
```

**Condition de fin globale (succès)** : toutes les activités `DONE`, **et** le *gate final*
(§ 3) passe. S'il reste des `BLOCKED`, rapporter au mainteneur avec la raison — ne pas
forcer.

**Garde anti-boucle infinie** : chaque activité ouverte (audit/housekeeping) a un compteur
`dry` ; après `K` passes consécutives sans nouvelle trouvaille, elle est déclarée convergée.
`K = 1` sauf indication contraire.

---

## 1. Commandes de vérification (Windows-natif d'abord)

> `make` ici est majoritairement POSIX-only (bash/find/wc/awk/date). Sur cet hôte Windows,
> primaire = `go`/PowerShell ; les cibles POSIX passent par WSL ou git-bash.

| But | Commande primaire (Windows) | Équivalent / note |
|---|---|---|
| Gate dur complet | `pwsh scripts/check.ps1` | build → vet → test → lint(advisory) → couverture ≥ 80 % |
| Build | `go build ./...` | |
| Vet | `go vet ./...` | |
| Tests (sans race) | `go test ./...` | `make test-win` |
| Tests `-race` | `wsl go test -race ./...` | CGO requis ; **WSL only** (pas de gcc Windows) |
| Étanchéité couches | `go test ./internal -run TestArchitectureLayering` | 3 arrows remontants interdits |
| Golden non-régression | `go test ./internal/fibonacci -run Golden` | fichier **immuable** sans ADR |
| Couverture + total | `go test -coverprofile coverage.out ./...` puis `go tool cover -func coverage.out` | plancher 80 %, cible ≥ 90 % |
| Lint | `golangci-lint run ./...` | advisory (n'échoue pas le gate) |
| Bench (3 algos) | `go test -bench='BenchmarkFibonacci/(FastDoubling\|MatrixExp\|FFTBased)' -benchmem -run='^$' -count=5 ./internal/fibonacci/` | PowerShell mé-parse `-bench=.` → garder le regex explicite |
| Baseline perf | `wsl make bench-baseline` | écrit `docs/audits/bench-baseline.txt` (POSIX-only) |
| Comparaison perf | `benchstat docs/audits/bench-baseline.txt new.txt` | **régression > 5 % = blocage** |
| Décompte pkg/LOC | `wsl make stats` | source canonique des chiffres (jamais coder en dur) |

---

## 2. Ledger d'état (mis à jour à chaque passe)

| ID | Activité (loop) | Priorité | Statut | dry | Dernière vérif |
|----|-----------------|:--------:|:------:|:---:|----------------|
| A0 | Baseline gate (Post-release baseline) | 0 | PENDING | — | — |
| A1 | Groundtruth audit (cartographie code-evidence) | 1 | PENDING | 0 | — |
| A2 | Champion / baseline perf gate | 2 | PENDING | — | — |
| A3 | Adversarial-review hot-path (Clodex / multi-LLM) | 3 | PENDING | — | — |
| A4 | Architecture satisfaction (refactor checkpointé) | 4 | PENDING | 0 | — |
| A5 | Docs sweep + Propagation compliance | 5 | PENDING | 0 | — |
| A6 | Test stabilizer (flaky `-race`) | 6 | PENDING | 0 | — |
| A7 | Housekeeper (code mort, ponytail-audit) | 7 | PENDING | 0 | — |
| A8 | Security-review `cli/completion` (échappement shell) | 8 | PENDING | — | — |

Statuts : `PENDING` · `IN_PROGRESS` · `DONE` · `BLOCKED`.

---

## 3. Gate final (condition de fin globale — succès)

Toutes vraies, sinon la boucle globale continue :

- [ ] `pwsh scripts/check.ps1` → **PASS** (build/vet/test/couverture ≥ 80 %)
- [ ] `go test ./internal -run TestArchitectureLayering` → PASS
- [ ] `go test ./internal/fibonacci -run Golden` → PASS (golden intact)
- [ ] `wsl go test -race ./...` → PASS (ou A6 = `BLOCKED: WSL indisponible`, consigné)
- [ ] `benchstat` baseline vs final → **aucune** régression > 5 % sur FastDoubling / MatrixExp / FFTBased
- [ ] Couverture totale ≥ 90 % (cible projet ; plancher dur 80 %)
- [ ] Aucune activité ≠ `DONE` sans raison `BLOCKED` consignée

---

## 4. Activités (chaque bloc = une loop avec condition d'arrêt)

### A0 — Baseline gate  *(Post-release baseline loop)*
- **But** : établir un point de départ **vert** et mesuré avant toute modif.
- **Corps** : `pwsh scripts/check.ps1` ; capturer `wsl make bench-baseline` → `docs/audits/bench-baseline.txt` ; `wsl make stats` → noter les chiffres courants.
- **Succès (arrêt)** : `check.ps1` exit 0 **et** baseline perf écrite **et** stats capturées.
- **BLOCKED** : gate déjà rouge à l'état initial → escalader (on n'audite pas sur une base cassée).

### A1 — Groundtruth audit  *(Groundtruth audit-loop)*  — **lecture seule**
- **But** : produire/rafraîchir un backlog de trouvailles **depuis le code**, pas la mémoire.
- **Corps** : balayer `internal/**` ; classer chaque trouvaille en `{bloat, bug, perf, docs-drift, flaky, invariant-risk, security}` ; écrire `docs/audits/groundtruth-2026-06-21.md` (chemin de fichier + invariant concerné + raison). Lire les invariants avant de juger un hot path « suspect ».
- **Succès (arrêt)** : backlog écrit ; **convergence** quand une passe fraîche ajoute 0 trouvaille (`K=1`).
- **Aiguillage** : chaque trouvaille alimente A4 (refactor/fix), A5 (docs), A6 (flaky), A7 (bloat), A8 (sécu).
- **Garde** : ne modifie **aucun** code. Un *bug actif* croisé ici part en commit `fix(scope):` **isolé** (CLAUDE.md § Directives #7), avant tout refactor.

### A2 — Champion / baseline perf gate  *(Self-improving champion loop)*
- **But** : ne « promouvoir » un changement que s'il **ne régresse pas** la perf.
- **Corps** : après chaque lot de A4/A7 touchant `fibonacci/`|`bigfft/`, relancer le bench (§ 1) → `new.txt` ; `benchstat baseline new`.
- **Succès (arrêt)** : `benchstat` < 5 % sur les 3 algos. Sinon → rollback du lot fautif (challenger perd).
- **BLOCKED** : `benchstat` absent / machine bruyante → consigner, exiger une mesure manuelle propre avant promotion.

### A3 — Adversarial-review hot-path  *(Clodex / Multi-LLM convergence)*
- **But** : contrer le piège n°1 du repo (« une régression naïve casse un invariant sans faire échouer un test trivial »).
- **Corps** : toute trouvaille A1 touchant `internal/fibonacci/` ou `internal/bigfft/` est soumise à **≥ 2 agents réfutateurs indépendants** (lenses distinctes : invariant, concurrence/`-race`, perf) **avant** tout commit du fix. Défaut = *réfuté* si doute.
- **Succès (arrêt)** : plus aucune trouvaille hot-path non revue ; chaque fix retenu a ≥ 2 confirmations, sinon rejeté/redéfini.
- **Exécution** : phase déléguée à des sous-agents (`dispatching-parallel-agents` / Workflow) — c'est là que le fan-out a de la valeur.

### A4 — Architecture satisfaction  *(Architecture satisfaction loop)*
- **Invariants à lire d'abord** : `fastdoubling.go`, `doubling_framework.go`, `pool.go`, `fft_cache.go`, `manager.go` (selon la trouvaille).
- **Corps** : appliquer les fix/refactors validés (A3) **en checkpoints testés** ; diff minimal ; un refactor > 50 LOC / > 2 fichiers se **justifie en message de commit** (raison, compromis, alternative écartée).
- **Vérif** : `go build ./...` + `go test ./...` + `TestArchitectureLayering` + golden, après **chaque** checkpoint ; puis A2.
- **Succès (arrêt)** : toutes les trouvailles en scope traitées **ou** différées-avec-raison ; étanchéité + golden + perf verts.
- **BLOCKED** : un fix exige de toucher le golden → `BLOCKED: requiert ADR` (CLAUDE.md § Directives #2).

### A5 — Docs sweep + Propagation compliance  *(Docs sweep + Propagation compliance loop)*
- **But** : docs/README/ADR alignés au code ; **zéro chiffre daté codé en dur**.
- **Corps** : remplacer tout décompte figé par renvoi à `make stats` ; propager les valeurs changées (seuils, ADR) dans toutes les références ; valider les liens internes.
- **Succès (arrêt)** : aucune référence numérique périmée détectée ; convergence `K=1`. Ne touche pas aux artefacts générés (`docs/dashboard/`).

### A6 — Test stabilizer  *(Test stabilizer loop)*
- **But** : éliminer le flaky de concurrence (pools/atomics/sémaphores).
- **Corps** : `wsl go test -race -count=10 ./...` ; sur échec → diagnostic racine (skill `systematic-debugging`), fix, re-run.
- **Succès (arrêt)** : 10 runs `-race` verts d'affilée.
- **BLOCKED** : `BLOCKED: WSL/CGO indisponible` (consigné ; le gate final accepte ce BLOCKED documenté).

### A7 — Housekeeper  *(Housekeeper loop)*
- **But** : retirer code mort / fichiers périmés, **low-risk uniquement**.
- **Corps** : passe `/ponytail-audit` ; ne supprimer **que** les orphelins créés par les modifs de cette boucle (jamais le code mort préexistant sans signalement) ; `go build` + `go test` après chaque suppression.
- **Succès (arrêt)** : `/ponytail-audit` ne trouve plus rien de nouveau (`K=1`) ; gate vert.

### A8 — Security-review `cli/completion`  *(cible unique)*
- **But** : valider l'échappement des identifiants vers le shell (risque latent documenté).
- **Corps** : revoir les 4 générateurs shell de `internal/cli/completion/` ; `gosec ./internal/cli/completion/...` si dispo ; cas-test d'injection d'identifiant.
- **Succès (arrêt)** : gosec propre sur le package **et** revue manuelle d'échappement consignée.

---

## 5. Invariants & garde-fous transverses (s'appliquent à chaque itération)

1. **Lire l'invariant avant de toucher** `fibonacci/` ou `bigfft/` (table CLAUDE.md).
2. **Golden immuable** sans ADR — aucun `-update`.
3. **Pas de nouveaux globaux** dans `bigfft/` (les 3 existants sont `atomic.*` privés).
4. **Diff chirurgical** : chaque ligne se rattache à une trouvaille du backlog A1.
5. **Bug avant refactor** : défaut actif → commit `fix(scope):` isolé d'abord.
6. **Perf-gated** : régression `benchstat` > 5 % = rollback (A2), pas un compromis.
7. **Commits conventionnels** : `feat/fix/docs/perf/test/refactor(scope): …`.
8. **Pas de fichier `progress*` neuf** sans consultation (CLAUDE.md § Directives #6).

---

## 6. Journal d'exécution (rempli par la boucle)

| Passe | Activité | Itération | Action | Vérif | Résultat |
|:-----:|----------|:---------:|--------|-------|----------|
| — | — | — | *(à remplir au lancement)* | — | — |
