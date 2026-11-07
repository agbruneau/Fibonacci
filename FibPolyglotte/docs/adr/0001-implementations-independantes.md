# Quatre implémentations indépendantes, reliées seulement par le contrat

Fib réalise le même calcul en Go, Rust, TypeScript et Python sans aucun code partagé : pas de noyau commun appelé par FFI, WASM ou cgo, pas de génération de code. Les implémentations ne se rejoignent que par le contrat de la CLI, que vérifie un test commun, et par l'oracle tiré de l'OEIS (voir l'amendement du 2026-10-10). Le but est d'exercer les skills d'agent une fois par langage. Un noyau partagé réduirait trois implémentations à des adaptateurs et cacherait justement ce qu'on veut observer.

## Considered Options

- **Noyau Rust appelé par les trois autres** : moins de code dupliqué, mais trois implémentations deviennent des enveloppes et le dépôt exige une chaîne de build FFI et WASM.
- **Termes attendus générés par l'une des implémentations**, comme le golden de FibCalc : un oracle qui partage les défauts du code qu'il vérifie. L'OEIS est une source indépendante.

## Consequences

Un écart entre langages ne se voit qu'à l'exécution du test de contrat. Celui-ci est donc le critère de fin de chaque ticket de langage.

## Amendement du 2026-10-10 : oracle généré

Le téléchargement du b-file OEIS a été bloqué par les permissions de la session. L'oracle est donc généré par la formule de Binet (`contracts/generer_oracle.py`), une méthode qui ne partage rien avec les trois algorithmes testés. L'indépendance tient à la vérification : ORA-04 et ORA-05 contrôlent la table par la définition seule, et toute table qui les passe est nécessairement la suite de Fibonacci. La table coïncide avec la page OEIS pour F(2000).

## Amendement du 2026-10-13 : la boucle différentielle

Chercher un écart entre les quatre implémentations demande un outil qui les lance sur les mêmes entrées et regroupe celles dont les sorties diffèrent, sans aucun attendu. Il a été refait hors du dépôt à trois reprises (l'audit du 2026-10-11, l'épreuve des cas, l'étape 9), et il a trouvé seul l'écart du stderr fermé de Python (CLI-V04). Il est versionné : `contracts/differentiel.ts`, `npm run differentiel`. C'est un outil de recherche, non un oracle : en tirer un attendu reviendrait à prendre le vrai dans les sorties des implémentations elles-mêmes, comme le golden de FibCalc que cet ADR écarte. Cinq conditions le tiennent à sa place :

- **Hors du contrat.** Aucun script `check*` ni `contract` ne la lance, et le test de contrat ne l'importe pas, même par un module qu'il importe. `HAR-6` le garde (`docs/cas-de-test.md` § 6).
- **Aucune référence, aucune majorité.** Elle regroupe les implémentations par sortie identique, dans l'ordre du PRD § 5, et ne dit pas laquelle a raison. Un groupe de trois ne vaut pas plus qu'un groupe d'une.
- **Une seule révision à la fois.** Elle compare les implémentations d'une même révision (`--revision REF` les prend toutes à `REF`), jamais deux révisions d'une même implémentation : c'est l'A/B du banc (ADR-0004), qui ne compare que des temps.
- **Ni oracle ni attendu.** Elle ne lit pas `contracts/b000045.txt`, et n'écrit qu'un aperçu des sorties : le début de chacune (24 octets de stdout, 100 de stderr), que la boucle ne relit ni ne compare.
- **Un écart se tranche ailleurs.** Par le PRD § 4, puis par un cas du catalogue : le cas entre dans le contrat, la boucle n'y entre jamais.

Un témoin, joué avant chaque recherche et hors du contrat, éprouve sa projection et son regroupement sur des sorties factices.
