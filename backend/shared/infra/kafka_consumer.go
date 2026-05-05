package infra

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumer encapsula o consumo de eventos do Kafka
type KafkaConsumer struct {
	reader *kafka.Reader
}

// NewKafkaConsumer cria um novo consumidor Kafka
func NewKafkaConsumer(brokers []string, topic, groupID string) *KafkaConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
	})
	return &KafkaConsumer{reader: r}
}

// ReadMessage lê uma mensagem do Kafka
func (kc *KafkaConsumer) ReadMessage(ctx context.Context) ([]byte, error) {
	message, err := kc.reader.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler mensagem do Kafka: %w", err)
	}
	return message.Value, nil
}

// Close fecha a conexão com Kafka
func (kc *KafkaConsumer) Close() error {
	return kc.reader.Close()
}
