# Estrutura do Projeto COXA

Esta documentação descreve a **estrutura real** do projeto COXA conforme implementado.

---

## 📁 Estrutura de Pastas

```
coxa/
├── backend/
│   ├── services/
│   │   ├── habit-service/
│   │   │   ├── main.go              # Tudo neste arquivo
│   │   │   ├── go.mod
│   │   │   ├── go.sum
│   │   │   └── Dockerfile
│   │   ├── scheduler-service/       # Mesma estrutura
│   │   ├── notification-service/    # Mesma estrutura
│   │   └── analytics-service/       # Mesma estrutura
│   └── shared/
│       ├── go.mod                   # Go module compartilhado
│       ├── go.sum
│       ├── database/
│       │   ├── connection.go        # Funções de conexão
│       │   ├── migrations.sql       # Schema do banco
│       │   └── outbox.go            # Operações do Outbox Pattern
│       ├── events/
│       │   └── event.go             # Tipos de eventos
│       └── infra/
│           ├── kafka_consumer.go    # Consumer Kafka
│           ├── kafka_producer.go    # Producer Kafka
│           ├── logger.go            # Logger customizado
│           ├── metrics.go           # Métricas Prometheus
│           └── redis_client.go      # Cliente Redis
├── frontend/
│   └── web-app/
│       ├── src/
│       │   ├── App.jsx
│       │   ├── main.jsx
│       │   ├── index.css
│       │   └── components/
│       │       ├── Button.jsx
│       │       ├── Card.jsx
│       │       └── Input.jsx
│       ├── index.html
│       ├── package.json
│       ├── vite.config.js
│       ├── tailwind.config.js
│       ├── postcss.config.js
│       └── Dockerfile
├── infra/
│   ├── debezium/        # Configuração Debezium CDC
│   ├── kafka/           # Configuração Kafka
│   ├── postgres/        # Scripts PostgreSQL
│   ├── prometheus/      # prometheus.yml
│   └── redis/           # Configuração Redis
├── docs/
│   ├── assets/          # Imagens (banner, avatar)
│   ├── ARCHITECTURE.md
│   ├── PROJECT_STRUCTURE.md
│   ├── SETUP.md
│   ├── DEVELOPMENT.md
│   ├── CONTRIBUTING.md
│   └── TESTE_END_TO_END.md
├── docker-compose.yml
├── setup.sh
├── setup-debezium.sh
└── readme.md
```

---

## 🔍 Explicação por Pasta

### `/backend/services/`

Cada serviço é um **módulo independente** contendo:

- **`main.go`** - Arquivo único contendo toda a lógica:
  - Funções `init()` - inicialização de dependências (DB, Kafka, Redis, Logger, Metrics)
  - Estruturas de dados (Medication, Request DTOs, Domain models)
  - Handlers HTTP
  - Consumers Kafka
  - Lógica de negócio

- **`go.mod`** - Dependências específicas do serviço
- **`Dockerfile`** - Build multi-stage para produção

**Por que tudo em um main.go?**
- Serviços são simples e focados em um domínio
- Facilita compreensão rápida do código
- Reduz complexidade de imports circulares
- À medida que cresce, pode ser refatorado para `internal/` com subdirs

---

### `/backend/shared/`

Código compartilhado entre todos os serviços:

#### `database/`
- **connection.go** - `ConnectDB()` para estabelecer conexão PostgreSQL
- **outbox.go** - Funções para salvar eventos em `outbox` table (Outbox Pattern)
- **migrations.sql** - Schema completo (executado na inicialização do Postgres)

#### `events/`
- **event.go** - Tipos de eventos e estruturas:
  ```go
  type Event struct {
      ID            int64
      AggregateID   string    // medication_id
      EventType     string    // MedicationCreated, MedicationDue, etc
      Payload       string    // JSON da entidade
      CorrelationID string    // Para rastrear request original
      CreatedAt     int64     // Timestamp
  }
  ```

#### `infra/`
- **kafka_consumer.go** - Wrapper para Kafka consumer (Confluent)
- **kafka_producer.go** - Wrapper para Kafka producer
- **logger.go** - Logger estruturado com níveis (LogProcessed, LogFailed, etc)
- **metrics.go** - Métricas Prometheus:
  - `events_processed_total` (Counter)
  - `event_processing_duration_seconds` (Histogram)
  - `errors_total` (Counter)
- **redis_client.go** - Cliente Redis para idempotência

---

## 🏗️ Padrão de Implementação

Todo serviço segue este padrão:

```go
package main

import (
    // dependências padrão
)

// 1. INICIALIZAÇÃO (init())
var (
    db            *sql.DB
    kafkaProducer *infra.KafkaProducer
    kafkaConsumer *infra.KafkaConsumer
    logger        *infra.Logger
    metrics       *infra.ServiceMetrics
    redisClient   *infra.RedisClient
)

func init() {
    // conectar DB, Kafka, Redis, inicializar logger e metrics
}

// 2. ESTRUTURAS DE DOMÍNIO
type Medication struct { ... }
type CreateMedicationRequest struct { ... }

// 3. HANDLERS HTTP (se produtor)
func handleCreateMedication(w http.ResponseWriter, r *http.Request) { ... }

// 4. CONSUMER KAFKA (se consumidor)
func consumeEvents(ctx context.Context) { ... }
func processMessage(ctx context.Context, msgData []byte) { ... }

// 5. MAIN() - Inicia tudo
func main() {
    http.HandleFunc("/medications", handleCreateMedication)
    http.Handle("/metrics", promhttp.Handler())
    
    go consumeEvents(context.Background())
    
    http.ListenAndServe(":8001", nil)
}
```

---

## 🧩 Serviços Atuais

### Habit Service (Port 8001)
- **Função**: Gerenciar medicamentos e criar eventos iniciais
- **Tipo**: Produtor + HTTP
- **Emite**: `MedicationCreated`, `MedicationScheduled`

### Scheduler Service (Port 8002)
- **Função**: Consumir eventos e criar agendamentos
- **Tipo**: Consumidor + Produtor + HTTP
- **Consome**: `MedicationCreated`, `MedicationGiven`
- **Emite**: `MedicationDue`, `MedicationOverdue`

### Notification Service (Port 8003)
- **Função**: Enviar notificações baseadas em eventos
- **Tipo**: Consumidor + HTTP
- **Consome**: `MedicationDue`, `MedicationOverdue`

### Analytics Service (Port 8004)
- **Função**: Coletar histórico e fornecer dados para análise
- **Tipo**: Consumidor + HTTP
- **Consome**: Todos os eventos

---

## 🗄️ Banco de Dados

### Tabelas Principais

```sql
medications                 -- Medicamentos cadastrados
medication_schedules        -- Agendamentos
notifications              -- Notificações enviadas
events                      -- Histórico de eventos
outbox                      -- CDC Outbox Pattern
```

Veja [migrations.sql](../backend/shared/database/migrations.sql) para schema completo.

---

## 🔄 Fluxo de Dados

```
┌──────────────────────┐
│  Usuário (Frontend)  │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────────────────────┐
│  Habit Service (/medications POST)   │  ◀─────── HTTP
└──────────┬───────────────────────────┘
           │
           ▼ (transação)
    ┌──────────────┬───────────────┐
    │              │               │
    ▼              ▼               ▼
 [medications] [outbox]     (commit)
    │              │
    └──────────────┘
           │
           ▼ (CDC)
    ┌─────────────────┐
    │ Debezium + WAL  │
    └────────┬────────┘
             │
             ▼
    ┌──────────────────┐
    │ Kafka Topic      │
    │ medication-events│
    └────────┬─────────┘
             │
    ┌────────┴──────────┬────────────────┐
    │                   │                │
    ▼                   ▼                ▼
[Scheduler]        [Notification]   [Analytics]
(Consumidor)       (Consumidor)     (Consumidor)
    │                   │                │
    ▼                   ▼                ▼
[PostgreSQL]        [Redis]         [PostgreSQL]
                   (Idempotência)
```

---

## 🚀 Como Estender o Projeto

### Adicionar um novo Serviço

1. Criar pasta: `backend/services/seu-service/`
2. Criar `main.go` seguindo o padrão descrito acima
3. Criar `go.mod` com:
   ```
   module coxa/seu-service
   go 1.22
   require coxa/shared v0.0.1
   ```
4. Criar `Dockerfile`
5. Adicionar entrada no `docker-compose.yml`
6. Adicionar novo tipo de evento em `backend/shared/events/event.go`

### Adicionar um novo Evento

1. Definir tipo em `backend/shared/events/event.go`
2. Consumir em outro serviço usando `processMessage()`
3. Publicar usando `kafkaProducer.SendMessage()`

### Adicionar um novo Endpoint

1. Criar handler em `main.go`
2. Registrar em `main()`: `http.HandleFunc("/meu-endpoint", handleMeu)`
3. Adicionar métrica no handler para tracking
4. Documentar em `/docs`

---

## 📊 Observabilidade

Todos os serviços expõem:
- **`/metrics`** - Prometheus endpoint (porta :8001-8004)
- **Logs estruturados** - Via `logger` com níveis
- **Métricas obrigatórias**:
  - `events_processed_total`
  - `event_processing_duration_seconds`
  - `errors_total`

Prometheus scrapeiam a cada 15s. Grafana conecta ao Prometheus.

---

## ⚙️ Configurações

### Variáveis de Ambiente

```bash
# Database
DATABASE_URL=user=coxa password=coxa dbname=coxa host=postgres port=5432 sslmode=disable

# Kafka
KAFKA_BROKERS=kafka:9092

# Redis
REDIS_URL=redis://redis:6379

# Serviço específico
SERVICE_NAME=habit-service
```

Todos os padrões estão codificados em `main.go`. Para ambiente de produção, considere usar config files.

---

## 🧪 Testando Localmente

Veja [DEVELOPMENT.md](./DEVELOPMENT.md) para instruções de setup e testes.

Veja [TESTE_END_TO_END.md](./TESTE_END_TO_END.md) para fluxo completo.
