# 11 — Étanchéité des couches

Date : 2026-04-28
Périmètre : `cmd/`, `internal/` (22 packages).

## Test 1 — Imports depuis cmd/

- `cmd/fibcalc/main.go` (l. 10) : `internal/app` — **conforme**.
- `cmd/fibcalc/main_test.go` : aucun import `internal/*`.
- `cmd/generate-golden/main.go`, `main_test.go`, `doc.go` : aucun import `internal/*` (oracle indépendant volontaire, cf. doc.go P2-04).

VERDICT : **OK**. Seul `internal/app` est importé par `cmd/`. La couche d'interface est respectée. À noter : `internal/cli` n'est jamais importé directement par `cmd/` (il transite via `app`), ce qui renforce l'étanchéité au-delà de la règle.

## Test 2 — Imports inverses (internal/ → cmd/)

`grep "github.com/agbru/fibcalc/cmd"` sous `internal/` : 0 match.

VERDICT : **OK**. Aucune fuite inverse.

## Test 3 — Fuite de types internes vers cmd/

| Type | Défini dans | Utilisé par cmd/ | OK ? |
|------|-------------|------------------|------|
| `app.Application` (via `app.New`) | `internal/app` | `cmd/fibcalc/main.go` | OK (façade) |
| `app.HasVersionFlag`, `app.PrintVersion`, `app.IsHelpError` | `internal/app` | `cmd/fibcalc/main.go` | OK (façade) |

Aucun type de `fibonacci`, `bigfft`, `orchestration`, `config`, `cli` n'est référencé depuis `cmd/`. Façade `app` correctement encapsulante.

VERDICT : **OK**.

## Test 4 — Domaine pur (fibonacci, bigfft)

`internal/bigfft` : 0 import interne. **Domaine strictement pur**.

`internal/fibonacci` : imports détectés
- `internal/bigfft` (domaine) — OK
- `internal/fibonacci/memory`, `internal/fibonacci/threshold` (sous-packages) — OK
- `internal/parallel` (utilitaire) — OK
- `internal/progress` — **fuite infra** : `progress_aliases.go` ré-exporte 7 types et 8 fonctions de `progress` pour rétrocompatibilité ; le domaine dépend ainsi d'une couche d'observation.

VERDICT : **1 violation** (`fibonacci → progress`). `bigfft` est conforme.

## Test 5 — Couche utilitaire

| Package | Imports internes | Feuille ? |
|---------|------------------|-----------|
| `internal/format` | aucun | Oui |
| `internal/parallel` | aucun | Oui |
| `internal/errors` | `internal/format` | Quasi-feuille (1 dép. utilitaire) |

VERDICT : **OK**. `format` et `parallel` sont des feuilles strictes ; `errors → format` est admissible (deux utilitaires).

## Synthèse

- **Score étanchéité** : 1 violation (Test 4) + 0 violation Tests 1/2/3/5.
- **Top 3 corrections** :
  1. Supprimer `internal/fibonacci/progress_aliases.go` ; faire migrer les consommateurs vers `internal/progress` (rompt la dépendance domaine→infra).
  2. À défaut, déplacer les types `Progress*` dans `internal/fibonacci` et faire de `internal/progress` un consommateur (inversion de dépendance).
  3. Documenter explicitement dans `CLAUDE.md` que `cmd/` ne doit importer **que** `internal/app` (la règle actuelle mentionne `app` et `cli`, mais `cli` n'est plus directement utilisé — clarifier).
