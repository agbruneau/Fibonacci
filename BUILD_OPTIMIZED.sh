#!/bin/bash
# Script de compilation optimisé pour fibcalc

echo "🚀 Compilation optimisée de fibcalc..."

# Compilation avec flags d'optimisation
go build -ldflags="-s -w" -o fibcalc ./cmd/fibcalc

if [ $? -eq 0 ]; then
    echo "✅ Compilation réussie !"
    echo ""
    echo "📦 Taille du binaire :"
    ls -lh fibcalc | awk '{print "   "$5" - fibcalc"}'
    echo ""
    echo "🎯 Utilisation recommandée :"
    echo "   ./fibcalc -n 1000000 --details"
    echo ""
    echo "🔧 Options avancées :"
    echo "   ./fibcalc --auto-calibrate     # Calibration rapide"
    echo "   ./fibcalc --calibrate          # Calibration complète"
    echo ""
else
    echo "❌ Erreur de compilation"
    exit 1
fi
