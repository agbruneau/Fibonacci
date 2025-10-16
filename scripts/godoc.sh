#!/bin/bash
set -e

# Ce script démarre un serveur godoc local et génère la documentation HTML
# pour le projet fibcalc.

# Le port sur lequel le serveur godoc écoutera.
PORT=${PORT:-6060}
# Le répertoire de sortie pour la documentation générée.
OUTPUT_DIR=${OUTPUT_DIR:-doc}

# S'assurer que l'outil godoc est installé.
echo "Vérification de l'installation de godoc..."
go install golang.org/x/tools/cmd/godoc@latest

echo "Lancement du serveur godoc sur le port $PORT..."
# Exécute godoc en arrière-plan.
~/go/bin/godoc -http=":$PORT" &> godoc.log &
GODOC_PID=$!

# Attendre que le serveur soit prêt.
echo "Attente du démarrage du serveur godoc..."
sleep 5

# Tuer le serveur godoc à la sortie.
trap "kill $GODOC_PID" EXIT

# Télécharger le site de documentation.
echo "Téléchargement de la documentation dans le répertoire $OUTPUT_DIR..."
wget --recursive --no-clobber --page-requisites --html-extension \
     --convert-links --restrict-file-names=windows --domains localhost \
     --no-parent http://localhost:$PORT/pkg/example.com/fibcalc/ -P "$OUTPUT_DIR"

# Nettoyage
mv "$OUTPUT_DIR/localhost+$PORT" "$OUTPUT_DIR/fibcalc"
rm "$OUTPUT_DIR/fibcalc/robots.txt"

echo "La documentation a été générée avec succès dans le répertoire $OUTPUT_DIR/fibcalc."
echo "Vous pouvez ouvrir $OUTPUT_DIR/fibcalc/pkg/example.com/fibcalc/index.html dans votre navigateur."