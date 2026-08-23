package worker

import (
	"context"
	"encoding/json"
	"log"

	"taskflow/internal/analytics"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type TaskEvent struct {
	TaskID    string `json:"task_id"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	Assignee  string `json:"assignee"`
}

type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	analytics     *analytics.ClickHouseAnalyticsClient
	topic         string
}

func NewKafkaConsumer(
	brokers []string,
	group string,
	topic string,
	analytics *analytics.ClickHouseAnalyticsClient,
) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGroup, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		return nil, err
	}
	return &KafkaConsumer{
		consumerGroup: consumerGroup,
		analytics:     analytics,
		topic:         topic,
	}, nil
}

// Run запускает основной цикл потребления сообщений
func (c *KafkaConsumer) Run(ctx context.Context) {
	handler := &consumerHandler{analytics: c.analytics}

	for {
		if err := c.consumerGroup.Consume(ctx, []string{c.topic}, handler); err != nil {
			log.Printf("Consumer error: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.consumerGroup.Close()
}

// consumerHandler реализует sarama.ConsumerGroupHandler
type consumerHandler struct {
	analytics *analytics.ClickHouseAnalyticsClient
}

func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var event TaskEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}
		taskID, err := uuid.Parse(event.TaskID)
		if err != nil {
			log.Printf("Invalid UUID: %v", err)
			continue
		}
		h.analytics.SendEvent(analytics.Event{
			TaskID:    taskID,
			EventType: event.EventType,
			Status:    event.Status,
			Assignee:  event.Assignee,
		})
		session.MarkMessage(msg, "")
	}
	return nil
}
