// Package config gère la configuration de l'application et l'initialisation du logger.
package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/user/fibbench/internal/fibonacci"
)

// Config encapsule tous les paramètres configurables de l'application.
type Config struct {
	N             int
	Timeout       time.Duration
	Verbose       bool
	AlgosInput    string
	LogLevel      slog.Level
	MetricsPort   int
	SelectedAlgos []fibonacci.AlgorithmKey
}

// Load analyse les drapeaux de ligne de commande, valide la configuration et configure le logger.
func Load() (*Config, error) {
	cfg := &Config{}
	var logLevelStr string

	// Définition des drapeaux
	flag.IntVar(&cfg.N, "n", 2500000, "Index N du terme de Fibonacci.")
	flag.DurationVar(&cfg.Timeout, "timeout", 2*time.Minute, "Temps d'exécution maximum global (ex: '1m', '30s').")
	flag.BoolVar(&cfg.Verbose, "v", false, "Sortie verbeuse : affiche le nombre complet et active le niveau DEBUG.")
	flag.StringVar(&cfg.AlgosInput, "algos", "all", "Liste des algorithmes à exécuter (clés séparées par des virgules, ou 'all').")
	flag.StringVar(&logLevelStr, "loglevel", "info", "Niveau de journalisation (debug, info, warn, error).")
	flag.IntVar(&cfg.MetricsPort, "metrics-port", 8080, "Port TCP pour exposer les métriques Prometheus (0 pour désactiver).")

	// Personnalisation de l'aide
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage de %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nAlgorithmes disponibles (-algos):\n")
		available := fibonacci.ListAlgorithms()
		for _, k := range available {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", k.Key, k.Name)
		}
	}

	flag.Parse()

	// Validation et traitement
	if err := cfg.validateAndProcess(logLevelStr); err != nil {
		return nil, err
	}

	// Configuration du logger global (slog)
	cfg.setupLogger()

	return cfg, nil
}

func (c *Config) validateAndProcess(logLevelStr string) error {
	// 1. Validation de N
	if c.N < 0 {
		return fmt.Errorf("l'index N doit être >= 0. Reçu : %d", c.N)
	}

	// 2. Traitement du niveau de log
	if c.Verbose {
		c.LogLevel = slog.LevelDebug
	} else {
		if err := c.LogLevel.UnmarshalText([]byte(strings.ToUpper(logLevelStr))); err != nil {
			return fmt.Errorf("niveau de log invalide: %s", logLevelStr)
		}
	}

	// 3. Traitement des algorithmes
	if err := c.processAlgos(); err != nil {
		return err
	}

	return nil
}

// processAlgos analyse la chaîne d'entrée des algorithmes et peuple SelectedAlgos.
func (c *Config) processAlgos() error {
	input := strings.ToLower(strings.TrimSpace(c.AlgosInput))

	if input == "all" || input == "" {
		available := fibonacci.ListAlgorithms()
		for _, algo := range available {
			c.SelectedAlgos = append(c.SelectedAlgos, algo.Key)
		}
		return nil
	}

	keys := strings.Split(input, ",")
	seen := make(map[fibonacci.AlgorithmKey]bool)
	var selected []fibonacci.AlgorithmKey

	for _, keyStr := range keys {
		key := fibonacci.AlgorithmKey(strings.TrimSpace(keyStr))
		if key == "" {
			continue
		}

		if !fibonacci.IsRegistered(key) {
			// Avertissement sans arrêter l'exécution.
			fmt.Fprintf(os.Stderr, "Avertissement: Algorithme non reconnu, ignoré: %s\n", key)
			continue
		}

		if !seen[key] {
			selected = append(selected, key)
			seen[key] = true
		}
	}

	if len(selected) == 0 {
		return fmt.Errorf("aucun algorithme valide sélectionné")
	}

	// Trier pour un ordre stable.
	sort.Slice(selected, func(i, j int) bool {
		return selected[i] < selected[j]
	})
	c.SelectedAlgos = selected

	return nil
}

func (c *Config) setupLogger() {
	opts := &slog.HandlerOptions{
		Level: c.LogLevel,
	}
	// Utilisation de NewTextHandler pour la CLI. NewJSONHandler serait préférable pour un service de production.
	handler := slog.NewTextHandler(os.Stderr, opts)
	slog.SetDefault(slog.New(handler))
}
