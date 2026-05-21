# PRD Execution Plan — FibCalc Hardening Initiative

**PRD amont** : `Audit - PRD - FibGo.md`
**Audit amont** : `Audit - Global - FibGo.md`
**Version** : 1.0
**Date** : 2026-05-21
**Stratégie** : multi-track parallèle exploitant les équipes d'agents Claude Code et Ruflo.

---

## 1. Vue Aérienne

Le plan articule **5 sprints** sur ~6 semaines calendaires (ajustable selon disponibilité), enchaînant un sprint préparatoire (S0) et quatre sprints de livraison (S1→S4). Chaque sprint comporte des *tracks* parallèles, chaque *track* étant pris en charge par une **équipe d'agents** dédiée. Les gates inter-sprints sont des moments de revue obligatoires avant ouverture du sprint suivant.

```
Time →  S0          S1                S2                  S3                  S4
        2j          2 semaines        1 semaine           1.5 semaines        1 semaine
        ┌──────┐    ┌──────────────┐  ┌────────────────┐  ┌────────────────┐  ┌────────┐
        │ PREP │ →  │ P0 PARALLEL  │→ │ P0 SEQUENCED   │→ │ P1 STABILIZE   │→ │ P2 + RE│
        │  S0  │    │ E1 E2 E5 E6  │  │ E3 E4 + E2 fin │  │ E7 E8 E9 partl │  │ AUDIT  │
        └──────┘    └──────────────┘  └────────────────┘  └────────────────┘  └────────┘
        gate G0     gate G1            gate G2             gate G3             gate G4
```

---

## 2. Composition des Équipes d'Agents

### 2.1 Équipes permanentes

| Équipe | Subagents Claude / Ruflo | Rôle |
|---|---|---|
| **Architect Team** | `ruflo-swarm:architect`, `Plan` (Claude) | Conception du *refactor design*, ADR, frontières de package, contrats d'interface. |
| **Coder Team** | `ruflo-core:coder` ×N, `general-purpose` | Implémentation TDD, *surgical changes* (max 50 LOC/2 fichiers/refactor). |
| **Researcher Team** | `ruflo-core:researcher`, `Explore`, `understand-anything:understand` | Cartographie d'impact, traversée du graphe de dépendances, pré-flight code reading. |
| **Reviewer Team** | `ruflo-core:reviewer`, `code-reviewer` (skill) | Revue PR, gate qualité, vérification *acceptance criteria*. |
| **Witness Team** | `ruflo-core:witness-curator` | Signature ADR-103 des correctifs P0 ; manifeste cryptographique des fix-markers. |
| **Coordinator** | `ruflo-swarm:coordinator`, `ruflo-autopilot:autopilot-coordinator` | Anti-drift, *task assignment*, suivi d'exécution. |

### 2.2 Équipes spécialisées (mobilisées ad hoc)

| Équipe | Quand | Quoi |
|---|---|---|
| **Security Team** | Sprint S1 / Epic E5 | `general-purpose` paramétré sécurité + skill `security-review` ; tests adversariaux completion. |
| **Perf Team** | Sprint S2 / Epic E4 | `general-purpose` benchmark + `skill: diagnose` ; baselines benchstat, gate CI. |
| **DevEx Team** | Sprint S1 / Epic E6 | `general-purpose` infra ; Dockerfile multi-stage, devcontainer. |
| **Doc Team** | Sprint S3 / Epic E9 | `understand-anything:tour-builder`, `claude-md-management:claude-md-improver` ; sync C4, badges. |
| **Knowledge Graph Team** | S0 et après chaque sprint | `understand-anything:understand-diff` ; impact analyse des PRs. |

### 2.3 Principes opérationnels

1. **Subagent-driven development** : skill `superpowers:subagent-driven-development` activé pour chaque sprint multi-tracks.
2. **Parallel dispatch** : skill `superpowers:dispatching-parallel-agents` activé dès que ≥ 2 tracks sont *strictement* indépendants (pas d'état partagé, pas de dépendance séquentielle).
3. **TDD obligatoire** sur tout track touchant `internal/fibonacci/` ou `internal/bigfft/` (skill `superpowers:test-driven-development` + projet directive 2 *golden tests*).
4. **Worktree isolation** : chaque Epic ouvre une branche dédiée `<type>/<epic-id>-<slug>` ; merges via PR.
5. **Verification before completion** : skill `superpowers:verification-before-completion` exigée avant toute clôture de track.

---

## 3. Sprint S0 — Préparation (2 jours)

### 3.1 Objectifs

- Cartographier précisément les call-sites cités dans l'audit (validation des `file:line`).
- Établir la baseline performance (`benchstat`).
- Reproduire empiriquement les data races sous `go test -race -count=100`.
- Décider du squelette ADR.
- Préparer les worktrees pour les Epics P0.

### 3.2 Tracks

| Track | Équipe | Livrable |
|---|---|---|
| **S0-T1** Validation des évidences audit | Researcher Team (`ruflo-core:researcher` + `Explore`) | Note de validation : chaque citation `file:line` vérifiée ou contestée. |
| **S0-T2** Baseline performance | Perf Team (`general-purpose`) | `docs/audits/bench-baseline-pre-hardening.txt` (5 tailles × 3 algos). |
| **S0-T3** Reproduction empirique data races | Researcher Team | Output `go test -race -count=100` capturé ; sites confirmés. |
| **S0-T4** Squelette ADR | Architect Team (`ruflo-swarm:architect` + `Plan`) | `docs/adr/0000-template.md`, `docs/adr/0001-dtm-decision.md` (stub), `docs/adr/0002-recover-strategy.md` (stub). |
| **S0-T5** Setup worktrees | Coordinator | 6 worktrees créés (un par Epic P0). |

### 3.3 Dispatch parallèle

`S0-T1` ∥ `S0-T2` ∥ `S0-T3` ∥ `S0-T4` — strictement indépendants.
`S0-T5` séquentiel après `S0-T4` (besoin du squelette ADR).

### 3.4 Gate G0 — Sortie S0

- [ ] Note de validation publiée et acquittée par mainteneur.
- [ ] Baseline `bench-baseline-pre-hardening.txt` commitée.
- [ ] Au moins 1 data race reproduite empiriquement (preuve par output captura).
- [ ] ADRs squelettes mergés sur `main`.
- [ ] Worktrees prêts.

---

## 4. Sprint S1 — P0 Parallèle (2 semaines)

### 4.1 Objectifs

Exécuter en parallèle les Epics P0 strictement indépendants : **E1** (concurrence), **E5** (security CLI), **E6** (containerisation), et **E8 phase 1** (extension golden/fuzz, prérequis de E2).

### 4.2 Tracks

| Track | Epic | Équipe | Livrable | Dépendances |
|---|---|---|---|---|
| **S1-T1** Atomic conversion DTM | E1-R1 | Coder Team + Reviewer | `atomic.Int64` sur `currentFFTThreshold`, `currentParallelThreshold`, `lastAdjustment` ; tests race x100. | S0-T3 (preuve race) |
| **S1-T2** TransformCache.config sync | E1-R2 | Coder Team + Reviewer | `RLock` sur 4 call-sites OU `atomic.Bool` ; tests concurrents. | S0-T3 |
| **S1-T3** Globaux bigfft → FFTContext | E1-R3 | Architect + Coder | Migration `fftThreshold` etc. vers `FFTContext` exclusif OU `atomic.Int64`. | Décision ADR-0003. |
| **S1-T4** Cache FFT refcount | E1-R4 | Architect + Coder + Reviewer | `cacheEntry.backing` refcount OU deep-copy `Get`. | — |
| **S1-T5** Tests adversariaux completion | E5 (R1+R2+R3) | Security Team | Suite tests injection + fuzz `internal/cli/completion/`. | — |
| **S1-T6** Dockerfile + devcontainer | E6 (R1+R2) | DevEx Team | `Dockerfile` multi-stage, `.devcontainer/devcontainer.json`. | — |
| **S1-T7** CI consomme image | E6-R3 | DevEx Team | `.github/workflows/ci.yml` référence l'image publiée. | S1-T6 |
| **S1-T8** Résolution contradiction race Windows | E6-R4 | Doc Team + Researcher | `CLAUDE.md` et `CHANGELOG.md` cohérents ; section toolchain MinGW. | — |
| **S1-T9** Extension golden FFT + fuzz | E8 | Coder Team + Witness | ≥ 5 entrées F(100k…10M) ; cibles `FuzzFermat*`. | S0-T2 (baseline). |

### 4.3 Dispatch parallèle

```
                ┌── S1-T1 ──┐
E1 Concurrence ─┼── S1-T2 ──┼── merge → E1 done
                ├── S1-T3 ──┤
                └── S1-T4 ──┘

E5 Security  ── S1-T5 ── merge → E5 done
                ┌── S1-T6 ── S1-T7 ──┐
E6 Container ───┤                    ├── merge → E6 done
                └── S1-T8 ───────────┘
E8 Coverage  ── S1-T9 ── merge → E8 partly done (prerequisite E2)
```

Toutes les chaînes ci-dessus sont **strictement parallèles** entre elles (équipes distinctes, branches distinctes, pas de fichier partagé hors `CLAUDE.md` qui sera mergé via PR séquentiel).

### 4.4 Coordination anti-drift

- Le `ruflo-swarm:coordinator` instancie un swarm `s1-p0-parallel` avec 9 agents.
- Le `ruflo-autopilot:autopilot-coordinator` planifie les *loops* d'exécution (1 par track).
- Chaque PR est gated par :
  1. CI verte sur 3 OS.
  2. Revue `ruflo-core:reviewer`.
  3. Skill `superpowers:verification-before-completion`.
  4. Si toucher `bigfft/` ou `fibonacci/` : skill `superpowers:requesting-code-review` obligatoire.

### 4.5 Gate G1 — Sortie S1

- [ ] `go test -race -count=100 ./...` propre sur Linux/macOS/Windows.
- [ ] Couverture `internal/cli/completion/` ≥ 80 %.
- [ ] `docker build .` réussi, image < 500 Mo.
- [ ] Devcontainer ouvre en VS Code → `make all` vert.
- [ ] ≥ 5 nouvelles entrées golden, ≥ 3 cibles fuzz `bigfft` exécutent au moins 60 s en CI.
- [ ] Aucune régression > 5 % vs baseline S0-T2.

---

## 5. Sprint S2 — P0 Séquencé (1 semaine)

### 5.1 Objectifs

Exécuter les Epics P0 qui exigent un état stable post-S1 :
- **E2** (suppression `recover()` global) : exige le filet de sécurité E8 (S1-T9).
- **E3-R1** (briser triangle `threshold → config`) : exige l'atomic conversion S1-T1.
- **E4** (perf gate bloquant) : exige la baseline S0-T2.

### 5.2 Tracks

| Track | Epic | Équipe | Livrable | Dépendances |
|---|---|---|---|---|
| **S2-T1** Suppression `recover()` global FFT | E2-R1 | Architect + Coder + Witness | `bigfft/fft.go` sans `recover()` global OU restreint à set sentinelle ; tests panic dédiés. | S1-T9, ADR-0002. |
| **S2-T2** Décision `*Safe` wrappers | E2-R2 | Architect + Coder | ADR-0002 finalisée ; wrappers exposés OU supprimés. | S2-T1. |
| **S2-T3** Log/count `recover()` muets | E2-R3 | Coder | Compteur `recoveredObservers` exposé. | — |
| **S2-T4** Briser triangle threshold→config | E3-R1 | Architect + Coder + Researcher | Injection `ThresholdTuningProfile` ; `go list -deps` propre. | S1-T1. |
| **S2-T5** Perf gate bloquant | E4 (R1+R2+R3) | Perf Team + DevEx | Workflow `ci.yml` avec `benchstat`, `bench-baseline.txt` versionnée. | S0-T2, S1 done. |

### 5.3 Dispatch

`S2-T1` → `S2-T2` séquentiel.
`S2-T3` ∥ `S2-T4` ∥ `S2-T5` parallèles entre eux et avec `S2-T1`.

### 5.4 Gate G2 — Sortie S2 (PRD acquitté côté P0)

- [ ] `bigfft/fft.go:41-101` ne contient plus de `recover()` global indistinct.
- [ ] Tests `TestFermatPostConditionPanic` verts.
- [ ] `go list -deps ./internal/fibonacci/threshold` n'inclut plus `internal/config`.
- [ ] PR introduisant volontairement une régression 6 % est **rejetée** par CI.
- [ ] ADR-0001, ADR-0002, ADR-0003 mergées.
- [ ] **Re-audit intermédiaire** (auto-mesure selon la grille) ≥ 88 / 100.

---

## 6. Sprint S3 — P1 Stabilisation (1.5 semaines)

### 6.1 Objectifs

Refermer la dette architecturale P1, trancher sur la calibration adaptative.

### 6.2 Tracks

| Track | Epic | Équipe | Livrable | Dépendances |
|---|---|---|---|---|
| **S3-T1** ADR DTM + benchstat on/off | E7-R1, E7-R2 | Perf Team + Architect | `docs/audits/bench-dtm-{on,off}.txt`, `docs/adr/0001-dtm-decision.md` finalisée. | S2 done. |
| **S3-T2** Implémentation décision DTM | E7-R3 | Coder + Reviewer + Witness | Suppression OU justification ; A-18 archivé. | S3-T1. |
| **S3-T3** Sortir `format` de `errors` | E3-R2 | Architect + Coder | Struct sérialisable ; `internal/errors` plus `format`. | — |
| **S3-T4** Découpler `tui → fibonacci` | E3-R3 | Architect + Coder + Reviewer | `internal/tui` passe par `orchestration`. | — |
| **S3-T5** Test d'architecture | E3-R4 | Coder + Reviewer | `internal/arch_test.go` enforçant la hiérarchie. | S3-T3, S3-T4. |
| **S3-T6** Sync C4 + badges + EVALUATION.md | E9 (R1+R2+R3+R4+R5) | Doc Team | Diagrammes mis à jour, `EVALUATION.md` traité, packages décompte unifié, badge dynamique, `.gitignore` enrichi. | — |

### 6.3 Dispatch

`S3-T1` → `S3-T2` séquentiel.
`S3-T3` ∥ `S3-T4` ∥ `S3-T6` parallèles.
`S3-T5` séquentiel après `S3-T3` ET `S3-T4`.

### 6.4 Gate G3 — Sortie S3

- [ ] ADR-0001 implémentée (DTM gardé avec preuve, ou supprimé).
- [ ] `internal/errors` n'importe plus `internal/format`.
- [ ] `internal/tui` n'importe plus `internal/fibonacci`.
- [ ] `go test ./internal/arch_test.go` vert.
- [ ] Diagrammes C4 sans `sysmon` orphelin.
- [ ] `EVALUATION.md` statué.

---

## 7. Sprint S4 — P2 + Ré-Audit (1 semaine)

### 7.1 Objectifs

Refermer le polish hygiène et déclencher un **ré-audit complet** par les mêmes méthodes que l'audit initial (Claude grille à 100 points + Gemini revue narrative).

### 7.2 Tracks

| Track | Epic | Équipe | Livrable |
|---|---|---|---|
| **S4-T1** Panic tests `fermat.go` | E10-R1 | Coder | ≥ 13 tests panic ciblés ; couverture > 80 %. |
| **S4-T2** Shadowing `cap` | E10-R2 | Coder | 3 renames `c := cap(...)`. |
| **S4-T3** Activer `govet shadow` warning | E10-R3 | DevEx | `.golangci.yml` étendu. |
| **S4-T4** Étoffer `doc.go` formels | E10-R4 | Doc Team | 4 packages enrichis. |
| **S4-T5** Cross-compile matrice | E10-R5 | DevEx + Perf | CI cross-compile `linux/arm64`, `darwin/arm64` ; `docs/PORTABILITY.md`. |
| **S4-T6** Ré-audit Claude grille à 100 points | Audit | Architect + Researcher + Reviewer (5 agents parallèles) | `Audit - Claude - FibGo - v2.md` ; note cible ≥ 92. |
| **S4-T7** Ré-audit Gemini narratif | Audit | Architect | `Audit - Gemini - FibGo - v2.md` ; note cible ≥ 92. |
| **S4-T8** Audit global v2 | Doc Team | `Audit - Global - FibGo - v2.md` consolidé. |

### 7.3 Dispatch

`S4-T1..T5` parallèles entre eux.
`S4-T6` ∥ `S4-T7` après `S4-T5`.
`S4-T8` séquentiel après `S4-T6` ET `S4-T7`.

### 7.4 Gate G4 — Sortie S4 (PRD clos)

- [ ] Tous les items P2 traités (mergés ou *won't fix* via ADR).
- [ ] Re-audit consolidé ≥ 92 / 100.
- [ ] `MEMORY.md` Ruflo mis à jour avec les invariants nouveaux.
- [ ] Manifeste witness ADR-103 signé.
- [ ] Release v1.0 tag candidate proposée.

---

## 8. Tableau Récapitulatif Mapping Equipe ↔ Sprint ↔ Epic

| Équipe | S0 | S1 | S2 | S3 | S4 |
|---|---|---|---|---|---|
| **Architect** | ADR templates | E1-R3 design | E2 design, E3-R1 design | E7 ADR, E3-R2/R3 design | (consult) |
| **Coder** | — | E1, E8, E5 impl | E2, E3-R1, E2-R3 impl | E7 impl, E3 impl, arch_test | E10 impl |
| **Researcher** | Validation citations, race repro | (consult) | (consult) | (consult) | Re-audit |
| **Reviewer** | — | PR reviews | PR reviews | PR reviews | PR + audit |
| **Witness** | — | Fix markers E8 | Fix markers E2 | Fix markers E7 | Manifeste final |
| **Security** | — | E5 | — | — | — |
| **Perf** | Baseline | (consult) | E4 | E7 benchstat | E10-R5, re-audit |
| **DevEx** | — | E6 | (consult) | — | E10-R3/R5 |
| **Doc** | — | E6-R4 | — | E9 | E10-R4, audit v2 |
| **Coordinator** | Setup | Anti-drift | Anti-drift | Anti-drift | Closeout |

---

## 9. Commandes Clés (référence rapide)

| Phase | Commande |
|---|---|
| Setup swarm S0 | Utiliser skill `ruflo-swarm:swarm-init` avec `--epic=hardening --agents=5`. |
| Dispatch parallèle S1 | Skill `superpowers:dispatching-parallel-agents` pour les 9 tracks. |
| Race reproduction | `go test -race -count=100 ./internal/fibonacci/threshold/... ./internal/bigfft/...` |
| Baseline bench | `go test -bench=BenchmarkFibonacci -benchmem -count=10 -run=^$ ./internal/fibonacci/ \| tee docs/audits/bench-baseline-pre-hardening.txt` |
| Gate perf | `benchstat docs/audits/bench-baseline.txt docs/audits/bench-current.txt` |
| Re-audit lancement | `Agent` × 5 (`subagent_type=general-purpose`), un par critère de la grille. |
| Verrouillage final | Skill `superpowers:finishing-a-development-branch`. |

---

## 10. Risk Register Exécution

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| **Conflits de merge** entre tracks S1 parallèles | Moyenne | Moyen | Worktree isolation ; rebase quotidien ; coordinator surveille les hot files (`fastdoubling.go`, `fft.go`, `fft_cache.go`). |
| **Régression perf** post-E2 (suppression recover) | Moyenne | Élevé | Filet E8 préalable ; perf gate E4 actif dès S2 ; rollback rapide via ADR-0002. |
| **Indisponibilité d'un agent Ruflo** | Faible | Moyen | Fallback `general-purpose` ; documenter la procédure dans `docs/adr/0004-agent-fallback.md`. |
| **Re-audit final < 92** | Moyenne | Élevé | Itération S4 prolongée d'un sprint ; identifier le critère faible et lancer un *track de rattrapage*. |
| **Suppression DTM casse un consommateur non documenté** | Faible | Faible | Période d'observation 1 release avec `Deprecated` avant suppression définitive (cf. NFR-D2). |
| **Image Docker > 500 Mo** | Moyenne | Faible | Multi-stage agressif `golang:1.26-alpine` runtime ; distroless si possible. |

---

## 11. Mesures de Pilotage Hebdomadaire

| Métrique | Fréquence | Cible | Owner |
|---|---|---|---|
| % tasks Open vs Done | Hebdo | ≥ +20%/semaine | Coordinator |
| Data races détectées | Continu | 0 (post-S1) | Researcher |
| Régression perf max observée | Continu | ≤ 5 % | Perf |
| Couverture globale | Hebdo | ≥ 80 % | Reviewer |
| ADRs ouverts | Hebdo | ≤ 2 simultanés | Architect |
| Drift documentaire détecté | Fin de sprint | 0 (post-S3) | Doc |

---

## 12. Convention de Communication des Agents

1. **Statut quotidien** : chaque équipe publie un *progress note* dans le swarm (anti-drift).
2. **Blocage** : tout *blocker* > 4 h est escaladé au Coordinator via `ruflo-swarm:watch`.
3. **Décision majeure** : matérialisée en ADR immédiatement (pas de décision orale).
4. **PR de track** : titre `feat(<epic>): <description>`, body avec lien vers PRD + critère d'acceptation citation.
5. **Merge** : strict squash-and-merge sur `main` après gate vert.

---

## 13. Critère Global de Clôture du Plan

Le plan est **terminé** quand :
1. Tous les sprints S0→S4 ont franchi leur gate respectif.
2. Le ré-audit consolidé `Audit - Global - FibGo - v2.md` rend une note ≥ 92 / 100.
3. Le tag candidat `v1.0.0-rc1` est créé et passe la CI sur les 3 OS.
4. Le manifeste witness Ruflo est signé et archivé.
5. Une *retrospective* est documentée dans `docs/retrospectives/2026-Q2-hardening.md`.
