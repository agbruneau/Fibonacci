# plan.md — Plan maître : exécution intégrale des 5 plans + nettoyage final

Ce document orchestre l'exécution **en totalité** des cinq plans, dans l'ordre, avec un point de contrôle bloquant après chacun, une validation globale finale, puis la **destruction des fichiers de plan** (les 5 `PLAN-*.md` et ce `plan.md`).

**Règle d'arrêt** : si un point de contrôle échoue, s'arrêter, corriger, re-vérifier. Ne jamais passer au plan suivant sur un contrôle rouge. Ne jamais sauter un plan.

**Règle git transversale** : les fichiers `PLAN-*.md` et `plan.md` sont des artefacts de travail **jamais committés**. Interdiction de `git add -A` / `git add .` tant qu'ils existent — chaque plan donne déjà son `git add <fichiers explicites>` ; s'y tenir.

---

## Phase 0 — Pré-vol (5 min)

1. `git status` → arbre propre (hors les 6 fichiers `*.md` de plan, non suivis).
2. `git log --oneline -1` → noter le SHA de départ.
3. `pwsh scripts/check.ps1` → **Overall: PASS**. Si rouge, corriger avant tout.
4. Vérifier la présence des 5 plans :
   `Get-ChildItem PLAN-*.md` → exactement 5 fichiers.

**Contrôle Phase 0** : gate vert + 5 plans présents. Sinon STOP.

---

## Phase 1 — Exécuter PLAN-1 (liens audit purgés) — ~15 min

Suivre `PLAN-1-purge-references-audit.md` étape par étape, sans en dévier.

**Contrôle Phase 1** (repris des critères d'acceptation du plan) :

- [ ] `Select-String` du plan → **zéro** lien `](audit.md)` / `](auditPlan.md)` hors `docs/dashboard/` et fichiers de plan.
- [ ] `git diff HEAD~1 --stat` → uniquement CLAUDE.md, README.md, CHANGELOG.md, docs/adr/0009-*.md.
- [ ] `pwsh scripts/check.ps1` → PASS.
- [ ] Commit `docs: repoint purged audit.md/auditPlan.md links...` poussé sur `main`.

---

## Phase 2 — Exécuter PLAN-2 (release v4.0.0) — ~30 min

**Dépendance** : Phase 1 terminée (ne pas figer des liens morts dans la release).

Suivre `PLAN-2-release-v4.md`. Rappels critiques : date du jour au format `AAAA-MM-JJ` ; aucune reconstitution rétroactive de `[3.0.0]`.

**Contrôle Phase 2** :

- [ ] `Select-String -Path CHANGELOG.md -Pattern '^## '` → 4 en-têtes dans l'ordre : `[Unreleased]`, `[4.0.0] - <date>`, `[1.0.0] - 2025-12-22`, `[0.1.0] - 2025-11-01`.
- [ ] `git diff HEAD~1 -- CHANGELOG.md` → insertions d'en-têtes/note uniquement, aucune suppression de contenu.
- [ ] `git ls-remote --tags origin` → `refs/tags/v4.0.0` présent.
- [ ] `pwsh scripts/check.ps1` → PASS.

---

## Phase 3 — Exécuter PLAN-3 (régénération PGO) — ~30 min + temps machine

Suivre `PLAN-3-regen-pgo.md`. Machine au calme pour la génération du profil (étape 1 du plan). Rappels critiques : jamais `-tags gmp` pour le profil ; ne pas toucher `docs/audits/bench-baseline.txt`.

**Contrôle Phase 3** :

- [ ] `git log -1 --format=%ad -- cmd/fibcalc/default.pgo` → date du jour.
- [ ] `go tool pprof -top -nodecount=25 cmd/fibcalc/default.pgo` → ≥ 1 symbole `internal/bigfft` **et** ≥ 1 `internal/fibonacci`.
- [ ] `make build-pgo` (WSL) → succès ; spot check F(10M) sans anomalie d'ordre de grandeur.
- [ ] `git status` → `docs/audits/bench-baseline.txt` **non modifié**.
- [ ] `pwsh scripts/check.ps1` → PASS ; commit poussé.

---

## Phase 4 — Exécuter PLAN-4 (gate GMP) — ~1 h

Suivre `PLAN-4-gate-gmp.md`. Rappels critiques : garde par headers dans un `if` (compatibilité `set -euo pipefail`) ; `check.ps1` intact ; jamais `-short` sur la passe gmp.

**Contrôle Phase 4** :

- [ ] `wsl -e bash -lc "dpkg -s libgmp-dev | grep Status"` → `install ok installed`.
- [ ] `wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && bash scripts/check.sh"` → contient `OK: gmp build tag` et `Overall: PASS`.
- [ ] `git diff HEAD~1 --stat` → uniquement scripts/check.sh, CLAUDE.md, docs/TESTING.md.
- [ ] `pwsh scripts/check.ps1` → PASS (inchangé) ; commit poussé.

---

## Phase 5 — Exécuter PLAN-5 (balayage multiplicateur d'arène) — ½–1 journée

**Dépendance pratique** : Phase 4 d'abord — `check.sh` couvre alors aussi le tag gmp lors de la validation finale du balayage. Machine au calme obligatoire (benchs).

Suivre `PLAN-5-balayage-arene.md` intégralement : branche `exp/arena-multiplier-sweep`, base ×15 fraîche, paliers m ∈ {12, 10, 8, 6}, règle de décision mécanique (Δ > +5 % = rejet ; +2–5 % = inversion d'ordre obligatoire), puis **Cas A** (adoption d'un palier) ou **Cas B** (×15 confirmé, addendum ADR seul).

**Les deux issues terminent la phase avec succès.** Ce qui la fait échouer : un balayage incomplet, un verdict sans benchstat archivé, ou un golden modifié.

**Contrôle Phase 5** :

- [ ] Fichiers `/tmp/sweep-m15.txt` + un par palier testé existent (WSL).
- [ ] Addendum ADR-0009 R4 committé avec les tableaux benchstat (Cas A comme Cas B).
- [ ] CLAUDE.md : la puce « Multiplicateur d'arène ×15 » reflète l'issue (nouvelle valeur, ou clôture datée du balayage).
- [ ] Cas A uniquement : `wsl go test -race ./...` vert, B/op réduit ≥ 15 % sur un bench 10M, `bench-baseline.txt` régénérée post-merge. Cas B uniquement : `git diff main -- internal/` vide avant le commit docs.
- [ ] Golden intact : `git log --oneline -- internal/fibonacci/testdata/` → aucun commit nouveau.
- [ ] Branche expérimentale supprimée ; commit(s) poussés sur `main`.

---

## Phase 6 — Validation globale finale

Sur `main` à jour, arbre propre (hors fichiers de plan) :

1. `pwsh scripts/check.ps1` → **Overall: PASS**.
2. `wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && bash scripts/check.sh"` → **Overall: PASS**, incluant `go test -race` complet **et** l'étape `OK: gmp build tag` (héritée de la Phase 4).
3. `git status` → seuls les 6 fichiers de plan restent non suivis ; rien d'autre en attente.
4. `git log --oneline <SHA de départ>..HEAD` → la liste des commits couvre bien les 5 plans (au moins : docs liens, release, pgo, gmp, ADR/perf arène).

**Contrôle Phase 6** : les 4 points ci-dessus verts. Sinon STOP — ne pas passer au nettoyage tant que la validation totale n'est pas constatée.

---

## Phase 7 — Destruction des fichiers de plan

**Précondition absolue** : Phase 6 intégralement verte. Les plans sont non suivis par git (vérifié en Phase 0) : leur suppression est une simple suppression disque, **aucun commit à faire**, et elle est irréversible — d'où la précondition.

```powershell
Remove-Item PLAN-1-purge-references-audit.md, PLAN-2-release-v4.md, PLAN-3-regen-pgo.md, PLAN-4-gate-gmp.md, PLAN-5-balayage-arene.md, plan.md -Confirm:$false
```

**Contrôle final** :

- [ ] `Get-ChildItem PLAN-*.md, plan.md -ErrorAction SilentlyContinue` → aucun fichier.
- [ ] `git status` → arbre propre, `main` synchronisée avec `origin` (`git status -sb` → pas de `ahead`).

Fin du plan maître.
