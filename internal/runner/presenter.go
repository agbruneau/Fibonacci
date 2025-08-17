package runner

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// runPresenter gère l'affichage en temps réel de la progression sur la console (CLI UI).
func (r *Runner) runPresenter(ctx context.Context, progressCh <-chan progressData) {
	// Initialisation de l'état de progression.
	status := make(map[string]float64)
	var taskNames []string
	for _, algo := range r.Algos {
		status[algo.Name] = 0.0
		taskNames = append(taskNames, algo.Name)
	}
	sort.Strings(taskNames)

	if len(taskNames) == 0 {
		return
	}

	ticker := time.NewTicker(progressRefreshRate)
	defer ticker.Stop()

	needsUpdate := true

	// Boucle principale de traitement des événements.
	for {
		select {
		case <-ctx.Done():
			// Contexte terminé (timeout/annulation). Affichage final et sortie.
			// Note: Il n'y a pas de garantie que toutes les mises à jour aient été reçues.
			r.printStatus(status, taskNames)
			fmt.Println("\n(Opération annulée ou délai dépassé)")
			return

		case p, ok := <-progressCh:
			if !ok {
				// Canal fermé (tous les workers ont fini). Affichage final et sortie.
				r.printStatus(status, taskNames)
				fmt.Println() // Nouvelle ligne finale.
				return
			}
			// Mise à jour de l'état.
			if current, exists := status[p.name]; exists && current != p.pct {
				status[p.name] = p.pct
				needsUpdate = true
			}

		case <-ticker.C:
			// Rafraîchissement périodique.
			if needsUpdate {
				r.printStatus(status, taskNames)
				needsUpdate = false
			}
		}
	}
}

// printStatus formate et affiche la ligne de statut actuelle.
func (r *Runner) printStatus(status map[string]float64, keys []string) {
	// Carriage return (\r) pour écraser la ligne actuelle.
	fmt.Print("\r")
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%-25s %6.2f%%", k, status[k]))
	}
	output := strings.Join(parts, " | ")

	// Padding pour effacer la ligne précédente.
	fmt.Printf("%-180s", output)
}
