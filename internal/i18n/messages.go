// Package i18n centralise les messages destinés à l'utilisateur pour la CLI.
// Elle fournit une base simple pour l'internationalisation et garantit
// l'uniformité du ton et du vocabulaire affichés par l'application.
package i18n

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


