package events

import (
	"encoding/json"
	"time"
)

// Event é o contrato base para todos os eventos do sistema
type Event struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	AggregateID   string          `json:"aggregate_id"`
	CorrelationID string          `json:"correlation_id"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

// MedicationCreated é emitido quando um medicamento é criado
type MedicationCreatedPayload struct {
	Name          string `json:"name"`
	FrequencyDays int    `json:"frequency_days"`
	Description   string `json:"description"`
	PetID         string `json:"pet_id"`
	OwnerID       string `json:"owner_id"`
}

// MedicationScheduled é emitido quando um medicamento é agendado
type MedicationScheduledPayload struct {
	MedicationID  string    `json:"medication_id"`
	ScheduledDate time.Time `json:"scheduled_date"`
	NextDueDate   time.Time `json:"next_due_date"`
}

// MedicationDue é emitido quando um medicamento vence
type MedicationDuePayload struct {
	MedicationID string    `json:"medication_id"`
	DueDate      time.Time `json:"due_date"`
	PetID        string    `json:"pet_id"`
	OwnerID      string    `json:"owner_id"`
}

// MedicationOverdue é emitido quando um medicamento está atrasado
type MedicationOverduePayload struct {
	MedicationID string    `json:"medication_id"`
	OverdueDate  time.Time `json:"overdue_date"`
	DaysOverdue  int       `json:"days_overdue"`
	PetID        string    `json:"pet_id"`
	OwnerID      string    `json:"owner_id"`
}

// MedicationGiven é emitido quando um medicamento é administrado
type MedicationGivenPayload struct {
	MedicationID string    `json:"medication_id"`
	AdminDate    time.Time `json:"admin_date"`
	Notes        string    `json:"notes"`
	NextDueDate  time.Time `json:"next_due_date"`
}

// NotificationSent é emitido quando uma notificação é enviada
type NotificationSentPayload struct {
	MedicationID string `json:"medication_id"`
	OwnerID      string `json:"owner_id"`
	Message      string `json:"message"`
	Channel      string `json:"channel"`
}

// NotificationFailed é emitido quando falha ao enviar notificação
type NotificationFailedPayload struct {
	MedicationID string `json:"medication_id"`
	OwnerID      string `json:"owner_id"`
	Reason       string `json:"reason"`
}
