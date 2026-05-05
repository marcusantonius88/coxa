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
	kafkaConsumer = infra.NewKafkaConsumer(brokers, "medication-events", "analytics-service")

	// Inicializar logger
	logger = infra.NewLogger("analytics-service")

	// Inicializar métricas
	metrics = infra.NewServiceMetrics("analytics-service")

	// Inicializar Redis
	redisClient = infra.NewRedisClient("redis:6379")
}

// consumeEvents consome eventos do Kafka para análise
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

// processMessage processa uma mensagem para análise
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

	// Armazenar toda administração para análise
	switch event.EventType {
	case "MedicationGiven":
		handleMedicationGiven(ctx, event)
	case "MedicationDue":
		handleMedicationDue(ctx, event)
	default:
		logger.LogFailed(event.EventID, event.CorrelationID, "tipo de evento desconhecido", event.EventType)
	}

	// Marcar como processado
	redisClient.SetProcessed(ctx, event.EventID, 24*time.Hour)
	metrics.EventsProcessedTotal.Inc()
}

// handleMedicationGiven registra administração para histórico
func handleMedicationGiven(ctx context.Context, event events.Event) {
	var payload events.MedicationGivenPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao desserializar payload", err.Error())
		return
	}

	adminID := uuid.New().String()

	// Salvar administração no banco para análise
	query := `
		INSERT INTO medication_administrations (id, medication_id, pet_id, owner_id, admin_date, notes, next_due_date)
		VALUES ($1, $2, (SELECT pet_id FROM medication_schedules WHERE medication_id = $2 LIMIT 1), 
				(SELECT owner_id FROM medication_schedules WHERE medication_id = $2 LIMIT 1), $3, $4, $5)
	`

	_, err := db.ExecContext(ctx, query, adminID, event.AggregateID, payload.AdminDate, payload.Notes, payload.NextDueDate)
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao salvar administração", err.Error())
		return
	}

	logger.LogProcessed(event.EventID, event.CorrelationID, "Administração registrada para análise")
}

// handleMedicationDue registra medicamentos vencidos para análise
func handleMedicationDue(ctx context.Context, event events.Event) {
	var payload events.MedicationDuePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		logger.LogFailed(event.EventID, event.CorrelationID, "erro ao desserializar payload", err.Error())
		return
	}

	logger.LogProcessed(event.EventID, event.CorrelationID, fmt.Sprintf("Medicamento %s vencido para pet %s", payload.MedicationID, payload.PetID))
}

func main() {
	// Iniciar consumidor de eventos
	ctx := context.Background()
	go consumeEvents(ctx)

	// Rotas HTTP
	http.HandleFunc("/timeline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetTimeline(w, r)
		} else {
			http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetStats(w, r)
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

	logger.LogProcessed("", "", "analytics-service iniciado na porta 8004")
	if err := http.ListenAndServe(":8004", nil); err != nil {
		panic(err)
	}
}

// handleGetTimeline retorna timeline de eventos
func handleGetTimeline(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")
	if ownerID == "" {
		http.Error(w, "owner_id é obrigatório", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, medication_id, pet_id, admin_date, notes
		FROM medication_administrations
		WHERE owner_id = $1
		ORDER BY admin_date DESC
		LIMIT 100
	`

	rows, err := db.Query(query, ownerID)
	if err != nil {
		http.Error(w, "erro ao buscar timeline", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "[")

	first := true
	for rows.Next() {
		var id, medicationID, petID, notes string
		var adminDate time.Time

		if err = rows.Scan(&id, &medicationID, &petID, &adminDate, &notes); err != nil {
			continue
		}

		if !first {
			fmt.Fprint(w, ",")
		}
		first = false

		fmt.Fprintf(w, `{"id":"%s","medication_id":"%s","pet_id":"%s","admin_date":"%s","notes":"%s"}`,
			id, medicationID, petID, adminDate.Format(time.RFC3339), notes)
	}

	fmt.Fprint(w, "]")
}

// handleGetStats retorna estatísticas
func handleGetStats(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")
	if ownerID == "" {
		http.Error(w, "owner_id é obrigatório", http.StatusBadRequest)
		return
	}

	// Total de administrações
	var totalAdmins int
	db.QueryRow("SELECT COUNT(*) FROM medication_administrations WHERE owner_id = $1", ownerID).Scan(&totalAdmins)

	// Medicamentos ativos
	var activeMeds int
	db.QueryRow("SELECT COUNT(*) FROM medications WHERE owner_id = $1", ownerID).Scan(&activeMeds)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"total_administrations":%d,"active_medications":%d}`, totalAdmins, activeMeds)
}
