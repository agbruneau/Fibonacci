#!/bin/bash
# Ce script a été enregistré avec des fins de ligne Unix (LF) pour garantir la
# compatibilité avec Git Bash sous Windows.

# Quitter immédiatement si une commande échoue.
set -e

# --- Vérification de l'environnement ---

# Vérifier si 'go' est dans le PATH.
if ! command -v go &> /dev/null; then
    echo "'go' introuvable. Tentative de localisation dans les répertoires communs de Windows..."
    # Chemins d'installation courants de Go sous Windows pour Git Bash
    COMMON_GO_PATHS=(
        "/c/Program Files/Go/bin"
        "/c/Go/bin"
        "$HOME/go/bin"
    )

    for p in "${COMMON_GO_PATHS[@]}"; do
        if [ -x "$p/go" ]; then
            echo "Go trouvé dans '$p'. Ajout au PATH pour ce script."
            export PATH="$p:$PATH"
            break
        fi
    done

    # Vérifier à nouveau après avoir modifié le PATH.
    if ! command -v go &> /dev/null; then
        echo "ERREUR: Impossible de trouver l'exécutable 'go'. Veuillez l'installer et"
        echo "l'ajouter à votre PATH avant d'exécuter ce script."
        exit 1
    fi
fi

# S'assurer que l'outil godoc est installé.
echo "Vérification et installation de l'outil godoc si nécessaire..."
go install golang.org/x/tools/cmd/godoc@latest

GODOC_CMD_PATH=$(go env GOPATH)/bin/godoc
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    if [ -f "${GODOC_CMD_PATH}.exe" ]; then
        GODOC_CMD_PATH="${GODOC_CMD_PATH}.exe"
    fi
fi

# --- Configuration ---

# Le port sur lequel le serveur godoc écoutera.
PORT=${PORT:-6060}
# Le répertoire de sortie pour la documentation générée.
OUTPUT_DIR=${OUTPUT_DIR:-doc}
# URL du paquet à documenter.
PACKAGE_URL="http://localhost:$PORT/pkg/example.com/fibcalc/"

# --- Exécution ---

echo "Lancement du serveur godoc en arrière-plan sur le port $PORT..."
# Exécute godoc en arrière-plan.
"$GODOC_CMD_PATH" -http=":$PORT" &> godoc.log &
GODOC_PID=$!

# Attendre que le serveur soit prêt.
echo "Attente du démarrage du serveur godoc (10 secondes)..."
sleep 10

# Tuer le serveur godoc à la sortie du script (succès ou erreur).
trap "echo 'Arrêt du serveur godoc...'; kill $GODOC_PID &> /dev/null" EXIT

echo "Téléchargement de la documentation dans le répertoire '$OUTPUT_DIR'..."
# Télécharger le site de documentation.
wget --recursive --no-clobber --page-requisites --html-extension \
     --convert-links --restrict-file-names=windows --domains localhost \
     --no-parent --no-verbose --show-progress \
     "$PACKAGE_URL" -P "$OUTPUT_DIR"

echo "Nettoyage des fichiers téléchargés..."
# Renommer le répertoire de sortie pour un nom plus propre.
mv "$OUTPUT_DIR/localhost+$PORT" "$OUTPUT_DIR/fibcalc"
# Supprimer le fichier robots.txt qui n'est pas utile.
rm -f "$OUTPUT_DIR/fibcalc/robots.txt"

echo ""
echo "---------------------------------------------------------------------"
echo "Documentation générée avec succès dans le répertoire '$OUTPUT_DIR/fibcalc'."
echo "Ouvrez le fichier suivant dans votre navigateur pour commencer:"
echo "file://$(pwd | sed 's/\\/\//g')/$OUTPUT_DIR/fibcalc/pkg/example.com/fibcalc/index.html"
echo "---------------------------------------------------------------------"