# Prompt — Refactorisation, optimisation et finalisation FibGo

> Prompt à soumettre tel quel à Claude Code CLI dans une session fraîche, à la racine du repo. Suppose les outils standard (Agent, Bash, Read, Edit, Grep, Glob, TodoWrite) disponibles.

---

## Mission

Mener un audit complet du repo FibGo (`github.com/agbru/fibcalc`) puis exécuter **uniquement** les améliorations justifiées par les findings. La sortie attendue est un repo plus simple, plus correct et plus rapide — **pas plus volumineux**. Toute modification doit tracer à un finding documenté.

**Référentiel comportemental** : `CLAUDE.md` (racine) et `~/.claude/CLAUDE.md`. En cas de conflit avec ce prompt, le CLAUDE.md projet l'emporte.

## Garde-fous (non négociables)

1. **Tests golden** : `internal/fibonacci/testdata/fibonacci_golden.json` doit passer avant et après chaque commit. Vérifier avec `make test`.
2. **Bench non régressant** : `make bench-versioned` ne doit pas régresser de plus de 5 % sur les benchmarks Fast Doubling, FFT et Matrix. Capturer une baseline avant toute modification de `internal/fibonacci/` ou `internal/bigfft/`.
3. **Lint clean** : `make lint` (24 linters) doit passer. Pas de nouveaux `//nolint` sans justification une-ligne.
4. **Étanchéité des couches** : rien dans `internal/` ne doit fuiter vers `cmd/` directement. Pas de nouveau couplage cmd → fibonacci/bigfft sans intermédiaire.
5. **Pas d'allocations gratuites** : tout patch dans le hot path (`internal/fibonacci/`, `internal/bigfft/`) doit préserver ou améliorer `-benchmem`.
6. **Modifications chirurgicales** : chaque ligne modifiée trace à un finding. Pas de réécriture stylistique « tant qu'on y est ».
7. **Aucune destruction** sans confirmation : suppression de packages, breaking changes API publique, migration go.mod → demander.

---

## Phase 1 — Audit parallèle (agents en parallèle, lecture seule)

Dispatcher **7 agents `Explore` en parallèle** dans un seul message. Chaque agent retourne un rapport structuré ≤ 400 mots avec cette forme :

```
### Findings
- [SEV: high|med|low] [PATH:line] Description courte. Suggestion concrète.
### Verified clean
- Liste des dimensions vérifiées sans finding.
### Out-of-scope
- Choses remarquées mais hors mandat.
```

### Mandats des agents

**Agent A — Code mort & exports inutilisés**
Détecte fonctions/types/constantes/variables exportés non utilisés hors de leur package, imports inutilisés, fichiers orphelins. Outils : `go vet`, `staticcheck`, `unused`, grep ciblé. Ignorer le code intentionnellement public (API publique CLI, doubles de test).

**Agent B — Violations architecturales**
Vérifie l'étanchéité Clean Architecture. Cherche : imports `cmd/` → `internal/fibonacci|bigfft` directs (court-circuit des couches), cycles d'imports, dépendances inversées (couche basse importe couche haute). Lister chaque violation avec chemin source → cible.

**Agent C — Duplication & sur-abstraction**
Cherche : interfaces à un seul implémenteur jamais mockées, wrappers triviaux, patterns « Builder/Factory » pour ≤ 2 cas d'usage, code dupliqué (≥ 20 lignes similaires) entre fichiers. Distinguer duplication accidentelle vs duplication justifiée par perf.

**Agent D — Dette de lint & complexité**
Liste tous les `//nolint` du repo avec justification (présente/absente/obsolète). Identifie les fonctions qui dépassent : cyclomatique 15, cognitive 30, longueur 100 lignes / 50 statements (cf. `.golangci.yml`). Pour chaque outlier : suggérer extract-method ou justifier conservation.

**Agent E — Couverture & qualité des tests**
Lance `make coverage` et reporte les packages < 70 % couverture. Vérifie présence : tests parallèles (`t.Parallel()`), property tests, fuzz tests, golden tests, edge cases (n=0, n=1, n négatif, n très grand). Identifier les `t.Skip` permanents.

**Agent F — Performance : allocations & contention**
Cible `internal/fibonacci/`, `internal/bigfft/`, `internal/orchestration/`. Cherche : allocations `big.Int` sans pool, slices sans `prealloc`, mutex contention possible (locks tenus pendant compute), goroutines sans contrôle de cycle de vie, `sync.Pool` mal dimensionnés. Croiser avec `make benchmark -benchmem` (lancer si nécessaire en short).

**Agent G — Documentation : fraîcheur & WHY vs WHAT**
Vérifie : présence d'un `doc.go` par package public, cohérence du commentaire de package avec le code, commentaires inline qui paraphrasent le code (à supprimer) vs commentaires qui justifient un choix non évident (à garder). Vérifier que `docs/ARCH.md`, `BUILD.md`, `PERFORMANCE.md`, `TESTING.md` ne contiennent pas d'infos contredites par le code actuel.

---

## Phase 2 — Synthèse & plan

Après réception des 7 rapports :

1. **Agréger** les findings dans un `TodoWrite` unique. Une todo par finding actionnable, classée par sévérité.
2. **Filtrer** :
   - Garder : findings high (correctness, perf, lint blocking) + findings med justifiés.
   - Écarter : findings low purement stylistiques, suggestions de « modernisation » sans gain mesurable, refactors qui violent « Surgical Changes ».
3. **Présenter le plan** à l'utilisateur sous forme tabulaire :

   | # | Sévérité | Dimension | Fichier(s) | Action | Risque |
   |---|---|---|---|---|---|
   
4. **Attendre validation** avant Phase 3 si l'une de ces conditions est vraie :
   - Plus de 10 findings actionnables
   - Au moins un finding touche l'API publique
   - Au moins un finding nécessite migration de tests
   
   Sinon, enchaîner directement.

---

## Phase 3 — Exécution chirurgicale

Pour chaque todo validée :

1. **Capturer baseline** si zone de perf : `make bench-versioned` → fichier sous `bench/baseline-<date>.txt`.
2. **Implémenter** la modification minimale qui résout le finding. Pas plus.
3. **Vérifier localement** : `make test-short` au minimum, `make test` pour changements algorithmiques.
4. **Marquer la todo `completed` immédiatement**, pas en lot.

**Parallélisation** : grouper les findings par package non-chevauchant. Lancer en parallèle uniquement si les diffs ne se touchent pas. En cas de doute, séquentiel.

**Rollback automatique** si :
- Un test golden échoue après modification → `git restore` le fichier, marquer la todo `blocked`, signaler.
- Un benchmark régresse > 5 % → idem.

## Phase 4 — Vérification finale

Lancer **en parallèle** dans un seul message :

```
make lint
make test
make bench-versioned
```

Comparer la sortie de `bench-versioned` à la baseline capturée en Phase 3. Échec si :
- Lint non-clean
- Tests rouges
- Régression bench > 5 % sur l'un des trois algorithmes principaux

En cas d'échec, ne pas tenter de « corriger » dans la foulée — rapporter à l'utilisateur la nature de l'échec et attendre instruction.

## Phase 5 — Rapport final

Produire un récapitulatif ≤ 300 mots :

```
## Résumé
- N findings audités, M actionnables, K appliqués
- Lignes touchées : +X / -Y sur Z fichiers
- Bench delta : Fast Doubling Δ%, FFT Δ%, Matrix Δ%
- Coverage avant → après

## Modifications par dimension
- [Dimension] : findings adressés, fichiers, gain mesuré

## Reportés / Écartés
- [Finding] : raison du report (hors scope, risque, demande utilisateur requise)

## Suggestions hors mandat
- (Mentionner sans agir, conformément à « Surgical Changes »)
```

---

## Antipatterns à éviter explicitement

- ❌ Renommer pour « cohérence » sans bug fonctionnel à la clé.
- ❌ Extraire des helpers utilisés à un seul endroit.
- ❌ Ajouter des interfaces « pour la testabilité » sans test à écrire dans la même PR.
- ❌ Migrer un package vers un nouveau pattern (ex. `errors.Join`, `slog`) sans demande explicite.
- ❌ Supprimer du code mort préexistant non lié à un finding (le mentionner uniquement).
- ❌ Toucher au formatage/imports dans des fichiers où aucune modification fonctionnelle n'est faite.
- ❌ Bumper les dépendances `go.mod` sans demande.
- ❌ Créer de nouveaux fichiers `*.md` sauf si un finding documentation l'exige.

## Critères de succès vérifiables

À la fin du processus, l'ensemble des conditions suivantes doit tenir :

- [ ] `make all` vert
- [ ] `make lint` zéro warning nouveau
- [ ] `make bench-versioned` sans régression > 5 % sur Fast Doubling, FFT, Matrix
- [ ] Couverture globale ≥ couverture initiale
- [ ] Aucune ligne du diff ne trace à autre chose qu'un finding listé en Phase 2
- [ ] Rapport Phase 5 produit
- [ ] `git status` clean (rien d'oublié non-commité)

---

**Démarrage** : commencer par lire `CLAUDE.md` puis lancer la Phase 1 immédiatement avec les 7 agents en parallèle dans un seul message. Pas de pré-questions sauf si le repo n'est pas dans l'état attendu (branche divergente, working tree sale, etc.).
