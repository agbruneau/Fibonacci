# ADR-0012: Audit 2026-09-07 « livre » — décisions D1 à D5

- **Status**: Accepted
- **Date**: 2026-09-07
- **Deciders**: André-Guy Bruneau (mainteneur)
- **Context source**: [`audit-2026-09-livre.md`](../audits/audit-2026-09-livre.md) — audit du dépôt à l'aune de
  Saeed Shahsavan, *Building Enterprise Projects with Go*, Apress 2026 (20
  chapitres). Gate rejoué le 2026-09-07 sur `c6ce7fb` : build / vet /
  `golangci-lint` / `gofmt` / `go mod tidy -diff` à zéro, 21 paquets verts,
  couverture 96,7 %.

## Context

L'audit 2026-09-07 confronte le dépôt à une grille externe. Il conclut que le
projet dépasse le livre sur le modèle d'erreurs, la concurrence, la profondeur
des tests et la traçabilité des décisions, et reste en deçà sur trois points :
l'exécution garantie du gate, l'épinglage des outils et l'étanchéité des
couches.

Un élément nouveau, absent des audits 0008 à 0011, a déclenché la révision de
la position « pas de CI » : sur l'hôte de référence lui-même, **trois outils
d'analyse ne démarrent plus** parce que leurs binaires ont été compilés pour
`go1.26` alors que la chaîne courante est `go1.27` :

```text
govulncheck ./...   « uses version go1.26 of the source-processing packages
                      but runs version go1.27 »
gosec ./...         « internal error: package "os" without types »
staticcheck ./...   « export data version 4 is greater than maximum supported
                      version 2 »
```

C'est le mode de défaillance de GATE-01 ([ADR-0010](0010-audit-2026-09-decisions.md)
D3, où `golangci-lint` a écrit `Overall: PASS` sans s'exécuter), reproduit sur
trois outils, et de nouveau silencieux : aucun gate ne les appelle, donc rien ne
le signale.

Cinq décisions conditionnaient le plan d'exécution de l'audit
([`audit-2026-09-livre.md` § 7.2](../audits/audit-2026-09-livre.md)). Elles
sont tranchées ici.

## Decision

### D1 — CI GitHub Actions réintroduite

**Retenu : oui.**

[ADR-0010](0010-audit-2026-09-decisions.md) D4 acte l'absence de CI, avec la
clause « À revoir si : le projet accueille des contributeurs dont on ne peut
pas supposer l'outillage local ». La clause visait des contributeurs tiers ;
elle est atteinte plus tôt et autrement — **l'outillage local du mainteneur
lui-même n'est plus supposable**, et l'échec est muet.

Le workflow n'est pas un remplacement du gate local mais son exécution
garantie : matrice `ubuntu-latest` / `windows-latest`, version Go lue depuis
`go.mod`, `gofmt` / `go vet` / `go build` / `go test -race -shuffle=on
-count=1` avec le plancher de couverture, lint et `govulncheck` épinglés (T02),
un job Ubuntu `libgmp-dev` qui exécute enfin l'étape `-tags gmp` restée en SKIP
sur l'hôte Windows, un build `GOARCH=386`, une construction de l'image Docker et
un job planifié de fuzzing.

L'épinglage retenu en T02 n'est pas celui que l'audit proposait. La directive
`tool` de `go.mod` a été essayée puis **rejetée sur mesure** : elle fait passer
le graphe de modules de 201 à 450 et, par MVS, remonte une dépendance de
production (`gopsutil` v4.26.3 → v4.26.7). Les versions vivent donc dans
`scripts/tools.env` et les outils s'exécutent par `go run <pkg>@<version>`, ce
qui donne la même garantie de reconstruction sans toucher au graphe.

Cette décision **renverse** ADR-0004 §B3 (volet « pas de CI ») et ADR-0010 D4.
ADR-0004 §B3 reste valide sur son objet propre : pas de *bench cross-arch*
ARM64 automatisé, faute de runners gratuits.

### D2 — Journalisation : `log/slog` injecté, `zerolog` retiré

**Retenu : `slog` injecté depuis la couche d'application.**

L'état constaté est inversé par rapport au livre (ch. 6 « Don't Log and
Return » p. 189 ; ch. 14 p. 370, « If a logger requires a third-party
dependency, keep it confined to the adapter layer ») : `zerolog` est importé
par six paquets de production, dont cinq du domaine, et **aucun des six
émetteurs n'est atteignable** — cinq loggers sont figés à `zerolog.Nop()` avec
des setters réservés aux tests, le sixième est filtré par
`zerolog.SetGlobalLevel(zerolog.InfoLevel)` posé dans `app.Run`. Un `TestMain`
existe uniquement pour reposer ce niveau global.

L'option minimale (tout supprimer) rendait le dépôt conforme au livre mais
perdait six événements de diagnostic déjà écrits et utiles (GC coupé/rétabli,
seuils ajustés, statistiques de la cache FFT). L'option retenue les garde et
les rend observables : `fibonacci.Options.Logger *slog.Logger` (nil →
`slog.DiscardHandler`), handler construit dans `app`, drapeau `--log-level` et
variable `FIBCALC_LOG_LEVEL`. `slog` est dans la bibliothèque standard : le
domaine ne dépend plus d'un tiers, et une dépendance directe disparaît.

### D3 — `internal/errors` renommé en `internal/apperrors`

**Retenu : oui, dans cette passe.**

Le répertoire s'appelle `errors` mais le paquet se déclare `apperrors` : chaque
importateur écrit `apperrors "…/internal/errors"`, et un `import "errors"`
voisin est une confusion classique. Le diff est large mais purement mécanique
(un commit `git mv`, un commit de réécriture des imports). Le faire dans la
même passe que la scission présentation/erreurs (T15), qui touche déjà tous les
importateurs, évite un second passage.

### D4 — Langue : français pour le narratif, anglais pour le code

**Retenu.** Documentation narrative (README, CHANGELOG, ADR, `docs/*.md`) en
**français** ; code, commentaires, messages d'erreur, sujets de commit et
`.golangci.yml` en **anglais**.

Le corpus est aujourd'hui mixte sans règle : `README`, `CHANGELOG`,
`PORTABILITY`, ADR-0010 et 0011 en français ; `ARCH`, `TESTING`, `BUILD`,
`PERFORMANCE`, `CALIBRATION`, `CONTRIBUTING` en anglais ; `ARCH.md` alterne les
deux dans le même document. La règle suit l'usage réel (le mainteneur écrit en
français, le code est déjà en anglais) et s'inscrit dans `CONTRIBUTING.md`.

> **Addendum du 2026-09-23 ([ADR-0013](0013-evaluation-2026-09-decisions.md) D2,
> EVAL-19).** La décision ci-dessus est conservée telle qu'écrite ; elle n'a pas
> été appliquée : l'évaluation du 2026-09-15 relève treize documents de
> référence restés en anglais. La règle est amendée pour décrire le corpus :
> narratif (README, CHANGELOG, ADR, `docs/audits/`) en français ; référence
> technique (`docs/*.md`, `docs/algorithms/`, `docs/architecture/`) en anglais ;
> un document ne mélange pas les deux. `docs/ARCH.md`, qui alternait, est rendu
> monolingue en anglais.

### D5 — `briandowns/spinner` remplacé par un rendu interne

**Retenu : oui.**

La bibliothèque rend une ligne de progression et coûte : une course connue
(CONC-01) contournée en arrêtant et relançant sa goroutine cinq fois par
seconde, une interface `Spinner`, un adaptateur `realSpinner`, une variable de
couture `newSpinner` et un test dédié. Le `ticker` de `DisplayProgress` existe
déjà ; un cadre de huit caractères et un `\r` le remplacent. Une dépendance
directe et quatre transitives disparaissent.

Contrepartie assumée : la forme de la ligne de progression change, et les
fichiers *golden* correspondants sont régénérés puis relus.

## Consequences

### Positive

- Le gate cesse de dépendre de la mémoire du mainteneur et de l'état de son
  `PATH` ; la classe de défaillance GATE-01 est fermée par construction (les
  outils sont reconstruits à chaque changement de chaîne Go).
- L'étape `-tags gmp` et le build 32 bits, jamais exécutés sur l'hôte de
  développement, entrent dans le cycle.
- Deux dépendances directes disparaissent (`zerolog`, `spinner`), et le domaine
  n'importe plus aucune bibliothèque tierce de journalisation.
- Six événements de diagnostic déjà écrits deviennent observables.

### Negative / Trade-offs

- Le dépôt réacquiert une dépendance à une plate-forme externe (GitHub
  Actions), refusée jusqu'ici.
- La ligne de progression change d'apparence (D5).
- Le renommage D3 produit un diff large qui traverse l'historique `git blame`
  des importateurs.

### Risks and Mitigations

- **Régression de performance sur le chemin chaud** (D2, journalisation dans
  `CalculateWithObservers`) : `slog.DiscardHandler` court-circuite avant
  formatage ; vérification `benchstat` en double ordre au seuil de 5 %
  (protocole [ADR-0009](0009-audit-2026-07-cleanup-and-rejected-fib05.md) R4).
- **CI qui diverge du gate local** : les deux lisent `scripts/tools.env`, donc
  les mêmes outils aux mêmes versions.
- **Renommage D3 qui casse un renvoi documentaire** : recherche des occurrences
  dans `docs/` et `README` incluse dans la tâche.

## Alternatives Considered

- **D1 — garder l'absence de CI et réinstaller les outils à la main** :
  rejeté ; c'est exactement la discipline qui a déjà échoué deux fois (GATE-01,
  puis les trois outils périmés), et l'échec est muet.
- **D2 — supprimer `zerolog` sans le remplacer** : rejeté ; conforme au livre,
  mais perd six diagnostics déjà écrits alors que `slog` ne coûte aucune
  dépendance.
- **D3 — reporter le renommage** : rejeté ; le coût est identique plus tard et
  le second passage sur les importateurs est évitable maintenant.
- **D5 — garder `spinner` et vivre avec le contournement** : rejeté ; le
  contournement est plus long que le remplacement.

## Exécution

Les cinq décisions ont été appliquées le 2026-09-07, en quatre phases et cinq
commits sur `main` : `fd0f21c` (outillage et CI), `24e3cdb` (correctifs de
comportement), `44e3520` (défauts trouvés par la CI), `c2329f3` (frontières et
dépendances), puis la clôture. Les 33 tâches du plan sont exécutées, les deux
optionnelles comprises ; le seul écart est consigné dans la ligne T25 du tableau
de suivi.

### Ce que la décision D1 a rapporté immédiatement

La CI a trouvé cinq défauts sur ses trois premiers passages, aucun reproductible
par le gate local :

1. `TestGenerateParallelThresholds` attendait un seuil séquentiel de `0`, valeur
   que FIB-02 avait remplacée par `ThresholdDisabled`. L'assertion vivait dans
   la branche « 4 cœurs ou moins », que l'hôte à 24 cœurs du mainteneur ne prend
   jamais : elle était morte depuis FIB-02. Un *runner* à 4 cœurs l'a touchée en
   trois minutes. Corrigé à la cause : le `switch` sur le nombre de cœurs est
   devenu une fonction pure, et les quinze cas de branche sont couverts sur
   n'importe quel hôte.
2. `go test -coverprofile=…` échoue sur le *runner* Windows, dont le shell par
   défaut est pwsh — le piège que `scripts/check.ps1` documentait déjà et que le
   workflow a répété.
3. `GOARCH=386` ne compilait pas (TYP-01), corrigé en phase 2.
4. `make build` était cassé partout où `GOFLAGS` est défini dans
   l'environnement — le devcontainer, le `Dockerfile` et la CI — parce que le
   `Makefile` avait nommé sa propre variable `GOFLAGS`. `make build` n'avait
   simplement jamais été lancé dans aucun des trois.
5. `govulncheck` a signalé GO-2026-4602 (`os`, `FileInfo` s'échappant d'un
   `Root`) **atteignable** : `gopsutil` appelle `os.File.Readdir` depuis
   `cpu.init`, et `internal/tui` l'importe. Présente dans go1.26.0, la version
   que `go.mod` déclarait ; le plancher est passé à go1.26.1. L'hôte du
   mainteneur tourne sous go1.27 et ne pouvait pas la voir.

Le job `gmp` a par ailleurs exécuté la suite `-tags gmp` pour la première fois
depuis que l'étape existe : elle était en SKIP sur l'hôte de développement,
faute d'en-têtes libgmp.

Le job `docker` a fourni, après `v4.1.0`, ce qu'aucun autre environnement du
projet n'avait eu : les digests des deux images de base (SEC-04, ouvert depuis
quatre audits), épinglés au commit `24cfd38`.

### Mesures

`benchstat`, 10 séries de 3 itérations, protocole ADR-0009 R4 : geomean sec/op
**−3,45 %**, B/op plat, allocations plates. Les deux écarts que `benchstat`
signale se reproduisent sur une comparaison A/A à code identique — la dispersion
de l'hôte dépasse le seuil de 5 % sur ce jeu de benchmarks.

Couverture 96,6 % → **96,1 %**, plancher 80 % tenu. Dépendances directes 10 → 8,
graphe de modules 201 → 198.

## References

- Audit : [`audit-2026-09-livre.md`](../audits/audit-2026-09-livre.md) § 7.2 (décisions), § 7.3 (plan) et § 7.8 (suivi)
- ADR renversés : [ADR-0004](0004-backlog-decisions.md) §B3 (volet CI),
  [ADR-0010](0010-audit-2026-09-decisions.md) D4
- ADR liés : [ADR-0009](0009-audit-2026-07-cleanup-and-rejected-fib05.md) R4
  (protocole `benchstat`), [ADR-0011](0011-audit-2026-09-ponytail.md)
