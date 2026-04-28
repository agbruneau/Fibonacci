# 72 — Cross-compilation

## Méthode
- Commande équivalente : `for os/arch in {linux,windows,darwin}/{amd64,arm64}: GOOS=$os GOARCH=$arch go build ./cmd/fibcalc`
- Log : `docs/audits/_build_raw.log` (en-têtes uniquement, aucune erreur stderr capturée).
- Re-vérification effectuée avec `go build -o /dev/null` lors de l'audit : 6/6 cibles OK.
- CGO implicitement désactivé (cross-compile standard, pas de toolchain C cross).

## Résultats
| OS | Arch | Statut | Notes |
|---|---|---|---|
| linux | amd64 | OK | Chemin `arith_amd64.go` + `cpu_amd64.go` (intrinsics SIMD). |
| linux | arm64 | OK | Fallback `arith_generic.go` (`//go:build !amd64`). |
| windows | amd64 | OK | Chemin amd64 complet. |
| windows | arm64 | OK | Fallback `!amd64`. |
| darwin | amd64 | OK | Chemin amd64 complet. |
| darwin | arm64 | OK (Apple Silicon) | Fallback `!amd64`. |

Taille binaire hôte (`windows/amd64`, build standard sans PGO, sans `-trimpath -ldflags=-s -w`) : **~6.96 MB**. Le Makefile ajoute `-trimpath -s -w` ce qui réduira encore la taille en pratique.

## Build tags
- **`amd64` / `!amd64`** : 5 fichiers `internal/bigfft/` portent ces contraintes (`arith_amd64.go`, `arith_generic.go`, `cpu_amd64.go`, plus tests). Le fallback `!amd64` (`arith_generic.go`) fournit des implémentations Go portables des fonctions vectorielles, et est correctement sélectionné pour les 3 cibles arm64 (linux/windows/darwin). Aucun symbole CPU n'est référencé en dehors du fichier `cpu_amd64.go` (aucune fuite hors compilation conditionnelle). **OK.**
- **`gmp`** : nécessite cgo + libgmp ; cross-compile hors scope (toolchain C cross-compile requise). Par défaut (sans tag), aucune dépendance cgo n'est introduite — confirmé par la compilation propre des 6 cibles. **OK pour le scope demandé.**

## Cohérence Makefile vs réel
Le Makefile expose `build-all` qui chaîne : `build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin` (ce dernier produit amd64+arm64 macOS). Le set Makefile couvre donc bien les **6 cibles** testées. Cohérent avec la réalité du build.

Écart documentaire (audit 7.1) : `BUILD.md` omet linux-arm64 et windows-arm64 alors que le Makefile les supporte. Action de doc requise (hors scope code).

Note : `build-pgo-all` ne couvre que linux/windows/darwin **amd64** (+ darwin arm64). Les cibles arm64 linux/windows ne bénéficient pas du build PGO via Makefile — incohérence mineure entre `build-all` et `build-pgo-all`.

## Synthèse
- **Score : 6/6 cibles OK.**
- Top actions :
  1. Synchroniser `BUILD.md` avec le Makefile (ajouter linux/arm64 et windows/arm64). [Doc, P2]
  2. Étendre `build-pgo-all` à linux/arm64 et windows/arm64 pour parité avec `build-all`. [Makefile, P3]
  3. (Optionnel) Capturer stderr dans `_build_raw.log` lors des prochains audits (`2>&1`) pour disposer d'une trace même en cas de succès silencieux.
