# 🚀 Setup Inicial - COXA

Guia passo a passo para colocar o projeto COXA rodando localmente.

---

## 📋 Pré-requisitos

Antes de começar, você precisa ter instalado:

### 1. **Docker & Docker Compose**
   
```bash
# Verificar instalação
docker --version
docker-compose --version

# Se não tiver:
# Linux: https://docs.docker.com/engine/install/
# macOS: https://docs.docker.com/desktop/install/mac-install/
# Windows: https://docs.docker.com/desktop/install/windows-install/
```

### 2. **Git**

```bash
git --version

# Se não tiver: https://git-scm.com/downloads
```

### 3. **Go 1.22+ (Opcional - apenas para desenvolvimento local)**

```bash
go version

# Se não tiver: https://golang.org/doc/install
```

### 4. **Node.js 18+ (Opcional - apenas para desenvolvimento do frontend)**

```bash
node --version
npm --version

# Se não tiver: https://nodejs.org/
```

---

## 🎯 Instalação Rápida

### 1. Clonar o Repositório

```bash
git clone https://github.com/seu-usuario/coxa.git
cd coxa
```

### 2. Executar o Setup Automático

```bash
bash setup.sh
```

Este script irá:
- ✅ Verificar se Docker está instalado
- ✅ Build de todas as imagens
- ✅ Iniciar todos os containers
- ✅ Aguardar ~30 segundos para inicialização
- ✅ Exibir endpoints disponíveis

Se tudo correr bem, você verá:
```
✅ COXA iniciado com sucesso!

📍 Endpoints:
  Frontend:              http://localhost:5173
  Habit Service:         http://localhost:8001
  Scheduler Service:     http://localhost:8002
  Notification Service:  http://localhost:8003
  Analytics Service:     http://localhost:8004
  Prometheus:            http://localhost:9090
  Grafana:               http://localhost:3000 (admin/admin)
```

---

## 🔧 Instalação Manual (se setup.sh falhar)

### 1. Build das Imagens

```bash
docker-compose build --no-cache
```

### 2. Iniciar os Containers

```bash
docker-compose up -d
```

### 3. Verificar Status

```bash
docker-compose ps
```

Você deve ver algo assim:
```
CONTAINER ID   IMAGE                    STATUS      PORTS
abc123...      coxa-habit-service       Up 2 min    0.0.0.0:8001->8001/tcp
abc124...      coxa-scheduler-service   Up 2 min    0.0.0.0:8002->8002/tcp
abc125...      coxa-postgres            Up 2 min    0.0.0.0:5432->5432/tcp
abc126...      coxa-kafka               Up 2 min    0.0.0.0:9092->9092/tcp
```

### 4. Aguardar Inicialização

Aguarde ~30 segundos para todos os serviços ficarem prontos, especialmente:
- Postgres (schema criado)
- Kafka (tópicos criados)
- Debezium (conectado)

### 5. Verificar Saúde

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

Esperado: `{"status":"ok"}`

---

## 🎮 Acessar Serviços

### Frontend

Abra no navegador:
```
http://localhost:5173
```

### APIs (para testes com curl)

```bash
# Listar medicamentos
curl http://localhost:8001/medications

# Listar agendamentos
curl http://localhost:8002/schedules

# Listar notificações
curl http://localhost:8003/notifications

# Timeline de eventos
curl "http://localhost:8004/timeline?owner_id=test"
```

### Admin/Dashboard

- **Prometheus** (métricas): http://localhost:9090
- **Grafana** (dashboards): http://localhost:3000
  - Login: `admin` / `admin`
- **pgAdmin** (gerenciar BD): http://localhost:5050
  - Login: `postgres@example.com` / `admin`

---

## 🧪 Teste Rápido

Após o setup, verifique se tudo está funcionando:

```bash
# 1. Criar um medicamento
curl -X POST http://localhost:8001/medications \
  -d "name=Vermífugo&description=Teste&frequency_days=30&pet_id=pet-1&owner_id=owner-1" \
  -H "Content-Type: application/x-www-form-urlencoded"

# 2. Listar medicamentos (deve aparecer o que acabou de criar)
curl http://localhost:8001/medications

# 3. Listar agendamentos (deve ter criado automaticamente via Scheduler)
curl http://localhost:8002/schedules

# 4. Verificar métricas
curl http://localhost:8001/metrics | grep "events_processed_total"
```

Se todos os comandos funcionarem, **seu setup está completo!** ✅

---

## 📚 Próximos Passos

1. **Entender a Arquitetura**
   - Leia [ARCHITECTURE.md](./ARCHITECTURE.md)
   - Leia [PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md)

2. **Fazer um Teste End-to-End Completo**
   - Siga [TESTE_END_TO_END.md](./TESTE_END_TO_END.md)

3. **Desenvolver Localmente**
   - Veja [DEVELOPMENT.md](./DEVELOPMENT.md)

4. **Contribuir**
   - Leia [CONTRIBUTING.md](./CONTRIBUTING.md)

---

## 🐛 Troubleshooting

### ❌ Erro: "Docker daemon is not running"

```bash
# macOS/Windows: Abra Docker Desktop
# Linux: Inicie o daemon
sudo systemctl start docker
```

### ❌ Erro: "Address already in use"

Alguma porta (5173, 8001, 8002, etc) já está em uso. Opções:

```bash
# Parar containers antigos
docker-compose down

# Ou mudar ports no docker-compose.yml
```

### ❌ Postgres não inicia

```bash
# Verificar logs
docker logs coxa-postgres

# Resetar volume
docker-compose down -v
docker-compose up -d
```

### ❌ Kafka não conecta

```bash
# Aguardar mais tempo
sleep 30

# Verificar logs
docker logs coxa-kafka
docker logs coxa-zookeeper
```

### ❌ Health check falha

```bash
# Ver logs do serviço específico
docker logs coxa-habit-service

# Ver logs detalhados
docker-compose logs habit-service
```

---

## 🛑 Parar e Limpar

```bash
# Parar serviços (mantém dados)
docker-compose stop

# Resumir serviços
docker-compose start

# Parar e remover tudo (apaga dados)
docker-compose down

# Limpar volumes também (CUIDADO - apaga banco de dados)
docker-compose down -v
```

---

## 💾 Persistência de Dados

Os dados são armazenados em volumes Docker:

```bash
# Ver volumes
docker volume ls | grep coxa

# Exemplo:
# coxa_postgres_data
# coxa_redis_data
```

Mesmo se parar os containers, os dados persistem. Para resetar:

```bash
docker-compose down -v  # Remove volumes também
docker-compose up -d    # Recria do zero
```

---

## 📊 Arquitetura Resumida

```
Frontend (React)
     ↓
  API Gateway
     ↓
┌──────────────────────────────┐
│   Habit Service (8001)       │
│   Scheduler Service (8002)   │
│   Notification Service (8003)│
│   Analytics Service (8004)   │
└──────────┬───────────────────┘
           │
    ┌──────┴─────┐
    │            │
    ▼            ▼
PostgreSQL    Kafka ← CDC/Debezium
    │            │
    ▼            ▼
[Data]    [Events] ─→ Consumer Services
             │
             ▼
          Redis (Idempotência)
```

---

## ✅ Checklist

- [ ] Docker e Docker Compose instalados
- [ ] Repositório clonado
- [ ] `docker-compose up -d` executado
- [ ] Todos os containers estão "Up"
- [ ] Health checks passaram
- [ ] Teste rápido (4 curl's) foram bem-sucedidos
- [ ] Frontend acessível em http://localhost:5173
- [ ] Prometheus em http://localhost:9090
- [ ] Pronto para começar! 🎉

---

Para dúvidas ou problemas, consulte [DEVELOPMENT.md](./DEVELOPMENT.md).
