# ADR-0004: Décisions de backlog formelles post-hardening

- **Status**: Accepted
- **Date**: 2026-05-21
- **Context source**: clôture du hardening sprint (commits `c0cc530` → `3d8b977`).

## Context

À la clôture du hardening sprint, cinq items du plan d'exécution
demeuraient en discussion. Cet ADR statue formellement sur leur sort
plutôt que de les laisser flotter en *backlog* implicite.

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

> **Correctif (OVR-08, audit 2026-07)** : la justification « le code de
> calibration qui s'appuie dessus » est caduque — vérifié par grep,
> `internal/calibration` n'appelle que `bigfft.Mul` ; aucun appelant
> n'invoque `SetFFTThreshold`/`SetFFTParallelismConfig`/les accesseurs
> `Get*` hors tests. Seul le premier argument (risque concurrentiel
> neutralisé par ADR-0003) reste valide pour ce WONT-FIX.

À revoir si : un *use case multi-tenant* concret (plusieurs `Mul`
concurrents avec configurations divergentes dans le même process) est
ouvert dans l'issue tracker.

> **Addendum (2026-07-11, audit Fable5 DEAD-01)** : l'API opt-in
> `FFTContext` elle-même (`context.go`, `fft_recursion_ctx.go`, ~572 LOC
> prod + ~530 LOC tests, **zéro appelant de production** vérifié par
> passe réfutative) a été **supprimée de l'arbre** sur décision
> mainteneur — l'abstraction avait été codée pour une migration que ce
> B1 classe précisément WONT-FIX. Le WONT-FIX demeure ; si la migration
> renaît un jour (clause « à revoir si » ci-dessus), le code se récupère
> de l'historique git (`git log --diff-filter=D -- internal/bigfft/context.go`
> donne le commit de suppression) plutôt que d'être maintenu mort dans
> l'arbre.

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

**Mise à jour (2026-06-21) :** il n'y a plus de CI dans le dépôt — le
répertoire `.github/` (et le job `cross-compile` d'E10-R5 cité ci-dessus)
a été supprimé (cf. `CHANGELOG.md`). La compilabilité cross-arch est
désormais vérifiée **localement** via `make build-all` /
`make build-windows-arm64` (`docs/PORTABILITY.md` §5 « Vérification locale » ;
§6 est « Limitations connues »). Le statut WONT-FIX
(bench cross-arch) reste valide ; seule la justification s'appuyant sur la
CI est caduque.

### Item B4 — Suppression définitive de `EVALUATION.md`

**Statut : DONE (exécuté) — voir mise à jour ci-dessous.**

Justification :
- Le fichier est déjà déplacé vers `docs/external-reviews/` avec un
  en-tête de transparence (E9-R2). Le risque réputationnel d'une
  auto-évaluation 98/100 non-revue est neutralisé.
- Garder l'historique permet de tracer la trajectoire (auto-eval →
  audit consolidé) pour les contributeurs futurs.

**Mise à jour (2026-06-21) :** exécuté. Le fichier (déplacé en
`docs/external-reviews/2026-02-08-jules-self-evaluation.md`) a été purgé
au commit `7ab9098` ; `docs/external-reviews/` n'existe plus dans l'arbre
(vérifié le 2026-09-04 : `docs/` contient `adr/`, `algorithms/`,
`architecture/`, `audits/` et les sept fichiers `.md` de tête ; le `dashboard/`
que listait la vérification du 2026-08-07 a lui aussi été purgé depuis, le
2026-08-09, commit `408a0c9`).
La décision de report n'a plus d'objet.

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

> **Correctif (annoté le 2026-09-04, l'action concrète est conservée telle
> qu'elle a été écrite)** : ce ne sont pas ces entrées-là. Le corpus
> effectivement ajouté est **F(50000), F(100000), F(200000)** — recompté le
> 2026-09-04 sur `internal/fibonacci/testdata/fibonacci_golden.json`, dont les
> trois dernières entrées sont `"n": 50000`, `"n": 100000`, `"n": 200000` ; ni
> F(500k) ni F(1M) n'y figurent. Voir *Status note (2026-06-10)*, en fin de
> document. Le `CLAUDE.md` cité juste au-dessus comme source du statut
> « immuable » a par ailleurs été retiré du dépôt le 2026-07-31 (commit
> `869bd6a`) : ce statut tient désormais du présent ADR seul.

## Consequences

- Le PRD est maintenant **formellement clos**. Aucun item flottant.
- Le backlog est entièrement matérialisé en ADR (0001, 0002, 0003,
  0004) et en `docs/PORTABILITY.md`. Tout nouveau mainteneur peut
  retrouver la décision et la justification sans archéologie git.

## References

- ADR-0001 (DTM KEEP)
- ADR-0002 (recover sentinel)
- ADR-0003 (globaux atomic)
- `docs/PORTABILITY.md`

## Status note (2026-06-10)

Item B5 — corpus effectivement ajouté en mai 2026 via `cmd/generate-golden` :
F(50000), F(100000), F(200000) — et non F(100k)/F(500k)/F(1M) comme
l'annonçait l'action concrète ci-dessus (vérifié sur
`internal/fibonacci/testdata/fibonacci_golden.json` à HEAD). Le fichier golden
reste immuable sans nouvel accord ADR ; toute extension future requiert le
même protocole.

## Status note (2026-08-07)

- `CLAUDE.md` (cité en B5 comme source du statut « immuable » du golden) a été
  retiré du dépôt le 2026-07-31 (commit `869bd6a`). Le statut immuable tient
  désormais de cet ADR seul ; il n'existe plus de fichier de directives dans
  l'arbre.
