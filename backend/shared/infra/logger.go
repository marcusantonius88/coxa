package infra

import (
	"encoding/json"
	"fmt"
	"time"
)

// StructuredLog é um log estruturado em JSON
type StructuredLog struct {
	Timestamp     time.Time `json:"timestamp"`
	Service       string    `json:"service"`
	EventID       string    `json:"event_id"`
	CorrelationID string    `json:"correlation_id"`
	Status        string    `json:"status"` // processed, failed
	Message       string    `json:"message"`
	Error         string    `json:"error,omitempty"`
}

// Logger encapsula logging estruturado
type Logger struct {
	serviceName string
}

// NewLogger cria um novo logger
func NewLogger(serviceName string) *Logger {
	return &Logger{serviceName: serviceName}
}

// LogProcessed loga sucesso no processamento
func (l *Logger) LogProcessed(eventID, correlationID, message string) {
	log := StructuredLog{
		Timestamp:     time.Now(),
		Service:       l.serviceName,
		EventID:       eventID,
		CorrelationID: correlationID,
		Status:        "processed",
		Message:       message,
	}
	jsonLog, _ := json.Marshal(log)
	fmt.Println(string(jsonLog))
}

// LogFailed loga falha no processamento
func (l *Logger) LogFailed(eventID, correlationID, message, errMsg string) {
	log := StructuredLog{
		Timestamp:     time.Now(),
		Service:       l.serviceName,
		EventID:       eventID,
		CorrelationID: correlationID,
		Status:        "failed",
		Message:       message,
		Error:         errMsg,
	}
	jsonLog, _ := json.Marshal(log)
	fmt.Println(string(jsonLog))
}
