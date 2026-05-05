package main

import (
	"database/sql"
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

var (
	db            *sql.DB
	kafkaProducer *infra.KafkaProducer
	logger        *infra.Logger
	metrics       *infra.ServiceMetrics
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

	// Inicializar Kafka Producer
	brokers := []string{"kafka:9092"}
	kafkaProducer = infra.NewKafkaProducer(brokers)

	// Inicializar logger
	logger = infra.NewLogger("habit-service")

	// Inicializar métricas
	metrics = infra.NewServiceMetrics("habit-service")
}

// Medication é a entidade de domínio
type Medication struct {
	ID            string
	PetID         string
	OwnerID       string
	Name          string
	Description   string
	FrequencyDays int
	CreatedAt     time.Time
}

// CreateMedicationRequest é o DTO de entrada
type CreateMedicationRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	FrequencyDays int    `json:"frequency_days"`
	PetID         string `json:"pet_id"`
	OwnerID       string `json:"owner_id"`
}

// handleCreateMedication cria um novo medicamento
func handleCreateMedication(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.EventProcessingDuration.Observe(duration)
	}()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "erro ao parsear formulário", http.StatusBadRequest)
		return
	}

	// Parse do formulário
	name := r.FormValue("name")
	description := r.FormValue("description")
	frequencyDays := 60 // padrão
	petID := r.FormValue("pet_id")
	ownerID := r.FormValue("owner_id")

	if name == "" || petID == "" || ownerID == "" {
		http.Error(w, "campos obrigatórios ausentes", http.StatusBadRequest)
		return
	}

	medicationID := uuid.New().String()
	correlationID := uuid.New().String()

	// Iniciar transação
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(medicationID, correlationID, "erro ao iniciar transação", err.Error())
		http.Error(w, "erro ao criar medicamento", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Salvar medicamento no banco
	query := `
		INSERT INTO medications (id, pet_id, owner_id, name, description, frequency_days, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.Exec(query, medicationID, petID, ownerID, name, description, frequencyDays, time.Now())
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(medicationID, correlationID, "erro ao salvar medicamento", err.Error())
		http.Error(w, "erro ao criar medicamento", http.StatusInternalServerError)
		return
	}

	// Salvar evento na outbox
	payload := events.MedicationCreatedPayload{
		Name:          name,
		FrequencyDays: frequencyDays,
		Description:   description,
		PetID:         petID,
		OwnerID:       ownerID,
	}

	err = database.SaveOutboxEvent(tx, medicationID, "MedicationCreated", correlationID, payload)
	if err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(medicationID, correlationID, "erro ao salvar evento na outbox", err.Error())
		http.Error(w, "erro ao criar medicamento", http.StatusInternalServerError)
		return
	}

	// Commit transação
	if err = tx.Commit(); err != nil {
		metrics.EventsFailedTotal.Inc()
		logger.LogFailed(medicationID, correlationID, "erro ao fazer commit", err.Error())
		http.Error(w, "erro ao criar medicamento", http.StatusInternalServerError)
		return
	}

	metrics.EventsProcessedTotal.Inc()
	logger.LogProcessed(medicationID, correlationID, "MedicationCreated processado com sucesso")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": "%s", "message": "Medicamento criado com sucesso"}`, medicationID)
}

// handleGetMedications retorna todos os medicamentos
func handleGetMedications(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, pet_id, owner_id, name, description, frequency_days, created_at
		FROM medications
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "erro ao buscar medicamentos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "[")

	first := true
	for rows.Next() {
		var id, petID, ownerID, name, description string
		var frequencyDays int
		var createdAt time.Time

		if err = rows.Scan(&id, &petID, &ownerID, &name, &description, &frequencyDays, &createdAt); err != nil {
			continue
		}

		if !first {
			fmt.Fprint(w, ",")
		}
		first = false

		fmt.Fprintf(w, `{"id":"%s","pet_id":"%s","owner_id":"%s","name":"%s","description":"%s","frequency_days":%d,"created_at":"%s"}`,
			id, petID, ownerID, name, description, frequencyDays, createdAt.Format(time.RFC3339))
	}

	fmt.Fprint(w, "]")
}

func main() {
	// Rotas HTTP
	http.HandleFunc("/medications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleCreateMedication(w, r)
		} else if r.Method == http.MethodGet {
			handleGetMedications(w, r)
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

	logger.LogProcessed("", "", "habit-service iniciado na porta 8001")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		panic(err)
	}
}
