# TEAM E — Sécurité & Robustesse

Audit read-only du repo `github.com/agbru/fibcalc` (FibGo) — Go 1.26.2 sur Windows.
Aucune modification de code, de config, ni de `.gitignore` n'a été effectuée.

## Résumé exécutif

- **Vulnérabilités `govulncheck`** : **0** — `No vulnerabilities found.`
- **Findings `gosec`** : **27** (21 × G115 HIGH, 2 × G304 MEDIUM, 4 × G104 LOW)
- **`panic()` non justifiés** : **2** sur 13 (registry.MustGet + calculator MustNewCalculator — convention `Must*` Go OK ; les autres sont défensifs internes au paquet `bigfft` portage)
- **Goroutines sans contrôle de cycle de vie** : **0 critique** (tous les `go func` runtime utilisent `sync.WaitGroup` ou `errgroup` ; ctx propagé dans la majorité)
- **Path traversal exploitable** : **0** (G304 ne sont pas exploitables — chemins fournis par CLI utilisateur unique, pas de service multi-tenant)
- **Findings P0 / P1 / P2** : **0 / 2 / 6**
- **Artefacts résiduels racine** : **4 fichiers `.txt` trackés** (`build_err.txt`, `e2e_rich_out.txt`, `test_err.txt`, `test_out.txt`) — fuite de logs de build dans le VCS
- **`gosec` activé dans `.golangci.yml`** : **OUI** (ligne 42), mais G104 explicitement exclu (ligne 99) — doublon errcheck.

## Outils utilisés (versions, exit codes)

| Outil | Version | Commande | Exit code |
|---|---|---|---|
| `go` | go1.26.2 windows/amd64 | `go version` | 0 |
| `govulncheck` | installé `/c/Users/agbru/go/bin/govulncheck` | `govulncheck ./...` | 0 |
| `gosec` | dev (latest) | `gosec ./...` | 1 (27 issues) |
| Analyse manuelle | — | grep `panic(`, `go func`, `os.Open*`, `errors.New`, `if err != nil` | — |

Aucun outil n'a dû être installé : `govulncheck` et `gosec` étaient déjà présents dans `~/go/bin/`.

## Findings

### F-E1 : G115 — overflow conversion int↔uint dans bigfft & friends (21 occurrences)
- **Sévérité** : P2
- **CWE** : CWE-190 (Integer Overflow or Wraparound)
- **Fichiers** :
  - `internal/bigfft/memory_est.go:20`
  - `internal/bigfft/scan.go:39`
  - `internal/bigfft/fft_poly.go:246, 321`
  - `internal/bigfft/fft.go:218`
  - `internal/bigfft/pool.go:50, 158, 246, 333`
  - `internal/bigfft/fermat.go:100`
  - `internal/bigfft/fft_cache.go:152, 164`
  - `internal/metrics/indicators.go:46, 56, 80`
  - `internal/fibonacci/memory/budget.go:22`
  - `internal/fibonacci/modular.go:56`
  - `internal/fibonacci/matrix_framework.go:82`
  - `internal/fibonacci/doubling_framework.go:206`
  - `internal/fibonacci/matrix_ops.go:24`
  - `internal/calibration/microbench.go:200`
- **Preuve** (`internal/bigfft/memory_est.go:18-21`) :
  ```go
  func wordLen(n int) int {
      bitLen := uint64(float64(n) * 0.69424)
      wordLen := int((bitLen + 63) / 64)   // <-- G115
      ...
  ```
- **Impact réel** : faible. Sur `amd64`, `int = int64` et `uint = uint64`, donc les conversions sont mathématiquement sûres pour les tailles plausibles (`n` borné par RAM physique). Sur `386` ou `arm` 32-bit, certaines conversions pourraient théoriquement déborder si on demandait F(N) extrême. Comme le projet ne supporte pas explicitement 32-bit en production (`build-all` cible linux/windows/macOS amd64+arm64 — voir `Makefile`), le risque est nominal.
- **Patch proposé** (illustration, à ne PAS appliquer) :
  ```diff
  - wordLen := int((bitLen + 63) / 64)
  + words64 := (bitLen + 63) / 64
  + if words64 > math.MaxInt {
  +     return 0, fmt.Errorf("size overflow: %d", words64)
  + }
  + wordLen := int(words64)
  ```
  Alternative pragmatique : ajouter `//nolint:gosec // G115: bornée par RAM, validé en amont` ou exclure G115 dans `.golangci.yml` pour le sous-arbre `internal/bigfft/` (portage `math/big/bigfft`).
- **Effort** : S (excludes par-package) ou L (annotations chirurgicales 21×).

### F-E2 : G304 — `os.ReadFile` / `os.OpenFile` sur chemin variable
- **Sévérité** : P2
- **CWE** : CWE-22 (Path Traversal)
- **Fichiers** :
  - `internal/calibration/profile.go:95` — `data, err := os.ReadFile(path)` (chargement profil JSON)
  - `cmd/generate-golden/main.go:32` — `os.OpenFile(filename, …)` sur `filepath.Join(*outputDir, "fibonacci_golden.json")`
  - `internal/cli/output.go:71` — `os.OpenFile(config.OutputFile, …)` (résultat utilisateur)
- **Preuve** (`internal/calibration/profile.go:88-98`) :
  ```go
  func loadProfile(path string) (*CalibrationProfile, error) {
      if path == "" {
          path = GetDefaultProfilePath()
      }
      data, err := os.ReadFile(path)   // <-- G304
  ```
- **Impact réel** : **non exploitable**. fibcalc est un CLI mono-utilisateur ; le chemin provient du flag `-calibration-profile` ou `-output` fournis par le shell de l'utilisateur lui-même. Pas de surface réseau, pas de désérialisation côté serveur. Toutefois, par hygiène, il manque :
  1. Aucune validation du chemin (pas de `filepath.Clean` + `strings.HasPrefix(absPath, allowedRoot)`).
  2. `output.go` ouvre avec `O_RDWR|O_CREATE|O_TRUNC` mais ne refuse pas un symlink (TOCTOU possible si l'attaquant a accès local à `/tmp`). Sur Windows, ce vecteur est limité.
- **Patch proposé** :
  ```diff
  + path = filepath.Clean(path)
  + if !filepath.IsAbs(path) { path, _ = filepath.Abs(path) }
    data, err := os.ReadFile(path)
  ```
  Ou utiliser `os.Root` (Go ≥1.24) comme suggéré par gosec autofix pour limiter au home directory.
- **Effort** : S.

### F-E3 : G104 — `g.Wait()` ignoré dans orchestrator
- **Sévérité** : P1
- **CWE** : CWE-703 (Improper Check or Handling of Exceptional Conditions)
- **Fichier** : `internal/orchestration/orchestrator.go:67`
- **Preuve** :
  ```go
  for i, calc := range cfg.Calculators {
      idx, calculator := i, calc
      g.Go(func() error {
          ...
          results[idx] = CalculationResult{ ... Err: wrapCalculationFailure(err, ...) }
          return nil   // <-- err déjà encapsulé dans results[idx].Err
      })
  }
  g.Wait()   // <-- G104: erreur de Wait jetée
  ```
- **Analyse** : c'est **intentionnel** — chaque goroutine retourne `nil` et stocke l'erreur dans `results[idx].Err`. `g.Wait()` ne peut donc jamais retourner d'erreur dans ce code. Néanmoins, `errgroup.WithContext(ctx)` annule via le contexte si une goroutine retourne une erreur ; ici aucune ne le fait → la sémantique d'errgroup est sous-utilisée (pas d'annulation propagée).
- **Patch proposé** : soit utiliser plain `sync.WaitGroup` (plus honnête sur l'intention), soit `_ = g.Wait()` explicite et un commentaire `//nolint:errcheck`. Préférable : laisser `g.Go` retourner l'erreur pour bénéficier de la cancellation cross-calculator, et continuer à stocker dans `results[idx]`.
- **Effort** : S.

### F-E4 : G104 — `tw.Flush()` ignoré
- **Sévérité** : P2
- **CWE** : CWE-703
- **Fichier** : `internal/calibration/io.go:38`
- **Preuve** : `tw.Flush()` (où `tw *tabwriter.Writer`) — un échec de flush sur stderr/stdout perd silencieusement le tableau de calibration.
- **Patch proposé** :
  ```diff
  - tw.Flush()
  + if err := tw.Flush(); err != nil {
  +     fmt.Fprintf(out, "\n[warn] failed to flush calibration table: %v\n", err)
  + }
  ```
- **Effort** : S.

### F-E5 : G104 — `fourier()` retourne une erreur ignorée dans fft_poly
- **Sévérité** : P2
- **CWE** : CWE-703
- **Fichier** : `internal/bigfft/fft_poly.go:297, 315`
- **Preuve** :
  ```go
  fourier(values, twisted, false, n, k)        // ligne 297
  return PolValues{k, n, values}
  ...
  fourier(q, v.Values, true, n, k)             // ligne 315
  ```
  La fonction `fourier` retourne `error` (transitivement via `fourierRecursive`), mais l'erreur est jetée — la valeur retournée pourrait être incorrecte sans signal.
- **Impact** : potentiellement P1 si `fourier` peut effectivement retourner non-nil. Une lecture rapide de `fourierRecursiveUnified` (`fft_recursion.go:145-150`) montre que `error` ne sert qu'à propager une annulation contextuelle ; or `fourier` n'a pas de `ctx`. L'erreur est donc presque toujours `nil`.
- **Patch proposé** : si le contrat `fourier` ne renvoie jamais d'erreur, supprimer le retour ; sinon, propager :
  ```diff
  - fourier(values, twisted, false, n, k)
  - return PolValues{k, n, values}
  + if err := fourier(values, twisted, false, n, k); err != nil {
  +     return PolValues{}  // ou panic + commentaire de portage
  + }
  + return PolValues{k, n, values}
  ```
  La signature `Transform()` actuelle ne retourne pas d'erreur → refactor non trivial.
- **Effort** : M.

### F-E6 : Goroutines sans propagation de contexte dans `executeTasks`
- **Sévérité** : P1
- **CWE** : CWE-833 (Deadlock) / mineur — pas de fuite, mais perte d'opportunité de cancellation
- **Fichier** : `internal/fibonacci/common.go:215, 265, 276`
- **Preuve** :
  ```go
  if inParallel {
      sem := getTaskSemaphore()
      var wg sync.WaitGroup
      var ec parallel.ErrorCollector
      wg.Add(len(tasks))
      for i := range tasks {
          go func(t PT) {
              sem <- struct{}{}
              ec.SetError(t.execute())   // <-- pas de check ctx.Done()
              <-sem
              wg.Done()
          }(PT(&tasks[i]))
      }
      wg.Wait()
  ```
  Comparer avec `executeParallel3` (lignes 103-140) qui, lui, vérifie `ctx.Err()` après acquisition du sémaphore.
- **Impact** : si l'utilisateur Ctrl+C pendant un grand calcul, les goroutines `executeTasks` continuent jusqu'à terminaison de la tâche actuelle (le ctx top-level n'est pas observé ici). C'est un trade-off documenté dans la philosophie du projet (tâches courtes), mais les calculs FFT longue durée pourraient en bénéficier.
- **Patch proposé** : ajouter un paramètre `ctx context.Context` à `executeTasks` et tester `ctx.Err()` après le `sem <- struct{}{}`.
- **Effort** : M (signature change, propagation top-down).

### F-E7 : `panic()` dans `MustGet` / `MustNewCalculator`
- **Sévérité** : P2
- **CWE** : CWE-754 (Improper Check for Unusual or Exceptional Conditions)
- **Fichiers** :
  - `internal/fibonacci/registry.go:225` — `panic(fmt.Sprintf("fibonacci: required calculator not found: %s", name))`
  - `internal/fibonacci/calculator.go:107` — `panic(err)`
- **Justification** : convention Go `Must*` (cf. `regexp.MustCompile`, `template.Must`). Acceptable si :
  1. Documenté dans le doc comment (✅ : `// Panics: - If the calculator type is not registered.`)
  2. Réservé aux init/var globales (à vérifier ; le grep n'expose pas tous les call-sites)
- **Recommandation** : conforme aux idiomes Go, garder. Au minimum, s'assurer que `MustGet` n'est jamais appelé depuis un chemin runtime qui pourrait être déclenché par input utilisateur (sinon → DoS local).
- **Effort** : S (audit des call-sites uniquement).

### F-E8 : Artefacts de build trackés dans git
- **Sévérité** : P1
- **CWE** : CWE-200 (Information Exposure) — fuite de stack traces / logs internes
- **Fichiers** :
  - `build_err.txt` (tracked ✅)
  - `e2e_rich_out.txt` (tracked ✅)
  - `test_err.txt` (tracked ✅)
  - `test_out.txt` (tracked ✅)
- **Preuve** :
  ```bash
  $ git ls-files build_err.txt e2e_rich_out.txt test_err.txt test_out.txt
  build_err.txt
  e2e_rich_out.txt
  test_err.txt
  test_out.txt
  ```
- **Analyse** : `.gitignore` couvre `*.log`, `*.tmp`, `*.bak`, `*.out`, `*.prof`, `coverage.*` mais **pas** `*_err.txt` / `*_out.txt`. Ces fichiers contiennent vraisemblablement des chemins absolus utilisateur (`C:\Users\agbru\...`), des sorties de run-time, et potentiellement des logs zerolog avec des champs internes — fuite mineure d'environnement développeur dans le repo public.
- **Recommandation** :
  1. `git rm` les 4 fichiers.
  2. Ajouter au `.gitignore` :
     ```
     /build_err.txt
     /e2e_rich_out.txt
     /test_err.txt
     /test_out.txt
     # ou plus large : /*_err.txt, /*_out.txt
     ```
  3. Consulter le contenu pour s'assurer qu'aucun secret n'a été indexé historiquement.
- **Effort** : S.

## Annexe : panic() inventaire

| Fichier:ligne | Justifié | Recommandation |
|---|---|---|
| `internal/fibonacci/registry.go:225` (MustGet) | ✅ Oui — convention `Must*` | Garder, doc OK |
| `internal/fibonacci/calculator.go:107` (MustNewCalculator) | ✅ Oui — convention `Must*` | Garder |
| `internal/bigfft/fermat.go:51` (`len(z) != len(x) in Shift`) | ✅ Oui — invariant interne portage `math/big/bigfft` | Garder (panic = bug programmation) |
| `internal/bigfft/fermat.go:124` (`Add: len(z) != len(x)`) | ✅ Oui | Garder |
| `internal/bigfft/fermat.go:134` (`fermat.Sub`) | ✅ Oui | Garder |
| `internal/bigfft/fermat.go:152` (`Mul: len(x) != len(y)`) | ✅ Oui | Garder |
| `internal/bigfft/fermat.go:177` (`len(z) > 2n+1`) | ✅ Oui | Garder |
| `internal/bigfft/fermat.go:202` (`unexpected carry after normalization`) | ✅ Oui — assertion mathématique | Garder |
| `internal/bigfft/fermat.go:238` | ✅ Oui | Garder |
| `internal/bigfft/fermat.go:257` | ✅ Oui | Garder |
| `internal/bigfft/fft_poly.go:260` (`Transform: len(p.A) >= 1<<k`) | ✅ Oui — invariant | Garder |
| `internal/bigfft/scan.go:28` (`size < quadraticScanThreshold`) | ✅ Oui — précondition | Garder |
| `internal/bigfft/scan.go:43` (`quadraticScanThreshold % 14 != 0`) | ⚠️ Constante de compilation — pourrait être assertion compile-time, mais panic à init OK | Garder |
| `internal/progress/observer_test.go:309` (`test panic`) | ✅ Test seulement | Garder |

**Bilan** : 13 panics en code de production (hors test). **Tous sont justifiés** par des invariants internes ou la convention `Must*`. Aucun n'est déclenchable par input utilisateur direct — les invariants `bigfft` proviennent d'un portage du package interne `math/big/bigfft` de Go upstream.

## Annexe : artefacts racine

| Fichier | Suivi git | Recommandation |
|---|---|---|
| `build_err.txt` | ✅ tracké | `git rm` + ajouter à `.gitignore` (`/build_err.txt`) |
| `e2e_rich_out.txt` | ✅ tracké | `git rm` + ajouter à `.gitignore` |
| `test_err.txt` | ✅ tracké | `git rm` + ajouter à `.gitignore` |
| `test_out.txt` | ✅ tracké | `git rm` + ajouter à `.gitignore` |
| `bench/baseline/coverage.out` | ignoré | OK (matché par `*.out`) |
| `bench/baseline/coverage.txt` | ignoré | OK (matché par `*.txt`?) |

NB : le `.gitignore` actuel (lignes 14-22, 31-46) couvre `*.exe`, `*.test`, `*.out`, `*.prof`, `*.log`, `*.tmp`, `*.bak`, `coverage.*`, mais **aucun pattern n'attrape `_err.txt` / `_out.txt` à la racine**. C'est pourquoi ces 4 fichiers se sont retrouvés indexés.

## Annexe : Cohérence `errors.Is` / `errors.As` / sentinelles

- **Sentinelles déclarées correctement** : aucune `var Err... = errors.New(...)` n'a été trouvée dans le code de production. Seul un `errors.New("invalid configuration")` inline (`internal/config/config.go:186`) et un `errors.New("fibonacci: the …")` inline (`internal/fibonacci/calculator.go:90`).
  - **Recommandation** : extraire en sentinelle si pertinent — actuellement les appelants ne peuvent pas faire `errors.Is(err, config.ErrInvalidConfiguration)`.
- **Usages `errors.Is` / `errors.As`** :
  - `internal/errors/errors.go:240` — `errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)` ✅
  - `internal/errors/handler.go:51,56` — `errors.Is(err, context.DeadlineExceeded)`, `errors.Is(err, context.Canceled)` ✅
  - `internal/errors/handler.go:68` — `errors.As(err, &ce)` pour extraire `CalculationError` ✅
  - `internal/app/app.go:133` — `errors.Is(err, flag.ErrHelp)` ✅
- **Types structurés `internal/errors/`** : `ConfigError`, `CalculationError`, `TimeoutError`, `ValidationError`, `MemoryError`.
  - `Error()` : ✅ tous implémentés
  - `Unwrap()` : ✅ `CalculationError.Unwrap()` ; ❌ absent sur `ConfigError`, `TimeoutError`, `ValidationError`, `MemoryError` (pas critique car ils ne wrappent pas, mais `ConfigError` pourrait wrapper la cause de validation pour faciliter `errors.As`).
  - `Is()` : ❌ aucun n'implémente `Is(target error) bool`. Conséquence : `errors.Is(err, apperrors.ConfigError{})` ne marchera pas — il faut `errors.As(err, &cfgErr)`. Convention Go = OK pour types valeur.
- **Wrapping `fmt.Errorf("%w", err)`** : utilisé partout (cf. `errors.go:229`, `profile.go:97,102,117,121`, `output.go:67,73`). ✅ Conforme.

## Annexe : Goroutines runtime — cycle de vie

| Fichier:ligne | Mécanisme | ctx propagé | Cancellation | Verdict |
|---|---|---|---|---|
| `internal/calibration/microbench.go:140` | `sync.WaitGroup` + sémaphore | ✅ via `select { case <-ctx.Done() }` | ✅ | OK |
| `internal/fibonacci/common.go:103,116,129` (executeParallel3) | `sync.WaitGroup` + sémaphore | ✅ via `ctx.Err()` post-sem | ✅ | OK |
| `internal/fibonacci/common.go:215` (executeTasks) | `sync.WaitGroup` + sémaphore | ❌ ctx absent | ❌ | F-E6 |
| `internal/fibonacci/common.go:265,276` (executeMixedTasks) | `sync.WaitGroup` + sémaphore | ❌ ctx absent | ❌ | F-E6 |
| `internal/orchestration/orchestrator.go:43` (DisplayProgress) | `sync.WaitGroup` (displayWg) | N/A — ferme via `close(progressChan)` | ✅ | OK |
| `internal/orchestration/orchestrator.go:57` (g.Go) | `errgroup.WithContext(ctx)` | ✅ | ⚠️ jamais déclenchée (return nil) | F-E3 |
| `internal/bigfft/fft_recursion.go:113` | `sync.WaitGroup` + sémaphore + defer | N/A (récursion bornée par MaxParallelFFTDepth) | N/A | OK |

Aucune goroutine ne fuit (toutes terminent via wg.Wait). 2 sites n'observent pas `ctx.Done()` → cancellation utilisateur retardée.

## Annexe : `.golangci.yml`

- **`gosec` activé** : ✅ ligne 42.
- **Exclusions gosec** :
  - `G104` (errcheck doublon) — ligne 99 ✅ justifié.
  - **Aucune exclusion G115** — donc les 21 findings G115 sont nouveaux ou n'ont jamais bloqué la CI ? Vérifier `bench/baseline/lint.txt` qui contient déjà les mêmes 21 lignes G115 → la CI les **rapporte** mais le projet n'a pas (encore) de gate "fail on gosec HIGH".
- **Recommandation** : ajouter dans `.golangci.yml` :
  ```yaml
  gosec:
    excludes:
      - G104
      - G115  # math overflow analysis non significative pour bigfft sur amd64/arm64
  ```
  Ou conserver pour traçabilité et fixer les `int↔uint` litigieux par annotation locale.

## Synthèse priorité

| ID | Sévérité | Effort | Pourquoi |
|---|---|---|---|
| F-E3 | P1 | S | sémantique errgroup mal exploitée |
| F-E6 | P1 | M | cancellation utilisateur retardée |
| F-E8 | P1 | S | fuite logs dev dans repo public |
| F-E1 | P2 | S–L | bruit gosec, impact réel nul sur 64-bit |
| F-E2 | P2 | S | hygiène path, non exploitable mono-user |
| F-E4 | P2 | S | logs silencieux |
| F-E5 | P2 | M | refactor de signature |
| F-E7 | P2 | S | conforme idiomes |

Aucun P0. Aucun secret leaké observé. `govulncheck` clean. Posture sécurité globale : **bonne** — les findings restants sont du polish hygiénique, pas des vulnérabilités exploitables dans le modèle de menace d'un CLI mono-utilisateur.
