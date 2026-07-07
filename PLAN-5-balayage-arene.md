# PLAN-5 — Balayage du multiplicateur d'arène ×15 (protocole ADR-0009 R4)

**Rang levier : 5/5 (le plus gros gain potentiel, le plus gros effort, l'issue la plus incertaine).** Effort : ½–1 journée (surtout du temps de bench). Impact potentiel : à F(10M), la réduction ×15→×5 économisait ~8 Mo (−31 % B/op FFT) mais coûtait +18 à +34 % CPU → **rejetée** (ADR-0009 R4). L'ADR laisse explicitement la porte ouverte : « À revoir si : un balayage complet du multiplicateur (plusieurs paliers, benchstat à chacun) démontre un palier intermédiaire à gain mémoire net sans régression CPU > 5 % ». Ce plan EST ce balayage. **Une issue « ×15 confirmé, aucun palier viable » est un livrable valide** — elle clôt R4 définitivement par un addendum ADR au lieu d'un point ouvert.

> **Règle d'or CLAUDE.md** : ce plan touche `internal/fibonacci/` → gate benchstat impératif (Directive #1, régression > 5 % = blocage), golden immuable (Directive #2). Travailler sur une **branche dédiée** (travail expérimental — section Workflow de CLAUDE.md) ; ne fusionner sur `main` que la variante gagnante OU le seul addendum ADR.

## Objectif

Mesurer les paliers m ∈ {12, 10, 8, 6} du multiplicateur d'arène contre la base ×15 (A/B même machine, même session), puis :

- **si un palier gagne** (CPU ≤ bruit, mémoire nettement réduite) : l'adopter, mettre à jour les invariants documentés, addendum ADR-0009 ;
- **sinon** : revenir à ×15 et écrire l'addendum ADR-0009 qui clôt R4 avec les tableaux de mesure.

## Fichiers concernés

**Code (les deux miroirs du multiplicateur — à changer EN MÊME TEMPS)** :

1. `internal/fibonacci/fastdoubling.go` — `acquireSizingForN` (lignes 296–308) : **trois** littéraux (`maxReasonableWords / 15` ligne 301, `maxReasonableWords/15` ligne 304, `wordsNeeded * 15` ligne 307) + les commentaires lignes 285–295 (« ×15 multiplication », « 15 temporaries »).
2. `internal/fibonacci/memory/arena.go` — `arenaTotalWords` (lignes 16–32) : **deux** littéraux (`maxReasonableWords/15` ligne 28, `wordsPerInt * 15` ligne 31) + commentaires lignes 16–19.

**Tests miroirs (sinon rouges après changement)** :

3. `internal/fibonacci/fastdoubling_test.go` ligne ~95 (`return w, w * 15`) et ligne ~106 (attend `maxReasonableWords/15`).
4. `internal/fibonacci/memory/arena_test.go` ligne ~123 (`... + 1) * 15`).
5. `internal/fibonacci/state_cache_test.go` lignes 52–61 — **voir cas limite n° 2, c'est le piège principal.**

**Docs (uniquement à la conclusion)** :

6. `docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md` — addendum sous R4.
7. `CLAUDE.md` — entrée invariants `fibonacci/fastdoubling.go`, puce « Multiplicateur d'arène ×15 ».

## Étapes

### Étape 0 — Préparation

```powershell
git checkout -b exp/arena-multiplier-sweep
# benchstat est absent de WSL (vérifié 2026-07-07) :
wsl -e bash -lc "go install golang.org/x/perf/cmd/benchstat@latest"
```

Machine au calme (rien d'autre ne tourne), portable branché secteur. Tous les benchs de ce plan s'exécutent **dans WSL** pour un environnement homogène. Répertoire WSL : `/mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo`.

### Étape 1 — Base ×15 (ne PAS utiliser `docs/audits/bench-baseline.txt`)

```bash
# (dans WSL, à la racine du dépôt)
go test -bench='BenchmarkFibonacci/(FastDoubling|FFTBased)' -benchmem \
  -benchtime=1s -count=10 -run='^$' ./internal/fibonacci/ | tee /tmp/sweep-m15.txt
```

**Vérifier** : 10 itérations par benchmark dans le fichier ; aucune erreur.

### Étape 2 — Pour chaque palier m ∈ {12, 10, 8, 6}, dans cet ordre

1. Remplacer **les cinq littéraux** `15` listés ci-dessus par `m` (les 3 de `fastdoubling.go`, les 2 de `arena.go`) et ajuster les 2 tests miroirs (points 3–4). Adapter les commentaires (« m temporaries »).
2. Appliquer le correctif du test de cache d'état (cas limite n° 2 ci-dessous).
3. Tests gardiens + suite du package :

```bash
go test -count=1 ./internal/fibonacci/... 
# doit inclure au vert : TestReleaseState_OverLimit_AliasesCleared,
# TestCalculatorStateCache_OverLimitNotCached, TestStateBump_*,
# TestAcquireSizingForN*, et le golden (fibonacci_golden.json, jamais -update).
```

4. Bench du palier :

```bash
go test -bench='BenchmarkFibonacci/(FastDoubling|FFTBased)' -benchmem \
  -benchtime=1s -count=10 -run='^$' ./internal/fibonacci/ | tee /tmp/sweep-m<m>.txt
benchstat /tmp/sweep-m15.txt /tmp/sweep-m<m>.txt
```

5. **Règle de décision (mécanique, pas de jugement)** sur le geomean sec/op :
   - Δ > +5 % → palier **rejeté** immédiatement (Directive #1). Si m=12 est déjà > +5 %, exécuter quand même m=10 (le comportement n'est pas forcément monotone), mais si deux paliers consécutifs dépassent +5 %, arrêter le balayage.
   - +2 % < Δ ≤ +5 % → **contre-mesure thermique obligatoire** : relancer la paire dans l'ordre inverse (palier d'abord, puis re-mesurer ×15 en re-committant les littéraux). Retenir le pire des deux verdicts.
   - Δ ≤ +2 % (bruit) **et** B/op FastDoubling/10M ou FFTBased/10M réduit d'au moins 15 % → palier **candidat**.
6. `git stash` ou commit local du palier étiqueté (`wip: m=<m>`), puis passer au suivant **en repartant de ×15** (chaque palier se compare à la même base, pas au palier précédent).

### Étape 3 — Conclusion

**Cas A — au moins un palier candidat.** Prendre le plus petit m candidat (gain mémoire maximal à CPU propre). Puis :

1. Re-valider ce m complet : `wsl go test -race ./...`, golden, `bash scripts/check.sh` (WSL).
2. Confirmation finale dédiée : nouvelle paire benchstat ×15 vs m, `-count=10`, ordre inversé une fois — les deux passes doivent rester ≤ +2 %.
3. Mettre à jour `CLAUDE.md` (invariant « Multiplicateur d'arène ×15 » → nouvelle valeur + renvoi à l'addendum) et écrire l'addendum ADR-0009 R4 (tableaux benchstat des DEUX ordres d'exécution).
4. `docs/audits/bench-baseline.txt` : régénérer via `make bench-baseline` **après** la fusion (la baseline doit refléter `main`).
5. Merge sur `main`, commit `perf(fibonacci): arena multiplier 15 -> <m> (ADR-0009 R4 sweep)` avec justification complète (Directive #5).

**Cas B — aucun palier candidat.** `git checkout main`, supprimer la branche de code, et committer UNIQUEMENT :

- addendum ADR-0009 sous R4 : « Balayage complet exécuté le <date> (m ∈ {12,10,8,6}, benchtime=1s, count=10, ordre inversé pour les paliers ambigus) — aucun palier sans régression CPU > 5 %. R4 clos. » + tableaux benchstat.
- `CLAUDE.md`, puce ×15 : remplacer « ne pas le réduire sans un balayage complet… » par « balayage complet exécuté (<date>, addendum ADR-0009) : aucun palier viable, ×15 définitif ».

Commit : `docs(adr): close ADR-0009 R4 — full arena multiplier sweep, x15 confirmed`.

## Cas limites (chacun a déjà cassé quelqu'un, ou l'aurait fait)

1. **Les clamps anti-débordement doivent suivre le multiplicateur.** Dans `acquireSizingForN`, les deux `maxReasonableWords/15` (lignes 301 et 304) et le `*15` (ligne 307) forment un invariant : `totalWords ≤ maxReasonableWords` sans débordement `int`. Changer le `*15` sans les `/15` fausse le clamp ; le test `TestAcquireSizingForN` sur `math.MaxUint64` (fastdoubling_test.go:106) attend précisément `maxReasonableWords/15` — il rougira si un seul des trois littéraux est oublié. Même structure dans `arena.go` (lignes 28 et 31).
2. **`state_cache_test.go` a une prémisse calibrée sur ×15** (lignes 52–61) : N=30M donne `30e6 × 0.694 / 64 × 15 ≈ 4,9 M mots`, juste AU-DESSUS de `maxCachedArenaWords` (4 M) et loin SOUS `maxArenaPoolWords` (50 M). À m=12 le produit tombe à ≈ 3,9 M < 4 M : le test échoue sur son propre garde-fou `t.Fatalf("test premise broken...")`. **Correctif attendu : augmenter N du test** pour que `N × 0.694 / 64 × m > 4,2e6` tout en restant < 50 M mots (ex. m=12 → N=45_000_000 donne ≈ 5,9 M mots : OK). **Interdit** : affaiblir ou supprimer le `t.Fatalf` — c'est lui qui empêche le test de devenir un faux vert vide.
3. **`maxCachedArenaWords` (4 M) et `maxArenaPoolWords` (50 M) restent FIXES pendant tout le balayage.** Ce sont des bornes en octets retenus, pas en slots ; les changer en même temps que m rendrait les mesures ininterprétables (deux variables à la fois). Noter en revanche dans l'addendum que réduire m déplace le seuil de N au-delà duquel un état n'est plus retenu dans le slot GC-immune — c'est un des mécanismes qui expliquait la régression ×5.
4. **Ne jamais comparer à `docs/audits/bench-baseline.txt`** : autre session, conditions différentes. Le protocole est A/B frais, même machine, même session — c'est ce qui a permis à l'audit de prouver que la régression ×5 « survit à l'inversion de l'ordre d'exécution » (non thermique). Reproduire cette inversion pour tout verdict ambigu.
5. **PowerShell mange `-bench=.`** (CLAUDE.md, Points d'attention) : si un bench est lancé côté Windows, toujours donner le motif complet `-bench='BenchmarkFibonacci/...'`. Le plan évite le problème en passant par WSL.
6. **Le golden est immuable** (Directive #2) : si un test golden rougit pendant le balayage, c'est un bug introduit par l'édition (probablement un clamp), jamais un fichier à régénérer.
7. **ADR-0009 R4 prédit la forme de la courbe** : ×5 = +18/+34 %. Si m=6 affiche des chiffres similaires, c'est cohérent ; si m=12 affiche +30 %, suspecter une erreur d'édition (littéral oublié, tests miroirs incohérents) avant de conclure.
8. **`-benchmem` est obligatoire** : sans lui, aucune donnée B/op, donc aucun moyen de constater le gain mémoire qui justifierait un candidat.
9. Chaque palier repart de la base ×15 propre (`git stash`/`checkout`), jamais du palier précédent — sinon les diffs s'accumulent et un littéral oublié devient invisible.

## Critères d'acceptation

1. **Traçabilité** : un fichier `/tmp/sweep-m15.txt` + un par palier testé existent ; les sorties `benchstat` (les deux ordres pour les paliers ambigus) sont archivées dans l'addendum ADR.
2. **Cas A (changement adopté)** : benchstat final ×15 vs m ≤ +2 % geomean sec/op dans les DEUX ordres d'exécution ; B/op réduit ≥ 15 % sur au moins un benchmark 10M ; `wsl go test -race ./...` vert ; `bash scripts/check.sh` PASS ; golden intact (`git diff --stat -- internal/fibonacci/testdata` vide) ; CLAUDE.md + addendum ADR-0009 mis à jour ; `bench-baseline.txt` régénérée post-merge.
3. **Cas B (×15 confirmé)** : `git diff main -- internal/` vide (aucun code modifié) ; addendum ADR-0009 committé avec les tableaux ; la phrase « ne pas le réduire sans un balayage complet » de CLAUDE.md remplacée par la clôture datée.
4. Dans les deux cas : `pwsh scripts/check.ps1` PASS sur `main` après merge ; commit(s) poussés.
