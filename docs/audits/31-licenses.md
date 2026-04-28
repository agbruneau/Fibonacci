# 31 — Licences & SBOM

Audit licences et SBOM des dépendances de FibGo (`github.com/agbru/fibcalc`). Source de vérification : module cache local (`go env GOMODCACHE` → `C:\Users\agbru\go\pkg\mod`). Aucune requête réseau effectuée.

## Licence projet

- **Apache 2.0 confirmée** — `LICENSE` à la racine, texte officiel intégral (Apache License Version 2.0, January 2004 ; appendix « How to apply » conservé). Aucun marquage SPDX explicite.
- **En-têtes Go : absents** dans le code FibGo. Grep sur tout le repo (`^// Copyright` dans `*.go`) → 1 seul match : `internal/bigfft/arith_decl.go`, qui porte l'en-tête BSD upstream `Copyright 2010 The Go Authors` (code dérivé de `math/big`). Les fichiers signature comme `cmd/fibcalc/main.go` ou `internal/fibonacci/calculator.go` commencent directement par `// Package …`. **Pratique FibGo : pas d'en-tête de licence par fichier**, conformité reposant uniquement sur le `LICENSE` racine. Conforme à Apache 2.0 § 4 (recommandé mais non obligatoire) ; toutefois l'appendix Apache *recommande* l'en-tête boilerplate.
- **`CONTRIBUTING.md`** : décrit code de conduite, conventions, PR process. **Aucune mention de licence ni de CLA/DCO**. C'est un manque mineur : un contributeur externe n'a aucune indication explicite que sa contribution est régie par Apache 2.0 § 5 (qui s'applique malgré tout par défaut, mais le rappel est de bonne pratique).

## SBOM dépendances directes

Confirmation locale via `<GOMODCACHE>/<module>@<version>/LICENSE` (lecture des 5–10 premières lignes pour identifier le type SPDX).

| Module | Version | Licence | Source confirmation | Compat. Apache 2.0 |
|---|---|---|---|---|
| `golang.org/x/sync` | v0.20.0 | BSD-3-Clause | `golang.org/x/sync@v0.20.0/LICENSE` (« Copyright 2009 The Go Authors », clauses BSD-3) | OK |
| `golang.org/x/sys` | v0.43.0 | BSD-3-Clause | `golang.org/x/sys@v0.43.0/LICENSE` (idem Go Authors) | OK |
| `github.com/briandowns/spinner` | v1.23.2 | Apache-2.0 | `briandowns/spinner@v1.23.2/LICENSE` (« Apache License Version 2.0 ») | OK (identique) |
| `github.com/charmbracelet/bubbles` | v0.21.1 | MIT | `charmbracelet/bubbles@v0.21.1/LICENSE` (« MIT License, Copyright (c) 2020-2025 Charmbracelet, Inc ») | OK |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | MIT | `charmbracelet/bubbletea@v1.3.10/LICENSE` | OK |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | MIT | `charmbracelet/lipgloss@v1.1.0/LICENSE` (« Copyright 2021-2023 Charmbracelet, Inc ») | OK |
| `github.com/leanovate/gopter` | v0.2.11 | MIT | `leanovate/gopter@v0.2.11/LICENSE` (« The MIT License (MIT), Copyright (c) 2016 leanovate ») | OK |
| `github.com/ncw/gmp` | v1.0.5 | BSD-3-Clause | `ncw/gmp@v1.0.5/LICENSE` (« Copyright (c) 2012 The Go Authors », clauses BSD-3 ; texte hérité de `math/big`) | OK (le binding Go ; **GMP natif lui-même est LGPL/GPL** — voir Conflits ci-dessous) |
| `github.com/rs/zerolog` | v1.35.0 | MIT | `rs/zerolog@v1.35.0/LICENSE` (« MIT License, Copyright (c) 2017 Olivier Poitrey ») | OK |
| `github.com/shirou/gopsutil/v4` | v4.26.3 | BSD-3-Clause | `shirou/gopsutil/v4@v4.26.3/LICENSE` (« gopsutil is distributed under BSD license… Copyright (c) 2014, WAKAYAMA Shirou ») | OK |

**Bilan direct : 10/10 modules sous licences whitelistées (5 MIT, 4 BSD-3, 1 Apache-2.0). Aucune licence copyleft directe.**

## Dépendances indirectes

Total : 26 modules. Confirmation locale par échantillonnage (28 LICENSE inspectés, couvrant 100 % des indirects listés dans `go.mod`). Résultats agrégés :

| Famille | Compte | Modules | Licence | Compat. |
|---|---|---|---|---|
| MIT | 19 | `aymanbagabas/go-osc52/v2`, `charmbracelet/colorprofile`, `charmbracelet/x/ansi`, `charmbracelet/x/cellbuf`, `charmbracelet/x/term`, `clipperhouse/displaywidth`, `clipperhouse/stringish`, `clipperhouse/uax29/v2`, `erikgeiser/coninput`, `fatih/color`, `go-ole/go-ole`, `lucasb-eyer/go-colorful`, `mattn/go-colorable`, `mattn/go-isatty` (« MIT License (Expat) »), `mattn/go-localereader` (README → MIT, pas de fichier LICENSE), `mattn/go-runewidth`, `muesli/ansi`, `muesli/cancelreader`, `muesli/termenv`, `power-devops/perfstat`, `rivo/uniseg`, `xo/terminfo`, `yusufpapurcu/wmi` | MIT | OK |
| BSD-3-Clause | 4 | `lufia/plan9stats`, `tklauser/go-sysconf`, `golang.org/x/term`, `golang.org/x/text` | BSD-3 | OK |
| Apache-2.0 | 2 | `ebitengine/purego`, `tklauser/numcpus` | Apache-2.0 | OK |

Notes :
- `mattn/go-localereader v0.0.1` n'a **pas de fichier `LICENSE` distribué** dans le module ; le `README.md` indique « ## License : MIT ». Risque mineur : à signaler upstream pour conformité Apache 2.0 § 4(c) (préservation des notices).
- `lucasb-eyer/go-colorful` : LICENSE sans en-tête SPDX explicite mais texte MIT (« Permission is hereby granted… »). Confirmé MIT.

**Bilan indirect : 26/26 sous licences whitelistées. Aucune GPL/LGPL/AGPL/MPL détectée dans le graphe Go.**

## Conflits potentiels

1. **`github.com/ncw/gmp` + libgmp natif (build tag `gmp`)** — *Risque modéré, conditionnel*. Le **binding Go** (`ncw/gmp`) est BSD-3. Mais il fait du `cgo` vers la **libgmp système**, qui est sous **LGPL v3** (ou GPL v2 dans certaines distributions). Conséquences :
   - Build par défaut (sans tag `gmp`) : **aucun risque**, libgmp non liée.
   - Build avec `-tags gmp` produisant un binaire lié dynamiquement à libgmp : **acceptable** (LGPL § 4 — linkage dynamique préserve la séparation de licence). Le redistributeur doit néanmoins fournir le code source de libgmp ou un pointeur vers celui-ci.
   - Build statique avec `-tags gmp` (CGO + lien statique) : **incompatible avec une distribution Apache 2.0 propriétaire** sans précautions LGPL § 6 (fournir les .o ou la source). À documenter explicitement si jamais distribué.
2. **Aucun autre conflit** : pas de copyleft fort dans le graphe Go pur.

## NOTICE / Attribution

- **Fichier `NOTICE` : ABSENT** à la racine du repo (vérifié via `Glob NOTICE*` → 0 résultat).
- **Apache 2.0 § 4(d)** : un fichier NOTICE n'est *obligatoire* que si l'œuvre originale en contient un et que des œuvres dérivées sont distribuées. FibGo en tant qu'œuvre originale n'a pas d'obligation stricte d'en publier un. **Risque pour distribution : faible** tant que :
  - Les LICENSE des dépendances sont conservées dans toute redistribution (cas du build Go statique : licences embarquées au niveau code source). Pour un binaire distribué (`make build-all`), aucune attribution n'est techniquement embarquée.
  - **Recommandation** : créer un `NOTICE` listant les copyrights amont (charmbracelet, Olivier Poitrey, Go Authors, WAKAYAMA Shirou, leanovate, etc.) et joindre les LICENSE tierces pour toute distribution binaire publique. Outil suggéré : `github.com/google/go-licenses` ou `go-licenses save` pour générer un dossier `THIRD_PARTY_LICENSES/`.

## Indicateurs amont

- **Pseudo-versions** (commits datés sans tag sémantique) parmi les indirects : `erikgeiser/coninput` (2021-10-04), `lufia/plan9stats` (2021-10-12), `muesli/ansi` (2023-03-16), `power-devops/perfstat` (2024-02-21), `xo/terminfo` (2022-09-10). Toutes sont des dépendances tirées par `bubbletea`, `termenv` ou `gopsutil` ; pas d'action de notre côté, le cycle de vie est piloté par les mainteneurs amont (charmbracelet, shirou).
- **Projets potentiellement peu maintenus** : `xo/terminfo` (dernier commit 2022), `power-devops/perfstat` (AIX-spécifique), `mattn/go-localereader` v0.0.1 (Windows code-page). Faible risque opérationnel (utilitaires bas-niveau stables) mais à surveiller pour CVE.
- **Aucun module marqué `retired`** dans `go.mod` ni d'avertissement détecté.

## Synthèse

| Critère | État |
|---|---|
| Licence projet (LICENSE) | OK — Apache 2.0 intégrale |
| En-têtes Go par fichier | Absent — pratique acceptée mais non recommandée par l'appendix Apache |
| CLA/DCO mentionné dans CONTRIBUTING | Manquant |
| SBOM directs (10) | OK — 100 % whitelistées |
| SBOM indirects (26) | OK — 100 % whitelistées |
| NOTICE racine | Absent — non bloquant pour code source, recommandé pour binaires |
| Conflits copyleft | Aucun en mode défaut ; LGPL via libgmp si build `-tags gmp` distribué statiquement |

**Score conformité licence : OK (avec ajustements mineurs recommandés).**

### Actions recommandées (par priorité)

1. **P2 — Ajouter un `NOTICE`** racine listant les attributions amont, et générer `THIRD_PARTY_LICENSES/` lors de `make build-all` (cible `make licenses` via `go-licenses save ./... --save_path=build/licenses`). Indispensable avant toute distribution binaire publique sur GitHub Releases.
2. **P3 — Documenter dans `README.md` ou `docs/`** la situation libgmp/LGPL pour les builds `-tags gmp` (clause d'avertissement + lien LGPL).
3. **P3 — Compléter `CONTRIBUTING.md`** avec une section « License » : « By contributing, you agree your contributions will be licensed under Apache 2.0 (see LICENSE). » Optionnellement DCO sign-off.
4. **P4 — En-têtes SPDX optionnels** : `// SPDX-License-Identifier: Apache-2.0` en première ligne des fichiers Go nouveaux. Pas de migration forcée du code existant (changement chirurgical hors scope).
5. **P4 — Automatiser la veille** : ajouter `go-licenses check ./... --disallowed_types=forbidden,restricted` dans la CI (`make lint` ou step dédié) pour détecter toute future dépendance copyleft introduite par dérive.
