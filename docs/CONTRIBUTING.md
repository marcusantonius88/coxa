# COXA - Contribuindo

Obrigado por querer contribuir ao COXA! Aqui estão as guidelines.

## Código de Conduta

- Respeite todos os contribuidores
- Forneça feedback construtivo
- Reporte bugs com detalhes

## Como Contribuir

### 1. Fork o Repositório

```bash
git clone https://github.com/seu-usuario/coxa.git
cd coxa
```

### 2. Criar Branch

```bash
git checkout -b feature/sua-feature
# ou
git checkout -b fix/seu-fix
```

### 3. Fazer Alterações

- Seguir o estilo de código existente
- Adicionar testes se aplicável
- Atualizar documentação

### 4. Commit

```bash
git commit -m "feat: descrição clara da mudança"
```

### 5. Push e Pull Request

```bash
git push origin feature/sua-feature
```

Abrir Pull Request no GitHub com descrição clara.

## Guidelines de Código

### Go

- Seguir `go fmt`
- Usar `go vet` para verificar
- Nomes de variáveis claros e descritivos
- Adicionar comments em funções públicas

### JavaScript/React

- Seguir ESLint config
- Usar functional components + hooks
- Props bem nomeadas
- Adicionar PropTypes ou TypeScript

## Áreas para Contribuir

- [ ] Testes unitários
- [ ] Testes de integração
- [ ] Documentação
- [ ] Novos serviços
- [ ] Novos tipos de eventos
- [ ] Bug fixes
- [ ] Performance
- [ ] UI/UX
- [ ] Integração com APIs externas (email, SMS, push)

---

## Adicionar um Novo Serviço

Se você quer criar um novo serviço (por ex: `reminder-service`), siga estes passos:

### 1. Criar estrutura de pasta

```bash
mkdir -p backend/services/reminder-service
cd backend/services/reminder-service
```

### 2. Criar arquivos base

**go.mod**:
```go
module coxa/reminder-service

go 1.22

require coxa/shared v0.0.1
```

**main.go** - Use o padrão de outro serviço como base (veja [PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md))

**Dockerfile**:
```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o reminder-service main.go

FROM alpine:latest
COPY --from=builder /app/reminder-service /
EXPOSE 8005
CMD ["/reminder-service"]
```

### 3. Atualizar docker-compose.yml

Adicionar entrada:
```yaml
reminder-service:
  build: ./backend/services/reminder-service
  container_name: coxa-reminder-service
  ports:
    - "8005:8005"
  depends_on:
    - postgres
    - kafka
    - redis
  environment:
    DATABASE_URL: user=coxa password=coxa dbname=coxa host=postgres port=5432 sslmode=disable
    KAFKA_BROKERS: kafka:9092
    REDIS_URL: redis://redis:6379
  networks:
    - coxa-network
```

### 4. Adicionar novos tipos de evento (se necessário)

Em `backend/shared/events/event.go`:
```go
const (
    EventReminderCreated   = "ReminderCreated"
    EventReminderSent      = "ReminderSent"
)
```

### 5. Implementar lógica do serviço

Seu `main.go` deve:
- Conectar ao DB, Kafka, Redis no `init()`
- Registrar handlers HTTP em `main()`
- Iniciar consumer Kafka em goroutine
- Processar eventos específicos em `processMessage()`

### 6. Atualizar prometheus.yml (se expor novas métricas)

Em `infra/prometheus/prometheus.yml`:
```yaml
  - job_name: 'reminder-service'
    static_configs:
      - targets: ['localhost:8005']
```

### 7. Testar localmente

```bash
docker-compose up -d
docker-compose logs -f reminder-service
```

### 8. Documentar em ARCHITECTURE.md

Adicionar seção descrevendo o novo serviço.

### 9. Abrir Pull Request

Descreva:
- Novo serviço criado
- Qual propósito/responsabilidade
- Eventos consumidos e publicados
- Como testar



## Reporting Bugs

Incluir:

1. Descrição clara do bug
2. Steps to reproduce
3. Comportamento esperado
4. Comportamento atual
5. Screenshots se aplicável
6. Environment (SO, versão Docker, etc)

## Feature Requests

Descrever:

1. O problema que resolve
2. Solução proposta
3. Alternativas consideradas
4. Impacto (performance, UX, etc)

---

Muito obrigado por contribuir! 🎉
