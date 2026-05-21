# ADR-0004: Décisions de backlog formelles post-hardening

- **Status**: Accepted
- **Date**: 2026-05-21
- **Audit source**: `Audit - Global - FibGo - v2.md` §5

## Context

Le ré-audit v2 a constaté que la cible PRD (≥ 92/100) est atteinte avec
93/100 ; cinq items demeurent. Cet ADR statue formellement sur leur
sort plutôt que de les laisser flotter en *backlog* implicite.

## Decision

### Item B1 — Suppression progressive des globaux `bigfft` au profit de `FFTContext` exclusif

**Statut : WONT-FIX (release actuelle), à reconsidérer post-v1.0.**

Justification :
- L'atomic conversion (ADR-0003) a neutralisé le risque concurrentiel
  qui motivait la trajectoire `FFTContext`.
- Une migration *big-bang* casserait l'API publique (`SetFFTParallelismConfig`,
  les accesseurs `Get*`) et le code de calibration qui s'appuie dessus.
- Le coût ingénierie pour un gain marginal (suppression de quelques
  variables atomiques) n'est pas justifiable face aux items productifs.

À revoir si : un *use case multi-tenant* concret (plusieurs `Mul`
concurrents avec configurations divergentes dans le même process) est
ouvert dans l'issue tracker.

### Item B2 — Simplification de `internal/calibration/` (1 686 LOC)

**Statut : WONT-FIX (release actuelle).**

Justification :
- ADR-0001 a tranché : `DynamicThresholdManager` est **conservé** car
  son coût de maintenance a chuté à ~0 après l'atomic conversion E1.
- La calibration au démarrage (`internal/calibration/`) a une logique
  distincte (micro-bench + heuristique CPU) qui n'est pas redondante
  avec DTM (qui ajuste *en cours de calcul*). Les deux ne sont **pas**
  identiquement substituables.
- Simplifier exigerait une refonte algorithmique, hors scope du
  hardening.

À revoir si : un profilage rigoureux (`benchstat` ≥ 20 runs) sur une
machine quiescente démontre que `calibration` n'apporte rien vs.
constantes statiques.

### Item B3 — Bench cross-arch automatisé (ARM64, etc.)

**Statut : WONT-FIX (release actuelle), tracé dans
`docs/PORTABILITY.md` §6.**

Justification :
- GitHub Actions ne fournit pas de runners ARM64 dans son tier gratuit
  pour les repos publics open-source de cette taille (le tier
  *Larger Runners* est payant).
- Le job `cross-compile` ajouté par E10-R5 vérifie déjà la
  *compilabilité* sur arm64/darwin ; l'absence de bench n'introduit pas
  de régression silencieuse.

À revoir si : le projet passe sous GitHub Enterprise / Actions Pro, ou
si un runner self-hosted ARM64 devient disponible.

### Item B4 — Suppression définitive de `EVALUATION.md`

**Statut : DEFER (sera supprimé après une release).**

Justification :
- Le fichier est déjà déplacé vers `docs/external-reviews/` avec un
  en-tête de transparence (E9-R2). Le risque réputationnel d'une
  auto-évaluation 98/100 non-revue est neutralisé.
- Garder l'historique permet de tracer la trajectoire (auto-eval →
  audit consolidé) pour les contributeurs futurs.

À revoir si : un mainteneur juge que l'historique n'a plus de valeur.

### Item B5 — Extension du golden au-delà de F(10 000)

**Statut : OPEN (peut être fait à tout moment).**

Justification :
- `internal/fibonacci/testdata/fibonacci_golden.json` est documenté
  comme « **immuable** sans accord explicite » (CLAUDE.md). L'extension
  via `cmd/generate-golden` est autorisée par cet ADR, qui constitue
  désormais l'accord explicite.
- Le coût est principalement en temps machine (F(1M) prend ~3-5 ms,
  F(10M) ~40 ms ; ajouter ces entrées + leur résultat sérialisé en
  base64 ou hex dans le JSON).

Action concrète : voir le commit qui clôt cet ADR pour les entrées
F(100k), F(500k), F(1M) ajoutées via `cmd/generate-golden`.

## Consequences

- Le PRD est maintenant **formellement clos**. Aucun item flottant.
- Le backlog est entièrement matérialisé en ADR (0001, 0002, 0003,
  0004) et en `docs/PORTABILITY.md`. Tout nouveau mainteneur peut
  retrouver la décision et la justification sans archéologie git.

## References

- `Audit - Global - FibGo - v2.md` §5
- ADR-0001 (DTM KEEP)
- ADR-0002 (recover sentinel)
- ADR-0003 (globaux atomic)
- `docs/PORTABILITY.md`
