# Fibonacci

Deux laboratoires d'expérimentation qui prennent un problème volontairement simple, calculer F(n), comme banc d'essai : la réponse est exactement vérifiable, plusieurs algorithmes se comparent entre eux, et le coût grandit avec n, si bien que tout écart de conception se mesure.

| Projet | Dossier | Dépôt d'origine | En bref |
|---|---|---|---|
| **FibCalc** | [`FibGo/`](FibGo/) | [agbruneau/FiboGo](https://github.com/agbruneau/FiboGo) | Calculateur haute performance en Go : Fast Doubling, matrices Strassen-Winograd, multiplication FFT, parallélisme adaptatif, PGO, interface TUI. |
| **Fib** | [`FibPolyglotte/`](FibPolyglotte/) | [agbruneau/FiboPolyGlotte](https://github.com/agbruneau/FiboPolyGlotte) | Une même CLI écrite quatre fois (Go, Rust, TypeScript, Python) sous un contrat commun vérifié à l'octet près. |

Chaque dossier est un submodule git qui pointe vers son dépôt d'origine. Son README détaille l'installation, l'utilisation, les mesures et l'état vérifié.

## Cloner

```bash
git clone --recurse-submodules https://github.com/agbruneau/Fibonacci.git
```

Dans un clone fait sans `--recurse-submodules` :

```bash
git submodule update --init
```

## FibCalc (`FibGo/`)

Calcul de nombres de Fibonacci arbitrairement grands, jusqu'à n = 100 000 000 dans les mesures du dépôt. Le projet sert aussi de terrain pour la méthode logicielle : Clean Architecture, tests golden et property-based, ADR, gate de qualité local.

- Prérequis : Go 1.26.1 ou plus.
- Démarrage :

  ```bash
  cd FibGo
  go build -o fibcalc ./cmd/fibcalc
  ./fibcalc -n 100 -c
  ```

  Sous Windows : `go build -o fibcalc.exe ./cmd/fibcalc`, puis `.\fibcalc.exe -n 100 -c`.
- Licence : Apache 2.0 ([`FibGo/LICENSE`](FibGo/LICENSE)).

Voir [`FibGo/README.md`](FibGo/README.md), [`FibGo/docs/ARCH.md`](FibGo/docs/ARCH.md) et l'historique des audits dans [`FibGo/docs/audits/HISTORY.md`](FibGo/docs/audits/HISTORY.md).

## Fib (`FibPolyglotte/`)

Terrain d'essai académique pour explorer, avec Claude Code, les [skills d'ingénierie de Matt Pocock](https://github.com/mattpocock/skills), du PRD à la revue de code. Les quatre implémentations ne partagent aucun code; elles ne se rejoignent que par le contrat de la CLI et un oracle de test.

- Prérequis : Go 1.26+, Rust stable (édition 2024), Node 24.2+ (ou 22.18+) avec npm, Python 3.12+.
- Vérification complète :

  ```bash
  cd FibPolyglotte
  npm install
  npm run check
  ```
- Licence : aucune déclarée pour l'instant.

Voir [`FibPolyglotte/README.md`](FibPolyglotte/README.md), le [PRD](FibPolyglotte/PRD.md) et les [ADR](FibPolyglotte/docs/adr/).

## Organisation

```
Fibonacci/
├── README.md
├── .gitmodules
├── FibGo/           submodule → agbruneau/FiboGo
└── FibPolyglotte/   submodule → agbruneau/FiboPolyGlotte
```

Le travail sur un projet se fait dans son dossier et se pousse vers son dépôt d'origine. Pour faire pointer ce dépôt sur la dernière version des deux :

```bash
git submodule update --remote
git commit -am "chore: mettre à jour les submodules"
```
