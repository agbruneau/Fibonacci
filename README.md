# Fibonacci

Deux laboratoires d'expérimentation qui prennent un problème volontairement simple, calculer F(n), comme banc d'essai : la réponse est exactement vérifiable, plusieurs algorithmes se comparent entre eux, et le coût grandit avec n, si bien que tout écart de conception se mesure.

| Projet | Dossier | Dépôt d'origine | En bref |
|---|---|---|---|
| **FibCalc** | [`FibGo/`](FibGo/) | [agbruneau/FiboGo](https://github.com/agbruneau/FiboGo) | Calculateur haute performance en Go : Fast Doubling, matrices Strassen-Winograd, multiplication FFT, parallélisme adaptatif, PGO, interface TUI. |
| **Fib** | [`FibPolyglotte/`](FibPolyglotte/) | [agbruneau/FiboPolyGlotte](https://github.com/agbruneau/FiboPolyGlotte) | Une même CLI écrite quatre fois (Go, Rust, TypeScript, Python) sous un contrat commun vérifié à l'octet près. |

Les deux projets sont intégrés ici en totalité, dans l'état de leur dépôt d'origine (FiboGo@6b51364, FiboPolyGlotte@18877d5); leur historique de commits reste dans ces dépôts. Le README de chaque dossier détaille l'installation, l'utilisation, les mesures et l'état vérifié.

## Cloner

```bash
git clone https://github.com/agbruneau/Fibonacci.git
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
- Licence : Apache 2.0; le code dérivé de `bigfft` reste sous BSD-3-Clause ([`FibGo/NOTICE`](FibGo/NOTICE)).

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
- Licence : Apache 2.0.

Voir [`FibPolyglotte/README.md`](FibPolyglotte/README.md), le [PRD](FibPolyglotte/PRD.md) et les [ADR](FibPolyglotte/docs/adr/).

## Organisation

```
Fibonacci/
├── README.md
├── LICENSE          Apache 2.0, pour tout le dépôt
├── NOTICE           droits d'auteur et code tiers
├── FibGo/           FibCalc, issu de agbruneau/FiboGo
└── FibPolyglotte/   Fib, issu de agbruneau/FiboPolyGlotte
```

Chaque projet garde ses propres fichiers de configuration (`.gitignore`, `.gitattributes`, `go.mod`, `package.json`…) dans son dossier; les commandes se lancent depuis ce dossier.

## Limites de l'intégration

- **Historique.** Les commits et les étiquettes de version (FibCalc v3.0.0 à v5.0.0) restent dans FiboGo et FiboPolyGlotte. Les hashes cités dans les docs des deux projets, comme `0590ab8`, ne se résolvent que là.
- **Issues.** Les issues citées par Fib (`agbruneau/Fib#…`) sont celles de FiboPolyGlotte; l'ancien nom redirige.
- **CI.** Le workflow `FibGo/.github/workflows/ci.yml` ne s'exécute pas ici : GitHub ne lit que le `.github/` de la racine. Il tourne dans FiboGo.
- **Révisions de Fib.** `npm run bench -- <id> --contre REF` et `npm run differentiel -- --revision REF` ne fonctionnent pas encore ici : l'extraction de la révision range ses fichiers sous `FibPolyglotte/` au lieu de la racine de la révision.
- **Chemin de module.** FibCalc garde `github.com/agbruneau/FibGo`, que GitHub redirige vers FiboGo.

## Licence

Tout le dépôt est sous [licence Apache 2.0](LICENSE), © 2026 André-Guy Bruneau. Le code tiers et ses licences sont recensés dans [`NOTICE`](NOTICE). `FibGo/` garde sa propre copie de `LICENSE` et `NOTICE`, que son image Docker embarque.
