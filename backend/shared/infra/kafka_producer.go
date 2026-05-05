package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer encapsula a produção de eventos para Kafka
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer cria um novo produtor Kafka
func NewKafkaProducer(brokers []string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}
	return &KafkaProducer{writer: w}
}

// PublishEvent publica um evento no Kafka
func (kp *KafkaProducer) PublishEvent(ctx context.Context, topic string, eventID string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento: %w", err)
	}

	message := kafka.Message{
		Topic: topic,
		Key:   []byte(eventID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event-id", Value: []byte(eventID)},
		},
	}

	err = kp.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("erro ao publicar evento no Kafka: %w", err)
	}

	return nil
}

// Close fecha a conexão com Kafka
func (kp *KafkaProducer) Close() error {
	return kp.writer.Close()
}
