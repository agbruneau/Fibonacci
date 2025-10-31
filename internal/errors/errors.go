// Package apperrors définit des types d'erreurs applicatives structurées
// permettant de distinguer clairement les classes d'erreurs (configuration,
// calcul, etc.) et de transporter la cause sous-jacente.
package apperrors

import "fmt"

// ConfigError représente une erreur de configuration utilisateur (flags, valeurs invalides, etc.).
type ConfigError struct {
    Message string
}

func (e ConfigError) Error() string { return e.Message }

func NewConfigError(format string, a ...interface{}) error {
    return ConfigError{Message: fmt.Sprintf(format, a...)}
}

// CalculationError permet d'encapsuler une erreur de calcul tout en gardant
// la cause d'origine (unwrap).
type CalculationError struct {
    Cause error
}

func (e CalculationError) Error() string { return e.Cause.Error() }
func (e CalculationError) Unwrap() error { return e.Cause }


