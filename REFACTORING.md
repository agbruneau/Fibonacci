# Guide de Migration - Refactoring Architectural de main.go

Ce document décrit les changements effectués lors du refactoring architectural de `cmd/fibcalc/main.go` et fournit un guide pour les développeurs qui travaillent sur le projet.

## Résumé des Changements

Le fichier `cmd/fibcalc/main.go` (368 lignes) a été décomposé en modules distincts avec une séparation claire des responsabilités :

### Avant
```
cmd/fibcalc/
└── main.go              # 368 lignes - toutes responsabilités mélangées
```

### Après
```
cmd/fibcalc/
└── main.go              # ~30 lignes - Point d'entrée minimal

internal/
├── app/                 # NOUVEAU - Application core
│   ├── app.go           # Application struct et Run()
│   ├── lifecycle.go     # Gestion context/timeout/signals
│   └── version.go       # Version info et affichage
├── cli/                 # ÉTENDU
│   └── calculate.go     # NOUVEAU - Helpers de calcul
└── ...                  # Packages existants inchangés
```

## Structure du Package `internal/app`

### `app.go`
- **`Application`** : Structure principale encapsulant la configuration et la factory
- **`New(args, errWriter)`** : Crée une nouvelle instance d'application
- **`Run(ctx, out)`** : Exécute l'application selon le mode configuré
- **`IsHelpError(err)`** : Vérifie si l'erreur est une demande d'aide

### `lifecycle.go`
- **`SetupContext(ctx, timeout)`** : Configure le timeout
- **`SetupSignals(ctx)`** : Configure la gestion des signaux
- **`SetupLifecycle(ctx, timeout)`** : Combine timeout et signaux
- **`CancelFuncs`** : Structure pour le cleanup

### `version.go`
- **Variables** : `Version`, `Commit`, `BuildDate` (ldflags)
- **`HasVersionFlag(args)`** : Détecte le flag --version
- **`PrintVersion(out)`** : Affiche les informations de version
- **`GetVersionInfo()`** : Retourne les infos de version

## Structure du Package `internal/cli` (Extensions)

### `calculate.go`
- **`GetCalculatorsToRun(cfg, factory)`** : Sélectionne les calculateurs
- **`PrintExecutionConfig(cfg, out)`** : Affiche la configuration
- **`PrintExecutionMode(calculators, out)`** : Affiche le mode d'exécution

## Migration du Code

### Avant (dans main.go)

```go
func main() {
    if hasVersionFlag(os.Args[1:]) {
        printVersion(os.Stdout)
        os.Exit(0)
    }
    cfg, err := config.ParseConfig(...)
    // ... 350+ lignes de code
}
```

### Après (nouveau main.go)

```go
func main() {
    if app.HasVersionFlag(os.Args[1:]) {
        app.PrintVersion(os.Stdout)
        os.Exit(apperrors.ExitSuccess)
    }
    application, err := app.New(os.Args, os.Stderr)
    if err != nil {
        if app.IsHelpError(err) {
            os.Exit(apperrors.ExitSuccess)
        }
        os.Exit(apperrors.ExitErrorConfig)
    }
    os.Exit(application.Run(context.Background(), os.Stdout))
}
```

## Migration des Tests

### Avant

```go
func TestRunFunction(t *testing.T) {
    cfg := config.AppConfig{...}
    exitCode := run(context.Background(), cfg, &buf)
}
```

### Après

```go
func TestApplicationRun(t *testing.T) {
    application := &app.Application{
        Config:    config.AppConfig{...},
        Factory:   fibonacci.GlobalFactory(),
        ErrWriter: &bytes.Buffer{},
    }
    exitCode := application.Run(context.Background(), &buf)
}
```

## Flags ldflags Mis à Jour

Les variables de build ont été déplacées vers le package `app` :

```bash
# Avant
go build -ldflags="-X main.Version=v1.0.0 -X main.Commit=abc123"

# Après
go build -ldflags="-X example.com/fibcalc/internal/app.Version=v1.0.0 -X example.com/fibcalc/internal/app.Commit=abc123 -X example.com/fibcalc/internal/app.BuildDate=2025-01-01T00:00:00Z"
```

## Compatibilité

### API CLI : 100% Compatible
- Tous les flags et options existants sont préservés
- Même comportement observable pour les utilisateurs
- Même format de sortie (JSON, quiet, verbose, etc.)

### API Go : Changements Mineurs
- La fonction `run()` n'est plus exportée depuis `main`
- Utiliser `app.Application.Run()` à la place
- Les tests doivent être mis à jour pour utiliser la nouvelle API

## Avantages du Refactoring

1. **Séparation des responsabilités** : Chaque package a une responsabilité unique
2. **Testabilité** : Les composants peuvent être testés indépendamment
3. **Maintenabilité** : Code plus facile à comprendre et modifier
4. **Réutilisabilité** : Le package `app` peut être utilisé comme bibliothèque
5. **Extensibilité** : Ajout facile de nouvelles commandes ou modes

## Tests

Exécuter tous les tests :

```bash
go test ./... -v
```

Vérifier la couverture :

```bash
go test ./internal/app/... -cover
go test ./cmd/fibcalc/... -cover
```

## Fichiers Modifiés/Créés

| Fichier | Action | Description |
|---------|--------|-------------|
| `cmd/fibcalc/main.go` | Modifié | Réduit à ~30 lignes |
| `internal/app/app.go` | Créé | Application core |
| `internal/app/lifecycle.go` | Créé | Gestion lifecycle |
| `internal/app/version.go` | Créé | Gestion version |
| `internal/app/app_test.go` | Créé | Tests app |
| `internal/app/lifecycle_test.go` | Créé | Tests lifecycle |
| `internal/app/version_test.go` | Créé | Tests version |
| `internal/cli/calculate.go` | Créé | Helpers CLI |
| `internal/cli/calculate_test.go` | Créé | Tests CLI |
| `cmd/fibcalc/main_test.go` | Modifié | Adapté à la nouvelle API |

