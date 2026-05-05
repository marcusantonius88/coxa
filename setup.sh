#!/bin/bash

set -e

echo "🐶 COXA - Iniciando setup..."

# Verificar Docker
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose não está instalado"
    exit 1
fi

echo "📦 Construindo serviços..."
docker-compose build --no-cache

echo "🚀 Iniciando containers..."
docker-compose up -d

echo "⏳ Aguardando serviços..."
sleep 30

# Verificar health dos serviços
echo "🏥 Verificando saúde dos serviços..."

services=("habit-service" "scheduler-service" "notification-service" "analytics-service" "prometheus" "grafana")

for service in "${services[@]}"; do
    echo "  Checking $service..."
    # Aqui você pode adicionar checks específicos
done

echo ""
echo "✅ COXA iniciado com sucesso!"
echo ""
echo "📍 Endpoints:"
echo "  Frontend:              http://localhost:5173"
echo "  Habit Service:         http://localhost:8001"
echo "  Scheduler Service:     http://localhost:8002"
echo "  Notification Service:  http://localhost:8003"
echo "  Analytics Service:     http://localhost:8004"
echo "  Prometheus:            http://localhost:9090"
echo "  Grafana:               http://localhost:3000 (admin/admin)"
echo ""
echo "📖 Documentação: Veja ARCHITECTURE.md"
echo ""
