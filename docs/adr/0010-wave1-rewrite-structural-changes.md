# ADR-0010: Vague 1 (PLAN.md) — correspondance des invariants après réécriture du cœur

- **Status**: Accepted
- **Date**: 2026-07-10
- **Deciders**: mainteneur (André-Guy Bruneau)
- **Context source**: PLAN.md révision 3, vague 1 (`bigfft` → `fibonacci/memory` →
  `fibonacci/threshold` → `fibonacci`), commits `ef4a2f4`..`HEAD`

## Context

PLAN.md §9.2 exige qu'un ADR trace la correspondance ancien → nouveau chaque
fois que la vague 1 renomme ou supprime un fichier ou un symbole que
`CLAUDE.md` nomme, pour qu'un futur agent ne conclue pas à tort qu'un
invariant a disparu quand ce n'est que son ancre qui a bougé.

La lecture complète de `bigfft` (16 fichiers de production) et de
`fibonacci` (18 fichiers de production, racine + `memory` + `threshold`) —
prescrite par la Règle d'or de `CLAUDE.md` avant toute modification non
triviale — a confirmé que le cœur est déjà minimal et invariant-correct
(zéro trouvaille ponytail dans `bigfft`, une seule dans `threshold` :
`writtenCount()` mort). La réécriture s'est donc concentrée sur les trois
coupes déjà prévues par PLAN.md §8 plutôt que sur un remplacement mécanique
de code déjà optimal — voir §1.1 de PLAN.md pour la justification de ce
choix, confirmée indépendamment par cette relecture.

Un des trois changements (migration `errgroup`) a d'abord régressé
`allocs/op` de +15 à +19 % sur `BenchmarkFibonacci/FastDoubling/1M`
(mesuré par `benchstat`, gate Directive #1) avant d'être corrigé — voir
Decision ci-dessous pour le détail technique, gardé ici car la cause
(fermetures nécessaires pour `errgroup.Group.Go`) n'est pas évidente et se
reproduira si quelqu'un généralise l'usage d'`errgroup` sur ce chemin chaud.

## Decision

### Table de correspondance

| Ancien symbole/fichier | Statut | Nouveau symbole/fichier |
|---|---|---|
| `internal/parallel` (package entier : `ErrorCollector`, `doc.go`) | **Supprimé** | — |
| `fibonacci/common.go: parallel.ErrorCollector` (usage dans `executeTasks`, `executeMixedTasks`) | Remplacé | `golang.org/x/sync/errgroup.Group` (déjà dépendance directe, déjà en production dans `orchestration`) |
| `fibonacci/common.go: parallel.ErrorCollector` (usage dans `executeParallel3`) | Remplacé | `parallel3Result` (struct locale à `common.go`, **pas** `errgroup`) — voir note perf ci-dessous |
| `fibonacci/common.go: runParallel3Op(ctx, sem, wg, ec, op)` | Resignaturé | `runParallel3Op(ctx, sem, r *parallel3Result, errOut *error, op)` |
| `fibonacci/fibonaccitest/` (package entier : `CoreStub`) | **Supprimé** | `internal/orchestration/contract_test.go: coreStub` (type non exporté, local au seul appelant) |
| `fibonacci/cache_strategy_bigfft.go` | **Renommé** | `fibonacci/cache_tuning.go` |
| `fibonacci/cache_strategy_bigfft_test.go` | **Renommé** | `fibonacci/cache_tuning_test.go` (contenu de `TestDecideCacheTuning` inchangé) |
| `fibonacci.CacheStrategy` (interface), `bigfftCacheStrategy` (struct), `NewBigFFTCacheStrategy()` | **Supprimés** | Appel direct de `decideCacheTuning` (fonction pure, conservée) depuis `ExecuteDoublingLoop` |
| `DoublingFramework.CacheStrategy` (champ exporté) | **Supprimé, sans remplacement** | Le portillon `dynamicThreshold != nil` existant sert seul de garde (il gouvernait déjà tout appel réel à `CacheStrategy.Sample`, le champ était redondant) |
| `TestDoublingFramework_CacheStrategyInjected`, `...SkippedWithoutDTM`, `...Error`, `mockCacheStrategy` | **Supprimés** | `TestDoublingFramework_CacheTuningRunsWithDTM` (smoke test) ; `TestDecideCacheTuning` couvre déjà l'heuristique pure |
| `threshold/metrics_buffer.go: (*MetricsBuffer).writtenCount()` | **Supprimé** (zéro appelant production, trouvé par grep pendant cette relecture) | — |

### Note de performance — pourquoi `executeParallel3` n'utilise pas `errgroup`

`executeTasks`/`executeMixedTasks` migrent sans risque vers `errgroup` : leur
fermeture par élément existait déjà dans le code original (`go func(t PT)
{...}(...)`), donc `errgroup` n'ajoute que le coût de son wrapper interne.

`executeParallel3` est différent : l'original appelait `go
runParallel3Op(ctx, sem, wg, ec, opN)` — un appel direct à une fonction
nommée avec arguments par valeur, qui n'a **pas** besoin d'environnement de
fermeture alloué au tas. Remplacer ceci par `g.Go(func() error {
runParallel3Op(...) })` force **deux** fermetures allouées par appel à
`Go()` (la mienne + celle d'`errgroup` en interne) × 3 appels — mesuré :
+15 à +19 % `allocs/op` sur `FastDoubling/1M`, seul chemin qui exerce
`executeParallel3` dans le benchmark standard (`FFTBasedCalculator` appelle
toujours `ExecuteDoublingLoop` avec `inParallel=false`).

Le correctif retenu (`parallel3Result`, une seule struct portant le
`sync.WaitGroup` et les trois `error`, une seule adresse prise) ramène
`executeParallel3` à **une** allocation par appel — mieux que l'original
(`&sync.WaitGroup{}` + `&parallel.ErrorCollector{}` = deux). `errgroup` reste
un candidat pour toute future généralisation, mais **pas** sur ce site
précis sans revalider par `benchstat`.

## Consequences

### Positive

- `internal/parallel` (95 LOC, réinvention d'`errgroup` déjà utilisé en
  production ailleurs) disparaît — constat ponytail le plus net du dépôt
  (PLAN.md §8).
- `fibonaccitest` (un package entier pour un seul consommateur test) et
  `CacheStrategy` (interface à une seule implémentation de production)
  disparaissent — deux indirections sans bénéfice de polymorphisme réel.
- `writtenCount()` mort supprimé (trouvaille indépendante, hors cut-list
  initiale — preuve que la relecture complète du §5.1 a de la valeur au-delà
  de l'exécution mécanique des coupes déjà listées).
- `executeParallel3` finit avec **moins** d'allocations que l'original.
- `benchstat` (`docs/audits/bench-local-pre.txt` vs `bench-local-post.txt`,
  machine WSL au calme) : geomean `sec/op` −6,7 %, aucune régression
  `> 5 %` avec `p < 0,05` retenue (voir Risks and Mitigations pour le faux
  positif intermédiaire et sa cause).

### Negative / Trade-offs

- `executeParallel3` a maintenant deux implémentations de concurrence
  cohabitant dans le même fichier (`errgroup` pour N variable,
  `parallel3Result` fait main pour 3 fixes) — asymétrie délibérée, documentée
  inline, pas une incohérence de style oubliée.
- Couverture de `executeParallel3` (`common.go`) fluctue 91,2–92,8 % d'un
  run à l'autre : les trois branches `if r.err1/r.err2/r.err3 != nil`
  dépendent de l'ordonnancement réel des 3 goroutines face à l'annulation de
  contexte (l'original collapsait tout en une seule ligne `ec.Err()` via
  `sync.Once`, insensible à l'ordonnancement pour la couverture). N'affecte
  pas le pass/fail des tests, seulement le pourcentage — moyenne sur 11 runs
  ≈ 92,1 %, dans la tolérance −1,0 pp de PLAN.md §4.2.

### Risks and Mitigations

- **Risque** : une migration future généralise `errgroup` à
  `executeParallel3` sans revalider par `benchstat`, réintroduisant la
  régression corrigée ici. **Mitigation** : commentaire inline sur
  `runParallel3Op` et cette ADR.
- **Faux positif rencontré pendant cette vague** : deux comparaisons
  `benchstat` intermédiaires ont montré des régressions `sec/op` jusqu'à
  +19 % sur les benchmarks 10M, non reproductibles — tracées à une VM WSL
  qui redémarre à chaque invocation `wsl bash -lc` (bootstrap systemd/snapd
  concurrent au premier run) et à un délai de stabilisation insuffisant
  (20 s) au second. Le relevé retenu utilise 90 s de stabilisation +
  vérification de `/proc/loadavg` avant mesure. **Mitigation pour l'avenir** :
  toujours vérifier `uptime`/`loadavg` avant un `benchstat` de gate, pas
  seulement supposer la machine calme.

## Alternatives Considered

- **Garder `errgroup` partout (y compris `executeParallel3`)** — rejeté :
  régression `allocs/op` mesurée et confirmée reproductible sur trois runs
  indépendants (+15 %, +17 %, +19 %), qui viole la Directive #1
  (> 5 % = blocage) sans bénéfice de simplicité proportionné (le hand-roll
  fait 15 lignes de plus que la version `errgroup`, mais reste plus court
  que l'original `internal/parallel`-based).
- **Réécriture mécanique intégrale de `bigfft`/`fibonacci`** — rejeté : la
  relecture complète (§5.1) n'a trouvé aucune ligne dont la réécriture
  aurait une justification autre que « produire un diff » ; voir PLAN.md
  §1.1 pour l'objection déjà actée par le mainteneur. Documenté ici comme
  hypothèse d'exécution nommée (CLAUDE.md global : « prends l'option la plus
  réversible et ouvre ton rapport en nommant l'hypothèse retenue »).

## References

- Implementation file(s) : `internal/fibonacci/common.go`,
  `internal/fibonacci/cache_tuning.go`, `internal/fibonacci/doubling_framework.go`,
  `internal/fibonacci/threshold/metrics_buffer.go`,
  `internal/orchestration/contract_test.go`
- Test(s) : `TestExecuteTasksParallel`, `TestExecuteMixedTasksParallel`,
  `TestDoublingFramework_CacheTuningRunsWithDTM`, `TestDecideCacheTuning`,
  `TestCalculatorsAgainstGoldenFile`, `TestFastDoubling_DynamicThresholds_Correctness`
- Related ADR(s) : ADR-0009 (audit 2026-07, cut-list ponytail dont ce plan
  hérite), PLAN.md §8 (cut-list), §9.2 (obligation de cette ADR)
