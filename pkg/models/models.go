/*
Package main defines the shared data structures for system observability.

These models are used for:
- **Structured Logging**: Standardized format for system health and audit trails.
- **Real-time Monitoring**: Decoupled metrics processing for the TUI.
*/

package models

import "encoding/json"

// LogLevel defines the severity levels for structured logging.
type LogLevel string

const (
	LogLevelINFO  LogLevel = "INFO"
	LogLevelERROR LogLevel = "ERROR"
)

// LogEntry defines the structure of a system health log record.
// It implements the "Application Health Monitoring" pattern, following
// a structured JSON format optimized for ingestion and visualization
// by modern monitoring and alerting stacks.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`          // Horodatage du log au format RFC3339.
	Level     LogLevel               `json:"level"`              // Niveau de sévérité (INFO, ERROR).
	Message   string                 `json:"message"`            // Message principal du log.
	Service   string                 `json:"service"`            // Nom du service émetteur.
	Error     string                 `json:"error,omitempty"`    // Message d'erreur, si applicable.
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // Données contextuelles supplémentaires.
}

// EventEntry defines the record structure for the business event journal.
// It implements the "Audit Trail" pattern by capturing an immutable,
// high-fidelity copy of every Kafka message received, along with its metadata.
// This journal serves as the source of truth for compliance and debugging.
type EventEntry struct {
	Timestamp      string          `json:"timestamp"`            // Horodatage de la réception au format RFC3339.
	EventType      string          `json:"event_type"`           // Type d'événement (ex: "message.received").
	KafkaTopic     string          `json:"kafka_topic"`          // Topic Kafka d'origine.
	KafkaPartition int32           `json:"kafka_partition"`      // Partition Kafka d'origine.
	KafkaOffset    int64           `json:"kafka_offset"`         // Offset du message dans la partition.
	RawMessage     string          `json:"raw_message"`          // Contenu brut du message.
	MessageSize    int             `json:"message_size"`         // Taille du message en octets.
	Deserialized   bool            `json:"deserialized"`         // Indique si la désérialisation a réussi.
	Error          string          `json:"error,omitempty"`      // Erreur de désérialisation, si applicable.
	OrderFull      json.RawMessage `json:"order_full,omitempty"` // Contenu complet de la commande désérialisée.
}
