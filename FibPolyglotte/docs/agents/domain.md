# Domain Docs

Comment les skills d'ingénierie consomment la documentation du domaine de ce dépôt quand elles explorent le code.

## Avant d'explorer, lire

- **`CONTEXT.md`** à la racine : le glossaire du domaine.
- **`docs/adr/`** : les ADR qui touchent la zone où tu t'apprêtes à travailler.

Si ces fichiers n'existent pas, **continuer sans rien dire**. Ne pas signaler leur absence ni proposer de les créer d'avance : la skill `/domain-modeling` (atteinte par `/grill-with-docs` et `/improve-codebase-architecture`) les crée au moment où un terme ou une décision est réellement tranché.

## Structure des fichiers

Dépôt single-context : un seul domaine, partagé par les quatre implémentations.

```
/
├── CONTEXT.md
├── docs/adr/
├── go/  rust/  ts/  python/
```

## Utiliser le vocabulaire du glossaire

Quand ta sortie nomme un concept du domaine (titre d'issue, proposition de refactoring, hypothèse, nom de test), emploie le terme défini dans `CONTEXT.md`. N'emploie pas les synonymes que le glossaire écarte explicitement.

Si le concept dont tu as besoin n'y figure pas encore, c'est un signal : soit tu inventes un langage que le projet n'emploie pas (reconsidère), soit il y a un vrai manque (note-le pour `/domain-modeling`).

## Signaler les conflits avec un ADR

Si ta sortie contredit un ADR existant, dis-le explicitement au lieu de le contourner en silence :

> _Contredit l'ADR-0002 (analyse manuelle des arguments), mais mérite d'être rouvert parce que…_
