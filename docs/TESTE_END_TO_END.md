# 🧪 Teste End-to-End Manual - COXA Event-Driven Architecture

## 📋 Fluxo Completo a Testar

```
1. Cadastrar Remédio (REST API)
   ↓
2. Salvar no Banco de Dados
   ↓
3. Salvar no Outbox (PostgreSQL)
   ↓
4. Debezium CDC publica no Kafka
   ↓
5. Scheduler Service consome do Kafka
   ↓
6. Cria Agendamento no Banco
   ↓
7. Publica MedicationScheduled no Kafka
   ↓
8. Métricas registradas no Prometheus
```

---

## 🚀 Passo a Passo Detalhado

### **PASSO 1: Verificar que todos os serviços estão rodando**

```bash
docker compose ps
```

**Esperado:** Ver todos os 12 containers com status `Up`

**Serviços críticos:**
- ✅ coxa-habit-service (8001)
- ✅ coxa-scheduler-service (8002)
- ✅ coxa-postgres (5432)
- ✅ coxa-kafka (9092)
- ✅ coxa-redis (6379)

---

### **PASSO 2: Testar saúde das APIs**

Execute estes comandos para verificar que as APIs estão respondendo:

```bash
# Habit Service
curl http://localhost:8001/health

# Scheduler Service
curl http://localhost:8002/health

# Notification Service
curl http://localhost:8003/health

# Analytics Service
curl http://localhost:8004/health
```

**Esperado:** Todas retornam
```json
{"status":"ok"}
```

---

### **PASSO 3: Criar um remédio via API**

```bash
curl -X POST http://localhost:8001/medications \
  -d "name=Remédio Teste&description=Teste E2E&frequency_days=7&pet_id=pet-teste&owner_id=owner-teste" \
  -H "Content-Type: application/x-www-form-urlencoded"
```

**Esperado:** Recebe uma resposta com o ID do remédio criado
```json
{"id": "uuid-aqui", "message": "Medicamento criado com sucesso"}
```

**⚠️ IMPORTANTE:** Salve o ID do remédio! Você vai precisar dele nos próximos passos.
```
ID = uuid-aqui
```

---

### **PASSO 4: Verificar se o remédio foi salvo no banco (Habit Service)**

```bash
curl http://localhost:8001/medications | python3 -m json.tool | head -20
```

**Esperado:** Ver o remédio criado no topo da lista com seus dados

---

### **PASSO 5: Verificar se foi publicado no Kafka**

Aguarde 1-2 segundos e leia a mensagem do Kafka:

```bash
docker exec coxa-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic medication-events \
  --max-messages 1 2>/dev/null | tail -1 | python3 -m json.tool
```

**Esperado:** Ver um JSON com:
```json
{
  "id": 1,
  "aggregate_id": "uuid-do-medicamento",
  "event_type": "MedicationCreated",
  "payload": "{\"name\": \"Remédio Teste\", \"pet_id\": \"pet-teste\", ...}",
  "correlation_id": "uuid",
  "created_at": 1777945412...
}
```

---

### **PASSO 6: Verificar logs do Scheduler Service**

```bash
docker logs coxa-scheduler-service 2>&1 | tail -20
```

**Esperado:** Ver logs mostrando:
```
"mensagem recebida: XXX bytes"
"MedicationCreated processado com sucesso"
```

Exemplo:
```json
{"timestamp":"2026-05-05T01:16:02.504796753Z","event_id":"uuid-medicamento-15","correlation_id":"uuid-correlation","status":"processed","message":"MedicationCreated processado com sucesso"}
```

---

### **PASSO 7: Verificar se o agendamento foi criado no Scheduler**

Use o ID do remédio que você salvou no Passo 3:

```bash
curl http://localhost:8002/schedules | python3 -c "
import sys, json
schedules = json.load(sys.stdin)
med_id = 'SEU_ID_AQUI'  # Substitua com o ID do Passo 3
matching = [s for s in schedules if s['medication_id'] == med_id]
if matching:
    print(f'✅ Encontrado! Agendamento criado:')
    print(json.dumps(matching[0], indent=2))
else:
    print(f'❌ Nenhum agendamento encontrado para {med_id}')
"
```

**Esperado:** Ver um agendamento com:
```json
{
  "id": "uuid-agendamento",
  "medication_id": "uuid-medicamento",
  "pet_id": "pet-teste",
  "owner_id": "owner-teste",
  "next_due_date": "2026-05-12T...",  // 7 dias no futuro
  "status": "pending"
}
```

---

### **PASSO 8: Verificar Métricas no Prometheus**

Abra no navegador:
```
http://localhost:9090
```

1. Na caixa de busca, digite: `events_processed_total`
2. Clique em "Execute"

**Esperado:** Ver um gráfico mostrando eventos processados:
```
events_processed_total{service="scheduler-service"} = número > 0
```

---

### **PASSO 9: Verificar Métricas no Grafana (Opcional)**

Abra no navegador:
```
http://localhost:3000
```

Login: `admin` / `admin`

1. Vá para "Home" → "Dashboards"
2. Procure por um dashboard de eventos/métricas
3. Você verá gráficos com eventos processados

---

### **PASSO 10: Testar Medicamento Vencido (Verificar Notification Service)**

Crie um remédio com agendamento para hoje/ontem:

```bash
# Usar frequency_days negativo não funciona, então vamos criar e esperar a verificação periódica
# O scheduler-service verifica a cada 1 minuto se há medicamentos vencidos

# Aguarde 1-2 minutos
sleep 120

# Verifique logs da notification-service
docker logs coxa-notification-service 2>&1 | tail -15
```

**Esperado:** Ver eventos de `MedicationDue` sendo processados

---

## ✅ Checklist Final - Sistema Funcionando Corretamente

- [ ] **Passo 1**: Todos os 12 containers estão `Up`
- [ ] **Passo 2**: Todas as APIs retornam `{"status":"ok"}`
- [ ] **Passo 3**: Remédio criado com sucesso e obtive um ID
- [ ] **Passo 4**: Remédio aparece na lista de medicamentos
- [ ] **Passo 5**: Mensagem apareceu no Kafka com `event_type: MedicationCreated`
- [ ] **Passo 6**: Logs mostram "MedicationCreated processado com sucesso"
- [ ] **Passo 7**: Agendamento foi criado automaticamente (status: pending)
- [ ] **Passo 8**: Métricas mostram `events_processed_total` > 0
- [ ] **Passo 9**: Dashboard Grafana mostra eventos processados
- [ ] **Passo 10**: Notification service recebe eventos (opcional)

---

## 🔍 Troubleshooting

### ❌ Remédio não aparece na lista do Passo 4
```bash
# Verifique banco diretamente
docker exec coxa-postgres psql -U coxa -d coxa -c "SELECT * FROM medications;"
```

### ❌ Mensagem não aparece no Kafka (Passo 5)
```bash
# Verifique se o Outbox foi preenchido
docker exec coxa-postgres psql -U coxa -d coxa -c "SELECT * FROM outbox;"

# Verifique logs do Debezium
docker logs coxa-debezium | tail -30
```

### ❌ Agendamento não foi criado (Passo 7)
```bash
# Verifique logs do scheduler-service
docker logs coxa-scheduler-service | tail -30

# Verifique se há erros de desserialização
docker logs coxa-scheduler-service | grep "erro"
```

### ❌ Métricas não aparecem (Passo 8)
```bash
# Verifique se o Prometheus está scrapeando os targets
curl http://localhost:9090/api/v1/targets 2>/dev/null | python3 -m json.tool | head -30
```

---

## 📊 Fluxo de Dados Mapeado

```
┌─────────────────────────────────────────────────────────┐
│ 1. Cadastro de Remédio (POST /medications)              │
│    → Habitat Service                                    │
└──────────────┬──────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Salva em PostgreSQL                                  │
│    → Tabela: medications                                │
│    → Tabela: outbox (para CDC)                          │
└──────────────┬──────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Debezium CDC detecta mudança                         │
│    → Lê do WAL do PostgreSQL                            │
│    → Publica em Kafka                                   │
└──────────────┬──────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Kafka Topic: medication-events                       │
│    → Mensagem: MedicationCreated                        │
└──────────────┬──────────────────────────────────────────┘
               │
      ┌────────┴────────┬──────────────┬──────────────┐
      ▼                 ▼              ▼              ▼
┌──────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ Scheduler    │ │Notif.    │ │Analytics │ │Habit     │
│ Service      │ │Service   │ │Service   │ │Service   │
└──┬───────────┘ └──────────┘ └──────────┘ └──────────┘
   │
   ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Cria Agendamento (medication_schedules)              │
│    → next_due_date = hoje + frequency_days              │
│    → status = "pending"                                 │
└──────────────┬──────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────┐
│ 6. Publica novo evento: MedicationScheduled             │
│    → Volta ao Kafka                                     │
│    → Registra métrica em Prometheus                     │
└─────────────────────────────────────────────────────────┘
```

---

## 🎯 Sucesso!

Se todos os checkboxes estão marcados ✅, seu sistema **Event-Driven Architecture está 100% funcional!**

Você demonstrou com sucesso:
- ✅ Criação de dados via REST API
- ✅ Persistência em banco de dados
- ✅ Change Data Capture (CDC) funcionando
- ✅ Publicação em Message Broker (Kafka)
- ✅ Consumo de eventos por múltiplos serviços
- ✅ Processamento distribuído
- ✅ Observabilidade com Prometheus
- ✅ Idempotência com Redis

**🎉 Arquitetura de Microsserviços Event-Driven completa e testada!**
