# 30 — Audit Sécurité

Date : 2026-04-28 — Périmètre : revue OWASP + supply-chain ciblée, sans appel réseau (pas de `govulncheck`).

## Surface d'attaque

- **Entrée CLI** (`flag.FlagSet`) — `internal/config/config.go:141-189`. Aucune borne supérieure sur `N` (uint64) ni sur les seuils int.
- **Variables d'environnement** `FIBCALC_*` — `internal/config/env.go`. Parsing best-effort (silencieux sur erreur).
- **Fichiers** lus : profil de calibration JSON (`~/.fibcalc_calibration.json`), golden tests JSON (test seul), fichier de sortie (write).
- **Appels système** : `os.UserHomeDir`, `runtime`, `gopsutil/v4` (sysmon).
- **cgo** : backend GMP optionnel (build tag `gmp`), `github.com/ncw/gmp v1.0.5`.
- **Pas de surface réseau** : aucun `net/http`, `net.Listen`, `net.Dial` dans le code applicatif.

## Supply chain

- **`go.sum`** : présent, 692 lignes, hashes h1+go.mod pour chaque module (intégrité OK).
- **`go.mod`** : `go 1.25.0`, toolchain `1.26.2`. **Aucune directive `replace` ni `exclude`** (vérifié, pas de match).
- **Modules notables** :
  - `github.com/ncw/gmp v1.0.5` — wrapper cgo de libgmp (dernière release 2022). Maintien faible mais surface limitée.
  - `github.com/charmbracelet/bubbletea v1.3.10` — récent, activement maintenu.
  - `github.com/shirou/gopsutil/v4 v4.26.3` — récent.
  - `github.com/rs/zerolog v1.35.0`, `golang.org/x/sync v0.20.0`, `golang.org/x/sys v0.43.0` — versions à jour.
- **Heuristique vulnérabilités connues** : aucun module historiquement compromis (gopkg.in/yaml.v2 < 2.2.8, etc.) présent. `crypto`/auth absents → surface étroite.
- **Recommandation** : exécuter `govulncheck ./...` lors de la prochaine connexion réseau.

## Validation des entrées

| Vecteur | Localisation | Validation présente | Risque |
|---|---|---|---|
| CLI `-n` (uint64) | `config.go:147` | Aucune borne haute ; `Validate()` vérifie seulement Timeout/Threshold ≥ 0 | DoS : `-n 18446744073709551615` engage RAM/CPU jusqu'à OOM. Mitigé par `--timeout` (5min défaut) et `--memory-limit`, mais ces deux sont opt-in. Sévérité : **Modéré** |
| CLI `-threshold`, `-fft-threshold`, `-strassen-threshold` (int) | `config.go:154-156` | `Validate()` rejette < 0 (`config.go:103-108`). Pas de borne max | Faible |
| CLI `-algo` | `config.go:153` | Whitelist via `availableAlgos` (`config.go:109-118`) | OK |
| CLI `-output` (chemin) | `config.go:161` | **Aucune validation** ; `os.OpenFile` direct (`output.go:71`). Pas de `filepath.Clean`/`Abs` | Path traversal théorique (déjà signalé G304 en P2-10). Outil mono-utilisateur local → exploitabilité nulle. Sévérité : **Faible** |
| CLI `-calibration-profile` | `config.go:159` | Idem : chargement direct `os.ReadFile` (`profile.go:95`) | Idem ci-dessus |
| ENV `FIBCALC_*` | `env.go:113-179` | Best-effort (silencieux si parse échoue) | Confusion utilisateur, pas un risque sécurité |
| JSON profil de calibration | `profile.go:101` | `json.Unmarshal` sur `CalibrationProfile` typé (champs scalaires) | Faible — types fermés, pas de `interface{}` |
| JSON golden tests | `fibonacci_golden_test.go:29` | Test-only, données contrôlées | Hors scope |

## OWASP findings

| Catégorie | Détail | Sévérité | Localisation |
|---|---|---|---|
| A01 Broken Access Control | N/A (mono-user CLI) | — | — |
| A03 Injection / Command exec | Aucun `exec.Command` en code de production. Tous les hits sont dans `test/e2e/*` et `cmd/fibcalc/main_test.go` (build du binaire, args constants). | **Aucun** | — |
| A03 Path traversal | `-output` et `-calibration-profile` non assainis | Faible (mono-user) | `cli/output.go:71`, `calibration/profile.go:95` |
| A05 Misconfiguration | `-memory-limit` non défini par défaut → DoS via `-n` géant | Modéré | `config.go:171`, `app.go` |
| A08 Software/data integrity | `go.sum` présent, modules pinés ; pas de `replace` douteuse | OK | `go.mod`, `go.sum` |
| A09 Logging | `zerolog` utilisé ; aucun log ne dump `os.Environ()` ni `flag.Args()`. Type `errors.ConfigExcerpt` documente "no secrets or tokens" (`errors.go:66`). | OK | — |
| Crypto faible | `math/rand` utilisé uniquement dans tests (`bigfft/arith_amd64_test.go`, `fermat_test.go`). `crypto/rand` utilisé dans tests aléatoires de précision. Aucun usage cryptographique réel — pas de `md5`/`sha1` dans le code. | OK | — |
| Overflow entiers | 21 G115 connus (P2-09) : conversions int↔uint sur valeurs bornées par `bitLen`, faux positifs documentés. Pas de validation explicite, mais `MaxFibUint64 = 93` (`calculator.go:17`) borne le chemin uint64. | Faible | `internal/bigfft/*`, `fibonacci/*` |
| `unsafe.Pointer` | 2 occurrences, **uniquement en test** (`memory/arena_test.go:53-54`) pour vérifier la contiguïté de l'arène | OK | — |

## Concurrence

- **`sync.Pool`** : utilisé largement (`bigfft/pool.go`, `fft_cache.go`, `pool_warming.go`) pour `big.Int`. Les objets ne contiennent pas de données sensibles (entiers de calcul) — risque sécurité nul. Hygiène (`Reset` avant `Put`) : à vérifier en revue perf, hors scope sécurité.
- **`errgroup`/sémaphores** : pattern documenté (CLAUDE.md), `golang.org/x/sync` utilisé. P2-02 signale une sur-souscription perf, pas une race.
- **Race detector** : `make test` active `-race` (CLAUDE.md). Audit Phase 1 note que la machine d'audit Windows n'a pas pu exécuter `-race` faute de gcc — à revalider sur Linux/macOS (déjà tracé dans `AUDIT_REPORT.md:218`).
- **Goroutines orphelines** : aucun `go func()` sans contexte/wg détecté dans le code de production lors de cette revue ciblée.

## Secrets

- **`.env.example`** (lu) : template de configuration applicative (FIBCALC_N, FIBCALC_TIMEOUT, etc.). **Aucun secret** — pas de token, password, clé API. Conforme à un template public.
- **Grep `password|secret|api_key|token|BEGIN PRIVATE KEY`** sur le repo : 3 hits, tous légitimes :
  - `PLAN.md:100` (texte d'audit qui parle de l'absence de secrets)
  - `errors/errors.go:66` (commentaire : `ConfigExcerpt` ne doit pas contenir de secrets)
  - `errors/errors_test.go:446` (string littéral `"line1\nline2\tsecret"` = donnée de test pour valider le strip)
- **`git log -- .env*`** : aucun fichier `.env`/`.env.local`/`.env.production` n'a jamais été commité (sortie vide). Seul `.env.example` est tracké (commit `9177ade`, contenu propre).
- **`.gitignore`** : `*.exe`, `*.out`, `*.pgo` couverts ; pas d'entrée explicite `.env` mais aucune fuite historique.

## Surface cgo (build tag gmp)

- **Localisation** : `internal/fibonacci/calculator_gmp.go` (build tag `//go:build gmp`).
- **API utilisée** : wrappers Go de `github.com/ncw/gmp` (`gmp.NewInt`, `MulUint32`, `Mul`, `Add`, `Sub`, `Set`, `Bytes`). **Aucun appel direct** à `C.malloc`/`C.free`/`C.CString` — toute la gestion mémoire est déléguée au wrapper, qui utilise `runtime.SetFinalizer` pour libérer les `mpz_t`.
- **Validation des entrées** : `n` est uint64 borné par l'API ; les `gmp.Int` temporaires sont préalloués (`t1`, `t2`) avant la boucle (`calculator_gmp.go:117-120`).
- **Annulation** : `ctx.Done()` vérifié à chaque itération (`calculator_gmp.go:134-138`) — pas de fuite de goroutine en cas d'annulation.
- **Risque résiduel** : dépendance non maintenue activement (`ncw/gmp` v1.0.5 — pas de release récente). Backend opt-in, donc surface effective nulle pour les builds par défaut.

## Synthèse

- **Findings critiques (P0)** : 0
- **Findings élevés (P1)** : 0
- **Findings modérés (P2)** : 2
  1. Absence de borne supérieure sur `-n` → DoS local (mémoire/CPU) si `--memory-limit` non spécifié.
  2. Path traversal théorique sur `-output` / `-calibration-profile` (déjà tracé G304 P2-10, non exploitable mono-user).
- **Findings faibles** : 2 (overflow G115 documentés P2-09 ; `os.OpenFile` permissions `0600` correctes — RAS).

### Top 5 actions priorisées

1. **(M-1)** Ajouter une borne par défaut sur `-n` (ex. `MaxN = 1e10`) ou rendre `--memory-limit` obligatoire au-delà d'un seuil — `config.go:99-120`.
2. **(M-2)** Assainir `-output` et `-calibration-profile` : `filepath.Clean` + interdire les paths absolus hors `$HOME` ou `cwd`, ou documenter explicitement la contrainte « usage local fiable » — `cli/output.go:71`, `calibration/profile.go:95`.
3. **(L-1)** Exécuter `govulncheck ./...` en CI dès reconnexion (cible Makefile à ajouter).
4. **(L-2)** Pinner `github.com/ncw/gmp` ou évaluer un fork maintenu ; documenter le risque dans `docs/algorithms/GMP.md`.
5. **(L-3)** Ajouter `.env` (sans `.example`) à `.gitignore` à titre préventif.
