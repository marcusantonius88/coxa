package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// OutboxEvent representa um evento na tabela outbox
type OutboxEvent struct {
	ID            int64           `db:"id"`
	AggregateID   string          `db:"aggregate_id"`
	EventType     string          `db:"event_type"`
	Payload       json.RawMessage `db:"payload"`
	CreatedAt     time.Time       `db:"created_at"`
	PublishedAt   *time.Time      `db:"published_at"`
	CorrelationID string          `db:"correlation_id"`
}

// SaveOutboxEvent salva um evento na tabela outbox (dentro de uma transação)
func SaveOutboxEvent(tx *sql.Tx, aggregateID, eventType, correlationID string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload: %w", err)
	}

	query := `
		INSERT INTO outbox (aggregate_id, event_type, payload, correlation_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = tx.Exec(query, aggregateID, eventType, payloadJSON, correlationID, time.Now())
	if err != nil {
		return fmt.Errorf("erro ao salvar evento na outbox: %w", err)
	}

	return nil
}

// GetUnpublishedEvents retorna eventos não publicados da outbox
func GetUnpublishedEvents(db *sql.DB, limit int) ([]OutboxEvent, error) {
	query := `
		SELECT id, aggregate_id, event_type, payload, created_at, correlation_id
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar eventos não publicados: %w", err)
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		err = rows.Scan(&event.ID, &event.AggregateID, &event.EventType, &event.Payload, &event.CreatedAt, &event.CorrelationID)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler evento: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// MarkEventAsPublished marca um evento como publicado
func MarkEventAsPublished(db *sql.DB, eventID int64) error {
	query := `
		UPDATE outbox
		SET published_at = $1
		WHERE id = $2
	`

	_, err := db.Exec(query, time.Now(), eventID)
	if err != nil {
		return fmt.Errorf("erro ao marcar evento como publicado: %w", err)
	}

	return nil
}
