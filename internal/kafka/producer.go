package kafka

import (
	"context"
	"encoding/json"

	"github.com/asmisnik/realty-parser/internal/model"
	kafka "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
	}
}

func (p *Producer) PublishFlat(ctx context.Context, flat *model.FlatInfo) error {
	data, err := json.Marshal(flat)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(flat.Link),
		Value: data,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
