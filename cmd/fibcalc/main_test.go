package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

// TestParseConfig valide la fonction d'analyse de la configuration.
func TestParseConfig(t *testing.T) {
	var errorSink bytes.Buffer

	testCases := []struct {
		name         string
		args         []string
		expectErr    bool
		expectedN    uint64
		expectedAlgo string
	}{
		{"Cas nominal (défauts)", []string{}, false, 250000000, "all"},
		{"Spécification de N", []string{"-n", "50"}, false, 50, "all"},
		{"Spécification de l'algorithme", []string{"-algo", "fast"}, false, 250000000, "fast"},
		{"Spécification de l'algorithme (insensible à la casse)", []string{"-algo", "MATRIX"}, false, 250000000, "matrix"},
		{"Cas d'erreur : seuil négatif", []string{"-threshold", "-100"}, true, 0, ""},
		{"Cas d'erreur : argument inconnu", []string{"-invalid-flag"}, true, 0, ""},
		{"Cas d'erreur : algorithme inconnu", []string{"-algo", "nonexistent"}, true, 0, ""},
		{"Cas d'erreur : timeout invalide", []string{"-timeout", "-5s"}, true, 0, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := parseConfig("test", tc.args, &errorSink)

			if tc.expectErr {
				if err == nil {
					t.Error("Une erreur était attendue, mais aucune n'a été retournée.")
				}
			} else {
				if err != nil {
					t.Errorf("Une erreur inattendue a été retournée : %v", err)
				}
				if config.N != tc.expectedN {
					t.Errorf("Champ N de la config incorrect. Attendu: %d, Obtenu: %d", tc.expectedN, config.N)
				}
				if config.Algo != tc.expectedAlgo {
					t.Errorf("Champ Algo de la config incorrect. Attendu: %q, Obtenu: %q", tc.expectedAlgo, config.Algo)
				}
			}
		})
	}
}

// TestRunFunction valide le comportement de la fonction d'orchestration principale `run`.
func TestRunFunction(t *testing.T) {

	t.Run("Exécution simple avec succès", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 10, Algo: "fast", Timeout: 1 * time.Minute, Threshold: fibonacci.DefaultParallelThreshold, FFTThreshold: 20000, Details: true}
		exitCode := run(context.Background(), config, &buf)

		if exitCode != ExitSuccess {
			t.Errorf("Code de sortie incorrect. Attendu: %d, Obtenu: %d", ExitSuccess, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "F(10) = 55") {
			t.Errorf("La sortie détaillée ne contient pas le résultat attendu 'F(10) = 55'. Sortie:\n%s", output)
		}
	})

	t.Run("Comparaison parallèle avec succès", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 20, Algo: "all", Timeout: 1 * time.Minute, Threshold: fibonacci.DefaultParallelThreshold, FFTThreshold: 20000, Details: false}
		exitCode := run(context.Background(), config, &buf)

		if exitCode != ExitSuccess {
			t.Errorf("Code de sortie incorrect. Attendu: %d, Obtenu: %d", ExitSuccess, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "Synthèse de la Comparaison") || !strings.Contains(output, "Statut Global : Succès") {
			t.Errorf("La sortie du mode comparaison est incorrecte. Sortie:\n%s", output)
		}
	})

	t.Run("Échec dû à un timeout", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 100_000_000, Algo: "fast", Timeout: 1 * time.Millisecond}
		exitCode := run(context.Background(), config, &buf)

		if exitCode != ExitErrorTimeout {
			t.Errorf("Code de sortie incorrect pour un timeout. Attendu: %d, Obtenu: %d", ExitErrorTimeout, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "Échec (Timeout)") {
			t.Errorf("La sortie devrait explicitement mentionner l'échec par timeout. Sortie:\n%s", output)
		}
	})

	t.Run("Échec dû à une annulation par le contexte", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 100_000_000, Algo: "fast", Timeout: 1 * time.Minute}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		exitCode := run(ctx, config, &buf)

		if exitCode != ExitErrorCanceled {
			t.Errorf("Code de sortie incorrect pour une annulation. Attendu: %d, Obtenu: %d", ExitErrorCanceled, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "Statut : Annulé") {
			t.Errorf("La sortie devrait explicitement mentionner l'annulation. Sortie:\n%s", output)
		}
	})
}