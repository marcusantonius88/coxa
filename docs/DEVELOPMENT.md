# COXA Development Guide

## Estrutura de Desenvolvimento

### Backend

Cada serviço segue Clean Architecture + Hexagonal Architecture:

```
service/
├── main.go              # Entrypoint e HTTP handlers
├── go.mod              # Dependências Go
└── Dockerfile          # Build da imagem Docker
```

### Frontend

```
frontend/web-app/
├── package.json        # Dependências npm
├── vite.config.js      # Configuração Vite
├── tailwind.config.js  # Configuração TailwindCSS
├── src/
│   ├── App.jsx         # Componente principal
│   ├── main.jsx        # Entrypoint React
│   ├── index.css       # Estilos globais
│   └── components/     # Componentes reutilizáveis
└── Dockerfile          # Build produção
```

## Desenvolvendo Localmente

### 1. Backend Service

```bash
cd backend/services/habit-service

# Instalar dependências
go mod download

# Executar serviço
go run main.go
```

**Variáveis de ambiente:**

```bash
export DATABASE_URL="user=coxa password=coxa dbname=coxa host=localhost port=5432 sslmode=disable"
export KAFKA_BROKERS="localhost:9092"
export REDIS_URL="redis://localhost:6379"
```

### 2. Frontend

```bash
cd frontend/web-app

# Instalar dependências
npm install

# Servidor de desenvolvimento (hot reload)
npm run dev

# Build produção
npm run build
```

**Acesso:** http://localhost:5173

## Fluxo de Desenvolvimento

### 1. Criar novo evento

1. Adicionar type em `backend/shared/events/event.go`
2. Publicar no Kafka Producer
3. Consumir no Kafka Consumer
4. Testar com curl

### 2. Adicionar novo endpoint

1. Criar handler em `main.go`
2. Registrar rota HTTP
3. Adicionar métrica Prometheus
4. Testar com curl

### 3. Adicionar novo componente React

1. Criar arquivo em `src/components/ComponentName.jsx`
2. Importar e usar em `App.jsx`
3. Seguir design system (cores, espaçamento)
4. Verificar responsividade

## Debugging

### Logs

```bash
# Ver logs em tempo real
docker-compose logs -f [service-name]

# Ver logs de um container específico
docker logs -f coxa-habit-service
```

### Kafka

```bash
# Consumir mensagens de um tópico
docker exec -it coxa-kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic medication-events --from-beginning

# Produzir mensagem de teste
docker exec -it coxa-kafka kafka-console-producer --broker-list localhost:9092 --topic medication-events
```

### PostgreSQL

```bash
# Conectar ao banco
docker exec -it coxa-postgres psql -U coxa -d coxa

# Consultas úteis
SELECT * FROM outbox WHERE published_at IS NULL;
SELECT * FROM medications ORDER BY created_at DESC LIMIT 10;
SELECT COUNT(*) FROM medication_schedules WHERE status = 'due';
```

### Redis

```bash
# Conectar ao Redis
docker exec -it coxa-redis redis-cli

# Ver chaves processadas (idempotência)
KEYS "processed:*"
TTL "processed:event-id-123"
```

## Testes

### Teste de Criação de Medicamento

```bash
curl -X POST http://localhost:8001/medications \
  -d "name=Antibiótico&description=Para infecção&frequency_days=7&pet_id=pet-123&owner_id=owner-456"
```

### Listar Medicamentos

```bash
curl http://localhost:8001/medications
```

### Listar Agendamentos

```bash
curl http://localhost:8002/schedules
```

### Listar Notificações

```bash
curl http://localhost:8003/notifications
```

### Timeline de Analytics

```bash
curl "http://localhost:8004/timeline?owner_id=owner-456"
```

### Stats de Analytics

```bash
curl "http://localhost:8004/stats?owner_id=owner-456"
```

### Métricas Prometheus

```bash
curl http://localhost:8001/metrics
curl http://localhost:8002/metrics
curl http://localhost:8003/metrics
curl http://localhost:8004/metrics
```

## Troubleshooting

### Serviço não conecta ao Kafka

```bash
# Verificar se Kafka está rodando
docker exec -it coxa-kafka kafka-broker-api-versions --bootstrap-server localhost:9092

# Verificar logs do Kafka
docker-compose logs kafka
```

### Banco de dados não inicializa

```bash
# Verificar se migrations rodaram
docker exec -it coxa-postgres psql -U coxa -d coxa -c "\dt"

# Executar migrations manualmente
docker exec -it coxa-postgres psql -U coxa -d coxa < backend/shared/database/migrations.sql
```

### Redis não persiste dados

```bash
# Verificar conexão
docker exec -it coxa-redis redis-cli ping

# Verificar dados
docker exec -it coxa-redis redis-cli DBSIZE
```

## Deploy

### Build Local

```bash
# Build de todas as imagens
docker-compose build

# Build de um serviço específico
docker-compose build habit-service
```

### Limpeza

```bash
# Parar containers
docker-compose down

# Remover volumes (limpar dados)
docker-compose down -v

# Remover imagens
docker-compose down --rmi all
```

## Performance

### Monitorar Latência

```bash
# No Prometheus (http://localhost:9090)
# Query: rate(event_processing_duration_seconds_sum[5m])
```

### Monitorar Throughput

```bash
# No Prometheus
# Query: rate(events_processed_total[1m])
```

### Verificar Idempotência

```bash
# Verificar chaves no Redis
docker exec -it coxa-redis redis-cli KEYS "processed:*" | wc -l
```

## Commits

Seguir convenção:

```
feat: adicionar novo endpoint de notificações
fix: corrigir idempotência em eventos duplicados
refactor: melhorar estrutura de domain
docs: atualizar DEVELOPMENT.md
chore: atualizar dependências Go
```

---

Para mais informações, consulte `ARCHITECTURE.md`
