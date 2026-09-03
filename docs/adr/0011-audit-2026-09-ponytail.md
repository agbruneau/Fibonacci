# ADR-0011: Audit de sur-ingénierie 2026-09-03 — décisions et candidats écartés

- **Status**: Accepted
- **Date**: 2026-09-03
- **Context source**: passe « ponytail » (sur-ingénierie uniquement) sur tout
  le code Go de production, quatre audits plus tard. Vérification : build /
  vet / `go test ./...` / `golangci-lint` à zéro, `benchstat` A/B en double
  ordre sur `BenchmarkFibonacci/(FastDoubling|FFTBased)/1M`.

## Context

Les ADR-0008 à 0010 ont fermé la plupart des candidats évidents. Cette passe
cherchait ce qui reste : enveloppes qui ne font que déléguer, flexibilité sans
appelant, bibliothèque standard réécrite. Le CHANGELOG liste ce qui a été
appliqué ; cet ADR ne conserve que ce qui demande une décision ou une preuve,
pour qu'un audit futur ne le re-propose pas sans élément nouveau.

## Decision

### D1 — `app.ExitAction` supprimé

L'énumération mappait 1:1 les codes POSIX d'`internal/errors`, et `Run`
faisait l'aller-retour int → enum → int. Son seul contenu propre était
`ActionVersionHandled` : « `main` ne doit pas appeler `os.Exit` », introduit
pour remplacer un sentinel `-1`. Or `main` n'a aucun `defer`, donc revenir de
`main` et `os.Exit(0)` sont indiscernables. `Run` retourne le code POSIX et
`main` fait `os.Exit(run(...))`. Le test qui appelait `main()` en processus
(`TestMainFunction_VersionPath`) disparaît avec la propriété qu'il épinglait ;
les tests sous-processus couvrent les deux branches de `main`.

### D2 — Notation scientifique sans `big.Float`

Mesure (F(10M), 6,9 M bits) : `Sprintf("%.6e", new(big.Float).SetInt(x))`
345 ms, `SetPrec(64)` d'abord 328 ms, `x.String()` 317 ms. `Float.Text`
décale la mantisse de tout l'exposant avant de convertir, donc la précision ne
change rien : `--details` payait une seconde conversion complète, celle que
L-11 venait de dédupliquer. La notation est dérivée de la chaîne décimale déjà
en main, avec l'arrondi demi-pair de `%.6e` reproduit exactement et épinglé
contre `big.Float` par `TestScientificNotation_MatchesBigFloat`.

### D3 — `bigfft.allocUnsafe` supprimé malgré DEAD-10

DEAD-10 (2026-07) l'avait dé-exporté plutôt que supprimé. Ses seuls appelants
restaient ses propres tests ; aucun chemin de production ne l'a jamais atteint.
Supprimé. À revoir si : un chemin chaud a besoin d'une allocation bump sans
`clear` — le récupérer de l'historique plutôt que de le garder mort.

## Candidats écartés

### R1 — Wrappers `*Safe` de `fermat.go` : **conservés** (ADR-0002 §5)

Supprimés en cours de passe puis **restaurés** : ADR-0002 §5 les conserve
explicitement comme documentation testée du contrat de pré-conditions, et rien
de nouveau n'est apparu depuis (toujours zéro appelant de production, ce que
l'ADR savait déjà). Le refactor `reduce` (queue commune de `Mul`/`Sqr`) est
appliqué indépendamment ; les sentinelles de `fermatPostConditionPanics` sont
reconstruites à l'identique.

### R2 — Flexibilité que seuls les tests exercent : **laissée au mainteneur**

Chacun de ces éléments n'a aucun réglage de production, mais des tests
épinglent son comportement (nombre de références en `_test.go` entre
parenthèses). Les retirer supprime les tests avec eux ; ce n'est pas une
décision d'audit.

- `CalibrationOptions.LoadProfile` et `tryUseCachedCalibrationProfile` :
  `RunCalibration` passe toujours `false` (17).
- `MicroBenchmark.runSingleTest` paramètre `parallel`, champ
  `testResult.parallel`, `findParallelCrossover` dont le résultat est jeté
  (`_ =`) : ~50 lignes de « travail futur » documenté comme tel (21).
- `Options.FFTCacheMinBitLen` / `FFTCacheMaxEntries` / `FFTCacheEnabled`,
  `Options.DynamicAdjustmentInterval`, `Options.MemoryLimitBytes` +
  `CanCalculate` : aucun appelant de production ne les renseigne ; R3.6 et
  ADR-0008 R3 ont voulu ces seams.
- `Set/GetDefaultStrassenThreshold` : « test-only safety net » de son propre
  aveu (5).
- `threshold.DynamicThresholdManager.{getThresholds,getStats,reset,…}` (18),
  `tui.LogsModel.entries` (copie O(N) de l'anneau à chaque push, gardée pour
  les tests boîte blanche, 38).

### R3 — Duplications défendues par un ADR ou un golden : **conservées**

- `executeTasks[T, PT]` / `executeMixedTasks` : replier les deux boucles sur
  un `[]task` ajouterait des allocations d'interface sur le chemin matriciel ;
  ADR-0008 R2 a déjà tranché sur ce voisinage. Non mesuré, non appliqué.
- `errors.formatBytesLocal` / `format.FormatBytes` / `memory.formatBytesInternal` :
  triplication imposée par `TestArchitectureLayering` (`errors ↛ format`,
  `config → memory`).
- `completion/fish.go` (sections + `filterFlags`), `zshHelpOverrides`,
  `FlagCompletion.BashGroup` (lu seulement par `powershell.go`) : la sortie est
  épinglée par les goldens ; toucher la forme du script est une décision
  produit.
- `CalibrationStrategy` : `runStrategy` ne reçoit jamais que
  `CompleteStrategy`, et `Name()` n'a pas d'appelant de production. R3.3 l'a
  voulu ainsi ; deux implémentations existent.

## Mesure

A/B `benchstat`, `-count=6` puis `-count=8` en ordre inversé, même session :

| Benchmark | nouveau mesuré en premier (n=6) | base mesurée en premier (n=8) |
|---|---:|---:|
| FastDoubling/1M sec/op | +4,0 % (p=0,004) | −1,7 % (p=0,000) |
| FFTBased/1M sec/op | ~ (p=0,82) | +0,9 % (p=0,038) |
| geomean sec/op | +1,4 % | −0,4 % |
| allocs/op | ~ | ~ |

Le seul changement sur le chemin chaud est `fermat.reduce` ; `FFTBased`, qui
l'exerce le plus, bouge de moins de 1 % dans les deux ordres. Le +4 % de
`FastDoubling` s'inverse quand l'ordre s'inverse : bruit d'ordre, sous le seuil
de 5 % de la Directive #1.

## References

- CHANGELOG « Audit de sur-ingénierie 2026-09-03 ».
- [ADR-0002](0002-recover-strategy.md) §5, [ADR-0008](0008-audit-2026-06-rejected-candidates.md) R2/R3,
  [ADR-0010](0010-audit-2026-09-decisions.md).
