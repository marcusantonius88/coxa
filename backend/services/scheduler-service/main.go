package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"coxa/shared/database"
	"coxa/shared/events"
	"coxa/shared/infra"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var (
	db            *sql.DB
	kafkaProducer *infra.KafkaProducer
	kafkaConsumer *infra.KafkaConsumer
	logger        *infra.Logger
	metrics       *infra.ServiceMetrics
	redisClient   *infra.RedisClient
)

func init() {
	var err error

	// Conectar ao banco de dados
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "user=coxa password=coxa dbname=coxa host=postgres port=5432 sslmode=disable"
	}

	db, err = database.ConnectDB(dsn)
	if err != nil {
		panic(fmt.Sprintf("erro ao conectar ao banco: %v", err))
	}

	// Inicializar Kafka
	brokers := []string{"kafka:9092"}
	kafkaProducer = infra.NewKafkaProducer(brokers)
	kafkaConsumer = infra.NewKafkaConsumer(brokers, "medication-events", "scheduler-service")

	// Inicializar logger
	logger = infra.NewLogger("scheduler-service")

	// Inicializar métricas
	metrics = infra.NewServiceMetrics("scheduler-service")

	// Inicializar Redis
	redisClient = infra.NewRedisClient("redis:6379")
}

// consumeEvents consome eventos do Kafka
func consumeEvents(ctx context.Context) {
	logger.LogProcessed("", "", "iniciando consumo de eventos do Kafka...")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := kafkaConsumer.ReadMessage(ctx)
			if err != nil {
				logger.LogFailed("", "", "erro ao ler mensagem", err.Error())
				continue
			}

			logger.LogProcessed("", "", fmt.Sprintf("mensagem recebida: %d bytes", len(msg)))
			processMessage(ctx, msg)
		}
	}
}

// processMessage processa uma mensagem do Kafka
func processMessage(ctx context.Context, msgData []byte) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.EventProcessingDuration.Observe(duration)
	}()

	// Parse como mapa genérico primeiro para inspecionar campos
	var msgMap map[string]interface{}
	if err := json.Unmarshal(msgData, &msgMap); err != nil {
		logger.LogFailed("", "", "erro ao desserializar JSON", err.Error())
		metrics.EventsFailedTotal.Inc()
		return
	}

	var event events.Event

	// Verificar se é Kafka message simples (com "id" e "aggregate_id" no top level)
	if id, hasID := msgMap["id"]; hasID {
		// É o formato esperado do Kafka (plain JSON após ExtractNewRecordState transform)
		event.EventID = fmt.Sprintf("%s-%v", msgMap["aggregate_id"], id)
		if agg, ok := msgMap["aggregate_id"].(string); ok {
			event.AggregateID = agg
		}
		if etype, ok := msgMap["event_type"].(string); ok {
			event.EventType = etype
		}
		if cid, ok := msgMap["correlation_id"].(string); ok {
			event.CorrelationID = cid
		}
		if payload, ok := msgMap["payload"].(string); ok {
			event.Payload = []byte(payload)
		}
		if createdAt, ok := msgMap["created_at"].(float64); ok {
			// Converter microsegundos para nanosegundos, depois para time.Time
			event.CreatedAt = time.Unix(0, int64(createdAt)*1000)
		}
	} else if payload, hasPayload := msgMap["payload"].(map[string]interface{}); hasPayload {
		// É o formato Debezium envelope com "payload" como objeto
		if after, hasAfter := payload["after"].(map[string]interface{}); hasAfter {
			var aggregateID, eventType, payloadStr, correlationID string
			var idVal interface{}

			if val, ok := after["aggregate_id"].(string); ok {
				aggregateID = val
			}
			if val, ok := after["event_type"].(string); ok {
				eventType = val
			}
			if val, ok := after["payload"].(string); ok {
				payloadStr = val
			}
			if val, ok := after["correlation_id"].(string); ok {
				correlationID = val
			}
			idVal = after["id"]

			if aggregateID != "" && eventType != "" && payloadStr != "" && correlationID != "" {
				var id int
				switch v := idVal.(type) {
				case float64:
					id = int(v)
				case int:
					id = v
				}
				event.EventID = fmt.Sprintf("%s-%d", aggregateID, id)
				event.AggregateID = aggregateID
				event.EventType = eventType
				event.CorrelationID = correlationID
				event.Payload = []byte(payloadStr)
			}
		}
	}

	if event.EventID == "" || event.EventType == "" {
		logger.LogFailed("", "", "erro ao desserializar evento", fmt.Sprintf("EventID: %s, EventType: %s, raw: %s", event.EventID, event.EventType, string(msgData[:min(len(msgData), 100)])))
		metrics.EventsFailedTotal.Inc()
		return
	}

	// Verificar idempotência
	isProcessed, err := redisClient.IsProcessed(ctx, event.EventID)
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao verificar idempotência", err.Error())
		return
	}

	if isProcessed {
		logger.LogProcessed(event.EventID, event.CorrelationID, "evento já foi processado (idempotência)")
		return
	}

	// Processar diferentes tipos de eventos
	switch event.EventType {
	case "MedicationCreated":
		handleMedicationCreated(ctx, event)
	case "MedicationGiven":
		handleMedicationGiven(ctx, event)
	default:
		logger.LogFailed(event.EventID, event.CorrelationID, "tipo de evento desconhecido", event.EventType)
	}

	// Marcar como processado
	redisClient.SetProcessed(ctx, event.EventID, 24*time.Hour)
	metrics.EventsProcessedTotal.Inc()
}

// handleMedicationCreated agenda um novo medicamento
func handleMedicationCreated(ctx context.Context, event events.Event) {
	var payload events.MedicationCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao desserializar payload", err.Error())
		return
	}

	scheduleID := uuid.New().String()
	now := time.Now()
	nextDueDate := now.AddDate(0, 0, payload.FrequencyDays)

	// Salvar agendamento no banco
	query := `
		INSERT INTO medication_schedules (id, medication_id, pet_id, owner_id, scheduled_date, next_due_date, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.ExecContext(ctx, query, scheduleID, event.AggregateID, payload.PetID, payload.OwnerID, now, nextDueDate, "pending")
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao criar agendamento", err.Error())
		return
	}

	// Publicar evento MedicationScheduled
	eventPayload := events.MedicationScheduledPayload{
		MedicationID:  event.AggregateID,
		ScheduledDate: now,
		NextDueDate:   nextDueDate,
	}

	payloadJSON, err := json.Marshal(eventPayload)
	if err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao serializar payload", err.Error())
		return
	}

	// Criar evento completo para publicar
	eventID := uuid.New().String()
	eventMsg := map[string]interface{}{
		"id":             1,          // ID simples para o evento
		"aggregate_id":   scheduleID, // ID do agendamento
		"event_type":     "MedicationScheduled",
		"payload":        string(payloadJSON),
		"correlation_id": event.CorrelationID,
		"created_at":     time.Now().UnixMicro(),
		"published_at":   nil,
	}

	err = kafkaProducer.PublishEvent(ctx, "medication-events", eventID, eventMsg)
	if err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao publicar MedicationScheduled", err.Error())
		return
	}

	logger.LogProcessed(event.EventID, event.CorrelationID, "MedicationCreated processado com sucesso")
}

// handleMedicationGiven agenda próxima dose
func handleMedicationGiven(ctx context.Context, event events.Event) {
	var payload events.MedicationGivenPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao desserializar payload", err.Error())
		return
	}

	// Atualizar agendamento com próxima data
	query := `
		UPDATE medication_schedules
		SET next_due_date = $1, status = $2, updated_at = $3
		WHERE medication_id = $4 AND status = 'due'
	`

	_, err := db.ExecContext(ctx, query, payload.NextDueDate, "pending", time.Now(), event.AggregateID)
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao atualizar agendamento", err.Error())
		return
	}

	logger.LogProcessed(event.EventID, event.CorrelationID, "MedicationGiven processado com sucesso")
}

// handleCheckDueSchedules verifica agendamentos vencidos periodicamente
func handleCheckDueSchedules(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkAndPublishDueSchedules(ctx)
		}
	}
}

// checkAndPublishDueSchedules verifica e publica medicamentos vencidos
func checkAndPublishDueSchedules(ctx context.Context) {
	query := `
		SELECT id, medication_id, pet_id, owner_id, next_due_date
		FROM medication_schedules
		WHERE status = 'pending' AND next_due_date <= $1
		LIMIT 100
	`

	rows, err := db.QueryContext(ctx, query, time.Now())
	if err != nil {
		logger.LogFailed("", "", "erro ao buscar agendamentos vencidos", err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var scheduleID, medicationID, petID, ownerID string
		var nextDueDate time.Time

		if err = rows.Scan(&scheduleID, &medicationID, &petID, &ownerID, &nextDueDate); err != nil {
			continue
		}

		eventID := uuid.New().String()
		payload := events.MedicationDuePayload{
			MedicationID: medicationID,
			DueDate:      nextDueDate,
			PetID:        petID,
			OwnerID:      ownerID,
		}

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			logger.LogFailed(eventID, "", "erro ao serializar payload MedicationDue", err.Error())
			continue
		}

		// Criar evento completo para publicar
		eventMsg := map[string]interface{}{
			"id":             1,
			"aggregate_id":   medicationID,
			"event_type":     "MedicationDue",
			"payload":        string(payloadJSON),
			"correlation_id": eventID,
			"created_at":     time.Now().UnixMicro(),
			"published_at":   nil,
		}

		if err = kafkaProducer.PublishEvent(ctx, "medication-events", eventID, eventMsg); err != nil {
			logger.LogFailed(eventID, "", "erro ao publicar MedicationDue", err.Error())
			continue
		}

		// Atualizar status para 'due'
		updateQuery := `UPDATE medication_schedules SET status = 'due' WHERE id = $1`
		db.ExecContext(ctx, updateQuery, scheduleID)

		logger.LogProcessed(eventID, "", "MedicationDue publicado")
	}
}

func main() {
	// Iniciar consumidor de eventos
	ctx := context.Background()
	go consumeEvents(ctx)

	// Iniciar verificação periódica de agendamentos vencidos
	go handleCheckDueSchedules(ctx)

	// Rotas HTTP
	http.HandleFunc("/schedules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetSchedules(w, r)
		} else {
			http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
		}
	})

	// Endpoint de métricas
	http.Handle("/metrics", promhttp.Handler())

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	logger.LogProcessed("", "", "scheduler-service iniciado na porta 8002")
	if err := http.ListenAndServe(":8002", nil); err != nil {
		panic(err)
	}
}

// handleGetSchedules retorna todos os agendamentos
func handleGetSchedules(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, medication_id, pet_id, owner_id, next_due_date, status
		FROM medication_schedules
		ORDER BY next_due_date ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "erro ao buscar agendamentos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "[")

	first := true
	for rows.Next() {
		var id, medicationID, petID, ownerID, status string
		var nextDueDate time.Time

		if err = rows.Scan(&id, &medicationID, &petID, &ownerID, &nextDueDate, &status); err != nil {
			continue
		}

		if !first {
			fmt.Fprint(w, ",")
		}
		first = false

		fmt.Fprintf(w, `{"id":"%s","medication_id":"%s","pet_id":"%s","owner_id":"%s","next_due_date":"%s","status":"%s"}`,
			id, medicationID, petID, ownerID, nextDueDate.Format(time.RFC3339), status)
	}

	fmt.Fprint(w, "]")
}
