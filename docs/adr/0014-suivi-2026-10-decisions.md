# ADR-0014: Suivi 2026-10 — horloge, `go:linkname`, défauts, porte d'architecture

- **Status**: Accepted
- **Date**: 2026-10-10
- **Deciders**: André-Guy Bruneau (mainteneur)
- **Context source**: constats laissés ouverts par l'évaluation du 2026-09-15
  ([§ 7](../audits/evaluation-academique-2026-09-15.md)) et analyse du dépôt du
  2026-10-10. Base : `main` à `5c4d765`.

## Context

L'analyse du 2026-10-10 a relevé plusieurs points ouverts. Deux d'entre eux
appelaient une décision, et deux autres sont apparus en les traitant :

- `fibcalc -algo all -n 100000` affichait par intermittence `< 1µs` pour un
  calcul d'environ 540 µs. Sous Windows, l'horloge du runtime Go (`time.Now`,
  lectures monotone et murale confondues) avance par pas de 513 à 531 µs
  (mesuré). Les durées de calcul affichées sont donc arrondies au pas, et
  celles des multiplications de 500 et 2 000 mots que chronomètre la calibration
  rapide (40 à 340 µs) se lisent comme 0 ou comme un pas.
- `internal/bigfft` atteint six fonctions internes de `math/big` par
  `go:linkname`, avec un repli `-tags purego` testé en CI ; le commentaire
  d'avertissement présentait le risque sans le mesurer.
- `fibcalc` sans argument calcule F(10⁸) avec les trois calculateurs ; le coût
  n'était consigné nulle part.
- `internal/arch_test.go` interdisait six importations nommées et laissait
  passer toute autre (constat de l'évaluateur indépendant).

## Decision

| # | Question | Décision |
|---|---|---|
| D1 | Horloge des mesures courtes | Nouveau paquet feuille `internal/clock` : `QueryPerformanceCounter` sous Windows, comme le paquet `testing` de Go (`src/testing/testing_windows.go`), et l'horloge du runtime ailleurs. Il sert aux durées de calcul affichées (`orchestration`, y compris `-last-digits`), à la boucle du micro-benchmark et aux balayages de calibration. Les échéances, délais et horodatages restent sur `time`. `CurrentProfileVersion` ne change pas : les profils existants expirent d'eux-mêmes après `-profile-max-age` (168 h par défaut). |
| D2 | `go:linkname` ou `purego` par défaut | **Garder `go:linkname`.** `math/big` (go1.27.0) maintient exprès les six symboles accessibles et nomme `remyoudompheng/bigfft` parmi leurs dépendants. [`bench-purego-2026-10.txt`](../audits/bench-purego-2026-10.txt) ne montre pas `purego` gratuit : pas d'écart sur `FFTBased/10M` (p = 0,937), +27,55 % sur `FastDoubling/10M` (p = 0,041, base dispersée à ± 39 %). Le repli reste testé en CI. |
| D3 | Défauts `-n 100000000 -algo all` | **Garder**, documenter le coût : 2,1 s et 675 Mo de mémoire résidente sur l'hôte de référence ([`mem-baseline-2026-09.txt`](../audits/mem-baseline-2026-09.txt), addendum). L'image Docker démarre sur `--help`. Changer un défaut documenté aurait changé le comportement des appels existants sans mesure qui l'exige. |
| D4 | Porte d'architecture | **Liste blanche exhaustive** (`allowedImports`) à la place des six règles d'interdiction : chaque paquet du module est classé, chacune de ses importations internes est nommée, et une arête listée qui disparaît fait échouer le test. La liste est le graphe exact de [`dependency-graph.md`](../architecture/dependency-graph.md) (47 arêtes). |

## Consequences

### Positive

- Sous Windows, les durées affichées descendent à la résolution du compteur
  (100 ns sur l'hôte de référence) ; `< 1µs` ne s'affiche plus pour un calcul
  réel.
- La calibration rapide mesure enfin ses deux plus petites tailles : dix passes
  consécutives donnent 115 200 bits dix fois, contre sept sur dix avant
  ([`microbench-stability-2026-09.txt`](../audits/microbench-stability-2026-09.txt),
  addendum). La confiance reste sous le seuil, donc la passe escalade toujours,
  mais elle le fait maintenant sur une mesure.
- Toute nouvelle flèche d'importation passe par une modification visible de
  `internal/arch_test.go`.

### Negative / Trade-offs

- Un paquet de plus et un appel système par lecture d'horloge sous Windows ;
  négligeable devant les durées mesurées (au moins des dizaines de µs).
- Retirer une importation exige aussi de retirer l'arête de la liste blanche.

### Risks and Mitigations

- Le résultat de D2 repose sur six tours sur un seul hôte. Une version de Go
  qui retirerait les symboles casserait la compilation par défaut, pas le
  comportement ; `-tags purego` reste le recours, et le job CI le compile.

## Alternatives Considered

- **D1, n'afficher que « < 1 ms » sous Windows** : cela aurait rendu le symptôme
  honnête sans corriger la calibration, qui compare des temps.
- **D2, `purego` par défaut et suppression de `arith_decl.go`** : ce serait moins
  de code et plus aucun `go:linkname`, mais le coût n'est pas établi comme nul.
- **D3, `-n 10000000` par défaut** : cela changerait un comportement documenté
  sans plainte ni mesure qui l'exige.

## References

- Implementation file(s) : `internal/clock/`, `internal/orchestration/orchestrator.go`,
  `internal/orchestration/lastdigits.go`, `internal/calibration/{microbench,calibration,runner}.go`,
  `internal/bigfft/arith_decl.go`, `internal/arch_test.go`
- Test(s) : `internal/clock/clock_test.go`, `internal/arch_test.go`,
  `internal/cli/presenter_test.go` (alignement des colonnes µs)
- Related ADR(s) : [ADR-0013](0013-evaluation-2026-09-decisions.md)
