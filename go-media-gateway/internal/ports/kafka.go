package ports

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

// KafkaPublisher writes usage events keyed by event id so one tenant
// stays ordered on a partition and the ledger can dedupe replays.
type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           50 * time.Millisecond,
		},
	}
}

func (p *KafkaPublisher) Publish(ctx context.Context, usage domain.Usage) error {
	body, err := json.Marshal(usage)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(usage.EventID),
		Value: body,
	})
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// LogPublisher records usage locally when Kafka is not configured.
type LogPublisher struct{}

func (LogPublisher) Publish(_ context.Context, usage domain.Usage) error {
	slog.Info("billing usage",
		"eventId", usage.EventID,
		"tenantId", usage.TenantID,
		"meter", usage.Meter,
		"quantity", usage.Quantity,
	)
	return nil
}

func (LogPublisher) Close() error { return nil }

var (
	_ UsagePublisher = (*KafkaPublisher)(nil)
	_ UsagePublisher = LogPublisher{}
)
