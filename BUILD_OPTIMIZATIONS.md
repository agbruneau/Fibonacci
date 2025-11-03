# Optimisations de Build Go

Ce document d?crit les optimisations de build disponibles pour r?duire la taille du binaire et am?liorer les performances.

## Options de Build pour R?duire la Taille du Binaire

### Build Minimal (Recommand? pour la Production)

```bash
# Build avec optimisations maximales et suppression des symboles de debug
go build -ldflags="-s -w" -trimpath -o fibcalc ./cmd/fibcalc
```

**Options utilis?es :**
- `-ldflags="-s -w"` : Supprime les tables de symboles (`-s`) et les informations de debug DWARF (`-w`), r?duisant significativement la taille
- `-trimpath` : Supprime les chemins de syst?me de fichiers des binaires pour la reproductibilit?

### Build avec UPX (Compression suppl?mentaire)

Apr?s le build, vous pouvez compresser le binaire avec UPX :

```bash
# Installation UPX (Linux)
sudo apt-get install upx-ucl  # Debian/Ubuntu
# ou
brew install upx  # macOS

# Compression du binaire
upx --best --lzma fibcalc
```

**Note :** UPX peut augmenter l?g?rement le temps de d?marrage mais r?duit significativement la taille du binaire.

### Build avec Stripping Avanc? (Linux uniquement)

```bash
go build -ldflags="-s -w -extldflags '-static'" -trimpath -o fibcalc ./cmd/fibcalc
strip --strip-all fibcalc
```

## Optimisations de Performance Compil?es

### Build avec Optimisations CPU Sp?cifiques

```bash
# Pour une architecture CPU sp?cifique (ex: AMD64 avec AVX2)
GOARCH=amd64 GOAMD64=v2 go build -ldflags="-s -w" -trimpath -o fibcalc ./cmd/fibcalc
```

### Build avec Contr?le de la GC

Pour des calculs tr?s longs, vous pouvez ajuster les param?tres du garbage collector :

```bash
GOGC=100 go build -ldflags="-s -w" -trimpath -o fibcalc ./cmd/fibcalc
```

Puis ex?cutez avec :
```bash
GOGC=200 ./fibcalc -n 100000000  # GC moins agressif, plus de m?moire utilis?e
```

## Comparaison des Tailles

| Configuration | Taille Approximative | Notes |
|--------------|---------------------|-------|
| Build standard | ~15-20 MB | Avec symboles et debug |
| Build optimis? (`-s -w`) | ~8-12 MB | Sans symboles, r?duction ~40% |
| Build optimis? + UPX | ~3-5 MB | Compression suppl?mentaire |
| Build optimis? + static | ~10-15 MB | Binaire statique (plus portable) |

## Script de Build Automatis?

Cr?ez un script `build.sh` :

```bash
#!/bin/bash
set -e

VERSION="${1:-dev}"
OUTPUT_DIR="./builds"
BINARY_NAME="fibcalc"

mkdir -p "$OUTPUT_DIR"

echo "Building optimized binary..."
go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -trimpath \
    -o "$OUTPUT_DIR/$BINARY_NAME" \
    ./cmd/fibcalc

echo "Binary size:"
ls -lh "$OUTPUT_DIR/$BINARY_NAME" | awk '{print $5}'

# Optionnel: compression UPX
if command -v upx &> /dev/null; then
    echo "Compressing with UPX..."
    upx --best --lzma "$OUTPUT_DIR/$BINARY_NAME"
    echo "Compressed size:"
    ls -lh "$OUTPUT_DIR/$BINARY_NAME" | awk '{print $5}'
fi

echo "Build complete: $OUTPUT_DIR/$BINARY_NAME"
```

## Optimisations Incluses dans le Code

Le code inclut d?j? plusieurs optimisations au niveau source :

1. **Object Pooling** : R?utilisation d'objets via `sync.Pool` pour r?duire les allocations
2. **Calculs Lazy** : ?vite les conversions co?teuses quand elles ne sont pas n?cessaires
3. **Cache des V?rifications** : Mise en cache des appels syst?me r?p?t?s (`runtime.NumCPU`, etc.)
4. **Formatage Optimis?** : Pr?calcul de la taille des buffers pour ?viter les r?allocations
5. **Parall?lisme Adaptatif** : Activation uniquement quand b?n?fique

Ces optimisations r?duisent ? la fois l'utilisation m?moire et am?liorent les performances d'ex?cution.
