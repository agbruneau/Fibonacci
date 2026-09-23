# ADR-0013: Évaluation académique 2026-09-15 — décisions D1 à D8

- **Status**: Proposed
- **Date**: 2026-09-23
- **Deciders**: André-Guy Bruneau (mainteneur)
- **Context source**: [`evaluation-academique-2026-09-15.md`](../audits/evaluation-academique-2026-09-15.md)
  (note 82,2/100, A-, recevable sous réserve de R3) et son plan
  [`plan-evaluation-2026-09-15.md`](../audits/plan-evaluation-2026-09-15.md),
  tâches `EVAL-00` à `EVAL-25`. Base : `main` à `93af525`.

## Context

L'évaluation note le dépôt contre une grille universitaire de dix critères. Elle
le déclare recevable **sous réserve** : `internal/bigfft` dérive de
`github.com/remyoudompheng/bigfft` (BSD-3-Clause) sans en conserver la notice
(constat P1). Elle relève en outre une borne de complexité fausse à six endroits
(P2), des fonctions inexistantes au CHANGELOG 1.0.0 (P3), des tableaux de
performance sans artefact (P6), un gestionnaire de seuils dynamiques mesuré
neutre et conservé (P7), l'absence de bibliographie et de mesure externe (P8),
et une couverture gardée à 80 % pour 96 % mesurés (P9).

Huit décisions conditionnaient le plan (§ 2). Elles sont tranchées ici.

## Decision

| # | Question | Décision |
|---|---|---|
| D1 | Sort du DTM | **(a)** supprimer le paquet `threshold`, le drapeau `-dynamic-thresholds`, `FIBCALC_DYNAMIC_THRESHOLDS` et la plomberie ; ADR-0001 passe en *Superseded* |
| D2 | Règle de langue | **(b)** amender la règle : narratif (README, CHANGELOG, ADR, `docs/audits/`) en français, référence technique (`docs/*.md`, `docs/algorithms/`, `docs/architecture/`) en anglais ; `docs/ARCH.md` rendu monolingue |
| D3 | CHANGELOG 1.0.0 | **(b)** retirer les quatre lignes réfutées, marquer 0.1.0 et 1.0.0 « préhistoire » |
| D4 | Plancher de couverture | **92 %**, sous réserve du relevé CI Ubuntu |
| D5 | Étiquettes `rewrite/*` | **(a)** supprimer, après sauvegarde ; sur `origin` avec l'accord explicite du mainteneur au moment de l'exécution |
| D6 | *Worktree* `lucid-nightingale-3e810f` | **(a)** retirer si propre et sans commit non fusionné utile ; sinon rapporter et conserver |
| D7 | Version de clôture | **`v5.0.0`** (D1 = a retire un drapeau et une variable d'environnement) |
| D8 | Mesure externe | **(a)** brancher `-algo gmp` sous le tag et mesurer dans le job CI `gmp` |

Tâches écartées, avec leur motif :

- **EVAL-11** — job CI de non-régression : le bruit des *runners* partagés dépasse
  le seuil de 5 %. À revoir si un *runner* dédié devient disponible.
- **EVAL-15** — couverture de `internal/testutil` : aide de test de 51 lignes.
- **EVAL-18** — entrées CHANGELOG `v2.x` / `v3.0.0` : la note en tête de 4.0.0
  dit déjà qu'elles n'existent pas et pourquoi.
- **EVAL-24** — unification des deux sémaphores : aucune mesure de contention.
  À revoir si un profil montre du sur-abonnement.

## Consequences

### Positive

- La condition de recevabilité R3 est levée par EVAL-01.
- Les affirmations chiffrées publiées renvoient toutes à un artefact archivé.

### Negative / Trade-offs

- D1 est une rupture d'interface (drapeau et variable retirés), d'où D7.
- D2 fige une documentation bilingue par genre plutôt qu'une langue unique.

### Risks and Mitigations

- Voir le plan, § 5.

## Alternatives Considered

- Options (b) de D1, (a) de D2, (a) de D3 et (b) de D5 à D8 : voir le plan, § 2,
  qui en donne le coût.

## References

- Évaluation : [`evaluation-academique-2026-09-15.md`](../audits/evaluation-academique-2026-09-15.md)
- Plan : [`plan-evaluation-2026-09-15.md`](../audits/plan-evaluation-2026-09-15.md)
- Related ADR(s) : [ADR-0001](0001-dtm-decision.md) (remplacé), [ADR-0012](0012-audit-2026-09-livre-decisions.md) (D4 amendée)
