#!/bin/bash

# Wait for Debezium to be ready
echo "Waiting for Debezium to be ready..."
sleep 10

# Delete old connector if exists
curl -X DELETE http://localhost:8083/connectors/postgres-cdc 2>/dev/null

sleep 1

# Create PostgreSQL CDC connector
echo "Creating Debezium PostgreSQL connector..."
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "postgres-cdc",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "postgres",
      "database.port": 5432,
      "database.user": "coxa",
      "database.password": "coxa",
      "database.dbname": "coxa",
      "database.server.name": "coxa-db",
      "table.include.list": "public.outbox",
      "topic.prefix": "coxa",
      "plugin.name": "pgoutput",
      "publication.name": "dbz_publication",
      "slot.name": "debezium_slot",
      "heartbeat.interval.ms": 10000,
      "key.converter": "org.apache.kafka.connect.storage.StringConverter",
      "value.converter": "org.apache.kafka.connect.json.JsonConverter",
      "value.converter.schemas.enable": "false",
      "transforms": "route,unwrap",
      "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
      "transforms.route.type": "org.apache.kafka.connect.transforms.RegexRouter",
      "transforms.route.regex": "([^.]+)\\.([^.]+)\\.([^.]+)",
      "transforms.route.replacement": "medication-events"
    }
  }'

echo ""
echo "Debezium connector created!"
