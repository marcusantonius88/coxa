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
	notifMetrics  *infra.NotificationMetrics
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
	kafkaConsumer = infra.NewKafkaConsumer(brokers, "medication-events", "notification-service")

	// Inicializar logger
	logger = infra.NewLogger("notification-service")

	// Inicializar métricas
	metrics = infra.NewServiceMetrics("notification-service")
	notifMetrics = infra.NewNotificationMetrics()

	// Inicializar Redis
	redisClient = infra.NewRedisClient("redis:6379")
}

// consumeEvents consome eventos do Kafka
func consumeEvents(ctx context.Context) {
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
		logger.LogFailed("", "", "erro ao desserializar evento", fmt.Sprintf("EventID: %s, EventType: %s", event.EventID, event.EventType))
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

	// Processar eventos de medicamento vencido
	switch event.EventType {
	case "MedicationDue":
		handleMedicationDue(ctx, event)
	case "MedicationOverdue":
		handleMedicationOverdue(ctx, event)
	default:
		logger.LogFailed(event.EventID, event.CorrelationID, "tipo de evento desconhecido", event.EventType)
	}

	// Marcar como processado
	redisClient.SetProcessed(ctx, event.EventID, 24*time.Hour)
	metrics.EventsProcessedTotal.Inc()
}

// handleMedicationDue envia notificação de medicamento vencido
func handleMedicationDue(ctx context.Context, event events.Event) {
	var payload events.MedicationDuePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao desserializar payload", err.Error())
		return
	}

	notifID := uuid.New().String()
	message := fmt.Sprintf("Medicamento vencido para %s. Favor administrar assim que possível.", payload.PetID)

	// Salvar notificação no banco
	query := `
		INSERT INTO notifications (id, medication_id, owner_id, message, channel, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := db.ExecContext(ctx, query, notifID, payload.MedicationID, payload.OwnerID, message, "log", "sent")
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		notifMetrics.NotificationsFailedTotal.Inc()
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao salvar notificação", err.Error())
		return
	}

	// Log (simular envio de notificação)
	fmt.Printf("[NOTIFICATION] To: %s | Message: %s\n", payload.OwnerID, message)

	// Atualizar status para sent
	db.ExecContext(ctx, "UPDATE notifications SET sent_at = $1 WHERE id = $2", time.Now(), notifID)

	notifMetrics.NotificationsSentTotal.Inc()
	logger.LogProcessed(event.EventID, event.CorrelationID, "Notificação de MedicationDue enviada")
}

// handleMedicationOverdue envia notificação de medicamento muito atrasado
func handleMedicationOverdue(ctx context.Context, event events.Event) {
	var payload events.MedicationOverduePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao desserializar payload", err.Error())
		return
	}

	notifID := uuid.New().String()
	message := fmt.Sprintf("⚠️ ALERTA: Medicamento de %s está %d dias atrasado!", payload.PetID, payload.DaysOverdue)

	// Salvar notificação no banco
	query := `
		INSERT INTO notifications (id, medication_id, owner_id, message, channel, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := db.ExecContext(ctx, query, notifID, payload.MedicationID, payload.OwnerID, message, "log", "sent")
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		notifMetrics.NotificationsFailedTotal.Inc()
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao salvar notificação", err.Error())
		return
	}

	// Log (simular envio de notificação)
	fmt.Printf("[NOTIFICATION] To: %s | Message: %s\n", payload.OwnerID, message)

	// Atualizar status para sent
	db.ExecContext(ctx, "UPDATE notifications SET sent_at = $1 WHERE id = $2", time.Now(), notifID)

	notifMetrics.NotificationsSentTotal.Inc()
	logger.LogProcessed(event.EventID, event.CorrelationID, "Notificação de MedicationOverdue enviada")
}

func main() {
	// Iniciar consumidor de eventos
	ctx := context.Background()
	go consumeEvents(ctx)

	// Rotas HTTP
	http.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetNotifications(w, r)
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

	logger.LogProcessed("", "", "notification-service iniciado na porta 8003")
	if err := http.ListenAndServe(":8003", nil); err != nil {
		panic(err)
	}
}

// handleGetNotifications retorna todas as notificações
func handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, medication_id, owner_id, message, status, sent_at
		FROM notifications
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "erro ao buscar notificações", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "[")

	first := true
	for rows.Next() {
		var id, medicationID, ownerID, message, status string
		var sentAt *time.Time

		if err = rows.Scan(&id, &medicationID, &ownerID, &message, &status, &sentAt); err != nil {
			continue
		}

		if !first {
			fmt.Fprint(w, ",")
		}
		first = false

		sentAtStr := ""
		if sentAt != nil {
			sentAtStr = sentAt.Format(time.RFC3339)
		}

		fmt.Fprintf(w, `{"id":"%s","medication_id":"%s","owner_id":"%s","message":"%s","status":"%s","sent_at":"%s"}`,
			id, medicationID, ownerID, message, status, sentAtStr)
	}

	fmt.Fprint(w, "]")
}
