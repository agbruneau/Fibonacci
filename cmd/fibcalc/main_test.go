//
// MODULE ACADÉMIQUE : TESTS D'INTÉGRATION DE LA RACINE DE COMPOSITION (MAIN)
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier de test illustre les techniques de validation pour la fonction `main` d'une
// application en ligne de commande (CLI) en Go. Le défi principal consiste à tester une
// logique qui, par nature, interagit avec l'état global du système (arguments de la ligne
// de commande, flux d'E/S standards, signaux du système d'exploitation).
//
// CONCEPTS DE CONCEPTION ET DE TEST ILLUSTRÉS :
//  1. TESTABILITÉ PAR CONCEPTION (DESIGN FOR TESTABILITY) : Le code du module `main` est
//     structuré pour être testable. Les fonctions `parseConfig` et `run` ont été extraites
//     et conçues pour être pures en acceptant leurs dépendances (arguments, `io.Writer`, `context`)
//     comme paramètres. Ceci est une application directe du principe d'Inversion de Dépendances
//     et constitue la pierre angulaire qui rend la validation systématique possible.
//  2. VALIDATION DE LA CONFIGURATION : `TestParseConfig` emploie des tests pilotés par les
//     données pour vérifier exhaustivement la logique de parsing et de validation des
//     arguments, couvrant les cas nominaux, les cas d'erreur et les cas limites, de manière
//     totalement isolée de l'environnement d'exécution (`os.Args`).
//  3. VALIDATION DE L'ORCHESTRATEUR (`run`) : `TestRunFunction` est un test d'intégration
//     qui valide la logique d'orchestration principale. Il vérifie que la fonction `run`
//     produit la sortie attendue et retourne les codes de sortie système corrects en fonction
//     de divers scénarios d'entrée.
//  4. SIMULATION DE L'ENVIRONNEMENT (TEST DOUBLES) :
//      - Le flux de sortie standard (`os.Stdout`) est remplacé par un "Test Double" de type
//        `bytes.Buffer`, qui agit comme un "Spy" pour capturer la sortie et permettre des
//        assertions sur son contenu.
//      - Le `context` est utilisé pour simuler des conditions d'exécution exceptionnelles,
//        telles qu'un timeout ou une annulation externe (simulant un `Ctrl+C`), permettant de
//        valider la robustesse et les chemins de code de l'arrêt contrôlé ("graceful shutdown").
//
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

// TestParseConfig valide la fonction de parsing et de validation de la configuration.
func TestParseConfig(t *testing.T) {
	// `bytes.Buffer` est utilisé comme un "sink" silencieux pour les messages d'erreur
	// potentiels, afin de ne pas polluer les journaux de test.
	var errorSink bytes.Buffer

	testCases := []struct {
		name        string
		args        []string
		expectErr   bool
		expectedN   uint64
		expectedAlgo string
	}{
		{"Cas nominal (défauts)", []string{}, false, 100000000, "all"},
		{"Spécification de N", []string{"-n", "50"}, false, 50, "all"},
		{"Spécification de l'algorithme", []string{"-algo", "fast"}, false, 100000000, "fast"},
		{"Spécification de l'algorithme (insensible à la casse)", []string{"-algo", "MATRIX"}, false, 100000000, "matrix"},
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
		// En mode comparaison, on vérifie la présence du tableau récapitulatif et le statut global.
		if !strings.Contains(output, "Synthèse de la Comparaison") || !strings.Contains(output, "Statut Global : Succès") {
			t.Errorf("La sortie du mode comparaison est incorrecte. Sortie:\n%s", output)
		}
	})

	t.Run("Échec dû à un timeout", func(t *testing.T) {
		var buf bytes.Buffer
		// Un N très grand et un timeout très court pour garantir l'échec.
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

		// NOTE PÉDAGOGIQUE : Simulation d'une annulation externe (e.g., Ctrl+C).
		// `context.WithCancel` permet de créer un contexte que l'on peut annuler
		// programmatiquement. L'appel immédiat à `cancel()` simule un signal
		// d'interruption reçu avant même le début du calcul.
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