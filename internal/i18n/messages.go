// Package i18n centralise les messages destinés à l'utilisateur pour la CLI.
// Elle fournit une base simple pour l'internationalisation et garantit
// l'uniformité du ton et du vocabulaire affichés par l'application.
package i18n

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
)

// Messages regroupe les messages destinés à l'utilisateur (i18n basique).
// Centraliser ces libellés facilite la maintenance, la cohérence et une
// éventuelle traduction multi-langues à l'avenir.
var Messages = map[string]string{
    "CalibrationTitle":       "--- Mode de calibration : recherche du seuil optimal de parallélisation ---",
    "CalibrationSummary":     "--- Résumé de la calibration ---",
    "OptimalRecommendation":  "✅ Recommandation pour cette machine : --threshold %d",
    "ExecConfigTitle":        "--- Configuration de l'exécution ---",
    "ExecStartTitle":         "--- Début de l'exécution ---",
    "ComparisonSummary":      "--- Résumé de la comparaison ---",
    "GlobalStatusSuccess":    "Statut global : SUCCÈS. Tous les résultats valides sont cohérents.",
    "GlobalStatusFailure":    "Statut global : ÉCHEC. Aucun algorithme n'a pu mener le calcul à terme.",
    "StatusCriticalMismatch": "Statut global : ERREUR CRITIQUE ! Une incohérence a été détectée entre les résultats des algorithmes.",
    "StatusCanceled":         "Statut : Calcul annulé par l'utilisateur",
    "StatusTimeout":          "Statut : Échec (temps dépassé). La limite d'exécution de %s a été atteinte%s.",
    "StatusFailure":          "Statut : Échec. Une erreur imprévue s'est produite : %v",
}


// LoadFromDir charge un fichier de traduction JSON (par ex. fr.json) depuis
// un répertoire donné. En cas de succès, remplace les entrées existantes de
// Messages par celles du fichier (fallback sur les valeurs déjà présentes).
// Le format attendu est un objet JSON { "Key": "Valeur", ... }.
func LoadFromDir(dir string, lang string) error {
    if dir == "" || lang == "" {
        return errors.New("i18n: répertoire ou langue vide")
    }
    path := filepath.Join(dir, fmt.Sprintf("%s.json", lang))
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()
    dec := json.NewDecoder(f)
    loaded := map[string]string{}
    if err := dec.Decode(&loaded); err != nil {
        return err
    }
    // Merge: les entrées chargées remplacent les valeurs par défaut
    for k, v := range loaded {
        Messages[k] = v
    }
    return nil
}


