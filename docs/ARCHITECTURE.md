# COXA Architecture

Documentação da arquitetura real de COXA - um sistema **orientado a eventos** que gerencia medicamentos de animais de estimação de forma assíncrona e confiável.

> ⚠️ **Documentação Atualizada**: Esta documentação descreve a **implementação atual** do projeto. Se você é novo no projeto, comece por [SETUP.md](./SETUP.md) e depois leia [PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md) para entender a estrutura.

---

## 🎯 Objetivo

Demonstrar um sistema robusto baseado em **Event-Driven Architecture (EDA)** com:
- ✅ Comunicação 100% assíncrona entre serviços
- ✅ Garantia de consistência eventual
- ✅ Recuperação automática de falhas
- ✅ Observabilidade completa (métricas, logs, traces)
- ✅ Escalabilidade horizontal

---

## 🏗️ Visão Geral da Arquitetura

```
┌─────────────────────────────────────────────────────────────┐
│                    COXA System                              │
│                                                              │
│  ┌──────────────┐                                           │
│  │ Frontend     │───────────────┐                           │
│  │ (React)      │               │ HTTP                      │
│  └──────────────┘               │                           │
│                                 ▼                           │
│  ┌─────────────────────────────────────────────────┐       │
│  │ API Layer (HTTP Services)                       │       │
│  ├─────────────┬──────────┬──────────┬──────────┤       │
│  │ Habit       │Scheduler │Notif.   │Analytics │       │
│  │ Service     │Service   │Service   │Service   │       │
│  │ (8001)      │(8002)    │(8003)    │(8004)    │       │
│  └────┬────────┴──────┬───┴──────┬───┴──────┬───┘       │
│       │                │          │          │            │
│  Write│         Read   │   Read   │   Read   │            │
│       ▼                ▼          ▼          ▼            │
│   ┌─────────────────────────────────────────┐            │
│   │      PostgreSQL Database                │            │
│   │  ┌─────────────────────────────────┐   │            │
│   │  │ medications                      │   │            │
│   │  │ medication_schedules             │   │            │
│   │  │ notifications                    │   │            │
│   │  │ events                           │   │            │
│   │  │ outbox (CDC)  ──────────┐        │   │            │
│   │  └─────────────────────────┼──────┐│   │            │
│   └──────────────────────────────────┼┼───┘            │
│                                      ││                 │
│  ┌───────────────────────────────────┘│                │
│  │ Debezium CDC                       │                │
│  │ (Change Data Capture)              │                │
│  └────────────────┬────────────────────                │
│                   │ Publica eventos                    │
│                   ▼                                     │
│   ┌──────────────────────────────────┐                │
│   │ Kafka                            │                │
│   │ Topic: medication-events         │                │
│   └────┬─────────┬──────────┬────────┘                │
│        │         │          │                         │
│  Subscr│ibes Subscr│ibes  Subscr│ibes              │
│        │         │          │                         │
│        ▼         ▼          ▼                         │
│    ┌────────┬────────┬──────────┐                    │
│    │Scheduler│Notif. │Analytics │                    │
│    │Service  │Service│Service   │                    │
│    └────┬────┴──┬─────┴──┬───────┘                    │
│         │       │        │                            │
│         ▼       ▼        ▼                            │
│    ┌────────────────────────────┐                    │
│    │ Redis                      │                    │
│    │ (Event Deduplication)      │                    │
│    │ TTL 24h                    │                    │
│    └────────────────────────────┘                    │
│                                                       │
│  ┌────────────────────────────────────────┐         │
│  │ Prometheus + Grafana                   │         │
│  │ (Observability)                        │         │
│  │ • Métricas por serviço                │         │
│  │ • Latência de eventos                 │         │
│  │ • Taxa de erros                       │         │
│  └────────────────────────────────────────┘         │
│                                                       │
└─────────────────────────────────────────────────────┘
```

---

## 🧩 Serviços

---

## 🧩 Serviços

### Habit Service (Port 8001)

Gerencia medicamentos e agendamentos iniciais.

**Endpoints:**
- `POST /medications` - Criar medicamento
- `GET /medications` - Listar medicamentos
- `GET /metrics` - Métricas Prometheus

**Eventos Publicados:**
- `MedicationCreated`
- `MedicationScheduled`

---

## 🧩 Serviços

Cada serviço é um **módulo independente** em Go com um arquivo `main.go`.

### **1. Habit Service (Port 8001)**

**Responsabilidade**: Criar medicamentos e iniciar o fluxo de eventos.

**Tipo**: Produtor de eventos + HTTP API

**Endpoints**:
- `POST /medications` - Criar novo medicamento
- `GET /medications` - Listar medicamentos
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

**Eventos Publicados**:
- `MedicationCreated` - Quando medicamento é criado
- `MedicationScheduled` - Quando agendamento inicial é criado

**Fluxo Interno**:
1. Recebe POST em `/medications`
2. Inicia transação com PostgreSQL
3. Salva medicamento em tabela `medications`
4. Salva evento em tabela `outbox` (mesma transação)
5. Commit da transação
6. Debezium CDC detecta mudança e publica no Kafka

---

### **2. Scheduler Service (Port 8002)**

**Responsabilidade**: Consumir eventos de medicamentos e criar/gerenciar agendamentos.

**Tipo**: Consumidor + Produtor + HTTP API

**Endpoints**:
- `GET /schedules` - Listar agendamentos
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

**Eventos Consumidos**:
- `MedicationCreated` - Cria novo agendamento
- `MedicationGiven` - Atualiza last_given_date

**Eventos Publicados**:
- `MedicationDue` - Medicamento está vencido
- `MedicationOverdue` - Medicamento está atrasado

**Processo**:
1. Consome mensagens do Kafka (topic: medication-events)
2. Valida com Redis se já foi processado (idempotência)
3. Processa dependendo do tipo de evento:
   - `MedicationCreated`: Cria registro em `medication_schedules` com `next_due_date`
   - `MedicationGiven`: Atualiza `last_given_date` e calcula próximo vencimento
4. A cada minuto, job verifica medicamentos vencidos e publica `MedicationDue`

---

### **3. Notification Service (Port 8003)**

**Responsabilidade**: Consumir eventos de medicamentos vencidos e enviar notificações.

**Tipo**: Consumidor + HTTP API

**Endpoints**:
- `GET /notifications` - Listar notificações
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

**Eventos Consumidos**:
- `MedicationDue` - Medicamento está vencido
- `MedicationOverdue` - Medicamento está atrasado

**Processo**:
1. Consome `MedicationDue` do Kafka
2. Marca como processado no Redis (idempotência)
3. Registra em `notifications` com status `sent`
4. Log: "🔔 Medicamento vencido para pet-xxx"

**Nota**: Atualmente apenas registra em banco. Pode ser estendido para enviar emails, SMS, push notifications, etc.

---

### **4. Analytics Service (Port 8004)**

**Responsabilidade**: Consumir eventos e fornecer dados para análise/histórico.

**Tipo**: Consumidor + HTTP API (read-only)

**Endpoints**:
- `GET /timeline?owner_id=xyz` - Timeline de eventos para um owner
- `GET /stats?owner_id=xyz` - Estatísticas agregadas
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

**Eventos Consumidos**:
- Todos os eventos para construir histórico

**Processo**:
1. Consome todos os eventos do Kafka
2. Marca como processado no Redis (idempotência)
3. Salva em tabela `events` com timestamp
4. Fornece APIs para query de histórico

---

## 📦 Padrões Arquiteturais

### 1. **Event-Driven Architecture (EDA)**

Toda comunicação entre serviços é **100% assíncrona** via Kafka:

```
Serviço A escribe em DB
    ↓
Debezium CDC detecta mudança
    ↓
Publica evento em Kafka
    ↓
Serviço B consome evento
    ↓
Serviço B processa e atualiza seu estado
```

**Benefícios**:
- ✅ Serviços desacoplados
- ✅ Resiliência a falhas (retry automático)
- ✅ Escalabilidade horizontal
- ✅ Histórico de eventos preservado

---

### 2. **Outbox Pattern**

Garante que eventos são publicados **atomicamente** com dados:

```go
tx := db.BeginTx()

// 1. Salvar dados
tx.Exec("INSERT INTO medications ...")

// 2. Salvar evento na MESMA transação
tx.Exec("INSERT INTO outbox (event_type, payload) ...")

// 3. Commit único
tx.Commit()  // Tudo ou nada
```

**Por quê?**: Evita perda de eventos por falhas na publicação. Se commit falhar, evento não é salvo. Se houver crash após commit, Debezium garante a publicação.

---

### 3. **Change Data Capture (CDC) com Debezium**

Debezium monitora PostgreSQL e publica mudanças no Kafka:

```
PostgreSQL WAL (Write-Ahead Log)
    ↓
Debezium Connector (lê mudanças)
    ↓
Kafka Topic: medication-events
    ↓
Serviços Consumidores
```

**Vantagens**:
- ✅ Não requer mudanças no código de insert
- ✅ Garante ordem de eventos
- ✅ CDC é agnóstico ao serviço (desacoplamento)
- ✅ Pode ser ligado/desligado sem impacto

---

### 4. **Idempotência com Redis**

Cada serviço consumidor marca eventos como processados:

```
Evento chega: "MedicationCreated-123"
    ↓
Redis: GET "processed:MedicationCreated-123" → null
    ↓
Processa normalmente
    ↓
Redis: SET "processed:MedicationCreated-123" "yes" EX 86400
    ↓
Se mesmo evento chegar novamente:
    → Redis GET → "yes" → Descarta (já processado)
```

**TTL 24h**: Eventos duplicados após 24h são reprocessados (seguro em nosso caso).

**Por quê?**: Kafka pode entregar mesma mensagem 2x em caso de falha/recovery. Redis garante processamento único.

---

### 5. **Database per Service**

Cada serviço tem seus próprios schemas/tables no PostgreSQL:

```
Habit Service → medications, outbox
Scheduler Service → medication_schedules
Notification Service → notifications
Analytics Service → events
```

**Dados compartilhados?** Via eventos no Kafka (eventual consistency).

---

## 🔄 Fluxo Completo de um Medicamento

Vamos rastrear um medicamento do início ao fim:

### **T=0s: Usuário cria medicamento**

```bash
POST /medications HTTP/1.1
Host: localhost:8001
Content-Type: application/x-www-form-urlencoded

name=Vermífugo&pet_id=pet-123&owner_id=owner-456&frequency_days=30
```

---

### **T=0.1s: Habit Service processa**

```go
// Habilit Service - handleCreateMedication()
tx := db.BeginTx()

// Salva medicamento
tx.Exec(`INSERT INTO medications 
         (id, name, pet_id, owner_id, frequency_days, created_at)
         VALUES (?, ?, ?, ?, ?, ?)`, 
         "med-uuid-123", "Vermífugo", "pet-123", "owner-456", 30, now)

// Salva evento no outbox
tx.Exec(`INSERT INTO outbox 
         (aggregate_id, event_type, payload, correlation_id)
         VALUES (?, ?, ?, ?)`,
         "med-uuid-123", "MedicationCreated", 
         {"name":"Vermífugo"...}, "correlation-id-456")

tx.Commit()  // ✅ Ambos salvos atomicamente
```

---

### **T=0.5s: Debezium CDC detecta mudança**

```
Debezium lê WAL do PostgreSQL
  → Detecta INSERT em outbox
  → Formata como JSON
  → Publica em Kafka
```

**Mensagem no Kafka**:
```json
{
  "id": 1,
  "aggregate_id": "med-uuid-123",
  "event_type": "MedicationCreated",
  "payload": "{\"name\": \"Vermífugo\", \"pet_id\": \"pet-123\", ...}",
  "correlation_id": "correlation-id-456",
  "created_at": 1718899500000
}
```

---

### **T=1s: Scheduler Service consome**

```go
// Scheduler Service - consumeEvents()
msg := kafkaConsumer.ReadMessage()
msgMap := json.Unmarshal(msg)

eventType := msgMap["event_type"]  // "MedicationCreated"

if eventType == "MedicationCreated" {
    // 1. Verifica idempotência
    key := fmt.Sprintf("processed:%s", msg.ID)
    exists := redisClient.GET(key)
    if exists != nil {
        return  // Já foi processado
    }
    
    // 2. Processa evento
    payload := json.Unmarshal(msgMap["payload"])
    
    db.Exec(`INSERT INTO medication_schedules
             (medication_id, pet_id, owner_id, next_due_date, status)
             VALUES (?, ?, ?, ?, ?)`,
             "med-uuid-123", "pet-123", "owner-456", 
             time.Now().AddDate(0, 0, 30),  // +30 dias
             "pending")
    
    // 3. Marca como processado
    redisClient.SET(key, "yes", "EX", 86400)
    
    // 4. Emite métrica
    metrics.EventsProcessedTotal.Inc()
}
```

---

### **T=2s: Scheduler Service publica novo evento**

```go
// Publica MedicationScheduled no Kafka
kafkaProducer.SendMessage("medication-events", {
    "aggregate_id": "med-uuid-123",
    "event_type": "MedicationScheduled",
    "payload": {...},
    "correlation_id": "correlation-id-456"
})
```

---

### **T=3s: Cada serviço consumidor processa (paralelo)**

**Notification Service**:
```go
if eventType == "MedicationScheduled" {
    // Pode registrar aviso inicial, etc
    logger.LogProcessed("", "", "Agendamento criado para pet-123")
}
```

**Analytics Service**:
```go
if eventType == "MedicationScheduled" {
    db.Exec(`INSERT INTO events 
             (aggregate_id, event_type, owner_id, created_at)
             VALUES (?, ?, ?, ?)`,
             "med-uuid-123", "MedicationScheduled", "owner-456", now)
}
```

---

### **T=30 dias depois: Scheduler detecta vencimento**

```go
// Job que roda a cada 1 minuto
schedules := db.Query(`SELECT * FROM medication_schedules 
                       WHERE next_due_date <= NOW() AND status = 'pending'`)

for schedule in schedules {
    // Publica evento de vencimento
    kafkaProducer.SendMessage("medication-events", {
        "aggregate_id": schedule.medication_id,
        "event_type": "MedicationDue",
        "payload": {...}
    })
}
```

---

### **T=30 dias + 1 minuto: Notification Service recebe**

```go
if eventType == "MedicationDue" {
    logger.LogProcessed("", "", "🔔 Medicamento vencido para " + petID)
    
    db.Exec(`INSERT INTO notifications
             (medication_id, pet_id, owner_id, status)
             VALUES (?, ?, ?, ?)`,
             medicationID, petID, ownerID, "sent")
}
```

---

### **T=30 dias + 1 minuto: Analytics registra**

```go
if eventType == "MedicationDue" {
    db.Exec(`INSERT INTO events
             (aggregate_id, event_type, owner_id, created_at)
             VALUES (?, ?, ?, ?)`,
             medicationID, "MedicationDue", ownerID, now)
}
```

---

### **Resultado Final (Timeline)**

```
owner-456 pode ver em Analytics:
  
  [T=0s]    MedicationCreated   - Vermífugo criado
  [T=1s]    MedicationScheduled - Próxima dose em 30 dias
  [T=30d]   MedicationDue       - Vermífugo vencido!
  [T=30d]   NotificationSent    - Notificação enviada
```

---

## 📊 Observabilidade

Todo serviço expõe `/metrics` em Prometheus format:

### Métricas Obrigatórias

```
# Counter - Total de eventos processados
events_processed_total{service="habit-service"} 150

# Histogram - Duração do processamento (em segundos)
event_processing_duration_seconds_bucket{le="0.1"} 140
event_processing_duration_seconds_bucket{le="1.0"} 145
event_processing_duration_seconds_bucket{le="+Inf"} 150

# Counter - Total de erros
errors_total{service="habit-service"} 2
```

### Scraping

Prometheus scrapeiam cada serviço a cada **15 segundos**:

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'habit-service'
    static_configs:
      - targets: ['localhost:8001']
  - job_name: 'scheduler-service'
    static_configs:
      - targets: ['localhost:8002']
  # ... etc
```

### Visualização

Grafana se conecta ao Prometheus e cria dashboards:
- Throughput (eventos/min)
- Latência (p50, p95, p99)
- Taxa de erros
- Distribuição de eventos por tipo

---

## 🚨 Cenários de Falha & Recuperação

### **Cenário 1: Scheduler Service cai**

```
Habit Service continue normalmente
    ↓
Medicamentos são criados e salvos em outbox
    ↓
Debezium continue publicando no Kafka
    ↓
Eventos acumulam na fila do Kafka
    ↓
Scheduler Service volta online
    ↓
Consome eventos acumulados
    ↓
Sistema se recupera automaticamente ✅
```

---

### **Cenário 2: PostgreSQL cai temporariamente**

```
Serviços que escrevem falham em transações
    ↓
Retentam automaticamente
    ↓
Kafka retém eventos
    ↓
PostgreSQL volta online
    ↓
Sistema recupera e processa tudo ✅
```

---

### **Cenário 3: Mesmo evento é publicado 2x (duplicata Kafka)**

```
T=0: Evento A chega ao Scheduler
T=1: Scheduler processa A, marca em Redis "processed:A"
T=2: Mesmo evento A chega novamente
T=3: Scheduler verifica Redis → já processado → Descarta ✅
```

---

## 🔧 Estrutura de Código

**Cada serviço está implementado em um único `main.go`:**

```go
package main

import "github.com/prometheus/client_golang/prometheus/promhttp"

// 1. DEPENDÊNCIAS GLOBAIS (init)
var (
    db            *sql.DB
    kafkaProducer *infra.KafkaProducer
    kafkaConsumer *infra.KafkaConsumer
    logger        *infra.Logger
    metrics       *infra.ServiceMetrics
    redisClient   *infra.RedisClient
)

func init() {
    // Conectar DB, Kafka, Redis, Logger, Metrics
}

// 2. ESTRUTURAS DE DOMÍNIO
type Medication struct {
    ID            string
    PetID         string
    OwnerID       string
    Name          string
    FrequencyDays int
    CreatedAt     time.Time
}

// 3. HANDLERS HTTP (Producers)
func handleCreateMedication(w http.ResponseWriter, r *http.Request) {
    // Parsing de form data
    // Validação
    // Transação: INSERT medications + INSERT outbox
    // Response JSON
}

// 4. CONSUMERS KAFKA
func consumeEvents(ctx context.Context) {
    for {
        msg := kafkaConsumer.ReadMessage(ctx)
        processMessage(ctx, msg)
    }
}

func processMessage(ctx context.Context, msgData []byte) {
    // Parse JSON
    // Validar idempotência com Redis
    // Processar por tipo de evento
    // Atualizar DB
    // Marcar como processado no Redis
}

// 5. MAIN - Orquestra tudo
func main() {
    // Registrar handlers HTTP
    http.HandleFunc("/medications", handleCreateMedication)
    http.Handle("/metrics", promhttp.Handler())
    
    // Iniciar consumer em goroutine
    go consumeEvents(context.Background())
    
    // Iniciar servidor HTTP
    http.ListenAndServe(":8001", nil)
}
```

**Por que um único main.go?**
- Serviços são simples e focados
- Fácil compreender o fluxo todo
- Sem complexidade de imports circulares
- À medida que cresce, refatorar para `internal/` é possível

---

## 📁 Arquivos Compartilhados

No diretório `/backend/shared/`, todos os serviços importam:

```go
import "coxa/shared/database"    // ConnectDB(), outbox operations
import "coxa/shared/events"      // Event struct definition
import "coxa/shared/infra"       // Kafka, Redis, Logger, Metrics
```

Isso garante consistência entre serviços.

---

## 🌉 Padrão de Comunicação

Não há chamadas HTTP diretas entre serviços. **100% via Kafka**:

```
❌ ERRADO (acoplado):
  scheduler-service → HTTP → habit-service
  ↑ Falha se habit-service cair
  ↑ Necessita sincronização

✅ CORRETO (desacoplado):
  habit-service → salva evento → outbox → Kafka
                      ↓
  scheduler-service ← consome do Kafka
  ↑ Funciona mesmo se scheduler cair
  ↑ Assíncrono, recuperação automática
```

---

## 🔐 Garantias do Sistema

| Propriedade | Como é Garantido |
|---|---|
| **Atomicidade** | Transação DB única (insert data + event) |
| **Idempotência** | Redis com chave `processed:${eventId}`, TTL 24h |
| **Ordenação** | Kafka mantém ordem dentro da partição |
| **Persistência** | Outbox + CDC + WAL do PostgreSQL |
| **Consistência Eventual** | Cada serviço converge para mesmo estado |
| **Recuperação** | Kafka retém mensagens, serviços consomem ao voltar |

---

## 🚀 Próximos Passos para Desenvolvedores

1. **Entender a estrutura real do projeto**
   - Leia [PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md)

2. **Fazer setup local**
   - Siga [SETUP.md](./SETUP.md)

3. **Executar teste end-to-end**
   - Siga [TESTE_END_TO_END.md](./TESTE_END_TO_END.md)

4. **Desenvolvimento local**
   - Leia [DEVELOPMENT.md](./DEVELOPMENT.md)

5. **Contribuir com código**
   - Leia [CONTRIBUTING.md](./CONTRIBUTING.md)

---

## 📚 Referências

- [Designing Data-Intensive Applications](https://dataintensive.net/) - Cap. 11 (Event Sourcing)
- [Building Event-Driven Systems](https://www.oreilly.com/library/view/building-event-driven-systems/9781492038023/)
- [Debezium CDC Documentation](https://debezium.io/documentation/)
- [Kafka Event Streaming](https://kafka.apache.org/)
- [Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)
- [Idempotent Consumers](https://kafka.apache.org/documentation/#semantics)

---

**Versão da Documentação**: 1.0 (Atualizado em Junho 2026)  
**Correspondência com Código**: ✅ Verificado em 2026-06-20

**Frontend** marca como administrado:
```
PUT /medications/{id}/given
```

Publica `MedicationGiven`:
```json
{
  "medication_id": "med-123",
  "admin_date": "2026-01-15T10:30:00Z",
  "next_due_date": "2026-03-16T10:30:00Z"
}
```

---

### 6. Analytics

**Analytics Service** consome todos:
- Registra administração em `medication_administrations`
- Calcula histórico, stats, timeline
- Disponibiliza em `/timeline` e `/stats`

---

## 📡 Contrato de Eventos

Todos os eventos seguem este padrão:

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "MedicationCreated",
  "aggregate_id": "med-123",
  "correlation_id": "corr-456",
  "payload": {
    "name": "Antiflea",
    "frequency_days": 60,
    "pet_id": "pet-123",
    "owner_id": "owner-456"
  },
  "created_at": "2026-01-01T10:00:00Z"
}
```

---

## 🎨 Frontend Design System

### Cores (Dark-first)

- **Background**: `#0F172A`
- **Surface**: `#1E293B`
- **Primary**: `#F59E0B`
- **Text Primary**: `#F8FAFC`
- **Text Secondary**: `#94A3B8`
- **Accent**: `#E6B370`
- **Border**: `#334155`

### Componentes

- **Button**: Primary (orange), Secondary (outline)
- **Card**: Surface background com border
- **Input**: Dark background, orange focus
- **Table**: Com hover effect

### Estilo Visual

- Dark mode por padrão
- Bordas arredondadas (`rounded-xl`)
- Sombras leves e minimalistas
- Espaçamento consistente
- Tipografia clara

---

## 🔧 Desenvolvimento

### Estrutura de Diretórios

```
/coxa
├── backend/
│   ├── services/
│   │   ├── habit-service/
│   │   ├── scheduler-service/
│   │   ├── notification-service/
│   │   └── analytics-service/
│   └── shared/
│       ├── events/
│       ├── infra/
│       └── database/
├── frontend/
│   └── web-app/
├── infra/
│   ├── prometheus/
│   ├── debezium/
│   └── postgres/
└── docker-compose.yml
```

---

### Build Local

```bash
# Backend - Habit Service
cd backend/services/habit-service
go run main.go

# Frontend
cd frontend/web-app
npm install
npm run dev
```

---

## 📚 Recursos Importantes

### Arquivos Chave

- `backend/shared/events/event.go` - Definição de eventos
- `backend/shared/infra/kafka_*.go` - Produtor/Consumidor Kafka
- `backend/shared/infra/redis_client.go` - Idempotência
- `backend/shared/infra/metrics.go` - Métricas Prometheus
- `backend/shared/database/migrations.sql` - Schema do banco
- `docker-compose.yml` - Orquestração de containers

---

## ✅ Checklist de Features

- ✅ 4 Microserviços independentes
- ✅ Arquitetura Orientada a Eventos (EDA)
- ✅ Kafka para comunicação assíncrona
- ✅ Outbox Pattern com CDC (Debezium)
- ✅ Idempotência com Redis
- ✅ Clean Architecture + Hexagonal
- ✅ PostgreSQL com migrations
- ✅ Prometheus + Grafana
- ✅ Logging estruturado (JSON)
- ✅ Health checks
- ✅ Docker Compose orchestration
- ✅ Frontend React com design system
- ✅ Tratamento de falhas e recuperação
- ✅ Correlação de eventos

---

## 🚀 Próximos Passos

1. Deploy em Kubernetes (Helm charts)
2. APM com Jaeger
3. Circuit breaker com Hystrix
4. GraphQL API
5. Autenticação/Autorização
6. Rate limiting
7. Versionamento de APIs
8. Testes de integração
9. Load testing
10. Disaster recovery

---

## 📞 Suporte

Para questões sobre arquitetura, eventos ou deployment, consulte:

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- Logs do Docker: `docker-compose logs -f [service-name]`

---

**COXA - 2026** | Event-Driven Architecture | Production-Ready System
