# CONVENTIONS — Règles partagées de la mission d'audit (à lire EN PREMIER)

> Ce document est le « CLAUDE.md de la mission ». **Tous** les sous-agents le lisent avant toute analyse.

## A. Langue & style

- **Français canadien** pour toute la rédaction. Les **noms d'outils et de commandes restent en anglais** (`go vet`, `staticcheck`, `go test -race`, `pprof`, API, RFC, `sync.Pool`, etc.).
- **Pyramide inversée** : conclusion / insight-clé d'abord, justification ensuite.
- **Premiers principes**, rigueur, concision. Pas de remplissage.
- **Pas d'emoji**. Pas de réécriture du code (audit en lecture seule).

## B. Marqueurs épistémiques — OBLIGATOIRES sur chaque constat

Apposer exactement un marqueur par constat :

- **[confirmé]** — prouvé par exécution d'une commande ou lecture directe non ambiguë du code.
- **[probable]** — forte présomption étayée mais non exécutée/non isolée.
- **[hypothèse]** — piste plausible nécessitant validation.
- **[à vérifier]** — non vérifiable dans le sandbox (ex. race detector indisponible) ; préciser **comment** lever le doute.

## C. Barème de sévérité (unique pour tous les axes)

| Sévérité | Définition |
|---|---|
| **CRITIQUE** | Corruption de résultat, data race, débordement silencieux, deadlock, fuite mémoire/goroutine, panic non maîtrisée sur chemin de production. |
| **MAJEUR** | Perte de correction ou de performance notable, API risquée, invariant fragile, couverture absente sur chemin critique. |
| **MINEUR** | Style, idiome, lisibilité, nommage, documentation manquante non bloquante. |
| **INFORMATIF** | Observation neutre, piste d'amélioration, contexte. |

## D. Format de chaque constat (gabarit imposé)

Chaque constat est un bloc structuré :

```
### [ID] Titre court
- **Sévérité** : CRITIQUE | MAJEUR | MINEUR | INFORMATIF
- **Axe** : (1 Correctness | 2 Concurrence | 3 Performance | 4 Idiomatique | 5 Structure/Tests/CI)
- **Emplacement** : chemin/fichier.go:ligne (ou plage)
- **Preuve** : extrait de code OU sortie de commande (bloc ```), citée fidèlement
- **Impact** : conséquence concrète
- **Recommandation** : action corrective précise et vérifiable
- **Marqueur** : [confirmé] | [probable] | [hypothèse] | [à vérifier]
```

Convention d'**ID** : `A<axe>-<NN>` — ex. `A1-01` (Correctness #1), `A3-04` (Performance #4).

## E. Sorties — un fichier par sous-agent (noms imposés, zéro conflit)

| Sous-agent | Fichier de sortie |
|---|---|
| 1 — Correctness & exactitude | `audit/01_correctness.md` |
| 2 — Concurrence & data races | `audit/02_concurrence.md` |
| 3 — Performance & benchmarks | `audit/03_performance.md` |
| 4 — Idiomatique & qualité | `audit/04_idiomatique.md` |
| 5 — Structure, tests & CI | `audit/05_structure_tests_ci.md` |

**Interdits absolus** : modifier un fichier source (`*.go`, `go.mod`, `go.sum`), `testdata/` (golden **immuable**), ou tout fichier hors `audit/`. Écrire **uniquement** son propre fichier de sortie. Aucun agent n'écrit dans le fichier d'un autre.

Chaque fichier de sortie débute par : (1) un **verdict d'axe** de 2–4 lignes (pyramide inversée), puis (2) un **tableau récapitulatif** des constats (ID, sévérité, titre), puis (3) le détail des constats par sévérité décroissante.

## F. Invariants du dépôt à RESPECTER (cf. `CLAUDE.md` racine)

Ne pas recommander de « corriger » ce qui suit — ce sont des invariants **volontaires et testés** :

- `fastdoubling.go` : `finalizeStateRelease` appelle `clearStateAliases` inconditionnellement (gardé par `TestReleaseState_OverLimit_AliasesCleared`).
- `memory/gc_control.go` : `WithGC(fn)` panic-safe ; `Begin`/`End` directs sont `Deprecated`.
- `bigfft/fft_cache.go` : `putByKey` alloue **toujours** un backing frais (anti-aliasing, Audit-PRD E1-R4) — ne pas réintroduire de recyclage à l'éviction.
- `bigfft/fft.go`, `fft_recursion.go` : seuils en `atomic.Int64/Uint64` **privés** + accesseurs ; pas de globaux mutables non synchronisés (ADR-0003).
- `bigfft/fft.go` (`Mul`/`Sqr`...) : le `recover()` re-propage les sentinels `isFermatPostConditionPanic` (gardé par `TestFermatPostConditionPanicClassifier`, ADR-0002).
- `threshold/manager.go` : champs `atomic.*` ; n'importe **pas** `internal/config` (passer par `threshold.SetTuning`).
- `errors/errors.go` : n'importe **pas** `internal/format` (helper local `formatBytesLocal` ; gardé par `TestArchitectureLayering`).
- `tui/` : n'importe **pas** `internal/fibonacci` directement (via aliases `orchestration.*` ; gardé par `TestArchitectureLayering`).
- Golden `internal/fibonacci/testdata/fibonacci_golden.json` : **immuable** sans ADR. Étendu à F(50k/100k/200k) sous ADR-0004 §B5.
- **Pas de nouveaux globaux dans `bigfft/`** ; régression perf **> 5 % = bloquant** (comparer aux baselines `docs/audits/`).

Un constat qui propose de modifier un invariant ci-dessus doit l'expliciter et le justifier comme **proposition d'ADR**, pas comme « bug ».

## G. Réalités d'exécution (issues du bootstrap — `audit/00_bootstrap.md`)

- **Commit** : `866b8cd` (2026-05-24). **Go** : `go1.26.3`, module cible `go 1.26.0`.
- **`go build ./...`** : propre (exit 0). **`go vet`**, **`gosec`**, **`govulncheck`** : disponibles.
- **`go test -race`** : **INDISPONIBLE** (CGO_ENABLED=0, pas de gcc/clang). Tout constat de race → **[à vérifier]** (reconfirmer sous Linux/WSL).
- **`gofmt -l .`** : **faux positifs massifs** dus à `core.autocrlf=true` (working tree CRLF / index LF). Le code commité est gofmt-propre. **Ne pas** rapporter « tout le code est non formaté ».
- **`staticcheck` / `golangci-lint`** : binaires préinstallés refusent go1.26 (compilés en go1.25) ; recompilés localement avec go1.26.3 (voir bootstrap §5). `.golangci.yml` est en schéma v1.
