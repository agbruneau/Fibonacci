// Package i18n provides enhanced internationalization support with a
// MessageCatalog that supports pluralization, formatting, and fallback languages.
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MessageCatalog manages multilingual messages with support for pluralization
// and formatted arguments.
type MessageCatalog struct {
	mu       sync.RWMutex
	messages map[string]map[string]string // lang -> key -> value
	fallback string
	current  string
}

// NewCatalog creates a new message catalog with a specified fallback language.
//
// Parameters:
//   - fallback: The language code to use when a message is not found in the
//     current language (e.g., "en").
//
// Returns:
//   - *MessageCatalog: A new message catalog instance.
func NewCatalog(fallback string) *MessageCatalog {
	return &MessageCatalog{
		messages: make(map[string]map[string]string),
		fallback: fallback,
		current:  fallback,
	}
}

// SetLanguage changes the current language.
//
// Parameters:
//   - lang: The language code to switch to (e.g., "fr", "es").
func (c *MessageCatalog) SetLanguage(lang string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = lang
}

// GetLanguage returns the current language code.
func (c *MessageCatalog) GetLanguage() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

// GetFallbackLanguage returns the fallback language code.
func (c *MessageCatalog) GetFallbackLanguage() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fallback
}

// Register adds messages for a specific language.
//
// Parameters:
//   - lang: The language code (e.g., "fr").
//   - messages: A map of message keys to their translated values.
func (c *MessageCatalog) Register(lang string, messages map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.messages[lang] == nil {
		c.messages[lang] = make(map[string]string)
	}

	for k, v := range messages {
		c.messages[lang][k] = v
	}
}

// Get retrieves a message by key with optional formatting arguments.
// It first looks in the current language, then falls back to the default
// language. If the key is not found in either, the key itself is returned.
//
// Parameters:
//   - key: The message key to look up.
//   - args: Optional formatting arguments (see fmt.Sprintf).
//
// Returns:
//   - string: The message text, formatted with args if provided.
func (c *MessageCatalog) Get(key string, args ...interface{}) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try current language
	if msgs, ok := c.messages[c.current]; ok {
		if msg, ok := msgs[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(msg, args...)
			}
			return msg
		}
	}

	// Fallback to default language
	if msgs, ok := c.messages[c.fallback]; ok {
		if msg, ok := msgs[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(msg, args...)
			}
			return msg
		}
	}

	// Return key as fallback
	return key
}

// GetPlural retrieves a pluralized message based on the count.
// It looks for keys with "_zero", "_one", "_few", or "_other" suffixes
// and selects the appropriate one based on the count.
//
// Parameters:
//   - key: The base message key.
//   - count: The count for pluralization.
//   - args: Optional formatting arguments (count is prepended automatically).
//
// Returns:
//   - string: The pluralized message.
func (c *MessageCatalog) GetPlural(key string, count int, args ...interface{}) string {
	suffix := c.getPluralSuffix(count)
	fullKey := key + suffix

	// Prepend count to args for formatting
	allArgs := append([]interface{}{count}, args...)

	// Try to get the pluralized version
	msg := c.Get(fullKey, allArgs...)
	if msg != fullKey {
		return msg
	}

	// Fall back to base key with "_other" suffix
	otherKey := key + "_other"
	msg = c.Get(otherKey, allArgs...)
	if msg != otherKey {
		return msg
	}

	// Final fallback to base key
	return c.Get(key, allArgs...)
}

// getPluralSuffix returns the appropriate plural suffix for a count.
// This implements basic English/French pluralization rules.
func (c *MessageCatalog) getPluralSuffix(count int) string {
	if count == 0 {
		return "_zero"
	}
	if count == 1 {
		return "_one"
	}
	if count >= 2 && count <= 4 {
		return "_few"
	}
	return "_other"
}

// LoadFromFile loads messages from a JSON file into the catalog.
//
// Parameters:
//   - path: The path to the JSON file.
//   - lang: The language code for these messages.
//
// Returns:
//   - error: An error if the file cannot be read or parsed.
func (c *MessageCatalog) LoadFromFile(path string, lang string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read translation file: %w", err)
	}

	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to parse translation file: %w", err)
	}

	c.Register(lang, messages)
	return nil
}

// LoadFromDirectory loads all JSON translation files from a directory.
// Files should be named "{lang}.json" (e.g., "fr.json", "es.json").
//
// Parameters:
//   - dir: The directory containing translation files.
//
// Returns:
//   - error: An error if the directory cannot be read.
func (c *MessageCatalog) LoadFromDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read translation directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".json" {
			continue
		}

		lang := name[:len(name)-len(ext)]
		path := filepath.Join(dir, name)

		if err := c.LoadFromFile(path, lang); err != nil {
			// Log warning but continue loading other files
			fmt.Fprintf(os.Stderr, "[i18n] Warning: failed to load %s: %v\n", name, err)
		}
	}

	return nil
}

// HasLanguage checks if a language is loaded in the catalog.
//
// Parameters:
//   - lang: The language code to check.
//
// Returns:
//   - bool: True if the language has any messages loaded.
func (c *MessageCatalog) HasLanguage(lang string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.messages[lang]
	return ok
}

// AvailableLanguages returns a list of all loaded language codes.
func (c *MessageCatalog) AvailableLanguages() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	langs := make([]string, 0, len(c.messages))
	for lang := range c.messages {
		langs = append(langs, lang)
	}
	return langs
}

// GlobalCatalog is the default global message catalog.
var GlobalCatalog = NewCatalog("en")

func init() {
	// Register default English messages
	GlobalCatalog.Register("en", defaultMessages)

	// Register French messages
	GlobalCatalog.Register("fr", frenchMessages)

	// Register Spanish messages
	GlobalCatalog.Register("es", spanishMessages)

	// Register German messages
	GlobalCatalog.Register("de", germanMessages)
}

// frenchMessages contains French translations.
var frenchMessages = map[string]string{
	"CalibrationTitle":       "--- Mode Calibration : Recherche du seuil de parallélisme optimal ---",
	"CalibrationSummary":     "--- Résumé de la calibration ---",
	"OptimalRecommendation":  "✅ Recommandation pour cette machine : --threshold %d",
	"ExecConfigTitle":        "--- Configuration d'exécution ---",
	"ExecStartTitle":         "--- Démarrage de l'exécution ---",
	"ComparisonSummary":      "--- Résumé de la comparaison ---",
	"GlobalStatusSuccess":    "Statut global : Succès. Tous les résultats valides sont cohérents.",
	"GlobalStatusFailure":    "Statut global : Échec. Aucun algorithme n'a pu terminer le calcul.",
	"StatusCriticalMismatch": "Statut global : ERREUR CRITIQUE ! Une incohérence a été détectée entre les résultats des algorithmes.",
	"StatusCanceled":         "Statut : Annulé",
	"StatusTimeout":          "Statut : Échec (Timeout). La limite d'exécution de %s a été atteinte%s.",
	"StatusFailure":          "Statut : Échec. Une erreur inattendue s'est produite : %v",

	// New CLI messages
	"InteractiveWelcome":    "Bienvenue dans le mode interactif !",
	"InteractiveHelp":       "Tapez 'help' pour voir les commandes disponibles.",
	"InteractiveGoodbye":    "Au revoir !",
	"QuietModeResult":       "F(%d) calculé en %s",
	"OutputSaved":           "Résultat sauvegardé dans : %s",
	"OutputError":           "Erreur lors de l'écriture du fichier : %v",

	// Pluralization examples
	"AlgorithmCount_zero":   "Aucun algorithme disponible",
	"AlgorithmCount_one":    "%d algorithme disponible",
	"AlgorithmCount_other":  "%d algorithmes disponibles",

	"DigitCount_zero":       "Aucun chiffre",
	"DigitCount_one":        "%d chiffre",
	"DigitCount_other":      "%d chiffres",

	"CalculationTime_one":   "Calcul terminé en %d seconde",
	"CalculationTime_other": "Calcul terminé en %d secondes",
}

// spanishMessages contains Spanish translations.
var spanishMessages = map[string]string{
	"CalibrationTitle":       "--- Modo Calibración: Buscando el umbral de paralelismo óptimo ---",
	"CalibrationSummary":     "--- Resumen de la calibración ---",
	"OptimalRecommendation":  "✅ Recomendación para esta máquina: --threshold %d",
	"ExecConfigTitle":        "--- Configuración de ejecución ---",
	"ExecStartTitle":         "--- Iniciando ejecución ---",
	"ComparisonSummary":      "--- Resumen de la comparación ---",
	"GlobalStatusSuccess":    "Estado global: Éxito. Todos los resultados válidos son consistentes.",
	"GlobalStatusFailure":    "Estado global: Fallo. Ningún algoritmo pudo completar el cálculo.",
	"StatusCriticalMismatch": "Estado global: ¡ERROR CRÍTICO! Se detectó una inconsistencia entre los resultados de los algoritmos.",
	"StatusCanceled":         "Estado: Cancelado",
	"StatusTimeout":          "Estado: Fallo (Timeout). Se alcanzó el límite de ejecución de %s%s.",
	"StatusFailure":          "Estado: Fallo. Ocurrió un error inesperado: %v",

	// New CLI messages
	"InteractiveWelcome":    "¡Bienvenido al modo interactivo!",
	"InteractiveHelp":       "Escriba 'help' para ver los comandos disponibles.",
	"InteractiveGoodbye":    "¡Hasta luego!",
	"QuietModeResult":       "F(%d) calculado en %s",
	"OutputSaved":           "Resultado guardado en: %s",
	"OutputError":           "Error al escribir el archivo: %v",

	// Pluralization examples
	"AlgorithmCount_zero":   "Ningún algoritmo disponible",
	"AlgorithmCount_one":    "%d algoritmo disponible",
	"AlgorithmCount_other":  "%d algoritmos disponibles",
}

// germanMessages contains German translations.
var germanMessages = map[string]string{
	"CalibrationTitle":       "--- Kalibrierungsmodus: Suche nach optimalem Parallelitätsschwellenwert ---",
	"CalibrationSummary":     "--- Kalibrierungszusammenfassung ---",
	"OptimalRecommendation":  "✅ Empfehlung für diesen Computer: --threshold %d",
	"ExecConfigTitle":        "--- Ausführungskonfiguration ---",
	"ExecStartTitle":         "--- Ausführung wird gestartet ---",
	"ComparisonSummary":      "--- Vergleichszusammenfassung ---",
	"GlobalStatusSuccess":    "Globaler Status: Erfolg. Alle gültigen Ergebnisse sind konsistent.",
	"GlobalStatusFailure":    "Globaler Status: Fehler. Kein Algorithmus konnte die Berechnung abschließen.",
	"StatusCriticalMismatch": "Globaler Status: KRITISCHER FEHLER! Eine Inkonsistenz wurde zwischen den Algorithmusergebnissen festgestellt.",
	"StatusCanceled":         "Status: Abgebrochen",
	"StatusTimeout":          "Status: Fehler (Zeitüberschreitung). Das Ausführungslimit von %s wurde erreicht%s.",
	"StatusFailure":          "Status: Fehler. Ein unerwarteter Fehler ist aufgetreten: %v",

	// New CLI messages
	"InteractiveWelcome":    "Willkommen im interaktiven Modus!",
	"InteractiveHelp":       "Geben Sie 'help' ein, um die verfügbaren Befehle anzuzeigen.",
	"InteractiveGoodbye":    "Auf Wiedersehen!",
	"QuietModeResult":       "F(%d) berechnet in %s",
	"OutputSaved":           "Ergebnis gespeichert in: %s",
	"OutputError":           "Fehler beim Schreiben der Datei: %v",

	// Pluralization examples
	"AlgorithmCount_zero":   "Keine Algorithmen verfügbar",
	"AlgorithmCount_one":    "%d Algorithmus verfügbar",
	"AlgorithmCount_other":  "%d Algorithmen verfügbar",
}

