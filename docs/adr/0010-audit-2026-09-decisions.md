# ADR-0010: Audit 2026-09 — décisions structurantes et candidats rejetés

- **Status**: Accepted
- **Date**: 2026-09-03
- **Context source**: audit exhaustif de tout le code Go de production
  (`audit.md`, 23 constats), exécuté phase par phase. Vérification par
  build/vet/test/lint, sondes de mesure temporaires et `benchstat` A/B en
  double ordre avant chaque commit sensible à la performance.

## Context

L'audit a relevé 23 constats : 3 de sévérité haute, 1 sur l'outillage de gate,
8 moyens, 11 bas. Quatre d'entre eux appelaient une **décision de mainteneur**
avant exécution (D1-D4), et plusieurs recommandations ont été **révisées ou
rejetées** au moment de les appliquer, sur mesure. Cet ADR matérialise les unes
et les autres pour qu'un audit futur ne les re-suggère pas sans élément nouveau,
selon la convention établie par [ADR-0008](0008-audit-2026-06-rejected-candidates.md)
et [ADR-0009](0009-audit-2026-07-cleanup-and-rejected-fib05.md).

## Decision

### D1 — Priorité du flag explicite sur le profil de calibration (M-03)

**Retenu : le flag l'emporte.**

Un profil valide écrasait `--threshold`, `--fft-threshold` et
`--strassen-threshold` sans condition. Le comportement était documenté à trois
endroits comme « KNOWN SURPRISE », épinglé par un test qui déclarait lui-même :
« If the repo owner ever decides flags should win, this test is the thing that
must change with the behavior ». Il l'a été.

`ParseConfig` marque chaque seuil explicite (flag présent **ou** variable
`FIBCALC_*` définie) ; les deux chemins qui **rejouent** un profil stocké
(`LoadCachedCalibration`, `applyCachedProfile`) ne remplissent que ce qui est
resté implicite. Un profil est la supposition de l'outil sur ce que
l'utilisateur n'a pas précisé ; il ne prime pas sur ce qu'il a précisé.

**Hors de la règle, délibérément** : une passe fraîche de `--calibrate` /
`--auto-calibrate`. L'utilisateur y demande une mesure, elle est affichée, et
c'est elle qui est persistée. Respecter le pin à cet endroit écrirait la valeur
épinglée dans le profil *comme si* elle avait été mesurée.

### D2 — Sort du DTM : câblé, mesuré, laissé désactivé (M-04)

**Retenu : option A (câbler puis mesurer). Résultat : défaut `false`.**

[ADR-0001](0001-dtm-decision.md) avait tranché KEEP sur la foi d'un gain de
5-6 % à F(10M). Rien, hors tests, n'avait jamais mis
`Options.EnableDynamicThresholds` à `true` : ni flag, ni variable, ni
`internal/app`. Tout `threshold/`, `CacheStrategy`, `decideCacheTuning` et
`wireThresholdTuning` étaient donc inatteignables depuis le binaire, pendant que
`ARCH.md` et `PERFORMANCE.md` présentaient le sous-système comme actif.

`--dynamic-thresholds` / `FIBCALC_DYNAMIC_THRESHOLDS` livre le câblage manquant.
La mesure qui suit (`docs/audits/bench-dtm-2026-09.txt`, `-count=8`) **ne
reproduit pas** le gain : les deux écarts CPU sont dans le bruit (p = 0,279 et
p = 0,382) et le seul mouvement significatif est un **surcoût** d'allocations de
+17,9 % à F(1M). La mesure d'origine était `-benchtime=1x -count=5`
*single-sample* et l'ADR-0001 la qualifiait lui-même de bruitée en appelant
exactement cette reprise.

**KEEP maintenu** (coût de maintenance nul depuis la conversion atomique, et le
sous-système est désormais réellement atteignable), **défaut `false`**. La
raison n° 2 du KEEP d'origine — « gain plausible 5-6 % au régime ≥ 10M » — est
caduque sur cette machine.

### D3 — Outillage lint : migration golangci-lint v2 (GATE-01)

**Retenu : option A (migration v2), et le lint devient dur.**

Le binaire épinglé v1.64.8 est compilé contre go1.26 et ne peut pas analyser le
module sous go1.27 : chaque paquet échoue en `export data version 4 is greater
than maximum supported version 2`. `staticcheck` 2026.1 échoue de même. Les deux
scripts de gate traitaient le lint comme consultatif : ils affichaient l'échec
puis écrivaient `Overall: PASS` et sortaient 0. Pendant une période inconnue,
seul `go vet` tournait réellement — `gosec`, `gocritic`, `revive`, `errcheck`,
`shadow` et `unused` étaient silencieusement éteints.

La migration v2 (backlog A5-07) n'était donc plus optionnelle. Le lint est
désormais **bloquant** dans `check.sh` et `check.ps1`, y compris quand le
binaire est **absent** : un gate qui ne vérifie rien en silence est pire que pas
de gate.

### D4 — Détecteur de course : mesuré disponible, activé localement

**Retenu : activer `-race` dans `check.ps1` sous détection ; pas de CI
réintroduite.**

L'audit recommandait de réintroduire une CI minimale au motif que `-race` ne
tournait nulle part de façon fiable. La vérification a infirmé la prémisse :
`go test -race ./...` passe sur les **21 paquets** de cet hôte Windows
(`CGO_ENABLED=1`, chaîne C présente). L'en-tête de `check.ps1` affirmait qu'un
hôte Windows ne pouvait pas exécuter le détecteur — c'était une caractéristique
d'une installation, énoncée comme une limite de plate-forme.

`check.ps1` sonde maintenant `CGO_ENABLED` et la présence d'un compilateur C, et
active `-race` quand les deux sont réunis. La CI n'est pas réintroduite : le
motif qui la justifiait a disparu, et `ADR-0004 §B3` (mise à jour 2026-06-21)
acte l'absence de CI comme un choix assumé du dépôt.

À revoir si : le projet accueille des contributeurs dont on ne peut pas
supposer l'outillage local.

## Candidats révisés ou rejetés en cours d'exécution

### R1 — Plafond mémoire de la cache FFT à ×4 : **rejeté sur mesure** (M-08)

L'audit proposait de borner `TransformCache` à 4 × la taille de F(n), en
s'appuyant sur un taux de succès **mesuré à zéro** (0 hit / 27 miss à F(10M) sur
le chemin matriciel) : la cache ne servait à rien et coûtait de la mémoire.

Le taux nul est réel, mais **il ne se généralise pas**. Un calcul **répété** du
même n — la touche `r` du TUI, un balayage de calibration, les benchmarks
eux-mêmes — rejoue des opérandes identiques et fait mouche. Mesure du plafond
×4 (`docs/audits/bench-fftcache-2026-09.txt`) :

| Benchmark | Δ sec/op | Δ B/op | Δ allocs/op |
|---|---:|---:|---:|
| MatrixExp/10M | **+22,17 %** (p=0,000) | +75,86 % | +137,08 % |

Gate Directive #1 (régression > 5 % = blocage) → **rejet**. Le plafond retenu
est **×48**, dimensionné pour contenir les transformées d'**un** calcul (20
entrées d'environ 2 × l'opérande mesurées à F(10M)). A/B adjacent dans le temps
à ce palier : geomean **−4,72 %** sec/op, aucune régression.

À revoir si : quelqu'un démontre que le rejeu d'opérandes identiques n'est pas
un cas d'usage à préserver.

### R2 — Quatre « symboles morts » : **conservés comme oracles de test** (L-02)

Le grep de l'audit avait raison sur les faits — aucun appelant de production —
mais supprimer ces symboles supprimerait la seule surface observable
d'invariants dont le code de production, lui, dépend. Même raisonnement que
[ADR-0009 R3](0009-audit-2026-07-cleanup-and-rejected-fib05.md). Chacun porte
désormais la mention dans son doc-comment :

- `memory.CalculationArena.AllocBigInt` / `UsedWords` : seule façon d'observer
  les invariants de bump dont dépend `PreSizeFromArena` (avance de l'offset,
  découpage à trois indices, repli sur le tas, `Reset`). Sept tests d'arène en
  vivent.
- `threshold.MetricsBuffer.writtenCount` : seule façon de distinguer le compteur
  de vie de `Count()`, qui plafonne à `MaxMetricsHistory` — précisément ce que
  les tests de bouclage épinglent.
- `memory.GCController.setLogger` : seul point d'injection permettant de
  vérifier que `Begin`/`End` émettent bien leurs événements.
- L'alias `"ps"` de `completion.Generate` : inatteignable depuis le CLI
  (`config.Validate` n'accepte que les quatre noms complets) mais alias **testé**
  de l'API du paquet, pour les appelants qui utilisent `Generate` directement.

Sont bien supprimés : `cli.displayResultWithConfig`, `cli.displayMemoryStats`,
`format.formatProgressBarWithETA`, `orchestration.ProgressAggregator.calculatorCount`.

### R3 — Stabilité du micro-benchmark : **améliorée, pas résolue** (M-01)

Le critère d'acceptation visé — « dix exécutions donnent le même seuil, ou des
seuils différents avec une confiance < 0,5 » — **n'est pas atteint de façon
déterministe**, et l'artefact
(`docs/audits/microbench-stability-2026-09.txt`) le dit.

Six défauts réels ont été corrigés (comparaison de code identique sous le seuil
bigfft, exécution concurrente mesurant de la contention, absence de marge,
absence de monotonicité, confiance forfaitaire dont le socle égalait la barre
d'escalade, `64` codé en dur). Huit essais sur dix passent désormais sous la
barre et escaladent vers le balayage complet, au lieu de persister un tirage au
sort avec 0,70 de confiance.

Le résidu est une limite de résolution : sur ce CPU, la taille de 2000 mots se
situe juste après le seuil d'activation de bigfft, si bien que la question « la
FFT gagne-t-elle là ? » n'a pas de réponse nette, et aucune statistique tirée de
~125 ms d'échantillonnage ne la tranchera. Ce qui a changé, c'est que
l'ambiguïté est **visible** dans le score et routée vers la mesure faisant
autorité.

À revoir si : quelqu'un ajoute des paliers de taille intermédiaires entre 2000
et 8000 mots et démontre que la résolution accrue stabilise le résultat dans un
budget de démarrage acceptable.

### R4 — Enveloppe de l'estimation mémoire : forme close hors de portée (H-03)

L'estimation sous-évaluait la mémoire réelle d'un facteur 5 à 12 à **tous** les
points mesurés, ce qui vide `--memory-limit` de son sens : il validait des
exécutions consommant plusieurs fois le budget déclaré.

Une forme close est **inaccessible depuis `internal/fibonacci/memory`** : le
terme dominant est le pré-chauffage des pools de `bigfft`, dont les classes de
taille en puissances de quatre font une fonction en escalier pouvant sauter d'un
facteur 4, et le gate d'architecture interdit `config → bigfft`, donc
`memory → bigfft`. Les multiplicateurs sont par conséquent une **enveloppe
empirique** calibrée sur l'algorithme le plus coûteux à chaque n mesuré, jamais
sous la mesure et au plus 2,5 × au-dessus.

Le critère « ≤ 2 × mesuré » proposé par l'audit est **inatteignable** : à F(10M)
les algorithmes s'étalent de 62 Mo (`fast`) à 141 Mo (`matrix`), soit plus d'un
facteur 2 à eux seuls. Une borne de sûreté doit couvrir le pire cas.

## Consequences

- Deux comportements observables changent et sont documentés dans le CHANGELOG :
  un flag de seuil explicite l'emporte désormais sur le profil, et `-1` est
  accepté comme « seuil désactivé » (H-02).
- `CurrentProfileVersion` passe de 3 à 4 : les profils écrits par l'ancienne
  recherche de crossover sont re-mesurés plutôt que rejoués.
- Le gate local vérifie strictement plus qu'avant : lint bloquant et `-race`
  sous Windows quand l'hôte le permet.
- Cinq artefacts de mesure entrent dans `docs/audits/` : mémoire, cache FFT,
  memclr des pools, stabilité du micro-benchmark, A/B du DTM.

## References

- `audit.md` (rapport et plan d'exécution, 493 lignes), section 5 pour la
  confrontation avec le document externe `audit Gemini.md` (400 lignes).
  **Ni l'un ni l'autre n'est dans l'arbre** — annotation du 2026-09-04, même
  traitement que le §Context d'[ADR-0009](0009-audit-2026-07-cleanup-and-rejected-fib05.md).
  `audit Gemini.md` a été purgé au commit `c8dab4a` et `audit.md` au commit
  `bf3d6a7`, tous deux le 2026-09-03. Ils restent lisibles dans l'historique
  git (`git show c8dab4a^:'audit Gemini.md'`, `git show bf3d6a7^:audit.md`) ;
  cet ADR et le CHANGELOG portent ce qui devait leur survivre.
- `docs/audits/mem-baseline-2026-09.txt`, `bench-fftcache-2026-09.txt`,
  `bench-poolclear-2026-09.txt`, `bench-dtm-2026-09.txt`,
  `microbench-stability-2026-09.txt`.
- [ADR-0001](0001-dtm-decision.md) (status note 2026-09-03),
  [ADR-0004](0004-backlog-decisions.md) §B1/§B3,
  [ADR-0008](0008-audit-2026-06-rejected-candidates.md),
  [ADR-0009](0009-audit-2026-07-cleanup-and-rejected-fib05.md).
