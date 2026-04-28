# PLAN — Analyse & Audit du dépôt FibGo

> Plan de travail pour produire un audit complet du dépôt `github.com/agbru/fibcalc`
> (FibGo / FibCalc — calculateur Fibonacci haute performance, ~37 000 LOC Go, 22 packages).

---

## 0. Tableau de suivi

| #    | Phase                                  | Agent / Skill                | Statut  | Livrable                                     |
| ---- | -------------------------------------- | ---------------------------- | ------- | -------------------------------------------- |
| 1.1  | Cartographie du dépôt                  | `general-purpose`            | ✅ DONE | `docs/audits/01-inventory.md`                |
| 1.2  | Inventaire des dépendances             | `general-purpose`            | ✅ DONE | `docs/audits/02-dependencies.md`             |
| 1.3  | Snapshot Git & historique récent       | `general-purpose`            | ✅ DONE | `docs/audits/03-git-snapshot.md`             |
| 2.1  | Audit architecture (Clean Arch)        | `general-purpose`            | ✅ DONE | `docs/audits/10-architecture.md`             |
| 2.2  | Étanchéité des couches `internal/`     | `general-purpose`            | ✅ DONE | `docs/audits/11-layering.md`                 |
| 2.3  | Cohérence des interfaces (ISP)         | `general-purpose`            | ✅ DONE | `docs/audits/12-interfaces.md`               |
| 3.1  | Revue qualité code (cyclo, longueurs)  | `general-purpose`            | ✅ DONE | `docs/audits/20-code-quality.md`             |
| 3.2  | Détection code mort / duplication      | `general-purpose`            | ✅ DONE | `docs/audits/21-dead-code.md`                |
| 3.3  | Conformité conventions FibGo           | `general-purpose`            | ✅ DONE | `docs/audits/22-conventions.md`              |
| 4.1  | Revue sécurité (OWASP, supply-chain)   | `general-purpose`            | ✅ DONE | `docs/audits/30-security.md`                 |
| 4.2  | Audit licences & SBOM                  | `general-purpose`            | ✅ DONE | `docs/audits/31-licenses.md`                 |
| 5.1  | Audit `internal/fibonacci/` (perf)     | `general-purpose`            | ✅ DONE | `docs/audits/40-fibonacci-perf.md`           |
| 5.2  | Audit `internal/bigfft/` & arènes      | `general-purpose`            | ✅ DONE | `docs/audits/41-bigfft-arena.md`             |
| 5.3  | Concurrence (sync.Pool, errgroup, sem) | `general-purpose`            | ✅ DONE | `docs/audits/42-concurrency.md`              |
| 5.4  | Benchmarks & PGO baseline              | `general-purpose` (go bench) | ✅ DONE | `docs/audits/43-bench-baseline.md`           |
| 6.1  | Couverture & golden tests              | `general-purpose`            | ✅ DONE | `docs/audits/50-tests-coverage.md`           |
| 6.2  | Race detector & tests parallèles       | `general-purpose` (CGO N/A)  | ✅ DONE | `docs/audits/51-race-detector.md`            |
| 7.1  | Documentation utilisateur (README/doc) | `general-purpose`            | ✅ DONE | `docs/audits/60-doc-user.md`                 |
| 7.2  | Documentation technique (`docs/`)      | `general-purpose`            | ✅ DONE | `docs/audits/61-doc-tech.md`                 |
| 7.3  | `doc.go` par package public            | `general-purpose`            | ✅ DONE | `docs/audits/62-doc-go.md`                   |
| 8.1  | Audit `Makefile` & cibles CI           | `general-purpose`            | ✅ DONE | `docs/audits/70-makefile.md`                 |
| 8.2  | Lint config (`.golangci.yml`, 24 lint) | `general-purpose` (lint log) | ✅ DONE | `docs/audits/71-lint.md`                     |
| 8.3  | Cross-compilation (build-all)          | `general-purpose` (go build) | ✅ DONE | `docs/audits/72-cross-build.md`              |
| 9.1  | Synthèse exécutive & priorisation      | `general-purpose`            | ✅ DONE | `docs/audits/99-executive-summary.md`        |
| 9.2  | Backlog d'améliorations chiffré        | `general-purpose`            | ✅ DONE | `docs/audits/99-backlog.md`                  |

**Légende statut :** ⬜ TODO · 🟡 EN COURS · ✅ DONE · ❌ BLOQUÉ

---

## 1. Règles d'exécution — Agents teams

> **Règle obligatoire :** chaque tâche du tableau §0 doit être exécutée par un **agent dédié** (sous-agent Claude Code). L'agent principal (toi) **orchestre** mais **ne fait pas le travail**, sauf pour la mise à jour du tableau de suivi.

### 1.1 Sélection de l'agent

| Type de tâche                                              | Agent à utiliser              |
| ---------------------------------------------------------- | ----------------------------- |
| Recherche ciblée, localisation de symbole/fichier          | `Explore` (`quick`)           |
| Cartographie multi-fichiers, inventaire                    | `Explore` (`medium` / `very thorough`) |
| Audit architectural, conception, priorisation              | `Plan`                        |
| Analyse transversale, code mort, conventions, perf         | `general-purpose`             |
| Revue de PR / qualité de code                              | Skill `review`                |
| Revue sécurité (OWASP, supply-chain)                       | Skill `security-review`       |
| Exécution de commandes (`make`, `go test`)                 | `Bash`                        |

### 1.2 Parallélisme

- Les tâches **indépendantes d'une même phase** doivent être lancées **en parallèle** (un seul message, plusieurs blocs `Agent`).
- Les phases sont **séquentielles entre elles** (la 2 dépend de la 1, etc.) — sauf §5.4, §6.2, §8.2, §8.3 qui sont des `Bash` indépendants et peuvent être lancés en background.

### 1.3 Briefing des agents

Chaque appel d'agent doit fournir :

1. **Contexte** — extrait pertinent de `CLAUDE.md` (architecture, conventions, contraintes perf).
2. **Question précise** — pas d'instruction générique « audite X ».
3. **Format de sortie** — markdown structuré, chemin de livrable explicite, `< 400 mots` pour les phases d'inventaire, `< 800 mots` pour les phases d'analyse.
4. **Critères de succès** — voir §2 ci-dessous.

### 1.4 Mise à jour du tableau

À la fin de chaque tâche réussie, **mettre à jour la cellule `Statut`** du tableau §0 (⬜ → ✅) **dans le même tour** que la livraison du fichier d'audit. En cas d'échec ou blocage, passer à ❌ et ajouter une note en §3 (Journal).

### 1.5 Mémoire

Si un agent découvre une convention non documentée, un piège récurrent, ou un fait projet non évident, sauvegarder en mémoire (`feedback` ou `project`) selon la politique standard.

---

## 2. Critères de succès par phase

### Phase 1 — Reconnaissance
- Inventaire complet : 22 packages identifiés, LOC par package, dépendances directes/indirectes listées.
- Snapshot Git : 20 derniers commits, branches actives, état working tree.

### Phase 2 — Architecture
- Diagramme C4 vérifié vs. réalité du code.
- **Test d'étanchéité** : `grep` confirme qu'aucun fichier `cmd/` n'importe au-delà de `internal/app` et `internal/cli`.
- Liste des interfaces avec ratio méthodes/implémentations (signal ISP).

### Phase 3 — Qualité
- Toute fonction > 100 lignes ou > 50 statements listée (référence `.golangci.yml`).
- Complexité cyclomatique > 15 listée.
- Conformité aux conventions FibGo : packages par responsabilité, erreurs structurées (`fmt.Errorf("%w", err)`), `t.Parallel()` systématique.

### Phase 4 — Sécurité
- OWASP Top 10 + supply-chain (go.sum, modules retired).
- Pas de secret commité, vérification `git log -p` sur fichiers sensibles.
- Licences compatibles Apache 2.0.

### Phase 5 — Performance
- Pas d'allocation inutile dans `internal/fibonacci/` et `internal/bigfft/` (vérifié `go test -bench -benchmem`).
- Baseline benchmarks sauvegardée pour comparaison future.
- Concurrence : aucune goroutine sans contrôle de cycle de vie.

### Phase 6 — Tests
- Couverture par package documentée (objectif informatif, pas de seuil imposé).
- Golden tests passent (`internal/fibonacci/testdata/fibonacci_golden.json`).
- `make test` (race detector ON) passe à vert.

### Phase 7 — Documentation
- Chaque package public a un `doc.go`.
- README.md à jour vs. flags CLI réels.
- `docs/architecture/` cohérent avec le code.

### Phase 8 — Outillage
- `make all` réussit en CI mode propre.
- `make lint` zéro warning bloquant.
- `make build-all` produit binaires linux/windows/macOS.

### Phase 9 — Synthèse
- Résumé exécutif `< 1 page`.
- Backlog priorisé par impact × effort, avec estimation grossière.

---

## 3. Journal d'exécution

Chaque entrée : `YYYY-MM-DD HH:MM — #tâche — agent — résumé`.

```
2026-04-28 — Phase 1 (1.1, 1.2, 1.3) — 3 × general-purpose en parallèle — inventaire, deps, snapshot Git livrés.
2026-04-28 — Phase 2 (2.1, 2.2, 2.3) — 3 × general-purpose en parallèle — Clean Arch sans cycle ; 1 violation étanchéité ; 13 interfaces ISP saines.
2026-04-28 — Phase 3 (3.1, 3.2, 3.3) — 3 × general-purpose en parallèle — qualité élevée, 17 symboles morts, 6,5/7 conventions.
2026-04-28 — Phase 4 (4.1, 4.2) — 2 × general-purpose en parallèle — 0 critique sécurité ; 36/36 licences whitelistées.
2026-04-28 — Phase 5 (5.1, 5.2, 5.3, 5.4) — 4 × general-purpose ; 5.4 ré-exécutée après échec `make` (utiliser `go test -bench` direct). 86 benchmarks parsés.
2026-04-28 — Phase 6 (6.1, 6.2) — 2 × general-purpose ; couverture 88,9 % ; race detector indisponible (CGO requis sur Windows).
2026-04-28 — Phase 7 (7.1, 7.2, 7.3) — 3 × general-purpose ; doc user 7/10, doc tech 8/10, doc.go 20/21.
2026-04-28 — Phase 8 (8.1, 8.2, 8.3) — 3 × general-purpose ; Makefile 7/10 ; lint cassé (golangci-lint v1.64 < Go 1.26) ; cross-build 6/6 OK.
2026-04-28 — Phase 9 (9.1, 9.2) — 2 × general-purpose ; synthèse 7,9/10 ; backlog 47 items (P0=6, P1=15, P2=15, P3=11).
```

---

## 4. Livrables

Tous les rapports sont écrits dans `docs/audits/` (créer le sous-dossier au besoin) en markdown.
La synthèse finale §9.1 sert de point d'entrée et lie les rapports détaillés.

## 5. Hors-scope

- Refactoring du code (audit uniquement — voir directive `Codebase mature` du `CLAUDE.md`).
- Modifications fonctionnelles ou correctifs de bugs (consignés dans le backlog §9.2).
- Migration d'algorithmes ou de dépendances (proposés, non exécutés).
